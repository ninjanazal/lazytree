package git

import (
	"context"
	"fmt"
	"strings"

	"github.com/eurico-martins/lazytree/internal/model"
)

// FetchDiff returns the parsed diff for a commit.
func FetchDiff(ctx context.Context, r *Runner, hash string) ([]model.DiffFile, error) {
	raw, err := r.Run(ctx, "show", "--patch", "--format=", hash)
	if err != nil {
		return nil, fmt.Errorf("git show: %w", err)
	}
	return ParseDiff(string(raw)), nil
}

// ParseDiff parses unified diff output into DiffFile structs.
func ParseDiff(raw string) []model.DiffFile {
	var files []model.DiffFile
	var cur *model.DiffFile
	var curHunk *model.DiffHunk

	lines := strings.Split(raw, "\n")
	for _, line := range lines {
		switch {
		case strings.HasPrefix(line, "diff --git "):
			if cur != nil {
				if curHunk != nil {
					cur.Hunks = append(cur.Hunks, *curHunk)
					curHunk = nil
				}
				files = append(files, *cur)
			}
			cur = &model.DiffFile{}
			parseDiffHeader(line, cur)

		case cur == nil:
			// ignore preamble

		case strings.HasPrefix(line, "--- "):
			if cur.OldPath == "" {
				cur.OldPath = strings.TrimPrefix(line, "--- a/")
				cur.OldPath = strings.TrimPrefix(cur.OldPath, "--- /dev/null")
			}

		case strings.HasPrefix(line, "+++ "):
			if cur.NewPath == "" {
				cur.NewPath = strings.TrimPrefix(line, "+++ b/")
				cur.NewPath = strings.TrimPrefix(cur.NewPath, "+++ /dev/null")
			}

		case strings.HasPrefix(line, "@@"):
			if curHunk != nil {
				cur.Hunks = append(cur.Hunks, *curHunk)
			}
			curHunk = &model.DiffHunk{Header: line}

		case curHunk != nil:
			var kind model.DiffLineKind
			switch {
			case strings.HasPrefix(line, "+"):
				kind = model.DiffAdded
			case strings.HasPrefix(line, "-"):
				kind = model.DiffRemoved
			default:
				kind = model.DiffContext
			}
			curHunk.Lines = append(curHunk.Lines, model.DiffLine{Kind: kind, Text: line})
		}
	}

	if cur != nil {
		if curHunk != nil {
			cur.Hunks = append(cur.Hunks, *curHunk)
		}
		files = append(files, *cur)
	}
	return files
}

func parseDiffHeader(line string, f *model.DiffFile) {
	// "diff --git a/foo b/foo"
	parts := strings.Fields(line)
	if len(parts) >= 4 {
		f.OldPath = strings.TrimPrefix(parts[2], "a/")
		f.NewPath = strings.TrimPrefix(parts[3], "b/")
	}
}
