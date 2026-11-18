# Historical specification drafts (archive index)

Archived 2026-11-18 during the PR consolidation recorded in
[PR-CONSOLIDATION-2026-11-17.md](../../plans/PR-CONSOLIDATION-2026-11-17.md).

`docs/specs/` is the canonical specification. Code, `docs/TRACEABILITY.md` and the audits cite its
IDs. The drafts below were written in parallel from `main@4b422ad` (a prompt-only `main`) before or
alongside it. They reuse the same ID prefixes (FR, DOM, SEC, GOV, UX, QA, DAT, OPS) with **different
meanings**, so none of their IDs may be cited or merged. For example, draft #29 `DOM-012` is audit
logging, but canonical `DOM-012` is capacity reservation.

Nothing here changes product policy. Items marked *owner decision* are open questions, not
accepted requirements.

## Sources

Draft PR branches are preserved on the remote. Each PR head also stays readable through
`refs/pull/<N>/head`. To read a file: `git show <sha>:<path>`.

| PR | Branch | Head | Content | Disposition |
|---|---|---|---|---|
| [#1](https://github.com/djshellshoxxx/circula/pull/1) | `docs/foundational-specifications` | `22e9202f` | 13 short spec drafts, 3 brainstorm notes, early wiki | Brainstorms archived in [`pr1/`](pr1/); specs superseded |
| [#2](https://github.com/djshellshoxxx/circula/pull/2) | `claude/sweet-turing-m3ftja` | `01b6e5c7` | 20 spec drafts, research notes, plans, ADR template, Python scripts | Docs 15, 16, 17 and 19 archived in [`pr2/`](pr2/). Two research notes moved to `docs/research/`. Rest superseded |
| [#3](https://github.com/djshellshoxxx/circula/pull/3) | `docs/foundational-spec-shells` | `d96f2e75` | Empty spec shells | Superseded; nothing unique |
| [#5](https://github.com/djshellshoxxx/circula/pull/5) | `docs/readme-and-wiki` | `db782649` | README/wiki stub saying "no code yet" | Superseded; obsolete |
| [#29](https://github.com/djshellshoxxx/circula/pull/29) | `claude/relaxed-hopper-8q4vhq` | `2da1a293` | Full rewrite of 14 specs (about 914 anchored requirements) and `tools/check_specs.py` | Index only (below). Most of its themes are already canonical (review estimate) |
| [#38](https://github.com/djshellshoxxx/circula/pull/38) | `claude/lucid-archimedes-upisa8` | `d2c1c609` | 13 condensed specs plus index | Superseded; one unique item listed below |
| none | `planning/future-cli-specs-20261110` | `8e2372c0` | Future CLI specification | Index only; conflicts listed below |

Moved to `docs/research/` (no ID conflict, no canonical equivalent):
[PRIOR-ART-MUTUAL-CREDIT.md](../../research/PRIOR-ART-MUTUAL-CREDIT.md) (LETS, Cyclos, Sardex, WIR) and
[REGULATORY-PRIVACY-QUESTIONS.md](../../research/REGULATORY-PRIVACY-QUESTIONS.md) (issues list for counsel,
not legal advice). Both keep their original [UNVERIFIED]/[U] markers.

## Unique material worth a later decision

| Source | Item | Notes |
|---|---|---|
| #2 `docs/specs/15-lifecycle-state-machines.md` §9.2 ([`pr2/`](pr2/15-lifecycle-state-machines.md)) | Race matrix R1–R18 and a model-based test rule | Candidate QA backlog. Main has state-machine and partial concurrency tests |
| #2 `docs/specs/16-network-policy-catalog.md` §6.1 ([`pr2/`](pr2/16-network-policy-catalog.md)) | Cross-parameter policy consistency rules R1–R13 | `DEC-050` bounds single values only. Candidate validation backlog |
| #2 `docs/specs/17-notifications-and-communications.md` §6.3, §8.2–8.3, §9 ([`pr2/`](pr2/17-notifications-and-communications.md)) | Digests and bounce handling | Candidate input for future module FUX-104 |
| #2 `docs/specs/19-legal-and-regulatory-open-questions.md` ([`pr2/`](pr2/19-legal-and-regulatory-open-questions.md)) | Legal gates H/S/P ordering | Owner decision |
| #2 `docs/specs/18-data-export-and-portability.md` §11 | Import and migration from other systems | Future candidate; export itself is canonical (FR-090..094) |
| #1 `docs/brainstorming/*` ([`pr1/`](pr1/)) | Mutual-aid mode, procurement, skills commons, governance templates, import for migration | Not yet in `docs/future-features/` |
| #1 `docs/brainstorming/business-model-options.md` | Value-capture principles: never gate export or exit on payment, never sell histories | Owner decision. Transaction-fee options conflict with scope |
| #29 `docs/specs/03-*` DOM-029 | Confirmation step for large amounts | Owner decision (policy option) |
| #29 `docs/specs/03-*` DOM-051 | Positive balance limit | Owner decision |
| #29 `docs/specs/09-*` SEC-066 | Member-to-member blocking | Owner decision |
| #29 `docs/specs/09-*` SEC-015 | Email-change verification | Owner decision |
| #29 `docs/specs/04-*` FR-083 | Member view of own sign-in activity | Owner decision |
| #29 `docs/specs/09-*` SEC-055 | Administrator mass-action alert | Extends canonical SEC-026 (open) |
| #29 `docs/specs/10-*` GOV-120 | Written-rules checklist; handover and dormant-administrator guidance | Relates to canonical GOV-021 (open) |
| #29 `docs/specs/06-*` DATA-083 | Erasure register | Owner decision |
| #29 `docs/specs/13-*` DEC-014 | Organizations as members | Owner decision. Canonical DEC-014 is a different decision (break-glass) |
| #29 `docs/specs/13-*` DEC-052 | Network posting halt | Compare with canonical DEC-076 |
| #38 GOV-032 | No eligible reviewer: the dispute stays open and administrators are notified | Fold into canonical DEC-072 (owner) |
| `planning/future-cli-specs-20261110` | `doctor` command, `--json` output, API-client report/export | Owner decision |

## Conflicts with canonical decisions (do not adopt without an owner decision)

| Draft | Draft rule | Canonical rule |
|---|---|---|
| #1 DEC-002, #29 DOM-028/DEC-004 | No capacity reservation | DEC-035, DOM-012 reserve capacity |
| #29 DOM-032 | A reversal may breach the credit limit | DEC-039, FR-074 |
| #29 DOM-033 | Only one reversal per trade | DEC-040 cumulative cap |
| #29 | 0–2 decimal places | DEC-050 allows 0–4 |
| #29 FR-027 | Public network directory | FR-020, FR-025: members only |
| #2 doc 14 | Banned price/money terms in copy | DEC-054 allows optional prices |
| #2 doc 16 | Configurable administrator approval quorum | SEC-011 dual control (scope is an open owner decision) |
| future CLI spec | Exit code 3 = operational failure, 4 = no results, 5 = cancelled | `cmd/circula/main.go`: 3 = ledger verification fault |
| future CLI spec | `transfer create`, `account suspend`, `limits set` from the CLI | DOM-011 consent before debit; PER-004 and ARC-008 operator limits |

## Out of scope for core

Tokenization, cryptocurrency redemption, clearing circles, reserve or guarantee circles, collateral,
demurrage, portable credentials and cross-network settlement (from #1 and #2) stay out of core scope.
Federation is handled only as isolated future module FNET-209.

`tools/check_specs.py` from #29 was not imported. Its ID rules expect `<a id>` anchors and an owner map;
canonical specs use `### FR-001 ·` headings and status tables. Its only transferable check (relative
links and heading anchors) found no broken relative links or anchors in `main` docs in the 2026-11-18 review (not added to CI).
