# 19 — Legal and Regulatory Open Questions

| Field | Value |
|---|---|
| Status | Draft |
| Owner role | Product architect (with the Platform Operator lead as counsel liaison) |
| Last updated | 2026-10-21 |
| Scope | Register of questions for qualified counsel (`LEG-Q-nn`); design guardrails that keep the product inside the facts those questions assume (`LEG-G-nn`); launch gating; counsel engagement checklist; unverified-claim register |
| Depends on | [00 — Conventions](00-conventions.md) (canon INV-014, D-08, D-09), [03 — Domain model](03-domain-model-and-mutual-credit-rules.md), [13 — Assumptions, risks, decisions](13-assumptions-risks-decisions-roadmap.md) |
| Open questions | All `LEG-Q-nn` rows in §3 (status Open) |

## 1. Purpose and scope

1.1 This document is **not legal advice** and asserts no legal, tax or regulatory conclusion. It lists the questions that qualified counsel in each relevant jurisdiction needs to answer, records which product behaviours trigger each question, and defines testable product constraints (guardrails) that keep the product within the facts the questions assume. Per [00 §4](00-conventions.md), legal/tax/regulatory matters are routed here rather than asserted elsewhere.

1.2 The questions and guardrails derive from the research note [research-regulatory-privacy-questions](../notes/research-regulatory-privacy-questions.md) (accessed 2026-10-21). Most statutory references in that note are `[U]` (unverified, written from background knowledge). This document keeps those flags (§9). Nothing here may be cited as evidence that Circula complies with, or is exempt from, any law.

1.3 Normative keywords in §5 apply to product behaviour only. They are not statements about legal status. "Resolved" means counsel has answered in writing and the answer is recorded as a `DEC-` entry in [13](13-assumptions-risks-decisions-roadmap.md); until then every status is **Open**.

## 2. Definitions used here

Terms (Network, Credit unit, Member, Network Administrator, Platform Operator, Trade, Adjustment, Reversal, Journal Entry) are as in [00 §5.1](00-conventions.md) and the glossary [14](14-glossary.md). Additional shorthand:

| Term | Meaning |
|---|---|
| Gate | Release milestone before which a question needs a recorded answer: **H** = first hosted launch; **S** = first self-host distribution; **P** = first paid subscription (hosted subscriptions are out of MVP scope per D-01) |
| Launch-blocking | **Y** = gate cannot be passed without a counsel answer; **N** = may ship with a recorded risk acceptance; **by-model** = depends on the answer to a deployment/product choice stated in the row |
| Counsel | Qualified legal adviser in the named jurisdiction; Circula authors are not counsel |

## 3. Question register

Source short names (IRS-1, FIN-1, etc.) are listed in §10. Canon links: INV-014 (no money-like claims), D-01 (deployment model), D-08 (controller split), D-09 (erasure vs ledger), D-06/INV-012 (consent to debit), INV-003/INV-004 (append-only, corrections), D-12 (disputes do not freeze balances), D-14 (WCAG 2.2 AA), D-10 (participation mode).

### 3.1 Tax and reporting

| ID | Topic | Trigger behaviour | Jurisdictions | Launch-blocking | Owner role | Gate | Status | Canon links |
|---|---|---|---|---|---|---|---|---|
| LEG-Q-01 | Is a Network or the Platform Operator a "barter exchange" (US Form 1099-B; TIN collection)? | Admin-run ledger of Member-to-Member Trades, esp. business Members | US federal | by-model (Y if hosted and US business Members are admitted) | Platform Operator lead | H | Open | D-01, INV-012 |
| LEG-Q-02 | Member income tax on units earned; wording of tax disclaimers; whether to show any currency valuation | Members earning units for services | US, UK, CA, AU, NZ, EU | N (disclaimer wording needed) | Product architect | H | Open | INV-014, LEG-G-14 |
| LEG-Q-03 | VAT/GST on Trades in units; invoice/tax-code fields | Business Members trading in units | UK, EU, CA, AU, NZ | N | Product architect | H | Open | INV-014 |
| LEG-Q-04 | UK treatment of LETS (no authoritative HMRC position found) | Any UK Network | UK | N | Platform Operator lead | H | Open | — |
| LEG-Q-05 | DAC7 / platform seller reporting | Hosted platform facilitating Member sales | EU, UK equivalent, OECD | by-model (Y if hosted in or serving EU) | Platform Operator lead | H | Open | D-01, D-08 |

### 3.2 Payments, money, credit, securities

