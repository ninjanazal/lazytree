package git

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// ResolveRepo asks git where the repository containing dir lives. It returns
// the work-tree top level for normal repos, worktrees and submodules, and the
// git directory itself for bare repositories.
func ResolveRepo(ctx context.Context, dir string) (string, error) {
	abs, err := filepath.Abs(dir)
	if err != nil {
		return "", fmt.Errorf("cannot resolve path %q: %w", dir, err)
	}
	info, err := os.Stat(abs)
	if err != nil {
		return "", fmt.Errorf("cannot access %q: %w", abs, err)
	}
	if !info.IsDir() {
		return "", fmt.Errorf("%q is not a directory", abs)
	}

	r := &Runner{RepoPath: abs}
	bare, err := r.Run(ctx, "rev-parse", "--is-bare-repository")
	if err != nil {
		return "", notARepo(abs, err)
	}

	arg := "--show-toplevel"
	if strings.TrimSpace(string(bare)) == "true" {
		arg = "--absolute-git-dir"
	}
	out, err := r.Run(ctx, "rev-parse", arg)
	if err != nil {
		return "", notARepo(abs, err)
	}
	root := strings.TrimSpace(string(out))
	if root == "" {
		// Inside a .git directory of a non-bare repo: no top level to show.
		return "", fmt.Errorf("%q is inside a git directory, not a work tree", abs)
	}
	return root, nil
}

func notARepo(abs string, err error) error {
	// Surface git's reason (e.g. safe.directory) unless it's the plain
	// "not a git repository" case.
	if gitErr, ok := errors.AsType[*Error](err); ok && strings.Contains(gitErr.Stderr, "not a git repository") {
		return fmt.Errorf("%q is not inside a git repository", abs)
	}
	return fmt.Errorf("%q: %w", abs, err)
}
