# Security review continuation — FUX-105

Integration HOLD. The prior independent audit is preserved verbatim in AUDIT.md, including its baseline, findings, fixes and qualification limits. The new search delta received a fresh independent review by the recovery messaging worker on 2026-11-16, separate from its implementation team.

## Reproduced baseline

The exact newer baseline `3b4fbe4497827072df15f75e8e518f89925d6435` passed the default modernc race suite on recovery. Its retained findings cover expired-hold resurrection, admission quota churn, strict JSON/Unicode, response truncation, operator paths, retention shortening and current counterparty policy. Older outage work is preserved separately, not silently discarded.

## Independently reviewed search delta

Review scope: runtime delta from `3b4fbe4497827072df15f75e8e518f89925d6435` to `0cc6585f2e6c998f5208c6fe569ea4afbf9cc392`, as preserved at recovery HEAD `d4498fc7a6bce9c99d27a9f717149e755e07f56b`. Inspected service SQL and transaction/authorizer paths, HTTP dispatch/method/CSRF/body decoding, no-JS rendering, CLI input/transport and the corresponding tests and OpenAPI contract. No production code was changed by this review.

- Every search uses the existing transaction and fresh member authorizer, then verifies network/participant/expiry/purge visibility. No separate authority is introduced.
- Hidden-through positions apply inside the same transaction. A legal hold never exposes expired messages through ordinary search.
- SQLite `instr(body, ?)` binds user text. No dynamic SQL, wildcard interpretation or additional index copies of private content are introduced.
- Queries are capped at 128 UTF-8 bytes and reject blank text, invalid UTF-8 and control characters. Result pages are capped at 100 and use monotonic sequence cursors.
- Browser/API searches use POST with current CSRF validation and no-store responses. CLI reads phrases from stdin. Search phrases are absent from query strings, redirects and process arguments.
- HTML uses existing escaped templates. Search does not update message activity, unread counts, read positions, retention, blocks or moderation.

New regressions are `TestSearchVisibilityPagingRestartAndRetention`, `TestSearchValidation`, `TestSearchAPIAndNoJSPrivacy` and `TestCLISearchUsesBoundedStdinAndPOST`. All passed under the race detector with the default driver.

No confirmed new authorization, SQL injection or content-disclosure defect was found in this bounded independent delta review. This is not a whole-program security certification. The recovered pre-cancelled-context test was preserved and executed; an independent boundary regression verifies that searching one second before expiry does not extend ordinary content visibility.

Search bounds response size, not the number of rows examined. A missing substring scans all remaining messages in the selected conversation, under the existing transaction's SQLite writer reservation. The diagnostic 10,000-message benchmark now verifies exact hit/miss counts and seeds a consistent thread sequence. Its timings are recorded in VALIDATION.md. Production workload limits, cancellation during a large in-flight scan, large retained conversations and contention across concurrent clients remain performance qualification gates; a 100-row response cap does not establish a latency bound.

Real-browser/WCAG/manual screen-reader evidence remains open. The prior module advisory scan is historical evidence; this continuation adds no dependency and does not claim a newly executed vulnerability scan. Host operational and ledger qualification remains separate.

## Fresh callback and form-boundary audit — milestone F

Reviewed successor of `ea06857109b145d986d7b6f3ac69f7cf72efb97e`. This fresh worker was separate from the search implementation and previous recovery audit. The following gaps were reproduced with synthetic fixtures and repaired in `b81473dd929b541a6d937cdf71c3a157fa32227f`, followed by query guard `0b1861f50ef1fcecb65eaf3e4227a647ce2dcd58`.

| Finding | Reproduced behavior | Correction |
|---|---|---|
| Resolve could consume a malformed form and cache partial values before module parsing | A discarded invalid-percent field was accepted; HTTP 303 and one message persisted | Parse the capped ordinary form before Resolve, retaining valid PostForm for host CSRF |
| Token or CSRF callback could revoke/replace the session while the old actor remained active | Private pages rendered HTTP 200; revoked/replaced CSRF callbacks still sent a message | Resolve the exact same valid actor after callbacks and reject cancellation before private reads/mutation |
| Search read private messages before Token callback | Callback membership/session revocation still released the earlier search result | Obtain token and reauthorize before the existing transaction-scoped Search operation |
| Raw query parsing was unbounded before Resolve and malformed query fields could be discarded | Oversized query reached host; observed HTTP403 with one host invocation | Cap RawQuery at 8192 bytes and reject ParseQuery errors before host callbacks; preserve existing route pagination fields |

Regressions cover inbox, thread, search and maintenance pages; revoked/replaced sessions, membership revocation, cancellation and unchanged controls; malformed form rejection with no persisted message; CSRF mutation denial; query syntax/budget rejection and exact-boundary compatibility. Current frozen full race suite and vet passed. The coordinator independently reviewed both the callback/form correction and the final query correction, including their regressions, and found no further issue in those bounded changes.

These are bounded in-handler callback checks. They do not create a globally atomic host session/store transaction. Resolve must be read-only, return the current actor consistently and honor cancellation. Pre-handler middleware must not consume a malformed body and discard its parse error before invoking this handler. Store membership/content authorization remains inside existing transactions. No schema, dependency, core registration, remote service or search indexing was added.
