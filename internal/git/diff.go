package git

import (
	"context"
	"strings"

	"github.com/eurico-martins/lazytree/internal/model"
)

// FetchDiff returns the parsed diff for a commit.
func FetchDiff(ctx context.Context, r *Runner, hash string) ([]model.DiffFile, error) {
	raw, err := r.Run(ctx, "show", "--patch", "--format=", hash)
	if err != nil {
		return nil, err
	}
	return ParseDiff(string(raw)), nil
}

// ParseDiff parses unified diff output (as produced by `git show --patch`)
// into DiffFile structs.
func ParseDiff(raw string) []model.DiffFile {
	var files []model.DiffFile
	var cur *model.DiffFile
	var curHunk *model.DiffHunk

	lines := strings.Split(raw, "\n")
	// git's output ends with a trailing newline after the last content
	// line, so Split yields one spurious empty trailing element; left in,
	// it would show up as a bogus blank context line at the end of the
	// last hunk.
	if len(lines) > 0 && lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}

	for _, line := range lines {
		switch {
		case strings.HasPrefix(line, "diff --git "):
			if cur != nil {
				if curHunk != nil {
					cur.Hunks = append(cur.Hunks, *curHunk)
					curHunk = nil
				}
				files = append(files, finalizeDiffFile(*cur))
			}
			cur = &model.DiffFile{Status: "M"}
			parseDiffHeader(line, cur)

		case cur == nil:
			// ignore preamble

		case strings.HasPrefix(line, "new file mode"):
			cur.Status = "A"

		case strings.HasPrefix(line, "deleted file mode"):
			cur.Status = "D"

		case strings.HasPrefix(line, "rename from "):
			cur.Status = "R"
			cur.OldPath = strings.TrimPrefix(line, "rename from ")

		case strings.HasPrefix(line, "rename to "):
			cur.NewPath = strings.TrimPrefix(line, "rename to ")

		case strings.HasPrefix(line, "copy from "):
			cur.Status = "C"
			cur.OldPath = strings.TrimPrefix(line, "copy from ")

		case strings.HasPrefix(line, "copy to "):
			cur.NewPath = strings.TrimPrefix(line, "copy to ")

		// The "--- "/"+++ " file headers only appear before a file's first
		// hunk. Inside a hunk, a removed line reading "-- a/x" or an added
		// line reading "++ b/x" looks identical, and must stay a diff line
		// rather than overwrite the paths.
		case curHunk == nil && strings.HasPrefix(line, "--- "):
			// Overrides the "diff --git" header's guess with the real old
			// path: see parseUnifiedPath for why this is the reliable
			// source when the path contains a space.
			if path, ok := parseUnifiedPath(line, "--- a/"); ok {
				cur.OldPath = path
			}

		case curHunk == nil && strings.HasPrefix(line, "+++ "):
			if path, ok := parseUnifiedPath(line, "+++ b/"); ok {
				cur.NewPath = path
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
		files = append(files, finalizeDiffFile(*cur))
	}
	return files
}

// finalizeDiffFile cleans up a fully-parsed DiffFile before it's returned.
// An added or deleted file has only one real path -- the other side is
// "/dev/null" in git's output and carries nothing usable (see
// parseUnifiedPath) -- so parseDiffHeader's space-ambiguous guess can be
// left in the unused field. Mirroring the real path onto it here means a
// caller never sees that guess, correct or not.
func finalizeDiffFile(f model.DiffFile) model.DiffFile {
	switch f.Status {
	case "A":
		f.OldPath = f.NewPath
	case "D":
		f.NewPath = f.OldPath
	}
	return f
}

// parseDiffHeader reads a first-guess pair of paths from
// "diff --git a/foo b/foo" by splitting on spaces. That guess is wrong for
// a path containing a space (there's no delimiter between "a/X" and "b/Y"
// other than the space inside X itself), so it's only a fallback: the
// "--- "/"+++ " and "rename from"/"rename to" cases in ParseDiff overwrite
// it with the real paths whenever those lines are present, which is every
// case except a content-free mode change (chmod) on a path with a space --
// too rare an edge case to warrant more than this comment.
func parseDiffHeader(line string, f *model.DiffFile) {
	parts := strings.Fields(line)
	if len(parts) >= 4 {
		f.OldPath = strings.TrimPrefix(parts[2], "a/")
		f.NewPath = strings.TrimPrefix(parts[3], "b/")
	}
}

// parseUnifiedPath extracts the path from a "--- a/X" or "+++ b/Y" unified
// diff line. It reports ok=false for "--- /dev/null"/"+++ /dev/null" (the
// added/deleted side has no real path) so the caller can leave whatever
// parseDiffHeader already set. git appends a trailing tab after a path that
// contains a space, to disambiguate it from the timestamp field unified
// diff historically carried here; that tab is trimmed if present.
func parseUnifiedPath(line, prefix string) (path string, ok bool) {
	if !strings.HasPrefix(line, prefix) {
		return "", false
	}
	path = strings.TrimPrefix(line, prefix)
	path = strings.TrimSuffix(path, "\t")
	return path, true
}
