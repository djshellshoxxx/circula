# FNET-205: Credit-circulation analytics

**HOLD — isolated development, incomplete feature, not approved for integration or release.**

## Recovery checkpoint — 2026-11-19

This candidate restores the saved implementation at `ef4bdf87d2bc915b26d84e548ea4e6a478079006` onto the published analytics branch at `0b2965101e1d5d1e3e6447e7c24197efc7f0cb9f`. All eight Go source/test files and `go.mod` are unchanged from that saved source. A dedicated `.github/workflows/future-analytics.yml` now requests native Linux, macOS and Windows qualification without activating the module. See [RECOVERY-20261119.md](RECOVERY-20261119.md) for provenance, current checks and unresolved gates.

The implementation and milestone records below were recovered from the earlier development branch. Their completion percentage, `tested_source` and historical test results are not a current-head qualification or a release-readiness claim.

The owner authorized independent future-feature development on 2026-11-16, while explicitly deferring integration until Circula and the feature are complete. This authorizes this branch's standalone development; it does not authorize enabling, merging or deploying the feature. Historical implementation prohibitions in the copied planning documents are superseded only for this isolated development scope.

## Isolation contract

- Current successor branch: `codex/future-analytics-master-20261116`. Recovered predecessor: `243f7fc181a14cecb124c4b9733b86c010d53993`. Baseline: `2eacf7718f5bb3da3be5a288024af879076a9597`.
- Recovered feature runtime and documentation are confined to `future/analytics/`, with the dedicated `.github/workflows/future-analytics.yml` qualification workflow. The candidate also restores two exact files already accepted on main: the audit-anchor command's missing `os` import and the departure regression's test-only proposal cap. No future-feature imports, routes, migrations, root dependencies, feature flags, background jobs or runtime configuration are added.
- This nested Go module has explicit pinned dependencies where listed in go.mod and no cross-feature imports. It has no dependency on another future module. It is excluded from the root module's normal `go test ./...` and build traversal.
- No `init`, automatic registration, background execution, DB connection, external service or ledger write is present.
- Inputs describing authorization, active membership, visibility or transaction facts are **trusted adapter snapshots**, never client JSON. This package is not an HTTP authorization boundary.
- No production adapter is registered. Module-owned explicit persistence, where present, remains isolated. Snapshot adapters must supply fresh authorization and isolated historical reads at the exact requested cutoff, and preserve published revision stability.
- Spec copies are historical reference snapshots; their original relative links refer to the source branch layout. No third-party source was copied.

## Implemented standalone scope

Ledger-verified ordinary trade/reversal flow with partial/later reversals; grants/adjustments excluded from turnover and included in own historical opening/closing; explicit scoped account/offer facts, member-dominance/complementary suppression, fixed releases and public-only revision fingerprints; exact monetary strings; read-only HTTP/no-JS/CSV/CLI.

## Existing application seams inspected

`internal/service/export.go; internal/histbal; internal/store`. A future adapter must map these through the existing service authorization boundary. No direct import or code change is made to these seams. APIs here are provisional until that adapter is qualified.

## Remaining feature work

Actual core authenticated historical-source adapter and immutable release/query-budget/correction policy; production API/CLI wiring and full ledger regression; browser/WCAG/screen-reader and snapshot/restore operations; owner-selected integration and release approval. See AUDIT.md and CONTRACT.md for exact implemented scope and evidence.

## Standalone validation

```sh
cd future/analytics
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

## Master-scope continuation

HTTP query bounds now precede host callbacks. Large-snapshot fixtures verify complete 12-period accounting at 10,000 and 100,000 events, including stock and participant counts. Current results are recorded in VALIDATION.md; criterion-level progress is in COMPLETION.md. Immutable release/query-budget/correction policy remains unfinished standalone privacy work, not a completed capability delegated by wording to the host.
