package ui

import (
	"fmt"
	"maps"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
	"github.com/eurico-martins/lazytree/internal/graph"
	"github.com/eurico-martins/lazytree/internal/model"
)

type logPane struct {
	commits []model.Commit
	layout  model.GraphLayout
	cursor  int
	offset  int // scroll offset
	height  int
	width   int
	showAll bool
	search  logSearch
	refs    refFilter // which ref pills to hide (display only)

	// Cached render state, kept current by setCommits/setSize. View() must
	// never recompute these directly: with tens of thousands of commits,
	// doing so on every frame (cursor move, spinner tick, 1s countdown)
	// would dominate render cost. Both setters run against the pointer
	// AppModel.Update holds, so the cache persists across frames; View has
	// a value receiver and any writes made from inside it would be
	// discarded when the frame returns.
	//
	// setCommits always rebuilds from scratch; appendCommits (used for each
	// page of the paginated load) renders only the new commits so loading
	// stays O(n) overall. cachedActiveLanes carries RenderCommitLine's lane
	// state entering the last row, so an append can resume from there.
	cachedGraphLines  []string
	cachedActiveLanes map[int]bool
	cachedGraphColW   int
	cachedHashColW    int
	cachedAuthorColW  int
	cachedMessageW    int
}

// dateColW is fixed at the widest relativeTime label ("just now",
// "11mo ago") because those labels change with wall-clock time; a width
// measured once and cached could later be too narrow.
const dateColW = 8

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
	widthChanged := w != p.width
	p.width = w
	p.height = h
	if widthChanged {
		// Only messageW depends on pane width; the graph/hash/author/date
		// widths are independent of it, so there's no need to touch the
		// (potentially expensive) graph-line render on a resize.
		p.recomputeMessageW()
	}
	p.ensureVisible()
}

// setCommits replaces the commit list and rebuilds the render cache.
func (p *logPane) setCommits(commits []model.Commit, layout model.GraphLayout) {
	p.updateCommits(commits, layout, 0)
}

// appendCommits adds the next page of the same history. layout must be the
// cumulative layout from the same Layouter that produced the existing rows.
func (p *logPane) appendCommits(newCommits []model.Commit, layout model.GraphLayout) {
	from := len(p.commits)
	p.updateCommits(append(p.commits, newCommits...), layout, from)
}

func (p *logPane) updateCommits(commits []model.Commit, layout model.GraphLayout, from int) {
	var prevHash string
	if c := p.selectedCommit(); c != nil {
		prevHash = c.Hash
	}
	prevDelta := p.cursor - p.offset

	p.commits = commits
	p.layout = layout
	p.extendCache(from)
	p.extendSearch(from)

	if prevHash != "" {
		if idx := indexOfHash(commits, prevHash); idx >= 0 {
			p.cursor = idx
			p.offset = max(0, p.cursor-prevDelta)
			return
		}
	}

	if p.cursor >= len(commits) {
		p.cursor = max(0, len(commits)-1)
	}
	p.offset = 0
}

