#!/usr/bin/env bash
# Release gate evidence (QA-039). Runs the automated gate checks and writes an evidence file.
# Does not tag, publish or push anything.
#
# Usage: scripts/release-gate.sh [-o EVIDENCE_DIR] [-s check,check]   (-s skips named checks; they are recorded SKIPPED)
# Checks: fmt vet race lint contract golden e2e load vuln pins hygiene drill defects thresholds
# Result per check: PASS, FAIL, SKIPPED (not run, reason given), MANUAL (needs a human record; never auto-passes).
# Exit 0 only when nothing is FAIL. SKIPPED/MANUAL are listed so a reader cannot mistake them for a pass.
set -uo pipefail
cd "$(dirname "$0")/.."

out="docs/qa/evidence"; skip=","
while getopts "o:s:" o; do
  case "$o" in o) out="$OPTARG" ;; s) skip=",$OPTARG," ;; *) exit 2 ;; esac
done
mkdir -p "$out"
stamp="$(date -u +%Y%m%dT%H%M%SZ)"
file="$out/gate-$stamp.md"
log="$(mktemp -d)"; trap 'rm -rf "$log"' EXIT
rows=""; fails=0

record() { # name result note
  rows+="| $1 | $2 | $3 |"$'\n'
  [ "$2" = FAIL ] && fails=$((fails+1))
  echo "[$2] $1 $3"
}
run() { # name command...  (output saved; last 3 lines in note on failure)
  local name="$1"; shift
  case "$skip" in *",$name,"*) record "$name" SKIPPED "skipped by -s"; return ;; esac
  local t0=$SECONDS
  if "$@" >"$log/$name.log" 2>&1; then
    record "$name" PASS "$((SECONDS-t0)) s"
  else
    record "$name" FAIL "$(tail -n 3 "$log/$name.log" | tr '\n|' ' /' | cut -c1-200)"
  fi
}
fmtcheck() { [ -z "$(gofmt -l .)" ] || { gofmt -l .; return 1; }; }
lintrun() { PATH="$HOME/go/bin:$PATH" command -v golangci-lint >/dev/null && PATH="$HOME/go/bin:$PATH" golangci-lint run; }
vulncheck() { go run golang.org/x/vuln/cmd/govulncheck@v1.1.4 ./...; }
loadsmoke() { CIRCULA_LOAD=1 CIRCULA_LOAD_WORKERS=4 CIRCULA_LOAD_MEMBERS=20 CIRCULA_LOAD_TRADES=25 go test -count=1 -run LoadConcurrentTrades ./internal/service; }
e2erun() { make e2e; }
raced() { go test -race -count=1 ./...; }
drillrun() { scripts/restore-drill-full.sh -o "$out"; }

run fmt fmtcheck
run vet go vet ./...
run race raced
run lint lintrun
run contract go test -count=1 -run TestAPIMatchesOpenAPI ./internal/contract
run golden go test -count=1 -run Golden ./internal/store
run e2e e2erun
run load loadsmoke
run vuln vulncheck
run pins scripts/check-action-pins.sh
run hygiene scripts/check-hygiene.sh
run drill drillrun

# Documentation checks.
open_blockers="$(grep -cE '^\| DEF-[0-9]+ .*\| A-block \| open ' docs/qa/DEFECTS.md || true)"
if [ "${open_blockers:-0}" -eq 0 ]; then record defects PASS "no open A-block defects"; else record defects FAIL "$open_blockers open A-block defects"; fi
if grep -q UNCONFIRMED docs/qa/THRESHOLDS.md; then record thresholds FAIL "DEC-023 values unconfirmed (QA-040)"; else record thresholds PASS "confirmed"; fi

# Items that need a human or another environment. Never auto-pass.
record screen-reader MANUAL "docs/qa/SCREEN-READER.md script; link the signed run record"
record exploratory MANUAL "L14 session records"
record security-review MANUAL "SEC-033 independent review record"
record run-smoke MANUAL "L13 smoke on each DEC-003 target, or documented reason"
record release-notes MANUAL "docs/releases/<version>.md (OPS-026)"

{
  echo "# Release gate evidence"
  echo
  echo "* Generated (UTC): $stamp"
  echo "* Commit: $(git rev-parse HEAD 2>/dev/null || echo unknown)"
  echo "* Working tree: $([ -z "$(git status --porcelain 2>/dev/null)" ] && echo clean || echo DIRTY)"
  echo "* Go: $(go version)"
  echo "* Host: $(uname -srm)"
  echo "* FAIL count: $fails"
  echo
  echo "| Check | Result | Note |"
  echo "|---|---|---|"
  printf '%s' "$rows"
  echo
  echo "PASS means the command exited 0 on this host. SKIPPED and MANUAL are not passes. The gate passes only with zero FAIL"
  echo "and every MANUAL item linked from the release record."
} >"$file"
echo "evidence: $file (FAIL=$fails)"
[ "$fails" -eq 0 ]
