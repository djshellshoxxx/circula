# 05 System Architecture and Trust Boundaries

Status: Draft v0.1 · Owner/role: Software architect · Last updated: 2026-10-21
Scope: Logical components, responsibilities, data flow, trust boundaries, deployment options, architectural qualities for the MVP. Technology-neutral at spec level, consistent with DEC-001..DEC-003.
Dependencies: [00-conventions.md](00-conventions.md) (glossary, LED-1..LED-7, DEC-001..DEC-004); [03-domain-model.md](03-domain-model.md) (domain rules); [06-data-ledger-audit.md](06-data-ledger-audit.md) (storage detail); [07-api-contract.md](07-api-contract.md) and [08-ux-accessibility.md](08-ux-accessibility.md) (interfaces); [09-security-privacy.md](09-security-privacy.md) (threats); [11-deployment-operations.md](11-deployment-operations.md) (operational detail).
Open questions: see section 9. Decisions are logged in [13-assumptions-risks-decisions.md](13-assumptions-risks-decisions.md).

This document uses the normative terms and glossary defined in [00-conventions.md](00-conventions.md). It does not repeat ledger invariants; it links to them (LED-1..LED-7) and states how the architecture enforces them.

---





## Build status (as built)

Snapshot: HEAD `925f637` (branch claude/circula-build). Statuses were first read at `31b1015` and re-checked at `925f637` (see TRACEABILITY.md). **built** = behaviour exists in code; **partial** = exists with the gaps stated, or the required verification was not run; **not built**; **deferred** = out of MVP by this spec or a recorded decision. Code paths and test names: [TRACEABILITY.md](../TRACEABILITY.md). Open owner decisions and resolved questions: [DECISIONS.md](../DECISIONS.md). Status is implementation status, not a compliance claim; no release gate has been run.

Summary: built 11, deferred 1, not built 2, partial 11.

| ID | Priority | Status | Note |
|---|---|---|---|
| ARC-001 | P0 | built | One executable per target; no edition split |
| ARC-002 | P0 | partial | Service is the write path; cmd reads SQLite directly for some checks |
| ARC-003 | P0 | built | Adapters map and render; rules in service |
| ARC-004 | P0 | partial | Store interface exists but lives beside SQLite; no contract suite for another store |
| ARC-005 | P0 | built | Only postTx writes entries and balances |
| ARC-006 | P0 | built | Per-account serialisation via write gate and BEGIN IMMEDIATE |
| ARC-007 | P0 | built | Network scoping in queries and tests |
| ARC-008 | P0 | partial | No operator API route; host-level CLI restore and SQL access remain outside service |
| ARC-009 | P1 | not built | Break-glass access not built |
| ARC-010 | P0 | partial | Input limits and tests exist; trust-boundary checklist and fuzzing not run |
| ARC-011 | P0 | built | One writer per database file; lock refuses second holder |
| ARC-012 | P1 | built | Expiry and purge idempotent; export artefacts not resumable |
| ARC-013 | P0 | built | Same artefact for self-hosted and multi-network use; hosted launch not done |
| ARC-014 | P2 | built | Single process constraint documented and enforced by lock (P2 item) |
| ARC-015 | P0 | built | Verify on demand and in hourly job; scheduled runs recorded in consistency_runs, audited only on failure or state change |
| ARC-016 | P1 | partial | Online backup and restore built; RPO and RTO not recorded (DEC-016); shutdown grace default 10 s (shutdown-grace) |
| ARC-017 | P0 | partial | Semantic server-rendered HTML; no automated WCAG tool in CI |
| ARC-018 | P1 | built | Layering enforced by depguard rules in .golangci.yml |
| ARC-019 | P0 | built | CI builds all five targets from one commit |
| ARC-020 | P0 | partial | Export through service; documented reference parser absent |
| ARC-021 | P1 | built | Clock and IDs injected into service |
| ARC-022 | P1 | partial | Structured logs, request IDs, metrics; posting counters not present |
| ARC-023 | P1 | built | Config from file, env and flags; secret files; gitleaks scan in CI |
| ARC-024 | P1 | built | Licence manifest docs/legal/DEPENDENCIES.md; CI fails on a licence outside the approved list |
| ARC-025 | P2 | deferred | Federation, mobile, payment and tokenisation not built; no route |

