# Security, Privacy and Abuse Resistance

Status: Draft | Owner: Security reviewer (role) | Updated: 2026-11-14 | Scope: Assets, actors, trust assumptions, threat model and requirements for authentication, authorization, recovery, administrator misuse, disputes, abuse reporting, rate limiting, logging, backups, incident response, data minimization, and export and visibility controls | Deps: [Shared context](_context.md), [02 Personas and roles](./02-personas-roles-journeys.md), [03 Domain model](./03-domain-model-rules.md), [05 Architecture](./05-system-architecture.md), [06 Data, ledger and audit](./06-data-ledger-audit.md), [07 API contract](./07-api-integration-contract.md) | Open questions: 16, see "Open decisions"

## Purpose

This spec sets the security, privacy and abuse-resistance requirements for the Circula platform at requirements level. It defines what is protected, who can act, what is assumed to be trustworthy, which threats are considered, and the observable behavior that addresses them.

It does not restate rules owned by other specs. Role rights live in [02](./02-personas-roles-journeys.md). Ledger invariants and dispute rules live in [03](./03-domain-model-rules.md). Trust boundaries (TB-1 to TB-6) and component ownership live in [05](./05-system-architecture.md). Entry, audit and access rules live in [06](./06-data-ledger-audit.md). Interface error, authorization and idempotency rules live in [07](./07-api-integration-contract.md). Where this spec cites an ID from those files, it cites it; it does not repeat it.

Normative terms (MUST, SHOULD, MAY) and glossary terms are defined in the [shared context](_context.md).

This spec makes no claim of anonymity, cryptographic guarantee, or regulatory compliance. Requirements here are verified by observable tests. Questions that need legal or counsel input are listed in "Open decisions" and are not decided here.

## 1. Assets

Each asset class below is protected at the level stated by the requirements that cite it.

| Asset | Description | Primary owner in [05](./05-system-architecture.md) | Main requirements |
|---|---|---|---|
| Entries and balances | Posted entries and derived balances per account | Ledger | SEC-001, SEC-009, SEC-010 |
| Audit events | Append-only record of security- and accounting-relevant actions | Audit | SEC-017, SEC-038 |
| Credentials, second factors, sessions | Authentication material and session state | Identity and access | SEC-002, SEC-004 to SEC-006 |
| Roles and network policy | Role assignments, reviewer roster, policy versions | Accounts and membership; Policy | SEC-009, SEC-018 |
| Dispute records | Disputes, reviewer decisions, linked evidence text | Disputes | SEC-022, SEC-023 |
| Member profile data | Fields outside the ledger (see [DAT-090](./06-data-ledger-audit.md)) | Accounts and membership | SEC-052 to SEC-056 |
| Listings and reports | Offers, requests, abuse reports and their free text | Listings; Reporting | SEC-026 to SEC-031 |
| Exports | Point-in-time copies of network data | Audit; export function | SEC-059 to SEC-064 |
| Backups | Copies of primary storage | Operations | SEC-041 to SEC-045 |
| Operational logs | Request, error and rate-limit records | Operations | SEC-036 to SEC-040 |

