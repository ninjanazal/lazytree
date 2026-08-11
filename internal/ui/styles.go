package ui

import "github.com/charmbracelet/lipgloss"

var (
	colorBase     = lipgloss.Color("240")
	colorSelected = lipgloss.Color("69")
	colorHash     = lipgloss.Color("214")
	colorAuthor   = lipgloss.Color("35")
	colorDate     = lipgloss.Color("240")
	colorBorder   = lipgloss.Color("238")
	colorTitle    = lipgloss.Color("69")
	colorError    = lipgloss.Color("196")
	colorHelp     = lipgloss.Color("238")

	colorRefHead   = lipgloss.Color("214")
	colorRefLocal  = lipgloss.Color("40")
	colorRefRemote = lipgloss.Color("33")
	colorRefTag    = lipgloss.Color("141")

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

	styleBorder = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(colorBorder)

	stylePane = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(colorBorder).
			Padding(0, 1)

	styleRefHead = lipgloss.NewStyle().
			Foreground(colorRefHead).
			Bold(true)

	styleRefLocal = lipgloss.NewStyle().
			Foreground(colorRefLocal)

	styleRefRemote = lipgloss.NewStyle().
			Foreground(colorRefRemote)

	styleRefTag = lipgloss.NewStyle().
			Foreground(colorRefTag)

	styleDiffAdded = lipgloss.NewStyle().
			Foreground(lipgloss.Color("40"))

	styleDiffRemoved = lipgloss.NewStyle().
			Foreground(lipgloss.Color("196"))

	styleDiffHeader = lipgloss.NewStyle().
			Foreground(lipgloss.Color("33")).
			Bold(true)

	styleDiffHunkHeader = lipgloss.NewStyle().
				Foreground(lipgloss.Color("240"))
)
