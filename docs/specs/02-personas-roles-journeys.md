# Personas, Roles and Core Journeys
Status: Draft | Owner: Product architect (role) | Updated: 2026-11-14 | Scope: roles, permissions and core member/administrator journeys for one network | Deps: [_context.md](./_context.md) | Open questions: 10, see "Open decisions"

## Purpose
Define who uses a Circula network, what each role may do, and the required behavior of the core journeys at requirements level. Rules that other specs own (ledger invariants, credit semantics, term definitions) are not restated here; they are referenced by spec prefix and by [_context.md](./_context.md).

Normative terms (MUST, SHOULD, MAY) and glossary terms (network, account, balance, credit limit, offer, request, transaction, dispute, adjustment, audit event) follow [_context.md](./_context.md).

Personas below are illustrative planning aids, not research findings.

## Personas

| Persona | Role | Typical goal | Main constraint |
|---|---|---|---|
| Provider member | Member | Offer goods or services and build a positive balance | Needs to see what others need and trust a completed exchange |
| Requester member | Member | Obtain goods or services and spend earned credit | Needs to see available offers and own remaining credit limit |
| Steward | Network administrator | Keep the network's rules current and its membership healthy | Must act within network policy; cannot edit balances directly |
| Neutral resolver | Dispute reviewer | Decide disputes fairly on recorded facts | Must not review disputes they are part of |
| Software operator | Platform operator | Keep the software running, upgraded and recoverable | Must not act as a participant or decision-maker in any network |

## Requirements

### Roles and general rules

**ROL-001** (MUST) The product MUST define exactly four roles: member, network administrator, dispute reviewer, platform operator.
- Rationale: these are the glossary roles; other specs depend on this set.
- Criterion: every role appears in the permissions matrix, and every matrix action names a role or "none".

**ROL-002** (MUST) Role grants MUST be scoped to one network. A grant in network A MUST confer no right in network B.
- Rationale: each network sets its own rules.
- Criterion: a test user holding admin in network A receives a denied response for every admin action in network B.

**ROL-003** (MUST) A person MAY hold several roles in the same network. Effective rights MUST be the union of the rights of the held roles, except as limited by ROL-004, ROL-005 and ROL-021.
- Criterion: a person who is both member and administrator can perform both sets of actions; the same person cannot review a dispute they are party to.

**ROL-004** (MUST) A dispute reviewer MUST NOT review a dispute in which the reviewer is a participant in the disputed transaction.
- Criterion: assignment logic excludes participants; a forced assignment attempt is refused and audited.

**ROL-005** (MUST) The platform operator role MUST NOT grant any right to post entries, change balances, decide disputes, or change network policy in any network.
- Criterion: an operator account calling each of these actions is refused.

**ROL-006** (MUST) Every role grant, revocation, or role-scope change MUST produce an audit event containing actor, target, role, network, timestamp, and reason.
- Criterion: audit event exists for each grant or revocation test case.

**ROL-007** (MUST) Role-facing copy MUST NOT state or imply that credits have monetary value or are redeemable. Wording follows the credit semantics section of [_context.md](./_context.md).
- Criterion: copy review checklist passes for every screen named in this spec.

**ROL-008** (SHOULD) The product SHOULD make each role's current rights visible to the holder (for example, a "what I can do here" view).
- Criterion: a holder can list their effective rights in each network they belong to.

### Member join journey

**ROL-010** (MUST) A prospective member MUST join a network only through the network's join rule (see OD-1).
- Criterion: each join rule value produces only its defined path; no other path creates an account.

**ROL-011** (MUST) A completed join MUST create exactly one account in that network with a balance of zero and no entries.
- Criterion: after join, the account balance is zero and the entry list is empty.

**ROL-012** (MUST) While a join request awaits decision, the request MUST be visible only to the requester and to network administrators.
- Criterion: other members and dispute reviewers receive a not-found or denied response for the pending request.

**ROL-013** (MUST) A denied or expired join request MUST notify the requester with a reason category. The notification MUST NOT name other members or reveal membership lists.
- Criterion: notification content is checked against a fixed list of allowed fields.

**ROL-014** (MUST) A repeated join request for the same person and network while one is pending MUST return the existing pending request and MUST NOT create a second one.
- Criterion: two submissions in quick succession produce one request record.

**ROL-015** (SHOULD) Before final confirmation, the join flow SHOULD present the network's current credit limit, dispute window, and governance summary, and SHOULD require an explicit acknowledgment.
- Criterion: the confirm control is not reachable until the acknowledgment is recorded.

### Offer and request journey

