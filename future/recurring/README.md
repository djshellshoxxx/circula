# FNET-207: Recurring trade templates and reminders

**HOLD — recovered standalone service composed with the reviewed calendar repairs. Combined-source review and qualification are pending. Not integrated, activated, beta-qualified or released.**

The owner authorized independent development of all fifteen future modules, preserving one-feature-at-a-time integration after both the feature and Circula core qualify. The reconciled candidate changes `future/recurring/` and its dedicated `.github/workflows/future-recurring.yml` only. It preserves the original isolated target and reviewed calendar candidate; exact archive and candidate provenance is recorded in `RECOVERY-PROVENANCE.md`. The feature is disabled unless explicitly constructed with `Enabled: true` and a current policy adapter.

## Implemented standalone scope

- Explicit SQLite migration, private owner/network-scoped CRUD/direct lookup, optimistic revisions, twenty-template quota and durable idempotency for mutations/manual draft actions.
- Daily/weekly/monthly structured calendars with pinned `robfig/cron` parser/date iteration and exact timezone-interval wall-time resolution. No arbitrary expressions, descriptors or TZ overrides. DST gaps skip, folds choose first, missing month-days skip, and downtime never replays a historical backlog.
- Unique logical-window firing and private reminder reference in one writer transaction. An optional core-inbox callback receives that **same transaction** and its failure rolls back the firing/reminder/audit. No remote callback, automatic proposal, acceptance or ledger API.
- Explicit bounded continuation scans, current transactional membership/policy checks, inactive-member suspension, pause/resume/skip/cancel/edit audit, and a 20-hour timezone-edit refiring guard.
- Deliberate private draft/prefill creation with fresh current terms/capacity authorization, random distinct intents and stable retries. Cancelled templates cannot create drafts.
- Unregistered authenticated JSON/native-form HTTP adapter with host CSRF hooks, no-store/CSP/body bounds and strict duplicate/unknown JSON checks; escaped no-JavaScript create/edit/state forms and draft review.
- Shared embedding CLI API for list/save/state/draft/preview/inbox, local candidate OpenAPI, member/developer help and operations guide.
- Explicit maintenance retention and departure hooks; minimal operation/firing tombstones remain to prevent replay. Closed-database backup/restore and failed migration rollback tests.

## Isolation and ownership

This is an independent nested Go module. It has no imports from Circula core or another unfinished feature, no root route/migration/config/dependency changes, no init/background registration and no automatically opened database. Supply one explicitly owned SQLite connection and a trusted authorization callback that reads current state on the supplied transaction. `Remind` is an adapter seam for the existing in-app inbox, not a dependency on the unfinished email/push feature. No signer, transport, user credentials or ledger writer is present.

The reviewed pure `Due` and `DueWithSchedule` APIs remain intact, including structured daily/weekly/monthly rules, exact timezone-interval resolution and original daily-key compatibility. Their tests are preserved in `due_with_schedule_test.go` and `schedule_regression_test.go`. The durable service uses the separately recovered `Calendar`, current authorization and SQL uniqueness. `Calendar` rejects a zero clock by its existing service contract; the pure helper continues to accept a valid zero instant. No production authorization claim is made for standalone activity booleans.

Pinned dependency provenance and actual downloaded notices are in `THIRD_PARTY_NOTICES.md` and `third_party/`. The production library imports only cron; modernc SQLite is a test driver. Read `GUIDE.md`, `openapi.yaml` and `VALIDATION.md` for exact embedding/operational contracts and reproduced evidence.

## Remaining qualification gates

A fresh independent audit fixed nine findings with adversarial regressions; see `AUDIT.md`. Actual Circula transactional membership/terms/limits and inbox adapters, per-network opt-in, scheduler restart/fairness orchestration, root API/CLI/help routing, full ledger regression, browser keyboard/WCAG/screen-reader qualification, backup operational approval and final release/license decisions remain separate gates. The isolated service's tests cannot qualify these.

Do not merge, auto-merge, deploy, tag or enable this branch. After independent feature and core qualification, the owner must select a single feature for a separate integration change, then test against that exact core version. No other future module may become a dependency implicitly. Disable/retention/restore procedures are in `GUIDE.md`.

Current reconciliation branch: `master/fnet207-recovery-20261119`. See `RECOVERY-PROVENANCE.md` for exact source composition and pending qualification. `COMPLETION.md`, `FINAL-HANDOFF.md`, `HANDOFF-MILESTONE-*.md`, `AUDIT.md`, `SECURITY-AUDIT.md`, and the retained evidence directories describe the archived branch and historical results; their percentages and passing checks are not current combined-source qualification. The dedicated workflow checks the explicit recovered dependency pins, module isolation, disabled/HOLD state, formatting, vet, full SQLite service tests and bounded calendar fuzzing. Linux/macOS use race detection; Windows runs ordinary native tests. All three jobs must be evaluated at the new publication head.
