# Remaining blockers and next actions

Current audit: 2026-11-18 Pacific / 2026-11-19 UTC. This list distinguishes external/owner gates from ordinary unfinished engineering. Broad audit authorization is not interpreted as approval to choose a conflicting governance policy or bypass required CI.

## Integration qualification

| Item | Evidence / effect | Concrete next action |
|---|---|---|
| Combined core CI | Source PR passes do not qualify PR #90. Local pure-Go SQLite compilation repeatedly exceeded the workspace process budget. | Obtain successful current-head Linux race/property/migration, lint/vulnerability, strict-JSON fuzz, contract/load, browser, native Windows/macOS, build and reproducibility jobs. Fix any real failures; keep hard timeouts and checks. |
| Required reviews/protection | Repository ruleset collection was empty; legacy protection read returned 403. The latter is unknown configuration, not absence of policy. | Use the normal expected-head PR merge API only after checks/reviews. If it rejects a required gate, retain the concrete candidate and record the actual rejection. |
| Legacy dispute mapping | Some old same-time/backward-clock records cannot prove the originally observed policy; missing mappings abort migration. | Follow documented migration backup/inspection procedure; resolve missing mappings from evidence without altering financial/outcome/audit history. |
| Local browser environment | Isolated PWA Chromium qualification was blocked by sandbox UNIX-socket denial. | Use the existing CI browser mode or a supported browser environment. Do not disable browser sandbox protections. |

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

- **UX-047:** setup/join and existing authentication/shared/financial flows now have catalogues, but shared refusal construction, administrator reset pages, other templates, JavaScript, API/service error text and complete date/number/locale handling still need systematic migration. Translation languages are a separate owner decision.
- **Session-inventory scale:** response pagination is implemented, but the repository materializes all retained rows for one owner and there is no per-person row-count cap. Add an indexed noncredential selector and bounded repository pagination after choosing its migration/interface shape; do not claim current database work is bounded by the API limit.
- **Storage-fault qualification:** extend the existing disk-full, pre-commit WAL-write and checkpoint database-write tests to read/fsync/truncate, checkpoint sync/truncate, other device errors and uncertain commit outcomes. Preserve genuinely committed entries through restart and retry.
- **Requirement evidence:** resolve 19 “built” rows without any test citation, refresh stale spec status notes against actual behavior and record per-criterion execution/manual/inspection evidence. A symbol citation is not a successful test or full acceptance proof.
- **Future implementations:** complete missing production adapters, persistence coverage and user journeys independently on each module branch. Low default-package coverage in QR/discovery/notifications reflected skipped SQLite paths and must not be sold as full validation. The expanded tagged results reduce, but do not eliminate, those gaps.
- **Release/deployment qualification:** first-release drills, per-target smoke execution, staging alerts and reference-hardware performance remain separate work. No deployment was requested or performed in this audit.

## Preservation and continuation

All starting refs are retained. Historical source that is patch-equivalent, behaviorally recovered or superseded remains linked in [BRANCH_CONSOLIDATION_MATRIX.md](BRANCH_CONSOLIDATION_MATRIX.md). Original PR heads are not force-pushed. Feed, notifications, signals and GOV-021 fixture repairs have separate publication/qualification records. Revisit current heads before resuming; do not assume a passing historical run covers later edits.