SEC-001 (MUST) Every data store, log stream, export format and backup in the architecture MUST be classified against this asset list before it is introduced. A store without a classification MUST NOT be merged.
- Rationale: unclassified data tends to receive no controls.
- Test: a review checklist applied to each component in [05 section 1](./05-system-architecture.md#1-logical-components) and to each new storage, log or export change shows a classification for every item. A fixture change adding an unclassified store fails the check.

## 2. Actors

| Actor | Description | Trust level |
|---|---|---|
| Unauthenticated visitor | Any party without a session | Untrusted |
| Member | Account holder in one or more networks | Untrusted in intent; authorized only for own rights |
| Network administrator | Governs one network | Trusted for own network only; may err or act in bad faith |
| Dispute reviewer | Assigned per dispute | Trusted for assigned disputes only |
| Platform operator | Runs the software and infrastructure | Trusted for infrastructure; not for network rights (TB-4) |
| Scheduler service identity | Runs background jobs | Narrow permissions only (TB-5) |
| Colluding group | Two or more members acting together | Treated as a realistic adversary |
| Insider with infrastructure access | Staff or automation with storage or log access | Treated as a realistic adversary for data confidentiality and integrity |

## 3. Trust assumptions

These are assumptions the design relies on. If one fails, the affected requirements still hold but their coverage is reduced, and the incident process applies (section 5.9).

1. Client devices and client code are untrusted. Any input may be forged ([TB-1](./05-system-architecture.md#3-trust-boundaries)).
2. Storage is trusted for atomicity and uniqueness only, not for business rules ([TB-3](./05-system-architecture.md#3-trust-boundaries)).
3. The platform operator can reach infrastructure and storage. Storage access by the platform operator is not, by itself, a control on data confidentiality.
4. A network administrator is trusted only for the rights their role grants in their own network.
5. Members may collude. A single report or dispute is not proof of wrongdoing.
6. Channels used for recovery or notification (for example, a registered contact method) can be compromised or shared by their owner.
7. Users may conduct agreements off the platform. The software cannot observe or enforce those agreements.
8. Network rules, including membership and sanctions, are set by each network. The software enforces the rules it has been configured with.

## 4. Practical threat model

Each threat is addressed by the requirements listed. "Residual" states what remains outside the software's control.

| # | Threat | Actor | Assets | Addressed by | Residual |
|---|---|---|---|---|---|
| 1 | Password guessing or credential stuffing against accounts | External | Credentials, balances | SEC-003, SEC-006, SEC-007, SEC-033 | Reused credentials remain a user risk |
| 2 | Session theft, fixation or long-lived sessions | External | Sessions | SEC-004, SEC-005 | Theft on a compromised device is not prevented |
| 3 | Takeover of a member account, then trades that benefit the attacker | External | Balances, listings | SEC-006, SEC-008, SEC-015, SEC-034 | Detection depends on counterpart reports |
| 4 | Social engineering of account recovery | External | Credentials | SEC-012 to SEC-016 | Human verification is only as strong as OD-09-3 |
| 5 | Misuse of network administrator rights | Network administrator | Policy, roles, listings, disputes | SEC-017 to SEC-019, SEC-060 | Administrator decisions are audited, not prevented by the software |
| 6 | Platform operator reads or alters network data | Platform operator | All | SEC-020, SEC-038, SEC-065 | Infrastructure-level compromise is covered only by detection and incident response |
| 7 | Data leaks across networks | Member or administrator of another network | Balances, entries, audit | SEC-011, SEC-067, [DAT-081](./06-data-ledger-audit.md) | Depends on enforcement in every read path |
| 8 | Forged amounts, statuses or permissions in requests | Member | Ledger, workflow | SEC-010, [ARC-009](./05-system-architecture.md) | None beyond server validation |
| 9 | Retaliatory or harassing disputes | Member | Counterpart standing, disputes | SEC-022, SEC-023, SEC-029 | Disputes remain available to any eligible participant |
| 10 | Collusion between members or between a reviewer and a participant | Colluding group | Disputes, balances | SEC-018, SEC-024, SEC-026, OD-09-7 | Collusion can be hidden; signals are probabilistic |
| 11 | Spam or abusive listings | Member | Listings | SEC-026, SEC-030, SEC-032 | Moderation quality depends on administrators |
| 12 | Abuse of the reporting channel to harass a member | Member | Member privacy | SEC-026, SEC-027, SEC-031 | Reports remain possible and must be reviewed by people |
| 13 | Exfiltration through exports | Administrator or compromised administrator | Exports, personal data | SEC-059 to SEC-062 | An administrator with valid rights can still export their network |
| 14 | Secrets or personal data exposed through logs or errors | Insider | Logs, credentials | SEC-036, SEC-039 | Depends on coverage of log filters and tests |
| 15 | Backup theft or rollback of state | Insider or infrastructure | Backups, ledger | SEC-041 to SEC-044 | Loss of recent entries is recoverable only through the incident process |
| 16 | Denial of service on posting and authentication endpoints | External | Availability | SEC-032, SEC-033, SEC-034 | Volumetric attacks depend on infrastructure, outside this spec |
| 17 | Enumeration of members, accounts or records | External or member | Member existence | SEC-003, SEC-013, [API-013](./07-api-integration-contract.md) | Public listings may reveal existence by design (OD-09-16) |
| 18 | Over-collection of personal data | Platform | Member profile data | SEC-052 to SEC-055 | Future feature requests can add data; SEC-054 governs them |

## 5. Requirements

Requirements use stable IDs. Each has a priority, a rationale and a test. Rights of roles are not repeated here: see [02](./02-personas-roles-journeys.md) and [07 section Authentication and authorization](./07-api-integration-contract.md#authentication-and-authorization-expectations).

### 5.1 Authentication

| ID | Priority | Requirement | Rationale | Test |
|---|---|---|---|---|
| SEC-002 | MUST | Authentication secrets MUST NOT be stored in plaintext or in a reversible form. Stored verifiers MUST be designed for password-like input. | Limits damage from storage read access. | Inspect a seeded test account's stored record and the backup of it. No plaintext or reversible value is found. |
| SEC-003 | MUST | Authentication responses MUST NOT reveal whether an account identifier exists. Failed attempts for existing and non-existing identifiers MUST produce the same response shape and error code. | Prevents account enumeration. | Compare responses for an existing and a non-existing identifier. Shape and code are identical. |
| SEC-004 | MUST | Sessions MUST expire after a period of inactivity and after an absolute maximum lifetime. Both values MUST be platform-bounded (OD-09-2). | Bounds exposure of stolen sessions. | Use a session after the inactivity limit and after the absolute limit. Both are refused. |
| SEC-005 | MUST | Session identifiers MUST be unpredictable, MUST NOT appear in URLs, and MUST be invalidated on logout, on credential change, and on role revocation that affects the session. | Limits session fixation and reuse. | Logout, then reuse the old identifier: refused. Revoke a role, then use the session for a dependent operation: refused on next request ([API-015](./07-api-integration-contract.md)). |
| SEC-006 | MUST | Network administrators, dispute reviewers and platform operators MUST authenticate with a second factor in addition to the primary credential. The set of accepted second-factor types is OD-09-1. | Privileged accounts are the highest-value targets. | Sign in as a privileged role with only the primary credential: the operation is refused until the second factor is supplied. |
| SEC-007 | MUST | Failed authentication attempts MUST be rate-limited per account identifier and per source (SEC-033). After a limit is reached, further attempts MUST be refused for a period, and the refusal MUST be logged as an operational record. | Slows online guessing. | Submit attempts above the limit. Later attempts are refused with a stable code. A log record exists for each refusal. |
| SEC-008 | MUST | Changing a credential or second factor MUST require a currently authenticated session and a second-factor confirmation where SEC-006 applies. Each change MUST produce an audit event. | Prevents takeover by a stolen session alone. | Attempt a change with a session lacking second-factor confirmation: refused. A successful change creates one audit event. |
| SEC-009 | MUST | Authorization MUST be deny-by-default. An operation not explicitly granted to a role in the target network MUST be refused. | Limits effect of new endpoints or mistakes. | Add a test operation with no grant. Every role is refused. |
| SEC-010 | MUST | Authorization decisions MUST use server-side role state read at request time. Role or scope claims supplied by the client MUST NOT be trusted. | Client input is untrusted ([TB-1](./05-system-architecture.md#3-trust-boundaries)). | Send a request with a forged role claim. The claim is ignored and the server decision is used. |
| SEC-011 | MUST | Each request MUST bind to exactly one network context. A request whose target record belongs to a different network MUST be refused without revealing the record ([API-013](./07-api-integration-contract.md)). | Prevents cross-network access by members of several networks. | Use a session for network A to read a record in network B. The refusal shape matches a nonexistent record. |

### 5.2 Account recovery

| ID | Priority | Requirement | Rationale | Test |
|---|---|---|---|---|
| SEC-012 | MUST | Account recovery MUST restore authentication only. Recovery MUST NOT grant any ledger, policy, role or dispute right, and MUST NOT change balances. | Recovery is a common path for takeover. | After recovery, the account's roles, balances and entries are unchanged. |
| SEC-013 | MUST | Recovery MUST require verification through a method the account holder registered before the request, or through the out-of-band process in OD-09-3. Recovery initiation and completion MUST each produce an audit event. | Prevents recovery by anyone who only knows an identifier. | Attempt recovery for an account without a registered method: refused. Successful recovery produces two audit events. |
| SEC-014 | MUST | Recovery for network administrators, dispute reviewers and platform operators MUST require at least two independent approvals. Neither approver MUST be the requesting identity, and the platform operator MUST NOT be the sole approver for any recovery. The approval model is OD-09-5. | Prevents a single actor from taking over a privileged account. | A recovery with one approval is not completed. A recovery approved by the platform operator alone is not completed. |
| SEC-015 | MUST | On successful recovery, all existing sessions and second factors for the account MUST be invalidated. Recovery MUST notify the account through each registered contact method, if any. | Ends an attacker's access once the holder regains control. | After recovery, an old session is refused. A notification is sent to each registered method (fixture). |
| SEC-016 | SHOULD | Recovery SHOULD include a waiting period between initiation and completion. Its length is set by OD-09-4. A holder who has a registered method MAY be notified during the waiting period. | Gives the holder time to cancel. | Initiate recovery and try completion before the period ends: refused. Cancellation before the period ends prevents completion. |

### 5.3 Administrator misuse and separation of duties

| ID | Priority | Requirement | Rationale | Test |
|---|---|---|---|---|
| SEC-017 | MUST | Every privileged administrative action listed in [API-014](./07-api-integration-contract.md) MUST produce an audit event that records actor, target, reason and time, in the same atomic unit as the action. | Makes administrator actions attributable. | Perform each listed action. Each produces an audit event with all fields. A failure in the audit write leaves no change. |
| SEC-018 | MUST | A network administrator MUST NOT assign themselves or any identity they control as dispute reviewer for a dispute in which that identity is a participant. | Limits self-dealing through roster control. | Assign the administrator as reviewer on a dispute where they are a participant: refused. |
| SEC-019 | SHOULD | Bulk administrative actions (for example, removing many listings or revoking several roles in one request) SHOULD require an explicit confirmation step, and each item MUST still be recorded as its own audit event. | Reduces accidental and abusive mass changes. | Submit a bulk removal without confirmation: refused. With confirmation, the audit log has one event per item. |
| SEC-020 | MUST | Any platform operator access to network records outside infrastructure operations MUST be recorded as an audit event with reason, and that event MUST appear in the network's audit export ([SEC-063](#511-network-level-visibility-and-export-controls)). | Platform operators hold infrastructure access but no network rights ([TB-4](./05-system-architecture.md#3-trust-boundaries)). | Perform a platform-operator read of a ledger record. The audit event exists and is included in the next network export. |
| SEC-021 | MUST | No administrative action MAY remove or hide an audit event from the network's audit export. | Keeps administrator actions reviewable by the network. | Attempt to hide or delete an event through each administrative path: refused. |

### 5.4 Transaction disputes (security aspects)

Dispute rules are owned by [03 section Disputes](./03-domain-model-rules.md#disputes). The requirements below cover abuse and access only.

| ID | Priority | Requirement | Rationale | Test |
|---|---|---|---|---|
| SEC-022 | MUST | Dispute actions (raise, assign reviewer, record decision) MUST be rate-limited per member and per network. Limits are platform defaults and MAY be tightened by the network (OD-09-6). | Bounds harassment through disputes. | Raise disputes above the limit: refused with a stable code. No balance or status change results from a refused attempt. |
| SEC-023 | MUST | Dispute details MUST be visible only to the two participants, assigned reviewers, and network administrators under [DAT-080](./06-data-ledger-audit.md). They MUST NOT appear in any member-facing listing or in another member's view. | Protects the counterpart's standing during review. | Read a dispute as a third member: no record or existence signal is returned. |
| SEC-024 | MAY | The system MAY expose to network administrators a count of disputes and reports between the same pair of identities. If implemented, it MUST show counts only, MUST NOT be visible to members, and MUST be governed by OD-09-7. | Signal for collusion without automatic action. | If enabled, the view shows counts for an administrator and nothing for a member. |

### 5.5 Fraud and abuse reporting

| ID | Priority | Requirement | Rationale | Test |
|---|---|---|---|---|
| SEC-026 | MUST | A member MUST be able to report a listing, transaction, dispute or member within their network. Each report MUST store reporter, target, reason code, time, and optional bounded free text. | Gives members a channel for problems that the ledger cannot see. | Submit one report of each target type. Each stored record has all fields. |
| SEC-027 | MUST | Reports MUST be visible to network administrators only. The reported party MUST NOT learn the reporter's identity through any API, notice or export. | Reduces retaliation. | Read reports as the reported member and as a third member: not returned. As an administrator: returned with reporter identity. |
| SEC-028 | MUST | Reporting MUST NOT change balances or account status by itself. Any restriction resulting from a report MUST be an administrative action with a reason and an audit event. | A report is a signal, not a finding. | Submit reports on an account. No balance or status changes occur until an administrator acts. |
| SEC-029 | SHOULD | Report submission SHOULD be rate-limited per reporter. Administrators SHOULD be able to see if one reporter files a high volume of reports or reports that are repeatedly dismissed. | Discourages abusive reporting. | Submit reports above the limit: refused. Dismissed-report counts are available to administrators. |
| SEC-030 | MUST | Multiple reports against the same target within the same review period MUST be linked to one case. They MUST NOT each trigger a separate action. | Prevents pile-on effects. | Submit five reports on one target. One case exists and no more than one action is recorded. |
| SEC-031 | MUST | Report free text MUST be bounded in length. Reports MUST NOT require or display personal data of the reported member beyond the target reference. | Limits privacy exposure in reports. | Submit a report above the length bound: refused. The report view shows no personal fields of the target. |

### 5.6 Rate limiting

| ID | Priority | Requirement | Rationale | Test |
|---|---|---|---|---|
| SEC-032 | MUST | Every MVP operation MUST be subject to rate limits per authenticated identity and per source. Exceeding a limit MUST return a refusal with a stable code and a retry indication ([API-020](./07-api-integration-contract.md)) and MUST NOT change state. | Limits abuse and denial of service. | Exceed the limit on each operation. Refusal is returned and state is unchanged. |
| SEC-033 | MUST | Authentication endpoints MUST be rate-limited per source and per account identifier, independently of SEC-032. | Pre-authentication attacks have no identity to limit. | Submit attempts from one source across many identifiers, and against one identifier from many sources. Both are limited. |
| SEC-034 | MUST | A retry of an already processed request, as defined by [API-040](./07-api-integration-contract.md) and [API-041](./07-api-integration-contract.md), MUST return the stored result and MUST NOT be refused by rate limits. | Keeps idempotent retries safe under load. | Exceed the limit, then retry a completed posting request with the same key. The stored result is returned and no new posting is created. |
| SEC-035 | SHOULD | Rate-limit refusals SHOULD be recorded as operational records, not as audit events, unless the refused operation is itself privileged. | Keeps audit store focused ([DAT-063](./06-data-ledger-audit.md)). | A refusal appears in operational logs and not in the audit store. |

### 5.7 Secure logging

| ID | Priority | Requirement | Rationale | Test |
|---|---|---|---|---|
| SEC-036 | MUST | Operational logs MUST NOT contain credentials, session identifiers, second-factor material, recovery material, or personal data beyond identifiers. | Logs are widely readable inside operations. | Run a test suite with seeded secrets and personal fields. Scan all logs. No seeded value is found. |
| SEC-037 | MUST | Each log entry MUST carry a UTC timestamp and a correlation identifier linking it to the request that caused it. Application components MUST NOT be able to edit or delete existing log entries. | Supports incident reconstruction. | Trace one request across components by its identifier. Attempt an edit from application credentials: refused. |
| SEC-038 | MUST | Operational logs and audit events MUST be stored separately ([DAT-063](./06-data-ledger-audit.md)). | Keeps retention and access rules distinct. | Inspect storage configuration: separate stores. |
| SEC-039 | MUST | Client-facing errors MUST NOT include stack traces, internal identifiers, query text or storage details. | Prevents information leakage. | Trigger internal failures. Responses contain only the stable error format from [API-020](./07-api-integration-contract.md). |
| SEC-040 | SHOULD | Access to operational logs SHOULD be limited to named platform operator roles and each access SHOULD be recorded. Log retention is set by OD-09-9. | Logs are a common source of exposure. | Access logs as an unnamed or unlisted identity: refused. Access by a named role is recorded. |

### 5.8 Backups and restore

| ID | Priority | Requirement | Rationale | Test |
|---|---|---|---|---|
| SEC-041 | MUST | Backups MUST be encrypted at rest and MUST be restricted to the same roles that can access the primary data they copy. | Backups otherwise bypass primary controls. | Read a backup artifact without the restricted key or role: refused. |
| SEC-042 | MUST | A restore MUST be performed as an audited administrative action and MUST run reconciliation (Reconciliation component in [05](./05-system-architecture.md#1-logical-components)) before the restored state serves operations. | Ensures the restored ledger is consistent. | Restore a fixture backup. An audit event exists and reconciliation completes before service resumes. |
| SEC-043 | MUST | If a restore rolls back state, the rolled-back period MUST be recorded, and any posted entries lost in the rollback MUST be recovered through the incident process (section 5.9) without editing history. | Preserves the append-only rule across recovery. | Simulate a rollback. The gap is recorded and recovery uses new entries or a documented process, never edits. |
| SEC-044 | MUST | Backup copies MUST NOT be used as a substitute for retention or deletion rules in [DAT-060](./06-data-ledger-audit.md) and SEC-054. Backup retention MUST be set by OD-09-10. | Prevents indefinite copies of removed data. | Remove a profile field. After the backup retention period, no backup copy holds the field. |
| SEC-045 | SHOULD | Restore procedures SHOULD be exercised on a schedule set by OD-09-10 and the result recorded. | Untested backups are unverified. | The restore exercise record exists for each scheduled interval. |

### 5.9 Incident response

| ID | Priority | Requirement | Rationale | Test |
|---|---|---|---|---|
| SEC-046 | MUST | The incident process MUST define at least these classes: suspected ledger integrity breach; credential compromise; unauthorized cross-network access; administrator misuse; personal data exposure. | Consistent triage. | The runbook lists each class with owner and first action. |
| SEC-047 | MUST | Each incident class MUST have at least one automated detection signal. Reconciliation mismatch and anomalous audit activity are required signals for ledger and administrator classes. | Detection drives response time. | Seed each class in a test environment. Each produces an alert or record. |
| SEC-048 | MUST | The platform MUST provide containment that suspends a credential, a role grant, or posting in one network without deleting data. Each containment action MUST produce an audit event and MUST be reversible by an authorized administrator. | Limits damage while investigating. | Suspend and restore each target. Data is unchanged and audit events exist. |
| SEC-049 | MUST | Each incident MUST have a record with timeline, affected networks and accounts, decisions, and corrective actions. Ledger corrections during an incident MUST use adjustments ([DOM-041](./03-domain-model-rules.md)). | Preserves traceability. | Close a test incident. Record is complete and no entry was edited. |
| SEC-050 | MUST | Network administrators of affected networks MUST be notified within the time bounds set by OD-09-11. Notifications MUST contain no data from other networks. | Lets networks act on their own members' data. | Run an exercise touching two networks. Each administrator receives notice within the bound, and neither notice contains the other network's data. |
| SEC-051 | SHOULD | A post-incident review SHOULD be completed for each high-severity incident and its findings recorded as requirements or decisions where needed. | Improves controls over time. | Review record exists for each high-severity test incident. |

### 5.10 Data minimization

| ID | Priority | Requirement | Rationale | Test |
|---|---|---|---|---|
| SEC-052 | MUST | The platform MUST maintain a data inventory listing every personal-data field, its purpose, its storage location, and its retention rule. | Makes data collection reviewable. | Compare the schema to the inventory. Every personal field is listed. |
| SEC-053 | MUST | Adding a personal-data field MUST require a recorded decision that cites this spec and a minimization review, as required by [DAT-091](./06-data-ledger-audit.md). Fields not used by any MVP operation MUST NOT be stored. | Prevents accumulation. | A schema change adding a field without a decision reference fails review. |
| SEC-054 | MUST | On a member's deletion or profile-removal request, profile fields MUST be removed or made unreadable. Ledger entries, transaction records and audit events MUST remain unchanged, and the account reference MUST remain. The exact handling is OD-09-14. | Reconciles removal with append-only history. | After removal, profile fields return nothing. Entries and audit events are byte-identical. |
| SEC-055 | MUST | Free-text fields (listings, dispute reasons, reports) MUST be length-bounded and MUST be included in the data inventory with their removal handling. | Limits unbounded personal data in free text. | Submit free text above the bound: refused. Each free-text field appears in the inventory. |
| SEC-056 | MUST | Member data MUST NOT be sent to a third party by default. Any sharing with a third party MUST be an explicit network decision recorded in the data inventory, and MUST be governed by OD-09-15. | Limits unintended disclosure. | Scan outbound integrations: none send personal data without a recorded decision. |
| SEC-057 | SHOULD | Operational records SHOULD be retained for the shortest period that supports operations and incident response. Periods are set by OD-09-9. | Operational logs hold incidental data. | Retention job removes records past their period in a fixture. |

### 5.11 Network-level visibility and export controls

| ID | Priority | Requirement | Rationale | Test |
|---|---|---|---|---|
| SEC-058 | MUST | Visibility of balances, entries and listings MUST be enforced on the server for every read path, according to network visibility policy and the default in [DAT-084](./06-data-ledger-audit.md). | Visibility must not depend on the client. | For each read path, a member without permission receives no data. A policy change to narrower visibility takes effect on the next read. |
| SEC-059 | MUST | Network export ([ARC-025](./05-system-architecture.md)) MUST be available only to network administrators of that network. A member export ([ROL-045](./02-personas-roles-journeys.md)) MUST include only that member's records. | Limits bulk exposure. | Request a network export as a member: refused. Request a member export and check its contents: only the requester's records. |
| SEC-060 | MUST | Every export MUST produce an audit event recording requester, time, scope, format version and record counts. Producing an export MUST NOT change state ([ARC-026](./05-system-architecture.md)). | Exports are sensitive and must be attributable. | Produce an export. One audit event exists, and entry, balance and status values are unchanged. |
| SEC-061 | MUST | Exports MUST be delivered only through the authenticated session that requested them. Any downloadable artifact MUST expire within the window in OD-09-12 and MUST NOT be reachable without authentication. | Prevents leaked links. | Request an export link without a session: refused. Use the link after the window: refused. |
| SEC-062 | MUST | Exports MUST NOT contain credentials, session data, second-factor material or recovery material. Exports MUST contain only data of the network named in the request. | Export is a high-value artifact. | Scan an export of a seeded network. No secret is found and no other network's identifier appears. |
| SEC-063 | MUST | Export of audit events MUST include all events for the network in the stated period, including events about administrators and platform operators. | Makes oversight possible. | Export a period containing administrator and platform-operator actions. Each action appears. |
| SEC-064 | MAY | Network administrators MAY enable member-visible aggregate statistics. Aggregates MUST NOT identify individuals and MUST be suppressed for groups smaller than the minimum in OD-09-13. | Shares network-level health without individual exposure. | With a group below the minimum, no aggregate is shown. Above it, the aggregate shows without identities. |

### 5.12 Transport and storage protection

| ID | Priority | Requirement | Rationale | Test |
|---|---|---|---|---|
| SEC-065 | MUST | Traffic crossing any trust boundary in [05 section 3](./05-system-architecture.md#3-trust-boundaries) MUST be encrypted in transit. Unencrypted connections MUST be refused. | Protects credentials and data on the wire. | Attempt an unencrypted connection at each boundary: refused. |
| SEC-066 | MUST | Secrets MUST NOT be stored in source control or in ordinary configuration. They MUST be held in a controlled secret store with access recorded. | Prevents leakage through the repository. | A secret scan of the repository and configuration finds no secrets. |
| SEC-067 | MUST | Network isolation required by [DAT-081](./06-data-ledger-audit.md) MUST be verified by tests that run for every read path and export. | Cross-network leakage is a critical failure. | The test suite includes cross-network reads and exports for each path. All pass. |

## 6. Open decisions

Each decision lists options and a recommended default. The default applies only until a decision is recorded. The default is a recommendation, not a requirement. Items marked **Counsel** need review by qualified counsel before any decision. This spec does not give a legal answer to them.

- **OD-09-1 Second-factor types (SEC-006).** Options: (a) a single type fixed by the platform; (b) a short platform-approved list, with the network choosing one or more; (c) any type the operator configures. **Default: (b).** Privileged roles MUST have at least one second factor from the list.
- **OD-09-2 Session lifetime (SEC-004).** Options: (a) fixed platform values; (b) network-configurable within platform bounds. **Default: (b)**, with platform bounds set in a recorded decision.
- **OD-09-3 Recovery verification method (SEC-013).** Options: (a) registered contact method only; (b) recovery codes issued at enrollment; (c) network-assisted recovery with approvals. **Default: (a) for members and (c) for privileged roles.** Recovery codes are not required in MVP.
- **OD-09-4 Recovery waiting period (SEC-016).** Options: (a) none; (b) fixed platform value; (c) network-configurable. **Default: (b)**, with the length recorded at implementation.
- **OD-09-5 Approvers for privileged recovery (SEC-014).** Options: (a) two network administrators of the same network; (b) one administrator and one dispute reviewer from the same network; (c) platform operator plus one other. **Default: (a).** Option (c) is excluded by SEC-014.
- **OD-09-6 Rate-limit values and tightening (SEC-022, SEC-032).** Options: (a) platform-fixed values; (b) platform defaults that networks may tighten. **Default: (b)**, values recorded at implementation.
- **OD-09-7 Collusion signals (SEC-024).** Options: (a) none in MVP; (b) administrator-visible counts per identity pair; (c) automated flags. **Default: (a).** Option (b) or (c) requires a recorded decision on purpose, retention and member notice. **Counsel** for any notice or data-use question.
- **OD-09-8 Reports about the platform (SEC-027).** Options: (a) report only to network administrators; (b) separate channel to platform operator for platform-level abuse. **Default: (b)**, kept separate from network reporting.
- **OD-09-9 Operational log and record retention (SEC-036, SEC-040, SEC-057).** Options: (a) one platform period; (b) per record class. **Default: (b)**, values recorded in the data inventory. **Counsel** before values are fixed.
- **OD-09-10 Backup retention and restore exercise interval (SEC-044, SEC-045).** Options: (a) fixed platform values; (b) network-configured retention within bounds. **Default: (a)**, with exercise interval recorded in operations. **Counsel** for retention values.
- **OD-09-11 Notification time bounds (SEC-050).** Options: (a) one bound for all incidents; (b) bounds per severity. **Default: (b)**, values recorded at the incident process. **Counsel** for any obligation to notify parties outside the network.
- **OD-09-12 Export link lifetime (SEC-061).** Options: (a) single use within a short window; (b) multiple use within a window. **Default: (a)**, window recorded at implementation.
- **OD-09-13 Minimum group size for aggregates (SEC-064).** Options: (a) fixed platform minimum; (b) network-configurable above a platform minimum. **Default: (a)**, minimum recorded at implementation.
- **OD-09-14 Handling of member removal requests (SEC-054).** Options: (a) remove or make unreadable profile fields and keep the account reference; (b) hold profile fields until a network-defined period ends, then remove. **Default: (a).** **Counsel** before any statement about the effect of removal on the member's rights or obligations.
- **OD-09-15 Third-party data sharing (SEC-056).** Options: (a) none permitted by platform; (b) permitted per network with recorded decision and member notice. **Default: (a).** **Counsel** for any sharing permitted under option (b).
- **OD-09-16 Contact details and public existence (SEC-003, SEC-013).** Options: (a) contact method required for every member; (b) optional. Whether listings reveal member existence. **Default: (b)** for contact method, and listings show only what [02](./02-personas-roles-journeys.md) and [03](./03-domain-model-rules.md) allow. **Counsel** for any question about the visibility of identities.

## 7. Related docs

- [Shared context](_context.md): terms, invariants and rules for authors.
- [02 Personas, roles and core journeys](./02-personas-roles-journeys.md): role rights and permissions matrix.
- [03 Domain model and mutual-credit rules](./03-domain-model-rules.md): ledger rules, disputes and audit requirements.
- [05 System architecture and trust boundaries](./05-system-architecture.md): components, TB-1 to TB-6, export and reconciliation.
- [06 Data, ledger and audit](./06-data-ledger-audit.md): data classes, access, retention and minimization.
- [07 API and integration contract](./07-api-integration-contract.md): authentication expectations, error format, idempotency and rate-related behavior.
- [13 Assumptions, risks and decisions](./13-assumptions-risks-decisions-roadmap.md): risks and decision log for the spec set.
