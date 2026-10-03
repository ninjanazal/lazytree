package ui

import (
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
