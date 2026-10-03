package git

import (
	"strings"
	"testing"

	"github.com/eurico-martins/lazytree/internal/model"
)

func TestParseDiff_Empty(t *testing.T) {
	if got := ParseDiff(""); got != nil {
		t.Errorf("ParseDiff(\"\") = %#v, want nil", got)
	}
}

// realDiff joins lines the way `git show --patch` actually emits them: with
// a trailing newline after the last line (see the trailing-newline test
// below for why that matters).
func realDiff(lines ...string) string {
	return strings.Join(lines, "\n") + "\n"
}

func TestParseDiff_SingleFileModified(t *testing.T) {
	raw := realDiff(
		"diff --git a/foo.go b/foo.go",
		"index 1234567..89abcdef 100644",
		"--- a/foo.go",
		"+++ b/foo.go",
		"@@ -1,3 +1,3 @@",
		" package main",
		"-func old() {}",
		"+func new() {}",
		" // trailing",
	)

	files := ParseDiff(raw)
	if len(files) != 1 {
		t.Fatalf("got %d files, want 1", len(files))
	}
	f := files[0]
	if f.OldPath != "foo.go" || f.NewPath != "foo.go" {
		t.Errorf("paths = %q/%q, want foo.go/foo.go", f.OldPath, f.NewPath)
	}
	if f.Status != "M" {
		t.Errorf("Status = %q, want M", f.Status)
	}
	if len(f.Hunks) != 1 {
		t.Fatalf("got %d hunks, want 1", len(f.Hunks))
	}
	hunk := f.Hunks[0]
	if hunk.Header != "@@ -1,3 +1,3 @@" {
		t.Errorf("hunk header = %q", hunk.Header)
	}
	wantKinds := []model.DiffLineKind{model.DiffContext, model.DiffRemoved, model.DiffAdded, model.DiffContext}
	if len(hunk.Lines) != len(wantKinds) {
		t.Fatalf("got %d lines, want %d: %#v", len(hunk.Lines), len(wantKinds), hunk.Lines)
	}
	for i, k := range wantKinds {
		if hunk.Lines[i].Kind != k {
			t.Errorf("line %d kind = %v, want %v (text %q)", i, hunk.Lines[i].Kind, k, hunk.Lines[i].Text)
		}
	}
}

// TestParseDiff_TrailingNewlineDoesNotAddBlankLine is a regression test:
// git's output always ends with a newline after the last content line,
// which used to leave a bogus empty DiffLine at the end of the last hunk.
func TestParseDiff_TrailingNewlineDoesNotAddBlankLine(t *testing.T) {
	raw := realDiff(
		"diff --git a/foo.go b/foo.go",
		"--- a/foo.go",
		"+++ b/foo.go",
		"@@ -1 +1 @@",
		"-old",
		"+new",
	)
	files := ParseDiff(raw)
	if len(files) != 1 {
		t.Fatalf("got %d files, want 1", len(files))
	}
	lines := files[0].Hunks[0].Lines
	if len(lines) != 2 {
		t.Fatalf("got %d lines, want 2 (no trailing blank line): %#v", len(lines), lines)
	}
}

func TestParseDiff_MultipleFiles(t *testing.T) {
	raw := realDiff(
		"diff --git a/a.go b/a.go",
		"--- a/a.go",
		"+++ b/a.go",
		"@@ -1 +1 @@",
		"-old",
		"+new",
		"diff --git a/b.go b/b.go",
		"--- a/b.go",
		"+++ b/b.go",
		"@@ -1 +1 @@",
		"-old2",
		"+new2",
	)

	files := ParseDiff(raw)
	if len(files) != 2 {
		t.Fatalf("got %d files, want 2", len(files))
	}
	if files[0].NewPath != "a.go" || files[1].NewPath != "b.go" {
		t.Errorf("got paths %q, %q", files[0].NewPath, files[1].NewPath)
	}
	for i, f := range files {
		if len(f.Hunks) != 1 || len(f.Hunks[0].Lines) != 2 {
			t.Errorf("file %d: got %d hunks, %#v", i, len(f.Hunks), f.Hunks)
		}
	}
}

