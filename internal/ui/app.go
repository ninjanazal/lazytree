package ui

import (
	"context"
	"path/filepath"
	"strings"
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
const fetchTimeout = 30 * time.Second
const deepSearchTimeout = 60 * time.Second
const animInterval = 120 * time.Millisecond
const countdownInterval = 1 * time.Second

type viewMode int

const (
	modeLog viewMode = iota
	modeSearch
	modeZen
)

type AppModel struct {
	runner       *git.Runner
	log          logPane
	diff         diffPane
	search       searchModel
	zen          zenPane
	spinner      spinner.Model
	mode         viewMode
	popupOpen    bool
	layoutPref   string // "auto", "split" or "popup"; the v key switches between split and popup
	flashText    string // transient footer message (e.g. "copied 2c8ab07")
	flashSeq     int
	deepSeq      int                          // guards MsgDeepSearch: bumped whenever a git-side search starts or is dropped
	deepPending  bool                         // a git-side search is running
	helpOpen     bool                         // the ? key-bindings overlay is showing
	searchOrigin struct{ cursor, offset int } // log position when `/` was pressed; esc restores it
	width        int
	height       int
	loading      bool
	loadedCount  int
	showAll      bool
	repoName     string
	branch       string
	banner       statusBanner
	debounceSeq  int
	diffCache    *diffCache // parsed diffs by commit hash; commits are immutable

	fetchEvery    time.Duration // background fetch period; 0 = disabled
	fetchSeq      int           // guards the self-rescheduling MsgFetchTick chain
	fetchInFlight bool          // true while a background `git fetch` is running
	nextFetchAt   time.Time     // when the next background fetch is scheduled to run
	reloadGen     int           // guards stale MsgCommitsLoaded/MsgCommitsBatch from a
	// reload superseded by a newer one (Toggle vs. background fetch)
	layouter *graphpkg.Layouter // carries lane state across MsgCommitsBatch pages for the current reloadGen

	// refsByHash/refsFingerprint are fetched once per reloadGen (at offset
	// 0) rather than once per page: refs can't change mid-load, so paging
	// through a large repo would otherwise repeat the same for-each-ref
	// work for every page. refsFingerprint lets fetchCmd cheaply detect
	// whether a background `git fetch` actually moved any ref before
	// paying for a full reload.
	refsByHash      map[string][]model.Ref
	refsFingerprint string

	// logStream is the open `git log` process for the current reloadGen,
	// read incrementally by nextPageCmd; nil once fully read. A generation
	// that's superseded while this is still open (Toggle, a fetch-changed
	// reload) must close it via closeLogStreamCmd before starting the next
	// one, or the abandoned `git log` process leaks.
	logStream *git.LogStream

	animSeq int // guards the self-rescheduling MsgAnimationTick chain (zen mode)
	tickSeq int // guards the self-rescheduling MsgCountdownTick chain (footer redraw)
}

// Options are the user-configurable settings (see internal/config).
type Options struct {
	FetchInterval time.Duration // background `git fetch --all` period; 0 disables it
	ShowAll       bool          // start with all refs rather than HEAD only
	HideTags      bool          // start with tag pills hidden
	HideRemotes   bool          // start with remote-branch pills hidden
	HideRefs      []string      // ref-name globs whose pills are always hidden
	Layout        string        // "auto" (split on wide terminals), "split" or "popup"
}

// DefaultOptions are used when there is no config file.
func DefaultOptions() Options {
	return Options{FetchInterval: 60 * time.Second, ShowAll: true, Layout: "auto"}
}

func NewApp(runner *git.Runner) AppModel {
	return NewAppWithOptions(runner, DefaultOptions())
}

func NewAppWithOptions(runner *git.Runner, opts Options) AppModel {
	sp := spinner.New()
	sp.Spinner = spinner.Dot

	m := AppModel{
		runner:     runner,
		log:        newLogPane(),
		diff:       newDiffPane(),
		diffCache:  newDiffCache(diffCacheSize),
		search:     newSearchModel(),
		zen:        newZenPane(),
		spinner:    sp,
		showAll:    opts.ShowAll,
		fetchEvery: opts.FetchInterval,
		repoName:   filepath.Base(runner.RepoPath),
		fetchSeq:   1,
		tickSeq:    1,
	}
	m.layoutPref = opts.Layout
	if m.layoutPref == "" {
		m.layoutPref = "auto"
	}
	m.log.refs = refFilter{hideTags: opts.HideTags, hideRemotes: opts.HideRemotes, globs: opts.HideRefs}
	if m.fetchEvery > 0 {
		m.nextFetchAt = time.Now().Add(m.fetchEvery)
	}
	return m
}

// Init has a value receiver, like Update and View: Bubble Tea only keeps the
// tea.Cmd it returns, not any mutation made to m, so the initial fetchSeq/
// tickSeq/nextFetchAt values must already be set by NewApp rather than by
// calling scheduleFetchTick/scheduleCountdownTick here (those mutate a copy
// that's discarded). The two tea.Tick calls below mirror what those methods
// build, using the seq NewApp already baked in.
func (m AppModel) Init() tea.Cmd {
	cmds := []tea.Cmd{
		m.spinner.Tick,
		m.startReloadCmd(m.reloadGen, nil),
		m.loadBranchCmd(),
		tea.Tick(countdownInterval, func(time.Time) tea.Msg {
			return MsgCountdownTick{Seq: m.tickSeq}
		}),
	}
	if m.fetchEvery > 0 {
		cmds = append(cmds, tea.Tick(m.fetchEvery, func(time.Time) tea.Msg {
			return MsgFetchTick{Seq: m.fetchSeq}
		}))
	}
	return tea.Batch(cmds...)
}

// scheduleFetchTick bumps the fetch sequence and returns a tea.Cmd that,
// after the fetch interval has elapsed, requests a background git fetch.
func (m *AppModel) scheduleFetchTick() tea.Cmd {
	m.fetchSeq++
	seq := m.fetchSeq
	m.nextFetchAt = time.Now().Add(m.fetchEvery)
	return tea.Tick(m.fetchEvery, func(time.Time) tea.Msg {
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

// fetchCmd runs `git fetch` in the background, then checks whether it moved
// any ref (branch/tag/remote-tracking hash) compared to m.refsFingerprint.
// Errors are reported via MsgFetchResult but are otherwise swallowed by the
// caller (live fetch is silent on failure). If the post-fetch ref check
// itself fails, Changed is reported true so a real change can never be
// silently dropped.
func (m AppModel) fetchCmd() tea.Cmd {
	before := m.refsFingerprint
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), fetchTimeout)
		defer cancel()
		if err := git.Fetch(ctx, m.runner); err != nil {
			return MsgFetchResult{Err: err}
		}
		refsByHash, err := git.BuildRefsByHash(ctx, m.runner)
		if err != nil {
			return MsgFetchResult{Changed: true}
		}
		if git.RefsFingerprint(refsByHash) == before {
			return MsgFetchResult{Changed: false}
		}
		// Pass the refs already fetched for this check through to the
		// reload it triggers, so loadCommitsCmd doesn't pay for a second
		// BuildRefsByHash call for the same generation.
		return MsgFetchResult{Changed: true, RefsByHash: refsByHash}
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
// includes every ref, otherwise git log defaults to the current branch
// (HEAD). No --skip/--max-count: the whole log streams through LogStream,
// paged client-side by startReloadCmd/nextPageCmd's Next(pageSize) calls.
func (m AppModel) logArgs() []string {
	if m.showAll {
		return []string{"--all"}
	}
	return nil
}

// startReloadCmd begins a new generation: starts a fresh LogStream and
// reads its first page. Refs are fetched once here too, reusing knownRefs
// when the caller already has them (the MsgFetchResult changed-reload
// path, which fetched fresh refs while checking whether anything changed).
func (m AppModel) startReloadCmd(gen int, knownRefs map[string][]model.Ref) tea.Cmd {
	args := m.logArgs()
	return func() tea.Msg {
		ctx := context.Background()
		stream, err := git.StartLogStream(ctx, m.runner, args...)
		if err != nil {
			return MsgError{Gen: gen, Err: err}
		}
		commits, done, err := stream.Next(pageSize)
		if err != nil {
			return MsgError{Gen: gen, Err: err}
		}

		refsByHash := knownRefs
		if refsByHash == nil {
			if r, err := git.BuildRefsByHash(ctx, m.runner); err == nil {
				refsByHash = r
			}
			// On failure, refsByHash stays nil: fail safe, don't attach a
			// stale prior-generation map to a new generation's commits.
		}
		if refsByHash != nil {
			git.AttachRefs(commits, refsByHash)
		}

		lt := graphpkg.NewLayouter()
		layout := lt.Append(commits)
		var openStream *git.LogStream
		if !done {
			openStream = stream
		}
		return MsgCommitsLoaded{
			Commits: commits, Layout: layout, Layouter: lt, Gen: gen,
			RefsByHash: refsByHash, Stream: openStream, Done: done,
		}
	}
}

// nextPageCmd reads the next page from an already-open stream (see
// startReloadCmd), for a generation already in progress.
func (m AppModel) nextPageCmd(stream *git.LogStream, gen int) tea.Cmd {
	return func() tea.Msg {
		commits, done, err := stream.Next(pageSize)
		if err != nil {
			return MsgError{Gen: gen, Err: err}
		}
		var openStream *git.LogStream
		if !done {
			openStream = stream
		}
		return MsgCommitsBatch{Commits: commits, Gen: gen, Stream: openStream, Done: done}
	}
}

// closeLogStreamCmd abandons an open stream whose generation has been
// superseded (the --all toggle, or a fetch-changed reload) before it
// finished reading. Without this, the abandoned `git log` process would
// keep running indefinitely.
func closeLogStreamCmd(s *git.LogStream) tea.Cmd {
	return func() tea.Msg {
		s.Close()
		return nil
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
			return m.followSelection(m.updateSearch(msg))
		}
		return m.followSelection(m.updateKeys(msg))

	case tea.MouseMsg:
		return m.followSelection(m.updateMouse(msg))

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.applyLayout()
		if m.splitActive() {
			if c := m.diffCmdIfChanged(); c != nil {
				cmds = append(cmds, c)
			}
		}

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
		m.loading = !msg.Done
		m.banner.dismiss()
		m.loadedCount = len(msg.Commits)
		m.log.setCommits(msg.Commits, msg.Layout)
		m.layouter = msg.Layouter
		m.refsByHash = msg.RefsByHash
		m.refsFingerprint = git.RefsFingerprint(msg.RefsByHash)
		m.logStream = msg.Stream
		if !msg.Done {
			cmds = append(cmds, m.nextPageCmd(msg.Stream, msg.Gen))
		}
		if cmd := m.diffCmdIfChanged(); cmd != nil {
			cmds = append(cmds, cmd)
		}

	case MsgCommitsBatch:
		if msg.Gen != m.reloadGen {
			break
		}
		if m.layouter == nil {
			// Defensive fallback (e.g. a caller/test built MsgCommitsLoaded
			// without a Layouter) — rebuild lane state from what's loaded so far.
			m.layouter = graphpkg.NewLayouter()
			m.layouter.Append(m.log.commits)
		}
		layout := m.layouter.Append(msg.Commits)
		m.log.appendCommits(msg.Commits, layout)
		m.loadedCount = len(m.log.commits)
		m.logStream = msg.Stream
		if !msg.Done {
			cmds = append(cmds, m.nextPageCmd(msg.Stream, msg.Gen))
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
		if msg.Err == nil && msg.Changed {
			if m.logStream != nil {
				cmds = append(cmds, closeLogStreamCmd(m.logStream))
				m.logStream = nil
			}
			m.reloadGen++
			cmds = append(cmds, m.startReloadCmd(m.reloadGen, msg.RefsByHash))
		}
		// On error, or a fetch that moved nothing: swallow silently, the
		// next tick retries/rechecks.

	case MsgDiffDebounce:
		if msg.Seq == m.debounceSeq {
			if c := m.log.selectedCommit(); c != nil && c.Hash == msg.Hash {
				if files, ok := m.diffCache.get(msg.Hash); ok {
					cc := *c
					m.diff.setDiff(msg.Hash, &cc, files)
				} else {
					cmds = append(cmds, m.loadDiffCmd(msg.Hash))
				}
			}
		}

	case MsgDiffLoaded:
		// A failed diff load is shown inside the diff pane itself (local to
		// the commit being viewed), not the app-wide banner: see "Done when"
		// in the M1 roadmap entry -- a bad diff must not kill the app.
		if msg.Err == nil {
			m.diffCache.put(msg.Hash, msg.Files)
		}
		if c := m.log.selectedCommit(); c != nil && c.Hash == msg.Hash {
			if msg.Err != nil {
				m.diff.setError(msg.Hash, msg.Err)
			} else {
				cc := *c
				m.diff.setDiff(msg.Hash, &cc, msg.Files)
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

	case MsgFlash:
		cmds = append(cmds, m.flash(msg.Text))

	case MsgFlashClear:
		if msg.Seq == m.flashSeq {
			m.flashText = ""
		}

	case msgOpenEditor:
		if msg.Err != nil {
			m.banner.set(msg.Err)
			break
		}
		cmds = append(cmds, openEditorCmd(msg.Path))

	case MsgEditorDone:
		if msg.Err != nil {
			m.banner.set(msg.Err)
		}

	case MsgDeepSearch:
		if msg.Seq != m.deepSeq {
			break // superseded or cancelled
		}
		m.deepPending = false
		if msg.Err != nil {
			m.banner.set(msg.Err)
			break
		}
		m.log.setDeepSearch(msg.Label, msg.Hashes)
		m.log.setCursor(m.searchOrigin.cursor, m.searchOrigin.offset)
		m.log.jumpToMatch(true, true)

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
		if m.mode == modeLog && !m.popupOpen && !m.helpOpen {
			m.log.clearSearch() // esc in the plain log view drops search highlights
			m.deepSeq++         // ...and abandons a git-side search still running
			m.deepPending = false
		}
		m.popupOpen = false
		m.helpOpen = false
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

	case key.Matches(msg, keys.Help) && m.mode != modeZen:
		m.helpOpen = !m.helpOpen

	case m.helpOpen:
		return m, nil // the overlay swallows every other key

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
		m.searchOrigin.cursor, m.searchOrigin.offset = m.log.cursor, m.log.offset
		m.search.Clear()
		m.mode = modeSearch
		m.search.Focus()
		m.applyLayout() // the log pane must shrink to make room for the search box

	case key.Matches(msg, keys.Split):
		if m.splitActive() {
			m.layoutPref = "popup"
		} else {
			m.layoutPref = "split"
		}
		m.popupOpen = false
		m.applyLayout()

	case key.Matches(msg, keys.Tags):
		m.log.refs.hideTags = !m.log.refs.hideTags

	case key.Matches(msg, keys.Remotes):
		m.log.refs.hideRemotes = !m.log.refs.hideRemotes

	case key.Matches(msg, keys.Toggle):
		m.showAll = !m.showAll
		m.loading = true
		m.log.setCommits(nil, model.GraphLayout{})
		m.layouter = nil
		m.reloadGen++
		var cmds []tea.Cmd
		if m.logStream != nil {
			cmds = append(cmds, closeLogStreamCmd(m.logStream))
			m.logStream = nil
		}
		cmds = append(cmds, m.startReloadCmd(m.reloadGen, nil))
		return m, tea.Batch(cmds...)

	default:
		return m.updateLogKeys(msg)
	}

	return m, nil
}

// commitAction handles the keys that act on the selected commit and work
// both in the log and inside the diff popup. ok is false if msg isn't one.
func (m AppModel) commitAction(msg tea.KeyMsg) (tea.Model, tea.Cmd, bool) {
	c := m.log.selectedCommit()
	switch {
	case key.Matches(msg, keys.Copy):
		if c == nil {
			return m, nil, true
		}
		return m, copyCmd(c.Hash, c.ShortHash), true
	case key.Matches(msg, keys.Edit):
		if c == nil {
			return m, nil, true
		}
		return m, m.writePatchCmd(c.Hash), true
	}
	return m, nil, false
}

const (
	splitMinWidthAuto   = 140 // "auto" layout splits at this terminal width
	splitMinWidthForced = 100 // an explicit "split" still needs this much room
	splitLogPct         = 55  // share of the width the log gets in split view
)

// splitActive reports whether the diff is shown in a permanent right-hand
// pane instead of a popup.
func (m AppModel) splitActive() bool {
	switch m.layoutPref {
	case "split":
		return m.width >= splitMinWidthForced
	case "popup":
		return false
	}
	return m.width >= splitMinWidthAuto
}

// followSelection keeps the split-view diff in step with the cursor: after
// any key or mouse event it requests the (debounced) diff of whichever
// commit is now selected. In popup layout it does nothing.
func (m AppModel) followSelection(next tea.Model, cmd tea.Cmd) (tea.Model, tea.Cmd) {
	am, ok := next.(AppModel)
	if !ok || !am.splitActive() {
		return next, cmd
	}
	if c := am.diffCmdIfChanged(); c != nil {
		return am, tea.Batch(cmd, c)
	}
	return am, cmd
}

// mouseWheelStep is how many rows one wheel notch moves the selection.
const mouseWheelStep = 3

// updateMouse handles the wheel (scroll the log, or the diff while its
// popup is open) and left-click (select the commit under the pointer).
// Everything else, including drags and mouse use over modal states, is
// ignored.
func (m AppModel) updateMouse(msg tea.MouseMsg) (tea.Model, tea.Cmd) {
	if m.mode != modeLog || m.helpOpen || msg.Action != tea.MouseActionPress {
		return m, nil
	}
	split := m.splitActive()
	overDiff := split && msg.X >= sideMargin+m.log.width
	switch msg.Button {
	case tea.MouseButtonWheelUp, tea.MouseButtonWheelDown:
		up := msg.Button == tea.MouseButtonWheelUp
		switch {
		case overDiff || (!split && m.popupOpen):
			if up {
				m.diff.viewport.ScrollUp(mouseWheelStep)
			} else {
				m.diff.viewport.ScrollDown(mouseWheelStep)
			}
		default:
			for range mouseWheelStep {
				if up {
					m.log.moveUp()
				} else {
					m.log.moveDown()
				}
			}
		}
	case tea.MouseButtonLeft:
		if overDiff {
			m.popupOpen = true // focus the diff pane
			return m, nil
		}
		if m.popupOpen && !split {
			return m, nil
		}
		m.popupOpen = false // a click in the log returns focus to it
		// Rows start below the toolbar line and the column-header line.
		if row := m.log.rowAtLine(msg.Y - toolbarHeight - 1); row >= 0 {
			m.log.setCursor(row, m.log.offset)
		}
	}
	return m, nil
}

func (m AppModel) updateLogKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if next, cmd, ok := m.commitAction(msg); ok {
		return next, cmd
	}
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
	case key.Matches(msg, keys.Parent):
		m.log.jumpToParent()
	case key.Matches(msg, keys.Child):
		m.log.jumpToChild()
	case key.Matches(msg, keys.Head):
		m.log.jumpToHead()
	case key.Matches(msg, keys.NextHit):
		m.log.jumpToMatch(true, false)
	case key.Matches(msg, keys.PrevHit):
		m.log.jumpToMatch(false, false)
	case key.Matches(msg, keys.Enter):
		m.popupOpen = true
		return m, m.diffCmdIfChanged()
	default:
		return m, nil
	}
	return m, nil
}

func (m AppModel) updatePopupKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if next, cmd, ok := m.commitAction(msg); ok {
		return next, cmd
	}
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
	case key.Matches(msg, keys.Parent):
		// Follow the parent link from inside the inspector: the log
		// selection moves and the popup loads that commit's diff.
		if m.log.jumpToParent() {
			return m, m.diffCmdIfChanged()
		}
	case key.Matches(msg, keys.Child):
		if m.log.jumpToChild() {
			return m, m.diffCmdIfChanged()
		}
	}
	return m, nil
}

func (m AppModel) updateSearch(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "enter":
		value := m.search.Value()
		m.search.Blur()
		m.mode = modeLog
		m.applyLayout() // give the log pane back the rows the search box used
		if kind, term, ok := parseDeepQuery(value); ok {
			// Git-side search: too slow to run per keystroke, so it starts
			// on enter and the result arrives as MsgDeepSearch.
			m.deepSeq++
			m.deepPending = true
			return m, m.deepSearchCmd(m.deepSeq, kind, strings.TrimSpace(value), term)
		}
		// In-memory search: already applied while typing; keep it so n/N
		// move between matches.
		if value == "" {
			m.log.clearSearch()
		}
	case "esc":
		m.search.Clear()
		m.log.clearSearch()
		m.log.setCursor(m.searchOrigin.cursor, m.searchOrigin.offset)
		m.search.Blur()
		m.mode = modeLog
		m.applyLayout()
	default:
		cmd := m.search.Update(msg)
		value := m.search.Value()
		if _, _, deep := parseDeepQuery(value); deep {
			// Wait for enter; drop any in-memory matches from earlier input.
			m.log.clearSearch()
			m.log.setCursor(m.searchOrigin.cursor, m.searchOrigin.offset)
			return m, cmd
		}
		// Incremental search: re-scan on every keystroke and jump to the
		// first match at or after where the search began.
		m.log.setSearch(value)
		m.log.setCursor(m.searchOrigin.cursor, m.searchOrigin.offset)
		m.log.jumpToMatch(true, true)
		return m, cmd
	}
	return m, nil
}

