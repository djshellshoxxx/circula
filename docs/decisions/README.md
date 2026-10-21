# Architecture and product decision records

This folder holds Architecture Decision Records (ADRs) for Circula, in MADR style.

- **Register (authoritative index):** [docs/specs/13-assumptions-risks-decisions-roadmap.md](../specs/13-assumptions-risks-decisions-roadmap.md). Every decision has a `DEC-nnn` entry there; an ADR resolves one or more of them. Local labels used in other specs (for example `SEC-OD-6`) are aliases listed in the register.
- **Template:** [0000-template.md](0000-template.md)
- **Process:** [0001 Record architecture decisions](0001-record-architecture-decisions.md)

## How to add an ADR

1. Find or create the `DEC-nnn` entry in doc 13. Do not start an ADR for something that is not registered.
2. Copy `0000-template.md` to `NNNN-kebab-case-title.md` using the next free number.
3. Fill every field; delete sections that are truly not applicable and say so.
4. Set Status to `Proposed`; open it for review. On acceptance set `Accepted` with the date.
5. In the same change, update the `DEC-` row status in doc 13 (from *Proposed default* to *Accepted*), add the ADR link, and update the affected specs.
6. Never edit an Accepted ADR except its status line; supersede it with a new ADR.

## Index

| ADR | Title | Status | Date | Register entries |
|---|---|---|---|---|
| [0000](0000-template.md) | Template | n/a | n/a | n/a |
| [0001](0001-record-architecture-decisions.md) | Record architecture decisions | Accepted | 2026-10-21 | DEC-002 |

Expected next ADRs (not yet written; see the roadmap in doc 13): reference technology stack and datastore (M1 entry); open-source licence (M0); supported deployment topologies; identifier format; modular-monolith boundary rules; rendering approach.
