# Assumptions, Risks, Decisions, and Roadmap
Status: Draft | Owner: Product architect (role) | Updated: 2026-11-14 | Scope: Assumptions, risks, open decisions, MVP boundary, dependency order, and decision log for the foundational spec set | Deps: [Shared context](_context.md), [Spec index](README.md) | Open questions: 12, see "Open decisions"

## Purpose

This spec records what the foundational spec set assumes, what could go wrong, which choices are still open, what the MVP contains, and the order in which later work MUST proceed. It owns no product rules. Each rule has one home in its own spec; this document links to those homes and does not restate them. Normative terms (MUST, SHOULD, MAY) and glossary terms are defined in the [shared context](_context.md).

Identifiers in this document use the `RSK` prefix. Items are grouped as follows: assumptions (RSK-001 to RSK-010), risks (RSK-011 to RSK-025), open decisions (RSK-026 to RSK-037), and roadmap rules (RSK-038 to RSK-042). Identifiers are never reused, including when an item is retired. A retired item is marked `Retired` and keeps its ID.

## Assumptions

Each assumption is stated as fact for planning. If an assumption is false, the listed impact applies and the owning spec MUST be revised. Each assumption has a verification criterion that a reviewer can check.

| ID | Pri. | Assumption | Impact if false | Verification |
|---|---|---|---|---|
| RSK-001 | MUST | A network is an isolated community. Transactions occur only between accounts in the same network. Cross-network exchange is out of MVP. | Ledger, identity, and authorization design change (see deferred federation work). | The domain and architecture specs state single-network transactions only, with no cross-network reference in MVP requirements. |
| RSK-002 | MUST | Credits are an accounting unit only, as defined in the shared context. Members accept or reject them by network agreement. | Product copy, onboarding, and feature scope may need redesign if the characterization changes. | UX copy review finds no string that states or implies monetary value, redeemability, security status, or cryptocurrency status. |
| RSK-003 | SHOULD | Members for MVP use a modern desktop or mobile browser with an internet connection. | Offline or native-only use cases require the deferred mobile work. | MVP UX acceptance passes at 360 px width with no horizontal scroll and no native-only capability in the required flows. |
| RSK-004 | MUST | Network administrators configure policy but cannot change posted ledger history. History integrity is enforced by the platform, not by administrator trust. | If an administrator can alter history, the audit trail has no independent value. | For every administrator-accessible action, a test confirms no update or delete of a posted entry or audit event is possible. |
| RSK-005 | MUST | The storage layer provides multi-row atomic commit and unique-key constraints. Ledger invariants 1, 5, and 7 depend on this. | Posting can produce partial or duplicate entries. Ledger design must be redone. | Fault-injection test interrupts a posting at each step. Result is all-or-nothing, with no duplicate on retry. |
| RSK-006 | SHOULD | MVP scale is one network with up to a few thousand members and modest transaction volume. Sub-second latency is not required; seconds are acceptable. | Scale-driven architecture (partitioning, caching, queues) becomes necessary earlier. | QA load test at the scale stated in the QA spec meets the latency target set there. Target values are owned by the QA spec. |
| RSK-007 | MUST | Credit limits, join rules, dispute windows, and visibility defaults are data-driven network policy. Changing them requires no code change or redeploy. | Each network needs a fork or a deployment, which blocks multi-network operation. | A policy change made through the administrative interface takes effect for new transactions without a deploy, and is recorded as an audit event. |
| RSK-008 | MUST | Platform does not verify legal identity. Identity assurance is set by each network's join rules. | Platform takes on verification duties and associated obligations. Unresolved legal questions apply (see RSK-028). | The security spec states no identity-verification requirement. Join rules accept a network-defined method. |
| RSK-009 | SHOULD | Dispute reviewers are members designated by a network administrator. They are not platform staff by default. | Dispute handling needs a platform-level staffing model. | Dispute spec defines reviewer designation as network policy. No requirement assumes platform staff. |
| RSK-010 | MUST | Each deployment has one authoritative clock. All ledger and audit timestamps come from that clock, not from client devices. | Dispute windows and ordering of postings become unreliable. | Test with a skewed client clock shows that stored timestamps and window checks use server time only. |

