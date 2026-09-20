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

type viewMode int

const (
	modeLog viewMode = iota
	modeSearch
)

type AppModel struct {
	runner      *git.Runner
	log         logPane
	diff        diffPane
	search      searchModel
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
	err         error
	debounceSeq int
}

func NewApp(runner *git.Runner) AppModel {
	sp := spinner.New()
	sp.Spinner = spinner.Dot

	return AppModel{
		runner:   runner,
		log:      newLogPane(),
		diff:     newDiffPane(),
		search:   newSearchModel(),
		spinner:  sp,
		showAll:  true,
		repoName: filepath.Base(runner.RepoPath),
	}
}

func (m AppModel) Init() tea.Cmd {
	return tea.Batch(
		m.spinner.Tick,
		m.loadCommitsCmd(0),
		m.loadBranchCmd(),
	)
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

func (m AppModel) loadCommitsCmd(offset int) tea.Cmd {
	args := m.logArgs(offset)
	return func() tea.Msg {
		ctx := context.Background()
		commits, err := git.FetchLog(ctx, m.runner, args...)
		if err != nil {
			return MsgError{Err: err}
		}
		if refsByHash, err := git.BuildRefsByHash(ctx, m.runner); err == nil {
			git.AttachRefs(commits, refsByHash)
		}
		if offset == 0 {
			layout := graphpkg.Layout(commits)
			return MsgCommitsLoaded{Commits: commits, Layout: layout}
		}
		return MsgCommitsBatch{Commits: commits, Offset: offset}
	}
}

func (m AppModel) loadDiffCmd(hash string) tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		files, err := git.FetchDiff(ctx, m.runner, hash)
		if err != nil {
			return MsgError{Err: err}
		}
		return MsgDiffLoaded{Hash: hash, Files: files}
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

	case MsgCommitsLoaded:
		m.loading = false
		m.loadedCount = len(msg.Commits)
		m.log.setCommits(msg.Commits, msg.Layout)
		// Load more if there might be more commits.
		if len(msg.Commits) == pageSize {
			m.loading = true
			cmds = append(cmds, m.loadCommitsCmd(len(msg.Commits)))
		}
		if cmd := m.diffCmdIfChanged(); cmd != nil {
			cmds = append(cmds, cmd)
		}

	case MsgCommitsBatch:
		all := append(m.log.commits, msg.Commits...)
		layout := graphpkg.Layout(all)
		m.log.setCommits(all, layout)
		m.loadedCount = len(all)
		if len(msg.Commits) == pageSize {
			cmds = append(cmds, m.loadCommitsCmd(len(all)))
		} else {
			m.loading = false
		}

	case MsgDiffDebounce:
		if msg.Seq == m.debounceSeq {
			if c := m.log.selectedCommit(); c != nil && c.Hash == msg.Hash {
				cmds = append(cmds, m.loadDiffCmd(msg.Hash))
			}
		}

	case MsgDiffLoaded:
		if c := m.log.selectedCommit(); c != nil && c.Hash == msg.Hash {
			m.diff.setDiff(msg.Hash, msg.Files)
		}

	case MsgError:
		m.err = msg.Err
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
		m.popupOpen = false

	case m.popupOpen:
		return m.updatePopupKeys(msg)

	case key.Matches(msg, keys.Search):
		m.mode = modeSearch
		m.search.Focus()

	case key.Matches(msg, keys.Toggle):
		m.showAll = !m.showAll
		m.loading = true
		m.log.setCommits(nil, model.GraphLayout{})
		return m, m.loadCommitsCmd(0)

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
	case "esc":
		m.search.Clear()
		m.log.applyFilter("")
		m.search.Blur()
		m.mode = modeLog
	default:
		cmd := m.search.Update(msg)
		return m, cmd
	}
	return m, nil
}

const (
	toolbarHeight  = 1
	footerHeight   = 1
	popupWidthPct  = 80
	popupHeightPct = 80
	sideMargin     = 2
)

func (m *AppModel) applyLayout() {
	contentWidth := max(m.width-2*sideMargin, 1)
	m.log.setSize(contentWidth, m.height-toolbarHeight-footerHeight)
	popupW := m.width * popupWidthPct / 100
	popupH := m.height * popupHeightPct / 100
	m.diff.setSize(popupW, popupH)
}

var styleMargin = lipgloss.NewStyle().Padding(0, sideMargin)

func (m AppModel) View() string {
	if m.err != nil {
		return styleError.Render(fmt.Sprintf("Error: %v", m.err))
	}

	if m.width == 0 {
		return ""
	}

	contentWidth := max(m.width-2*sideMargin, 1)
	toolbar := renderToolbar(contentWidth, m.repoName, m.branch, m.showAll)
	footer := renderFooter(contentWidth, m.loading, m.spinner.View(), m.loadedCount, m.showAll, m.log.cursor, m.log.visibleCount())

	if m.mode == modeSearch {
		content := toolbar + "\n" + m.log.View(true) + "\n" + m.search.View() + "\n" + footer
		return styleMargin.Render(content)
	}

	base := styleMargin.Render(toolbar + "\n" + m.log.View(true) + "\n" + footer)

	if m.popupOpen {
		return overlayCenter(base, m.diff.View(true), m.width, m.height)
	}

	return base
}
