# UX and accessibility requirements

**Status:** Draft  
**Owner:** Circula project owner  
**Last updated:** 2026-10-20  
**Scope:** Information architecture and accessible MVP interaction requirements  
**Dependencies:** [Roles and journeys](02-roles-and-journeys.md), [domain model](03-domain-model.md)  
**Open questions:** Language support, target accessibility conformance level, and visual design system.

## Information architecture

MVP navigation should provide: network home and policy, member directory, offers and requests, propose exchange, account balance/history, disputes, and administration for authorized roles. The interface MUST make the current network visible when a user belongs to more than one network.

## Balance comprehension

Every balance view MUST state the network’s unit label and explain the sign convention: positive is net value provided; negative is net value received. Show current posted balance, applicable negative-balance limit, remaining available credit, and pending transactions in separate labeled areas. A pending proposal MUST NOT be included in posted balance. Show amount direction and both counterparties before a user confirms a trade. Reversals and adjustments MUST link to their source event.

## UX requirements

- UX-001: Core workflows MUST work at narrow and wide viewport sizes without loss of actions or transaction meaning.
- UX-002: All actions MUST be operable by keyboard with visible focus and a logical focus order.
- UX-003: Form controls MUST have programmatic labels, instructions, and field-level validation messages.
- UX-004: Color MUST NOT be the only way to communicate balance sign, status, warning, or success.
- UX-005: Confirmation screens MUST summarize amount, both parties, resulting balance direction, and whether the change is pending or final.
- UX-006: Errors MUST identify whether the action committed and give a safe next step.
- UX-007: Tables and history MUST expose meaningful headers and reading order to assistive technology.
- UX-008: Users MUST be able to find the applicable policy and unit explanation before accepting a transaction.

Adopt a recognized accessibility conformance target before public launch; WCAG level and test methodology remain an open decision. Accessibility testing MUST include automated checks and keyboard/screen-reader review of balance and transaction journeys.
