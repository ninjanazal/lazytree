package ui

import (
	"context"
	"fmt"
	"path/filepath"
	"time"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/eurico-martins/lazytree/internal/git"
	graphpkg "github.com/eurico-martins/lazytree/internal/graph"
	"github.com/eurico-martins/lazytree/internal/model"
)

const pageSize = 500
const diffDebounce = 150 * time.Millisecond
const fetchInterval = 60 * time.Second
const fetchTimeout = 30 * time.Second
const animInterval = 120 * time.Millisecond
const countdownInterval = 1 * time.Second

type viewMode int

const (
	modeLog viewMode = iota
	modeSearch
	modeZen
)

type AppModel struct {
	runner      *git.Runner
	log         logPane
	diff        diffPane
	search      searchModel
	zen         zenPane
	spinner     spinner.Model
	mode        viewMode
	popupOpen   bool
	width       int
	height      int
	loading     bool
	loadedCount int
	showAll     bool
	repoName    string
	branch      string
	banner      statusBanner
	debounceSeq int

	fetchSeq      int       // guards the self-rescheduling MsgFetchTick chain
	fetchInFlight bool      // true while a background `git fetch` is running
	nextFetchAt   time.Time // when the next background fetch is scheduled to run
	reloadGen     int       // guards stale MsgCommitsLoaded/MsgCommitsBatch from a
	// reload superseded by a newer one (Toggle vs. background fetch)
	layouter *graphpkg.Layouter // carries lane state across MsgCommitsBatch pages for the current reloadGen

	animSeq int // guards the self-rescheduling MsgAnimationTick chain (zen mode)
	tickSeq int // guards the self-rescheduling MsgCountdownTick chain (footer redraw)
}

func NewApp(runner *git.Runner) AppModel {
	sp := spinner.New()
	sp.Spinner = spinner.Dot

	return AppModel{
		runner:      runner,
		log:         newLogPane(),
		diff:        newDiffPane(),
		search:      newSearchModel(),
		zen:         newZenPane(),
		spinner:     sp,
		showAll:     true,
		repoName:    filepath.Base(runner.RepoPath),
		fetchSeq:    1,
		nextFetchAt: time.Now().Add(fetchInterval),
		tickSeq:     1,
	}
}

// Init has a value receiver, like Update and View: Bubble Tea only keeps the
// tea.Cmd it returns, not any mutation made to m, so the initial fetchSeq/
// tickSeq/nextFetchAt values must already be set by NewApp rather than by
// calling scheduleFetchTick/scheduleCountdownTick here (those mutate a copy
// that's discarded). The two tea.Tick calls below mirror what those methods
// build, using the seq NewApp already baked in.
func (m AppModel) Init() tea.Cmd {
	return tea.Batch(
		m.spinner.Tick,
		m.loadCommitsCmd(0, m.reloadGen),
		m.loadBranchCmd(),
		tea.Tick(fetchInterval, func(time.Time) tea.Msg {
			return MsgFetchTick{Seq: m.fetchSeq}
		}),
		tea.Tick(countdownInterval, func(time.Time) tea.Msg {
			return MsgCountdownTick{Seq: m.tickSeq}
		}),
	)
}

// scheduleFetchTick bumps the fetch sequence and returns a tea.Cmd that,
// after fetchInterval has elapsed, requests a background git fetch.
func (m *AppModel) scheduleFetchTick() tea.Cmd {
	m.fetchSeq++
	seq := m.fetchSeq
	m.nextFetchAt = time.Now().Add(fetchInterval)
	return tea.Tick(fetchInterval, func(time.Time) tea.Msg {
		return MsgFetchTick{Seq: seq}
	})
}

// scheduleCountdownTick bumps the countdown sequence and returns a tea.Cmd
// that, after countdownInterval has elapsed, forces a footer redraw so the
// "next fetch in Ns" countdown stays current. It reschedules itself for the
// lifetime of the app.
func (m *AppModel) scheduleCountdownTick() tea.Cmd {
	m.tickSeq++
	seq := m.tickSeq
	return tea.Tick(countdownInterval, func(time.Time) tea.Msg {
		return MsgCountdownTick{Seq: seq}
	})
}