| ID | Topic | Trigger behaviour | Jurisdictions | Launch-blocking | Owner role | Gate | Status | Canon links |
|---|---|---|---|---|---|---|---|---|
| LEG-Q-06 | FinCEN money transmitter / prepaid access | Any redemption, cash-in, or operator holding of funds | US federal | Y | Platform Operator lead | H | Open | INV-014 |
| LEG-Q-07 | State money transmission / stored value laws; state matrix; geofencing | Fees or funds through operator; cash-in/out | US states (NY, CA, TX, FL first) | by-model (Y if any funds flow; else confirm and record) | Platform Operator lead | H | Open | INV-014, D-01 |
| LEG-Q-08 | Do credit limits amount to lending? Enforceability of negative balances in currency | Negative balances within `credit_limit`; settlement on exit | US, UK, EU, CA, AU, NZ | Y | Product architect | H | Open | INV-005, INV-014, `departure_settlement_mode` |
| LEG-Q-09 | Deposit-taking: positive balances vs "deposits" | Positive balances treated as owed money | UK, EU, US, AU | Y | Product architect | H | Open | INV-014 |
| LEG-Q-10 | PSD2 limited-network exclusion / e-money; EUR 1m notification trigger | EU Networks with many Members; any link to funds | EU member states | by-model (Y if EU Networks hosted) | Platform Operator lead | H | Open | INV-014, D-01 |
| LEG-Q-11 | PSD3 / Payment Services Regulation status and limited-network treatment | As LEG-Q-10 | EU | N (monitor; confirm at launch) | Platform Operator lead | H | Open | — |
| LEG-Q-12 | UK PSRs 2017 / EMRs 2011 limited-network exclusion; FCA stance | UK Networks | UK | by-model (Y if UK Networks hosted) | Platform Operator lead | H | Open | INV-014 |
| LEG-Q-13 | Canada: RPAA / FINTRAC | Any funds flow; "virtual currency" definition | Canada | by-model (geofence if unanswered) | Platform Operator lead | H | Open | INV-014 |
| LEG-Q-14 | Australia: AUSTRAC, ASIC non-cash payment facility, credit, ATO barter reporting | Any funds flow; credit | Australia | by-model (geofence if unanswered) | Platform Operator lead | H | Open | INV-014 |
| LEG-Q-15 | New Zealand: FSP registration, AML/CFT, CCCFA | "Payment system" or value-transfer service | New Zealand | by-model (geofence if unanswered) | Platform Operator lead | H | Open | INV-014 |
| LEG-Q-16 | Fees in national currency (per-trade or deducted from balances) | Fiat fees paid to admins/operator | All | Y (MVP stance: no fiat fees on Trades; confirm) | Product architect | H | Open | INV-014, LEG-G-04 |
| LEG-Q-17 | Hosted subscription structure: customer (admin vs Member), VAT/sales tax, consumer duties | Admin or Member pays operator in fiat | All | N for H; Y for P | Platform Operator lead | P | Open | D-01 |
| LEG-Q-18 | Securities / investment-contract risk; marketing and copy rules | Marketing, scarcity, appreciation, interest, resale | US, CA, UK, AU, NZ | Y | Product architect | H | Open | INV-014, LEG-G-09 |
| LEG-Q-19 | MiCA / crypto-asset scope if ledger ever uses DLT | Any blockchain anchoring or tokenised representation | EU, UK | N (DLT out of MVP; re-run before any change) | Product architect | H | Open | D-04, LEG-G-10 |

### 3.3 Contracts, consumer protection, governance

| ID | Topic | Trigger behaviour | Jurisdictions | Launch-blocking | Owner role | Gate | Status | Canon links |
|---|---|---|---|---|---|---|---|---|
| LEG-Q-20 | Terms of service; liability allocation operator/admin/Member; unfair-terms review; arbitration | All users | All | Y | Platform Operator lead | H | Open | D-01, D-08 |
| LEG-Q-21 | Marketplace / platform duties (DSA Art. 30, P2B) | Hosted directory of Offers/Requests | EU | by-model (Y if hosted directory serves EU) | Platform Operator lead | H | Open | D-10 |
| LEG-Q-22 | Dispute resolution and reversals: who may amend ledger and on what basis | Admin Reversal/Adjustment; chargeback-like expectations | All | Y | Product architect | H | Open | INV-003, INV-004, INV-010, INV-012, D-12 |
| LEG-Q-23 | Wind-up and exit of balances; debt characterisation; insolvency of a Network entity | Network closing; Member leaving with positive or negative balance | All | Y | Product architect | H | Open | INV-014, `departure_settlement_mode` |

### 3.4 Privacy and data protection

