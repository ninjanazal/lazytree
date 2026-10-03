package ui

import "slices"

// jumpToParent moves the cursor to the first parent of the selected commit.
// Parents come after their children in log order, so the scan only looks
// down the list. It reports false for a root commit or when the parent
// hasn't been loaded yet (the log streams in pages).
func (p *logPane) jumpToParent() bool {
	c := p.selectedCommit()
	if c == nil || len(c.Parents) == 0 {
		return false
	}
	for i := p.cursor + 1; i < len(p.commits); i++ {
		if p.commits[i].Hash == c.Parents[0] {
			p.setCursor(i, p.offset)
			return true
		}
	}
	return false
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
