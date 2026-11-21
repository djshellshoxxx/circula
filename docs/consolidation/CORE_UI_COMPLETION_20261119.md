# Core UI completion and qualification checkpoint

The subsequent publication, full normal-suite results, security checks, recovery
follow-ups and current gates are recorded in
[CORE_VERIFICATION_20261119.md](CORE_VERIFICATION_20261119.md).
The source-completion checkpoint below is preserved with its original boundary.

## Integration authority and source boundaries

The owned continuation branch is
`integration/core-ui-completion-20261119`, based on local
`44aa1a352bd9f7aa28e443f5a9ab3c4a3aedc412`. That base has tree
`721e0e803245a58eac3d01367371434c22cc5cb2`, exactly matching the frozen
PR #109 provider head `5db3efc084e6a807bb4702c09d3235490d42b71d`.
PR #109 remains targeted at the authoritative broader Core integration,
PR #90 at `839e86935767288e594914ad711486616a604eca`.

The last observed main remains
`ead63b5ce24b2bcf8352d75fd0d1ff7be711f625`. Both current provider candidates
were mergeable, with all nine exact-head CI jobs queued. No main merge,
deployment or release has occurred. The earlier continuation record is
retained in [CONTINUATION_20261119.md](CONTINUATION_20261119.md); its source
and receipts describe the first batch, not a qualification of this newer one.

All 15 future modules remain isolated. SEC-026, GOV-021, reviewer-reset and
other recorded owner-policy holds are unchanged. No active peer branch is
overwritten by this continuation.

## Completed source in the second batch

The combined English catalogue contains **1,431 entries**, adding 344 to the
frozen first batch's 1,087. The independent source contributions are:

| Source commits | Scope | New entries |
|---|---|---:|
| `f522e013a29acd1ed4d2a80f01f35200e24ca832`, `5f64e92b9c22a96bad8123afccf87051a0a1c6d0` | Member actions, invitations and inactivity | 88 |
| `b19f7012e20caebc3c64e501845f1b6c10798e55`, `82ee9eb752858087100b7fd76d270cb3d28e05ab` | Network policy, suspension and closure | 96 |
| `8bd40428a769849eca9a00bcfa88791020e8e7bb`, `84a321bcdeedd5c09ded398c65163e3d33fa7baa` | Existing self/admin erasure presentation | 20 |
| `f8a9d7efdd94602685c1b98bea3455ccb65e3f57` | Shared error/help/notice pages and both historical-balance template loaders | 64 |
| `fefaf0124dc4270205b1f38220e362868b71476d`, `4ed93b3b0d148303ef2225b12088cfad3a6669e2`, `8a061c584fa7ca90ea26ce014ccc684573ecae89` | Announcements, administrator overview/stale queue and read-only operator access history | 74 |
| `9061e7f` | Remaining dispute amount and proof-refusal details | 2 |

The catalogue merges compare base/current/source values and reject conflicting
changes. Existing order is preserved. The obsolete self-only reset refusal is
not restored. A follow-up from `82b2c90eba250d229f360749fcc75d43b50789c4`
makes 14 string slots in 12 new shared-page messages explicitly positional,
without changing rendered English.

Invitation revocation is now driven by a separate boolean derived from the
existing invitation state, with Used/Revoked/Expired/Unused precedence and the
same strict expiration boundary. Display labels do not determine control
visibility. Known member/role/removal/suspension/operator statuses receive
display labels; unknown values remain ordinary escaped text.

Amounts retain exact minor-unit parsing and decimal strings; identifiers and
versions remain ungrouped. New `AroundCount` and `AroundTwoVersion` helpers
accept typed integers and return ordinary strings around template-owned
markup. Submitted reasons, names and historical text never become trusted HTML.
Existing route wrappers, form controls, CSRF, service calls, confirmation
proofs, statuses, authorization and retention policy remain in place.

## Focused verification and review

