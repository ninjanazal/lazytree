package git

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func requireGit(t *testing.T) {
	t.Helper()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not found in PATH")
	}
}

// runGit runs git in dir with a hermetic config/author environment.
func runGit(t *testing.T, dir string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(),
		"GIT_CONFIG_GLOBAL=/dev/null",
		"GIT_CONFIG_SYSTEM=/dev/null",
		"GIT_AUTHOR_NAME=Test",
		"GIT_AUTHOR_EMAIL=test@example.com",
		"GIT_COMMITTER_NAME=Test",
		"GIT_COMMITTER_EMAIL=test@example.com",
	)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
	return string(out)
}

func initRepo(t *testing.T, dir string) {
	t.Helper()
	runGit(t, filepath.Dir(dir), "init", "--initial-branch=main", filepath.Base(dir))
	runGit(t, dir, "commit", "--allow-empty", "-m", "init")
}

func resolve(t *testing.T, dir string) (string, error) {
	t.Helper()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	return ResolveRepo(ctx, dir)
}

func evalSym(t *testing.T, p string) string {
	t.Helper()
	resolved, err := filepath.EvalSymlinks(p)
	if err != nil {
		t.Fatalf("EvalSymlinks(%q): %v", p, err)
	}
	return resolved
}

func TestResolveRepo_NormalRoot(t *testing.T) {
	requireGit(t)
	tmp := t.TempDir()
	repo := filepath.Join(tmp, "r")
	initRepo(t, repo)

	got, err := resolve(t, repo)
	if err != nil {
		t.Fatalf("ResolveRepo: %v", err)
	}
	if evalSym(t, got) != evalSym(t, repo) {
		t.Errorf("got %q, want %q", got, repo)
	}
}

func TestResolveRepo_Subdirectory(t *testing.T) {
	requireGit(t)
	tmp := t.TempDir()
	repo := filepath.Join(tmp, "r")
	initRepo(t, repo)

	sub := filepath.Join(repo, "a", "b")
	if err := os.MkdirAll(sub, 0o755); err != nil {
		t.Fatal(err)
	}

	got, err := resolve(t, sub)
	if err != nil {
		t.Fatalf("ResolveRepo: %v", err)
	}
	if evalSym(t, got) != evalSym(t, repo) {
		t.Errorf("got %q, want %q", got, repo)
	}
}

func TestResolveRepo_Worktree(t *testing.T) {
	requireGit(t)
	tmp := t.TempDir()
	repo := filepath.Join(tmp, "r")
	initRepo(t, repo)

	wt := filepath.Join(tmp, "wt")
	runGit(t, repo, "worktree", "add", wt)

	got, err := resolve(t, wt)
	if err != nil {
		t.Fatalf("ResolveRepo: %v", err)
	}
	if evalSym(t, got) != evalSym(t, wt) {
		t.Errorf("got %q, want %q", got, wt)
	}
}

func TestResolveRepo_Submodule(t *testing.T) {
	requireGit(t)
	tmp := t.TempDir()
	outer := filepath.Join(tmp, "outer")
	initRepo(t, outer)

	inner := filepath.Join(tmp, "inner")
	initRepo(t, inner)

	subPath := filepath.Join(outer, "sub")
	runGit(t, outer, "-c", "protocol.file.allow=always", "submodule", "add", inner, subPath)

	got, err := resolve(t, subPath)
	if err != nil {
		t.Fatalf("ResolveRepo: %v", err)
	}
	if evalSym(t, got) != evalSym(t, subPath) {
		t.Errorf("got %q, want %q (submodule root, not superproject)", got, subPath)
	}
}

func TestResolveRepo_Bare(t *testing.T) {
	requireGit(t)
	tmp := t.TempDir()
	bare := filepath.Join(tmp, "b.git")
	runGit(t, tmp, "init", "--bare", bare)

	got, err := resolve(t, bare)
	if err != nil {
		t.Fatalf("ResolveRepo: %v", err)
	}
	if evalSym(t, got) != evalSym(t, bare) {
		t.Errorf("got %q, want %q", got, bare)
	}
}

func TestResolveRepo_NotARepo(t *testing.T) {
	requireGit(t)
	tmp := t.TempDir()

	if _, err := resolveWithCeiling(t, tmp); err == nil {
		t.Error("expected error for non-repo directory, got nil")
	}
}

// resolveWithCeiling calls ResolveRepo with GIT_CEILING_DIRECTORIES set so
// git does not walk up past tmp's parent into an ancestor repository (e.g.
// this project's own .git during `go test`). t.Setenv restores the previous
// value (or unsets it) automatically once the test ends.
func resolveWithCeiling(t *testing.T, dir string) (string, error) {
	t.Helper()
	t.Setenv("GIT_CEILING_DIRECTORIES", filepath.Dir(dir))
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	return ResolveRepo(ctx, dir)
}

func TestResolveRepo_MissingPath(t *testing.T) {
	requireGit(t)
	_, err := resolve(t, "/definitely/does/not/exist/lazytree-test")
	if err == nil {
		t.Error("expected error for missing path, got nil")
	}
}
