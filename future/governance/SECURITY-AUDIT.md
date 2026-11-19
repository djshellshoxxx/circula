# Governance master delta review

Predecessor `e4381aa19292f1d4b24f09e1de51f3e2e5a2930b` already has a fresh independent audit in AUDIT.md. This continuation reviews that source against the master requirements. New correction source is `7b8e64a`; its independent correction review and actual validation are recorded below when completed.

## Findings

1. URL.Query().Get silently ignored malformed/duplicate/unknown query fields, and no byte budget preceded host authentication. POST executed before list-cursor validation. A shared strict parser now caps raw query bytes at 8192, permits only one proposal/format/cursor field, validates format/proposal and rejects ambiguous detail-plus-cursor requests. Signed cursor validation now precedes any mutation. If a proposal remains in the current form URL, it must equal the parsed command proposal. Valid forms continue posting to the current URL; no-JavaScript behavior is preserved.
2. GET detail/list and POST error forms called the trusted CSRF provider after reading private state. That callback could revoke or replace the session and still expose the previous private projection. Token selection now precedes a fresh same-identity session decision and service reads. Service independently checks current membership in its own SQL transaction. Tests cover revoked/replaced sessions, revoked membership, cancellation and unchanged controls across all three paths.

These are bounded callback and request-validation corrections. They cannot make arbitrary concurrent host session changes atomic with a database transaction or network response. Hosts must honor the current-authority transaction contract and bound callbacks; no external network I/O belongs in Authority methods. No new schema, dependency, automatic policy application or ledger write is introduced.

## Scope and limits

Historical audit remains valid only for its recorded source; the new regressions qualify the correction separately. Browser/manual accessibility, translations, protected production session/admin-MFA adapter and retention/backup operations remain unqualified. All features stay disabled and integration HOLD. No remote publication occurred.

## Follow-up findings and evidence

The focused tests exposed an ignored private reload failure on POST error rendering. Correction `86624d5` now returns unavailable when a nonempty proposal cannot be freshly read; no private projection is rendered. The unchanged conflict fixture correctly preserves 409.

A separate fresh receipts auditor reported Go's cached ParseForm error behavior. Governance reproduced the same defect: a CSRF provider using PostFormValue discarded the first malformed-percent parsing error; the later ParseForm call returned cached partial values and a proposal was created. Correction `8b74e7db0e237df15acb46a27609f6015eadb4ed` performs bounded parsing and duplicate/unknown/UTF-8 validation before host callbacks. Red evidence:303 with one persisted proposal. Final focused/full-race/vet/checksum validation passes. No extra dependency or schema changed.

The coordinator independently reviewed the initial `7b8e64acc066ea3283d4168e774fe282e54bc02d` query/callback correction and tests, with no additional issue identified. The coordinator also independently reviewed `86624d5` and `8b74e7db0e237df15acb46a27609f6015eadb4ed`: private reload denial and bounded pre-host strict form validation were consistent with their contracts; no additional issue was identified. This bounded delta review does not replace the predecessor full audit or external production qualification.
