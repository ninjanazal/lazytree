package ui

import (
	"fmt"
	"slices"
	"sort"
	"strings"

	"github.com/charmbracelet/bubbles/key"
)

type keyMap struct {
	Up        key.Binding
	Down      key.Binding
	PageUp    key.Binding
	PageDown  key.Binding
	Top       key.Binding
	Bottom    key.Binding
	Enter     key.Binding
	Back      key.Binding
	Search    key.Binding
	Toggle    key.Binding
	Zen       key.Binding
	Help      key.Binding
	NextHit   key.Binding
	PrevHit   key.Binding
	Parent    key.Binding
	NthParent key.Binding
	Child     key.Binding
	Head      key.Binding
	Copy      key.Binding
	Edit      key.Binding
	Tags      key.Binding
	Split     key.Binding
	Remotes   key.Binding
	Quit      key.Binding
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
		key.WithHelp("enter", "open"),
	),
	Back: key.NewBinding(
		key.WithKeys("esc"),
		key.WithHelp("esc", "back"),
	),
	Search: key.NewBinding(
		key.WithKeys("/"),
		key.WithHelp("/", "search"),
	),
	Toggle: key.NewBinding(
		key.WithKeys("a"),
		key.WithHelp("a", "refs"),
	),
	Zen: key.NewBinding(
		key.WithKeys("z"),
		key.WithHelp("z", "zen"),
	),
	Help: key.NewBinding(
		key.WithKeys("?"),
		key.WithHelp("?", "help"),
	),
	NextHit: key.NewBinding(
		key.WithKeys("n"),
		key.WithHelp("n", "next match"),
	),
	PrevHit: key.NewBinding(
		key.WithKeys("N"),
		key.WithHelp("N", "previous match"),
	),
	Parent: key.NewBinding(
		key.WithKeys("p"),
		key.WithHelp("p", "parent commit"),
	),
	NthParent: key.NewBinding(
		key.WithKeys("1", "2", "3", "4", "5", "6", "7", "8", "9"),
		key.WithHelp("1-9", "nth parent (merges)"),
	),
	Child: key.NewBinding(
		key.WithKeys("c"),
		key.WithHelp("c", "child commit"),
	),
	Head: key.NewBinding(
		key.WithKeys("H"),
		key.WithHelp("H", "jump to HEAD"),
	),
	Copy: key.NewBinding(
		key.WithKeys("y"),
		key.WithHelp("y", "copy commit hash"),
	),
	Edit: key.NewBinding(
		key.WithKeys("o"),
		key.WithHelp("o", "open commit in $EDITOR"),
	),
	Split: key.NewBinding(
		key.WithKeys("v"),
		key.WithHelp("v", "toggle split view"),
	),
	Tags: key.NewBinding(
		key.WithKeys("t"),
		key.WithHelp("t", "hide/show tags"),
	),
	Remotes: key.NewBinding(
		key.WithKeys("r"),
		key.WithHelp("r", "hide/show remote branches"),
	),
	Quit: key.NewBinding(
		key.WithKeys("q", "ctrl+c"),
		key.WithHelp("q", "quit"),
	),
}

// helpGroup is a titled list of bindings shown together in the ? overlay.
type helpGroup struct {
	title    string
	bindings []key.Binding
}

// helpGroups lists every binding, grouped, for the ? overlay. Both it and
// the footer hint are generated from the keyMap, so they can't drift from
// the keys the app actually handles.
func (k keyMap) helpGroups() []helpGroup {
	return []helpGroup{
		{"Navigate", []key.Binding{k.Down, k.Up, k.PageDown, k.PageUp, k.Top, k.Bottom, k.Head, k.Parent, k.NthParent, k.Child}},
		{"Search", []key.Binding{k.Search, k.NextHit, k.PrevHit}},
		{"Commits", []key.Binding{k.Enter, k.Copy, k.Edit, k.Toggle}},
		{"View", []key.Binding{k.Split, k.Tags, k.Remotes, k.Zen, k.Help, k.Back, k.Quit}},
	}
}

// helpView renders the one-line footer hint.
func (k keyMap) helpView() string {
	parts := []string{firstKey(k.Down) + "/" + firstKey(k.Up) + " move"}
	for _, b := range []key.Binding{k.Enter, k.Search, k.Toggle, k.Zen, k.Help, k.Quit} {
		h := b.Help()
		parts = append(parts, h.Key+" "+h.Desc)
	}
	return styleHelp.Render(strings.Join(parts, " · "))
}

