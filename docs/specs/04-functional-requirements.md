# Functional Requirements (MVP)
Status: Draft | Owner: Product architect (role) | Updated: 2026-11-14 | Scope: MVP functional requirements for one Network: setup, membership, listings, transactions, balance and history, admin controls, disputes, audit, basic exports | Deps: [_context.md](./_context.md), [01-product-vision-scope.md](./01-product-vision-scope.md), [02-personas-roles-journeys.md](./02-personas-roles-journeys.md), [03-domain-model-rules.md](./03-domain-model-rules.md), [13-assumptions-risks-decisions-roadmap.md](./13-assumptions-risks-decisions-roadmap.md) | Open questions: 6, see "Open decisions"

## Purpose

This spec defines the functional requirements (FR) the MVP must provide to run one Network end to end (see PRD-022 to PRD-023 in [01](./01-product-vision-scope.md)). Each FR states one observable behavior, its priority, its rationale, and acceptance criteria a tester can check.

This spec does not define rules. Entities, states, ledger rules and invariants live in [03](./03-domain-model-rules.md). Roles and journeys live in [02](./02-personas-roles-journeys.md). Terms are defined in [_context.md](./_context.md). Where an FR implements a rule, it links to the rule's ID and does not restate it.

Conventions: priority keywords follow [_context.md](./_context.md#normative-terms). Each FR has one priority. "Implements" lists the DOM (domain), ROL (role/journey), PRD (product) and other IDs the FR realizes. Acceptance criteria are written as checks. A check passes or fails.

Out of scope for this spec: UX layout and accessibility detail (UX prefix, not yet written), security controls beyond the FRs below (SEC prefix, not yet written), and deferred capabilities listed at the end.

## Requirements

### Network setup

- **FR-001** (MUST) An administrator MUST be able to create a Network with a name and a unique identifier within the installation.
  - Rationale: every other object belongs to one Network (DOM-001), so a Network must exist first.
  - Verify: create two Networks with the same identifier; the second is rejected. Create one with a valid identifier; it is returned by a read.
  - Implements: [DOM-001](./03-domain-model-rules.md), [PRD-023](./01-product-vision-scope.md), [ROL-001](./02-personas-roles-journeys.md)

- **FR-002** (MUST) A Network MUST store a value for every configurable item listed in [_context.md](./_context.md#glossary) (credit limit default, join rule, dispute window, visibility default, acceptance window, listing expiry, settlement on departure). Each item MUST have a value before the Network accepts transactions.
  - Rationale: network policy is per-Network (PRD-003); an unset item would cause undefined behavior.
  - Verify: create a Network without setting one item; activation is refused and the missing item is named. Set all items; activation succeeds.
  - Implements: [PRD-003](./01-product-vision-scope.md), [DOM-080](./03-domain-model-rules.md)

- **FR-003** (MUST) A policy change MUST take effect only for actions created after the change is saved. It MUST NOT alter posted entries or existing transactions.
  - Rationale: prevents silent change to agreed exchanges (ROL-062).
  - Verify: change a credit limit; an entry posted before the change is unchanged; a transaction proposed before the change keeps its original window.
  - Implements: [ROL-062](./02-personas-roles-journeys.md), [DOM-080](./03-domain-model-rules.md)

- **FR-004** (MUST) The system MUST reject any policy value that would disable, weaken or override a ledger invariant (DOM-020 to DOM-026) or the credit semantics (DOM-082, DOM-083).
  - Verify: submit each prohibited setting through the admin interface and any interface path; each attempt is rejected and writes no change.
  - Implements: [DOM-081](./03-domain-model-rules.md), [PRD-002](./01-product-vision-scope.md)

### Membership

- **FR-005** (MUST) A prospective member MUST become a member of a Network only through that Network's join rule.
  - Verify: attempt to create an active account without the join rule being satisfied; the attempt is rejected.
  - Implements: [ROL-010](./02-personas-roles-journeys.md), [ROL-015](./02-personas-roles-journeys.md)

- **FR-006** (MUST) Completing a join MUST create exactly one account for that member in that Network, with balance zero and no entries. A second join for the same member and Network MUST NOT create a second account.
  - Verify: complete a join twice for one person; one account exists; its balance is zero and its entry list is empty.
  - Implements: [DOM-002](./03-domain-model-rules.md), [ROL-011](./02-personas-roles-journeys.md)

- **FR-007** (MUST) Each account MUST have exactly one standing: ACTIVE, SUSPENDED or DEPARTED. Only an ACTIVE account MUST be able to propose, accept or complete a transaction.
  - Verify: for each non-ACTIVE standing, attempt propose, accept and complete; each is rejected.
  - Implements: [DOM-003](./03-domain-model-rules.md)

- **FR-008** (MUST) A member or a network administrator MUST be able to start departure of an account. Departure MUST move the account to DEPARTED, retain its entries and balance, and MUST NOT delete the account. A non-zero balance MUST be settled only by a traceable adjustment under the Network's settlement rule.
  - Verify: depart an account with balance +5; the account is DEPARTED; entries are unchanged; no balance change occurs without an adjustment entry; the network balance sum is still zero.
  - Implements: [DOM-060](./03-domain-model-rules.md), [DOM-061](./03-domain-model-rules.md), [DOM-062](./03-domain-model-rules.md), [DOM-064](./03-domain-model-rules.md)

- **FR-009** (MAY) A network administrator MAY suspend an ACTIVE account and MAY lift the suspension. Suspension MUST record a reason category.
  - Verify: suspend an account; it cannot transact (FR-007); lift it; it can transact again; both events have audit records (FR-042).
  - Implements: [DOM-063](./03-domain-model-rules.md)

- **FR-010** (MUST) A network administrator MUST be able to assign and revoke the member, network administrator and dispute reviewer roles for members of their Network. Each grant or revocation MUST be scoped to that Network.
  - Verify: grant reviewer role in Network A; the person has no reviewer right in Network B.
  - Implements: [ROL-002](./02-personas-roles-journeys.md), [ROL-066](./02-personas-roles-journeys.md)

- **FR-011** (MUST NOT) A platform operator MUST NOT post entries, change balances, decide disputes, read dispute statements or change network policy through any product function.
  - Verify: for each of those actions, the operator role is refused in an automated permission test.
  - Implements: [ROL-005](./02-personas-roles-journeys.md), [ROL-057](./02-personas-roles-journeys.md)

### Member directory and listings

- **FR-012** (MUST) An ACTIVE member MUST be able to create, edit, pause, withdraw and close their own offers and requests within their Network. Listings MUST move through the states defined in [DOM-006](./03-domain-model-rules.md) and [DOM-007](./03-domain-model-rules.md).
  - Verify: each action on an own listing succeeds; the same action on another member's listing is refused.
  - Implements: [ROL-020](./02-personas-roles-journeys.md), [DOM-006](./03-domain-model-rules.md), [DOM-007](./03-domain-model-rules.md)

- **FR-013** (MUST) Creating, editing or changing the state of a listing MUST NOT change any balance and MUST NOT create any entry.
  - Verify: perform each listing action; the entry count and all balances are unchanged.
  - Implements: [ROL-022](./02-personas-roles-journeys.md), [DOM-008](./03-domain-model-rules.md)

- **FR-014** (MUST) Listings MUST be visible only to ACTIVE and SUSPENDED members of the same Network. A listing MUST NOT be returned in any query from another Network.
  - Verify: query listings as a member of Network B; no Network A listing is returned, including by direct identifier.
  - Implements: [ROL-023](./02-personas-roles-journeys.md), [DOM-001](./03-domain-model-rules.md), [PRD-025](./01-product-vision-scope.md)

- **FR-015** (SHOULD) Members SHOULD be able to search listings in their Network by text and by category.
  - Verify: a search for a unique word in a listing title returns that listing; a category filter returns only listings in that category.
  - Implements: [ROL-024](./02-personas-roles-journeys.md)

- **FR-016** (MUST) Each listing MUST expire after the Network's listing expiry period. An expired listing MUST be marked EXPIRED and MUST NOT be offered as an option for a new proposal.
  - Verify: set expiry to 1 day in a test Network; a listing aged 2 days is EXPIRED and fails proposal.
  - Implements: [DOM-006](./03-domain-model-rules.md), [DOM-007](./03-domain-model-rules.md)

- **FR-017** (MAY) A network administrator MAY remove a listing that breaks Network rules. The removal MUST record a reason and MUST notify the listing owner with the reason category.
  - Verify: remove a listing without reason; refused. Remove with reason; listing is removed, owner is notified, audit event exists.
  - Implements: [ROL-021](./02-personas-roles-journeys.md), [DOM-070](./03-domain-model-rules.md)

### Transactions

- **FR-018** (MUST) An ACTIVE member MUST be able to propose a transaction with a counterpart who is an ACTIVE member of the same Network. The proposal MUST record provider, receiver, amount, optional listing reference, description and status.
  - Verify: propose with each required field; a proposal missing any required field is refused. A proposal to a member of another Network is refused.
  - Implements: [ROL-030](./02-personas-roles-journeys.md), [DOM-009](./03-domain-model-rules.md), [DOM-011](./03-domain-model-rules.md)

- **FR-019** (MUST) A proposal MUST create a transaction in status PROPOSED and MUST NOT change any balance or create any entry.
  - Verify: create a proposal; status is PROPOSED; entry count and balances are unchanged.
  - Implements: [ROL-031](./02-personas-roles-journeys.md), [DOM-010](./03-domain-model-rules.md)

- **FR-020** (MUST) Only the counterparty MUST be able to accept or reject a PROPOSED transaction. Only the proposer MUST be able to withdraw (cancel) it before acceptance. Cancellation rules MUST follow [DOM-013](./03-domain-model-rules.md).
  - Verify: the proposer attempts accept; refused. The counterparty attempts withdraw; refused. Each permitted action succeeds once and produces the correct status.
  - Implements: [ROL-032](./02-personas-roles-journeys.md), [DOM-012](./03-domain-model-rules.md), [DOM-013](./03-domain-model-rules.md)

- **FR-021** (MUST) Before accepting, the system MUST check that the acceptance would not take the paying account below its negative credit limit. If it would, acceptance MUST be refused and the refusal MUST state the reason category "credit limit reached".
  - Verify: set a limit of 0 and balance 0; a transaction of 1 from that account is refused at acceptance with the stated reason.
  - Implements: [ROL-033](./02-personas-roles-journeys.md), [DOM-027](./03-domain-model-rules.md)

- **FR-022** (MUST) Completion of an accepted transaction MUST require confirmation under the Network's completion model ([DOM-015](./03-domain-model-rules.md)). Completion MUST post exactly two equal and opposite entries and set status COMPLETED in one atomic unit. If either part fails, neither MUST persist.
  - Verify: inject a failure after the first entry write; no entry persists and status remains ACCEPTED. A successful completion yields two entries summing to zero.
  - Implements: [ROL-034](./02-personas-roles-journeys.md), [DOM-016](./03-domain-model-rules.md), [DOM-026](./03-domain-model-rules.md)

- **FR-023** (MUST) Repeated accept, complete or other posting requests for the same transaction (double submit, client retry) MUST produce one posting and MUST return the original result.
  - Verify: send the completion request 5 times concurrently; exactly two entries exist.
  - Implements: [ROL-035](./02-personas-roles-journeys.md), [DOM-024](./03-domain-model-rules.md)

- **FR-024** (MUST) Posting MUST be serialized per account. Concurrent postings MUST NOT jointly breach the credit limit.
  - Verify: two concurrent debits that each fit alone but together exceed the limit; exactly one succeeds.
  - Implements: [DOM-027](./03-domain-model-rules.md), [DOM-028](./03-domain-model-rules.md)

- **FR-025** (SHOULD) A PROPOSED transaction not accepted within the Network's acceptance window SHOULD transition to CANCELLED by the system, with an audit event.
  - Verify: set window to 1 day in a test Network; a 2-day-old PROPOSED transaction is CANCELLED and an audit event exists.
  - Implements: [DOM-014](./03-domain-model-rules.md)

- **FR-026** (MUST) The UI and API MUST show the current transaction status at all times. A pending, unconfirmed or possibly stale state MUST be shown as pending or stale, not as final. Data older than the Network's staleness threshold MUST be labelled.
  - Verify: simulate a delayed confirmation; the status shows "pending", not "completed". Simulate stale data; a stale label is shown.
  - Implements: [ROL-036](./02-personas-roles-journeys.md), [ROL-046](./02-personas-roles-journeys.md), [ROL-075](./02-personas-roles-journeys.md)

### Balance and history

- **FR-027** (MUST) A member MUST be able to view their own balance, credit limit and available headroom (credit limit plus balance).
  - Verify: with balance +3 and limit 10, headroom shows 13.
  - Implements: [ROL-040](./02-personas-roles-journeys.md), [DOM-004](./03-domain-model-rules.md), [DOM-005](./03-domain-model-rules.md)

- **FR-028** (MUST) A member MUST be able to view their own entry history as an append-only sequence. A reversal or adjustment MUST appear as a new item that references the original.
  - Verify: post a correction; the original item is unchanged; a new item references it.
  - Implements: [ROL-041](./02-personas-roles-journeys.md), [DOM-021](./03-domain-model-rules.md), [DOM-023](./03-domain-model-rules.md)

- **FR-029** (MUST) The displayed balance MUST equal the sum of the account's posted entries. A mismatch MUST be logged as a defect and MUST NOT be silently corrected.
  - Verify: an automated check recomputes every account balance from entries and compares it with the stored balance; zero mismatches.
  - Implements: [ROL-042](./02-personas-roles-journeys.md), [DOM-025](./03-domain-model-rules.md)

- **FR-030** (MUST) A member MUST NOT see another member's balance or entries. A counterpart MAY see the amount and status of a shared transaction.
  - Verify: query another member's balance by identifier; refused. Counterpart view shows amount and status only.
  - Implements: [ROL-043](./02-personas-roles-journeys.md)

### Administrator views

- **FR-031** (MUST) A network administrator MUST be able to view account balances and entry history for accounts in their Network. Each view MUST be recorded as an audit event.
  - Verify: an administrator opens an account's history; an audit event records the actor, account and time.
  - Implements: [ROL-044](./02-personas-roles-journeys.md), [DOM-070](./03-domain-model-rules.md)

### Admin controls

- **FR-032** (MUST) A network administrator MUST be able to view and change every policy value listed in [_context.md](./_context.md#glossary) for their Network. Each change MUST record old value, new value, actor, timestamp and reason.
  - Verify: change the credit limit default with a reason; the old and new values, actor, time and reason are stored in one audit event. A change without reason is refused.
  - Implements: [ROL-060](./02-personas-roles-journeys.md), [ROL-061](./02-personas-roles-journeys.md), [DOM-072](./03-domain-model-rules.md)

- **FR-033** (SHOULD) Before confirming a join-rule or credit-limit change, the administrator SHOULD see the number of accounts the change would affect.
  - Verify: preview for a limit reduction shows the count of accounts whose headroom would fall below zero.
  - Implements: [ROL-065](./02-personas-roles-journeys.md)

- **FR-034** (MUST NOT) A network administrator MUST NOT edit an account balance or post an entry directly. Balance changes MUST occur only through transactions, adjustments and settlements defined in this spec and [03](./03-domain-model-rules.md).
  - Verify: no interface or API path accepts a direct balance or entry write from an administrator; attempts are refused.
  - Implements: [ROL-064](./02-personas-roles-journeys.md), [DOM-022](./03-domain-model-rules.md)

### Disputes

- **FR-035** (MUST) A participant of a COMPLETED transaction MUST be able to raise a dispute on it within the Network's dispute window. The dispute MUST include a reason category and a non-empty description. A dispute referencing a non-COMPLETED transaction, or from a non-participant, MUST be refused.
  - Verify: raise with empty description; refused. Raise after window; refused. Raise as participant within window; accepted.
  - Implements: [ROL-037](./02-personas-roles-journeys.md), [ROL-050](./02-personas-roles-journeys.md), [ROL-051](./02-personas-roles-journeys.md), [DOM-050](./03-domain-model-rules.md)

- **FR-036** (MUST) Raising a dispute MUST NOT change any balance or entry. It MUST set the open-dispute marker on the transaction and MUST prevent any further completion of that transaction.
  - Verify: raise a dispute; balances and entry count unchanged; a completion attempt on that transaction is refused.
  - Implements: [ROL-052](./02-personas-roles-journeys.md), [DOM-052](./03-domain-model-rules.md)

- **FR-037** (MUST) A dispute MUST be assigned to a dispute reviewer from the Network's current reviewer roster. The assigned reviewer MUST NOT be a participant in the disputed transaction. Assignment MUST be recorded in an audit event.
  - Verify: attempt to assign a participant; refused. Assign an eligible reviewer; assignment and audit event exist.
  - Implements: [ROL-053](./02-personas-roles-journeys.md), [ROL-004](./02-personas-roles-journeys.md), [DOM-051](./03-domain-model-rules.md)

- **FR-038** (MUST) Both participants MUST be able to see the dispute status and submit a written statement. The assigned reviewer MUST be able to see both statements.
  - Verify: each participant sees status and submits; the reviewer sees both; a non-participant non-reviewer sees neither.
  - Implements: [ROL-054](./02-personas-roles-journeys.md)

- **FR-039** (MUST) A dispute outcome MUST be one of: UPHELD (no entries), REVERSED (full reversal), or PARTIALLY ADJUSTED. Any financial effect MUST be posted only as new traceable adjustments or reversals. No posted entry MUST be edited or deleted by a dispute outcome.
  - Verify: resolve a dispute as PARTIALLY ADJUSTED; new adjustment entries reference the transaction; original entries are byte-identical.
  - Implements: [ROL-055](./02-personas-roles-journeys.md), [ROL-056](./02-personas-roles-journeys.md), [DOM-053](./03-domain-model-rules.md), [DOM-040](./03-domain-model-rules.md), [DOM-041](./03-domain-model-rules.md)

- **FR-040** (MUST) A transaction MUST have at most one open dispute at a time. A resolved dispute is final. A further dispute MAY be raised only on a COMPLETED transaction, as a new dispute.
  - Verify: raise a second dispute while one is open; refused. Resolve; raise again; accepted as a new dispute record.
  - Implements: [DOM-054](./03-domain-model-rules.md)

- **FR-041** (MUST) A reversal or adjustment MUST cite exactly one authority: a dispute outcome, or a recorded mutual agreement of both participants. A correction MAY take the provider below the credit limit only as permitted by [DOM-043](./03-domain-model-rules.md).
  - Verify: post an adjustment with no authority; refused. Post with dispute outcome; accepted and audited.
  - Implements: [DOM-042](./03-domain-model-rules.md), [DOM-043](./03-domain-model-rules.md)

### Audit history

- **FR-042** (MUST) Each ledger posting, status transition, dispute event, standing change, adjustment, role change and policy change MUST produce an audit event containing actor, action, object reference, timestamp and reason where one applies.
  - Verify: run each listed action; one audit event per action exists with the required fields.
  - Implements: [DOM-070](./03-domain-model-rules.md), [ROL-006](./02-personas-roles-journeys.md)

- **FR-043** (MUST) Audit events MUST be append-only. Any update or delete request MUST be rejected and the rejection MUST be logged.
  - Verify: attempt to update and delete an audit event through every interface; each is rejected; the rejections are logged.
  - Implements: [DOM-071](./03-domain-model-rules.md)

### Basic exports

- **FR-044** (MAY) A member MAY export their own entry history in a documented, machine-readable format. The export MUST contain only that member's entries and transaction references.
  - Verify: export; the file contains only the member's entries; the format matches its published description.
  - Implements: [ROL-045](./02-personas-roles-journeys.md), [OD-04-1](#open-decisions)

- **FR-045** (MUST) A network administrator MUST be able to export the Network's entries and audit events in a documented, machine-readable format. The export MUST contain enough data to recompute every balance from entries (FR-029).
  - Verify: export; recompute each balance from the export alone; results match stored balances.
  - Implements: [DOM-025](./03-domain-model-rules.md), [DOM-070](./03-domain-model-rules.md)

- **FR-046** (MUST) Exports MUST NOT include another member's identifying data beyond what the export scope in [OD-04-6](#open-decisions) allows.
  - Verify: export as administrator; the fields present match the recorded default.
  - Implements: [ROL-043](./02-personas-roles-journeys.md)

### Cross-cutting

- **FR-047** (MUST) No user-facing text, export header or API description produced for MVP functions MUST state or imply that credits have monetary value, are redeemable, are a security or are a cryptocurrency.
  - Verify: text scan of all user-facing strings and exports for the listed claims; zero matches.
  - Implements: [DOM-082](./03-domain-model-rules.md), [DOM-083](./03-domain-model-rules.md), [ROL-007](./02-personas-roles-journeys.md)

## Deferred work

The following capabilities are not required for MVP. They are listed so that FRs above do not depend on them. Each needs its own spec before work starts. Nothing here commits to a design.

- Mobile applications.
- Federation between Networks (PRD-025 forbids cross-network transactions in MVP).
- Tokenization of credits.
- Conversion of credits to any payment form.
- Reputation scoring.
- Advanced analytics.
- Third-party integrations.

## Open decisions

Each item lists options and a recommended default. The default applies until a decision is recorded in this spec.

- **OD-04-1** Export format for member and administrator exports (FR-044, FR-045).
  - Options: (a) one structured text format with published schema; (b) tabular text format; (c) both, same field set.
  - Default: (a) for FR-045 and FR-044. Tabular output MAY be added later with the same field set.

- **OD-04-2** Join approval mode (input to FR-005). See the join-rule decision in [02](./02-personas-roles-journeys.md#open-decisions).
  - Options: (a) administrator approves each request; (b) invitation token issued by an administrator; (c) open join.
  - Default: (a). Recommended because it keeps membership under administrator control.

- **OD-04-3** Default listing expiry (input to FR-016).
  - Options: (a) no expiry; (b) fixed period set by the platform, overridable per Network; (c) no platform default, Network must set a value (FR-002).
  - Default: (b), 90 days, overridable per Network. The value is a placeholder until a Network sets its own.

- **OD-04-4** Default acceptance window for PROPOSED transactions (input to FR-025). Timing is also covered in [03](./03-domain-model-rules.md#open-decisions).
  - Options: (a) 7 days; (b) 14 days; (c) no window.
  - Default: (a) 7 days, overridable per Network.

- **OD-04-5** Counterparty visibility beyond amount and status (input to FR-030).
  - Options: (a) none beyond amount and status; (b) also the counterpart's current headroom.
  - Default: (a), consistent with ROL-043.

- **OD-04-6** Identifying fields in administrator exports (input to FR-046).
  - Options: (a) pseudonymous account identifiers only; (b) display names included.
  - Default: (a). Display names MAY be added only if the Network records a reason in a policy change (FR-032).

## Related docs

- [_context.md](./_context.md): shared terms, normative keywords, ledger invariants.
- [01-product-vision-scope.md](./01-product-vision-scope.md): product goals and MVP boundaries.
- [02-personas-roles-journeys.md](./02-personas-roles-journeys.md): roles (ROL), journeys and permissions matrix.
- [03-domain-model-rules.md](./03-domain-model-rules.md): entities, states, ledger rules (DOM).
- [13-assumptions-risks-decisions-roadmap.md](./13-assumptions-risks-decisions-roadmap.md): assumptions, risks, roadmap and deferred work.
