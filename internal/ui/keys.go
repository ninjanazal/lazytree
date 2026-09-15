package ui

import "github.com/charmbracelet/bubbles/key"

type keyMap struct {
	Up          key.Binding
	Down        key.Binding
	PageUp      key.Binding
	PageDown    key.Binding
	Top         key.Binding
	Bottom      key.Binding
	Enter       key.Binding
	Back        key.Binding
	Diff        key.Binding
	Search      key.Binding
	Toggle      key.Binding
	FocusSwitch key.Binding
	Quit        key.Binding
}

var keys = keyMap{
	Up: key.NewBinding(
		key.WithKeys("k", "up"),
		key.WithHelp("k/↑", "up"),
	),
	Down: key.NewBinding(
		key.WithKeys("j", "down"),
		key.WithHelp("j/↓", "down"),
	),
	PageUp: key.NewBinding(
		key.WithKeys("ctrl+u", "pgup"),
		key.WithHelp("ctrl+u", "page up"),
	),
	PageDown: key.NewBinding(
		key.WithKeys("ctrl+d", "pgdn"),
		key.WithHelp("ctrl+d", "page down"),
	),
	Top: key.NewBinding(
		key.WithKeys("g", "home"),
		key.WithHelp("g", "first"),
	),
	Bottom: key.NewBinding(
		key.WithKeys("G", "end"),
		key.WithHelp("G", "last"),
	),
	Enter: key.NewBinding(
		key.WithKeys("enter"),
		key.WithHelp("enter", "diff"),
	),
	Back: key.NewBinding(
		key.WithKeys("esc"),
		key.WithHelp("esc", "back"),
	),
	Diff: key.NewBinding(
		key.WithKeys("d"),
		key.WithHelp("d", "diff"),
	),
	Search: key.NewBinding(
		key.WithKeys("/"),
		key.WithHelp("/", "search"),
	),
	Toggle: key.NewBinding(
		key.WithKeys("a"),
		key.WithHelp("a", "toggle all refs"),
	),
	FocusSwitch: key.NewBinding(
		key.WithKeys("tab"),
		key.WithHelp("tab", "focus diff/log"),
	),
	Quit: key.NewBinding(
		key.WithKeys("q", "ctrl+c"),
		key.WithHelp("q", "quit"),
	),
}

func (k keyMap) helpView() string {
	return styleHelp.Render(
		"j/k scroll · tab focus diff · enter/d full diff · / search · a toggle · q quit",
	)
}
