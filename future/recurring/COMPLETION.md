# Historical FNET-207 weighted completion criteria

This table is the archived assessment at `a755732a02dfe0eaa884691d59a6f748f96039e7`. The 84.25 subtotal is not a score or qualification claim for the recovered composition. Current exact-source checks and integration/manual gates are recorded in `VALIDATION.md` and `RECOVERY-PROVENANCE.md`.

Status: partial standalone candidate, disabled, integration HOLD. Each criterion has equal weight within its category. The score reports evidence, not release readiness. A mandatory unfinished gate prevents full qualification regardless of percentage.

| Category | Weight | Counted / total | Earned | Counted criteria and unfinished gates |
|---|---:|---:|---:|---|
| Domain functionality | 25 | 6/6 | 25 | Scoped templates; daily/weekly/monthly calendar; gap/fold/month-end semantics; unique reminders; pause/resume/skip/edit/cancel; deliberate private drafts. No automatic trades |
| Persistence/lifecycle | 15 | 4/5 | 12 | Atomic migration/schema validation; restart and concurrent workers; retention/departure with replay tombstones; closed-file restore/fault rollback. Production archive/erasure and scheduler recovery policy pending |
| Service/API/CLI | 15 | 3/4 | 11.25 | Transactional service; HTTP/native forms; shared CLI. Current core auth/inbox adapter and actual API contract qualification deferred |
| UI/accessibility | 10 | 2/4 | 5 | Escaped semantic forms; no-JS lifecycle and draft display. Real browser/keyboard/zoom and screen-reader/WCAG acceptance pending |
| Security/privacy | 15 | 5/5 | 15 | Current transaction authorization; scope/privacy/tombstones; bounded strict input; independent predecessor audit; independent coordinator correction review (688c22ac) |
| Automated verification | 15 | 4/5 | 12 | Race/unit acceptance; parser/calendar fuzz; restart/concurrency/fault cases; checksum/source isolation. Browser and operational end-to-end matrix pending |
| Documentation/operations | 5 | 4/5 | 4 | Member/developer guide; adoption/integration contracts; upstream notices; current handoff/evidence. Operator-selected backup/erasure/recovery procedure acceptance pending |
| Total | 100 | | 84.25 | Incomplete; no full qualification claimed |

Evidence is in VALIDATION.md and AUDIT.md. Prior source `0a677a14aea6362bebe16d37d7cd3c2254b8e836` passed the recovered baseline; that result does not qualify new commits. Current test results and review disposition must be read from the latest handoff. Integration gates are separately unauthorized; no root changes are proposed by this score.
