package ui

import (
	"testing"
)

// These golden lists are part of lazytree's 1.x compatibility promise (see
// README "Compatibility"): names may be added, never renamed or removed
// without a 2.0. If a test here fails, you are changing the public surface.

func TestActionNamesAreStable(t *testing.T) {
	want := []string{"back", "bottom", "child", "copy", "down", "edit", "enter", "head", "help",
		"next_match", "page_down", "page_up", "parent", "prev_match", "quit", "search",
		"toggle_refs", "toggle_remotes", "toggle_tags", "top", "up", "zen"}
	assertStable(t, "action", want, ActionNames())
}

func TestPaletteNamesAreStable(t *testing.T) {
	want := []string{"accent", "added", "author", "border", "branch", "date", "diff_file",
		"diff_hunk", "error", "hash", "head", "help", "remote", "removed", "selected",
		"stash", "tag", "title"}
	assertStable(t, "palette color", want, PaletteNames())
}

// assertStable fails if any promised name is gone; new names only need the
// golden list extended.
func assertStable(t *testing.T, kind string, want, got []string) {
	t.Helper()
	have := map[string]bool{}
	for _, n := range got {
		have[n] = true
	}
	for _, n := range want {
		if !have[n] {
			t.Errorf("%s %q was removed or renamed (breaking change)", kind, n)
		}
	}
	if len(got) != len(want) {
		t.Errorf("%s names changed: got %v, golden %v; if you added one, extend the golden list", kind, got, want)
	}
}
