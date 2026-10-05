package ui

import (
	"errors"
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/eurico-martins/lazytree/internal/git"
	"github.com/eurico-martins/lazytree/internal/graph"
	"github.com/eurico-martins/lazytree/internal/model"
)

func newSizedTestApp(t *testing.T, width, height int) AppModel {
	t.Helper()
	m := NewApp(&git.Runner{RepoPath: "."})
	updated, _ := m.Update(tea.WindowSizeMsg{Width: width, Height: height})
	m = updated.(AppModel)
	commits := branchCommits()
	m.log.setCommits(commits, graph.Layout(commits))
	return m
}

// TestMsgError_IsNonFatal is the M1 "Done when" regression test: a failed
// commits load must show a dismissible banner, not blank out the rest of
// the UI. See internal/ui/banner.go: statusBanner.
func TestMsgError_IsNonFatal(t *testing.T) {
	m := newSizedTestApp(t, 80, 24)

	before := m.View()
	if strings.Contains(before, "⚠") {
		t.Fatalf("no error yet, but banner glyph already present:\n%s", before)
	}

	updated, _ := m.Update(MsgError{Gen: m.reloadGen, Err: errors.New("fatal: bad object deadbeef")})
	m = updated.(AppModel)

	view := m.View()
	if !strings.Contains(view, "bad object deadbeef") {
		t.Fatalf("error banner missing from view:\n%s", view)
	}
	if !strings.Contains(view, "root") {
		// "root" is one of branchCommits()' subjects: the log pane must
		// still be visible underneath the banner, not replaced by it.
		t.Fatalf("log pane no longer visible once an error banner is shown:\n%s", view)
	}
}

// TestMsgError_ClearedByBack mirrors the Esc/Back dismissal wired in
// updateKeys: pressing Back clears a stale error banner.
func TestMsgError_ClearedByBack(t *testing.T) {
	m := newSizedTestApp(t, 80, 24)

	updated, _ := m.Update(MsgError{Gen: m.reloadGen, Err: errors.New("boom")})
	m = updated.(AppModel)
	if !strings.Contains(m.View(), "boom") {
		t.Fatalf("expected banner after MsgError")
	}

	updated, _ = m.updateKeys(tea.KeyMsg{Type: tea.KeyEsc})
	m = updated.(AppModel)
	if strings.Contains(m.View(), "boom") {
		t.Fatalf("expected banner to be cleared after Back/Esc, got:\n%s", m.View())
	}
}

// TestMsgError_StaleGenIgnored is a regression test: an error from a load
// that a newer reload has already superseded must not stop the spinner or
// raise a banner for a request nobody is waiting on anymore. Mirrors how
// MsgCommitsLoaded/MsgCommitsBatch already guard on Gen.
func TestMsgError_StaleGenIgnored(t *testing.T) {
	m := newSizedTestApp(t, 80, 24)
	staleGen := m.reloadGen
	m.reloadGen++ // a newer reload (e.g. triggered by a background fetch) started
	m.loading = true

	updated, _ := m.Update(MsgError{Gen: staleGen, Err: errors.New("stale failure")})
	m = updated.(AppModel)

	if strings.Contains(m.View(), "stale failure") {
		t.Errorf("a stale MsgError raised the banner:\n%s", m.View())
	}
	if !m.loading {
		t.Error("a stale MsgError cleared m.loading while a newer reload is still in flight")
	}
}

// TestDiffError_DoesNotRaiseAppBanner and TestCommitsError_DoesNotClearDiffError
// together are the regression test for two failures being independent: a
// failed diff load shows inside the diff pane, not the shared banner, and a
// later commits-load success/failure doesn't erase an unrelated diff error
// (or vice versa). Before this, both used the single app-wide MsgError/m.err,
// so whichever happened last silently won.
func TestDiffError_DoesNotRaiseAppBanner(t *testing.T) {
	m := newSizedTestApp(t, 80, 24)

	updated, _ := m.Update(MsgDiffLoaded{Hash: m.log.selectedCommit().Hash, Err: errors.New("bad diff")})
	m = updated.(AppModel)

	if strings.Contains(m.View(), "⚠") || strings.Contains(m.View(), "bad diff") {
		t.Errorf("a diff error leaked into the app-wide banner:\n%s", m.View())
	}
	if !strings.Contains(m.diff.renderContent(), "bad diff") {
		t.Errorf("diff pane does not show its own load error: %q", m.diff.renderContent())
	}
}