## 1. Architectural summary

Circula is one server program, built from one codebase, that can run as:

- a **self-hosted** instance serving one community network, operated by that community; or
- a **hosted** instance serving many independent networks, operated by a platform operator.

Both modes use the same binary (DEC-001). Each network is a tenant boundary. Ledger rules live in one domain service layer. Two transport adapters (server-rendered HTML and the versioned JSON API, DEC-002) call that layer and contain no business rules of their own.

```
 Browser (HTML, progressive enhancement)     API client (/api/v1)
            |                                      |
   [TB1 HTTP edge: TLS, sessions, CSRF, rate limits, request ids]
            |                                      |
     HTML adapter                           JSON API adapter
            \                                      /
             +------ [TB2 domain service layer] ----+
                 membership · listings · transactions/ledger
                 disputes · governance/admin · audit · export
                                  |
                    [TB3 storage interface (repositories)]
                                  |
                    SQLite (default)  |  other store (future, P2)
                                  |
                    [TB4 host filesystem / backups / config]

 Background worker (expiry, export jobs, consistency checks) uses TB2 like any caller.
 Operator tooling (CLI/config on the host) crosses TB4 and TB5 (operator plane).
```

## 2. Logical components

| Component | Responsibility | May call | Must not |
|---|---|---|---|
| HTTP edge | TLS termination (or deployment-provided TLS), routing, request ids, body limits, rate limiting, security headers, session and token resolution | HTML adapter, API adapter | Contain business rules |
| HTML adapter | Render server-side pages, parse form input, map results to pages and error messages | Domain services | Write to storage directly |
| JSON API adapter | Map requests to domain commands, map results and errors to the contract in 07 | Domain services | Write to storage directly |
| Identity and access | Accounts for people, sessions, personal API tokens, role resolution per network | Storage (identity repository) | Read ledger data to make access decisions beyond role/membership |
| Membership | Join modes, invitations, approvals, suspension, departure | Storage, Audit, Ledger (for settlement on departure) | Post entries outside the Ledger |
| Listings and directory | Offers and requests, member directory visibility | Storage, Audit | Expose data beyond network visibility rules |
| Transactions and ledger | Proposal, acceptance, decline, cancellation, expiry, posting, reversal, adjustment; balance computation and checks | Storage, Audit | Be bypassed by any adapter or job |
| Disputes | Case creation, review, decisions, linkage to transactions | Ledger (to request reversal/adjustment), Audit | Edit original transactions |
| Governance and admin | Policy configuration, policy change records, announcements | Storage, Audit | Change ledger rows |
| Audit | Append-only event recording in the same atomic unit as each state change (LED-6) | Storage | Be disabled by configuration |
| Export | Member and network exports, closure exports | Storage (read-only snapshot) | Write state |
| Background jobs | Expiry sweeps, export generation, scheduled consistency checks | Domain services | Hold state that is not in storage |
| Storage | Persistence, transactions, locks, migrations | Host filesystem | Contain business rules |

## 3. Data flow: trade proposal to completion

1. A member submits a proposal through the HTML adapter or the API (TB1). The edge assigns a request id and checks the session or token, CSRF (for cookie-authenticated unsafe methods), body size and rate limits.
2. The adapter maps the input to a domain command and passes it to the Transactions service (TB2). Input validation that is part of the contract runs here; rule validation (membership status, amount limits, listing ownership) runs in the domain.
3. The Transactions service opens an atomic storage unit (TB3) that: reserves the idempotency key (06), checks the negative-balance and policy rules against the current account state, writes the `proposed` transaction and its audit event, and commits.
4. Acceptance follows the same pattern. The posting unit writes both ledger entries, updates the cached balances, sets the status to `completed`, writes the audit event and status history, and commits, or it writes nothing (LED-4). The `accepted` status is a state within this unit in the MVP (see 06 and open question OQ-ARC-04).
5. The response is returned to the adapter, which renders HTML or JSON. Reads (balances, history) come from storage through the same domain layer; they never bypass it.