// deepSearchCmd runs a git-side search (see git.SearchHashes) off the UI
// goroutine. label is the user's full input, shown in the footer.
func (m AppModel) deepSearchCmd(seq int, kind git.SearchKind, label, term string) tea.Cmd {
	showAll := m.showAll
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), deepSearchTimeout)
		defer cancel()
		hashes, err := git.SearchHashes(ctx, m.runner, kind, term, showAll)
		return MsgDeepSearch{Seq: seq, Label: label, Hashes: hashes, Err: err}
	}
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
	m.search.setWidth(contentWidth)
	if m.splitActive() {
		logW := contentWidth * splitLogPct / 100
		m.log.setSize(logW, logHeight)
		// The diff pane's bordered box is 2 columns narrower than the
		// width it is given (see diffPane.View), hence the +2.
		m.diff.setSize(contentWidth-logW+2, logHeight)
	} else {
		m.log.setSize(contentWidth, logHeight)
		popupW := m.width * popupWidthPct / 100
		popupH := m.height * popupHeightPct / 100
		m.diff.setSize(popupW, popupH)
	}
	m.zen.setSize(m.width, m.height-fetchBarHeight-m.banner.Height())
}

var styleMargin = lipgloss.NewStyle().Padding(0, sideMargin)

// searchStatus is the search text for the footer, including the
// in-progress state of a git-side search.
func (m AppModel) searchStatus() string {
	if m.flashText != "" {
		return m.flashText
	}
	if m.deepPending {
		return "searching…"
	}
	return m.log.searchStatus()
}

