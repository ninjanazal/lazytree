package ui

import (
	"fmt"

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
