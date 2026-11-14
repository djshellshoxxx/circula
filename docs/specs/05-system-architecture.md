# System Architecture and Trust Boundaries
Status: Draft | Owner: Product architect (role) | Updated: 2026-11-14 | Scope: Logical components, data flow, trust boundaries, deployment models, and architectural quality attributes for the MVP and its successors | Deps: [_context.md](./_context.md), [01](./01-product-vision-scope.md), [02](./02-personas-roles-journeys.md), [03](./03-domain-model-rules.md), [13](./13-assumptions-risks-decisions-roadmap.md) | Open questions: 8, see "Open decisions"

## Purpose

This spec defines the logical architecture that an implementation team builds from. It covers:

- the logical components and what each is responsible for;
- how data moves between components for the primary flows;
- where trust boundaries sit and what crosses them;
- the deployment models the software must support;
- the architectural quality attributes each part is held to.

The spec is technology-neutral. It does not choose languages, frameworks, hosting providers, storage products, or vendors. It does not restate ledger rules (owned by the domain and ledger specs, DOM and DAT prefixes, not yet written) or security controls (SEC prefix, not yet written). It links to them where needed.

Normative terms (MUST, SHOULD, MAY) follow [_context.md](./_context.md).

## 1. Logical components

Each row names one component. A component is a logical unit with a defined public interface. Section 4 makes the boundaries testable.

| Component | Responsibility | Owns | MUST NOT |
|---|---|---|---|
| Ledger | Posts entry sets, enforces ledger invariants 1 to 7 and the credit limit on every posting, serves balances and entry history. | Entries, idempotency records, derived balance read model. | Accept edits or deletes of posted entries. |
| Transaction workflow | Manages transaction status lifecycle. On completion, requests a posting from the Ledger. | Transactions and their status history. | Write balances or entries directly. |
| Accounts and membership | Creates accounts under the network's join method; manages member status. | Accounts, memberships. | Change balances. |
| Listings | Manages offers and requests. | Offers, requests. | Affect balances or transaction status. |
| Disputes | Opens disputes, assigns reviewers, records decisions, flags disputed transactions. | Disputes, reviewer decisions. | Alter posted entries. Corrections go through the Ledger as Adjustments. |
| Policy | Stores network policy as versioned data. Supplies the policy version in force for each new transaction. | Policy versions. | Apply retroactively to existing records. |
| Audit | Append-only store of security- and accounting-relevant events. | Audit events. | Update or delete events. |
| Identity and access | Authenticates principals, manages sessions, evaluates role and scope for each request. | Credentials references, sessions, role assignments. | Grant permissions outside the role model in the roles spec ([02](./02-personas-roles-journeys.md)). |
| Reconciliation | Recomputes balances from entries, compares them to stored values, reports mismatches. | Reconciliation reports. | Correct balances. Corrections are Adjustments. |
| Export | Produces machine-readable exports of network data (see ARC-025 to ARC-027). | Export artifacts (transient). | Change any state. |
| Scheduler | Runs reconciliation, exports, and maintenance jobs under a service identity. | Job records. | Run work outside its defined job list. |
| Network administration | Interfaces for network administrators: policy, membership, reviewer designation. | None directly; acts through the owning components. | Touch ledger history or other networks. |
| Platform administration | Interfaces for platform operators: deployment, software updates, backups, restore, time source, secrets. | None directly. | Read or change network data except through audited actions (ARC-014). |
| Client | Responsive web interface for members and administrators. | Local view state only. | Hold authority over amounts, balances, times, or permissions. |
| Storage | Durable persistence with the properties in ARC-022. | Stored records. | Enforce business rules. |

## 2. Data flow

Primary flows. Each flow passes the boundary checks in section 3 at every crossing.

- **F1 Listing.** Client sends an offer or request. API authenticates and authorizes it (ARC-010), checks network scope (ARC-007), stores the listing, emits an audit event.
- **F2 Transaction completion.** Client requests completion with an idempotency key. Transaction workflow checks the transaction state. It requests a posting from the Ledger. The Ledger evaluates the credit limit under the policy version in force, then commits the entry set and the audit event in one atomic unit (ARC-003). The response returns the posting result. A retry with the same key returns the original result (ARC-004).
- **F3 Correction.** An administrator action or an upheld dispute requests a reversal or an Adjustment. The Ledger writes new entries that reference the originals. History is not edited (ARC-012).
- **F4 Policy change.** A network administrator submits a policy change. Policy stores a new version, effective for transactions created after it takes effect. An audit event is written (ARC-006).
- **F5 Read.** Client requests a balance or history. API applies role and scope checks (ARC-010). The Ledger serves data from entries.
- **F6 Reconciliation.** Scheduler starts a run. Reconciliation reads entries and stored balances, reports mismatches as audit events (ARC-016). It does not write balances.
- **F7 Export.** An authorized requester starts an export. Export reads entries and related records as of one point in time (ARC-026). The export event is audited.
- **F8 Backup and restore.** Platform administration takes a backup covering all records together (ARC-031). After a restore, Reconciliation runs before the network accepts new postings.

