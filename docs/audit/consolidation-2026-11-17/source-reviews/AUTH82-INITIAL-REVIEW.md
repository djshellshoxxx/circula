# PR82 authentication review — independent supplemental audit

- Branch: `claude/aud4-auth-fixes`.
- Reviewed head: `0a7ef6ee8efbec15c4603bd9fbb14492c7e4ddbf`.
- Tree: `e5766dda89f542b3868c2e363ab0f1f9c09dc3d5`.
- Main and merge base: `a62071d2c4e65027f4918c90d63e804b1deceb60`.
- Ahead/behind: 2/0 (one implementation commit `91c9374dd5d2bc76af1e91149f4bcf641a7a1990`, plus merge of current main).
- PR82 is draft, mergeable=true, mergeable_state=unstable at inspected current head. External Claude session owns this work; no branch/PR mutations were made by this reviewer.
- Changed files: `internal/service/auth.go`, `internal/service/recovery.go`, new `internal/service/aud4_auth_test.go`, new `docs/audit/consolidation-completion-2026-11/AUD4-AUTH-FIXES.md`.
- No migrations, dependencies, ledger semantics, future module code, CI workflow or break-glass implementation changes.

## Compatible core behavior: AUD4-F06 and AUD4-F04

F06 re-reads the person and password hash inside the session-issuing write unit after the expensive password check. The old hash therefore cannot issue a session after a concurrent password change/reset commits. Password changes replace only the hash they verified and invalidate outstanding MFA challenges in that same write unit. This aligns with SEC003 session invalidation; `docs/specs/09-security-privacy.md:149-156` expressly requires invalidation on password change/recovery.

F04 reserves a durable attempt under the existing `recgen` person budget before checking the current password. It reuses the established re-authentication throttle rather than adding a new table or policy setting. Success clears this budget. Source review found no additional defect in these two changes.

## Policy-affecting extension: AUD4-F02

The new `recovery.go` check refuses an administrator-assisted reset of a CURRENT reviewer, requiring the platform operator instead. That extends existing as-built recovery policy:

- `docs/specs/addenda/SEC-008.md:42-53` excludes current administrators and targets with non-departed membership in another network; reviewers are not excluded.
- The addendum's deviations table retains second-approver and direct operator recovery questions for the owner; it does not decide an operator-only reviewer reset policy.
- `docs/specs/03-domain-model.md:277-279` (DOM038) bars a decider who is a transaction party or who proposed, accepted, adjusted or reversed it. It records reviewer identity, not lifetime role history.
- `docs/specs/09-security-privacy.md:226-233` (SEC010) bars party/recorded-conflict decisions and self-approval. It does not add a former-reviewer reset rule.
- PR82's own `AUD4-AUTH-FIXES.md:49-50` acknowledges widening SA2-01 and names mandatory reviewer MFA as an alternative for the owner.

No canonical lifetime reviewer restriction or reset-provenance separation rule was found. `changeRole` in `membership.go` allows a lasting administrator to remove and later grant a reviewer role; the new reset guard inspects current roles only. Revoke/reset/regrant is consequently a SOURCE-INFERRED limitation of this proposed hardening. It was not reproduced, no new probe was written, and it is not classified here as a demonstrated violation of an existing lifetime-history contract. Do not invent such a contract in this consolidation.

Recommendation: separate/defer executable F02 until the owner explicitly chooses the recovery policy; preserve the useful compatible F04/F06 changes. Root posted review `5952884296` with this distinction. The role/history limitation must not be represented as full reviewer independence qualification.

## DEC014 and dependencies

A3-F02/DEC014 remains explicitly unresolved in the audit note and canonical decisions. There is no break-glass code change. The branch includes current main's merged PR71/72 lineage, but its unique auth implementation uses existing transaction, person lookup, MFA-challenge and throttle methods; no unmerged branch is needed for these unique changes. PR79 catalogue touches web files, so no direct auth implementation overlap exists.

## Executed existing tests

Exact archived source head `0a7ef6e`, Go 1.26.11, GOTOOLCHAIN=local, CGO_ENABLED=1:

```sh
go test -mod=readonly -race -count=1 -v -run '^Test(LoginStraddlingPasswordChangeGetsNoSession|ChangePasswordReplacesOnlyTheVerifiedHash|ChangePasswordGuessesAreThrottled|AdminCannotResetReviewerPassword)$' ./internal/service
```

All four top-level tests passed, including both straddled-login subtests, in 3.214 seconds. This confirms the supplied direct-path regressions at the reviewed head. It does not qualify untested role-history sequences or approve recovery policy. No baseline source was reverted and no independent fail-before claim is made; the audit note's fail-before statement belongs to its author.

## Preservation

Current GitHub branch enumeration requires two pages (100+1): 101 branches total. All original 89 remain present. Only original `main` has changed head; the other 88 original heads remain unchanged. Exact current names/heads and original comparison are in `AUTH82-REVIEW.json`.

Root checkout and repository source were not changed. Tests ran only from an isolated git-archive copy. No new probes, commits, pushes, PR mutations, deployments or releases were performed.
