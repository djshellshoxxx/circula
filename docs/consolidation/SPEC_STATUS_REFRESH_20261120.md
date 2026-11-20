# Core source status refresh — 2026-11-20 Pacific

Reviewed integration source: `b00706e`, including the ordinary reporting,
reviewer projection and metrics work. This review corrects nineteen obsolete
absence claims in the specification summary tables. It preserves their
historical status labels, counts, normative requirements and owner decisions.
The labels are not a current release or security attestation.

## Existing implementation reconciled

| Area | Requirements | Source conclusion and remaining boundary |
|---|---|---|
| Network identity and membership | VIS-002, PER-010, PER-015, PER-016 | About/README acceptance text, joining terms, limit notices, adjustment dual control, network suspension, departure notices and self-export are implemented. Broader control policy and manual accessibility acceptance remain separate. |
| Export and migration evidence | ARC-020, DAT-048 | Service-owned exports and both standalone readers exist. The original-schema golden fixture records its measured migration duration. Later schema versions, reference hardware and a release budget need their own evidence. |
| Shared interface behavior | UX-022, UX-024, UX-031 | Source includes summary focus, nonsecret value retention, zone-labelled `time` elements and persistent suspended-membership guidance. HTTP checks are distinct from real-browser and screen-reader execution. |
| Authentication and membership controls | SEC-003, SEC-004, SEC-006, SEC-027, SEC-032 | Role timeouts, session-bound CSRF, network join/invitation limits, revocation/key rotation and grant controls exist. Unresolved owner recommendations are not silently made defaults. |
| Governance and operations | GOV-001, GOV-011, GOV-027, OPS-010, OPS-033 | Binding-event policy versions, removal challenges, linked support boundaries, rotation procedure and per-network consistency freezes exist. Support commitments, deployed drills and physical failure qualification remain separate. |

Each changed note identifies the relevant implementation or test source. Source
inspection prevents duplicate implementations; it does not make every listed
acceptance criterion pass. In particular, no future module, general escalation
privilege, wider quorum rule or owner-held default is enabled by this refresh.

## Execution provenance

The full normal run at `c7104872cc2fe049b62d175fbb95a9751bf31bdd` completed
all packages without a compiler kill. It recorded 3,317 passing test/subtest
events and one failure: the migration-review registry had not yet recognised
the preserving report-table rebuild. The narrowly reviewed registry repair
and its focused tests are recorded separately. That run is not a full pass,
and it predates the historical-limit and explicit-case-context additions.

The full linter passed with zero issues at
`c9dbc846597a496592a434814a4a4f0b22b0f014`. Its source remained clean and
unchanged for the complete run. The final integrated source still needs its
own combined qualification. Neither result is promoted to a later source.

Current completion and exact-source evidence are maintained in
[the continuation record](UNIQUE_CORE_CONTINUATION_20261119.md) and
[its evidence index](evidence/unique-core-20261119/README.md).