## 3. Trust boundaries

| ID | Boundary | Trust assumption | Controls (see requirements) |
|---|---|---|---|
| TB-1 | Client to API | Client is untrusted. Any input may be forged. | ARC-009, ARC-010, ARC-032 |
| TB-2 | API to domain components | API has authenticated the principal. Domain components still check role and scope for privileged operations. | ARC-007, ARC-010 |
| TB-3 | Domain components to Storage | Storage is trusted for atomicity and uniqueness only. It is not trusted to enforce business rules. | ARC-001, ARC-003, ARC-022 |
| TB-4 | Platform plane to network plane | Platform operators can reach infrastructure and storage. Network administrators can reach policy and membership. Neither role alone is trusted to change posted history. | ARC-011, ARC-013, ARC-014, OD-05-8 |
| TB-5 | Scheduler to domain components | Jobs run under a service identity with narrow permissions. | ARC-018 |
| TB-6 | Network to network | No crossing in MVP. | ARC-017 |

Outbound integrations with external services are out of MVP scope (see [13](./13-assumptions-risks-decisions-roadmap.md), RSK-019). When one is added, it becomes a new boundary and requires a new row in this table.

## 4. Requirements

### 4.1 Components, boundaries, and data integrity

- **ARC-001** (MUST) Balance-changing behavior MUST be reachable only through the Ledger component's public interface. No other component MUST write to entry or balance storage.
  - Rationale: Ledger invariants 3 and 7 ([_context.md](./_context.md)).
  - Verify: Access review of all write paths to entry and balance storage. Automated check finds no write path outside the Ledger.
- **ARC-002** (MUST) Stored balances MUST be derived data. Any balance MUST be recomputable from posted entries alone.
  - Rationale: invariant 6; RSK-012.
  - Verify: After a randomized sequence of postings, recomputed and stored balances match for 100% of accounts.
- **ARC-003** (MUST) An entry set and its audit event MUST commit in one atomic unit. If any step fails, neither MUST persist.
  - Rationale: invariants 3 and 7; RSK-005, RSK-011.
  - Verify: Fault injection at each step of a posting. Result is all-or-nothing.
- **ARC-004** (MUST) Each posting request MUST carry an idempotency key unique within its network scope. A repeat with the same key MUST return the original outcome and MUST NOT create a second posting.
  - Rationale: invariant 5.
  - Verify: Send the same request N times. Exactly one posting exists.
- **ARC-005** (MUST) Each transaction MUST record the policy version in force when it was created. A later policy change MUST NOT alter that record.
  - Rationale: RSK-017.
  - Verify: Change a credit limit. Existing transaction's recorded policy version and outcome are unchanged.
- **ARC-006** (MUST) Network policy MUST be data. Policy MUST be changeable through the administration interface without code change or redeploy. Each change MUST produce an audit event and MUST apply only to transactions created after it takes effect.
  - Rationale: RSK-007, RSK-017.
  - Verify: Apply a policy change in a running instance. New transactions use it; prior ones do not; audit event exists.
- **ARC-007** (MUST) Every record (account, listing, transaction, entry, dispute, policy version, audit event) MUST carry a network scope identifier. Every query and command MUST be filtered by scope. A request without a scope MUST be rejected.
  - Rationale: RSK-027, RSK-018.
  - Verify: Request each record type from another scope. Response contains no data. Scope-less request is refused.
- **ARC-008** (MUST) Timestamps used for ordering, dispute windows, or audit MUST come from the deployment's authoritative server clock. Client-supplied times MUST NOT be used for these purposes.
  - Rationale: RSK-010, RSK-034.
  - Verify: Submit requests from a client with a skewed clock. Stored timestamps and window checks use server time.
- **ARC-009** (MUST) The server MUST validate every client-supplied amount, balance, status transition, timestamp, and permission claim. The server MUST reject any transition the workflow does not allow.
  - Rationale: TB-1.
  - Verify: Tampered requests for each field are refused with no state change.
- **ARC-010** (MUST) Every request crossing TB-1 or TB-2 MUST be authenticated and authorized for its role and scope before any domain operation runs.
  - Rationale: RSK-020; security controls are owned by the SEC spec.
  - Verify: For each endpoint, an unauthenticated request and an out-of-role request are refused with no state change.
