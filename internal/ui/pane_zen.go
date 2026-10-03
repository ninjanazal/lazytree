package ui

import (
	"fmt"
	"math/rand/v2"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/eurico-martins/lazytree/internal/graph"
	"github.com/eurico-martins/lazytree/internal/model"
)

// zenScrambleChars are the glyphs shown in place of a not-yet-resolved
// title character while it "decrypts" into place.
const zenScrambleChars = "!<>-_\\/[]{}—=+*^?#~$%&@01"

// zenRevealTicks is the tick budget (see animInterval) a title character is
// given to resolve; each character picks its own random reveal tick within
// it, so the title fizzles in unevenly rather than as a uniform wipe.
const zenRevealTicks = 26

// zenFlickerOdds is the 1-in-N chance an already-resolved title character
// glitches back to a scramble glyph for a single tick, once the title has
// settled — a small ambient shimmer so zen mode doesn't go fully static.
const zenFlickerOdds = 250

// zenGrowTargetTicks is roughly how many animation ticks the tree takes to
// grow in fully, regardless of how many commits it has.
const zenGrowTargetTicks = 30

// zenCommitGlyph is the rune graph.RenderCommitLine uses for a commit dot
// (see internal/graph/render.go's glyphCommit); breathing swaps it for a
// smaller/larger sibling glyph without touching the rest of an
// already-color-rendered row.
const zenCommitGlyph = '●'

// zenBreathGlyphs cycles a commit dot through these sizes once the tree has
// fully grown in, so the whole tree pulses gently in place like it's
// breathing.
var zenBreathGlyphs = []rune{'●', '◉', '○', '◉'}

const zenBreathHold = 10 // ticks each breath phase is held

type zenChar struct {
	final    rune
	revealAt int // tick at which this char locks in; 0 for spaces (never scrambled)
}

// zenPane renders an animated, decrypting title (repo name + live commit
// count) above the commit graph itself — the same lanes and connectors as
// the normal log view's graph column, no frame, no hash/subject/author/
// date. The tree grows in from its root, then breathes gently in place.
// Meant to be left open and glanced at, not worked from.
type zenPane struct {
	width, height int
	title         []zenChar
	treeLines     []string
	revealed      int // tree rows revealed so far, growing from the root (end of treeLines)
	linesPerTick  int
	tick          int
}

func newZenPane() zenPane {
	return zenPane{}
}

func (z *zenPane) setSize(w, h int) {
	z.width, z.height = w, h
}

// rebuild sets the title text and tree, restarting both animations from
// scratch.
func (z *zenPane) rebuild(repoName string, commits []model.Commit, layout model.GraphLayout) {
	text := strings.ToUpper(fmt.Sprintf("%s · %d commits", repoName, len(commits)))
	runes := []rune(text)
	z.title = make([]zenChar, len(runes))
	for i, r := range runes {
		revealAt := 0
		if r != ' ' {
			revealAt = rand.IntN(zenRevealTicks) + 1
		}
		z.title[i] = zenChar{final: r, revealAt: revealAt}
	}

	z.treeLines = zenTreeLines(layout)
	z.revealed = 0
	z.linesPerTick = max(len(z.treeLines)/zenGrowTargetTicks, 1)
	z.tick = 0
}

// zenTreeLines flattens the graph into plain rows (graph column only),
// reusing the exact same lane/connector rendering as the normal log pane.
func zenTreeLines(layout model.GraphLayout) []string {
	var lines []string
	activeLanes := make(map[int]bool)
	for i, node := range layout.Nodes {
		var next *model.GraphNode
		if i+1 < len(layout.Nodes) {
			next = &layout.Nodes[i+1]
		}
		rendered := graph.RenderCommitLine(node, next, activeLanes, layout.Width)
		row, conn, _ := strings.Cut(rendered, "\n")
		lines = append(lines, row)
		if conn != "" {
			lines = append(lines, conn)
		}
	}
	return lines
}

func (z *zenPane) advance() {
	z.tick++
	if z.revealed < len(z.treeLines) {
		z.revealed = min(z.revealed+z.linesPerTick, len(z.treeLines))
	}
}

func (z zenPane) titleString() string {
	style := lipgloss.NewStyle().Foreground(graph.LanePalette[0]).Bold(true)
	var b strings.Builder
	for _, c := range z.title {
		b.WriteRune(z.glyphFor(c))
	}
	return style.Render(b.String())
}

func (z zenPane) glyphFor(c zenChar) rune {
	if c.revealAt == 0 {
		return c.final
	}
	if z.tick < c.revealAt {
		return rune(zenScrambleChars[rand.IntN(len(zenScrambleChars))])
	}
	if rand.IntN(zenFlickerOdds) == 0 {
		return rune(zenScrambleChars[rand.IntN(len(zenScrambleChars))])
	}
	return c.final
}

// breathDot returns the commit-dot glyph for the current tick: the full dot
// while the tree is still growing, otherwise a slowly cycling size to give
// a breathing pulse.
func (z zenPane) breathDot() rune {
	if z.revealed < len(z.treeLines) {
		return zenCommitGlyph
	}
	period := len(zenBreathGlyphs) * zenBreathHold
	phase := (z.tick % period) / zenBreathHold
	return zenBreathGlyphs[phase]
}

func (z zenPane) View() string {
	if z.width == 0 || z.height == 0 || len(z.title) == 0 {
		return ""
	}

	var body strings.Builder
	body.WriteString(z.titleString())

	if len(z.treeLines) > 0 && z.revealed > 0 {
		body.WriteString("\n\n")

		start := len(z.treeLines) - z.revealed
		visible := z.treeLines[start:]
		if rowBudget := max(z.height-6, 1); len(visible) > rowBudget {
			visible = visible[len(visible)-rowBudget:]
		}

		dot := string(z.breathDot())
		for _, line := range visible {
			body.WriteString(strings.ReplaceAll(line, string(zenCommitGlyph), dot))
			body.WriteString("\n")
		}
	}

	content := strings.TrimRight(body.String(), "\n")
	return lipgloss.Place(z.width, z.height, lipgloss.Center, lipgloss.Center, content)
}
