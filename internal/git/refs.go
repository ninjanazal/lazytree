package git

import (
	"context"
	"strings"

	"github.com/eurico-martins/lazytree/internal/model"
)

// FetchRefs returns all refs in the repository.
func FetchRefs(ctx context.Context, r *Runner) ([]model.Ref, error) {
	raw, err := r.Run(ctx, "for-each-ref",
		"--format=%(refname)\t%(objectname)\t%(HEAD)",
		"refs/heads", "refs/remotes", "refs/tags",
	)
	if err != nil {
		return nil, err
	}

	headHash, _ := r.Run(ctx, "rev-parse", "HEAD")
	head := strings.TrimSpace(string(headHash))

	var refs []model.Ref
	for _, line := range strings.Split(strings.TrimSpace(string(raw)), "\n") {
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, "\t", 3)
		if len(parts) < 2 {
			continue
		}
		refname := parts[0]
		objHash := parts[1]
		isHead := objHash == head

		ref := classifyRef(refname, isHead)
		refs = append(refs, ref)
	}

	return refs, nil
}

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
		"refs/heads", "refs/remotes", "refs/tags",
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
		"refs/heads", "refs/remotes", "refs/tags",
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
	return result, nil
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

// FetchChangedFiles returns the list of files changed in a commit.
func FetchChangedFiles(ctx context.Context, r *Runner, hash string) ([]string, error) {
	raw, err := r.Run(ctx, "show", "--name-status", "--format=", hash)
	if err != nil {
		return nil, err
	}
	var files []string
	for _, line := range strings.Split(strings.TrimSpace(string(raw)), "\n") {
		line = strings.TrimSpace(line)
		if line != "" {
			files = append(files, line)
		}
	}
	return files, nil
}
