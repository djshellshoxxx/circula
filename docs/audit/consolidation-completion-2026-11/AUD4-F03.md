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

No migration was added. The version in force at opening is reconstructed from `network_policies` (append-only, triggers since 0001), using `effective_at` and the version number. This holds only while wall-clock time moves forward and a later version never shares the opening millisecond. Neither the dispute window (DOM-017) nor any other policy rule changes.

**Initial status: partial remediation.** Ordinary forward-time policy changes no longer affect existing disputes. The independent review reproduced two remaining boundaries with test-only clock variants of `TestDisputeTimelinesFixedAtOpen`:

- **Same millisecond.** A policy update whose `effective_at` equals the dispute's `OpenedAt` wins, because ties go to the higher version, as in `GetNetwork`.
- **Clock rollback.** After the host clock steps backward, a later update has an earlier `effective_at` than the opening.

In both variants the shown response deadline shrinks, an overdue notice fires, a decision succeeds without a response or waiver, and the re-review window is removed. An administrator cannot pick the millisecond or move the host clock through the product. An operator or a host time error can still produce the second case.

**Initial outstanding design item (addressed below):** bind the policy version (or the two deadlines) on the dispute row at opening. This needs a schema migration and a backfill rule for existing disputes, in the same way migration 0016 stored `dispute_policy_version` at completion. It has to be coordinated with the migration numbers used by the open GOV-021 and SEC-026 drafts.

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


## Durable binding follow-up

The follow-up binds `disputes.policy_version` inside the serialized `OpenDispute`
write, using the policy actually in force at its recorded opening instant. It
records that version in the new opening audit event. The response gate,
re-review gate, views and reminders now read that exact immutable version through
`GetPolicyVersion`, rather than choosing a version again from wall time. Later
policy writes cannot change the binding when their timestamps are equal to or
before the opening time. Public API and export projections stay unchanged.

Migration `0019_dispute_policy_binding.sql` is additive and forward-only. It
backfills pre-upgrade disputes with the highest policy version whose effective
time is not after their opening time, which freezes the preceding
`PolicyInForceAt` interpretation. A missing version aborts the whole migration
with `dispute_policy_backfill_needs_owner_mapping`; it never falls back to the
current policy. Ledger entries, balances, outcomes and existing audit events
remain unchanged. The bound version, network and opening time cannot be edited;
conflict replacement of a dispute or an existing policy version is also refused.

**Historical limit:** a pre-upgrade dispute did not record its original version.
If a same-millisecond policy write or a clock rollback already made the historical
interpretation ambiguous, this migration cannot prove which policy was observed
at the original opening. It preserves the pre-upgrade reconstruction and makes it
stable. Such historical cases require an operator's evidence review rather than
claiming an automatically recovered original version. New disputes have a durable
binding and do not have this limitation.

Additional regression coverage:

- `TestDisputeTimelinesSurviveEqualOrReversedPolicyClocks`: equal and backward
  clock cases retain existing response/re-review rights; new disputes receive the
  new policy. Decisions, displayed dates and reminders agree.
- `TestDisputePolicyBackfillAndBindingSurviveRestart`: schema 18 upgrades keep the
  previous policy interpretation, scheduled cancellation semantics, legacy field
  defaults and existing export projections; appended earlier-dated policies and
  process restart do not change bindings.
- `TestDisputePolicyBackfillStopsWithoutApplicablePolicy`: incomplete historical
  data aborts and rolls back the migration before writes can be served.
- `TestDisputePolicyBindingGuards` and repository contract `DisputePolicyBinding`:
  exact version and tenant scoping, lifecycle-update immutability, missing/pending
  version rejection, direct-update and conflict-replacement protections.

Execution evidence for the follow-up belongs in the comprehensive audit report;
these test names identify coverage and do not by themselves assert a passing run.
