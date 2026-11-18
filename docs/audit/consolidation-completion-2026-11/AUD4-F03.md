# AUD4-F03: dispute timelines applied retroactively

Severity: Medium (CWE-840, business logic). Baseline: `28ed716` (main).

## Finding

An administrator could change `response_period_days` (down to 0) or `re_review_window_days` (down to 1) through `UpdatePolicy` (API `PATCH .../admin/policy` or the web policy form). Neither field is an adverse change under GOV-003, so the change took effect at once, and the service computed every dispute deadline from the policy in force at the time of reading:

- `responseGate` (`internal/service/disputes_decide.go`) let an open dispute be decided with no response and no waiver.
- `RequestReReview` (`internal/service/disputes_response.go`) refused the re-review of an already decided dispute.
- The dispute view (API `response_deadline`, web page), the overdue reminder job and the web re-review date (`internal/web/h_dispute_review.go`) moved with them.

The specification says otherwise: the policy table in [10 section 2](../../specs/10-governance-operations.md) gives "Dispute timelines | Configurable within bounds | DEC-012 | Prospective", and GOV-001 requires each policy change to follow that policy's effective-time rule. DOM-017 already fixes the dispute window at completion.

## Fix

A dispute keeps the response period and re-review window of the policy version in force when it was opened. A later change, shorter or longer, applies only to disputes opened once it is in force. "Prospective" in the policy table is unqualified, so lengthening does not extend existing disputes either.

- Repository contract: `NetworkRepo.PolicyInForceAt(netID, at)` returns the highest policy version whose `effective_at` is not after `at`. This is the rule `GetNetwork` applies under `SetAsOf`, so a scheduled version (notice period) is not in force before its date, and a withdrawn one never is. Fields missing from older stored versions read with `domain.DefaultPolicy`, as in `GetNetwork`. The SQLite store implements it in `internal/store/sqlite_network.go`. `storetest` gains the `PolicyInForceAt` contract.
- Domain: `DeadlinesOf` takes the policy in force at opening, and `DisputeDeadlines` gains `ReReviewCloses` (`domain.ReReviewCloses`).
- Service: `disputePolicy` and `disputeDeadlines` (`internal/service/disputes_deadlines.go`) are the only source of these two values for an existing dispute. The decision gate, `RequestReReview`, `GetDispute` (and so the API and web page) and `RemindOverdueDisputes` all use them. If no version was in force at opening, the read fails with an internal error and never falls back to the current policy (the same rule migration 0016 follows). `OpenDispute` does not create that state: it reads the policy in force at the same clock instant it records as the opening time.
- Web: the re-review date and form use `DisputeView.Deadlines.ReReviewCloses`. The adapter no longer computes the deadline from the current policy, and its inclusive end matches the service.

No migration is needed. `network_policies` is append-only (triggers since 0001), and every new version takes effect at or after the moment it is written. The version in force at a past instant therefore never changes, so it can be derived from the history and does not need to be stored on the dispute. Neither the dispute window (DOM-017) nor any other policy rule changes.

Residual: if a policy change and the opening of a dispute fall in the same millisecond, the change counts as in force at opening (ties go to the higher version, as in `GetNetwork`). Storing the version on the dispute would close this gap, but it would need a migration. Nobody can control that timing, so this note records it and stores nothing.

## Tests

- `internal/service/dispute_timelines_test.go`
  - `TestDisputeTimelinesFixedAtOpen`: the finding's probe, adapted. One hour after opening, the period is cut to 0 days and the window to 1 day. The open dispute stays undecidable without a waiver until exactly 7 days after opening. Its shown deadline and the reminder agree. The dispute decided before the change can still ask for its re-review 3 days later, and the shown `ReReviewCloses` is 14 days after the decision.
  - `TestDisputeTimelineChangesApplyToLaterDisputes`: disputes opened after a lengthening (30 days / 60 days) or a shortening (0 days / 1 day) use the new values. Disputes opened before the lengthening are not extended.
  - `TestScheduledDisputeTimelineChangeWaitsForItsDate`: the change is announced with 10 days' notice. Disputes opened during the notice keep the old values, also after the change takes effect. A dispute opened on the effective date uses the new values.
- `internal/web/dispute_timelines_test.go`, `TestDisputePageReReviewDeadlineFixedAtOpen`: after the window is cut, the dispute page still offers the re-review form and shows the date the service enforces. The request succeeds.
- `internal/store/policy_in_force_test.go`, `TestPolicyInForceAtReadsLegacyVersionsWithDefaults`, plus the `storetest` `PolicyInForceAt` contract. The contract covers: effective times are inclusive, a scheduled version waits for its date, a withdrawn version never comes into force, equal effective times go to the higher version, there is no version before the first, and versions are per network.

Fail-before evidence (baseline code, new tests only):

```
--- FAIL: TestDisputeTimelinesFixedAtOpen
    (a) shown response deadline 2026-03-01 12:00:00 +0000 UTC overdue=true, want 2026-03-08 12:00:00 +0000 UTC (7 days after opening, as enforced)
    (a) 1 overdue reminders sent 2 days into a 7-day response period
    (a) response period removed for an open dispute: decided with no response and no waiver
    (b) re-review right stripped retroactively: invalid-transaction-state: the re-review period closed on 2026-03-02
--- FAIL: TestDisputeTimelineChangesApplyToLaterDisputes
    earlier dispute: response deadline 2026-03-31 12:00:00 +0000 UTC, want 7 days after opening (not extended)
    earlier dispute: its 7-day response period was extended by the later change: invalid-transaction-state: ...
    earlier dispute: its 14-day re-review window was extended by the later change
--- FAIL: TestScheduledDisputeTimelineChangeWaitsForItsDate
    dispute opened during the notice lost its re-review window when the change took effect: invalid-transaction-state: the re-review period closed on 2026-03-03
--- FAIL: TestDisputePageReReviewDeadlineFixedAtOpen
    page /n/town/disputes/... lacks "Ask for a re-review"
    page /n/town/disputes/... lacks "datetime=\"2026-03-15T12:00:00Z\""
    page /n/town/disputes/.../rereview (status 422) lacks "Re-review requested"
```

All of them pass with the fix. The `ReReviewCloses` assertions and the store and contract tests were added together with the field and the read they exercise.
