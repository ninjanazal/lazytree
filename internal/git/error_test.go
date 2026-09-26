package git

import (
	"errors"
	"strings"
	"testing"
)

func TestError_LeadsWithFirstStderrLine(t *testing.T) {
	e := &Error{
		Args:     []string{"show", "--patch", "deadbeef"},
		Stderr:   "fatal: bad object deadbeef\nhint: check the hash\n",
		ExitCode: 128,
		Err:      errors.New("exit status 128"),
	}
	got := e.Error()
	want := "fatal: bad object deadbeef (git show)"
	if got != want {
		t.Errorf("Error() = %q, want %q", got, want)
	}
}

// TestError_OmitsArgsBeyondTheSubcommand is a regression test: some callers
// (FetchLog) pass a --format string containing raw control bytes as a field
// separator. Only Args[0] must appear in Error()'s text, or those bytes
// (and other argument noise) would reach a terminal or UI banner.
func TestError_OmitsArgsBeyondTheSubcommand(t *testing.T) {
	e := &Error{
		Args:   []string{"log", "--format=%x00%H\x01%s%x00"},
		Stderr: "fatal: bad revision 'x'\n",
		Err:    errors.New("exit status 128"),
	}
	got := e.Error()
	if strings.ContainsAny(got, "\x00\x01") {
		t.Errorf("Error() leaked control bytes from Args: %q", got)
	}
	want := "fatal: bad revision 'x' (git log)"
	if got != want {
		t.Errorf("Error() = %q, want %q", got, want)
	}
}

func TestError_FallsBackToUnderlyingErrWhenStderrEmpty(t *testing.T) {
	e := &Error{
		Args: []string{"log"},
		Err:  errors.New("exit status 1"),
	}
	got := e.Error()
	want := "git log: exit status 1"
	if got != want {
		t.Errorf("Error() = %q, want %q", got, want)
	}
}

func TestError_Unwrap(t *testing.T) {
	underlying := errors.New("boom")
	e := &Error{Err: underlying}
	if !errors.Is(e, underlying) {
		t.Error("errors.Is should see through Error to the underlying error")
	}
}

func TestFirstLine(t *testing.T) {
	cases := map[string]string{
		"":                          "",
		"single line":               "single line",
		"  padded  ":                "padded",
		"first\nsecond":             "first",
		"  first  \nsecond\nthird ": "first",
		"\n\nfirst after blanks\n":  "first after blanks",
	}
	for in, want := range cases {
		if got := firstLine(in); got != want {
			t.Errorf("firstLine(%q) = %q, want %q", in, got, want)
		}
	}
}
