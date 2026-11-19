# FNET-207: Recurring trade templates

**HOLD — isolated development, incomplete feature, not approved for integration or release.**

The owner authorized independent future-feature development on 2026-11-16, while explicitly deferring integration until Circula and the feature are complete. This authorizes this branch's standalone development; it does not authorize enabling, merging or deploying the feature. Historical implementation prohibitions in the copied planning documents are superseded only for this isolated development scope.

## Isolation contract

- Branch: `future/fnet-207-recurring-20261116`. Baseline: `2eacf7718f5bb3da3be5a288024af879076a9597`.
- Feature implementation remains entirely inside `future/recurring/`. The dedicated `.github/workflows/future-recurring.yml` checks this isolated module. No application imports, routes, migrations, root dependencies, feature flags, existing workflows, runtime jobs or configuration are changed.
- This nested Go module has no third-party dependencies and no dependency on another future module. It is excluded from the root module's normal `go test ./...` and build traversal.
- No `init`, automatic registration, background execution, DB connection, external service or ledger write is present.
- Inputs describing authorization, active membership, visibility or transaction facts are **trusted adapter snapshots**, never client JSON. This package is not an HTTP authorization boundary.
- No production persistence is provided. Pure preflight checks do not guarantee atomicity, idempotency, authorization freshness or restart durability; future adapters must supply these within existing service/store transactions.
- Spec copies are historical reference snapshots; their original relative links refer to the source branch layout. No third-party source was copied.

## Implemented milestone

Daily, weekly and monthly IANA-local reminder windows; stable per-date dedupe keys; DST gap skipping and first-occurrence fold selection; exact historical second-offset handling; and no catch-up backlog. Invalid UTF-8 identities and the host-dependent `Local` timezone are rejected.

### Structured calendar API

Existing callers can continue using `Due(template, network, now)` for daily reminders. `Template` retains its existing fields. The additional API accepts an explicit rule:

```go
daily := recurring.Schedule{Frequency: recurring.Daily}
weekly := recurring.Schedule{Frequency: recurring.Weekly, Weekday: time.Monday}
monthly := recurring.Schedule{Frequency: recurring.Monthly, DayOfMonth: 31}
occurrence, due, err := recurring.DueWithSchedule(template, monthly, network, now)
```

`Weekday` is meaningful only for weekly rules and must be Sunday through Saturday. `DayOfMonth` is meaningful only for monthly rules and must be 1 through 31. Unused selectors must be zero, and an empty or unknown frequency is invalid. The original `Due` wrapper supplies the explicit daily rule for existing callers.

All rules operate on the current local calendar date. A missing monthly date is skipped: a day-31 schedule does not run on April 30 or May 1. Nonexistent DST minutes are skipped; repeated minutes select the first physical instant. This implements the provisional isolated-module calendar semantics recorded in `UPSTREAM-RESEARCH.md`; core policy adoption remains on HOLD. Seconds, arbitrary cron expressions and shorter-than-daily schedules are not exposed.

Changing a rule or reminder time preserves the existing key for the same network, template and local date. Existing valid daily keys are unchanged. This prevents a later atomic firing adapter from interpreting a same-date edit as a new occurrence. It does not itself enforce uniqueness: timezone edits that move the current local date, pause/skip state, revisions and durable firing transactions still need an explicit adapter policy.

Resolution examines timezone intervals within a fixed 72-hour UTC window. It validates exact wall-time round trips and selects the earliest matching instant. This avoids the original minute scan's error for historical timezone offsets containing seconds. It also handles extrapolated timezone rules at leap-year boundaries without relying on their sometimes stale interval end bounds. No third-party parser or source is needed for these three structured rules.

Compatibility: templates using invalid UTF-8 identities or `Timezone: "Local"` now fail validation. An adapter must repair the identity or require an explicit named timezone; it must not silently normalize bytes or substitute the host timezone.

## Existing application seams inspected

`internal/service/notify.go; internal/service/transactions.go`. A future adapter must map these through the existing service authorization boundary. No direct import or code change is made to these seams. APIs here are provisional until that adapter is qualified.

## Remaining feature work

Persistent templates/firings; atomic reminder plus unique firing; pause/skip/edit audit; explicit quotas, retention and timezone-edit policy; optional manually confirmed draft adapter; crash/concurrent worker tests; current authorization adapters; and accessible HTTP/CLI/UI acceptance. Calendar computation is implemented, but those durable service and integration requirements remain unfinished. No production Core recurring feature is enabled.

## Standalone validation

```sh
cd future/recurring
go test -race -cover ./...
go vet ./...
go test -run '^$' -fuzz '^FuzzScheduleInvariants$' -fuzztime=100000x -parallel=2 -timeout=90s ./...
```

Tests use synthetic fixtures only. See `VALIDATION.md` for observed results. These tests qualify the stated core milestone, not the full roadmap feature, production behavior, browser accessibility, or Circula's release.

The dedicated provider workflow verifies the disabled/HOLD manifest, standard-library-only module boundary, Go formatting, vet, race regressions and 100,000 fuzz executions. Its job and fuzz process have hard time limits. Provider results must be checked at the exact publication head before consolidation; local results do not replace that gate.

## Later integration, one feature at a time

1. Finish this module's remaining scope and acceptance tests. Audit its own code and record unresolved findings.
2. Qualify the then-current Circula core separately. Do not infer completion from this module's tests.
3. Rebase this branch on the qualified core; review the baseline diff and any API/schema drift. No promise of a conflict-free future merge is made.
4. Add a narrow adapter, reviewed API contracts, additive migrations (if needed), explicit per-network opt-in, CLI and accessible/no-JS UI in a separate integration change. Keep all other future features absent.
5. Recheck tenant/auth/CSRF/idempotency behavior, concurrency/restart, backup/restore, full ledger regression and relevant manual/browser acceptance.
6. Obtain the owner's explicit selection of this feature for integration. Only then merge that single integration change. Never auto-merge this hold branch or batch-enable the backlog.
7. Disable by removing routes/jobs and adapter registration; do not roll back ledger history. Any feature-data retention/migration procedure must be documented before integration.

No work on this branch changes Circula's core completion or release status.