A failure at any step before commit leaves no partial ledger state. A failure after commit but before the response is delivered is recovered by an idempotent retry (LED-4, API section of 07).

## 4. Trust boundaries

| ID | Boundary | Trusted side | Untrusted input | Controls required |
|---|---|---|---|---|
| TB1 | Internet or LAN to HTTP edge | Edge | Browsers, API clients, bots | ARC-010 controls, rate limiting, TLS in transit |
| TB2 | Adapters to domain services | Domain | Adapter-supplied values | Commands carry an authenticated actor and network id; domain re-checks authorization |
| TB3 | Domain to storage | Storage interface | None directly; storage is trusted but must enforce constraints | Constraints in schema (06), atomic units, no raw SQL from adapters |
| TB4 | Host to filesystem, backups, configuration | Host administrator | Configuration files and environment on disk | Secrets handling (11), file permissions, backup encryption (09) |
| TB5 | Platform operator to network data (hosted) | Platform operator, within its role | Operator actions | Operator plane limits and audit (ARC-008, ARC-009) |
| TB6 | Member to other networks | Each network | Members of other networks | Tenant isolation: no cross-network reads or writes (ARC-007) |

## 5. Requirements

### ARC-001 · Single binary for both deployment modes · P0
**Requirement.** The MVP MUST be built as one server executable (per target in DEC-003) that supports self-hosted and hosted operation through configuration. MUST NOT ship separate server editions with divergent domain code.
**Rationale.** DEC-001 requires one artifact. Divergent editions would split the ledger rules that LED-1..LED-7 depend on.
**Acceptance.**
- A single build pipeline produces one server executable per target in DEC-003.
- The same test suite for the domain layer passes against both a single-network and a multi-network configuration.
- No build tag, package, or configuration flag selects a different ledger implementation.

### ARC-002 · Domain service layer is the only write path · P0
**Requirement.** Every state change (LED-6) MUST be executed through a domain service command. The HTML adapter, JSON API adapter, background jobs and operator tooling MUST NOT write to storage except through domain services or the storage migration tool.
**Rationale.** DEC-002 requires both interfaces to share a service layer. Single-path writes keep LED-1..LED-7 enforceable in one place.
**Acceptance.**
- Static checks (package import rules in CI) fail if an adapter package imports a storage implementation package directly.
- For each MVP operation in 07, an integration test runs the operation through the HTML route and through the API route and asserts identical ledger and audit outcomes.

### ARC-003 · Adapters contain no business rules · P0
**Requirement.** Adapters MUST only parse, authenticate at the edge, map, and render. Rules such as limit checks, status transitions and role decisions MUST be in the domain layer.
**Rationale.** Prevents drift between UI and API, which 07 (section 1) and 08 (section 3) both depend on.
**Acceptance.**
- Code review checklist item and a unit test per rule show the rule is reachable only through a domain command.
- Removing an adapter does not change the outcome of any domain test.

### ARC-004 · Storage behind a repository interface · P0
**Requirement.** Domain services MUST access persistence only through repository interfaces defined in the domain-facing package. The MVP default implementation is an embedded SQLite database in WAL mode, using a pure-Go driver with no cgo (DEC-001, DEC-003). No domain type MAY depend on SQLite-specific types.
**Rationale.** DEC-001 allows PostgreSQL to be added later. The interface must not leak SQLite behaviour such as single-writer semantics into domain rules.
**Acceptance.**
- The repository interface compiles without importing the SQLite driver.
- A contract test suite for repositories is written against the interface and runs against the SQLite implementation in CI.
- Release builds for all DEC-003 targets succeed with cgo disabled.

