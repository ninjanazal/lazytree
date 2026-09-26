package ui

import (
	"github.com/eurico-martins/lazytree/internal/graph"
	"github.com/eurico-martins/lazytree/internal/model"
)

type MsgCommitsLoaded struct {
	Commits  []model.Commit
	Layout   model.GraphLayout
	Layouter *graph.Layouter // lane state to continue with on MsgCommitsBatch; may be nil
	Gen      int
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

// MsgDiffLoaded reports the result of a diff load, success or failure. A
// failed diff is not fatal to the app (Err is shown inside the diff pane
// itself, via diffPane.setError) -- see the M1 roadmap's "Done when: ... a
// bad diff doesn't kill the app".
type MsgDiffLoaded struct {
	Hash  string
	Files []model.DiffFile
	Err   error
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

// MsgCountdownTick fires every second for the lifetime of the app, purely to
// force a redraw of the footer's "next fetch in Ns" countdown.
type MsgCountdownTick struct {
	Seq int
}

// MsgError reports a failed commits load. Gen ties it to the reload that
// produced it, the same way MsgCommitsLoaded/MsgCommitsBatch are, so a
// stale error from a superseded reload can't clobber a newer one that's
// still in flight (e.g. a background fetch triggers a reload, then an
// earlier load's error arrives after the reload has already started).
type MsgError struct {
	Gen int
	Err error
}

type MsgSearchResult struct {
	Query   string
	Indices []int
}
