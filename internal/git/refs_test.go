package git

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/eurico-martins/lazytree/internal/model"
)

func TestClassifyRef(t *testing.T) {
	cases := []struct {
		name    string
		refname string
		isHead  bool
		want    model.Ref
	}{
		{
			name:    "local branch, is head",
			refname: "refs/heads/main",
			isHead:  true,
			want:    model.Ref{Name: "main", Kind: model.RefLocalBranch, IsHead: true},
		},
		{
			name:    "local branch, not head",
			refname: "refs/heads/feature/x",
			isHead:  false,
			want:    model.Ref{Name: "feature/x", Kind: model.RefLocalBranch, IsHead: false},
		},
		{
			name:    "remote branch",
			refname: "refs/remotes/origin/main",
			isHead:  true,
			want:    model.Ref{Name: "origin/main", Kind: model.RefRemoteBranch, IsHead: true},
		},
		{
			name:    "tag always non-head",
			refname: "refs/tags/v1.0.0",
			isHead:  true, // classifyRef must ignore this for tags
			want:    model.Ref{Name: "v1.0.0", Kind: model.RefTag, IsHead: false},
		},
		{
			name:    "stash",
			refname: "refs/stash",
			isHead:  true,
			want:    model.Ref{Name: "stash", Kind: model.RefStash},
		},
		{
			name:    "unrecognized refname falls back to RefHead",
			refname: "HEAD",
			isHead:  true,
			want:    model.Ref{Name: "HEAD", Kind: model.RefHead, IsHead: true},
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := classifyRef(c.refname, c.isHead)
			if got != c.want {
				t.Errorf("classifyRef(%q, %v) = %+v, want %+v", c.refname, c.isHead, got, c.want)
			}
		})
	}
}

func TestRefsFingerprint(t *testing.T) {
	a := map[string][]model.Ref{
		"hash1": {{Name: "main", Kind: model.RefLocalBranch}},
		"hash2": {{Name: "origin/main", Kind: model.RefRemoteBranch}},
	}
	// Same content, different map (and thus potentially different
	// iteration order) — the fingerprint must still match.
	b := map[string][]model.Ref{
		"hash2": {{Name: "origin/main", Kind: model.RefRemoteBranch}},
		"hash1": {{Name: "main", Kind: model.RefLocalBranch}},
	}
	if RefsFingerprint(a) != RefsFingerprint(b) {
		t.Errorf("expected equal-content maps to produce the same fingerprint")
	}

	// Moving a branch to a new hash must change the fingerprint, so a
	// fetch that fast-forwards a ref is detected.
	moved := map[string][]model.Ref{
		"hash3": {{Name: "main", Kind: model.RefLocalBranch}},
		"hash2": {{Name: "origin/main", Kind: model.RefRemoteBranch}},
	}
	if RefsFingerprint(a) == RefsFingerprint(moved) {
		t.Errorf("expected a moved ref to change the fingerprint")
	}

	if RefsFingerprint(nil) != RefsFingerprint(map[string][]model.Ref{}) {
		t.Errorf("expected nil and empty maps to produce the same fingerprint")
	}
}

func TestBuildRefsByHash_StashLabels(t *testing.T) {
	requireGit(t)
	dir := t.TempDir()
	initRepo(t, dir)
	if err := os.WriteFile(filepath.Join(dir, "f.txt"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	runGit(t, dir, "add", "f.txt")
	runGit(t, dir, "stash")

	refs, err := BuildRefsByHash(context.Background(), &Runner{RepoPath: dir})
	if err != nil {
		t.Fatal(err)
	}
	stash := strings.TrimSpace(runGit(t, dir, "rev-parse", "refs/stash"))
	index := strings.TrimSpace(runGit(t, dir, "rev-parse", "refs/stash^2"))
	if got := refs[stash]; len(got) != 1 || got[0].Name != "stash" || got[0].Kind != model.RefStash {
		t.Errorf("stash commit refs = %+v", got)
	}
	if got := refs[index]; len(got) != 1 || got[0].Kind != model.RefStashHelper {
		t.Errorf("index commit refs = %+v", got)
	}

	files, err := FetchDiff(context.Background(), &Runner{RepoPath: dir}, stash, true)
	if err != nil || len(files) != 1 {
		t.Errorf("stash first-parent diff = %+v, %v; want f.txt", files, err)
	}
}

func TestFoldStash(t *testing.T) {
	commits := []model.Commit{
		{Hash: "s", Parents: []string{"base", "idx"}, Refs: []model.Ref{{Name: "stash", Kind: model.RefStash}}},
		{Hash: "idx", Parents: []string{"base"}, Refs: []model.Ref{{Name: "stash index", Kind: model.RefStashHelper}}},
		{Hash: "base"},
	}
	got := FoldStash(commits)
	if len(got) != 2 || got[0].Hash != "s" || got[1].Hash != "base" {
		t.Fatalf("FoldStash = %+v", got)
	}
	if len(got[0].Parents) != 1 || got[0].Parents[0] != "base" {
		t.Errorf("stash parents = %v, want [base]", got[0].Parents)
	}
}