func TestCommitsError_DoesNotClearDiffError(t *testing.T) {
	m := newSizedTestApp(t, 80, 24)
	hash := m.log.selectedCommit().Hash

	updated, _ := m.Update(MsgDiffLoaded{Hash: hash, Err: errors.New("bad diff")})
	m = updated.(AppModel)

	updated, _ = m.Update(MsgCommitsLoaded{Commits: m.log.commits, Layout: m.log.layout, Gen: m.reloadGen})
	m = updated.(AppModel)

	if !strings.Contains(m.diff.renderContent(), "bad diff") {
		t.Errorf("an unrelated successful commits load cleared the diff pane's error")
	}
}

// TestZenHeight_MatchesApplyLayout is a regression test: the 'z' key used to
// call m.zen.setSize(m.width, m.height) directly, bypassing applyLayout's
// height accounting for the fetch bar and status banner, so zen mode drew
// taller than the terminal until the next resize event.
func TestZenHeight_MatchesApplyLayout(t *testing.T) {
	m := newSizedTestApp(t, 80, 24)

	updated, _ := m.updateKeys(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("z")})
	m = updated.(AppModel)

	wantHeight := m.height - fetchBarHeight - bannerHeight
	if m.zen.height != wantHeight {
		t.Errorf("zen height = %d, want %d (m.height=%d)", m.zen.height, wantHeight, m.height)
	}
}

// TestZenExit_RecomputesLayout is a regression test: exiting zen mode (via
// either 'z' again or Back/Esc) used to leave the log pane's height exactly
// as applyLayout set it at zen-entry, relying on nothing relevant to that
// formula (m.width, m.height, m.mode, m.banner.Height()) changing while zen
// was active. That's true today, but only by coincidence, and it stops
// being true the moment something like a conditional-height banner is
// added. To make that explicit rather than relied upon, exiting zen now
// calls applyLayout unconditionally, which this test verifies directly by
// corrupting m.log.height beforehand and checking it gets recomputed.
func TestZenExit_RecomputesLayout(t *testing.T) {
	for _, exitKey := range []tea.KeyMsg{
		{Type: tea.KeyRunes, Runes: []rune("z")},
		{Type: tea.KeyEsc},
	} {
		m := newSizedTestApp(t, 80, 24)

		updated, _ := m.updateKeys(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("z")})
		m = updated.(AppModel)
		wantHeight := m.log.height

		m.log.height = -1 // corrupt it so we can tell whether exit recomputes it

		updated, _ = m.updateKeys(exitKey)
		m = updated.(AppModel)
		if m.mode != modeLog {
			t.Fatalf("exit key %v did not return to modeLog", exitKey)
		}
		if m.log.height != wantHeight {
			t.Errorf("exit key %v: log height = %d, want %d (applyLayout not called on zen exit)",
				exitKey, m.log.height, wantHeight)
		}
	}
}

// TestSearchMode_DoesNotAddOverflowBeyondBaseline is a regression test:
// entering search mode inserts a 3-row bordered search box between the log
// pane and the footer (see View()'s modeSearch branch). Before this fix,
// applyLayout didn't shrink the log pane to compensate, so search mode added
// a full searchHeight worth of extra rows on top of whatever modeLog was
// already rendering.
//
// This compares against the modeLog baseline rather than asserting an
// absolute fit against the terminal height: logPane.View() has a separate,
// pre-existing quirk (its returned string always ends in "\n", adding one
// phantom line when joined with "\n" elsewhere) that surfaces only when the
// pane is fully packed with commits, regardless of search mode. That's a
// different bug in different code; this test isolates the search-mode fix
// from it by checking the two view heights are equal, not by checking
// either one against a fixed number.
//
// This needs more commits than fit in one screen: the log pane only ever
// renders as many rows as it has commits (up to its height budget), so with
// too few commits it never fills its allotted height and neither bug shows.
func TestSearchMode_DoesNotAddOverflowBeyondBaseline(t *testing.T) {
	m := newSizedTestApp(t, 80, 24)
	commits := manyCommits(100)
	m.log.setCommits(commits, graph.Layout(commits))

	baselineLines := strings.Split(m.View(), "\n")

	updated, _ := m.updateKeys(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("/")})
	m = updated.(AppModel)
	if m.mode != modeSearch {
		t.Fatalf("expected modeSearch, got %v", m.mode)
	}
	searchLines := strings.Split(m.View(), "\n")

	if len(searchLines) != len(baselineLines) {
		t.Errorf("search mode view is %d lines tall, modeLog baseline is %d; entering search must not add net height",
			len(searchLines), len(baselineLines))
	}
}

