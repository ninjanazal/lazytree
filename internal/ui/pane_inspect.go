package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/viewport"
	"github.com/eurico-martins/lazytree/internal/model"
)

type inspectPane struct {
	commit       *model.Commit
	changedFiles []string
	viewport     viewport.Model
	width        int
	height       int
	ready        bool
}

func newInspectPane() inspectPane {
	return inspectPane{}
}

func (p *inspectPane) setSize(w, h int) {
	p.width = w
	p.height = h
	if !p.ready {
		p.viewport = viewport.New(w-6, h-6)
		p.ready = true
	} else {
		p.viewport.Width = w - 6
		p.viewport.Height = h - 6
	}
	if p.commit != nil {
		p.viewport.SetContent(p.renderContent())
	}
}

func (p *inspectPane) setCommit(c model.Commit, files []string) {
	p.commit = &c
	p.changedFiles = files
	if p.ready {
		p.viewport.SetContent(p.renderContent())
		p.viewport.GotoTop()
	}
}

func (p *inspectPane) renderContent() string {
	if p.commit == nil {
		return ""
	}
	c := p.commit
	var sb strings.Builder

	sb.WriteString(styleHash.Render(c.ShortHash) + "  " + c.Subject + "\n\n")
	sb.WriteString(fmt.Sprintf("  Hash:       %s\n", styleHash.Render(c.Hash)))
	sb.WriteString(fmt.Sprintf("  Author:     %s <%s>\n", styleAuthor.Render(c.Author), c.AuthorEmail))
	sb.WriteString(fmt.Sprintf("  Committer:  %s <%s>\n", styleAuthor.Render(c.Committer), c.CommitterEmail))
	sb.WriteString(fmt.Sprintf("  Date:       %s  (%s)\n",
		c.Timestamp.Format("2006-01-02 15:04:05 -0700"),
		relativeTime(c.Timestamp)))

	if len(c.Parents) > 0 {
		sb.WriteString(fmt.Sprintf("  Parents:    %s\n", strings.Join(shortHashes(c.Parents), " ")))
	}

	if len(c.Refs) > 0 {
		sb.WriteString(fmt.Sprintf("  Refs:       %s\n", renderRefs(c.Refs)))
	}

	if c.Body != "" {
		sb.WriteString("\n" + c.Body + "\n")
	}

	if len(p.changedFiles) > 0 {
		sb.WriteString("\n" + styleTitle.Render("Files") + "\n")
		for _, f := range p.changedFiles {
			parts := strings.SplitN(f, "\t", 2)
			if len(parts) == 2 {
				status := parts[0]
				path := parts[1]
				var statusStyled string
				switch status {
				case "A":
					statusStyled = styleDiffAdded.Render("A")
				case "D":
					statusStyled = styleDiffRemoved.Render("D")
				case "M":
					statusStyled = styleDate.Render("M")
				default:
					statusStyled = status
				}
				sb.WriteString(fmt.Sprintf("  %s  %s\n", statusStyled, path))
			} else {
				sb.WriteString("  " + f + "\n")
			}
		}
	}

	return sb.String()
}

func (p *inspectPane) View() string {
	if p.commit == nil {
		return stylePane.Width(p.width - 4).Height(p.height - 2).Render(
			styleHelp.Render("Select a commit and press enter"),
		)
	}

	title := styleTitle.Render("  COMMIT") + "  " + styleHelp.Render("esc back · d diff")
	content := title + "\n" + p.viewport.View()

	return stylePane.
		Width(p.width - 4).
		Height(p.height - 2).
		Render(content)
}

func shortHashes(hashes []string) []string {
	out := make([]string, len(hashes))
	for i, h := range hashes {
		if len(h) > 7 {
			out[i] = h[:7]
		} else {
			out[i] = h
		}
	}
	return out
}
