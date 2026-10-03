package ui

import (
	"fmt"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
)

// renderToolbar renders the top bar: app/repo name on the left, the current
// branch and ref-filter state on the right.
func renderToolbar(width int, repoName, branch string, showAll bool, hidden string) string {
	right := ""
	if hidden != "" {
		right += styleHelp.Render(hidden) + "  "
	}
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
	return ansi.Truncate(styleToolbar.Render(fmt.Sprintf("%s%s%s", left, repeatSpaces(pad), right)), width, "")
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
