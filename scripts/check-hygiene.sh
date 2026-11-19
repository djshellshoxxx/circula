#!/usr/bin/env bash
# Static repository checks:
#  * QA-043: no real-looking personal data in tests, fixtures or e2e (email addresses must use a reserved domain,
#    no phone-number or IBAN-like patterns).
#  * QA-037: migrations are numbered 0001.. without gaps or duplicates, and never edited destructively
#    (no destructive statements or append-only updates without a reviewed
#    `-- allow-destructive: reason` comment on a statement line).
#  * Generated root executables from `go build ./cmd/circula` are not tracked.
set -euo pipefail
cd "$(dirname "$0")/.."
fail=0

# Build output belongs outside source control. Check the index, not the working
# tree, so a developer's ignored local build does not fail repository hygiene.
artifacts="$(git ls-files -- circula circula.exe | grep -E '^(circula|circula\.exe)$' || true)"
if [ -n "$artifacts" ]; then
  echo "hygiene: generated root executables must not be tracked:"
  echo "$artifacts"
  fail=1
fi

# --- QA-043 -----------------------------------------------------------------------------------------------------
files="$(git ls-files 'e2e/*.js' 'internal/**/*_test.go' 'internal/testutil/*.go' 'docs/qa/*' | grep -v package-lock || true)"
emails="$(echo "$files" | xargs grep -HEoh '[A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+\.[A-Za-z]{2,}' 2>/dev/null | sort -u || true)"
badmail="$(echo "$emails" | grep -viE '@([a-z0-9.-]+\.)?(example(\.(org|com|net))?|invalid|test|localhost)$|@(noreply|users\.noreply)' | grep -vE '^$' || true)"
if [ -n "$badmail" ]; then echo "QA-043: email addresses outside reserved domains:"; echo "$badmail"; fail=1; fi
phones="$(echo "$files" | xargs grep -HnE '(\+[0-9]{1,3}[ -]?)?\(?[0-9]{3}\)?[ -][0-9]{3}[ -][0-9]{4}' 2>/dev/null | grep -v 'ledger\|time\|uuid' || true)"
if [ -n "$phones" ]; then echo "QA-043: phone-number-like strings:"; echo "$phones"; fail=1; fi

# --- QA-037 -----------------------------------------------------------------------------------------------------
n=0
for f in $(ls internal/store/migrations/*.sql | sort); do
  n=$((n+1)); want="$(printf '%04d_' "$n")"
  case "$(basename "$f")" in "$want"*) ;; *) echo "QA-037: expected migration $want*, found $(basename "$f")"; fail=1 ;; esac
done
if ! python3 scripts/check_migrations.py internal/store/migrations/*.sql; then
  fail=1
fi
if ! python3 -m unittest discover -s scripts -p 'test_check_migrations.py'; then
  fail=1
fi
# --- GOV-026 ----------------------------------------------------------------------------------------------------
# Each hosted platform policy exists and carries a version and a date.
for p in terms privacy operator-access suspension-categories incident-notification; do
  f="docs/platform/$p.md"
  if [ ! -f "$f" ] || ! grep -qE '^Version: .+' "$f" || ! grep -qE '^Date: [0-9]{4}-[0-9]{2}-[0-9]{2}$' "$f"; then
    echo "GOV-026: $f missing or without Version:/Date: lines"; fail=1
  fi
done
[ "$fail" -eq 0 ] && echo "hygiene checks: ok"
exit "$fail"