// scheduleAnimTick bumps the animation sequence and returns a tea.Cmd that,
// after animInterval has elapsed, advances zen mode's grow/breathe
// animation. Callers stop the chain simply by not rescheduling once zen
// mode has been left.
func (m *AppModel) scheduleAnimTick() tea.Cmd {
	m.animSeq++
	seq := m.animSeq
	return tea.Tick(animInterval, func(time.Time) tea.Msg {
		return MsgAnimationTick{Seq: seq}
	})
}

// fetchCmd runs `git fetch` in the background. Errors are reported via
// MsgFetchResult but are otherwise swallowed by the caller (live fetch is
// silent on failure).
func (m AppModel) fetchCmd() tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), fetchTimeout)
		defer cancel()
		return MsgFetchResult{Err: git.Fetch(ctx, m.runner)}
	}
}

func (m AppModel) loadBranchCmd() tea.Cmd {
	return func() tea.Msg {
		branch, err := git.CurrentBranch(context.Background(), m.runner)
		if err != nil {
			return nil
		}
		return MsgBranchInfo{Branch: branch}
	}
}

// logArgs builds the git log args for the current showAll setting: --all
// includes every ref, otherwise git log defaults to the current branch (HEAD).
func (m AppModel) logArgs(offset int) []string {
	args := []string{fmt.Sprintf("--skip=%d", offset), fmt.Sprintf("--max-count=%d", pageSize)}
	if m.showAll {
		args = append([]string{"--all"}, args...)
	}
	return args
}

func (m AppModel) loadCommitsCmd(offset, gen int) tea.Cmd {
	args := m.logArgs(offset)
	return func() tea.Msg {
		ctx := context.Background()
		commits, err := git.FetchLog(ctx, m.runner, args...)
		if err != nil {
			return MsgError{Gen: gen, Err: err}
		}
		if refsByHash, err := git.BuildRefsByHash(ctx, m.runner); err == nil {
			git.AttachRefs(commits, refsByHash)
		}
		if offset == 0 {
			lt := graphpkg.NewLayouter()
			layout := lt.Append(commits)
			return MsgCommitsLoaded{Commits: commits, Layout: layout, Layouter: lt, Gen: gen}
		}
		return MsgCommitsBatch{Commits: commits, Offset: offset, Gen: gen}
	}
}

func (m AppModel) loadDiffCmd(hash string) tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		files, err := git.FetchDiff(ctx, m.runner, hash)
		return MsgDiffLoaded{Hash: hash, Files: files, Err: err}
	}
}

// scheduleDiffLoad bumps the debounce sequence and returns a tea.Cmd that,
// after diffDebounce has elapsed with no further cursor movement, requests
// the diff for hash.
func (m *AppModel) scheduleDiffLoad(hash string) tea.Cmd {
	m.debounceSeq++
	seq := m.debounceSeq
	return tea.Tick(diffDebounce, func(time.Time) tea.Msg {
		return MsgDiffDebounce{Hash: hash, Seq: seq}
	})
}

// diffCmdIfChanged returns a Cmd to (debounce-)load the diff for the
// currently selected commit, if it differs from what's already loaded or
// pending.
func (m *AppModel) diffCmdIfChanged() tea.Cmd {
	c := m.log.selectedCommit()
	if c == nil || c.Hash == m.diff.hash {
		return nil
	}
	return m.scheduleDiffLoad(c.Hash)
}

