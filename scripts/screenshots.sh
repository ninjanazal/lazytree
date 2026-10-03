#!/usr/bin/env bash
# Regenerates the README screenshots and demo GIF in docs/img/ by driving the
# real lazytree in a headless terminal (charmbracelet/vhs, in Docker).
#
#   scripts/screenshots.sh              # all tapes
#   scripts/screenshots.sh split-dark   # just docs/tapes/split-dark.tape
#
# Needs Docker. Everything happens locally; nothing is uploaded.
set -euo pipefail

cd "$(dirname "${BASH_SOURCE[0]}")/.."
ROOT="$PWD"

WORK="$(mktemp -d "${TMPDIR:-/tmp}/lazytree-shots.XXXXXX")"
trap 'rm -rf "$WORK"' EXIT

echo "==> building lazytree (linux/amd64, static)"
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o "$WORK/lazytree" ./cmd/lazytree

echo "==> generating the demo repository"
DEMO="$(./scripts/demo-repo.sh | tail -1)"
cp -r "$DEMO" "$WORK/repo"
rm -rf "$DEMO"

# Fixed configs so every run looks the same: forced theme, no background fetch.
for theme in dark light; do
  mkdir -p "$WORK/cfg-$theme/lazytree"
  printf 'theme = "%s"\nfetch_interval = "off"\n' "$theme" > "$WORK/cfg-$theme/lazytree/config.toml"
done

echo "==> building the recorder image"
docker build -q -f docker/Dockerfile.vhs -t lazytree-vhs . >/dev/null

mkdir -p docs/img
if [[ $# -gt 0 ]]; then tapes=("$@"); else
  tapes=(); for f in docs/tapes/*.tape; do tapes+=("$(basename "$f" .tape)"); done
fi

for t in "${tapes[@]}"; do
  echo "==> recording $t"
  docker run --rm \
    -v "$WORK:/work" \
    -v "$ROOT/docs:/vhs/docs" \
    --entrypoint sh lazytree-vhs \
    -c "vhs docs/tapes/$t.tape; rc=\$?; chown -R $(id -u):$(id -g) docs/img /work; exit \$rc"
done

echo "==> done: docs/img/"
ls -1 docs/img
