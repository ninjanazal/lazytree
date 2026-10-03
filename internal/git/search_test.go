package git

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestSearchHashes(t *testing.T) {
	dir := t.TempDir()
	run := func(args ...string) {
		t.Helper()
		runGit(t, dir, args...)
	}
	write := func(name, body string) {
		t.Helper()
		if err := os.WriteFile(filepath.Join(dir, name), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	run("init", "-q", "-b", "main")
	write("a.txt", "alpha\n")
	run("add", ".")
	run("commit", "-q", "-m", "add alpha file")
	write("b.txt", "needle in a haystack\n")
	run("add", ".")
	run("commit", "-q", "-m", "add b")

	r := &Runner{RepoPath: dir}
	ctx := context.Background()

	grep, err := SearchHashes(ctx, r, SearchGrep, "ALPHA", true)
	if err != nil || len(grep) != 1 {
		t.Fatalf("grep (case-insensitive): got %v, err %v", grep, err)
	}
	code, err := SearchHashes(ctx, r, SearchCode, "needle", true)
	if err != nil || len(code) != 1 {
		t.Fatalf("pickaxe: got %v, err %v", code, err)
	}
	path, err := SearchHashes(ctx, r, SearchPath, "a.txt", true)
	if err != nil || len(path) != 1 {
		t.Fatalf("path: got %v, err %v", path, err)
	}
	if grep[0] == code[0] {
		t.Error("grep and pickaxe should hit different commits")
	}
	none, err := SearchHashes(ctx, r, SearchGrep, "zzz-nothing", true)
	if err != nil || len(none) != 0 {
		t.Fatalf("no-match search: got %v, err %v", none, err)
	}
}