| ID | Topic | Trigger behaviour | Jurisdictions | Launch-blocking | Owner role | Gate | Status | Canon links |
|---|---|---|---|---|---|---|---|---|
| LEG-Q-24 | Controller/processor allocation (operator vs Network admin); DPA; joint-controller risk for cross-Network features | Hosted multi-Network service; any cross-Network directory | EU, UK, others | Y | Platform Operator lead | H | Open | D-08, D-07, INV-011 |
| LEG-Q-25 | Lawful basis and transparency; privacy notice templates | Member profile, ledger visibility, directory | EU, UK | Y | Product architect | H | Open | D-08, `balance_visibility`, `directory_visibility` |
| LEG-Q-26 | DPIA requirement and template | Trust scores, credit-limit logic, large Networks | EU, UK | N | Product architect | H | Open | D-08 |
| LEG-Q-27 | Erasure/rectification vs append-only ledger and audit retention; pseudonymous IDs; anonymisation procedure | Immutable entries; Audit Events | EU, UK, CA, others | Y | Product architect | H | Open | D-09, INV-003, INV-004, INV-010 |
| LEG-Q-28 | Subject access and portability exports exposing counterparty data; redaction rules | Member data export | EU, UK, CA | Y | Product architect | H | Open | D-09, [18](18-data-export-and-portability.md) |
| LEG-Q-29 | CCPA/CPRA and other US state privacy laws | Hosted operator with California (etc.) Members | US states | by-model (depends on thresholds) | Platform Operator lead | H | Open | D-08 |
| LEG-Q-30 | Children and minors; minimum age; guardian sub-accounts | Family/youth accounts in timebank-style Networks | EU, UK, US, others | Y (minimum-age rule) | Product architect | H | Open | D-07, LEG-G-13 |
| LEG-Q-31 | Cross-border transfers; hosting regions; sub-processors | Hosting region; support access | EU, UK, CA, AU, NZ | Y | Platform Operator lead | H | Open | D-01, D-08 |
| LEG-Q-32 | Breach notification and processor duties; contractual notice time to admins | Security incident at operator | EU, UK, US states, CA, AU, NZ | Y | Platform Operator lead | H | Open | D-08, LEG-G-19 |
| LEG-Q-33 | Record retention periods per data category | Ledger, invoices, identity, backups, tombstones | All | Y | Product architect | H | Open | D-09, LEG-G-18 |

### 3.5 Accessibility and screening

| ID | Topic | Trigger behaviour | Jurisdictions | Launch-blocking | Owner role | Gate | Status | Canon links |
|---|---|---|---|---|---|---|---|---|
| LEG-Q-34 | Accessibility law scope (EAA, ADA, Equality Act, others); accessibility statement | Hosted web app; commercial subscription | EU, US, UK, CA, AU | N for S; by-model for H/P (Y if commercial hosted) | Product architect | H | Open | D-14, LEG-G-17 |
| LEG-Q-35 | AML/sanctions screening; geo-restrictions | Onboarding Members across borders | US (OFAC), UK, EU | Y for hosted | Platform Operator lead | H | Open | D-10, LEG-G-16 |

### 3.6 Security research, software licence and copy review (added 2026-10-21)

| ID | Topic | Trigger behaviour | Jurisdictions | Launch-blocking | Owner role | Gate | Status | Canon links |
|---|---|---|---|---|---|---|---|---|
| LEG-Q-36 | Safe-harbour wording for good-faith vulnerability researchers, and any bounty promise: what protection can lawfully be stated, in which jurisdictions, and by whom (operator versus project) (unverified; no jurisdiction researched) | Publishing `SECURITY.md` or a disclosure policy beyond process (SEC-087) | Not yet researched; hosted operator's jurisdiction first | N for process-only policy; Y before any safe-harbour or bounty text | Platform Operator lead | H | Open | SEC-087, SEC-OD-11 |
| LEG-Q-37 | Software licence choice, warranty disclaimer and deployer-responsibility wording (unverified) | Publishing the source and releases (LEG-G-20) | Not yet researched | Y before first public release | Product architect | H | Open | D-01, LEG-G-20 |
| LEG-Q-38 | Counsel review of the credit-unit explanation text and the sensitive-state copy (direction words earned/spent, "owe" wording, negative-balance and limit messages) for money-like or debt-claim implications (unverified) | Onboarding unit explanation; balance, limit and state labels (UX-OD-4, UX-OD-12; INV-014) | Launch jurisdictions (not yet fixed) | Y before launch in any jurisdiction | Product architect | H | Open | INV-014, UX-OD-4, UX-OD-12, LEG-G-01 |

