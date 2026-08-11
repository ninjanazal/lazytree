package git

import (
	"bytes"
	"context"
	"fmt"
	"strings"
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
		return nil, fmt.Errorf("git log: %w", err)
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
