#!/usr/bin/env bash
# Release record check (OPS-026, QA-039, QA-044) for docs/releases/<version>.md, written from
# docs/releases/TEMPLATE.md. Run by the release workflow before anything is built and by scripts/release-gate.sh -r.
#
#   scripts/check-release-notes.sh NOTES VERSION
#
# Fails unless:
#   * the first heading names VERSION and no template text is left (vX.Y.Z, [bracketed placeholders]);
#   * each OPS-026 section exists exactly once and is not empty (write "none" when nothing applies):
#     Breaking changes, Schema changes, Configuration changes, API changes, Security fixes, Required operator actions;
#   * Schema changes is "none" or states "Backup required before upgrade: yes" (or "no");
#   * an Evidence section links docs/qa/evidence/VERSION/ and that directory exists (QA-044).
# HTML comments (<!-- ... -->) are ignored, so template guidance can stay in a copied file.
# EVIDENCE_ROOT overrides docs/qa/evidence (tests only).
set -euo pipefail
cd "$(dirname "$0")/.."

[ "$#" -eq 2 ] || { echo "usage: scripts/check-release-notes.sh NOTES VERSION" >&2; exit 2; }
notes="$1"; version="$2"
evidence_root="${EVIDENCE_ROOT:-docs/qa/evidence}"
fail=0
bad() { echo "release notes $notes: $*" >&2; fail=1; }

[ -s "$notes" ] || { echo "release notes $notes: missing or empty; write it from docs/releases/TEMPLATE.md" >&2; exit 1; }

# Strip HTML comments, including ones that span lines.
body="$(awk '
  { line = $0; out = "" }
  { while (1) {
      if (incomment) { i = index(line, "-->"); if (i == 0) { line = ""; break }; line = substr(line, i + 3); incomment = 0 }
      i = index(line, "<!--"); if (i == 0) { out = out line; break }
      out = out substr(line, 1, i - 1); line = substr(line, i + 4); incomment = 1
    }
    print out }' "$notes")"

first="$(printf '%s\n' "$body" | grep -m1 -E '^# ' || true)"
case "$first" in *" $version"|*" $version "*) ;; *) bad "first heading must name $version, found '${first:-none}'" ;; esac
if printf '%s\n' "$body" | grep -q 'vX\.Y\.Z'; then bad "template text vX.Y.Z is left in"; fi
if printf '%s\n' "$body" | grep -nE '\[[^]]+\]([^(]|$)' >&2; then bad "bracketed template placeholders are left in (lines above)"; fi

section() { # heading -> body lines without blanks, up to the next ## heading
  printf '%s\n' "$body" | awk -v h="## $1" '
    $0 == h { on = 1; next }
    on && /^## / { exit }
    on && NF { print }'
}
count() { printf '%s\n' "$body" | grep -cxF "## $1" || true; }

for s in "Breaking changes" "Schema changes" "Configuration changes" "API changes" "Security fixes" \
  "Required operator actions" "Evidence"; do
  n="$(count "$s")"
  if [ "$n" -ne 1 ]; then bad "section '## $s' appears $n times, want exactly once"; continue; fi
  [ -n "$(section "$s")" ] || bad "section '## $s' is empty; write \"none\" when nothing applies"
done

schema="$(section "Schema changes")"
if [ -n "$schema" ]; then
  plain="$(printf '%s\n' "$schema" | sed -E 's/^[[:space:]]*[-*][[:space:]]*//; s/[[:space:].]+$//' | tr '[:upper:]' '[:lower:]')"
  if [ "$plain" != "none" ] && ! printf '%s\n' "$schema" | grep -qiE 'backup required before upgrade:[[:space:]]*[*_]*(yes|no)([^a-z]|$)'; then
    bad "Schema changes must be \"none\" or state \"Backup required before upgrade: yes\" (or no) (OPS-026)"
  fi
fi

esc_version="$(printf '%s' "$version" | sed 's/[.]/\\./g')"
if ! section "Evidence" | grep -qE "qa/evidence/${esc_version}/"; then
  bad "Evidence must link docs/qa/evidence/$version/ (QA-044)"
fi
[ -d "$evidence_root/$version" ] || bad "evidence directory $evidence_root/$version/ does not exist (QA-044, docs/qa/EVIDENCE.md)"

[ "$fail" -eq 0 ] || exit 1
echo "release notes $notes: ok"