### ARC-005 · Ledger component owns posting · P0
**Requirement.** Only the Ledger capability inside the Transactions service MUST create ledger entries and update cached balances. Entries and cached balances MUST be written in the same atomic unit as the status change that caused them and the audit event (LED-4, LED-6).
**Rationale.** Keeps LED-1, LED-3, LED-5 in one implementation.
**Acceptance.**
- Schema-level or repository-level tests show that no code path writes an entry without a transaction id and a status change.
- Fault injection between the entry write and the audit write leaves the database with neither (rollback verified).

### ARC-006 · Negative-balance check is serialised per account · P0
**Requirement.** Any posting that debits a non-system account MUST check the resulting balance against the account's credit limit (LED-5) inside the same atomic unit, and concurrent postings that touch the same account MUST be serialised so that two postings cannot both pass a check that only one can satisfy.
**Rationale.** Without serialisation, two concurrent trades can each pass the limit and jointly exceed it.
**Acceptance.**
- Concurrency test: N concurrent proposals/acceptances against one account with a limit that allows only k of them complete exactly k, for at least N = 50 and several random interleavings, with no balance below the negative limit.
- The mechanism is documented: under SQLite the single-writer transaction provides serialisation; under any future store, a row lock on the account is required (see 06 section 5.1, DAT-015).

### ARC-007 · Network tenant isolation · P0
**Requirement.** Every domain operation, repository query, cache key, log line, export and background job MUST be scoped to exactly one network id. No operation MAY read or write data of another network, including operations performed by platform operators on behalf of a network without an explicit, audited access path (ARC-009).
**Rationale.** Networks are independent tenant boundaries (glossary). Cross-network leakage is a critical privacy failure (09).
**Acceptance.**
- Automated test: a member of network A requesting any resource id that belongs to network B receives the same response as for a non-existent id (no existence leak).
- Repository tests fail if a query on a network-scoped table omits the network id predicate (enforced by the repository layer, not left to callers).
- Exports and listings contain no rows from another network in multi-network test fixtures.

### ARC-008 · Platform and network governance are separate · P0
**Requirement.** The platform operator role MUST NOT be able to change ledger entries, network policies, member balances, or dispute outcomes. Platform operator capabilities are limited to: provisioning and deprovisioning networks, software upgrades, backups and restores, availability and health operations, and access-controlled support actions defined in 09 and 11.
**Rationale.** Separates hosting responsibility from community governance (see [10-governance-operations.md](10-governance-operations.md): policy versus software enforcement). Protects members from operator misuse (threat model in 09).
**Acceptance.**
- The operator API surface (if any) has no route that mutates ledger, policy, membership or dispute records; verified by route inventory test.
- Operator actions that touch network data are recorded in the network's audit log with actor type `platform_operator` (06).

### ARC-009 · Operator access to network data is break-glass only · P1
**Requirement.** When a platform operator needs to read network data for support, the access MUST require a documented reason, a time limit, and notification to the network administrator (or a recorded reason why notification was not possible). Access MUST be audited. Read-only access MUST be the default.
**Rationale.** Supports support and incident response without standing access to member data.
**Acceptance.**
- A support session cannot be opened without a reason string and an expiry no longer than the configured maximum (default proposed: 4 hours, product decision).
- Each access creates an audit event in the affected network with the operator identity, reason, and expiry.
- Test: an expired support grant cannot read network data.

### ARC-010 · Trust boundaries are enumerated and enforced · P0
**Requirement.** Each trust boundary in section 4 MUST have its controls implemented and tested before release. Boundary crossings MUST validate input type, size, and encoding at the boundary.
**Rationale.** A boundary that is documented but not checked gives false assurance.
**Acceptance.**
- A checklist mapping TB1..TB6 to tests and to the SEC requirements in 09 is maintained and passes in QA (12).
- Malformed input fuzzing at TB1 and TB2 entry points produces no panics and no state changes for rejected input.