## 4. Prioritisation

Priority is a planning aid for counsel time, not a legal assessment.

### 4.1 Must resolve before first hosted launch (Gate H)

Order of engagement:

1. **Closed-loop posture (money/credit/securities):** LEG-Q-06, 07, 08, 09, 16, 18, 23. These all depend on the facts held fixed by INV-014 and guardrails LEG-G-01..09. Ask counsel to confirm those facts are sufficient, or to name what must change.
2. **Jurisdiction-specific money regimes for launch jurisdictions only:** LEG-Q-10, 12, 13, 14, 15 (and 11 as a status check). A jurisdiction without an answer is geofenced (LEG-G-16), not launched.
3. **Data protection foundation:** LEG-Q-24, 25, 27, 28, 30, 31, 32, 33. These shape architecture (see §6) and must precede data-model freeze in [06](06-data-ledger-and-audit.md).
4. **Contract and governance:** LEG-Q-20, 22.
5. **Screening:** LEG-Q-35.
6. **Disclosure and reporting (may ship with recorded risk acceptance):** LEG-Q-01 to 05, 21, 26, 29, 34.

### 4.2 Must resolve before first self-host distribution (Gate S)

Self-hosting shifts operator duties to deployers but not the effect of software defaults. Before publishing a release for self-hosting:

- LEG-Q-24 (what the software defaults imply; privacy by design), LEG-Q-27, 28, 33 (erasure/export/retention tooling shipped with defaults), LEG-Q-30 (minimum-age default), LEG-Q-18 and LEG-Q-19 (terminology and DLT-out defaults), LEG-Q-02 (disclaimer text in docs), LEG-Q-34 (deployer-facing accessibility notes).
- Software licence and deployer-responsibility wording (LEG-G-20): counsel to review the licence choice and warranty disclaimer. Licence selection is registered as LEG-Q-37.

### 4.3 Must resolve before paid subscriptions (Gate P)

Hosted subscriptions are out of MVP scope (D-01). Before any exist: LEG-Q-17 (structure, customer identity, tax registration), LEG-Q-16 (confirm the subscription is never per Trade and never touches balances), LEG-Q-20 (consumer-protection terms), LEG-Q-05 (reporting), LEG-Q-29 (US state thresholds), LEG-Q-34 (commercial accessibility scope).

## 5. Design guardrails (`LEG-G-nn`)

Guardrails are product constraints that reduce regulatory exposure. They do not remove the need for counsel and do not assert exemption. Each is **P0** (MVP launch-blocking per [00 §2](00-conventions.md)) unless stated, and carries a verification method so it can be tested or inspected. Stage-1 verification applies to the specification set and Stage-2 to the implementation.