func indexOfHash(commits []model.Commit, hash string) int {
	for i, c := range commits {
		if c.Hash == hash {
			return i
		}
	}
	return -1
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

// rowHeight is how many screen lines the visible row idx takes: 1, or 2
// when its graph cell carries a connector line (a merge/fork diagonal)
// drawn beneath it.
func (p *logPane) rowHeight(idx int) int {
	commitIdx := p.visibleIndex(idx)
	if commitIdx >= 0 && commitIdx < len(p.cachedGraphLines) &&
		strings.Contains(p.cachedGraphLines[commitIdx], "\n") {
		return 2
	}
	return 1
}

// viewRows is the number of screen lines available for commit rows (the
// pane height minus the column-header row).
func (p *logPane) viewRows() int {
	return max(p.height-1, 1)
}

// ensureVisible adjusts the scroll offset so the cursor row, including its
// connector line, lies fully inside the window. Scrolling counts screen
// lines rather than commits, because connector rows make some commits two
// lines tall.
func (p *logPane) ensureVisible() {
	count := p.visibleCount()
	if count == 0 {
		p.offset = 0
		return
	}
	p.cursor = min(max(p.cursor, 0), count-1)
	p.offset = min(max(p.offset, 0), p.cursor)
	rows := p.viewRows()
	used := 0
	for i := p.offset; i <= p.cursor; i++ {
		used += p.rowHeight(i)
	}
	for used > rows && p.offset < p.cursor {
		used -= p.rowHeight(p.offset)
		p.offset++
	}
}

// rowAtLine returns the visible row shown at line y of the commit area
// (0 = first commit line, just under the column header), or -1 if y is
// below the last row. A commit's connector line belongs to that commit.
func (p *logPane) rowAtLine(y int) int {
	if y < 0 {
		return -1
	}
	line := 0
	for idx := p.offset; idx < p.visibleCount(); idx++ {
		line += p.rowHeight(idx)
		if y < line {
			return idx
		}
	}
	return -1
}

func (p *logPane) moveUp() {
	if p.cursor > 0 {
		p.cursor--
		p.ensureVisible()
	}
}

func (p *logPane) moveDown() {
	if p.cursor < p.visibleCount()-1 {
		p.cursor++
		p.ensureVisible()
	}
}

func (p *logPane) moveTop() {
	p.cursor = 0
	p.offset = 0
}

func (p *logPane) moveBottom() {
	p.cursor = max(0, p.visibleCount()-1)
	p.ensureVisible()
}

func (p *logPane) pageUp() {
	n := max(p.viewRows()-1, 1)
	p.cursor = max(0, p.cursor-n)
	p.offset = max(0, p.offset-n)
	p.ensureVisible()
}

func (p *logPane) pageDown() {
	n := max(p.viewRows()-1, 1)
	p.cursor = min(p.visibleCount()-1, p.cursor+n)
	p.offset += n
	p.ensureVisible()
}

func (p *logPane) visibleCount() int {
	return len(p.commits)
}

// visibleIndex maps a row position to a commit index, or -1 if out of range.
func (p *logPane) visibleIndex(cursor int) int {
	if cursor < 0 || cursor >= len(p.commits) {
		return -1
	}
	return cursor
}

// extendCache renders rows from index `from` onward and folds them into the
// cached column widths. from == 0 is a full rebuild; from > 0 continues an
// append, redoing row from-1 because it was rendered as the last row (next
// == nil) and may now need a diagonal connector to its new successor.
func (p *logPane) extendCache(from int) {
	nodes := p.layout.Nodes
	if from <= 0 || from > len(p.cachedGraphLines) {
		from = 0
		p.cachedGraphLines = nil
		p.cachedActiveLanes = nil
		p.cachedGraphColW, p.cachedHashColW, p.cachedAuthorColW = 0, 0, 0
	}

	startLines := max(from-1, 0)
	p.cachedGraphLines = p.cachedGraphLines[:startLines]

	activeLanes := make(map[int]bool, len(p.cachedActiveLanes))
	if startLines > 0 {
		maps.Copy(activeLanes, p.cachedActiveLanes)
	}
	for i := startLines; i < len(nodes); i++ {
		var next *model.GraphNode
		if i+1 < len(nodes) {
			next = &nodes[i+1]
		}
		if i == len(nodes)-1 {
			p.cachedActiveLanes = maps.Clone(activeLanes)
		}
		line := graph.RenderCommitLine(nodes[i], next, activeLanes, p.layout.Width)
		p.cachedGraphLines = append(p.cachedGraphLines, line)

		row, conn, _ := strings.Cut(line, "\n")
		p.cachedGraphColW = max(p.cachedGraphColW, lipgloss.Width(row), lipgloss.Width(conn))
	}
	// Reserve room for a few lanes even when history is currently linear, so
	// the graph column reads as a proper wide lane rather than a thin sliver
	// that only widens once a branch/merge appears.
	p.cachedGraphColW = max(p.cachedGraphColW, minGraphColW)

	for _, c := range p.commits[from:] {
		p.cachedHashColW = max(p.cachedHashColW, lipgloss.Width(c.ShortHash))
		p.cachedAuthorColW = max(p.cachedAuthorColW, lipgloss.Width(truncate(c.Author, 16)))
	}

	p.recomputeMessageW()
}

// recomputeMessageW derives the flexible message-column width from the
// other (independent of pane width) cached column widths. Split out from
// extendCache so a pane resize can refresh it without re-rendering the
// graph.
func (p *logPane) recomputeMessageW() {
	fixed := p.cachedGraphColW + 2 + p.cachedHashColW + 2 + p.cachedAuthorColW + len(authorGap) + dateColW + len(authorGap)
	p.cachedMessageW = max(p.width-2-fixed, 10)
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
	// correct even when a filter hides some rows' text. Read the cache
	// built by setCommits/setSize rather than recomputing here.
	graphLines := p.cachedGraphLines
	graphColW, hashColW, messageW, authorColW := p.cachedGraphColW, p.cachedHashColW, p.cachedMessageW, p.cachedAuthorColW

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
		q := p.search.highlightQuery()
		hash := padRight(highlightMatches(c.ShortHash, q, lipgloss.NewStyle().Foreground(hashColor).Bold(true)), hashColW)
		subject := c.Subject
		author := padRight(highlightMatches(truncate(c.Author, 16), q, styleAuthor), authorColW)
		date := styleDate.Width(dateColW).Render(relativeTime(c.Timestamp))
		pills := renderRefPills(p.refs.apply(c.Refs))

		// Available width for subject within the fixed message column.
		pillsW := lipgloss.Width(pills)
		subjectW := messageW - pillsW
		if pills != "" {
			subjectW--
		}
		if subjectW < 5 && pillsW < messageW-5 {
			subjectW = 5
		}
		subject = highlightMatches(truncate(subject, subjectW), q, lipgloss.NewStyle())

		messageField := subject
		if pills != "" {
			if subjectW <= 0 {
				messageField = ansi.Truncate(pills, messageW, "…")
			} else {
				messageField = pills + " " + subject
			}
		}
		messageField = padRight(messageField, messageW)

		marker := " "
		if p.search.matches(commitIdx) {
			marker = styleMatchMarker.Render("▌")
		}
		line := fmt.Sprintf("%s%s  %s  %s%s%s%s%s",
			marker, graphRow, hash, messageField, authorGap, author, authorGap, date,
		)

		if selected {
			line = highlightRow(line, p.width-1)
		}

		// On a terminal narrower than the fixed columns (or with ref pills
		// wider than the message column), clip the row rather than let it
		// overflow and wrap.
		sb.WriteString(ansi.Truncate(line, p.width, ""))
		sb.WriteString("\n")
		lineBudget--

		if graphConn != "" && lineBudget > 0 {
			sb.WriteString(ansi.Truncate(" "+graphConn, p.width, ""))
			sb.WriteString("\n")
			lineBudget--
		}
	}

	header := renderLogHeader(graphColW, hashColW, messageW, authorColW, dateColW)

	// Every row above is written with a trailing "\n"; drop the last one so
	// the pane is exactly header + rows tall. Callers join panes with "\n",
	// and a trailing newline here would add a phantom blank line.
	out := ansi.Truncate(" "+header, p.width, "") + "\n" + strings.TrimSuffix(sb.String(), "\n")

	// Pad short lists with blank lines up to the pane height so the footer
	// and fetch bar stay pinned to the bottom of the terminal rather than
	// floating right under the last commit.
	if missing := p.height - (strings.Count(out, "\n") + 1); missing > 0 {
		out += strings.Repeat("\n", missing)
	}
	return out
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

// highlightRow marks a row as selected by giving it a background across the
// full width while keeping the foreground colors of its parts (lane colors,
// ref pills, author, ...). Every reset sequence inside line would otherwise
// end the background early, so it is re-applied after each one.
func highlightRow(line string, width int) string {
	if pad := width - lipgloss.Width(line); pad > 0 {
		line += strings.Repeat(" ", pad)
	}
	bg := lipgloss.NewStyle().Background(colorSelectedBg).Render("x")
	seq, _, found := strings.Cut(bg, "x")
	if !found || seq == "" {
		// No color support: fall back to reverse video so the row is
		// still distinguishable.
		return lipgloss.NewStyle().Reverse(true).Render(ansi.Strip(line))
	}
	const reset = "\x1b[0m"
	return seq + strings.ReplaceAll(line, reset, reset+seq) + reset
}