func (m AppModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		if m.mode == modeSearch {
			return m.updateSearch(msg)
		}
		return m.updateKeys(msg)

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.applyLayout()

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		cmds = append(cmds, cmd)

	case MsgAnimationTick:
		if msg.Seq == m.animSeq && m.mode == modeZen {
			m.zen.advance()
			cmds = append(cmds, m.scheduleAnimTick())
		}

	case MsgCommitsLoaded:
		if msg.Gen != m.reloadGen {
			break // superseded by a newer reload
		}
		m.loading = false
		m.banner.dismiss()
		m.loadedCount = len(msg.Commits)
		m.log.setCommits(msg.Commits, msg.Layout)
		m.layouter = msg.Layouter
		// Load more if there might be more commits.
		if len(msg.Commits) == pageSize {
			m.loading = true
			cmds = append(cmds, m.loadCommitsCmd(len(msg.Commits), msg.Gen))
		}
		if cmd := m.diffCmdIfChanged(); cmd != nil {
			cmds = append(cmds, cmd)
		}

	case MsgCommitsBatch:
		if msg.Gen != m.reloadGen {
			break
		}
		all := append(m.log.commits, msg.Commits...)
		if m.layouter == nil {
			// Defensive fallback (e.g. a caller/test built MsgCommitsLoaded
			// without a Layouter) — rebuild lane state from what's loaded so far.
			m.layouter = graphpkg.NewLayouter()
			m.layouter.Append(m.log.commits)
		}
		layout := m.layouter.Append(msg.Commits)
		m.log.setCommits(all, layout)
		m.loadedCount = len(all)
		if len(msg.Commits) == pageSize {
			cmds = append(cmds, m.loadCommitsCmd(len(all), msg.Gen))
		} else {
			m.loading = false
		}

	case MsgCountdownTick:
		if msg.Seq == m.tickSeq {
			cmds = append(cmds, m.scheduleCountdownTick())
		}

	case MsgFetchTick:
		if msg.Seq == m.fetchSeq {
			cmds = append(cmds, m.scheduleFetchTick())
			if !m.fetchInFlight {
				m.fetchInFlight = true
				cmds = append(cmds, m.fetchCmd())
			}
		}

	case MsgFetchResult:
		m.fetchInFlight = false
		if msg.Err == nil {
			m.reloadGen++
			cmds = append(cmds, m.loadCommitsCmd(0, m.reloadGen))
		}
		// On error: swallow silently, the next tick retries.

	case MsgDiffDebounce:
		if msg.Seq == m.debounceSeq {
			if c := m.log.selectedCommit(); c != nil && c.Hash == msg.Hash {
				cmds = append(cmds, m.loadDiffCmd(msg.Hash))
			}
		}

	case MsgDiffLoaded:
		// A failed diff load is shown inside the diff pane itself (local to
		// the commit being viewed), not the app-wide banner: see "Done when"
		// in the M1 roadmap entry -- a bad diff must not kill the app.
		if c := m.log.selectedCommit(); c != nil && c.Hash == msg.Hash {
			if msg.Err != nil {
				m.diff.setError(msg.Hash, msg.Err)
			} else {
				m.diff.setDiff(msg.Hash, msg.Files)
			}
		}

	case MsgError:
		if msg.Gen != m.reloadGen {
			break // superseded by a newer reload; that one's outcome wins
		}
		// Non-fatal: shown as a dismissible banner while the rest of the UI
		// stays interactive.
		m.banner.set(msg.Err)
		m.loading = false

	case MsgBranchInfo:
		m.branch = msg.Branch
	}

	return m, tea.Batch(cmds...)
}

func (m AppModel) updateKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, keys.Quit):
		return m, tea.Quit

	case key.Matches(msg, keys.Back):
		m.banner.dismiss()
		m.popupOpen = false
		if m.mode == modeZen {
			m.mode = modeLog
			// applyLayout, matching zen-entry below: nothing that affects
			// the log pane's height formula currently changes while zen is
			// active, which is why this has been a harmless no-op so far,
			// but that's an easy invariant for a future change (e.g. a
			// banner that only reserves a row when it has something to
			// show) to break silently. Calling it here removes the need to
			// rely on that at all.
			m.applyLayout()
		}

	case key.Matches(msg, keys.Zen):
		if m.mode == modeZen {
			m.mode = modeLog
			m.applyLayout()
			return m, nil
		}
		// applyLayout, not a bare setSize, so zen's height stays in sync
		// with the fetch bar and error banner it's drawn above (previously
		// this used m.height directly, which drew 2 rows too tall).
		m.applyLayout()
		m.zen.rebuild(m.repoName, m.log.commits, m.log.layout)
		m.mode = modeZen
		return m, m.scheduleAnimTick()

	case m.mode == modeZen:
		return m, nil

	case m.popupOpen:
		return m.updatePopupKeys(msg)

	case key.Matches(msg, keys.Search):
		m.mode = modeSearch
		m.search.Focus()
		m.applyLayout() // the log pane must shrink to make room for the search box

	case key.Matches(msg, keys.Toggle):
		m.showAll = !m.showAll
		m.loading = true
		m.log.setCommits(nil, model.GraphLayout{})
		m.layouter = nil
		m.reloadGen++
		return m, m.loadCommitsCmd(0, m.reloadGen)

	default:
		return m.updateLogKeys(msg)
	}

	return m, nil
}

