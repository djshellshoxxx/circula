# Plans

Delivery planning for Circula. Plans are **not normative**: the specifications in [`docs/specs/`](../specs/README.md) and the registers in [doc 13](../specs/13-assumptions-risks-decisions-roadmap.md) win on any conflict. Plans cite requirement IDs; they never redefine them. Numeric effort and date figures are labelled estimates.

| Plan | Purpose | Owner role | Status |
|---|---|---|---|
| [Roadmap](roadmap.md) | Time-boxed phases and effort ranges per team size (estimates), critical path, parallel tracks, release criteria per milestone, go/no-go gates tied to counsel questions (doc 19) and QA gates (doc 12), schedule risks. Links to, does not duplicate, milestones M0 to M9 in doc 13. | Product owner | Draft |
| [MVP implementation plan](mvp-implementation-plan.md) | Epics and stories by workstream with the FR/INV/API/DAT/UX/SEC/OPS IDs they implement, acceptance via QA-E2E scenarios, dependencies, sizes and sequencing; walking skeleton; technology selection process (ADR candidates, no decisions); repository bootstrap checklist; definition of ready/done; working agreements. | Engineering lead | Draft |
| [Master TODO](todo.md) | Checkbox list with owners (roles), priority and dependencies: A) owner decisions, B) counsel engagement, C) spec reconciliation and verification, D) deferred-spec authoring, E) repo bootstrap, F) implementation epics, G) ops/hosting prerequisites. | Product owner | Draft (living) |
| [Traceability](traceability.md) | **Generated** requirement-to-test citation matrix, coverage gaps and undefined IDs. Regenerate with `python3 scripts/gen-traceability.py`; never edit by hand. | QA lead | Generated |

## Reading order

1. [Roadmap](roadmap.md) for the shape and the gates.
2. [Master TODO](todo.md) section A for what needs the owner's attention now.
3. [MVP implementation plan](mvp-implementation-plan.md) for what gets built, in what order.
4. [Traceability](traceability.md) for where the specs and the test plan disagree or are thin.

## Maintenance

- Regenerate traceability and run `python3 scripts/check-specs.py` after each batch of spec changes.
- Plan documents are updated in the same change as the milestone or register change that affects them; epic and story labels (`EP-nn`) are planning labels, not requirement IDs, and must not be cited from specs.
- The ADR process and index are in [`docs/decisions/`](../decisions/README.md).
