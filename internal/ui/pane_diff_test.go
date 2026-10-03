package ui

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/eurico-martins/lazytree/internal/model"
)

func TestDiffPaneInspectorHeader(t *testing.T) {
	c := &model.Commit{
		Hash:        "0123456789abcdef0123456789abcdef01234567",
		Parents:     []string{"aaaaaaaaaaaaaaaa", "bbbbbbbbbbbbbbbb"},
		Author:      "Ann",
		AuthorEmail: "ann@example.com",
		Committer:   "Bob",
		Timestamp:   time.Date(2026, 10, 3, 12, 0, 0, 0, time.UTC),
		Subject:     "fix: thing",
		Body:        "longer explanation",
		Refs:        []model.Ref{{Name: "main", Kind: model.RefLocalBranch}},
	}
	files := []model.DiffFile{{
		OldPath: "a.go", NewPath: "a.go", Status: "M",
		Hunks: []model.DiffHunk{{Header: "@@ -1 +1 @@", Lines: []model.DiffLine{
			{Kind: model.DiffRemoved, Text: "-x", OldN: 1},
			{Kind: model.DiffAdded, Text: "+y", NewN: 1},
		}}},
	}}
	p := newDiffPane()
	p.setSize(120, 40)
	p.setDiff(c.Hash, c, files)
	out := p.renderContent()

	for _, want := range []string{
		c.Hash, "ann@example.com", "committer", "Bob", "aaaaaaaa bbbbbbbb",
		"main", "fix: thing", "longer explanation", "1 files changed", "+1", "-1",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("inspector output missing %q", want)
		}
	}
}

func TestDiffPaneEmptyDiffKeepsHeader(t *testing.T) {
	c := &model.Commit{Hash: "deadbeefdeadbeef", Author: "Ann", Committer: "Ann", Subject: "merge"}
	p := newDiffPane()
	p.setSize(120, 40)
	p.setDiff(c.Hash, c, nil)
	out := p.renderContent()
	if !strings.Contains(out, "merge") || !strings.Contains(out, "No diff available") {
		t.Errorf("empty diff should show header and notice, got %q", out)
	}
}

// TestRenderDiffContent_HugeLinesAreFast guards the startup freeze: a commit
// whose files are single 180k-character lines (Excalidraw JSON, minified
// assets) must render quickly, with long lines clipped and not highlighted.
func TestRenderDiffContent_HugeLinesAreFast(t *testing.T) {
	huge := "+" + strings.Repeat(`{"type":"text","x":1,"y":2},`, 6500) // ~180k chars
	var files []model.DiffFile
	for i := range 40 {
		files = append(files, model.DiffFile{NewPath: fmt.Sprintf("page%d.md", i), Status: "M",
			Hunks: []model.DiffHunk{{Header: "@@ -1 +1 @@", Lines: []model.DiffLine{
				{Kind: model.DiffAdded, Text: huge, NewN: 1},
			}}}})
	}
	start := time.Now()
	out := renderDiffContent(&model.Commit{Hash: "abc"}, files, 100)
	if d := time.Since(start); d > 2*time.Second {
		t.Errorf("rendering took %v; huge lines must be clipped, not highlighted", d)
	}
	if len(out) > 40*(maxDiffLineLen+500) {
		t.Errorf("output is %d bytes; long lines should be clipped to ~%d", len(out), maxDiffLineLen)
	}
}

func TestRenderDiffContent_TruncatesHugeDiffs(t *testing.T) {
	lines := make([]model.DiffLine, maxDiffRenderLines+500)
	for i := range lines {
		lines[i] = model.DiffLine{Kind: model.DiffAdded, Text: "+x", NewN: i + 1}
	}
	out := renderDiffContent(&model.Commit{Hash: "abc"},
		[]model.DiffFile{{NewPath: "big.txt", Status: "A", Hunks: []model.DiffHunk{{Header: "@@", Lines: lines}}}}, 100)
	if !strings.Contains(out, "diff truncated") {
		t.Error("a diff over the line cap should end with a truncation notice")
	}
}
