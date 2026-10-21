# Master TODO

| Field | Value |
|---|---|
| Status | Draft (living checklist) |
| Owner role | Product owner (keeper); each item names its own owner role |
| Last updated | 2026-10-21 |
| Scope | Actionable, small items to get from the 2026-10-21 specification baseline to a delivered MVP: owner decisions, counsel engagement, spec reconciliation and verification, deferred-spec authoring, repository bootstrap, implementation epics, ops/hosting prerequisites |
| Depends on | [13](../specs/13-assumptions-risks-decisions-roadmap.md) (registers), [19](../specs/19-legal-and-regulatory-open-questions.md), [12](../specs/12-quality-strategy-and-test-plan.md), [Roadmap](roadmap.md), [MVP implementation plan](mvp-implementation-plan.md), [Traceability](traceability.md) |

**How to use.** Tick a box only when the evidence exists (ADR merged, register row updated, written advice recorded). Item IDs (`A01`, `C07`, ...) are local to this file and never cited from specs. **Pri**: P0 blocks the next gate in the [roadmap](roadmap.md#6-gono-go-gates); P1 needed within the milestone; P2 opportunistic. **Owner** is a role from [13 section 2](../specs/13-assumptions-risks-decisions-roadmap.md#2-definitions-used-here), never a person. **Dep** lists prerequisites (item IDs, `DEC-`, `LEG-Q-`). Decisions are Accepted only through the register and ADR process (13 section 6); the defaults below are the register's **Proposed defaults**, offered for quick approval.

## A) Owner decisions needed now

Highest-impact decisions with Needed-by M0 (plus DEC-002, which gates the walking skeleton). The Product owner (or the named owner role) can approve each by replying "approve default" or by choosing an alternative; the keeper then updates the register row in the same change (13 section 6.3). Items marked **(counsel)** also need written advice before Accepted.

| # | DEC | Question | Recommended default (as in register) | Why it is high-impact | Owner | Pri | Approve? |
|---|---|---|---|---|---|---|---|
| A01 | DEC-015 | Open-source licence and contribution model (repo has none, ASM-032) | Strong copyleft closing the network-service gap; confirm with owner. **(counsel)** | Blocks first public push and external contributions (RSK-045); Gate S wording (LEG-G-20) | Product owner | P0 | [ ] |
| A02 | DEC-002 | Reference stack: technology neutrality and when to choose | Neutral specs; reference stack chosen by ADR at M1 entry, via the selection process in the [plan](mvp-implementation-plan.md#6-technology-selection-process) | Gates the walking skeleton; RSK-021 | Product architect | P0 | [ ] |
| A03 | DEC-001 | Deployment model | One codebase for operator-hosted multi-tenant and self-hosted; no hosted subscriptions in MVP | Shapes topology, ops, legal (Gate H vs S) | Product owner | P0 | [ ] |
| A04 | DEC-016 | Design envelope (Members per Network, Networks per deployment, trades/day) | Committed upper design limit 2,000 Members per Network; pilot envelope smaller (ASM-001) | Sizing of ledger serialisation, tests (QA-GATE-13) | Product architect | P0 | [ ] |
| A05 | DEC-019 | First audience | Both time-banks and local trading circles; `unit_scale` default 2 with a one-click "hours" preset | Presets, copy, pilot recruitment (ASM-031) | Product owner | P0 | [ ] |
| A06 | DEC-005 | Funds holds/reservations on pending Trades | No holds in MVP; limits enforced at posting; pending Trades informational (D-05) | Core ledger semantics; FSP-09 extension | Product owner | P0 | [ ] |
| A07 | DEC-006 | Counterparty acceptance | Acceptance always required; standing orders deferred (FSP-08) | INV-012 and consent model; LEG-Q-08 | Product owner | P0 | [ ] |
| A08 | DEC-007 | Identity model | One login, many Memberships; email plus passkey/TOTP (D-07) | Auth architecture (SEC-001 to SEC-017) | Security lead | P0 | [ ] |
| A09 | DEC-008 | Data controller split | Network as controller, Operator as processor when hosted. **(counsel, LEG-Q-24)** | Architecture and contracts; RSK-043 | Counsel (via Product owner) | P0 | [ ] |
| A10 | DEC-009 | Erasure versus append-only ledger | Anonymise on closure; ledger keeps opaque account IDs only (D-09). **(counsel, LEG-Q-27)** | Data-model freeze (roadmap gate G1) | Data protection lead | P0 | [ ] |
| A11 | DEC-010 | Network participation default | Invite or admin-approval default; open sign-up optional per Network | Abuse resistance, onboarding journeys (RSK-009) | Product owner | P0 | [ ] |
| A12 | DEC-011 | Policy change notice | 14-day default notice, safety fixes excepted (D-11) | Policy engine, notifications, disputes | Product owner | P0 | [ ] |
| A13 | DEC-012 | Disputes and balances | Disputes do not freeze balances; remedy is Reversal/Adjustment (D-12) | Dispute workflow and legal posture (LEG-Q-22) | Product owner | P0 | [ ] |
| A14 | DEC-014 | Accessibility target | WCAG 2.2 Level AA (D-14) | UX effort and QA-GATE-07 | UX lead | P0 | [ ] |
| A15 | DEC-028 | Who executes the ledger remedy of an upheld Dispute; limit-breach handling | Administrator-executed under quorum (two-step: reviewer decides, admins execute); breach handled as Adjustment | Dispute and Adjustment design (EP-61, EP-62) | Product owner | P0 | [ ] |
| A16 | DEC-062 | Default `balance_visibility` | `admins` (authority: doc 16) | Privacy default; UX of balance panel | Product owner | P0 | [ ] |
| A17 | DEC-074 | Minimum age and minors | 18; defer youth model. **(counsel, LEG-Q-30)** | Sign-up flow, terms (RSK-044) | Counsel (via Product owner) | P0 | [ ] |
| A18 | DEC-101 | Retention schedule per entity | As proposed in doc 13. **(counsel, LEG-Q-33)** | Retention jobs, backups, erasure (EP-72) | Data protection lead | P0 | [ ] |
| A19 | DEC-120 | RPO/RTO per tier | Hosted 5 min / 4 h; self-host recommended 15 min / 8 h; minimum 24 h / 24 h | Backup design, datastore choice, restore drill | Operations lead | P0 | [ ] |
| A20 | DEC-038 | Member-run voting | Deferred (P2, FSP-16) | Scope; avoids governance rework | Product owner | P1 | [ ] |

**Batch-approve candidates** (also Needed-by M0, low impact; approve together): DEC-052 (keep local statuses), DEC-132 (no SMS/push channel in MVP). Decisions Needed-by M1 that follow the skeleton: DEC-003 (identifier format), DEC-004 (ledger model), DEC-013, DEC-022, DEC-040, DEC-049, DEC-058, DEC-064 to DEC-069, DEC-077, DEC-080, DEC-082 to DEC-092, DEC-107, DEC-109, DEC-111, DEC-112, DEC-147, DEC-148, DEC-151.

- [ ] A21 Product owner: confirm the team-size profile used for planning (Lean/Core/Full, [roadmap section 2](roadmap.md#2-team-size-assumptions-estimate)) and whether estimates may be shared externally. Pri P1. Dep: none.
- [ ] A22 Product owner: name the keeper of the decision tracker and agree the weekly review (working agreement 13). Pri P1.
- [ ] A23 Product owner: approve the launch-jurisdiction shortlist for pilot and Gate H (19 section 8 open item). Pri P0. Dep: B03.

## B) Counsel engagement

Counsel is engaged through the Product owner; nothing here is a legal conclusion. Order of engagement follows [19 section 4](../specs/19-legal-and-regulatory-open-questions.md#4-prioritisation); the preparation checklist is [19 section 7](../specs/19-legal-and-regulatory-open-questions.md#7-counsel-engagement-checklist).

- [ ] B01 Product owner: select and engage counsel for the pilot jurisdiction(s) (ASM-023, RSK-054); agree response dates per question. Pri P0. Dep: A23.
- [ ] B02 Product owner + Product architect: prepare the counsel pack: product behaviours (INV-014, LEG-G-01 to LEG-G-03), deployment facts (D-01), data-flow diagram, retention skeleton, this register with the Status column (19 section 7). Pri P0. Dep: none.
- [ ] B03 Product owner: decide launch jurisdictions for Gate H (pilot first); every other jurisdiction is geofenced (LEG-G-16). Pri P0.
- [ ] B04 Counsel: closed-loop posture, first sitting: LEG-Q-06, LEG-Q-07, LEG-Q-08, LEG-Q-09, LEG-Q-16, LEG-Q-18, LEG-Q-23 (19 section 4.1 item 1). Pri P0 (needed before G4). Dep: B02.
- [ ] B05 Counsel: data-protection foundation that precedes data-model freeze: LEG-Q-24, LEG-Q-25, LEG-Q-27, LEG-Q-28, LEG-Q-30, LEG-Q-31, LEG-Q-32, LEG-Q-33. Pri P0 (needed before G1). Dep: B02.
- [ ] B06 Counsel: jurisdiction-specific money regimes for the pilot jurisdictions only: LEG-Q-10 to LEG-Q-15 as applicable. Pri P0 (before G4). Dep: B03.
- [ ] B07 Counsel: contract and governance: LEG-Q-20 (terms), LEG-Q-22 (reversal/adjustment basis). Pri P1 (before G4). Dep: B04.
- [ ] B08 Counsel: screening LEG-Q-35 for hosted onboarding. Pri P1. Dep: B03.
- [ ] B09 Counsel: disclosure and reporting items that may ship with recorded risk acceptance: LEG-Q-01 to LEG-Q-05, LEG-Q-21, LEG-Q-26, LEG-Q-29, LEG-Q-34. Pri P2 (before G5). Dep: B04.
- [ ] B10 Counsel: Gate S set before self-host distribution: LEG-Q-24, 27, 28, 33, 30, 18, 19, 02, 34 and licence/warranty wording (LEG-G-20). Decide whether licence becomes `LEG-Q-36`. Pri P1 (before first self-host release). Dep: A01.
- [ ] B11 Counsel: copy review of balance wording and unit framing (DEC-114, DEC-117, DEC-061; RSK-013, RSK-048). Pri P1 (before M4 usability study). Dep: B04.
- [ ] B12 Counsel: review of erasure design (DEC-009, DEC-106) and the retention schedule (DEC-101). Pri P0 (before G1). Dep: B05.
- [ ] B13 Product owner: record each counsel answer as a `DEC-` entry and update the Status column in 19 (19 section 7 last paragraph); track as "answered / assumptions / jurisdictions / invalidating changes / date". Pri P0. Dep: B04 onward.
- [ ] B14 Product owner: assign `ASM-`/`DEC-` numbers to the open items in 19 section 8 (launch jurisdictions, licence as LEG-Q-36, DPIA/DPA templates, retention values). Pri P1.
- [ ] B15 Counsel + Data protection lead: DPA and privacy-notice outlines; sub-processor list skeleton (LEG-Q-24, LEG-Q-31). Pri P1 (before G4). Dep: B05.
- [ ] B16 Counsel + Product owner: terms of service and Network charter templates before hosted launch (RSK-047). Pri P1 (before G4). Dep: B07.

## C) Spec reconciliation and verification backlog

### C1) Conflicts to reconcile at M0 (doc 13 [section 5.5](../specs/13-assumptions-risks-decisions-roadmap.md#55-conflicts-to-reconcile-at-m0))

Each resolution is made by editing the owning spec in a change that also updates the register row; the doc owners are asked, not edited from this plan.

- [ ] C01 Security lead: step-up recency, one platform constant owned by doc 09 (SEC-009); align API (15 min) and FR/PRT (10 min) — DEC-067. Pri P0. Dep: A08.
- [ ] C02 Engineering lead: idempotency retention: posting keys 72 h, non-posting 24 h, exports 7 days; doc 07 cites doc 06 — DEC-090. Pri P0 (before G1).
- [ ] C03 Product architect: API deprecation window: 12 months for MAJOR API removal, 6 months for other feature removals; align doc 11 — DEC-108. Pri P1.
- [ ] C04 Operations lead: post-closure export window: policy minimum 30 days, hosted terms recommend 90 — DEC-119. Pri P1.
- [ ] C05 Product owner: Member invitations in MVP? If yes keep quota 3 (doc 04 vs doc 10) — DEC-031. Pri P1.
- [ ] C06 Product architect: dispatch latency: 60 s p95 target, 10-minute alert floor — DEC-123. Pri P1.
- [ ] C07 Product owner: invitation lifetime: 14 days; fix the option lists in doc 15 (7/14/30) and doc 04 (3/14/30) — DEC-030. Pri P2.
- [ ] C08 Security lead: patch SLAs: single table in doc 09; doc 11 cites it — DEC-075. Pri P1.

### C2) Defects and gaps found while preparing the plans (verify, then fix in the owning doc)

- [ ] C10 Product architect: doc 13 front matter and section 1 say milestones "M0 to M6" but section 7.3 defines M0 to M9; align wording. Pri P2.
- [ ] C11 Product owner: doc 13 section 8 says effort/date estimates are intentionally omitted; the [roadmap](roadmap.md#1-how-to-read-this-document) adds labelled estimates. Decide whether to keep them in plans only (recommended) or reference from doc 13. Pri P2.
- [ ] C12 QA lead: DEC-151 says the traceability file "does not yet exist"; a first generator now exists (`scripts/gen-traceability.py`). Decide: adopt as the QA-101 tool (needs covers-tag parsing, Withdrawn-ID check, change-vs-test flag per QA-101 and QA-106) or replace; update DEC-151. Pri P1.
- [ ] C13 QA lead: add QA-E2E scenarios or explicit lower-layer evidence for areas with no scenario citation. Largest gaps in [traceability section 2.2](traceability.md#22-gap-counts-by-fr-area) at generation time: FR-ADM, FR-NET and FR-LST (admin controls/appeals/moderation, Network setup and operating status, listings and directory). Pri P0 (before M2 exit). Dep: C14.
- [ ] C14 QA lead: triage every P0/P1 FR listed in [traceability section 2.1](traceability.md#21-p0p1-fr-with-no-doc-12-citation): assign layer (unit/integration/E2E/inspection) and add the citation in doc 12 (QA-102). Pri P0.
- [ ] C15 QA lead: INV-006 (max_balance) and INV-014 (no money-like claims) have no QA-E2E scenario; confirm lower-layer coverage is sufficient or add scenarios (12 section 5 owns the invariant table). Pri P1.
- [ ] C16 QA lead: no end-to-end scenario for erasure Phases 1 and 2, invitation lifecycle, moderation/appeal, or Network closure; decide whether to add (see plan EP-32, EP-64, EP-72, EP-63). Pri P1.
- [ ] C17 Product architect: re-run `python3 scripts/check-specs.py` and the generator after every spec batch; resolve any DANGLING or BROKEN LINK reports with the owning doc author. Pri P0 (recurring).
- [ ] C18 Product architect: ranges such as "FR-TRD-001 to 016" are expanded by the generator only for FR; consider a convention note in doc 00 (cite ranges with full IDs) so tooling is exact. Pri P2.

### C3) Unverified external claims to verify (tags from `docs/notes`)

Legend from the [notes README](../notes/README.md): `[V]` verified, `[K]` background knowledge, `[U]` unverified. Each item: open the primary source, record URL + access date in the owning spec, and change the tag. Statutory content is verified by counsel, not by engineers; engineers verify technical standards.

Regulatory and privacy note ([research-regulatory-privacy-questions](../notes/research-regulatory-privacy-questions.md); 19 section 9):

- [ ] C20 Counsel: all statutory section numbers and thresholds marked `[U]` (IRC 6045(c)(3), Treas. Reg. 1.6045-1(e), PSD2 Arts. 3(k) and 37(2), GDPR articles, Directive 2006/112/EC Arts. 2 and 73, EAA scope/dates, CCD2 application date). Pri P1 (before G4).
- [ ] C21 Counsel: HMRC position on LETS (LEG-Q-04); only forum commentary found. Pri P2.
- [ ] C22 Counsel: PSD3/PSR status (LEG-Q-11). Pri P2 (monitor; confirm at launch).
- [ ] C23 Counsel: Canada, Australia, New Zealand regimes (LEG-Q-13, 14, 15) entirely `[U]`; research only if those jurisdictions are in scope, else geofence. Pri P1 (conditional on B03).
- [ ] C24 Counsel: EDPB Guidelines 02/2025 on blockchain v2.0 (details from secondary summaries; PDF not opened). Pri P2 (relevant only if DLT is proposed, LEG-Q-19).
- [ ] C25 Counsel: EU-US Data Privacy Framework status, EAA scope, UK crypto-asset regime, retirement of the EU ODR platform in 2025. Pri P2.
- [ ] C26 Product architect: re-fetch primary sources that failed in the research sandbox (irs.gov, fincen.gov, eba.europa.eu) from a network with access; record access dates. Pri P1.
- [ ] C27 Counsel: remaining sources in the note's reference list marked "not opened" (19 section 9 item 8). Pri P2.

Prior-art note ([research-prior-art-mutual-credit](../notes/research-prior-art-mutual-credit.md); ASM-034):

- [ ] C30 Product architect: re-verify prior-art claims marked UNVERIFIED before any is used as a design input: licensing statements (Cyclos, Community Forge/Credit Commons, TimeOverflow AGPL vs mirror, GPL/STROhalm), counts for CES and Community Forge, Sardex model and licensing, per-edge limits in trust-line systems. Doc 13 M0 exit criterion 6 needs "verified or marked non-inputs". Pri P0.
- [ ] C31 Product architect: confirm no requirement relies on the unfound facts: typical default credit-limit number, dispute mechanics of Cyclos/Sardex/CES, fraud/collusion evidence ("No source retrieved"). Mark as non-inputs in the register (ASM-034). Pri P0.
- [ ] C32 Product owner: decide whether to commission 3 to 5 interviews with existing time-bank/LETS administrators to replace desk research for ASM-002, ASM-025 (validation in the pilot at M5 as planned). Pri P2.

Technical standards note ([research-technical-standards](../notes/research-technical-standards.md)):

- [ ] C40 Engineering lead: confirm `[K]` claims used by specs before they become test oracles: RFC 9562 (UUIDv7), RFC 9457 (problem details), RFC 9110 ETag/If-Match semantics, RFC 3339 timestamps, keyset pagination guidance. Pri P1 (before G1).
- [ ] C41 Security lead: current maturity of WebAuthn Level 3 (marked `[U]`) and the state of OAuth 2.1 drafts and the Idempotency-Key header draft (not RFCs; spec must say "draft" and not claim conformance). Pri P1.
- [ ] C42 Security lead: OWASP ASVS version/level mapping and API Top 10 (2023) references against SEC-115; session, recovery and CSRF guidance (cheat sheets) current. Pri P1.
- [ ] C43 UX lead: WCAG 2.2 AA criteria list used in doc 08 section 10 against the W3C text; EAA applicability dates. Pri P1.
- [ ] C44 Engineering lead: ledger-design claims attributed to external systems (TigerBeetle, Modern Treasury, Square, Stripe) are context only; do not cite as requirements. Pri P2.
- [ ] C45 Operations lead: OpenTelemetry signal maturity, 12-factor, SemVer, expand/contract guidance used by doc 11. Pri P2.

### C4) First-class review of the documents (Draft to Review)

Per 00 section 3 status meanings: **Draft** to **Review** means cross-checked; **Baseline** needs owner sign-off. For each document: cross-check against 00 canon, check IDs and links (`scripts/check-specs.py`), check every `[DECISION NEEDED]` has a `DEC-` alias in doc 13, then record the status change. Reviewer is a role other than the document owner.

- [ ] C50 Review 00 conventions (canon 5.8 confirmed, ASM-021). Reviewer: Product owner. Pri P0.
- [ ] C51 Review 01 vision and scope (VIS ids, success measures, Out items, VIS-039). Reviewer: Product architect. Pri P0.
- [ ] C52 Review 02 personas and journeys (JRN-001 to JRN-011 coverage, permission matrix). Reviewer: UX lead. Pri P0.
- [ ] C53 Review 03 domain model (INV list and DOM rules vs canon). Reviewer: Engineering lead. Pri P0.
- [ ] C54 Review 04 functional requirements (priorities, Given/When/Then completeness, FR-ADM-027 reference consistency). Reviewer: QA lead. Pri P0.
- [ ] C55 Review 05 architecture (ARC-035 gate, quality targets). Reviewer: Security lead. Pri P0.
- [ ] C56 Review 06 ledger and audit data (DAT constants vs DEC theme H). Reviewer: Engineering lead. Pri P0.
- [ ] C57 Review 07 API contract (vs 06 idempotency, vs 09 step-up). Reviewer: Engineering lead. Pri P0.
- [ ] C58 Review 08 UX and accessibility (banned terms, DEC-114, DEC-117). Reviewer: Product owner. Pri P0.
- [ ] C59 Review 09 security (SEC gates; patch SLA table owner). Reviewer: Operations lead. Pri P0.
- [ ] C60 Review 10 governance (defaults vs doc 16; GOV vs DEC theme C). Reviewer: Product owner. Pri P0.
- [ ] C61 Review 11 operations (RPO/RTO, deprecation window, dispatch floor). Reviewer: Security lead. Pri P0.
- [ ] C62 Review 12 quality strategy (QA-OD decisions registered; thresholds; QA-101 tool). Reviewer: Engineering lead. Pri P0.
- [ ] C63 Review 13 registers (counts, aliases, needed-by, milestone wording). Reviewer: Product architect. Pri P0.
- [ ] C64 Review 14 glossary (banned terms vs UI strings). Reviewer: UX lead. Pri P1.
- [ ] C65 Review 15 lifecycle state machines (race matrix IDs, DEC-147). Reviewer: QA lead. Pri P0.
- [ ] C66 Review 16 policy catalog (bounds, defaults, presets). Reviewer: Governance lead. Pri P0.
- [ ] C67 Review 17 notifications (catalogue vs FR-NTF; latency target). Reviewer: Operations lead. Pri P1.
- [ ] C68 Review 18 export and portability (verifier ownership; schema versioning). Reviewer: QA lead. Pri P1.
- [ ] C69 Review 19 legal questions (no legal conclusions asserted; wording). Reviewer: Product owner with Counsel. Pri P0.
- [ ] C70 Review notes README and the three research notes: every claim tagged; sources have URL and access date. Reviewer: Product architect. Pri P1.

## D) Spec gaps and deferred spec authoring

Deferred sub-specifications are defined in [13 section 7.2](../specs/13-assumptions-risks-decisions-roadmap.md#72-deferred-sub-specifications-future-specs-to-author) (labels `FSP-nn`). A spec is authored only when its wave's dependencies are specified and pilot evidence supports it; one item is recommended early. Each task: author the spec in the standard template (00 section 3), register new IDs, register DEC/ASM entries, review per C4.

**Early**
- [ ] D01 Product architect: author **FSP-11** payments-adjacent exclusions spec (standing list of prohibited features and the counsel gate); depends on none; reviewed before each wave. Pri P1 (author during M2 to M3). Dep: B04.

**Wave 1 (M7)** — start only after M6
- [ ] D02 UX lead: FSP-01 Mobile (installable PWA then native); dep: MVP API stable; DEC-083, DEC-112. Pri P2.
- [ ] D03 UX lead: FSP-02 Multi-language and right-to-left localisation; dep: MVP UX stable; DEC-013. Pri P2.
- [ ] D04 Product architect: FSP-03 Import from other mutual-credit tools; dep: export (EP-71); DEC-145, DEC-146; LEG-Q-25, LEG-Q-28. Pri P2.

**Wave 2 (M8)**
- [ ] D05 Product architect: FSP-04 Integrations and webhooks (events, tokens, partner apps); dep: stable API, M4 hardening; DEC-108. Pri P2.
- [ ] D06 Product owner: FSP-05 Analytics and reporting; dep: export; DEC-018; LEG-Q-26. Pri P2.
- [ ] D07 Product owner: FSP-07 Organisations as Members and sub-accounts; DEC-024; LEG-Q-01, LEG-Q-03. Pri P2.
- [ ] D08 Security lead: FSP-13 Single sign-on and social login; dep: WS-2; DEC-007. Pri P2.
- [ ] D09 Product owner: FSP-14 Hosted subscriptions and billing; dep: operator entity (ASM-004), FSP-04, FSP-11; LEG-Q-16, LEG-Q-17; DEC-020. Pri P2.
- [ ] D10 UX lead: FSP-17 Images/media on Listings, map/location search, matching; DEC-053. Pri P2.
- [ ] D11 Product owner: FSP-20 Additional notification channels (SMS, push); dep: FSP-01 for push; DEC-132. Pri P2.

**Wave 3 (M9)**
- [ ] D12 Product owner: FSP-06 Reputation, endorsements, trust signals; dep: FSP-05, pilot dispute data; LEG-Q-26. Pri P2.
- [ ] D13 Product architect: FSP-08 Standing orders and recurring Trades; dep: FSP-09, FSP-07 optional; DEC-006; LEG-Q-08. Pri P2.
- [ ] D14 Product architect: FSP-09 Holds and reservations; dep: pilot proves WS-1/WS-4; DEC-005; RSK-016. Pri P2.
- [ ] D15 Product architect: FSP-10 Federation and inter-Network clearing; dep: FSP-04, FSP-11, isolation proven; new canon decision; LEG-Q-24, LEG-Q-31. Pri P2.
- [ ] D16 Product owner: FSP-12 Demurrage and community-fund replenishment; dep: WS-6, hoarding evidence (RSK-007); DEC-050; counsel on interest-like treatment. Pri P2.
- [ ] D17 Product owner: FSP-15 Collateral-backed and bilateral trust-line limits; counsel (LEG-Q-08, LEG-Q-09). Pri P2.
- [ ] D18 Governance lead: FSP-16 Member-run voting and panel/jury review; DEC-038. Pri P2.
- [ ] D19 Governance lead: FSP-18 External mediation integration; LEG-Q-22. Pri P2.
- [ ] D20 Security lead: FSP-19 Tamper-evidence anchoring and SIEM export; DEC-097; LEG-Q-19 if distributed ledger. Pri P2.

**Other spec gaps (candidates; confirm need)**
- [ ] D30 Product architect: decide whether a standalone specification for Network presets and templates (charter templates, preset documents POL-039 to POL-041) is needed for pilot onboarding. Pri P2.
- [ ] D31 Operations lead: runbooks beyond outlines (OPS-083 to OPS-085): author full runbooks for restore, reconciliation failure, email outage, loss-of-all-admins. Pri P1 (by M4). Dep: EP-87.
- [ ] D32 Security lead: threat model document (SEC-099 refers to threat-model deltas; no baseline model artifact exists). Pri P1 (by G1).
- [ ] D33 UX lead: content style guide and string catalogue seed (plain-language levels DEC-115, banned terms). Pri P1 (by M2).
- [ ] D34 Product owner: pilot plan and measurement spec (entry criteria, thresholds DEC-017, interview guide; RSK-052). Pri P1 (by M4).

## E) Repository bootstrap

Detailed checklist: [plan section 7](mvp-implementation-plan.md#7-repository-bootstrap-checklist). Owners below; tick here.

- [ ] E01 Product owner: choose licence via ADR A9 (DEC-015, counsel B10); add `LICENSE`. Pri P0. Dep: A01.
- [ ] E02 Security lead: write `SECURITY.md` (private reporting route; SEC-084 to SEC-087; no bounty/safe-harbour claim, DEC-075). Pri P0. Dep: none.
- [ ] E03 Product owner: write `CONTRIBUTING.md` (spec-first, ADR process, covers tags, DCO/CLA decision). Pri P0. Dep: E01.
- [ ] E04 Governance lead: add `CODE_OF_CONDUCT.md` with a named reporting contact. Pri P0.
- [ ] E05 Engineering lead: issue templates (bug, feature citing IDs, decision request, security redirect, accessibility defect). Pri P1.
- [ ] E06 Engineering lead: pull-request template with definition-of-done checklist ([plan section 8.2](mvp-implementation-plan.md#82-definition-of-done-a-change-is-merged)). Pri P1.
- [ ] E07 Engineering lead: `CODEOWNERS` for ledger, authorization, recovery, export, logging (SEC-093). Pri P1. Dep: E01.
- [ ] E08 Engineering lead: protected main branch, required checks, labels (priority, WS, EP, blocked-by-DEC, needs-counsel). Pri P1.
- [ ] E09 Engineering lead: stack ADR accepted (A02) then repository layout and tooling. Pri P0. Dep: A02.
- [ ] E10 Engineering lead: dev container with one-command start (app, datastore, mail catcher). Pri P1. Dep: E09.
- [ ] E11 Engineering lead: seed-data command (Network, two Members) and the Standard-envelope generator for soak tests. Pri P1. Dep: E09.
- [ ] E12 QA lead: CI stages 1 to 3 including `scripts/check-specs.py` and `scripts/gen-traceability.py --check` as required checks; copy lint skeleton (QA-025). Pri P0. Dep: E09.
- [ ] E13 Security lead: secret scan, dependency scan, SAST, SBOM in CI (G2, DEC-077). Pri P1. Dep: E12.
- [ ] E14 Engineering lead: single artifact build, test-what-you-deploy (OPS-009). Pri P1. Dep: E12.
- [ ] E15 QA lead: covers-tag convention documented and parsed (QA-100); decide generator evolution (C12). Pri P1. Dep: C12.
- [ ] E16 Operations lead: staging environment definition and test-data policy (OPS-010 to OPS-014). Pri P1.
- [ ] E17 Product owner: add `README.md` at repo root pointing to specs, plans, ADRs, status (currently no root README). Pri P1.

## F) Implementation epics

Source of truth is the [MVP implementation plan](mvp-implementation-plan.md#3-epic-overview); one checkbox per epic, stories are tracked in the issue tracker once it exists. Owner = suggested lead role.

**W0 (M0/M1 entry)**
- [ ] F01 EP-00 Repository bootstrap and CI (M). Owner: Engineering lead. Pri P0. Dep: E01 to E14.
- [ ] F02 EP-02 Invariants and property test harness (M). Owner: QA lead. Pri P0. Dep: F01.
- [ ] F03 EP-01 Walking skeleton (L). Owner: Engineering lead. Pri P0. Dep: F01, A02.

**W1 (M1)**
- [ ] F10 EP-10 Atomic posting and cached balances (L). Owner: Engineering lead. Pri P0. Dep: F03.
- [ ] F11 EP-11 Idempotency (M). Owner: Engineering lead. Pri P0. Dep: F10, C02.
- [ ] F12 EP-12 Reconciliation and integrity (M). Owner: Engineering lead. Pri P0. Dep: F10.
- [ ] F13 EP-13 Non-trade entry types engine (M). Owner: Engineering lead. Pri P1. Dep: F10.
- [ ] F14 EP-20 Tenancy and Network model (M). Owner: Security lead. Pri P0. Dep: F03.
- [ ] F15 EP-21 Authentication and sessions (L). Owner: Security lead. Pri P0. Dep: F14, A08.
- [ ] F16 EP-23 Roles and authorization matrix (M). Owner: Security lead. Pri P0. Dep: F14, F15.
- [ ] F17 EP-24 Policy engine, versions, notice (L). Owner: Product architect. Pri P0. Dep: F14.
- [ ] F18 EP-89 API contract and OpenAPI (M). Owner: Engineering lead. Pri P1. Dep: F14.
- [ ] F19 EP-90 Design system and application shell (M). Owner: UX lead. Pri P1. Dep: F01.

**W2 (M2)**
- [ ] F20 EP-30 Membership lifecycle (L). Owner: Product owner. Pri P0. Dep: F15, F17.
- [ ] F21 EP-31 Listings and directory (M). Owner: Product owner. Pri P1. Dep: F20.
- [ ] F22 EP-32 Invitations, charter acceptance, onboarding (M). Owner: Product owner. Pri P1. Dep: F20, C05.
- [ ] F23 EP-40 Trade lifecycle (L). Owner: Engineering lead. Pri P0. Dep: F10, F11, F17, F20.
- [ ] F24 EP-41 Balance, history, statements (M). Owner: UX lead. Pri P0. Dep: F23.
- [ ] F25 EP-42 Expiry timers and sweeps (S). Owner: Engineering lead. Pri P0. Dep: F23.
- [ ] F26 EP-50 Notification outbox and email transport (M). Owner: Operations lead. Pri P0. Dep: F10.
- [ ] F27 EP-51 Notification catalogue and preferences (M). Owner: Product owner. Pri P1. Dep: F26.
- [ ] F28 EP-22 Account recovery and anti-takeover (M). Owner: Security lead. Pri P0. Dep: F15, F26.
- [ ] F29 EP-70 Audit events (M). Owner: Security lead. Pri P0. Dep: F16.

**W3 (M3)**
- [ ] F30 EP-60 Admin controls and quorum (L). Owner: Governance lead. Pri P0. Dep: F16, F17, F29.
- [ ] F31 EP-61 Adjustments and Reversals (M). Owner: Governance lead. Pri P0. Dep: F13, F30.
- [ ] F32 EP-62 Disputes (L). Owner: Governance lead. Pri P0. Dep: F31, F27.
- [ ] F33 EP-63 Departure and Network closure (L). Owner: Governance lead. Pri P0. Dep: F13, F20, F30.
- [ ] F34 EP-64 Appeals, moderation, conflict handling (M). Owner: Governance lead. Pri P1. Dep: F30, F21.
- [ ] F35 EP-65 Operator boundary, break-glass, admin recovery (M). Owner: Security lead. Pri P0. Dep: F16, F29.
- [ ] F36 EP-71 Exports and portability (L). Owner: Data protection lead. Pri P0. Dep: F12, F29.
- [ ] F37 EP-72 Erasure, retention, privacy tooling (L). Owner: Data protection lead. Pri P0. Dep: F20, F29, B05.

**W4 (M4)**
- [ ] F40 EP-80 Packaging and install paths T1/T2 (L). Owner: Operations lead. Pri P0. Dep: F01.
- [ ] F41 EP-81 Observability and alerting (M). Owner: Operations lead. Pri P0. Dep: F01 (grows each wave).
- [ ] F42 EP-82 Backup, restore, recovery drills (M). Owner: Operations lead. Pri P0. Dep: F12.
- [ ] F43 EP-83 Migrations and release engineering (M). Owner: Operations lead. Pri P0. Dep: F10.
- [ ] F44 EP-84 Security hardening and testing (L). Owner: Security lead. Pri P0. Dep: F16.
- [ ] F45 EP-85 Accessibility audit and usability study (L). Owner: UX lead. Pri P0. Dep: F19, F24.
- [ ] F46 EP-86 Performance, soak, chaos (M). Owner: QA lead. Pri P0. Dep: F23.
- [ ] F47 EP-87 Support model, incident handling, hosted prerequisites (M). Owner: Operations lead. Pri P0. Dep: F41, G01.
- [ ] F48 EP-88 Internationalisation readiness and copy lint (S/M). Owner: UX lead. Pri P1. Dep: F19.

## G) Ops and hosting prerequisites

Non-engineering and platform items that gate the pilot and hosted launch ([roadmap gates G4 and G5](roadmap.md#6-gono-go-gates)). Self-hosted (T1/T2) deployments need the starred items for their own operators' guidance.

- [ ] G01 Product owner: confirm the Platform Operator entity (legal form, funding, accountable contacts) before M4 (ASM-004, RSK-027, OPS-002). Pri P0. Dep: none.
- [ ] G02 Operations lead: choose hosting provider(s) and regions via ADR A4 with counsel input on data location and sub-processors (LEG-Q-31). Pri P0. Dep: B05.
- [ ] G03 Operations lead: DNS, domains and TLS plan; certificate renewal automation. Pri P1.
- [ ] G04 Operations lead: transactional email provider and sending domain (SPF/DKIM/DMARC); deliverability test for recovery mail (ASM-026, RSK-032). Pri P0. Dep: A5 ADR.
- [ ] G05 Operations lead: backups: destination, encryption, key custody (two custodians, three for T3; OPS-003), retention per DEC-121, first restore drill (QA-E2E-14). Pri P0. Dep: A19. (*)
- [ ] G06 Operations lead: secrets management and rotation interval (DEC-081, OPS-015 to OPS-022). Pri P1. (*)
- [ ] G07 Operations lead: monitoring, alert routing and paging tool; Sev-1 on-call roster confirmed before M5 (ASM-028, DEC-023, RSK-028). Pri P0.
- [ ] G08 Operations lead: status page and maintenance-notice process (DEC-128). Pri P1.
- [ ] G09 Security lead: security contact mailbox, vulnerability handling procedure, disclosure policy wording reviewed by counsel (SEC-084 to SEC-087, DEC-075). Pri P0. Dep: E02.
- [ ] G10 Security lead: break-glass and production-access procedure with dual control for hosted (DEC-073, DEC-079, OPS-101 to OPS-104). Pri P1.
- [ ] G11 Security lead: schedule the independent security assessment before the pilot (SEC-095, security gate G5). Pri P1.
- [ ] G12 UX lead: book the external or independent accessibility audit and usability participants (QA-091, UX-112). Pri P1.
- [ ] G13 Operations lead: support model: tiers, contact channels, response targets, Network admin onboarding pack (OPS-069 to OPS-074). Pri P1.
- [ ] G14 Operations lead: operations record template per deployment and service description for hosted (OPS-003, OPS-004). Pri P1. (*)
- [ ] G15 Operations lead: staging environment from the release artifact; production-sized data for migration rehearsal (OPS-009, DAT-086). Pri P1.
- [ ] G16 Product owner + Counsel: publish terms, privacy notice, accessibility statement, sub-processor list before hosted onboarding (LEG-Q-20, LEG-Q-25, LEG-Q-34; OPS-004). Pri P0 (before G4/G5). Dep: B15, B16.
- [ ] G17 Operations lead: self-host guidance: install guide, pre-flight checker, upgrade and backup minimums (OPS-088 to OPS-095). Pri P1. Dep: F40. (*)
- [ ] G18 Product owner: pilot Network recruitment: criteria, outreach, onboarding sessions; at least 2 Networks (ASM-024, RSK-052). Pri P1 (start at M2). Dep: D34.
- [ ] G19 Operations lead: capacity plan and cost model from the Standard-envelope load test (OPS-096 to OPS-100). Pri P1.
- [ ] G20 Operations lead: exit/wind-down procedure for the operator and a Network package escrow of docs (RSK-030, OPS-007). Pri P2.

## Maintenance of this file

- Re-tick and re-date weekly (working agreement 13); at each milestone exit, close or re-prioritise open items and re-score the risk register (13 section 4).
- Newly discovered issues are added under C2 with owner and priority; do not edit specs from this file.
- When a decision is Accepted, strike the row in section A in the same change as the register update.