## Risks

Likelihood and impact are qualitative (Low, Medium, High). Each risk has a mitigation the owning spec MUST implement and a test or review that shows the mitigation works.

| ID | Pri. | Risk (L / I) | Mitigation | Verification |
|---|---|---|---|---|
| RSK-011 | MUST | Partial or duplicate postings from a retried or interrupted request (L: Medium / I: High). | Atomic posting (ledger invariant 7). Idempotency key on every posting request (invariant 5). | Retry test: same request sent N times yields one posting. Fault test: no partial entries remain. |
| RSK-012 | MUST | Stored balances diverge from posted entries (L: Low / I: High). | Balances are reconstructable from entries (invariant 6). A reconciliation job compares stored and recomputed balances. | After a randomized transaction sequence, recomputed and stored balances match for 100% of accounts. A seeded mismatch is detected and reported. |
| RSK-013 | MUST | Product copy or network material describes credits as money, a security, or a currency, creating regulatory exposure for operators and networks (L: Medium / I: High). | Copy review against the shared context. Disclosure text is an open decision (RSK-028). No legal conclusion is drawn in this set. | Review checklist signed off before each release. Automated string scan finds no forbidden claim terms in member-facing text. |
| RSK-014 | MUST | Duplicate or fake accounts inflate balances (sybil) (L: Medium / I: Medium). | Join rules set by the network. Admin approval and invite-only options (RSK-029). Audit event on every account creation. | Join-rule tests: each configured method blocks an unauthorized join. Audit event exists for each account. |
| RSK-015 | MUST | Collusion between members creates fictitious transactions that inflate balances (L: Medium / I: High). | Credit limit enforced on every posting. Disputes with reviewer decision. Audit trail of all transactions. Anomaly detection is deferred (analytics). | Scenario test: colluding pair cannot exceed credit limit. Test: disputed transaction is flagged until decided. |
| RSK-016 | SHOULD | Dispute process is abused or reviewers are biased (L: Medium / I: Medium). | Dispute window. Reviewer designation by administrator. Reviewer decision recorded with reason in an audit event. Appeal path is an open decision (RSK-030). | Each dispute resolution produces an audit event naming the reviewer, decision, reason, and time. |
| RSK-017 | MUST | Network rules change retroactively, undermining member trust (L: Medium / I: High). | Policy changes apply to new transactions only. Each transaction records the policy version in force. | Test: changing a credit limit does not alter an existing transaction's recorded policy version. |
| RSK-018 | MUST | Balances or histories are visible to members who should not see them (L: Medium / I: High). | Visibility is network policy with a restrictive default (RSK-032). Queries are scoped to the requesting account's role. | For each role, an access test confirms that out-of-scope balances and histories return no data. |
| RSK-019 | SHOULD | Scope expands into payments, monetization, or integrations before the core ledger is proven (L: High / I: High). | MVP boundary and deferred list (see "MVP foundations and deferred work"). Roadmap gates (RSK-039). | MVP build contains no payment, subscription, or external-integration dependency. Gate checklist is signed before any deferred work starts. |
| RSK-020 | MUST | Administrator or member credentials are compromised (L: Medium / I: High). | Authentication, session, and privileged-action controls in the security spec. Privileged actions produce audit events. | Security test suite in the QA spec covers each control listed in the security spec. |
| RSK-021 | SHOULD | Operators misconfigure self-hosted deployments (time source, backups, secrets) (L: High / I: Medium). | Operations checklist and default-safe configuration in the operations spec. | Clean-install check against the checklist passes. Misconfigurations listed in the operations spec are detected at startup or by health check. |
| RSK-022 | MUST | Backup and restore loses history or creates ledger gaps (L: Low / I: High). | Restore procedure defined in the operations spec. Reconciliation (RSK-012) run after restore. | Restore drill: restored ledger has identical entry count and reconciles with zero mismatch. |
| RSK-023 | SHOULD | Schema choices that are hard to reverse are made before ledger invariants are tested (L: Medium / I: High). | Ledger invariant tests are written before feature code that posts entries. Schema changes are additive where possible. | Gate: invariant test suite exists and passes before any feature endpoint that posts entries is merged. |
| RSK-024 | MUST | Accessibility gaps exclude members (L: Medium / I: High). | Accessibility requirements in the UX spec. | Accessibility audit of MVP flows reports no failure of the criteria the UX spec adopts. |
| RSK-025 | SHOULD | Maintainer capacity is limited. Sub-specs and features stall or drift from the core spec (L: High / I: Medium). | Small MVP. One authoritative home per rule. Deferred work starts only at gates. | Spec index lists owner and status for every spec. No spec remains Draft past its gate without a recorded reason. |

