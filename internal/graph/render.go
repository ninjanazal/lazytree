package graph

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/eurico-martins/lazytree/internal/model"
)

const (
	glyphCommit    = "●"
	glyphVert      = "│"
	glyphDiagRight = "╲"
	glyphDiagLeft  = "╱"
	glyphSpace = " "
)

// RenderGraph returns two strings for a commit row:
// - commitLine: the graph glyphs for the commit itself (●, │, etc.)
// - connLine:   the connecting line between this row and the next
//
// width is the total number of lanes to render.
func RenderGraph(node, next *model.GraphNode, width int) (commitLine, connLine string) {
	// Build commit line.
	cols := make([]string, width)
	for i := range cols {
		cols[i] = glyphSpace
	}

	// Draw vertical pass-throughs for all lanes < node.Lane.
	// (Lanes > node.Lane that are active would need context we don't have here;
	// we rely on the caller to pass an accurate next node.)
	cols[node.Lane] = colorize(glyphCommit, node.Color)

	commitLine = strings.Join(cols, " ")

	// Build connector line (between this row and next).
	if next == nil {
		connLine = ""
		return
	}

	connCols := make([]string, width)
	for i := range connCols {
		connCols[i] = glyphSpace
	}

	// Vertical lines for parents that stay in the same lane.
	for j, pl := range node.Parents {
		if pl < width {
			if pl == node.Parents[0] && j == 0 {
				connCols[pl] = colorize(glyphVert, node.Color)
			} else {
				connCols[pl] = colorize(glyphVert, pl%len(LanePalette))
			}
		}
	}

	connLine = strings.Join(connCols, " ")
	return
}

// RenderCommitLine renders a single commit row with full graph context.
// activeLanes is the set of lane indices that are "open" (have commits below them).
func RenderCommitLine(node model.GraphNode, activeLanes map[int]bool, width int) string {
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
	return strings.Join(cols, " ")
}

// RenderConnectorLine renders the inter-row connector between two consecutive nodes.
func RenderConnectorLine(cur, next model.GraphNode, activeLanes map[int]bool, width int) string {
	if width < 1 {
		width = 1
	}
	cols := make([]string, width)
	for i := range cols {
		cols[i] = glyphSpace
	}

	// Vertical continuations for all active lanes.
	for i := range cols {
		if activeLanes[i] {
			cols[i] = colorize(glyphVert, i%len(LanePalette))
		}
	}

	// Diagonal for merge parents that are opening new lanes.
	for _, pl := range cur.Parents[1:] {
		if pl >= width {
			continue
		}
		// Draw diagonal from cur.Lane to pl.
		if pl > cur.Lane {
			for col := cur.Lane + 1; col <= pl && col < width; col++ {
				cols[col] = colorize(glyphDiagRight, cur.Color)
			}
		} else if pl < cur.Lane {
			for col := pl; col < cur.Lane && col < width; col++ {
				cols[col] = colorize(glyphDiagLeft, cur.Color)
			}
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
