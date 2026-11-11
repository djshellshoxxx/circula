#!/usr/bin/env bash
# OPS-035 release archive scan: packaging must not carry configuration with secret values or test fixtures with
# personal data. Each archive (.tar.gz or .zip) is listed and extracted, then checked by name and by content:
#
#   names    database, backup, environment, key and lock files; anything named secret or credential; test
#            sources and fixture directories; absolute or parent-relative member paths
#   all      PEM private keys, well-known cloud and forge token formats, the secret canary marker
#   files    (circula-canary..., canary-<word>-<hex>) and the e2e fixture secrets read from e2e/*.js
#   text     secret-looking assignments (key=value with a long literal value), e-mail addresses outside the
#   files    reserved domains (as QA-043), phone-number-like and IBAN-like strings
#
#   scripts/scan-release.sh [-c canary]... ARCHIVE...      exit 0 clean, 1 on any finding, 2 on usage error
#
# -c adds a literal canary string that must not appear anywhere (binaries included). Findings name the archive,
# the member and the rule, never the matched secret.
set -euo pipefail
cd "$(dirname "$0")/.."

canaries=()
while getopts "c:" o; do
  case "$o" in c) canaries+=("$OPTARG") ;; *) exit 2 ;; esac
done
shift $((OPTIND - 1))
[ "$#" -gt 0 ] || { echo "usage: scripts/scan-release.sh [-c canary]... ARCHIVE..." >&2; exit 2; }

# Fixture secrets from the end-to-end suite: if any of these ships, test material was packaged.
for f in e2e/global-setup.js e2e/secfixture.js; do
  [ -f "$f" ] || continue
  while IFS= read -r v; do
    [ -n "$v" ] && canaries+=("$v")
  done < <(sed -nE "s/^const (PASSWORD|[A-Z_]*SECRET) = '([^']{8,})';.*$/\\2/p" "$f")
done

name_re='(^|/)\.env($|\.)|\.(db|db-wal|db-shm|db\.enc|sqlite|sqlite3|bak|partial|lock|pem|key|p12|pfx|jks|kdbx|har)$|\.manifest\.json$|(^|/)id_(rsa|dsa|ecdsa|ed25519)|secret|credential|_test\.go$|(^|/)(testdata|fixtures?|e2e|\.state)(/|$)'
key_re='-----BEGIN ([A-Z0-9]+ )*PRIVATE KEY( BLOCK)?-----'
token_re='AKIA[0-9A-Z]{16}|gh[pousr]_[A-Za-z0-9]{36}|github_pat_[A-Za-z0-9_]{40,}|xox[abposr]-[0-9A-Za-z-]{10,}|sk_live_[0-9A-Za-z]{16,}|AIza[0-9A-Za-z_-]{35}|glpat-[0-9A-Za-z_-]{20}'
canary_re='[Cc][Ii][Rr][Cc][Uu][Ll][Aa][-_][Cc][Aa][Nn][Aa][Rr][Yy]|canary-[a-z0-9]+-[0-9a-f]{8,}'
assign_re='(secret|password|passwd|api[_-]?key|token|encryption[_-]?key|backup[_-]?key|private[_-]?key)["'"'"']?[[:space:]]*[:=][[:space:]]*["'"'"']?[A-Za-z0-9+_=-][A-Za-z0-9+/_=-]{15,}'
email_re='[A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+\.[A-Za-z]{2,}'
email_ok_re='@([a-z0-9.-]+\.)?(example(\.(org|com|net))?|invalid|test|localhost)$|@(noreply|users\.noreply)'
phone_re='(\+[0-9]{1,3}[ -]?)?\(?[0-9]{3}\)?[ -][0-9]{3}[ -][0-9]{4}'
iban_re='(^|[^A-Za-z0-9])[A-Z]{2}[0-9]{2}( ?[A-Z0-9]{4}){3,7}( ?[A-Z0-9]{1,3})?([^A-Za-z0-9]|$)'

work="$(mktemp -d)"
trap 'rm -rf "$work"' EXIT
findings=0
finding() { # archive member rule
  echo "scan-release: $1: $2: $3" >&2
  findings=$((findings + 1))
}

list_zip() {
  if command -v unzip >/dev/null 2>&1; then unzip -Z1 "$1"; else python3 -c 'import sys,zipfile; print("\n".join(zipfile.ZipFile(sys.argv[1]).namelist()))' "$1"; fi
}
extract_zip() {
  if command -v unzip >/dev/null 2>&1; then unzip -q -o "$1" -d "$2"; else python3 -m zipfile -e "$1" "$2"; fi
}

n=0
for archive in "$@"; do
  [ -f "$archive" ] || { echo "scan-release: $archive: not a file" >&2; exit 2; }
  n=$((n + 1))
  dir="$work/$n"
  mkdir -p "$dir"
  case "$archive" in
    *.tar.gz|*.tgz) names="$(tar -tzf "$archive")"; tar -xzf "$archive" -C "$dir" ;;
    *.zip) names="$(list_zip "$archive")"; extract_zip "$archive" "$dir" ;;
    *) echo "scan-release: $archive: not a .tar.gz or .zip archive" >&2; exit 2 ;;
  esac
  [ -n "$names" ] || finding "$archive" "-" "archive is empty"

  while IFS= read -r m; do
    [ -n "$m" ] || continue
    case "$m" in /*|../*|*/../*|..) finding "$archive" "$m" "member path escapes the archive root" ;; esac
    if printf '%s\n' "$m" | grep -Eiq "$name_re"; then finding "$archive" "$m" "forbidden file name"; fi
  done <<<"$names"

  # Every regular file, binaries included: high-signal secret material only.
  while IFS= read -r f; do
    m="${f#"$dir"/}"
    if grep -aEq -- "$key_re" "$f"; then finding "$archive" "$m" "private key block"; fi
    if grep -aEq -- "$token_re" "$f"; then finding "$archive" "$m" "access token format"; fi
    if grep -aEq -- "$canary_re" "$f"; then finding "$archive" "$m" "secret canary marker"; fi
    for c in ${canaries[@]+"${canaries[@]}"}; do
      if grep -aFq -- "$c" "$f"; then finding "$archive" "$m" "fixture secret or canary"; fi
    done
  done < <(find "$dir" -type f | LC_ALL=C sort)

  # Text files: secret assignments and personal-data patterns. grep -I skips binary files.
  while IFS= read -r f; do
    m="${f#"$dir"/}"
    if grep -IEiq -- "$assign_re" "$f"; then finding "$archive" "$m" "secret value in configuration"; fi
    bad="$(grep -IEoh -- "$email_re" "$f" | grep -viE "$email_ok_re" || true)"
    if [ -n "$bad" ]; then finding "$archive" "$m" "e-mail address outside reserved domains"; fi
    if grep -IEq -- "$phone_re" "$f"; then finding "$archive" "$m" "phone-number-like string"; fi
    if grep -IEq -- "$iban_re" "$f"; then finding "$archive" "$m" "IBAN-like string"; fi
  done < <(grep -rlI '' "$dir" | LC_ALL=C sort)
done

if [ "$findings" -ne 0 ]; then
  echo "scan-release: $findings finding(s) in $# archive(s)" >&2
  exit 1
fi
echo "scan-release: $# archive(s) clean"
