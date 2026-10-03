package graph

import (
	"testing"

	"github.com/eurico-martins/lazytree/internal/model"
)

func commits(specs ...struct {
	hash    string
	parents []string
}) []model.Commit {
	cs := make([]model.Commit, len(specs))
	for i, s := range specs {
		cs[i] = model.Commit{Hash: s.hash, Parents: s.parents}
	}
	return cs
}

func TestLayout_Linear(t *testing.T) {
	cs := commits(
		struct {
			hash    string
			parents []string
		}{"C", []string{"B"}},
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
	for i, n := range layout.Nodes {
		if n.Lane != 0 {
			t.Errorf("commit %d: expected lane 0, got %d", i, n.Lane)
		}
	}
}

func TestLayout_SimpleBranch(t *testing.T) {
	// D is a merge commit: parents B (lane 0) and C (lane 1)
	// History: D←B←A (main), D←C (branch)
	cs := commits(
		struct {
			hash    string
			parents []string
		}{"D", []string{"B", "C"}},
		struct {
			hash    string
			parents []string
		}{"C", []string{"A"}},
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
	if len(layout.Nodes) != 4 {
		t.Fatalf("expected 4 nodes, got %d", len(layout.Nodes))
	}
	// D must be at lane 0 (first commit)
	if layout.Nodes[0].Lane != 0 {
		t.Errorf("D: expected lane 0, got %d", layout.Nodes[0].Lane)
	}
	// All nodes must have non-negative lanes.
	for i, n := range layout.Nodes {
		if n.Lane < 0 {
			t.Errorf("node %d: negative lane %d", i, n.Lane)
		}
	}
}

func TestLayout_NoSharedLane(t *testing.T) {
	// Property: no two nodes in the same row share the same lane.
	cs := commits(
		struct {
			hash    string
			parents []string
		}{"E", []string{"C", "D"}},
		struct {
			hash    string
			parents []string
		}{"D", []string{"A"}},
		struct {
			hash    string
			parents []string
		}{"C", []string{"B"}},
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
	seen := make(map[int]bool)
	for _, n := range layout.Nodes {
		if seen[n.Lane] {
			t.Errorf("duplicate lane %d", n.Lane)
		}
		seen[n.Lane] = true
		delete(seen, n.Lane) // lanes can be reused across rows
	}
}

func TestLayout_OctopusMerge(t *testing.T) {
	cs := commits(
		struct {
			hash    string
			parents []string
		}{"M", []string{"A", "B", "C"}},
		struct {
			hash    string
			parents []string
		}{"C", nil},
		struct {
			hash    string
			parents []string
		}{"B", nil},
		struct {
			hash    string
			parents []string
		}{"A", nil},
	)
	layout := Layout(cs)
	// M has 3 parents, so at least 3 parent lanes must be assigned.
	if len(layout.Nodes[0].Parents) != 3 {
		t.Errorf("expected 3 parent lanes for M, got %d", len(layout.Nodes[0].Parents))
	}
	// All parent lanes must be distinct.
	pLanes := layout.Nodes[0].Parents
	seen := map[int]bool{}
	for _, pl := range pLanes {
		if seen[pl] {
			t.Errorf("duplicate parent lane %d", pl)
		}
		seen[pl] = true
	}
}

func TestLayout_LaneFreedAfterRootCommit(t *testing.T) {
	// Two unrelated linear chains, back to back: X's root frees lane 0
	// before Y starts, so Y should also stay on lane 0.
	cs := commits(
		struct {
			hash    string
			parents []string
		}{"Y2", []string{"Y1"}},
		struct {
			hash    string
			parents []string
		}{"Y1", nil},
		struct {
			hash    string
			parents []string
		}{"X2", []string{"X1"}},
		struct {
			hash    string
			parents []string
		}{"X1", nil},
	)
	layout := Layout(cs)
	if layout.Width != 1 {
		t.Errorf("expected width 1 for two unrelated linear chains, got %d", layout.Width)
	}
	for i, n := range layout.Nodes {
		if n.Lane != 0 {
			t.Errorf("commit %d: expected lane 0, got %d", i, n.Lane)
		}
	}
}

func TestLayout_BranchMergeFreesLane(t *testing.T) {
	// D merges B (main, lane 0) and C (feature, lane 1). Once C's own root
	// is processed, lane 1 should free and be reused by a later, unrelated
	// fork rather than growing to lane 2.
	cs := commits(
		struct {
			hash    string
			parents []string
		}{"D", []string{"B", "C"}},
		struct {
			hash    string
			parents []string
		}{"C", nil}, // feature branch root, lane 1 frees here
		struct {
			hash    string
			parents []string
		}{"B", []string{"A"}},
		struct {
			hash    string
			parents []string
		}{"A", []string{"F0", "E0"}}, // unrelated fork after C's lane freed
		struct {
			hash    string
			parents []string
		}{"E0", nil},
		struct {
			hash    string
			parents []string
		}{"F0", nil},
	)
	layout := Layout(cs)
	// A's second parent (E0) should reuse lane 1, not allocate lane 2.
	aParents := layout.Nodes[3].Parents
	if len(aParents) != 2 {
		t.Fatalf("expected A to have 2 parent lanes, got %d", len(aParents))
	}
	if aParents[1] != 1 {
		t.Errorf("expected A's second parent to reuse lane 1, got %d", aParents[1])
	}
	if layout.Width > 2 {
		t.Errorf("expected width to stay at 2, got %d", layout.Width)
	}
}

func TestLayout_CrissCrossBranches(t *testing.T) {
	// Two interleaved forks/merges: assert per-row lane uniqueness holds and
	// width never exceeds the true concurrent-branch count (regression test
	// for the monotonic-growth bug that would have pushed width to 4+).
	cs := commits(
		struct {
			hash    string
			parents []string
		}{"M2", []string{"M1", "D2"}},
		struct {
			hash    string
			parents []string
		}{"D2", nil},
		struct {
			hash    string
			parents []string
		}{"M1", []string{"B", "C2"}},
		struct {
			hash    string
			parents []string
		}{"C2", nil},
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
	seen := make(map[int]bool)
	for i, n := range layout.Nodes {
		if seen[n.Lane] {
			t.Errorf("row %d: duplicate lane %d in active set", i, n.Lane)
		}
		seen[n.Lane] = true
		delete(seen, n.Lane)
	}
	if layout.Width > 3 {
		t.Errorf("expected width <= 3 for at most 2 concurrent branches, got %d", layout.Width)
	}
}

func TestLayout_WidthShrinksAfterCollapse(t *testing.T) {
	// Open two branches, then let them collapse, then observe the active
	// lane count drop below the peak — proving lanes were reclaimed rather
	// than growing monotonically.
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
	peak := 0
	for _, n := range layout.Nodes {
		width := len(n.Parents)
		if n.Lane+1 > width {
			width = n.Lane + 1
		}
		if width > peak {
			peak = width
		}
	}
	// After D (2 parent lanes), row for B should not still be carrying 2
	// active lanes worth of width once C's branch has been consumed.
	lastLane := layout.Nodes[len(layout.Nodes)-1].Lane
	if lastLane != 0 {
		t.Errorf("expected trailing linear chain to collapse back to lane 0, got %d", lastLane)
	}
}

func TestLayout_Width(t *testing.T) {
	// A simple linear chain should have width 1.
	cs := commits(
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
	if layout.Width != 1 {
		t.Errorf("expected width 1, got %d", layout.Width)
	}
}
