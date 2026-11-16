#!/usr/bin/env bash
# Builds circula for every OPS-003 target and packages each one with the README, docs/ops and docs/legal, then
# writes one CycloneDX 1.5 SBOM per target (SEC-029) and SHA256SUMS over archives and SBOMs (OPS-035).
#
#   VERSION=v1.2.3 OUT=bin/dist scripts/package-release.sh
#   VERSION=v0.0.0-test OUT=/tmp/dist TARGETS="linux/amd64" scripts/package-release.sh   # subset, local checks
#
# Binaries are built with CGO_ENABLED=0 and -trimpath; the version, commit and commit date are injected with
# -ldflags. Module checksums are enforced: go.sum is read-only (-mod=readonly), a go.sum mismatch fails the build
# (scripts/test-gosum-tamper.sh proves it) and `go mod verify` checks the module cache after building. Finally
# scripts/verify-release.sh checks the finished directory: contents list, checksums, each SBOM against its
# binary, and the OPS-035 archive scan for secrets and personal data (scripts/scan-release.sh).
set -euo pipefail

cd "$(dirname "$0")/.."

VERSION="${VERSION:?set VERSION, for example v1.2.3}"
OUT="${OUT:-bin/dist}"
GO="${GO:-go}"
TARGETS="${TARGETS:-windows/amd64 linux/amd64 linux/arm64 darwin/amd64 darwin/arm64}"

if ! [[ "$VERSION" =~ ^v?[0-9]+\.[0-9]+\.[0-9]+(-[0-9A-Za-z.-]+)?$|^ci-[0-9a-f]+$|^dev$ ]]; then
  echo "unexpected VERSION: $VERSION" >&2
  exit 1
fi
for t in $TARGETS; do
  case "$t" in windows/amd64|linux/amd64|linux/arm64|darwin/amd64|darwin/arm64) ;;
    *) echo "unexpected target $t (OPS-003 targets only)" >&2; exit 1 ;; esac
done

COMMIT="$(git rev-parse --short=12 HEAD 2>/dev/null || echo unknown)"
DATE="$(git log -1 --format=%cI 2>/dev/null || echo unknown)"
LDFLAGS="-s -w -X main.version=${VERSION} -X main.commit=${COMMIT} -X main.date=${DATE}"

rm -rf "$OUT"
mkdir -p "$OUT/stage"
export CGO_ENABLED=0
# SEC-029: never let the build rewrite go.mod/go.sum; a checksum that does not match go.sum stops the build.
export GOFLAGS="-mod=readonly"

# The SBOM generator runs on the build host, not the target.
sbom="$OUT/stage/sbom-tool"
GOOS="" GOARCH="" "$GO" build -o "$sbom" ./scripts/sbom
"$GO" mod graph >"$OUT/stage/modgraph.txt"
modcache="$("$GO" env GOMODCACHE)"

for t in $TARGETS; do
  goos="${t%/*}"; goarch="${t#*/}"
  name="circula_${VERSION}_${goos}_${goarch}"
  stage="$OUT/stage/$name"
  mkdir -p "$stage"
  bin="circula"
  [ "$goos" = "windows" ] && bin="circula.exe"

  echo "building $name"
  GOOS="$goos" GOARCH="$goarch" "$GO" build -trimpath -ldflags "$LDFLAGS" -o "$stage/$bin" ./cmd/circula
  cp README.md "$stage/README.md"
  cp -r docs/ops "$stage/ops"
  cp -r docs/legal "$stage/legal"

  sbom_args=(-bin "$stage/$bin" -version "$VERSION" -modcache "$modcache" -graph "$OUT/stage/modgraph.txt"
    -licenses docs/legal/DEPENDENCIES.md -o "$OUT/$name.cdx.json")
  [ "$DATE" != unknown ] && sbom_args+=(-timestamp "$DATE") # commit time keeps the bill reproducible
  "$sbom" "${sbom_args[@]}"

  if [ "$goos" = "windows" ]; then
    (cd "$OUT/stage" && zip -r -q -X "../$name.zip" "$name")
  else
    tar -C "$OUT/stage" --sort=name --owner=0 --group=0 --numeric-owner -czf "$OUT/$name.tar.gz" "$name"
  fi
  rm -rf "$stage"
done
rm -rf "$OUT/stage"

# The builds downloaded or reused every linked module; check the module cache still matches go.sum.
"$GO" mod verify

(
  cd "$OUT"
  files=(circula_*)
  if command -v sha256sum >/dev/null 2>&1; then
    sha256sum "${files[@]/#/./}" > SHA256SUMS
  else
    shasum -a 256 "${files[@]/#/./}" > SHA256SUMS
  fi
)

# Contents list, checksums, SBOM against each binary, and the OPS-035 scan for secrets and personal data.
TARGETS="$TARGETS" scripts/verify-release.sh -s "$OUT" "$VERSION"
echo "packaged into $OUT:"
ls -1 "$OUT"
