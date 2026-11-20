# Main integration log

The owner authorizes qualified, compatible Core merges. This log records only
observed merges, never plans or pending checks.

## Starting state

- Fetched main: `ead63b5ce24b2bcf8352d75fd0d1ff7be711f625`.
- Candidate branch: `integration/core-recovery-20261119`.
- No merge into main has been performed by this continuation at this checkpoint.
- No release or deployment has been performed.

## Required integration procedure

Before a merge, re-fetch main and the exact source head, confirm all required
checks/reviews and compatibility, preserve concurrent changes, and record the
resulting main SHA. An old successful source PR does not qualify a newer
combined integration tree. Do not force-push main or bypass checks.

API-036/#90 and #96 remain active independent work. FNET-204/#92 targets its
future-module branch and must not be merged into main under this scope.

## Continuation snapshot — 2026-11-19 12:06 UTC

Main remains `ead63b5ce24b2bcf8352d75fd0d1ff7be711f625`. PR #101 merged into
PR #90's integration branch, not main. Current #90/#99/#102/#104 CI runs each
have nine queued jobs. No main merge, release or deployment is recorded. The
owned catalogue/security/qualification continuation targets #90; see
[CONTINUATION_20261119.md](CONTINUATION_20261119.md).

## Verification follow-up — 2026-11-19 13:45–13:46 UTC

Main remains `ead63b5ce24b2bcf8352d75fd0d1ff7be711f625`.
No main merge, release or deployment has occurred in this continuation.
Published UI PR #112 (`08d4b0078e8c308ae7f353849783913f4302882a`) targets
PR #109 (`5db3efc084e6a807bb4702c09d3235490d42b71d`), which targets
PR #90 (`839e86935767288e594914ad711486616a604eca`). Current checks are
pending or failed; the observed scanner failure has a locally reviewed repair.
Full candidate normal tests do not substitute for exact-head protected checks.
See [the current verification record](CORE_VERIFICATION_20261119.md).
