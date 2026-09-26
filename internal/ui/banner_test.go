package ui

import (
	"errors"
	"strings"
	"testing"
)

func TestStatusBanner_EmptyByDefault(t *testing.T) {
	var b statusBanner
	if got := b.View(80); got != "" {
		t.Errorf("View() = %q, want empty", got)
	}
}

func TestStatusBanner_SetAndDismiss(t *testing.T) {
	var b statusBanner
	b.set(errors.New("boom"))
	if !strings.Contains(b.View(80), "boom") {
		t.Fatalf("expected banner to show the set error")
	}
	b.dismiss()
	if got := b.View(80); got != "" {
		t.Errorf("after dismiss, View() = %q, want empty", got)
	}
}

// TestStatusBanner_FlattensMultilineErrors is a regression test: a git
// error's stderr often spans several lines ("fatal: ..." followed by
// "hint: ..." lines). Left as-is, that would make this "one-line" banner
// several rows tall and throw off the fixed-height layout the caller
// assumes (see bannerHeight in app.go).
func TestStatusBanner_FlattensMultilineErrors(t *testing.T) {
	var b statusBanner
	b.set(errors.New("fatal: bad object deadbeef\nhint: check the hash\nhint: or try again"))
	got := b.View(200)
	if strings.Count(got, "\n") != 0 {
		t.Errorf("View produced %d newlines, want 0:\n%q", strings.Count(got, "\n"), got)
	}
	if !strings.Contains(got, "fatal: bad object deadbeef") {
		t.Errorf("banner missing the error text: %q", got)
	}
}

func TestStatusBanner_TruncatesToWidth(t *testing.T) {
	var b statusBanner
	b.set(errors.New(strings.Repeat("x", 500)))
	got := b.View(40)
	// The rendered string carries ANSI styling, but the plain text run
	// (before styling) must fit within width runes; check the unstyled
	// message via truncate directly instead of parsing ANSI codes here.
	plain := truncate("⚠ "+strings.Repeat("x", 500), 40)
	if len([]rune(plain)) > 40 {
		t.Fatalf("test setup: truncate itself exceeded width")
	}
	if !strings.Contains(got, plain) {
		t.Errorf("banner does not contain the truncated message %q: %q", plain, got)
	}
}

func TestStatusBanner_Height(t *testing.T) {
	var b statusBanner
	if b.Height() != bannerHeight {
		t.Errorf("Height() = %d, want %d", b.Height(), bannerHeight)
	}
}
