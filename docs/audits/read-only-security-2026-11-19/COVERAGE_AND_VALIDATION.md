# Coverage, evidence and validation limits

## What this review did

The coordinating reviewer and three parallel reviewers inspected pinned source, specifications, existing tests, prior audit dispositions and read-only GitHub metadata. The first passes divided authentication/authority, ledger/lifecycle, web/API, and operations/delivery work. Two follow-up passes challenged selected findings for counterevidence and precise prerequisites. A further pass reviewed two disabled future modules. The detailed appendices retain the actual file coverage and limitations for each pass.

The review used main `aa01563fcc891da60fefebb8335eca27855678e8` and Core candidate `22c767a90f69d286642ef96e6b3c174836637a3c`. Production service/store/domain code is identical at those pins; five store test files differ. Main and candidate respectively contain 1,442 and 1,719 tracked files; each has 283 production Go files and 21 SQL migrations. The inventory snapshot contains 145 remote branches. Counts and branch pins are in [branch-inventory.json](evidence/branch-inventory.json).

## Semantic coverage

| Lane | Material inspected | Coverage limit |
|---|---|---|
| Authentication and authority | Login/password hashing, MFA/TOTP, recovery, tokens, session self/other management, membership/join, current roles, grant ancestry, break-glass, erasure, supporting person/member/MFA/recovery repositories | Application-use inspection, not cryptographic primitive analysis. No production account, deployed MFA policy or operator identity validation. Exact paths in [AUTH.md](AUTH.md). |
| Ledger and lifecycle | Posting/proposals/acceptance, adjustment/reversal, disputes, idempotency, policies/limits, departure, closure, exports, verification, money/state domain types and relevant schema/repository paths | No formal proof of all states. Migration coverage was selective and shared across lanes. No abnormal-volume or concurrency reproduction. Exact paths and fixed-prior-issue reconciliation in [LEDGER.md](LEDGER.md). |
| HTTP and privacy | API route inventory, auth/middleware, strict decoding, cursors, replay, export lifecycle, handlers, reporting/listings; web wrappers/session/CSRF, account/security flows, rendering and selected templates/JavaScript | API/service boundaries received deeper coverage than every web form/localization. No new browser matrix or live reverse-proxy test. Exact coverage in [WEB_API.md](WEB_API.md). |
| Operations and delivery | Backup framing, staging, restore/swap, manifests, config/keys, startup/HTTP deadlines, ops endpoints, audit anchors/mirrors, export readers, deployment recipes and release/CI workflows | Some long files traced selectively. No native OS ACL/rename validation, real disk fault, live ingress, secret custody or deployed load assessment. Paths below and [OPERATIONS.md](OPERATIONS.md). |
| Independent challenge review | AUTH-01/02, WAPI-01/02, LED-R01/04, OPS-01/02; secondary corroboration of AUTH-03 | Source-only second passes, not independent external certification. Qualifications incorporated into findings. [CROSSCHECKS.md](CROSSCHECKS.md). |
| Isolated feature review | Notification dispatch/transport/secret handling and federation persistent trust, protocol, replay, transport and selected lifecycle adapters | Two explicit non-Core pins; no activation, provider sends, interoperability or full integration validation. Messaging/Windows follow-up excluded. [FEATURES.md](FEATURES.md). |

Operations paths read or traced include `cmd/circula/config.go`, `commands.go`, `backupcrypt.go`, `backup_work.go`, `snapshot.go`, `restore_security.go`, `restore_swap.go`, `statekey.go`, `util.go`, `fileident_unix.go`, `fileident_other.go`, `serve.go`, `ops_http.go`, `auditanchor.go`, `auditmirror.go`, `exportspool.go` and `verify_export.go`; supporting `internal/store/sqlite.go`, `sqlite_audit.go`, service audit/verify/restore-credential paths; `deploy/Dockerfile`, `compose.yaml`, `circula.service`; CI, release, scorecard and export-reader workflows; and action-pin, packaging, verification, release-scanning and export-reader scripts. Relevant README, SECURITY, operations/security specs and prior audit sections were cross-referenced. This list describes reading/tracing, not execution of each file.

## Toolchain and local execution

The installed `/usr/bin/go` was a Go board-game program. A real Go toolchain was provisioned separately for ordinary checks: **Go 1.26.11, linux/amd64**, downloaded from the official distribution. The archive SHA-256 matched the published value:

```text
9849f4e04ee35303714e34fbc3ac8ab44ea3e448a95aa182f91e3b9b79c710ec
```

