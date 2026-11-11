#!/usr/bin/env bash
# Smoke test: run the binary with the same settings as deploy/circula.service (minus systemd) in a temp dir,
# check /healthz and /readyz, then stop it. Usage: scripts/smoke-deploy.sh [path-to-binary]
# Without an argument it builds a static binary (CGO_ENABLED=0). Needs curl.
set -euo pipefail
cd "$(dirname "$0")/.."

bin="${1:-}"
if [ -z "$bin" ]; then
  mkdir -p bin
  CGO_ENABLED=0 go build -trimpath -o bin/circula-smoke ./cmd/circula
  bin="bin/circula-smoke"
fi
case "$bin" in /*) ;; *) bin="$PWD/$bin" ;; esac

work="$(mktemp -d)"
pid=""
cleanup() {
  if [ -n "$pid" ]; then kill "$pid" 2>/dev/null || true; fi
  rm -rf "$work"
}
trap cleanup EXIT

umask 0077 # UMask=0077
mkdir -p "$work/state/backups"
chmod 0700 "$work/state"
port="${SMOKE_PORT:-18080}"

# The unit must still start the binary this way; fail loudly if it drifts.
grep -q '^ExecStart=/usr/local/bin/circula serve$' deploy/circula.service \
  || { echo "deploy/circula.service ExecStart changed; update this script"; exit 1; }
for v in LISTEN PUBLIC_BASE_URL DATABASE BACKUP_DIR STATE_DIR TRUSTED_PROXIES; do
  grep -q "^Environment=CIRCULA_$v=" deploy/circula.service \
    || { echo "unit no longer sets CIRCULA_$v"; exit 1; }
done

# Same environment as the unit's Environment= lines, with paths and port swapped.
export CIRCULA_LISTEN="127.0.0.1:$port"
export CIRCULA_PUBLIC_BASE_URL="http://127.0.0.1:$port" # http on loopback is allowed; the unit uses https
export CIRCULA_DATABASE="$work/state/circula.db"
export CIRCULA_BACKUP_DIR="$work/state/backups"
export CIRCULA_STATE_DIR="$work/state"
export CIRCULA_TRUSTED_PROXIES="127.0.0.1/32"

"$bin" migrate >/dev/null
"$bin" serve >"$work/serve.log" 2>&1 &
pid=$!

ok=0
for _ in $(seq 1 50); do
  if curl -fsS "http://127.0.0.1:$port/readyz" >/dev/null 2>&1; then ok=1; break; fi
  kill -0 "$pid" 2>/dev/null || { echo "serve exited early:"; cat "$work/serve.log"; exit 1; }
  sleep 0.2
done
[ "$ok" -eq 1 ] || { echo "readyz never became ready:"; cat "$work/serve.log"; exit 1; }
curl -fsS "http://127.0.0.1:$port/healthz" >/dev/null

perm="$(stat -c %a "$work/state/circula.db")"
[ "$perm" = "600" ] || { echo "database mode is $perm, want 600"; exit 1; }

kill -TERM "$pid"
wait "$pid" || { echo "serve did not exit cleanly on SIGTERM"; exit 1; }
pid=""
echo "smoke-deploy: ok"
