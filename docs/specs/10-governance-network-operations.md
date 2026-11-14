# Governance and Network Operations

Status: Draft | Owner: Product architect (role) | Updated: 2026-11-14 | Scope: Network administrator authority, policy change, member challenge and appeal, membership lifecycle, administrator continuity, dispute governance, negative-balance recovery, suspension, closure and data export | Deps: [02](./02-personas-roles-journeys.md), [03](./03-domain-model-rules.md), [04](./04-functional-requirements.md), [06](./06-data-ledger-audit.md), [_context](./_context.md) | Open questions: 7, see [Open decisions](#open-decisions)

## Purpose

This spec defines how a network is governed in operation: what a network administrator may configure, how changes are announced and recorded, how members challenge decisions, and how membership, administrator continuity, disputes, balance shortfalls, suspension, closure and exit are handled.

It separates two things:

- **Governance policy**: decisions made by people inside a network (members, network administrators, dispute reviewers). Each network decides these for itself. Software records, applies and checks them, but does not make them.
- **Software enforcement**: behavior the platform guarantees regardless of what a network decides. Requirements in this class are testable on the software alone.

Rules that already have an authoritative home in another spec are referenced by ID and not restated. Mutual-credit mechanics live in [03](./03-domain-model-rules.md). Roles and journeys live in [02](./02-personas-roles-journeys.md). Functional behavior lives in [04](./04-functional-requirements.md). Ledger, retention and audit storage live in [06](./06-data-ledger-audit.md).

## Classification of governed areas

| Area | Governance policy (human decision) | Software enforcement (platform guarantee) |
|---|---|---|
| Configurable policy values | Which values a network chooses | Only listed values are settable; invariants are not (DOM-081) |
| Policy changes | Whether and why to change; the reason given | Notice floor, audit record, no retroactive effect |
| Join | Join rule chosen (open, approval, invitation) | Only that rule creates accounts (ROL-010) |
| Challenge and appeal | Who decides and on what grounds | Challenge is recorded; decider cannot review own decision |
| Disputes | Dispute window and reviewer pool | Participants excluded; no automatic decision |
| Negative balance | Notification threshold; whether to suspend; any write-off | Credit limit enforced (DOM-005); no automatic write-off (DOM-064) |
| Suspension | Cause, procedure, duration | State effects; read access kept; no balance change |
| Closure | Decision, date, settlement method | Blockers listed; records retained; network becomes read-only |
| Exit export | Retention period and post-exit access | Export content and reproducibility |

## Requirements

Priority values follow [_context](./_context.md#normative-terms): MUST, SHOULD, MAY.

### 1. Configurable rules

| ID | Pri | Requirement | Rationale | Test criterion |
|---|---|---|---|---|
| GOV-001 | MUST | A network administrator MUST be able to view and set only the items marked "Network policy" in the configurable table of [03](./03-domain-model-rules.md#configurable-vs-invariant). | Keeps the policy surface defined in one place. | For each "Network policy" item, set and read succeed for an administrator. For a non-listed key, set is refused. |
| GOV-002 | MUST | No governance procedure, administrator role or policy value MUST change an item marked "Platform invariant". Attempts MUST be rejected as DOM-081 requires. | Ledger integrity is uniform across networks. | Attempt to disable atomicity, append-only, or the credit-unit rule through any admin path is refused and an audit event is written. |
| GOV-003 | MUST | Each policy value MUST have a documented default, an allowed range or value set, and an explicit "none" state where the configurable table says the default is none. | Makes new networks predictable and bounds platform behavior. | Create a network with no changes: every value equals its documented default. Out-of-range input is refused with a field-level reason. |
| GOV-004 | MUST | A policy change MUST apply only to postings and transitions at or after its effective time and MUST NOT alter existing entries (DOM-080). | History stays fixed while rules move forward. | Post before the change, apply the change, then compare prior entries and balances: unchanged. |
| GOV-005 | SHOULD | Before confirming a policy change, the administrator SHOULD see the current value, the new value, the effective time, and the number of accounts affected. | Reduces accidental broad changes (see FR-033). | Confirmation view shows all four fields; the count matches a query of active accounts. |

### 2. Change announcement and record

| ID | Pri | Requirement | Rationale | Test criterion |
|---|---|---|---|---|
| GOV-006 | MUST | Every policy change, join-rule change, role grant or revocation, and standing change MUST produce an audit event with actor, old value, new value, time and reason (DOM-070, DOM-072, ROL-006). | Changes are reconstructable and attributable. | For each change type, one audit event exists containing all listed fields. |
| GOV-007 | MUST | A policy change MUST NOT be saved without a reason category. Free-text detail MAY be optional. | Gives members a basis for challenge. | Submit without a reason category: refused. |
| GOV-008 | MUST | Before a policy change takes effect, it MUST be announced to every active member of the network through the network's notification channel. The announcement MUST state the change, the effective time and the reason category. | Members must learn of rules before they apply. | For a test change, every active member has a delivered announcement record whose time precedes the effective time. |
| GOV-009 | MUST | The time between announcement and effect MUST be at least the floor set in OD-10-1. The platform MUST refuse a schedule shorter than the floor. | Gives members time to respond. | Schedule a change inside the floor: refused. Schedule at the floor: accepted. |
| GOV-010 | SHOULD | A network SHOULD publish its policy history to members: each past and current value with its effective dates. | Lets members verify what applied when. | A member can list past values and effective dates for each policy item. |

### 3. Challenge and appeal

| ID | Pri | Requirement | Rationale | Test criterion |
|---|---|---|---|---|
| GOV-011 | MUST | A member MUST be able to challenge each of the following decisions that affect them: join denial (ROL-013), listing removal (ROL-021, FR-017), suspension of their account (DOM-063, FR-009), and their departure when initiated by an administrator (GOV-024). | Affected members need a route to contest decisions. | For each listed decision type, a challenge control is available to the affected member and not to others. |
| GOV-012 | MUST | Dispute outcomes are not challenged under this spec. A participant who disagrees with an outcome MAY raise a new dispute only where DOM-054 allows. | Avoids two review tracks for the same transaction. | A challenge filed against a dispute outcome is refused with a pointer to DOM-054. |
| GOV-013 | MUST | Filing a challenge MUST NOT by itself change a balance, transaction status or account standing. | A challenge must not be a lever for unilateral effects. | Before and after a filed challenge, balances, transaction statuses and standing are identical. |
| GOV-014 | MUST | A challenge MUST be filed within the challenge window set by the network. The window MUST be a network policy value with a documented default. | Bounds stale challenges. | Challenge after window end is refused; within window it is accepted. |
| GOV-015 | MUST | A challenge MUST be decided by a person other than the one who made the decision under challenge. The platform MUST refuse assignment of the original decider. The fallback when no second person is available is set in OD-10-2. | Independent review. | Assign the original decider as reviewer: refused and audited. |
| GOV-016 | MUST | The outcome of a challenge (upheld, overturned or modified) MUST be recorded with a reason, and MUST be notified to the challenger and the original decider. Notices MUST follow the content limits of ROL-013. | Closes the loop and keeps notices privacy-safe. | Outcome record exists; notice contents match the allowed-field list. |
| GOV-017 | SHOULD | A network SHOULD set a decision deadline for challenges and, when it passes without a decision, notify the network administrators. | Prevents challenges that stall. | With a test deadline passed and no decision, administrators receive a notice. |

### 4. Onboarding and offboarding

| ID | Pri | Requirement | Rationale | Test criterion |
|---|---|---|---|---|
| GOV-018 | MUST | Admission MUST follow only the network's join rule, as ROL-010 and the join requirements in [04](./04-functional-requirements.md#membership) define. | One governed path to membership. | Each join-rule value produces only its defined path. |
| GOV-019 | MUST | Before an account is created, the prospective member MUST record the acknowledgment defined in ROL-015 against the current policy version. | Members join knowing the rules in force. | No account exists without an acknowledgment record that references a policy version. |
| GOV-020 | MUST | On account creation, the platform MUST store the policy version acknowledged. | Supports later challenge and audit. | Account record references the acknowledged policy version. |
| GOV-021 | MUST | Departure MUST proceed only under the preconditions in DOM-060. Where departure is blocked, the member MUST be shown each blocking item and what is needed to clear it. | A departing member must know what remains open. | A departure attempt with an open transaction shows that transaction as a blocker. |
| GOV-022 | MUST | A member-initiated departure MUST NOT require administrator approval. | Exit is a member right within DOM-060. | Member departure with no blockers completes without an administrator action. |
| GOV-023 | MUST | Settlement of a non-zero balance at departure MUST follow DOM-062 and the network's settlement policy (OD-04 in [03](./03-domain-model-rules.md#open-decisions)). The member MUST see the balance and the proposed settlement before any settlement posts. | No silent zeroing; the member sees the effect first. | Departure with non-zero balance: settlement preview shown; no adjustment posted before confirmation. |
| GOV-024 | MUST | An administrator-initiated departure MUST record a reason category, MUST notify the member, and MUST be challengeable under GOV-011. | Protects members from unexplained exclusion. | Admin departure produces a reason record, a member notice and a challenge control. |

### 5. Administrator continuity and succession

| ID | Pri | Requirement | Rationale | Test criterion |
|---|---|---|---|---|
| GOV-025 | MUST | A network MUST keep at least the minimum number of active network administrators set in OD-10-3. The platform MUST refuse any role change or departure that would drop the count below that minimum. | A network must always have someone able to govern it. | With the minimum at two and two administrators, removing one is refused. |
| GOV-026 | MUST | Granting the network administrator role MUST be an action by an existing network administrator, or by the path OD-10-3 defines for a network with no active administrator. It MUST be audited (ROL-006). | Controls who gains authority. | Grant by a non-administrator is refused except via the OD-10-3 path; the grant creates an audit event. |
| GOV-027 | MUST | Each network MUST have a documented succession path for loss of all administrator access, as set in OD-10-3. Recovery MUST NOT grant any right beyond the network administrator role; in particular it MUST NOT grant posting, balance or dispute-decision rights to any party (ROL-005). | Recovery should restore governance without a new back door. | Run the recovery path in a test network: resulting roles equal the administrator role only. |
| GOV-028 | SHOULD | Administrator actions SHOULD be visible to all active administrators of the network in one governance log. | Shared oversight among administrators. | Each administrator sees the same log entries for a test set of actions. |

### 6. Dispute governance

| ID | Pri | Requirement | Rationale | Test criterion |
|---|---|---|---|---|
| GOV-029 | MUST | Dispute reviewers MUST be drawn from the network's dispute reviewers and MUST NOT be participants in the disputed transaction (ROL-004, DOM-051). A reviewer MUST NOT decide a dispute on a transaction that is the subject of a challenge they decided (GOV-015). | Impartiality. | Assignment tests exclude participants and the original decider. |
| GOV-030 | MUST | The dispute window and dispute reviewer pool MUST be network policy values, as the configurable table in [03](./03-domain-model-rules.md#configurable-vs-invariant) states. A change to the window MUST NOT apply to transactions completed before the change (GOV-004). | Consistent rules over time. | Change the window; a transaction completed earlier keeps its original window. |
| GOV-031 | MUST | A dispute outcome MUST cite its authority (DOM-042, DOM-053), MUST record a reason, and MUST notify both participants. | Outcomes are explainable. | Each outcome record has authority, reason and two participant notices. |
| GOV-032 | MUST | If no eligible reviewer exists, the dispute MUST remain open. The platform MUST notify network administrators and MUST NOT resolve the dispute automatically. | No default outcome by silence. | With an empty eligible pool, the dispute stays open and an administrator notice is recorded. |
| GOV-033 | MUST | The platform MUST NOT decide, reverse or adjust a disputed transaction without a recorded reviewer outcome or a recorded mutual agreement under DOM-042. | Keeps decisions with people. | No automatic status change on a disputed transaction occurs in test runs without a reviewer outcome. |

### 7. Negative-balance recovery

| ID | Pri | Requirement | Rationale | Test criterion |
|---|---|---|---|---|
| GOV-034 | MUST | A balance below zero MUST NOT go below the account's credit limit (DOM-005, DOM-043). This spec adds no path around that rule. | Floor is a platform check. | A posting that would breach the limit is refused. |
| GOV-035 | SHOULD | When an account's balance is at or below the notification threshold set in OD-10-4, the member SHOULD receive a notice with the current balance, the credit limit, and the ordinary ways to restore balance. | Early warning before limits bind. | Set a test threshold; crossing it produces one notice to the member. |
| GOV-036 | MUST | A member's balance MUST improve only through completed transactions (DOM-016). The platform MUST NOT offer automated netting, forgiveness or write-off. | Recovery stays mutual and traceable. | No endpoint or job changes a balance except completion postings and recorded adjustments. |
| GOV-037 | MUST | Any write-off, forgiveness or default settlement MUST be a network decision, recorded as an adjustment that cites its authority (DOM-042, DOM-064) and a reason. The platform MUST NOT apply it automatically. | Write-off is a governance act, not a software default. | Without a recorded decision, no write-off adjustment exists. |
| GOV-038 | MAY | A network MAY define a non-performance procedure: notice, then suspension of the account (DOM-063, FR-009), then a challenge window (GOV-011). Each step MUST be recorded. | Lets networks handle default in a documented, reviewable way. | Run the procedure in a test network; each step has a record and a challenge control. |

### 8. Service suspension

| ID | Pri | Requirement | Rationale | Test criterion |
|---|---|---|---|---|
| GOV-039 | MUST | A network-wide suspension (no new proposals, acceptances or completions) MUST be initiated only by a network administrator, or by the platform operator under the conditions in OD-10-5. Each suspension MUST record initiator, reason category, scope, start time and expected end. | Clear authority and a record of who stopped the network. | Attempt by another role is refused; a valid suspension record has all fields. |
| GOV-040 | MUST | During a network-wide suspension, the platform MUST refuse new postings and transitions, and MUST keep member read access to balances, history and exports (FR-044). | Members keep visibility of their position while trading stops. | Write attempts are refused; history and export requests succeed. |
| GOV-041 | MUST | During any suspension, the platform operator MUST NOT change balances, entries or policy (ROL-005). | Suspension stops activity; it does not rewrite the ledger. | Operator attempts to post or change policy during suspension are refused. |
| GOV-042 | MUST | Starting or lifting a network-wide suspension MUST be announced to members and recorded (GOV-006, GOV-008). The notice period in GOV-009 does not apply to suspensions. | Members must know the network is not operating. | Suspension start and lift each produce a member announcement and an audit event. |
| GOV-043 | MUST | If the platform operator initiates a suspension, the network administrators MUST receive the cause and scope at the time of action, and the network MUST have a route to challenge it. | Operator action must be visible and contestable by the network. | Operator suspension produces an administrator notice and a challenge record path. |

### 9. Network closure

| ID | Pri | Requirement | Rationale | Test criterion |
|---|---|---|---|---|
| GOV-044 | MUST | Closure MUST be a network administrator decision, recorded with a reason category and a closure date. Members MUST be notified before the closure date, with notice of at least the floor in OD-10-1. | Closure is a governed, announced event. | Closure record has all fields; notice reaches all active members before the date. |
| GOV-045 | MUST | Before closure takes effect, the platform MUST block new proposals and listings network-wide. Each PROPOSED transaction MUST be cancelled, and each ACCEPTED transaction MUST be completed or cancelled under DOM-013. Each open dispute MUST be resolved, or listed as an open item in the closure record. | Closure must not leave obligations half-formed. | Closure precondition check lists every remaining item; closure cannot complete while a PROPOSED transaction exists. |
| GOV-046 | MUST | Non-zero balances at closure MUST be settled by the method set in OD-10-6. Any change to a balance MUST be a traceable adjustment (DOM-062, DOM-064). No balance may be zeroed silently. | Same integrity rules as departure. | After closure, the sum of balances is unchanged by any step that is not a recorded adjustment. |
| GOV-047 | MUST | Closure MUST NOT delete ledger, audit or governance records. They MUST be kept under the retention rules in [06](./06-data-ledger-audit.md#retention) and the period set in OD-10-7. | Records remain reconstructable after closure. | After closure, a record count query matches the pre-closure count. |
| GOV-048 | MUST | After closure takes effect, the network MUST be read-only: no postings, no new accounts, no new listings and no role grants. | A closed network must not change. | Each write attempt on a closed network is refused. |

### 10. Data export on exit

| ID | Pri | Requirement | Rationale | Test criterion |
|---|---|---|---|---|
| GOV-049 | MUST | A departing member MUST be offered an export of their own entries (FR-044) during the departure flow. An export MUST NOT be a precondition for departure. | Exit includes the member's data. | Departure flow shows an export option; departure completes whether or not the export is taken. |
| GOV-050 | MUST | Exports MUST contain only the fields allowed by FR-046 and the export scope in [04](./04-functional-requirements.md#open-decisions) (OD-04-6). Each export MUST be reproducible: balances recomputed from the export equal stored balances (FR-045). | Exports are minimal and verifiable. | Recompute each balance from an export: all match. Field list matches the published scope. |
| GOV-051 | MUST | Member and administrator exports MUST remain available for the post-exit period set in OD-10-7. | Members and administrators need access after exit or closure. | Request an export inside and after the period: available inside, refused after, with the refusal reason recorded. |
| GOV-052 | SHOULD | Each export request SHOULD be recorded as an audit event containing actor, scope, time and format, not the exported content. | Accountability without duplicating data. | Export produces one audit event with those fields and no content. |

### 11. Operator boundaries

| ID | Pri | Requirement | Rationale | Test criterion |
|---|---|---|---|---|
| GOV-053 | MUST | The platform operator MUST NOT decide disputes, change policy, post entries or change balances in any network (ROL-005). | Operator is outside network governance. | Each listed action by an operator account is refused. |
| GOV-054 | MUST | Operator access to governance records or member content MUST follow the authorization model in OD-06-5 of [06](./06-data-ledger-audit.md#open-decisions). Every such access MUST produce an audit event. | Access is controlled and visible. | Operator read of a record without authorization is refused; authorized reads produce events. |

## Out of scope

- Legal conclusions on notice periods, retention periods, consumer rights or regulatory duties. These are handled as open decisions, not requirements.
- Voting or token-weighted governance. Not part of this spec.
- Cross-network governance and federation (deferred in [04](./04-functional-requirements.md#deferred-work)).
- Monetization of network administration.

## Open decisions

- **OD-10-1 Notice floor for policy changes and closure (GOV-009, GOV-044).** Options: (a) no floor, network chooses; (b) fixed platform floor; (c) platform floor plus longer network-set notice. **Default: (c)**, with the floor value to be set by the product owner. Option (a) weakens member protection; option (b) is inflexible for small networks.

- **OD-10-2 Challenge reviewer when few administrators exist (GOV-015).** Options: (a) a second administrator reviews; (b) a dispute reviewer who is not a participant reviews; (c) platform operator checks process only, and the network decides. **Default: (a)**, with a recommended minimum of two administrators per OD-10-3. Where neither (a) nor (b) is possible, the challenge stays open and administrators are notified (GOV-017). Option (c) gives the operator no decision authority over balances (ROL-005), but it does not produce a decision.

- **OD-10-3 Minimum administrator count and succession recovery (GOV-025 to GOV-027).** Options: (a) platform enforces a minimum of two active administrators and a documented recovery path that restores only the administrator role; (b) minimum of one, with a designated successor recorded in advance; (c) operator-assisted re-grant of the administrator role after verified identity, audited. **Default: (a)** with the recovery path defined as in (c). Option (b) fails if the successor is also unavailable.

- **OD-10-4 Negative-balance notice threshold (GOV-035).** Options: (a) no threshold notice; (b) network-set threshold with a platform default; (c) platform-fixed threshold. **Default: (b).** The default threshold value is to be set by the product owner.

- **OD-10-5 Network-wide suspension by the platform operator (GOV-039, GOV-043).** Options: (a) operator cannot suspend a network; only infrastructure-level outage handling applies; (b) operator may suspend for defined causes (for example, security incident or unlawful use, causes to be defined), with immediate notice to administrators, read access kept, and a challenge route; (c) operator may suspend without stated cause. **Default: (b).** Option (c) is excluded because it removes accountability.

- **OD-10-6 Settlement of non-zero balances at network closure (GOV-046).** Options: (a) balances kept as final historical record, no settlement; (b) network chooses a settlement method, each step a traceable adjustment; (c) mandatory single method set by the platform. **Default: (b).** Option (c) conflicts with network-level governance; option (a) leaves balances that no longer have a counterparty.

- **OD-10-7 Retention and post-exit access period (GOV-047, GOV-051).** Options: (a) retain indefinitely, read-only; (b) retain for a network-set period with a platform minimum; (c) delete on request where permitted. **Default: (b).** Any retention duration required by law is outside this spec and must be decided by the network and its advisers, not by the software. Option (c) interacts with append-only rules in [03](./03-domain-model-rules.md) and must be checked against them before adoption.

## Related docs

- [_context](./_context.md): shared terms, normative terms, ledger invariants, configurable vs invariant.
- [02 Personas, roles and core journeys](./02-personas-roles-journeys.md): roles (ROL-001 to ROL-006), join and rule-administration journeys.
- [03 Domain model and mutual-credit rules](./03-domain-model-rules.md): configurable-vs-invariant table, departure, dispute, policy rules (DOM-0xx).
- [04 Functional requirements](./04-functional-requirements.md): membership, administrator controls, disputes, exports (FR-0xx).
- [06 Data, ledger and audit](./06-data-ledger-audit.md): retention, audit content, access and operator authorization.
- [07 API and integration contract](./07-api-integration-contract.md): interface behavior for the operations above.
- [13 Assumptions, risks, decisions and roadmap](./13-assumptions-risks-decisions-roadmap.md): tracking of open decisions in this spec.