## Open decisions

Each open decision lists options and a recommended default. The default applies until a decision is recorded in the decision log below. The default is a recommendation, not a requirement, until recorded. Decisions that depend on law are marked for counsel.

- **RSK-026 Deployment model.** Options: (a) centralized hosted service only; (b) self-hostable software only; (c) both, from one codebase. **Default: (c).** Self-hostable operation is a MUST of the software. A hosted offering is a deployment of the same software and MUST NOT carry behavior that self-hosted deployments lack.
- **RSK-027 Tenancy per deployment.** Options: (a) one network per deployment; (b) many networks per deployment with isolation. **Default: (b) in the data model, (a) in operation for MVP.** Every ledger and member record carries a network scope from the start, so (b) can be enabled later without a schema rewrite.
- **RSK-028 Legal characterization and jurisdiction.** Options: none decided in this set. **Default: no characterization in product material.** Legal and jurisdiction questions are unresolved for counsel. Counsel review MUST complete before any public launch or hosted offering. This spec makes no legal determination.
- **RSK-029 Member admission method.** Options: (a) invite-only; (b) administrator approval; (c) open self-signup. **Default: (b), set per network.** The platform performs no legal identity verification (RSK-008).
- **RSK-030 Dispute appeal path.** Options: (a) reviewer decision is final; (b) one re-review by a different reviewer; (c) panel review. **Default: (b).** Re-review count and reviewer eligibility are network policy.
- **RSK-031 Credit limit default.** Options: (a) network MUST set explicitly; (b) platform default of zero (no negative balance); (c) platform default of a fixed positive value. **Default: (a), with (b) as the effective value until set.**
- **RSK-032 Balance visibility default.** Options: (a) members see only their own balance; (b) members see aggregate network figures; (c) members see counterparty balances. **Default: (a).** Networks MAY widen visibility by policy; widening MUST be recorded as an audit event.
- **RSK-033 Unit representation.** Options: (a) integer minor units; (b) fixed-precision decimal; (c) floating point. **Default: (a).** Floating point MUST NOT be used for balances.
- **RSK-034 Authoritative time.** Options: (a) server clock; (b) client clock; (c) hybrid. **Default: (a),** consistent with RSK-010.
- **RSK-035 MVP client form.** Options: (a) responsive web only; (b) native mobile plus web; (c) web with progressive capabilities. **Default: (a).** Native work is a deferred sub-spec (see below).
- **RSK-036 Retention of ledger and audit records.** Options: (a) indefinite; (b) network-configured with a platform minimum; (c) fixed platform period. **Default: (a) for ledger and audit events.** Retention of other personal data is unresolved for counsel.
- **RSK-037 Decision record process.** Options: (a) decision log in this spec only; (b) one decision record per decision in a repository folder; (c) both, with this log as index. **Default: (c).** If the repository already defines a decision-record convention, that convention MUST be used and this log MUST link to it.

## MVP foundations and deferred work

MVP contains the minimum needed to run one network with a correct ledger. Each foundation is owned by a spec listed in the spec index.

**MVP foundations:**
- Accounts, offers, and requests for a single network.
- Transactions with a status lifecycle. Completion posts balanced, atomic entries.
- Adjustments and reversals as new traceable entries.
- Basic disputes with reviewer decision and audit event.
- Audit events for security- and accounting-relevant actions.
- Roles: member, dispute reviewer, network administrator, platform operator.
- Network policy: credit limit, join method, dispute window, visibility.
- Responsive web interface meeting the UX accessibility requirements.
- Security baseline: authentication, sessions, privileged-action controls.
- Operations baseline: deployment checklist, backup, restore, reconciliation.
- Ledger invariant test suite, written first.