// A "diff --git a/X b/X" header always carries both paths, even for added
// or deleted files (git shows the same name on both sides, not /dev/null,
// in the header itself -- /dev/null only appears further down in the
// "--- "/"+++ " lines).
func TestParseDiff_NewFile(t *testing.T) {
	raw := realDiff(
		"diff --git a/new.go b/new.go",
		"new file mode 100644",
		"index 0000000..1234567",
		"--- /dev/null",
		"+++ b/new.go",
		"@@ -0,0 +1,2 @@",
		"+package main",
		"+func main() {}",
	)

	files := ParseDiff(raw)
	if len(files) != 1 {
		t.Fatalf("got %d files, want 1", len(files))
	}
	f := files[0]
	if f.Status != "A" {
		t.Errorf("Status = %q, want A", f.Status)
	}
	if f.OldPath != "new.go" || f.NewPath != "new.go" {
		t.Errorf("paths = %q/%q, want new.go/new.go (from the diff --git header)", f.OldPath, f.NewPath)
	}
	if len(f.Hunks) != 1 || len(f.Hunks[0].Lines) != 2 {
		t.Fatalf("unexpected hunks: %#v", f.Hunks)
	}
	for _, l := range f.Hunks[0].Lines {
		if l.Kind != model.DiffAdded {
			t.Errorf("line %q kind = %v, want DiffAdded", l.Text, l.Kind)
		}
	}
}

func TestParseDiff_DeletedFile(t *testing.T) {
	raw := realDiff(
		"diff --git a/gone.go b/gone.go",
		"deleted file mode 100644",
		"index 1234567..0000000",
		"--- a/gone.go",
		"+++ /dev/null",
		"@@ -1,1 +0,0 @@",
		"-package main",
	)

	files := ParseDiff(raw)
	if len(files) != 1 {
		t.Fatalf("got %d files, want 1", len(files))
	}
	f := files[0]
	if f.Status != "D" {
		t.Errorf("Status = %q, want D", f.Status)
	}
	if f.OldPath != "gone.go" || f.NewPath != "gone.go" {
		t.Errorf("paths = %q/%q, want gone.go/gone.go (from the diff --git header)", f.OldPath, f.NewPath)
	}
}

// A pure rename (no content change) has no "--- "/"+++ " lines or hunks at
// all -- just the header plus "similarity index"/"rename from"/"rename to".
func TestParseDiff_PureRename(t *testing.T) {
	raw := realDiff(
		"diff --git a/old.go b/new.go",
		"similarity index 100%",
		"rename from old.go",
		"rename to new.go",
	)

	files := ParseDiff(raw)
	if len(files) != 1 {
		t.Fatalf("got %d files, want 1", len(files))
	}
	f := files[0]
	if f.Status != "R" {
		t.Errorf("Status = %q, want R", f.Status)
	}
	if f.OldPath != "old.go" || f.NewPath != "new.go" {
		t.Errorf("paths = %q/%q, want old.go/new.go", f.OldPath, f.NewPath)
	}
	if len(f.Hunks) != 0 {
		t.Errorf("got %d hunks, want 0 for a content-free rename", len(f.Hunks))
	}
}

// TestParseDiff_PathWithSpace_Modified is a regression test: git doesn't
// quote a space in the "diff --git a/X b/Y" header, so splitting that line
// on spaces (parseDiffHeader's fallback) can't tell where X ends and Y
// begins. The "--- "/"+++ " lines are the reliable source here -- git
// appends a trailing tab after a path with a space, exactly so a reader can
// tell where the path ends.
func TestParseDiff_PathWithSpace_Modified(t *testing.T) {
	raw := realDiff(
		"diff --git a/my file.go b/my file.go",
		"index 1234567..89abcdef 100644",
		"--- a/my file.go\t",
		"+++ b/my file.go\t",
		"@@ -1 +1 @@",
		"-old",
		"+new",
	)
	files := ParseDiff(raw)
	if len(files) != 1 {
		t.Fatalf("got %d files, want 1", len(files))
	}
	f := files[0]
	if f.OldPath != "my file.go" || f.NewPath != "my file.go" {
		t.Errorf("paths = %q/%q, want %q/%q", f.OldPath, f.NewPath, "my file.go", "my file.go")
	}
}

// TestParseDiff_PathWithSpace_NewFile is a regression test: for an added
// file, the "--- " line reads "/dev/null" (no real old path there), so
// OldPath must come from finalizeDiffFile mirroring NewPath rather than
// from the header's ambiguous, and here wrong, guess.
func TestParseDiff_PathWithSpace_NewFile(t *testing.T) {
	raw := realDiff(
		"diff --git a/my file.go b/my file.go",
		"new file mode 100644",
		"index 0000000..1234567",
		"--- /dev/null",
		"+++ b/my file.go\t",
		"@@ -0,0 +1 @@",
		"+content",
	)
	files := ParseDiff(raw)
	if len(files) != 1 {
		t.Fatalf("got %d files, want 1", len(files))
	}
	f := files[0]
	if f.Status != "A" {
		t.Errorf("Status = %q, want A", f.Status)
	}
	if f.OldPath != "my file.go" || f.NewPath != "my file.go" {
		t.Errorf("paths = %q/%q, want %q/%q", f.OldPath, f.NewPath, "my file.go", "my file.go")
	}
}

