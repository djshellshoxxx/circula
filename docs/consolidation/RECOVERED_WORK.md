# Recovered work and provenance

## Current continuation

The current source, recovered implementation, completed repairs, verification
boundaries and remaining gates are recorded in
[UNIQUE_CORE_CONTINUATION_20261119.md](UNIQUE_CORE_CONTINUATION_20261119.md).
The owned continuation is [PR #127](https://github.com/djshellshoxxx/circula/pull/127),
which recovers PR #125 and adds the reviewed correctness repairs. Main was last
observed at `aa01563f`. Complete combined qualification and a main merge remain
pending in that record; focused worker passes do not qualify a later combination.
All earlier reports and the snapshot below remain historical source-specific
records. Existing future-module and policy holds continue to apply.

## Historical initial-recovery checkpoint

## Historical branch recovery

The complete 114-branch snapshot and its exact source heads are retained in
BRANCH_INVENTORY.md and `evidence/branch-inventory-20261119.json`. Existing
November 17/18 consolidation ledgers were reused only after checking their
recorded heads, ancestry, patch equivalence, and implemented behavior.

One historical branch required a new disposition:
`claude/quirky-babbage-y9fpmm@af69702be08aaf14cbe9a705ddf7f08796d5e976` had moved
beyond the obsolete prompt described by closed PR #39. Its later Python
traceability implementation was otherwise superseded by the current Go tool,
but its rejection of duplicate requirement definitions was missing from the
current qualification command.

The candidate ports that behavior into `scripts/traceability/main.go` using
the existing sorted duplicate diagnostics. It preserves diagnostic report
generation and legitimate repeated references. It does not restore the old
Python implementation or tracked bytecode caches. The before/after real-CLI
evidence is retained under `evidence/traceability-duplicate-recovery.md`.

## New Core repairs discovered during consolidation

| Repair | Source and verification |
|---|---|
| Export authorization inside the data snapshot | Review of main's separate audit/admission and export-read units; 24 administrator revocation cases, four erasure cases, and two retained-history controls. |
| Interrupted web exports | Review of ignored Export/ZIP Close errors and web panic recovery; CSV/JSON generation and finalization failures, valid archives, and early-refusal controls. |
| Listing pagination scope/filter retention | Review of `mine=true` links against a parser expecting `mine=1`, dropped filters, and phantom final pages; real HTTP continuations reproduce and verify the repair. |
| Shared validation and administrator reset catalogue | Continued the existing English catalogue design; 123 entries, positional complete sentences, exact amounts, and HTML escaping. |
| Marketplace and directory catalogue | 66 entries and a shared unit-display correction: singular credits and custom Hours no longer gain an incorrect suffix. |
| Freshness and session-warning catalogue | Seven complete messages, escaped data attributes and plain text browser interpolation; no session timing or extension-policy change. |
| Committed download regression | Real TCP client receives 4 KiB before a controlled SQLite row failure; original main incorrectly completes normally, repaired CSV/JSON downloads end with unexpected EOF and an invalid ZIP. |

These are new repairs, not additional historical branch merges.

## Active work preserved

PRs #78, #82, #87, #88, and #89 are already represented in the active PR #90
integration history. Their source-head checks do not qualify the combined
current #90 tree. API-036/#90, its #101 continuation (superseding closed #96), the separate
#99 consolidation, and FNET-204/#92 were not overwritten. Held governance/security branches and all future modules
retain their own source histories.

## Runtime recovery provenance

The first executor became unavailable during compilation. Its temporary
worktrees disappeared on restart. No interrupted command is counted as a test
pass. The persistent branch analysis survived; source patches were recovered
from recorded edits and immutable GitHub blobs, checked against the exact main
baseline, formatted again, and tested after restart. The recreated UX patch
matches its recorded 662 additions/75 deletions and all published blob hashes;
the lost local commit itself is not claimed as a remotely published revision.

The additive refresh in BRANCH_REFRESH_20261119.md accounts for 120 branches,
including every new and moved head observed at 10:59:30 UTC. PR #102 was
published with a provider tree matching local 5da8bc7 byte for byte. A later
executor restart interrupted broader qualification; the published source and
persistent worktrees survived. No interrupted run is passing evidence.