**Deferred sub-specifications.** Each item below MUST NOT begin until its gate (RSK-039) is met. None is in MVP scope.

| Deferred item | Reason deferred | Precondition to start |
|---|---|---|
| Mobile native clients | Web covers MVP (RSK-035). | MVP passes its acceptance gate. Native client requirements are approved. |
| Federation (cross-network exchange) | Requires identity and ledger design across networks. | RSK-027 resolved to multi-network. Ledger reconciliation across networks specified. |
| Tokenization | Representation of credits outside the ledger raises characterization questions. | RSK-028 resolved by counsel. Ledger remains sole source of balances. |
| Payment conversion | Converting between credits and national currency is outside the software's stated scope (shared context). | Counsel review complete. Separate governance decision recorded. |
| Reputation | Derived scores need defined inputs and fairness rules. | Dispute and transaction data from MVP operating for a defined period. |
| Analytics | Needs stable event model and privacy review. | Audit and event model stable. Privacy review complete. RSK-036 resolved. |
| Integrations | External systems need interface stability. | Public interface spec stable. Authentication for integrations specified. |
| Paid monetization (subscriptions, paid administration) | Business commitment, not a software requirement. | Counsel and business review complete. Separate governance decision recorded. |

## Dependency order

Later work MUST follow this order. Steps with the same number MAY run in parallel.

1. Shared context and conventions (done; see [shared context](_context.md)).
2. Domain rules and ledger invariants. Input to all later steps.
3. Data model for accounts, entries, transactions, policies, and audit events. Depends on step 2.
4. Architecture, interfaces, and security. Depend on step 3.
5. Governance, user experience, and quality plan. Depend on steps 2 and 3. Quality plan defines the invariant test suite (RSK-023).
6. Operations. Depends on steps 3 and 4.
7. Implementation of MVP foundations, ledger invariant tests first.
8. Deferred sub-specifications, one at a time, each at its gate.

## Requirements for roadmap control

- **RSK-038** (MUST) Every spec MUST list its open decisions in its "Open decisions" section. A decision listed in no spec and no log is not decided.
  Verify: link check and grep for open-question counts in every spec header matches the section contents.
- **RSK-039** (MUST) A gate for deferred work is met only when the listed precondition for that item is recorded as complete in the decision log, with the date and approver.
  Verify: each deferred item in the table above has a log entry before its spec leaves Draft.
- **RSK-040** (MUST) Implementation of posting logic MUST NOT merge before the ledger invariant test suite (RSK-023) exists and passes.
  Verify: continuous integration blocks merge of posting code when the suite is absent or failing.
- **RSK-041** (SHOULD) Scope changes to MVP foundations SHOULD be recorded in the decision log and reflected in this spec's MVP list within the same change.
  Verify: change review finds a log entry for each MVP scope change.
- **RSK-042** (MUST) Assumptions RSK-001 to RSK-010 MUST be re-checked at each gate. A failed check MUST be recorded as a decision and the owning spec MUST be revised before the gate passes.
  Verify: each gate record names the assumptions checked and their result.

## Decision log

New decisions are appended. Entries are never deleted. A superseded decision is marked `Superseded by RSK-NNN` and keeps its entry. Use the next unused RSK identifier, which is RSK-043 unless a later revision of this spec says otherwise.

| ID | Date | Decision | Options considered | Rationale | Owner (role) | Status | Links |
|---|---|---|---|---|---|---|---|
| RSK-XXX | YYYY-MM-DD | One-sentence statement of what was decided. | Option list. | Why this option was chosen. | Role that decided. | Proposed / Accepted / Superseded / Retired | Relative links to affected specs or records. |

Rules for entries:
- An open decision moves to the log when accepted. Its entry in "Open decisions" is annotated with the log ID and date.
- An entry MUST name the affected requirement IDs in its Links column.
- Dates are taken from the environment, not estimated.

## Related docs

- [Shared context and glossary](_context.md)
- [Spec index](README.md)
- Other foundational specs are linked from the spec index by their prefixes (PRD, ROL, DOM, FR, ARC, DAT, API, UX, SEC, GOV, OPS, QA). This spec refers to them by prefix and does not copy their requirements.
