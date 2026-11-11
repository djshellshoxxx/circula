# Personas, Roles, and Core User Journeys

| Field | Value |
|---|---|
| Status | Draft |
| Owner role | Product owner |
| Last updated | 2026-11-11 |
| Scope | The four personas Circula serves, the roles and permissions matrix (authoritative home), and the core user journeys with their error and accessibility states, at requirements level. No screen designs. |
| Dependencies | [00-conventions.md](00-conventions.md), [03-domain-model-and-mutual-credit-rules.md](03-domain-model-and-mutual-credit-rules.md) |
| Related documents | [04-functional-requirements.md](04-functional-requirements.md), [08-ux-and-accessibility.md](08-ux-and-accessibility.md), [09-security-privacy-and-abuse-resistance.md](09-security-privacy-and-abuse-resistance.md), [10-governance-and-network-operations.md](10-governance-and-network-operations.md), [13-assumptions-risks-decisions-and-roadmap.md](13-assumptions-risks-decisions-and-roadmap.md) |

## 1. Purpose

This document answers three questions for implementers and reviewers: who uses Circula, what each role may and may not do, and what the core journeys look like from start to finish, including the failure and accessibility states that every journey has to handle.

It owns the identifier prefixes **PER** (personas, roles, permissions) and **JRN** (journeys). It uses the glossary in [00-conventions.md](00-conventions.md) verbatim and does not restate entity definitions, lifecycles, or ledger rules; those live in [03-domain-model-and-mutual-credit-rules.md](03-domain-model-and-mutual-credit-rules.md). Where a journey depends on a rule that belongs elsewhere, it links to the owning document instead of copying the rule. Testable acceptance criteria for these capabilities are written in [04-functional-requirements.md](04-functional-requirements.md); this document states the behavior a journey needs, not the test.

