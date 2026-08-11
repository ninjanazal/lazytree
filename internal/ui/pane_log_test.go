package ui

import (
	"strings"
	"testing"

	"github.com/eurico-martins/lazytree/internal/graph"
	"github.com/eurico-martins/lazytree/internal/model"
)

func branchCommits() []model.Commit {
	return []model.Commit{
		{Hash: "d", ShortHash: "d", Subject: "merge feature", Parents: []string{"b", "c"}},
		{Hash: "c", ShortHash: "c", Subject: "feature work", Parents: nil},
		{Hash: "b", ShortHash: "b", Subject: "main work", Parents: []string{"a"}},
		{Hash: "a", ShortHash: "a", Subject: "root", Parents: nil},
	}
}

func newTestLogPane() logPane {
	p := newLogPane()
	commits := branchCommits()
	layout := graph.Layout(commits)
	p.setSize(80, 20)
	p.setCommits(commits, layout)
	return p
}

func TestLogPane_FilterKeepsGraphGlyphs(t *testing.T) {
	p := newTestLogPane()
	p.applyFilter("root")

	view := p.View(false)
	if !strings.Contains(view, "root") {
		t.Fatalf("expected filtered view to contain matching commit, got:\n%s", view)
	}

	// Regression: the graph column used to be forced blank ("graphStr = \" \"")
	// whenever a filter was active. Find the matching row and assert it still
	// carries a non-blank graph glyph.
	var matchLine string
	for line := range strings.SplitSeq(view, "\n") {
		if strings.Contains(line, "root") && !strings.Contains(line, "filter:") {
			matchLine = line
			break
		}
	}
	if matchLine == "" {
		t.Fatalf("could not find matching row in view:\n%s", view)
	}
	if !strings.Contains(matchLine, "o") {
		t.Errorf("expected matching row to still show a commit-dot graph glyph, got: %q", matchLine)
	}
}

func TestLogPane_ScrollWithConnectors(t *testing.T) {
	p := newTestLogPane()
	p.setSize(80, 6) // small height, forces a tight line budget

	// Should not panic or get stuck; cursor should be able to reach the end.
	for range len(p.commits) + 2 {
		p.moveDown()
	}
	if p.cursor != len(p.commits)-1 {
		t.Errorf("expected cursor to reach last commit (%d), got %d", len(p.commits)-1, p.cursor)
	}

	// View should render without panicking even when connector lines are
	// present and the line budget is small.
	view := p.View(false)
	if view == "" {
		t.Errorf("expected non-empty view")
	}
}
