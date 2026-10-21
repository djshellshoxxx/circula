#!/usr/bin/env bash
# Restore drill (OPS-021, QA-029). Backs up a database, restores the backup into a separate directory, verifies the
# ledger on the restored copy and writes a drill record (JSON) naming the backup, duration and outcome.
#
# Usage: scripts/restore-drill.sh [-s SOURCE_DB] [-o RECORD_DIR]
#   SOURCE_DB   database to drill from (default: a fresh, migrated, empty database; weaker evidence, say so in the record)
#   RECORD_DIR  where drill-<UTC time>.json is written (default: docs/qa/evidence)
# Env: CIRCULA_BIN (default bin/circula, built if missing). Exit 0 only if every step passed.
set -euo pipefail
cd "$(dirname "$0")/.."

src=""; out="docs/qa/evidence"
while getopts "s:o:" o; do
  case "$o" in s) src="$OPTARG" ;; o) out="$OPTARG" ;; *) exit 2 ;; esac
done
bin="${CIRCULA_BIN:-bin/circula}"
[ -x "$bin" ] || make build >/dev/null
work="$(mktemp -d)"; trap 'rm -rf "$work"' EXIT
mkdir -p "$work/src" "$work/drill" "$out"

kind="existing database"
if [ -z "$src" ]; then
  kind="fresh migrated empty database (no ledger data)"
  src="$work/src/circula.db"
  CIRCULA_DATABASE="$src" CIRCULA_MIGRATE_MODE=auto "$bin" migrate >/dev/null
fi

now() { date +%s.%N; }
status=pass; detail=""
fail() { status=fail; detail="$1"; }

bk="$work/backups"; mkdir -p "$bk"
CIRCULA_DATABASE="$src" CIRCULA_BACKUP_DIR="$bk" "$bin" backup "$bk" >"$work/backup.log" 2>&1 || fail "backup failed"
backup_file="$(find "$bk" -maxdepth 1 -type f ! -name '*.json' | head -n1 || true)"
[ -n "$backup_file" ] || fail "no backup file produced"

dur=0
if [ "$status" = pass ]; then
  t0="$(now)"
  CIRCULA_DATABASE="$work/drill/circula.db" CIRCULA_BACKUP_DIR="$work/drill/bk" "$bin" restore "$backup_file" >"$work/restore.log" 2>&1 || fail "restore failed"
  dur="$(awk -v a="$t0" -v b="$(now)" 'BEGIN{printf "%.3f", b-a}')"
fi
if [ "$status" = pass ]; then
  CIRCULA_DATABASE="$work/drill/circula.db" CIRCULA_BACKUP_DIR="$work/drill/bk" "$bin" verify-ledger >"$work/verify.log" 2>&1 || fail "verify-ledger failed on restored copy"
fi

stamp="$(date -u +%Y%m%dT%H%M%SZ)"
rec="$out/drill-$stamp.json"
printf '{\n  "kind": "restore-drill",\n  "date_utc": "%s",\n  "commit": "%s",\n  "source": "%s",\n  "backup": "%s",\n  "restore_seconds": %.3f,\n  "ledger_verification": "%s",\n  "outcome": "%s",\n  "detail": "%s"\n}\n' \
  "$stamp" "$(git rev-parse --short=12 HEAD 2>/dev/null || echo unknown)" "$kind" "$(basename "${backup_file:-none}")" "$dur" \
  "$([ "$status" = pass ] && echo ok || echo failed)" "$status" "$detail" >"$rec"
echo "drill record: $rec ($status)"
[ "$status" = pass ]