**ROL-020** (MUST) An active member MUST be able to create, edit, pause, and close their own offers and requests within their network.
- Criterion: each operation succeeds for the owner and is denied for other members.

**ROL-021** (MUST) A network administrator MAY remove an offer or request that breaks network rules. The removal MUST record a reason and an audit event, and MUST NOT alter any balance.
- Criterion: removal sets a removed status with reason; balances and entries are unchanged.

**ROL-022** (MUST) Creating, editing, pausing, or closing an offer or request MUST NOT change any balance or create any entry.
- Criterion: ledger entry count and balances are identical before and after each operation.

**ROL-023** (MUST) Offers and requests MUST be visible only to members of the same network, unless the network visibility policy (OD-5) says otherwise.
- Criterion: a non-member or member of another network cannot list or fetch them.

**ROL-024** (SHOULD) Members SHOULD be able to search offers and requests within their network by text and category.
- Criterion: search returns only same-network items matching the query.

### Propose and accept trade journey

**ROL-030** (MUST) An active member MUST be able to propose a transaction to another active member in the same network. A proposal MUST state counterpart, amount in credit units, description, and MAY reference an offer or request.
- Criterion: proposals missing any required field are refused with a field-level reason.

**ROL-031** (MUST) Creating a proposal MUST create a transaction in status "proposed" and MUST NOT change any balance.
- Criterion: balances are unchanged after proposal creation.

**ROL-032** (MUST) Only the counterpart MUST be able to accept or decline a proposal. Only the proposer MUST be able to withdraw it before acceptance. A proposal not acted on before its expiry MUST move to "expired".
- Criterion: each actor/action pair not listed is denied; expiry transition is tested at the boundary.

**ROL-033** (MUST) Before accepting, the system MUST check that the acceptance would not take the paying account below its credit limit (see [_context.md](./_context.md), credit limit). If it would, acceptance MUST be refused with a reason that names the limit condition and does not reveal the other party's balance.
- Criterion: acceptance at exactly the limit succeeds; one unit over is refused.

**ROL-034** (MUST) Completion of an accepted transaction MUST post balanced entries atomically, as required by the ledger invariants owned by the DAT specs. The journey MUST NOT show completion until the posting is confirmed.
- Criterion: a simulated failure during posting leaves the transaction in "accepted" with no partial entries.

**ROL-035** (MUST) Repeated accept or complete actions for the same transaction (double submit, client retry) MUST produce one posting.
- Criterion: five identical requests produce exactly one posting.

**ROL-036** (MUST) The UI MUST show transaction status at all times (proposed, accepted, completed, declined, withdrawn, expired, disputed) with the same meaning for both parties.
- Criterion: both parties see identical status after each transition.

**ROL-037** (MUST) Either participant MUST be able to raise a dispute on an accepted or completed transaction within the dispute window. Journey detail is in the dispute journey (ROL-050 onward).
- Criterion: raising a dispute after the window is refused with a reason.

### Balance and history journey

**ROL-040** (MUST) A member MUST be able to view their own balance, credit limit, available headroom (credit limit plus balance), and their own entries and transactions.
- Criterion: values match a recomputation from entries (see ROL-042).

**ROL-041** (MUST) History MUST be shown as an append-only sequence. A reversal or adjustment MUST appear as a new item that references the original; the original MUST remain visible.
- Criterion: a corrected transaction shows both the original and the adjustment.

**ROL-042** (MUST) A displayed balance MUST equal the sum of the account's posted entries. Any mismatch MUST be treated as a defect.
- Criterion: automated check compares displayed balance with recomputed balance for test accounts.

**ROL-043** (MUST) A member MUST NOT see another member's balance or entries, except that a counterpart MAY see the amount and status of a transaction they are party to (see OD-4).
- Criterion: cross-account view requests return denied unless a shared transaction exists.

**ROL-044** (MUST) A network administrator MUST be able to view account balances and entry history for accounts in their network, subject to OD-6.
- Criterion: administrator views succeed for same-network accounts and fail for other networks.

**ROL-045** (MAY) A member MAY export their own history in a documented, machine-readable format.
- Criterion: export contains the same entries shown in the history view.

**ROL-046** (MUST) When displayed data may be stale (network unavailable or last sync older than a defined threshold), the UI MUST label the balance as possibly out of date.
- Criterion: with simulated disconnection, the stale label appears and is cleared on resync.

### Dispute journey

**ROL-050** (MUST) Only a participant of a transaction MUST be able to raise a dispute on it, within the network's dispute window.
- Criterion: non-participants and late submissions are refused.

**ROL-051** (MUST) A dispute MUST include a reason category and a description. A dispute with an empty description MUST NOT be accepted.
- Criterion: validation refuses empty description; accepts a valid one.

