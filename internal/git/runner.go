package git

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"os/exec"
)

type Runner struct {
	RepoPath string
}

// newCmd builds an *exec.Cmd for `git <args...>` with the setup shared by
// Run and Start: working directory and a forced C locale so git's stderr
// text is in English (ResolveRepo pattern-matches on it, e.g. "not a git
// repository", and a stable language keeps that working regardless of the
// user's environment).
func (r *Runner) newCmd(ctx context.Context, args ...string) *exec.Cmd {
	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Dir = r.RepoPath
	cmd.Env = append(os.Environ(), "LC_ALL=C")
	return cmd
}

func (r *Runner) Run(ctx context.Context, args ...string) ([]byte, error) {
	cmd := r.newCmd(ctx, args...)
	var out, errOut bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &errOut
	if err := cmd.Run(); err != nil {
		if errors.Is(err, exec.ErrNotFound) {
			return nil, errors.New("git not found in PATH")
		}
		return nil, &Error{
			Args:     args,
			Stderr:   errOut.String(),
			ExitCode: exitCodeOf(err),
			Err:      err,
		}
	}
	return out.Bytes(), nil
}

// StreamCmd is a running `git` subprocess whose stdout is read
// incrementally instead of buffered in full by Run. Built by Runner.Start;
// see LogStream (internal/git/log.go) for the NUL-record reader built on
// top of it.
type StreamCmd struct {
	cmd    *exec.Cmd
	Stdout io.ReadCloser
	stderr bytes.Buffer
	cancel context.CancelFunc
	args   []string
}

// Start launches `git <args...>` and returns once it's running, without
// waiting for it to finish. Callers must eventually call Wait or Kill.
func (r *Runner) Start(ctx context.Context, args ...string) (*StreamCmd, error) {
	ctx, cancel := context.WithCancel(ctx)
	cmd := r.newCmd(ctx, args...)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		cancel()
		return nil, err
	}
	sc := &StreamCmd{cmd: cmd, Stdout: stdout, cancel: cancel, args: args}
	cmd.Stderr = &sc.stderr
	if err := cmd.Start(); err != nil {
		cancel()
		if errors.Is(err, exec.ErrNotFound) {
			return nil, errors.New("git not found in PATH")
		}
		return nil, &Error{Args: args, ExitCode: -1, Err: err}
	}
	return sc, nil
}

// Wait blocks until the process exits, returning a *Error (the same shape
// Run produces) on a non-zero exit.
func (s *StreamCmd) Wait() error {
	err := s.cmd.Wait()
	s.cancel()
	if err != nil {
		return &Error{Args: s.args, Stderr: s.stderr.String(), ExitCode: exitCodeOf(err), Err: err}
	}
	return nil
}

// Kill terminates the process and reaps it, for abandoning a stream before
// it's finished (e.g. a superseded reload). Unlike Wait, its result is
// discarded: a killed process is expected to exit non-zero.
func (s *StreamCmd) Kill() {
	s.cancel()
	_ = s.cmd.Wait()
}
