package ui

import (
	"strings"
	"testing"

	"github.com/eurico-martins/lazytree/internal/graph"
	"github.com/eurico-martins/lazytree/internal/model"
)

func zenTestPane(w, h int) zenPane {
	z := newZenPane()
	z.setSize(w, h)
	cs := longHistory(20)
	z.rebuild("demo", cs, graph.Layout(cs))
	return z
}

func TestZenPane_EmptyUntilSized(t *testing.T) {
	z := newZenPane()
	if z.View() != "" {
		t.Error("an unsized, unbuilt zen pane should render nothing")
	}
}

func TestZenPane_TreeGrowsThenBreathes(t *testing.T) {
	z := zenTestPane(80, 30)
	if z.revealed != 0 {
		t.Fatalf("a fresh zen pane starts with nothing revealed, got %d", z.revealed)
	}
	total := len(z.treeLines)
	if total == 0 {
		t.Fatal("tree lines should be built from the layout")
	}
	prev := 0
	for range total + zenGrowTargetTicks {
		z.advance()
		if z.revealed < prev {
			t.Fatal("the revealed tree must never shrink")
		}
		prev = z.revealed
	}
	if z.revealed != total {
		t.Errorf("after enough ticks the whole tree is revealed: %d of %d", z.revealed, total)
	}
	if z.breathDot() == zenCommitGlyph && len(zenBreathGlyphs) > 1 {
		// once grown the dot cycles through the breathing glyphs; at tick
		// phase 0 it may legitimately equal the first glyph, so only check
		// that advancing eventually changes it.
		seen := map[rune]bool{}
		for range len(zenBreathGlyphs) * zenBreathHold {
			z.advance()
			seen[z.breathDot()] = true
		}
		if len(seen) < 2 {
			t.Error("the commit dot should pulse once the tree is fully grown")
		}
	}
}

func TestZenPane_TitleRevealsToFinalText(t *testing.T) {
	z := zenTestPane(80, 30)
	for range zenRevealTicks + 5 {
		z.advance()
	}
	// After the reveal window each glyph is either final or a rare flicker;
	// the underlying characters spell the title.
	var final strings.Builder
	for _, c := range z.title {
		final.WriteRune(c.final)
	}
	if want := "DEMO · 20 COMMITS"; final.String() != want {
		t.Errorf("title = %q, want %q", final.String(), want)
	}
}

func TestZenPane_ViewFitsItsSize(t *testing.T) {
	z := zenTestPane(60, 12)
	for range 100 {
		z.advance()
	}
	view := z.View()
	if view == "" {
		t.Fatal("expected a rendered frame")
	}
	if got := strings.Count(view, "\n") + 1; got > 12 {
		t.Errorf("zen frame is %d lines, pane height is 12", got)
	}
}

func TestZenTreeLines_MatchesLayoutRows(t *testing.T) {
	cs := []model.Commit{
		{Hash: "b", Parents: []string{"a"}},
		{Hash: "a"},
	}
	lines := zenTreeLines(graph.Layout(cs))
	if len(lines) < 2 {
		t.Errorf("each commit contributes at least one row, got %d", len(lines))
	}
}
