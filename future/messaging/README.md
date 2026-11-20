# Recovery status — 2026-11-19

**Disabled; integration HOLD. Independent recovery-delta review and Linux real-driver race/vet/checksum checks passed. Native and provider qualification remains pending.**

This branch restores verified archived `messaging` work from `5375bdf6ad5c48d2913c0581c4f8755b7c0f573a`. [RECOVERY-20261119.md](RECOVERY-20261119.md) records the exact source, new parser regressions, current evidence and remaining gates. The earlier README, validation/audit claims and `feature.json` completion percentage below are preserved historical evidence; they are not current-head or cross-platform qualification.

The recovery's scope is `future/messaging/`, its new `.github/workflows/future-messaging.yml`, and two explicitly authorized exact restorations from accepted main: `cmd/circula/auditanchor.go` and `internal/service/departure_pagination_regression_test.go`. No other Core runtime, dependency, migration, shared workflow or feature path changes.

The current master directive authorizes reviewed isolated publication and normal gated merges into this module target. Any older publication prohibition in the preserved manifest or historical documents describes the earlier authorization state; it is not a current publication blocker. HOLD continues to prohibit Core activation.

Messaging full-package fuzz remains unqualified, and the archived Unix-mode privacy guard presents a source-based native Windows portability gate. Neither is counted as a pass. See the recovery report for exact evidence and scope.

---

# FUX-105: Private member messaging

**HOLD — isolated development, incomplete feature, not approved for integration or release.**

The owner authorized independent future-feature development on 2026-11-16, while explicitly deferring integration until Circula and the feature are complete. This authorizes this branch's standalone development; it does not authorize enabling, merging or deploying the feature. Historical implementation prohibitions in the copied planning documents are superseded only for this isolated development scope.

## Isolation contract

- Development branch: `work/requalify-messaging-20261116`, based on `origin/future/fux-105-messaging-20261116` at `7f92251`.
- Only `future/messaging/` changes. No root routes, schema, dependencies, configuration, background workers, ledger methods or release artifacts change.
- This independent nested module now requires Go 1.26.11 and pins the core-selected `modernc.org/sqlite v1.60.1`. Dependency checksums and runtime notices are included. There are no cross-feature dependencies.
- The module defines no `init`; SQLite registers its driver without opening a database. No automatic migration, server listener, worker or network service starts. SQLite installs only via explicit `Migrate`. Host integration and deployment remain on HOLD.
- Fresh actor/network/membership/role/reference decisions are mandatory through a transactional Authorizer and per-request HTTP security callbacks. These adapters are supplied by the host, never client JSON. Missing callbacks fail closed.

## Implemented standalone scope

Explicit private-file SQLite storage and additive migration; network-scoped conversations and optional authorized local references; immutable monotonic messages; atomic create/send idempotency with conflicting-key rejection; serialized multi-connection writes; persistent read/unread and per-participant hiding; directional blocks; close semantics; abuse reports and paginated moderator disposition; separate legal holds; default 30-day inactivity retention with authorized shortening of existing deadlines, explicit authorized purge and bounded visible JSON exports; privacy-scoped literal search through service, POST API, no-JS forms and stdin-based CLI; transactional quotas; escaped no-JavaScript member/moderator/retention forms; strict JSON HTTP service/RFC 9457 errors; HTTPS CLI reading message text from stdin; backup/restore acceptance.

Messages are server-readable text, **not end-to-end encrypted**. No message operation can post or accept a trade. The optional realtime transport is omitted: normal HTTP with durable unread counts satisfies the no-JavaScript workflow without a new dependency.

Read [CONTRACT.md](CONTRACT.md) for authorization actions, endpoints, quotas, retention, migration and restore semantics; [HELP.md](HELP.md) for member/CLI use; [openapi.json](openapi.json) for the unregistered API fragment; [THIRD_PARTY_NOTICES.md](THIRD_PARTY_NOTICES.md) for pinned dependency notices. The inherited spec snapshot is historical reference, not a claim that root APIs changed.

## Qualification remaining

The newest audited local implementation was recovered at `3b4fbe4497827072df15f75e8e518f89925d6435`; the older outage implementation is preserved separately at `1b5fe42883c29a2dcfd9e924f7d28facac9b541a`. Implementation and automated standalone acceptance are present. See COMPLETION.md for weighted criteria and explicit gates. Independent security audit and module advisory scanning are recorded in AUDIT.md. Real-browser/no-JavaScript and WCAG/screen-reader review, eventual core API adoption, real host authorization/CSRF adapter qualification and full core/ledger regression remain outstanding. Root Circula completion is independent. Do not label this production/beta-qualified or release-ready based on module tests alone.

## Standalone validation

```sh
cd future/messaging
go test -race -cover ./...
go vet ./...
go test -run=^$ -fuzz=FuzzPrepare -fuzztime=3s -parallel=1 .
```

Go 1.26.11 is required. Tests use synthetic principals/SQLite fixtures only. The HTTP test header authorizer is not a deployable authentication implementation. See `VALIDATION.md` for observed results and the acceptance matrix.

## Later integration, one feature at a time

1. Finish this module's remaining scope and acceptance tests. Audit its own code and record unresolved findings.
2. Qualify the then-current Circula core separately. Do not infer completion from this module's tests.
3. Rebase this branch on the qualified core; review the baseline diff and any API/schema drift. No promise of a conflict-free future merge is made.
4. Add a narrow adapter, reviewed API contracts, additive migrations (if needed), explicit per-network opt-in, CLI and accessible/no-JS UI in a separate integration change. Keep all other future features absent.
5. Recheck tenant/auth/CSRF/idempotency behavior, concurrency/restart, backup/restore, full ledger regression and relevant manual/browser acceptance.
6. Obtain the owner's explicit selection of this feature for integration. Only then merge that single integration change. Never auto-merge this hold branch or batch-enable the backlog.
7. Disable by removing routes/jobs and adapter registration; do not roll back ledger history. Any feature-data retention/migration procedure must be documented before integration.

No work on this branch changes Circula's core completion or release status.
