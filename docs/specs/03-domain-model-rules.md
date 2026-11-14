# Domain Model and Mutual-Credit Rules
Status: Draft | Owner: Product architect (role) | Updated: 2026-11-14 | Scope: Domain entities, transaction lifecycle, ledger rules, disputes, reversals, departure and default for one Circula network | Deps: [_context.md](./_context.md) | Open questions: 12, see "Open decisions"

## Purpose

This spec is the authoritative home for domain entities, mutual-credit semantics, and ledger invariants. It defines what the platform MUST enforce and which behavior each network MAY configure. Shared terms, normative keywords, and the seven platform-wide ledger invariants are defined in [_context.md](./_context.md) and are referenced here, not redefined.

Out of scope: data storage schema (planned DAT-*), API shapes (planned API-*), governance and join procedures (planned GOV-*), security controls (planned SEC-*), and any legal characterization of credits.

## Entities

Glossary definitions are in [_context.md, Glossary](./_context.md#glossary). This section adds the rules that apply to each entity.

| ID | Pri | Requirement | Rationale | Test criterion |
|---|---|---|---|---|
| DOM-001 | MUST | Every account, offer, request, transaction, dispute, adjustment, and audit event belongs to exactly one network. Cross-network references are rejected. | Networks are isolated ledgers with their own rules. | Creating a transaction whose accounts belong to different networks is rejected and writes no records. Queries scoped to network A never return network B records. |
| DOM-002 | MUST | A member has at most one account per network (see OD-10). | Keeps balance attribution unambiguous. | Second account creation for the same member and network is rejected. |
| DOM-003 | MUST | An account has exactly one standing: ACTIVE, SUSPENDED, or DEPARTED. Only ACTIVE accounts may propose, accept, or confirm transactions. | Standing controls participation without touching the ledger. | Each standing transition is audited (DOM-072). Attempts by SUSPENDED or DEPARTED accounts to propose, accept, or confirm are rejected. |
| DOM-004 | MUST | Account balance equals the signed sum of its posted entries. Positive means the member has provided more value than received. | Defines the sign convention used by all other rules. | For any account, recomputing the sum of entries equals the stored balance. |
| DOM-005 | MUST | Each account has a credit limit: a non-negative value, set by the network, that is the floor for its balance. The network sets a default limit; a network administrator MAY set a per-account limit. | Limits are network policy (see Configurable vs invariant). | Limit set to L: balance may reach -L and no lower (see DOM-027). Setting a negative limit is rejected. |
| DOM-006 | MUST | An offer is a member listing of goods or services they can provide. Offer states: OPEN, WITHDRAWN, EXPIRED. Creating, editing, or withdrawing an offer creates no ledger entries and changes no balance. | Listings are discovery, not obligation. | Offer lifecycle operations produce zero entries and zero balance change. |
| DOM-007 | MUST | A request is a member listing of goods or services they want. Request states match offers (OPEN, WITHDRAWN, EXPIRED). The same no-ledger rule as DOM-006 applies. | Symmetric with offers. | As DOM-006. |
| DOM-008 | MUST | Offers and requests are not commitments. Only a transaction creates obligations and postings. | Prevents listings from being treated as trades. | Accepting a listing without a transaction record is impossible in the model; no posting references a listing without a transaction. |
| DOM-009 | MUST | A transaction records: provider account, receiver account, amount, optional offer or request reference, description, status, and timestamps. Provider and receiver MUST be distinct accounts in the same network. Amount MUST be a positive integer in network minor units (see OD-08). | Two-party exchange is the only trade shape in scope. | Zero, negative, non-integer, or self-referencing (provider = receiver) transactions are rejected. |

## Transaction lifecycle

| ID | Pri | Requirement | Rationale | Test criterion |
|---|---|---|---|---|
| DOM-010 | MUST | Transaction status is one of: PROPOSED, ACCEPTED, COMPLETED, REJECTED, CANCELLED, REVERSED. Permitted transitions are exactly: PROPOSED to ACCEPTED, REJECTED, or CANCELLED; ACCEPTED to COMPLETED or CANCELLED; COMPLETED to REVERSED (via DOM-041 only). REJECTED, CANCELLED, and REVERSED are terminal. | A single state machine keeps posting rules checkable. | Every permitted transition succeeds; every other transition is rejected and writes an audit event recording the attempt. |
| DOM-011 | MUST | Proposal requires an ACTIVE proposer who is the provider or receiver. The proposer MUST NOT be both. | Authorization at entry. | Proposal by a non-party, by a non-ACTIVE member, or self-dealing is rejected. |
| DOM-012 | MUST | Only the counterparty may accept or reject a PROPOSED transaction. | Consent of the other side is required before obligations exist. | Accept or reject by the proposer, or by a third party, is rejected. |
| DOM-013 | MUST | Cancellation rules: the proposer MAY cancel a PROPOSED transaction at any time. An ACCEPTED transaction MAY be cancelled only with recorded agreement of both parties (see OD-07). | Prevents one-sided withdrawal after agreement. | Unilateral cancel of ACCEPTED is rejected. Cancel with both confirmations succeeds and posts no entries. |
| DOM-014 | SHOULD | A PROPOSED transaction not accepted within the network-configured acceptance window transitions to CANCELLED by the system. The window is network policy. | Stops stale proposals from lingering. | With window W, a proposal left PROPOSED past W becomes CANCELLED with an audit event. |
| DOM-015 | MUST | Completion requires confirmation from both parties (see OD-01). Completion is the only event that posts entries. | Both sides attest the exchange happened. | Completion with one confirmation leaves status ACCEPTED and writes no entries. Completion after both confirmations posts entries (DOM-016). |
| DOM-016 | MUST | Completing a transaction posts exactly two entries in one atomic unit with the status change to COMPLETED: provider account +amount, receiver account -amount. | Equal and opposite postings are the core mutual-credit rule. | After completion, the two entries sum to zero and balances change by +amount and -amount. Fault injection between entry write and status write leaves neither or both applied. |

## Ledger rules

Platform-wide invariants are defined in [_context.md, Ledger invariants](./_context.md#ledger-invariants-platform-wide-not-configurable). Each is restated here as a testable rule with a DOM ID; the wording in `_context.md` is authoritative.

| ID | Pri | Requirement | Rationale | Test criterion |
|---|---|---|---|---|
| DOM-020 | MUST | Balanced: each COMPLETED transaction's entries sum to zero, and the sum of all account balances in a network is zero. ([invariant 1](./_context.md#ledger-invariants-platform-wide-not-configurable)) | Conservation of credit. | After any test sequence, network balance sum is 0 and every transaction's entry sum is 0. |
| DOM-021 | MUST | Append-only: posted entries are never updated or deleted. ([invariant 2](./_context.md#ledger-invariants-platform-wide-not-configurable)) | History must be trustworthy. | Update or delete of a posted entry is rejected at every layer that can reach it. |
| DOM-022 | MUST | No silent mutation: every balance change is caused by a posted entry, and every posted entry references an audit event. ([invariant 3](./_context.md#ledger-invariants-platform-wide-not-configurable)) | Every change is attributable. | A balance difference with no matching entry is detected by the reconstruction check (DOM-025). An entry without an audit reference is rejected. |
| DOM-023 | MUST | Corrections are new entries that reference the entry or transaction they correct. Original entries are untouched. ([invariant 4](./_context.md#ledger-invariants-platform-wide-not-configurable)) | Traceable correction. | After a reversal, original entry content is byte-identical and a new entry references its id. |
| DOM-024 | MUST | Posting is idempotent per (transaction, posting type). A retried request returns the original result and creates no second entry. ([invariant 5](./_context.md#ledger-invariants-platform-wide-not-configurable)) | Network retries are normal. | Submitting the same completion request N times yields exactly one pair of entries. |
| DOM-025 | MUST | Any account balance can be recomputed from its entries alone. ([invariant 6](./_context.md#ledger-invariants-platform-wide-not-configurable)) | Enables audit and integrity checks. | Recomputed balance equals stored balance for all accounts after a test sequence. |
| DOM-026 | MUST | Posting is atomic: both sides of a transaction post together or neither does. ([invariant 7](./_context.md#ledger-invariants-platform-wide-not-configurable)) | Prevents one-sided balance changes. | Injected failure after the first entry leaves zero entries for that transaction. |
| DOM-027 | MUST | A posting MUST be rejected if it would leave any account balance below its negative credit limit (-L). A rejected posting writes no entries. | Credit limit is the floor. | With L = 100 and balance 0, a debit of 100 succeeds and a debit of 101 is rejected with no entries written. |
| DOM-028 | MUST | Postings touching the same account are serialized. Concurrent postings cannot jointly breach the limit. | Race conditions would otherwise breach limits. | Two concurrent debits of 60 against L = 100 and balance 0: exactly one succeeds. |
| DOM-029 | MUST | Lowering a credit limit does not alter existing entries. If the balance is already below the new floor, further debits are rejected; credits (positive entries) are still accepted. Debits resume only when the balance is at or above the new floor. | Limit changes are policy, not history rewrites. | Lower L from 100 to 50 with balance -80: a debit of 1 is rejected; a credit of 30 is accepted (balance -50); a debit of 1 is still rejected (floor is -50, debit would go to -51). |
| DOM-030 | MAY | A network MAY configure a positive-balance ceiling. Default is no ceiling (see OD-03). | Ceiling is policy. | If configured, postings that would exceed it are rejected by the same mechanism as DOM-027. |

## Reversals and adjustments

| ID | Pri | Requirement | Rationale | Test criterion |
|---|---|---|---|---|
| DOM-040 | MUST | An adjustment is a new entry set with its own audit event. It references the original transaction and entry ids. Adjustments are the only way to correct posted value. | Implements DOM-023 at the domain level. | No code path updates an existing transaction's entries; each adjustment's audit event links to the original. |
| DOM-041 | MUST | A reversal of a COMPLETED transaction posts equal and opposite entries referencing the original, sets status REVERSED, and records a cited authority (DOM-042). Partial adjustments are allowed. Cumulative adjustments MUST NOT exceed the original amount. | Reverses value cleanly and bounds corrections. | Reversal of 50 on a 50 transaction nets to zero. A second reversal attempt is rejected. A partial adjustment of 30 followed by 30 is rejected at the second step. |
| DOM-042 | MUST | A reversal or adjustment MUST cite one authority: (a) a dispute outcome (DOM-053), or (b) a recorded mutual agreement of both parties. A network administrator MAY NOT reverse unilaterally (see OD-09). | Stops unilateral rewriting of trades. | Adjustment without an authority reference is rejected. |
| DOM-043 | MUST | Reversals and adjustments are subject to the credit limit check (DOM-027), except that a correction MAY take the provider below the floor if the posting is flagged as a correction and audited (see OD-06). Such a flag blocks further debits per DOM-029. | Corrections must not be blocked by the very limit they correct, but must be visible. | Correction that breaches floor succeeds only with the flag; the audit event records the breach; a subsequent debit is rejected. |

## Disputes

| ID | Pri | Requirement | Rationale | Test criterion |
|---|---|---|---|---|
| DOM-050 | MUST | A dispute references exactly one COMPLETED transaction. Only a participant (provider or receiver) may raise it, within the network dispute window (see OD-02). | Bounded, attributable challenge. | Dispute by a non-participant, against a non-COMPLETED transaction, or after the window is rejected. |
| DOM-051 | MUST | A dispute reviewer MUST NOT be a participant in the disputed transaction. | Impartiality. | Assigning a participant as reviewer is rejected. |
| DOM-052 | MUST | Raising a dispute does not change balances or transaction status. The transaction carries an "open dispute" marker. | Disputes are review, not automatic reversal. | Before resolution, balances and status are unchanged. |
| DOM-053 | MUST | Dispute outcomes are: UPHELD (no entries), REVERSED (full adjustment per DOM-041), or PARTIALLY ADJUSTED (partial adjustment). Each outcome records a reason and an audit event. | Closed set of outcomes. | Each outcome produces the specified entries (none, full, or partial) and an audit event with reason. |
| DOM-054 | MUST | At most one open dispute exists per transaction. Resolution is final; re-opening requires a new dispute on a COMPLETED transaction still within the window (see OD-02). | Prevents conflicting reviews. | Second concurrent dispute on the same transaction is rejected. |

## Member departure and default

| ID | Pri | Requirement | Rationale | Test criterion |
|---|---|---|---|---|
| DOM-060 | MUST | Departure moves an account from ACTIVE to DEPARTED, initiated by the member or a network administrator. Before the transition, PROPOSED transactions are cancelled and ACCEPTED transactions are completed or cancelled (DOM-013, DOM-015). A DEPARTED account accepts no new proposals, acceptances, or confirmations. | Clean cut-off of in-flight trades. | Departure with an open ACCEPTED transaction is blocked until that transaction is resolved. After departure, new proposals involving the account are rejected. |
| DOM-061 | MUST | A DEPARTED account's entries and balance are retained. Its deletion is not permitted. The network balance sum remains zero. | History and conservation. | Departed account's entries are readable; network sum check (DOM-020) still passes. |
| DOM-062 | MUST | A non-zero balance at departure is never silently zeroed. Any settlement is a traceable adjustment under the network's settlement rule (see OD-04). Default: no settlement; the balance is retained and flagged. | Silent zeroing breaks DOM-022. | Departure with non-zero balance produces no entries and a flagged record; any settlement produces adjustment entries with authority. |
| DOM-063 | MAY | A network administrator MAY SUSPEND an ACTIVE account for non-performance or other network-defined cause. Suspension changes standing only; it posts no entries. A SUSPENDED account MAY be reinstated to ACTIVE by a network administrator. | Default handling is a standing change, not a ledger change. | Suspension writes zero entries and changes zero balances; reinstatement restores ACTIVE; both are audited with reason. |
| DOM-064 | MUST | The platform MUST NOT write off or forgive any balance automatically. Write-off, if any, is a network decision recorded as adjustment entries (see OD-05). | Forgiveness is a governance and accounting choice, not software behavior. | No automatic job creates adjustment entries without a recorded authority. |

## Audit

| ID | Pri | Requirement | Rationale | Test criterion |
|---|---|---|---|---|
| DOM-070 | MUST | Each ledger posting, status transition, dispute event, standing change, adjustment, and policy change produces an audit event with: actor, action, timestamp, reason (required for administrative actions), and reference ids. | Accountability for every change. | Each listed action produces exactly one audit event with all fields populated; missing reason on an administrative action is rejected. |
| DOM-071 | MUST | Audit events are append-only. | Audit must be trustworthy. | Update or delete of an audit event is rejected. |
| DOM-072 | MUST | Policy and standing changes record old and new values. | Makes configuration history reconstructable. | After changing a credit limit from 100 to 50, the audit event shows 100 and 50. |

## Network policy and credit semantics

| ID | Pri | Requirement | Rationale | Test criterion |
|---|---|---|---|---|
| DOM-080 | MUST | Network administrators MAY configure the policies listed in "Configurable vs invariant." Policy changes apply to postings and transitions after the change time. They do not alter existing entries. | Policy is per network; history is fixed. | A policy change does not change any existing entry; a posting after the change uses the new value. |
| DOM-081 | MUST | No network configuration may disable, weaken, or override a platform invariant (DOM-020 to DOM-026). Attempts are rejected. | Keeps ledger integrity uniform across networks. | Configuration attempting to disable atomicity or append-only is rejected. |
| DOM-082 | MUST | Credit is an accounting unit only. No user-facing text, API field description, or documentation produced under this spec states or implies that credit is money, redeemable for national currency, a security, or a cryptocurrency. See [_context.md, Credit semantics](./_context.md#credit-semantics). | Required by the shared context. | Review of all user-facing strings and API descriptions finds no such claim. |
| DOM-083 | MUST | Acceptance of credit by members is not guaranteed by the software. The platform MUST NOT require a member to accept a payment in credit. | Acceptance is a network matter. | Member can decline or list only offers they choose; no code path forces acceptance. |

### Configurable vs invariant

| Item | Class | Owner | Notes |
|---|---|---|---|
| Ledger invariants 1 to 7 (DOM-020 to DOM-026) | Platform invariant | Platform | Not configurable. See [_context.md](./_context.md#ledger-invariants-platform-wide-not-configurable). |
| Credit unit as accounting unit (DOM-082) | Platform invariant | Platform | Not configurable. |
| Transaction state machine (DOM-010) | Platform invariant | Platform | Shape fixed; timing parameters configurable. |
| Two-party, equal-and-opposite postings (DOM-009, DOM-016) | Platform invariant | Platform | Not configurable. |
| Audit event requirements (DOM-070 to DOM-072) | Platform invariant | Platform | Not configurable. |
| Credit limit default and per-account override (DOM-005) | Network policy | Network administrator | Floor only; enforcement is invariant (DOM-027). |
| Positive-balance ceiling (DOM-030) | Network policy | Network administrator | Default none. |
| Dispute window (DOM-050) | Network policy | Network administrator | Default in OD-02. |
| Acceptance window (DOM-014) | Network policy | Network administrator | Default in OD-07 area; see Open decisions. |
| Join rules | Network policy | Network administrator | Defined in GOV-* (planned). |
| Visibility defaults for offers and requests | Network policy | Network administrator | Defined in UX-* / SEC-* (planned). |
| Completion confirmation model (DOM-015) | Platform default, network-selectable | Network administrator | See OD-01. |
| Settlement on departure (DOM-062) | Network policy | Network administrator | See OD-04. |
| Listing expiry (DOM-006, DOM-007) | Network policy | Network administrator | See OD-11. |

## Open decisions

Each item lists options and the default the spec uses until a decision is recorded. Defaults are not legal conclusions.

- **OD-01 Completion confirmation.** Options: (a) both parties confirm; (b) provider confirms only; (c) network selects a or b. **Default: (a) both parties.**
- **OD-02 Dispute window.** Options: (a) fixed platform value; (b) network-configurable within platform bounds; (c) no window. **Default: (b), default value 30 days from completion.**
- **OD-03 Positive-balance ceiling.** Options: (a) none; (b) network-configurable ceiling. **Default: (a) none.**
- **OD-04 Settlement on departure with non-zero balance.** Options: (a) retain balance and flag, no entries; (b) network-defined adjustment to a network clearing account; (c) per-case administrator adjustment with authority. **Default: (a).**
- **OD-05 Write-off and default resolution.** Options: (a) suspension only, no write-off; (b) network-chosen write-off as adjustment entries with authority; (c) platform automatic write-off. **Default: (a). (c) is excluded by DOM-064.**
- **OD-06 Correction beyond the credit floor.** Options: (a) reject; (b) allow with correction flag and audit, debits blocked until back within floor; (c) allow silently. **Default: (b). (c) is excluded by DOM-022.**
- **OD-07 Cancellation of an accepted transaction.** Options: (a) both parties agree; (b) either party before completion; (c) network-defined. **Default: (a).** Acceptance window (DOM-014) default: 7 days, network-configurable.
- **OD-08 Amount precision.** Options: (a) integer minor units, fixed by network; (b) decimal with fixed scale. **Default: (a), fixed by network at creation.** Changing scale after postings is not permitted.
- **OD-09 Reversal authority.** Options: (a) dispute outcome or both-party agreement only (DOM-042); (b) network administrator may also reverse with recorded reason. **Default: (a).**
- **OD-10 Accounts per member per network.** Options: (a) one; (b) several with separate ledgers. **Default: (a).**
- **OD-11 Listing expiry.** Options: (a) no expiry; (b) network-configured expiry. **Default: (b), 90 days.**
- **OD-12 Legal characterization of credit.** Whether any jurisdiction treats credit or its acceptance as regulated is not decided here. No conclusion is drawn. **Default: none asserted; route to governance and legal review (GOV-* planned) before any public claim.**

## Related docs

- [_context.md](./_context.md): shared terms, normative keywords, glossary, ledger invariants, credit semantics.
- Planned, not yet present in `docs/specs/`: GOV-* (governance, join rules), DAT-* (ledger storage and audit schema), API-* (interfaces), SEC-* (security and privacy), UX-* (user experience), FR-* (functional requirements). Links will be added when those specs exist.