func (m AppModel) updateLogKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, keys.Up):
		m.log.moveUp()
	case key.Matches(msg, keys.Down):
		m.log.moveDown()
	case key.Matches(msg, keys.PageUp):
		m.log.pageUp()
	case key.Matches(msg, keys.PageDown):
		m.log.pageDown()
	case key.Matches(msg, keys.Top):
		m.log.moveTop()
	case key.Matches(msg, keys.Bottom):
		m.log.moveBottom()
	case key.Matches(msg, keys.Enter):
		m.popupOpen = true
		return m, m.diffCmdIfChanged()
	default:
		return m, nil
	}
	return m, nil
}

func (m AppModel) updatePopupKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, keys.Up):
		m.diff.viewport.ScrollUp(1)
	case key.Matches(msg, keys.Down):
		m.diff.viewport.ScrollDown(1)
	case key.Matches(msg, keys.PageUp):
		m.diff.viewport.HalfPageUp()
	case key.Matches(msg, keys.PageDown):
		m.diff.viewport.HalfPageDown()
	case key.Matches(msg, keys.Top):
		m.diff.viewport.GotoTop()
	case key.Matches(msg, keys.Bottom):
		m.diff.viewport.GotoBottom()
	}
	return m, nil
}

func (m AppModel) updateSearch(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "enter":
		query := m.search.Value()
		m.log.applyFilter(query)
		m.search.Blur()
		m.mode = modeLog
		m.applyLayout() // give the log pane back the rows the search box used
	case "esc":
		m.search.Clear()
		m.log.applyFilter("")
		m.search.Blur()
		m.mode = modeLog
		m.applyLayout()
	default:
		cmd := m.search.Update(msg)
		return m, cmd
	}
	return m, nil
}

const (
	toolbarHeight  = 1
	footerHeight   = 1
	fetchBarHeight = 1
	bannerHeight   = 1 // statusBanner's fixed row count; read via m.banner.Height()
	searchHeight   = 3 // search.View()'s bordered box: top border + input line + bottom border
	popupWidthPct  = 80
	popupHeightPct = 80
	sideMargin     = 2
)

func (m *AppModel) applyLayout() {
	contentWidth := max(m.width-2*sideMargin, 1)
	logHeight := m.height - toolbarHeight - footerHeight - fetchBarHeight - m.banner.Height()
	if m.mode == modeSearch {
		// The search box is inserted between the log pane and the footer
		// (see View()'s modeSearch branch) without shrinking the terminal,
		// so the log pane must give up those rows or the view overflows.
		logHeight -= searchHeight
	}
	m.log.setSize(contentWidth, logHeight)
	popupW := m.width * popupWidthPct / 100
	popupH := m.height * popupHeightPct / 100
	m.diff.setSize(popupW, popupH)
	m.zen.setSize(m.width, m.height-fetchBarHeight-m.banner.Height())
}

var styleMargin = lipgloss.NewStyle().Padding(0, sideMargin)

func (m AppModel) View() string {
	if m.width == 0 {
		return ""
	}

	if m.mode == modeZen {
		fetchBar := renderFetchBar(m.width, m.fetchInFlight, m.nextFetchAt)
		return m.zen.View() + "\n" + fetchBar + "\n" + m.banner.View(m.width)
	}

	contentWidth := max(m.width-2*sideMargin, 1)
	toolbar := renderToolbar(contentWidth, m.repoName, m.branch, m.showAll)
	footer := renderFooter(contentWidth, m.loading, m.spinner.View(), m.loadedCount, m.showAll, m.log.cursor, m.log.visibleCount())
	fetchBar := renderFetchBar(contentWidth, m.fetchInFlight, m.nextFetchAt)
	banner := m.banner.View(contentWidth)

	if m.mode == modeSearch {
		content := toolbar + "\n" + m.log.View(true) + "\n" + m.search.View() + "\n" + footer + "\n" + fetchBar + "\n" + banner
		return styleMargin.Render(content)
	}

	base := styleMargin.Render(toolbar + "\n" + m.log.View(true) + "\n" + footer + "\n" + fetchBar + "\n" + banner)

	if m.popupOpen {
		return overlayCenter(base, m.diff.View(true), m.width, m.height)
	}

	return base
}
