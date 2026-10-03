package git

import (
	"errors"
	"io"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestParseLog_Simple(t *testing.T) {
	// NUL-delimited record with SOH field separator.
	raw := "\x00abc1234\x01parent111\x01Alice\x01alice@example.com\x01Alice\x01alice@example.com\x012024-01-15 10:30:00 +0000\x01feat: add authentication\x01Body text.\x00"

	commits, err := ParseLog([]byte(raw))
	if err != nil {
		t.Fatalf("ParseLog error: %v", err)
	}
	if len(commits) != 1 {
		t.Fatalf("expected 1 commit, got %d", len(commits))
	}

	c := commits[0]
	if c.Hash != "abc1234" {
		t.Errorf("Hash: got %q, want %q", c.Hash, "abc1234")
	}
	if len(c.Parents) != 1 || c.Parents[0] != "parent111" {
		t.Errorf("Parents: got %v", c.Parents)
	}
	if c.Author != "Alice" {
		t.Errorf("Author: got %q", c.Author)
	}
	if c.AuthorEmail != "alice@example.com" {
		t.Errorf("AuthorEmail: got %q", c.AuthorEmail)
	}
	if c.Subject != "feat: add authentication" {
		t.Errorf("Subject: got %q", c.Subject)
	}
	if c.Body != "Body text." {
		t.Errorf("Body: got %q", c.Body)
	}
	if c.ShortHash != "abc1234" {
		t.Errorf("ShortHash: got %q", c.ShortHash)
	}
	expectedTime := time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC)
	if !c.Timestamp.Equal(expectedTime) {
		t.Errorf("Timestamp: got %v, want %v", c.Timestamp, expectedTime)
	}
}

func TestParseLog_Multiple(t *testing.T) {
	raw := "\x00aaabbb\x01\x01Alice\x01a@x.com\x01Alice\x01a@x.com\x012024-01-01 12:00:00 +0000\x01first\x01\x00" +
		"\x00cccddd\x01aaabbb\x01Bob\x01b@x.com\x01Bob\x01b@x.com\x012024-01-02 12:00:00 +0000\x01second\x01\x00"

	commits, err := ParseLog([]byte(raw))
	if err != nil {
		t.Fatalf("ParseLog error: %v", err)
	}
	if len(commits) != 2 {
		t.Fatalf("expected 2 commits, got %d", len(commits))
	}
	if commits[0].Hash != "aaabbb" {
		t.Errorf("first hash: got %q", commits[0].Hash)
	}
	if commits[1].Hash != "cccddd" {
		t.Errorf("second hash: got %q", commits[1].Hash)
	}
	if commits[0].Parents != nil {
		t.Errorf("root commit should have no parents, got %v", commits[0].Parents)
	}
}

func TestParseLog_MergeCommit(t *testing.T) {
	raw := "\x00merge000\x01parent111 parent222\x01Dev\x01dev@x.com\x01Dev\x01dev@x.com\x012024-03-01 08:00:00 +0000\x01Merge branch\x01\x00"

	commits, err := ParseLog([]byte(raw))
	if err != nil {
		t.Fatalf("ParseLog error: %v", err)
	}
	if len(commits[0].Parents) != 2 {
		t.Fatalf("expected 2 parents, got %d", len(commits[0].Parents))
	}
	if commits[0].Parents[0] != "parent111" || commits[0].Parents[1] != "parent222" {
		t.Errorf("wrong parents: %v", commits[0].Parents)
	}
}

func TestParseLog_ShortHash(t *testing.T) {
	raw := "\x00abcdef1234567890\x01\x01A\x01a@b.com\x01A\x01a@b.com\x012024-01-01 00:00:00 +0000\x01subj\x01\x00"
	commits, err := ParseLog([]byte(raw))
	if err != nil {
		t.Fatal(err)
	}
	if commits[0].ShortHash != "abcdef1" {
		t.Errorf("ShortHash: got %q", commits[0].ShortHash)
	}
}

func TestParseLog_Empty(t *testing.T) {
	commits, err := ParseLog([]byte(""))
	if err != nil {
		t.Fatal(err)
	}
	if len(commits) != 0 {
		t.Errorf("expected empty, got %d", len(commits))
	}
}

// fiveRecordLog is five well-formed records back to back, matching what a
// real `git log --format=logFormat` stream looks like (each record wrapped
// in a leading+trailing \x00, so consecutive records touch).
const fiveRecordLog = "" +
	"\x00c1\x01\x01A\x01a@x.com\x01A\x01a@x.com\x012024-01-01 00:00:00 +0000\x01one\x01\x00" +
	"\x00c2\x01c1\x01A\x01a@x.com\x01A\x01a@x.com\x012024-01-02 00:00:00 +0000\x01two\x01\x00" +
	"\x00c3\x01c2\x01A\x01a@x.com\x01A\x01a@x.com\x012024-01-03 00:00:00 +0000\x01three\x01\x00" +
	"\x00c4\x01c3\x01A\x01a@x.com\x01A\x01a@x.com\x012024-01-04 00:00:00 +0000\x01four\x01\x00" +
	"\x00c5\x01c4\x01A\x01a@x.com\x01A\x01a@x.com\x012024-01-05 00:00:00 +0000\x01five\x01\x00"

