#!/usr/bin/env bash
# Builds a synthetic ~108k-commit repository (a long main line with merged
# side branches) to check lazytree's large-repo behaviour. Needs git and
# python3; takes a few seconds.
#
# Usage:
#   scripts/big-repo.sh [dir]     # prints the repo path (default: a temp dir)
#
# Then, to measure load/scroll/search timings and heap use:
#   LAZYTREE_BIGREPO=<dir> go test ./internal/ui -run LargeRepo -v
# or just look at it:
#   ./lazytree <dir>
set -euo pipefail

DIR="${1:-$(mktemp -d "${TMPDIR:-/tmp}/lazytree-big.XXXXXX")}"
mkdir -p "$DIR"
git init -q -b main "$DIR"

python3 - <<'PY' | git -C "$DIR" fast-import --quiet
import sys
N = 100000
out = sys.stdout
t = 1700000000
mark = 0
mainl = None
side = None
for i in range(1, N + 1):
    mark += 1
    t += 60
    msg = f"commit {i}: change something in module {i % 97}"
    out.write(f"commit refs/heads/main\nmark :{mark}\ncommitter Dev <d@x.io> {t} +0000\ndata {len(msg)}\n{msg}\n")
    if mainl:
        out.write(f"from :{mainl}\n")
    if i % 12 == 0 and side:
        out.write(f"merge :{side}\n")
        side = None
    body = f"{i}\n"
    out.write(f"M 100644 inline f{i % 50}.txt\ndata {len(body)}\n{body}\n")
    mainl = mark
    if i % 10 == 0:
        mark += 1
        t += 1
        m2 = f"side {i}"
        out.write(f"commit refs/heads/side{i % 7}\nmark :{mark}\ncommitter Dev <d@x.io> {t} +0000\ndata {len(m2)}\n{m2}\nfrom :{mainl}\nM 100644 inline s.txt\ndata 2\nx\n\n")
        side = mark
PY

echo "$DIR"
