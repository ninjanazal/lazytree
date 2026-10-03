#!/usr/bin/env bash
# Runs inside the test container (see docker/Dockerfile).
set -euo pipefail

echo "== $(go version)"
echo "== $(git --version)"

make lint test

# macOS's temp dir (/var/folders/... -> /private/var/...) is a symlink, which
# newer git rejects in some submodule/worktree paths. Re-run the git-facing
# tests from behind a symlinked TMPDIR to catch that class of bug on Linux.
real="$(mktemp -d)"
link="$(mktemp -u)"
ln -s "$real" "$link"
echo "== re-running git-facing tests with TMPDIR behind a symlink ($link -> $real)"
TMPDIR="$link" go test -count=1 ./internal/git/... ./cmd/...

echo "== OK"
