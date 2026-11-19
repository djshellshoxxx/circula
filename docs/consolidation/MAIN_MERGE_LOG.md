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
