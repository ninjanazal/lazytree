package ui

import (
	"context"
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/eurico-martins/lazytree/internal/git"
	graphpkg "github.com/eurico-martins/lazytree/internal/graph"
	"github.com/eurico-martins/lazytree/internal/model"
)

const narrowThreshold = 120
const pageSize = 500

type viewMode int

const (
	modeLog viewMode = iota
	modeInspect
	modeDiff
	modeSearch
)

type AppModel struct {
	runner  *git.Runner
	log     logPane
	inspect inspectPane
	diff    diffPane
	search  searchModel
	spinner spinner.Model
	mode    viewMode
	width   int
	height  int
	narrow  bool
	loading bool
	loadedCount int
	showAll bool
	err     error
	cancel  context.CancelFunc
}

func NewApp(runner *git.Runner) AppModel {
	sp := spinner.New()
	sp.Spinner = spinner.Dot

	return AppModel{
		runner:  runner,
		log:     newLogPane(),
		inspect: newInspectPane(),
		diff:    newDiffPane(),
		search:  newSearchModel(),
		spinner: sp,
		showAll: true,
	}
}

func (m AppModel) Init() tea.Cmd {
	return tea.Batch(
		m.spinner.Tick,
		m.loadCommitsCmd(0),
	)
}

func (m AppModel) loadCommitsCmd(offset int) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithCancel(context.Background())
		_ = cancel // managed by caller
		args := []string{"--all", fmt.Sprintf("--skip=%d", offset), fmt.Sprintf("--max-count=%d", pageSize)}
		commits, err := git.FetchLog(ctx, m.runner, args...)
		if err != nil {
			return MsgError{Err: err}
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

func (m AppModel) loadInspectCmd(c model.Commit) tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		files, err := git.FetchChangedFiles(ctx, m.runner, c.Hash)
		if err != nil {
			return MsgError{Err: err}
		}
		return MsgInspectLoaded{Commit: c, ChangedFiles: files}
	}
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
		m.narrow = msg.Width < narrowThreshold
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

	case MsgInspectLoaded:
		m.inspect.setCommit(msg.Commit, msg.ChangedFiles)
		m.mode = modeInspect

	case MsgDiffLoaded:
		m.diff.setDiff(msg.Hash, msg.Files)
		m.mode = modeDiff

	case MsgError:
		m.err = msg.Err
		m.loading = false
	}

	return m, tea.Batch(cmds...)
}

func (m AppModel) updateKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, keys.Quit):
		return m, tea.Quit

	case key.Matches(msg, keys.Back):
		switch m.mode {
		case modeDiff:
			m.mode = modeInspect
		case modeInspect:
			m.mode = modeLog
		}

	case key.Matches(msg, keys.Search):
		m.mode = modeSearch
		m.search.Focus()

	case key.Matches(msg, keys.Toggle):
		m.showAll = !m.showAll

	case m.mode == modeLog:
		return m.updateLogKeys(msg)

	case m.mode == modeInspect:
		return m.updateInspectKeys(msg)

	case m.mode == modeDiff:
		return m.updateDiffKeys(msg)
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
		if c := m.log.selectedCommit(); c != nil {
			return m, m.loadInspectCmd(*c)
		}
	}
	return m, nil
}

func (m AppModel) updateInspectKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, keys.Diff):
		if c := m.log.selectedCommit(); c != nil {
			return m, m.loadDiffCmd(c.Hash)
		}
	case key.Matches(msg, keys.Up):
		m.inspect.viewport.ScrollUp(1)
	case key.Matches(msg, keys.Down):
		m.inspect.viewport.ScrollDown(1)
	case key.Matches(msg, keys.PageUp):
		m.inspect.viewport.HalfPageUp()
	case key.Matches(msg, keys.PageDown):
		m.inspect.viewport.HalfPageDown()
	}
	return m, nil
}

func (m AppModel) updateDiffKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, keys.Up):
		m.diff.viewport.ScrollUp(1)
	case key.Matches(msg, keys.Down):
		m.diff.viewport.ScrollDown(1)
	case key.Matches(msg, keys.PageUp):
		m.diff.viewport.HalfPageUp()
	case key.Matches(msg, keys.PageDown):
		m.diff.viewport.HalfPageDown()
	case msg.String() == "n":
		m.diff.nextFile()
	case msg.String() == "p":
		m.diff.prevFile()
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

func (m *AppModel) applyLayout() {
	if m.narrow {
		// Single pane: full width.
		m.log.setSize(m.width, m.height-1)
		m.inspect.setSize(m.width, m.height-1)
		m.diff.setSize(m.width, m.height-1)
	} else {
		// Wide: 60/40 split.
		left := m.width * 60 / 100
		right := m.width - left
		m.log.setSize(left, m.height-1)
		m.inspect.setSize(right, m.height-1)
		m.diff.setSize(right, m.height-1)
	}
}

func (m AppModel) View() string {
	if m.err != nil {
		return styleError.Render(fmt.Sprintf("Error: %v", m.err))
	}

	if m.width == 0 {
		return ""
	}

	status := m.statusBar()

	if m.mode == modeSearch {
		return m.log.View() + "\n" + m.search.View() + "\n" + status
	}

	if m.narrow {
		switch m.mode {
		case modeInspect:
			return m.inspect.View() + "\n" + status
		case modeDiff:
			return m.diff.View() + "\n" + status
		default:
			return m.log.View() + "\n" + status
		}
	}

	// Wide layout: log on left, detail on right.
	var right string
	switch m.mode {
	case modeInspect:
		right = m.inspect.View()
	case modeDiff:
		right = m.diff.View()
	default:
		right = m.inspect.View()
	}

	main := lipgloss.JoinHorizontal(lipgloss.Top, m.log.View(), right)
	return main + "\n" + status
}

func (m AppModel) statusBar() string {
	var parts []string
	if m.loading {
		parts = append(parts, m.spinner.View()+" loading…")
	}
	if m.loadedCount > 0 {
		parts = append(parts, fmt.Sprintf("%d commits", m.loadedCount))
	}
	if m.showAll {
		parts = append(parts, "all refs")
	}
	switch m.mode {
	case modeInspect:
		parts = append(parts, "inspect")
	case modeDiff:
		parts = append(parts, "diff")
	}

	left := styleHelp.Render("lazytree")
	right := ""
	if len(parts) > 0 {
		right = styleHelp.Render(" · ")
		for i, p := range parts {
			if i > 0 {
				right += styleHelp.Render(" · ")
			}
			right += styleHelp.Render(p)
		}
	}

	bar := left + right
	padding := m.width - lipgloss.Width(bar)
	if padding > 0 {
		bar += styleHelp.Render(repeatStr(" ", padding))
	}
	return bar
}

func repeatStr(s string, n int) string {
	if n <= 0 {
		return ""
	}
	var sb strings.Builder
	for range n {
		sb.WriteString(s)
	}
	return sb.String()
}
