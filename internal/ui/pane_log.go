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

// minGraphColW reserves room for several lanes worth of glyphs so the graph
// column reads as a wide, explicit lane even for currently-linear history.
const minGraphColW = 9

// authorGap is the (wider than the other columns') separator before AUTHOR
// and before DATE, giving them extra breathing room on the right side.
const authorGap = "      "

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
	if len(p.commits) == 0 {
		return styleHelp.Render("Loading history…")
	}

	var sb strings.Builder
	count := p.visibleCount()
	visible := max(p.height-1, 1) // account for header row

	// Graph rendering always walks the full commit list in order (not just
	// the visible window) so activeLanes bookkeeping and diagonals stay
	// correct even when a filter hides some rows' text.
	graphLines := p.graphLines()

	// Fixed column widths (computed over all commits, not just the visible
	// window) so columns don't reflow between scrolls and stay aligned with
	// the header row above them. messageW is a flexible column padded/
	// truncated to a fixed width so author/date always start at the same
	// horizontal offset regardless of message length.
	graphColW, hashColW, messageW, authorColW, dateColW := p.columnWidths(graphLines)

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
		graphRow = padRight(graphRow, graphColW)

		// Commit fields, padded to fixed column widths so the columns that
		// follow stay aligned across rows. The hash is colored to match its
		// commit's lane/branch color in the graph column.
		hashColor := graph.LanePalette[0]
		if commitIdx < len(p.layout.Nodes) {
			node := p.layout.Nodes[commitIdx]
			hashColor = graph.LanePalette[node.Color%len(graph.LanePalette)]
		}
		hash := lipgloss.NewStyle().Foreground(hashColor).Bold(true).Width(hashColW).Render(c.ShortHash)
		subject := c.Subject
		author := styleAuthor.Width(authorColW).Render(truncate(c.Author, 16))
		date := styleDate.Width(dateColW).Render(relativeTime(c.Timestamp))
		pills := renderRefPills(c.Refs)

		// Available width for subject within the fixed message column.
		pillsW := lipgloss.Width(pills)
		subjectW := messageW - pillsW
		if pills != "" {
			subjectW--
		}
		if subjectW < 5 && pillsW < messageW-5 {
			subjectW = 5
		}
		subject = truncate(subject, subjectW)

		messageField := subject
		if pills != "" {
			if subjectW <= 0 {
				messageField = pills
			} else {
				messageField = pills + " " + subject
			}
		}
		messageField = padRight(messageField, messageW)

		line := fmt.Sprintf(" %s  %s  %s%s%s%s%s",
			graphRow, hash, messageField, authorGap, author, authorGap, date,
		)

		if selected {
			// Strip the per-field ANSI styling first: each inner
			// lipgloss.Render call emits its own reset, which would
			// otherwise cut the selection background short after the
			// first colored segment. Render the plain text through
			// styleSelected instead so the background spans the full row.
			plain := ansi.Strip(line)
			lineW := lipgloss.Width(plain)
			paneW := p.width - 1
			if lineW < paneW {
				plain += strings.Repeat(" ", paneW-lineW)
			}
			line = styleSelected.Render(plain)
		}

		sb.WriteString(line)
		sb.WriteString("\n")
		lineBudget--

		if graphConn != "" && lineBudget > 0 {
			sb.WriteString(" ")
			sb.WriteString(graphConn)
			sb.WriteString("\n")
			lineBudget--
		}
	}

	header := renderLogHeader(graphColW, hashColW, messageW, authorColW, dateColW)

	return " " + header + "\n" + sb.String()
}

// columnWidths computes the fixed widths for every column, shared by the
// header and every data row so they can never drift apart. graphColW,
// hashColW, authorColW, and dateColW are maxima over all commits (not just
// the visible window) so columns stay stable while scrolling; messageW is
// whatever horizontal space remains, used to pad/truncate the message
// column to a fixed width so author/date always start at the same offset.
func (p *logPane) columnWidths(graphLines []string) (graphColW, hashColW, messageW, authorColW, dateColW int) {
	for _, c := range p.commits {
		hashColW = max(hashColW, lipgloss.Width(c.ShortHash))
		dateColW = max(dateColW, lipgloss.Width(relativeTime(c.Timestamp)))
		authorColW = max(authorColW, lipgloss.Width(truncate(c.Author, 16)))
	}
	for _, gl := range graphLines {
		row, conn, _ := strings.Cut(gl, "\n")
		graphColW = max(graphColW, lipgloss.Width(row), lipgloss.Width(conn))
	}
	// Reserve room for a few lanes even when history is currently linear, so
	// the graph column reads as a proper wide lane rather than a thin sliver
	// that only widens once a branch/merge appears.
	graphColW = max(graphColW, minGraphColW)

	fixed := graphColW + 2 + hashColW + 2 + authorColW + len(authorGap) + dateColW + len(authorGap)
	messageW = max(p.width-2-fixed, 10)
	return
}

// renderLogHeader builds the column-label header row, aligned to the same
// fixed column widths used by the data rows below it.
func renderLogHeader(graphColW, hashColW, messageW, authorColW, dateColW int) string {
	graphHeader := styleHeader.Render(padRight("GRAPH", graphColW))
	hashHeader := styleHeader.Render(padRight("COMMIT", hashColW))
	messageHeader := styleHeader.Render(padRight("MESSAGE", messageW))
	authorHeader := styleHeader.Render(padRight("AUTHOR", authorColW))
	dateHeader := styleHeader.Render(padRight("DATE", dateColW))

	return graphHeader + "  " + hashHeader + "  " + messageHeader + authorGap + authorHeader + authorGap + dateHeader
}

// padRight pads s with trailing spaces so its rendered (ANSI-aware) width
// reaches w. If s is already at or beyond w, it is returned unchanged.
func padRight(s string, w int) string {
	sw := lipgloss.Width(s)
	if sw >= w {
		return s
	}
	return s + strings.Repeat(" ", w-sw)
}

// renderRefPills renders HEAD/local-branch/remote-branch/tag ref names as
// styled inline "pills" (comma-joined, no surrounding parens) meant to sit
// immediately before a commit's subject text.
func renderRefPills(refs []model.Ref) string {
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
			if r.IsHead {
				s = styleRefHead.Render("HEAD -> " + r.Name)
			} else {
				s = styleRefLocal.Render(r.Name)
			}
		case model.RefRemoteBranch:
			s = styleRefRemote.Render(r.Name)
		case model.RefTag:
			s = styleRefTag.Render("tag: " + r.Name)
		}
		parts = append(parts, s)
	}
	return strings.Join(parts, " ")
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
