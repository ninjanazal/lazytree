package git

import (
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
