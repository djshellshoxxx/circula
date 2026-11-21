# Circula operations and delivery security review

Review date: 2026-11-19 Pacific / 2026-11-20 UTC. Main: `aa01563fcc891da60fefebb8335eca27855678e8`. Core candidate, PR #125: `22c767a90f69d286642ef96e6b3c174836637a3c`.

This is source review and read-only inspection of GitHub metadata. No deployed instance, real database, operator account or live network was tested. Application code and settings were not changed. No exploit artifacts were created.

## OPS-01 — Routine verification and external anchors omit the platform audit chain

**Severity:** Medium. **Confidence:** High in the omission; impact requires corruption or database-storage write access. **State:** Both pinned revisions. This is an integrity-monitoring gap, not a new way to obtain database access.

`internal/service/audit.go:40–45` writes platform events with an empty network ID, including identity, recovery, MFA, token and operator activity. `cmd/circula/snapshot.go:139–158` enumerates actual networks and calls ledger verification for each. `internal/service/verify.go:41–90` verifies only the requested network's audit chain. Startup, scheduled verification, backup verification and the ordinary operator verify command use this network enumeration.

`cmd/circula/auditanchor.go:75–102` builds anchors by joining `networks` to `audit_events`, excluding the platform scope. The identifier at lines 59–71 hashes one initial event ID, not the remaining platform history. `verifyAnchor` checks recorded network heads at lines 139–151. The only production call found that explicitly verifies the empty-scope chain is restore-specific `internal/service/restore_credentials.go:39–45`. An independent authentication reviewer confirmed this call graph.

**Consequence:** Routine integrity/anchor success does not cover platform security history. Network ledger verification remains useful; restore does check the platform chain after credential invalidation. The optional mirror includes platform events and can provide separate evidence when independently retained; it does not change the scope of ordinary verification or anchors.

**Repair:** Add explicit platform-chain verification to startup, scheduled and CLI paths. Add a platform head to a versioned anchor format; label legacy anchors as limited coverage. Define alert/failure behavior without unnecessarily freezing unrelated financial networks.

**Acceptance:** A synthetic fixture with both scopes reports both. Complete history and legitimate growth pass; inconsistent platform history is reported. Legacy anchors cannot silently claim platform coverage. Empty platform history has deliberate semantics.

## OPS-02 — A configured audit mirror can hold the store writer gate after commit

**Severity:** Medium, deployment-dependent availability. **Confidence:** High in control flow; runtime impact unmeasured. **State:** Both pins.

`internal/store/sqlite.go:140–147` holds the writer gate until `s.run` returns. Commit completes at lines 221–224; the audit sink is then called synchronously at 225–229. `cmd/circula/auditmirror.go:47–65` performs a blocking file write under a mutex, without context, deadline, queue or capacity policy. Its comments explicitly permit a log-shipper pipe. `cmd/circula/serve.go:148–155` installs this sink.

**Consequence:** A stalled pipe consumer or mirror filesystem can delay an already committed operation while later writes wait behind the same gate. The retained gate belongs to the in-process store instance; SQLite's transaction write lock has already been released. Pure `View` units bypass it, other processes are not held behind it, and queued updates can return on cancellation. The already blocked context-free mirror write itself is not interrupted by HTTP cancellation. Returned errors are logged, but a blocked write has not returned. This qualifies the documented best-effort behavior.

**Repair:** Decouple mirror delivery from the database writer using a bounded queue or durable outbox, with explicit loss/backpressure policy, ordering, observable failures and bounded shutdown. An unbounded queue is unsuitable.

**Acceptance:** With an inert controlled test sink that does not complete delivery, transaction acknowledgement follows the documented policy and unrelated writes remain available. Recovery preserves ordering, errors are observable and shutdown is bounded.

## OPS-03 — Windows confidentiality depends on manually provisioned ACLs

**Severity:** Medium with permissive inherited permissions; not a finding against a correctly restricted NTFS deployment. **Confidence:** High in missing application enforcement; no native Windows ACL check performed. **State:** Both pins; partly documented already.

`cmd/circula/util.go:95–106,113–137`, `statekey.go:59–76`, and `config.go:706–721` skip Unix permission enforcement on Windows. `fileident_other.go:7–18` accepts staging parents without ownership/ACL checks and lacks a link-count restriction. `internal/api/export_spool.go` uses temporary-file creation permissions; `cmd/circula/exportspool.go:21–53` checks existence/writability rather than a private security descriptor. No Windows DACL implementation was found in Core.

Go documents that Windows `Chmod` affects the read-only attribute rather than owner-only read permissions: https://pkg.go.dev/os#Chmod. `docs/ops/backup-restore.md:74–75` already requires manual NTFS ACLs. That requirement must consistently cover the database and sidecars, signing/MFA/backup keys, export spool and transient plaintext.

**Consequence:** On permissive storage, another local account may read financial records, password hashes or keys. No remote HTTP access is established by this observation.

**Repair:** Enforce or validate a Windows security descriptor for private artifacts, or refuse an unqualified storage layout. Define service-principal/administrators/SYSTEM and reparse-point policies. Isolated messaging Windows work does not automatically secure Core.

**Acceptance:** Native Windows tests with a second unprivileged principal cover creation, reopen, sidecars, backup staging, export cleanup and unsafe inherited permissions. Unix mode tests cannot satisfy this requirement.

## OPS-04 — Export verification permits multi-gigabyte in-memory inputs

