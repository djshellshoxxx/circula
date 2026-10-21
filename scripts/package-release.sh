#!/usr/bin/env bash
# Builds circula for every OPS-003 target and packages each one with the README
# and docs/ops, then writes SHA256SUMS and the Go module list (partial SBOM).
#
#   VERSION=v1.2.3 OUT=bin/dist scripts/package-release.sh
#
# Binaries are built with CGO_ENABLED=0 and -trimpath. The version, commit and
# commit date are injected with -ldflags.
set -euo pipefail

cd "$(dirname "$0")/.."

VERSION="${VERSION:?set VERSION, for example v1.2.3}"
OUT="${OUT:-bin/dist}"
GO="${GO:-go}"

if ! [[ "$VERSION" =~ ^v?[0-9]+\.[0-9]+\.[0-9]+(-[0-9A-Za-z.-]+)?$|^ci-[0-9a-f]+$|^dev$ ]]; then
  echo "unexpected VERSION: $VERSION" >&2
  exit 1
fi

COMMIT="$(git rev-parse --short=12 HEAD 2>/dev/null || echo unknown)"
DATE="$(git log -1 --format=%cI 2>/dev/null || echo unknown)"
LDFLAGS="-s -w -X main.version=${VERSION} -X main.commit=${COMMIT} -X main.date=${DATE}"

TARGETS=(
  "windows amd64"
  "linux amd64"
  "linux arm64"
  "darwin amd64"
  "darwin arm64"
)

rm -rf "$OUT"
mkdir -p "$OUT/stage"
export CGO_ENABLED=0

for t in "${TARGETS[@]}"; do
  read -r goos goarch <<<"$t"
  name="circula_${VERSION}_${goos}_${goarch}"
  stage="$OUT/stage/$name"
  mkdir -p "$stage"
  bin="circula"
  [ "$goos" = "windows" ] && bin="circula.exe"

  echo "building $name"
  GOOS="$goos" GOARCH="$goarch" "$GO" build -trimpath -ldflags "$LDFLAGS" -o "$stage/$bin" ./cmd/circula
  cp README.md "$stage/README.md"
  cp -r docs/ops "$stage/ops"

  if [ "$goos" = "windows" ]; then
    (cd "$OUT/stage" && zip -r -q -X "../$name.zip" "$name")
  else
    tar -C "$OUT/stage" --sort=name --owner=0 --group=0 --numeric-owner -czf "$OUT/$name.tar.gz" "$name"
  fi
  rm -rf "$stage"
done
rm -rf "$OUT/stage"

# Packaging must not carry secrets or personal data (OPS-035). A simple name scan.
for archive in "$OUT"/*.zip "$OUT"/*.tar.gz; do
  [ -e "$archive" ] || continue
  if tar -tzf "$archive" 2>/dev/null | grep -Eiq '\.db$|\.env$|secret|\.lock$|\.partial$'; then
    echo "archive $archive contains a forbidden file name" >&2
    exit 1
  fi
done

"$GO" list -m all > "$OUT/go-modules-${VERSION}.txt"

cd "$OUT"
if command -v sha256sum >/dev/null 2>&1; then
  sha256sum ./*.zip ./*.tar.gz ./go-modules-*.txt > SHA256SUMS
else
  shasum -a 256 ./*.zip ./*.tar.gz ./go-modules-*.txt > SHA256SUMS
fi
echo "packaged into $OUT:"
ls -1 "$OUT"
