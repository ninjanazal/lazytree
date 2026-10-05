package ui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/eurico-martins/lazytree/internal/git"
	"github.com/eurico-martins/lazytree/internal/graph"
)

func TestTooSmallTerminal(t *testing.T) {
	sizes := [][2]int{{1, 1}, {5, 2}, {10, 3}, {23, 30}, {80, 9}, {minWidth - 1, minHeight - 1}}
	for _, sz := range sizes {
		w, h := sz[0], sz[1]
		// Enter each mode at a normal size, then shrink: keys are ignored
		// once the terminal is too small, so that's the only real path.
		m := NewApp(&git.Runner{RepoPath: "."})
		updated, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
		m = updated.(AppModel)
		cs := realisticCommits()
		m.log.setCommits(cs, graph.Layout(cs))

		shrink := func(m AppModel) AppModel {
			u, _ := m.Update(tea.WindowSizeMsg{Width: w, Height: h})
			return u.(AppModel)
		}
		states := map[string]struct {
			m    AppModel
			mode viewMode
		}{
			"log": {shrink(m), modeLog},
			"zen": {shrink(press(t, m, "z")), modeZen},
		}
		for name, st := range states {
			if st.m.mode != st.mode {
				t.Fatalf("%dx%d %s: mode is %v, want %v", w, h, name, st.m.mode, st.mode)
			}
			view := st.m.View()
			lines := strings.Split(view, "\n")
			if len(lines) > h {
				t.Errorf("%dx%d %s: %d lines, terminal has %d", w, h, name, len(lines), h)
			}
			for i, l := range lines {
				if lw := lipgloss.Width(l); lw > w {
					t.Errorf("%dx%d %s: line %d is %d wide: %q", w, h, name, i, lw, l)
				}
			}
			if w >= lipgloss.Width(tooSmallTitle) && h >= 3 && !strings.Contains(view, tooSmallTitle) {
				t.Errorf("%dx%d %s: missing too-small message:\n%s", w, h, name, view)
			}
		}
	}
}

func TestTooSmallRecoversOnResize(t *testing.T) {
	m := NewApp(&git.Runner{RepoPath: "."})
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 10, Height: 5})
	updated, _ = updated.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	if v := updated.View(); strings.Contains(v, "Terminal too small") {
		t.Fatalf("still showing too-small message at 80x24:\n%s", v)
	}
}

func TestTooSmallIgnoresKeysExceptQuit(t *testing.T) {
	m := NewApp(&git.Runner{RepoPath: "."})
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 10, Height: 5})
	m = updated.(AppModel)
	cs := realisticCommits()
	m.log.setCommits(cs, graph.Layout(cs))
	before := m.log.cursor

	for _, k := range []string{"/", "z", "o", "y", "j", "enter", "?"} {
		next, cmd := m.Update(keyMsg(k))
		got := next.(AppModel)
		if cmd != nil || got.mode != m.mode || got.log.cursor != before || got.popupOpen != m.popupOpen {
			t.Errorf("key %q changed state while too small (mode %v→%v, cmd=%v)", k, m.mode, got.mode, cmd != nil)
		}
	}
	_, cmd := m.Update(keyMsg("q"))
	if cmd == nil {
		t.Fatal("q must still quit")
	}
	if _, ok := cmd().(tea.QuitMsg); !ok {
		t.Fatal("q did not produce tea.Quit")
	}
}

func TestTooSmallHintFollowsRemappedQuit(t *testing.T) {
	old := keys
	t.Cleanup(func() { keys = old })
	if err := ApplyKeys(map[string][]string{"quit": {"x"}}); err != nil {
		t.Fatal(err)
	}
	if v := tooSmallView(20, 6); !strings.Contains(v, "x to quit") {
		t.Fatalf("hint ignores remapped quit key:\n%s", v)
	}
}

func keyMsg(k string) tea.KeyMsg {
	if k == "enter" {
		return tea.KeyMsg{Type: tea.KeyEnter}
	}
	return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(k)}
}