- **ARC-011** (MUST) No interface, for any role, MUST provide update or delete of a posted entry or an audit event.
  - Rationale: invariant 2; RSK-004.
  - Verify: Enumerate every operation exposed by every role. Test each for update or delete of entries and audit events. None succeeds.
- **ARC-012** (MUST) Corrections to posted history MUST be made only by new Adjustment or reversal entries that reference the original entries.
  - Rationale: invariant 4.
  - Verify: Reverse a transaction. Original entries are unchanged. Reversal entries reference them.
- **ARC-013** (MUST) Platform-plane permissions and network-plane permissions MUST be separately assignable. A network administrator permission MUST NOT imply any platform-plane permission, and a platform-plane permission MUST NOT imply any network-plane permission. One person MAY hold both, through separate grants.
  - Rationale: TB-4.
  - Verify: Role matrix test. Each permission grants only its own plane's operations.
- **ARC-014** (MUST) Every platform-plane action that reads or changes network data, including support access, restore, migration, and key rotation, MUST produce an audit event naming the actor, the scope, and the reason.
  - Rationale: TB-4; RSK-004.
  - Verify: Perform each listed action. An audit event with all three fields exists.
- **ARC-015** (MUST) Network administrators MUST be able to change policy, membership, and reviewer designation within their own network scope. They MUST NOT be able to alter entries, audit events, or data of another network.
  - Rationale: RSK-004, RSK-009.
  - Verify: Attempt each prohibited operation from a network administrator account. All are refused.
- **ARC-016** (MUST) Reconciliation MUST compare recomputed and stored balances on a schedule and after every restore. Each mismatch MUST be reported as an audit event. Reconciliation MUST NOT modify balances or entries.
  - Rationale: RSK-012, RSK-022.
  - Verify: Seed a mismatch. Mismatch is reported. No write occurs in ledger or entry storage during the run.
- **ARC-017** (MUST) MVP MUST NOT contain any data path, interface, or reference that crosses network scope.
  - Rationale: RSK-001.
  - Verify: A transaction naming an account in another network is refused. No cross-scope identifier appears in MVP interface definitions.
- **ARC-018** (MUST) Scheduled and background jobs MUST run under a service identity whose permissions are limited to the job's function. Each job run MUST produce a record.
  - Rationale: least privilege; TB-5.
  - Verify: A job attempting an out-of-function operation is refused. Each run has a record.

### 4.2 Deployment and portability

- **ARC-019** (MUST) The software MUST be installable and operable by a party that is not the platform operator, using only the deployment checklist in the operations spec (OPS prefix, not yet written).
  - Rationale: RSK-026; RSK-021.
  - Verify: Clean-install check performed by a tester with no platform-operator access.
- **ARC-020** (MUST) A hosted deployment, when one exists, MUST run the same release artifacts as self-hosted deployments. It MUST NOT depend on any behavior that a self-hosted deployment lacks.
  - Rationale: RSK-026.
  - Verify: Feature-parity test runs against a hosted and a self-hosted build from the same release. Results match.
- **ARC-021** (MUST) Domain components MUST NOT depend on a specific hosting provider, storage product, or infrastructure service. Each such dependency MUST sit behind a defined adapter boundary.
  - Rationale: portability.
  - Verify: Dependency review. The QA suite passes against a second adapter implementation.
- **ARC-022** (MUST) Storage MUST provide multi-row atomic commit, unique-key constraints, and durable write acknowledgement. The software MUST refuse to start against storage that does not provide these.
  - Rationale: RSK-005.
  - Verify: Startup check against a storage target that lacks each property. Startup is refused.
- **ARC-023** (MUST) Each deployment MUST have one authoritative clock source. The software MUST NOT post entries while it cannot establish server time from that source, and MUST report the condition.
  - Rationale: RSK-010, RSK-034. Source choice is OD-05-6.
  - Verify: Remove or disable the time source. Postings are refused and the condition is reported.
- **ARC-024** (SHOULD) The architecture SHOULD allow a network to move to its own partition, storage, or deployment without schema change, using the scope attribute in ARC-007.
  - Rationale: RSK-027; RSK-006.
  - Verify: Add a second network to a test instance. No schema change is needed.

### 4.3 Data exportability

- **ARC-025** (MUST) The system MUST provide a network administrator with an export of the network's accounts, transactions, entries, audit events, and policy versions. The export MUST be in a documented, machine-readable form that does not require the software to read.
  - Rationale: quality attribute, data exportability.
  - Verify: An independent reader that does not run the software reads the export. Balances recomputed from its entries match the stored balances at export time.
