# Circula branch and handoff state — 2026-10-30

This is the source-pinned review input for the consolidated continuation prompt.
The live inventory contained **17 branches and 11 open PRs**. The owned
documentation branch advances when this packet is published; its row below
records the **pre-update input**, not a self-referential claim about the new head.
Resolve the current packet with `git rev-parse HEAD` or PR #16's verified head.
Recheck all live refs and local ownership before implementation.

## Exact reviewed branch inputs

| Branch | Reviewed SHA | State and disposition |
|---|---|---|
| `checkpoint/circula-handoff-20261025` | `3d0080a9c43680f69f81217af3b42374e7160bf8` | Frozen archive, draft #13; six unique commits, C-09 failing regression; preserve separately. |
| `claude/awesome-darwin-y6eaiz` | `827e651fe151f49659cc20aefe85d4af67bdaa97` | Product plus merged interim audit; application starting point. |
| `claude/circula-build` | `fc488445af5f43165a7103f1c1fd12483b39d448` | Older code stack head, draft #4; contained in product tip. |
| `claude/circula-completeness` | `9a86ea69c4196660fac6a4cec652a2072d1bb2ff` | Older code stack head, draft #7; contained in product tip. |
| `claude/circula-finish` | `d350474e49522c34213d90a66904b296b9509f6c` | Older code stack head, draft #8; contained in product tip. |
| `claude/circula-security-features` | `f8278a047392541a077d57c560823471053d1e1d` | Historical security stack head; contained in product tip. |
| `claude/sharp-archimedes-tu76rh` | `fe111799ffabb2427d6d94feb32a037fa3ed2af8` | External audit source; #12 merged, no runtime remediation implied. |
| `claude/sweet-turing-m3ftja` | `01b6e5c7cfc5bd9d660b80aaafe2b35de3912904` | Competing early specs, draft #2; unique commits and conflicts with product. |
| `claude/vibrant-brahmagupta-3uofnj` | `33b200e00c088c86ffb0d0d8176f1575376a9b49` | Original prompt-only source, draft #15; preserved predecessor of #16. |
| `codex/circula-integration-plan-20261030` | `84dc72a3f26be40552629fe9b944e62ce0e97488` | Owned prompt/plan packet, draft #16; this input SHA precedes the current documentation update. |
| `codex/circula-resume-20261024` | `f4bfe6bc3f81c94b859abeae902ae862ea6e143e` | Verified #10 merge receipt; separate ancestry to preserve during consolidation. |
| `codex/circula-support-boundaries-20261028` | `837d51b0d5374822b28f1b3a9206f9e2d23bd60c` | #14 merged; source contained in product tip; original local worktree remains here. |
| `docs/foundational-spec-shells` | `d96f2e75634bff074d26965ae93c2fe00c6f7f24` | Competing early spec shells, non-draft #3; conflict with product spec index. |
| `docs/foundational-specifications` | `22e9202f19f55aa46be4b115edbe84178e9cee0b` | Competing early specs, draft #1; unique content and conflicts; includes deferred scope. |
| `docs/readme-and-wiki` | `db7826495ec2e8bd3f5459afa5f619cdc8fe8def` | Early README/wiki, draft #5; README conflicts with product. |
| `main` | `4b422ad1e8578f3e36c5a38c68eafe81f6ee0aa7` | Prompt-only default branch; no application or application CI. |
| `work/spec-completion` | `d39bfef12d4ac1430815fa49ad799fb1c4bf9964` | Older code stack, draft #9; contained in product tip, old CI failed. |

