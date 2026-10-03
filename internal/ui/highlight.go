package ui

import (
	"strings"

	"github.com/alecthomas/chroma/v2"
	"github.com/alecthomas/chroma/v2/formatters"
	"github.com/alecthomas/chroma/v2/lexers"
	"github.com/alecthomas/chroma/v2/styles"
	"github.com/charmbracelet/lipgloss"
)

// maxHighlightLines caps how many lines of one file get syntax-colored.
// setDiff runs synchronously inside Update, so a huge generated file must
// not stall the UI; lines past the cap render with plain diff colors.
const maxHighlightLines = 2000

var (
	highlightFormatter = formatters.TTY256
)

// syntaxStyle picks a chroma theme matching the terminal background:
// monokai's pale colors vanish on a light background.
func syntaxStyle() *chroma.Style {
	name := "monokai"
	if !lipgloss.HasDarkBackground() {
		name = "github"
	}
	if st := styles.Get(name); st != nil {
		return st
	}
	return styles.Fallback
}

// lineHighlighter colors single lines of source for one file. Lines are
// tokenized independently (a diff shows fragments, not whole files), so
// constructs spanning several lines, like block comments, may be colored
// imperfectly -- an accepted trade-off for a diff viewer.
type lineHighlighter struct {
	lexer chroma.Lexer
}

// newLineHighlighter picks a lexer from the file name. It returns nil when
// no lexer matches, so callers fall back to plain diff colors.
func newLineHighlighter(path string) *lineHighlighter {
	lexer := lexers.Match(path)
	if lexer == nil {
		return nil
	}
	return &lineHighlighter{lexer: chroma.Coalesce(lexer)}
}

// highlight returns text with ANSI colors, or text unchanged on any error.
func (h *lineHighlighter) highlight(text string) string {
	if h == nil || text == "" {
		return text
	}
	it, err := h.lexer.Tokenise(nil, text)
	if err != nil {
		return text
	}
	var sb strings.Builder
	if err := highlightFormatter.Format(&sb, syntaxStyle(), it); err != nil {
		return text
	}
	return strings.TrimRight(sb.String(), "\n")
}