- **ARC-026** (MUST) An export MUST represent one stated point in time. Producing an export MUST NOT change any state.
  - Rationale: reproducibility; ARC-016.
  - Verify: Postings during an export do not appear in it. Entry counts before and after are equal.
- **ARC-027** (MUST) The export format MUST be versioned, its documentation MUST ship with each release, and each release MUST read exports produced by the previous supported version.
  - Rationale: long-term portability of member and network records.
  - Verify: Export from version N-1 is read by version N.

### 4.4 Correctness, availability, and recovery

- **ARC-028** (MUST) The ledger invariant test suite MUST exist and pass before any feature endpoint that posts entries is merged.
  - Rationale: RSK-023, RSK-040.
  - Verify: Continuous integration blocks merge of posting endpoints when the suite is absent or failing.
- **ARC-029** (MUST) Availability targets MUST be defined for each deployment class and MUST be measured. Target values are set under OD-05-5 and owned by the OPS and QA specs.
  - Rationale: availability quality attribute.
  - Verify: Targets appear in the QA spec. Monitoring reports measured values against them.
- **ARC-030** (SHOULD) While posting is unavailable, balance and history reads SHOULD remain available. Unavailability MUST NOT leave partial state.
  - Rationale: availability; ARC-003.
  - Verify: Disable the posting path under fault injection. Reads succeed. No partial entry sets exist.
- **ARC-031** (MUST) Backups MUST cover entries, audit events, policy versions, accounts, and transactions taken together, so every restore point is internally consistent.
  - Rationale: RSK-022.
  - Verify: Restore drill. Entry counts match. Reconciliation (ARC-016) reports zero mismatches.

### 4.5 Accessibility enablement

- **ARC-032** (MUST) Every error response MUST carry a stable, machine-readable code and, where applicable, a reference to the field or operation that failed. Error meaning MUST NOT be conveyed by color or by message text alone.
  - Rationale: lets the client present accessible errors (UX prefix owns the presentation rules).
  - Verify: Each error path returns the code and reference. Automated test confirms the code set is stable across releases.
- **ARC-033** (MUST) Every operation required by the MVP user flows MUST be available through the documented interfaces that the client uses. The client MUST NOT hold logic that decides permissions, amounts, or balances.
  - Rationale: TB-1; ensures accessibility behavior is a client concern and does not require server divergence.
  - Verify: Each MVP user flow completes using the documented interfaces alone.
- **ARC-034** (SHOULD) Long-running operations (exports, reconciliation runs, restores) SHOULD expose status through an interface so the client can present progress accessibly.
  - Rationale: accessibility of long waits.
  - Verify: Each listed operation reports state transitions through its status interface.

### 4.6 Maintainability

- **ARC-035** (MUST) Each component in section 1 MUST have an explicit module boundary and a documented public interface. Components MUST NOT read or write another component's storage directly.
  - Rationale: maintainability; one authoritative home per rule.
  - Verify: Dependency check in continuous integration finds no cross-component storage access.
- **ARC-036** (MUST) Each architectural decision made after this spec is approved MUST be recorded under the decision record convention in [13](./13-assumptions-risks-decisions-roadmap.md) (RSK-037).
  - Rationale: traceability.
  - Verify: Each decision in this spec's Open decisions list, once resolved, has a linked record.
- **ARC-037** (SHOULD) Changes to a component's public interface SHOULD be additive. A breaking change MUST be recorded as a decision before release.
  - Rationale: maintainability; RSK-025.
  - Verify: Interface diff review at each release. Each breaking change has a decision record.

### 4.7 Quality attribute summary

| Attribute | Architectural requirements | Measurable target owned by |
|---|---|---|
| Correctness | ARC-001 to ARC-005, ARC-009, ARC-011, ARC-012, ARC-016, ARC-028 | QA spec (invariant test suite) |
| Availability | ARC-023, ARC-029, ARC-030, ARC-031 | OPS and QA specs (OD-05-5) |
| Accessibility | ARC-032 to ARC-034 | UX spec |
| Maintainability | ARC-021, ARC-035 to ARC-037 | This spec; OPS spec for release process |
| Portability | ARC-019 to ARC-022 | OPS spec |
| Data exportability | ARC-025 to ARC-027 | DAT spec (export format detail) |

## 5. Deployment models

Three deployment models are in scope. Each uses the same software (ARC-020).

