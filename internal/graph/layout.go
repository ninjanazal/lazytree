package graph

import (
	"slices"

	"github.com/eurico-martins/lazytree/internal/model"
)

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

	laneOf := make(map[string]int, len(commits))
	// freeLanes is a sorted stack of available lane indices.
	var freeLanes []int
	nextLane := 0
	maxWidth := 0

	// activeLanes tracks which lanes are currently "open" (have a pending child).
	// We use this to draw continuation lines between rows.
	// activeLanes[lane] = parent hash that owns this lane (or "" if the lane is the commit's own).
	activeLanes := make(map[int]string)

	nodes := make([]model.GraphNode, len(commits))

	for i, c := range commits {
		// Step 1: Determine this commit's lane.
		lane, alreadyAssigned := laneOf[c.Hash]
		if !alreadyAssigned {
			lane = allocLane(&freeLanes, &nextLane)
			laneOf[c.Hash] = lane
		}
		// Remove from active since we're consuming this node.
		delete(activeLanes, lane)

		// Step 2: Assign lanes to parents.
		parentLanes := make([]int, len(c.Parents))
		for j, p := range c.Parents {
			if pl, ok := laneOf[p]; ok {
				parentLanes[j] = pl
			} else {
				var pl int
				if j == 0 {
					// First parent inherits our lane.
					pl = lane
				} else {
					// Merge parents get a fresh lane.
					pl = allocLane(&freeLanes, &nextLane)
				}
				laneOf[p] = pl
				parentLanes[j] = pl
			}
			// Mark the lane as active (owned by parent p).
			activeLanes[parentLanes[j]] = p
		}

		// Step 3: Free our lane if no parent inherits it (root commit, or a
		// branch tip whose first parent already belongs to another lane —
		// i.e. this branch is rejoining/collapsing into an already-active lane).
		if !slices.Contains(parentLanes, lane) {
			freeLane(&freeLanes, lane)
		}

		// Track width.
		width := currentWidth(activeLanes, lane)
		if width > maxWidth {
			maxWidth = width
		}

		nodes[i] = model.GraphNode{
			CommitIndex: i,
			Lane:        lane,
			Parents:     parentLanes,
			Color:       lane % len(LanePalette),
		}
	}

	return model.GraphLayout{
		Nodes: nodes,
		Width: maxWidth + 1,
	}
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
