# FUX-104: Email and push notifications

**HOLD — isolated development, incomplete feature, not approved for integration or release.**

The owner authorized independent future-feature development on 2026-11-16, while explicitly deferring integration until Circula and the feature are complete. This authorizes this branch's standalone development; it does not authorize enabling, merging or deploying the feature. Historical implementation prohibitions in the copied planning documents are superseded only for this isolated development scope.

## Isolation contract

- Original branch: `future/fux-104-notifications-20261116`. Baseline: `2eacf7718f5bb3da3be5a288024af879076a9597`. Binding repairs are isolated on `audit/future-notifications-binding-20261119`.
- The repair branch merges published core audit candidate PR #90 at `9b1fc6a0bb4ae4ad95c4f505929ce8088f10bf2c` and includes reviewed monetary-guard test follow-up `d627af48ff4042db73adac9a72f6208db268a924`. Core qualification remains a dependency; this refresh does not establish a core release pass.
- Its final refresh merges published core candidate `4140bc135479077f6297d53ed2e9dfcd85c056d6`, including setup, catalogue and account-session work. The notifications module's runtime source is unchanged by this merge; provider qualification remains a separate gate.
- Module runtime changes stay within `future/notifications/`. The dedicated `.github/workflows/future-notifications.yml` qualifies its explicit SQLite test suite using the pinned core Go version. Existing core workflows, application imports, routes, migrations, root dependencies, feature flags and runtime configuration remain unchanged by the module.
- This nested Go module has no dependency on another future module. It uses the explicitly pinned SMTP and Web Push libraries in `go.mod`; SQLite is linked by the test-only `sqlite_cgo_test` tag. It is excluded from the root module's normal `go test ./...` and build traversal.
- The module contains a caller-configured SQL store, dispatcher and opt-in SMTP/Web Push transports. Nothing registers or runs them in Circula core; no ledger posting API is exposed.
- Inputs describing authorization, active membership, visibility or transaction facts are **trusted adapter snapshots**, never client JSON. This package is not an HTTP authorization boundary.
- The SQL outbox accepts a caller-owned transaction. A future core adapter must enqueue alongside committed source events and supply fresh membership/session authorization, credentials, egress configuration and worker lifecycle. Standalone SQL tests do not qualify that absent adapter.
- Spec copies are historical reference snapshots; their original relative links refer to the source branch layout. No third-party source was copied.

## Implemented milestone

The source now includes SQL preferences, encrypted push/email targets, confirmation and unsubscribe tokens, an idempotent transactional outbox, retry/quiet-hour dispatch, fresh directory checks, key rotation, and SMTP/Web Push transport adapters. These extend the initial pure delivery-policy milestone recorded in the historical validation section.

The 2026-11-19 repairs bind encrypted targets to unambiguous network/member/row tuples and reject invalid UTF-8 before JSON can normalize an identity in a dedupe key or signed token. All notification channels remain opt-in and core integration remains disabled/HOLD.

## Target-binding compatibility

Targets whose kind, network, member and row identifiers contain no `|` retain their exact existing `ntf/v1` authenticated data and remain readable. Delimiter-bearing tuples use the separate `ntf/v2` namespace with 64-bit byte-length prefixes and raw field bytes. Opening and key rotation use the same selection rule; there is no fallback to ambiguous legacy authenticated data.

An older target sealed with a delimiter-bearing tuple now fails closed. Re-register a push subscription to replace its sealed payload. For email, call `RevokeEmailTarget` and then `AddEmailTarget`, followed by address verification: adding the same address alone can take an existing-row fast path without resealing. Do not copy legacy ciphertext to another owner or attempt to guess its original tuple.

Actor, event and token identifiers must be valid UTF-8. `UnsubscribeToken` returns an empty string for invalid token identity bytes; adapters must treat that as invalid input. If an experimental deployment previously minted links from invalid UTF-8 identities, invalidate those old links by rotating the token-signing key before accepting new ones. Ordinary valid UTF-8 identities, existing safe target blobs and valid tokens keep their encoding.

## Existing application seams inspected

`internal/service/notify.go; internal/store`. A future adapter must map these through the existing service authorization boundary. No direct import or code change is made to these seams. APIs here are provisional until that adapter is qualified.

## Remaining feature work

Qualify the production transaction and current-authorization adapters, worker lifecycle and crash/restart behavior; exercise real SMTP/Web Push transport and configured egress policy; complete bounce handling, operator CLI and accessible/no-JS preference management; validate credential custody/rotation, backup/restore and browser opt-in/logout behavior. Existing source and bounded tests do not establish those acceptance gates.

## Standalone validation

```sh
cd future/notifications
go test -race -cover ./...
go vet ./...
# Explicit database qualification; requires a C compiler for the test driver:
go test -race -count=1 -tags=sqlite_cgo_test -cover ./...
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
