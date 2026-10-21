# Governance and network operations

**Status:** Draft  
**Owner:** Circula project owner  
**Last updated:** 2026-10-20  
**Scope:** Network-level policy management and lifecycle  
**Dependencies:** [Domain model](03-domain-model.md), [roles](02-roles-and-journeys.md), [security](09-security-privacy.md)  
**Open questions:** Default notice/appeal timelines and treatment of open disputes at closure.

## Governance boundary

Circula enforces recorded network policy and platform-wide ledger invariants; it does not decide what a community’s rules should be. A network administrator’s authority is delegated by that network and MUST NOT be confused with platform operator access.

## Governance requirements

- GOV-001: Each network MUST expose its current participation, credit-limit, transaction, dispute, and visibility policies to members in understandable form.
- GOV-002: Policy changes MUST be permission-checked, versioned, attributable, timestamped, and announced to affected members before or when effective according to network policy.
- GOV-003: The system SHOULD support a member challenge or appeal path for membership, transaction, and dispute decisions; the chosen process and responsible role must be documented by the network.
- GOV-004: The system MUST record administrator succession and role changes. A network MUST retain a viable recovery path if an administrator becomes unavailable.
- GOV-005: Member suspension or departure MUST block new actions as configured while preserving required transaction and audit history.
- GOV-006: Negative-balance recovery, write-off, service suspension, and closure actions MUST be explicit policy-governed decisions with traceable records.
- GOV-007: Network closure MUST allow authorized export and provide a documented treatment for unresolved disputes and outstanding balances.

## Operational lifecycle

Onboarding communicates the credit unit, participation rules, privacy visibility, transaction acceptance, and dispute path before a member’s first trade. Offboarding disables future access in accordance with policy while preserving accounting references and records required for reconciliation. A transfer of network administration requires an auditable role change and secure account recovery. Service suspension must communicate which user actions remain unavailable and how to access records or request support.

Recommended default: require a second authorized administrator or a documented recovery contact for each active network. This is a recommendation pending pilot and product-owner approval, not an MVP invariant.
