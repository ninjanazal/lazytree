package ui

import "github.com/eurico-martins/lazytree/internal/model"

type MsgCommitsLoaded struct {
	Commits []model.Commit
	Layout  model.GraphLayout
	Gen     int
}

type MsgCommitsBatch struct {
	Commits []model.Commit
	Offset  int
	Gen     int
}

// MsgFetchTick fires every fetchInterval while lazytree is open. It kicks
// off a background `git fetch` (if one isn't already running) and
// reschedules itself for the next interval.
type MsgFetchTick struct {
	Seq int
}

// MsgFetchResult reports that a background `git fetch` completed. Fetch
// failures are silently ignored -- they never populate m.err/MsgError and
// never surface in the UI; the next scheduled tick simply retries.
type MsgFetchResult struct {
	Err error
}

type MsgDiffLoaded struct {
	Hash  string
	Files []model.DiffFile
}

type MsgDiffDebounce struct {
	Hash string
	Seq  int
}

type MsgBranchInfo struct {
	Branch string
}

// MsgAnimationTick drives zen mode's title decrypt/flicker animation. It
// fires on a short fixed interval while zen mode is active and is not
// rescheduled once zen mode is left, so the tick chain dies on its own.
type MsgAnimationTick struct {
	Seq int
}

type MsgError struct {
	Err error
}

type MsgSearchResult struct {
	Query   string
	Indices []int
}
