# 03 — Domain Model and Mutual-Credit Rules

| Field | Value |
|---|---|
| Status | Draft |
| Owner role | Domain architect (ledger and rules) |
| Last updated | 2026-10-21 |
| Scope | Domain entities, balance and sign semantics, platform-wide invariants (`INV-`), Trade authorisation/acceptance/cancellation/reversal, limits, adjustments, member departure, default handling, inactivity, and the split between configurable Network policy and fixed platform invariants |
| Depends on | [00 — Conventions and canon](00-conventions.md) |
| Open questions | See section 10 below; consolidated in [13 — Assumptions, risks, decisions](13-assumptions-risks-decisions-roadmap.md) |

## 1. Purpose and scope

This document is the single authoritative home for **what the ledger means** in Circula and **which rules can never be bent**. It defines the domain objects, the arithmetic of balances, the invariants `INV-001`..`INV-0xx`, and the business rules for Trades, Reversals, Adjustments, departures, defaults and inactivity. It uses the shared canon in [00 section 5](00-conventions.md) without redefinition; where this document elaborates a canon item it keeps the canon's name, sign and ID.

Out of scope here (owned elsewhere): state-machine mechanics ([15](15-lifecycle-state-machines.md)); storage layout and audit record fields ([06](06-data-ledger-and-audit.md)); API shapes and error format ([07](07-api-and-integration-contract.md)); policy defaults and ranges ([16](16-network-policy-catalog.md)); governance procedure ([10](10-governance-and-network-operations.md)); legal characterisation of any rule ([19](19-legal-and-regulatory-open-questions.md)).

### 1.1 The unit is an accounting unit

`[DOM-001]` is the controlling statement; everything else in the product MUST be consistent with it.

The credit unit is a **bookkeeping unit internal to one Network**. Circula records who has provided and who has received goods or services within the Network. A positive balance is a record of net contribution; a negative balance is a record of net receipt. The unit is **not** money, not legal tender, not a security or investment, not a cryptocurrency or token, bears no interest, and is **not redeemable** for national currency or any asset by the Platform. Whether any person accepts credits in exchange for goods or services is a decision of that person and the Network's community; Circula does not guarantee value or acceptance. Whether a given Network's use of the unit has legal or tax consequences in a jurisdiction is a question for counsel (see [19](19-legal-and-regulatory-open-questions.md)).

## 2. Definitions used here

Glossary entries live in [14](14-glossary.md); this section states the domain-level shape of each object only.

| Object | Domain definition | Key attributes (conceptual) |
|---|---|---|
| **Network** | Isolated tenant with its own unit, members, policy set and ledger (canon 5.1). | `network_id`, `unit_name`, `unit_symbol`, `unit_scale`, policy set, `created_at` |
| **Membership** | A person's relationship to one Network, with a lifecycle ([15](15-lifecycle-state-machines.md)). Carries roles (`member`, `network_admin`, `dispute_reviewer`). | `membership_id`, `network_id`, `person_ref`, roles, state, ban flag and reason, `credit_limit` (per-member override, nullable) |
| **Member Account** | The ledger account owned by exactly one Membership, created with it. | `account_id`, `membership_id`, cached `balance` |
| **System Account** | Non-member account that balances Adjustments, openings and departure settlements. Exempt from limits. Two per Network: `network_clearing` and `community_fund`. | `account_id`, `system_role` |
| **Balance** | `Σ posting amounts` of an account (canon 5.2). | integer minor units, signed |
| **Credit limit** | Magnitude of negative balance permitted; floor is `−credit_limit`. | integer minor units, ≥ 0 |
| **Listing (Offer / Request)** | Informational advertisement; **no ledger effect**. | `listing_id`, kind (`offer` or `request`), owner membership, text, state |
| **Transaction** | Business-level record with a `type`: `trade`, `reversal`, `adjustment`, `opening_balance`, `departure_settlement`. Exactly one Journal Entry results from each *posted* Transaction. | `transaction_id`, type, status, initiator, reason, links |
| **Trade** | Transaction of type `trade` between exactly two Member Accounts. | amount, `initiated_by` (`provider` or `recipient`), Provider, Recipient, description, optional Listing link, `expires_at` |
| **Transaction status** | For Trades, the lifecycle state of canon 5.3 (`proposed`, `completed`, `declined`, `cancelled`, `expired`, `reversed`). For other types see [15 section 7](15-lifecycle-state-machines.md). | |
| **Journal Entry** | Immutable ledger record: set of Postings that sum to zero. | `journal_entry_id`, `transaction_id`, `network_id`, `posted_at` (UTC), links to originals |
| **Posting** | One signed line of a Journal Entry against one account. | `account_id`, signed integer amount |
| **Dispute** | Formal challenge to a completed or proposed Trade; has no ledger effect itself. | parties, Trade link, reason, state, resolution, remedy link |
| **Adjustment** | Admin-initiated correcting Journal Entry (type `adjustment`) with mandatory reason, Audit Event and Member notification. | amount, reason code, free-text reason, linked dispute (optional) |
| **Audit Event** | Append-only record of a security- or governance-relevant action; separate from the ledger. | actor, time, action, target, reason (full field list in [06](06-data-ledger-and-audit.md)) |

