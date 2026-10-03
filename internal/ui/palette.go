package ui

import (
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/eurico-martins/lazytree/internal/graph"
)

// paletteTargets maps the color names usable in the config's [colors] table
// to the color variables behind them.
func paletteTargets() map[string]*lipgloss.AdaptiveColor {
	return map[string]*lipgloss.AdaptiveColor{
		"selected":  &colorSelectedBg,
		"hash":      &colorHash,
		"author":    &colorAuthor,
		"date":      &colorDate,
		"help":      &colorHelp,
		"border":    &colorBorder,
		"accent":    &colorBorderActive,
		"title":     &colorTitle,
		"error":     &colorError,
		"head":      &colorRefHead,
		"branch":    &colorRefLocal,
		"remote":    &colorRefRemote,
		"tag":       &colorRefTag,
		"added":     &colorDiffAdded,
		"removed":   &colorDiffRemoved,
		"diff_file": &colorDiffHeader,
		"diff_hunk": &colorDiffHunkHeader,
	}
}

var hexColor = regexp.MustCompile(`^#[0-9a-fA-F]{6}$`)

// validColor accepts "#rrggbb" or an ANSI-256 index "0".."255".
func validColor(c string) bool {
	if hexColor.MatchString(c) {
		return true
	}
	n, err := strconv.Atoi(c)
	return err == nil && n >= 0 && n <= 255 && strconv.Itoa(n) == c
}

// PaletteNames lists the valid [colors] keys, for error messages.
func PaletteNames() []string {
	names := make([]string, 0, 18)
	for n := range paletteTargets() {
		names = append(names, n)
	}
	sort.Strings(names)
	return names
}

// ApplyColors overrides the built-in colors. colors maps a name (see
// PaletteNames) to "#rrggbb" or an ANSI-256 index; the color then applies
// to both light and dark terminals. lanes replaces the graph lane colors.
// Nothing is changed if any entry is invalid. Call it before starting the
// program.
func ApplyColors(colors map[string]string, lanes []string) error {
	targets := paletteTargets()
	for name, c := range colors {
		if _, ok := targets[name]; !ok {
			return fmt.Errorf("unknown color %q (valid: %s)", name, strings.Join(PaletteNames(), ", "))
		}
		if !validColor(c) {
			return fmt.Errorf("color %q: %q is not #rrggbb or a number 0-255", name, c)
		}
	}
	for i, c := range lanes {
		if !validColor(c) {
			return fmt.Errorf("lane_colors[%d]: %q is not #rrggbb or a number 0-255", i, c)
		}
	}
	for name, c := range colors {
		*targets[name] = lipgloss.AdaptiveColor{Light: c, Dark: c}
	}
	if len(lanes) > 0 {
		palette := make([]lipgloss.AdaptiveColor, len(lanes))
		for i, c := range lanes {
			palette[i] = lipgloss.AdaptiveColor{Light: c, Dark: c}
		}
		graph.SetLanePalette(palette)
	}
	buildStyles()
	return nil
}