### ARC-011 · Concurrency model is explicit · P0
**Requirement.** The architecture MUST define one concurrency model: each posting is one serialisable atomic unit; reads use a consistent snapshot per request; background jobs take a lease per unit of work so that two workers do not process the same expiry or export. Under SQLite the MVP MUST run one write-capable process per database file.
**Rationale.** SQLite provides single-writer serialisation; multi-process writing to one file would require a different design. Expiry and export jobs must be idempotent under restarts.
**Acceptance.**
- Starting a second server process against the same database file while the first runs either fails clearly or enters read-only mode; it MUST NOT corrupt data (tested).
- A job interrupted mid-run resumes or restarts without duplicating effects (tested for expiry and export).

### ARC-012 · Background jobs are idempotent and restart-safe · P1
**Requirement.** Expiry of proposals, export generation, and scheduled consistency checks MUST be idempotent. A job MUST record its progress in storage and MUST be safe to run twice.
**Rationale.** Process restarts and retries are normal in self-hosted environments.
**Acceptance.**
- Running the expiry job twice in succession produces the same state and one audit event per expired transaction.
- Killing the process during an export leaves no partially-written export marked as complete.

### ARC-013 · Deployment modes are both supported in the MVP · P0
**Requirement.** The MVP MUST support (a) a self-hosted single-network deployment and (b) a hosted multi-network deployment from the same artifact. Hosted operation MAY be delayed behind a launch gate, but the architecture MUST NOT require any change to domain code to enable it.
**Rationale.** DEC-001 names both. Hosted subscriptions are out of scope (see 01), but hosting capability is in scope.
**Acceptance.**
- Both modes run the same end-to-end test scenarios (12) with no mode-specific code paths in domain packages.
- Operational steps for each mode are documented in 11.

### ARC-014 · Single-process MVP; horizontal scaling is deferred · P2
**Requirement.** The MVP runs one application process with one writable database. Running several application instances against one shared database MUST require the post-MVP store adapter (ARC-004) and a separate design review.
**Rationale.** SQLite (DEC-001) cannot provide concurrent writers across processes. Scaling is not an MVP goal.
**Acceptance.**
- Documentation and startup checks state the single-process constraint.
- The deferred item is listed in 13 as a roadmap entry.

### ARC-015 · Correctness is verified by invariant checks · P0
**Requirement.** The system MUST provide a consistency check that verifies LED-1 (per-transaction zero sum, per-network zero sum), LED-3 (cached balance equals sum of entries) and LED-2 indicators (no entry without a parent transaction). The check MUST run on demand and on a schedule, and its result MUST be recorded.
**Rationale.** Gives operators evidence that the ledger is intact after restores, migrations and incidents.
**Acceptance.**
- A seeded corruption (one cached balance changed directly in the database) is detected by the check, which reports the network, account and expected/actual values.
- The check passes on a database produced by the full test scenario suite (12).
- Check runs are recorded in audit history.

### ARC-016 · Availability and recoverability targets · P1
**Requirement.** Each deployment MUST have a documented recovery point objective (RPO) and recovery time objective (RTO). The MVP architecture MUST support consistent backups without stopping writes for longer than a configured window, and a restore MUST be verified by ARC-015.
**Rationale.** Availability is a quality attribute. Numeric service levels for the hosted service are product decisions.
**Acceptance.**
- Backup and restore test: restore of a backup taken during write load passes the consistency check (ARC-015).
- RPO and RTO values are recorded in 11 with the owner. Initial placeholders: RPO 24 hours, RTO 4 hours, marked as product decision.
- The graceful shutdown path completes or rolls back in-flight postings within 10 seconds (default; configurable).

### ARC-017 · Accessibility is a build property of the HTML adapter · P0
**Requirement.** The HTML adapter MUST emit semantic HTML that works without JavaScript for core flows (DEC-002). Accessibility requirements in 08 are verified in CI for every page template.
**Rationale.** Accessibility cannot be retrofitted cheaply; progressive enhancement keeps flows usable with assistive technology.
**Acceptance.**
- Core flows in 08 complete with JavaScript disabled.
- Automated accessibility checks run on every template in CI, with zero violations at the configured WCAG 2.2 AA rule set, and manual checks are recorded (08 section 9).

