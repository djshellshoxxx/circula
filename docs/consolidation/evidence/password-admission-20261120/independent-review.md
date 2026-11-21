# AUTH-04 independent source and evidence review

Date: 2026-11-20 UTC. Reviewer: the separate `password_admission_review`
worker. Disposition: **no unresolved finding in the scoped AUTH-04 diff**.

## Exact source

The independent review examined implementation commit
`0bd96859034c7341015dbaf56a137f6cb84f5b55` on
`fix/core-password-admission-20261120`, based on
`21933a3c650690db5a8e1e44cc236d8d1b5cdb24`. The implementation worktree was
clean at the final review.

The consolidation owner subsequently cherry-picked that implementation as
`025ad9d3b23abd29c99dbe3ea376ff1622844313`, based on published Core source
`d6843366dc06d723f8d9efe07242997c858a6b3f`. This report independently checked
that both commits have stable patch ID
`8a28558de3622b8f4ee31f6421b90edb652da0f9` and that all 22 recorded Go source
file hashes still match the integrated `025ad9d` blobs. This is source
correspondence evidence, not a combined-source test result.

## Reviewed boundaries

- `internal/passwordwork/admission.go` has one normal process default with a
  fixed capacity of two operations and no queued waiters. Ticket bookkeeping
  rejects simultaneous reuse, releases exactly once, and retains a slot until
  synchronous work returns even when cancellation or early release is requested.
  It does not hold its mutex across the work callback or observation hook.
- Both production Argon2 call sites are covered: service password work in
  `internal/service/password.go`, and passphrase backup derivation in
  `cmd/circula/backupcrypt.go`. Normal service construction and backup derivation
  share the same process object. The explicit service override is documented
  for isolated test accounting.
- Service startup, registration/operator provisioning, sign-in and dummy work,
  password replacement, recovery/reset, and password-protected MFA operations
  acquire before their credential-specific repository effects and durable
  guessing reservations. One ticket spans sequential proof and replacement.
- Verification preserves the distinction between a false proof and an error.
  Cancellation is checked around admission and synchronous hashing, including
  after the reentrant test hook and before the actual service Argon call.
  Already committed attempt reservations remain charged. Later protected
  credential transitions stop on the propagated error.
- The source retains PHC parsing and accepted parameters, cryptographic
  strength/formats, dummy-work equivalence, durable attempt budgets,
  stored-password rechecks, recovery-code atomicity and existing authority
  checks. There is no observer-specific production bypass.
- HTTP handlers preserve `unavailable` as 503 with `Retry-After`; HTML
  credential-failure counters exclude it. MFA, recovery, reset and password
  handlers reach their existing busy-error response. Independent transport
  authentication, CSRF and request limits retain their original ordering.
  Backup constructors propagate the typed busy result and encryption refuses
  before publishing its header.

The only requested review adjustment was explicit isolated capacity two in
`internal/service/aud4_auth_test.go` and
`internal/service/password_proof_test.go`. The worker applied it and retained
the original stale-proof, session, MFA and recovery assertions. The final
focused race receipt covers the revised fixtures.

## Evidence checked

The reviewer parsed the completed [execution receipts](README.md), independently
recomputed their counts, and checked all **28 recorded SHA-256 entries against
committed Git blobs**: 22 source files and six run logs. There were no mismatches.

| Recorded execution | Top-level passed | Passed test/subtest events | Failed | Skipped |
|---|---:|---:|---:|---:|
| Focused service race | 20 | 50 | 0 | 0 |
| Backup/API/HTML adapter race | 11 | 11 | 0 | 0 |
| Final admission and stale-proof race | 17 | 44 | 0 | 0 |
| Credential regressions and bounded property fixtures | 96 | 219 | 0 | 1 |
| Final limiter race after lint correction | 8 | 8 | 0 | 0 |

Across these overlapping runs, 116 distinct top-level tests and 239 distinct
test/subtest names passed, with zero failed tests and one skipped opt-in
`TestLoginTimingStatistical`. The final changed-code lint log reports zero
issues. The earlier tool-path failure and corrected test-only SA4000 finding
are retained separately. All three adapted Argon2 benchmark variants completed
a one-iteration smoke run; this is not hardware or throughput qualification.

The [manifest](manifest.json) records exact commands, environments and hashes.
This review concerns those recorded values; a later lossless representation
change from a file/digest map to explicit path/SHA rows does not expand the
reviewed source or transfer these results to a later combined head.

## Limits

The conclusion establishes the reviewed aggregate operation-admission boundary.
It does not establish an RSS ceiling: valid historical PHCs retain their
recorded costs. Existing context-free backup constructors remain synchronous.
Custom embedding scopes cannot claim the normal application's shared default.

The reviewer performed source and evidence review only, with no source edits,
heavy compilation or remote mutation. Full combined-branch tests, provider CI,
new fuzz campaigns, vulnerability scanning and minimum-hardware qualification
remain separately owned integration gates. No future module was activated.
