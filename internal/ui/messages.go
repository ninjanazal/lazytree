package ui

import "github.com/eurico-martins/lazytree/internal/model"

type MsgCommitsLoaded struct {
	Commits []model.Commit
	Layout  model.GraphLayout
}

type MsgCommitsBatch struct {
	Commits []model.Commit
	Offset  int
}

type MsgDiffLoaded struct {
	Hash  string
	Files []model.DiffFile
}

type MsgInspectLoaded struct {
	Commit       model.Commit
	ChangedFiles []string
}

type MsgError struct {
	Err error
}

type MsgSearchResult struct {
	Query   string
	Indices []int
}
