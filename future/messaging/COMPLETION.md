# Weighted completion — FUX-105

**90% of the weighted standalone acceptance checklist is evidenced. Partial; integration HOLD.** This score is not release approval. The baseline is the newer local implementation at `3b4fbe4`, with the continuation adding required private search and fresh independent search and callback/form-boundary audits.

| Category | Weight | Verified criteria | Score | Evidence and remaining gate |
|---|---:|---:|---:|---|
| Domain functionality | 25 | 17/17 | 25 | Conversations; order; read positions; idempotent sends; blocks; report handling; moderation; retention; own-view hiding; legal holds; quotas; search; pagination; network scope; literal UI; explicit privacy claims; no ledger authority. Store/audit/search suites cover these. |
| Persistence and lifecycle | 15 | 8/8 | 15 | Explicit schema; migration rerun; restart; rollback; two-pool concurrent writes; retention boundary; purge; consistent backup/restore. |
| Service/API/CLI adapters | 15 | 5/5 | 15 | Mandatory authorizer; strict JSON; bounded HTTP; member/operator CLI; POST search parity. Host production installation is a separate integration gate. |
| UI and accessibility | 10 | 3/6 | 5 | Escaped labeled forms, no-JS handler flows, CSRF/private errors verified. Real browser, automated WCAG and manual screen reader unverified. |
| Security and privacy | 15 | 5/5 | 15 | Tenant/fresh membership, own visibility/retention, transactional abuse limits, previous independent audit and fresh independent search-delta review evidenced. Scope and performance limits are explicit in SECURITY-AUDIT.md. |
| Automated verification | 15 | 4/6 | 10 | Unit/negative, race/concurrency, SQL restart/restore and fuzz suites exercised. Broad performance qualification and actual browser automation remain unverified. |
| Documentation/operations | 5 | 7/7 | 5 | README, contracts/OpenAPI/help, notices/reuse, migration, integration, validation, exact continuation handoff present. Operational host qualification is explicitly deferred. |
| Total | 100 | — | 90 | No category is silently waived. |

Tests measure the adopted standalone contracts, not deployed host adapters. Search adds no dependency and preserves network/participant/hide/expiry policies. Core authorization/CSRF/retention/audit integration, platform backups and complete core/ledger regression remain outside this authorized development phase. They are not falsely reported as passed. Final standalone qualification remains blocked by unavailable browser/manual evidence and broader performance qualification; the 10,000-message diagnostic benchmark does not close that gate.

Milestone F repaired newly reproduced HTTP callback/form/query boundary defects and reran the full race suite. This maintains existing criteria rather than increasing their count; the score remains 90%. See HANDOFF-MILESTONE-F.md for the exact tested successor.
