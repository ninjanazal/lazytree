package graph

import "github.com/charmbracelet/lipgloss"

// LanePalette provides distinct colors for graph lanes. Each has a Light
// variant (readable on a light terminal background) and a Dark one;
// lipgloss picks between them from the detected terminal background.
var LanePalette = []lipgloss.AdaptiveColor{
	{Light: "27", Dark: "33"},   // blue
	{Light: "166", Dark: "208"}, // orange
	{Light: "28", Dark: "40"},   // green
	{Light: "160", Dark: "196"}, // red
	{Light: "91", Dark: "141"},  // purple
	{Light: "30", Dark: "51"},   // cyan / teal
	{Light: "136", Dark: "226"}, // yellow / gold
}

// SetLanePalette replaces the lane colors. It must be called before any
// rendering (rendered rows are cached with the colors in effect at the
// time). An empty slice is ignored.
func SetLanePalette(colors []lipgloss.AdaptiveColor) {
	if len(colors) > 0 {
		LanePalette = colors
	}
}
