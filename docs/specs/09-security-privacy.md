# Security, privacy, and abuse-resistance baseline

**Status:** Draft  
**Owner:** Circula project owner  
**Last updated:** 2026-10-20  
**Scope:** Product security requirements and practical threat baseline  
**Dependencies:** [Architecture](05-architecture.md), [data and ledger](06-data-ledger-audit.md)  
**Open questions:** Identity provider, support access model, retention schedule, and incident notification obligations by jurisdiction.

## Assets and actors

Assets include credentials, member profile data, network policies, private listings, transaction history, disputes/evidence, balances, exports, audit records, and service secrets. Actors include members, applicants, network administrators, dispute reviewers, platform operators, compromised accounts, and external attackers. Threats include account takeover, unauthorized balance mutation, administrator abuse, cross-network disclosure, duplicate/replayed transactions, evidence exposure, denial of service, and misleading transaction status.

## Security requirements

- SEC-001: The system MUST authenticate a principal before exposing non-public network data or accepting mutations.
- SEC-002: Authorization MUST be checked server-side for every operation and every network-scoped resource; deny by default.
- SEC-003: Account recovery MUST verify control through a documented process and MUST NOT silently change balances or membership roles.
- SEC-004: Privileged actions, ledger postings, corrections, role changes, and dispute outcomes MUST be attributable and auditable.
- SEC-005: The system MUST protect against duplicate/replayed mutations using idempotency, and MUST enforce rate limits on authentication, recovery, and mutation paths.
- SEC-006: Logs MUST exclude passwords, tokens, secrets, and unnecessary dispute evidence or personal data.
- SEC-007: Network exports MUST be restricted to authorized roles, scoped, logged, and designed to avoid including unrelated networks.
- SEC-008: Backup access MUST be limited to operators who need it; restore exercises MUST verify confidentiality and ledger reconciliation.
- SEC-009: The product MUST provide a path to report suspected fraud, account compromise, or abuse to the responsible network contact or service operator.
- SEC-010: Incident procedures MUST define triage, containment, preservation of relevant audit records, recovery, and communication ownership before production.

## Privacy baseline

Collect only information required for membership, exchange coordination, authentication, and support. Networks MUST be able to understand which member information is visible to other members, administrators, reviewers, and platform operators. Personal profile visibility and export permissions MUST be explicit. Sensitive evidence MUST be access-restricted. Do not claim anonymity, cryptographic guarantees, or regulatory compliance without evidence.

A jurisdiction-specific review is required before launch for privacy, consumer-protection, tax, financial-regulatory, and money-services questions. This is a decision gate, not a legal conclusion; see [open decisions](13-decisions-risks-roadmap.md).
