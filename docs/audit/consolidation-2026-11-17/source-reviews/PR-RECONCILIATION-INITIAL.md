# Independent PR and archive reconciliation

Reviewed canonical baseline: `b53f757682bff5f488be9c2686c2e8aa8ad4054a`. Consolidation candidate: `163fe1596bd7cfb8c2a22891629fc4459e6b639c`.

## Verdict

PR #69 passes this source/documentation review. The eight historical PRs #1, #2, #13, #16, #20, #29, #35 and #42 may be closed as superseded only after #69 is merged and required test/CI gates pass. At handoff, root reports #69 merged as `ef54238ee9ebc2184919b6a8ec30c3398df55ed5` after all 14 local packages, vet, module verification, hygiene, action pins, traceability and Python checks passed; an independent reviewer confirmed the four new tests by mutation. Historical PR closures remain for root to execute with fresh-head checks. Those run results are root/other-reviewer evidence, not tests executed by this workstream.

No canonical financial rule, ledger semantics, migration, future-module implementation or break-glass policy changes appear in the reviewed #69 diff. Only docs/specs/README.md changes within canonical specifications, adding a historical-archive pointer. docs/DECISIONS.md and the normative specifications remain unchanged.

## Scope and evidence

All 70 initial PR heads were reconciled. All are available as refs/pr-heads/{number}; 38 of historical #1–#68 heads are ancestors of baseline main. Ancestry proves their source history is retained; squash/replacement cases were checked by path content and current implementation.

The eight proposed closure heads match both their fetched PR head and current branch head. PR #39 is an exception elsewhere: its closed PR head remains 92d204dc, while its branch advanced to af69702b. The later branch work is not classified as obsolete; Agent A reviews it separately.

31 added files carry archive provenance banners. After banner removal, 21 bodies are byte-identical to their sources. The remaining 10 differ only by documented link/format repairs: 195 Markdown destinations map to their exact source-commit path or the correct relocated archived document; two bare URLs gain https://; trailing whitespace and equivalent Markdown hardbreak formatting change. All 195 changed Markdown destinations were checked independently. Do not describe the whole archive as byte-identical.

Evidence artifacts: archive-provenance.json records source SHA/path for each archive; archive-link-verification.json records each changed destination with zero unverified mappings; archive-content-diffs.txt preserves all body diffs for inspection. proposed-closure-comments.json contains individual evidence comments for root to post after merge.

## Unresolved decisions retained

A3-F02/DEC-014 remains unresolved. The archive index keeps draft/canonical financial conflicts, SEC-011 quorum scope, PRIV-01/02 retention, WAL checkpoint design, GOV-021 recovery, future CLI semantics and other owner decisions explicit. PR #67 and the 15 future modules remain outside core. NEW-01 is described as unverified hardening rather than a proven fixed defect.

## PR disposition ledger