Source: [official archive](https://dl.google.com/go/go1.26.11.linux-amd64.tar.gz) and [checksum](https://dl.google.com/go/go1.26.11.linux-amd64.tar.gz.sha256). Runtime and module caches were outside application checkouts. Modules downloaded successfully; module/lock files were unchanged. The selected constrained runs used `GOTOOLCHAIN=local`, `GOMAXPROCS=2`, `GOMEMLIMIT=512MiB` and `GOFLAGS=-p=1`. These controls do not guarantee a compiler fits the execution environment.

All following fresh execution results concern the pinned **Core candidate**. The future nested modules were not included in the root-module dependency scan.

| Check | Result | Evidence / interpretation |
|---|---|---|
| `go test -mod=readonly -count=1 -timeout=3m ./internal/domain ./internal/httpx ./internal/logx` | **Passed**, exit 0 | [baseline-tests.log](evidence/baseline-tests.log). Existing ordinary package suites. |
| `go mod verify` | **Passed**, exit 0 | [module-verification.log](evidence/module-verification.log). Downloaded module content verification, not vulnerability or authorship certification. |
| `go run golang.org/x/vuln/cmd/govulncheck@v1.1.4 -scan=package ./...` | **Passed**, exit 0; no vulnerabilities reported | [govulncheck-package.log](evidence/govulncheck-package.log). Package-level analysis only, weaker than a completed symbol/call-graph analysis. |
| Full source/call-graph `govulncheck` for `./...` | **Incomplete**, exit 1, `signal: killed` | [govulncheck.log](evidence/govulncheck.log). No full-scan assurance. |
| Selected existing CLI backup/key baselines | **Did not execute assertions**; compilation killed | [cli-baseline-tests.log](evidence/cli-baseline-tests.log). `modernc.org/libc` build was killed; no test pass or assertion failure established. Exact selection in [commands.json](evidence/commands.json). |
| Selected existing API route/session/entry-read baselines | **Did not execute assertions**; compilation killed | Worker-recorded command and failure in [WEB_API.md](WEB_API.md). No separate raw stdout artifact retained for this lane; evidence is the worker receipt. |
| `python3 -m unittest discover -s scripts -p test_read_export.py` | **41 passed**, exit 0 | [python-export-tests.log](evidence/python-export-tests.log). Existing suite; bounded fixtures. |
| `node --test scripts/test_app_messages.cjs scripts/test_e2e_kbd.cjs` | **15 passed**, exit 0 | [node-helper-tests.log](evidence/node-helper-tests.log). Existing ordinary helper checks. |
| `bash scripts/check-action-pins.sh` | **Passed**, exit 0 | [workflow-pins.log](evidence/workflow-pins.log). Action pin/permission policy only. |

Killed-process logs establish incomplete local execution. Resource limits were the operational impediment, but no independent kernel OOM attribution was captured. These results must not be presented as code assertion failures, full-suite success, a completed race run, or absence of vulnerabilities. No optional stress/fuzz/reproduction tests were added. Existing source/test inspection supplies the finding evidence separately from these baseline results.

## Exact-head GitHub evidence

[github-observations.json](evidence/github-observations.json) records a minimal settings/check snapshot with the observation time, endpoints, pins, job URLs and step results. At that time:

- Main was unprotected, required check enforcement was off, and repository/inherited rulesets were empty.
- PR #125 remained draft at the reviewed head. Its Ubuntu test, e2e/accessibility/keyboard, API contract/load, cross-build, reproducibility and fuzz jobs passed. The Python reader passed on Linux, macOS and Windows.
- Its combined lint/security job failed at the scanner-control metadata step. Later formatting, vet, traceability, helper, lint and vulnerability steps were skipped. The expected metadata references absent composed audit documents; the failure is not evidence of a leaked credential.
- The native Core cross-platform job was skipped. A passing cross-build or platform Python run is not a native Core test.

These are inspected provider results. The review did not execute external load/fuzz jobs, rerun CI, change protection, merge PRs or deploy code. Provider observations are time-bound and should be refreshed on a repair revision.

## Inventory of isolated work

[future-manifests.json](evidence/future-manifests.json) records fifteen canonical feature-branch manifests, all disabled and on HOLD. Those original canonical branches may be older than repair/recovery descendants; they are inventory evidence, not a claim that those heads are the latest completed implementation. The feature review explicitly pins the newer notification and federation descendants. Held administrator recovery is separately pinned in AUTH.md. No future feature is present in either reviewed Core tree.

The audit does not claim that all 145 branch histories, every future module, every template, every migration transition, dependencies' internals, deployed infrastructure, native operating systems or production datasets were exhaustively audited. It establishes a broad source-based handoff with explicit coverage and actionable repair invariants.

## Evidence integrity and handoff boundary

All publication paths are confined to this audit folder under `docs/audits/`. The supplied [evidence manifest](evidence/report-manifest.json) records file SHA-256 digests and byte sizes; it excludes itself to avoid a recursive hash. These hashes detect accidental changes to the notes and receipts. They do not replace repository trust, independent signatures or review of the repair itself.

The source-reading worktrees remained clean. The documentation branch is based on the pinned main commit and contains notes/evidence only. Final publication verification checks the remote diff for that boundary. Source pins and findings stay attached to their original versions even after a future worker updates status.
