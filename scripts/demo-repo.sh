#!/usr/bin/env bash
# Builds a throwaway git repo with a realistic, densely-branched history
# (deeply nested feature branches, diamond merges, a release branch with
# backports, hotfixes, tags, and remote branches that lag behind their
# local counterpart) so lazytree's graph rendering (lane colors,
# merge/branch diagonals, ref pills) can be seen in action instead of the
# flat single-branch history of this repo itself.
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
t=$(date -d '-6 weeks' +%s 2>/dev/null || date -v-6w +%s)

commit() {
	t=$((t + 3600))
	GIT_AUTHOR_DATE="@$t" GIT_COMMITTER_DATE="@$t" git commit -q --allow-empty -m "$1"
}

merge() {
	t=$((t + 3600))
	GIT_AUTHOR_DATE="@$t" GIT_COMMITTER_DATE="@$t" git merge -q --no-ff "$1" -m "${2:-Merge branch '$1'}"
}

cherry() {
	t=$((t + 3600))
	GIT_AUTHOR_DATE="@$t" GIT_COMMITTER_DATE="@$t" git cherry-pick -q "$1" >/dev/null 2>&1 || true
}

# --- trunk gets going -------------------------------------------------
git config user.name "Jane Doe"
commit "initial commit"
commit "feat: keyboard shortcuts"
commit "feat: responsive layout"

git remote add origin "$BARE"
git push -q origin main

# --- feature/ui, and feature/auth branching off of it (nested 2 deep) --
git config user.name "Taylor Kim"
git checkout -q -b feature/ui
commit "feat: keyboard shortcuts polish"
commit "feat: responsive layout tweaks"
git push -q origin feature/ui

git config user.name "Priya Nair"
git checkout -q -b feature/auth
commit "feat: login screen scaffold"
commit "feat: session token storage"

# feature/auth grows a nested sub-branch (3 deep)
git config user.name "Priya Nair"
git checkout -q -b feature/auth-oauth
commit "feat: oauth provider scaffolding"
commit "feat: google oauth flow"

# and a 4th-level branch off that, for extra nesting
git config user.name "Morgan Lee"
git checkout -q -b feature/auth-oauth-refresh
commit "feat: refresh token rotation"

# --- meanwhile, trunk keeps moving -------------------------------------
git config user.name "Jane Doe"
git checkout -q main
commit "feat: better error messages"
commit "fix: handle detached HEAD"
git push -q origin main
git tag v0.1.0

# feature/ui keeps moving locally after its remote push, so it now
# lags behind the local branch (visible as local-ahead-of-remote)
git config user.name "Taylor Kim"
git checkout -q feature/ui
commit "feat: focus ring styling"

# --- feature/perf, off an older point on trunk -------------------------
git config user.name "Taylor Kim"
git checkout -q -b feature/perf main~1
commit "perf: cache git results"
commit "perf: lazy load commit details"
git push -q origin feature/perf

# feature/perf grows its own sub-branch
git config user.name "Sam Wilson"
git checkout -q -b feature/perf-workers
commit "perf: move layout to worker pool"
commit "perf: batch graph recomputation"

# --- feature/search, off the same point, with a nested sub-branch ------
git config user.name "Sam Wilson"
git checkout -q -b feature/search main~1
commit "fix: search highlighting"
commit "feat: add search by author and ref"

git config user.name "Alex Chen"
git checkout -q -b feature/search-ui
commit "feat: search result highlighting"

git config user.name "Alex Chen"
git checkout -q -b feature/search-ui-kbd
commit "feat: search keyboard navigation"

# diamond: search-ui-kbd merges back into feature/search-ui before that
# in turn merges into feature/search
git config user.name "Alex Chen"
git checkout -q feature/search-ui
merge feature/search-ui-kbd

git config user.name "Sam Wilson"
git checkout -q feature/search
merge feature/search-ui

# --- trunk merges search (bringing the whole diamond along with it) ----
git config user.name "Alex Chen"
git checkout -q main
merge feature/search
commit "docs: update README with screenshots"

# --- a quick hotfix, branched and merged right back in ------------------
git config user.name "Jane Doe"
git checkout -q -b hotfix/crash-on-empty-repo
commit "fix: crash when opening an empty repository"
git checkout -q main
merge hotfix/crash-on-empty-repo
git push -q origin main

# --- release branch, cut from main, with a backported hotfix -----------
git config user.name "Jane Doe"
git checkout -q -b release/0.2 main
commit "chore: cut release/0.2"
git tag v0.2.0-rc1

git config user.name "Jane Doe"
git checkout -q -b hotfix/nil-pointer-on-refresh release/0.2
commit "fix: nil pointer when refreshing empty log"
git checkout -q release/0.2
merge hotfix/nil-pointer-on-refresh

# backport the same hotfix onto main via cherry-pick
git config user.name "Jane Doe"
git checkout -q main
cherry hotfix/nil-pointer-on-refresh
git push -q origin main

# --- trunk absorbs the remaining long-running branches ------------------
git config user.name "Morgan Lee"
git checkout -q main
merge feature/auth-oauth-refresh "Merge branch 'feature/auth-oauth-refresh' via feature/auth-oauth"
commit "feat: commit inspector with file list"

git config user.name "Priya Nair"
merge feature/auth
commit "refactor: split renderer into components"

git config user.name "Sam Wilson"
merge feature/perf-workers
commit "perf: reduce allocations in lane assignment"

git config user.name "Taylor Kim"
merge feature/ui

git config user.name "Jane Doe"
commit "fix: handle large diffs without blocking UI"
commit "feat: add diff viewer with syntax highlighting"
git push -q origin main
git tag v0.2.0

# --- an abandoned experiment, never merged, left dangling --------------
git config user.name "Morgan Lee"
git checkout -q -b experiment/gpu-render main~4
commit "wip: prototype gpu-accelerated graph rendering"
commit "wip: benchmark against CPU renderer"

# --- a fresh in-flight feature branch, still open on trunk tip ---------
git config user.name "Priya Nair"
git checkout -q -b feature/multi-repo main
commit "feat: repo picker scaffold"
commit "feat: recent repos list"

git checkout -q main >/dev/null

echo "$DIR"

if [[ "${1:-}" == "--run" ]]; then
	(cd "$repo_root" && go build -o lazytree ./cmd/lazytree)
	exec "$repo_root/lazytree" "$DIR"
fi
