# FUX-104: Opt-in email and web push notifications

Status: **PROPOSED — future only; NOT implemented or beta-qualified**  
Priority: High · Recommended phase: 2 · Origin: future-features planning (2026-11-10 America/Vancouver)  
Source baseline: draft integration PR #31 (`claude/circula-beta-completion-1npjw6`, `bffcede8c7dc`).

## Goal and existing behavior

Members are notified about relevant activity without turning delivery providers into a source of sensitive trade metadata.

**Current baseline:** The application already has in-app notifications, an unread counter and network announcements; email and push do not ship in the beta.

This document is an incremental, non-authoritative planning specification. It does **not** amend `docs/specs/00-conventions.md`, `docs/specs/03-domain-model.md`, or existing P0/P1 requirements until reviewed and adopted by the repository owner. Keep the current beta branch release gates unchanged.

## Functional requirements

1. Add per-member per-network channel preferences for proposal received, accepted/declined, dispute, expiring proposal and account security notices.
2. Implement optional SMTP sender and standards-based Web Push endpoint; no provider required for core operation.
3. Use durable outbox rows created with the underlying event and async delivery attempts; delivery failures never undo or repeat ledger entries.
4. Use privacy-minimal notification bodies by default ('You have a new Circula activity'); sensitive details only inside authenticated app.
5. Support unsubscribe, quiet hours/time zone, bounce handling, revoked browser subscriptions and preferences audit.
6. Bound retries and dedupe by source event, recipient, channel and notification kind; inspectability via CLI/metrics.

## Proposed persistence and migration

- notification_delivery(id,network_id,member_id,event_id,channel,status,attempts,next_attempt_at,expiry,provider_message_id) UNIQUE(event_id,member_id,channel).
- notification_preferences(member_id,network_id,event_type,channel,enabled,quiet_start,quiet_end,tz).
- push_subscriptions(member_id,endpoint_hash,encrypted_endpoint_and_keys,created_at,revoked_at); secret at rest.

Apply additive forward-only SQLite migrations; use the existing service/store transaction boundaries and one-process-per-file guarantee. New fields may be null/default for all historical records. Specify index/retention and backup behavior before merge.

## Interface and API contract

- Proposed GET/PATCH /api/v1/me/notification-preferences; POST/DELETE /api/v1/me/push-subscriptions.
- Read existing /api/v1/notifications unchanged; no delivery credentials or push keys returned to nonowners.
- Optional CLI diagnostics: circula notifications status/test using redacted metadata.

All future endpoints are **proposals**, not existing routes. Before implementation, update `docs/api/openapi.yaml`, `docs/specs/07-api-contract.md`, and service contract tests. Preserve Circula's authenticated `/api/v1` contract, X-Network scoping, 64 KiB API body policy unless an explicitly reviewed media-upload exception is adopted, RFC 9457 errors, cursor paging, cookie CSRF, no-store responses and idempotency.

## Security and privacy boundaries

- SSRF-safe Web Push URL validation and explicit endpoint policy; bound message size, retries and egress.
- Credentials via existing secret-file/environment path; protect VAPID private key and SMTP password.
- No trade amount/contact information in email subject, push payload or logs by default; unsubscribe revokes instantly.

None of the above may bypass the existing transaction lifecycle, credit limits, two-entry sum-zero invariant, audit, tenant isolation or member authorization checks.

## Acceptance and negative tests

1. Fake SMTP/Web Push servers validate formatting, minimal payload, TLS failures and nonleaking logs.
2. Restart after event commit before send: exactly one logical outbox event and bounded at-least-once delivery.
3. Account logout and subscription revocation stop subsequent pushes; inactive members never receive protected content.
4. Opt-out and quiet hour behavior across DST transitions; a provider outage never changes ledger.

In addition: check migration rollback, restart recovery, `go test -race ./...`, no-JavaScript essential paths, WCAG 2.2 AA automation, manual screen-reader qualification, auth/tenant threat matrix, API compatibility, secret scanning and full ledger regression.

## Upstream reuse assessment

- wneessen/go-mail — MIT Go SMTP/message composition.
- SherClockHolmes/webpush-go — MIT Go Web Push/VAPID.
- gotify/server — MIT, optional separate self-hosted transport reference, not default required service.

**No upstream source is copied by this spec.** Before porting: pin a full upstream commit and file path, review its exact per-file license and dependencies, record ownership/attribution, run advisory scanning and build/API tests. A permissive license permits reuse subject to its conditions, but never means that the other project's business logic is safe for Circula. Circula currently has no approved product license; owner/license review must precede third-party copying or redistribution.

## Explicit non-goals

- Mandatory email identity; marketing blasts; delivering full balances/transaction detail through providers.

## Definition of done

- [ ] Owner approves scope, decisions, privacy and license/attribution.
- [ ] Upstream dependency or reused files pinned and traced to original license.
- [ ] SQL migration + rollback/restore acceptance and least-privilege service access.
- [ ] New API contract, CLI/UI/help/wiki and user/developer docs updated.
- [ ] Positive, negative, concurrency, restart, cross-network and accessibility tests pass.
- [ ] CI, SCA, human checks and release disposition recorded.
- [ ] Feature enabled only behind explicit network/owner configuration; no silent rollout to beta.
