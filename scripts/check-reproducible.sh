#!/usr/bin/env bash
# Reproducible-build check: build the linux/amd64 binary twice, the second time from a copy of the tree in a
# different directory with a different build cache, using the release flags (CGO_ENABLED=0, -trimpath, version
# stamps from the commit). The two SHA-256 sums must match.
set -euo pipefail
cd "$(dirname "$0")/.."

GO="${GO:-go}"
commit="$(git rev-parse --short=12 HEAD)"
date="$(git log -1 --format=%cI)"
ldflags="-s -w -X main.version=repro -X main.commit=$commit -X main.date=$date"

work="$(mktemp -d)"
trap 'rm -rf "$work"' EXIT

build() { # dir out cache
  (cd "$1" && CGO_ENABLED=0 GOOS=linux GOARCH=amd64 GOFLAGS="-buildvcs=false" GOCACHE="$3" \
    "$GO" build -trimpath -ldflags "$ldflags" -o "$2" ./cmd/circula)
}

build "$PWD" "$work/a" "$work/cache-a"
mkdir "$work/tree"
tar --exclude=./bin --exclude=./.git --exclude=./e2e/node_modules -cf - . | tar -xf - -C "$work/tree"
build "$work/tree" "$work/b" "$work/cache-b"

sa="$(sha256sum "$work/a" | cut -d' ' -f1)"
sb="$(sha256sum "$work/b" | cut -d' ' -f1)"
echo "build 1: $sa"
echo "build 2: $sb"
if [ "$sa" != "$sb" ]; then
  echo "build is not reproducible"
  exit 1
fi
echo "reproducible build: ok"
