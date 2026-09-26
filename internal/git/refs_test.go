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
