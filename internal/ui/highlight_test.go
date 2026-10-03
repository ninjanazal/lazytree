package ui

import (
	"strings"
	"testing"
)

func TestLineHighlighter(t *testing.T) {
	if newLineHighlighter("README.unknownext") != nil {
		t.Error("unknown extension should have no highlighter")
	}
	var nilH *lineHighlighter
	if got := nilH.highlight("x := 1"); got != "x := 1" {
		t.Errorf("nil highlighter must pass text through, got %q", got)
	}
	h := newLineHighlighter("main.go")
	if h == nil {
		t.Fatal("expected a Go lexer")
	}
	got := h.highlight("func main() {}")
	if !strings.Contains(got, "\x1b[") {
		t.Errorf("expected ANSI colors in %q", got)
	}
	if strings.Contains(got, "\n") {
		t.Errorf("output must be a single line, got %q", got)
	}
}
