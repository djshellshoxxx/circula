# SEC-011 owner decision: dual control beyond adjustments (SA2-12)

Status: open. No code changed for this item. Audit 2 finding SA2-12 (docs/audit/SECURITY-AUDIT-2.md) says credit-position actions other than direct adjustments bypass dual control. This note lists the options so the owner can pick a scope; the recommendation is at the end.

## What is uncontrolled today

| Path | Code | Effect on credit position |
|---|---|---|
| Reversal of a completed adjustment | `ReverseTransaction` (reversal.go) | Undoes an approved adjustment, or reverses an adjustment of any size, on one administrator's say. Nets to a free adjustment in the other direction. |
| Per-member limit override | `SetCreditLimit` (membership.go) | Raises a member's overdraft without a second administrator. |
| Network default limit and sign-up grant | `UpdatePolicy` (networks.go) | Raises every member's overdraft, or the grant minted for each new member. |
| Splitting | `CreateAdjustmentRequest` | Many adjustments of `threshold` or less each avoid approval; there is no aggregate. |
| Dispute resolution | `ResolveDispute` (disputes_decide.go) | Reverses a completed trade on one administrator's say. This is a different risk (a party asked for it, a statement exists) and is left out below. |

## Why nothing was implemented

- Limit raises and policy changes need a new request type: a new table next to `pending_adjustments` (a migration), store methods, API and web screens for the approver, expiry handling and tests. That is a feature, not a patch, and touches code owned outside the service layer.
- The rolling cap needs a number the owner must choose (see option C). A wrong default either blocks normal bookkeeping or does nothing.
- Reversal of adjustments changes how administrators fix mistakes. The audit does not say whether a reversal should be blocked, pending, or free.

## Options

### A. Reversal of adjustments

1. Fail safe. Reversing a transaction of kind adjustment whose reversed amount exceeds `DualControlThreshold` returns `approval-required` (as `CreateAdjustment` does today). The administrator posts the opposite adjustment through `CreateAdjustmentRequest`, which is already dual-controlled. Small change (about 10 lines in `ReverseTransaction`), no migration, no new screens. The reversal link between the two transactions is lost; the audit reason carries it.
2. Pending reversal. Add a `kind` column to `pending_adjustments` and let a request carry the original transaction id. Keeps the link; needs a migration, a second code path at approval, and UI text changes.

### B. Limit raises and policy

1. Operator only above a threshold. Raising a member's limit, the default limit or the sign-up grant by more than `DualControlThreshold` in one step requires the operator CLI (`circula dual-control` style command). Mirrors SA2-01b for lowering the threshold. No new tables; administrators lose the ability to make large raises themselves.
2. Pending limit changes. A new `pending_policy_changes` request (member limit, default limit, grant) approved by a second administrator. Full feature; most flexible; same cost as A.2 plus a policy diff view.
3. Do nothing, document. Accept that administrators set limits (they are the network's governing role) and rely on the audit trail and member notices.

### C. Rolling cap on sub-threshold adjustments

Count, per administrator, the adjustments they posted directly in the last 24 hours (`adjustment.posted` and the direct `ActAdjustment` events, or sum of `transactions` of kind adjustment with `created_by` = the administrator). When the new amount plus that sum exceeds a cap, the adjustment goes through the pending flow even though it is at or below the threshold.

- Cap value, to be chosen: `1 x threshold` (the strictest: a day of small adjustments cannot exceed what one approved adjustment could), `2 x threshold`, or a separate policy field `DualControlDailyCap`. A separate field costs a policy migration (policy is stored as JSON, so no SQL migration) and a UI field.
- Inherits D6: only active when the threshold is set and two administrators exist; with one administrator the cap cannot apply.
- Debit and credit are summed as absolute amounts; netting would let an administrator move value out and back.

## Recommendation

1. Do C with the cap at `1 x threshold`, derived from the existing threshold so there is no new setting. It closes the splitting hole at the lowest cost and falls back to the existing pending flow.
2. Do A.1 now (fail safe on large adjustment reversals). It is small, needs no migration and follows decision D8.
3. For B choose B.1 for the next release: large raises need the operator, small ones stay with administrators. Defer B.2 until networks ask for it.
4. Leave dispute resolution as is; document it as a known gap with its own audit trail (the dispute record names both parties and the statement).

Each step needs: a regression test (split attack stays blocked, large reversal refused with `approval-required`, large raise refused outside the operator CLI), a line in docs/specs/addenda/SEC-011.md Gaps, and an update of SA2-12 in the audit backlog.

## Questions for the owner

1. Is `1 x threshold` per 24 hours the right cap, or should it be configurable?
2. Should administrators be able to reverse a large adjustment at all, or only the operator?
3. Is B.1 (operator for large raises) acceptable, or do administrators need to raise limits unaided?
