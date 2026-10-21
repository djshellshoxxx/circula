# 0001. Record architecture decisions

| Field | Value |
|---|---|
| Status | Accepted |
| Date | 2026-10-21 |
| Decision owner (role) | Product architect |
| Deciders | Product owner, Product architect |
| Consulted | Engineering lead, Security lead, Operations lead |
| Register entries | [DEC-002](../specs/13-assumptions-risks-decisions-roadmap.md) (technology-neutral specs; stack chosen later via ADR); every `DEC-` entry in the [register](../specs/13-assumptions-risks-decisions-roadmap.md) that is moved from *Proposed default* to *Accepted* |
| Affected specs | [00 conventions](../specs/00-conventions.md) (section 1, SHOULD deviations need a decision log entry), [05](../specs/05-architecture-and-trust-boundaries.md) (`ARC-035`), [13](../specs/13-assumptions-risks-decisions-roadmap.md) |
| Milestone | M0 (decisions and counsel) |

## Context and problem statement

Circula is specified technology-neutrally (baseline default D-02). Many choices (stack, datastore, hosting topology, identifier format, retention periods) are recorded as *Proposed default* entries in the decision register. Without a durable, reviewable record of why each was settled, later contributors cannot tell a deliberate constraint from an accident, and spec changes drift from the register.

## Decision drivers

- Specs say deviations from SHOULD need a recorded justification (`00` section 1) and architectural changes to sections 4 to 8 of doc 05 MUST be recorded as ADRs (`ARC-035`).
- The register in doc 13 must stay the single authoritative index of decisions, with other documents deferring to it.
- Low ceremony: volunteers and small teams must be able to write one in minutes.
- Decisions must be traceable to spec IDs and to the milestone that needs them.

## Considered options

1. Record decisions only in the register (doc 13), with no per-decision documents.
2. Use Architecture Decision Records (MADR-style, one file per decision in `docs/decisions/`), with doc 13 as the index and status authority.
3. Use an external tracker or wiki.

## Decision outcome

Chosen option: "Option 2", because it keeps the rationale next to the code and specs, is reviewable through the normal change process, survives tool changes, and lets the register stay compact.

Rules:

1. **Register is the index.** Every decision has a `DEC-nnn` entry in [doc 13](../specs/13-assumptions-risks-decisions-roadmap.md). An ADR resolves one or more `DEC-` entries and is linked from them; the register status moves from *Proposed default* to *Accepted* (or *Rejected*, *Superseded*) in the same change.
2. **When an ADR is required.** Any decision that: changes a Baseline canon default (`00` section 5.5); alters architecture sections 4 to 8 of doc 05; selects a technology, vendor, licence or hosting topology; sets a security, privacy, retention or availability threshold marked `[DECISION NEEDED]`; or deviates from a SHOULD. Smaller constants may be accepted by editing the register row and citing the approver and date.
3. **Format.** [0000-template.md](0000-template.md) (MADR-style). Files are named `NNNN-kebab-case-title.md`; numbers are never reused.
4. **Immutability.** An Accepted ADR is not rewritten. Changes are made by a new ADR that supersedes it; the old one gets status "Superseded by ADR-NNNN" and nothing else.
5. **Legal matters.** ADRs record that counsel was consulted and cite the `LEG-Q-nn` question in [doc 19](../specs/19-legal-and-regulatory-open-questions.md); they never assert a legal conclusion.
6. **Index.** [README.md](README.md) lists every ADR with status and linked `DEC-` IDs.

### Consequences

- Good: rationale is durable and reviewable; the register and ADRs cross-check each other (the traceability check can flag a `DEC-` marked Accepted with no ADR or approver).
- Bad: small overhead per significant decision.
- Follow-up: the first technical ADRs are expected at M0/M1 (reference stack, datastore, identifier format, licence, topology scope); they are listed in the register as the needed-by items for those milestones.

### Confirmation

Reviewers check, for any change that edits a Baseline default or doc 05 sections 4 to 8, that an ADR exists and the register row is updated. The traceability check covers dangling `DEC-` references.

## More information

- MADR: <https://adr.github.io/madr/> (accessed 2026-10-21)
- Nygard, "Documenting Architecture Decisions": <https://cognitect.com/blog/2011/11/15/documenting-architecture-decisions> (accessed 2026-10-21)
