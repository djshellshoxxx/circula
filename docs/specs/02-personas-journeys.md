# 02 Personas, Roles and Core Journeys

Status: Draft v0.1 · Owner: Product architect · Last updated: 2026-10-21 · Scope: the four roles, permission matrix, core journeys with accessible and error states, at requirements level · Depends on: [00-conventions.md](00-conventions.md), [01-vision-scope.md](01-vision-scope.md), [03-domain-model.md](03-domain-model.md) (rules), [04-functional-requirements.md](04-functional-requirements.md) (capabilities), [08-ux-accessibility.md](08-ux-accessibility.md) (screens), [09-security-privacy.md](09-security-privacy.md) (authentication and authorization) · Open questions: section 7

Personas are behavioural descriptions used to derive requirements. They are not market segments and do not describe real individuals. Trade rules are not restated here; journeys link to the authoritative rule IDs in [03-domain-model.md](03-domain-model.md).

---





## Build status (as built)

Snapshot: HEAD `925f637` (branch claude/circula-build). Statuses were first read at `31b1015` and re-checked at `925f637` (see TRACEABILITY.md). **built** = behaviour exists in code; **partial** = exists with the gaps stated, or the required verification was not run; **not built**; **deferred** = out of MVP by this spec or a recorded decision. Code paths and test names: [TRACEABILITY.md](../TRACEABILITY.md). Open owner decisions and resolved questions: [DECISIONS.md](../DECISIONS.md). Status is implementation status, not a compliance claim; no release gate has been run.

Summary: built 4, not built 1, partial 10.

| ID | Priority | Status | Note |
|---|---|---|---|
| PER-001 | P0 | built | Member role: balance, listings, trades, disputes, export |
| PER-002 | P0 | partial | Admin limits and self-restrictions built; dual control, notices and reviewer-conflict audit partly missing |
| PER-003 | P0 | partial | Reviewer decides with independence checks; admin without reviewer role can also decide (open decision DEC-072) |
| PER-004 | P0 | partial | No operator role routes; break-glass and operator access record not built |
| PER-010 | P0 | partial | Invite, approval and open joins built; onboarding terms (GOV-009) not built |
| PER-011 | P0 | built | Create, edit, pause, archive listing; browse by type and text |
| PER-012 | P0 | built | Propose, accept, decline, cancel; review step; idempotent commands |
| PER-013 | P0 | partial | Balance and history views built; stale-data indicator (PER-022) not built |
| PER-014 | P0 | partial | Dispute open, review, other-party response (FR-076) and decision built |
| PER-015 | P0 | partial | Admin controls built; limit warnings, dual control and network suspension not built |
| PER-016 | P1 | partial | Departure with blockers built; counterparty notices and self-export offer not built |
| PER-017 | P1 | partial | CloseNetwork in service only; no web or API route calls it |
| PER-020 | P0 | built | Not-found is uniform for other-network and missing ids |
| PER-021 | P0 | partial | Empty-state copy in templates; no UX test run |
| PER-022 | P1 | not built | No stale-data indicator on balance views |

## 1. Roles

A **role** is a permission set scoped to one network. A person MAY hold several roles in one network (for example, member and administrator) and MUST hold a separate membership in each network they join.

### PER-001 · P0 · Network member (role)
- **Who.** A person with an active membership in one network who gives or receives value through trades.
- **Goals.** Find a trade partner; propose and complete trades; know what they have earned and what credit remains available; contest a trade they believe is wrong.
- **Constraints.** May have low digital literacy, limited time, assistive technology needs, or intermittent connectivity. Must not need to understand double-entry accounting to read their balance.
- **Rationale.** Members are the primary users; G3, G4, G5.
- **Acceptance.**
  - A member can complete the journeys J1 to J5 in section 3 using only the member-facing screens (verified in usability testing and scripted end-to-end tests, see [12-quality-test-plan.md](12-quality-test-plan.md)).
  - No member-role action is available to a non-member, a suspended member, or a departed member, except viewing their own history and exporting their own data (automated authorization tests).

### PER-002 · P0 · Network administrator (role)
- **Who.** A trusted member appointed to operate one network's rules and membership.
- **Goals.** Set up the network; approve or invite members; set limits and policies; make audited corrections; keep the network's governance records; hand over administration when needed.
- **Constraints.** Administrators are themselves members of the network and can be parties to trades. They are a misuse risk (see [09-security-privacy.md](09-security-privacy.md)).
- **Rationale.** Network governance is per network (principle 4).
- **Acceptance.**
  - Every administrator action is audited with the actor, reason where required, and the target (LED-6; FR-080).
  - An administrator cannot approve, adjust or decide on a transaction in which they are a party (automated test; DOM-037).
  - A network always has at least one active administrator (FR-069).

