package ui

import "github.com/charmbracelet/lipgloss"

var (
	colorSelected     = lipgloss.Color("39")
	colorHash         = lipgloss.Color("215")
	colorAuthor       = lipgloss.Color("115")
	colorDate         = lipgloss.Color("242")
	colorBorder       = lipgloss.Color("237")
	colorBorderActive = lipgloss.Color("135")
	colorTitle        = lipgloss.Color("135")
	colorError        = lipgloss.Color("203")
	colorHelp         = lipgloss.Color("242")

	colorRefHead   = lipgloss.Color("215")
	colorRefLocal  = lipgloss.Color("114")
	colorRefRemote = lipgloss.Color("111")
	colorRefTag    = lipgloss.Color("183")

	colorDiffAdded      = lipgloss.Color("114")
	colorDiffRemoved    = lipgloss.Color("203")
	colorDiffHeader     = lipgloss.Color("111")
	colorDiffHunkHeader = lipgloss.Color("242")

	styleSelected = lipgloss.NewStyle().
			Background(colorSelected).
			Foreground(lipgloss.Color("231"))

	styleHash = lipgloss.NewStyle().
			Foreground(colorHash).
			Bold(true)

	styleAuthor = lipgloss.NewStyle().
			Foreground(colorAuthor)

	styleDate = lipgloss.NewStyle().
			Foreground(colorDate)

	styleTitle = lipgloss.NewStyle().
			Foreground(colorTitle).
			Bold(true)

	styleHelp = lipgloss.NewStyle().
			Foreground(colorHelp)

	styleError = lipgloss.NewStyle().
			Foreground(colorError).
			Bold(true)

	stylePane = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(colorBorder).
			Padding(0, 1)

	styleRefHead = lipgloss.NewStyle().
			Foreground(lipgloss.Color("0")).
			Background(colorRefHead).
			Bold(true).
			Padding(0, 1)

	styleRefLocal = lipgloss.NewStyle().
			Foreground(lipgloss.Color("0")).
			Background(colorRefLocal).
			Padding(0, 1)

	styleRefRemote = lipgloss.NewStyle().
			Foreground(lipgloss.Color("0")).
			Background(colorRefRemote).
			Padding(0, 1)

	styleRefTag = lipgloss.NewStyle().
			Foreground(lipgloss.Color("0")).
			Background(colorRefTag).
			Padding(0, 1)

	styleDiffAdded = lipgloss.NewStyle().
			Foreground(colorDiffAdded)

	styleDiffRemoved = lipgloss.NewStyle().
				Foreground(colorDiffRemoved)

	styleDiffHeader = lipgloss.NewStyle().
			Foreground(colorDiffHeader).
			Bold(true)

	styleDiffHunkHeader = lipgloss.NewStyle().
				Foreground(colorDiffHunkHeader)

	styleHeader = lipgloss.NewStyle().
			Foreground(colorHelp).
			Bold(true)

	styleToolbar = lipgloss.NewStyle()

	styleToolbarPill = lipgloss.NewStyle().
				Foreground(lipgloss.Color("231")).
				Background(colorBorderActive).
				Padding(0, 1)

	styleFooter = lipgloss.NewStyle()

	stylePopup = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(colorBorderActive).
			Padding(0, 1)
)