### ARC-018 · Maintainability through enforced module boundaries · P1
**Requirement.** Packages MUST follow the layering in section 2. Import rules MUST be checked in CI. Domain packages MUST NOT import HTTP, HTML, or storage implementation packages.
**Rationale.** Keeps the domain portable (ARC-021) and the code reviewable.
**Acceptance.**
- CI rule fails the build on a forbidden import (tested with a deliberate violation).
- Each domain package has unit tests that run without network, filesystem or database access, except through in-memory test doubles.

### ARC-019 · Portability across release targets · P0
**Requirement.** The server MUST build and run on each target in DEC-003 (windows/amd64, linux/amd64, linux/arm64, darwin/amd64, darwin/arm64). Platform-specific behaviour (paths, file permissions, signals) MUST be isolated in the host adapter.
**Rationale.** DEC-003 is the release target list. Portability is a stated architectural quality.
**Acceptance.**
- CI builds all five targets from a clean checkout.
- The same smoke test (12) passes on linux/amd64 and at least one other target before each release. Each target not tested is listed as unverified in the release notes.

### ARC-020 · Data exportability is an architectural property · P0
**Requirement.** The export capability MUST read through the domain layer (or a read model built on it), MUST NOT depend on the HTML or API adapter, and MUST produce data in documented, open, machine-readable formats (format details in 06 and 07).
**Rationale.** Members and networks must be able to leave with their data (see [10-governance-operations.md](10-governance-operations.md), closure).
**Acceptance.**
- An export produced by the CLI or a background job equals the export produced through the API for the same snapshot.
- The export format is documented and a reference parser reads it without the application.

### ARC-021 · Domain portability · P1
**Requirement.** The domain packages (membership, listings, transactions/ledger, disputes, governance, audit model) MUST NOT depend on the operating system, the clock, random sources, or the HTTP stack except through injected interfaces.
**Rationale.** Makes rules testable and keeps a future store or platform change local.
**Acceptance.**
- Domain tests use a fake clock and fixed id generator; tests are deterministic across 100 repeated runs.

### ARC-022 · Observability without sensitive content · P1
**Requirement.** The system MUST emit structured logs, request ids and basic metrics (request count, latency, error class, posting success/failure counts, consistency-check results). Logs MUST NOT contain passwords, tokens, session identifiers, full email addresses, or ledger amounts tied to a named member (09).
**Rationale.** Operators need signals; logs are a common leak path.
**Acceptance.**
- Log redaction test: a request with a token and a member email produces log lines containing neither value.
- Metrics expose counts without member identifiers as labels.

### ARC-023 · Configuration and secrets · P1
**Requirement.** Configuration MUST come from a file and environment variables with documented precedence. Secrets MUST NOT be stored in the repository or in the network's ledger database in plain text. Full detail is in 11.
**Rationale.** TB4 is a host-administrator boundary; secrets must be separable from data.
**Acceptance.**
- A startup check refuses to run with a placeholder secret value.
- Repository scanning finds no secrets in the default branch (automated check in CI).

### ARC-024 · Dependencies are minimal and reviewed · P1
**Requirement.** Third-party runtime dependencies MUST be listed, pinned, and reviewed for license and maintenance status before adoption. The license of the product itself is undecided (DEC-004), so dependency licenses MUST be recorded to make the decision possible.
**Rationale.** Supply-chain risk and licensing are both product risks.
**Acceptance.**
- A dependency manifest with license per module exists before the first release candidate.
- Build fails on dependencies with licenses not in an approved list maintained by the owner.

