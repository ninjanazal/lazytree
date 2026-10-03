package graph

import (
	"fmt"
	"math/rand"
	"reflect"
	"testing"

	"github.com/eurico-martins/lazytree/internal/model"
)

// appendInBatches feeds cs into a single Layouter in chunks of the given
// sizes (repeating the last size to cover any remainder) and returns the
// cumulative layout.
func appendInBatches(cs []model.Commit, batchSize int) model.GraphLayout {
	l := NewLayouter()
	var layout model.GraphLayout
	for i := 0; i < len(cs); i += batchSize {
		end := min(i+batchSize, len(cs))
		layout = l.Append(cs[i:end])
	}
	return layout
}

// fixtures mirrors the commit histories used in layout_test.go, so the
// incremental Layouter is checked against the exact same scenarios the
// one-shot Layout is tested with.
func fixtures() map[string][]model.Commit {
	return map[string][]model.Commit{
		"Linear": commits(
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
		),
		"SimpleBranch": commits(
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
		),
		"NoSharedLane": commits(
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
		),
		"OctopusMerge": commits(
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
		),
		"LaneFreedAfterRootCommit": commits(
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
		),
		"BranchMergeFreesLane": commits(
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
			}{"A", []string{"F0", "E0"}},
			struct {
				hash    string
				parents []string
			}{"E0", nil},
			struct {
				hash    string
				parents []string
			}{"F0", nil},
		),
		"CrissCrossBranches": commits(
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
		),
		"WidthShrinksAfterCollapse": commits(
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
		),
	}
}

func TestLayouter_MatchesLayout_AllBatchSizes(t *testing.T) {
	for name, cs := range fixtures() {
		want := Layout(cs)
		for size := 1; size <= len(cs); size++ {
			got := appendInBatches(cs, size)
			if !reflect.DeepEqual(got, want) {
				t.Errorf("%s: batch size %d: got %+v, want %+v", name, size, got, want)
			}
		}
	}
}

func TestLayouter_MatchesLayout_SyntheticHistory(t *testing.T) {
	cs := syntheticHistory(2000, 1)
	want := Layout(cs)
	for _, size := range []int{1, 7, 500} {
		got := appendInBatches(cs, size)
		if !reflect.DeepEqual(got, want) {
			t.Errorf("batch size %d: got %d nodes / width %d, want %d nodes / width %d",
				size, len(got.Nodes), got.Width, len(want.Nodes), want.Width)
		}
	}
}

// syntheticHistory builds a pseudo-random commit history of n commits in
// git-log order (each commit before its parents): commit i's parents are
// always chosen from indices > i, so the ordering precondition holds. A mix
// of merges, octopus merges, and extra roots is included.
func syntheticHistory(n int, seed int64) []model.Commit {
	r := rand.New(rand.NewSource(seed))
	hash := func(i int) string { return fmt.Sprintf("c%d", i) }
	cs := make([]model.Commit, n)
	for i := range n {
		var parents []string
		switch {
		case i == n-1:
			// last commit is always a root
		case r.Intn(20) == 0:
			// occasional extra root
		case r.Intn(15) == 0 && i+2 < n:
			// merge or octopus merge
			numParents := 2 + r.Intn(2)
			seen := map[int]bool{}
			for len(parents) < numParents {
				p := i + 1 + r.Intn(n-i-1)
				if seen[p] {
					continue
				}
				seen[p] = true
				parents = append(parents, hash(p))
			}
		default:
			parents = []string{hash(i + 1)}
		}
		cs[i] = model.Commit{Hash: hash(i), Parents: parents}
	}
	return cs
}

func TestLayouter_FreeLaneReuseAcrossBatchBoundary(t *testing.T) {
	cs := fixtures()["BranchMergeFreesLane"]
	// Split as [D,C] | [B,A,E0,F0]: lane 1 frees while processing C in batch
	// 1, and A (in batch 2) must reuse it for its second parent E0, not
	// allocate a new lane 2.
	l := NewLayouter()
	l.Append(cs[:2])
	layout := l.Append(cs[2:])

	aParents := layout.Nodes[3].Parents
	if len(aParents) != 2 {
		t.Fatalf("expected A to have 2 parent lanes, got %d", len(aParents))
	}
	if aParents[1] != 1 {
		t.Errorf("expected A's second parent to reuse lane 1 across the batch boundary, got %d", aParents[1])
	}
	if layout.Width > 2 {
		t.Errorf("expected width to stay at 2, got %d", layout.Width)
	}

	want := Layout(cs)
	if !reflect.DeepEqual(layout, want) {
		t.Errorf("got %+v, want %+v", layout, want)
	}
}

func TestLayouter_PendingLaneAcrossBatchBoundary(t *testing.T) {
	cs := fixtures()["BranchMergeFreesLane"]
	// Split as [D] | [C,B,A,E0,F0]: C's lane (1) is assigned while processing
	// D in batch 1, and must still be honored when C itself arrives in batch 2.
	l := NewLayouter()
	l.Append(cs[:1])
	layout := l.Append(cs[1:])

	if layout.Nodes[1].Lane != 1 {
		t.Errorf("expected C to land on lane 1 (assigned by D in the prior batch), got %d", layout.Nodes[1].Lane)
	}

	want := Layout(cs)
	if !reflect.DeepEqual(layout, want) {
		t.Errorf("got %+v, want %+v", layout, want)
	}

	// Sanity check: this fixture genuinely needs carried state — running
	// Layout independently per batch and concatenating nodes would not match.
	naive := Layout(cs[:1])
	naive2 := Layout(cs[1:])
	if reflect.DeepEqual(append(append([]model.GraphNode{}, naive.Nodes...), naive2.Nodes...), want.Nodes) {
		t.Fatalf("fixture does not exercise cross-batch state: naive per-batch Layout matched the one-shot result")
	}
}

func TestLayouter_FreshAfterReload(t *testing.T) {
	old := fixtures()["BranchMergeFreesLane"]
	l := NewLayouter()
	l.Append(old[:2])
	l.Append(old[2:])

	// New, unrelated history reusing some hashes (A, B) that the old
	// Layouter already has lane assignments for.
	next := commits(
		struct {
			hash    string
			parents []string
		}{"B", []string{"A"}},
		struct {
			hash    string
			parents []string
		}{"A", nil},
	)

	fresh := NewLayouter()
	got := fresh.Append(next)
	want := Layout(next)
	if !reflect.DeepEqual(got, want) {
		t.Errorf("fresh Layouter after unrelated reload: got %+v, want %+v", got, want)
	}
}

func TestLayouter_SnapshotNotMutated(t *testing.T) {
	cs := fixtures()["BranchMergeFreesLane"]
	l := NewLayouter()
	snapshot := l.Append(cs[:2])
	snapshotCopy := model.GraphLayout{
		Nodes: append([]model.GraphNode{}, snapshot.Nodes...),
		Width: snapshot.Width,
	}

	l.Append(cs[2:])

	if !reflect.DeepEqual(snapshot, snapshotCopy) {
		t.Errorf("snapshot mutated by later Append: got %+v, want %+v", snapshot, snapshotCopy)
	}
}

func TestLayouter_Empty(t *testing.T) {
	l := NewLayouter()
	if got := l.Layout(); !reflect.DeepEqual(got, model.GraphLayout{}) {
		t.Errorf("zero-value Layout(): got %+v, want zero value", got)
	}
	if got := l.Append(nil); !reflect.DeepEqual(got, model.GraphLayout{}) {
		t.Errorf("Append(nil): got %+v, want zero value", got)
	}
}
