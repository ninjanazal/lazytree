package git

import (
	"context"
	"strings"
)

// SearchKind selects which `git log` filter SearchHashes uses.
type SearchKind int

const (
	SearchGrep SearchKind = iota // commit messages: --grep
	SearchCode                   // added/removed text: -S (pickaxe)
	SearchPath                   // commits touching a path: -- <path>
)

// SearchHashes returns the hashes of every commit matching term, scanning
// the whole history (not just what the UI has loaded so far). all adds
// --all, mirroring the log view's refs toggle.
func SearchHashes(ctx context.Context, r *Runner, kind SearchKind, term string, all bool) ([]string, error) {
	args := []string{"log", "--format=%H"}
	if all {
		args = append(args, "--all")
	}
	switch kind {
	case SearchGrep:
		args = append(args, "--regexp-ignore-case", "--fixed-strings", "--grep="+term)
	case SearchCode:
		args = append(args, "-S"+term)
	case SearchPath:
		args = append(args, "--", term)
	}
	raw, err := r.Run(ctx, args...)
	if err != nil {
		return nil, err
	}
	var hashes []string
	for line := range strings.SplitSeq(string(raw), "\n") {
		if line = strings.TrimSpace(line); line != "" {
			hashes = append(hashes, line)
		}
	}
	return hashes, nil
}
