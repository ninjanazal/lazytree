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
