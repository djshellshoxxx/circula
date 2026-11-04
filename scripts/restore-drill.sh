#!/bin/sh
# Restore drill (OPS-021). Restores a backup into a scratch directory, verifies
# the ledger, times both steps and appends one JSON line to the drill log.
#
# usage: scripts/restore-drill.sh <backup-file> [drill-log]
#   CIRCULA_BIN   the circula binary (default: circula on PATH)
#   DRILL_DIR     scratch directory (default: a new temp directory, removed after)
# Exit status 0 only when the restore and the ledger verification both pass.
# The production database is never touched: the restore target is DRILL_DIR.
set -u

backup=${1:-}
log=${2:-restore-drills.jsonl}
bin=${CIRCULA_BIN:-circula}

if [ -z "$backup" ] || [ ! -f "$backup" ]; then
  echo "usage: $0 <backup-file> [drill-log]" >&2
  exit 2
fi

own_dir=0
if [ -z "${DRILL_DIR:-}" ]; then
  DRILL_DIR=$(mktemp -d) || exit 2
  own_dir=1
fi
mkdir -p "$DRILL_DIR" || exit 2
db="$DRILL_DIR/circula.db"
cleanup() { [ "$own_dir" = 1 ] && rm -rf "$DRILL_DIR"; }
trap cleanup EXIT

now() { date +%s; }

t0=$(now)
restore_status=ok
"$bin" restore -database "$db" "$backup" >"$DRILL_DIR/restore.out" 2>&1 || restore_status=failed
t1=$(now)
verify_status=skipped
if [ "$restore_status" = ok ]; then
  verify_status=ok
  "$bin" verify-ledger -database "$db" >"$DRILL_DIR/verify.out" 2>&1 || verify_status=failed
fi
t2=$(now)

printf '{"date":"%s","backup":"%s","restore":"%s","restore_seconds":%s,"ledger":"%s","verify_seconds":%s}\n' \
  "$(date -u +%Y-%m-%dT%H:%M:%SZ)" "$(basename "$backup")" "$restore_status" "$((t1 - t0))" \
  "$verify_status" "$((t2 - t1))" >>"$log"

cat "$DRILL_DIR/restore.out"
[ -f "$DRILL_DIR/verify.out" ] && cat "$DRILL_DIR/verify.out"
echo "drill: restore=$restore_status ledger=$verify_status (recorded in $log)"
[ "$restore_status" = ok ] && [ "$verify_status" = ok ]
