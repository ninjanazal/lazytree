package git

import "context"

// Fetch runs `git fetch` against all remotes, quietly. It is intended for
// background/periodic use (live fetch): callers should swallow the returned
// error and retry on the next tick rather than surface it in the UI.
func Fetch(ctx context.Context, r *Runner) error {
	_, err := r.Run(ctx, "fetch", "--quiet", "--all")
	return err
}
