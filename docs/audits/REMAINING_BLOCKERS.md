# Remaining blockers and next actions

> Current composition: [combined core review and qualification checkpoint](CORE_COMPOSITION_20261119.md) records the reconciled PR90/96/99/101/102/104 source, focused passing checks, unchanged main and pending full provider gates. The [bounded branch refresh](CONCURRENT_BRANCH_SNAPSHOT_20261119.md#bounded-refresh--112708-utc) records 123 retained refs at 11:27:08 UTC. Earlier sections and receipts keep their named source and observation time.

Current audit: 2026-11-18 Pacific / 2026-11-19 UTC. This list distinguishes external/owner gates from ordinary unfinished engineering. Broad audit authorization is not interpreted as approval to choose a conflicting governance policy or bypass required CI.

## Integration qualification

| Item | Evidence / effect | Concrete next action |
|---|---|---|
| API-036 final qualification | [Implementation and measured evidence](API036_COMPLETION.md) are published in PR #101, including migrations 0020/0021, bounded reverse/forward reads, bulk revocation and cursor hardening. Focused production Go packages and large retained-history measurements passed. The first full suite found two fixture compatibility failures; narrow repairs preserve their assertions. | Require the repaired exact-head full/race/vet/check, mutation fuzz, provider browser and Linux/macOS/Windows results. Local Playwright is blocked by Chromium socket() EPERM before assertions. No historical or skipped check qualifies this candidate. |
| Combined core CI | Source PR passes do not qualify PR #90. Local pure-Go SQLite compilation repeatedly exceeded the workspace process budget. | Obtain successful current-head Linux race/property/migration, lint/vulnerability, strict-JSON fuzz, contract/load, browser, native Windows/macOS, build and reproducibility jobs. Fix any real failures; keep hard timeouts and checks. |
| Required reviews/protection | Repository ruleset collection was empty; legacy protection read returned 403. The latter is unknown configuration, not absence of policy. | Use the normal expected-head PR merge API only after checks/reviews. If it rejects a required gate, retain the concrete candidate and record the actual rejection. |
| Legacy dispute mapping | Some old same-time/backward-clock records cannot prove the originally observed policy; missing mappings abort migration. | Follow documented migration backup/inspection procedure; resolve missing mappings from evidence without altering financial/outcome/audit history. |
| Local browser environment | Isolated PWA Chromium qualification was blocked before assertions by sandbox UNIX-socket denial. | PR #95 executed the existing harness successfully at f25d037 (run44496951525/job135466500874). Its launch mode is unchanged. That is historical synthetic-flow qualification; the later f153bf5 core refresh and final combined browser suite still require their own current-head checks. |

## Owner and external gates

| Gate | Affected work | Required decision / evidence |
|---|---|---|
| Reviewer account recovery | #83, AUD4-F02 | Select a complete recovery/decision-independence invariant and test revoke/reset/regrant. Current-role-only restriction is incomplete. |
| Independent dual-control approvers | AUD4-F01, SEC-011 | Decide quorum, operator-rooted approver or proposer delay; qualify appointment and credential-reset attack paths. |
| Operator/break-glass authority | DEC-014, SEC-013, #85 | Reconcile session-bound temporary powers with permanent role assignment; authenticate independent operators; settle waiting/objection/expiry rules. |
| Activity-signal policy | #86 / independent repaired SEC-026 branch | Decide subject visibility, threshold/disable authority and retention. Defaults are repaired to off and member-readable policy settings restored; those fixes do not supply the missing policy. |
| Privacy retention | SEC-014/018, DEC-020 | Decide retention of last activity, time zone, challenge/reason fields, free-text audit reasons and post-erasure WAL treatment. |
| Numerical release criteria | DEC-023, QA-040 | Record approved budgets/coverage/performance/login-timing thresholds on reference hardware. Existing 100,000-sequence evidence does not approve an unconfirmed threshold. |
| Manual accessibility | UX-055/056, QA-035 | Perform and record keyboard/screen-reader walkthroughs for the release build. Automated axe/keyboard checks are useful but insufficient for this gate. |
| External security review | SEC-033 | Obtain the defined independent professional assessment and retain the report for the supported release lifetime. AI-assisted engineering review is a separate record. |
| Licence, jurisdictions and support | DEC-004 and open product/operations decisions | Confirm the project licence, pilot jurisdictions, support commitments and relevant notice defaults before release. |
| Future-module adoption | All 15 modules | Verify core adapters/interfaces, permission mapping, migration ownership, data retention, operations, key custody where applicable, accessibility and product acceptance before removing HOLD/enabling. |

## Engineering still unfinished

These are not silently converted into owner blockers. Continue them after the current repair candidate is qualified; their presence means the whole specification is not complete.

- **UX-047:** the 525-key catalogue now includes shared refusals/validation, administrator reset, marketplace/directory and browser freshness/session warnings in addition to earlier authentication/onboarding/session/financial flows. Focused messages, actual templates and Node checks pass. Other templates, API/service error text, complete date/number/locale handling and full combined browser qualification remain. Translation languages are a separate owner decision.
- **Storage-fault qualification:** extend the existing disk-full, pre-commit WAL-write and checkpoint database-write tests to read/fsync/truncate, checkpoint sync/truncate, other device errors and uncertain commit outcomes. Preserve genuinely committed entries through restart and retry.
- **Requirement evidence:** three direct automated citations were supplied and SEC-031 was honestly changed to partial because the approved reporting/contact channel is unresolved. The remaining 15 uncited built rows are classified in BUILT-REQUIREMENT-EVIDENCE.md by automated, inspection, manual and deployment criteria. Complete the genuine missing acceptance evidence; do not add vacuous tests. A symbol citation is not a passing run or full acceptance proof.
- **Future implementations:** complete missing production adapters, persistence coverage and user journeys independently on each module branch. Low default-package coverage in QR/discovery/notifications reflected skipped SQLite paths and must not be sold as full validation. The expanded tagged results reduce, but do not eliminate, those gaps.
- **Release/deployment qualification:** first-release drills, per-target smoke execution, staging alerts and reference-hardware performance remain separate work. No deployment was requested or performed in this audit.

## Preservation and continuation

All starting refs are retained. Historical source that is patch-equivalent, behaviorally recovered or superseded remains linked in [BRANCH_CONSOLIDATION_MATRIX.md](BRANCH_CONSOLIDATION_MATRIX.md). Original PR heads are not force-pushed. Feed, notifications, signals and GOV-021 fixture repairs have separate publication/qualification records. Revisit current heads before resuming; do not assume a passing historical run covers later edits.