## 3. Core domain rules

Requirement table columns: ID, Priority, Requirement, Rationale, Verification. Priorities follow [00 section 2](00-conventions.md).

### 3.1 Unit, Network and accounts

| ID | Pri | Requirement | Rationale | Verification |
|---|---|---|---|---|
| DOM-001 | P0 | Every user-visible surface (UI, API documentation, emails, exports, marketing copy shipped with the Platform) MUST describe the credit unit as an accounting unit and MUST NOT offer, imply or label redemption, cash-out, interest, investment return, or fiat/crypto conversion. | Prevents money-like claims (INV-014); limits legal exposure. | Inspection: string/copy audit against a denylist plus review checklist; QA gate in [12](12-quality-strategy-and-test-plan.md) |
| DOM-002 | P0 | Each Network MUST be an isolated ledger: every Journal Entry references accounts of exactly one Network (INV-011), and no read or write API MAY traverse Networks except for a person's own list of memberships. | Tenant isolation; no hidden cross-community exposure. | Test: attempt cross-network posting and cross-network reads with valid credentials in both Networks; expect rejection |
| DOM-003 | P0 | Creating a Membership MUST create exactly one Member Account with balance 0 in the same atomic step; a Member Account MUST NOT exist without a Membership. | One owner per account; no orphan balances. | Test: failure injection mid-creation leaves neither or both |
| DOM-004 | P0 | Creating a Network MUST create its two System Accounts (`network_clearing`, `community_fund`) atomically with the Network. System Accounts MUST NOT be owned by any Member and MUST NOT appear as a Trade party. | Adjustments and settlements need a balancing account (INV-001). | Test: Trade proposal naming a System Account is rejected |
| DOM-005 | P0 | A person with Memberships in two Networks MUST have two independent Member Accounts; no balance, limit, listing or history is shared or transferable between them. | Canon 5.1; INV-011. | Test: same login, two Networks, independent balances |
| DOM-006 | P0 | Listings MUST have no ledger effect. A Trade MAY reference a Listing but the Listing's content (price, hours) MUST NOT constrain or auto-fill the posted amount. | Keeps ledger authority in the Trade alone. | Test: Trade amount differs from listing text without error |
| DOM-007 | P0 | `unit_name`, `unit_symbol` and `unit_scale` MUST be set at Network creation. `unit_scale` MUST be an integer 0–4 (default 2). Unit identity and scale are immutable once any Journal Entry exists (INV-013). `unit_name`/`unit_symbol` display-label edits are also blocked after the first Journal Entry in MVP. | Re-scaling would silently change every balance's meaning. | Test: change attempt after first entry is rejected with an Audit Event |

### 3.2 Amounts and balances

| ID | Pri | Requirement | Rationale | Verification |
|---|---|---|---|---|
| DOM-008 | P0 | All amounts MUST be integers in minor units; no floating-point type MAY appear in storage, API or computation of amounts or balances. UI decimal input MUST be parsed from text to integer with the Network's `unit_scale`; input with more fractional digits than `unit_scale` MUST be rejected (not rounded). | Exactness; no rounding creep. | Test: inputs `1.005` at scale 2 rejected; `1.50` parsed to 150; property test on parse/format round-trip |
| DOM-009 | P0 | `balance(account) = Σ posting amounts` for all Postings on that account. A positive balance means net *provided*; a negative balance means net *received* (canon 5.2). | Single arithmetic definition. | Test: recompute from postings equals cached value (INV-008) |
| DOM-010 | P0 | `available_to_spend = max(0, balance + credit_limit)` for Member Accounts. Pending (`proposed`) Trades MUST NOT reduce it (no holds, D-05). Surfaces MUST label pending Trades as informational. | Prevents false assurance; avoids deadlock from reservations. | Test: with a `proposed` Trade of any size, `available_to_spend` is unchanged |
| DOM-011 | P0 | The effective `credit_limit` of a Member Account is its per-member override if set, else `default_credit_limit`. It MUST satisfy `0 ≤ credit_limit ≤ max_credit_limit`. Setting it is an admin action producing an Audit Event (INV-010). | Bounded exposure per member. | Test: set above `max_credit_limit` rejected; change audited |
| DOM-012 | P0 | Lowering a credit limit below a Member's current debt (that is, new floor above current balance) MUST be allowed and leaves the account **over limit**: the account MUST NOT be debited by any Trade (it MUST still be creditable) until `balance ≥ −credit_limit`. Existing balances MUST NOT be altered. | Governance can respond to risk without silent balance mutation. | Test: limit lowered to 20 with balance −35: debit trade rejected; credit trade accepted |
| DOM-013 | P1 | If `max_balance` is set, the same over-cap logic applies: lowering `max_balance` below a Member's balance leaves the account **over cap**; it MUST NOT be credited by any Trade until `balance ≤ max_balance`, and MAY still be debited. | Symmetry with DOM-012. | Test as DOM-012 mirrored |
| DOM-014 | P0 | Policy bounds evaluated for a Trade (`min_trade_amount`, `max_trade_amount`, credit floor, `max_balance`) MUST be evaluated at **acceptance** against the policy in force at that moment; proposal-time evaluation is for early feedback only and MUST NOT be relied on for enforcement. | Policy may change during the proposal's life; ledger safety needs current data. | Test: policy tightened between proposal and accept; accept uses new bounds |
| DOM-015 | P0 | The system MUST NOT debit or credit any account by any mechanism other than a Journal Entry (no interest, demurrage, fee, decay or automatic expiry of balances in MVP). | No silent mutation (INV-004); no money-like features (INV-014). | Inspection: no scheduled job posts entries other than those enumerated in [15](15-lifecycle-state-machines.md); test: idle Network over simulated 5 years produces no postings |

