package ui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
)

// overlayCenter renders popup centered on top of base, occluding the
// portion of base it covers. Both strings are rendered against a canvas of
// width x height; popup is expected to already be sized to fit within it.
func overlayCenter(base, popup string, width, height int) string {
	baseLines := padLines(strings.Split(base, "\n"), width, height)
	popupLines := strings.Split(popup, "\n")
	// A popup larger than the terminal is cropped, not allowed to overflow.
	if len(popupLines) > height {
		popupLines = popupLines[:height]
	}
	for i, l := range popupLines {
		popupLines[i] = ansi.Truncate(l, width, "")
	}

	popupW := 0
	for _, l := range popupLines {
		if w := lipgloss.Width(l); w > popupW {
			popupW = w
		}
	}
	popupH := len(popupLines)

	top := max(0, (height-popupH)/2)
	left := max(0, (width-popupW)/2)

	for i, pl := range popupLines {
		row := top + i
		if row < 0 || row >= len(baseLines) {
			continue
		}
		plW := lipgloss.Width(pl)
		leftPart := ansi.Cut(baseLines[row], 0, left)
		rightPart := ansi.Cut(baseLines[row], left+plW, width)
		baseLines[row] = leftPart + pl + rightPart
	}

	return strings.Join(baseLines, "\n")
}

// padLines pads/truncates a slice of lines to exactly height rows of width
// columns (ANSI-aware), so overlay math has a stable canvas to index into.
func padLines(lines []string, width, height int) []string {
	out := make([]string, height)
	for i := range height {
		if i < len(lines) {
			w := lipgloss.Width(lines[i])
			if w < width {
				out[i] = lines[i] + strings.Repeat(" ", width-w)
			} else {
				out[i] = ansi.Cut(lines[i], 0, width)
			}
		} else {
			out[i] = strings.Repeat(" ", width)
		}
	}
	return out
}
