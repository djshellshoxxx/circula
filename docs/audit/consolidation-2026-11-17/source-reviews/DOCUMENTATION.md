# Workstream E: documentation and specification reconciliation

## Reviewed baseline and local deliverable

- Core baseline: `ef54238ee9ebc2184919b6a8ec30c3398df55ed5` (PR #69 merge).
- Local branch: `integration/throttle-documentation-20261117`.
- Local commit: `f2954c80181c958818b11d9e1e7a9f19d83dd881`.
- Tree: `43afbf34807483df1bab020c773f8791c7dfdd60`.
- Worktree: `/workspace/scratch/9ceb45b04668/audit-e-worktree` (clean).
- No remote writes were performed by this workstream.

The focused change updates `docs/specs/addenda/SEC-007.md`,
`docs/specs/addenda/SEC-008.md`, `docs/dev/recovery.md` and adds
`docs/audit/A3-F21-THROTTLE-DOCS-2026-11-17.md`. No executable code, migration,
financial behavior, governance decision, retention policy or break-glass rule
changes. No changes to TRACEABILITY, TODO, HANDOFF or another agent's files.

## Findings corrected

| Finding | Old statement | Current implementation and correction |
|---|---|---|
| A3-F21 | Locks double from 15 minutes to 24 hours | `internal/service/throttle.go` uses a fixed lock. Defaults and bounds in `throttle_config.go` are 5 local attempts (configurable 4–5), 100 login-wide attempts across addresses (configurable down to the local budget), and a 15-minute lock (configurable up to 24 hours in whole seconds). No exponential backoff. |
| A3-F16 stale wording | MFA sign-in and management share one counter | `mfa.go` uses separate sign-in and management keys. Sign-in locks refuse management calls, but management failures do not lock sign-in. The dated 2026-11-04 amendment is retained. |
| SA2-01a / SA2-03 stale runbook | One ordinary administrator may reset another administrator, with a high-risk flag | `recovery.go` refuses ordinary administrators resetting an administrator in this network or a person with any non-departed membership in another network. Existing operator recovery remains available. |
| A3-04 / BR-05 stale wording | Invalid recovery attempts perform Argon2 work | Refused unknown, malformed, wrong and used-code attempts perform no Argon2 work. An admitted attempt with a valid unused code hashes the new password before atomic consumption. |

The documents also distinguish per-address budgets from the login-wide budget,
record the half-allowance restored on lock expiry, and distinguish defaults from
operator overrides. Password-reset dual approval remains an owner decision.

## Verification

The following 13 existing tests passed with `go test -count=1`, filtered to
these names in `./internal/service` and `./cmd/circula`:

- `TestConfiguredAuthenticationThrottleLoginAndGlobal`
- `TestConfiguredAuthenticationThrottleRecoveryAndSensitivePassword`
- `TestConfiguredAuthenticationThrottleMFA`
- `TestAuthenticationThrottleRejectsUnsafeServiceConfiguration`
- `TestStolenSessionCannotLockOwnerOutOfMFA`
- `TestMFASignInLockNotRenewableByOneFailure`
- `TestMFASessionThrottleBoundAndUnlock`
- `TestPersistentAuthThrottleSettings`
- `TestPersistentAuthThrottleServiceWiring`
- `TestAdminResetGuards`
- `TestAdminResetRefusesMultiNetworkTarget`
- `TestA3_04_RecoverSkipsHashingBeforeCodeCheck`
- `TestRecoveryRefusalsDoEqualWork`

Observed package results: the initial seven service tests passed in 0.190s;
the two CLI tests passed in 0.567s; the final four recovery tests passed in
1.041s. `git diff --check` passed. No new test was necessary for this
documentation-only patch. Root retains aggregate qualification and independent
review responsibility before publication/merge.

## Independent PR #70 scope review

Reviewed head: `1ee699f4e41c2779f32bd27063b14a4ff22e3a7d`.
Comparison baseline: `b53f757682bff5f488be9c2686c2e8aa8ad4054a`.

The diff is exactly six files under `docs/audit/2026-10-28/`:

1. `coverage-gaps-circula.md`
2. `interim-verification-circula.json`
3. `interim-verification-circula.md`
4. `wip-fixes/README.md`
5. `wip-fixes/closure-export.patch.txt`
6. `wip-fixes/sweeps.patch.txt`

No executable source or workflow changes. The verification record explicitly
pins the historical `bb5a36f` snapshot, says the synthesis pass did not finish,
and distinguishes the later five Medium fixes at `c34d947` from unresolved
A3-F02 / DEC-014. `c34d947b6bb4b8f75d79ccc094a167e20b776962` is an ancestor of
the reviewed core baseline. The WIP README labels the text patches unapplied
and unreviewed; one is incomplete. Coverage gaps remain investigation leads,
not new findings or proof of a completed current audit.

**Scope verdict:** suitable to merge as a historical evidence handoff after
root's required CI/review checks. It is not executable remediation, does not
establish that all 55 historical findings remain present, and does not close
the current security review. No overlap with this workstream's four files.

## Historical specifications and remaining decisions

The PR #69 archive index at `docs/archive/specs-historical/README.md` preserves
the source PRs/SHAs, explains colliding requirement IDs, and explicitly keeps
`docs/specs/` authoritative. Useful historical ideas remain readable without
being promoted into financial rules or core requirements. No further copying
into canonical requirements is justified by this review.

| Topic | Evidence | Recommended handling |
|---|---|---|
| Break-glass powers and notice | `docs/specs/addenda/SEC-013.md`, `SEC-011-owner-decision.md`, DEC-014 and A3-F02 | Preserve the open decision. Owner must choose covered CLI actions, lasting effects, second approval and notice policy. Do not infer approval from working operator commands. |
| Dual control beyond direct adjustments | `docs/specs/addenda/SEC-011-owner-decision.md` | Preserve open scope choices for reversals, limits, policy changes, aggregate thresholds and password recovery. Historical recommendations are not approval. |
| Recovery channel and code lifetime | SEC-008 deviations and DEC-009 | Current channel-free, single-use codes do not resolve verified-contact requirements or the proposed 60-minute recovery token. Owner must choose the channel/fallback and policy before behavior changes. |
| Personal-data retention and legal holds | `docs/specs/addenda/SEC-018.md`, DEC-020 | Keep open: retention clock, legal hold, immutable audit free text, counterparty memos and backup retention policy. Existing explicit erasure is not a retention decision. |
| Core SEC-006 recommendation | `docs/specs/09-security-privacy.md` still recommends increasing delays; A3-F16 remediation and current implementation use bounded fixed locks | Keep factual addenda accurate; reconcile the normative recommendation explicitly in a future spec revision instead of silently changing authentication semantics. |
| Stale build-status rows | `docs/DECISIONS.md` still says MFA, break-glass and log retention are not built; newer `docs/specs/13-assumptions-risks-decisions.md` rows identify the implementations | A future factual edit may label/update the old snapshot while retaining every open owner decision. No policy acceptance should be inferred from implementation. |
| Other stale addendum descriptions | SEC-018 decision 6 says MFA tables do not exist, while `internal/service/erasure.go` calls `DeleteMFA`, `ReplaceRecoveryCodes` and `DeleteAccountRecoveryCodes`; SEC-013 still calls password reset a future command | Narrow factual follow-up is appropriate. These were outside the authorized throttle/recovery patch; do not apply wider policy changes under cover of wording cleanup. |

No release approval, completed independent security audit or owner decision is
claimed by this workstream.