func manyCommits(n int) []model.Commit {
	commits := make([]model.Commit, n)
	for i := range n {
		hash := fmt.Sprintf("h%d", i)
		var parents []string
		if i+1 < n {
			parents = []string{fmt.Sprintf("h%d", i+1)}
		}
		commits[i] = model.Commit{Hash: hash, ShortHash: hash, Subject: fmt.Sprintf("commit %d", i), Parents: parents}
	}
	return commits
}

// TestSearchMode_ExitRestoresLogHeight is a regression test: leaving search
// mode (via Esc) must give the log pane back the rows the search box used,
// not leave it permanently shrunk.
func TestSearchMode_ExitRestoresLogHeight(t *testing.T) {
	m := newSizedTestApp(t, 80, 24)
	originalHeight := m.log.height

	updated, _ := m.updateKeys(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("/")})
	m = updated.(AppModel)
	if m.log.height == originalHeight {
		t.Fatalf("expected log pane to shrink when entering search mode")
	}

	updated, _ = m.updateSearch(tea.KeyMsg{Type: tea.KeyEsc})
	m = updated.(AppModel)
	if m.log.height != originalHeight {
		t.Errorf("log height after leaving search = %d, want %d (restored)", m.log.height, originalHeight)
	}
}

// TestView_FitsTerminalHeight is a regression test: logPane.View() used to
// end with a trailing "\n", so a pane packed with commits rendered one row
// taller than the terminal once app.go joined it with the other blocks.
func TestView_FitsTerminalHeight(t *testing.T) {
	const height = 24
	for _, search := range []bool{false, true} {
		m := newSizedTestApp(t, 80, height)
		commits := manyCommits(100)
		m.log.setCommits(commits, graph.Layout(commits))
		if search {
			updated, _ := m.updateKeys(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("/")})
			m = updated.(AppModel)
		}
		if n := len(strings.Split(m.View(), "\n")); n != height {
			t.Errorf("search=%v: view is %d lines, want exactly %d", search, n, height)
		}
	}
}

// TestCommitsBatch_ReloadGenDropsStaleLayouter is a regression test for the
// incremental Layouter carried across MsgCommitsBatch pages: a batch from a
// reload that's since been superseded must not touch the new reload's lane
// state, and a fresh reload must produce the same layout a one-shot
// graph.Layout would.
func TestCommitsBatch_ReloadGenDropsStaleLayouter(t *testing.T) {
	m := newSizedTestApp(t, 80, 24)

	gen1 := m.reloadGen
	page0 := []model.Commit{{Hash: "b", Parents: []string{"a"}}}
	page1 := []model.Commit{{Hash: "a", Parents: nil}}

	lt1 := graph.NewLayouter()
	layout1 := lt1.Append(page0)
	updated, _ := m.Update(MsgCommitsLoaded{Commits: page0, Layout: layout1, Layouter: lt1, Gen: gen1})
	m = updated.(AppModel)

	updated, _ = m.Update(MsgCommitsBatch{Commits: page1, Gen: gen1})
	m = updated.(AppModel)
	if len(m.log.commits) != 2 {
		t.Fatalf("expected 2 commits loaded, got %d", len(m.log.commits))
	}

	// A newer reload starts (e.g. a background fetch or the --all toggle).
	m.reloadGen++
	gen2 := m.reloadGen

	// A stale batch from gen1, still in flight, must be dropped.
	stalePage := []model.Commit{{Hash: "stale", Parents: nil}}
	updated, _ = m.Update(MsgCommitsBatch{Commits: stalePage, Gen: gen1})
	m = updated.(AppModel)
	if len(m.log.commits) != 2 {
		t.Fatalf("stale batch was applied: now have %d commits", len(m.log.commits))
	}

	gen2Page0 := []model.Commit{{Hash: "y", Parents: []string{"x"}}}
	gen2Page1 := []model.Commit{{Hash: "x", Parents: nil}}

	lt2 := graph.NewLayouter()
	layout2 := lt2.Append(gen2Page0)
	updated, _ = m.Update(MsgCommitsLoaded{Commits: gen2Page0, Layout: layout2, Layouter: lt2, Gen: gen2})
	m = updated.(AppModel)

	updated, _ = m.Update(MsgCommitsBatch{Commits: gen2Page1, Gen: gen2})
	m = updated.(AppModel)

	wantAll := append(append([]model.Commit{}, gen2Page0...), gen2Page1...)
	want := graph.Layout(wantAll)
	if !reflect.DeepEqual(m.log.layout, want) {
		t.Errorf("got layout %+v, want %+v", m.log.layout, want)
	}
}

