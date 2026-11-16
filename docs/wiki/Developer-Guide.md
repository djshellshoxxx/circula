# Developer Guide

[Home](Home) · [User Guide](User-Guide)

**Source status:** Development instructions for branch `claude/cool-gauss-vdxlod` of the repository. They are **not deployment authorization**. Feature state follows [`docs/plans/circula-completion-matrix.md`](https://github.com/djshellshoxxx/circula/blob/claude/cool-gauss-vdxlod/docs/plans/circula-completion-matrix.md) (2026-11-11). Owner decisions, the licence and the release gate are still open. Nothing here has been tagged or published.

Links below point at the `claude/cool-gauss-vdxlod` branch. Update them when that branch merges.

## Module map

| Path | Role |
|---|---|
| `cmd/circula` | The `circula` command: HTTP process, migrations, backups, restore, ledger verification, operator jobs |
| `internal/domain` | Money (int64 minor units), invariants, enums, policy and state machines |
| `internal/service` | Authorization, ledger posting, trades, audit, governance (dual control, break-glass, erasure), recovery, export |
| `internal/store` | SQLite storage, transactions, WAL, forward-only migrations, append-only triggers |
| `internal/api` | JSON API under `/api/v1`: scoped tokens, idempotency keys, problem responses, keyset paging |
| `internal/web` | Server-rendered HTML: templates, static assets, account, trade, dispute and administration pages |
| `internal/httpx` | HTTP edge policy shared by the HTML and JSON adapters (cookie and host policy) |
| `internal/histbal` | Adapter-facing contract for a member's balance at a point in time |
| `internal/contract` | Checks the running JSON API against [`docs/api/openapi.yaml`](https://github.com/djshellshoxxx/circula/blob/claude/cool-gauss-vdxlod/docs/api/openapi.yaml) |
| `internal/testutil` | Deterministic service environments for tests |
| `e2e/` | Playwright, axe and keyboard tests in headless Chromium |
| `deploy/` | systemd unit, Dockerfile, compose file, sysusers and tmpfiles |
| `scripts/` | Release gate, packaging, restore drill, hygiene and action-pin checks, smoke deploy |
| `docs/` | Specs, operations, guides, QA, audits, decision log |

Rules that the code depends on (from [`docs/dev/SERVICE.md`](https://github.com/djshellshoxxx/circula/blob/claude/cool-gauss-vdxlod/docs/dev/SERVICE.md)): the service layer is the write path and enforces network scope; money is integer minor units, never floating point; a trade is re-validated before it posts; the audit and ledger are append-only; failures are closed.

## Prerequisites

- **Go 1.26.11**, as declared in [`go.mod`](https://github.com/djshellshoxxx/circula/blob/claude/cool-gauss-vdxlod/go.mod). The SQLite driver is pure Go, so ordinary builds need no C compiler.
- **Race tests need cgo.** The Makefile defaults `CGO_ENABLED` to `0`, so set `CGO_ENABLED=1` and have a C compiler for `make test-race`.
- **End-to-end tests:** Node 20 or later and a Chromium. `make e2e` reads `CHROMIUM_PATH` (default `/opt/pw-browsers`) and uses the locked `e2e/package-lock.json`.
- Optional for `make check`: `golangci-lint` and `govulncheck` (the `vuln` target runs govulncheck through `go run`, so it needs network access to the vulnerability database).

## Local instance

```sh
make build
mkdir -p data
export CIRCULA_PUBLIC_BASE_URL=http://127.0.0.1:8080
export CIRCULA_COOKIE_SECURE=false                       # loopback http only
export CIRCULA_SETUP_SECRET="$(head -c 24 /dev/urandom | base64)"
export CIRCULA_DATABASE="$PWD/data/circula.db"
./bin/circula migrate
printf '%s\n' 'a-long-operator-password' | ./bin/circula admin create-operator -login ops-admin
./bin/circula serve
```

Then open `http://127.0.0.1:8080`. Check `/healthz`, `/readyz`, and `/metrics` (`/metrics` answers only from loopback or `CIRCULA_METRICS_ALLOW` addresses). Run `./bin/circula serve -check-config` to validate settings without touching the database.

Notes:
- `serve` listens on `127.0.0.1:8080` unless `CIRCULA_LISTEN` or `-listen` is set.
- `serve` refuses an `http://` public URL on a non-loopback address.
- Keep `CIRCULA_COOKIE_SECURE` at its default (`true`) behind TLS. Set `false` only for loopback development.
- Secrets (setup secret, metrics token, backup keys, MFA key) come from the environment or a 0600 file, never from a flag.
- Use a disposable database for tests and local runs. Do not expose a development instance or its setup secret on a public network.

## CLI reference

Run `circula help` for the built-in list. Every command accepts `-config` (or `CIRCULA_CONFIG`). Precedence: flags, then `CIRCULA_*` environment variables, then the JSON config file, then defaults.

| Command | Purpose |
|---|---|
| `serve [-check-config]` | Run the HTTP service (web, JSON API, health, metrics) |
| `migrate [-dry-run]` | Back up if needed, apply pending migrations, verify every network |
| `backup <dest>` | Verified online snapshot and manifest (file, or directory for a timestamped name) |
| `restore [-overwrite] <backup-file>` | Restore into the configured database. Stop `serve` first |
| `verify-ledger [-network id-or-slug]` | Recompute ledger invariants. Exit `3` if a check fails |
| `verify-export <file\|dir>` | Check an export package against its manifest |
| `maintenance on\|off\|status` | Toggle maintenance mode of a running instance (503 for web and API) |
| `setup-reset` | Re-arm the one-time setup secret. Stop `serve` first |
| `keygen` | Print a random key for `backup-encryption-key` |
| `admin create-operator -login X` | Create a person. Password from `CIRCULA_OPERATOR_PASSWORD` or stdin |
| `reset-password -login X -reason "..."` | Set a temporary password (audited) |
| `reset-mfa -login X -reason "..."` | Remove a lost second factor (audited) |
| `unlock -login X` | Clear sign-in and recovery lockouts for a login (audited) |
| `revoke-sessions -reason "..." [-login Y] [-rotate-web-key]` | Sign out everyone, or one person. `-rotate-web-key` also replaces the web signing key |
| `suspend start\|lift` | Platform suspension of a network |
| `dual-control -network X -threshold-minor N -reason "..."` | Set or disable a network's dual-control threshold. Administrators can only tighten it |
| `break-glass open\|close\|list\|grant-admin` | Operator break-glass sessions. See `docs/dev/break-glass.md` |
| `erase-member -network X -member ID -reason "..." -confirm` | Irreversibly anonymise a departed member. See `docs/dev/erasure.md` |
| `audit-anchor write\|verify` | Write or check an off-host audit-chain checkpoint |
| `prune-logs -dir <dir>` | Delete log files older than `log-retention` |
| `version` | Print version, commit and build date |

Exit codes: `0` success, `1` failure, `2` usage or configuration error, `3` operation completed but ledger verification failed.

Full settings reference: [`docs/ops/configuration.md`](https://github.com/djshellshoxxx/circula/blob/claude/cool-gauss-vdxlod/docs/ops/configuration.md). Operations runbooks: [`docs/ops/README.md`](https://github.com/djshellshoxxx/circula/blob/claude/cool-gauss-vdxlod/docs/ops/README.md).

## Build and test

```sh
make build            # bin/circula, CGO_ENABLED=0, -trimpath
make test             # go test -count=1 ./...
make vet
make contract         # JSON API against docs/api/openapi.yaml
make test-race        # needs CGO_ENABLED=1 and a C compiler
make e2e              # builds the binary, installs e2e deps, runs Playwright
make load             # concurrent-trade load test (CIRCULA_LOAD=1)
make restore-drill    # scripts/restore-drill-full.sh
make check            # fmt-check, vet, test-race, lint, vuln, actions-check, hygiene
make release-gate     # scripts/release-gate.sh: evidence checks
make release-local VERSION=vX.Y.Z   # local archives in bin/dist
```

`make e2e` is not part of `make check`. `make release-gate` runs evidence checks. Its script contains no tag, push or release commands. `make release-local` builds local archives only.

Run `make help` for the authoritative target list.

## Continuous integration

[`.github/workflows/ci.yml`](https://github.com/djshellshoxxx/circula/blob/claude/cool-gauss-vdxlod/.github/workflows/ci.yml) defines:

- **Every pull request:** the Linux gate. Lint (vet, gofmt, golangci-lint, govulncheck, action-pin check, repository hygiene, dependency licence check against the approved list, secret scan), race tests, build, reproducible build, fuzz smoke, API contract and load smoke, and the e2e job (Playwright, axe, keyboard).
- **Main pushes, nightly and manual runs:** `test-cross`, the Go test suite on macOS Intel (race) and Windows. Not run on pull requests.

Run results are on the Actions tab. This page does not record them (UNVERIFIED for this update). Use the run for the exact commit you are testing; a later head is not tested by an earlier run.

Release artifacts come from [`.github/workflows/release.yml`](https://github.com/djshellshoxxx/circula/blob/claude/cool-gauss-vdxlod/.github/workflows/release.yml) on a `v*` tag. Read the [release gate](https://github.com/djshellshoxxx/circula/blob/claude/cool-gauss-vdxlod/docs/qa/RELEASE-GATE.md) before tagging. Tagging, publishing and deployment need explicit owner authorization.

## Debugging and operations

- `./bin/circula verify-ledger`: recompute invariants. Exit code `3` means a ledger check failed.
- Frozen networks, backups and restore: [runbook](https://github.com/djshellshoxxx/circula/blob/claude/cool-gauss-vdxlod/docs/ops/runbook.md), [backup and restore](https://github.com/djshellshoxxx/circula/blob/claude/cool-gauss-vdxlod/docs/ops/backup-restore.md), [upgrade](https://github.com/djshellshoxxx/circula/blob/claude/cool-gauss-vdxlod/docs/ops/upgrade.md).
- Keep `/metrics`, operator endpoints, audit data and backups private.
- Operational subsystems have their own notes in `docs/dev/`: [MFA](https://github.com/djshellshoxxx/circula/blob/claude/cool-gauss-vdxlod/docs/dev/mfa.md), [recovery](https://github.com/djshellshoxxx/circula/blob/claude/cool-gauss-vdxlod/docs/dev/recovery.md), [dual control](https://github.com/djshellshoxxx/circula/blob/claude/cool-gauss-vdxlod/docs/dev/dual-control.md), [durability](https://github.com/djshellshoxxx/circula/blob/claude/cool-gauss-vdxlod/docs/dev/durability.md), [Argon2 benchmark](https://github.com/djshellshoxxx/circula/blob/claude/cool-gauss-vdxlod/docs/dev/argon2-benchmark.md).

## Specifications and contracts

- [Specifications index](https://github.com/djshellshoxxx/circula/blob/claude/cool-gauss-vdxlod/docs/specs/README.md). Data and ledger: [06](https://github.com/djshellshoxxx/circula/blob/claude/cool-gauss-vdxlod/docs/specs/06-data-ledger-audit.md). API contract: [07](https://github.com/djshellshoxxx/circula/blob/claude/cool-gauss-vdxlod/docs/specs/07-api-contract.md). Security and privacy: [09](https://github.com/djshellshoxxx/circula/blob/claude/cool-gauss-vdxlod/docs/specs/09-security-privacy.md). Some status columns are stale; the completion matrix takes precedence.
- [OpenAPI document](https://github.com/djshellshoxxx/circula/blob/claude/cool-gauss-vdxlod/docs/api/openapi.yaml). Check changes with `make contract`.
- [Decision log](https://github.com/djshellshoxxx/circula/blob/claude/cool-gauss-vdxlod/docs/DECISIONS.md).
- [Traceability](https://github.com/djshellshoxxx/circula/blob/claude/cool-gauss-vdxlod/docs/TRACEABILITY.md) and [historical completeness audit](https://github.com/djshellshoxxx/circula/blob/claude/cool-gauss-vdxlod/docs/audit/COMPLETENESS.md). Both predate the matrix; treat them as history.

## Contributing and security

- [CONTRIBUTING.md](https://github.com/djshellshoxxx/circula/blob/claude/cool-gauss-vdxlod/CONTRIBUTING.md): formatting, tests, dependency and action-pin rules, release rules.
- [SECURITY.md](https://github.com/djshellshoxxx/circula/blob/claude/cool-gauss-vdxlod/SECURITY.md): private reporting. Do not disclose exploitable unpatched findings in public issues.
- Do not change financial policy, cryptographic trust paths or release permissions without owner approval.

## Open engineering work

From the completion matrix (2026-11-11):

- **O2:** container build and deploy files are UNVERIFIED (Docker was not run in the evidence).
- **O4:** CI is PARTIAL. Remove duplicate runs and move macOS and Windows tests to main or nightly.
- **W10:** export runs in memory; the streaming or spool slice is not integrated.
- **L9:** posting fault injection is at the store interface, not real SQLite I/O errors.
- **A6:** account-enumeration timing equivalence is not measured.
- **W5:** an administrator without the reviewer role may decide disputes (DEC-072).

Owner gates (not engineering work): licence, DEC-005, DEC-011, DEC-013, DEC-014, DEC-021, DEC-022, DEC-023, DEC-025, break-glass scope A3-F02, independent security review SEC-033, manual screen-reader run UX-056 and QA-035, and the first tagged release.
