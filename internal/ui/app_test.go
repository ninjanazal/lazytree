package ui

import (
	"errors"
	"fmt"
	"reflect"
	"strings"
	"testing"

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

	updated, _ = m.Update(MsgCommitsBatch{Commits: page1, Offset: 1, Gen: gen1})
	m = updated.(AppModel)
	if len(m.log.commits) != 2 {
		t.Fatalf("expected 2 commits loaded, got %d", len(m.log.commits))
	}

	// A newer reload starts (e.g. a background fetch or the --all toggle).
	m.reloadGen++
	gen2 := m.reloadGen

	// A stale batch from gen1, still in flight, must be dropped.
	stalePage := []model.Commit{{Hash: "stale", Parents: nil}}
	updated, _ = m.Update(MsgCommitsBatch{Commits: stalePage, Offset: 2, Gen: gen1})
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

	updated, _ = m.Update(MsgCommitsBatch{Commits: gen2Page1, Offset: 1, Gen: gen2})
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
	updated, _ := m.Update(MsgCommitsBatch{Commits: more, Offset: len(m.log.commits), Gen: m.reloadGen})
	m = updated.(AppModel)

	want := graph.Layout(append(append([]model.Commit{}, branchCommits()...), more...))
	if !reflect.DeepEqual(m.log.layout, want) {
		t.Errorf("got layout %+v, want %+v", m.log.layout, want)
	}
}