// TestToggle_ClearsLayouter ensures the --all toggle's reload doesn't leak
// the previous reload's lane-assignment state into the next one.
func TestToggle_ClearsLayouter(t *testing.T) {
	m := newSizedTestApp(t, 80, 24)
	m.layouter = graph.NewLayouter()
	m.layouter.Append(branchCommits())

	updated, _ := m.updateKeys(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("a")})
	m = updated.(AppModel)

	if m.layouter != nil {
		t.Error("expected m.layouter to be nil after the --all toggle reload")
	}
}

// TestCommitsBatch_NilLayouterFallback covers a MsgCommitsLoaded delivered
// without a Layouter (as newSizedTestApp's direct m.log.setCommits path
// does): a subsequent MsgCommitsBatch must still rebuild a correct layout.
func TestCommitsBatch_NilLayouterFallback(t *testing.T) {
	m := newSizedTestApp(t, 80, 24) // populates m.log.commits directly, m.layouter stays nil
	if m.layouter != nil {
		t.Fatalf("expected m.layouter to start nil")
	}

	more := []model.Commit{{Hash: "z", Parents: nil}}
	updated, _ := m.Update(MsgCommitsBatch{Commits: more, Gen: m.reloadGen})
	m = updated.(AppModel)

	want := graph.Layout(append(append([]model.Commit{}, branchCommits()...), more...))
	if !reflect.DeepEqual(m.log.layout, want) {
		t.Errorf("got layout %+v, want %+v", m.log.layout, want)
	}
}

func press(t *testing.T, m AppModel, k string) AppModel {
	t.Helper()
	var msg tea.KeyMsg
	switch k {
	case "enter":
		msg = tea.KeyMsg{Type: tea.KeyEnter}
	case "esc":
		msg = tea.KeyMsg{Type: tea.KeyEsc}
	default:
		msg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(k)}
	}
	updated, _ := m.Update(msg)
	return updated.(AppModel)
}

func TestApp_HelpOverlayToggles(t *testing.T) {
	m := newSizedTestApp(t, 100, 40)
	m = press(t, m, "?")
	if !strings.Contains(m.View(), "next match") {
		t.Fatalf("? should open the key overlay:\n%s", m.View())
	}
	m = press(t, m, "j") // swallowed while the overlay is open
	if m.log.cursor != 0 {
		t.Errorf("keys must not reach the log while help is open, cursor=%d", m.log.cursor)
	}
	m = press(t, m, "esc")
	if strings.Contains(m.View(), "next match") {
		t.Error("esc should close the overlay")
	}
}

