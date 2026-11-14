# Circula Specs: Shared Context

Status: Draft | Owner: Product architect (role) | Updated: 2026-11-14 | Scope: shared terms for all specs | Deps: none | Open questions: none

Single source of shared terms. Other specs MUST link here, not redefine.

## Normative terms
- **MUST** / **MUST NOT**: absolute requirement. Violation = spec non-conformance.
- **SHOULD** / **SHOULD NOT**: strong default. Deviation needs recorded reason.
- **MAY**: optional, permitted.

## ID prefixes (format `PREFIX-NNN`, unique repo-wide, never reused)
PRD product vision | ROL roles/journeys | DOM domain rules | FR functional req | ARC architecture | DAT data/ledger/audit | API interface | UX user experience/a11y | SEC security/privacy | GOV governance | OPS deploy/operations | QA quality/test | RSK risks/assumptions/decisions

## Glossary
- **Network**: one community running Circula with its own members and rules.
- **Account**: a member's ledger holder inside one network. Holds balance.
- **Balance**: signed sum of an account's posted entries. Positive = provided more than received.
- **Credit limit**: network-configured floor on balance (max negative). Platform enforces; network sets value.
- **Offer**: member listing of goods/services they can provide.
- **Request**: member listing of goods/services they want.
- **Transaction**: agreed exchange between two accounts. Has a status. Completion posts balanced entries.
- **Dispute**: formal challenge to a transaction, raised by a participant, reviewed by a dispute reviewer.
- **Adjustment**: new, traceable ledger entry that corrects or reverses earlier entries. Never edits history.
- **Audit event**: append-only record of a security- or accounting-relevant action (who, what, when, why, ref).
- **Platform operator**: runs the software. **Network administrator**: governs one network. **Dispute reviewer**: resolves disputes. **Member**: participates.

## Credit semantics
Credit is an **accounting unit only**. Circula MUST NOT state or imply that credits have monetary value, are redeemable for national currency, are a security, or are a cryptocurrency. Acceptance by members is a network matter, not guaranteed by software.

## Ledger invariants (platform-wide, not configurable)
1. **Balanced**: every completed transaction posts equal and opposite entries; network sum of balances is zero.
2. **Append-only**: posted entries are never updated or deleted.
3. **No silent mutation**: balances change only via posted entries, each linked to an audit event.
4. **Traceable correction**: reversals/adjustments are new entries referencing the original.
5. **Idempotent**: a retried posting request yields one posting, not two.
6. **Reconstructable**: any balance can be recomputed from entries alone.
7. **Atomic**: both sides post together or neither does.

## Configurable (network policy) vs invariant
Credit limits, join rules, dispute windows, visibility defaults = network policy. Section above = platform invariant.

## Rules for spec authors
- Specs only; no code, no vendor picks, no legal/compliance claims. Legal questions -> "Open decisions".
- Each requirement: ID, priority (MUST/SHOULD/MAY), rationale, testable criterion.
- One authoritative home per rule; link, do not copy.
- Unknowns -> "Open decisions" with options + recommended default.

## Spec header template (copy to top of each spec)
```
# <Title>
Status: Draft | Owner: <role> | Updated: 2026-11-14 | Scope: <1 line> | Deps: <links> | Open questions: <count, see "Open decisions">
```
Required sections after header: Purpose; Requirements/Content (with IDs); Open decisions (option list + default); Related docs (relative links).