**ROL-052** (MUST) Raising a dispute MUST NOT change balances. It MUST set the transaction to "disputed" and MUST prevent completion posting while the dispute is open.
- Criterion: balances are unchanged after dispute creation; completion action is refused while open.

**ROL-053** (MUST) A dispute MUST be assigned to a dispute reviewer from the network's current reviewer roster, excluding participants (ROL-004). Assignment MUST be audited.
- Criterion: assignment never selects a participant; assignment event exists.

**ROL-054** (MUST) Both participants MUST be able to see the dispute status and to submit a statement. The assigned reviewer MUST be able to see both statements and the transaction record.
- Criterion: each party sees the other's statement after submission; non-assigned reviewers are denied.

**ROL-055** (MUST) A reviewer's decision MUST be one of: uphold the transaction, or resolve it by posting traceable adjustments (DAT). Each decision MUST include a reason and an audit event.
- Criterion: decisions without a reason are refused; adjustments reference the original entries.

**ROL-056** (MUST) A dispute resolution MUST NOT edit or delete posted entries. Its effect MUST appear only as new entries (see ledger invariants, [_context.md](./_context.md)).
- Criterion: original entries are unchanged after resolution.

**ROL-057** (MUST) A platform operator MUST NOT decide disputes, post adjustments, or see dispute statements unless a network administrator grants a support path (see OD-7).
- Criterion: operator access to dispute content is denied by default.

**ROL-058** (SHOULD) The product SHOULD show each party an expected next step and a timeframe for the dispute, drawn from network policy.
- Criterion: dispute view shows the current state and the policy-defined timeframe.

### Rule administration journey

**ROL-060** (MUST) A network administrator MUST be able to view and change the network policy values that [_context.md](./_context.md) lists as configurable: credit limit values, join rules, dispute window, visibility defaults, and the reviewer roster.
- Criterion: each listed setting can be changed by an administrator and by no other role.

**ROL-061** (MUST) Each policy change MUST record old value, new value, actor, timestamp, and reason in an audit event.
- Criterion: change history contains all five fields for every change.

**ROL-062** (MUST) A policy change MUST apply only to transactions and actions created after it takes effect. It MUST NOT alter posted entries or the outcome of existing transactions or disputes.
- Criterion: an existing transaction keeps its original limit evaluation after a limit change.

**ROL-063** (MUST) Lowering a credit limit below an account's current balance floor MUST NOT alter existing balances. Further debits that would breach the new limit MUST be refused until the account is back within limit.
- Criterion: balance is unchanged after limit reduction; the next debit is refused.

**ROL-064** (MUST NOT) A network administrator MUST NOT edit an account balance or post an entry directly. Balance changes MUST come only from transactions, completions, or adjustments.
- Criterion: no administrator action in the product writes an entry without a transaction or adjustment reference.

**ROL-065** (SHOULD) Before confirming a join-rule or credit-limit change, the administrator SHOULD see a preview of the number of accounts affected.
- Criterion: preview count matches a test query over accounts.

**ROL-066** (MUST) A network administrator MUST be able to assign and revoke dispute reviewers. A change MUST apply to disputes opened after the change and MUST be audited.
- Criterion: revoking a reviewer does not reassign open disputes unless the admin's documented procedure does so (see OD-8).

### Accessibility and error states

**ROL-070** (MUST) Every journey in this spec MUST be completable with keyboard only and with a screen reader. Concrete interaction rules belong in the UX specs (prefix UX), which this spec does not restate.
- Criterion: each journey passes a keyboard-only walkthrough and an assistive technology walkthrough.

**ROL-071** (MUST) Each journey MUST define a user-visible outcome for these error states: permission denied, credit limit reached, stale data, duplicate submission, network interruption, and validation failure.
- Criterion: a checklist of the six states has an outcome for every journey in this spec.

**ROL-072** (MUST) Error messages MUST name the cause and the next available action, and MUST NOT disclose another member's private data.
- Criterion: message review confirms cause plus action and no private fields.

**ROL-073** (MUST) Status changes (for example accepted, disputed, completed, expired) MUST be conveyed to assistive technology without requiring the user to move focus.
- Criterion: automated and manual check confirm programmatic announcement for each status change.

**ROL-074** (MUST) Status and error meaning MUST NOT rely on color alone.
- Criterion: each status also carries text or an icon with a text alternative.

**ROL-075** (MUST) A state change that is pending or not yet confirmed MUST be shown as pending, not as final.
- Criterion: an accepted-but-not-posted transaction never displays as completed.

## Permissions matrix

