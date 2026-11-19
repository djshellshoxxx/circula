# Independent identity-binding repair — 2026-11-19 UTC

## Core refresh and dedicated CI

The final publication candidate also merges core audit head
`4140bc135479077f6297d53ed2e9dfcd85c056d6`. Its module Go source is unchanged
from the already tested identity repair. The generated traceability conflict
is resolved with the checked-in generator and published core base `4140bc1`.
Final-core provider checks and the module workflow remain independent gates;
the earlier local module results below are not relabelled as new provider runs.

After the identity repair, the isolated branch merges published core audit
candidate PR #90 at `9b1fc6a0bb4ae4ad95c4f505929ce8088f10bf2c` and includes
reviewed monetary-guard test follow-up
`d627af48ff4042db73adac9a72f6208db268a924`. This preserves the original
notifications source history and keeps core qualification as a separate gate.

`.github/workflows/future-notifications.yml` uses the core `go.mod` toolchain
pin, verifies the disabled/HOLD and module boundaries, checks module checksums
and formatting, vets and race-tests with `sqlite_cgo_test`, and configures a
dependency vulnerability check. The workflow has read-only repository
permissions and no production credentials or deployment steps. Its remote
execution remains pending publication.

Local refresh checks pass: the complete tagged race suite reports 71.0%
root-package coverage (1.432 seconds), tagged `go vet`, module checksums,
formatting, YAML/shell syntax, live HOLD/module checks, pinned workflow actions
and generated traceability. These local database commands retain the lower
memory C flags documented below. The provider vulnerability check is configured
but has not been executed locally; its outcome remains a separate CI gate.

## Identity repair evidence

Source baseline: `3bdf8e92d84af2c660c615bcf956cf76752e86ae`, on
`future/fux-104-notifications-20261116`. Repairs are confined to the isolated
`audit/future-notifications-binding-20261119` branch and this nested module.
No core adapter, production credentials or transport deployment is enabled.

Two defect groups were reproduced on unchanged source before repair:

1. Accepted actor scopes `(network|extra, member)` and
   `(network, extra|member)` produced the same delimiter-joined target AAD.
   Encryption therefore did not distinguish those scopes. The regression
   fails for both email and push. Pipe-free legacy bindings remain byte-for-byte
   compatible; delimiter-bearing fields now use domain-separated, fixed-width
   byte-length prefixes with no legacy fallback.
2. Invalid UTF-8 identifiers were normalized by JSON encoding before dedupe
   hashing or token signing. Distinct raw identities could become the same
   dedupe key or a token naming a different normalized scope. Regressions
   fail for invalid Actor/Event identifiers, all DedupeKey fields and every
   signed payload string. The repair rejects those inputs and retains valid
   Unicode identifiers without normalization.

Observed commands use Go 1.26.11 linux/amd64, `GOTOOLCHAIN=local`,
`GOMAXPROCS=2`, `GOGC=50`. The tagged database run uses `CGO_ENABLED=1` and
`CGO_CFLAGS='-O0 -g0'` to fit the available compiler memory; it still includes
Go race instrumentation.

| Check | Observed result |
|---|---|
| New regressions on original source | FAIL, reproducing both defect groups |
| `go test -race -p=1 -count=1 -cover ./...` after repair | PASS; root package 7.2%; database tests skipped without their tag |
| `go test -race -p=1 -count=1 -tags=sqlite_cgo_test -cover -timeout=90s ./...` after repair | PASS; root package 71.0%; real test-only SQLite driver |
| Same tagged suite at original source | PASS; root package 70.4%; that suite alone did not detect the new identity defects |
| `go vet ./...`, `gofmt -l .`, `git diff --check` | PASS; original `targets.go` formatting corrected |
| Independent source and regression review | Approved for isolated development; no additional blocker found in the repair |

The transport subpackages have no executed tests in these runs and report
0% coverage. This is not real SMTP/Web Push delivery, browser opt-in, worker
crash recovery or production integration qualification. Remaining gates and
the required revoke-then-re-add recovery for ambiguous legacy email targets
are documented in README.md. The historical initial-milestone evidence below
is retained with its original scope and toolchain.

# Historical validation record — 2026-11-16

Toolchain: Go 1.25.4 linux/amd64, standard library only. The root application requires Go 1.26.11; it was not built or tested here. Its files remain unchanged.

```
$ go test -race -cover ./...
ok  	github.com/djshellshoxxx/circula/future/notifications	1.013s	coverage: 85.7% of statements
$ go vet ./...
PASS (exit 0, no diagnostics)
```

Manual source review checked: no app/internal imports, no cross-feature imports, no init/registration or background jobs, no external dependencies, no network transport or production DB writes, and no ledger posting API. All feature inputs requiring trusted current state are documented as adapter responsibilities.

The suite tests this initial core milestone only. Coverage is not proof of security or feature completion. Durable stores, fresh authorization adapters, full acceptance tests, browser/CLI experience and independent security review remain unfinished as detailed in README.md. No claim of race-safe DB persistence, production qualification or release readiness is made.
