package git

import (
	"bytes"
	"context"
	"os/exec"
)

type Runner struct {
	RepoPath string
}

func (r *Runner) Run(ctx context.Context, args ...string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Dir = r.RepoPath
	var out, errOut bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &errOut
	if err := cmd.Run(); err != nil {
		return nil, err
	}
	return out.Bytes(), nil
}