Key: Y = permitted; N = denied; Own = only on records the actor owns or is party to; Assigned = only on the dispute assigned to the actor; Policy = depends on OD values; Sys = system process only.

| Action | Member | Network admin | Dispute reviewer | Platform operator |
|---|---|---|---|---|
| Join network (via join rule) | Y (as applicant) | N | N | N |
| Approve or deny join request | N | Y | N | N |
| View own balance, entries, history | Own | Own (if also a member) | N | N |
| View other accounts' balances | Policy (counterpart on shared transaction only, OD-4) | Y in own network (OD-6) | N | N |
| Create, edit, close own offer or request | Own | Own (if also a member) | N | N |
| Remove an offer or request for rule breach | N | Y (audited) | N | N |
| Propose a transaction | Y | Own (if also a member) | N | N |
| Accept or decline a proposal | Own (as counterpart) | Own (if also a member) | N | N |
| Withdraw a proposal | Own (as proposer) | Own (if also a member) | N | N |
| Complete a transaction | Own (party) | N | N | N (Sys posts) |
| Raise a dispute | Own (as participant) | Own (if also a participant) | N (when participant, ROL-004) | N |
| Submit dispute statement | Own (as participant) | Own (if also a participant) | N | N |
| View dispute content | Own (as participant) | N | Assigned | N (OD-7) |
| Decide dispute | N | N | Assigned | N |
| Post adjustment from a decision | Sys | N | Assigned (via decision) | N |
| Edit an account balance directly | N | N | N | N |
| View or change network policy | N | Y | N | N |
| Assign or revoke dispute reviewers | N | Y | N | N |
| View network audit events | Own-related only | Y (own network) | Assigned dispute only | Operational logs only (no ledger content, OD-7) |
| Grant or revoke any role | N | Y (own network, ROL-006) | N | N |
| Deploy, upgrade, back up, restore platform | N | N | N | Y |
| Read ledger contents of a network | N | Y (ROL-044) | Assigned dispute only | N (OD-7) |

## Open decisions

**OD-1 Join rule.** How a person joins a network.
- Options: (a) invite-only; (b) open join with network-defined terms; (c) join request approved by an administrator.
- Default: (c). Admin approval, with invitation as an additional path. Reason: safest without legal or policy input.

**OD-2 Accounts per person per network.**
- Options: (a) one account per person per network; (b) several accounts per person, each with separate balances.
- Default: (a). One account per person per network. Simplifies ROL-011 and ROL-043.

**OD-3 Who confirms completion.**
- Options: (a) both parties confirm; (b) proposer confirms; (c) system completes at a set time after acceptance.
- Default: (a). Both parties confirm before posting.

**OD-4 Counterpart visibility of balances.**
- Options: (a) counterpart sees only the amount and status of shared transactions; (b) counterpart also sees the other's balance; (c) network-wide visibility.
- Default: (a). Narrowest visibility. Changing it is a network policy decision.

**OD-5 Offer and request visibility.**
- Options: (a) members only; (b) public listing; (c) network-configurable.
- Default: (c), with members-only as the default value.

**OD-6 Administrator view of balances and entries.**
- Options: (a) full balance and entry access; (b) balances only, no entry detail; (c) aggregate only.
- Default: (a) for network administrators, with an audit event on each view of another account's entries. Alternative (b) if privacy review requires it.

**OD-7 Platform operator access to network content.**
- Options: (a) no access to ledger or dispute content ever; (b) time-limited support access granted by the network administrator and audited; (c) unrestricted.
- Default: (b) for support, with (a) as the baseline when no grant exists. Option (c) is rejected by ROL-005.

**OD-8 Reviewer revocation with open disputes.**
- Options: (a) open disputes keep their assigned reviewer; (b) reassign open disputes to the remaining roster automatically; (c) administrator must reassign manually.
- Default: (a). Least change to in-progress decisions.

**OD-9 Appeal of a dispute decision.**
- Options: (a) no appeal, decision final; (b) one appeal to a different reviewer; (c) network-defined appeal path.
- Default: (b). One appeal, to a reviewer not previously involved. Escalation rules belong in the GOV specs.

**OD-10 Leaving a network.**
- Options: (a) account frozen with balance retained and no further trades; (b) account closed after all open transactions complete; (c) account balance settled by network-defined process.
- Default: (b). Open transactions and disputes must finish first. Settlement semantics are a network decision and are not defined by the software.

## Related docs
- [_context.md](./_context.md): shared terms, normative terms, ledger invariants, configurable vs invariant rules.
- Planned, not yet written, referenced by prefix only: DOM (domain rules), DAT (ledger, audit), SEC (security, privacy), UX (interaction and accessibility), GOV (governance and policy changes), FR (functional requirements).
