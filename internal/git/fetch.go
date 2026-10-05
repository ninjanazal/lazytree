package git

import (
	"context"
	"os"
)

// fetchEnv makes git non-interactive. A background fetch has no way to
// answer a credential prompt, and a prompt on the controlling terminal
// would corrupt the TUI: GIT_TERMINAL_PROMPT=0 stops git's own prompts,
// GIT_ASKPASS=true (answers with nothing) stops askpass helpers, and ssh
// runs in BatchMode so it fails instead of asking for a passphrase or
// host-key confirmation. Credential helpers and ssh-agent keep working.
//
// A user-supplied GIT_SSH_COMMAND is left alone. A user-supplied
// GIT_ASKPASS is overridden on purpose: it is often a GUI dialog that
// would pop up every fetch interval.
func fetchEnv() []string {
	env := []string{"GIT_TERMINAL_PROMPT=0", "GIT_ASKPASS=true", "SSH_ASKPASS=true"}
	if os.Getenv("GIT_SSH_COMMAND") == "" {
		env = append(env, "GIT_SSH_COMMAND=ssh -o BatchMode=yes")
	}
	return env
}

// Fetch runs `git fetch` against all remotes, quietly and without ever
// prompting. It is intended for background/periodic use (live fetch):
// callers bound it with ctx, swallow the returned error and retry on the
// next tick rather than surface it in the UI.
func Fetch(ctx context.Context, r *Runner) error {
	_, err := r.runEnv(ctx, fetchEnv(), "fetch", "--quiet", "--all")
	return err
}
