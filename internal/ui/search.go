package ui

import (
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

type searchModel struct {
	input  textinput.Model
	active bool
}

func newSearchModel() searchModel {
	ti := textinput.New()
	ti.Placeholder = "search commits…"
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
	return stylePane.Render("  / " + s.input.View())
}
