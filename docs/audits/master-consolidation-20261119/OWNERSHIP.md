# Master consolidation ownership registry

Starting main: `ead63b5ce24b2bcf8352d75fd0d1ff7be711f625`.

This registry records the master workstream's assignments and discovered external ownership. Current publication identities and targets are in `PUBLICATIONS.json`; live heads, changes and required checks must be refreshed before integration. Historical snapshots are retained in `BRANCH_INVENTORY.*` and dated `BRANCH_DELTA.*` records.

## Reserved external work

| Workstream | Observed branches / PRs | Reserved scope and handoff |
|---|---|---|
| API-036 and combined Core | #90 `integration/comprehensive-audit-20261119`; #101 included in that topic; #111 qualification continuation; #96 superseded | Session/authentication handlers and interfaces, token/session tests, store/service integration, migrations, OpenAPI and generated traceability. No copying, retargeting, rebasing or integration of unfinished source. |
| Core recovery and UI | #99/#119, #102, #109 and #112; active workspaces include `beb70d57deba`, `bfba682cba37`, `516d1916c4d6` and `93aeb012c206` | Transaction/store/recovery, export-admission, catalogue/UI, shared messages, shared specifications and CI. Main-merge eligibility requires completed handoff and current checks. |
| FNET-204 feed | #92 and its #98 topic inclusion; related signed-cursor branches | All feed code, pagination/cursors, performance, tests, workflow and evidence. Remains isolated and disabled/HOLD. |
| Notifications and PWA refresh | #100 and #95 | Their Core/workflow/PWA work remains reserved. The master's four transport files and its new separate native-transport workflow/note are reviewed changes; the archived notification implementation is not substituted for the active design. |
| Core governance and policy holds | #107/#85 GOV-021; SEC-026 and reviewer-reset policy streams | Separate from the master's isolated FNET-208 advisory module. No policy adoption or unfinished source copying. |
| Federation README | #106, targeting the master's #105 source | `future/federation/README.md` is kept byte-identical in the full recovery. The original archived README has a separate filename; current information goes in owned recovery documents. |
| Recurring baseline/digest handoff | #110, targeting the master's #97 source | Two accepted-main Core files and the original calendar-test digest naming correction. The full recovery preserves the legacy test under a different filename; handoff/rename reconciliation must precede a stacked merge. |
| Matching continuation | Existing future-module automation/sequence | Fresh ownership/handoff is required before altering its current branch. No competing implementation is introduced from the recovery archive. |

Topic inclusion, a draft, an older green check, or source availability does not establish a completed handoff. These reservations apply to unpublished work as well as remote heads.

## Master assignments

