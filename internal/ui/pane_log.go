package ui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
	"github.com/eurico-martins/lazytree/internal/graph"
	"github.com/eurico-martins/lazytree/internal/model"
)

type logPane struct {
	commits     []model.Commit
	layout      model.GraphLayout
	cursor      int
	offset      int // scroll offset
	height      int
	width       int
	showAll     bool
	filterQuery string
	filtered    []int // indices into commits matching filter
}

func newLogPane() logPane {
	return logPane{showAll: true}
}

func (p *logPane) setSize(w, h int) {
	p.width = w
	p.height = h
}

func (p *logPane) setCommits(commits []model.Commit, layout model.GraphLayout) {
	p.commits = commits
	p.layout = layout
	if p.cursor >= len(commits) {
		p.cursor = max(0, len(commits)-1)
	}
	p.filtered = nil
}

func (p *logPane) selectedCommit() *model.Commit {
	if len(p.commits) == 0 {
		return nil
	}
	idx := p.visibleIndex(p.cursor)
	if idx < 0 || idx >= len(p.commits) {
		return nil
	}
	return &p.commits[idx]
}

func (p *logPane) moveUp() {
	if p.cursor > 0 {
		p.cursor--
		if p.cursor < p.offset {
			p.offset = p.cursor
		}
	}
}

func (p *logPane) moveDown() {
	count := p.visibleCount()
	if p.cursor < count-1 {
		p.cursor++
		if p.cursor >= p.offset+p.height-2 {
			p.offset = p.cursor - (p.height - 3)
		}
	}
}

func (p *logPane) moveTop() {
	p.cursor = 0
	p.offset = 0
}

func (p *logPane) moveBottom() {
	count := p.visibleCount()
	p.cursor = max(0, count-1)
	p.offset = max(0, p.cursor-(p.height-3))
}

func (p *logPane) pageUp() {
	p.cursor = max(0, p.cursor-(p.height-2))
	p.offset = max(0, p.offset-(p.height-2))
}

func (p *logPane) pageDown() {
	count := p.visibleCount()
	p.cursor = min(count-1, p.cursor+(p.height-2))
	p.offset = min(max(0, count-(p.height-2)), p.offset+(p.height-2))
}

func (p *logPane) applyFilter(query string) {
	p.filterQuery = query
	if query == "" {
		p.filtered = nil
		return
	}
	q := strings.ToLower(query)
	var matches []int
	for i, c := range p.commits {
		if strings.Contains(strings.ToLower(c.Subject), q) ||
			strings.Contains(strings.ToLower(c.Hash), q) ||
			strings.Contains(strings.ToLower(c.Author), q) {
			matches = append(matches, i)
		}
	}
	p.filtered = matches
	p.cursor = 0
	p.offset = 0
}

func (p *logPane) visibleCount() int {
	if p.filtered != nil {
		return len(p.filtered)
	}
	return len(p.commits)
}

func (p *logPane) visibleIndex(cursor int) int {
	if p.filtered != nil {
		if cursor < 0 || cursor >= len(p.filtered) {
			return -1
		}
		return p.filtered[cursor]
	}
	return cursor
}

// graphLines renders the full, unfiltered graph for every commit in order,
// indexed by commit index. Filtering only hides which rows' text gets
// displayed (see View); the graph itself is always walked in full so
// activeLanes bookkeeping and diagonals stay correct regardless of filter.
func (p *logPane) graphLines() []string {
	lines := make([]string, len(p.layout.Nodes))
	activeLanes := make(map[int]bool)
	for i, node := range p.layout.Nodes {
		var next *model.GraphNode
		if i+1 < len(p.layout.Nodes) {
			next = &p.layout.Nodes[i+1]
		}
		lines[i] = graph.RenderCommitLine(node, next, activeLanes, p.layout.Width)
	}
	return lines
}

func (p *logPane) nearBottom() bool {
	return p.cursor >= p.visibleCount()-50
}

