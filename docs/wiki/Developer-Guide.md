# Developer Guide

[Home](Home) · [User Guide](User-Guide)

**Source status:** Application development is in **draft PR #19**, pinned here to `d98734cf1b2cc9a826397a6e43f7902de2304109`; `main` is prompt-only. These are development instructions, **not deployment authorization**. The continuation policy is draft [PR #20](https://github.com/djshellshoxxx/circula/pull/20).

## Architecture and module map

- `cmd/circula`: CLI, HTTP process, maintenance, migrations, backups, restore, verification and operational jobs.
- `internal/domain`: money (int64 minor units), invariants, enums, policy and state machines.
- `internal/service`: core authorization, ledger posting, transactions, audit, member governance and recoveries.
- `internal/store`: SQLite storage, transactions, WAL, migrations and append-only triggers.
- `internal/api`: JSON HTTP API `/api/v1`, request validation, scoped tokens, idempotency and problem responses.
- `internal/web`: server-rendered HTML, account, trade and administration workflows.
- `internal/contract`: contract tests against the [OpenAPI specification](https://github.com/djshellshoxxx/circula/blob/d98734cf1b2cc9a826397a6e43f7902de2304109/docs/api/openapi.yaml).
- `e2e`: Playwright, axe and keyboard tests; `docs/ops` and `docs/specs`: operator and normative references.

The service is the authorized application write path and controls network-specific access. The database has a single-writer-process design; record money as integer minor units, never floating-point amounts. A trade must be revalidated before posting. Preserve append-only audit/ledger behavior and fail-closed errors.

See [service-layer reference](https://github.com/djshellshoxxx/circula/blob/d98734cf1b2cc9a826397a6e43f7902de2304109/docs/dev/SERVICE.md) and [specifications](https://github.com/djshellshoxxx/circula/blob/d98734cf1b2cc9a826397a6e43f7902de2304109/docs/specs/README.md).

## Prerequisites and local setup

Work on an isolated branch based on PR #19's application head; **do not run `make` on prompt-only `main`**.

- Go **1.26.10** as declared in the pinned [go.mod](https://github.com/djshellshoxxx/circula/blob/d98734cf1b2cc9a826397a6e43f7902de2304109/go.mod); pure-Go SQLite driver, no C compiler required for ordinary builds.
- Node 20+ and Chromium for browser tests; npm dependencies locked in `e2e/package-lock.json`.
- A C toolchain is required for `go test -race`; `golangci-lint` and `govulncheck` for full checking.

```sh
git checkout claude/happy-noether-444cum
make build
mkdir -p data
export CIRCULA_DATABASE="$PWD/data/circula.db"
export CIRCULA_PUBLIC_BASE_URL=http://127.0.0.1:8080
export CIRCULA_COOKIE_SECURE=false  # loopback development only
export CIRCULA_SETUP_SECRET="$(head -c 24 /dev/urandom | base64)"
./bin/circula migrate
./bin/circula serve
```

With a local instance, browse `http://127.0.0.1:8080`. Use [README setup notes](https://github.com/djshellshoxxx/circula/blob/d98734cf1b2cc9a826397a6e43f7902de2304109/README.md) for first-operator creation. Do not expose the local HTTP setup or fixtures on a public network. Confirm the configuration using `./bin/circula serve -check-config`.

## Build, tests and debugging

```sh
make build
make test             # go test -count=1 ./...
make vet
make contract         # OpenAPI vs JSON API
make test-race        # requires cgo toolchain
make e2e              # Node, Chromium; starts an ephemeral test instance
make check            # fmt, vet, race, lint, vuln, action pins, hygiene
./bin/circula verify-ledger
```

`make release-gate` performs evidence checks but **does not tag, push or publish**. The repository's [CI workflow](https://github.com/djshellshoxxx/circula/blob/d98734cf1b2cc9a826397a6e43f7902de2304109/.github/workflows/ci.yml) includes Go on Linux/macOS/Windows, contract/load, browser accessibility, fuzz smoke and reproducible cross-build checks. At exact PR #19 head, two CI runs [41806043004](https://github.com/djshellshoxxx/circula/actions/runs/41806043004) and [41806032535](https://github.com/djshellshoxxx/circula/actions/runs/41806032535) succeeded (nine jobs each). Manual screen-reader, independent security audit and owner thresholds remain unqualified. For failures inspect the failing job and the same exact SHA; do not infer a later branch head is tested.

### Exact-head CI interpretation (November 10, 2026)

A subsequent isolated terms-return regression fix exists in **draft [PR #24](https://github.com/djshellshoxxx/circula/pull/24)**, **not** in the pinned PR #19 base. Its exact-head [Actions run 42838956843](https://github.com/djshellshoxxx/circula/actions/runs/42838956843) at `7620d7f51913363dd013b80cc682eb670eeff3a5` completed with **eight successful jobs and one cancelled macOS-14 test job**. The successful jobs include Ubuntu/Windows Go tests, browser accessibility, contract/load, fuzz, lint/vulnerability checks and builds. The workflow's overall conclusion is **cancelled, not success**. A targeted retry of the macOS job was also cancelled without executed steps/logs. **Subsequent runner-image investigation identified an obsolete runner label:** GitHub's [official retirement notice](https://github.com/actions/runner-images/issues/13518) states that `macos-14` became unsupported on **2026-11-02**; this matches the observed no-step cancellations, though the specific internal cancellation reason was not returned by the job API. A workflow-only correction is proposed in stacked **draft [PR #25](https://github.com/djshellshoxxx/circula/pull/25)** at `1760551f5b03a89ec64d09269d631e72f2e95959`, changing only the macOS test runner to the supported [macOS 15 Intel image](https://github.com/actions/runner-images/issues/13045). It keeps x86_64 coverage and the full test matrix; it is **not qualified until exact-head CI passes**. Do not interpret the old cancellations as a Go test assertion failure or a successful run. Do not remove OS coverage or relax any security/CI gate; retain manual screen-reader and independent security-review holds.

**Follow-up exact-head macOS race qualification:** PR #25's replacement runner did execute the complete race suite, but its [run 42869647736](https://github.com/djshellshoxxx/circula/actions/runs/42869647736) and [run 42869590887](https://github.com/djshellshoxxx/circula/actions/runs/42869590887) each finished **8 successful jobs, 1 failed macOS test job**. Logs show Go's default `panic: test timed out after 10m0s` during SQLite-heavy tests (including `internal/service`; the second run also timed out in `cmd/circula` and `internal/api`). Unlike the retired-runner cancellation, the new macOS runner actually executed steps; no discrete test assertion failure was established by those logs. A separate stacked **draft [PR #26](https://github.com/djshellshoxxx/circula/pull/26)** at `2078bf0964353892e80214b4601211e36818b889` retains full macOS race coverage with `go test -race -count=1 -p 2 -timeout=25m ./...` and a 45-minute macOS job cap while Linux and Windows test commands remain unchanged. Its [push CI run 42880207869](https://github.com/djshellshoxxx/circula/actions/runs/42880207869) and [pull-request CI run 42880245458](https://github.com/djshellshoxxx/circula/actions/runs/42880245458) have both **completed successfully at exact commit `2078bf0964353892e80214b4601211e36818b889`**. The PR run completed **9/9 jobs**, including macOS Intel race tests, Ubuntu and Windows tests, vet/gofmt/lint/govulncheck, browser accessibility automation, OpenAPI/load, fuzz smoke, release-target builds and reproducible builds. The adjustment changes execution resource bounds, not accounting/security policy or test assertions. This result applies only to the **unmerged stacked PR #26 source**: the separate documentation/audit/handoff branches still run an older macOS-14 matrix and their exact-head workflows were cancelled; those heads are **not** fully CI-qualified. Automated browser checks also do not replace manual screen-reader or independent security-review gates.

### Read-only decision-hint expiry correction (draft #27; awaiting new CI)

On the stacked, **unmerged** source, `internal/service/dualcontrol.go:DecidableAdjustments` generates a UI/API eligibility hint from stored requests; it is **not** an authorization or posting decision. `ApproveAdjustment` and `RejectAdjustment` perform authoritative checks in their write transaction. The hint originally filtered `PendingOpen` requests without checking `ExpiresAt`, so an expired, not-yet-swept request could still appear eligible even though the approval path refused it. Draft [PR #27](https://github.com/djshellshoxxx/circula/pull/27) added the expiry-boundary regression in `internal/service/dualcontrol_decidable_expiry_test.go` at test-only SHA `eaf2a0360dd409c033f6caa982428e5bf361aea2`. GitHub-hosted [PR CI 42900149246](https://github.com/djshellshoxxx/circula/actions/runs/42900149246) and [push CI 42900136716](https://github.com/djshellshoxxx/circula/actions/runs/42900136716) **both failed as expected**: Ubuntu, macOS 15 Intel and Windows each reported `dualcontrol_decidable_expiry_test.go:26: expired pending request incorrectly offered as decidable`. Other jobs in the PR run passed. The minimal correction was subsequently committed as `b61a8e719030ff56044d71fee632b5ba824d7698`, checking a single sampled current time against `ExpiresAt` in the read-only hint; authoritative approval/rejection, posting, ledger and administrator-policy code were not changed. **Fresh implementation-head [PR CI 42908897181](https://github.com/djshellshoxxx/circula/actions/runs/42908897181) and [push CI 42908887240](https://github.com/djshellshoxxx/circula/actions/runs/42908887240) are pending; do not claim the correction qualified until both finish.** This is draft engineering evidence, not a merged or deployed capability. Operators must continue to treat the authoritative decision result as final.

Use a disposable SQLite database for tests; keep `/metrics`, operator endpoints, audit data and backups private. For diagnosing frozen networks and backup issues, consult [operator runbook](https://github.com/djshellshoxxx/circula/blob/d98734cf1b2cc9a826397a6e43f7902de2304109/docs/ops/runbook.md), [configuration](https://github.com/djshellshoxxx/circula/blob/d98734cf1b2cc9a826397a6e43f7902de2304109/docs/ops/configuration.md) and [backup/restore](https://github.com/djshellshoxxx/circula/blob/d98734cf1b2cc9a826397a6e43f7902de2304109/docs/ops/backup-restore.md).

## API contracts, contributions and security rules

- [OpenAPI](https://github.com/djshellshoxxx/circula/blob/d98734cf1b2cc9a826397a6e43f7902de2304109/docs/api/openapi.yaml) — request/response contracts; check its exact-head implementation against `internal/api`.
- [Data and ledger specs](https://github.com/djshellshoxxx/circula/blob/d98734cf1b2cc9a826397a6e43f7902de2304109/docs/specs/06-data-ledger-audit.md), [security spec](https://github.com/djshellshoxxx/circula/blob/d98734cf1b2cc9a826397a6e43f7902de2304109/docs/specs/09-security-privacy.md), and [decision log](https://github.com/djshellshoxxx/circula/blob/d98734cf1b2cc9a826397a6e43f7902de2304109/docs/DECISIONS.md).
- [CONTRIBUTING.md](https://github.com/djshellshoxxx/circula/blob/d98734cf1b2cc9a826397a6e43f7902de2304109/CONTRIBUTING.md) — formatting, tests, dependency workflow, action pinning and release rules.
- [SECURITY.md](https://github.com/djshellshoxxx/circula/blob/d98734cf1b2cc9a826397a6e43f7902de2304109/SECURITY.md) — private vulnerability reporting and disclosure restrictions.

Create isolated branches and draft PRs. Add a failing regression test before correcting a reproducible defect, then qualify the relevant test and broad CI. Never modify another contributor's active branch, rewrite existing tests to mask failures, or change financial policy, cryptographic trust or release permissions without approval. Do not disclose exploitable unpatched findings in public GitHub issues.

## Release process, ADRs/specs and outstanding engineering work

Release process: review [release gate](https://github.com/djshellshoxxx/circula/blob/d98734cf1b2cc9a826397a6e43f7902de2304109/docs/qa/RELEASE-GATE.md), record evidence, resolve owner thresholds and manual accessibility/security items, prepare release notes from [releases source](https://github.com/djshellshoxxx/circula/tree/d98734cf1b2cc9a826397a6e43f7902de2304109/docs/releases), and obtain explicit owner authorization before tag/publish/deploy. `make release-local` builds local archives only.

Key open decisions include licence, SEC-033 independent review, DEC-014 (A3-F02), owner financial/security choices and manual QA-035/UX-056. Historical [completeness audit](https://github.com/djshellshoxxx/circula/blob/d98734cf1b2cc9a826397a6e43f7902de2304109/docs/audit/COMPLETENESS.md) and [traceability matrix](https://github.com/djshellshoxxx/circula/blob/d98734cf1b2cc9a826397a6e43f7902de2304109/docs/TRACEABILITY.md) need source-pinned requalification; passing automated checks is not an assertion of full completion.

Current work tracking: [PR #19](https://github.com/djshellshoxxx/circula/pull/19) implementation, [PR #20](https://github.com/djshellshoxxx/circula/pull/20) ChatGPT coordination proposal, and [PR #21](https://github.com/djshellshoxxx/circula/pull/21) exact-head qualification handoff. Use the active branch SHA and open PRs as truth, not a stale changelog.