| ID | Pri | Guardrail | Verification | Canon / INV / D | Related questions |
|---|---|---|---|---|---|
| LEG-G-01 | P0 | The product **MUST NOT** provide any redemption, cash-out, or conversion of the Credit unit to national currency, in UI, API, admin tooling, exports or documentation. | Inspection of API/UI surface; negative test that no such endpoint, field or flow exists | INV-014 | LEG-Q-06, 07, 09 |
| LEG-G-02 | P0 | The product **MUST NOT** create units in exchange for money. Units arise only from completed Trades, Adjustments with reason (INV-010), and opening entries against the System Account. | Ledger inspection: every posting traces to a Trade, Adjustment or opening entry; no payment-linked posting type | INV-001, INV-004, INV-012, INV-014 | LEG-Q-06, 07, 18 |
| LEG-G-03 | P0 | The product **MUST NOT** compute, accrue or display interest, yield, or holding charges on any balance. | Negative test: no policy parameter or job applies a rate to balances; the parameter catalog [16](16-network-policy-catalog.md) contains none | INV-014 | LEG-Q-08, 09, 18 |
| LEG-G-04 | P0 | The product **MUST NOT** charge fees in national currency per Trade, and **MUST NOT** deduct any fee from, or settle any fee against, a Member Account balance. | Inspection: no fee field on Trade; no posting type for fees; billing data store separate from ledger | INV-014, INV-007, D-01 | LEG-Q-16, 17 |
| LEG-G-05 | P0 | The Platform Operator **MUST NOT** hold or move Member funds; no escrow or pooled-funds feature. Any subscription (post-MVP) **MUST** use a third-party processor and **MUST NOT** store card data. | Architecture review; data inventory shows no card data and no funds balance | INV-014, D-01 | LEG-Q-06, 07, 17 |
| LEG-G-06 | P0 | Units **MUST** be usable only between Members of the same Network; there is no merchant onboarding for non-members and no cross-Network posting. | Test: posting between accounts of different Networks is rejected | INV-011, INV-014 | LEG-Q-10, 12, 13, 14, 15 |
| LEG-G-07 | P0 | Credit limits **MUST** be Network policy (`default_credit_limit`, `max_credit_limit`); the product **MUST NOT** enforce, collect or report negative balances as a debt in national currency, and **MUST NOT** apply a rate to them. | Inspection of exports, notifications and wind-up flows for currency-denominated demand language | INV-005, INV-014 | LEG-Q-08, 23 |
| LEG-G-08 | P0 | User-facing text, API names, schema and code comments **MUST** use the canon vocabulary ([00 §5.1](00-conventions.md)) and **MUST NOT** use "coin", "token", "mint", "stake", "invest", "yield", "airdrop", "loan", "deposit", or bank-sense "wallet". | Automated lint over strings/identifiers/docs with an allow-list for quoted legal terms in this document | INV-014, [00 §4](00-conventions.md) | LEG-Q-18, 19 |
| LEG-G-09 | P0 | The product and its marketing copy **MUST NOT** imply appreciation, resale value or investment return, **MUST NOT** offer pre-sales, and **MUST NOT** support unit-to-unit exchange rates between Networks. | Copy review checklist; test that no exchange-rate data exists | INV-011, INV-014 | LEG-Q-18 |
| LEG-G-10 | P0 | The ledger **MUST** be a conventional append-only store; no distributed-ledger anchoring and no on-chain personal data in MVP. | Architecture decision record; inspection | D-04, D-02, INV-003 | LEG-Q-19, 27 |
| LEG-G-11 | P0 | Ledger rows **MUST** reference accounts by opaque identifier only; personal profile data **MUST** be separable from ledger rows and removable/anonymisable without altering any Journal Entry amounts or balances. | Test: anonymise a Member on closure; INV-007/INV-008 checks still pass; ledger contains no name/contact fields | D-09, D-03, INV-003, INV-008 | LEG-Q-24, 27, 28 |
| LEG-G-12 | P0 | Network creation **MUST** record the controller/processor allocation acknowledgement (DPA acceptance, when counsel supplies the template); a sub-processor list **SHOULD** be published. Platform Operator access to Network content **MUST NOT** be granted by default. | Test: Network cannot be activated in hosted mode without recorded acceptance; role matrix check | D-08, [00 §5.1](00-conventions.md) (Platform Operator) | LEG-Q-24, 31 |
| LEG-G-13 | P0 | Onboarding **MUST** capture a minimum-age confirmation (default 18) and the product **MUST NOT** ship guardian or youth sub-accounts until counsel review closes LEG-Q-30. | UI test; absence of sub-account type | D-07 | LEG-Q-30 |
| LEG-G-14 | P1 | The product **SHOULD** provide per-Member and per-Network transaction exports without computing or asserting tax values, and **SHOULD** show a plain-language notice that Trades may be taxable and that Circula does not provide tax advice. | Export content inspection; UI copy check | INV-014 | LEG-Q-01, 02, 03, 05, 28 |
| LEG-G-15 | P0 | Value changes by an administrator **MUST** be Adjustments only, visible to the affected Member, with reason and Audit Event; no administrator-created Trade without the Member's acceptance. | Test per INV-010 and INV-012 | INV-010, INV-012, D-06 | LEG-Q-22 |
| LEG-G-16 | P1 | The deployment **SHOULD** support disabling hosted onboarding or specific features per jurisdiction by configuration (feature flags), without code forks. | Configuration test: flag off removes the feature for the named jurisdiction | D-01, D-10 | LEG-Q-07, 13, 14, 15, 35 |
| LEG-G-17 | P0 | The UI **MUST** meet WCAG 2.2 Level AA as the engineering target, verified in CI plus manual keyboard/screen-reader tests; an accessibility statement **SHOULD** be published. | Per [08](08-ux-and-accessibility.md) and [12](12-quality-strategy-and-test-plan.md) | D-14 | LEG-Q-34 |
| LEG-G-18 | P1 | Retention periods **SHOULD** be configurable per data category (ledger, identity, support logs, backups, tombstones), with a documented anonymisation procedure for departed Members. | Configuration and runbook inspection; backup-expiry test | D-09 | LEG-Q-27, 33 |
| LEG-G-19 | P1 | An incident procedure **SHOULD** escalate security incidents to affected Network Administrators within 24 hours of the operator becoming aware, so controllers can meet their own clocks. | Runbook inspection; tabletop exercise record in [11](11-deployment-operations-and-support.md) | D-08 | LEG-Q-32 |
| LEG-G-20 | P0 | Licence text and deployment docs **MUST** state that deployers are responsible for local compliance and that the software gives no legal, tax or regulatory assurance. | Documentation inspection | D-01 | LEG-Q-24, 36 (if added) |

