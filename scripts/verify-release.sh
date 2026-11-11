#!/usr/bin/env bash
# Release directory check (OPS-035, OPS-003, SEC-029): run by scripts/package-release.sh on its output, by the
# release workflow before publishing, and by scripts/release-gate.sh.
#
#   scripts/verify-release.sh [-s] [-n NOTES] DIST VERSION
#
# Checks that DIST holds exactly the required set for VERSION and nothing else: per OPS-003 target one archive
# (.zip for windows, .tar.gz otherwise) and one CycloneDX SBOM (circula_<v>_<os>_<arch>.cdx.json), plus
# SHA256SUMS listing each of them once with matching values; that each archive holds only its own top directory
# with the binary, README.md, ops/ and legal/; and that scripts/scan-release.sh finds nothing.
#   -s  also check each SBOM against the binary in its archive (hash, target, linked modules; needs Go)
#   -n  also check the release notes with scripts/check-release-notes.sh
# TARGETS (space-separated os/arch) narrows the expected set; default all five DEC-003 targets.
set -euo pipefail
cd "$(dirname "$0")/.."

sbomcheck=0; notes=""
while getopts "sn:" o; do
  case "$o" in s) sbomcheck=1 ;; n) notes="$OPTARG" ;; *) exit 2 ;; esac
done
shift $((OPTIND - 1))
[ "$#" -eq 2 ] || { echo "usage: scripts/verify-release.sh [-s] [-n NOTES] DIST VERSION" >&2; exit 2; }
dist="$1"; version="$2"
TARGETS="${TARGETS:-windows/amd64 linux/amd64 linux/arm64 darwin/amd64 darwin/arm64}"
GO="${GO:-go}"
[ -d "$dist" ] || { echo "verify-release: $dist is not a directory" >&2; exit 1; }

fail=0
bad() { echo "verify-release: $*" >&2; fail=1; }
work="$(mktemp -d)"
trap 'rm -rf "$work"' EXIT

# 1. Contents list: exactly the required set.
archives=(); expected=()
for t in $TARGETS; do
  goos="${t%/*}"; goarch="${t#*/}"
  name="circula_${version}_${goos}_${goarch}"
  if [ "$goos" = windows ]; then a="$name.zip"; else a="$name.tar.gz"; fi
  archives+=("$a")
  expected+=("$a" "$name.cdx.json")
done
printf '%s\n' "${expected[@]}" SHA256SUMS | LC_ALL=C sort >"$work/want"
(cd "$dist" && find . -mindepth 1 -maxdepth 1 | sed 's|^\./||' | LC_ALL=C sort) >"$work/have"
missing="$(LC_ALL=C comm -23 "$work/want" "$work/have")"
extra="$(LC_ALL=C comm -13 "$work/want" "$work/have")"
[ -z "$missing" ] || bad "missing from $dist: $(echo "$missing" | tr '\n' ' ')"
[ -z "$extra" ] || bad "not part of the release set in $dist: $(echo "$extra" | tr '\n' ' ')"

# 2. SHA256SUMS lists every archive and SBOM exactly once, and every value matches.
if [ -f "$dist/SHA256SUMS" ]; then
  sed -E 's/^[0-9a-f]{64}  (\*|\.\/)?//' "$dist/SHA256SUMS" | LC_ALL=C sort >"$work/listed"
  printf '%s\n' "${expected[@]}" | LC_ALL=C sort >"$work/want-sums"
  if ! cmp -s "$work/listed" "$work/want-sums"; then
    bad "SHA256SUMS does not list exactly the archives and SBOMs: $(LC_ALL=C diff "$work/want-sums" "$work/listed" | grep '^[<>]' | tr '\n' ' ')"
  fi
  if grep -vqE '^[0-9a-f]{64}  ' "$dist/SHA256SUMS"; then bad "SHA256SUMS has a malformed line"; fi
  if command -v sha256sum >/dev/null 2>&1; then sumcmd=(sha256sum -c); else sumcmd=(shasum -a 256 -c); fi
  (cd "$dist" && "${sumcmd[@]}" SHA256SUMS >/dev/null) || bad "checksums in SHA256SUMS do not match the files"
fi

# 3. Archive layout, 4. content scan, 5. SBOM against the binary it describes.
present=()
for a in "${archives[@]}"; do
  [ -f "$dist/$a" ] && present+=("$dist/$a")
done
if [ "$sbomcheck" -eq 1 ] && [ "${#present[@]}" -gt 0 ]; then
  GOOS="" GOARCH="" CGO_ENABLED=0 "$GO" build -o "$work/sbom-tool" ./scripts/sbom
fi
for path in ${present[@]+"${present[@]}"}; do
  a="${path##*/}"
  case "$a" in *.zip) name="${a%.zip}"; bin="circula.exe" ;; *) name="${a%.tar.gz}"; bin="circula" ;; esac
  x="$work/x-$name"
  mkdir -p "$x"
  case "$a" in
    *.zip) if command -v unzip >/dev/null 2>&1; then unzip -q "$path" -d "$x"; else python3 -m zipfile -e "$path" "$x"; fi ;;
    *) tar -xzf "$path" -C "$x" ;;
  esac
  others="$(cd "$x" && find . -mindepth 1 -maxdepth 1 ! -name "$name" | sed 's|^\./||')"
  [ -z "$others" ] || bad "$a: entries outside $name/: $(echo "$others" | tr '\n' ' ')"
  for need in "$bin" README.md ops legal; do
    [ -e "$x/$name/$need" ] || bad "$a: $name/$need is missing"
  done
  if [ "$sbomcheck" -eq 1 ] && [ -f "$x/$name/$bin" ] && [ -f "$dist/$name.cdx.json" ]; then
    "$work/sbom-tool" -bin "$x/$name/$bin" -verify "$dist/$name.cdx.json" || bad "$name.cdx.json does not describe the binary in $a"
  fi
done
if [ "${#present[@]}" -gt 0 ]; then
  scripts/scan-release.sh "${present[@]}" || bad "archive scan found secrets or personal data (OPS-035)"
fi

if [ -n "$notes" ]; then
  scripts/check-release-notes.sh "$notes" "$version" || bad "release notes check failed (OPS-026)"
fi

[ "$fail" -eq 0 ] || exit 1
echo "verify-release: $dist holds the complete $version release set (${#archives[@]} targets)"