### PER-003 · P0 · Dispute reviewer (role)
- **Who.** A member appointed to decide disputes for one network. May be the same person as an administrator, but not for disputes involving their own trades.
- **Goals.** Read the transaction and the parties' statements; decide dismissed, upheld or partially upheld; record a rationale.
- **Constraints.** Must not see more personal data than needed to decide a dispute.
- **Rationale.** Disputes need a decision separate from routine administration (G5).
- **Acceptance.**
  - A reviewer cannot record a decision on a dispute where they are a party to the transaction (FR-072; DOM-038).
  - Reviewer views show only the dispute, the transaction and the two parties' display names and dispute statements, not the full ledger of either party (automated authorization test).

### PER-004 · P0 · Platform operator (role)
- **Who.** The person or organisation running a Circula deployment (hosted service provider, or the self-hosting owner who runs the software for one or more networks).
- **Goals.** Keep the service available; apply upgrades and migrations; take and test backups; respond to incidents; support networks with technical problems.
- **Constraints.** The operator is a separate concern from network governance: operators do not decide members' disputes or set a network's rules. Access to network data by operators is restricted and recorded (see [09-security-privacy.md](09-security-privacy.md); open question OQ-PER-2).
- **Rationale.** Separates platform operation from network governance (see [05-architecture.md](05-architecture.md)).
- **Acceptance.**
  - Operator actions that do not require network data (deploy, backup, restart, health check) are available without a network role.
  - No operator action can change a ledger entry, balance, transaction status or dispute outcome (automated authorization and migration tests; LED-2).
  - Any operator access to network content is recorded as an audit event (FR-082; SEC).

## 2. Permission matrix

The matrix states the minimum role required. "Self" means the actor is the member the record belongs to. "Not party" means the actor is not a party to the transaction concerned. Blank cells are denied. This matrix is a requirement summary; the authoritative rules are in the linked FR and DOM items.

| Action | Non-member | Member (active) | Member (suspended) | Administrator | Dispute reviewer | Platform operator |
|---|---|---|---|---|---|---|
| Join network (per join mode) | Yes (invite, approval, or open) | — | — | — | — | — |
| View network directory | — | Yes | No (view own history only) | Yes | — | — |
| Create or edit own listing | — | Yes | No | Yes (own only) | — | — |
| Propose trade | — | Yes (not to a suspended or departed member) | No | Yes (not as a reviewer on own trades) | — | — |
| Accept trade addressed to self | — | Yes | No | Yes | — | — |
| Decline trade addressed to self, or withdraw own proposal (exit only) | — | Yes | Yes (DOM-039) | Yes | — | — |
| View own balance and history | — | Yes | Yes | Yes | — | — |
| View another member's balance | — | No | No | Yes (limit and balance only, audited) | No | No |
| Export own data | — | Yes | Yes | Yes | — | — |
| Export network ledger | — | — | — | Yes | — | No (unless break-glass, OQ-PER-2) |
| Open dispute on own completed trade (within window) | — | Yes | No | Yes | — | — |
| Decide dispute | — | — | — | Yes, not on own trades | Yes, not on own trades | — |
| Set credit limit, suspend, make adjustment | — | — | — | Yes (adjustment: reason mandatory) | — | — |
| Change network policy | — | — | — | Yes (within bounds, audited) | — | — |
| Assign or remove administrator or reviewer | — | — | — | Yes (at least one admin remains) | — | — |
| Read audit history | — | Own-account summary only (FR-082) | Own-account summary only | Full network audit | Dispute-related audit | Operational logs only |
| Deploy, back up, restore, upgrade | — | — | — | — | — | Yes |
| Modify any ledger entry | — | — | — | — | — | — (never; LED-2) |

## 3. Core journeys

Each journey lists the trigger, main path, error and accessible states, and the requirements it depends on. Journeys are requirement-level; screen design is in [08-ux-accessibility.md](08-ux-accessibility.md).

### PER-010 · P0 · J1 Join a network
- **Trigger.** A person receives an invitation, requests to join, or follows an open join link (join mode is a network setting; DOM-003).
- **Main path.**
  1. Person opens the join link or invitation.
  2. Person identifies themselves through the authentication method the deployment provides ([09-security-privacy.md](09-security-privacy.md)).
  3. Person enters the minimum profile (display name) and accepts network rules.
  4. Invite or open mode: membership becomes active immediately. Approval mode: membership becomes pending until an administrator decides.
- **Error states.** Expired or already-used invitation: specific message and a route to request a new one. Name missing: field-level message. Denied application: reason shown if the administrator provided one.
- **Accessible states.** Pending membership is announced to the person in text, not only by colour or icon. Waiting state offers the expected approval timeframe if the network set one.
- **Requirements.** FR-010, FR-011, FR-012, FR-013, FR-016, FR-017.
- **Acceptance.** Automated journey test covers all three join modes; the pending and denied states are reachable and have text labels (screen-reader test per UX).

