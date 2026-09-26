package ui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
)

// renderFooter renders the bottom bar: keybind hints on the left, loading
// state and commit position on the right.
func renderFooter(width int, loading bool, spinnerView string, loadedCount int, showAll bool, cursor, total int) string {
	left := keys.helpView()

	var right string
	if loading {
		right = spinnerView + " loading… "
	}
	if total > 0 {
		right += fmt.Sprintf("%d/%d commits", cursor+1, total)
	} else if loadedCount > 0 {
		right += fmt.Sprintf("%d commits", loadedCount)
	}
	if showAll {
		right += " · all refs"
	}
	right = styleHelp.Render(right)

	pad := max(width-lipgloss.Width(left)-lipgloss.Width(right), 1)
	return styleFooter.Render(fmt.Sprintf("%s%s%s", left, repeatSpaces(pad), right))
}

// renderFetchBar renders a thin full-width bar that fills left-to-right as
// the next background fetch approaches, so the countdown is visible at a
// glance without needing to read text. It's full while a fetch is running.
func renderFetchBar(width int, fetchInFlight bool, nextFetchAt time.Time) string {
	width = max(width, 1)

	progress := 1.0
	if !fetchInFlight && !nextFetchAt.IsZero() {
		remaining := time.Until(nextFetchAt)
		progress = 1 - float64(remaining)/float64(fetchInterval)
	}
	progress = min(max(progress, 0), 1)

	filled := min(int(progress*float64(width)), width)

	var b strings.Builder
	if filled > 0 {
		b.WriteString(styleFetchBarFill.Render(strings.Repeat("─", filled)))
	}
	if filled < width {
		b.WriteString(styleFetchBarEmpty.Render(strings.Repeat("─", width-filled)))
	}
	return b.String()
}
