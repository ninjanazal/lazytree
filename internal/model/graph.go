package model

type GraphNode struct {
	CommitIndex int
	Lane        int
	Parents     []int // which lane each parent lives in
	Color       int   // palette index
}

type GraphLayout struct {
	Nodes []GraphNode
	Width int // number of lanes in widest row
}