### PER-011 · P0 · J2 Offer or request goods and services
- **Trigger.** A member wants to advertise something they provide or want.
- **Main path.** Member creates a listing with type (offer or request), title, description and optional price in credits; listing becomes visible to active members; member can pause, edit or archive it.
- **Error states.** Title empty or too long: field-level message. Price not a positive amount in the network's decimal format: field-level message with an example in the network format.
- **Accessible states.** Listing type is stated in text ("Offer" or "Request"), not only by colour. Paused and archived states are stated in text.
- **Requirements.** FR-020, FR-021, FR-022, FR-023, FR-025.
- **Acceptance.** A listing created by J2 appears in directory search for another active member within one refresh; a paused listing does not.

### PER-012 · P0 · J3 Propose and accept a trade
- **Trigger.** A member wants to pay another member for value, or to invoice another member for value they provided.
- **Main path.**
  1. Proposer chooses counterparty (from directory or listing) and direction (proposer pays, or proposer will be paid).
  2. Proposer enters amount, optional note and optional listing reference.
  3. System shows the effect before submission: the amount, who is debited, who is credited, and the effect on the payer's available credit.
  4. Proposal is created in state `proposed`; the counterparty is notified in-app.
  5. Counterparty reviews and accepts. Acceptance posts both entries atomically and the transaction becomes `completed`.
- **Error states.**
  - Amount exceeds the payer's available credit: the message states the available amount in network format and suggests a smaller amount. The proposal is not created (DOM-012).
  - Counterparty is suspended or departed: proposal refused before it is created.
  - Acceptance fails because capacity changed in the meantime: acceptance is refused, the transaction stays `proposed`, the message explains that available credit changed, and the counterparty may decline or ask the proposer to adjust (DOM-014).
  - Proposal expired: the transaction is shown as `expired`, and both parties can see the expiry in history.
  - Duplicate submission (double click, retry): one transaction results (LED-4).
- **Accessible states.** Direction is stated in words ("You pay Ana 25.00" / "Ana pays you 25.00"). Pending and completed status are stated in text. Confirmation after accept is announced to assistive technology. Keyboard can complete the whole path.
- **Requirements.** FR-030, FR-031, FR-032, FR-033, FR-034, FR-035, FR-036, FR-038; DOM-011..DOM-014.
- **Acceptance.** End-to-end scenario E2E-02 in [12-quality-test-plan.md](12-quality-test-plan.md) passes; the balances shown afterwards equal the recomputed sums of the postings.

### PER-013 · P0 · J4 Review balance and history
- **Trigger.** A member wants to know their position.
- **Main path.** Member opens the balance view and sees: current balance, direction in words, credit limit, available credit, amount reserved by pending outgoing proposals, and the last completed transactions. Member opens the history and filters by status.
- **Error states.** Data temporarily unavailable: the page states which data could not be loaded and when it was last current; it does not show stale figures as current.
- **Accessible states.** Negative balance is described in words ("You have used 10.00 of your credit"), not only by a minus sign or colour. Available credit of zero is stated as "no credit available right now", not as an error. Each history row states direction, counterpart, status and date.
- **Requirements.** FR-050, FR-051, FR-053, FR-054; DOM-006, DOM-008.
- **Acceptance.** For a fixture network, the displayed balance and available credit equal the values computed from postings in the test oracle, and every history row is reachable by keyboard.

### PER-014 · P0 · J5 Raise a dispute
- **Trigger.** A member believes a completed trade is wrong (for example: service not provided, goods not as described, or a posting error).
- **Main path.**
  1. Member opens a completed transaction within the dispute window.
  2. Member chooses a reason category and writes a bounded statement.
  3. System marks the transaction `disputed`; postings do not change.
  4. Counterparty is notified and may respond (FR-076).
  5. A reviewer decides; the decision and rationale are shown to both parties.
- **Error states.** Dispute window closed: refused, with the closing date shown. Statement too long: field-level message. Member is not a party: refused.
- **Accessible states.** Status "disputed" is shown as a text label with the date. Decision outcome is stated in words ("Dismissed: the balance stands as before", "Upheld: a reversal of 25.00 has been posted").
- **Requirements.** FR-070, FR-071, FR-073, FR-075, FR-076, FR-077; DOM-017..DOM-021.
- **Acceptance.** Scenario E2E-04 passes; the dispute outcome is visible to both parties and to administrators with the rationale; no postings change when the dispute is opened.

