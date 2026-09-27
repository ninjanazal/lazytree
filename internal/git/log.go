package git

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"strings"
	"sync"
	"time"

	"github.com/eurico-martins/lazytree/internal/model"
)

// logFormat uses NUL as record separator and SOH as field separator.
// This avoids ambiguity with newlines in commit bodies.
const logFormat = "%x00%H\x01%P\x01%an\x01%ae\x01%cn\x01%ce\x01%ai\x01%s\x01%b%x00"

// FetchLog retrieves commits from the repository.
func FetchLog(ctx context.Context, r *Runner, extraArgs ...string) ([]model.Commit, error) {
	args := []string{"log", "--format=" + logFormat}
	args = append(args, extraArgs...)
	raw, err := r.Run(ctx, args...)
	if err != nil {
		return nil, err
	}
	return ParseLog(raw)
}

// ParseLog parses the output of git log with logFormat.
func ParseLog(raw []byte) ([]model.Commit, error) {
	// Records are delimited by NUL bytes; split and clean up.
	records := bytes.Split(raw, []byte("\x00"))
	var commits []model.Commit
	for _, rec := range records {
		rec = bytes.TrimSpace(rec)
		if len(rec) == 0 {
			continue
		}
		c, err := parseRecord(rec)
		if err != nil {
			return nil, err
		}
		commits = append(commits, c)
	}
	return commits, nil
}

// LogStream reads commits from a `git log` process incrementally via
// Next, instead of FetchLog's buffer-the-whole-output-then-parse. It's
// built to replace `--skip=N` paging: `--skip` makes git re-walk history
// from the start of the ref range on every page, which is O(n²/pageSize)
// total work on a large repo, whereas a stream reads the same walk once.
//
// Next and Close can be called from different goroutines: the UI runs each
// page read as its own tea.Cmd, and a stream can be abandoned (Close) by a
// superseded reload while a page read is still blocked in Next. mu guards
// the reaped transition so wait/kill is invoked exactly once even when
// both race to finish the stream at the same time.
type LogStream struct {
	buf  *bufio.Reader
	wait func() error
	kill func()

	mu     sync.Mutex
	reaped bool
}

// StartLogStream starts `git log --format=<logFormat> extraArgs...` and
// returns a LogStream over its stdout. Unlike FetchLog, extraArgs should
// not include --skip/--max-count: the whole log streams through Next.
func StartLogStream(ctx context.Context, r *Runner, extraArgs ...string) (*LogStream, error) {
	args := append([]string{"log", "--format=" + logFormat}, extraArgs...)
	sc, err := r.Start(ctx, args...)
	if err != nil {
		return nil, err
	}
	return newLogStream(sc.Stdout, sc.Wait, sc.Kill), nil
}

// newLogStream is StartLogStream's underlying logic, decoupled from a real
// subprocess so it can be unit-tested against a plain io.Reader.
func newLogStream(r io.Reader, wait func() error, kill func()) *LogStream {
	return &LogStream{buf: bufio.NewReaderSize(r, 64*1024), wait: wait, kill: kill}
}

// Next reads up to n more commits from the stream. done is true once the
// stream is exhausted or fails, at which point the underlying process has
// already been waited on (or killed) -- callers don't need to call Close
// themselves unless they're abandoning the stream before done is reached.
func (s *LogStream) Next(n int) (commits []model.Commit, done bool, err error) {
	for len(commits) < n {
		rec, rerr := s.buf.ReadBytes(0)
		// logFormat wraps every record in a leading and trailing \x00, so
		// consecutive records produce an empty read between them -- skip
		// those exactly like ParseLog's bytes.Split + skip-empty loop does.
		if trimmed := bytes.TrimSpace(bytes.TrimSuffix(rec, []byte{0})); len(trimmed) > 0 {
			c, perr := parseRecord(trimmed)
			if perr != nil {
				s.finishKill()
				return commits, true, perr
			}
			commits = append(commits, c)
		}
		if rerr != nil {
			if rerr != io.EOF {
				s.finishKill()
				return commits, true, rerr
			}
			return commits, true, s.finishWait()
		}
	}
	return commits, false, nil
}

// Close abandons the stream before Next has reported done, killing the
// underlying process. Idempotent and safe to call after done, too, and
// safe to call concurrently with an in-flight Next (see LogStream's doc).
func (s *LogStream) Close() {
	s.finishKill()
}

// finishWait and finishKill both terminate the stream, from Next reaching
// EOF/an error or from Close abandoning it. Whichever runs first performs
// the underlying wait/kill and marks reaped; the other becomes a no-op --
// wait/kill must never be invoked twice for the same process.
func (s *LogStream) finishWait() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.reaped {
		return nil
	}
	s.reaped = true
	return s.wait()
}

func (s *LogStream) finishKill() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.reaped {
		return
	}
	s.reaped = true
	s.kill()
}

func parseRecord(rec []byte) (model.Commit, error) {
	parts := strings.SplitN(string(rec), "\x01", 9)
	if len(parts) < 8 {
		return model.Commit{}, fmt.Errorf("malformed commit record: %q", rec)
	}

	hash := strings.TrimSpace(parts[0])
	parentStr := strings.TrimSpace(parts[1])
	author := parts[2]
	authorEmail := parts[3]
	committer := parts[4]
	committerEmail := parts[5]
	dateStr := strings.TrimSpace(parts[6])
	subject := strings.TrimSpace(parts[7])
	body := ""
	if len(parts) == 9 {
		body = strings.TrimSpace(parts[8])
	}

	var parents []string
	if parentStr != "" {
		for _, p := range strings.Fields(parentStr) {
			parents = append(parents, p)
		}
	}

	ts, err := time.Parse("2006-01-02 15:04:05 -0700", dateStr)
	if err != nil {
		// Try ISO 8601 without space
		ts, err = time.Parse("2006-01-02T15:04:05-07:00", dateStr)
		if err != nil {
			ts = time.Time{}
		}
	}

	shortHash := hash
	if len(hash) > 7 {
		shortHash = hash[:7]
	}

	return model.Commit{
		Hash:           hash,
		ShortHash:      shortHash,
		Parents:        parents,
		Author:         author,
		AuthorEmail:    authorEmail,
		Committer:      committer,
		CommitterEmail: committerEmail,
		Timestamp:      ts,
		Subject:        subject,
		Body:           body,
	}, nil
}
