package git

import (
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
