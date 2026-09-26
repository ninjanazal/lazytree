package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/viewport"
	"github.com/eurico-martins/lazytree/internal/model"
)

type diffPane struct {
	files    []model.DiffFile
	hash     string
	err      error // set by setError; a failed load is local to this pane
	loaded   bool  // false until the first setDiff/setError, distinct from "loaded but empty"
	viewport viewport.Model
	width    int
	height   int
	ready    bool
}

func newDiffPane() diffPane {
	return diffPane{}
}

func (p *diffPane) setSize(w, h int) {
	p.width = w
	p.height = h
	if !p.ready {
		p.viewport = viewport.New(w-6, h-6)
		p.ready = true
	} else {
		p.viewport.Width = w - 6
		p.viewport.Height = h - 6
	}
	if p.loaded {
		p.viewport.SetContent(p.renderContent())
	}
}

func (p *diffPane) setDiff(hash string, files []model.DiffFile) {
	p.hash = hash
	p.files = files
	p.err = nil
	p.loaded = true
	if p.ready {
		p.viewport.SetContent(p.renderContent())
		p.viewport.GotoTop()
	}
}

// setError records a failed diff load for hash. The pane shows the error
// in place of content, rather than leaving a previous commit's stale diff
// on screen -- see the M1 roadmap's "Done when: ... a bad diff doesn't kill
// the app" (a failed diff load is local to this pane, not the whole app).
func (p *diffPane) setError(hash string, err error) {
	p.hash = hash
	p.files = nil
	p.err = err
	p.loaded = true
	if p.ready {
		p.viewport.SetContent(p.renderContent())
		p.viewport.GotoTop()
	}
}

func (p *diffPane) renderContent() string {
	if p.err != nil {
		return styleError.Render(fmt.Sprintf("couldn't load diff: %v", p.err))
	}
	if len(p.files) == 0 {
		return styleHelp.Render("No diff available")
	}

	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("%s  %s changed\n\n",
		styleHelp.Render("all files"),
		styleHash.Render(fmt.Sprintf("%d", len(p.files))),
	))

	for i, file := range p.files {
		if i > 0 {
			sb.WriteString("\n")
		}
		header := fmt.Sprintf("diff  %s  →  %s", file.OldPath, file.NewPath)
		sb.WriteString(styleDiffHeader.Render(header) + "\n")

		for _, hunk := range file.Hunks {
			sb.WriteString(styleDiffHunkHeader.Render(hunk.Header) + "\n")
			for _, line := range hunk.Lines {
				switch line.Kind {
				case model.DiffAdded:
					sb.WriteString(styleDiffAdded.Render(line.Text) + "\n")
				case model.DiffRemoved:
					sb.WriteString(styleDiffRemoved.Render(line.Text) + "\n")
				case model.DiffHeader:
					sb.WriteString(styleDiffHeader.Render(line.Text) + "\n")
				default:
					sb.WriteString(line.Text + "\n")
				}
			}
		}
	}

	return sb.String()
}

func (p *diffPane) View(focused bool) string {
	pane := stylePopup

	if !p.loaded {
		return pane.Width(p.width - 4).Height(p.height - 2).Render(
			styleHelp.Render("Loading diff…"),
		)
	}

	shortHash := p.hash
	if len(shortHash) > 8 {
		shortHash = shortHash[:8]
	}

	title := styleTitle.Render(" "+shortHash) + "  " +
		styleHelp.Render("j/k scroll · esc to close")
	content := title + "\n" + p.viewport.View()

	return pane.
		Width(p.width - 4).
		Height(p.height - 2).
		Render(content)
}
