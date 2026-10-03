package ui

import (
	"strings"

	"github.com/eurico-martins/lazytree/internal/model"
)

// refFilter hides some ref pills in the log. It is display-only: commits,
// the graph and search are unaffected, and HEAD is never hidden.
type refFilter struct {
	hideTags    bool
	hideRemotes bool
	globs       []string // ref names to hide; "*" matches any characters
}

func (f refFilter) active() bool {
	return f.hideTags || f.hideRemotes || len(f.globs) > 0
}

// apply returns refs without the hidden ones. It returns the input slice
// untouched (no allocation) when nothing is hidden.
func (f refFilter) apply(refs []model.Ref) []model.Ref {
	if !f.active() || len(refs) == 0 {
		return refs
	}
	out := make([]model.Ref, 0, len(refs))
	for _, r := range refs {
		if !f.hides(r) {
			out = append(out, r)
		}
	}
	return out
}

func (f refFilter) hides(r model.Ref) bool {
	if r.IsHead || r.Kind == model.RefHead {
		return false
	}
	switch {
	case f.hideTags && r.Kind == model.RefTag:
		return true
	case f.hideRemotes && r.Kind == model.RefRemoteBranch:
		return true
	}
	for _, g := range f.globs {
		if globMatch(g, r.Name) {
			return true
		}
	}
	return false
}

// label summarizes what is hidden, for the toolbar.
func (f refFilter) label() string {
	var parts []string
	if f.hideTags {
		parts = append(parts, "no tags")
	}
	if f.hideRemotes {
		parts = append(parts, "no remotes")
	}
	if len(f.globs) > 0 {
		parts = append(parts, "filtered")
	}
	return strings.Join(parts, " · ")
}

// globMatch reports whether name matches pattern, where "*" matches any
// run of characters (including "/") and everything else matches literally.
func globMatch(pattern, name string) bool {
	parts := strings.Split(pattern, "*")
	if len(parts) == 1 {
		return pattern == name
	}
	if !strings.HasPrefix(name, parts[0]) {
		return false
	}
	name = name[len(parts[0]):]
	last := parts[len(parts)-1]
	for _, mid := range parts[1 : len(parts)-1] {
		i := strings.Index(name, mid)
		if i < 0 {
			return false
		}
		name = name[i+len(mid):]
	}
	return strings.HasSuffix(name, last)
}
