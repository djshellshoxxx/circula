# Remaining Core work — 2026-11-20

This table separates remaining implementation from decisions and qualification.
The accepted Core recovery already implements the migration invariant gate,
posting-time credit evidence, report context/privacy projections, conditional
listing revisions, retained-membership recovery guard, bounded ancestry,
versioned closure checksums, platform audit anchors, bounded audit mirror,
metrics, interface repairs and shared password-work admission. Their current
integration boundary is recorded in the
[continuation](CORE_QUALIFICATION_CONTINUATION_20261120.md), with source-finding
status in the [security reconciliation](SECURITY_HANDOFF_RECONCILIATION_20261120.md).
These completed implementations should not be rebuilt from older branches.

Estimates below are preliminary engineering judgment in focused engineer-days
for one engineer familiar with the code. They include the described code,
regression tests and documentation after the necessary decision is recorded.
They exclude owner/reviewer waiting, CI queues, access to reference hardware,
external services, physical-device testing and production rollout. They are
not commitments or permission to select a held policy. Ranges overlap and
should not be summed into a calendar forecast.

| Item | Exact remaining boundary | Estimated engineering effort | Decision or external prerequisite |
|---|---|---:|---|
| OPS-04 — export-reader resource use | Design bounded/streaming Go and Python parsing, injectable small-limit tests, aggregate/parser accounting and clear refusal behavior while preserving supported export formats and existing integrity checks. Current four-GiB per-member admission is not a practical memory ceiling. | 7–15 days | Record supported operator budgets, override behavior and compatibility for existing large exports before introducing new rejecting defaults. |
| OPS-03 — Core Windows private files | Implement and qualify private staging/output ownership, ACL and reparse-point behavior for Core backup/export operations; retain the existing Unix protections and failure recovery. Future messaging storage is a separate implementation, not a Core fix. | 5–10 days | Select supported service identities, ACL inheritance/ownership policy and platform support expectations; obtain native Windows acceptance evidence. |
| LED-R02 — availability disclosure | Apply the chosen actor-aware error-detail policy consistently to service refusals, JSON and HTML, retaining atomic refusal and idempotency. | 2–4 days | Resolve contradictory existing specifications about disclosing exact payer availability. |
| GOV-021 — administrative recovery | Reconcile the preserved recovery implementations with the selected authority, independence, waiting and audit rules; qualify role/session and membership boundaries. | 5–10 days | Choose the recovery authority and governance policy. Three preserved GOV-021 branches are alternatives, not three independent missing features. |
| AUD4-F02 — reviewer reset authority | Reconcile the retained reset-policy branch with current MFA, reviewer independence and credential lifecycle rules; add the selected contract regressions. | 3–6 days | Decide which actors may reset reviewers and how independent authority survives recovery. |
| SEC-026 — abuse signals | Complete the selected data, visibility, retention and export/erasure behavior using the retained implementations and current authorization contracts. | 5–10 days | Decide what signals may be retained, who can see them and their lifecycle. The two preserved implementations must not silently choose that policy. |
| Report subjects and all-conflicted escalation | Implement the selected handling of system-account subjects and cases where every ordinary administrator is conflicted; preserve the currently implemented subject confidentiality and reviewer projections. | 3–6 days | Define the escalation actor, evidence visibility and system-account subject policy. No general bypass privilege is inferred. |
| Competing CLI contract | Reconcile the preserved CLI planning branch with current commands and compatibility requirements, then scope any selected implementation separately. | 2–4 days for contract/document reconciliation; implementation unscoped | Select the intended interface first. The planning branch alone does not establish product requirements for new commands. |
| DEC-006 — reference-hardware and password-work sizing | Run benign reference-hardware measurements and document admission/throughput/memory observations, including historical PHC costs and backup overlap. The three one-iteration smoke benchmarks do not provide these results. | 1–3 days of measurement/documentation after hardware is available | Choose reference hardware and operating headroom. Any new byte-based admission or historical-PHC policy needs a separate compatibility decision. |
| Final combined-source qualification | Complete the current full race run and all final-head provider gates, including native platforms, lint/vulnerability, build/reproducibility, API/load, fuzz and browser checks; repair only observed failures. | No additional implementation estimate while checks are running | Queue/runtime dependent. A passing old source or identical subset is not a final-head pass. Manual screen-reader, physical-failure and production drills remain distinct acceptance evidence. |

## Preserved decision branches

The [148-head inventory](BRANCH_REFRESH_20261120.md) records these seven held
names and their exact source commits. None was deleted or forced into Core:

- `audit/gov021-final-core-refresh-20261119`
- `audit/gov021-fixture-20261119`
- `claude/gov-021-admin-recovery`
- `claude/aud4-f02-reviewer-reset-policy`
- `audit/sec026-independent-20261119`
- `claude/sec-026-abuse-signals`
- `planning/future-cli-specs-20261110`

The [future-module table](FUTURE_MODULE_REMAINING_WORK_20261120.md) separately
records all fifteen independent streams. Their host adapters, selection and
activation are future work; no module is enabled by this consolidation.
