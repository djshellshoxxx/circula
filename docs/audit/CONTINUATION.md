# Autonomous specification completion

The integration branch `work/spec-completion` starts at `claude/circula-completeness`
(9a86ea6) and incorporates `claude/circula-finish` (d350474). Main contains the
original prompt, not the application. Keep developing the integrated application;
do not replace it with main or the older specification-only branch.

## Integration decisions

- Preserve scheduled closure, its notice period, final export and read-only retention.
  Apply the security branch's lasting-action restriction to schedule, cancellation
  and export verification, rather than resurrecting immediate closure.
- Preserve notification name redaction during erasure, and restrict person-wide
  login erasure to self-service and the operator. Preserve replay-record deletion.
- Preserve native TLS, audit anchors/mirror, maintenance mode and log retention,
  together with automatic backup encryption enforcement and configurable durability.
- Keep member notifications and security notices as separate existing service methods.
  Password recovery/reset must also revoke pending MFA challenges.
- Retain both frozen-network refusal rendering and secure CSRF cookie names.

## Work sequence

1. Verify the integrated branches with unit/contract tests and vet; fix integration failures.
2. Complete FR-064's missing API adapter with strict input, existing service authorization,
   OpenAPI contract coverage and assertions that flags never change the ledger.
3. Re-audit remaining partial rows against code: COMPLETENESS.md contains historical
   rows already superseded by later changes. Prioritize P0 behavior over stale status text.
4. Continue FR-054 historical balances, export resource bounds and other verified gaps.
5. Run release checks and code/security review. Keep manual accessibility and independent
   review evidence explicitly pending until actually performed.

This is an in-progress checkpoint, not a claim that all specifications are implemented.

## Verification and progress

- The balance-bound migration from the security branch is numbered 0015 after
  completeness migrations 0008–0014. No completeness migration is renumbered.
- Both amount fuzz suites are retained with distinct target names.
- FR-064 API regression test first failed with the expected 404, then passed after
  implementing the route. OpenAPI contract suite passes with the new operation.
- Pagination fixtures distribute active listings among members to honor the 100-listing
  anti-flood cap. Dual-control fixtures honor independent appointment and the 24-hour delay.
- Full integrated suite, API/store suite and vet are being run; final results follow.
