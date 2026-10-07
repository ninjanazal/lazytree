package git

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/eurico-martins/lazytree/internal/model"
)

func classifyRef(refname string, isHead bool) model.Ref {
	switch {
	case strings.HasPrefix(refname, "refs/heads/"):
		name := strings.TrimPrefix(refname, "refs/heads/")
		return model.Ref{Name: name, Kind: model.RefLocalBranch, IsHead: isHead}
	case strings.HasPrefix(refname, "refs/remotes/"):
		name := strings.TrimPrefix(refname, "refs/remotes/")
		return model.Ref{Name: name, Kind: model.RefRemoteBranch, IsHead: isHead}
	case strings.HasPrefix(refname, "refs/tags/"):
		name := strings.TrimPrefix(refname, "refs/tags/")
		return model.Ref{Name: name, Kind: model.RefTag, IsHead: false}
	case refname == "refs/stash":
		return model.Ref{Name: "stash", Kind: model.RefStash}
	default:
		return model.Ref{Name: refname, Kind: model.RefHead, IsHead: isHead}
	}
}

// AttachRefs merges ref information into commits by matching hashes.
func AttachRefs(commits []model.Commit, refsByHash map[string][]model.Ref) {
	for i, c := range commits {
		if rs, ok := refsByHash[c.Hash]; ok {
			commits[i].Refs = rs
		}
	}
}

// BuildRefsByHash runs for-each-ref and maps full commit hash → []Ref.
func BuildRefsByHash(ctx context.Context, r *Runner) (map[string][]model.Ref, error) {
	raw, err := r.Run(ctx, "for-each-ref",
		"--format=%(objectname)\t%(refname)\t*%(objectname)",
		"refs/heads", "refs/remotes", "refs/tags", "refs/stash",
	)
	if err != nil {
		return nil, err
	}

	headRaw, _ := r.Run(ctx, "rev-parse", "HEAD")
	headHash := strings.TrimSpace(string(headRaw))

	// Also get dereferenced tag targets
	rawDeref, _ := r.Run(ctx, "for-each-ref",
		"--format=%(objectname)\t%(refname)\t%(object)",
		"--dereference",
		"refs/heads", "refs/remotes", "refs/tags", "refs/stash",
	)

	result := map[string][]model.Ref{}

	parseLines := func(data []byte) {
		for _, line := range strings.Split(strings.TrimSpace(string(data)), "\n") {
			if line == "" {
				continue
			}
			parts := strings.SplitN(line, "\t", 3)
			if len(parts) < 2 {
				continue
			}
			hash := strings.TrimSpace(parts[0])
			refname := strings.TrimSpace(parts[1])
			if refname == "" || hash == "" {
				continue
			}
			// Skip the peeled ^{} lines that appear in some formats
			if strings.HasSuffix(refname, "^{}") {
				// use the object (parts[2]) as target hash
				if len(parts) == 3 && parts[2] != "" {
					hash = strings.TrimSpace(parts[2])
					refname = strings.TrimSuffix(refname, "^{}")
				} else {
					continue
				}
			}
			isHead := hash == headHash
			ref := classifyRef(refname, isHead)
			result[hash] = append(result[hash], ref)
		}
	}

	parseLines(raw)
	parseLines(rawDeref)
	addStashParents(ctx, r, result)
	return result, nil
}

// RefsFingerprint returns a deterministic string summarizing a refsByHash
// map (as built by BuildRefsByHash), suitable for cheaply detecting whether
// ref state changed (e.g. after a background `git fetch`) without diffing
// the maps directly or re-walking commit history. Map iteration order is
// randomized in Go, so the lines are sorted before joining.
func RefsFingerprint(refsByHash map[string][]model.Ref) string {
	lines := make([]string, 0, len(refsByHash))
	for hash, refs := range refsByHash {
		for _, r := range refs {
			lines = append(lines, fmt.Sprintf("%s %d %s", hash, r.Kind, r.Name))
		}
	}
	sort.Strings(lines)
	return strings.Join(lines, "\n")
}

// CurrentBranch returns the current branch name, or "" if HEAD is detached.
func CurrentBranch(ctx context.Context, r *Runner) (string, error) {
	raw, err := r.Run(ctx, "rev-parse", "--abbrev-ref", "HEAD")
	if err != nil {
		return "", err
	}
	name := strings.TrimSpace(string(raw))
	if name == "HEAD" {
		return "", nil
	}
	return name, nil
}

// addStashParents marks the helper commits git creates for a stash. A stash
// is a merge whose 2nd parent holds the staged changes ("index on ...") and,
// for `git stash -u`, whose 3rd parent holds untracked files. FoldStash uses
// the marks to hide them, so the stash draws as one commit off its base.
func addStashParents(ctx context.Context, r *Runner, refs map[string][]model.Ref) {
	for n, label := range map[string]string{"2": "stash index", "3": "stash untracked"} {
		raw, err := r.Run(ctx, "rev-parse", "--verify", "--quiet", "refs/stash^"+n)
		if err != nil {
			continue
		}
		hash := strings.TrimSpace(string(raw))
		if hash != "" {
			refs[hash] = append(refs[hash], model.Ref{Name: label, Kind: model.RefStashHelper})
		}
	}
}

// FoldStash hides stash helper commits (see addStashParents) and trims the
// stash commit to its first parent, so it renders as a single commit branching
// off the commit it was made on rather than a confusing merge. Refs must
// already be attached. It filters in place and returns the shortened slice.
func FoldStash(commits []model.Commit) []model.Commit {
	out := commits[:0]
	for _, c := range commits {
		helper, stash := false, false
		for _, r := range c.Refs {
			helper = helper || r.Kind == model.RefStashHelper
			stash = stash || r.Kind == model.RefStash
		}
		if helper {
			continue
		}
		if stash && len(c.Parents) > 1 {
			c.Parents = c.Parents[:1]
		}
		out = append(out, c)
	}
	return out
}

// IsStash reports whether c carries the stash ref.
func IsStash(c model.Commit) bool {
	for _, r := range c.Refs {
		if r.Kind == model.RefStash {
			return true
		}
	}
	return false
}
