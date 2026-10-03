package ui

import (
	"path/filepath"
	"strings"
	"sync"

	"github.com/alecthomas/chroma/v2"
	"github.com/alecthomas/chroma/v2/formatters"
	"github.com/alecthomas/chroma/v2/lexers"
	"github.com/alecthomas/chroma/v2/styles"
	"github.com/charmbracelet/lipgloss"
)

// maxHighlightLines caps how many lines of one file get syntax-colored.
// Rendering runs in a background command, but a huge generated file would
// still delay the diff appearing; lines past the cap render with plain diff colors.
const maxHighlightLines = 2000

// maxHighlightLineLen skips syntax colouring for very long lines (minified
// JS, generated JSON): tokenising a 100k-character line takes seconds.
const maxHighlightLineLen = 400

// maxDiffLineLen clips what is kept of a single diff line. The viewport only
// shows the first screenful of columns anyway (no horizontal scroll), so
// rendering the rest of a huge line is pure waste.
const maxDiffLineLen = 1000

// maxDiffRenderLines caps the lines rendered for one commit; the rest is
// replaced by a notice pointing at `o` (open the full patch in $EDITOR).
const maxDiffRenderLines = 20000

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
	ext := filepath.Ext(path)
	if ext == "" {
		ext = filepath.Base(path)
	}
	lexerMu.Lock()
	defer lexerMu.Unlock()
	if h, ok := lexerCache[ext]; ok {
		return h
	}
	var h *lineHighlighter
	if lexer := lexers.Match(path); lexer != nil {
		h = &lineHighlighter{lexer: chroma.Coalesce(lexer)}
	}
	lexerCache[ext] = h // nil is cached too: "no lexer for this extension"
	return h
}

// lexerCache memoises lexer lookup per file extension: lexers.Match scans
// every registered lexer and is slow when called for each file of a diff.
// Diffs are rendered off the UI goroutine, hence the mutex.
var (
	lexerMu    sync.Mutex
	lexerCache = map[string]*lineHighlighter{}
)

// highlightLines colours several lines in one pass (one tokenise + format
// call instead of one per line, which dominated the cost of big diffs). It
// always returns exactly len(lines) strings; on any error the input is
// returned unchanged.
func (h *lineHighlighter) highlightLines(lines []string) []string {
	if h == nil || len(lines) == 0 {
		return lines
	}
	out := strings.Split(h.highlight(strings.Join(lines, "\n")), "\n")
	if len(out) != len(lines) {
		return lines
	}
	return out
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
