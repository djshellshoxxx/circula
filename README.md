# Circula

Circula is a self-hosted credit-ledger service for trading networks. Members of a network propose, accept and settle
trades that move credit between their accounts. Every posting is an append-only double entry, every command writes an
audit event, and a verification operation recomputes the ledger invariants.

This repository contains the Go core (domain, service layer, SQLite store) and the `circula` command, which runs the
HTTP service, the web UI and the JSON API, and operates the database.

- Specifications: [`docs/specs/`](docs/specs/README.md)
- Operations (configuration, backup and restore, upgrades, monitoring): [`docs/ops/`](docs/ops/README.md)
- Developer notes: [`docs/dev/SERVICE.md`](docs/dev/SERVICE.md)
- Contributing: [`CONTRIBUTING.md`](CONTRIBUTING.md). Security reports: [`SECURITY.md`](SECURITY.md).

## Quickstart

Requirements: Go 1.24 or newer. No C compiler is needed (the SQLite driver is pure Go).

```sh
make build                       # writes bin/circula

export CIRCULA_PUBLIC_BASE_URL=http://127.0.0.1:8080   # the address users open (https:// for anything but loopback)
export CIRCULA_COOKIE_SECURE=false                      # local http only; keep the default (true) behind TLS
export CIRCULA_SETUP_SECRET="$(head -c 24 /dev/urandom | base64)"   # asked for when the first network is created
export CIRCULA_DATABASE="$PWD/data/circula.db"          # created on first use

mkdir -p data
./bin/circula migrate                                   # back up (if any), migrate, verify
printf '%s\n' 'a-long-operator-password' | ./bin/circula admin create-operator -login ops-admin
./bin/circula serve
```

Then check the instance:

```sh
curl -s http://127.0.0.1:8080/healthz    # {"status":"ok","version":"..."}
curl -s http://127.0.0.1:8080/readyz     # {"status":"ready"}
curl -s http://127.0.0.1:8080/metrics    # only from loopback or the ranges in CIRCULA_METRICS_ALLOW
```

`serve` listens on `127.0.0.1:8080` unless you set `CIRCULA_LISTEN` or `-listen`. Put a TLS-terminating reverse proxy in
front of it for any other use, set `trusted-proxies` to that proxy, and do not forward `/metrics` through it. `serve`
refuses an `http://` public URL on a non-loopback address. Secrets (setup secret, metrics token, backup key) come from the
environment or a 0600 file, never from a flag: see [`docs/ops/configuration.md`](docs/ops/configuration.md#secrets).

## Features

- Double-entry credit ledger per network, hash-chained audit log, ledger verification.
- Web UI and JSON API (scoped API tokens, idempotency keys, keyset paging).
- Accounts: TOTP multi-factor sign-in with recovery codes, account recovery, security notices, password policy with a common-password blocklist.
- Governance: dual-control approval of large adjustments, time-limited break-glass access, member erasure, administrator and operator password reset.
- Abuse limits: sign-in lock keyed by login and client address (with a login-wide cap and `circula unlock`), per-address rate limits (IPv6 by /64), `__Host-` cookies over HTTPS.
- Operations: encrypted, signed backups (restore refuses unsigned backups unless `-allow-unsigned`), forward-only migrations, optional `db-synchronous=full`.
- Quality: fuzz, property and stress tests, RFC 4226/6238 test vectors, audits in `docs/audit/`, research notes in `docs/research/`.

## Commands

| Command | Purpose |
|---|---|
| `circula serve` | Run the service. `-check-config` validates settings and prints the effective values without touching the database. |
| `circula migrate [-dry-run]` | Take a backup if the database has data and migrations are pending, apply them, then verify every network. |
| `circula backup <dest>` | Write a verified online snapshot and a manifest. `dest` is a file, or a directory for a timestamped name. |
| `circula restore [-overwrite] <backup-file>` | Restore a backup into the configured database. Stop `serve` first. |
| `circula verify-ledger [-network id-or-slug]` | Recompute the ledger invariants for each network. |
| `circula admin create-operator -login <name>` | Create a person. The password comes from `CIRCULA_OPERATOR_PASSWORD` or the first line of stdin, never from the command line. |
| `circula unlock -login <name>` | Clear sign-in, recovery and MFA lockouts for a login (audited). |
| `circula version` | Print the version, commit and build date. |

Exit codes: `0` success, `1` failure, `2` usage or configuration error, `3` the operation completed but a ledger
verification failed.

Configuration is read from flags, then `CIRCULA_*` environment variables, then a JSON file given by `-config`
(or `CIRCULA_CONFIG`), then built-in defaults. Every setting is listed in [`docs/ops/configuration.md`](docs/ops/configuration.md).

## Build and test

```sh
make check          # gofmt, vet, race tests, golangci-lint, govulncheck, workflow checks
make release-local VERSION=v0.1.0   # all five release targets into bin/dist
```

Release artifacts are built by [`.github/workflows/release.yml`](.github/workflows/release.yml) from a `v*` tag. Each
archive contains the binary, this README and `ops/`, and `SHA256SUMS` lists every archive.

## Status and limits

- Platforms: windows/amd64, linux/amd64, linux/arm64, darwin/amd64, darwin/arm64 (`CGO_ENABLED=0`).
- One process per database file. Multi-node operation is out of scope for the MVP.
- No licence file is included yet; the owner has not chosen a licence. Do not redistribute until one is added.
- Known gaps in the operations tooling are listed in [`docs/ops/README.md`](docs/ops/README.md#known-gaps).
