# Governance weighted completion criteria

Partial; disabled; integration HOLD. Each row uses equal criterion weights internally. Scores describe observed acceptance evidence, not production readiness. Current correction tests and bounded independent coordinator review pass. External and manual gates remain open.

| Category | Weight | Counted/total | Earned | Criteria and open gates |
|---|---:|---:|---:|---|
| Domain functionality | 25 | 5/5 | 25 | Proposals/moderation; frozen eligibility/unique ballots; deadlines/withdrawal; advisory close/certification; read-only policy links/process disputes |
| Persistence/lifecycle | 15 | 4/5 | 12 | Atomic writes/idempotency; schema validation; restart/concurrency; backup restore/safety at saturation. Actual deployment retention/backup acceptance open |
| Service/API/CLI | 15 | 4/5 | 12 | Shared service; strict HTTP; authenticated CLI seam; no-JS forms/cursors. Actual host transaction/admin-MFA adapter remains separately deferred |
| UI/accessibility | 10 | 2/5 | 4 | Escaped semantic labels/no-default ballot; no-JS forms/errors. Browser, manual assistive technology and translations open |
| Security/privacy | 15 | 5/5 | 15 | Current authority; own-ballot isolation; replay/audit/transaction failures; strict inputs/callback regression design. Fresh independent coordinator correction review completed |
| Automated verification | 15 | 4/5 | 12 | Unit/race lifecycle; restart/concurrency/fault cases; bounded parser fuzz; vet/checksums. Real browser/accessibility acceptance unexecuted |
| Documentation/operations | 5 | 4/5 | 4 | Specs/contracts; current evidence/handoff; reuse/notices; operations/integration boundaries. Production operating acceptance open |
| Total | 100 | | 84 | Evidence-based partial acceptance; mandatory external/manual gates remain open |

The inherited implementation/audit and fresh baseline at `e4381aa19292f1d4b24f09e1de51f3e2e5a2930b` support the existing criteria. Current source `8b74e7db0e237df15acb46a27609f6015eadb4ed` passed fresh tests; see VALIDATION.md for genuine command outcomes. A higher total cannot override unresolved mandatory privacy, persistence, accessibility or audit gates. No passing standalone test permits activation or publication.