Where a guardrail's verification references another document, that document owns the detailed requirement; this document owns only the legal-driven constraint and its traceability.

## 6. Architecture-affecting questions

Two questions can force a change to data design if counsel's answer differs from the baseline defaults. Neither baseline is a legal conclusion.

### 6.1 Controller/processor split (D-08; LEG-Q-24, 25, 26, 29, 31, 32)

- **Baseline (D-08):** the Network, acting through its administrators, is controller for Member data it collects; the Platform Operator is processor for hosted service, and controller for its own account, billing, security and analytics data. Alternative: joint controllers.
- **What changes if counsel advises otherwise:**
  - Joint controllership, especially if cross-Network directory or search is added, would alter tenant isolation (INV-011), data-sharing flows and the Operator's visibility into Network content, and would require a joint arrangement and shared data-subject-request handling.
  - Operator-as-controller for Member data would affect the "no default access to Network content" rule for the Platform Operator ([00 §5.1](00-conventions.md)).
  - A DPA acceptance step at Network creation (LEG-G-12) and published sub-processor list become data-model and onboarding requirements.
- **Design constraint now:** keep Operator-owned data (account, billing, security logs) logically separable from Network-owned data, so either allocation can be implemented. Cross-Network features **MUST NOT** be added to MVP without reopening LEG-Q-24.

### 6.2 Erasure vs append-only ledger (D-09; LEG-Q-27, 28, 33)

- **Baseline (D-09):** on closure, personal data is anonymised or pseudonymised; ledger rows keep opaque account identifiers only. Alternative: retain identity for a fixed period.
- **Tension:** INV-003 (append-only) and INV-004 (corrections via new entries) must hold while personal data can be removed. Whether a pseudonymised record remains personal data (for example if the controller can re-identify) is an open legal question `[U]`.
- **What changes if counsel advises otherwise:**
  - A fixed identity-retention period would require a retention schedule per category, scheduled anonymisation jobs, and restricted-access storage for retained identity data.
  - If counterparty details in exports (LEG-Q-28) need redaction, [18](18-data-export-and-portability.md) must specify redaction rules.
  - Audit Events (INV-010) name actors; counsel must advise how long actor identity may be retained after closure.
- **Design constraint now:** LEG-G-11 (opaque IDs, separable profile data) keeps both outcomes achievable.

### 6.3 Other questions with data-model effect

LEG-Q-30 (minimum age and any guardian account type), LEG-Q-01/05 (possible TIN or seller-reporting fields), LEG-Q-03 (invoice or tax-code fields on Trades): do not add these fields speculatively; add only if counsel requires and record as `DEC-`.

## 7. Counsel engagement checklist

Bring to each engagement. Items are inputs for counsel, not outputs of this document.

**Product behaviours (state precisely, with the canon reference):**
- [ ] The Credit unit has no redemption, cash-out, cash-in, interest or fiat conversion in MVP (INV-014; LEG-G-01..03).
- [ ] Units arise only from Trades between two consenting Members, Adjustments with reason, and System Account opening entries (INV-012; LEG-G-02).
- [ ] Negative balances are permitted within a Network-set `credit_limit`; what happens to residual balances on departure (`departure_settlement_mode`: `settle`, `write_off`, `community_fund`) and on Network wind-up (LEG-Q-08, 23). Counsel to say which modes are acceptable.
- [ ] Fees: none on Trades; any future subscription is billed outside the ledger (LEG-G-04, 05).
- [ ] Disputes do not freeze balances; remedy is Reversal or Adjustment after resolution (D-12).
- [ ] Participation modes (`membership_mode`: `invite`, `approval`, `open`) and Member types (individuals only at MVP; organisations post-MVP).
- [ ] Visibility settings (`balance_visibility`, `directory_visibility`).
- [ ] No distributed ledger (LEG-G-10).

**Deployment and commercial facts:**
- [ ] Deployment models: hosted multi-tenant and self-hosted (D-01); candidate launch jurisdictions; who the contracting parties are (Operator, Network Administrator, Member).
- [ ] Software licence candidate and warranty disclaimer.
- [ ] Planned Network sizes, presence of business Members, expected volume of Trades.

