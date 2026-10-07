package model

import "time"

type RefKind int

const (
	RefHead RefKind = iota
	RefLocalBranch
	RefRemoteBranch
	RefTag
	RefStash
	// RefStashHelper marks the extra commits git creates for a stash (its
	// staged-changes and untracked-files parents). They are folded out of the
	// log by git.FoldStash, never displayed.
	RefStashHelper
)

type Ref struct {
	Name   string
	Kind   RefKind
	IsHead bool
}

type Commit struct {
	Hash           string
	ShortHash      string
	Parents        []string
	Author         string
	AuthorEmail    string
	Committer      string
	CommitterEmail string
	Timestamp      time.Time
	Subject        string
	Body           string
	Refs           []Ref
}