func TestApp_SearchJumpsAndEscRestores(t *testing.T) {
	m := newSizedTestApp(t, 100, 40) // subjects: merge feature, feature work, main work, root
	m = press(t, m, "/")
	for _, r := range "work" {
		m = press(t, m, string(r))
	}
	if m.log.cursor != 1 {
		t.Fatalf("incremental search should land on first match (1), got %d", m.log.cursor)
	}
	m = press(t, m, "enter")
	m = press(t, m, "n")
	if m.log.cursor != 2 {
		t.Errorf("n should go to the next match (2), got %d", m.log.cursor)
	}
	m = press(t, m, "N")
	if m.log.cursor != 1 {
		t.Errorf("N should go back to (1), got %d", m.log.cursor)
	}
	m = press(t, m, "esc") // clears highlights, keeps position
	if m.log.searchActive() {
		t.Error("esc in the log view should clear the search")
	}

	// Cancelling while typing restores where the search began.
	m = press(t, m, "G")
	m = press(t, m, "/")
	m = press(t, m, "m")
	m = press(t, m, "esc")
	if m.log.cursor != 3 {
		t.Errorf("esc during search should restore the cursor (3), got %d", m.log.cursor)
	}
}

func TestApp_MouseWheelAndClick(t *testing.T) {
	m := newSizedTestApp(t, 100, 40)

	updated, _ := m.Update(tea.MouseMsg{Action: tea.MouseActionPress, Button: tea.MouseButtonWheelDown})
	m = updated.(AppModel)
	if m.log.cursor == 0 {
		t.Error("wheel down should move the selection")
	}

	// Click on the third commit line: y = toolbar(1) + header(1) + line.
	target := m.log.rowAtLine(2)
	updated, _ = m.Update(tea.MouseMsg{Action: tea.MouseActionPress, Button: tea.MouseButtonLeft, X: 10, Y: toolbarHeight + 1 + 2})
	m = updated.(AppModel)
	if m.log.cursor != target {
		t.Errorf("click should select row %d, got %d", target, m.log.cursor)
	}

	// A click below the last row selects nothing.
	before := m.log.cursor
	updated, _ = m.Update(tea.MouseMsg{Action: tea.MouseActionPress, Button: tea.MouseButtonLeft, Y: 30})
	m = updated.(AppModel)
	if m.log.cursor != before {
		t.Errorf("click on empty space moved the cursor to %d", m.log.cursor)
	}
}

func TestParseDeepQuery(t *testing.T) {
	cases := []struct {
		in   string
		ok   bool
		term string
	}{
		{"g:fix crash", true, "fix crash"},
		{"s:needle", true, "needle"},
		{"p:internal/ui", true, "internal/ui"},
		{"g:", false, ""},
		{"x:foo", false, ""},
		{"plain", false, ""},
		{"", false, ""},
	}
	for _, c := range cases {
		_, term, ok := parseDeepQuery(c.in)
		if ok != c.ok || term != c.term {
			t.Errorf("parseDeepQuery(%q) = %q, %v; want %q, %v", c.in, term, ok, c.term, c.ok)
		}
	}
}

func TestApp_DeepSearchResultMarksMatches(t *testing.T) {
	m := newSizedTestApp(t, 100, 40) // hashes d c b a
	m = press(t, m, "/")
	for _, r := range "g:x" {
		m = press(t, m, string(r))
	}
	if m.log.searchActive() {
		t.Error("a deep query must not run an in-memory search while typing")
	}
	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = updated.(AppModel)
	if cmd == nil || !m.deepPending {
		t.Fatal("enter should start a git-side search")
	}
	if got := m.searchStatus(); got != "searching…" {
		t.Errorf("footer should show progress, got %q", got)
	}

	// A stale result is ignored; the current one marks commits b and c.
	updated, _ = m.Update(MsgDeepSearch{Seq: m.deepSeq - 1, Label: "g:x", Hashes: []string{"a"}})
	m = updated.(AppModel)
	if m.log.searchActive() {
		t.Error("stale result must be dropped")
	}
	updated, _ = m.Update(MsgDeepSearch{Seq: m.deepSeq, Label: "g:x", Hashes: []string{"c", "b"}})
	m = updated.(AppModel)
	if m.deepPending || m.log.search.count != 2 {
		t.Fatalf("expected 2 matches and no pending search, got count=%d pending=%v", m.log.search.count, m.deepPending)
	}
	if m.log.cursor != 1 {
		t.Errorf("cursor should jump to the first match (c, row 1), got %d", m.log.cursor)
	}
	m = press(t, m, "n")
	if m.log.cursor != 2 {
		t.Errorf("n should move to the next match (b, row 2), got %d", m.log.cursor)
	}
	if st := m.searchStatus(); !strings.Contains(st, "g:x") || !strings.Contains(st, "2/2") {
		t.Errorf("status should show label and position, got %q", st)
	}
	m = press(t, m, "esc")
	if m.log.searchActive() {
		t.Error("esc should clear a deep search")
	}
}