| Slice | Observed evidence |
|---|---|
| Member/invitation/inactive | Original templates and translated-label controls fail before; messages and six actual rendering tests pass after. The isolated old base required `-vet=off` for its known unrelated three format diagnostics; this is not a vet pass. |
| Policy/network | Baseline amount/count/catalogue cases fail; after messages and six actual-template checks pass. The isolated old base has the same disclosed vet limitation. The integration retains the literal-format repair. |
| Erasure | Missing-message and original-template probes fail before; complete messages package and all three actual-template tests pass after with default vet. |
| Shared pages | Complete messages package passes under race; production HTTP/error and both historical-balance loader tests pass, 35 test/subtest events, default vet. |
| Administrator history | Baseline control invariants pass; singular/grouped-count and all four catalogue probes fail. After, messages and seven new rendering tests plus two support-boundaries tests pass with default vet. |
| Dispute validation | Missing-message control fails; complete messages package passes after. Expanded actual HTTP confirmation tests await combined qualification. |
| Independent source reviews | Member/policy/erasure, shared-page and administrator-history reviews report no behavior or escaping defect in their bounded scopes. These are AI-assisted source reviews, not external security approval or provider approvals. |

Review found a singular-precision integration defect: the shared refusal helper
recognized only `decimal places`, so a one-decimal dispute refusal received
generic correction advice. The bounded follow-up in source
`0daa4d6a9553ef05854915bbf2ae1d27b39906a6` and
`4be42059a3f05e4ce973002e862319c3d4776392` accepts both forms and
pluralizes the precision correction. The singular message and actual refusal
both fail before; after, the complete messages package and all five direct
refusal cases pass with default vet, including zero/four decimals and absent
network/context controls. See
[the exact receipt](../qa/dispute-precision-remediation-qualification.md).

## Aggregate qualification boundary

The full race command on first-batch local `d11ec74` finished with exit 1:
`modernc.org/sqlite/lib` compilation was killed. Eight smaller packages and
467 test/subtest events passed; seven dependent packages failed to build.
The saved JSON includes the compiler diagnostic. This is a failed aggregate
run, not a passing race qualification or merely an unknown interrupted result.

After the executor restarted and temporary caches were restored, an isolated
`go build -race -p 1 modernc.org/sqlite/lib` passed in 101.212 seconds using
GOMAXPROCS=1, GOMEMLIMIT=250MiB, GOGC=40 and CGO_ENABLED=1. No compiler flags,
dependency versions or production checks were changed. That dependency build
clears the observed build barrier but does not itself qualify application tests.

The next full combined normal/race, fuzz, vet/static, source vulnerability and
browser results must identify their actual candidate source. The earlier full
normal failure and repaired web-only pass remain honestly bounded to their
listed sources. A standalone checkout preserves ordinary Go VCS stamping for
the restore-drill build. Heavy compilation is serialized on this executor.

## Remaining engineering and external gates

1. Freeze and qualify the complete second UI batch, then publish its exact tree
   through a draft stacked PR without moving PR #109's pending source head.
2. Exercise a successful SQLite COMMIT whose driver acknowledgment reports a
   nonretryable error afterward. This new test-only candidate must distinguish
   committed state from the existing pre-COMMIT failure control, verify live
   connection recovery and idempotent retry/reopen, and avoid claiming physical
   fsync/device/power-loss coverage.
3. Complete the candidate's exact-source Go, fuzz, static/source-vulnerability,
   browser and required provider checks. Local Playwright previously failed
   before its first page at the managed Chromium process-singleton socket;
   that attempt is not browser qualification.
4. Refresh branch/PR dispositions after publication and before eligible merges.
   The original 114-branch inventory and additive 128-branch/109-PR discovery
   remain preserved with their observed timestamps.
5. Merge only fully eligible heads through normal protected-branch procedures,
   verify the resulting main and reassess outstanding Core work.

The bounded remaining-work audit found no further confirmed unowned product/UI
implementation slice after these continuations. That finding is not full Core
completion: aggregate gates, broader storage-failure boundaries, manual screen
reader work, independent external security review, disclosure contacts and
recorded owner-policy decisions remain open. English externalization does not
approve new languages or establish complete locale/date/number support.