### PER-015 · P0 · J6 Administer network rules
- **Trigger.** An administrator needs to set up, change or correct the network.
- **Main path.** Administrator sets up the network (FR-001, FR-002), changes configurable policies (FR-003) or a member's limit (FR-060), suspends or restores a member (FR-061), makes an audited adjustment (FR-062), approves or denies pending members (FR-011), assigns reviewers (FR-065), or hands over administration (FR-069).
- **Error states.** Invalid policy value: refused with the allowed range (DOM-029). Adjustment without reason: refused. Action that would leave no active administrator: refused. Action that would breach a member's limit: refused with the reason (LED-5).
- **Accessible states.** Confirmation dialogs for irreversible or financially material actions are operable by keyboard, name the target and the effect, and do not rely on colour alone.
- **Requirements.** FR-001..FR-003, FR-011, FR-060..FR-069.
- **Acceptance.** Each administrative action produces an audit event (LED-6) with reason where required; each refusal reason is tested.

### PER-016 · P1 · J7 Leave a network or be offboarded
- **Trigger.** A member chooses to leave, or an administrator removes a member under the network's rules.
- **Main path.** Member or administrator starts departure; system checks the departure preconditions (DOM-024); pending proposals are cancelled; the member keeps read-only access to their own history and export; the balance is retained as defined by the network's departure rule.
- **Error states.** Open disputes or proposed transactions block departure until resolved or cancelled, with the list of blockers shown.
- **Accessible states.** Blockers are listed as text links to the items, not only as counts.
- **Requirements.** FR-014, FR-015, FR-090.
- **Acceptance.** Departure with each blocker type is refused with the correct list; a successful departure leaves the member able to export their history.

### PER-017 · P1 · J8 Close a network
- **Trigger.** An administrator (with the network's governance process) wants to close a network.
- **Main path.** Administrator starts closure; system refuses while blockers exist; on success, new transactions stop, history stays readable to members, and exports are available (FR-068, FR-094).
- **Error states.** Blockers listed as in J7.
- **Accessible states.** As J7.
- **Requirements.** FR-068, FR-094.
- **Acceptance.** Closed network refuses all postings and proposals; read and export still work.

## 4. Error and empty states (requirements level)

### PER-020 · P0 · Error states are specific and safe
- **Statement.** Every user-facing error MUST state what happened, what the user can do next, and MUST NOT reveal whether another network or another member exists.
- **Rationale.** Tenant isolation (VIS-003) and usability.
- **Acceptance.** Automated test checks that not-found and unauthorized responses for other networks' identifiers are indistinguishable from "not found" for a non-existent identifier; message copy reviewed against a checklist in [08-ux-accessibility.md](08-ux-accessibility.md).

### PER-021 · P0 · Empty states explain the next step
- **Statement.** Empty directory, empty history, no pending proposals and no disputes each MUST show a sentence explaining what will appear and a next action, where one exists.
- **Acceptance.** Each empty state is covered by a UX test in [12-quality-test-plan.md](12-quality-test-plan.md).

### PER-022 · P1 · Loading and stale data
- **Statement.** Financial figures MUST NOT be shown as current when the view failed to refresh; the stale time is shown.
- **Acceptance.** Simulated network failure on the balance view shows a stale indicator with time; no figure is displayed without a freshness statement.

## 5. Permissions summary by requirement

- Authentication and session rules: [09-security-privacy.md](09-security-privacy.md).
- Route-level enforcement: [07-api-contract.md](07-api-contract.md). The UI MUST NOT be the only enforcement point: server-side authorization is authoritative, and the same checks apply to the web interface and the API.

## 6. Traceability

| Journey | Requirements |
|---|---|
| J1 Join | FR-010..FR-017 |
| J2 Listings | FR-020..FR-025 |
| J3 Trade | FR-030..FR-038, DOM-011..DOM-014 |
| J4 Balance | FR-050..FR-054 |
| J5 Dispute | FR-070..FR-078, DOM-017..DOM-021 |
| J6 Admin | FR-001..FR-003, FR-060..FR-069 |
| J7 Departure | FR-014, FR-015, DOM-024 |
| J8 Closure | FR-068, FR-094 |

## 7. Open questions

- **OQ-PER-1 (resolved as built).** Question moved to [DECISIONS.md](../DECISIONS.md): see DEC-053.
- **OQ-PER-2 (product owner and counsel-reviewed policy).** Whether platform operators may access network content for support, and under what break-glass controls. Current spec: no routine access; any access is audited and cannot alter ledger data.
- **OQ-PER-3 (resolved as built).** Question moved to [DECISIONS.md](../DECISIONS.md): see DEC-053.
- **OQ-PER-4 (resolved as built).** Question moved to [DECISIONS.md](../DECISIONS.md): see DEC-053.

Open questions are labelled `OQ-PER-n` and are to be consolidated into [13-assumptions-risks-decisions.md](13-assumptions-risks-decisions.md).