func TestApp_FlashShowsAndClears(t *testing.T) {
	m := newSizedTestApp(t, 100, 40)
	updated, cmd := m.Update(MsgFlash{Text: "copied abc1234"})
	m = updated.(AppModel)
	if cmd == nil {
		t.Fatal("a flash must schedule its own clearing")
	}
	if !strings.Contains(m.View(), "copied abc1234") {
		t.Errorf("flash missing from footer:\n%s", m.View())
	}
	// A stale clear (older flash) must not remove the current message.
	updated, _ = m.Update(MsgFlashClear{Seq: m.flashSeq - 1})
	m = updated.(AppModel)
	if m.flashText == "" {
		t.Error("stale clear removed the newer flash")
	}
	updated, _ = m.Update(MsgFlashClear{Seq: m.flashSeq})
	m = updated.(AppModel)
	if m.flashText != "" {
		t.Error("matching clear should remove the flash")
	}
}

func TestApp_CopyAndEditNeedACommit(t *testing.T) {
	m := newSizedTestApp(t, 100, 40)
	if _, cmd := m.updateLogKeys(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("y")}); cmd == nil {
		t.Error("y should return a copy command for the selected commit")
	}
	if _, cmd := m.updateLogKeys(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("o")}); cmd == nil {
		t.Error("o should return an editor command for the selected commit")
	}
	empty := NewApp(&git.Runner{RepoPath: "."})
	if _, cmd := empty.updateLogKeys(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("y")}); cmd != nil {
		t.Error("y with no commits must do nothing")
	}
}

func TestEditorCommandPrefersVisual(t *testing.T) {
	t.Setenv("VISUAL", "code -w")
	t.Setenv("EDITOR", "nano")
	if got := editorCommand(); len(got) != 2 || got[0] != "code" || got[1] != "-w" {
		t.Errorf("VISUAL should win and keep its args, got %v", got)
	}
	t.Setenv("VISUAL", "")
	if got := editorCommand(); got[0] != "nano" {
		t.Errorf("EDITOR fallback, got %v", got)
	}
	t.Setenv("EDITOR", "")
	if got := editorCommand(); got[0] != "vi" {
		t.Errorf("vi fallback, got %v", got)
	}
}

func TestApp_FetchDisabledByOptions(t *testing.T) {
	m := NewAppWithOptions(&git.Runner{RepoPath: "."}, Options{FetchInterval: 0, ShowAll: false})
	if m.showAll {
		t.Error("ShowAll option should be respected")
	}
	if !m.nextFetchAt.IsZero() {
		t.Error("no fetch should be scheduled when the interval is 0")
	}
	bar := renderFetchBar(20, false, m.nextFetchAt, 0)
	if strings.Contains(bar, "\x1b[") == false && strings.Count(bar, "─") != 20 {
		t.Errorf("disabled fetch bar should be empty, got %q", bar)
	}
}

func TestApplyKeys(t *testing.T) {
	orig := keys
	defer func() { keys = orig }()

	for name, ov := range map[string]map[string][]string{
		"unknown action": {"fly": {"f"}},
		"empty list":     {"zen": {}},
		"empty key":      {"zen": {""}},
		"conflict":       {"zen": {"q"}}, // q is quit
		"reserved digit": {"zen": {"2"}}, // 1-9 jump to the nth parent
	} {
		if err := ApplyKeys(ov); err == nil {
			t.Errorf("%s: expected an error", name)
		}
	}
	if keys.Zen.Keys()[0] != "z" {
		t.Fatal("a rejected override must not change any binding")
	}

	// Swapping two actions' keys is not a conflict.
	if err := ApplyKeys(map[string][]string{"zen": {"x"}, "toggle_refs": {"z"}, "quit": {"Q"}}); err != nil {
		t.Fatal(err)
	}
	m := newSizedTestApp(t, 100, 40)
	m = press(t, m, "z")
	if m.mode == modeZen {
		t.Error("z now toggles refs, not zen")
	}
	m = press(t, m, "x")
	if m.mode != modeZen {
		t.Error("x should now enter zen mode")
	}
	if !key.Matches(tea.KeyMsg{Type: tea.KeyCtrlC}, keys.Quit) {
		t.Error("ctrl+c must always quit")
	}
	if h := keys.Zen.Help().Key; h != "x" {
		t.Errorf("help label should follow the new key, got %q", h)
	}
}

