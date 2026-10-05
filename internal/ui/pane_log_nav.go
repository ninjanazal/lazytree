package ui

import (
	"fmt"
	"slices"

	tea "github.com/charmbracelet/bubbletea"
)

// parentIndex turns a "1"–"9" key press into a 0-based parent index.
func parentIndex(msg tea.KeyMsg) int { return int(msg.String()[0] - '1') }

// jumpToParent moves the cursor to the first parent of the selected commit.
func (p *logPane) jumpToParent() bool { return p.jumpToNthParent(0) }

// jumpToNthParent moves the cursor to parent n (0-based) of the selected
// commit; a merge's side branches are parents 1, 2, … Parents usually come
// after their children, so the scan looks down the list first, then up:
// git log is date-ordered, not topo-ordered, and clock skew can list a
// parent above its child. It reports false when there is no such parent or
// it hasn't been loaded yet (the log streams in pages).
func (p *logPane) jumpToNthParent(n int) bool {
	c := p.selectedCommit()
	if c == nil || n < 0 || n >= len(c.Parents) {
		return false
	}
	want := c.Parents[n]
	for i := p.cursor + 1; i < len(p.commits); i++ {
		if p.commits[i].Hash == want {
			p.setCursor(i, p.offset)
			return true
		}
	}
	for i := p.cursor - 1; i >= 0; i-- {
		if p.commits[i].Hash == want {
			p.setCursor(i, p.offset)
			return true
		}
	}
	return false
}

// parentMiss explains why parent n (0-based) of the selected commit can't
// be jumped to, for the footer flash.
func (p *logPane) parentMiss(n int) string {
	c := p.selectedCommit()
	switch {
	case c == nil:
		return ""
	case len(c.Parents) == 0:
		return "root commit: no parent"
	case n >= len(c.Parents):
		return fmt.Sprintf("commit has %d parent(s), no parent %d", len(c.Parents), n+1)
	default:
		h := c.Parents[n]
		return "parent " + h[:min(len(h), 8)] + " not loaded"
	}
}

// jumpToChild moves the cursor to the nearest commit above the selected one
// that has it as a parent (the closest child when there are several).
func (p *logPane) jumpToChild() bool {
	c := p.selectedCommit()
	if c == nil {
		return false
	}
	for i := p.cursor - 1; i >= 0; i-- {
		if slices.Contains(p.commits[i].Parents, c.Hash) {
			p.setCursor(i, p.offset)
			return true
		}
	}
	return false
}

// jumpToHead moves the cursor to the commit HEAD points at.
func (p *logPane) jumpToHead() bool {
	for i, c := range p.commits {
		for _, r := range c.Refs {
			if r.IsHead {
				p.setCursor(i, p.offset)
				return true
			}
		}
	}
	return false
}
