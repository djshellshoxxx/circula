# 01 — Product Vision and Scope

| Field | Value |
|---|---|
| Status | Draft |
| Owner role | Product owner |
| Last updated | 2026-10-21 |
| Scope | Why Circula exists, who it is for, what it will and will not do, the principles that settle trade-offs, measurable success targets, and the MVP boundary |
| Depends on | [00 — Conventions and shared canon](00-conventions.md) |
| Open questions | VIS-OD-1 to VIS-OD-6 in section 12; thresholds marked `[DECISION NEEDED]` |

## 1. Purpose and scope

This document is the top of the specification set. It states the product problem, the people Circula serves, the goals and non-goals, the guiding principles, how success will be measured, and where the MVP boundary sits. Every other spec is read in this context. When a lower-level spec seems to conflict with a goal, non-goal or principle here, this document wins and the conflict is raised as a defect against the lower-level spec.

In scope: product-level intent and boundaries. Out of scope: detailed behaviour (see [04 — Functional requirements](04-functional-requirements.md)), domain rules ([03](03-domain-model-and-mutual-credit-rules.md)), roles and journeys ([02](02-personas-roles-and-journeys.md)), legal analysis ([19](19-legal-and-regulatory-open-questions.md)).

The `VIS-` requirements below are product-level constraints. Their verification methods are: **T** test, **I** inspection of artefacts or documents, **A** analysis or review, **D** demonstration to the project owner, **M** measurement in a pilot or production. Priorities follow [00 section 2](00-conventions.md#2-requirement-identifiers).

## 2. Definitions used here

All terms (Network, Member, Credit unit, Trade, Offer, Request, Journal Entry, Adjustment, Reversal, Dispute, Audit Event, Platform Operator, Network Administrator, Dispute Reviewer) are used exactly as in [00 section 5.1](00-conventions.md#51-vocabulary-exact-spellings) and the [glossary](14-glossary.md). They are not redefined here. Policy parameter names (for example `default_credit_limit`, `membership_mode`) are as in [00 section 5.6](00-conventions.md#56-network-policy-parameter-names-exact-catalog-and-defaults-in-16-network-policy-catalogmd) and [16](16-network-policy-catalog.md).

## 3. Problem statement

Small communities (neighbourhoods, makerspaces, transition-town groups, cooperatives, professional guilds, time-banks, regional trade circles) often have unused capacity: people with time, skills, tools and surplus goods who would trade with each other if everyone could keep an honest, shared account of who has given and who has received. Three obstacles recur:

1. **Bilateral barter is brittle.** Direct swaps need a double coincidence of wants. Without a shared unit of account, a gardener who fixes a neighbour's bike cannot easily "bank" that contribution and use it later with someone else.
2. **Spreadsheet and chat-group bookkeeping does not scale or build trust.** Informal ledgers are editable by whoever holds the file, lack consent records, drift out of balance, lose history when a coordinator leaves, and give no fair way to resolve a disagreement.
3. **Generic payment and ledger tools do not fit.** They assume a national currency, assume a regulated intermediary, charge fees, or require identity checks and data collection that a small community neither wants nor needs. They also lack community-level rules: credit limits set by the group, dispute review by peers, balance visibility chosen by the group, and a process for members who leave owing or owed.

Mutual credit addresses the first obstacle: a member who provides value receives a positive balance, a member who receives value takes a negative balance, and the sum across the network is always zero. Prior art shows recurring failure modes that the product must address rather than ignore: thin liquidity, persistent deficits with disenchanted creditors, volunteer-administrator burnout and key-person risk, hoarding, dormancy, and weak dispute processes (see [prior-art research note](../notes/research-prior-art-mutual-credit.md), section 2.4; some items there are marked unverified). What is missing is dependable, open, accessible software that enforces the accounting rules, leaves governance to the community, and avoids pretending to be money.

## 4. Vision

**Circula is software that lets a community run its own mutual-credit network: a shared, trustworthy account of who has given and received value, governed by rules the community chooses.**

In a successful Circula Network:

- A new member joins in minutes, understands exactly what a positive and a negative balance mean, and can see who is offering and requesting what.
- Two members agree a Trade in a few steps; both consent; both balances change by equal and opposite amounts in one indivisible step; either can see it in their history immediately.
- Nobody, including administrators and the Platform Operator, can silently change a balance. Every correction is a visible, explained, reversible-by-further-entry record.
- The community can see and change its own rules (credit limits, visibility, who joins) with fair notice, and members can challenge decisions.
- A Network can leave Circula taking all its data with it.

### 4.1 Software boundary statement

**Circula is software for operating a mutual-credit network. It does not itself guarantee that credits have monetary value, that any member or business will accept them, that any member will provide goods or services, or that any balance can be converted to anything else.** The credit unit is an accounting unit defined per Network. By Platform design it is not money, not redeemable for national currency, not a security or investment, and not a cryptocurrency or token (INV-014). Whether and how a given Network's use of the software is subject to law is a matter for that Network and its counsel; see [19 — Legal and regulatory open questions](19-legal-and-regulatory-open-questions.md). Nothing in the specification set, the product UI or its marketing may state or imply the contrary.

## 5. Intended users

Detailed personas, roles and permissions are in [02](02-personas-roles-and-journeys.md). At product level:

| Audience | Who they are | What they need from Circula | Priority for MVP |
|---|---|---|---|
| **Network Members** | Individuals in a community who offer and request goods and services | Simple, clear way to find, agree and record trades; unmistakable balance meaning; privacy by default; fair handling of disputes | Primary |
| **Network Administrators** | Volunteer or elected coordinators of one community | Configure rules, admit members, fix mistakes visibly, keep the network healthy, produce records; low workload; protection from being accused of tampering | Primary |
| **Dispute Reviewers** | Trusted members who judge disagreements | Enough context to decide fairly, without gaining extra powers; clear time limits; conflict-of-interest protection | Primary |
| **Platform Operators** | Whoever deploys Circula (a host organisation, or a community self-hosting) | Reliable operation, backup/restore, upgrades, abuse handling, without the ability to alter ledgers | Primary |
| **Applicants and visitors** | People considering joining, or reading public listings | Honest explanation of what a Network is and what the credit unit is not; a low-friction, accessible application | Secondary |
| **Contributors and integrators** | Developers extending or self-hosting Circula | Open, technology-neutral specs; stable API contract; portable data | Secondary |

Network size assumed for design: 10 to 2,000 Members per Network at MVP, typically 30 to 300; 1 to several hundred Networks per deployment. [DECISION NEEDED: VIS-OD-2 design envelope for members per Network and Networks per deployment]. These are planning figures for performance and operations specs, not product promises.

## 6. Goals

| ID | Goal | Priority |
|---|---|---|
| **VIS-001** | The Platform **MUST** provide a Network with an append-only, double-entry accounting of Trades in the Network's credit unit, in which each completed Trade changes exactly two Member Accounts by equal and opposite amounts (INV-001, INV-002, INV-003). *Rationale: this is the core promise; everything else supports it.* *Verification: T (ledger invariant suite, see [12](12-quality-strategy-and-test-plan.md)), A.* | P0 |
| **VIS-002** | The Platform **MUST** enforce platform-wide invariants identically on every entry path (UI, API, admin tools, import) and **MUST NOT** allow any role to silently alter a balance (INV-004, INV-010, INV-012). *Rationale: trust in the network is trust in its books. Verification: T (every entry path), I.* | P0 |
| **VIS-003** | A Network **MUST** be able to set and later change, with notice, its own participation, credit-limit, dispute, visibility and inactivity rules within platform invariants, using the parameters in [16](16-network-policy-catalog.md). *Rationale: communities differ; the platform provides mechanism, not policy. Verification: T, D.* | P0 |
| **VIS-004** | Every Member **MUST** be able to see, at any time and without ambiguity, their balance, its direction and meaning, available credit, pending proposals and completed history. *Rationale: misunderstanding of negative balances is the main adoption and trust risk. Verification: T (UX acceptance in [08](08-ux-and-accessibility.md)), usability test.* | P0 |
| **VIS-005** | Members **MUST** be able to offer and request goods and services in a browsable directory, and to propose a Trade in a few steps with the explicit consent of both parties (D-06). *Rationale: the ledger is useless without a way to find counterparties. Verification: T, D.* | P0 |
| **VIS-006** | Members **MUST** be able to raise a Dispute about a Trade, have it reviewed by a Dispute Reviewer who is not a party, and see the outcome and any resulting Reversal or Adjustment. *Rationale: informal systems fail when members disagree. Verification: T, D.* | P0 |
| **VIS-007** | The product **MUST** be usable to WCAG 2.2 Level AA (D-14) on current mainstream browsers on phone, tablet and desktop for all MVP journeys in [02](02-personas-roles-and-journeys.md). *Rationale: community networks include older and disabled members; exclusion defeats the purpose. Verification: T (automated plus manual audit), A.* | P0 |
| **VIS-008** | A Network **MUST** be able to export its complete data (members, listings, Journal Entries, Audit Events, policies) in a documented, machine-readable form and a Member **MUST** be able to export their own data ([18](18-data-export-and-portability.md)). *Rationale: no lock-in; a basis for trust and for leaving. Verification: T (export correctness and round-trip).* | P0 |
| **VIS-009** | The software **MUST** be operable by a single small team or volunteer group with documented procedures for deployment, backup, restore, upgrade and incident response ([11](11-deployment-operations-and-support.md)), in both operator-hosted and self-hosted modes (D-01). *Rationale: target communities have little technical capacity. Verification: D (restore drill), I.* | P0 |
| **VIS-010** | The Platform **SHOULD** collect the minimum personal data needed to run a Network, give Networks control over who sees what, and support erasure or anonymisation of personal data on closure without altering ledger arithmetic (D-09; [09](09-security-privacy-and-abuse.md)). *Rationale: privacy is a trust and legal-risk issue. Verification: I, T.* | P1 |
| **VIS-011** | Specifications and code **SHOULD** remain technology-neutral at the specification level (D-02) so that multiple implementations and deployment styles are possible. *Rationale: portability and longevity. Verification: I.* | P1 |

## 7. Non-goals

The following are explicitly **not** goals of Circula in the MVP. Items marked "designed-for" are not built now but the MVP MUST NOT make them impossible later.

| ID | Non-goal | Treatment |
|---|---|---|
| **VIS-012** | Circula **MUST NOT** provide redemption, cash-out, withdrawal, interest, yield, fiat conversion, card or bank-payment integration, or a token/coin/blockchain representation of credits (INV-014). Any later proposal is a new product decision accompanied by legal review. *Rationale: avoids money-transmission, securities and consumer-finance exposure and prevents misleading members. Verification: I (feature inventory), T (no such endpoints).* | Out; not designed-for |
| **VIS-013** | Circula **MUST NOT** guarantee, in software, UI copy or documentation, that credits have monetary value, will be accepted, or can be redeemed (section 4.1). *Rationale: honesty with members. Verification: I (copy review gate).* | Out |
| **VIS-014** | Circula **MUST NOT** act as escrow, guarantor, insurer or debt collector for Trades or negative balances; recovery of negative balances is a Network governance matter ([10](10-governance-and-network-operations.md)). *Rationale: the Platform does not underwrite quality or default. Verification: I.* | Out |
| **VIS-015** | Circula **MUST NOT** perform identity verification (KYC/AML), credit scoring or reputation scoring as a platform feature in the MVP. Networks may apply their own human admission checks. *Rationale: data minimisation and scope; regulatory questions go to counsel. Verification: I.* | Out; reputation designed-for post-MVP |
| **VIS-016** | The MVP **MUST NOT** include cross-Network postings, federation or inter-Network clearing (INV-011). *Rationale: isolates trust domains and complexity. Verification: T.* | Deferred, designed-for |
| **VIS-017** | The MVP **MUST NOT** include native mobile apps, offline mode, or push-notification infrastructure beyond email; the web UI is responsive. *Rationale: focus. Verification: I.* | Deferred |
| **VIS-018** | The MVP **MUST NOT** include paid subscriptions, billing, or monetised admin features. Hosted-service billing is out of scope (D-01). *Rationale: stated scope of this specification pass. Verification: I.* | Deferred |
| **VIS-019** | The MVP **MUST NOT** include organisation (business) memberships, multi-signature accounts, standing orders, recurring trades, funds holds/reservations (D-05, D-06), or group/split trades. *Rationale: each adds domain complexity; the model must not preclude them. Verification: I.* | Deferred, designed-for |
| **VIS-020** | The MVP **MUST NOT** include advanced analytics, recommendation/matching engines, payments to third parties, or external accounting integrations. Basic exports ([18](18-data-export-and-portability.md)) are the integration path. *Rationale: scope. Verification: I.* | Deferred |
| **VIS-021** | The MVP **MUST NOT** include general-purpose messaging or social networking beyond listing contact, Trade notes and Dispute threads. *Rationale: avoids moderation and safety scope. Verification: I.* | Out |

## 8. Guiding principles

These principles are used to settle design trade-offs, in this order when they conflict.

| # | Principle | Practical consequence |
|---|---|---|
| P1 | **Ledger integrity first.** | When correctness conflicts with convenience or speed, correctness wins. A rejected Trade with a clear reason is better than a wrong balance. |
| P2 | **Consent for every debit.** | No role can debit a Member without that Member's acceptance (INV-012, D-06). Admin corrections are visible Adjustments. |
| P3 | **Visible, explained change.** | No silent mutation. Every correction, rule change and role change is recorded with actor, time and reason (INV-004, INV-010) and shown to the affected Member. |
| P4 | **Governance is the community's; mechanism is the Platform's.** | Policies are configurable per Network within invariants; the software enforces what the Network chose, and does not choose for it. The Platform Operator has no ledger powers. |
| P5 | **Honest about what credits are.** | Balances are described as net provided/received, never as money, savings or debt in national currency. |
| P6 | **Clarity over cleverness.** | Plain language, unmistakable balance direction, few steps, no dark patterns, no gamified pressure to trade. |
| P7 | **Accessible by default.** | WCAG 2.2 AA is a release gate, not a later improvement (D-14). |
| P8 | **Minimal data, maximum control.** | Collect only what is needed; Networks choose visibility; members can export and leave. |
| P9 | **No lock-in.** | Open specifications, portable exports, technology-neutral contracts (D-02). |
| P10 | **Operable by small teams.** | Prefer fewer moving parts; document recovery; fail safe (stop posting) rather than guess. |
| P11 | **Design for what is deferred, build only what is in scope.** | Identifiers, schemas and contracts leave room for organisations, holds, federation and reputation without implementing them. |

**VIS-022** — Specifications, UI copy and API documentation **MUST** be consistent with principles P1 to P3 and P5; a change that contradicts them requires a recorded decision (decision log, [13](13-assumptions-risks-decisions-roadmap.md)). *Rationale: keeps the product coherent as it grows. Verification: I (spec review checklist).* Priority P0.

## 9. Success measures

Targets are provisional until the pilot is defined; thresholds marked `[DECISION NEEDED]` need owner sign-off (see VIS-OD-3). Measurements use only data the Platform already holds for operating the Network; no extra tracking of Members is introduced to measure success (VIS-010). Aggregate figures are computed per Network and reported to that Network's Administrators; any cross-Network reporting to a Platform Operator is aggregate-only and opt-in [DECISION NEEDED: VIS-OD-4 telemetry policy].

### 9.1 Product health measures (pilot and production)

| ID | Measure | Target | How measured | Priority |
|---|---|---|---|---|
| **VIS-023** | Ledger integrity: reconciliation runs (recomputed balances vs cached balances; per-Network sum of balances; Journal Entry zero-sum) | 100 percent of scheduled runs pass; 0 confirmed invariant violations in production; any failure is a Severity-1 incident ([11](11-deployment-operations-and-support.md)) | Automated reconciliation job output (M, T) | P0 |
| **VIS-024** | Trade completion latency: time from Counterparty acceptance to both balances visible | 95th percentile at or under 2 seconds under reference load [DECISION NEEDED: VIS-OD-2 reference load] | Performance test and production timing (T, M) | P0 |
| **VIS-025** | Time to first value: median time from a person accepting an invitation (or approval) to seeing their account and the directory | At or under 10 minutes median in usability testing with at least 8 participants, none with prior knowledge of mutual credit | Moderated test (D, M) | P1 |
| **VIS-026** | Activation: share of newly activated Members completing a first Trade within 30 days | At least 40 percent in pilot Networks [DECISION NEEDED: VIS-OD-3 pilot target] | Network-level aggregate (M) | P1 |
| **VIS-027** | Comprehension: share of test participants who correctly state, for a given account view, whether the Member has provided or received more, and how much further they can spend | At least 90 percent correct across 5 scenarios including a negative balance and a balance at the limit | Usability test script from [08](08-ux-and-accessibility.md) (D) | P0 |
| **VIS-028** | Proposal outcomes: share of Trade proposals that end `completed` (versus `declined`, `cancelled`, `expired`) | Reported, not targeted, in MVP; investigate if `expired` exceeds 30 percent [DECISION NEEDED: VIS-OD-3] | Network-level aggregate (M) | P2 |
| **VIS-029** | Dispute handling: share of Disputes reaching a `resolved_*`, `withdrawn` or `escalated` state within `dispute_review_sla_days` of being `open` | At least 90 percent | Dispute timestamps (M) | P1 |
| **VIS-030** | Dispute prevalence: Disputes per 100 completed Trades | Reported, not targeted; used to detect UX confusion if above 3 [DECISION NEEDED: VIS-OD-3] | Aggregate (M) | P2 |
| **VIS-031** | Silent-change defects: balance changes without a linked Journal Entry and, where applicable, Audit Event | 0 | Audit completeness check (T, M) | P0 |
| **VIS-032** | Administrator workload: median Administrator time to approve a membership application | At or under 2 minutes in usability testing | Moderated test (D) | P2 |

### 9.2 Delivery and quality measures

| ID | Measure | Target | Verification | Priority |
|---|---|---|---|---|
| **VIS-033** | Accessibility: WCAG 2.2 AA failures in MVP journeys at release | 0 open failures at severity "blocker" or "serious"; remaining issues documented with owner and date | Automated scan plus manual keyboard and screen-reader pass (T, A) | P0 |
| **VIS-034** | Restore: full restore of a representative Network from backup, with ledger reconciliation passing | Completed and rehearsed before launch; recovery time and recovery point objectives are set in [11](11-deployment-operations-and-support.md) [DECISION NEEDED: RTO/RPO] | Restore drill (D) | P0 |
| **VIS-035** | Export correctness: export then re-import (or independent recomputation) reproduces all balances | 100 percent of accounts match for the pilot dataset and the synthetic large dataset | Round-trip test (T) | P0 |
| **VIS-036** | Concurrency correctness: parallel Trades touching the same accounts, with retries | 0 limit breaches, 0 duplicate postings in the soak test defined in [12](12-quality-strategy-and-test-plan.md) | Test (T) | P0 |
| **VIS-037** | Spec health: dangling requirement references, duplicate IDs, unmarked uncertain choices | 0 at each baseline review | Traceability check (I) | P1 |

## 10. MVP boundary

"In" = P0/P1 scope that must ship. "Deferred" = not built in MVP but the design must leave room. "Out" = not planned; requires a new decision and (where noted) counsel. Functional detail and acceptance criteria are in [04](04-functional-requirements.md).

| Capability area | In MVP | Deferred (designed-for) | Out |
|---|---|---|---|
| **Network setup** | Create a Network; set unit name/symbol/scale; set policy parameters; invite first Administrator(s); set policy notice period | Network templates; Network cloning; multi-tenant billing | Changing unit identity or scale after first posting (INV-013) |
| **Identity and access** | Email-based account; passkey or TOTP-capable authentication; one login with several memberships (D-07); account recovery; sessions; role-based authorization | Single sign-on; social login; delegated/agent access | Platform-level identity verification (VIS-015) |
| **Membership** | Invite, approval and (optional) open sign-up per `membership_mode`; suspend; departure and closure; admin removal with reason and appeal | Organisation members; sub-accounts; family accounts; membership fees | Public global member registry |
| **Listings and directory** | Offers and Requests with title, description, category, availability; search/browse; pause/archive; moderation by Administrators; `directory_visibility` | Images and media; map/location radius search; matching/recommendation | Paid promotion; ratings and reviews |
| **Trades** | Propose by either party; accept/decline/cancel; expiry; atomic posting; limits; notes; idempotent retries | Standing/recurring orders; split or group trades; holds/reservations (D-05); partial acceptance; multi-currency | Unilateral debit; credits-for-money sale |
| **Balances and history** | Balance, available credit, pending proposals; filterable history; per-Network `balance_visibility`; statement view | Charts and trend analytics; scheduled statements | Interest/demurrage accrual; yield |
| **Admin controls** | Policy edit with notice; member management; Adjustment with reason; initiate Reversal; listing moderation; role assignment; `admin_quorum` | Delegated sub-roles; automated policy rules; bulk operations | Posting Trades on a Member's behalf (INV-012) |
| **Disputes** | Raise, evidence note, reviewer assignment with conflict check, resolve (upheld/rejected/settled), withdraw, escalate; resulting Reversal/Adjustment; SLA tracking | External mediation integration; panel/jury voting; reputation consequences | Balance freezing (D-12) |
| **Audit** | Append-only Audit Events; Administrator view; Member view of events about themselves | Tamper-evident anchoring/hash-chaining beyond MVP baseline; SIEM export | Public audit log of personal data |
| **Export and portability** | Member export; Network export; documented schema; ledger re-computation check | Scheduled exports; import from other mutual-credit tools | Third-party accounting integrations |
| **Notifications** | Email notifications for proposals, outcomes, disputes, policy notices ([17](17-notifications-and-communications.md)) | SMS; push; digest preferences beyond basics | Marketing messaging |
| **Platform operation** | Operator-hosted multi-tenant and self-hosted install (D-01); backups/restore; monitoring; upgrades; Network suspension/closure by Operator for abuse or non-payment of hosting | Managed SaaS offering with subscription billing; regional data residency options | Operator access to ledger mutation |
| **Cross-Network** | None | Federation; inter-Network clearing; shared identity across Networks beyond a single login | Cross-Network postings in MVP (INV-011) |
| **Localisation** | i18n-ready; English UI (D-13) | Additional languages; right-to-left layouts | — |
| **Mobile** | Responsive web | Installable PWA; native apps | — |

**VIS-038** — Each MVP-In capability in the table above **MUST** map to at least one `FR-` requirement in [04](04-functional-requirements.md), and each Deferred item **MUST** be listed as deferred in [13](13-assumptions-risks-decisions-roadmap.md) with the extension point that keeps it possible. *Rationale: prevents scope drift and silent omissions. Verification: I (traceability check).* Priority P0.

**VIS-039** — No Out item **MAY** be introduced without a Baseline-status decision recorded in the decision log and, for items touching VIS-012 to VIS-015, written advice from counsel recorded against the relevant `LEG-Q-` question in [19](19-legal-and-regulatory-open-questions.md). *Rationale: protects the non-goals from erosion. Verification: I.* Priority P0.

## 11. Roles at product level

Circula distinguishes **platform operation** from **Network governance** (elaborated in [05](05-architecture-and-trust-boundaries.md) and [10](10-governance-and-network-operations.md)):

| Layer | Who | Controls | Cannot |
|---|---|---|---|
| Platform operation | Platform Operator (`platform_operator`) | Infrastructure, backups, upgrades, abuse handling, provisioning and suspending Networks | Post, edit or reverse Journal Entries; read private Network content by default; act as a Member |
| Network governance | Network Administrators (`network_admin`), supported by Dispute Reviewers (`dispute_reviewer`) | Network policy within invariants, membership, moderation, Adjustments and Reversals with reason and audit | Alter the platform invariants; debit a Member without their acceptance; change `unit_scale` after first posting |
| Participation | Members | Own listings, proposals, acceptances, disputes, exports | See others' private data beyond Network policy |

**VIS-040** — The Platform **MUST** keep these layers separate so that holding an operational role never implies a governance role and vice versa; a single person holding several roles **MUST** be visibly acting in exactly one capacity per action (Audit Event records the role used). *Rationale: separation of duties limits administrator and operator misuse. Verification: T, I.* Priority P0.

## 12. Assumptions and open decisions

Items here are proposals for the project owner. They are registered in [13](13-assumptions-risks-decisions-roadmap.md); the canon defaults D-01 to D-14 in [00 section 5.5](00-conventions.md#55-baseline-defaults-to-be-registered-as-dec--entries-with-status-proposed-default-in-doc-13) apply.

| Ref | Question | Options | Recommended default |
|---|---|---|---|
| VIS-OD-1 | Open-source licence and contribution model (repository has none yet) | Permissive (for example Apache-2.0 or MIT); strong copyleft (for example AGPL-3.0); source-available | Strong copyleft that closes the network-service gap so hosted forks share changes; confirm with the project owner. *This is a licensing choice with legal consequences: a question for counsel.* |
| VIS-OD-2 | Design envelope: Members per Network, Networks per deployment, trades per day | (a) 300 / 50 / 1,000; (b) 2,000 / 500 / 20,000; (c) larger | (b) as an upper design limit, with (a) as the pilot envelope |
| VIS-OD-3 | Pilot success thresholds (activation 40 percent, expiry 30 percent, dispute rate 3 per 100) | Stricter; looser; report-only | Keep as provisional; revisit after first pilot wave |
| VIS-OD-4 | Telemetry: whether any usage data flows to the Platform Operator beyond operational necessity | None; aggregate opt-in per Network; aggregate default-on | Aggregate, opt-in per Network, no personal data |
| VIS-OD-5 | Intended first audience: time-banks (equal-value hours), local trading circles (priced in units), or both | Hours-only template; both with Network-chosen unit | Both; `unit_scale` default 2 as in the canon, with a one-click "hours" preset (scale 2, 60-minute convention is a Network matter) |
| VIS-OD-6 | Whether the Platform Operator may host Networks for non-member third parties in MVP (hosting service terms) | Operator-hosted only; self-host only; both (D-01) | Both, per D-01; hosting terms are a business and legal matter outside this pass |

Assumptions: (A1) Networks are communities with a degree of mutual trust and a human governance process; the software does not replace governance. (A2) Members have access to a modern web browser and an email address. (A3) Networks, not the Platform, decide the working meaning and informal value of a unit. (A4) Legal characterisation of any particular Network is unresolved and jurisdiction-specific; see [19](19-legal-and-regulatory-open-questions.md).

## 13. References

Internal:

- [00 — Conventions and shared canon](00-conventions.md)
- [02 — Personas, roles and journeys](02-personas-roles-and-journeys.md)
- [03 — Domain model and mutual-credit rules](03-domain-model-and-mutual-credit-rules.md)
- [04 — Functional requirements](04-functional-requirements.md)
- [05 — Architecture and trust boundaries](05-architecture-and-trust-boundaries.md)
- [08 — UX and accessibility](08-ux-and-accessibility.md)
- [09 — Security, privacy and abuse](09-security-privacy-and-abuse.md)
- [10 — Governance and network operations](10-governance-and-network-operations.md)
- [11 — Deployment, operations and support](11-deployment-operations-and-support.md)
- [12 — Quality strategy and test plan](12-quality-strategy-and-test-plan.md)
- [13 — Assumptions, risks, decisions, roadmap](13-assumptions-risks-decisions-roadmap.md)
- [14 — Glossary](14-glossary.md)
- [16 — Network policy catalog](16-network-policy-catalog.md)
- [17 — Notifications](17-notifications-and-communications.md)
- [18 — Data export and portability](18-data-export-and-portability.md)
- [19 — Legal and regulatory open questions](19-legal-and-regulatory-open-questions.md)

External: WCAG 2.2, W3C Recommendation, <https://www.w3.org/TR/WCAG22/> (accessed 2026-10-21). Prior-art findings on existing mutual-credit systems: [research-prior-art-mutual-credit](../notes/research-prior-art-mutual-credit.md). Regulatory and privacy questions: [research-regulatory-privacy-questions](../notes/research-regulatory-privacy-questions.md).