| Model | Description | Main trade-offs |
|---|---|---|
| A. Centralized hosted | One operator runs deployments for one or more networks. | Lowest operational burden for networks. Concentrates operator trust and availability risk (TB-4). |
| B. Self-hosted | A network's community, or a party it chooses, runs its own deployment. | Network controls its infrastructure. Operational risk (RSK-021) and availability depend on that party. |
| C. Both | Same software, same release artifacts, two operating modes. | Requires disciplined portability (ARC-019 to ARC-021). Gives networks a choice of trust model. |

**Recommendation: C.** This matches the default in [13](./13-assumptions-risks-decisions-roadmap.md) (RSK-026). For MVP, each deployment runs one network, while the data model is network-scoped from the start (ARC-007). This matches RSK-027 default. Alternatives A-only and B-only are recorded in OD-05-1.

**Separation of platform operation and network governance.** Platform operation (software releases, infrastructure, backups, time source, secrets) and network governance (policy, membership, reviewer designation, dispute outcomes) are separate planes (ARC-013). The separation holds in every deployment model. In self-hosted deployments the same person may hold both planes, but the software still records and separates them (ARC-014).

## Open decisions

Option lists and defaults follow the rules in [_context.md](./_context.md). Open-decision IDs use the `OD-05-N` form.

- **OD-05-1 Initial deployment topology.** Options: (a) centralized hosted only; (b) self-hosted only; (c) both, from one codebase. **Default: (c)**, as recorded in [13](./13-assumptions-risks-decisions-roadmap.md) RSK-026. If RSK-026 is resolved differently, this spec MUST be revised to match before its gate passes (RSK-042).
- **OD-05-2 Process topology.** Options: (a) one deployable unit with enforced internal module boundaries (ARC-035); (b) separately deployed services per component. **Default: (a),** given the scale in RSK-006. Option (b) adds operating burden for self-hosted operators (RSK-021) and is deferred until a measured need exists.
- **OD-05-3 Background work model.** Options: (a) scheduled batch jobs only; (b) queued job system with retries; (c) event-driven processing. **Default: (a)** for reconciliation, export, and maintenance. Posting (ARC-003) stays synchronous in the request path under every option.
- **OD-05-4 Export scope and format detail.** Options: (a) member-level history only (ROL-045 in [02](./02-personas-roles-journeys.md)); (b) network-level export for administrators (ARC-025); (c) both. **Default: (c).** Field-level format belongs to the DAT spec. Member-level export is governed by ROL-045 and is not restated here.
- **OD-05-5 Availability and recovery targets.** Options: (a) per-deployment-class targets set by the operator, within a minimum set by the OPS spec; (b) fixed platform-wide targets; (c) targets set only for hosted deployments. **Default: (a).** This spec sets no numeric values. ARC-029 requires that values exist and are measured.
- **OD-05-6 Clock source for self-hosted deployments.** Options: (a) operator-provided network time, with a startup and continuous synchronization check (ARC-023); (b) any reliable source, with a documented skew tolerance; (c) a platform-supplied time service. **Default: (a),** consistent with RSK-034 (server clock).
- **OD-05-7 Isolation when multi-network is enabled.** Options: (a) shared storage with scope enforced in the data-access layer; (b) separate storage per network; (c) separate deployment per network. **Default: (a)** until the federation and tenancy gate (RSK-027, RSK-039) is met. ARC-007 holds under all three options.
- **OD-05-8 Tamper evidence against storage-level access.** Options: (a) access control and audit events only; (b) chained, verifiable digests over entries and audit events, checkable by any network administrator; (c) option (b) plus periodic digest copies held outside the platform operator's control. **Default: (a) for MVP.** Under (a), software cannot detect alteration made directly in storage by a party with infrastructure access (TB-4). This is a known gap against RSK-004. Option (b) or (c) is recommended before any hosted deployment.

## Related docs

- [_context.md](./_context.md): shared terms, ledger invariants, spec header rules.
- [01 Product vision and scope](./01-product-vision-scope.md): PRD-002 and PRD-003 constrain network policy and invariants.
- [02 Personas, roles, and journeys](./02-personas-roles-journeys.md): role definitions and ROL-045.
- [03 Domain model and rules](./03-domain-model-rules.md): domain rules that the Ledger and Transaction workflow enforce.
- [13 Assumptions, risks, decisions, and roadmap](./13-assumptions-risks-decisions-roadmap.md): RSK-001, RSK-004, RSK-005, RSK-007, RSK-010, RSK-012, RSK-017, RSK-019, RSK-026, RSK-027, RSK-034, RSK-037, RSK-040, RSK-042.
- Not yet written, referenced by prefix only: DOM (domain rules), DAT (ledger, audit, export format), SEC (security, privacy), OPS (deployment, backup, restore), UX (interaction and accessibility), QA (quality and test targets), GOV (governance), API (interfaces).
