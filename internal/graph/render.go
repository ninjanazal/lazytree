package graph

import (
	"maps"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/eurico-martins/lazytree/internal/model"
)

const (
	glyphCommit = "o"
	glyphVert   = "│"
	glyphHoriz  = "─"
	glyphArcDR  = "╭" // connects down and right
	glyphArcDL  = "╮" // connects down and left
	glyphArcUR  = "╰" // connects up and right
	glyphArcUL  = "╯" // connects up and left
	glyphSpace  = " "
)

// RenderCommitLine renders one full graph row for node: the commit dot and
// vertical pass-throughs for lanes active above this row, plus (when the
// topology changes) a second, diagonal connector line leading into next's
// lane layout. activeLanes reflects lanes open above this row on entry and
// is mutated in place to reflect lanes open below this row on return, ready
// for the caller's next call.
//
// The returned string contains a single line for a plain pass-through row,
// or two lines joined by "\n" when a merge/branch-collapse diagonal must be
// drawn between this row and the next.
func RenderCommitLine(node model.GraphNode, next *model.GraphNode, activeLanes map[int]bool, width int) string {
	if width < node.Lane+1 {
		width = node.Lane + 1
	}

	cols := make([]string, width)
	for i := range cols {
		if activeLanes[i] {
			cols[i] = colorize(glyphVert, i%len(LanePalette))
		} else {
			cols[i] = glyphSpace
		}
	}
	cols[node.Lane] = colorize(glyphCommit, node.Color)
	commitLine := strings.Join(cols, " ")

	// "old" activeLanes (above this row) vs "new" (below this row, owned by
	// this node's parents) — the diagonal segment needs both simultaneously.
	oldActive := activeLanes
	delete(oldActive, node.Lane)

	newActive := make(map[int]bool, len(oldActive)+len(node.Parents))
	maps.Copy(newActive, oldActive)
	for _, pl := range node.Parents {
		newActive[pl] = true
	}

	needsDiagonal := false
	for _, pl := range node.Parents {
		if pl != node.Lane {
			needsDiagonal = true
			break
		}
	}

	var connLine string
	if next != nil && needsDiagonal {
		connLine = renderConnector(node, newActive, width)
	}

	// Commit downstream state for the caller.
	clear(activeLanes)
	maps.Copy(activeLanes, newActive)

	if connLine == "" {
		return commitLine
	}
	return commitLine + "\n" + connLine
}

// renderConnector draws the diagonal transition between a commit row and the
// next, showing lanes that open (merge parents) or collapse (branch tips
// rejoining an already-active lane) at this row. newActive is the set of
// lanes open below this row; verticals are drawn for all of them, then
// overwritten with a diagonal glyph along the path of any parent lane that
// differs from node.Lane.
func renderConnector(node model.GraphNode, newActive map[int]bool, width int) string {
	cols := make([]string, width)
	for i := range cols {
		cols[i] = glyphSpace
	}

	for lane := range newActive {
		if lane < width {
			cols[lane] = colorize(glyphVert, lane%len(LanePalette))
		}
	}

	drawDiagonal := func(from, to, color int) {
		if to > from {
			if from < width {
				cols[from] = colorize(glyphArcUR, color)
			}
			for col := from + 1; col < to && col < width; col++ {
				cols[col] = colorize(glyphHoriz, color)
			}
			if to < width {
				cols[to] = colorize(glyphArcDL, color)
			}
		} else if to < from {
			if from < width {
				cols[from] = colorize(glyphArcUL, color)
			}
			for col := to + 1; col < from && col < width; col++ {
				cols[col] = colorize(glyphHoriz, color)
			}
			if to < width {
				cols[to] = colorize(glyphArcDR, color)
			}
		}
	}

	for _, pl := range node.Parents {
		if pl != node.Lane && pl < width {
			drawDiagonal(node.Lane, pl, node.Color)
		}
	}

	return strings.Join(cols, " ")
}

func colorize(s string, colorIdx int) string {
	if colorIdx < 0 || colorIdx >= len(LanePalette) {
		colorIdx = 0
	}
	return lipgloss.NewStyle().Foreground(LanePalette[colorIdx]).Render(s)
}

// StripColors removes ANSI escape codes for width calculations.
// Use lipgloss.Width() instead for accurate terminal width.
func GraphWidth(width int) int {
	// Each lane is 1 glyph + 1 space separator, minus trailing space.
	if width <= 0 {
		return 0
	}
	return width*2 - 1
}
