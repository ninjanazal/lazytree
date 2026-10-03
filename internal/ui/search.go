package ui

import (
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

type searchModel struct {
	input  textinput.Model
	active bool
	width  int // total width of the bordered box; 0 = natural width
}

// setWidth sizes the search box to the pane it sits under, so it never
// overflows a narrow terminal.
func (s *searchModel) setWidth(w int) {
	s.width = w
	// Border (2) + padding (2) + "  / " prompt (4), plus a cursor cell.
	s.input.Width = max(w-9, 1)
}

func newSearchModel() searchModel {
	ti := textinput.New()
	ti.Placeholder = "search…  g:message  s:code  p:path"
	ti.CharLimit = 100
	return searchModel{input: ti}
}

func (s *searchModel) Focus() {
	s.active = true
	s.input.Focus()
}

func (s *searchModel) Blur() {
	s.active = false
	s.input.Blur()
}

func (s *searchModel) Value() string {
	return s.input.Value()
}

func (s *searchModel) Clear() {
	s.input.Reset()
}

func (s *searchModel) Update(msg tea.Msg) tea.Cmd {
	var cmd tea.Cmd
	s.input, cmd = s.input.Update(msg)
	return cmd
}

func (s *searchModel) View() string {
	box := stylePane
	if s.width > 0 {
		box = box.Width(max(s.width-2, 1)) // Width excludes the border
	}
	return box.Render("  / " + s.input.View())
}
