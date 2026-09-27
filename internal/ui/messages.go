package ui

import (
	"github.com/eurico-martins/lazytree/internal/git"
	"github.com/eurico-martins/lazytree/internal/graph"
	"github.com/eurico-martins/lazytree/internal/model"
)

// Stream carries the open LogStream forward to the next page's command
// when Done is false; it's nil once the stream is exhausted (LogStream.Next
// already reaps/closes it internally in that case -- see internal/git/log.go).
type MsgCommitsLoaded struct {
	Commits    []model.Commit
	Layout     model.GraphLayout
	Layouter   *graph.Layouter // lane state to continue with on MsgCommitsBatch; may be nil
	Gen        int
	RefsByHash map[string][]model.Ref // refs fetched once for this generation; reused for later pages
	Stream     *git.LogStream         // open stream to read the next page from; nil once Done
	Done       bool                   // true once the stream is fully read (or failed)
}

type MsgCommitsBatch struct {
	Commits []model.Commit
	Gen     int
	Stream  *git.LogStream
	Done    bool
}

// MsgFetchTick fires every fetchInterval while lazytree is open. It kicks
// off a background `git fetch` (if one isn't already running) and
// reschedules itself for the next interval.
type MsgFetchTick struct {
	Seq int
}

// MsgFetchResult reports that a background `git fetch` completed. Fetch
// failures are silently ignored -- they never populate m.err/MsgError and
// never surface in the UI; the next scheduled tick simply retries. Changed
// is only meaningful when Err is nil: it's true when ref state (branch/tag/
// remote hashes) differs from before the fetch, so the caller can skip a
// full commit reload when the fetch brought in nothing new. RefsByHash
// carries the refs already fetched while checking for a change, so the
// reload MsgFetchResult triggers can reuse them instead of fetching refs a
// second time; it's only set when Changed is true.
type MsgFetchResult struct {
	Err        error
	Changed    bool
	RefsByHash map[string][]model.Ref
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
