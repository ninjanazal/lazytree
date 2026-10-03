#!/usr/bin/env bash
# Prepares a release ENTIRELY LOCALLY. It never pushes and never publishes.
#
#   scripts/release.sh v0.1.0              # check, tag, build files, write notes
#   scripts/release.sh v0.1.0 --dry-run    # preview: no tag; files + notes only
#   FAST=1 scripts/release.sh v0.1.0       # skip the (slower) Docker matrix
#
# What it does, in order:
#   1. refuses unless the working tree is clean and the tag does not exist
#   2. runs `make check` and `make docker-matrix` (FAST=1 skips the matrix)
#   3. creates the annotated git tag locally
#   4. builds the archives + checksums into ./dist (goreleaser, publish skipped)
#   5. writes the release text to dist/RELEASE_NOTES.md and prints it
#   6. prints the manual steps left for you (push the tag, upload, paste text)
set -euo pipefail

cd "$(dirname "${BASH_SOURCE[0]}")/.."

VERSION="${1:-}"
DRY=0
[[ "${2:-}" == "--dry-run" ]] && DRY=1

if [[ ! "$VERSION" =~ ^v[0-9]+\.[0-9]+\.[0-9]+(-[0-9A-Za-z.]+)?$ ]]; then
  echo "usage: scripts/release.sh vX.Y.Z [--dry-run]   (e.g. v0.1.0)" >&2
  exit 2
fi

bold() { printf '\033[1m%s\033[0m\n' "$*"; }

# ---------------------------------------------------------------- 1. guards
# Obsidian rewrites its own workspace/plugin files constantly; they are not
# part of the program, so they don't count as a dirty tree.
dirty="$(git status --porcelain -- . ':!LazyTreeObsidian/.obsidian')"
if [[ $DRY -eq 0 ]]; then
  if [[ -n "$dirty" ]]; then
    echo "working tree is not clean; commit or stash first:" >&2
    echo "$dirty" >&2
    exit 1
  fi
  if git rev-parse -q --verify "refs/tags/$VERSION" >/dev/null; then
    echo "tag $VERSION already exists (delete it with: git tag -d $VERSION)" >&2
    exit 1
  fi
  branch="$(git rev-parse --abbrev-ref HEAD)"
  if [[ "$branch" != "main" ]]; then
    echo "note: releasing from branch '$branch', not main" >&2
  fi
else
  bold "DRY RUN: no tag will be created"
  [[ -n "$dirty" ]] && echo "note: working tree has uncommitted changes (a real release would refuse)"
fi

# ---------------------------------------------------------------- 2. checks
bold "==> make check"
make check
if [[ -z "${FAST:-}" ]]; then
  bold "==> make docker-matrix"
  make docker-matrix
else
  echo "(FAST=1: skipping the Docker matrix)"
fi

# ---------------------------------------------------------------- 3. tag
if [[ $DRY -eq 0 ]]; then
  bold "==> tagging $VERSION"
  git tag -a "$VERSION" -m "lazytree $VERSION"
fi

# ---------------------------------------------------------------- 4. files
bold "==> building release files into ./dist"
GORELEASER="go run github.com/goreleaser/goreleaser/v2@latest"
if [[ $DRY -eq 0 ]]; then
  # The tag now sits on HEAD, so goreleaser versions the binaries from it.
  # validate is skipped because Obsidian's own files may be dirty.
  $GORELEASER release --clean --skip=publish,announce,validate
else
  $GORELEASER release --snapshot --clean --skip=publish
fi

# ---------------------------------------------------------------- 5. notes
if [[ $DRY -eq 0 ]]; then
  prev="$(git describe --tags --abbrev=0 --match 'v*' "$VERSION^" 2>/dev/null || true)"
  head_ref="$VERSION"
else
  prev="$(git describe --tags --abbrev=0 --match 'v*' 2>/dev/null || true)"
  head_ref="HEAD"
fi
range="$head_ref"
[[ -n "$prev" ]] && range="$prev..$head_ref"

notes="dist/RELEASE_NOTES.md"
{
  echo "## lazytree $VERSION"
  echo
  python3 - "$range" <<'PY'
import re, subprocess, sys

rng = sys.argv[1]
log = subprocess.run(
    ["git", "log", "--no-merges", "--reverse", "--pretty=%s", rng],
    capture_output=True, text=True, check=True,
).stdout.splitlines()

SECTIONS = [
    ("feat", "Features"),
    ("fix", "Fixes"),
    ("perf", "Performance"),
    ("refactor", "Internal improvements"),
]
skip = {"chore", "docs", "test", "ci", "build", "style"}
groups = {k: [] for k, _ in SECTIONS}
other = []
pat = re.compile(r"^(\w+)(?:\(([^)]*)\))?!?:\s*(.+)$")
for subject in log:
    m = pat.match(subject)
    if not m:
        if subject.strip() and not subject.startswith("Initial commit"):
            other.append(subject.strip())
        continue
    kind, scope, text = m.groups()
    if kind in skip:
        continue
    entry = f"**{scope}:** {text}" if scope else text
    (groups.get(kind) if kind in groups else other).append(entry if kind in groups else subject)

printed = False
for key, title in SECTIONS:
    if groups[key]:
        printed = True
        print(f"### {title}")
        for e in groups[key]:
            print(f"- {e}")
        print()
if other:
    printed = True
    print("### Other changes")
    for e in other:
        print(f"- {e}")
    print()
if not printed:
    print("Initial release.\n")
PY
  echo "### Install"
  echo
  echo "Download the archive for your system (Linux or macOS, amd64 or arm64), unpack it, and put \`lazytree\` on your \`PATH\`. Verify with \`checksums.txt\`. Requires \`git\`."
  echo
  echo '```bash'
  echo "go install github.com/eurico-martins/lazytree/cmd/lazytree@$VERSION"
  echo '```'
  if [[ -n "$prev" ]]; then
    echo
    echo "**Full changelog:** https://github.com/eurico-martins/lazytree/compare/$prev...$VERSION"
  fi
} > "$notes"

# ---------------------------------------------------------------- 6. summary
echo
bold "================ RELEASE NOTES (also saved to $notes) ================"
cat "$notes"
bold "======================================================================"
echo
bold "Files to upload (in ./dist):"
ls -1 dist/*.tar.gz dist/checksums.txt
echo
if [[ $DRY -eq 1 ]]; then
  bold "Dry run finished. Nothing was tagged, pushed or published."
else
  bold "Done. Nothing was pushed or published. Your manual steps:"
  cat <<STEPS
  1. Push the tag:        git push origin $VERSION
  2. On GitHub: Releases -> Draft a new release -> choose tag $VERSION
  3. Paste the text from  $notes
  4. Attach the files listed above
  To undo before pushing: git tag -d $VERSION
STEPS
fi
