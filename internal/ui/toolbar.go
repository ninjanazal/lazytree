package ui

import (
	"fmt"

	"github.com/charmbracelet/lipgloss"
)

// renderToolbar renders the top bar: app/repo name on the left, the current
// branch and ref-filter state on the right.
func renderToolbar(width int, repoName, branch string, showAll bool) string {
	right := ""
	if branch != "" {
		right = styleToolbarPill.Render(" "+branch+" ") + "  "
	}
	if showAll {
		right += styleHelp.Render("⊙ all refs")
	} else {
		right += styleHelp.Render("⊸ current branch")
	}

	appName := styleTitle.Render(" lazytree")
	budget := max(width-lipgloss.Width(appName)-lipgloss.Width(right)-4, 0)

	left := appName
	if repoName != "" {
		left += styleHelp.Render("  " + truncate(repoName, budget))
	}

	pad := max(width-lipgloss.Width(left)-lipgloss.Width(right), 1)
	return styleToolbar.Render(fmt.Sprintf("%s%s%s", left, repeatSpaces(pad), right))
}

func repeatSpaces(n int) string {
	if n <= 0 {
		return ""
	}
	b := make([]byte, n)
	for i := range b {
		b[i] = ' '
	}
	return string(b)
}
