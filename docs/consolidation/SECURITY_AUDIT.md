# Core consolidation security review

Baseline: `ead63b5ce24b2bcf8352d75fd0d1ff7be711f625`.
Candidate: `integration/core-recovery-20261119`.

## Confirmed findings and repairs

### Export authorization could become stale before the data snapshot

**Impact:** a previously admitted administrator could still export network
data after demotion, suspension, or removal committed before the read snapshot.
Fresh own-data exports also lacked the erasure guard already used for artifact
downloads.

`Service.Export` now uses the same authorization helper for admission and the
data snapshot; `AuthorizeExportDownload` uses that helper too. Current roles,
membership, scopes, network binding, MFA/break-glass rules, and erasure are read
inside the snapshot before any sink file is created. The helper performs no
writes within that read transaction.

Actual SQLite/service regressions fail before the repair and pass after it.
Already-erased requests create no files or new export audit event. Valid
admission followed by revocation retains the truthful audit of that attempt.
Suspended/departed members retain access to their own un-erased history.

This closes the admission-to-snapshot gap. It does not claim continuous
cancellation of a web stream after a consistent snapshot has already been
authorized and opened.

### Interrupted exports could look like successful partial ZIP archives

**Impact:** the HTML adapter ignored export-generation and ZIP finalization
errors after creating a ZIP writer. It could return normally with an incomplete
archive. Its recovery middleware also swallowed the standard HTTP abort
sentinel.

The adapter now finalizes only a successful export, logs an interruption once
through the configured redacting logger with its request ID, and aborts the
response on generation or finalization failure. Web recovery propagates
`http.ErrAbortHandler`, matching the existing API behavior. Early refusals
still produce normal HTML errors before download headers are set.

Focused CSV/JSON regressions fail before the repair and pass after it. They
exercise real SQLite rows, the actual recovery layer, injected adapter read and
response-write failures, invalid partial ZIP detection, complete-archive CRC
reads, and early-refusal controls. These injected adapter faults are not a
substitute for physical SQLite device-failure qualification.

## Other correctness findings

- Listing continuation discarded query/type filters and encoded the owner view
  as `mine=true`, which its own parser did not recognize. The repaired link
  preserves the requested view and uses lookahead to avoid empty final pages.
  Existing service visibility checks remain intact.
- Traceability qualification accepted an already regenerated report containing
  duplicate definitions. The recovered guard rejects those reports without
  rejecting valid cross-references or consecutive rule rows.

## Review boundaries and active work

The parallel ledger/storage review found no additional narrow confirmed defect
in its inspected accounting/atomicity/migration subset. This statement is not
proof that the repository has no vulnerabilities. Existing financial,
authentication, password-proof, session, and cursor repairs in active PRs
#90/#96 remain subject to their own current revision verification. Future feed
security stays on #92's isolated module branch.

No security checks, ledger invariants, required CI jobs, branch protection, or
policy HOLDs were disabled. No production data was used. Full candidate race,
dependency/security, browser, and integration outcomes are tracked separately
in TEST_RESULTS.md. This AI-assisted review does not close SEC-033's independent
external review gate.

## Committed-response verification and concurrent security work

The follow-up `TestWebExportCommittedReadFailureAbortsHTTP` uses the complete
operations/web middleware chain and a real TCP client. In each CSV/JSON case,
the client receives 4 KiB before releasing a controlled repository read fault.
The original main revision reports a normal completed body; the repair yields
`io.ErrUnexpectedEOF` and an invalid ZIP. This verifies transport interruption,
not only an in-process panic sentinel.

The #99 source review (5973679547) records a separate live-session authority
gap in #90's token mutation path. That repair is in #99, not this independent
export candidate. #90 must not merge alone before the repair. None of these
source reviews replaces the pending combined current-head security/regression
checks or the independent external security review requirement.
