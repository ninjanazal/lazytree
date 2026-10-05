package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
)

// The smallest terminal the normal layout is built for (TestNarrowTerminals
// checks every view from this size up). Below it View shows tooSmallView.
const (
	minWidth  = 24
	minHeight = 10
)

const tooSmallTitle = "Terminal too small"

func tooSmall(w, h int) bool { return w < minWidth || h < minHeight }

// tooSmallView replaces the whole UI on a terminal below minWidth×minHeight:
// a short centered message, every line clipped so it never wraps. While it
// shows, Update ignores every key except quit; resizing brings the normal
// view back.
func tooSmallView(w, h int) string {
	msg := []string{
		styleTitle.Render(tooSmallTitle),
		styleHelp.Render(fmt.Sprintf("%d×%d, need %d×%d", w, h, minWidth, minHeight)),
		styleHelp.Render(keys.Quit.Help().Key + " to quit"),
	}
	if len(msg) > h {
		msg = msg[:h]
	}
	lines := make([]string, 0, h)
	for range (h - len(msg)) / 2 {
		lines = append(lines, "")
	}
	for _, l := range msg {
		l = ansi.Truncate(l, w, "…")
		lines = append(lines, lipgloss.PlaceHorizontal(w, lipgloss.Center, l))
	}
	return strings.Join(lines, "\n")
}
