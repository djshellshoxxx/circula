# Independent standalone governance audit — 2026-11-16

Reviewed implementation `6eed7eb` independently against the adopted isolated module scope, SPEC-SNAPSHOT.md, UPSTREAM-RESEARCH.md and the actual service/web/CLI code. Integration remains **HOLD**. This review authorizes no production registration, publication, deployment, policy application or ledger authority.

## Findings fixed

1. **Silent writes could acknowledge nonexistent ballots or outcomes.** SQLite `RAISE(IGNORE)` can return success without persisting an insert/update. All service mutations, immutable outcome writes, idempotency rows, audit rows and schema-version writes now require exactly one affected row. One transaction rolls back every dependent write on failure. The same checked schema-row write acquires the SQLite writer lock before current authorization and mutable policy/snapshot reads.
2. **Existing schemas were trusted by name.** Migration now verifies required table/index definitions and object kinds, exact version cardinality and constraint literals inside the same migration transaction. Incompatible indexes/tables/views cannot be accepted merely because `CREATE ... IF NOT EXISTS` succeeded. Schema comparison preserves quoted literal case.
3. **Stored payloads could disagree with indexed tenant/state facts.** Proposal reads/listing bind body network/ID/author/state/revision to the selected columns, validate bounded content/lifecycle, and fail closed on corruption. Direct detail reads validate frozen electorate cardinality; result/own-ballot/comment/dispute records are bounded and validated. Close verifies ballot eligibility and original cast-time boundaries before counting. Failed reads return no partial private detail.
4. **Administrative idempotent retries ignored role revocation.** Persisted replies are bounded and bound to their actual same-network proposal and original command; retries recheck current operation authority. Recorded votes remain safely retryable after closure, while revoked administrators cannot replay certify/link/resolve acknowledgments. The provider-owned electorate slice is copied before sorting.
5. **Quota churn and saturation affected lifecycle safety.** Proposal quota counts all retained history, so create/withdraw cannot evade it. Normal commands have a 10,000-intent/member cap; withdrawal, hide and required close/review safety actions remain available at saturation. Already-hidden content cannot generate unlimited fresh hide actions. Per-member comment/process-dispute bounds and duplicate unresolved concern checks prevent one actor consuming every per-proposal slot.
6. **Deadline results could be discarded through late withdrawal.** An open proposal cannot be withdrawn at or after its deadline; its original ballots remain available for explicit close. Existing successful pre-deadline withdrawals retain stable idempotent retry behavior.
7. **JSON/form parsing accepted ambiguous or replaced input.** Shared HTTP/CLI parsing rejects duplicate names (including case aliases), invalid UTF-8, unpaired UTF-16 escapes, unknown fields, excess nesting and trailing data. Forms reject duplicate/unknown fields and invalid Unicode/revisions. Network/CLI requests remain limited to 64 KiB. Stored projections allow a separately bounded 128 KiB because valid 12,000-byte text can expand under JSON escaping; a regression proves documented content limits remain readable.
8. **Path/cursor/CLI boundary evidence was incomplete.** Host web rejects absolute, protocol-relative, backslash/dot paths and oversized/malformed cursors. Forms omit `action` and stay on the current document; CSP restricts forms to self. The CLI accepts an explicit `list AFTER_ID` continuation, preserving the same private service boundary.

All findings above have regression coverage. No remaining blocking defect was identified in the reviewed standalone candidate during this bounded audit; that is not a proof of security or completion of the external gates below.

## Requirement-to-evidence matrix

| Requirement | Evidence and disposition |
|---|---|
| Draft content, schedule, comments/category and lifecycle | Original lifecycle suite plus bounded stored-record validation, retained-history quota, escaped maximum-size content and immutable deadline tests. No author editing of opened content. |
| Frozen electorate and current membership | Opening occurs under checked writer transaction; copied, sorted, distinct bounded roster is stored once. Existing suspension/non-snapshot tests and borrowed-slice regression pass. Actual core Authority must read the same transaction. |
| One eligible member ballot; duplicate/late/tenant denial | Composite unique/FK schema, current principal, original cast bounds, race fixture and ignored-write rollback regressions pass. No anonymous or one-human uniqueness claim. |
| Deterministic advisory outcomes and abstain | Existing tally fixtures, close-time record validation, empty-vote outcome, immutable result insertion and no late withdrawal. No quorum/pass rule is invented. |
| Separate independent certification and policy linkage | Existing challenged-reviewer/report resolution tests and new revoked-admin retry denial. PolicyExists remains a read-only current host contract; no policy or ledger writer exists. |
| Moderation, process concerns and privacy | Own ballot only; hidden content erased; bounded validated comments/disputes; duplicate unresolved concerns rejected; current same-tenant reads and no partial error DTO. Existing comment/report conflict tests pass. |
| Durable schema/idempotency/audit and concurrency | Exact schema/object/constraint checks, checked affected rows, intent and audit failure rollback, existing two-connection race, restart, migration rollback and VACUUM restore tests pass. |
| Saturated history does not prevent safety actions | Regression seeds more than 10,000 intent rows, refuses normal edit but permits freshly authorized withdrawal. Hide/withdraw/resolve/close/certify/link are exempt from normal intent cap and retain their individual lifecycle/conflict checks. |
| HTTP, CLI and no-JavaScript forms | Existing labeled no-default radio/edit/lifecycle/moderation/dispute forms; strict JSON/form/CLI rejection and hostile-path regressions; bounded cursor; no-store/no-referrer/CSP and RFC9457 responses. No core routes registered. |
| Reuse, isolation and ledger safety | Pinned GOV.UK markup provenance and complete dependency notices preserved. No root file, cross-feature import, automatic job, QuietVote integration, role grant, credit movement or automatic policy application added. |
| Browser, core integration and operations | **Pending** actual Circula auth/admin-MFA/store/policy-history adapter, full core/API/unchanged-ledger regression, real browser/WCAG/screen-reader/translation checks, deployment/retention approval and owner-selected single-feature integration. |

## Reproduced validation

Go 1.26.11 linux/amd64, synthetic fixtures and explicit standalone SQLite database only:

- `GOGC=20 GOMEMLIMIT=450MiB GOMAXPROCS=1 go test -p 1 -race -cover -timeout 90s ./...` — PASS; final coverage recorded in VALIDATION.md.
- `go vet ./...` — PASS.
- `go mod verify` — PASS.
- Parser-only fuzz: `go test -run=^$ -fuzz=FuzzAuditStrictJSON -fuzztime=3s -parallel=1 strict.go strict_fuzz_test.go` — PASS, 18,890 executions. It deliberately avoids instrumenting unrelated SQLite dependencies.
- `govulncheck` v1.1.4 runtime symbol scan — no vulnerabilities found.

Browser/manual or whole-core results are not inferred from static template or standalone unit tests. Historical immutable governance retention is a deliberate reference policy; the host must approve its actual retention/disable procedure before integration. No destructive ballot-history cleanup or retrospective voter identity change is introduced by this audit.