// renderHelpOverlay builds the centered ? popup listing all key bindings.
func renderHelpOverlay() string {
	var sb strings.Builder
	sb.WriteString(styleTitle.Render(" Keys") + "  " + styleHelp.Render("? or esc to close") + "\n")
	for _, g := range keys.helpGroups() {
		sb.WriteString("\n" + styleHeader.Render(g.title) + "\n")
		for _, b := range g.bindings {
			h := b.Help()
			sb.WriteString("  " + styleHash.Render(fmt.Sprintf("%-8s", h.Key)) + " " + h.Desc + "\n")
		}
	}
	sb.WriteString("\n" + styleHeader.Render("Search prefixes") + "\n")
	sb.WriteString("  " + styleHelp.Render("g:text  commit message (git --grep)") + "\n")
	sb.WriteString("  " + styleHelp.Render("s:text  code added/removed (git -S)") + "\n")
	sb.WriteString("  " + styleHelp.Render("p:path  commits touching a path") + "\n")
	return stylePopup.Render(strings.TrimRight(sb.String(), "\n"))
}

// firstKey is the primary key of a binding, for compact hints.
func firstKey(b key.Binding) string {
	if ks := b.Keys(); len(ks) > 0 {
		return ks[0]
	}
	return "?"
}

// bindingTargets maps the action names usable in the config's [keys] table
// to the bindings behind them.
func bindingTargets() map[string]*key.Binding {
	return map[string]*key.Binding{
		"up":             &keys.Up,
		"down":           &keys.Down,
		"page_up":        &keys.PageUp,
		"page_down":      &keys.PageDown,
		"top":            &keys.Top,
		"bottom":         &keys.Bottom,
		"enter":          &keys.Enter,
		"back":           &keys.Back,
		"search":         &keys.Search,
		"next_match":     &keys.NextHit,
		"prev_match":     &keys.PrevHit,
		"parent":         &keys.Parent,
		"child":          &keys.Child,
		"head":           &keys.Head,
		"copy":           &keys.Copy,
		"edit":           &keys.Edit,
		"toggle_refs":    &keys.Toggle,
		"toggle_tags":    &keys.Tags,
		"toggle_remotes": &keys.Remotes,
		"zen":            &keys.Zen,
		"help":           &keys.Help,
		"quit":           &keys.Quit,
	}
}

// ActionNames lists the valid [keys] names, for error messages.
func ActionNames() []string {
	names := make([]string, 0, 24)
	for n := range bindingTargets() {
		names = append(names, n)
	}
	sort.Strings(names)
	return names
}

// ApplyKeys rebinds actions: overrides maps an action name (see
// ActionNames) to the keys that should trigger it, replacing its defaults.
// The result must not bind one key to two actions. ctrl+c always quits.
// Nothing is changed if any entry is invalid. Call it before starting the
// program.
func ApplyKeys(overrides map[string][]string) error {
	targets := bindingTargets()
	planned := map[string][]string{} // action -> keys after the change
	for name, b := range targets {
		planned[name] = b.Keys()
	}
	for name, ks := range overrides {
		if _, ok := targets[name]; !ok {
			return fmt.Errorf("unknown action %q (valid: %s)", name, strings.Join(ActionNames(), ", "))
		}
		if len(ks) == 0 {
			return fmt.Errorf("action %q has no keys", name)
		}
		for _, k := range ks {
			if strings.TrimSpace(k) == "" {
				return fmt.Errorf("action %q has an empty key", name)
			}
		}
		planned[name] = ks
	}
	if q := planned["quit"]; !slices.Contains(q, "ctrl+c") {
		planned["quit"] = append(slices.Clone(q), "ctrl+c")
	}
	owner := map[string]string{}
	for _, name := range ActionNames() {
		for _, k := range planned[name] {
			if slices.Contains(keys.NthParent.Keys(), k) {
				return fmt.Errorf("key %q is reserved for jumping to the nth parent (action %q)", k, name)
			}
		}
	}
	for _, name := range ActionNames() {
		for _, k := range planned[name] {
			if prev, dup := owner[k]; dup && prev != name {
				return fmt.Errorf("key %q is bound to both %q and %q", k, prev, name)
			}
			owner[k] = name
		}
	}
	for name := range overrides {
		b := targets[name]
		desc := b.Help().Desc
		*b = key.NewBinding(key.WithKeys(planned[name]...), key.WithHelp(strings.Join(overrideLabels(planned[name], name), "/"), desc))
	}
	return nil
}

// overrideLabels trims the implicit ctrl+c from the quit label.
func overrideLabels(ks []string, name string) []string {
	if name == "quit" && len(ks) > 1 {
		return ks[:len(ks)-1]
	}
	return ks
}
