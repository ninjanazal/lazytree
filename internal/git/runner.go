package git

import (
	"bytes"
	"context"
	"errors"
	"os"
	"os/exec"
)

type Runner struct {
	RepoPath string
}

func (r *Runner) Run(ctx context.Context, args ...string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Dir = r.RepoPath
	// Force the C locale so git's stderr text is in English: ResolveRepo
	// pattern-matches on it (e.g. "not a git repository"), and a stable
	// language keeps that working regardless of the user's environment.
	cmd.Env = append(os.Environ(), "LC_ALL=C")
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
