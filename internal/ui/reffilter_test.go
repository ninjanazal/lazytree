package ui

import (
	"testing"

	"github.com/eurico-martins/lazytree/internal/model"
)

func TestGlobMatch(t *testing.T) {
	cases := []struct {
		pat, name string
		want      bool
	}{
		{"main", "main", true},
		{"main", "mainline", false},
		{"origin/*", "origin/feature/x", true},
		{"origin/*", "upstream/x", false},
		{"*-wip", "feat-wip", true},
		{"*-wip", "feat-wip2", false},
		{"dependabot/*/lib", "dependabot/npm/lib", true},
		{"dependabot/*/lib", "dependabot/npm/other", false},
		{"*", "anything", true},
	}
	for _, c := range cases {
		if got := globMatch(c.pat, c.name); got != c.want {
			t.Errorf("globMatch(%q, %q) = %v, want %v", c.pat, c.name, got, c.want)
		}
	}
}

func TestRefFilter(t *testing.T) {
	refs := []model.Ref{
		{Name: "main", Kind: model.RefLocalBranch, IsHead: true},
		{Name: "feat", Kind: model.RefLocalBranch},
		{Name: "origin/main", Kind: model.RefRemoteBranch},
		{Name: "v1", Kind: model.RefTag},
	}
	names := func(rs []model.Ref) []string {
		var out []string
		for _, r := range rs {
			out = append(out, r.Name)
		}
		return out
	}
	if got := names(refFilter{}.apply(refs)); len(got) != 4 {
		t.Errorf("empty filter must keep everything, got %v", got)
	}
	if got := names(refFilter{hideTags: true}.apply(refs)); len(got) != 3 {
		t.Errorf("hideTags: got %v", got)
	}
	if got := names(refFilter{hideTags: true, hideRemotes: true}.apply(refs)); len(got) != 2 {
		t.Errorf("hide tags+remotes: got %v", got)
	}
	// HEAD is never hidden, even by a glob that matches it.
	got := names(refFilter{globs: []string{"*"}}.apply(refs))
	if len(got) != 1 || got[0] != "main" {
		t.Errorf("only HEAD should survive a catch-all glob, got %v", got)
	}
	if l := (refFilter{hideTags: true, hideRemotes: true}).label(); l != "no tags · no remotes" {
		t.Errorf("label = %q", l)
	}
}
