package graph

import (
	"regexp"
	"testing"

	"github.com/eurico-martins/lazytree/internal/model"
)

var ansiRE = regexp.MustCompile("\x1b\\[[0-9;]*m")

func stripANSI(s string) string {
	return ansiRE.ReplaceAllString(s, "")
}

func TestRenderCommitLine_LinearNoDiagonal(t *testing.T) {
	// A plain single-lane continuation stays on one row — no extra
	// connector row is spent unless the topology actually changes here, so
	// straight history stays compact.
	node := model.GraphNode{CommitIndex: 0, Lane: 0, Parents: []int{0}, Color: 0}
	next := model.GraphNode{CommitIndex: 1, Lane: 0, Parents: nil, Color: 0}
	activeLanes := map[int]bool{}
	got := stripANSI(RenderCommitLine(node, &next, activeLanes, 1))
	want := "●"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
	if !activeLanes[0] {
		t.Errorf("lane 0 should remain active for a first-parent continuation")
	}
}

func TestRenderCommitLine_RootCommitNoConnector(t *testing.T) {
	// A root commit (no parents) has nothing continuing below it, so no
	// connector line should be drawn even though there is a next node.
	node := model.GraphNode{CommitIndex: 0, Lane: 0, Parents: nil, Color: 0}
	next := model.GraphNode{CommitIndex: 1, Lane: 0, Parents: nil, Color: 0}
	activeLanes := map[int]bool{}
	got := stripANSI(RenderCommitLine(node, &next, activeLanes, 1))
	want := "●"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestRenderCommitLine_MergeDiagonal(t *testing.T) {
	// D merges B (lane 0, continues) and C (lane 1, opens).
	node := model.GraphNode{CommitIndex: 0, Lane: 0, Parents: []int{0, 1}, Color: 0}
	next := model.GraphNode{CommitIndex: 1, Lane: 1, Parents: nil, Color: 1}
	activeLanes := map[int]bool{}
	got := stripANSI(RenderCommitLine(node, &next, activeLanes, 2))
	wantCommit := "●  "
	gotCommit, gotConn, hasConn := cutLine(got)
	if gotCommit != wantCommit {
		t.Errorf("commit line: got %q, want %q", gotCommit, wantCommit)
	}
	if !hasConn {
		t.Fatalf("expected a connector line for an opening merge lane, got none")
	}
	wantConn := "╰ ╮"
	if gotConn != wantConn {
		t.Errorf("connector line: got %q, want %q", gotConn, wantConn)
	}
	if !activeLanes[0] || !activeLanes[1] {
		t.Errorf("both lanes should be active after the merge row, got %v", activeLanes)
	}
}

func TestRenderCommitLine_BranchCollapse(t *testing.T) {
	// A commit whose own lane (1) collapses because its first parent
	// already belongs to lane 0 (the branch is rejoining the trunk).
	node := model.GraphNode{CommitIndex: 0, Lane: 1, Parents: []int{0}, Color: 1}
	next := model.GraphNode{CommitIndex: 1, Lane: 0, Parents: nil, Color: 0}
	activeLanes := map[int]bool{0: true}
	got := stripANSI(RenderCommitLine(node, &next, activeLanes, 2))
	_, gotConn, hasConn := cutLine(got)
	if !hasConn {
		t.Fatalf("expected a connector line for a collapsing lane, got none")
	}
	wantConn := "╭ ╯"
	if gotConn != wantConn {
		t.Errorf("connector line: got %q, want %q", gotConn, wantConn)
	}
	// Lane 1 must not still be active after the collapse.
	if activeLanes[1] {
		t.Errorf("collapsed lane 1 should not remain active, got %v", activeLanes)
	}
}

func TestRenderCommitLine_EndToEnd(t *testing.T) {
	cs := commits(
		struct {
			hash    string
			parents []string
		}{"D", []string{"B", "C"}},
		struct {
			hash    string
			parents []string
		}{"C", nil},
		struct {
			hash    string
			parents []string
		}{"B", []string{"A"}},
		struct {
			hash    string
			parents []string
		}{"A", nil},
	)
	layout := Layout(cs)
	activeLanes := map[int]bool{}
	var lines []string
	for i, node := range layout.Nodes {
		var next *model.GraphNode
		if i+1 < len(layout.Nodes) {
			next = &layout.Nodes[i+1]
		}
		out := stripANSI(RenderCommitLine(node, next, activeLanes, layout.Width))
		lines = append(lines, out)
	}
	if len(lines) != 4 {
		t.Fatalf("expected 4 rendered rows, got %d", len(lines))
	}
	// D's row must contain a connector (merge opens lane 1).
	if _, _, hasConn := cutLine(lines[0]); !hasConn {
		t.Errorf("expected D's row to have a connector line, got %q", lines[0])
	}
}

func cutLine(s string) (first, second string, hasSecond bool) {
	for i := range s {
		if s[i] == '\n' {
			return s[:i], s[i+1:], true
		}
	}
	return s, "", false
}
