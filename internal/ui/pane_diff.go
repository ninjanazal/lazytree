package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/viewport"
	"github.com/charmbracelet/lipgloss"
	"github.com/eurico-martins/lazytree/internal/model"
)

type diffPane struct {
	files    []model.DiffFile
	hash     string
	commit   *model.Commit // metadata shown in the inspector header; nil if unknown
	err      error         // set by setError; a failed load is local to this pane
	loaded   bool          // false until the first setDiff/setError, distinct from "loaded but empty"
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

func (p *diffPane) setDiff(hash string, commit *model.Commit, files []model.DiffFile) {
	p.hash = hash
	p.commit = commit
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
	p.commit = nil
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
		return p.renderHeader() + styleHelp.Render("No diff available")
	}

	var sb strings.Builder

	sb.WriteString(p.renderHeader())
	sb.WriteString(renderStat(p.files))
	sb.WriteString("\n")

	for i, file := range p.files {
		if i > 0 {
			sb.WriteString("\n")
		}
		header := fmt.Sprintf("diff  %s  →  %s", file.OldPath, file.NewPath)
		sb.WriteString(styleDiffHeader.Render(header) + "\n")
		switch {
		case file.Binary:
			sb.WriteString(styleHelp.Render("Binary file not shown") + "\n")
		case len(file.Hunks) == 0:
			sb.WriteString(styleHelp.Render("No content changes") + "\n")
		}

		hl := newLineHighlighter(file.NewPath)
		budget := maxHighlightLines
		for _, hunk := range file.Hunks {
			sb.WriteString(styleDiffHunkHeader.Render(hunk.Header) + "\n")
			for _, line := range hunk.Lines {
				gutter := styleHelp.Render(lineGutter(line))
				if line.Kind == model.DiffHeader {
					sb.WriteString(styleDiffHeader.Render(line.Text) + "\n")
					continue
				}
				// "\ No newline at end of file" has no line numbers and
				// is not source code.
				if line.OldN == 0 && line.NewN == 0 || line.Text == "" || budget <= 0 {
					sb.WriteString(gutter + plainDiffLine(line) + "\n")
					continue
				}
				budget--
				prefix, content := line.Text[:1], line.Text[1:]
				switch line.Kind {
				case model.DiffAdded:
					prefix = styleDiffAdded.Render(prefix)
				case model.DiffRemoved:
					prefix = styleDiffRemoved.Render(prefix)
				}
				sb.WriteString(gutter + prefix + hl.highlight(content) + "\n")
			}
		}
	}

	return sb.String()
}

func (p *diffPane) View(focused bool) string {
	pane := stylePopup
	if !focused {
		// Split view: the unfocused diff pane gets a quiet border.
		pane = pane.BorderForeground(colorBorder)
	}

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
		styleHelp.Render(firstKey(keys.Down)+"/"+firstKey(keys.Up)+" scroll · "+firstKey(keys.Parent)+" parent · "+firstKey(keys.Child)+" child · "+firstKey(keys.Back)+" close")
	content := title + "\n" + p.viewport.View()

	return pane.
		Width(p.width - 4).
		Height(p.height - 2).
		Render(content)
}

// lineGutter renders the "old new │ " line-number gutter. A side with no
// number (the other side of an added/removed line, or a "\ No newline"
// annotation) is left blank.
func lineGutter(l model.DiffLine) string {
	num := func(n int) string {
		if n == 0 {
			return "    "
		}
		return fmt.Sprintf("%4d", n)
	}
	return num(l.OldN) + " " + num(l.NewN) + " │ "
}

// renderHeader renders the commit inspector header: full hash, subject,
// author/committer, parents, refs and body. It returns "" when the commit
// is unknown.
func (p *diffPane) renderHeader() string {
	c := p.commit
	if c == nil {
		return ""
	}
	wrap := lipgloss.NewStyle().Width(max(p.viewport.Width, 20))
	label := func(s string) string { return styleHelp.Render(fmt.Sprintf("%-10s", s)) }

	var sb strings.Builder
	sb.WriteString(label("commit") + styleHash.Render(c.Hash) + "\n")
	sb.WriteString(label("author") + styleAuthor.Render(c.Author) + styleHelp.Render(emailSuffix(c.AuthorEmail)) + "\n")
	if c.Committer != c.Author || c.CommitterEmail != c.AuthorEmail {
		sb.WriteString(label("committer") + styleAuthor.Render(c.Committer) + styleHelp.Render(emailSuffix(c.CommitterEmail)) + "\n")
	}
	sb.WriteString(label("date") + styleDate.Render(c.Timestamp.Format("Mon Jan 2 15:04:05 2006 -0700")) + "\n")
	if len(c.Parents) > 0 {
		short := make([]string, len(c.Parents))
		for i, h := range c.Parents {
			short[i] = h[:min(len(h), 8)]
		}
		sb.WriteString(label("parents") + styleHash.Render(strings.Join(short, " ")) + "\n")
	}
	if len(c.Refs) > 0 {
		names := make([]string, len(c.Refs))
		for i, r := range c.Refs {
			names[i] = r.Name
		}
		sb.WriteString(label("refs") + wrap.Render(strings.Join(names, ", ")) + "\n")
	}
	sb.WriteString("\n" + wrap.Bold(true).Render(c.Subject) + "\n")
	if body := strings.TrimSpace(c.Body); body != "" {
		sb.WriteString("\n" + wrap.Bold(false).Render(body) + "\n")
	}
	sb.WriteString("\n")
	return sb.String()
}

func emailSuffix(email string) string {
	if email == "" {
		return ""
	}
	return " <" + email + ">"
}

// renderStat renders a per-file +/- summary, like `git show --stat`,
// computed from the already-parsed diff (no extra git call).
func renderStat(files []model.DiffFile) string {
	var sb strings.Builder
	totalAdd, totalDel := 0, 0
	for _, f := range files {
		add, del := 0, 0
		for _, h := range f.Hunks {
			for _, l := range h.Lines {
				switch l.Kind {
				case model.DiffAdded:
					add++
				case model.DiffRemoved:
					del++
				}
			}
		}
		totalAdd += add
		totalDel += del
		path := f.NewPath
		if f.Status == "R" || f.Status == "C" {
			path = f.OldPath + " → " + f.NewPath
		}
		sb.WriteString(fmt.Sprintf(" %s %s  %s %s\n",
			styleHash.Render(f.Status), path,
			styleDiffAdded.Render(fmt.Sprintf("+%d", add)),
			styleDiffRemoved.Render(fmt.Sprintf("-%d", del))))
	}
	sb.WriteString(styleHelp.Render(fmt.Sprintf(" %d files changed, ", len(files))) +
		styleDiffAdded.Render(fmt.Sprintf("+%d", totalAdd)) + " " +
		styleDiffRemoved.Render(fmt.Sprintf("-%d", totalDel)) + "\n")
	return sb.String()
}

// plainDiffLine renders a line with only the diff's add/remove color.
func plainDiffLine(l model.DiffLine) string {
	switch l.Kind {
	case model.DiffAdded:
		return styleDiffAdded.Render(l.Text)
	case model.DiffRemoved:
		return styleDiffRemoved.Render(l.Text)
	}
	return l.Text
}
