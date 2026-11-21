# Independent Circula source-artifact and traceability review

Reviewed main: `87651d36558a8e421e41fbd76c514045d88c74ed`.
Whole tree: `43e65ee91a90ff70b70cc69d0b517b8367a2585f`.

This review downloaded and classified existing provider evidence. It ran no
product tests, changed no repository files, and performed no remote mutations.

## Provider artifact and source identity

Artifact `14685119490`, `test-evidence-ubuntu-24.04`, belongs to Core run
[44843655337](https://github.com/djshellshoxxx/circula/actions/runs/44843655337),
Linux job `136657974828`. The downloaded ZIP is 137,945 bytes; its independently
computed SHA-256 exactly matches provider metadata:
`d4368bca6e6d52bd22ac3e8a1acef798ac501322652aaa500a80315197ed8476`.

Only the two expected regular-file entries were read: `traceability-results.json`
and `TRACEABILITY-results.md`. No archive path was used for general extraction.

The execution JSON records clean synthetic PR merge
`6eb49aa2b2c7859fd2a8ceda207ddbb01eaa9d65`, command
`go test -json -count=1 -race -timeout 30m ./...`, and exit code 0.
An independent reproduction of the recorder's source-digest algorithm over the
current clean main checkout's 2,023 included files, excluding only the generated
`docs/TRACEABILITY.md`, exactly matches the artifact digest:
`9de685071744f0b9e260d53fc7d934cb4289c3b10f84a020db8e319227b77100`.

This binds the retained evidence to the current source bytes. It does not replace
the separate actual-main push workflows or constitute rerunning the product.

## Terminal-event classification

| Classification | Passed | Failed | Skipped |
|---|---:|---:|---:|
| All terminal events | 3,274 | 0 | 8 |
| Named tests, including subtests | 3,258 | 0 | 6 |
| Top-level named tests | 1,293 | 0 | 6 |
| Top-level `Test` functions | 1,270 | 0 | 6 |
| Top-level `Fuzz` entries | 23 | 0 | 0 |
| Subtests | 1,965 | 0 | 0 |
| Packages | 16 | 0 | 2 |

The JSON has 3,282 terminal events with no duplicate package/test identities.
Do not report 3,282 as a count of passed tests. The two skipped packages are
`internal/storetest` and `internal/testutil`, which contain helper code without
named package tests.

The six named skips are preserved:

| Test | Classification from pinned source |
|---|---|
| `TestBackupStagingRefusesForeignOwnedParent` | Requires root on Unix and available `chown`; terminal-only JSON does not preserve which source skip condition executed. |
| `TestBackupWorkLeaseChild` | Subprocess helper; parent `TestBackupWorkLeaseProtectsOtherProcessAndReclaimsCrash` passed. |
| `TestRestoreSwapCrashChild` | Subprocess helper; parent `TestRestoreSwapCrashRefusesStartupAndPreservesFiles` passed. |
| `TestQA010CrashChild` | Subprocess helper; parent `TestQA010CrashDuringPosting` passed. |
| `TestLoadConcurrentTrades` | Opt-in `CIRCULA_LOAD=1` test; separate API/load qualification is distinct evidence. |
| `TestLoginTimingStatistical` | Opt-in `CIRCULA_TIMING_TEST=1`; machine-sensitive threshold remains subject to DEC-023/QA-040. |

`TestDefaultIDGeneratorPersistsCanonicalUUIDv7` and
`TestInjectedIDGeneratorReachesPersistedPerson` both have passing events.

## Thirteen built requirements without automated test citations

The exact current list is:

`VIS-011, VIS-012, ARC-001, ARC-013, ARC-019, ARC-024, DAT-003, UX-007,
UX-042, UX-043, UX-058, GOV-017, OPS-001`.

These are all inherited from the nineteen-row list on preceding main
`aa01563fcc891da60fefebb8335eca27855678e8`. There is no newly uncited built row.
Three direct citations were added for ARC-005, ARC-014 and DAT-017; SEC-031 was
honestly changed to partial; meaningful DAT-001/DAT-002 regressions now exist and
passed in this artifact. These six changes account for 19 becoming 13.

| Group | IDs | Remaining evidence boundary |
|---|---|---|
| Negative-scope inspection | VIS-011, VIS-012, GOV-017 | Review monetization absence, current deferrals, and dispute payload/route scope. A keyword-only test would not establish those claims. |
| Release and operational acceptance | ARC-001, ARC-013, ARC-019, ARC-024, OPS-001 | Same-artifact deployment qualification, native packaged-artifact smoke, dependency/license maintenance review and release receipts. |
| Identifier privacy inspection | DAT-003 | Production allocator input and scope review; no statistical anonymity claim follows from UUID format. |
| Content and accessibility acceptance | UX-007, UX-042, UX-043, UX-058 | Wording approval, global keyboard/drag inventory, pointer cancellation, reduced-motion and flashing acceptance. |

The prior bounded source review is
`docs/consolidation/evidence/unique-core-20261119/uncited-built-acceptance.md`.
The earlier nineteen-row correction record is
`docs/audit/consolidation-2026-11-17/source-reviews/BUILT-REQUIREMENT-EVIDENCE.md`.
Their historical observations remain source-specific. No new accepted Core
implementation defect was identified by this bounded evidence review.

## Eleven unresolved frozen legacy citations

The resolver semantics in `scripts/traceability/scan.go` were independently
applied to all 906 frozen TSV rows. Exactly 895 resolve. All eleven unresolved
rows are **code citations**, not lost test-function citations. All eleven are
identical on preceding main `aa01563...` and earlier audit source `244e135...`.
Every cited file still exists.

| TSV line | ID | Frozen path/symbol | Current explanation |
|---:|---|---|---|
| 30 | DEC-001 | `internal/store/migrate.go` / `migrate` | Current function is `migrateTo`. |
| 206 | FR-014 | `internal/service/departure.go` / `DepartMember, PreviewDeparture` | Both functions exist; the frozen combined string is not one source symbol. |
| 207 | FR-014 | `internal/web/h_gov.go` / `leaveForm, leavePost` | Both handlers exist; the frozen combined string does not match. |
| 450 | DAT-042 | `internal/store/migrate.go` / `migrate` | Current migration function is `migrateTo`; direct citation exists. |
| 453 | DAT-044 | `cmd/circula/serve.go` / `checkPragmas` | Startup now calls `svc.CheckStorage`; the store owns `CheckPragmas`. Migration/verification assertions have current citations. |
| 474 | API-010 | `internal/api/h_core.go` / `notImpl` | Token handling is in `h_tokens.go`; admin scope remains explicitly documented deviation D-07, with 501 refusal. This review does not resolve that contract choice. |
| 560 | API-064 | `internal/api/h_core.go` / `notImpl` | Announcements are implemented in `h_f2.go` with direct requirement citations; the old stub is gone. |
| 726 | GOV-003 | `internal/service/networks.go` / `UpdatePolicy, noticeDays` | Both functions exist; only the frozen joined-string match fails. |
| 740 | GOV-009 | `internal/service/join.go` / `RedeemInvitationTerms, JoinNetworkTerms` | Both functions exist; only the frozen joined-string match fails. |
| 796 | OPS-011 | `cmd/circula/serve.go` / `checkPragmas` | `service.CheckStorage` delegates to the store's `CheckPragmas`; current direct citations cover these paths. |
| 824 | OPS-022 | `cmd/circula/schema.go` / `checkSchemaNotNewer` | `SchemaState.CheckNotNewer` lives in `internal/store/inspect.go` and is called by startup and command database admission. |

The frozen TSV explicitly says not to add rows. Retain it and the historical
895/906 count. These exact misses do not justify recreating old functions,
loosening resolver checks, or declaring new feature failures.

## Recommended next action

Preserve the original ZIP, this classification and the matching source digest
in the finalization evidence. Keep actual-main push results and remaining
manual/operational/policy acceptance separate. Do not add tests merely to make
the citation count zero or change the qualified runtime head for these inherited
warnings. The API-010 D-07 admin-scope contract remains explicitly unresolved.

Machine-readable counts, provenance, exact rows and skip classifications are
in `test-evidence-review.json`; `legacy-citation-review.json` retains the frozen
row comparison. Repository source was unchanged throughout this review.
