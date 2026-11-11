# 13 Assumptions, Risks, Decisions and Roadmap

Status: Draft v0.1 · Owner/role: Product architect (decision log owner) · Last updated: 2026-10-21 · Scope: assumptions (ASM), risks (RSK), the decision register (DEC) including DEC-001..DEC-004 from [00](00-conventions.md), open decisions with recommended defaults, the MVP roadmap and dependency order, deferred sub-specifications, the decision log template, and legal questions reserved for counsel · Dependencies: all of 00 to 12 · Open questions: section 8

Normative terms follow [00 section 1](00-conventions.md#1-normative-terms). Glossary terms are in [00 section 3](00-conventions.md#3-glossary-authoritative).

**Authority rule.** DEC-001 to DEC-004 are defined in [00 section 6](00-conventions.md#6-shared-technical-decisions-recommended-defaults-see-dec-log), which is authoritative for their wording. The register below records their status and where they apply; if the two ever disagree, [00](00-conventions.md) wins until this log is updated. Decisions DEC-005 onward are defined here.

Identifiers `ASM-`, `RSK-` and `DEC-` are never reused. A superseded decision keeps its ID with status `superseded`.

---





## Build status of the decision register (as built)

Snapshot: HEAD `925f637` (branch claude/circula-build). Statuses were first read at `31b1015` and re-checked at `925f637` (see TRACEABILITY.md). Decisions made during the build, and open owner decisions raised by it, are in [DECISIONS.md](../DECISIONS.md) with their IDs. Test names and code paths: [TRACEABILITY.md](../TRACEABILITY.md).

Summary: decided 2, deferred 1, open 20.

| ID | Priority | Status | Note |
|---|---|---|---|
| DEC-005 | - | open, partly built | Single idle (720 h) and absolute (2160 h) timeouts for all roles; role split not built |
| DEC-006 | - | open, partly built | Min length 12 and Argon2id built; benchmark and breached-password list not |
| DEC-007 | - | open, not built | No second factor built |
| DEC-008 | - | open, partly built | One shared database with network-scoped rows; per-network files not built |
| DEC-009 | - | open, not built | No recovery channel and no mail |
| DEC-010 | - | open, partly built | Retain default and settle-to-zero rule built; counsel review open |
| DEC-011 | - | open, partly built | Grant off by default; approval or invite requirement and caps not built |
| DEC-012 | - | partly built | Response period is a network policy (default 7 days) and gates decisions; decision-target and appeal timelines not built |
| DEC-013 | - | open, not built | Dual control not built |
| DEC-014 | - | open, not built | Break-glass not built |
| DEC-015 | - | open, partly built | Unkeyed hash chain built; keyed or external publication deferred |
| DEC-016 | - | open, not built | RPO and RTO not recorded |
| DEC-017 | - | decided, built | Flags, environment, file, defaults; explicit migrate default |
| DEC-018 | - | decided, built | Text exposition; loopback allow-list and optional token |
| DEC-019 | - | open, not built | Log retention not implemented |
| DEC-020 | - | open, not built | Retention periods not set |
| DEC-021 | - | open, partly built | Unilateral administrator change built; objection right not built |
| DEC-022 | - | open, not built | Support objectives not set |
| DEC-023 | - | open, not built | Numerical thresholds not confirmed |
| DEC-024 | - | open, not built | Accessibility level and browser list not recorded |
| DEC-025 | - | open, partly built | Closure notice built (30 days minimum, no override); adverse policy changes use the 14-day default of GOV-003 |
| DEC-026 | - | open, partly built | Private vulnerability reporting named in SECURITY.md; owner approval not recorded |
| DEC-027 | - | deferred | Code signing and notarisation deferred |

## 1. Decision register

### 1.1 Accepted baseline (from 00)

| ID | Decision (short form) | Status | Applies to | Authoritative text |
|---|---|---|---|---|
| DEC-001 | Single self-hostable binary plus optional hosted deployment; Go; embedded SQLite; storage behind an interface | Recommended default, adopted as baseline | [05](05-architecture.md), [11](11-deployment-operations.md), OPS-001 to OPS-005 | [00 §6](00-conventions.md#6-shared-technical-decisions-recommended-defaults-see-dec-log) |
| DEC-002 | Server-rendered HTML with progressive enhancement, plus versioned JSON API; both use one domain service layer | Recommended default, adopted as baseline | [07](07-api-contract.md), [08](08-ux-accessibility.md), SEC-023, QA-034 | [00 §6](00-conventions.md#6-shared-technical-decisions-recommended-defaults-see-dec-log) |
| DEC-003 | Release targets windows/amd64, linux/amd64, linux/arm64, darwin/amd64, darwin/arm64, with SHA-256 checksums | Recommended default, adopted as baseline | OPS-003, OPS-035, QA-039 | [00 §6](00-conventions.md#6-shared-technical-decisions-recommended-defaults-see-dec-log) |
| DEC-004 | Product license undecided; repository ships without LICENSE until decided; recommendation AGPL-3.0 or MIT | Open, owner decision pending | Repository release gate; SEC-031 | [00 §6](00-conventions.md#6-shared-technical-decisions-recommended-defaults-see-dec-log) |

### 1.2 Open decisions with recommended defaults

Each open decision has a recommended default. A default is not a decision: it is used for planning and MUST be confirmed or replaced by a recorded decision before the milestone named in "Needed before".

| ID | Question | Options | Recommended default | Affects | Needed before |
|---|---|---|---|---|---|
| DEC-005 | Session idle and absolute timeouts | Shared values; separate values by role | Members: idle 30 days (matches the default in [06](06-data-ledger-audit.md) section 9), absolute 90 days. Administrators and reviewers: idle 60 minutes, absolute 12 hours | SEC-003, OPS-008 | Authentication build |
| DEC-006 | Password minimum length, maximum accepted length, breached-password check source, Argon2id or bcrypt parameters | Parameter sets; check via offline list or none | Minimum 12, accept at least 64; offline breached-password check enabled; Argon2id, parameters chosen by benchmark on minimum hardware | SEC-001, SEC-002 | Authentication build |
| DEC-007 | Mandatory second factor for privileged roles | Mandatory for administrators; for reviewers too; optional for all | Mandatory for network administrators and platform operators; optional for members and reviewers | SEC-007 | Authentication build |
| DEC-008 | Hosted storage isolation: one database file per network, or one shared file with tenant scoping | Per-network file; shared file | One database file per network in hosted mode. Revisit if operations cost is too high | SEC-015, OPS-017 to OPS-020, OPS-005 | Storage design in 05 and 06 |
| DEC-009 | Recovery channel and mail dependency | Verified email required; admin-assisted only; both | Verified email for recovery; admin-assisted recovery with dual control (SEC-011) when no verified contact exists; notices are in-app and by mail when configured | SEC-008, OPS-002, OPS-029 | Authentication build |
| DEC-010 | Departure settlement: residual positive and negative balances | Keep recorded; forfeit; settle by network rule; write off by adjustment | Freeze account; keep residual balances recorded; residual negative balance may be written off only by approved adjustment (GOV-018); no redemption in any form | GOV-010, GOV-018 | Membership build; counsel review (RSK-010) |
| DEC-011 | Sign-up grants in MVP | Off; on with caps and approval | Off by default (grant zero); when enabled, approval or invite join mode required | SEC-032, GOV-008 | Membership build |
| DEC-012 | Dispute and appeal timelines | Various | Dispute window 30 days after completion; response 7 days; decision target 14 days after response deadline; appeal within 14 days; no default outcome on silence | GOV-013, GOV-014, GOV-016 | Dispute build |
| DEC-013 | Dual control for adjustments and write-offs | Single administrator; dual control; threshold-based | Dual control whenever the network has two or more administrators; single administrator otherwise, with the restriction shown to members | SEC-011, GOV-018 | Dispute and admin build |
| DEC-014 | Platform operator break-glass controls and notice period | Technical gating; process-only; notice windows | Two-person approval and audit trail readable by affected administrators; access expires after 4 hours (the proposal in [05](05-architecture.md) ARC-009); notice to affected administrators within 24 hours where legally permitted (recommended, counsel review) | SEC-013, SEC-030, GOV-021, GOV-024 | Hosted launch |
| DEC-015 | Tamper-evident audit chain | Defer; implement in MVP | Defer to post-MVP (SEC-034); keep stable per-network sequence in the data model | SEC-034, 06 | Data model freeze |
| DEC-016 | Backup recovery point and recovery time objectives, off-host copy policy | Various | Hosted: recovery point at most 1 hour, recovery time at most 4 hours, one off-host copy. Self-hosted: daily backups, off-host copy recommended | OPS-019, OPS-034 | Hosted launch |
| DEC-017 | Configuration file format and precedence; migration mode default | Various | Precedence flags, environment, file, defaults; explicit migration mode for self-hosted; explicit for hosted | OPS-006, OPS-025 | Packaging build |
| DEC-018 | Metrics exposition format and default access range | Text exposition format of a common kind; other | Common text exposition format; restricted to loopback or a configured internal range by default | OPS-013 | Monitoring build |
| DEC-019 | Log retention period | Various | 90 days, then pruned | SEC-021 | Monitoring build |
| DEC-020 | Retention of personal data and of closed-network records | Various | Retention periods to be set after counsel review; backups follow the stated retention (SEC-028) | SEC-018, SEC-028, GOV-025 | Hosted launch; counsel review (RSK-010) |
| DEC-021 | Governance decision method for policy changes | Unilateral with notice and objection; member vote; committee | Unilateral administrator change with notice and objection rights; member voting deferred | GOV-006 | Governance build |
| DEC-022 | Support objectives and any external commitment | Best effort; response targets; service levels | Documented targets, not presented as service levels until the platform operator approves them | OPS-030 | Hosted launch |
| DEC-023 | Numerical thresholds: property-test run sizes, interleaving counts, coverage, performance, timing-equality threshold | Various | Values in QA-001, QA-007, QA-008, QA-040 as recommended there; each confirmed on reference hardware before release | QA-001, QA-007, QA-008, QA-040, OPS-017 | Release candidate |
| DEC-024 | Accessibility conformance level and supported browser and screen reader combinations | Level A, AA, AAA; combinations | WCAG 2.2 level AA; current evergreen browsers; one screen reader per major platform, recorded | QA-033, QA-035, UX requirements in 08 | Interface build |
| DEC-025 | Notice periods for adverse policy changes and for closure | Various | 14 days for adverse policy changes; 30 days for network closure | GOV-003, GOV-025, GOV-026 | Governance build |
| DEC-026 | Vulnerability disclosure contact | A role mailbox; a platform form; a public tracker | A role-based contact approved by the owner; no personal address published without approval | SEC-031 | First public release |
| DEC-027 | Code signing and notarisation of release artifacts | Sign now; defer | Deferred to post-MVP | OPS-035 | Post-MVP |

### 1.3 Decision log template

Use one entry per decision. Store entries in this file (for DEC-005 onward) or in an ADR directory if the project adopts one; the template is the same. Entries are not edited after acceptance except for status changes.

- ID: DEC-nnn (next unused number)
- Title:
- Status: proposed / open / accepted / superseded (by DEC-mmm) / withdrawn
- Date opened and date decided (ISO 8601, UTC):
- Owner/role who decides:
- Context and question, with links to the requirement IDs affected:
- Options considered, each with consequences:
- Decision:
- Rationale:
- Recommended default at the time of opening, if any, and whether it was adopted:
- Requirements or documents changed, with their last-updated dates:
- Verification impact (tests or drills added, per [12](12-quality-test-plan.md)):
- Review or revisit date:
- Counsel review required (yes/no), and the outcome if yes:

---

## 2. Assumptions

| ID | Assumption | Impact if false | How validated | Owner/role |
|---|---|---|---|---|
| ASM-001 | A typical network has fewer than 1,000 members and modest transaction volume | Single-writer SQLite may limit throughput; DEC-008 and OPS-005 would need revisiting | Load testing (QA-008, L15); early user feedback | Product architect |
| ASM-002 | Single-node embedded deployment is acceptable for the MVP, with explicit operator-managed backup | Need for high availability; scope of OPS-005 moves into MVP | Pilot operator interviews | Operations lead |
| ASM-003 | Members can provide a verified email address, or an administrator can verify identity out of band | Account recovery falls back to administrator-assisted flow more often | Pilot onboarding data | Product architect |
| ASM-004 | Network administrators are community members acting in good faith but can make mistakes or be tempted to misuse power | Need for stronger governance controls and dual control | Design review of SEC-010 to SEC-013 | Security reviewer |
| ASM-005 | A hosted platform operator is trusted to operate the software but cannot be assumed to be free of operational access to data | Client-side or end-to-end encryption would need a new specification | Review of TA-5 with the operator | Security reviewer |
| ASM-006 | Credits are an accounting unit and do not depend on any monetary value or convertibility, as stated in [00](00-conventions.md) | Product and legal classification would change; counsel review (RSK-010) | Counsel review | Product architect and counsel |
| ASM-007 | Members use current evergreen browsers with scripting enabled, though core flows work without scripting | Unsupported browsers may fail accessibility or security checks | Browser analytics at pilot, if collected under SEC-014 | Interface lead |
| ASM-008 | The team can build and maintain a Go service with an embedded SQLite store and a server-rendered interface | Schedule and skill risk | Team assessment before build | Engineering lead |
| ASM-009 | Transaction and member volumes are unknown; sizing will be based on pilot data | Performance targets may be wrong | Measurement on reference hardware (QA-040, DEC-023) | Quality lead |
| ASM-010 | Dispute volumes are low enough for a single dispute reviewer per network, plus an alternate | Backlogs and longer decisions | Pilot data | Product architect |
| ASM-011 | Members will accept text-only evidence in disputes for the MVP | Pressure to add uploads, which need new security specification | Pilot feedback | Product architect |
| ASM-012 | The repository's current contents (00 and the prompt) are the only prior specification | Conflicting prior material would need reconciliation | Repository review at each milestone | Product architect |

## 3. Risks

Likelihood (L) and impact (I) are rated low, medium or high before mitigation. "Owner" is a role.

| ID | Risk | L | I | Mitigation (requirement IDs) | Owner |
|---|---|---|---|---|---|
| RSK-001 | Collusive or wash trades create false balances that software cannot distinguish from real trade | High | High | SEC-026 signals; dispute process (GOV-013 to GOV-016); reviewer independence (SEC-010); residual risk accepted and disclosed (TA-7) | Product architect |
| RSK-002 | Sybil accounts farm sign-up grants or exploit open join mode | Medium | High | Default grant zero (DEC-011); SEC-032; join modes (GOV-008) | Product architect |
| RSK-003 | Ledger misposting from an implementation defect | Medium | High | LED invariants; QA-001 to QA-010; verification (OPS-016); freeze on failure (OPS-033) | Quality lead |
| RSK-004 | Over-limit postings under concurrency | Medium | High | LED-5 serialisation; OPS-012; QA-007 | Engineering lead |
| RSK-005 | Embedded database contention or corruption limits throughput or availability | Medium | Medium | ASM-001, ASM-002; OPS-011, OPS-012, OPS-020; DEC-008 | Operations lead |
| RSK-006 | Backups exist but are not restorable or are incomplete | Medium | High | OPS-017 to OPS-021; QA-028; drill records | Operations lead |
| RSK-007 | Operator misuse or compromise exposes network data in hosted mode | Medium | High | SEC-013; DEC-014; SEC-019; GOV-024 | Security reviewer |
| RSK-008 | Administrator misuse (self-dealing, removal of dissenters, unilateral rule changes) | Medium | High | SEC-010 to SEC-012; GOV-001 to GOV-003; GOV-016 | Product architect |
| RSK-009 | Credential stuffing or account takeover | High | Medium | SEC-001, SEC-006, SEC-007, SEC-008 | Security reviewer |
| RSK-010 | Legal classification of network credits, negative balances, write-offs, departure settlement, closure, or consumer and data-protection obligations is adverse to the product's design | Unknown | High | Counsel review before launch (see section 4); product stays technology-neutral; SEC-014, SEC-018, GOV-018 | Product architect with counsel |
| RSK-011 | Erasure obligations conflict with append-only ledger retention | Medium | High | SEC-018 pseudonymisation; DEC-020; counsel review | Security reviewer with counsel |
| RSK-012 | Users treat credits as money, or a network is marketed as a payment or investment product, causing harm or reputational damage | Medium | High | Product principle in [01](01-vision-scope.md); credit unit definition in [00](00-conventions.md); terms reviewed by counsel | Product architect |
| RSK-013 | Supply-chain compromise of a dependency or release artifact | Low | High | SEC-029; OPS-035; DEC-027 deferred | Engineering lead |
| RSK-014 | Accessibility regressions reach release because automated checks cover only part of the requirements | Medium | Medium | QA-033 to QA-036; DEC-024; manual review QA-035 | Interface lead |
| RSK-015 | Scope creep into deferred capabilities before foundations are complete | High | Medium | Section 5 deferral list; sub-specification rule in section 6 | Product architect |
| RSK-016 | Licence undecided blocks public release and external contribution | High | Medium | DEC-004; repository ships without LICENSE until decided | Product owner |
| RSK-017 | Security or governance claims made in marketing exceed what is verified | Medium | High | Scope statement in [09](09-security-privacy.md) (no unverified anonymity, cryptographic or compliance claims); traceability and release record (QA-042) | Product owner |
| RSK-018 | Key loss for backup encryption or session secrets makes backups unusable | Low | High | OPS-009, OPS-010; key storage separation (SEC-028) | Operations lead |

## 4. Legal questions reserved for counsel

These questions are unresolved. This specification does not answer them, does not give legal advice, and does not assert that any design is compliant. The product behaviour described in the referenced requirements MAY need review. No question here blocks the technology-neutral requirements, but each MUST be answered before launch in a given jurisdiction. Jurisdictions to be considered are not yet listed; the owner decides them.

| ID | Question | Product behaviour under review | Decision needed before launch |
|---|---|---|---|
| RSK-019 | Whether credits in a network can be characterised as money, a payment instrument, a security or a financial product in any relevant jurisdiction | Glossary definition; ASM-006; whole product | Launch and marketing positioning |
| RSK-020 | Whether negative balances, write-offs, departure settlement or closure with unsettled balances create obligations, tax consequences or consumer-protection duties | GOV-010, GOV-018, GOV-025; DEC-010 | Network rules template |
| RSK-021 | Whether a hosted operator's role triggers money-services, payment or licensing requirements, and what the product must display if so | Hosted deployment; SEC-013; GOV-024 | Hosted launch |
| RSK-022 | Data-protection obligations for personal data collected, including lawful basis, notices, rights of access, rectification and erasure, international transfers, and processor roles for hosted operators | SEC-014, SEC-016, SEC-017, SEC-018; DEC-020 | Hosted launch |
| RSK-023 | Whether the append-only ledger and retention of ledger records conflict with erasure requests, and how to reconcile them | SEC-018; DEC-020; LED-2 | Hosted launch |
| RSK-024 | Tax treatment of trade and of credit balances for members, networks and operators, by jurisdiction | Whole product; reporting exports (GOV-022) | Launch in each jurisdiction |
| RSK-025 | Effect of dispute outcomes, reversals and adjustments on contracts between members | GOV-013 to GOV-016 | Network rules template |
| RSK-026 | Breach notification duties to regulators and individuals, and the period they require | SEC-030; DEC-014 | Hosted launch |
| RSK-027 | Whether the product's terms of use, privacy information and network rules templates are adequate for the intended jurisdictions | GOV-009, GOV-026; SEC-014 | Hosted launch |
| RSK-028 | Consumer-protection rules for online platforms and for services offered through networks | Hosted deployment; governance rules | Hosted launch |
| RSK-029 | Lawful requests from authorities: scope of disclosure, notice to members and networks, and what the platform can technically provide | TA and actor "authority with lawful request" in SEC section 2; SEC-013 | Hosted launch |

## 5. Roadmap and dependency order

The MVP foundations below are ordered by dependency. Each step has an exit condition that refers to the verification in [12](12-quality-test-plan.md). Steps are not calendar commitments.

| Step | Foundation | Depends on | Exit condition |
|---|---|---|---|
| 0 | Specification baseline (00 to 13) reviewed; DEC-005 to DEC-027 triaged | None | Each open decision has an owner and a "needed before" milestone |
| 1 | Domain model and ledger rules as pure logic (entries, balances, limits, transaction states) | Step 0 | QA-001 to QA-006 pass in the short run |
| 2 | Storage with transactional posting, idempotency records and audit atomicity | Step 1 | QA-006, QA-009, QA-010 pass; OPS-011 and OPS-012 verified |
| 3 | Authentication, sessions, CSRF, rate limiting, uniform responses | Step 2 for storage | SEC-001 to SEC-006, SEC-022 to SEC-024 pass |
| 4 | Authorization model and matrix | Step 3 | QA-012 passes with zero mismatches |
| 5 | Networks, administrators, membership (invite and approval), onboarding terms | Steps 3 and 4 | QA-015 membership tests and QA-017 pass |
| 6 | Proposals, acceptance, completion, decline, cancel, expiry | Steps 1, 2, 5 | QA-018, QA-019, QA-020 pass; QA-007 passes |
| 7 | Balances and history views; interface and API parity | Step 6 | QA-036 and QA-034 pass; contract tests pass |
| 8 | Disputes, reversals, adjustments, dual control | Steps 6 and 7 | QA-013, QA-014, QA-016, QA-021 pass |
| 9 | Exports and reconciliation | Steps 6 and 7 | QA-030 to QA-032, QA-025 pass |
| 10 | Monitoring, health and metrics, backup and restore, migrations | Steps 2 and 9 | QA-027, QA-028, QA-024, QA-026 pass; drill recorded |
| 11 | Packaging for all DEC-003 targets and smoke tests | Steps 0 to 10 | OPS-003 and OPS-035 verified |
| 12 | Hardening: independent security review, accessibility review, performance measurement, release candidate | Step 11 | QA-039 release gate passes |
| 13 | Limited pilot with a small number of networks | Step 12 and counsel review for the pilot jurisdiction | Pilot review records; decisions DEC-010, DEC-014, DEC-020 confirmed |

## 6. Deferred sub-specifications

The following are deferred. Each requires its own specification, decision record and verification plan before work starts. None is created by this pass.

- Mobile applications and native clients
- Federation between networks and cross-network trade
- Tokenisation or any representation of credits as transferable instruments
- Conversion to or from national currency, or payment processing
- Reputation scoring
- Advanced analytics and fraud detection beyond SEC-026
- Third-party integrations, including accounting exports beyond GOV-022
- Dispute evidence uploads (GOV-017)
- Member voting on policy (GOV-006, DEC-021)
- Tamper-evident audit chain (SEC-034, DEC-015)
- Alternative databases and multi-node operation (OPS-005)
- Code signing and notarisation (DEC-027)
- Internationalisation beyond the initial language decided in 08
- Hosted billing and paid administration features (outside the scope of this pass)

## 7. Document governance for this log

- New DEC entries are added at the end of the register and are never renumbered.
- A DEC decision that changes a requirement triggers an update of the requirement's document and its last-updated date in the same change.
- Each release record (QA-039) lists the DEC entries accepted, changed or superseded since the previous release.
- Counsel review items (section 4) are closed only by a recorded note from counsel or the owner's documented acceptance of residual risk. Neither is recorded in this draft.

## 8. Open questions for the owner

1. Confirm or replace each recommended default in section 1.2, starting with DEC-006, DEC-007, DEC-008, DEC-009 and DEC-014, which block the authentication and hosted-launch work.
2. Decide the license (DEC-004) before the repository is published or accepts outside contributions.
3. Decide the jurisdictions for pilot and hosted launch so that section 4 can be scoped.
4. Confirm whether DEC-011 (sign-up grants) is part of the MVP at all.
5. Confirm the decision method for policy changes (DEC-021).
6. Confirm the vulnerability disclosure contact (DEC-026).
7. Confirm whether [01](01-vision-scope.md) through [08](08-ux-accessibility.md) identifiers are bound to the cross-references used in 09 to 13 at integration. This document refers to them by file and prefix only where no exact ID exists yet.

## 9. Owner decisions still open after the build

These owner decisions were raised or confirmed by the build and have no decision yet. Each keeps its recommended default; a default is not a decision. Full options, context and the other open items (DEC-005, 006, 008, 010 to 016, 019 to 026, DEC-072 to 079) are in [DECISIONS.md](../DECISIONS.md), sections 2 and 3.

| ID | Question | Status after build | Recommended default |
|---|---|---|---|
| DEC-004 | Product licence | open; no LICENSE file in the repository | AGPL-3.0 or MIT (00 section 6). Needed before the repository is published or takes outside contributions (RSK-016). |
| DEC-007 | Mandatory second factor for privileged roles | open; no second factor built | Mandatory for network administrators and platform operators; optional for members and reviewers. Blocks the authentication build (SEC-007). |
| DEC-009 | Recovery channel and mail dependency | open; no recovery flow and no mail settings built | Verified email for recovery; administrator-assisted recovery with dual control when no contact exists. Blocks SEC-008. |
| DEC-014 | Platform operator break-glass controls and notice period | open; no break-glass path built (operator has no data role in the product) | Two-person approval; audit readable by affected administrators; access expires after 4 hours; notice within 24 hours where legally permitted (counsel review). |
| DEC-072 | Who may decide a dispute: reviewer role only, or administrators too | open; build allows an administrator without the reviewer role to decide (LIGHT-QA BUG-3) | (a): require the reviewer role, and allow an administrator to hold it; a party or conflicted person never decides. |
| DEC-074 | Should an accepted-but-not-posted state ever be visible | open; build never persists or shows `accepted` (DEC-036) | (a) keep `accepted` internal, as built, until a two-phase flow is decided (03 DOM-014, 08 UX-016) |

Resolved questions (OQ-*) were moved out of the open lists in their specs and into [DECISIONS.md](../DECISIONS.md) section 4, with the as-built answer.