**Severity:** Low, local-tool availability. **Confidence:** High in allocations; no memory-exhaustion experiment run. **State:** Both retain the 4 GiB ceiling; PR #125 adds archive correctness checks.

`cmd/circula/verify_export.go:28–29,168–179` allows a member up to `1 << 32` bytes and uses `io.ReadAll`; row counting also materializes parsed data (`:250–272`). `scripts/read-export.py:26–38,301–352` uses the same ceiling, constructs bytes/text/CSV or JSON objects, and retains account/entry collections for reconciliation. The manifest itself uses the general member limit.

**Consequence:** A large package can exhaust an operator verifier's resources or contend with colocated services. This is an explicit local command, not an HTTP upload endpoint. A 4 GiB byte ceiling is not a practical memory budget for the example 512 MiB container.

**Repair:** Separate a small manifest cap, member-count cap, aggregate uncompressed-byte budget and configurable operator budget. Stream hashing/counting and use incremental reconciliation where practical.

**Acceptance:** Small fixtures with injected limits exercise each refusal. Normal exports still verify; both declared and actual sizes obey the budget. No multi-gigabyte fixture is needed.

## QG-01 — Main-branch merge protections are not enforced by current metadata

**Classification:** Medium delivery-control gap; requires a write-capable account or its compromised credentials.

Read-only API inspection returned `protected:false` for `main`, protection disabled, required status checks off with no contexts, and an empty repository/inherited ruleset list. Sources: https://api.github.com/repos/djshellshoxxx/circula/branches/main and https://api.github.com/repos/djshellshoxxx/circula/rulesets?includes_parents=true&per_page=100. The evidence summary records the observation time. This does not grant anonymous repository writes; it leaves review/status enforcement to procedure.

**Worker action:** Prepare an owner-reviewed ruleset/protection policy requiring relevant current-commit checks and reviews. Verify tag controls separately. This audit changed none of these settings.

## QG-02 — PR #125's combined lint/security job stops before several checks

**Classification:** Qualification failure, not evidence of a leaked credential. **Head:** `22c767a90f69d286642ef96e6b3c174836637a3c`.

Run https://github.com/djshellshoxxx/circula/actions/runs/44660259330, job https://github.com/djshellshoxxx/circula/actions/runs/44660259330/job/136036460029, fails at the secret-scan step. `scripts/check_gitleaks_config.py` expects scanner finding metadata for documents no longer present in the composed source; expected and observed path/rule/line sets differ. The full scanner follows this control in a stop-on-error shell, so it is not reached. Gofmt, vet, traceability, helper tests, golangci-lint and govulncheck are marked skipped.

Other checks on this head were successful when read: Ubuntu tests, Playwright/axe/keyboard, API contract/load, cross-builds, reproducibility, fuzz smoke, and Python export reader on Linux/macOS/Windows. Native Core cross-platform tests were skipped under the PR opt-in policy.

**Worker action:** Reconcile expected evidence with the exact composed documents, retaining positive/negative scanner controls and narrow exceptions. Rerun the job on the repaired revision. Do not remove the scan or broadly exempt audit directories.

## Other retained deployment/design risks

- **Mutable image inputs:** `deploy/Dockerfile:5–8,20` uses tags and explicitly defers digest pinning. Pin reviewed build/runtime image digests before release. No image compromise is alleged.
- **Broad trusted-proxy example:** `deploy/compose.yaml` defaults to `172.16.0.0/12`; safety depends on network isolation. Use actual proxy addresses and validate forwarding behavior. No ordinary remote bypass is established.
- **Restore reconciliation:** `internal/service/restore_credentials.go:15–19` preserves passwords, MFA material, recovery codes and membership snapshot semantics while invalidating bearer credentials. Reconcile post-backup authority changes before reopening a restored instance. This is distinct from the repaired restored-session/token issue.
- **Release authenticity:** Checksums/SBOMs are integrity checks, not an independently authenticated publisher signature when supplied beside the same archive. Settle the existing release-signing/attestation decision.
- **Archive tool hardening:** `scripts/verify-release.sh` and `scripts/scan-release.sh` extract before completing content/path checks. Normal inputs are same-process release artifacts. If intended for externally supplied archives, add member/type/link/size preflight and confinement. No traversal or arbitrary-write behavior was established or reproduced.

## Positive controls inspected

Backup encryption authenticates per-file salt/nonce context, chunk sequencing and final markers. Restore authenticates manifests by default, validates the actual staged copy and invalidates restored bearer credentials before swap. Private staging leases and Unix ownership/link checks address earlier cleanup defects. Restore markers preserve interrupted-swap recovery. Native TLS requires paired certificate/key configuration and restricts protocol versions. Metrics checks peer addresses, forwarding headers and optional bearer authorization. Release workflows pin action commits, use limited permissions and disable restored build caches; tagged source must be reachable from the default branch.

## Coverage and limits

Deep reads covered backup/encryption/staging, restore/swap, secret/config/key files, startup/HTTP deadlines, operational endpoints, audit anchor/mirror, export verification, deployment recipes, workflows and release scripts, plus relevant store/audit callers and prior notes. Some long files were traced selectively rather than proved line by line. Coverage manifest lists actual paths.

Both Core pins contain 283 production Go files and 21 migrations; these were inventoried, not all semantically proved. All 145 remote branches were inventoried. Fifteen canonical future manifests are disabled and on `HOLD`; neither Core pin contains `future/`. No native Windows/macOS deployment, real proxy, physical disk failure, production load test or external penetration test was performed.
