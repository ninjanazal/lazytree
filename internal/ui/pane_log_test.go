package ui

import (
	"fmt"
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
	"github.com/eurico-martins/lazytree/internal/graph"
	"github.com/eurico-martins/lazytree/internal/model"
	"github.com/muesli/termenv"
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

func TestLogPane_SearchKeepsAllRowsAndGraphGlyphs(t *testing.T) {
	p := newTestLogPane()
	p.setSearch("root")
	if !p.jumpToMatch(true, true) {
		t.Fatal("expected a match for 'root'")
	}

	view := p.View(false)
	for _, subj := range []string{"merge feature", "feature work", "main work", "root"} {
		if !strings.Contains(view, subj) {
			t.Errorf("search must not hide rows; %q missing from:\n%s", subj, view)
		}
	}
	// The matching row keeps its graph glyph and carries the match marker.
	var matchLine string
	for line := range strings.SplitSeq(view, "\n") {
		if strings.Contains(line, "root") {
			matchLine = line
			break
		}
	}
	if !strings.Contains(matchLine, "▌") || !strings.Contains(matchLine, "o") {
		t.Errorf("matching row should have the marker and a graph glyph, got %q", matchLine)
	}
}

func TestLogPane_SearchJumpAndWrap(t *testing.T) {
	p := newLogPane()
	commits := longHistory(10) // subjects "ch000".."ch009"
	commits[2].Body = "mentions needle here"
	commits[7].Refs = []model.Ref{{Name: "needle-branch", Kind: model.RefLocalBranch}}
	p.setSize(80, 20)
	p.setCommits(commits, graph.Layout(commits))

	p.setSearch("NEEDLE") // case-insensitive; matches body and ref names
	if p.search.count != 2 {
		t.Fatalf("expected 2 matches (body + ref), got %d", p.search.count)
	}
	steps := []struct {
		forward bool
		want    int
	}{{true, 2}, {true, 7}, {true, 2}, {false, 7}, {false, 2}}
	for i, st := range steps {
		if !p.jumpToMatch(st.forward, i == 0) {
			t.Fatalf("step %d: no match found", i)
		}
		if p.cursor != st.want {
			t.Errorf("step %d: cursor %d, want %d", i, p.cursor, st.want)
		}
	}
	if got := p.searchStatus(); !strings.Contains(got, "1/2") {
		t.Errorf("status should show position, got %q", got)
	}

	p.setSearch("zzz")
	if p.jumpToMatch(true, false) {
		t.Error("no match expected")
	}
	if got := p.searchStatus(); !strings.Contains(got, "no matches") {
		t.Errorf("status should say no matches, got %q", got)
	}
}

func TestLogPane_SearchExtendsOnAppend(t *testing.T) {
	p := newLogPane()
	commits := longHistory(10)
	p.setSize(80, 20)
	p.setCommits(commits[:5], graph.Layout(commits[:5]))
	p.setSearch("ch007")
	if p.search.count != 0 {
		t.Fatalf("no match expected yet, got %d", p.search.count)
	}
	p.appendCommits(commits[5:], graph.Layout(commits))
	if p.search.count != 1 || !p.search.matches(7) {
		t.Errorf("match in appended page not found: count=%d", p.search.count)
	}
}

func TestHighlightMatches(t *testing.T) {
	old := lipgloss.ColorProfile()
	lipgloss.SetColorProfile(termenv.ANSI256)
	defer lipgloss.SetColorProfile(old)

	got := highlightMatches("Fix the Fix", "fix", lipgloss.NewStyle())
	if strings.Count(got, "\x1b[") < 4 || ansi.Strip(got) != "Fix the Fix" {
		t.Errorf("unexpected highlight output %q", got)
	}
	if got := highlightMatches("plain", "", lipgloss.NewStyle()); got != "plain" {
		t.Errorf("empty query must not alter text, got %q", got)
	}
}

func TestKeyMapHelpIsComplete(t *testing.T) {
	seen := map[string]bool{}
	for _, g := range keys.helpGroups() {
		for _, b := range g.bindings {
			h := b.Help()
			if h.Key == "" || h.Desc == "" {
				t.Errorf("binding in %q has no help text: %+v", g.title, b)
			}
			seen[h.Key] = true
		}
	}
	for _, want := range []string{"?", "n", "N", "/"} {
		if !seen[want] {
			t.Errorf("help overlay is missing %q", want)
		}
	}
	if out := renderHelpOverlay(); !strings.Contains(out, "next match") {
		t.Errorf("overlay should list bindings, got %q", out)
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

// TestLogPane_CursorStaysVisible walks the cursor through the whole list
// (and back) and checks the selected commit is always inside the rendered
// window, and the view never exceeds the pane height. Connector rows make
// some commits two lines tall, so scrolling must count lines, not commits.
func TestLogPane_CursorStaysVisible(t *testing.T) {
	check := func(p *logPane, step string) {
		t.Helper()
		view := p.View(false)
		if h := strings.Count(view, "\n") + 1; h > p.height {
			t.Fatalf("%s: view is %d lines, pane height %d", step, h, p.height)
		}
		if c := p.selectedCommit(); c == nil || !strings.Contains(view, c.ShortHash) {
			t.Fatalf("%s: selected commit (cursor %d, offset %d) not visible in view:\n%s", step, p.cursor, p.offset, view)
		}
	}
	for _, h := range []int{4, 6, 9} {
		p := newLogPane()
		commits := longHistory(40)
		p.setSize(80, h)
		p.setCommits(commits, graph.Layout(commits))
		check(&p, "initial")
		for range len(p.commits) {
			p.moveDown()
			check(&p, "down")
		}
		p.pageUp()
		check(&p, "pageUp")
		for range len(p.commits) {
			p.moveUp()
			check(&p, "up")
		}
		p.moveBottom()
		check(&p, "bottom")
		p.pageUp()
		check(&p, "pageUp2")
		p.pageDown()
		check(&p, "pageDown")
		p.moveTop()
		check(&p, "top")
	}
}

// longHistory builds n commits (newest first) where every third commit is a
// merge with a side-branch parent, so the graph has many connector rows.
func longHistory(n int) []model.Commit {
	commits := make([]model.Commit, n)
	name := func(i int) string { return fmt.Sprintf("h%03d", i) }
	for i := range n {
		c := model.Commit{Hash: name(i), ShortHash: name(i), Subject: "c" + name(i)}
		if i+1 < n {
			c.Parents = []string{name(i + 1)}
			if i%3 == 0 && i+3 < n {
				c.Parents = append(c.Parents, name(i+3))
			}
		}
		commits[i] = c
	}
	return commits
}

// TestHighlightRow_KeepsForegroundColors checks the selection background is
// re-applied after every inner reset (so it spans the row) while the row's
// own foreground colors survive.
func TestHighlightRow_KeepsForegroundColors(t *testing.T) {
	old := lipgloss.ColorProfile()
	lipgloss.SetColorProfile(termenv.ANSI256)
	defer lipgloss.SetColorProfile(old)

	inner := lipgloss.NewStyle().Foreground(lipgloss.Color("196")).Render("red") + " plain"
	got := highlightRow(inner, 20)

	bg, _, _ := strings.Cut(lipgloss.NewStyle().Background(colorSelectedBg).Render("x"), "x")
	if bg == "" {
		t.Fatal("expected a background sequence under ANSI256")
	}
	if !strings.HasPrefix(got, bg) {
		t.Errorf("row should start with the background sequence")
	}
	if !strings.Contains(got, "\x1b[0m"+bg) {
		t.Errorf("background must be re-applied after inner resets: %q", got)
	}
	if !strings.Contains(got, "196") {
		t.Errorf("inner foreground color was lost: %q", got)
	}
	if w := lipgloss.Width(got); w != 20 {
		t.Errorf("row should be padded to the pane width, got %d", w)
	}
}

// TestLogPane_ViewFillsHeight checks a short list is padded to the pane
// height, so the footer below it stays at the bottom of the terminal.
func TestLogPane_ViewFillsHeight(t *testing.T) {
	p := newTestLogPane() // 4 commits in a 20-line pane
	if got := strings.Count(p.View(false), "\n") + 1; got != p.height {
		t.Errorf("view is %d lines, want pane height %d", got, p.height)
	}
}

func TestLogPane_JumpParentChildHead(t *testing.T) {
	p := newTestLogPane() // d(merge b,c) c b(parent a) a
	p.commits[2].Refs = []model.Ref{{Name: "main", Kind: model.RefHead, IsHead: true}}

	if !p.jumpToHead() || p.commits[p.cursor].Hash != "b" {
		t.Fatalf("H should land on b, cursor=%d", p.cursor)
	}
	if !p.jumpToParent() || p.commits[p.cursor].Hash != "a" {
		t.Fatalf("parent of b is a, cursor=%d", p.cursor)
	}
	if p.jumpToParent() {
		t.Error("a root commit has no parent to jump to")
	}
	if !p.jumpToChild() || p.commits[p.cursor].Hash != "b" {
		t.Fatalf("child of a is b, cursor=%d", p.cursor)
	}
	p.setCursor(0, 0)
	if p.jumpToChild() {
		t.Error("the top commit has no child above it")
	}
	// A merge follows its FIRST parent (b), not the side branch (c).
	if !p.jumpToParent() || p.commits[p.cursor].Hash != "b" {
		t.Fatalf("merge d should follow first parent b, cursor=%d", p.cursor)
	}
}

func TestApplyColors(t *testing.T) {
	origHash, origLanes := colorHash, graph.LanePalette
	defer func() {
		colorHash = origHash
		graph.LanePalette = origLanes
		buildStyles()
	}()

	if err := ApplyColors(map[string]string{"hash": "#112233", "nope": "1"}, nil); err == nil {
		t.Error("unknown color name should be rejected")
	}
	if err := ApplyColors(map[string]string{"hash": "red"}, nil); err == nil {
		t.Error("invalid color value should be rejected")
	}
	if err := ApplyColors(nil, []string{"1", "300"}); err == nil {
		t.Error("out-of-range lane color should be rejected")
	}
	if colorHash != origHash {
		t.Fatal("a failed ApplyColors must not change anything")
	}

	if err := ApplyColors(map[string]string{"hash": "#112233"}, []string{"1", "#00ff00"}); err != nil {
		t.Fatal(err)
	}
	if colorHash.Dark != "#112233" || colorHash.Light != "#112233" {
		t.Errorf("hash color not applied: %+v", colorHash)
	}
	if len(graph.LanePalette) != 2 || graph.LanePalette[1].Dark != "#00ff00" {
		t.Errorf("lane palette not applied: %+v", graph.LanePalette)
	}
}

func TestLogPane_JumpToNthParent(t *testing.T) {
	p := newTestLogPane() // d(merge b,c) c b(parent a) a
	p.setCursor(0, 0)

	if !p.jumpToNthParent(1) || p.commits[p.cursor].Hash != "c" {
		t.Fatalf("second parent of the merge is c, cursor=%d", p.cursor)
	}
	p.setCursor(0, 0)
	if p.jumpToNthParent(2) || p.cursor != 0 {
		t.Error("a two-parent merge has no third parent")
	}
	if !p.jumpToNthParent(0) || p.commits[p.cursor].Hash != "b" {
		t.Fatalf("first parent is b, cursor=%d", p.cursor)
	}
}

func TestLogPane_JumpToNthParentAbove(t *testing.T) {
	// Clock skew: git log (date order) lists the side parent c above d.
	p := newTestLogPane()
	p.commits = []model.Commit{
		{Hash: "c", Subject: "side"},
		{Hash: "d", Subject: "merge", Parents: []string{"b", "c"}},
		{Hash: "b", Subject: "main"},
	}
	p.setCursor(1, 0)
	if !p.jumpToNthParent(1) || p.commits[p.cursor].Hash != "c" {
		t.Fatalf("a parent listed above its child must still be found, cursor=%d", p.cursor)
	}
}