func (p *logPane) View(focused bool) string {
	pane := stylePane
	if focused {
		pane = stylePaneActive
	}

	if len(p.commits) == 0 {
		return pane.Width(p.width - 4).Render(
			styleHelp.Render("Loading history…"),
		)
	}

	var sb strings.Builder
	count := p.visibleCount()
	visible := p.height - 4 // account for border + title + help
	if visible < 1 {
		visible = 1
	}

	// Graph rendering always walks the full commit list in order (not just
	// the visible window) so activeLanes bookkeeping and diagonals stay
	// correct even when a filter hides some rows' text.
	graphLines := p.graphLines()

	// Fixed column widths (computed over all commits, not just the visible
	// window) so hash/date/author don't reflow between scrolls and the graph
	// column that follows them lands in the same x-offset on every row.
	hashColW, dateColW, authorColW := 0, 0, 0
	for _, c := range p.commits {
		hashColW = max(hashColW, lipgloss.Width(c.ShortHash))
		dateColW = max(dateColW, lipgloss.Width(relativeTime(c.Timestamp)))
		authorColW = max(authorColW, lipgloss.Width(truncate(c.Author, 16)))
	}

	lineBudget := visible
	for idx := p.offset; idx < count && lineBudget > 0; idx++ {
		commitIdx := p.visibleIndex(idx)
		if commitIdx < 0 || commitIdx >= len(p.commits) {
			break
		}

		c := p.commits[commitIdx]
		selected := idx == p.cursor

		graphStr := " "
		if commitIdx < len(graphLines) {
			graphStr = graphLines[commitIdx]
		}
		graphRow, graphConn, _ := strings.Cut(graphStr, "\n")

		// Commit fields, padded to fixed column widths so the graph column
		// that follows stays aligned across rows.
		hash := styleHash.Width(hashColW).Render(c.ShortHash)
		subject := c.Subject
		author := styleAuthor.Width(authorColW).Render(truncate(c.Author, 16))
		date := styleDate.Width(dateColW).Render(relativeTime(c.Timestamp))
		refs := renderRefs(c.Refs)

		graphW := lipgloss.Width(graphRow)
		hashW := lipgloss.Width(hash)
		authorW := lipgloss.Width(author)
		dateW := lipgloss.Width(date)
		refsW := lipgloss.Width(refs)

		// Available width for subject.
		fixed := hashW + 1 + dateW + 1 + authorW + 1 + graphW + 1 + refsW + 2
		subjectW := p.width - 8 - fixed
		if subjectW < 10 {
			subjectW = 10
		}
		subject = truncate(subject, subjectW)

		line := fmt.Sprintf("%s %s %s %s%s %s",
			hash, date, author, graphRow,
			func() string {
				if refs != "" {
					return " " + refs
				}
				return ""
			}(),
			subject,
		)

		if selected {
			// Strip the per-field ANSI styling first: each inner
			// lipgloss.Render call emits its own reset, which would
			// otherwise cut the selection background short after the
			// first colored segment. Render the plain text through
			// styleSelected instead so the background spans the full row.
			plain := ansi.Strip(line)
			lineW := lipgloss.Width(plain)
			paneW := p.width - 8
			if lineW < paneW {
				plain += strings.Repeat(" ", paneW-lineW)
			}
			line = styleSelected.Render(plain)
		}

		sb.WriteString(line)
		sb.WriteString("\n")
		lineBudget--

		if graphConn != "" && lineBudget > 0 {
			sb.WriteString(graphConn)
			sb.WriteString("\n")
			lineBudget--
		}
	}

	title := styleTitle.Render("  HISTORY")
	if p.filterQuery != "" {
		title += styleHelp.Render(fmt.Sprintf("  filter: %s (%d matches)", p.filterQuery, p.visibleCount()))
	} else {
		title += styleHelp.Render(fmt.Sprintf("  %d commits", len(p.commits)))
	}

	content := title + "\n" + sb.String()
	help := keys.helpView()

	inner := content + "\n" + help

	return pane.
		Width(p.width - 4).
		Height(p.height - 2).
		Render(inner)
}

func renderRefs(refs []model.Ref) string {
	if len(refs) == 0 {
		return ""
	}
	var parts []string
	for _, r := range refs {
		var s string
		switch r.Kind {
		case model.RefHead:
			s = styleRefHead.Render("HEAD")
		case model.RefLocalBranch:
			name := r.Name
			if r.IsHead {
				s = styleRefHead.Render("HEAD -> ") + styleRefLocal.Render(name)
			} else {
				s = styleRefLocal.Render(name)
			}
		case model.RefRemoteBranch:
			s = styleRefRemote.Render(r.Name)
		case model.RefTag:
			s = styleRefTag.Render("tag: " + r.Name)
		}
		parts = append(parts, s)
	}
	inner := strings.Join(parts, ", ")
	return lipgloss.NewStyle().Foreground(colorBorder).Render("(") +
		inner +
		lipgloss.NewStyle().Foreground(colorBorder).Render(")")
}

func relativeTime(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	d := time.Since(t)
	switch {
	case d < time.Minute:
		return "just now"
	case d < time.Hour:
		return fmt.Sprintf("%dm ago", int(d.Minutes()))
	case d < 24*time.Hour:
		return fmt.Sprintf("%dh ago", int(d.Hours()))
	case d < 7*24*time.Hour:
		return fmt.Sprintf("%dd ago", int(d.Hours()/24))
	case d < 30*24*time.Hour:
		return fmt.Sprintf("%dw ago", int(d.Hours()/(24*7)))
	case d < 365*24*time.Hour:
		return fmt.Sprintf("%dmo ago", int(d.Hours()/(24*30)))
	default:
		return fmt.Sprintf("%dy ago", int(d.Hours()/(24*365)))
	}
}

func truncate(s string, n int) string {
	if n <= 0 {
		return ""
	}
	runes := []rune(s)
	if len(runes) <= n {
		return s
	}
	if n <= 1 {
		return string(runes[:n])
	}
	return string(runes[:n-1]) + "…"
}