Circula is software for operating a network. Nothing here implies that credits have monetary value or that members will accept them (see [DEC-008](13-assumptions-risks-decisions-and-roadmap.md#dec-008)).

Status vocabulary used below (full lifecycles in [03-domain-model-and-mutual-credit-rules.md](03-domain-model-and-mutual-credit-rules.md)):

| Object | Statuses |
|---|---|
| Membership | Invited, Pending, Active, Suspended, Departed; terminal invitation/application states Expired, Revoked, Rejected |
| Account | Active, Frozen, Closed |
| Transaction | Proposed, Completed, Declined, Cancelled, Expired, Reversed |
| Dispute | Open, Under review, Resolved (outcome: dismissed, reversed, adjusted, withdrawn; see [DOM-062](03-domain-model-and-mutual-credit-rules.md#dom-062)) |
| Listing | Active, Paused, Closed |
| Network | Setup, Active, Suspended, Closed |

## 2. Personas

Personas describe people the product is designed for. They are composites for requirements work, not research findings; assumptions about them are recorded in [13-assumptions-risks-decisions-and-roadmap.md](13-assumptions-risks-decisions-and-roadmap.md).

<a id="per-001"></a>**PER-001 Network member**

| Attribute | Description |
|---|---|
| Role | Member (every other role is held in addition to membership). |
| Who | A neighbor, tradesperson, household, or small organization that joined a local network to exchange goods and services with others in the community. |
| Goals | Find people who offer what they need; offer their own skills or goods; propose and accept trades with little effort; always know their own balance, available credit, and what is still pending; fix a mistake or contest a trade that went wrong. |
| Context | Uses the product occasionally, often for a few minutes after a face-to-face exchange. May hold memberships in more than one network under one sign-in. Usually does not read network rules in full. |
| Pain points | Unsure whether a negative balance is a debt or normal; fear of a trade posting twice after a poor connection; unclear whether the other party has agreed yet; no visible route to challenge a decision. |
| Accessibility and device assumptions | Phone browser is the main device; desktop browser second. Includes people who use screen readers, magnification, keyboard-only or switch input, and people with limited digital experience. Connections can be intermittent. Email is the only assumed notification channel ([DEC-007](13-assumptions-risks-decisions-and-roadmap.md#dec-007)). |

<a id="per-002"></a>**PER-002 Network administrator**

| Attribute | Description |
|---|---|
| Role | Network administrator. |
| Who | A volunteer coordinator or small committee member who runs one network. Usually also an ordinary member who trades. |
| Goals | Onboard members and keep the member list accurate; set default credit limits and other network policy and change them with a record of why; correct errors with traceable adjustments; keep disputes moving; export the network's data. |
| Context | Works in bursts (membership drives, month-end review). Holds the trust of the community, so every action must be explainable afterwards. May serve several networks. May also be the dispute reviewer when policy permits. |
| Pain points | Fear of making an irreversible mistake; no safe way to see what a policy change will affect; pressure to act on member complaints without a recorded basis; handover difficulty when an administrator steps down. |
| Accessibility and device assumptions | Desktop or laptop browser for bulk work, phone for quick approvals. Needs lists and tables that work with keyboard and screen reader, and confirmation steps that state the effect of an action in text. |

<a id="per-003"></a>**PER-003 Dispute reviewer**

| Attribute | Description |
|---|---|
| Role | Dispute reviewer. |
| Who | A respected member chosen under network policy to examine disputes. May be an administrator unless policy forbids the overlap. |
| Goals | See the disputed transaction or adjustment, the parties' statements, and the relevant history in one place; record a resolution with reasons; avoid reviewing a dispute they are party to. |
| Context | Handles few disputes, infrequently, and is often a party's neighbor. Needs the process to be impartial and visibly so. |
| Pain points | Incomplete information; pressure from either party; unclear authority over what a resolution may change; no record of how similar cases were resolved. |
| Accessibility and device assumptions | Same as members; reading long statements means text must resize and reflow, and status changes must be announced to assistive technology. |

<a id="per-004"></a>**PER-004 Platform operator**

| Attribute | Description |
|---|---|
| Role | Platform operator (instance level). |
| Who | A person or organization that runs one instance: installs, upgrades, backs up, and supports it. May be the same organization as a network's administrators, but acts in a different role. |
| Goals | Keep the instance running; create networks and assign each one's first administrator; restore service after failure; answer support requests without reading members' trading data. |
| Context | Technical, works through an operations interface and the command line rather than the member-facing product. Has no ledger role by default. |
| Pain points | Needing to look at network data to diagnose a fault with no accountable way to do so; being asked by members to "fix" a balance; handing a network over to its own administrators cleanly. |
| Accessibility and device assumptions | Desktop browser and terminal. Operations screens follow the same keyboard and screen-reader requirements as member screens ([08-ux-and-accessibility.md](08-ux-and-accessibility.md)). |

## 3. Roles and permissions matrix

### 3.1 Role model

<a id="per-005"></a>**PER-005** (P0) Roles MUST be held per membership, not per person. One person MAY hold memberships in several networks with different roles in each. The role set is: member, network administrator, dispute reviewer (all network-level, held through a membership), and platform operator (instance-level, not a membership). Stored role values are `administrator` and `reviewer`; `member` is implicit ([03-domain-model-and-mutual-credit-rules.md](03-domain-model-and-mutual-credit-rules.md) section 3.3). The API scope named `admin` is a token scope, not a role.
*Rationale:* a coordinator of one network is an ordinary member of another; authority must not leak across networks.

<a id="per-006"></a>**PER-006** (P0) A membership MAY hold the administrator and reviewer roles at once unless network policy forbids it. A dispute reviewer MUST NOT review or resolve a dispute in which that person's own account is the disputed account or a party to the disputed transaction or adjustment. Every Active network MUST have at least one Active administrator; the software MUST reject any action (role removal, suspension, departure) that would leave it with none.
*Rationale:* impartial review and continuity of governance. Recusal rules beyond this floor are network policy ([10-governance-and-network-operations.md](10-governance-and-network-operations.md)).

<a id="per-007"></a>**PER-007** (P0) The platform operator MUST NOT hold a role in any network's ledger by virtue of operating the instance, and MUST NOT post transactions, post adjustments, or change network policy through the normal interface or API. Operator access to a network's data is limited to the break-glass path in PER-033, which is read-only except the administrator grant of SEC-142, and to the succession contact record named in PER-033.
*Rationale:* separates platform operation from network governance ([05-system-architecture.md](05-system-architecture.md)).

<a id="per-008"></a>**PER-008** (P0) Every permission in the matrix MUST be enforced by the platform on the server side for the user interface and the API alike. A permission not listed as Allowed is denied. A denied action MUST fail with an authorization error that names no data the actor could not otherwise see ([07-api-and-integration-contract.md](07-api-and-integration-contract.md)).
*Rationale:* one enforcement path ([DEC-006](13-assumptions-risks-decisions-and-roadmap.md#dec-006)).

<a id="per-009"></a>**PER-009** (P0) Every action in the matrix that changes ledger state, policy, membership, roles, or access MUST create an audit event ([06-data-model-ledger-and-audit.md](06-data-model-ledger-and-audit.md)).
*Rationale:* no action by any role is untraceable.

### 3.2 Matrix

Values: **Allowed**, **Not allowed**, **Policy-dependent** (the network configures it; the recommended default is in the Notes column). "Member" means any Active member, including holders of the other network roles when acting for themselves. Cells for the administrator and reviewer columns describe the additional authority those roles grant within their own network. All actions apply within one network unless stated. A person whose membership is Suspended is treated per JRN-007, not as a member in this matrix; a write by such a person is rejected with `forbidden`, except the read and export rights stated at PER-027 and PER-030. Actions the matrix does not cover are in section 3.3.

| ID | Action | Member | Network administrator | Dispute reviewer | Platform operator | Notes |
|---|---|---|---|---|---|---|
| <a id="per-010"></a>**PER-010** | Create network | Not allowed | Not allowed | Not allowed | Allowed | Operator creates the network in status Setup and assigns the first administrator. Self-service creation is open ([OQ-PER-1](#5-open-questions)). |
| <a id="per-011"></a>**PER-011** | Configure network policy | Not allowed | Allowed | Not allowed | Not allowed | Changes are within platform invariants, recorded with a reason, and announced per [10-governance-and-network-operations.md](10-governance-and-network-operations.md). Operator cannot change policy by any path; break-glass is read-only except the administrator grant of SEC-142 ([PER-033](#per-033)). |
| <a id="per-012"></a>**PER-012** | Invite members | Policy-dependent | Allowed | Not allowed | Not allowed | Members may invite only when policy [DOM-115](03-domain-model-and-mutual-credit-rules.md#dom-115) is on; recommended default off. |
| <a id="per-013"></a>**PER-013** | Approve or decline applications (Pending to Active) | Not allowed | Allowed | Not allowed | Not allowed | |
| <a id="per-014"></a>**PER-014** | Suspend a member | Not allowed | Allowed | Not allowed | Not allowed | Reason mandatory; see JRN-007. Cannot target the last Active administrator (PER-006). |
| <a id="per-015"></a>**PER-015** | Remove a member (membership to Departed) | Not allowed | Allowed | Not allowed | Not allowed | Handling of a non-zero balance is defined in [03-domain-model-and-mutual-credit-rules.md](03-domain-model-and-mutual-credit-rules.md) and [10-governance-and-network-operations.md](10-governance-and-network-operations.md). A member may depart voluntarily (JRN-006). |
| <a id="per-016"></a>**PER-016** | Assign or remove roles | Not allowed | Allowed | Not allowed | Not allowed | Subject to PER-006. The operator assigns only the first administrator at network creation, and, under break-glass, the administrator role to an existing Active member of a network with no administrator able to act (SEC-142, [PER-033](#per-033)). |
| <a id="per-017"></a>**PER-017** | Publish and manage own listings (offers, requests) | Allowed | Allowed | Allowed | Not allowed | Requires Active membership. Administrator MAY pause or close another member's listing for a recorded reason. |
| <a id="per-018"></a>**PER-018** | Propose a transaction | Allowed | Allowed | Allowed | Not allowed | As payer or payee, for the proposer's own account only, or for the network account when the proposer is an administrator ([DOM-021](03-domain-model-and-mutual-credit-rules.md#dom-021), [DOM-043](03-domain-model-and-mutual-credit-rules.md#dom-043)). Requires both accounts Active. |
| <a id="per-019"></a>**PER-019** | Accept a transaction | Allowed | Allowed | Allowed | Not allowed | Counterparty only; nobody accepts on another member's behalf. For the network account, an administrator other than the proposer ([DOM-022](03-domain-model-and-mutual-credit-rules.md#dom-022)). |
| <a id="per-020"></a>**PER-020** | Decline a transaction | Allowed | Allowed | Allowed | Not allowed | Counterparty only. For the network account, an administrator other than the proposer ([DOM-022](03-domain-model-and-mutual-credit-rules.md#dom-022)). |
| <a id="per-021"></a>**PER-021** | Cancel a transaction | Allowed | Allowed | Allowed | Not allowed | Proposer only, while Proposed. For the network account, an administrator other than the proposer ([DOM-022](03-domain-model-and-mutual-credit-rules.md#dom-022)). |
| <a id="per-022"></a>**PER-022** | Request reversal of a completed transaction | Allowed | Allowed | Allowed | Not allowed | Either party requests; the other must agree; within the policy window ([DOM-101](03-domain-model-and-mutual-credit-rules.md#dom-101)). |
| <a id="per-023"></a>**PER-023** | Reverse a transaction outside mutual request (as dispute resolution or administrator action) | Not allowed | Allowed | Not allowed | Not allowed | Full reversal only; reason mandatory. The reviewer records the resolution; the effect on the ledger follows [03-domain-model-and-mutual-credit-rules.md](03-domain-model-and-mutual-credit-rules.md). |
| <a id="per-024"></a>**PER-024** | Post an adjustment | Not allowed | Allowed | Not allowed | Not allowed | Reason mandatory; usually against the network account ([DEC-003](13-assumptions-risks-decisions-and-roadmap.md#dec-003)); affected member notified; can be disputed. |
| <a id="per-025"></a>**PER-025** | Raise a dispute | Allowed | Allowed | Allowed | Not allowed | Raiser must be a party to the transaction or the account affected by the adjustment. |
| <a id="per-026"></a>**PER-026** | Review and resolve a dispute | Not allowed | Not allowed | Allowed | Not allowed | An administrator reviews only if also granted the `reviewer` role, which [DOM-113](03-domain-model-and-mutual-credit-rules.md#dom-113) permits or forbids; never where PER-006 recusal applies. |
| <a id="per-027"></a>**PER-027** | View own balance and history | Allowed | Allowed | Allowed | Not allowed | Always available while the membership exists in any status that [10-governance-and-network-operations.md](10-governance-and-network-operations.md) keeps readable. Suspended and Departed memberships keep PER-027 and PER-030 for the personal-data retention period of DATA-080 (2 years after Departed; GOV-042, GOV-080). |
| <a id="per-028"></a>**PER-028** | View other members' balances | Policy-dependent | Allowed | Policy-dependent | Not allowed | Recommended default for members and reviewers: hidden. A reviewer sees the parties' postings for a dispute assigned to them regardless of this setting. |
| <a id="per-029"></a>**PER-029** | View audit log | Not allowed | Allowed | Allowed (events of disputes assigned to them only) | Not allowed | Administrator sees the network's audit log only. A member sees events about their own account through history (PER-027), not the audit log. Reviewer sees events relating to assigned disputes only. Instance-level operational logs are not this log ([09-security-privacy-and-abuse-resistance.md](09-security-privacy-and-abuse-resistance.md)). |
| <a id="per-030"></a>**PER-030** | Export own data | Allowed | Allowed | Allowed | Not allowed | Own memberships, listings, transaction history, disputes raised. Kept by Suspended and Departed memberships as stated at PER-027. |
| <a id="per-031"></a>**PER-031** | Export network data | Not allowed | Allowed | Not allowed | Not allowed | Instance backups are an operations matter ([11-deployment-operations-and-support.md](11-deployment-operations-and-support.md)), not a network export. |
| <a id="per-032"></a>**PER-032** | Break-glass access to network data | Not allowed | Not allowed | Not allowed | Allowed | Exceptional, justified, time-limited, audited; read-only except the administrator grant of SEC-142; no posting authority; no export ([PER-033](#per-033)). |

<a id="per-033"></a>**PER-033** (P0) Break-glass access is the only route by which the platform operator reaches a network's data or state, with one named exception: the succession contact record of GOV-053 (FR-069) is readable in the operator console without break-glass, because it contains no ledger data. It MUST be a path separate from the normal interface, MUST require a recorded justification, MUST be time-limited, and MUST create audit events visible to that network's administrators. The path MUST be read-only, with one exception: the administrator grant of SEC-142, which assigns the administrator role to an existing Active member of a network with no administrator able to act. That grant always needs dual control (a single-operator instance cannot perform it), produces an ordinary administrator role with no expiry, and is notified to every Active member; only the break-glass session itself is time-boxed per SEC-140. The path MUST NOT allow the operator to post a transaction or adjustment, change policy, or export network data (SEC-144). Details are owned by [09-security-privacy-and-abuse-resistance.md](09-security-privacy-and-abuse-resistance.md).
*Rationale:* operators sometimes need access to repair a fault; that access must be exceptional and visible.

<a id="per-034"></a>**PER-034** (P0) Where the matrix says Policy-dependent, the network's configured value MUST be applied by the platform, and the recommended default MUST apply to a new network until an administrator changes it. The list of policy keys is owned by [03-domain-model-and-mutual-credit-rules.md](03-domain-model-and-mutual-credit-rules.md).
*Rationale:* a fresh network behaves predictably.

### 3.3 Additional actions

These rows complete the matrix for actions that other documents rely on. Values and conventions are as in section 3.2. PER-008, PER-009 and PER-034 apply to them.

| ID | Action | Member | Network administrator | Dispute reviewer | Platform operator | Notes |
|---|---|---|---|---|---|---|
| <a id="per-035"></a>**PER-035** | Browse the directory and search listings | Allowed | Allowed | Allowed | Not allowed | Anonymous access follows [DOM-097](03-domain-model-and-mutual-credit-rules.md#dom-097). |
| <a id="per-036"></a>**PER-036** | List own notifications and set notification preferences | Allowed | Allowed | Allowed | Not allowed | Preferences cannot disable the mandatory notification set of FR-084. |
| <a id="per-037"></a>**PER-037** | Trigger and view balance verification | Not allowed | Allowed (own network) | Not allowed | Allowed (results without account data) | FR-057, FR-058. |
| <a id="per-038"></a>**PER-038** | Suspend or reinstate a network | Not allowed | Allowed | Not allowed | Allowed | Reason mandatory; effects in [DOM-016](03-domain-model-and-mutual-credit-rules.md#dom-016) and DOM section 3.2.1. |
| <a id="per-039"></a>**PER-039** | Close a network; complete setup | Not allowed | Allowed | Not allowed | Not allowed | Setup completion needs the values set at setup ([DOM-094](03-domain-model-and-mutual-credit-rules.md#dom-094)). |
| <a id="per-040"></a>**PER-040** | Break-glass administrator grant | Not allowed | Not allowed | Not allowed | Allowed (dual control) | Only under SEC-142 and GOV-054; see PER-033. |
| <a id="per-041"></a>**PER-041** | Report abuse; handle reports | Allowed (report) | Allowed (handle, but not a report about themselves) | Allowed (report) | Not allowed | FR-068, SEC-060 to SEC-065. |
| <a id="per-042"></a>**PER-042** | Download an export file | Requester only | Requester only | Requester only | Not allowed | Follows PER-030 and PER-031: only the person who requested the export. |
| <a id="per-043"></a>**PER-043** | Cancel a scheduled policy version | Not allowed | Allowed | Not allowed | Not allowed | [DOM-017](03-domain-model-and-mutual-credit-rules.md#dom-017); recorded with a reason. |
| <a id="per-044"></a>**PER-044** | Freeze or unfreeze an account; change account limits | Not allowed | Allowed | Not allowed | Not allowed | Reason mandatory; the affected member is notified (SEC-051). |
| <a id="per-045"></a>**PER-045** | Approve another administrator's adjustment or reversal; confirm or decline a dispute remedy | Not allowed | Allowed (not the initiator, not a party; for a remedy, not the deciding reviewer) | Not allowed | Not allowed | [DOM-041](03-domain-model-and-mutual-credit-rules.md#dom-041), [DOM-062](03-domain-model-and-mutual-credit-rules.md#dom-062). |
| <a id="per-046"></a>**PER-046** | Assign a dispute; withdraw a dispute; file or hear an appeal | Withdraw (raiser only); file an appeal (party) | Assign; hear an appeal as governance sets | Take assignment; hear an appeal as governance sets | Not allowed | [DOM-061](03-domain-model-and-mutual-credit-rules.md#dom-061), [DOM-063](03-domain-model-and-mutual-credit-rules.md#dom-063); the appeal process is in GOV-066. |
| <a id="per-047"></a>**PER-047** | Revoke another member's sessions | Not allowed | Allowed | Not allowed | Not allowed | SEC-046, SEC-011. |
| <a id="per-048"></a>**PER-048** | Edit the rules statement, the notice board, and the succession contact | Not allowed | Allowed | Not allowed | Not allowed | FR-008, FR-089, FR-069. The operator reads the succession contact record only (PER-033). |
| <a id="per-049"></a>**PER-049** | View the network account's balance and events | Allowed | Allowed | Allowed | Not allowed | [DOM-053](03-domain-model-and-mutual-credit-rules.md#dom-053); counterparties are named only as DOM-053 allows. |

## 4. Core journeys

Each journey states the actor, precondition, numbered steps, postcondition, error and edge states, and accessibility states. They describe required behavior, not screens. Interaction details beyond this level are deferred. The accessibility states below apply in addition to the general requirements in [08-ux-and-accessibility.md](08-ux-and-accessibility.md).

Requirements common to all journeys:

<a id="jrn-009"></a>**JRN-009 Common journey requirements** (P0)
1. Every state-changing step MUST accept an idempotency key so that a retry after a lost response has the same effect as the first request ([07-api-and-integration-contract.md](07-api-and-integration-contract.md)). A resubmitted form MUST show the original result, never create a second object.
2. Every journey MUST be completable with keyboard only, with visible focus and logical focus order.
3. Every error MUST state in text what happened, what the state of the object now is, and what the person can do next. Meaning MUST NOT rely on color alone.
4. A change of balance, transaction status, or membership status that results from the person's own action MUST be announced to assistive technology without moving focus away from the current control.
5. Sessions MUST NOT expire while a person is mid-step without warning; when a session does end, entered data MUST be preserved or the person told it was not (session limits are set in [09-security-privacy-and-abuse-resistance.md](09-security-privacy-and-abuse-resistance.md)). Time limits on the business objects themselves (invitation validity, proposal expiry) are measured in days, not seconds, and are shown as dates.

The journeys that follow are numbered JRN-001 to JRN-008.

### <a id="jrn-001"></a>JRN-001 Join a network

| Item | Description |
|---|---|
| Actor | Prospective member (PER-001). Administrator (PER-002) for approval. |
| Precondition | Network is Active. Either the person holds an unexpired invitation (membership Invited), or network policy allows applications. The person has an email address ([DEC-007](13-assumptions-risks-decisions-and-roadmap.md#dec-007)). |

Steps:
1. The person opens the invitation link or the network's join page.
2. The person provides the minimum data the network requires (email address, display name) and confirms they have read the network rules. No further personal data is collected unless the network policy adds it.
3. The platform verifies control of the email address.
4. For an invitation sent by an administrator, the membership becomes Active. For an invitation sent by a member ([DOM-115](03-domain-model-and-mutual-credit-rules.md#dom-115)) or an application, it becomes Pending ([DOM-095](03-domain-model-and-mutual-credit-rules.md#dom-095)).
5. For a Pending membership, an administrator reviews the application and approves or declines it with a reason ([PER-013](#per-013)). The applicant is notified of the outcome.
6. On approval, the platform creates the member account with a balance of zero and the network's default credit limit, and the member sees their own balance view.

Postcondition: membership Active; one member account Active with a balance of zero; audit events for invitation or application, approval, and account creation.

Error and edge states:
- Invitation expired, already used, or revoked: the person is told which, and how to request a new one. No membership is created.
- Email already has a membership in this network: the platform shows the existing membership's state; no duplicate is created.
- Person already belongs to another network: the same sign-in is reused; a separate membership and account are created.
- Duplicate submission or retry of the application: exactly one Pending membership exists.
- Network is Suspended, Closed, or in Setup: joining is refused with an explanation.
- Application declined: the membership enters the terminal state `Rejected` (membership lifecycle in [03-domain-model-and-mutual-credit-rules.md](03-domain-model-and-mutual-credit-rules.md#331-membership-lifecycle)) and the applicant is notified with the reason. Whether and when the person may apply again is a governance matter ([10-governance-and-network-operations.md](10-governance-and-network-operations.md)); retention of the record follows [06-data-model-ledger-and-audit.md](06-data-model-ledger-and-audit.md) ([OQ-PER-6](#5-open-questions)).
- Administrator does not respond: the application stays Pending; there is no automatic approval. The applicant can see that it is awaiting approval.

Accessibility states: form fields have programmatic labels and error messages tied to fields; email verification does not depend on a time limit shorter than one day, and offers a resend; the Pending state is conveyed in text, not only by an icon; completing the step announces the new status.

### <a id="jrn-002"></a>JRN-002 Publish an offer or request

| Item | Description |
|---|---|
| Actor | Active member (PER-001). |
| Precondition | Membership Active. |

Steps:
1. The member chooses to add a listing and selects offer or request.
2. The member enters a title and description, and optionally a category, availability, and an indicative amount in the network's credit unit.
3. The platform validates the entries and shows what other members will see, including whether contact details are visible per network policy.
4. The member confirms; the listing becomes Active and appears in the member directory and listing search.
5. Later, the member can edit the listing, set it Paused, or set it Closed.

Postcondition: a listing exists with status Active (or Paused or Closed after later action), owned by the member; audit event recorded.

Error and edge states:
- Missing or over-length fields: the error names the field and the limit.
- Duplicate submission: one listing is created.
- Member is Suspended or has Departed: publishing is refused; what happens to existing listings follows [10-governance-and-network-operations.md](10-governance-and-network-operations.md) ([OQ-PER-3](#5-open-questions)).
- Indicative amount invalid (not a whole number of minor units, or exceeds the network's decimal places): rejected with an explanation. A listing amount is informational and creates no obligation or posting.
- Personal data entered into a description: the form warns that descriptions are visible to the audience set by network policy.

Accessibility states: listing status and type are available as text; Pause and Close controls state their effect; search results announce the number of results and status changes after filtering.

### <a id="jrn-003"></a>JRN-003 Propose and accept a trade

| Item | Description |
|---|---|
| Actor | Proposer and counterparty, each an Active member (PER-001); either may be payer or payee. |
| Precondition | Both memberships Active, both accounts Active, same network. Proposer and counterparty are different accounts. |

Steps:
1. The proposer selects the counterparty, states whether they are the payer or the payee, and enters the amount and a description. The proposer MAY link a listing.
2. The platform validates the amount (positive, within the network's decimal places) and shows the proposer's current balance and available credit for information. Available credit shown at this stage is not a reservation ([DEC-004](13-assumptions-risks-decisions-and-roadmap.md#dec-004)).
3. The proposer submits. Submitting is the proposer's consent. The transaction becomes Proposed and the counterparty is notified by email.
4. The counterparty opens the proposal, sees amount, direction (whether their balance would rise or fall), description, and expiry date, and accepts.
5. At acceptance the platform checks credit limits atomically, creates the posting pair, and sets the transaction Completed ([03-domain-model-and-mutual-credit-rules.md](03-domain-model-and-mutual-credit-rules.md)).
6. Both parties are notified; each sees the updated balance.

Alternative outcomes: the counterparty declines (Declined); the proposer cancels while Proposed (Cancelled); the proposal passes the policy-set window ([DOM-100](03-domain-model-and-mutual-credit-rules.md#dom-100)) (Expired).

Postcondition: Completed with exactly one pair of postings and updated balances for both parties, or no postings and a terminal status of Declined, Cancelled, or Expired. Audit events for each state change.

Error and edge states:
- Limit exceeded: if at acceptance the payer's balance minus the amount would fall below the negative of their credit limit, or the payee's positive limit would be exceeded, acceptance is rejected with `limit_exceeded` (payer's credit limit) or `positive_limit_exceeded` (payee's positive limit), stating which account and which limit without disclosing the other party's balance. No postings are created. The status afterwards follows [03-domain-model-and-mutual-credit-rules.md](03-domain-model-and-mutual-credit-rules.md); the proposer can cancel and re-propose a smaller amount.
- Expired proposal: acceptance is rejected, the status shows Expired, and either party can propose again.
- The actor's own membership is Suspended: the write is refused by the authorization gate with `forbidden` (PER-008; the read and export rights of PER-027 and PER-030 are unaffected). Counterparty account Frozen or Closed, or counterparty Departed: proposing or accepting is refused with the reason (`account_not_active`). Proposals already outstanding are cancelled by the system with the status reason recorded and both parties notified ([DOM-024](03-domain-model-and-mutual-credit-rules.md#dom-024)).
- Duplicate submission or retry: one transaction. A retried accept after a lost response returns the original result and does not post twice.
- Concurrent actions (counterparty accepts while proposer cancels, or two sessions accept): exactly one outcome occurs; the loser sees the resulting status.
- Counterparty not in this network or not found: the error does not reveal whether the person belongs to other networks.
- Proposer tries to accept their own proposal: refused.

Accessibility states: direction is stated in words ("your balance will decrease by 20"), never by color or sign alone; the proposal screen and notification email contain the same information; the Completed result and new balance are announced on completion; the expiry date is stated as a date; no step depends on a timer shorter than the session policy.

### <a id="jrn-004"></a>JRN-004 Review balances and history

| Item | Description |
|---|---|
| Actor | Member (PER-001); administrator (PER-002) for all balances in their network; reviewer (PER-003) for disputed items. |
| Precondition | Membership exists in a status whose view rights are granted by [PER-027](#per-027). |

Steps:
1. The member opens their account overview and sees the balance, credit limit, available credit, and the number of Proposed transactions that involve them, with pending amounts shown separately from completed ones.
2. The member opens the history, a chronological list of transactions, reversals, and adjustments affecting the account, filterable by date range, status, and counterparty.
3. The member opens one entry and sees its status, amount, direction, description, parties, timestamps, and links to related entries (a reversal and its original, an adjustment and its reason, a dispute).
4. Where network policy allows, the member opens the member list and sees other members' balances ([PER-028](#per-028)). Administrators see all balances in their network.

Postcondition: no state changes; viewing creates no audit event unless a view of another member's balance is logged by network policy ([OQ-PER-5](#5-open-questions)).

Error and edge states:
- Balances hidden by policy: the member list shows no balance and states that balances are not shown in this network, rather than showing blank or zero.
- Negative balance: shown with the sign and the phrase required by UX-041 in [08-ux-and-accessibility.md](08-ux-and-accessibility.md), and the remaining available credit.
- Frozen account or Suspended membership: own history remains readable; the reason is stated.
- Long history: paginated; a position in the list is announced; filters preserve keyboard focus.
- A displayed balance changes while the page is open (another party accepted a trade): the page states that the balance has changed and offers refresh; it does not alter figures silently.
- Departed member: read access to own history follows [10-governance-and-network-operations.md](10-governance-and-network-operations.md).

Accessibility states: balance, credit limit, and available credit are labelled text, each with direction; tables have headers and sort state is announced; a balance change from an incoming completed trade is announced politely without interrupting the current task; amounts are readable by a screen reader as numbers with the credit unit name.

### <a id="jrn-005"></a>JRN-005 Raise a dispute

| Item | Description |
|---|---|
| Actor | A member who is a party to a Completed transaction or the account affected by an adjustment (PER-001); the dispute reviewer (PER-003); the administrator (PER-002) for acting on the outcome. |
| Precondition | Transaction is Completed, or an adjustment exists, and any dispute window set by network policy is open. No Open or Under review dispute exists on the same object. |

Steps:
1. The member opens the transaction or adjustment and chooses to raise a dispute, selects a reason category, and writes a statement.
2. The dispute becomes Open. The other party and the network's administrators and eligible reviewers are notified.
3. A reviewer who is not a party takes the dispute; its status becomes Under review. The other party adds a statement.
4. The reviewer examines the statements and the related history and records a resolution with reasons: `dismissed`, `reversed`, or `adjusted` (`withdrawn` is recorded when the raiser withdraws; outcomes in [DOM-062](03-domain-model-and-mutual-credit-rules.md#dom-062)).
5. If the outcome is `reversed` or `adjusted`, the remedy the resolution calls for (a full reversal or an adjustment) is carried out by an authorized party as defined in [03-domain-model-and-mutual-credit-rules.md](03-domain-model-and-mutual-credit-rules.md) and [10-governance-and-network-operations.md](10-governance-and-network-operations.md), and the resulting postings reference the dispute.
6. Both parties are notified of the resolution and of any appeal route.

Postcondition: dispute Resolved with an outcome and reasons; any resulting reversal or adjustment recorded as new postings; the original postings are unchanged; audit events recorded.

Error and edge states:
- Reviewer is a party: the platform prevents that person from taking or resolving the dispute (PER-006).
- No eligible reviewer (the only reviewer is a party, or no one holds the role): the dispute stays Open and the administrators are told; the escalation route is defined in [10-governance-and-network-operations.md](10-governance-and-network-operations.md).
- Duplicate submission or a second dispute on the same object: one dispute exists; the second attempt shows the existing one.
- Transaction already Reversed: raising a dispute on the original is refused with `already_reversed` and an explanation. A reversal event cannot itself be disputed ([DOM-060](03-domain-model-and-mutual-credit-rules.md#dom-060)); the member challenges it through the appeal on the dispute that caused it ([DOM-063](03-domain-model-and-mutual-credit-rules.md#dom-063)) or, for an administrator-initiated reversal, through the governance route in [10-governance-and-network-operations.md](10-governance-and-network-operations.md).
- Dispute window closed: refused with the closing date.
- Other party Suspended or Departed: the dispute proceeds; notification goes to the address on record.
- A `reversed` or `adjusted` remedy would leave an account over its limit: the reversal is still posted with the limit-override flag recorded and the account shown as over limit ([DOM-032](03-domain-model-and-mutual-credit-rules.md#dom-032)); the dispute resolves normally.

Accessibility states: statements are plain-text fields that accept long input without character-counter timeouts; dispute status changes are announced and sent by email; the status timeline is available as an ordered list in text.

### <a id="jrn-006"></a>JRN-006 Administer network rules

| Item | Description |
|---|---|
| Actor | Network administrator (PER-002). |
| Precondition | Network is Setup or Active; actor holds the administrator role in it. |

Steps:
1. The administrator opens the network policy and sees each setting with its current value and recommended default.
2. The administrator changes one or more settings (for example default credit limit, proposal expiry, reversal window, balance visibility, whether members may invite).
3. The platform validates the change against platform invariants and shows, in text, what the change affects (for example, how many existing accounts are already beyond a lowered limit; rules in [03-domain-model-and-mutual-credit-rules.md](03-domain-model-and-mutual-credit-rules.md)).
4. The administrator enters a reason and confirms.
5. The platform saves the change as a new policy version, records an audit event, and notifies members as the announcement rules in [10-governance-and-network-operations.md](10-governance-and-network-operations.md) require.

Postcondition: policy version saved with who, when, what, and why; previous values remain retrievable; no balance has changed as a side effect.

Error and edge states:
- A value that would relax a platform invariant (for example, disabling the equal-and-opposite postings): rejected, naming the invariant.
- A value out of range: rejected naming the allowed range.
- Two administrators edit at once: the second save is rejected with a conflict and shows the current values.
- Duplicate submission: one new version.
- Change would leave no Active administrator or violate PER-006: rejected.
- Administrator is also the subject of a decision (for example their own limit): allowed only as network policy permits and recorded; flagged for the audit view.
- Platform operator attempts the change through the normal interface: refused (PER-007).

Accessibility states: each setting has a label, current value, and help text that are programmatically associated; invalid fields are listed in a summary that receives focus; the confirmation step restates the change in text; saved policy confirmation is announced.

### <a id="jrn-007"></a>JRN-007 Administrator suspends a member

| Item | Description |
|---|---|
| Actor | Network administrator (PER-002); the affected member (PER-001) is notified. |
| Precondition | Target membership is Active; the actor holds the administrator role. |

Steps:
1. The administrator opens the member's record and chooses Suspend.
2. The platform shows the effects in text: the member cannot propose or accept transactions or publish listings; outstanding Proposed transactions involving them are cancelled ([DOM-024](03-domain-model-and-mutual-credit-rules.md#dom-024)); open disputes continue.
3. The administrator enters a mandatory reason and confirms.
4. The membership becomes Suspended; the member's account status changes as defined in [03-domain-model-and-mutual-credit-rules.md](03-domain-model-and-mutual-credit-rules.md) (expected: Frozen); the member is notified of the reason and of the route to challenge the decision ([10-governance-and-network-operations.md](10-governance-and-network-operations.md)).
5. Later, an administrator MAY reinstate the member (Active) with a recorded reason.

Postcondition: membership Suspended, no postings created or altered, balances unchanged, audit event recorded with actor and reason.

Error and edge states:
- Target is the last Active administrator: refused (PER-006).
- Target is a party to an Open or Under review dispute, or the assigned reviewer: the suspension is allowed; the dispute continues, and if the target was the assigned reviewer the dispute returns to Open ([DOM-061](03-domain-model-and-mutual-credit-rules.md#dom-061)), with both recorded.
- Target is a party to a Proposed transaction: the proposal is cancelled by the system with status reason `account_frozen` and the counterparty is notified ([DOM-024](03-domain-model-and-mutual-credit-rules.md#dom-024)); either party may propose again after reinstatement.
- Duplicate or retried submission: one suspension, one notification.
- Target already Suspended or Departed: the action is refused and shows the current status.
- Suspended member signs in: sees own balance and history ([PER-027](#per-027)), can export own data ([PER-030](#per-030)), and sees the reason and challenge route; whether they may raise a dispute against the suspension is open ([OQ-PER-3](#5-open-questions)).
- Suspension with a non-zero balance: balances stay as they are; recovery of a negative balance is a governance matter ([10-governance-and-network-operations.md](10-governance-and-network-operations.md)).

Accessibility states: the confirmation dialog moves focus into itself, restates the member and effects, returns focus to the member record, and announces the new status; the reason field is a labelled text area; the suspended member's notice does not depend on color.

### <a id="jrn-008"></a>JRN-008 Leave a network

| Item | Description |
|---|---|
| Actor | Active member (PER-001). |
| Precondition | Membership Active (a Suspended member may also leave, [DOM-070](03-domain-model-and-mutual-credit-rules.md#dom-070)). The member is not the last Active administrator. |

Steps:
1. The member chooses to leave the network.
2. The platform shows, in text, what leaving involves: current balance, Proposed transactions and open disputes that must be cancelled or resolved first ([DOM-070](03-domain-model-and-mutual-credit-rules.md#dom-070)), active listings that will be closed, and the option to export own data first ([PER-030](#per-030)).
3. If the balance is not zero, the platform states that the member may submit a departure request, which an administrator completes ([DOM-071](03-domain-model-and-mutual-credit-rules.md#dom-071)); see also [10-governance-and-network-operations.md](10-governance-and-network-operations.md).
4. The member confirms.
5. The platform closes their listings, sets the membership to Departed, and sets the account status as defined in [03-domain-model-and-mutual-credit-rules.md](03-domain-model-and-mutual-credit-rules.md).
6. The administrators are notified.

Postcondition: membership Departed; completed postings and history preserved; the person's other memberships are unaffected; audit event recorded.

Error and edge states:
- Negative balance: the member may submit a departure request; an administrator completes it with a `write_off` ([DOM-071](03-domain-model-and-mutual-credit-rules.md#dom-071)); the member is told what happens next, without legal conclusions.
- Positive balance: handling follows the same policy; the platform never converts or redeems it.
- Open dispute involving the member, or a Proposed transaction naming the member's account: departure is blocked (`departure_blocked`) until it is resolved or cancelled.
- Last Active administrator: refused (PER-006); the member is directed to assign another administrator first.
- Duplicate or retried confirmation: one departure.
- Counterparty departed while a proposal to or from them is outstanding: the proposal can no longer be completed and is cancelled with notice to the remaining party.
- Departed person wants to return: handled as a new invitation or application (JRN-001), with the earlier history linked per [10-governance-and-network-operations.md](10-governance-and-network-operations.md).

Accessibility states: the consequences are listed as text before the confirm control; the confirm control is not the default focus; completion is announced and a confirmation email is sent, since the person's access to the network ends.

## 5. Open questions

Each question is mirrored as a decision-log entry in [13-assumptions-risks-decisions-and-roadmap.md](13-assumptions-risks-decisions-and-roadmap.md).

| ID | Question | Decision needed | Decided by |
|---|---|---|---|
| OQ-PER-1 | Can people create their own networks on an instance, or only the platform operator? | Recommended default: operator creates networks (PER-010). Self-service creation changes the operator's role and abuse exposure. | Project owner |
| OQ-PER-2 | Resolved: answered by [DOM-115](03-domain-model-and-mutual-credit-rules.md#dom-115) (members invite only when it is on; default `off`). | None; kept for traceability. | Resolved |
| OQ-PER-3 | What may a Suspended or Departed member still do (read history, export, raise a dispute against the decision), and what happens to their listings? | Read and export rights are resolved: answered by GOV-042, GOV-080, FR-016 and FR-090 (see PER-027 and PER-030). Still open: whether a challenge uses the dispute process or a separate appeal in [10-governance-and-network-operations.md](10-governance-and-network-operations.md), and what happens to listings. | Project owner with the governance document owner |
| OQ-PER-4 | Resolved: when a party's account becomes Frozen or Closed, or a membership departs, outstanding proposals are cancelled by the system ([DOM-024](03-domain-model-and-mutual-credit-rules.md#dom-024)). | None; kept for traceability. | Resolved |
| OQ-PER-5 | Is a view of another member's balance by an administrator or reviewer recorded as an audit event? | Trade-off between accountability and audit-log volume and personal-data minimization. | Security reviewer |
| OQ-PER-6 | Resolved: retention of `Rejected` applications and `Departed` memberships is answered by DATA-080 (2 years); the reapplication view follows [DOM-076](03-domain-model-and-mutual-credit-rules.md#dom-076). | None; kept for traceability. | Resolved |
| OQ-PER-7 | Resolved: a reversal is posted even when it breaches a limit, with the override recorded ([DOM-032](03-domain-model-and-mutual-credit-rules.md#dom-032)). | None; kept for traceability. | Resolved |

## 6. Change history

| Date | Change |
|---|---|
| 2026-11-11 | Initial draft. |
| 2026-11-11 | Consistency review fixes. |
