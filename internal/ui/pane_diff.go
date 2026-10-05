package ui

import (
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/charmbracelet/bubbles/viewport"
	"github.com/charmbracelet/lipgloss"
	"github.com/eurico-martins/lazytree/internal/model"
)

type diffPane struct {
	files    []model.DiffFile
	hash     string
	commit   *model.Commit // metadata shown in the inspector header; nil if unknown
	err      error         // set by setError; a failed load is local to this pane
	content  string        // last rendered content; reused on resize
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
	// The rendered diff is kept as is on resize: re-rendering a big diff
	// here would block the UI, and the viewport clips to the new size anyway.
	if p.loaded && p.content == "" {
		p.viewport.SetContent(p.renderContent())
	}
}

// renderDiffContent renders a commit's inspector + diff for a pane width
// without touching any live pane, so it can run off the UI goroutine.
func renderDiffContent(commit *model.Commit, files []model.DiffFile, width int) string {
	tmp := diffPane{commit: commit, files: files}
	tmp.viewport.Width = width
	return tmp.renderContent()
}

// setPending shows the commit's header immediately (cheap) while its diff is
// being fetched and rendered in the background.
func (p *diffPane) setPending(hash string, commit *model.Commit) {
	p.hash = hash
	p.commit = commit
	p.files = nil
	p.err = nil
	p.loaded = true
	p.content = p.renderHeader() + styleHelp.Render("Loading diff…")
	if p.ready {
		p.viewport.SetContent(p.content)
		p.viewport.GotoTop()
	}
}

// setRendered installs a diff rendered in the background (see
// renderDiffContent / MsgDiffLoaded).
func (p *diffPane) setRendered(hash string, commit *model.Commit, files []model.DiffFile, content string) {
	p.hash = hash
	p.commit = commit
	p.files = files
	p.err = nil
	p.loaded = true
	if content == "" {
		content = p.renderContent() // not pre-rendered (e.g. built by hand in tests)
	}
	p.content = content
	if p.ready {
		p.viewport.SetContent(content)
		p.viewport.GotoTop()
	}
}

func (p *diffPane) setDiff(hash string, commit *model.Commit, files []model.DiffFile) {
	p.hash = hash
	p.commit = commit
	p.files = files
	p.err = nil
	p.loaded = true
	p.content = p.renderContent()
	if p.ready {
		p.viewport.SetContent(p.content)
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
	p.content = ""
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

	rendered := 0
	budget := maxHighlightLines // shared by every file of the commit
	for i, file := range p.files {
		if rendered >= maxDiffRenderLines {
			break
		}
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
		for _, hunk := range file.Hunks {
			if rendered >= maxDiffRenderLines {
				break
			}
			sb.WriteString(styleDiffHunkHeader.Render(hunk.Header) + "\n")
			lines := hunk.Lines
			if room := maxDiffRenderLines - rendered; len(lines) > room {
				lines = lines[:room]
			}
			rendered += len(lines)

			// Pick the lines worth colouring (real code lines, not too long,
			// within the per-file budget) and highlight them in one pass.
			var idx []int
			var src []string
			for i, line := range lines {
				t := line.Text
				if len(t) < 2 || len(t) > maxHighlightLineLen || line.Kind == model.DiffHeader ||
					(line.OldN == 0 && line.NewN == 0) || budget <= 0 {
					continue
				}
				budget--
				idx = append(idx, i)
				src = append(src, t[1:])
			}
			colored := map[int]string{}
			for j, c := range hl.highlightLines(src) {
				colored[idx[j]] = c
			}

			for i, line := range lines {
				text := clipLine(line.Text)
				gutter := styleHelp.Render(lineGutter(line))
				if line.Kind == model.DiffHeader {
					sb.WriteString(styleDiffHeader.Render(text) + "\n")
					continue
				}
				c, ok := colored[i]
				if !ok {
					sb.WriteString(gutter + plainDiffLine(model.DiffLine{Kind: line.Kind, Text: text}) + "\n")
					continue
				}
				prefix := text[:1]
				switch line.Kind {
				case model.DiffAdded:
					prefix = styleDiffAdded.Render(prefix)
				case model.DiffRemoved:
					prefix = styleDiffRemoved.Render(prefix)
				}
				sb.WriteString(gutter + prefix + c + "\n")
			}
		}
	}
	if rendered >= maxDiffRenderLines {
		sb.WriteString("\n" + styleHelp.Render(fmt.Sprintf(
			"… diff truncated after %d lines; press %s to open the full patch in your editor",
			maxDiffRenderLines, firstKey(keys.Edit))) + "\n")
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
		styleHelp.Render(firstKey(keys.Down)+"/"+firstKey(keys.Up)+" scroll · "+firstKey(keys.Parent)+" parent · "+keys.NthParent.Help().Key+" nth parent · "+firstKey(keys.Child)+" child · "+firstKey(keys.Back)+" close")
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
			if len(c.Parents) > 1 && i < 9 {
				// A merge: number the parents so 1-9 can jump to them.
				short[i] = fmt.Sprintf("%d:%s", i+1, short[i])
			}
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

// clipLine shortens an overlong diff line to maxDiffLineLen bytes (on a rune
// boundary) and marks the cut.
func clipLine(s string) string {
	if len(s) <= maxDiffLineLen {
		return s
	}
	cut := maxDiffLineLen
	for cut > 0 && !utf8.RuneStart(s[cut]) {
		cut--
	}
	return s[:cut] + "…"
}
