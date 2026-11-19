# Recovered work and provenance

> Later consolidation: [combined recovery and validation status](../audits/CONSOLIDATION_CONTINUATION_2026-11-19.md). This file retains its original source snapshot and execution scope; final candidate status is recorded in PR #99.

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

These are new repairs, not additional historical branch merges.

## Active work preserved

PRs #78, #82, #87, #88, and #89 are already represented in the active PR #90
integration history. Their source-head checks do not qualify the combined
current #90 tree. API-036/#90, its #96 cursor follow-up, and FNET-204/#92 were
not overwritten. Held governance/security branches and all future modules
retain their own source histories.

## Runtime recovery provenance

The first executor became unavailable during compilation. Its temporary
worktrees disappeared on restart. No interrupted command is counted as a test
pass. The persistent branch analysis survived; source patches were recovered
from recorded edits and immutable GitHub blobs, checked against the exact main
baseline, formatted again, and tested after restart. The recreated UX patch
matches its recorded 662 additions/75 deletions and all published blob hashes;
the lost local commit itself is not claimed as a remotely published revision.
