package ui

import (
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/eurico-martins/lazytree/internal/git"
	"github.com/eurico-martins/lazytree/internal/graph"
	"github.com/eurico-martins/lazytree/internal/model"
)

func realisticCommits() []model.Commit {
	cs := longHistory(30)
	for i := range cs {
		cs[i].Author = "Alexandria Montgomery"
		cs[i].Timestamp = time.Now().Add(-time.Duration(i) * 36 * time.Hour)
		cs[i].Subject = "feat(ui): a fairly long commit subject that needs truncating"
	}
	cs[0].Refs = []model.Ref{
		{Name: "feature/very-long-branch-name", Kind: model.RefLocalBranch, IsHead: true},
		{Name: "origin/feature/very-long-branch-name", Kind: model.RefRemoteBranch},
		{Name: "v1.2.3", Kind: model.RefTag},
	}
	return cs
}

// TestNarrowTerminals renders every view at small and awkward sizes and
// checks nothing panics and each frame fits the terminal exactly: no line
// wider than the terminal, no more lines than its height.
func TestNarrowTerminals(t *testing.T) {
	sizes := [][2]int{{24, 10}, {30, 10}, {40, 12}, {60, 15}, {80, 24}, {99, 30}, {160, 40}, {220, 50}}
	for _, sz := range sizes {
		w, h := sz[0], sz[1]
		m := NewApp(&git.Runner{RepoPath: "."})
		updated, _ := m.Update(tea.WindowSizeMsg{Width: w, Height: h})
		m = updated.(AppModel)
		cs := realisticCommits()
		m.log.setCommits(cs, graph.Layout(cs))
		m.diffCache.put(cs[0].Hash, []model.DiffFile{{NewPath: "a.go", Status: "M", Hunks: []model.DiffHunk{{
			Header: "@@ -1 +1 @@", Lines: []model.DiffLine{{Kind: model.DiffAdded, Text: "+x := 1", NewN: 1}},
		}}}})

		states := map[string]func(AppModel) AppModel{
			"log":    func(m AppModel) AppModel { return m },
			"search": func(m AppModel) AppModel { return press(t, m, "/") },
			"help":   func(m AppModel) AppModel { return press(t, m, "?") },
			"popup": func(m AppModel) AppModel {
				m = press(t, m, "enter")
				d, _ := m.Update(MsgDiffLoaded{Hash: cs[0].Hash, Files: m.diffCacheFiles(cs[0].Hash)})
				return d.(AppModel)
			},
			"zen": func(m AppModel) AppModel { return press(t, m, "z") },
		}
		for name, setup := range states {
			view := setup(m).View()
			lines := strings.Split(view, "\n")
			if len(lines) > h {
				t.Errorf("%dx%d %s: %d lines, terminal has %d", w, h, name, len(lines), h)
			}
			for i, l := range lines {
				if lw := lipgloss.Width(l); lw > w {
					t.Errorf("%dx%d %s: line %d is %d wide: %q", w, h, name, i, lw, l)
					break
				}
			}
		}
	}
}

func (m AppModel) diffCacheFiles(hash string) []model.DiffFile {
	files, _ := m.diffCache.get(hash)
	return files
}