All four early documentation heads (#1/#2/#3/#5) are independent of the product
line and produce conflicts against the current product tree. Preserve them for
content reconciliation. The older code heads (#4/#7/#8/#9 and the security
branch) are ancestors, not new missing product work. Source deletion statistics
between two trees do not predict a three-way merge's deletions.

At the reviewed inputs, product `827e651...` versus development receipt
`f4bfe6b...` has four product-only commits and one receipt-only merge commit.
Their ordinary merge is clean and retains the same product-plus-audit tree.
Product versus documentation input `84dc72a...` has 166 product-only and three
documentation-only commits. Its conflict simulation reports **only HANDOFF.md**
as an add/add conflict. Preserve both notes and resolve the root handoff when
consolidating on a new owned branch; this review did not perform that merge.
Any moved input requires a fresh comparison.

## Handoff inventory and reading order

1. **Current packet:** `HANDOFF.md`, `docs/INTEGRATION-PLAN.md` and
   `prompts/continue-circula.md` on the owned #16 branch. These contain the
   verified #10/#12 merge outcomes and supersede older next-action descriptions.
   They are not yet present on the product branch.
2. **Product historical notes:** `HANDOFF.md` at `827e651...` is identical to
   that on `f4bfe6b...` (173 lines; blob
   `dddfee155569f736b02e5de743bd8e2d99ef5118`). It records earlier source
   `5000a5e...` and the later #14 integration receipt, but still describes #10/
   #12 as pending/in progress. Preserve it as history; use current PR metadata
   and this packet for completed merge state.
3. **Detailed product restart:** `docs/audit/RESUME-2026-10-24.md` on the
   product, development receipt and support branch shares blob
   `c01e847e4d4367f786b293f75f06ace34e2dafbd`. It contains layered historical
   specs, local/remote results and later continuation receipts; respect their
   recorded source and executor rather than flattening all results into one pass.
4. **Archive:** #13's `HANDOFF.md` (120 lines) begins with an Oct 28 publication/
   CI receipt, followed by historical local-only/denial text. Its detailed
   restart (287 lines) and C-09 packet remain source-pinned. The exact publication
   approval was exercised and verified; older publication wording does not undo
   that outcome. Separate network/access denials remain unresolved.
5. **External audit:** `docs/audit/2026-10-28/README.md`,
   `interim-verification-circula.md/.json`, `poc/**` and `wip-fixes/README.md`
   from `fe11179...` are already merged in the product. Its older restart file
   (248 lines) predates GOV-027 and the latest merge receipts. The first-wave
   README and interim verification table represent different phases, not a
   completed final audit. Retain them with their provenance.
6. **Original prompt #15:** its source `33b200e...` is preserved as an ancestor
   of #16. Earlier stale CI/axe-gap and idle-ownership assumptions are corrected
   in this packet. Use the consolidated prompt for new sessions.

The local recovery ref `codex/circula-resume-20261024@91851e0560c4c7a656968aa56a0f14ecec511d99`
is intentionally different from the current remote development receipt. Its
three formerly unpushed commits are all reachable through the remote archive;
preserve this local ref and do not reset it to remote. Historical missing
`c134253`/`ee78a7f` remain unrecovered.

## Merged work and exact verification

- #14 is merged at `837d51b0d5374822b28f1b3a9206f9e2d23bd60c`.
- #10 is merged into `codex/circula-resume-20261024` at
  `f4bfe6bc3f81c94b859abeae902ae862ea6e143e`.
  [Post-merge CI 41032607314](https://github.com/djshellshoxxx/circula/actions/runs/41032607314)
  completed success, 9/9 jobs passed.
- #12 is merged into `claude/awesome-darwin-y6eaiz` at
  `827e651fe151f49659cc20aefe85d4af67bdaa97`.
  [Post-merge CI 41032669029](https://github.com/djshellshoxxx/circula/actions/runs/41032669029)
  completed success, 9/9 jobs passed.
- #13 stays draft: source `59f7af9bcb39162c94756560178f71e439aef474`
  has genuine C-09 unchanged-ETag and 409-versus-412 failures, plus QF1003 lint;
  its later documentation head is `3d0080a...`. It conflicts with the product
  in the root handoff and dual-control E2E fixture. Preserve, do not merge wholesale.

No new full application test, download or CI dispatch ran for this documentation
refresh. Prior uncached-module downloads (notably SQLite through the official
module proxy) and artifact-byte proxy retrieval were denied with HTTP 403 and
remain unresolved in this executor. No retry or alternate route was attempted.
Other sessions' local runs and the green remote CI above are distinct evidence.

## Development queue and ownership

The following are **candidate next slices**, not a claim that ownership has
transferred or every finding is finally established:

| Candidate | Existing packet / condition |
|---|---|
| Tracked root build artifact | Product tracks `circula`, 21,703,878 bytes. Reserve hygiene work, remove from index in a normal commit and ignore future output; preserve history. |
| A1-01 | Interim settle-to-zero network-state gating candidate; reproduce on current source after ownership reconciliation. |
| A1-02 / A1-11 | Scheduled decimal-scale reinterpretation / creation-time immutability candidate; follow accepted specs and preserve failing-before evidence. |
| A1-03 | Re-review versus another live dispute candidate; coordinate with the retained dispute fixer/WIP. |
| A3-F01 | Encrypted-backup pre-migration integrity-check candidate. |
| A3-F02 | Break-glass lasting-change coverage candidate; extra-lens skeptic result is refuted while verifier/impact report Medium. Resolve evidence, do not invent consensus. |
| A3-F16 | Renewable/shared second-factor throttle candidate; additional lens verification is incomplete. |
| C-09 / A2-07 | Preserve archived regression and coordinate existing policy-ETag ownership; no duplicate independent fix. |
| Lower/info findings | Read only the selected subsystem's interim evidence and current specs; explicitly reserve work first. |

WIP patches are saved text, not applied fixes; the disputes patch is incomplete.
A1/A3 PoCs assert desired behavior (fail before fix); A2 PoCs assert presence of
the defect (pass before fix). Adjust regression direction explicitly rather than
deleting or weakening assertions. The audit has 55 batch-verifier entries and
incomplete extra-lens/coverage/consolidation work; it is AI-assisted, not a
professional independent security review.

Already present: FR-054 historical balances, independent-founder/grant-chain and
24-hour approval behavior, GOV-027 support guidance, the request-ID assertion
correction, FR-067 admin counts, UX-024 time zones and automated axe/keyboard CI.
Historical completeness/gap tables must be revalidated before assigning missing
features.

## Boundaries and milestone completion

The next agent may consolidate inputs on a fresh owned work branch, implement
accepted unowned slices and publish source/handoff draft checkpoints under the
existing development request. Existing branch ownership, required review,
tests and specific merge/approval boundaries remain in force. A merged or idle
audit PR is not an ownership handover. Keep a single integration owner and
disjoint worker scopes; avoid competing teams and repeated broad audits.

Product integration to main, old-PR closure and release/pilot remain proposed,
not executed. Owner gates include licence, DEC-072/074, UX-047 locale scope,
confirmation of built defaults, jurisdictions, unconfirmed release thresholds,
manual screen-reader evidence and professional independent security review.
Preserve monetary/ledger/approval invariants. Respect access denials and any
uncertain remote action; never claim a skipped or unavailable check passed.

Each milestone records exact inputs/HEAD, changed requirements, tests and actual
outcomes, local dirty/untracked/unpushed state, ownership/blockers and the next
bounded task; commit source plus handoff and verify the normal remote push.
No paid model call or extra worker was used for this refresh; remaining shared
budget is not visible. Preserve a checkpointing reserve.
