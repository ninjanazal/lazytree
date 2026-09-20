#!/usr/bin/env bash
# Builds a throwaway git repo with a realistic, densely-branched history
# (nested feature branches, a hotfix, overlapping merges, a tag, and a
# lagging remote) so lazytree's graph rendering (lane colors, merge/branch
# diagonals, ref pills) can be seen in action instead of the flat
# single-branch history of this repo itself.
#
# Usage:
#   scripts/demo-repo.sh            # prints the generated repo path
#   scripts/demo-repo.sh --run      # builds lazytree and launches it against it
set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

DIR="$(mktemp -d "${TMPDIR:-/tmp}/lazytree-demo.XXXXXX")"
BARE="$(mktemp -d "${TMPDIR:-/tmp}/lazytree-demo-origin.XXXXXX")"

git init -q -b main "$DIR"
git init -q --bare "$BARE"

cd "$DIR"
git config user.email "demo@example.com"
git config user.name "Demo User"
export GIT_AUTHOR_DATE GIT_COMMITTER_DATE
t=$(date -d '-3 weeks' +%s 2>/dev/null || date -v-3w +%s)

commit() {
	t=$((t + 3600))
	GIT_AUTHOR_DATE="@$t" GIT_COMMITTER_DATE="@$t" git commit -q --allow-empty -m "$1"
}

merge() {
	t=$((t + 3600))
	GIT_AUTHOR_DATE="@$t" GIT_COMMITTER_DATE="@$t" git merge -q --no-ff "$1" -m "${2:-Merge branch '$1'}"
}

# --- trunk gets going -------------------------------------------------
git config user.name "Jane Doe"
commit "initial commit"
commit "feat: keyboard shortcuts"
commit "feat: responsive layout"

git remote add origin "$BARE"
git push -q origin main

# --- feature/ui, and feature/auth branching off of it (nested) --------
git config user.name "Taylor Kim"
git checkout -q -b feature/ui
commit "feat: keyboard shortcuts polish"
commit "feat: responsive layout tweaks"

git config user.name "Priya Nair"
git checkout -q -b feature/auth
commit "feat: login screen scaffold"
commit "feat: session token storage"

# --- meanwhile, trunk keeps moving -------------------------------------
git config user.name "Jane Doe"
git checkout -q main
commit "feat: better error messages"
commit "fix: handle detached HEAD"
git push -q origin main
git tag v0.1.0

# --- feature/perf, off an older point on trunk -------------------------
git config user.name "Taylor Kim"
git checkout -q -b feature/perf main~1
commit "perf: cache git results"
commit "perf: lazy load commit details"

# --- feature/search, off the same point, with a nested sub-branch ------
git config user.name "Sam Wilson"
git checkout -q -b feature/search main~1
commit "fix: search highlighting"
commit "feat: add search by author and ref"

git config user.name "Alex Chen"
git checkout -q -b feature/search-ui
commit "feat: search result highlighting"

# --- trunk merges search-ui (bringing feature/search along with it) ----
git config user.name "Alex Chen"
git checkout -q main
merge feature/search-ui
commit "docs: update README with screenshots"

# --- a quick hotfix, branched and merged right back in ------------------
git config user.name "Jane Doe"
git checkout -q -b hotfix/crash-on-empty-repo
commit "fix: crash when opening an empty repository"
git checkout -q main
merge hotfix/crash-on-empty-repo

# --- trunk absorbs the remaining long-running branches ------------------
merge feature/perf
commit "feat: commit inspector with file list"

git config user.name "Priya Nair"
merge feature/auth
commit "refactor: split renderer into components"

git config user.name "Jane Doe"
commit "fix: handle large diffs without blocking UI"
commit "feat: add diff viewer with syntax highlighting"
git push -q origin main

git checkout -q main >/dev/null

echo "$DIR"

if [[ "${1:-}" == "--run" ]]; then
	(cd "$repo_root" && go build -o lazytree ./cmd/lazytree)
	exec "$repo_root/lazytree" "$DIR"
fi