| Workstream | Agent / reviewer | Owned scope | Current boundary |
|---|---|---|---|
| Go export verifier | `independent_triage`; independently reviewed by `branch_recovery` | `cmd/circula/verify_export.go`, ambiguity and boundary regression files | Reviewed source published in #104; Go bytes remain identical to the preceding provider-qualified code. New-head CI and shared traceability remain separate gates. |
| Python export verifier | `independent_triage`; independently reviewed by `branch_recovery` | `scripts/read-export.py`, `scripts/test_read_export.py` | Complete ZIP reads, manifest types/ordered aliases and JSON constants are independently reviewed and composed. Final 38-test suite passes; current native provider gates remain. |
| Native Python reader CI | Master integrator; reviewed by `recurring_module` | New `.github/workflows/export-reader.yml` | Uses only the existing allowed checkout action and hosted interpreters, recording exact versions and source. Shared CI is unchanged. |
| Recurring recovered implementation | `recurring_module`; Calendar/workflow review by `branch_recovery`, full service and follow-up review by root | `future/recurring/**`, owned dedicated workflow | #118 is stacked on #97. Full current Linux race/vet/checksum and two bounded fuzz campaigns pass. External #110 remains reserved. |
| Federation recovered implementation | `recurring_module`; complete eight-file source review by `independent_triage`, composition/workflow review by `branch_recovery` and root | `future/federation/**` except reserved README; owned dedicated workflow | #117 is stacked on #105. Current Linux real-driver race/vet/checksum and three bounded fuzz campaigns pass. External #106 remains reserved. |
| Notification transports | `independent_triage`; reviewed by `branch_recovery` | Four `pushsend`/`smtpsend` source/test files, new `.github/workflows/notification-transport-native.yml`, and uniquely named qualification note | #103 preserves current AAD/session fixes. The preceding source passed Linux module and all seven executed Core jobs. The reviewed new native-workflow publication requires fresh provider results; no shared #100 workflow is edited. |
| Archived notification repairs | `independent_triage`; reviewed by `branch_recovery` | Separate archive-based UTF-8 identity and JSON-alias patches/evidence | Preserved as portable patches, not applied over the competing current notification architecture. No composed-patch race result is claimed. |
| Analytics recovery | Master integrator; restoration/workflow review by `branch_recovery`, accepted-main follow-up by `independent_triage` | `future/analytics/**`, owned dedicated workflow, allowed two-file baseline below | #108. Original Go/module files remain exact. Current archive race/vet/checksum and 100,000-execution fuzz campaign pass. |
| Receipts recovery and exact integers | Restoration/docs/workflow by root; numeric repair by `independent_triage` and independently reviewed by root; composition reviewed separately | `future/receipts/**`, owned dedicated workflow, allowed two-file baseline | #113. Original tests retained. Repaired full race, vet, checksum and all three bounded fuzz campaigns pass. Production-key/interoperability/Core adapter acceptance remains separate. |
| Areas, advisory governance, messaging recovery | `branch_recovery`; new source/workflow review by `independent_triage`, final evidence composition by root | Respective module, new dedicated workflow and allowed two-file baseline | #115/#114/#116. Complete Linux real-driver suites pass; areas/governance parser fuzz passes. Messaging native storage/fuzz remain separate. |
| Windows messaging privacy | `independent_triage`; independent review required before publication | Separate `circula-messaging-windows` worktree from final messaging recovery; platform storage helpers, relevant tests/contract and owned workflow | Conservative process-user owner-only ACL/directory/sidecar/backup profile. Existing unsafe state must fail without silent security mutation. Cross-compilation is not native acceptance. |
| QR camera lifecycle | `recurring_module`; independently reviewed by `branch_recovery` and root | Current-target QR scanner/tests, new dedicated Node workflow, module scope note and allowed two-file baseline | Published as #120 with all 27 local cases and two independent probes passing. Current scanner/vendor/module Go/API preserved. Native provider/manual/browser acceptance remains; no PWA/shared UI work. |
| Four divergent module comparisons | `recurring_module` | Read-only favorites, QR, discovery and onboarding comparison and current ownership | Complete report in `FOUR_MODULE_RECONCILIATION.md`; preserves richer current behavior, identifies portable fixes and actual policy/API conflicts. QR repair is published; other hypotheses are not claimed as defects. |
| Onboarding retention investigation | `recurring_module`; independent review required if a repair is reproduced | Separate `circula-onboarding-retention` worktree, current module SQL retention implementation and meaningful regression tests | Fresh 35-PR ownership survey found no overlap. Use current pinned mattn SQLite driver/tag and preserve global strict-before retention policy, schema and API. No Go until the Windows author releases the sole slot. |
| Inventory, recovered archive QA and independent review receipts | `branch_recovery` | `BRANCH_INVENTORY.*`, `BRANCH_DELTA.*`, `REVIEW.*`, `ARCHIVE_RECOVERY.*`, `ARCHIVE_MODULE_QUALIFICATION.json`; task-owned evidence | All 48 archive refs retained. Current unchanged-archive Linux qualification is 15/15; earlier 14/15 and failed attempts remain preserved. |
| Master reporting/publication | Root | This uniquely named audit directory, `MASTER_REPORT.md`, `OWNERSHIP.md`, `CONTINUATION.md`, `FUTURE_MODULE_STATUS.md` and reviewed publications | Publish frozen exact blobs/trees, preserve failures and scope, refresh current provider/handoff state. No shared generated file or workflow is taken over. |

## Exact accepted-main baseline allowance

For analytics, receipts, areas, advisory governance, messaging, federation and QR #120, the coordinator separately authorized only these two complete files already merged into main:

| File | Accepted blob | Purpose |
|---|---|---|
| `cmd/circula/auditanchor.go` | `e2a0d8ee5d2d340b5222fcc2b86ad50104255587` | Missing `os` import already corrected on main |
| `internal/service/departure_pagination_regression_test.go` | `3b117ecfe2f01e7e2363b9d201503cde6e0f5595` | Fixture cap 1000 permits the existing 501-proposal regression; production limit unchanged |

Their introducing commits and exact path comparisons are preserved in module recovery receipts. This limited allowance does not authorize other Core, root dependency, migration, contract, generated-document or shared-CI changes. Recurring's equivalent handoff remains with #110.

## Shared execution and integration rules

Only one heavy Go build/test stream runs at a time in the constrained shared environment. Commands have explicit bounds and actual result receipts. Only this workstream's unused reproducible caches may be removed; other workers' files/caches and all source/evidence are preserved. Interrupted or resource-killed commands are never treated as passes.

The generated DAT-040 traceability change remains an unapplied patch for the active owner's combined regeneration. The older patch is provenance, not a patch to apply blindly to a moving shared document. All future modules stay disabled/HOLD and out of Core. Normal expected-head checks, required reviews and ordinary repository protections govern any eventual merge. No deployment, release, branch deletion or shared force push is included.
