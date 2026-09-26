package git

import (
	"errors"
	"fmt"
	"os/exec"
	"strings"
)

// Error wraps a failed git invocation. Runner.Run is the only place that
// constructs one, so this is the single spot where a git failure gets
// formatted into text -- callers (FetchLog, FetchDiff, ResolveRepo, the UI
// banner, ...) just propagate it instead of wrapping it again.
//
// Error() leads with git's own stderr message, since that's the useful part
// a human wants to read first; the subcommand follows in parentheses for
// context. That order matters when the message later gets truncated to fit
// a single-line UI banner.
//
// Only Args[0] (the subcommand, e.g. "log" or "show") appears in that
// context, not the full argument list: some callers (FetchLog) pass a
// --format string containing raw control bytes as a field separator, and a
// long argument list is noise a human doesn't need anyway. The full Args
// slice stays available on the struct for a caller that wants it.
type Error struct {
	Args     []string // the args passed to `git` (without the "git" itself)
	Stderr   string   // git's raw, possibly multi-line, stderr output
	ExitCode int      // -1 if git never produced an exit code
	Err      error    // the underlying *exec.ExitError (or other exec failure)
}

func (e *Error) Error() string {
	cmd := "git"
	if len(e.Args) > 0 {
		cmd = "git " + e.Args[0]
	}
	if first := firstLine(e.Stderr); first != "" {
		return fmt.Sprintf("%s (%s)", first, cmd)
	}
	return fmt.Sprintf("%s: %v", cmd, e.Err)
}

func (e *Error) Unwrap() error { return e.Err }

// firstLine returns the first non-blank line of s, trimmed. git's stderr
// often carries a "fatal: ..." line followed by "hint: ..." lines; callers
// that show this in a constrained space (a one-line banner) want just the
// first.
func firstLine(s string) string {
	s = strings.TrimSpace(s)
	if i := strings.IndexByte(s, '\n'); i >= 0 {
		s = strings.TrimSpace(s[:i])
	}
	return s
}

func exitCodeOf(err error) int {
	if exitErr, ok := errors.AsType[*exec.ExitError](err); ok {
		return exitErr.ExitCode()
	}
	return -1
}
