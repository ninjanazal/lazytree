package graph

import (
	"slices"

	"github.com/eurico-martins/lazytree/internal/model"
)

// Layouter incrementally assigns lanes to commits fed in git-log order across
// multiple Append calls. The ordering precondition is the same as Layout's:
// across all calls, commits must arrive reverse-chronologically, each before
// its parents, appended only (never reordered or backfilled). Not safe for
// concurrent use. The zero value is ready to use.
type Layouter struct {
	laneOf      map[string]int
	freeLanes   []int
	nextLane    int
	maxWidth    int
	activeLanes map[int]string
	nodes       []model.GraphNode
}

// NewLayouter returns a ready-to-use Layouter.
func NewLayouter() *Layouter {
	return &Layouter{}
}

// Append lays out commits as the continuation of everything appended so far
// and returns the cumulative GraphLayout.
func (l *Layouter) Append(commits []model.Commit) model.GraphLayout {
	if l.laneOf == nil {
		l.laneOf = make(map[string]int)
		l.activeLanes = make(map[int]string)
	}

	base := len(l.nodes)
	for i, c := range commits {
		// Step 1: Determine this commit's lane.
		lane, alreadyAssigned := l.laneOf[c.Hash]
		if !alreadyAssigned {
			lane = allocLane(&l.freeLanes, &l.nextLane)
			l.laneOf[c.Hash] = lane
		}
		// Remove from active since we're consuming this node.
		delete(l.activeLanes, lane)

		// Step 2: Assign lanes to parents.
		parentLanes := make([]int, len(c.Parents))
		for j, p := range c.Parents {
			if pl, ok := l.laneOf[p]; ok {
				parentLanes[j] = pl
			} else {
				var pl int
				if j == 0 {
					// First parent inherits our lane.
					pl = lane
				} else {
					// Merge parents get a fresh lane.
					pl = allocLane(&l.freeLanes, &l.nextLane)
				}
				l.laneOf[p] = pl
				parentLanes[j] = pl
			}
			// Mark the lane as active (owned by parent p).
			l.activeLanes[parentLanes[j]] = p
		}

		// Step 3: Free our lane if no parent inherits it (root commit, or a
		// branch tip whose first parent already belongs to another lane —
		// i.e. this branch is rejoining/collapsing into an already-active lane).
		if !slices.Contains(parentLanes, lane) {
			freeLane(&l.freeLanes, lane)
		}

		// Track width.
		width := currentWidth(l.activeLanes, lane)
		if width > l.maxWidth {
			l.maxWidth = width
		}

		l.nodes = append(l.nodes, model.GraphNode{
			CommitIndex: base + i,
			Lane:        lane,
			Parents:     parentLanes,
			Color:       lane % len(LanePalette),
		})
	}

	return l.Layout()
}

// Layout returns the cumulative layout built so far. The returned Nodes slice
// is capacity-capped so a later Append can never mutate a snapshot a caller
// is still holding (e.g. zen mode's rebuild).
func (l *Layouter) Layout() model.GraphLayout {
	if len(l.nodes) == 0 {
		return model.GraphLayout{}
	}
	n := len(l.nodes)
	return model.GraphLayout{
		Nodes: l.nodes[:n:n],
		Width: l.maxWidth + 1,
	}
}

// Layout computes lane assignments for all commits.
// Commits must be in reverse-chronological order (as git log emits them), with
// each commit appearing before its parents — lane freeing relies on this to
// know when a lane's owning branch has been fully consumed.
// The result is a GraphLayout where each node knows its lane and the lanes
// of its parents, enabling the renderer to draw edges between rows.
func Layout(commits []model.Commit) model.GraphLayout {
	if len(commits) == 0 {
		return model.GraphLayout{}
	}
	return NewLayouter().Append(commits)
}

func allocLane(free *[]int, next *int) int {
	if len(*free) > 0 {
		lane := (*free)[0]
		*free = (*free)[1:]
		return lane
	}
	lane := *next
	*next++
	return lane
}

func freeLane(free *[]int, lane int) {
	// Insert in sorted order so we prefer lower-numbered lanes.
	i := 0
	for i < len(*free) && (*free)[i] < lane {
		i++
	}
	*free = append(*free, 0)
	copy((*free)[i+1:], (*free)[i:])
	(*free)[i] = lane
}

// currentWidth returns the highest active lane index (including the current commit's lane).
func currentWidth(activeLanes map[int]string, commitLane int) int {
	max := commitLane
	for lane := range activeLanes {
		if lane > max {
			max = lane
		}
	}
	return max
}