### ARC-025 · Extension points are deferred · P2
**Requirement.** Federation between networks, mobile clients, payment conversion, tokenisation, and integrations are out of MVP scope. The architecture MUST NOT preclude them, but MUST NOT implement them. Any future extension MUST change this document before implementation.
**Rationale.** Deferred in [01-vision-scope.md](01-vision-scope.md); the architecture must not hard-code them.
**Acceptance.**
- No code path or route for the deferred capabilities exists in the MVP.
- Deferred items are listed in 13 as roadmap entries with dependencies.

## 6. Architectural qualities

These qualities are satisfied through the requirements above. Each row names the owning requirements.

| Quality | Meaning for Circula | Requirements |
|---|---|---|
| Correctness | Ledger invariants hold across failures, retries, concurrency and restores | ARC-002, ARC-005, ARC-006, ARC-011, ARC-015 |
| Availability | Clean restarts, consistent backups, defined RPO/RTO | ARC-011, ARC-012, ARC-016 |
| Accessibility | Semantic HTML, no-JS core flows, automated and manual checks | ARC-017 and 08 |
| Maintainability | Enforced layers, deterministic domain tests | ARC-002, ARC-003, ARC-018, ARC-021 |
| Portability | Five release targets, no cgo, host-specific code isolated | ARC-004, ARC-019, ARC-021 |
| Data exportability | Open formats, export independent of adapters | ARC-020 and 06, 07 |
| Security isolation | Tenant and operator separation | ARC-007, ARC-008, ARC-009, ARC-010 and 09 |

## 7. Deployment options

| Option | Who runs it | Data location | Status |
|---|---|---|---|
| Self-hosted, one network | Network community or its technical volunteer | Host filesystem chosen by operator | Recommended default (DEC-001, ARC-013) |
| Hosted, many networks | Platform operator | Operator-managed host, one isolated store per network (see OQ-ARC-02) | Supported by architecture; launch is a product decision |
| Centralised federation of networks | Not applicable | Not applicable | Out of scope (ARC-025) |

Detailed environments, upgrades and recovery are in [11-deployment-operations.md](11-deployment-operations.md). This document does not select infrastructure vendors.

## 8. Risks specific to architecture

| ID | Risk | Mitigation | Owner decision |
|---|---|---|---|
| ARC-R1 | SQLite single-writer limits throughput for a large hosted network | Measure in QA (12); PostgreSQL adapter path behind ARC-004 | Scale threshold, product |
| ARC-R2 | Interface leaks SQLite semantics and blocks a later store | Contract tests for repositories (ARC-004) | None |
| ARC-R3 | Hosted operator can access data beyond policy | ARC-008, ARC-009, audit and 09 controls | Access policy wording, product and counsel |
| ARC-R4 | Adapters diverge in behaviour | ARC-002, ARC-003 dual-route tests | None |

Risks in other areas are recorded in 13.

## 9. Open questions

- **OQ-ARC-01 (decision needed):** Confirm that centralised hosting is supported in the MVP launch scope or deferred. Recommended default: architecture supports both; hosted launch gated.
- **OQ-ARC-02 (decision needed):** Storage layout for hosted multi-network: one database file per network (recommended: simpler isolation, export and closure) or one shared database with network id scoping (smaller operational footprint, weaker isolation). Owner: architecture lead with operations.
- **OQ-ARC-03:** Maximum acceptable RTO/RPO for the hosted service. Placeholders in ARC-016 require product approval.
- **OQ-ARC-04 (resolved as built).** Question moved to [DECISIONS.md](../DECISIONS.md): see DEC-036. Owner question on visibility stays open: DEC-074.
- **OQ-ARC-05 (decision needed):** Operator break-glass expiry default (ARC-009 proposes 4 hours).

Cross-references: domain rules [03-domain-model.md](03-domain-model.md); data [06-data-ledger-audit.md](06-data-ledger-audit.md); API [07-api-contract.md](07-api-contract.md); UX [08-ux-accessibility.md](08-ux-accessibility.md); security [09-security-privacy.md](09-security-privacy.md); operations [11-deployment-operations.md](11-deployment-operations.md); test plan [12-quality-test-plan.md](12-quality-test-plan.md).