**Data flows (diagram and table):**
- [ ] Data inventory: what personal data is collected, where it is stored, who can read it by role (Member, Network Administrator, Dispute Reviewer, Platform Operator).
- [ ] Hosting regions, sub-processors, support-access paths, backup locations (LEG-Q-31).
- [ ] Ledger vs profile separation and anonymisation-on-closure design (D-09; LEG-G-11).
- [ ] Export contents and counterparty data exposure (LEG-Q-28).
- [ ] Audit Event contents and retention (INV-010).
- [ ] Incident-escalation path (LEG-G-19).

**Documents to prepare for counsel (drafts, not final):** terms of service outline, privacy notice outline, DPA outline, retention schedule skeleton, jurisdiction launch list, this register with the status column.

**Outputs to request from counsel (per question):** written answer, assumptions relied on, jurisdictions covered, what product change would invalidate it, date and currency of law relied on. Record each as a `DEC-` entry and update this register's Status.

## 8. Assumptions and open decisions

- Statuses in §3 are all **Open**; no question is resolved by this document.
- Baselines D-01, D-08, D-09 and others cited above are *Proposed defaults* (see [00 §5.5](00-conventions.md); register in [13](13-assumptions-risks-decisions-roadmap.md)). Specific `ASM-`/`DEC-` numbers for the items below are to be assigned in 13; this document does not invent them:
  - Which launch jurisdictions are in scope for Gate H.
  - Licence selection is `LEG-Q-37`; researcher safe harbour is `LEG-Q-36`; credit-unit and sensitive-state copy review is `LEG-Q-38`.
  - Whether a DPIA template and DPA template ship with the platform (LEG-Q-24, 26).
  - Per-category retention schedule values (LEG-Q-33).

## 9. Unverified claims

The research note's `[U]` flags are retained. Specifically, treat the following as unverified until confirmed against primary sources:

1. **All statutory section numbers and thresholds** marked `[U]` in the note (for example IRC 6045(c)(3), Treas. Reg. 1.6045-1(e), PSD2 Arts. 3(k) and 37(2) details beyond the search summary, GDPR articles, Directive 2006/112/EC Arts. 2 and 73, EAA scope and dates, CCD2 application date) were written from background knowledge. Secondary summaries support only the IRS barter-exchange outline `[V]`, the FinCEN money-transmitter outline `[V]`, and the EBA limited-network guidelines of 24 February 2022 `[V]`.
2. **HMRC position on LETS (LEG-Q-04):** no authoritative guidance found; only forum commentary. The "mutual trading" argument is an unsettled practitioner view.
3. **PSD3 / PSR status (LEG-Q-11):** not found; check current legal state.
4. **Canada, Australia, New Zealand regimes (LEG-Q-13, 14, 15; parts of 02, 03, 08, 31, 32):** entirely `[U]`; not researched with sources.
5. **EDPB Guidelines 02/2025 on blockchain, v2.0 adopted 7 July 2026:** details from search listings and secondary summaries; the PDF was not opened. Verify before relying on it.
6. **EU-US Data Privacy Framework status, EAA scope, UK crypto-asset regime development, retirement of the EU ODR platform in 2025:** `[U]` and subject to change.
7. **Primary-source fetches failed** for irs.gov, fincen.gov and eba.europa.eu (DNS error in the research sandbox); IRS and EBA material comes from search-engine summaries.
8. **Most primary sources in §10 were not opened** (marked "not opened").

## 10. References

Internal: [00 — Conventions](00-conventions.md), [03 — Domain model](03-domain-model-and-mutual-credit-rules.md), [06](06-data-ledger-and-audit.md), [08](08-ux-and-accessibility.md), [11](11-deployment-operations-and-support.md), [12](12-quality-strategy-and-test-plan.md), [13](13-assumptions-risks-decisions-roadmap.md), [16](16-network-policy-catalog.md), [18](18-data-export-and-portability.md); research note [research-regulatory-privacy-questions](../notes/research-regulatory-privacy-questions.md).

External sources (from the research note; all accessed 2026-10-21; "search" = surfaced in a search result but not opened; "not opened" = listed for counsel to check):

