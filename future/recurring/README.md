# FNET-207: Recurring trade templates

**HOLD — isolated development, incomplete feature, not approved for integration or release.**

The owner authorized independent future-feature development on 2026-11-16, while explicitly deferring integration until Circula and the feature are complete. This authorizes this branch's standalone development; it does not authorize enabling, merging or deploying the feature. Historical implementation prohibitions in the copied planning documents are superseded only for this isolated development scope.

## Isolation contract

- Branch: `future/fnet-207-recurring-20261116`. Baseline: `2eacf7718f5bb3da3be5a288024af879076a9597`.
- Only `future/recurring/` is added. No application imports, routes, migrations, root dependencies, feature flags, workflows, jobs or runtime configuration are changed.
- This nested Go module has no third-party dependencies and no dependency on another future module. It is excluded from the root module's normal `go test ./...` and build traversal.
- No `init`, automatic registration, background execution, DB connection, external service or ledger write is present.
- Inputs describing authorization, active membership, visibility or transaction facts are **trusted adapter snapshots**, never client JSON. This package is not an HTTP authorization boundary.
- No production persistence is provided. Pure preflight checks do not guarantee atomicity, idempotency, authorization freshness or restart durability; future adapters must supply these within existing service/store transactions.
- Spec copies are historical reference snapshots; their original relative links refer to the source branch layout. No third-party source was copied.

## Implemented milestone

Daily IANA-local reminder windows, stable daily dedupe keys, DST gap skipping/fold first-occurrence selection and no catch-up backlog.

## Existing application seams inspected

`internal/service/notify.go; internal/service/transactions.go`. A future adapter must map these through the existing service authorization boundary. No direct import or code change is made to these seams. APIs here are provisional until that adapter is qualified.

## Remaining feature work

Persistent templates/firings; atomic reminder plus unique firing; pause/skip/edit audit; quotas; weekly/monthly schedules and calendar policy; optional manually confirmed draft adapter; crash/concurrent worker tests. Current core supports daily reminders only.

## Standalone validation

```sh
cd future/recurring
go test -race -cover ./...
go vet ./...
```

Tests use synthetic fixtures only. See `VALIDATION.md` for observed results. These tests qualify the stated core milestone, not the full roadmap feature, production behavior, browser accessibility, or Circula's release.

## Later integration, one feature at a time

1. Finish this module's remaining scope and acceptance tests. Audit its own code and record unresolved findings.
2. Qualify the then-current Circula core separately. Do not infer completion from this module's tests.
3. Rebase this branch on the qualified core; review the baseline diff and any API/schema drift. No promise of a conflict-free future merge is made.
4. Add a narrow adapter, reviewed API contracts, additive migrations (if needed), explicit per-network opt-in, CLI and accessible/no-JS UI in a separate integration change. Keep all other future features absent.
5. Recheck tenant/auth/CSRF/idempotency behavior, concurrency/restart, backup/restore, full ledger regression and relevant manual/browser acceptance.
6. Obtain the owner's explicit selection of this feature for integration. Only then merge that single integration change. Never auto-merge this hold branch or batch-enable the backlog.
7. Disable by removing routes/jobs and adapter registration; do not roll back ledger history. Any feature-data retention/migration procedure must be documented before integration.

No work on this branch changes Circula's core completion or release status.
