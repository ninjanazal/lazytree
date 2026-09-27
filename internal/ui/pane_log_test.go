package ui

import (
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
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

// TestLogPane_CacheBuiltBySetters guards against a regression where the
// render cache was only built lazily inside View. AppModel.View has a value
// receiver, so any cache written from inside logPane.View would be thrown
// away when the frame returns — every frame would silently rebuild the
// whole graph despite the cache existing. setCommits and setSize must build
// the cache themselves, so it's already populated before View ever runs.
func TestLogPane_CacheBuiltBySetters(t *testing.T) {
	p := newTestLogPane() // setSize + setCommits already ran

	if len(p.cachedGraphLines) != len(p.commits) {
		t.Fatalf("expected cache to be populated by setCommits before View ran, got %d cached lines for %d commits",
			len(p.cachedGraphLines), len(p.commits))
	}
	if p.cachedGraphColW == 0 {
		t.Fatalf("expected column widths to be populated by setCommits before View ran")
	}

	// Changing width must rebuild the cache too.
	prevMessageW := p.cachedMessageW
	p.setSize(40, 20)
	if p.cachedMessageW == prevMessageW {
		t.Errorf("expected cachedMessageW to change after a width change, still %d", p.cachedMessageW)
	}

	// A same-width setSize call must not touch the cache identity.
	before := p.cachedGraphLines
	p.setSize(40, 5) // height-only change
	if &before[0] != &p.cachedGraphLines[0] {
		t.Errorf("expected cache to be left untouched when width doesn't change")
	}
}

// TestLogPane_AppendMatchesFullRebuild checks that loading history page by
// page with appendCommits renders the same graph as loading it in one go.
// The split is right after d, the only commit with a diagonal connector:
// as the last row of page 1 it is rendered without one (next == nil), so
// appendCommits must redo it once its successor arrives.
//
// Rows are compared with trailing spaces stripped: rows from earlier pages
// are rendered at the layout width known at the time, so if a later page
// widens the graph they carry fewer trailing blank lanes. padRight in View
// hides that difference.
func TestLogPane_AppendMatchesFullRebuild(t *testing.T) {
	full := branchCommits() // d (merge of b,c), c, b, a

	pFull := newLogPane()
	pFull.setSize(80, 20)
	pFull.setCommits(full, graph.NewLayouter().Append(full))

	l := graph.NewLayouter()
	pPaged := newLogPane()
	pPaged.setSize(80, 20)
	pPaged.setCommits(full[:1], l.Append(full[:1]))
	pPaged.appendCommits(full[1:], l.Append(full[1:]))

	if len(pFull.cachedGraphLines) != len(pPaged.cachedGraphLines) {
		t.Fatalf("cached line count mismatch: full=%d paged=%d", len(pFull.cachedGraphLines), len(pPaged.cachedGraphLines))
	}
	trim := func(s string) string {
		lines := strings.Split(s, "\n")
		for i := range lines {
			lines[i] = strings.TrimRight(lines[i], " ")
		}
		return strings.Join(lines, "\n")
	}
	for i := range pFull.cachedGraphLines {
		if trim(pFull.cachedGraphLines[i]) != trim(pPaged.cachedGraphLines[i]) {
			t.Errorf("line %d differs:\n full:  %q\n paged: %q", i, pFull.cachedGraphLines[i], pPaged.cachedGraphLines[i])
		}
	}
	if pFull.cachedGraphColW != pPaged.cachedGraphColW {
		t.Errorf("cachedGraphColW differs: full=%d paged=%d", pFull.cachedGraphColW, pPaged.cachedGraphColW)
	}
	if got, want := pPaged.View(false), pFull.View(false); got != want {
		t.Errorf("paginated view differs from full-load view:\npaged: %s\nfull:  %s", got, want)
	}
}

// TestLogPane_SetCommitsAlwaysRebuilds covers a reload whose history keeps
// the same length and oldest commit but changes the newest one (an amend +
// force-push). setCommits must rescan everything, not treat it as an append.
func TestLogPane_SetCommitsAlwaysRebuilds(t *testing.T) {
	old := branchCommits()
	p := newLogPane()
	p.setSize(80, 20)
	p.setCommits(old, graph.Layout(old))

	amended := branchCommits()
	amended[0].Hash, amended[0].ShortHash = "d2", "d2"
	amended[0].Author = "a much longer author"
	p.setCommits(amended, graph.Layout(amended))

	if want := lipgloss.Width(truncate(amended[0].Author, 16)); p.cachedAuthorColW < want {
		t.Errorf("author column not rescanned after reload: got %d, want >= %d", p.cachedAuthorColW, want)
	}
	if len(p.cachedGraphLines) != len(amended) {
		t.Errorf("got %d cached lines for %d commits", len(p.cachedGraphLines), len(amended))
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