- IRS-1: IRS, Topic no. 420, Bartering income. <https://www.irs.gov/taxtopics/tc420.html> (search)
- IRS-2: IRS, Barter exchanges. <https://www.irs.gov/businesses/small-businesses-self-employed/barter-exchanges> (search)
- IRS-3: IRS Publication 525. <https://www.irs.gov/publications/p525> (not opened; via secondary source)
- CFR-1: 26 CFR 1.6045-1; IRC 6045(c)(3). <https://www.ecfr.gov>, <https://www.law.cornell.edu/uscode/text/26/6045> (not opened)
- FIN-1: FinCEN FIN-2019-G001. <https://www.fincen.gov/sites/default/files/2019-05/FinCEN%20CVC%20Guidance%20FINAL.pdf> (search)
- FIN-2: FinCEN MSB definition. <https://www.fincen.gov/money-services-business-definition> (not opened; fetch failed)
- MTMA: CSBS Money Transmission Modernization Act. <https://www.csbs.org> (not opened)
- HMRC-1: HMRC manuals <https://www.gov.uk/hmrc-internal-manuals>; forum commentary <https://www.accountingweb.co.uk/any-answers/lets-barter-scheme> (search)
- ATO-1: <https://brislets.com/tax-centrelink-and-gst/> (secondary, search); <https://www.ato.gov.au> (not opened)
- VATD: Directive 2006/112/EC. <https://eur-lex.europa.eu/eli/dir/2006/112/oj> (not opened)
- DAC7: Directive (EU) 2021/514. <https://eur-lex.europa.eu/eli/dir/2021/514/oj> (not opened)
- PSD2: Directive (EU) 2015/2366. <https://eur-lex.europa.eu/eli/dir/2015/2366/oj> (not opened)
- EBA-LN: EBA limited-network guidelines (24 Feb 2022), summarised at <https://www.loyensloeff.com/insights/news--events/news/eba-guidelines-on-the-limited-network-exclusion-under-psd2/> and <https://www.regulationtomorrow.com/2022/02/eba-publishes-final-guidelines-on-the-limited-network-exclusion-under-the-psd2/> (search); <https://www.eba.europa.eu> (fetch failed)
- EU-PAY: Commission PSD3/PSR proposals. <https://finance.ec.europa.eu> (not opened; status unverified)
- EMD2: Directive 2009/110/EC. <https://eur-lex.europa.eu/eli/dir/2009/110/oj> (not opened)
- MiCA: Regulation (EU) 2023/1114. <https://eur-lex.europa.eu/eli/reg/2023/1114/oj> (not opened)
- CCD2: Directive (EU) 2023/2225. <https://eur-lex.europa.eu/eli/dir/2023/2225/oj> (not opened)
- FCA-PSR / FCA-CC: <https://www.fca.org.uk> (not opened)
- FSMA: <https://www.legislation.gov.uk> (not opened)
- FINTRAC / BOC: <https://fintrac-canafe.canada.ca>; <https://www.bankofcanada.ca/core-functions/retail-payments-supervision/> (not opened)
- AUSTRAC / ASIC: <https://www.austrac.gov.au>; <https://asic.gov.au> (not opened)
- FSPR: <https://fspr.companiesoffice.govt.nz> (not opened)
- HOWEY / SEC: SEC v. W.J. Howey Co., 328 U.S. 293 (1946); <https://www.sec.gov> (not opened)
- UCTD / CRA / CRD / DMCCA: <https://eur-lex.europa.eu>; <https://www.legislation.gov.uk> (not opened)
- DSA: Regulation (EU) 2022/2065. <https://eur-lex.europa.eu/eli/reg/2022/2065/oj> (not opened)
- GDPR (Arts. 8, 15, 17, 33, 35, Ch. V): <https://eur-lex.europa.eu/eli/reg/2016/679/oj> (not opened)
- EDPB-CP: Guidelines 07/2020. <https://www.edpb.europa.eu> (not opened)
- ICO-CP: <https://ico.org.uk/for-organisations/> (not opened)
- EDPB-BC: Guidelines 02/2025 (v2.0, 7 July 2026). <https://www.edpb.europa.eu/system/files/2026-07/edpb_guidelines_202502_blockchain_v2_en.pdf> (search listing); <https://www.mondaq.com/fin-tech/1617044/blockchain-and-gdpr-edpb-guidelines-022025-adopted>
- EDPB-DPIA: WP248 / national DPIA lists. <https://www.edpb.europa.eu> (not opened)
- CCPA: <https://cppa.ca.gov> (not opened)
- COPPA: 16 CFR Part 312. <https://www.ecfr.gov> (not opened)
- AADC: <https://ico.org.uk> (not opened)
- OPC / OAIC: <https://www.priv.gc.ca>; <https://www.oaic.gov.au>; <https://www.privacy.org.nz> (not opened)
- EAA: Directive (EU) 2019/882. <https://eur-lex.europa.eu/eli/dir/2019/882/oj> (not opened)
- ADA-WEB: <https://www.ada.gov> (not opened)
- OFAC: <https://ofac.treasury.gov> (not opened)
