# 04 Functional Requirements and Acceptance Criteria (MVP)

Status: Draft v0.1 · Owner: Product architect · Last updated: 2026-10-21 · Scope: MVP functional requirements for network setup, membership, directory and listings, trade proposal and acceptance, balances and history, administrator controls, disputes, audit history, CSV/JSON exports; explicit deferred list · Depends on: [01-vision-scope.md](01-vision-scope.md), [02-personas-journeys.md](02-personas-journeys.md), [03-domain-model.md](03-domain-model.md), [05-architecture.md](05-architecture.md), [06-data-ledger-audit.md](06-data-ledger-audit.md), [07-api-contract.md](07-api-contract.md), [08-ux-accessibility.md](08-ux-accessibility.md), [09-security-privacy.md](09-security-privacy.md), [10-governance-operations.md](10-governance-operations.md), [12-quality-test-plan.md](12-quality-test-plan.md) · Open questions: section 9

Terms are defined in [00-conventions.md §3](00-conventions.md#3-glossary-authoritative). Ledger rules are in [03-domain-model.md](03-domain-model.md) and are referenced by ID, not restated. Platform invariants are referenced as LED-n from [00-conventions.md §4](00-conventions.md#4-platform-wide-invariants-not-configurable).





## Build status (as built)

Snapshot: HEAD `925f637` (branch claude/circula-build). Statuses were first read at `31b1015` and re-checked at `925f637` (see TRACEABILITY.md). **built** = behaviour exists in code; **partial** = exists with the gaps stated, or the required verification was not run; **not built**; **deferred** = out of MVP by this spec or a recorded decision. Code paths and test names: [TRACEABILITY.md](../TRACEABILITY.md). Open owner decisions and resolved questions: [DECISIONS.md](../DECISIONS.md). Status is implementation status, not a compliance claim; no release gate has been run.

Summary: built 52, not built 3, partial 10.

| ID | Priority | Status | Note |
|---|---|---|---|
| FR-001 | P0 | built | Create network with policy; system account; audit |
| FR-002 | P0 | partial | One-time setup secret; used flag is per process, so a restart re-enables it while the secret is set |
| FR-003 | P0 | built | Policy change within bounds, versioned and audited |
| FR-004 | P0 | built | Setup to active once an admin exists |
| FR-005 | P0 | built | Decimals fixed after first posting |
| FR-006 | P0 | built | Cross-network identifier test covers web and API |
| FR-010 | P0 | built | Single-use invitation token hashed; redemption creates member |
| FR-011 | P0 | built | Approve and deny applicants; deny via API and web |
| FR-012 | P1 | built | Open join creates active member; join attempts rate limited (10 per 10 minutes per person and address, retry-later page) |
| FR-013 | P0 | built | Membership transitions table-tested |
| FR-014 | P0 | built | Departure in service, JSON API and a web "Leave this network" form that shows blockers and consequences |
| FR-015 | P0 | built | Admin removal with mandatory reason; last-admin guard |
| FR-016 | P1 | built | Display name is the only required profile field |
| FR-017 | P0 | built | Duplicate join returns existing membership |
| FR-020 | P0 | built | Directory returns active members only; keyset paging and exact counts since 925f637 |
| FR-021 | P0 | built | Create offer or request listing; no entries |
| FR-022 | P0 | built | Edit, pause, resume, archive own listing |
| FR-023 | P0 | built | Filter by type and text; paginated |
| FR-024 | P1 | not built | Member reports not built |
| FR-025 | P1 | built | Members-only pages require session; X-Robots-Tag, robots meta and robots.txt disallow |
| FR-030 | P0 | built | Propose to active member in same network |
| FR-031 | P0 | built | Amount, limit and effect preview on review page |
| FR-032 | P0 | built | Accept posts both entries atomically; limit re-checked |
| FR-033 | P0 | built | Decline and withdraw release reservation |
| FR-034 | P0 | built | Scheduled expiry, idempotent, injected clock |
| FR-035 | P0 | built | Idempotency on propose, accept, decline, cancel, reverse |
| FR-036 | P0 | built | Concurrent accepts for last credit: one succeeds |
| FR-037 | P1 | built | Plain-text note and listing reference on proposal |
| FR-038 | P1 | partial | Dashboard lists pending items; no inbox, unread state or outcome notices |
| FR-050 | P0 | built | Balance, limit, available, reserved and time shown |
| FR-051 | P0 | built | Reverse-chronological history with status and counterparty |
| FR-052 | P0 | built | Read-only ledger verification from admin page and CLI |
| FR-053 | P1 | built | Pending view paged; dashboard shows 100 newest pending |
| FR-054 | P1 | partial | BalanceAt in service only; no UI or API |
| FR-060 | P0 | built | Set or remove override with reason; audited |
| FR-061 | P0 | built | Suspend and restore with reason; audited |
| FR-062 | P0 | built | Admin adjustment with reason; two entries |
| FR-063 | P0 | built | List, revoke invitations; approve or deny join requests |
| FR-064 | P1 | built | Web admin plus PUT /admin/members/{id}/default-flag; service audit and payer restriction; API and contract regression tests |
| FR-065 | P1 | built | Assign or remove reviewer; reviewer cannot review own dispute |
| FR-066 | P1 | built | Admin reversal via API with reason |
| FR-067 | P1 | partial | Admin home shows pending-member count only |
| FR-068 | P1 | built | Closure in web and API; announced with notice (GOV-025); refuses with dispute blockers; members notified |
| FR-069 | P0 | built | Last-admin guard; role changes audited |
| FR-070 | P0 | built | Open dispute within window by party on completed trade |
| FR-071 | P0 | built | Postings unchanged on open, withdraw, dismiss |
| FR-072 | P0 | built | Party and proposer conflicts refused |
| FR-073 | P0 | built | Dismissed, upheld, partially upheld with rationale |
| FR-074 | P0 | built | Blocked reversal enters posting_pending; override then retry |
| FR-075 | P1 | built | Opener withdraws before decision |
| FR-076 | P1 | built | Response by other party; response-period policy (default 7 days) holds a decision unless a response exists or the reviewer waives it |
| FR-077 | P0 | built | Window measured from completion with server clock |
| FR-078 | P1 | not built | Re-review of a decision not built |
| FR-080 | P0 | built | Sweep: every command writes one audit event |
| FR-081 | P0 | built | Audit event carries required fields |
| FR-082 | P0 | partial | Admin reads full audit; member and reviewer views exist in service only |
| FR-083 | P0 | built | No update or delete path; triggers |
| FR-084 | P1 | built | Admin audit filters by action, actor, target, dates; paged |
| FR-085 | P2 | built | Hash chain built ahead of P2 deferral (DEC-044); not keyed |
| FR-090 | P0 | built | Member exports own history as CSV or JSON |
| FR-091 | P0 | built | Admin ledger export with manifest and checksums |
| FR-092 | P0 | built | Members export scope with role and limit fields |
| FR-093 | P0 | built | Audit export scope with manifest |
| FR-094 | P1 | built | Full export scope incl. closed networks |
| FR-095 | P1 | partial | Export buffered in memory by API adapter; no streaming or size gates |

## 0. How to read this file

- Each requirement has: **ID**, **priority** (P0 = MVP-blocking, P1 = MVP, P2 = post-MVP), **rationale**, and **acceptance criteria** written as observable, testable checks.
- Acceptance criteria name the observable result. Test layers and IDs are assigned in [12-quality-test-plan.md](12-quality-test-plan.md).
- Where a requirement depends on an open decision, it is marked **[OQ-FR-n]** and listed in section 9.
- The MVP is complete when every P0 and P1 requirement passes its acceptance criteria, or a deferral is recorded (VIS-006).
- Requirement IDs are never reused. A withdrawn requirement keeps its ID with the status "withdrawn".

---

## 1. Network setup

### FR-001 · P0 · Create a network
- **Rationale.** Networks are the tenant boundary and the place where policy lives (VIS-003, VIS-005).
- **Acceptance.**
  - An administrator can create a network by providing name, slug, and the configurable policy values in [00-conventions.md §5](00-conventions.md#5-configurable-network-policies-defaults) and [03-domain-model.md §13](03-domain-model.md#13-configuration-versus-invariants), or accept the defaults where a default exists.
  - Creation creates the network's system account (DOM-026) and an audit event (LED-6), and writes no entries.
  - Invalid values (wrong type, out of bounds, duplicate slug) are refused with the field name and the reason; nothing is created.
  - The slug is unique within the deployment and is not reused after deletion (tenant identifiers, FR-006).

### FR-002 · P0 · First administrator bootstrap
- **Rationale.** A new deployment needs a first administrator without a shared default password.
- **Acceptance.**
  - The first administrator is created through a setup step that requires a one-time setup secret produced at install time (see [09-security-privacy.md](09-security-privacy.md)).
  - After setup completes, the setup route returns "not found" or an equivalent refusal and cannot create an administrator.
  - The setup secret is not written to logs and is invalidated after first use (automated test).

### FR-003 · P0 · Change configurable policies
- **Rationale.** Each network sets its own rules (G2).
- **Acceptance.**
  - An administrator can change any configurable policy in DOM-029 within its bounds; out-of-bounds values are refused with the allowed range.
  - Each change writes an audit event with old value, new value, actor and reason (DOM-030, LED-6).
  - A change is applied only to transactions and decisions made after it (DOM-030): a test that changes the dispute window does not change the status of any existing dispute.

### FR-004 · P0 · Network activation
- **Rationale.** A network should not accept members before its configuration is complete.
- **Acceptance.**
  - A new network is in setup status until required policies are set and at least one administrator exists; then an administrator can activate it.
  - While in setup status, no membership can become active and no transaction can be proposed.

### FR-005 · P0 · Decimals fixed after first posting
- **Rationale.** DOM-031.
- **Acceptance.** Decimal changes are refused after the first entry; before it, the change is accepted and reflected in input formatting and exports.

### FR-006 · P0 · Network isolation in every interface
- **Rationale.** VIS-003, DOM-001.
- **Acceptance.** Cross-network identifier test passes for every route, export and listing; a non-existent identifier and an identifier from another network produce indistinguishable responses.

---

## 2. Membership

### FR-010 · P0 · Join by invitation
- **Rationale.** Invite mode is the most controlled join path (DOM-003).
- **Acceptance.**
  - An administrator can create an invitation with a single-use, expiring token. The token is not stored in plain text.
  - Redeeming a valid invitation creates an active member (invite mode) with the network's default limit.
  - Redeeming an expired, used or unknown token is refused with a text message and no membership is created.
  - Invitation by non-administrator members is an open question (OQ-FR-1).

### FR-011 · P0 · Join by approval
- **Rationale.** Approval mode lets an administrator vet applicants.
- **Acceptance.**
  - An applicant can submit a join request with a display name; it creates a `pending` membership (DOM-003).
  - An administrator can approve (membership becomes `active`) or deny (membership becomes `departed` with reason shown to the applicant if provided).
  - A pending member cannot propose, accept or list.
  - Each decision writes an audit event.

### FR-012 · P1 · Join by open link
- **Rationale.** Open mode is useful for low-risk communities.
- **Acceptance.**
  - When the join mode is `open`, anyone with the network's join link can create an active membership.
  - Joining is rate-limited per the abuse baseline in [09-security-privacy.md](09-security-privacy.md); exceeding it returns a retry-later message.
  - The administrator can view and suspend members created this way like any others.

### FR-013 · P0 · Membership lifecycle states
- **Rationale.** DOM-003 and DOM-004.
- **Acceptance.** Each permitted transition in DOM-003 succeeds with an audit event; each other transition is refused with no change (table-driven test).

### FR-014 · P0 · Voluntary departure
- **Rationale.** Members must be able to leave a network (J7).
- **Acceptance.**
  - A member can request departure; the system refuses while blockers exist (DOM-024) and lists each blocker as a link to the item.
  - When no blockers exist, departure completes per the network's departure rule, the membership becomes `departed`, and pending proposals are cancelled with audit events.
  - After departure the member can still export their own history (FR-090).

### FR-015 · P0 · Administrative offboarding
- **Rationale.** Networks must be able to remove members under their rules.
- **Acceptance.**
  - An administrator can start offboarding with a mandatory reason; the same blocker and departure rules apply as FR-014.
  - An administrator cannot offboard themselves if they are the last active administrator (FR-069).
  - The reason is stored in the audit event and is visible to administrators only.

### FR-016 · P1 · Member profile minimum
- **Rationale.** Data minimisation (VIS-010).
- **Acceptance.**
  - The only required profile field is a display name. Contact details, if supported, are optional and visible only as the member chooses (directory visibility rules, FR-020).
  - The profile schema has no field outside the data inventory (VIS-010).

### FR-017 · P0 · Duplicate join is idempotent
- **Rationale.** DOM-002.
- **Acceptance.** Joining a network with an identity that already has a membership in it returns the existing membership status and creates no second membership; repeated submissions with the same idempotency key return the same result (DOM-016).

---

## 3. Member directory and listings

### FR-020 · P0 · Directory visible to active members only
- **Rationale.** Members-only visibility is the MVP default (00 §5).
- **Acceptance.**
  - Only active members of the network can list other members' display names and listings.
  - Non-members, pending members, suspended members and departed members receive "not found" or an equivalent refusal for directory queries.
  - Directory responses contain no balances, limits or contact details.

### FR-021 · P0 · Create a listing
- **Rationale.** Offers and requests are how members find trades (J2).
- **Acceptance.**
  - An active member can create a listing with type `offer` or `request`, a title (required, bounded length), description (bounded), and an optional price in credits in the network's decimal format.
  - Creating a listing creates no entries (DOM-040).
  - Invalid price (zero, negative, too many decimals) is refused with a field-level message.

### FR-022 · P0 · Edit, pause, archive own listing
- **Rationale.** Members must keep listings accurate.
- **Acceptance.**
  - A member can edit, pause, resume and archive only their own listings; archived listings cannot be reactivated and must be recreated.
  - Paused and archived listings are excluded from search results and display their state in text.
  - Editing a listing does not change any transaction that referenced it (the reference is informational, DOM-040).

### FR-023 · P0 · Search and browse listings
- **Rationale.** Discovery (G3).
- **Acceptance.**
  - Members can filter active listings by type and free-text search over title and description.
  - Results are paginated (see [07-api-contract.md](07-api-contract.md)) and sorted by a stated order (default: most recently updated first).
  - A search returns no listing from another network (FR-006).

### FR-024 · P1 · Report a listing or member
- **Rationale.** Abuse reporting (SEC).
- **Acceptance.**
  - Any active member can report a listing or member with a category and a bounded note.
  - Reports appear in the administrator queue; reporting does not automatically hide, suspend or change anything.
  - The reporter's identity is visible to administrators only, and is not shown to the reported member.
- **Status (as built).** Built (service, API `POST /reports`, `GET /admin/reports`, web). Categories: spam, scam, inappropriate, harassment, other. One open report per reporter and target; at most 10 reports per reporter per 24 hours. The queue is administrator-only.

### FR-025 · P1 · Members-only visibility of listings
- **Rationale.** Prevents external indexing of member data by default (VIS-010).
- **Acceptance.** Listing and directory pages send a "no index" directive and are not served without authentication; a crawler test without session receives a refusal.

---

## 4. Trade proposal and acceptance

### FR-030 · P0 · Propose a trade
- **Rationale.** Core trade flow (G3, J3).
- **Acceptance.**
  - An active member can propose a trade to another active member in the same network, choosing the direction (proposer pays, or proposer will be paid).
  - The proposal creates a `proposed` transaction and a reservation against the payer's available credit (DOM-012).
  - The counterparty sees the proposal in their inbox (FR-038).
  - Proposing to self, to a non-member, or to a suspended or departed member is refused.

### FR-031 · P0 · Proposal validation and effect preview
- **Rationale.** Members must understand the effect before submitting (J3).
- **Acceptance.**
  - Amount must be a positive integer in minor units, no more than the maximum transaction amount, and no more than the payer's available credit (DOM-005, DOM-008, DOM-012).
  - Before submission, the UI shows payer, payee, amount, and the payer's available credit after the proposal (computed by the same rule as the server).
  - Each refusal returns a specific reason, field, and the limit that was exceeded.

### FR-032 · P0 · Accept and post a trade atomically
- **Rationale.** LED-1, LED-4, LED-5, LED-6; DOM-014.
- **Acceptance.**
  - The non-proposing party can accept a `proposed` transaction; acceptance results in `completed` with exactly two entries summing to zero (DOM-015) and an audit event, all in one atomic unit.
  - Fault injection between entry insertion and status update leaves no partial state.
  - If the payer's effective limit would be breached at acceptance, acceptance is refused with the reason; the transaction remains `proposed`.
  - Acceptance of an expired, declined, cancelled or completed transaction is refused.

### FR-033 · P0 · Decline and withdraw
- **Rationale.** Exits from `proposed` (DOM-010).
- **Acceptance.**
  - The non-proposing party can decline; the proposer can withdraw. Each releases the reservation and writes an audit event; neither creates entries.
  - Declining or withdrawing a transaction that is not `proposed` is refused.

### FR-034 · P0 · Proposal expiry
- **Rationale.** Stale proposals must not hold capacity indefinitely (DOM-013).
- **Acceptance.**
  - A scheduled system process moves proposals past their expiry to `expired`, releases reservations, writes an audit event (attributed to the system), and creates no entries.
  - An expired proposal cannot be accepted; a test with an injected clock proves the boundary (accepted exactly at expiry minus one second; refused at expiry).
  - Expiry processing is idempotent: running it twice yields the same result.

### FR-035 · P0 · Idempotent propose, accept and other commands
- **Rationale.** LED-4; DOM-016.
- **Acceptance.**
  - Propose, accept, decline, withdraw and reversal commands accept an idempotency key. Ten concurrent identical requests with one key produce one result.
  - A key reused with a different payload is refused; a key reused with the same payload returns the original response.

### FR-036 · P0 · Concurrent acceptance safety
- **Rationale.** LED-5 under concurrency (DOM-014).
- **Acceptance.**
  - Two accepts racing for the same last available credit result in exactly one completed transaction; the other is refused and stays `proposed`.
  - A load test with at least the concurrency level in [12-quality-test-plan.md](12-quality-test-plan.md) produces no account below its effective limit and a zero network sum.

### FR-037 · P1 · Note and listing reference on a trade
- **Rationale.** Context for the counterparty and for dispute review.
- **Acceptance.**
  - A proposal can include a plain-text note (bounded length) and an optional reference to an active listing.
  - The note is shown as plain text (no HTML rendering); markup in the note is displayed literally.
  - The note and reference are part of the proposal's terms for audit, but the note is not a term for consent (DOM-013).

### FR-038 · P1 · In-app notifications and inbox
- **Rationale.** Counterparties must see proposals and outcomes (J3, J5).
- **Acceptance.**
  - An active member has an inbox listing proposals awaiting their action, disputes awaiting response, and recent outcomes.
  - Unread state is shown in text as well as visually.
  - Email notifications are not part of this requirement (OQ-VIS-4).
- **Status (as built).** Built, in-app only (service `ListNotifications`, API `GET /notifications`, web inbox with an unread badge that also states the count in text). Emitted for proposal received, accepted, declined, withdrawn, dispute events, re-review events and report outcome; limit warnings come from the hourly job. Old read entries are not yet purged.

---

## 5. Balances and history

### FR-050 · P0 · Balance and available credit view
- **Rationale.** G4 and DOM-006 to DOM-008.
- **Acceptance.**
  - The view shows: balance (signed number and direction in words), effective credit limit, available credit, amount reserved by outgoing proposals, and the time the figures were computed.
  - For a fixture network, displayed values equal the test oracle's recomputation from entries.
  - A zero available credit is labelled "no credit available right now", not as an error.

### FR-051 · P0 · Transaction history
- **Rationale.** Members must be able to check what happened (G4, G6).
- **Acceptance.**
  - History lists the member's transactions and entries in reverse chronological order, paginated, with direction, counterpart display name, kind, status, amount and date.
  - Reversals link to their original and vice versa.
  - History shows completed, disputed, reversed, and terminal non-completed items with their status text.
  - The history of one member never includes entries of another member's account except the counterpart line on a shared transaction, which shows counterpart display name and amount only.

### FR-052 · P0 · Balance integrity check
- **Rationale.** LED-1 and LED-3 must be checkable in production.
- **Acceptance.**
  - An operator or administrator command recomputes every account balance from entries and compares it with any cached balance, and reports any mismatch and the network sum.
  - On a healthy network the command reports zero mismatches and zero network sum; on a seeded mismatch it reports the account and the difference.
  - The command writes no entries and runs read-only.

### FR-053 · P1 · Pending items view
- **Rationale.** Pending proposals must be visible separately from completed trades (J4).
- **Acceptance.** A pending view lists `proposed` transactions with direction, expiry time and actions available to the viewer; it does not mix them into completed history.

### FR-054 · P1 · Balance at a past date
- **Rationale.** Dispute and audit questions need a historical position.
- **Acceptance.** For any date within the network's history, the balance shown equals the sum of entries posted on or before that instant (test with a fixture of postings around the boundary).

---

## 6. Administrator controls

### FR-060 · P0 · Set a member's credit limit override
- **Rationale.** DOM-007.
- **Acceptance.**
  - An administrator can set or remove a per-member override with a reason; the effective limit follows DOM-007.
  - Lowering a limit writes no entries (DOM-032) and is audited with old and new values.
  - An administrator cannot change their own limit (DOM-037).

### FR-061 · P0 · Suspend and restore a member
- **Rationale.** Governance needs a non-destructive restriction (DOM-039).
- **Acceptance.**
  - Suspension requires a reason, changes the membership to `suspended`, and blocks the actions listed in DOM-039.
  - Restoration returns the membership to `active`. Both are audited.
  - Balances and history are unchanged by suspension or restoration.

### FR-062 · P0 · Administrative adjustment
- **Rationale.** Corrections that are not trades (DOM-022).
- **Acceptance.**
  - An administrator can create an adjustment with amount (positive integer, direction stated), target member, and a mandatory reason (bounded length).
  - It posts exactly two entries, one on the member and one on the system account (DOM-015).
  - It is refused for the administrator's own account (DOM-037) and when it would breach the member's effective limit (LED-5).
  - It is audited with reason, actor, target and amount.

### FR-063 · P0 · Manage invitations and join requests
- **Rationale.** Joining controls (FR-010, FR-011).
- **Acceptance.** An administrator can list open invitations, revoke an invitation, and view, approve or deny pending join requests; each action is audited.

### FR-064 · P1 · Default flag
- **Rationale.** Governance recording without ledger effect (DOM-025).
- **Acceptance.**
  - An administrator can set or clear a default flag with a reason. Setting it does not change balances or entries.
  - The flag's configured effect (for example, blocking new payer proposals) applies and is shown to the member in text.
  - Setting and clearing are audited.

### FR-065 · P1 · Assign dispute reviewers
- **Rationale.** Reviewer role (PER-003).
- **Acceptance.**
  - An administrator can assign or remove the reviewer role for a member; the assignment is audited.
  - A reviewer assignment does not grant administrator permissions.
  - A reviewer cannot be assigned to review a dispute in which they are a party (FR-072).

### FR-066 · P1 · Administrative reversal
- **Rationale.** DOM-036.
- **Acceptance.** An administrator can reverse a completed transaction with a reason (DOM-019, DOM-020). Refused for administrator-party transactions (DOM-037). Audited.

### FR-067 · P1 · Administrator overview
- **Rationale.** Admins need to see what needs attention.
- **Acceptance.**
  - The overview shows counts and links for: pending members, pending proposals older than a stated age, open disputes, disputes in `posting_pending`, members at or beyond their effective limit, and expiring invitations.
  - Each count is computed from the ledger and dispute data, not cached without a timestamp.

### FR-068 · P1 · Network closure
- **Rationale.** Networks can end (J8).
- **Acceptance.**
  - Closure is refused while any proposal, open dispute or `posting_pending` dispute exists, and the refusal lists the blockers.
  - On closure, new memberships, proposals, acceptances, adjustments and grants are refused; history, disputes already resolved and exports remain available to members and administrators.
  - Closure is audited with the reason.

### FR-069 · P0 · Administrator succession
- **Rationale.** A network must never be left without an administrator (PER-002).
- **Acceptance.**
  - An administrator can assign or remove the administrator role for other active members.
  - Any change that would leave zero active administrators is refused.
  - A member's role change is audited, including the actor and time.

---

## 7. Disputes

### FR-070 · P0 · Open a dispute
- **Rationale.** Recourse for completed trades (G5, DOM-017).
- **Acceptance.**
  - A party to a `completed` transaction can open a dispute within the dispute window, with a reason category and a bounded statement.
  - Opening a dispute after the window, by a non-party, or on a non-completed transaction is refused with the reason.
  - Opening a dispute creates no entries and sets the transaction to `disputed`.

### FR-071 · P0 · Dispute does not change postings
- **Rationale.** Principle 2; DOM-017.
- **Acceptance.** Test compares all entries and balances before and after dispute open, decision, withdrawal and dismissal; they are identical except for entries produced by an upheld reversal.

### FR-072 · P0 · Reviewer independence
- **Rationale.** DOM-038.
- **Acceptance.**
  - A reviewer who is a party to the transaction, or who proposed, accepted, adjusted or reversed it, cannot be assigned to or decide the dispute.
  - The refusal is tested for each role combination.

### FR-073 · P0 · Decide a dispute
- **Rationale.** Outcomes must be recorded with reasons (SM-5, DOM-018).
- **Acceptance.**
  - A reviewer records one of: `dismissed`, `upheld` (full), `partially_upheld` (amount stated), each with a rationale (required, bounded).
  - `upheld` and `partially_upheld` post the reversal in the same atomic unit as the dispute state change, subject to DOM-021.
  - Both parties and administrators see the outcome and rationale; the reviewer's identity is shown to administrators and to the parties according to the policy in [09-security-privacy.md](09-security-privacy.md).

### FR-074 · P0 · Upheld reversal blocked by the limit
- **Rationale.** DOM-021.
- **Acceptance.**
  - If the reversal would breach LED-5, no entry is written, the dispute enters `posting_pending`, and both the reviewer and administrators see the reason.
  - Posting completes only after the resolution path in OQ-DOM-2 is applied (for example, an audited, time-limited override), and the state change and entries are atomic.

### FR-075 · P1 · Withdraw a dispute
- **Rationale.** Parties may resolve disputes privately.
- **Acceptance.** The opener can withdraw before a decision; the transaction returns to `completed`; the withdrawal is audited; no entries are created.

### FR-076 · P1 · Response by the other party
- **Rationale.** Fair process (governance).
- **Acceptance.**
  - The other party can submit a bounded response before a decision; the response is visible to the reviewer and to the opener.
  - A decision cannot be recorded before the response period in the network's policy has ended, or before the response is submitted or waived. The response period is a network policy (OQ-FR-2).
- **Status (as built).** Built (service `RespondToDispute`, API `POST /disputes/{id}/response`, web dispute page). One response of up to 2000 characters; visible to opener, reviewer and administrators. Network policy `response_period_days` (default 7, 0 to 90; 0 means no waiting period; DEC-012 recommendation). `ResolveDispute` is refused with invalid-transaction-state until a response exists, the period since the dispute was opened has elapsed, or the reviewer sets `waive_response` with a `waive_reason` (audited). Silence never decides the case. OQ-FR-2 stays open for counsel review of the default.

### FR-077 · P0 · Dispute window enforcement and deadlines
- **Rationale.** DOM-017.
- **Acceptance.**
  - The window is computed from completion time with the server clock; the dispute screen shows the closing date.
  - Windows that close while a dispute is open do not cause the dispute to be closed or decided automatically.

### FR-078 · P1 · One review of a decision
- **Rationale.** Governance: members can challenge decisions (see [10-governance-operations.md](10-governance-operations.md)).
- **Acceptance.**
  - A party can request one re-review of a decided dispute within a stated period.
  - The re-review is assigned to a different reviewer than the original and is recorded as a new decision linked to the first. The original decision is never edited.
- **Status (as built).** Built (service `RequestReReview` / `DecideReReview`, API `POST /disputes/{id}/re-review[/decision]`, web). Either party, once (database-enforced), within `re_review_window_days` of the decision (network policy, default 14, 1 to 365). A different reviewer decides, recorded as a linked record; the original decision is not edited. Outcomes: confirmed, upheld, partially_upheld (a further reversal of at most what is still reversible; refused if LED-5 would be breached).

---

## 8. Audit history

### FR-080 · P0 · Every state change is audited
- **Rationale.** LED-6; DOM-035.
- **Acceptance.** A sweep test over all state-changing commands finds an audit event for each, written in the same atomic unit; fault injection between state change and audit leaves no state change.

### FR-081 · P0 · Audit event content
- **Rationale.** The glossary defines audit events; DAT defines the schema.
- **Acceptance.** Every audit event contains: event id, network id, time (server), actor (member id or system/operator identifier) and actor role, action, target type and id, before and after summary (no full personal data), reason where the action requires one, and request id. Schema test fails on missing fields.

### FR-082 · P0 · Audit visibility by role
- **Rationale.** Accountability without exposing unnecessary data (VIS-010).
- **Acceptance.**
  - Administrators can read the full audit history of their network.
  - Members can read audit entries that concern their own account, in summarized form, without other members' identifiers beyond counterpart display names on shared transactions.
  - Dispute reviewers can read audit entries for disputes they are assigned to.
  - Operators can read operational logs, not network audit content, unless a recorded break-glass access applies (OQ-PER-2).
  - Each access to network audit content by an operator is itself audited.

### FR-083 · P0 · Audit entries are append-only
- **Rationale.** LED-2 applies to audit records too.
- **Acceptance.** No API or repository path updates or deletes audit events; a test attempts both and expects refusal.

### FR-084 · P1 · Audit search and filter
- **Rationale.** Administrators must investigate incidents.
- **Acceptance.** Administrators can filter audit events by date range, action, actor and target; results are paginated and the filter applies only to the network's events.

### FR-085 · P2 · Tamper-evident audit chain
- **Rationale.** Stronger integrity evidence than append-only storage alone.
- **Acceptance.** Deferred. Post-MVP: each audit event includes a hash over its contents and the previous event; a verification command detects a modified event. No cryptographic guarantee is claimed for the MVP.

---

## 9. Exports (CSV and JSON)

### FR-090 · P0 · Member exports own history
- **Rationale.** Portability and departure (G7, FR-014).
- **Acceptance.**
  - A member can export their own transactions and entries as CSV and as JSON, including after departure or suspension.
  - Row count equals the history view for the same filter.
  - Amounts appear as integer minor units and as the formatted decimal value in the network's format.
  - The export contains no other member's identifiers, contact details, credentials or session data; counterparty appears as display name only.

### FR-091 · P0 · Admin exports the network ledger
- **Rationale.** Auditable, portable ledger (VIS-009).
- **Acceptance.**
  - Administrators can export all entries and transactions of their network in CSV and JSON, with network metadata (decimals, export time, schema version).
  - Sum of all exported entries is zero; per-account sums equal exported balances (round-trip check computed by an independent script).
  - The export includes a checksum of its content so a recipient can verify integrity.

### FR-092 · P0 · Admin exports the member list
- **Rationale.** Administrative records.
- **Acceptance.** The export includes only the fields in the data inventory for member records (display name, membership state, join date, roles, limit override) and never includes credentials or recovery data.

### FR-093 · P0 · Admin exports the audit history
- **Rationale.** Accountability outside the application.
- **Acceptance.** Export includes all audit fields in FR-081 for the requested range; sorted by sequence; with checksum; the export of a range equals the events visible by a query over the same range.

### FR-094 · P1 · Full network export for closure or portability
- **Rationale.** Closure and portability (J8, VIS-009).
- **Acceptance.**
  - A single export contains members, accounts, listings, transactions, entries, disputes, decisions, policies (history) and audit events, in JSON, with a documented schema version.
  - The export can be produced on a closed network.
  - Import is deferred (see the deferred list); the schema is documented in [06-data-ledger-audit.md](06-data-ledger-audit.md).

### FR-095 · P1 · Export correctness and performance gates
- **Rationale.** Exports must be correct at realistic sizes.
- **Acceptance.**
  - Export output for the fixture network equals the independently computed expected output byte-for-byte for CSV and semantically equal for JSON.
  - Size thresholds and maximum duration for a large fixture are TBD by product decision (OQ-FR-3); the export must stream, not hold the whole ledger in memory, and the test records peak memory.

---

## 10. Deferred capabilities (explicit list)

These capabilities are **not** part of the MVP. No route, screen or configuration for them may be enabled in the MVP build (VIS-012). Each is a candidate for a later sub-specification after the MVP baseline is accepted.

| Deferred capability | Reason for deferral | Earliest dependency |
|---|---|---|
| Native mobile applications | Web delivery first; accessibility and offline needs must be measured | Pilot results |
| Federation between networks | Tenant isolation first (VIS-003) | Data and API contracts |
| Tokenisation or blockchain representation of credits | Conflicts with the accounting-unit definition; no basis yet | Legal review (out of this pass) |
| Payment or national-currency conversion, redemption | Out of scope (VIS-001); legal and financial review required | Not scheduled |
| Reputation or credit scoring | Automated member decisions need governance and fairness work | Governance spec |
| Advanced analytics and dashboards beyond FR-067 | Not needed for MVP decisions | Pilot data needs |
| Third-party integrations (payments, accounting, identity providers, chat) | Not needed; adds trust boundaries | Architecture spec |
| Email notifications | Open question OQ-VIS-4 | Product decision |
| Mutual reversal without dispute (DOM-034) | Adds a consent path to reversals; requires further design | Domain review |
| Tamper-evident audit chain (FR-085) | P2; integrity evidence beyond append-only | Audit design |
| Import of exported data (restore into a new network) | P2; needs conflict rules | Migration design |
| Recurring or subscription trades; escrow; multi-party trades | Would change the two-entry trade model | Domain review |
| Multi-currency or multiple units within one network | Would change balance semantics | Domain review |
| Automated debt collection, automatic default or sanctions | Governance and legal questions (DOM-025) | Counsel review |
| Hosted subscription and paid administration features | Monetisation is outside this pass (VIS-011) | Business decision |
| Single sign-on and external identity providers | P2; depends on [09-security-privacy.md](09-security-privacy.md) decisions | Security design |

## 11. Requirement summary

| Area | Requirements | P0 | P1 | P2 |
|---|---|---|---|---|
| Network setup | FR-001 to FR-006 | 6 | 0 | 0 |
| Membership | FR-010 to FR-017 | 6 | 2 | 0 |
| Directory and listings | FR-020 to FR-025 | 4 | 2 | 0 |
| Trade proposal and acceptance | FR-030 to FR-038 | 7 | 2 | 0 |
| Balances and history | FR-050 to FR-054 | 3 | 2 | 0 |
| Administrator controls | FR-060 to FR-069 | 5 | 5 | 0 |
| Disputes | FR-070 to FR-078 | 6 | 3 | 0 |
| Audit history | FR-080 to FR-085 | 4 | 1 | 1 |
| Exports | FR-090 to FR-095 | 4 | 2 | 0 |

Counts are a summary for planning and are checked against the file in the traceability step of [12-quality-test-plan.md](12-quality-test-plan.md).

## 12. Open questions

- **OQ-FR-1 (resolved as built).** Question moved to [DECISIONS.md](../DECISIONS.md): see DEC-053.
- **OQ-FR-2 (product owner, counsel-reviewed where relevant).** Default response period for disputes (FR-076), and whether a decision may be recorded without a response.
- **OQ-FR-3 (product owner and QA).** Size and duration thresholds for FR-095.
- **OQ-FR-4 (product owner).** Whether members see the reviewer's identity in dispute decisions (FR-073). Current spec: visible to administrators; visible to parties per the policy in [09-security-privacy.md](09-security-privacy.md). Raised by the build: DEC-077 in [DECISIONS.md](../DECISIONS.md).
- **OQ-FR-5 (resolved as built).** Question moved to [DECISIONS.md](../DECISIONS.md): see DEC-054.
- **OQ-FR-6 (resolved as built).** Question moved to [DECISIONS.md](../DECISIONS.md): see DEC-054.

Open questions are labelled `OQ-FR-n` and are to be consolidated into [13-assumptions-risks-decisions.md](13-assumptions-risks-decisions.md).