### 3.3 Worked examples

All examples use Network "Riverbend" with unit symbol `h`, `unit_scale = 2` (1.00 h = 100 minor units), `default_credit_limit` = 50.00 h (5000). Columns show minor units. Network sums are shown to demonstrate INV-007.

#### Example A — simple Trade

Alice (Provider) provided 20.00 h of gardening to Bob (Recipient). Alice proposed (`initiated_by = provider`); Bob accepts.

| Account | Balance before | Posting in JE-1001 | Balance after |
|---|---:|---:|---:|
| Alice (Provider) | 1000 | +2000 | 3000 |
| Bob (Recipient) | −500 | −2000 | −2500 |
| `network_clearing` | −500 | — | −500 |
| **Sum** | **0** | **0** | **0** |

Checks: Bob's floor is −5000; −2500 ≥ −5000, accepted. Postings sum to zero (INV-001); both lines are in one Journal Entry (INV-002). Trade state `proposed` → `completed`.

#### Example B — Trade rejected for exceeding credit limit (and the max-balance variant)

Bob has balance −4500, `credit_limit` 5000, so `available_to_spend = 500` (5.00 h). A Trade of 10.00 h (1000) is proposed by Alice and Bob attempts to accept.

| Check | Computation | Result |
|---|---|---|
| Recipient floor (INV-005) | −4500 − 1000 = −5500 < −5000 | **Fail** |
| Provider cap (INV-006), if `max_balance` = 10000 and Alice = 9500 | 9500 + 1000 = 10500 > 10000 | **Fail** (independent of the above) |