// TestParseDiff_PathWithSpace_Rename is a regression test: "rename from"/
// "rename to" lines each carry exactly one path per line, so they're
// unambiguous even with a space, unlike the shared header line.
func TestParseDiff_PathWithSpace_Rename(t *testing.T) {
	raw := realDiff(
		"diff --git a/old name.go b/new name.go",
		"similarity index 100%",
		"rename from old name.go",
		"rename to new name.go",
	)
	files := ParseDiff(raw)
	if len(files) != 1 {
		t.Fatalf("got %d files, want 1", len(files))
	}
	f := files[0]
	if f.Status != "R" {
		t.Errorf("Status = %q, want R", f.Status)
	}
	if f.OldPath != "old name.go" || f.NewPath != "new name.go" {
		t.Errorf("paths = %q/%q, want %q/%q", f.OldPath, f.NewPath, "old name.go", "new name.go")
	}
}

func TestParseDiff_PreambleIgnoredBeforeFirstFile(t *testing.T) {
	raw := realDiff(
		"commit abc123",
		"Author: x",
		"",
		"diff --git a/f b/f",
		"--- a/f",
		"+++ b/f",
		"@@ -1 +1 @@",
		"-x",
		"+y",
	)

	files := ParseDiff(raw)
	if len(files) != 1 {
		t.Fatalf("got %d files, want 1", len(files))
	}
}

// TestParseDiff_HunkLinesLookingLikeFileHeaders is a regression test: a
// removed line "-- a/evil" appears in the diff as "--- a/evil" (and an added
// "++ b/evil" as "+++ b/evil"). Inside a hunk these are content, not file
// headers: they must not overwrite the file's paths or vanish from the hunk.
func TestParseDiff_HunkLinesLookingLikeFileHeaders(t *testing.T) {
	raw := realDiff(
		"diff --git a/notes.md b/notes.md",
		"--- a/notes.md",
		"+++ b/notes.md",
		"@@ -1 +1 @@",
		"--- a/evil",
		"+++ b/evil",
	)
	f := ParseDiff(raw)[0]
	if f.OldPath != "notes.md" || f.NewPath != "notes.md" {
		t.Errorf("paths = %q/%q, want notes.md/notes.md", f.OldPath, f.NewPath)
	}
	lines := f.Hunks[0].Lines
	if len(lines) != 2 || lines[0].Kind != model.DiffRemoved || lines[1].Kind != model.DiffAdded {
		t.Errorf("hunk lines = %#v, want one removed and one added line", lines)
	}
}

func TestParseDiff_LineNumbers(t *testing.T) {
	raw := "diff --git a/f.txt b/f.txt\n" +
		"--- a/f.txt\n+++ b/f.txt\n" +
		"@@ -10,4 +20,4 @@ func x()\n" +
		" ctx\n-gone\n+new\n" +
		"\\ No newline at end of file\n" +
		" tail\n"
	files := ParseDiff(raw)
	if len(files) != 1 || len(files[0].Hunks) != 1 {
		t.Fatalf("unexpected parse: %+v", files)
	}
	want := []struct{ old, new int }{{10, 20}, {11, 0}, {0, 21}, {0, 0}, {12, 22}}
	lines := files[0].Hunks[0].Lines
	if len(lines) != len(want) {
		t.Fatalf("got %d lines, want %d", len(lines), len(want))
	}
	for i, w := range want {
		if lines[i].OldN != w.old || lines[i].NewN != w.new {
			t.Errorf("line %d (%q): got old=%d new=%d, want old=%d new=%d",
				i, lines[i].Text, lines[i].OldN, lines[i].NewN, w.old, w.new)
		}
	}
}

func TestParseDiff_BinaryFile(t *testing.T) {
	raw := "diff --git a/img.png b/img.png\n" +
		"index 111..222 100644\n" +
		"Binary files a/img.png and b/img.png differ\n" +
		"diff --git a/f.txt b/f.txt\n" +
		"--- a/f.txt\n+++ b/f.txt\n@@ -1 +1 @@\n-a\n+b\n"
	files := ParseDiff(raw)
	if len(files) != 2 {
		t.Fatalf("got %d files, want 2", len(files))
	}
	if !files[0].Binary || len(files[0].Hunks) != 0 {
		t.Errorf("first file should be binary with no hunks: %+v", files[0])
	}
	if files[1].Binary || len(files[1].Hunks) != 1 {
		t.Errorf("second file should be a normal text diff: %+v", files[1])
	}
}
