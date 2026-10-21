# 10 Governance and Network Operations

Status: Draft v0.1 · Owner/role: Product architect (legal items pending counsel review) · Last updated: 2026-10-21 · Scope: rules a network administrator can configure, how changes are announced and recorded, membership lifecycle, administrator succession, disputes, challenges and appeals, negative-balance recovery, suspension, closure, data export, and the platform operator's commitments for the MVP · Dependencies: [00](00-conventions.md) (glossary, LED invariants, configurable policy defaults), [03](03-domain-model.md), [06](06-data-ledger-audit.md), [07](07-api-contract.md), [09](09-security-privacy.md), [11](11-deployment-operations.md), [12](12-quality-test-plan.md), [13](13-assumptions-risks-decisions.md) · Open questions: section 10

Normative terms follow [00 section 1](00-conventions.md#1-normative-terms). Glossary terms (Network, Member, Account, Balance, Credit limit, Transaction, Posting, Adjustment, Reversal, Dispute, Audit event, System account) have the meanings in [00 section 3](00-conventions.md#3-glossary-authoritative) and are not redefined here.

This document separates two things. **Software enforcement** is behaviour the product guarantees by code and tests. **Governance policy** is behaviour the network agrees to through its rules, which the software records but cannot fully enforce, for example whether a described good was actually delivered. Each rule in this document is labelled with one of three classes:

- **Platform invariant**: not configurable; enforced by software (LED-1..LED-7 and the requirements that implement them).
- **Configurable policy**: set by the network administrator within bounds; enforced by software once set; recorded as a policy change.
- **Governance practice**: a rule the network agrees to and the software supports with records and workflow, but cannot decide on the merits.

The product does not decide whether credits have value, whether members will accept them, or whether any network's rules are lawful. These are outside this document and are listed for counsel in [13](13-assumptions-risks-decisions.md).

---





## Build status (as built)

Snapshot: HEAD `925f637` (branch claude/circula-build). Statuses were first read at `31b1015` and re-checked at `925f637` (see TRACEABILITY.md). **built** = behaviour exists in code; **partial** = exists with the gaps stated, or the required verification was not run; **not built**; **deferred** = out of MVP by this spec or a recorded decision. Code paths and test names: [TRACEABILITY.md](../TRACEABILITY.md). Open owner decisions and resolved questions: [DECISIONS.md](../DECISIONS.md). Status is implementation status, not a compliance claim; no release gate has been run.

Summary: built 11, not built 6, partial 10.

| ID | Priority | Status | Note |
|---|---|---|---|
| GOV-001 | P0 | partial | Policy versions prospective; policy version stored on each transaction not verified |
| GOV-002 | P0 | built | Changes audited with before and after; member-visible change log at /n/{slug}/policy and GET /network/policy-log |
| GOV-003 | P0 | built | Adverse changes wait 14 days by default (notice_days up to 90); a shorter notice needs a reason shown in the change log; members get an inbox notice; a scheduled change can be cancelled |
| GOV-004 | P0 | built | Decimals lock after first completed transaction |
| GOV-005 | P1 | built | Bounds with boundary tests |
| GOV-006 | P1 | partial | Unilateral admin change built; objection right not built |
| GOV-007 | P0 | built | Corrective actions are transactions; property tests |
| GOV-008 | P0 | built | Invite, approval, open modes; audit per decision |
| GOV-009 | P0 | built | Append-only terms versions; acceptance recorded at join and re-required before proposing or accepting a trade after an update; reading and leaving stay open |
| GOV-010 | P0 | partial | Blockers, retain and settle rules built; counterparties of cancelled proposals are notified; the leave page points to the self-export on History, no export is offered in the flow |
| GOV-011 | P0 | partial | Admin removal with reason; the member gets an inbox notice; challenge not built |
| GOV-012 | P1 | built | Optional inactivity rule (policy inactivity_days): flags, notifies, lists for administrators; never removes |
| GOV-013 | P0 | built | Window and party rule; dispute keeps entries unchanged |
| GOV-014 | P0 | partial | Statement and decision with reasons; response period built (network policy); decision-target deadline not built |
| GOV-015 | P0 | built | Reversal and adjustment are new transactions; originals unchanged |
| GOV-016 | P0 | not built | Challenges and appeals not built |
| GOV-017 | P1 | built | No dispute upload route |
| GOV-018 | P0 | partial | Trades continue within limits; write-off by approved adjustment not built (SEC-011) |
| GOV-019 | P0 | built | Limit reduction posts nothing; increases refused |
| GOV-020 | P0 | partial | Last-admin guard built; read-only administrator-required state not built |
| GOV-021 | P0 | not built | Platform-assisted recovery not built |
| GOV-022 | P0 | partial | Member, network and full exports built; reconciliation summary in manifest; import not built |
| GOV-023 | P0 | built | Administrator suspension overlay: postings, acceptances and joins refused; reads and exports kept; members notified; audited |
| GOV-024 | P0 | built | Operator command with mandatory category; administrators notified; audit visible to the network |
| GOV-025 | P0 | partial | Closure exposed in web and API with blockers, confirm and member notices; closure-date notice period, final export checksum and erasure not built |
| GOV-026 | P1 | partial | Policy documents drafted in docs/platform, dated and versioned; counsel review and advance-notice test outstanding |
| GOV-027 | P1 | partial | Support boundary in docs/ops; not linked from admin UI |

## 1. Roles and responsibilities

| Role | Governance responsibility | Limits |
|---|---|---|
| Network administrator | Sets configurable policies, approves members (per join mode), appoints dispute reviewers, initiates adjustments and write-offs, publishes notices | Cannot change a posted transaction; cannot approve own value-moving action (SEC-010); subject to dual control (SEC-011) |
| Dispute reviewer | Decides disputes in its network | Cannot decide a case where it is a party or conflicted; cannot change policy |
| Member | Proposes, accepts and declines trades; raises disputes and challenges; exports own data | Cannot change network rules |
| Platform operator | Runs the software, publishes platform policy and notices, handles support and incidents | Does not decide network disputes or rules except to enforce platform suspension (section 9); no routine data access (SEC-013) |

## 2. Configurable network policies

Each policy below has a class, a default, and a change rule. The default for each value that is a recommendation rather than a settled decision refers to the DEC entry in [13](13-assumptions-risks-decisions.md).

| Policy | Class | Default | Change rule |
|---|---|---|---|
| Default credit limit for new members | Configurable | 0 (per [00 section 5](00-conventions.md#5-configurable-network-policies-defaults)) | Prospective only (GOV-001) |
| Per-member credit limit override | Configurable | None | Prospective; notice required (GOV-003) |
| Maximum transaction amount | Configurable | Set by administrator; no network-wide default | Prospective |
| Proposal expiry | Configurable | 7 days | Prospective; applies to proposals created after change |
| Decimals (minor-unit scale) | Configurable at creation | 2 | Locked after first completed transaction (GOV-004) |
| Member directory visibility | Configurable | Members-only | Prospective; notice required |
| Join mode | Configurable | Invite | Prospective; see section 4 |
| Dispute window | Configurable | 30 days after completion | Prospective; shortening requires notice (GOV-003) |
| Departure rule | Configurable | See section 4 and DEC-010 | Prospective; notice required |
| Sign-up grant | Configurable | 0 (DEC-011) | Prospective; approval rules in SEC-032 |
| Dispute timelines | Configurable within bounds | DEC-012 | Prospective |
| Adjustment approval threshold | Configurable | DEC-013 | Prospective; itself a value-related change requiring dual control |

**GOV-001** · P0 · Prospective-only policy changes
- Requirement: A policy change MUST apply only to postings, proposals, decisions and memberships that occur after the change takes effect. It MUST NOT alter any existing entry, transaction status that was already reached, or any completed decision.
- Rationale: Retroactive change would break the mutual-credit promise that completed trades are fixed (LED-2).
- Acceptance criteria:
  - Lowering a credit limit does not alter an existing completed transaction or its entries (test).
  - A proposal created before a proposal-expiry change keeps the expiry in force at its creation (test).
  - Each transaction stores the policy version in force at acceptance (test; field defined in [06](06-data-ledger-audit.md)).

**GOV-002** · P0 · Recording policy changes
- Requirement: Every policy change MUST be recorded as an audit event (SEC-012) with the actor, the previous and new value, the effective time and an optional reason, and MUST be shown to members in a readable change log within the network.
- Rationale: Members must be able to see what rules applied when (T-16).
- Acceptance criteria:
  - Changing each configurable policy in section 2 produces exactly one audit event with before and after values (test).
  - The change log lists the change and its effective time for members (test).
  - A policy change that fails validation produces no audit event and no value change (test).

**GOV-003** · P0 · Notice before adverse changes
- Requirement: Changes that reduce a member's available credit, shorten a dispute window, change the departure rule, change directory visibility to a less restrictive setting, or change the join mode to a more open setting MUST be announced to affected members before they take effect. The minimum notice period is set by DEC-025 (recommended default: 14 days). An administrator MAY set a shorter period only with a recorded reason, which MUST be shown in the change log.
- Rationale: Surprise changes to credit positions harm members and undermine trust.
- Acceptance criteria:
  - An adverse change with a notice period of 13 days is refused under a 14-day rule (test).
  - Affected members receive a notice when the change is announced (test).
  - A shortened notice requires a reason and appears in the change log (test).

**GOV-004** · P0 · Decimal scale lock
- Requirement: The minor-unit scale (decimals) of a network MUST be set at creation and MUST NOT change after the first completed transaction in that network.
- Rationale: Changing scale would re-interpret stored integers (LED-7).
- Acceptance criteria:
  - An attempt to change decimals after a completed transaction is refused (test).
  - An attempt before any completed transaction is accepted with an audit event and a notice (test).

**GOV-005** · P1 · Policy validation bounds
- Requirement: The product MUST enforce bounds on each configurable policy so that it cannot be set to a value that breaks platform invariants. Examples: credit limits MUST be zero or positive; expiry MUST be at least one hour and at most a platform-defined maximum; maximum transaction amount MUST be a positive integer in minor units.
- Rationale: Invalid policy values can silently disable controls.
- Acceptance criteria:
  - Negative credit limit is refused (test).
  - Expiry below the minimum is refused (test).
  - Each bound has a test that sets the boundary value and the value just outside it (test; see [12](12-quality-test-plan.md)).

**GOV-006** · P1 · Policy decision method
- Requirement: For the MVP, a network administrator MAY change configurable policies unilaterally, subject to GOV-003 notice, GOV-002 recording, and the member objection right in GOV-016. The product MUST NOT implement member voting in the MVP. Member voting is a deferred capability (section 11). The decision method is recorded in DEC-021.
- Rationale: A single administrator is the simplest workable method for small networks; voting needs its own specification.
- Acceptance criteria:
  - A non-administrator cannot change any configurable policy (authorization test).
  - The administrator interface states the current decision method to the administrator (UX check, [08](08-ux-accessibility.md)).

## 3. Platform invariants that governance cannot override

The following are not configurable by any network and are restated only as governance boundaries. Their authoritative definitions are in [00 section 4](00-conventions.md#4-platform-wide-invariants-not-configurable).

- A completed transaction has two equal and opposite postings (LED-1).
- Entries are never edited or deleted; corrections are new transactions (LED-2).
- Balances are sums of entries (LED-3).
- Both sides post atomically and retries do not duplicate (LED-4).
- No non-system account may go below its credit limit at posting time (LED-5). A network may only lower a limit prospectively; it cannot force an existing negative balance to be reversed by software. Recovery is governed by section 7.
- Every state change writes an audit event (LED-6).
- Money is integer minor units (LED-7).

**GOV-007** · P0 · No governance override of ledger invariants
- Requirement: No administrator, reviewer or platform operator action MAY post an entry that violates LED-1 through LED-7. Any corrective action MUST be a new transaction of kind adjustment or reversal.
- Rationale: The mutual-credit ledger is the product's core integrity guarantee.
- Acceptance criteria:
  - The adjustment and reversal flows reject a request that would create an unbalanced transaction (test).
  - Property tests in [12](12-quality-test-plan.md) pass after random sequences of administrative actions (test).

## 4. Membership lifecycle

**GOV-008** · P0 · Join modes
- Requirement: A network MUST operate in exactly one join mode: invite (an existing member or administrator issues an invitation), approval (an applicant requests membership and an administrator approves or declines), or open (anyone can join, subject to the network's onboarding terms). The default is invite. Every membership decision MUST record the actor and the time as an audit event.
- Rationale: Join mode is the first defence against Sybil accounts (T-18).
- Acceptance criteria:
  - Under invite mode, a join without a valid invitation is refused (test).
  - Under approval mode, an applicant is not a member until approved (test).
  - Every join decision produces one audit event (test).
  - Open mode with a sign-up grant follows SEC-032 (test).

**GOV-009** · P0 · Onboarding
- Requirement: On joining, a member MUST accept the network's current rules and onboarding terms, which MUST be shown and recorded with the version accepted. A member whose accepted version is outdated MUST see the new terms before creating a proposal (not before reading history or leaving).
- Rationale: Members need to be bound only by rules they have seen.
- Acceptance criteria:
  - A new member cannot complete joining without accepting the displayed terms version (test).
  - After a terms update, a member is prompted before creating a proposal (end-to-end test).
  - Reading balances and history is allowed before accepting updated terms (test).

**GOV-010** · P0 · Voluntary departure
- Requirement: A member MUST be able to leave a network, subject to the departure preconditions in [03](03-domain-model.md) DOM-024: no `disputed` transaction in which the member is a party, no pending dispute posting, and no `proposed` transaction involving the member unless it is first resolved, declined, cancelled or expired. Where a blocker exists, departure is refused and the blockers are listed to the member. Open proposals MAY be cancelled automatically by the departure process, with the cancellation recorded and each counterparty notified. On departure the account is frozen for new proposals and acceptances, the member keeps read-only access to their own history and export, and the member MUST be offered a self-export (SEC-017) before departure completes. The account and its entries MUST remain in the ledger (LED-2).
- Rationale: Departure must not destroy the network's ledger, must not strand open disputes, and must not leave counterparties with orphaned proposals.
- Acceptance criteria:
  - Departure with an open dispute in which the member is a party is refused and lists the blocker (test).
  - A departed member cannot create or accept a proposal (test).
  - Automatic cancellation of open proposals records an audit event and notifies each counterparty (test).
  - The departing member's entries and balances are unchanged by departure (test).
  - The member is offered a self-export before departure completes (test).

**GOV-011** · P0 · Removal by administrator
- Requirement: An administrator MAY remove a member only for a reason recorded from a documented list (for example: breach of network rules, confirmed fraud, prolonged non-response to a dispute) and MUST notify the member. The removal MUST be subject to a challenge right under section 6. Removal MUST follow the same freeze and cancellation rules as GOV-010. A removal for fraud or breach MUST NOT be decided by the same administrator who is the counterparty in the matter (SEC-010).
- Rationale: Removal is the most severe membership action and is open to misuse (T-16).
- Acceptance criteria:
  - Removal without a reason from the list is refused (test).
  - Removal produces a notice to the member and an audit event (test).
  - Removal of a member by a conflicted administrator is refused (test).

**GOV-012** · P1 · Inactivity
- Requirement: A network MAY define an inactivity rule (for example, no login and no trade for a set period). The product MUST NOT remove a member for inactivity automatically. An inactivity rule MAY only produce a notice and a flagged list for an administrator to decide (recommended default: no inactivity rule).
- Rationale: Automatic removal would lose balances and create disputes without human judgement.
- Acceptance criteria:
  - An inactive member is flagged and notified and remains a member after the rule runs (test with injected clock).

## 5. Transactions, disputes and reversals

**GOV-013** · P0 · Dispute window and eligibility
- Requirement: A dispute MAY be raised on a completed transaction only by one of its two parties, within the dispute window (default 30 days after completion, DEC-012). A dispute MUST NOT change the status or entries of the transaction on its own; the transaction status moves to `disputed` as defined in [03](03-domain-model.md). A dispute after the window is refused, except that an administrator MAY open a case on a documented fraud finding under GOV-011, with the reason recorded.
- Rationale: Bounded windows give finality. Disputes without a window reopen settled trades indefinitely.
- Acceptance criteria:
  - A non-party cannot raise a dispute (authorization test).
  - A dispute raised one day after the window closes is refused (test with injected clock).
  - A dispute raised within the window sets the transaction to `disputed` and leaves all entries unchanged (test).

**GOV-014** · P0 · Dispute process
- Requirement: The dispute process MUST include: a statement from the complainant with a required reason category and free text within a length limit; a response from the other party within the response period (recommended 7 days, DEC-012); a decision by an authorized dispute reviewer who is not a party or conflicted (SEC-010); a written decision with reasons; and an outcome that is one of: no change, reversal, adjustment, or partial remedy recorded as a reversal or adjustment of the agreed amount. The decision target is recommended at 14 days from the response deadline (DEC-012). A missed response deadline MUST NOT by itself decide the case in favour of the complainant without reviewer assessment.
- Rationale: A fair process is needed for trust. Automatic outcomes on silence would be easy to abuse.
- Acceptance criteria:
  - Each step's state and deadline is visible to parties (test; [08](08-ux-accessibility.md)).
  - A decision cannot be recorded without written reasons (test).
  - A missed response deadline leaves the case open for reviewer assessment (test with injected clock).
  - Each outcome creates the corresponding ledger change through a reversal or adjustment with a reference to the dispute, and never edits the original transaction (test; LED-2).

**GOV-015** · P0 · Reversals and adjustments
- Requirement: A reversal MUST reference the original transaction, MUST be a new transaction of kind reversal, and MUST post entries equal and opposite to the original's entries. An adjustment MUST carry a mandatory reason, MUST post equal and opposite entries against the network's system account (00 glossary), and MUST be approved as required by SEC-011. Neither may alter an existing transaction.
- Rationale: Traceability of corrections is a core requirement (LED-2).
- Acceptance criteria:
  - A reversal of a completed transaction restores both balances to their values before the original, through new entries (test).
  - An adjustment without a reason is refused (test).
  - After reversal or adjustment the original transaction and its entries are byte-for-byte unchanged in storage (test; see [12](12-quality-test-plan.md)).

**GOV-016** · P0 · Challenges and appeals
- Requirement: A member MUST be able to challenge the following: a membership removal, a dispute outcome, a credit-limit reduction applied to the member, and a listing removal. A member MAY also object to a policy change during its notice period (GOV-006). An objection is recorded and shown to the administrator, and it does not by itself stop the change unless the network's written rules say so. A challenge MUST be decided by an authorized person who was not involved in the original decision. A dispute outcome MAY be appealed once, within the appeal period (recommended 14 days, DEC-012), to a reviewer or administrator not involved in the first decision. Each decision MUST include reasons, and the outcome MUST be recorded as an audit event. Objections to policy changes (GOV-006) are recorded in the same way.
- Rationale: Members need a route to contest decisions that affect their balances or membership.
- Acceptance criteria:
  - A challenge by a member against their own removal is accepted for review and assigned to a non-involved reviewer (test).
  - An appeal after the appeal period is refused (test with injected clock).
  - An appeal decided by the original reviewer is refused (authorization test).
  - Every challenge and appeal decision produces an audit event and a member notice (test).

**GOV-017** · P1 · Dispute evidence
- Requirement: For the MVP, dispute evidence is limited to text entered in the case and references to the transaction and its audit events. Uploading files is out of scope and deferred (section 11), because it introduces storage and content-safety requirements not yet specified.
- Rationale: Keeps the MVP minimal and avoids unspecified file handling (see [09](09-security-privacy.md) SEC-023/SEC-024).
- Acceptance criteria:
  - No dispute route accepts a file upload (route inventory test).

## 6. Negative-balance and credit-limit events

**GOV-018** · P0 · Recovery of negative balances
- Requirement: A member whose balance is below zero MUST be able to restore it by completing trades with others in the network, which the product MUST NOT block as long as the trades are within their own limits. The product MUST NOT treat a negative balance as a default automatically. A negative balance that a network considers unrecoverable MAY be written off only by an adjustment that has a reason, requires approval under SEC-011, and is shown in the change log. The product MUST NOT initiate collection outside the network.
- Rationale: Natural recovery is the core mutual-credit mechanism. Write-off must be visible and not hidden. Collection is a legal question for counsel.
- Acceptance criteria:
  - A trade that increases a negative member's balance but keeps it within the credit limit is accepted (test).
  - A trade that increases the balance of a member already below their limit (possible only after GOV-019) is accepted and does not worsen the position (test).
  - A write-off without a reason or approval is refused (test).
  - A write-off appears in the change log and the member's history (test).

**GOV-019** · P0 · Reduction of credit limit below current negative balance
- Requirement: When a credit limit is reduced below a member's current negative balance, the existing balance MUST remain unchanged. The member MUST be prevented from any new posting that increases the negative position, and MUST still be allowed postings that reduce it. The member MUST be notified under GOV-003.
- Rationale: Software must not silently take value or force reversals. Preventing new debt while allowing recovery is consistent with LED-5.
- Acceptance criteria:
  - After reducing a limit below the balance, a posting that increases the negative position is refused and a posting that reduces it is accepted (test).
  - Existing entries are unchanged (test).

## 7. Administrator succession and continuity

**GOV-020** · P0 · Administrator continuity
- Requirement: Every active network MUST have at least one active administrator at all times (SEC-010). Networks SHOULD have at least two administrators before they accept a sign-up grant or enter open join mode. The product MUST provide a documented succession process: an existing administrator appoints a successor; an administrator may step down only after a successor is active or the network is suspended; and if no administrator is active, the recovery process in GOV-021 applies.
- Rationale: Networks with no administrator cannot approve members, disputes or adjustments.
- Acceptance criteria:
  - Stepping down as the last administrator is refused (test).
  - Appointing a successor produces an audit event and a notice (test).
  - A network with no active administrator enters a read-only "administrator required" state, visible to members, and no value-moving action is allowed (test).

**GOV-021** · P0 · Recovery when no administrator is available
- Requirement: If no administrator can act, the platform operator MAY assist recovery only through a documented, dual-approved, audited process (SEC-013 break-glass) that verifies the requester's identity against the network's recorded members and notifies all members of the network. The platform operator MUST NOT appoint an administrator on its own authority and MUST NOT change balances as part of recovery.
- Rationale: Platform-led recovery is a high-risk path (T-17) and must be narrowly bounded.
- Acceptance criteria:
  - An appointment without second approval is refused (test).
  - Recovery actions produce audit events visible to the network (test).
  - No recovery action changes a balance or entry (test).

## 8. Network data export and portability

**GOV-022** · P0 · Export contents and portability
- Requirement: The product MUST support: (a) a member self-export of the member's own profile, balances history and transactions in which they are a party, (b) a network export for the network's administrators containing members, configuration, policy change log, transactions, entries, disputes and audit events for that network, and (c) a full network export sufficient for migration to another deployment of the same product. Exports MUST be machine-readable in a documented format defined in [07](07-api-contract.md) and MUST include a reconciliation summary: the sum of all entries and the balance totals. Exports MUST follow SEC-017.
- Rationale: Data exportability is an architectural quality (see [05](05-architecture.md)) and protects members from lock-in.
- Acceptance criteria:
  - A full network export re-imported into a clean instance (migration test) yields equal balances and entries, verified by the reconciliation summary (test; [12](12-quality-test-plan.md)).
  - The reconciliation summary shows a sum of zero for all entries (test).
  - A member self-export contains no other member's personal data (test).

## 9. Service suspension and closure

**GOV-023** · P0 · Network suspension by an administrator
- Requirement: An administrator MAY suspend trading in a network. A suspended network MUST NOT accept new proposals, acceptances or postings, but members MUST still be able to read balances, history and their own export. The reason and time of suspension MUST be recorded, and members MUST be notified.
- Rationale: Networks need a way to pause during a dispute or security review without losing access to records.
- Acceptance criteria:
  - A suspended network refuses new postings (test).
  - Balance and history reads succeed for members during suspension (test).
  - Resumption is recorded as an audit event (test).

**GOV-024** · P0 · Platform suspension
- Requirement: The platform operator MAY suspend a network only for documented reasons in one of these categories: a legal order, a serious security incident, persistent abuse of the platform, or non-performance of a written hosting agreement (for hosted deployments). Platform suspension MUST notify the network's administrators within the period set by DEC-014 unless legally prohibited, MUST preserve member read and export access as in GOV-023, and MUST be recorded as an audit event visible to the network. The platform operator MUST NOT suspend a network for a reason related to its members' trades or balances.
- Rationale: Platform power must be bounded and visible (T-17). Trade-related suspension would make the operator a party to network decisions.
- Acceptance criteria:
  - A platform suspension without a category is refused (test).
  - The affected administrators receive a notice (test).
  - Members retain read and export access during platform suspension (test).

**GOV-025** · P0 · Network closure
- Requirement: A network closure MUST proceed in these steps: (a) announce a closure date with at least the notice period in DEC-025 (recommended default: 30 days for closure), (b) freeze new proposals and postings at the closure date, (c) cancel open proposals with notices, (d) keep read and export access for a period (recommended default: 90 days, DEC-020), (e) produce a final export with a reconciliation summary and a checksum, and (f) apply retention and erasure rules (SEC-018) after the period. Closure MUST be recorded as an audit event.
- Rationale: Orderly closure protects members' balance records and their right to take them.
- Acceptance criteria:
  - A closure cannot take effect with less notice than the configured minimum (test).
  - After closure the network refuses postings and keeps read and export access during the period (test).
  - The final export contains a checksum that matches the recomputed value (test).

## 10. Platform operator commitments

**GOV-026** · P1 · Published platform policies
- Requirement: For hosted deployments, the platform operator MUST publish: its terms, its privacy information for operators' data processing, its operator access policy (SEC-013), its service suspension categories (GOV-024), and its incident notification practice (SEC-030). Each MUST be dated and versioned, and material changes MUST be announced to network administrators before taking effect (period DEC-025). Self-hosted operators are responsible for their own equivalents.
- Rationale: Hosted members and administrators need to know the platform's own rules, distinct from network rules.
- Acceptance criteria:
  - Each listed document exists, is dated and versioned before hosted launch (release gate).
  - A material change to a platform policy is announced in advance in a test environment notice flow (test).

**GOV-027** · P1 · Support boundaries
- Requirement: The support responsibilities for each deployment MUST be documented: network administrators support members on network rules and trades; the platform operator supports software, deployment and platform policy. Support staff MUST NOT access network financial data for support requests unless SEC-013 break-glass applies.
- Rationale: Clear boundaries prevent support from becoming a back door to data.
- Acceptance criteria:
  - The support boundary is documented in [11](11-deployment-operations.md) and linked from the administrator interface (documentation and UX check).

## 11. Deferred governance capabilities (not in MVP)

The following are explicitly deferred. Each would need its own specification before implementation: member voting on policy, delegated or federated governance across networks, dispute evidence uploads, automated inactivity removal, public transparency reports, and multi-administrator committees beyond the dual control in SEC-011.

## 12. Open questions

1. Decision method for policy changes (DEC-021): unilateral administrator with notice, or a member vote in some form. The MVP recommendation is unilateral with notice and objection rights.
2. Departure settlement rule (DEC-010): whether residual positive balances stay recorded, whether residual negative balances remain recorded as debt until written off, and whether a network may close with unsettled balances. Legal review is required (RSK entries).
3. Notice periods (DEC-025) for adverse changes and for closure.
4. Dispute timelines (DEC-012) and whether a response deadline has a default outcome.
5. Adjustment approval threshold and dual-control scope (DEC-013).
6. Whether platform suspension categories need an appeal path beyond notice.
7. Retention of closed-network data (DEC-020); counsel review.
8. Who may be a dispute reviewer when a network has only one member besides the administrator.
9. Whether the platform operator must perform any function that a network administrator cannot perform for a network with no administrator (section 7, GOV-021).