func TestApp_RefFilterKeysAndToolbar(t *testing.T) {
	m := newSizedTestApp(t, 120, 40)
	m.log.commits[0].Refs = []model.Ref{
		{Name: "v9", Kind: model.RefTag},
		{Name: "origin/dev", Kind: model.RefRemoteBranch},
	}
	m.log.setCommits(m.log.commits, m.log.layout)

	view := m.View()
	if !strings.Contains(view, "tag: v9") || !strings.Contains(view, "origin/dev") {
		t.Fatalf("both pills should show by default:\n%s", view)
	}
	m = press(t, m, "t")
	view = m.View()
	if strings.Contains(view, "tag: v9") || !strings.Contains(view, "origin/dev") || !strings.Contains(view, "no tags") {
		t.Errorf("t should hide tags only and say so in the toolbar:\n%s", view)
	}
	m = press(t, m, "r")
	if strings.Contains(m.View(), "origin/dev") {
		t.Error("r should hide remote branches")
	}
	m = press(t, m, "t")
	m = press(t, m, "r")
	if !strings.Contains(m.View(), "tag: v9") {
		t.Error("toggling again should bring pills back")
	}
}

func TestApp_SplitViewLayoutAndFocus(t *testing.T) {
	m := newSizedTestApp(t, 160, 40) // "auto" layout splits at >= 140 columns
	if !m.splitActive() {
		t.Fatal("a 160-column terminal should use the split layout")
	}
	view := m.View()
	if h := strings.Count(view, "\n") + 1; h != 40 {
		t.Errorf("split view is %d lines, want the terminal height 40", h)
	}
	if !strings.Contains(view, "Loading diff") {
		t.Errorf("the diff pane should be visible next to the log:\n%s", view)
	}

	// Moving the cursor requests the diff of the newly selected commit.
	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")})
	m = updated.(AppModel)
	if cmd == nil {
		t.Error("in split view, moving must schedule a diff load")
	}

	// The diff pane shows the inspector for the loaded commit.
	c := m.log.selectedCommit()
	updated, _ = m.Update(MsgDiffLoaded{Hash: c.Hash, Files: []model.DiffFile{{
		NewPath: "x.go", Status: "M",
		Hunks: []model.DiffHunk{{Header: "@@ -1 +1 @@", Lines: []model.DiffLine{{Kind: model.DiffAdded, Text: "+hello", NewN: 1}}}},
	}}})
	m = updated.(AppModel)
	if v := m.View(); !strings.Contains(v, "hello") || !strings.Contains(v, c.Subject) {
		t.Errorf("split view should show the selected commit's diff and header:\n%s", v)
	}

	// enter focuses the diff (keys scroll it); esc returns to the log.
	m = press(t, m, "enter")
	if !m.popupOpen {
		t.Error("enter should focus the diff pane in split view")
	}
	before := m.log.cursor
	m = press(t, m, "j")
	if m.log.cursor != before {
		t.Error("with the diff focused, j must not move the log cursor")
	}
	m = press(t, m, "esc")
	if m.popupOpen {
		t.Error("esc should return focus to the log")
	}

	// v switches to the popup layout and back.
	m = press(t, m, "v")
	if m.splitActive() {
		t.Error("v should switch to popup layout")
	}
	m = press(t, m, "v")
	if !m.splitActive() {
		t.Error("v again should return to split")
	}
}