| PR | Reviewed head | State | Disposition | Evidence |
|---|---|---|---|---|
| #1 | `22e9202f19f5` | open | archive-and-supersede-after-69 | Three unique brainstorming documents preserved in docs/archive/specs-historical/pr1/; short competing specifications and early wiki superseded by canonical docs/specs/ and current wiki. No application code in the PR. |
| #2 | `01b6e5c7cfc5` | open | archive-and-supersede-after-69 | Unique lifecycle, network-policy, notification, legal and standards documents preserved in docs/archive/specs-historical/pr2/; prior-art and regulatory notes preserved under docs/research/. Remaining draft requirements superseded; policy conflicts and import proposal indexed without adoption. No application code. |
| #3 | `d96f2e75634b` | closed | already-closed-obsolete | The 16 changed paths contain specification shells with explicit requirements-not-authored status and placeholder comments, not implemented requirements. |
| #4 | `fc488445af5f` | closed | integrated-by-ancestry | Head fc488445af5f43165a7103f1c1fd12483b39d448 is an ancestor of canonical main b53f757682bff5f488be9c2686c2e8aa8ad4054a. |
| #5 | `db7826495ec2` | closed | already-closed-obsolete | README/wiki stub says no application exists. Current main has application, install/build documentation and wiki; no unique executable behavior. |
| #6 | `f8278a047392` | closed | integrated-by-ancestry | Head f8278a047392541a077d57c560823471053d1e1d is an ancestor of canonical main b53f757682bff5f488be9c2686c2e8aa8ad4054a. |
| #7 | `fcd9fc587848` | closed | already-closed-superseded | Only unique diff from merge base is docs/HANDOFF.md. Application base claude/circula-build is an ancestor of main; old handoff is superseded. |
| #8 | `d350474e4952` | closed | integrated-by-ancestry | Head d350474e49522c34213d90a66904b296b9509f6c is an ancestor of canonical main b53f757682bff5f488be9c2686c2e8aa8ad4054a. |
| #9 | `d39bfef12d4a` | closed | integrated-by-ancestry | Head d39bfef12d4ac1430815fa49ad799fb1c4bf9964 is an ancestor of canonical main b53f757682bff5f488be9c2686c2e8aa8ad4054a. |
| #10 | `ae512e51d4f7` | closed | integrated-by-ancestry | Head ae512e51d4f7a611119a5567225b862969770207 is an ancestor of canonical main b53f757682bff5f488be9c2686c2e8aa8ad4054a. |
| #11 | `a04fb74e2ed7` | closed | integrated-by-ancestry | Head a04fb74e2ed7857f075c4707b8339ada77c258fd is an ancestor of canonical main b53f757682bff5f488be9c2686c2e8aa8ad4054a. |
| #12 | `fe111799ffab` | closed | integrated-by-ancestry | Head fe111799ffabb2427d6d94feb32a037fa3ed2af8 is an ancestor of canonical main b53f757682bff5f488be9c2686c2e8aa8ad4054a. |
| #13 | `3d0080a9c436` | open | archive-and-supersede-after-69 | TestPendingCurrencyNamePolicyPatchChangesETag retained in internal/api/more_test.go, with only equivalent if/switch formatting. Security browser fixture retains password entries and newer independent-approver rejection/acceptance coverage. Unique CI diagnostic archived in docs/qa/evidence/ci-40103567118-dual-control.md. |
| #14 | `837d51b0d537` | closed | integrated-by-ancestry | Head 837d51b0d5374822b28f1b3a9206f9e2d23bd60c is an ancestor of canonical main b53f757682bff5f488be9c2686c2e8aa8ad4054a. |
| #15 | `33b200e00c08` | closed | integrated-by-ancestry | Head 33b200e00c088c86ffb0d0d8176f1575376a9b49 is an ancestor of canonical main b53f757682bff5f488be9c2686c2e8aa8ad4054a. |
| #16 | `127943ba7aa2` | open | archive-and-supersede-after-69 | docs/audit/VALIDATED-CONSOLIDATION-2026-10-30.md preserves CI run IDs and historical qualification. HANDOFF.md and continuation prompt have stale prompt-only-main and ownership assumptions; current handoff supersedes them. |
| #17 | `7643fb616208` | closed | integrated-by-ancestry | Head 7643fb616208cc318939ded809e6cada66bb7415 is an ancestor of canonical main b53f757682bff5f488be9c2686c2e8aa8ad4054a. |
| #18 | `9c0bc500cb19` | closed | integrated-by-ancestry | Head 9c0bc500cb19f11f18694c76d134f4bfebf896e7 is an ancestor of canonical main b53f757682bff5f488be9c2686c2e8aa8ad4054a. |
| #19 | `34ccb81a54ab` | closed | integrated-by-ancestry | Head 34ccb81a54ab63875d4a5305076437d6b3221049 is an ancestor of canonical main b53f757682bff5f488be9c2686c2e8aa8ad4054a. |
| #20 | `3d9c56196625` | open | archive-and-supersede-after-69 | Entire sole changed document archived in docs/handoffs/archive/2026-11-09-pr20-chatgpt-zero-codex-continuation-spec.md with historical status banner. |
| #21 | `63b033d842a2` | closed | integrated-by-ancestry | Head 63b033d842a2fa40b56ff9d79e69c259db951c83 is an ancestor of canonical main b53f757682bff5f488be9c2686c2e8aa8ad4054a. |
| #22 | `2e729346d629` | closed | integrated-by-ancestry | Head 2e729346d6297075b6ff1f7828d48681d48320b6 is an ancestor of canonical main b53f757682bff5f488be9c2686c2e8aa8ad4054a. |
| #23 | `a376a4ed5924` | closed | integrated-by-ancestry | Head a376a4ed592451d90d2ff7b4d3db1489a2231f96 is an ancestor of canonical main b53f757682bff5f488be9c2686c2e8aa8ad4054a. |
| #24 | `7620d7f51913` | closed | integrated-by-ancestry | Head 7620d7f51913363dd013b80cc682eb670eeff3a5 is an ancestor of canonical main b53f757682bff5f488be9c2686c2e8aa8ad4054a. |
| #25 | `1760551f5b03` | closed | integrated-by-ancestry | Head 1760551f5b03a89ec64d09269d631e72f2e95959 is an ancestor of canonical main b53f757682bff5f488be9c2686c2e8aa8ad4054a. |
| #26 | `2078bf096435` | closed | integrated-by-ancestry | Head 2078bf0964353892e80214b4601211e36818b889 is an ancestor of canonical main b53f757682bff5f488be9c2686c2e8aa8ad4054a. |
| #27 | `b61a8e719030` | closed | integrated-by-ancestry | Head b61a8e719030ff56044d71fee632b5ba824d7698 is an ancestor of canonical main b53f757682bff5f488be9c2686c2e8aa8ad4054a. |
| #28 | `7e8481004801` | closed | integrated-by-ancestry | Head 7e8481004801c01aea5d22f4720e1ead7381c4f7 is an ancestor of canonical main b53f757682bff5f488be9c2686c2e8aa8ad4054a. |
| #29 | `2da1a293ebf2` | open | archive-and-supersede-after-69 | All 15 specification Markdown files archived in docs/archive/specs-historical/pr29/. Conflicting IDs and decisions documented in archive index. tools/check_specs.py expects draft anchors and owner map, incompatible with canonical requirement headings; current traceability generator serves canonical schema. |
| #30 | `1d165989e217` | closed | already-closed-superseded | cmd/circula/acceptance_http_test.go is byte-identical on baseline main, recovered by #65 at 050798c9cf; remaining CI/toolchain changes superseded. |
| #31 | `bffcede8c7dc` | closed | integrated-by-ancestry | Head bffcede8c7dca0d702e1a5cd5b5e8b5d363c7c3b is an ancestor of canonical main b53f757682bff5f488be9c2686c2e8aa8ad4054a. |
| #32 | `f4a051c1c1af` | closed | integrated-by-ancestry | PR head is an ancestor of main; future specifications recovered through #64. This does not merge future module code. |
| #33 | `4a8db9e80509` | closed | integrated-by-ancestry | Head 4a8db9e80509d23d3937c68575c632f4650576aa is an ancestor of canonical main b53f757682bff5f488be9c2686c2e8aa8ad4054a. |
| #34 | `02f9f0a7bb87` | closed | already-closed-superseded | Persistence tests, UX-021 refusal and reversal pagination are present; current TestReversalsOfResultLimit checks all expected IDs/order. Historical qualification documents recovered by #69. |
| #35 | `c033cfa448fb` | open | recover-and-supersede-after-69 | Acceptance and persistence regression files byte-identical to baseline main; current QA-001/007/010/026/028, refusal, export, reversal and log coverage present. Missing startup gate regression recovered by cb79676c and strengthened at call site by 37958e97. Historical CI qualification archived. |
| #36 | `8f742877fb53` | closed | already-closed-superseded | Only unique file is docs/audit/CONSOLIDATION-DELTA-2026-11-14.md. Described implementation exists on main; underlying qualification records recovered by #69. |
| #37 | `3bfb2f78fb0a` | closed | already-closed-obsolete | Single continuation prompt assumes specifications are unfinished; current canonical specification set supersedes that premise. |
| #38 | `d2c1c6090a05` | closed | already-closed-superseded | Competing condensed specifications superseded; unique GOV-032 no-eligible-reviewer rule indexed as an unresolved owner decision aligned to canonical DEC-072. |
| #39 | `92d204dc0273` | closed | closed-pr-obsolete-branch-advanced | Closed PR head 92d204dc is the obsolete continuation prompt. Its source branch now has new head af69702be08aaf14cbe9a705ddf7f08796d5e976; do not infer that later branch work is obsolete. Agent A reviews those later commits separately. |
| #40 | `fe832faced82` | closed | merged-other-branch-superseded | Merged into PR35 branch. Historical Python checker scans all status-table ID families; canonical scripts/traceability understands requirement family definitions, headings, status tables and actual test evidence, with dedicated tests. Historical checker also reports ASM/RSK rows excluded intentionally by canonical requirement scope. |
| #41 | `3d6ed55b2e44` | closed | integrated-by-ancestry | Head 3d6ed55b2e44d31f7350814d87ba44564e8344c0 is an ancestor of canonical main b53f757682bff5f488be9c2686c2e8aa8ad4054a. |
| #42 | `36a80335abf5` | open | archive-and-supersede-after-69 | Only unique file relative to PR35 is docs/security/CHATGPT_FREE_AUDIT_HANDOFF.md. Its entire body archived at docs/audits/CHATGPT_SECURITY_AUDIT_HANDOFF_2026-11-15.md. Current status recorded in docs/security/security-review.md; #69 adds service-level reversal/MFA regressions. A3-F02/DEC-014 and privacy-policy decisions remain open. |
| #43 | `8b4c8ba23c99` | closed | integrated-by-ancestry | Head 8b4c8ba23c99d4418348d76ff8cb38a72e7b958d is an ancestor of canonical main b53f757682bff5f488be9c2686c2e8aa8ad4054a. |
| #44 | `3d6ed55b2e44` | closed | integrated-by-ancestry | Head 3d6ed55b2e44d31f7350814d87ba44564e8344c0 is an ancestor of canonical main b53f757682bff5f488be9c2686c2e8aa8ad4054a. |
| #45 | `e02085a6aa1a` | closed | squash-integrated-current-stronger | Private atomic output-file implementation and util_atomic_test.go retained; snapshot later strengthens manifest authentication and backup names. Later boundary imports replace store direct access. NEW-01 residual remains explicit, not treated as solved by #45. |
| #46 | `8fbc724fa093` | closed | integrated-by-ancestry | Head 8fbc724fa093e91b57b6f0eda4aed4d7b649f388 is an ancestor of canonical main b53f757682bff5f488be9c2686c2e8aa8ad4054a. |
| #47 | `3fa76cdb5901` | closed | integrated-by-ancestry | Head 3fa76cdb590182eaff42fc4044cf3538275eea6b is an ancestor of canonical main b53f757682bff5f488be9c2686c2e8aa8ad4054a. |
| #48 | `4a6ea9ab3dff` | closed | closed-integrated-by-68 | go.mod/go.sum upgraded golang.org/x/text to v0.44.0 in #68. |
| #49 | `f14c75a5cbba` | closed | closed-integrated-by-68 | Fetched PR head f14c75a5 and verified its modernc.org/sqlite v1.62.0 upgrade in current go.mod/go.sum via #68. |
| #50 | `c34a9801853d` | closed | closed-integrated-by-68 | go.mod/go.sum upgraded golang.org/x/crypto to v0.59.0 in #68. |
| #51 | `29e46e5016a0` | closed | squash-integrated-current-stronger | Reversal authorization guard retained; helper regression file byte-identical. Current code additionally validates transaction IDs and uses repository boundary; #69 adds service-entry regression. |
| #52 | `4ffb5c78d6b4` | closed | squash-integrated-current-stronger | principal admin MFA logic and regression file retained; only principal.go difference from PR head is repository import alias. #69 adds service-entry regression. |
| #53 | `26b57163d965` | closed | squash-integrated-current-stronger | Notification erasure now selects member/resource relations in network-bound query; replaces the narrower member-only patch and excludes shared-name substring selection. TestNotificationErasureUsesResourceRelations covers strengthened behavior. |
| #54 | `8abff32c1b4a` | closed | squash-integrated-current-stronger | Departure pagination retained. Existing regression adjusts setup MaxOpenProposalsPerMember to 1000 so 501 proposals can test the intended boundary without newer abuse cap blocking setup. |
| #55 | `64fa4f685780` | closed | integrated-by-ancestry | Head 64fa4f685780b93a349a862e3741df5b4db067cc is an ancestor of canonical main b53f757682bff5f488be9c2686c2e8aa8ad4054a. |
| #56 | `e3c60a62c065` | closed | integrated-by-ancestry | Head e3c60a62c065567ff67fc2e219a724b6fd91ca76 is an ancestor of canonical main b53f757682bff5f488be9c2686c2e8aa8ad4054a. |
| #57 | `dc89212a8bf1` | closed | integrated-by-ancestry | Head dc89212a8bf1e1207e5443ffc719670623de70e1 is an ancestor of canonical main b53f757682bff5f488be9c2686c2e8aa8ad4054a. |
| #58 | `f6ea3213e663` | closed | integrated-by-ancestry | Head f6ea3213e663c7e37d678b018861b6703947e3e0 is an ancestor of canonical main b53f757682bff5f488be9c2686c2e8aa8ad4054a. |
| #59 | `12175ec348d5` | closed | integrated-by-ancestry | Head 12175ec348d5234c27e1c72dc8df96a091cd111c is an ancestor of canonical main b53f757682bff5f488be9c2686c2e8aa8ad4054a. |
| #60 | `96059bb4b8ae` | closed | integrated-by-ancestry | Head 96059bb4b8ae23ad6e4e45428bf69288bec4c182 is an ancestor of canonical main b53f757682bff5f488be9c2686c2e8aa8ad4054a. |
| #61 | `cf5abf5cfb00` | closed | integrated-by-ancestry | Head cf5abf5cfb00638e1772a7e34014a352048d6b35 is an ancestor of canonical main b53f757682bff5f488be9c2686c2e8aa8ad4054a. |
| #62 | `a112c979932b` | closed | integrated-by-ancestry | Head a112c979932bc7b9e209455d3df14dfebc60243a is an ancestor of canonical main b53f757682bff5f488be9c2686c2e8aa8ad4054a. |
| #63 | `d53196cc6907` | closed | closed-integrated-by-68 | All three touched dashboard units/meter files are byte-identical to baseline main. |
| #64 | `3a776e630455` | closed | integrated-by-ancestry | Head 3a776e6304551c21ff372a8ae962606108dde4ff is an ancestor of canonical main b53f757682bff5f488be9c2686c2e8aa8ad4054a. |
| #65 | `050798c9cf94` | closed | integrated-by-ancestry | Head 050798c9cf94598efd0fff15f7a8a74bb3c4f4a1 is an ancestor of canonical main b53f757682bff5f488be9c2686c2e8aa8ad4054a. |
| #66 | `372a8f97ae01` | closed | future-only-merged-into-module | PR head is an ancestor of future/fnet-202-favorites-20261116. All unique changed paths are future/favorites/. Keep isolated from core. |
| #67 | `7a528c611bff` | open | future-only-open-hold | All unique changed paths are future/feed/. PR targets future/fnet-204-feed-20261116; retain draft for that module. Do not merge to core. |
| #68 | `047e92e88a0c` | closed | squash-integrated-tree-equal | All nine touched files are byte-identical to baseline main b53f757, the squash merge. |
| #69 | `163fe1596bd7` | open | reviewed-core-consolidation-awaiting-gates | Source/document review passes. Production diff is startup gate extraction/help only, with targeted regressions and historical archive. Root owns execution/CI and merge decision. |
| #70 | `1ee699f4e41c` | open | active-security-draft-separate-review | Initial head 1ee699f4 is an audit handoff/docs-only change with unreviewed patch references; independent security workstream reviews current state and claims. Do not treat handoff as fixed findings. |

## Limits

This workstream performed source, history, archive and implementation comparisons. Root and other reviewers own runtime tests, CI/review metadata collection, security reproductions and merge execution. No repository or remote content was changed by this workstream.