func newTestLogStream(raw string) (*LogStream, *int) {
	waitCalls := 0
	wait := func() error { waitCalls++; return nil }
	kill := func() {}
	return newLogStream(strings.NewReader(raw), wait, kill), &waitCalls
}

func TestLogStream_PagesAcrossBoundaries(t *testing.T) {
	s, waitCalls := newTestLogStream(fiveRecordLog)

	page1, done, err := s.Next(2)
	if err != nil || done || len(page1) != 2 {
		t.Fatalf("page1: got %d commits, done=%v, err=%v", len(page1), done, err)
	}
	if page1[0].Hash != "c1" || page1[1].Hash != "c2" {
		t.Fatalf("page1 hashes: %q, %q", page1[0].Hash, page1[1].Hash)
	}

	page2, done, err := s.Next(2)
	if err != nil || done || len(page2) != 2 {
		t.Fatalf("page2: got %d commits, done=%v, err=%v", len(page2), done, err)
	}
	if page2[0].Hash != "c3" || page2[1].Hash != "c4" {
		t.Fatalf("page2 hashes: %q, %q", page2[0].Hash, page2[1].Hash)
	}

	// Final page: only one commit left, so Next must report done even
	// though it asked for 2.
	page3, done, err := s.Next(2)
	if err != nil || !done || len(page3) != 1 || page3[0].Hash != "c5" {
		t.Fatalf("page3: got %d commits (%v), done=%v, err=%v", len(page3), page3, done, err)
	}
	if *waitCalls != 1 {
		t.Errorf("expected wait to be called exactly once on EOF, got %d", *waitCalls)
	}

	// Close after done must be a no-op: wait must not be called again.
	s.Close()
	if *waitCalls != 1 {
		t.Errorf("Close after done re-invoked wait: %d calls", *waitCalls)
	}
}

// TestLogStream_ExactMultiple covers requesting exactly as many records as
// the stream has: like FetchLog's old len(commits)==pageSize heuristic,
// landing exactly on the end isn't detectable until one more Next() call
// hits EOF -- the first call can't know without over-reading.
func TestLogStream_ExactMultiple(t *testing.T) {
	s, _ := newTestLogStream(fiveRecordLog)

	all, done, err := s.Next(5)
	if err != nil || done || len(all) != 5 {
		t.Fatalf("got %d commits, done=%v, err=%v", len(all), done, err)
	}

	rest, done, err := s.Next(5)
	if err != nil || !done || len(rest) != 0 {
		t.Fatalf("got %d commits, done=%v, err=%v", len(rest), done, err)
	}
}

func TestLogStream_Empty(t *testing.T) {
	s, _ := newTestLogStream("")
	commits, done, err := s.Next(10)
	if err != nil || !done || len(commits) != 0 {
		t.Fatalf("got %d commits, done=%v, err=%v", len(commits), done, err)
	}
}

func TestLogStream_MalformedRecordKillsAndReturnsError(t *testing.T) {
	killed := false
	s := newLogStream(strings.NewReader("\x00not-enough-fields\x00"),
		func() error { return nil },
		func() { killed = true })

	commits, done, err := s.Next(10)
	if err == nil {
		t.Fatal("expected an error for a malformed record")
	}
	if !done {
		t.Error("expected done=true alongside the error")
	}
	if len(commits) != 0 {
		t.Errorf("expected no commits back from a malformed first record, got %d", len(commits))
	}
	if !killed {
		t.Error("expected a malformed record to kill (clean up) the stream")
	}

	// Close after an error-triggered cleanup must be a no-op.
	killed = false
	s.Close()
	if killed {
		t.Error("Close after an error cleanup re-invoked kill")
	}
}

func TestLogStream_WaitErrorPropagates(t *testing.T) {
	wantErr := errors.New("git log exited 128")
	s := newLogStream(strings.NewReader(fiveRecordLog[:0]), func() error { return wantErr }, func() {})

	_, done, err := s.Next(1)
	if !done || !errors.Is(err, wantErr) {
		t.Fatalf("got done=%v, err=%v, want done=true, err=%v", done, err, wantErr)
	}
}

// TestLogStream_ConcurrentNextAndClose is a regression test for a race the
// UI can trigger for real: nextPageCmd's Next() runs as its own tea.Cmd
// goroutine and can still be blocked reading when a superseded generation's
// closeLogStreamCmd calls Close() concurrently on the same stream. Both
// must be safe to race, and wait/kill must fire exactly once between them
// -- run under `go test -race` to catch a shared reaped bool or a doubled
// process-Wait call.
func TestLogStream_ConcurrentNextAndClose(t *testing.T) {
	pr, pw := io.Pipe()
	var calls int32
	wait := func() error { atomic.AddInt32(&calls, 1); return nil }
	kill := func() { atomic.AddInt32(&calls, 1); pw.Close() }
	s := newLogStream(pr, wait, kill)

	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		s.Next(10) // blocks on the pipe read until Close's Kill closes it
	}()
	go func() {
		defer wg.Done()
		s.Close()
	}()
	wg.Wait()

	if got := atomic.LoadInt32(&calls); got != 1 {
		t.Fatalf("expected wait/kill to fire exactly once across the race, got %d", got)
	}
}
