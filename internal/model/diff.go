package model

type DiffLineKind int

const (
	DiffContext DiffLineKind = iota
	DiffAdded
	DiffRemoved
	DiffHeader
	DiffHunkHeader
)

type DiffLine struct {
	Kind DiffLineKind
	Text string
	OldN int
	NewN int
}

type DiffHunk struct {
	Header string
	Lines  []DiffLine
}

type DiffFile struct {
	OldPath string
	NewPath string
	Status  string // A, M, D, R, C
	Binary  bool   // git reported "Binary files ... differ"; there are no hunks
	Hunks   []DiffHunk
}