func (m AppModel) View() string {
	if m.width == 0 {
		return ""
	}

	if m.mode == modeZen {
		fetchBar := renderFetchBar(m.width, m.fetchInFlight, m.nextFetchAt, m.fetchEvery)
		return m.zen.View() + "\n" + fetchBar + "\n" + m.banner.View(m.width)
	}

	contentWidth := max(m.width-2*sideMargin, 1)
	toolbar := renderToolbar(contentWidth, m.repoName, m.branch, m.showAll, m.log.refs.label())
	footer := renderFooter(contentWidth, m.loading, m.spinner.View(), m.loadedCount, m.showAll, m.log.cursor, m.log.visibleCount(), m.searchStatus())
	fetchBar := renderFetchBar(contentWidth, m.fetchInFlight, m.nextFetchAt, m.fetchEvery)
	banner := m.banner.View(contentWidth)

	body := m.log.View(true)
	split := m.splitActive()
	if split {
		// Pad the log block to its full width so the diff pane starts at a
		// fixed column regardless of how wide each row happens to be.
		left := strings.Join(padLines(strings.Split(body, "\n"), m.log.width, m.log.height), "\n")
		body = lipgloss.JoinHorizontal(lipgloss.Top, left, m.diff.View(m.popupOpen))
	}

	if m.mode == modeSearch {
		content := toolbar + "\n" + body + "\n" + m.search.View() + "\n" + footer + "\n" + fetchBar + "\n" + banner
		return styleMargin.Render(content)
	}

	base := styleMargin.Render(toolbar + "\n" + body + "\n" + footer + "\n" + fetchBar + "\n" + banner)

	if m.helpOpen {
		return overlayCenter(base, renderHelpOverlay(), m.width, m.height)
	}

	if m.popupOpen && !split {
		return overlayCenter(base, m.diff.View(true), m.width, m.height)
	}

	return base
}
