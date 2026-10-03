package ui

import "github.com/charmbracelet/lipgloss"

// ac builds a color with a Light variant (for light terminal backgrounds)
// and a Dark one; lipgloss chooses from the detected terminal background,
// so every text color keeps its contrast in both environments.
func ac(light, dark string) lipgloss.AdaptiveColor {
	return lipgloss.AdaptiveColor{Light: light, Dark: dark}
}

var (
	// colorSelectedBg tints the selected row; foreground colors are kept
	// (see highlightRow), so it must stay subtle in both modes.
	colorSelectedBg   = ac("189", "238")
	colorHash         = ac("166", "215")
	colorAuthor       = ac("29", "115")
	colorDate         = ac("241", "245")
	colorBorder       = ac("250", "237")
	colorBorderActive = ac("91", "135")
	colorTitle        = ac("91", "135")
	colorError        = ac("160", "203")
	colorHelp         = ac("241", "245")

	colorRefHead   = ac("173", "215")
	colorRefLocal  = ac("114", "114")
	colorRefRemote = ac("111", "111")
	colorRefTag    = ac("183", "183")

	colorDiffAdded      = ac("28", "114")
	colorDiffRemoved    = ac("160", "203")
	colorDiffHeader     = ac("25", "111")
	colorDiffHunkHeader = ac("241", "245")
)

var (
	styleHash           lipgloss.Style
	styleAuthor         lipgloss.Style
	styleDate           lipgloss.Style
	styleTitle          lipgloss.Style
	styleHelp           lipgloss.Style
	styleError          lipgloss.Style
	stylePane           lipgloss.Style
	styleRefHead        lipgloss.Style
	styleRefLocal       lipgloss.Style
	styleRefRemote      lipgloss.Style
	styleRefTag         lipgloss.Style
	styleDiffAdded      lipgloss.Style
	styleDiffRemoved    lipgloss.Style
	styleDiffHeader     lipgloss.Style
	styleDiffHunkHeader lipgloss.Style
	styleHeader         lipgloss.Style
	styleMatch          lipgloss.Style
	styleMatchMarker    lipgloss.Style
	styleToolbar        lipgloss.Style
	styleToolbarPill    lipgloss.Style
	styleFooter         lipgloss.Style
	styleFetchBarFill   lipgloss.Style
	styleFetchBarEmpty  lipgloss.Style
	stylePopup          lipgloss.Style
)

// buildStyles (re)creates every style from the color variables above. It
// runs at startup and again after a user palette is applied (ApplyColors).
func buildStyles() {
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

	styleMatch = lipgloss.NewStyle().
		Foreground(lipgloss.Color("0")).
		Background(ac("227", "178")).
		Bold(true)

	styleMatchMarker = lipgloss.NewStyle().
		Foreground(ac("166", "214")).
		Bold(true)

	styleToolbar = lipgloss.NewStyle()

	styleToolbarPill = lipgloss.NewStyle().
		Foreground(lipgloss.Color("231")).
		Background(colorBorderActive).
		Padding(0, 1)

	styleFooter = lipgloss.NewStyle()

	styleFetchBarFill = lipgloss.NewStyle().
		Foreground(colorBorderActive)

	styleFetchBarEmpty = lipgloss.NewStyle().
		Foreground(colorBorder)

	stylePopup = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(colorBorderActive).
		Padding(0, 1)
}

func init() { buildStyles() }
