# 00 — Specification Conventions and Shared Canon

| Field | Value |
|---|---|
| Status | Draft (baseline) |
| Owner role | Product architect |
| Last updated | 2026-10-21 |
| Scope | How all Circula specs are written, identified, linked and kept consistent; the shared canon (names, states, signs, baseline defaults) every other spec MUST follow |
| Depends on | — |
| Open questions | See [13 — Assumptions, risks, decisions](13-assumptions-risks-decisions-roadmap.md) |

## 1. Normative language

The key words **MUST**, **MUST NOT**, **SHOULD**, **SHOULD NOT** and **MAY** are used as in [RFC 2119](https://www.rfc-editor.org/rfc/rfc2119) / [RFC 8174](https://www.rfc-editor.org/rfc/rfc8174) when (and only when) written in bold capitals.

- **MUST / MUST NOT** — absolute requirement; an implementation that violates it is non-conformant. Release-blocking.
- **SHOULD / SHOULD NOT** — strong default; deviations need a recorded justification (decision log entry).
- **MAY** — optional; permitted without justification.

Lower-case "must/should/may" carry no normative weight.

## 2. Requirement identifiers

- Format: `PREFIX-NNN` (three digits, zero-padded), e.g. `SEC-014`. Functional requirements add an area code: `FR-TRD-003`.
- IDs are **stable**: never renumber or reuse. A removed requirement keeps its ID with status `Withdrawn`.
- Each prefix is owned by exactly one document (table below). Other documents **cite** IDs; they do not redefine them.
- Cite as `[DOM-012]` in prose, with a relative link to the owning document on first mention in a section.
- Each requirement row/bullet states: ID, priority, statement (with normative keyword), rationale (one line), verification (test/inspection/analysis) where the owning doc is a requirements doc.
- Priorities: **P0** = MVP launch-blocking; **P1** = MVP, may slip one release with a recorded decision; **P2** = post-MVP (deferred but designed-for).

### 2.1 Prefix registry (owner document)

| Prefix | Meaning | Owner document |
|---|---|---|
| `VIS` | Vision, goals, non-goals, success measures | [01](01-product-vision-and-scope.md) |
| `PER` | Personas and roles/permissions matrix | [02](02-personas-roles-and-journeys.md) |
| `JRN` | User journeys | [02](02-personas-roles-and-journeys.md) |
| `DOM` | Domain definitions and rules | [03](03-domain-model-and-mutual-credit-rules.md) |
| `INV` | Platform-wide invariants (seeded in §5.4 below, elaborated in 03) | [03](03-domain-model-and-mutual-credit-rules.md) |
| `FR-<AREA>` | Functional requirements. Areas: `NET` network setup, `MEM` membership, `LST` listings/directory, `TRD` trades, `BAL` balances/history, `ADM` admin controls, `DSP` disputes, `AUD` audit, `EXP` exports, `NTF` notifications | [04](04-functional-requirements.md) |
| `ARC` | Architecture, trust boundaries, quality attributes | [05](05-architecture-and-trust-boundaries.md) |
| `DAT` | Data model, ledger, audit data | [06](06-data-ledger-and-audit.md) |
| `API` | API and integration contract | [07](07-api-and-integration-contract.md) |
| `UX` | UX and accessibility | [08](08-ux-and-accessibility.md) |
| `SEC` | Security, privacy, abuse resistance | [09](09-security-privacy-and-abuse.md) |
| `GOV` | Governance and network operations | [10](10-governance-and-network-operations.md) |
| `OPS` | Deployment, operations, support | [11](11-deployment-operations-and-support.md) |
| `QA` | Quality strategy, gates, acceptance scenarios (`QA-E2E-nn` for scenarios) | [12](12-quality-strategy-and-test-plan.md) |
| `ASM` / `RSK` / `DEC` | Assumptions / risks / decisions (`DEC-nnn`) | [13](13-assumptions-risks-decisions-roadmap.md) |
| `GLO` | Glossary terms (no numbering; alphabetical) | [14](14-glossary.md) |
| `LIF` | Lifecycle state machines | [15](15-lifecycle-state-machines.md) |
| `POL` | Network policy parameters (catalog) | [16](16-network-policy-catalog.md) |
| `NTF` | Notification and communication requirements | [17](17-notifications-and-communications.md) |
| `PRT` | Data export, portability, import | [18](18-data-export-and-portability.md) |
| `LEG` | Legal/regulatory questions for counsel (`LEG-Q-nn`) | [19](19-legal-and-regulatory-open-questions.md) |

### 2.2 Cross-reference rules

1. One authoritative home per rule. Elsewhere: cite the ID and, at most, paraphrase in one clause.
2. Only cite IDs that exist. The traceability check (`plans/traceability` (document to be created under `docs/plans/`)) fails on dangling references.
3. Relative links only. Filenames are lowercase kebab-case.
4. Sources for external facts: URL **and** access date in the owning document; long research lives in [`docs/notes/`](../notes/README.md).

## 3. Document template

Every spec begins with an H1 and this front-matter table, then sections in this order (omit a section only if truly N/A and say so):

```
# NN — Title
| Field | Value |  Status (Draft/Review/Baseline) · Owner role · Last updated (YYYY-MM-DD) · Scope · Depends on · Open questions
## 1. Purpose and scope
## 2. Definitions used here (link to glossary; no redefinitions)
## 3+. Body (requirements tables / rules / diagrams)
## n-1. Assumptions and open decisions (cite ASM-/DEC- IDs)
## n. References (internal links; external URL + access date)
```

Status meanings: **Draft** — written, unreviewed; **Review** — cross-checked, awaiting owner sign-off; **Baseline** — owner approved, changes via decision log.

## 4. Writing rules

- Specify observable behaviour. Replace "secure", "fast", "user-friendly" with measurable statements or mark the threshold `[DECISION NEEDED: DEC-nnn]`.
- Technology-neutral unless repo evidence or a Baseline decision commits to a technology. No database vendor, cloud vendor or framework is committed in the MVP specs.
- Uncertain choices are never silent requirements: record as `ASM-`/`DEC-` and reference.
- Legal/tax/regulatory matters are never asserted; route to [19](19-legal-and-regulatory-open-questions.md).
- Never imply the credit unit is redeemable for national currency, a security, a cryptocurrency, or an investment.
- Diagrams: Mermaid in fenced blocks (renders on GitHub). Keep them small and also described in text.
- Dates: ISO 8601. Times in UTC in data; localised in UI.

## 5. Shared canon (binding on every spec)

The names, signs, states and defaults below are the single source for terminology used across all specs. Definitions are elaborated in [03](03-domain-model-and-mutual-credit-rules.md), [15](15-lifecycle-state-machines.md), [16](16-network-policy-catalog.md) and [14](14-glossary.md); if those documents need to change something here, change **this section first** and record a decision.

### 5.1 Vocabulary (exact spellings)

| Term | Meaning (short) |
|---|---|
| **Platform** | The Circula software plus the party operating a given deployment (**Platform Operator**). |
| **Network** | One independent mutual-credit community with its own unit, members, rules and ledger. Networks are isolated tenants; no cross-network postings in MVP. |
| **Credit unit** (a.k.a. "unit") | The Network's accounting unit. Name/symbol set per Network (e.g. "hours", "seeds"). Not money, not a security, not crypto, not redeemable for national currency by Platform design. |
| **Member** | A person (or, post-MVP, organisation) with a membership in a Network. A **Member Account** is the ledger account owned by the membership. One person in two Networks has two memberships/accounts. |
| **Network Administrator** | Member-role with rule-configuration and admin powers in one Network (`network_admin`). |
| **Dispute Reviewer** | Member-role that can review disputes in one Network (`dispute_reviewer`). Cannot review a dispute they are party to. |
| **Platform Operator** | Staff of the deployment operator (`platform_operator`); operates infrastructure; has **no** ledger-posting powers and no default access to Network content. |
| **Offer / Request** | A **Listing** advertising goods/services provided (Offer) or sought (Request). Informational only; has no ledger effect. |
| **Trade** | A proposed or completed exchange of value between exactly two Member Accounts. User-facing word "Trade"; domain object `Transaction` of type `trade`. |
| **Provider / Recipient** | In a Trade: Provider supplies goods/services and is **credited**; Recipient receives them and is **debited**. |
| **Proposer / Counterparty** | Member who creates a Trade proposal / the other Member, who must accept. |
| **Journal Entry / Posting** | Ledger double-entry record / one signed line within it. |
| **System Account** | Per-Network non-member account(s) that balance adjustments and openings (`network_clearing`). |
| **Adjustment** | Admin-initiated correcting Journal Entry with mandatory reason, linked audit event, and member notification. |
| **Reversal** | A new Journal Entry that negates a prior completed Journal Entry; never an edit or delete. |
| **Dispute** | A formal challenge to a completed or proposed Trade, reviewed under Network rules. |
| **Audit Event** | Append-only record of a security- or governance-relevant action. Distinct from ledger Journal Entries. |

### 5.2 Amount and balance semantics

- Amounts are **integers in minor units** of the Network's credit unit; each Network fixes `unit_scale` (decimal places, 0–4, default 2; immutable after first posting). No floating point anywhere (UI input is parsed as decimal text → integer).
- Every posted amount is **strictly positive** on a Trade; Postings carry signed minor-unit amounts.
- `balance(account) = Σ posting amounts`. **Positive balance** = Member has net *provided* more than received (a claim on the Network's goods/services, not on money). **Negative balance** = Member has net *received* more than provided (an obligation to provide goods/services, not a debt in national currency).
- Trade completion posts **−amount** to Recipient's account and **+amount** to Provider's account in one Journal Entry.
- `credit_limit` (≥ 0, minor units): the magnitude of negative balance permitted. Floor = `−credit_limit`. Optional `max_balance` (positive cap; null = none).
- `available_to_spend = balance + credit_limit` (never shown below 0; informational pending Trades do **not** reduce it in MVP — no holds, see DEC baseline D-05).
- A posting that would take any Member Account below its floor or above its `max_balance` MUST be rejected atomically (INV-005, INV-006). System Accounts are exempt from limits.

### 5.3 Lifecycle state names (exact)

- **Trade** (`LIF`, owner [15](15-lifecycle-state-machines.md)): `proposed` → `completed` | `declined` | `cancelled` | `expired`; `completed` → `reversed` (terminal; reflects a linked Reversal entry). Acceptance and posting are one atomic step: there is **no** separate persisted `accepted` state. `disputed` is **not** a Trade state; a Trade has zero or more linked Disputes.
- **Dispute**: `open` → `under_review` → `resolved_upheld` | `resolved_rejected` | `resolved_settled`; `open|under_review` → `withdrawn`; `open|under_review` → `escalated` (to Network Administrator/outside process, GOV-defined).
- **Membership**: `applicant` → `active` | `rejected`; `active` ↔ `suspended`; `active|suspended` → `departing` → `closed`; `banned` is a flag-with-reason on `closed`/`suspended`, not a separate state.
- **Listing**: `draft` → `published` ↔ `paused`; `published|paused` → `archived`; admin `removed` (moderation) is terminal.
- **Proposal origin**: a Trade proposal is created by either the Provider (`initiated_by = provider`, "I provided X, please accept a charge of N") or the Recipient (`initiated_by = recipient`, "I received X, I offer to pay N"). In both cases the *other* party accepts; the Recipient's consent (by proposing or by accepting) is what authorises the debit, and the Provider's consent (by proposing or accepting) is required for the credit (INV-012).

### 5.4 Seeded platform invariants (elaborated in 03; IDs frozen)

| ID | Invariant |
|---|---|
| INV-001 | Every Journal Entry's Postings sum to exactly zero. |
| INV-002 | Both sides of a Trade are posted atomically; a partial Trade posting MUST NOT be observable or persistable. |
| INV-003 | Ledger records are append-only; no UPDATE/DELETE of Journal Entries or Postings in normal operation. Exactly three audited exceptions exist: disaster recovery, versioned migration, and ledger disposal at Network closure or retention expiry (5.8 item 1). |
| INV-004 | Corrections are made only by Reversals or Adjustments, each a new Journal Entry linked to the original and to an Audit Event. |
| INV-005 | A Member Account balance MUST NOT go below `−credit_limit` as the result of a Trade posting. |
| INV-006 | A Member Account balance MUST NOT exceed `max_balance` (if set) as the result of a Trade posting. |
| INV-007 | Per Network, the sum of all balances (Member + System Accounts) is exactly zero. |
| INV-008 | Stored/cached balances equal the sum of Postings and can be recomputed from the ledger alone. |
| INV-009 | A retried request with the same idempotency key and payload yields the same outcome and never a second posting; same key with different payload is rejected. |
| INV-010 | Every Adjustment, Reversal, policy change, role change and dispute resolution produces an Audit Event naming actor, time, reason and target. |
| INV-011 | No cross-Network postings; Journal Entries reference accounts of exactly one Network. |
| INV-012 | The Platform Operator and Network Administrators cannot create a Trade on behalf of a Member without that Member's acceptance; admin value changes are Adjustments only, visible to the affected Member. |
| INV-013 | `unit_scale` and the Network's unit identity are immutable once any Journal Entry exists. |
| INV-014 | Money-like claims are prohibited in the product: the Platform exposes no redemption, cash-out, interest, or fiat conversion in MVP. |
| INV-015 | A Trade is valid only if Provider and Recipient are distinct Member Accounts of the same Network, both Memberships are `active` at posting time (exception: `departing` Memberships under 03 DOM-035), and the amount is an integer ≥ 1 minor unit within `min_trade_amount`/`max_trade_amount` where set. |
| INV-016 | After a Membership is `closed`, its Member Account balance is exactly 0 and no further Journal Entry references it. |

### 5.5 Baseline defaults (to be registered as `DEC-` entries with status *Proposed default* in doc 13)

| Ref | Baseline default | Alternative(s) |
|---|---|---|
| D-01 | **Deployment**: single codebase supporting both Platform-Operator-hosted multi-tenant and self-hosted single-Network or multi-Network installs; hosted subscriptions out of MVP scope. | Hosted-only; self-host-only. |
| D-02 | **Technology-neutral** specs; reference stack chosen later via ADR (see plans). | Commit a stack now. |
| D-03 | **Identifiers**: opaque, time-sortable, non-guessable (UUIDv7 recommended). | UUIDv4; ULID. |
| D-04 | **Ledger model**: double-entry Journal Entry + Postings; balances derived from postings with transactionally-maintained cached balance per account. | Event-sourced only; balance-only. |
| D-05 | **No funds holds/reservations** in MVP; limits enforced at posting time; pending Trades informational. | Reserve on proposal. |
| D-06 | **Member-initiated Trades require counterparty acceptance**; no unilateral debit by any role. | Pre-authorised standing orders (deferred). |
| D-07 | **Identity**: email-based account + passkey/TOTP-capable authentication; one login can hold memberships in multiple Networks. | Per-Network accounts only. |
| D-08 | **Data controller split**: Network (via its admins) is controller for member data it collects; Platform Operator acts as processor for hosted; **needs counsel** (LEG). | Joint controllers. |
| D-09 | **Erasure vs ledger**: personal data anonymised/pseudonymised on closure; ledger rows keep opaque account IDs only. | Retain identity for fixed period. |
| D-10 | **Network participation default**: invite or admin-approval; open sign-up optional per Network. | Open-only. |
| D-11 | **Policy changes** take effect prospectively after notice period (default 14 days) except safety fixes. | Immediate. |
| D-12 | **Disputes do not freeze balances** in MVP; remedy is Reversal/Adjustment after resolution. | Freeze counterparty funds. |
| D-13 | **Localisation**: UI i18n-ready (externalised strings, locale-aware number/date); MVP ships English only. | Multi-language MVP. |
| D-14 | **Accessibility target**: WCAG 2.2 Level AA. | AAA partial. |

### 5.6 Network policy parameter names (exact; catalog and defaults in [16](16-network-policy-catalog.md))

`unit_name`, `unit_symbol`, `unit_scale`, `membership_mode` (`invite` | `approval` | `open`), `default_credit_limit`, `max_credit_limit`, `max_balance`, `min_trade_amount`, `max_trade_amount`, `proposal_expiry_days`, `dispute_window_days`, `dispute_review_sla_days`, `balance_visibility` (`self` | `admins` | `members`), `directory_visibility` (`members` | `public_listings`), `inactivity_threshold_days`, `departure_settlement_mode` (`settle` | `write_off` | `community_fund`), `policy_notice_days`, `admin_quorum` (number of admins required for sensitive actions; default 1).

### 5.7 Document set

Core: 00–13 as required by the foundational prompt; supporting: 14 glossary, 15 lifecycle state machines, 16 policy catalog, 17 notifications, 18 export/portability, 19 legal questions. Plans and notes live in `docs/plans/` and `docs/notes/`; ADRs in `docs/decisions/`.

### 5.8 Canon amendments (2026-10-21, post-draft reconciliation)

**Status: applied (2026-10-21).** Adopted from the first-draft review; these bind all specs and supersede conflicting wording above. Sections 5.3 to 5.6 and the INV table in 5.4 now reflect items 1 to 3 and 6; docs 03, 06, 12, 13 and 15 follow them. Doc 12 ([12](12-quality-strategy-and-test-plan.md)) exists and owns the `QA` prefix; registry links resolve.

1. **Invariants added** (owned by doc 03): `INV-015` Trade validity (distinct active parties in the same Network, positive amount within `min_trade_amount`/`max_trade_amount`, both memberships `active`); `INV-016` a `closed` Member Account holds a zero balance. `INV-003` permits exactly three exceptions: disaster recovery, versioned migration, and ledger disposal at Network closure/retention expiry (all audited).
2. **System Accounts**: `network_clearing` (adjustments/openings) and `community_fund` (used only when `departure_settlement_mode = community_fund` or Network-defined fund). Both exempt from limits.
3. **Transaction types**: `trade`, `reversal`, `adjustment`, `opening_balance`, `departure_settlement`. Only `trade` requires counterparty acceptance (INV-012 applies to `trade`; the others are admin-initiated, visible to the affected Member, and audited).
4. **Adjustment/Reversal statuses** are defined in doc 15 (`pending`, `posted`, `refused`, `withdrawn`, `expired`); Network operating status, Appeal and policy-version statuses are defined in docs 10/16.
5. **Unit identity** (`unit_name` incl. singular/plural forms, `unit_symbol`, `unit_scale`) is stored on the Network record, not in versioned policy (INV-013); doc 16 documents them as "Network identity parameters".
6. **Admin visibility**: Network Administrators can see all ledger entries of their Network irrespective of `balance_visibility` (which governs Member-to-Member visibility).
7. **Proposal wording**: UI copy must not use "pay/price/charge" (see glossary banned terms); canon 5.3's wording is descriptive only.
8. **Local open-decision labels** (`VIS-OD-n`, `PER-OD-n`, `FR-OD-n`, `ARC-OD-n`, `API-OD-n`, `UX-OD-n`, `SEC-OD-n`, `OPS-OD-n`, `NTF-OD-n`, …) are registered in doc 13 as `DEC-nnn` and keep their local label as an alias.

## 6. Assumptions and open decisions

- ASM/DEC entries for the canon above are registered in doc 13 (Proposed defaults D-01…D-14).

## 7. References

- RFC 2119: <https://www.rfc-editor.org/rfc/rfc2119> (accessed 2026-10-21)
- RFC 8174: <https://www.rfc-editor.org/rfc/rfc8174> (accessed 2026-10-21)
- Foundational authoring prompt: [`prompts/create-foundational-specs.md`](../../prompts/create-foundational-specs.md)
