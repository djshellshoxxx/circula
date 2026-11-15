# Dual control for adjustments (SEC-011)

Spec decisions and acceptance mapping: `docs/specs/addenda/SEC-011.md`.

## Operating it

1. A network needs two or more active administrators.
2. Set the threshold (minor units; `0` turns it off):
   `PATCH /api/v1/admin/policy` with `If-Match`, body `{"dual_control_threshold_minor": "100000", "reason": "..."}`.
   Refused with `dual_control_threshold` if fewer than two active administrators exist. The web policy form keeps the stored value but does not edit it.
3. An administrator creates an adjustment as before (web: Adjustments; API: `POST /api/v1/admin/adjustments`).
   - At or below the threshold: posts at once (API 201, web "Adjustment posted").
   - Above it: API returns 202 with the pending request and `Location: /api/v1/admin/adjustments/{id}`; the web redirects to the pending list. Nothing is posted.
4. Another administrator (not the proposer, not the adjusted member) decides:
   - Web: `/n/{slug}/admin/adjustments/pending`, buttons Approve and post, Reject. The proposer sees Cancel request.
   - API: `POST /api/v1/admin/adjustments/{id}/approve|reject|cancel`, `GET /api/v1/admin/adjustments?status=` and `GET .../{id}`. Approve returns the posted transaction.
5. Requests not decided within 7 days (`service.Config.PendingAdjustmentTTL`) expire and post nothing.

## Service API (internal/service)

- `CreateAdjustmentRequest(ctx, actor, netID, AdjustInput) (*AdjustResult, error)`: exactly one of `Tx` or `Pending` is set. Idempotency key required, retries return the original result.
- `CreateAdjustment(...)`: unchanged signature; returns `domain.CodeApprovalRequired` above the threshold. Use only where an immediate posting is the only meaningful outcome.
- `ApproveAdjustment`, `RejectAdjustment`, `CancelAdjustment`, `GetPendingAdjustment`, `ListPendingAdjustments`.
- `ExpirePendingAdjustments(ctx, systemActor, limit)`: called by the expiry job in `cmd/circula/serve.go` (`jobRunner.expire`) every run.

## Failure behaviour

| Situation | Result |
|---|---|
| Proposer approves or rejects own request | `forbidden` |
| Approver is the adjusted member | `forbidden` |
| Request of another network, unknown id | `not-found` |
| Already decided, expired | `invalid-transaction-state` (an expired request is closed and audited by the attempt) |
| Target departed, proposer no longer administrator, network closed | `invalid-transaction-state`, request stays pending until it expires or is cancelled |
| Debit would breach the member's limit | `insufficient-available-credit`, request stays pending |
| Dual control on, only one administrator left | new above-threshold request refused with `invalid-transaction-state` |

## Data

Table `pending_adjustments` (migration 0005): rows are never deleted; triggers make the request body immutable and a decided request final. The threshold lives in the versioned policy JSON (`dual_control_threshold`, absent = 0), so no schema change is needed for it and older policy versions read as disabled.

## Audit

`adjustment.pending`, `adjustment.approved`, `adjustment.rejected`, `adjustment.cancelled`, `adjustment.expired`, and `adjustment.posted` on the transaction (with `request`, `proposer`, `approver`). Constants: `internal/domain/audit_dualcontrol.go`.
