package ui

import (
	"fmt"
	"strings"
)

// statusBanner is the one-line, non-fatal banner shown above the fetch bar
// for repo-level failures (a failed commits load). A failed diff load is
// shown inside diffPane instead, since it's local to the commit being
// viewed rather than the whole app -- see diffPane.setError.
//
// It owns its render, height and dismiss rules in one place, rather than
// leaving app.go to juggle an *error field plus scattered clearing logic
// across several Update cases.
type statusBanner struct {
	err error
}

// set raises the banner. Called only from the MsgError case in Update,
// which already checks the error's Gen against the current reload before
// calling this -- set itself doesn't need to know about generations.
func (b *statusBanner) set(err error) {
	b.err = err
}

// dismiss clears the banner unconditionally (Esc/Back, or a fresh commits
// load that supersedes whatever failed before).
func (b *statusBanner) dismiss() {
	b.err = nil
}

// Height is the fixed number of rows the banner reserves in the layout,
// whether or not it currently has anything to show.
func (b *statusBanner) Height() int {
	return bannerHeight
}

// View renders the banner, or an empty line when there is nothing to show
// (callers reserve a fixed row for it either way, via Height).
//
// The message is flattened to one line before truncating: a git error can
// carry a multi-line stderr (a "fatal: ..." line plus "hint: ..." lines),
// and left unflattened that would make this "one-line" banner several rows
// tall, throwing off the fixed-height layout callers assume.
func (b *statusBanner) View(width int) string {
	if b.err == nil {
		return ""
	}
	oneLine := strings.Join(strings.Fields(b.err.Error()), " ")
	msg := truncate(fmt.Sprintf("⚠ %s", oneLine), width)
	return styleError.Render(msg)
}