func TestApp_SplitViewNeedsWidth(t *testing.T) {
	if newSizedTestApp(t, 120, 40).splitActive() {
		t.Error("auto layout must not split below 140 columns")
	}
	m := NewAppWithOptions(&git.Runner{RepoPath: "."}, Options{Layout: "split", FetchInterval: 0})
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 90, Height: 30})
	if updated.(AppModel).splitActive() {
		t.Error("even a forced split needs 100 columns")
	}
	pm := NewAppWithOptions(&git.Runner{RepoPath: "."}, Options{Layout: "popup"})
	updated, _ = pm.Update(tea.WindowSizeMsg{Width: 200, Height: 50})
	if updated.(AppModel).splitActive() {
		t.Error("layout = popup must never split")
	}
}

func TestApp_SplitViewMouse(t *testing.T) {
	m := newSizedTestApp(t, 160, 40)
	diffX := sideMargin + m.log.width + 5

	updated, _ := m.Update(tea.MouseMsg{Action: tea.MouseActionPress, Button: tea.MouseButtonLeft, X: diffX, Y: 5})
	m = updated.(AppModel)
	if !m.popupOpen {
		t.Error("clicking the diff pane should focus it")
	}
	updated, _ = m.Update(tea.MouseMsg{Action: tea.MouseActionPress, Button: tea.MouseButtonLeft, X: 10, Y: toolbarHeight + 1 + 1})
	m = updated.(AppModel)
	if m.popupOpen {
		t.Error("clicking the log should return focus to it")
	}
	if m.log.cursor != m.log.rowAtLine(1) {
		t.Errorf("click in the log should still select a row, cursor=%d", m.log.cursor)
	}
}

func newMergeTestApp(t *testing.T) AppModel {
	t.Helper()
	m := newSizedTestApp(t, 80, 24)
	commits := []model.Commit{
		{Hash: "d", Subject: "merge", Parents: []string{"b", "c"}},
		{Hash: "c", Subject: "side", Parents: []string{"a"}},
		{Hash: "b", Subject: "main", Parents: []string{"a"}},
		{Hash: "a", Subject: "root", Parents: []string{"notloaded"}},
	}
	m.log.setCommits(commits, graph.Layout(commits))
	m.log.setCursor(0, 0)
	return m
}

func TestNthParentKeysInLog(t *testing.T) {
	m := newMergeTestApp(t)

	updated, _ := m.Update(keyMsg("2"))
	m = updated.(AppModel)
	if got := m.log.commits[m.log.cursor].Hash; got != "c" {
		t.Fatalf("2 should jump to the second parent c, got %s", got)
	}
	if m.flashText != "" {
		t.Errorf("a successful jump should not flash, got %q", m.flashText)
	}

	// c has one parent: 3 can't jump and says why.
	updated, cmd := m.Update(keyMsg("3"))
	m = updated.(AppModel)
	if got := m.log.commits[m.log.cursor].Hash; got != "c" || cmd == nil || !strings.Contains(m.flashText, "no parent 3") {
		t.Errorf("missing parent: cursor=%s flash=%q cmd=%v", got, m.flashText, cmd != nil)
	}

	// a's parent hasn't streamed in yet.
	m.log.setCursor(3, 0)
	updated, _ = m.Update(keyMsg("p"))
	if f := updated.(AppModel).flashText; !strings.Contains(f, "not loaded") {
		t.Errorf("unloaded parent should flash, got %q", f)
	}
}

func TestNthParentKeysInPopup(t *testing.T) {
	m := newMergeTestApp(t)
	m.popupOpen = true

	updated, cmd := m.Update(keyMsg("2"))
	m = updated.(AppModel)
	if got := m.log.commits[m.log.cursor].Hash; got != "c" {
		t.Fatalf("2 should jump to the second parent c, got %s", got)
	}
	if cmd == nil {
		t.Error("jumping inside the popup must load the new commit's diff")
	}
	if !m.popupOpen {
		t.Error("the popup should stay open after a jump")
	}
	updated, _ = m.Update(keyMsg("9"))
	m = updated.(AppModel)
	if got := m.log.commits[m.log.cursor].Hash; got != "c" || !strings.Contains(m.flashText, "no parent 9") {
		t.Errorf("a missing parent must not move the cursor and must flash, got %s %q", got, m.flashText)
	}
}