Outcome: accept is rejected atomically with reason `credit_limit_exceeded` (or `max_balance_exceeded`; final error codes are owned by [07](07-api-and-integration-contract.md)). No Journal Entry is written; all balances are unchanged; the Trade stays `proposed` until it is declined, cancelled or expires, so the parties can fix the cause (for example Bob earns credit first, or an admin raises Bob's limit) and Bob can retry within `proposal_expiry_days`. Amount is immutable (DOM-018): to change it, cancel and re-propose.

#### Example C — Reversal

JE-1001 (Example A) is later reversed after a dispute is `resolved_upheld`. Meanwhile Alice spent 15.00 h with Carol (JE-1002: Alice −1500, Carol +1500), so before the Reversal: Alice 1500, Bob −2500, Carol 1500, `network_clearing` −500.

| Account | Balance before | Posting in JE-1003 (reversal of JE-1001) | Balance after |
|---|---:|---:|---:|
| Alice | 1500 | −2000 | −500 |
| Bob | −2500 | +2000 | −500 |
| Carol | 1500 | — | 1500 |
| `network_clearing` | −500 | — | −500 |
| **Sum** | **0** | **0** | **0** |

The Reversal posts the exact negation of JE-1001 and links to it; JE-1001 is untouched (INV-003). The Trade becomes `reversed`. Alice's resulting −500 is within her floor; had it breached her floor, the Reversal would still post (DOM-027) and Alice would be **over limit** (DOM-012). Carol's balance is never touched: Reversal undoes only the original Journal Entry.

#### Example D — Adjustment via `network_clearing`

Dana's opening balance was imported as 10.00 h but the agreed legacy figure is 30.00 h. An admin posts an upward Adjustment of 20.00 h. (Independent mini-network: Dana 1000, `network_clearing` −1000.)

| Account | Balance before | Posting in JE-2001 (adjustment) | Balance after |
|---|---:|---:|---:|
| Dana | 1000 | +2000 | 3000 |
| `network_clearing` | −1000 | −2000 | −3000 |
| **Sum** | **0** | **0** | **0** |

A downward Adjustment of 5.00 h to Dana would post Dana −500 and `network_clearing` +500. Every Adjustment carries a reason, an Audit Event naming the acting admin, and a notification to Dana (DOM-031). `network_clearing` may be positive or negative; it is a balancing account, not a Member claim.

#### Example E — Departure with negative balance

Eve leaves with balance −1200 (12.00 h). Frank holds +2000; `network_clearing` is −800 (sum 0).

| `departure_settlement_mode` | Journal Entry | Eve | Frank | `network_clearing` | `community_fund` | Sum |
|---|---|---:|---:|---:|---:|---:|
| (before) | — | −1200 | 2000 | −800 | 0 | 0 |
| `settle` | Trade, Frank Recipient, 1200 (Eve earns by providing a service) | 0 | 800 | −800 | 0 | 0 |
| `write_off` | JE-3001 `departure_settlement`: Eve +1200, `network_clearing` −1200 | 0 | 2000 | −2000 | 0 | 0 |
| `community_fund` | JE-3001: Eve +1200, `community_fund` −1200 | 0 | 2000 | −800 | −1200 | 0 |

Interpretation: in `write_off` and `community_fund` the Network community absorbs the shortfall (Frank's claim is no longer matched by a Member obligation). The Platform does not collect, convert or demand national currency for this. Membership then transitions `departing` → `closed` in the same atomic step as the settlement entry (DOM-036).

## 4. Platform-wide invariants (INV)

Invariants are properties that MUST hold for every Network regardless of policy. No configuration, role or deployment option MAY disable them. Each lists its enforcement point (where the system makes violation impossible, not merely unlikely) and how it is verified. Enforcement points reference logical components defined in [05](05-architecture-and-trust-boundaries.md).

| ID | Invariant (frozen meaning) | Pri | Rationale | Enforcement point | Verification |
|---|---|---|---|---|---|
| INV-001 | Every Journal Entry's Postings sum to exactly zero. | P0 | Double-entry integrity; makes INV-007 provable. | Ledger write path validates the sum before commit; storage-level constraint where the chosen technology supports it. | Property test over random entries (sum ≠ 0 always rejected); reconciliation job asserts for all entries. |
| INV-002 | Both sides of a Trade are posted atomically; a partial Trade posting MUST NOT be observable or persistable. | P0 | A half-posted Trade corrupts both accounts. | Single transaction boundary containing the Journal Entry, all Postings, cached balance updates and Trade state change. | Failure-injection test at each step; read-your-writes test under concurrent readers never shows one leg. |
| INV-003 | Ledger records are append-only; no UPDATE/DELETE of Journal Entries or Postings in normal operation. | P0 | Auditability and trust. | Application layer exposes no mutate/delete path; the ledger store's credentials used by the application lack mutate/delete rights on ledger tables/collections where supported. "Normal operation" excludes only documented disaster-recovery and migration procedures that preserve content and are themselves audited ([11](11-deployment-operations-and-support.md)). | Inspection of privileges; negative test attempting update/delete through every interface; migration test proves entry hashes/content unchanged. |
| INV-004 | Corrections are made only by Reversals or Adjustments, each a new Journal Entry linked to the original (where one exists) and to an Audit Event. | P0 | No silent edits; every correction is explainable. | Reversal and Adjustment are the only transaction types that can reference an existing entry; both require reason and create the Audit Event in the same atomic step. | Test: no code path posts a correction without both links; audit completeness query. |
| INV-005 | A Member Account balance MUST NOT go below `−credit_limit` as the result of a Trade posting. | P0 | Bounds each member's exposure. | Check inside the INV-002 transaction against the balance read under account-level serialisation (DOM-020). | Boundary tests (exactly at floor allowed, one minor unit below rejected); concurrency test (DOM-020). |
| INV-006 | A Member Account balance MUST NOT exceed `max_balance` (if set) as the result of a Trade posting. | P1 | Prevents hoarding where the Network chooses to cap. | As INV-005. | Mirrored boundary and concurrency tests. |
| INV-007 | Per Network, the sum of all balances (Member + System Accounts) is exactly zero. | P0 | Direct consequence of INV-001 and INV-011; detects corruption. | Derived property; reconciliation job per Network. | Periodic reconciliation (frequency in [11](11-deployment-operations-and-support.md)); test after every example in section 3.3. |
| INV-008 | Stored/cached balances equal the sum of Postings and can be recomputed from the ledger alone. | P0 | Cache is never an authority. | Cached balance updated only inside the posting transaction; recompute tool needs no other data. | Recompute-and-compare job; fault-injection test; restore-from-ledger test ([12](12-quality-strategy-and-test-plan.md)). |
| INV-009 | A retried request with the same idempotency key and payload yields the same outcome and never a second posting; same key with different payload is rejected. | P0 | Network retries are normal; double charge is unacceptable. | Idempotency record created in the same transaction as the posting; key scope is (Network, authenticated actor, key). Record retention and key format: [06](06-data-ledger-and-audit.md), [07](07-api-and-integration-contract.md). | Replay tests (identical, mutated payload, concurrent duplicates). |
| INV-010 | Every Adjustment, Reversal, policy change, role change and dispute resolution produces an Audit Event naming actor, time, reason and target. | P0 | Governance accountability. | Audit Event written in the same transaction as the action; the action fails if the Audit Event cannot be written. | Test: forced audit-write failure aborts the action; coverage query per action type. |
| INV-011 | No cross-Network postings; Journal Entries reference accounts of exactly one Network. | P0 | Tenant isolation (DOM-002). | Ledger write path rejects entries whose accounts span Networks. | Test with crafted cross-network payloads. |
| INV-012 | The Platform Operator and Network Administrators cannot create a Trade on behalf of a Member without that Member's acceptance; admin value changes are Adjustments only, visible to the affected Member. | P0 | No unilateral debit by staff (D-06). | Trade creation requires the acting identity to be a party; acceptance requires the *other* party's identity; admin role grants no Trade authority. Adjustments are a distinct transaction type with mandatory notification. | Authorisation matrix tests per role ([02](02-personas-roles-and-journeys.md)); test that admin credentials cannot propose or accept for others. |
| INV-013 | `unit_scale` and the Network's unit identity are immutable once any Journal Entry exists. | P0 | DOM-007. | Network update path checks for existing entries (ledger count > 0) inside the same transaction as the update. | Test; concurrency test of first entry vs scale change. |
| INV-014 | Money-like claims are prohibited in the product: no redemption, cash-out, interest, or fiat conversion in MVP. | P0 | DOM-001; legal exposure. | Absence of such features is a release gate; no transaction type or account type supports them. Copy audit (DOM-001). | Inspection and copy audit at each release. |
| INV-015 | A Trade is valid only if: Provider and Recipient are distinct Member Accounts of the same Network; both Memberships are `active` at posting time (exception: DOM-035 for `departing`); the amount is an integer ≥ 1 minor unit, ≥ `min_trade_amount` and ≤ `max_trade_amount` where set. | P0 | Closes self-dealing, ghost-member and zero/negative-amount loopholes. | Validation inside the INV-002 transaction. | Table-driven tests of every invalid combination. |
| INV-016 | After a Membership is `closed`, its Member Account balance is exactly 0 and no further Journal Entry references it. | P0 | Closed identities (anonymised per D-09) must not accumulate silent state. | Closure transition requires zero balance (settled or settlement entry in the same atomic step); ledger write path rejects entries naming closed accounts. | Test: reversal/adjustment naming a closed account is rejected (DOM-026). |

## 5. Trade rules

### 5.1 Authorisation and acceptance

| ID | Pri | Requirement | Rationale | Verification |
|---|---|---|---|---|
| DOM-016 | P0 | A Trade MUST be proposed by one of its two parties. `initiated_by = provider` means the Provider asks to be credited; `initiated_by = recipient` means the Recipient offers to be debited. Only the *other* party (the Counterparty) MAY accept or decline. | Both sides consent (canon 5.3; INV-012). | Test: Proposer cannot accept own proposal; third parties and admins cannot accept |
| DOM-017 | P0 | A Trade completes only on acceptance, and acceptance and posting are one atomic step (INV-002). There is no persisted intermediate "accepted" state. | Canon 5.3. | Test: failure injection between acceptance and posting leaves Trade `proposed` |
| DOM-018 | P0 | A Trade's Provider, Recipient, amount and `initiated_by` are immutable after creation. Changes are made by `cancelled` then a new proposal. Only description text MAY be edited by the Proposer while `proposed`, and the edit MUST be visible to the Counterparty before acceptance (accept binds to the version shown: a stale `proposal_version` is rejected). | Prevents bait-and-switch between display and accept. | Test: edit during counterparty view; accept with old version fails |
| DOM-019 | P0 | At acceptance the system MUST re-validate INV-015, INV-005, INV-006 and DOM-012/013 against current state, then post. A rejection leaves the Trade `proposed` and unchanged. | Safe under changing balances and policies (DOM-014). | Examples B; test for each failing guard |
| DOM-020 | P0 | Limit checks and the resulting posting MUST be evaluated against the account balance as of the same atomic unit. Concurrent postings that touch the same account MUST have the effect of some serial order. Implementations MUST acquire any account-level serialisation in a deterministic order (for example ascending `account_id`) to avoid deadlock. | Two simultaneous accepts must not each pass the floor check on the same stale balance. | Test: 100 concurrent accepts against a recipient who can afford exactly 3: exactly 3 succeed, balance equals floor, others rejected `credit_limit_exceeded`. |
| DOM-021 | P0 | A `proposed` Trade MUST have `expires_at = created_at + proposal_expiry_days` (fixed at creation). At or after `expires_at` it MUST NOT be acceptable, even if the expiry sweeper has not yet recorded state `expired`. | Deterministic outcome independent of job timing. | Test: accept at `expires_at` and one tick before with the sweeper paused |
| DOM-022 | P0 | `declined`, `cancelled` and `expired` Trades MUST have no ledger effect and MUST NOT alter any balance or `available_to_spend`. | Pending states carry no financial weight (D-05). | Test: balances identical before/after |
| DOM-023 | P0 | Completion posts exactly two Postings: `−amount` to the Recipient and `+amount` to the Provider, in one Journal Entry (canon 5.2). | Equal and opposite. | Test per example A |
| DOM-024 | P1 | Descriptions are free text limited to a maximum length [DECISION NEEDED: description length limit; recommended 500 characters] and MUST NOT be used as a ledger-significant field. | Avoids personal-data accumulation and ambiguity. | Test: length limit enforced; no logic reads description |

### 5.2 Reversal

A Reversal is a new Journal Entry that negates a prior completed Trade's entry (canon 5.1). It is not an edit, delete, refund workflow or partial credit.

| ID | Pri | Requirement | Rationale | Verification |
|---|---|---|---|---|
| DOM-025 | P0 | A Reversal MUST target exactly one `completed` Trade, post the exact negation of that Trade's Postings, link to it, require a reason, create an Audit Event (INV-010) and notify both parties; the Trade becomes `reversed` (terminal). A Trade MAY be reversed at most once. | INV-004 traceability; one remedy per entry. | Test: second reversal rejected; link/audit/notification assertions |
| DOM-026 | P0 | A Reversal MUST be rejected if either account's Membership is `closed` (INV-016). Remedy in that case is an Adjustment between the remaining account and `network_clearing`, linked to the original Trade and any Dispute. | Closed accounts stay at zero. | Test |
| DOM-027 | P0 | Reversal MUST NOT be subject to INV-005/006 (it is not a Trade posting); if it leaves an account below its floor or above its cap, the entry MUST be flagged `limit_exceeded_by_reversal` and the account is **over limit** or **over cap** under DOM-012/013. | Truth of the correction outranks the limit; the exposure is then visible and bounded to the original amount. | Test: example C variant with Alice at floor |
| DOM-028 | P0 | Reversal MAY be initiated only (a) by both parties jointly (one proposes, the other accepts) within `dispute_window_days` of completion, or (b) by a Network Administrator (subject to `admin_quorum`) as the remedy of a Dispute that has been decided `resolved_upheld` or `resolved_settled`. A Reversal MUST NOT be requested by the Platform Operator. | INV-012; limits discretionary power. | Authorisation matrix test; test of window boundary |
| DOM-029 | P1 | Partial reversal is not supported in MVP. Partial correction is performed by an Adjustment or a counter-Trade. Reversals of Reversals or of Adjustments are not supported; correct them with a new Adjustment linked to the original. | Simplicity; keeps reversal graph acyclic. | Test: attempt rejected with a clear error |

## 6. Adjustments

| ID | Pri | Requirement | Rationale | Verification |
|---|---|---|---|---|
| DOM-030 | P0 | An Adjustment MUST post between exactly one Member Account and one System Account (`network_clearing` by default; `community_fund` only for fund-related reasons), with strictly positive magnitude and a direction (credit or debit to the Member). It MUST carry a reason code from a Network-visible list and free-text justification, link to the original entry or Dispute where one exists, and create an Audit Event. | INV-004, INV-010; examples D. | Test: missing reason or link-required code rejected |
| DOM-031 | P0 | The affected Member MUST be notified of an Adjustment, and the Adjustment MUST appear in that Member's history labelled as an Adjustment with the reason, before or when the balance change is visible. An Adjustment MUST NOT be hidden from the affected Member by any role. | INV-012 visibility. | Test: history and notification exist; no flag suppresses them |
| DOM-032 | P0 | An Adjustment is not a Trade: it needs no Member acceptance but MUST obey `admin_quorum` (the initiating admin counts as one approval; approvals come from distinct admins) and an admin MUST NOT approve an Adjustment affecting their own Member Account. | Prevents self-enrichment. | Test with quorum 1, 2; self-target |
| DOM-033 | P0 | An Adjustment that leaves a Member Account outside its limits MUST be flagged (as DOM-027) and requires an explicit `override_reason`. | Visible exceptions. | Test |
| DOM-034 | P1 | The Platform provides one flow to post **opening balances** for imported or migrated communities (type `opening_balance`) against `network_clearing`; it is only available in a Network's setup phase (before the first Trade) or via Adjustment afterwards. | Network onboarding without breaking INV-007. | Test: opening posting sum zero; blocked after first Trade |

## 7. Member departure, default and inactivity

Definitions. A **departure** is any path ending in Membership `closed`: voluntary (member requests), administrative (admin removes), or by ban. A **default** is the situation in which a Member Account is negative at departure, or a Member is unresponsive while negative; there is no legal-collection function in Circula and the Platform MUST NOT contact third parties about a balance. How the community responds beyond the options below is a governance matter ([10](10-governance-and-network-operations.md)) and any enforcement outside the Platform is a question for counsel ([19](19-legal-and-regulatory-open-questions.md)).

| ID | Pri | Requirement | Rationale | Verification |
|---|---|---|---|---|
| DOM-035 | P0 | While a Membership is `departing` the Member MAY propose or accept only Trades that move the balance **toward zero without crossing it**: Recipient role only when `balance > 0`, Provider role only when `balance < 0`, and amount ≤ the absolute value of the balance. All Listings of the member are archived on entry to `departing`; open `proposed` Trades where the member is a party and which do not satisfy the rule are `cancelled` by the system. | Orderly wind-down; no new exposure. | Test table across balance signs |
| DOM-036 | P0 | Transition `departing` → `closed` MUST occur only when balance is 0. In mode `settle` this happens when the Member's trades bring it to 0. In `write_off` the system posts a `departure_settlement` Journal Entry between the Member Account and `network_clearing` that zeroes the account (Example E); in `community_fund` the same with `community_fund`. The entry and the `closed` transition are one atomic step; the Audit Event records mode, amount, sign and initiating actor. | INV-016; transparent cost-bearing. | Examples E; atomicity test |
| DOM-037 | P0 | Mode `settle` MUST NOT post any automatic system entry. If the balance is not 0 when the settlement window ends [DECISION NEEDED: settlement window length; recommended 30 days], the Membership stays `departing`, admins are notified, and an admin MAY (subject to `admin_quorum`) record a decision to apply `write_off` or `community_fund` for that member with reason. | Never a silent write-off; governance decision is visible. | Test: window end leaves balances unchanged and notifies |
| DOM-038 | P0 | For a **positive** balance at departure, the Member MUST be shown, before confirming, that the Platform offers no redemption (DOM-001) and the amount that will be released to the Network under the configured mode. Under `write_off`/`community_fund` the entry is `Member −B` and `network_clearing` or `community_fund` `+B`. Whether releasing a positive balance is enforceable or lawful in a jurisdiction is a question for counsel ([19](19-legal-and-regulatory-open-questions.md)). | Informed consent; avoids implied cash-out. | UX test; test of entry signs |
| DOM-039 | P0 | `departure_settlement_mode` is a Network policy changed only prospectively with notice (D-11) and snapshotted into the Membership when it enters `departing`. Later policy changes MUST NOT alter the mode applied to an already-`departing` Membership. | Predictability for the leaver. | Test |
| DOM-040 | P1 | For administrative departure of a negative-balance member (including ban) the admin MUST enter a reason; the same mode applies; `community_fund` balance shortfall is reported to admins and in Network-visible summary according to `balance_visibility`. Members MUST NOT be pursued through the Platform (no dunning messages beyond one pre-departure notification). | Limits harm; proportional response. | Test notification count; summary visibility |
| DOM-041 | P0 | A ban is a flag with reason on a `suspended` or `closed` Membership; a banned account follows the same balance rules. Ban reason is visible to the banned person and appealable under [10](10-governance-and-network-operations.md). | Canon 5.3; due process. | Test |

### 7.1 Suspension and inactivity

| ID | Pri | Requirement | Rationale | Verification |
|---|---|---|---|---|
| DOM-042 | P0 | A `suspended` Membership MUST NOT propose, accept or be named in a *new* Trade proposal; its existing `proposed` Trades remain, cannot be accepted by it, can be cancelled/declined by the other party, and otherwise expire. Its Listings are not visible in the directory; balance and history remain viewable by the Member. | Safe pause. | Test |
| DOM-043 | P0 | A Membership is **inactive** (a derived flag, not a lifecycle state) when no Trade has been proposed, accepted or completed by it and it has not authenticated for `inactivity_threshold_days`. Becoming inactive MUST produce a notification to the Member and an admin-visible flag. | Observable, reversible, non-punitive. | Test with clock control |
| DOM-044 | P0 | Inactivity MUST NOT change any balance, limit or state automatically in MVP. Any consequence (reminder, suspension, departure) is an explicit admin action. Demurrage, decay and inactivity fees are excluded (DOM-015, INV-014). | Avoids money-like mechanics and surprises. | Test: 5-year idle simulation |

## 8. Disputes (domain rules)

State mechanics are in [15 section 5](15-lifecycle-state-machines.md); procedure and appeals in [10](10-governance-and-network-operations.md).

| ID | Pri | Requirement | Rationale | Verification |
|---|---|---|---|---|
| DOM-045 | P0 | Only a party to a Trade MAY raise a Dispute on it; a `completed` Trade only within `dispute_window_days` of completion; a `proposed` Trade at any time while `proposed`. A Dispute has no ledger effect and does not freeze balances (D-12); the Trade remains in its state. | Bounded, party-driven; no freezes in MVP. | Test |
| DOM-046 | P0 | A Dispute Reviewer MUST NOT review a Dispute to which they are a party and MUST NOT approve a remedy affecting their own account. Reviewer, raiser and respondent identities are visible to the parties. | Impartiality. | Test |
| DOM-047 | P0 | Remedies of an upheld or settled Dispute are exclusively: Reversal (DOM-028), Adjustment (section 6), cancellation of a still-`proposed` Trade, or "no ledger remedy". Remedy and resolution are recorded in the same atomic step with an Audit Event (INV-010). | Closed remedy set; traceable. | Test |
| DOM-048 | P1 | A Dispute open longer than `dispute_review_sla_days` MUST be flagged to admins; the consequence of exceeding the SLA is defined in [15](15-lifecycle-state-machines.md). | Prevents silent stalls. | Test with clock control |

## 9. Configurable policy versus platform invariants

Policies (left) are set per Network by a Network Administrator within platform-set bounds, take effect prospectively after `policy_notice_days` (default 14, D-11) except safety fixes, and each change is an Audit Event (INV-010). Invariants (right) cannot be configured. Catalogue, defaults and ranges: [16](16-network-policy-catalog.md).

| Policy parameter | What it governs | Platform-fixed boundary (cannot be configured away) |
|---|---|---|
| `unit_name`, `unit_symbol` | Display labels | Immutable once a Journal Entry exists (INV-013, DOM-007); never presented as money (DOM-001) |
| `unit_scale` | Decimal places 0–4 | Immutable after first posting (INV-013); integer arithmetic only (DOM-008) |
| `membership_mode` | `invite`, `approval`, `open` | Every member has an identified, authenticated Membership (D-07); Membership lifecycle fixed ([15](15-lifecycle-state-machines.md)) |
| `default_credit_limit`, `max_credit_limit` | Size of permitted negative balance | Limit is a floor on Trade postings (INV-005); `0 ≤ credit_limit ≤ max_credit_limit` (DOM-011); System Accounts exempt |
| `max_balance` | Optional positive cap | Applies to Trade postings only (INV-006, DOM-027) |
| `min_trade_amount`, `max_trade_amount` | Trade size bounds | Amount ≥ 1 minor unit (INV-015) |
| `proposal_expiry_days` | Proposal lifetime | Proposals always expire; expiry has no ledger effect (DOM-021, DOM-022) |
| `dispute_window_days`, `dispute_review_sla_days` | When and how fast disputes run | Disputes never mutate balances directly; remedies limited to DOM-047 |
| `balance_visibility`, `directory_visibility` | Who sees balances / listings | A Member always sees their own balance and history; the affected Member always sees Adjustments and Reversals touching them (DOM-031) |
| `inactivity_threshold_days` | When inactive flag is raised | No automatic balance effect (DOM-044) |
| `departure_settlement_mode` | `settle`, `write_off`, `community_fund` | Balance zero at close (INV-016); no cash-out (INV-014) |
| `policy_notice_days` | Notice before changes bite | Minimum notice for non-safety changes is not below the platform floor set in [16](16-network-policy-catalog.md) |
| `admin_quorum` | Admins needed for sensitive actions | Value ≥ 1 and ≤ number of active `network_admin` members; Operator holds no ledger powers (INV-012) |

Platform-fixed, no parameter: append-only ledger (INV-003), double-entry (INV-001), atomic posting (INV-002), correction only via Reversal/Adjustment (INV-004), idempotency (INV-009), audit completeness (INV-010), Network isolation (INV-011), no Platform-Operator posting powers.

| ID | Pri | Requirement | Rationale | Verification |
|---|---|---|---|---|
| DOM-049 | P0 | A policy change MUST be validated against the platform bounds in the table above and MUST NOT be applied retroactively to any completed Journal Entry or to a `departing` Membership (DOM-039). | Policy governs the future, not history. | Test |
| DOM-050 | P0 | A `default_credit_limit` change MUST NOT alter existing per-member overrides, and MUST apply to members without an override from its effective date; raising limits takes effect at the effective date and lowering follows DOM-012. | Predictable limits. | Test |

## 10. Assumptions and open decisions

| Ref | Item | Options | Recommended default |
|---|---|---|---|
| [DECISION NEEDED: description length limit] | Max Trade description length (DOM-024) | 140 / 500 / 2000 chars | 500 |
| [DECISION NEEDED: settlement window length] | How long `departing` waits for `settle` (DOM-037) | 14 / 30 / 60 days | 30 days |
| Assumption A-1 | `community_fund` is a second System Account created with every Network; replenishment mechanism (voluntary contributions as a distinct transaction type) is deferred (P2). | Deferred vs MVP | Deferred |
| Assumption A-2 | Reversal and Adjustment are exempt from INV-005/006 (limits apply to "Trade posting" per canon); exceptions are flagged and make the account over limit/cap rather than blocking the correction. | Block vs flag | Flag |
| Assumption A-3 | Added INV-015 (Trade validity) and INV-016 (closed account is zero) as additions to the canon's INV-001..014; canon registry should be extended. | — | Accept |
| Assumption A-4 | Transaction types `reversal`, `adjustment`, `opening_balance`, `departure_settlement` extend the canon's note that `Transaction` has type `trade`. | — | Accept |
| Assumption A-5 | Joint reversal within `dispute_window_days` (DOM-028 a) needs no Dispute. Alternative: all Reversals require a Dispute. | Joint vs dispute-only | Joint |
| Assumption A-6 | No holds on pending Trades (D-05) means two pending Trades can each pass proposal checks but only those that fit at acceptance time succeed. | — | Accept |
| Canon note | Canon 5.3 has no transition out of `departing` other than `closed`; this document therefore treats rescinding a departure as unsupported (rejoin = new Membership). | Allow `departing` → `active` | Unsupported in MVP |
| Legal | Treatment of released positive balances, write-offs, and any tax or regulatory aspect: a question for counsel ([19](19-legal-and-regulatory-open-questions.md)). | — | — |

Cross-doc needs: [06](06-data-ledger-and-audit.md) to carry the transaction-type list, `limit_exceeded_by_reversal` flag and the System Account pair; [16](16-network-policy-catalog.md) to carry bounds in section 9; [04](04-functional-requirements.md) to derive FR-TRD/FR-ADM from DOM-016..DOM-050.

## 11. References

- [00 — Conventions and canon](00-conventions.md)
- [15 — Lifecycle state machines](15-lifecycle-state-machines.md)
- [16 — Network policy catalog](16-network-policy-catalog.md)
- [14 — Glossary](14-glossary.md)
- [19 — Legal and regulatory open questions](19-legal-and-regulatory-open-questions.md)
- Prior-art notes: [research-prior-art-mutual-credit](../notes/research-prior-art-mutual-credit.md), section 2.2 (default and departure handling: account reaches zero before closing, loss socialised or absorbed by a reserve/community fund; the basis for `departure_settlement_mode`) and section 2.4 (hoarding and permanent-deficit failure modes; basis for optional `max_balance` and over-limit handling). The note's demurrage-to-community-fund pattern is deliberately **not** adopted in MVP (DOM-015, INV-014). Sources there were accessed 2026-10-21 and carry `[UNVERIFIED]` flags that this spec does not rely on.
- Regulatory and privacy questions: [research-regulatory-privacy-questions](../notes/research-regulatory-privacy-questions.md)
