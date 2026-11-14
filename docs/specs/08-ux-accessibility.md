# UX and Accessibility Requirements
Status: Draft | Owner: Product architect (role) | Updated: 2026-11-14 | Scope: MVP information architecture, screen inventory, balance and transaction presentation, responsive layout, keyboard operation, validation and error messaging, accessibility baseline | Deps: [_context.md](./_context.md), [02](./02-personas-roles-journeys.md), [03](./03-domain-model-rules.md), [04](./04-functional-requirements.md), [07](./07-api-integration-contract.md) | Open questions: 6, see "Open decisions"

## Purpose
Define the user-facing structure of Circula's MVP and the observable presentation and interaction properties that any screen must have. The requirements here make balance direction, credit availability, and transaction status unambiguous, and set a verifiable accessibility baseline. This spec does not define visual design, copy text, or feature-level interaction flows; those are out of scope for this pass (see Scope).

## Scope and boundaries
- In scope: information architecture; MVP screen inventory (name and purpose only); presentation rules for balances, entries, and transaction status; responsive layout requirements; keyboard operation; validation and error messaging; accessibility baseline and its target level.
- Out of scope: visual design, color palettes, typography, component styling, copy wording, and feature-level interaction specifications. Those belong to later design artifacts that must conform to this spec.
- Rules owned elsewhere are referenced, not restated. Term definitions, credit semantics, and ledger invariants are in [_context.md](./_context.md). Role rights are in [02](./02-personas-roles-journeys.md). Domain rules for balances, credit limits, and transaction states are in [03](./03-domain-model-rules.md). Functional behavior is in [04](./04-functional-requirements.md). Interface error, pagination, and authorization contracts are in [07](./07-api-integration-contract.md).
- Normative terms (MUST, SHOULD, MAY) follow [_context.md](./_context.md).
- This spec does not claim that any product surface currently meets any accessibility conformance level. Conformance is a verification outcome, not a statement of this document.

## Requirements

### Information architecture

**UX-001** (MUST) Member-facing navigation MUST expose these top-level areas: Balance, Transactions, Listings, and Account. Administration MUST be a separate top-level area visible only to roles that hold administrative rights.
- Rationale: groups the MVP journeys in [02](./02-personas-roles-journeys.md) and the functions in [04](./04-functional-requirements.md) into stable locations.
- Test: for each role defined in [02](./02-personas-roles-journeys.md), the navigation shows exactly the areas that role may use, and no others.

**UX-002** (MUST) Navigation visibility MUST reflect the viewer's effective rights in their network, as defined in [ROL-002](./02-personas-roles-journeys.md) and [ROL-003](./02-personas-roles-journeys.md). Hiding a control or area MUST NOT be the only protection; the interface contract in [07](./07-api-integration-contract.md) governs what data is returned.
- Test: a direct request for a screen the viewer may not use yields the same refusal behavior specified in [07](./07-api-integration-contract.md), not the screen's content.

**UX-003** (MUST) A member MUST be able to reach their own balance view, their entry history, and each of their transactions from the top-level navigation without leaving the product.
- Test: from any MVP screen, each of these destinations is reachable within the navigation structure, without external links.

**UX-004** (MUST) Every screen MUST have exactly one page-level heading that states the screen's purpose. The heading text MUST match the label of the navigation item that leads to the screen.
- Test: automated heading audit per screen; manual check of label consistency.

### MVP screen inventory

**UX-010** (MUST) The MVP MUST provide the following member-facing screens. Each screen's purpose is stated here only; behavior is defined by the referenced requirements.

| Screen | Purpose | Governing requirements |
|---|---|---|
| M1 Join entry | Start or resume joining a network through its join rules | [FR-005](./04-functional-requirements.md), [ROL-010](./02-personas-roles-journeys.md) |
| M2 Join status | Show the requester's own pending, denied, or expired join request | [ROL-012](./02-personas-roles-journeys.md), [ROL-013](./02-personas-roles-journeys.md) |
| M3 Balance overview | Show balance, credit limit, and available credit with direction and status | [FR-027](./04-functional-requirements.md), [UX-020](#balance-and-transaction-presentation) to [UX-024](#balance-and-transaction-presentation) |
| M4 Entry history | Show the member's append-only entries | [FR-028](./04-functional-requirements.md), [UX-025](#balance-and-transaction-presentation) |
| M5 Listing directory | Browse and search offers and requests visible to the member | [FR-014](./04-functional-requirements.md), [FR-015](./04-functional-requirements.md) |
| M6 Own listings | Create, edit, pause, withdraw, or close the member's listings | [FR-012](./04-functional-requirements.md) |
| M7 Transaction list | Show the member's transactions and their current statuses | [FR-026](./04-functional-requirements.md) |
| M8 Transaction detail | Show one transaction's parties, status, and available next actions | [FR-026](./04-functional-requirements.md), [ROL-058](./02-personas-roles-journeys.md) |
| M9 Propose transaction | Propose a transaction with a counterpart | [FR-018](./04-functional-requirements.md) |
| M10 Dispute | Raise a dispute on a transaction and view its status and submissions | [FR-035](./04-functional-requirements.md), [FR-038](./04-functional-requirements.md) |
| M11 Account | View the member's own identity, role summary, and standing | [ROL-008](./02-personas-roles-journeys.md), [FR-007](./04-functional-requirements.md) |

**UX-011** (MUST) The administrative area MUST provide screens for these purposes, visible only to roles that hold the corresponding rights: join request decisions; account list with standing; network policy view and change, including change history; role assignment; dispute queue for dispute reviewers; and entry and audit views with export.
- Governing requirements: [FR-010](./04-functional-requirements.md), [FR-031](./04-functional-requirements.md), [FR-032](./04-functional-requirements.md), [FR-045](./04-functional-requirements.md), [ROL-060](./02-personas-roles-journeys.md), [ROL-061](./02-personas-roles-journeys.md).

**UX-012** (MUST) A screen MUST NOT display data that the viewing role may not see. In particular, no member-facing screen MUST show another member's balance or entries, except the counterpart transaction information permitted by [FR-030](./04-functional-requirements.md).

**UX-013** (MUST) Every displayed balance, entry, status, and limit MUST be the value returned by the platform. A screen MUST NOT compute or adjust balances, available credit, or statuses locally.
- Governing requirements: [FR-029](./04-functional-requirements.md), [API-001](./07-api-integration-contract.md).

### Balance and transaction presentation

**UX-020** (MUST) Every balance value MUST display its direction in text. The text MUST state whether a positive value means the member has provided more than received, whether a negative value means the member has received more than provided, and that zero means neither. Direction MUST NOT be conveyed by sign, color, or icon alone.
- Test: for positive, zero, and negative test balances, a screen-reader-only reading of the balance includes the direction statement, and the same statement is visible without color.

**UX-021** (MUST) A negative balance MUST be displayed with an explicit minus sign and a text statement that the account is below zero. The presentation MUST also show the remaining available credit in the same view, so the member can see how much further the account may go negative.
- Test: a negative test balance renders the sign, the below-zero statement, and the available credit figure together.

**UX-022** (MUST) Balance, credit limit, and available credit MUST be shown as three separately labeled figures. They MUST NOT be merged into one figure or one unlabeled sentence. The definitions used for each figure are those in [03](./03-domain-model-rules.md) and [_context.md](./_context.md).
- Test: each figure has its own visible label; no figure appears without a label.

**UX-023** (MUST) The balance figure MUST be labeled as reflecting completed (posted) entries. Pending transactions (proposed or accepted but not completed) MUST NOT be included in the balance figure. Pending transactions MUST be presented in a separately labeled area or list, and the presentation MUST state that they are not yet reflected in the balance.
- Governing requirements: [FR-019](./04-functional-requirements.md) (proposals change no balance) and [FR-022](./04-functional-requirements.md) (posting occurs on completion). Whether any projected figure is shown is an open decision, see OD-08-4.

**UX-024** (MUST) Each transaction status MUST be shown as a text label that names the status. Each status MUST have a distinct label. Status MUST be available to assistive technology as text, not as color or icon only. The set of statuses is defined in [03](./03-domain-model-rules.md).
- Test: each status defined in [03](./03-domain-model-rules.md) renders a unique label; the label is present in the accessibility tree.

**UX-025** (MUST) Each entry in the entry history MUST show: the date and time, the signed amount with an explicit plus or minus sign, the entry type (posting, reversal, or adjustment), and a reference to the originating transaction or dispute. Reversals and adjustments MUST be shown as their own rows that reference the original entry, consistent with the ledger invariants in [_context.md](./_context.md). Entries MUST NOT be presented as edited.
- Test: a reversal row references the original; no UI affordance allows editing a posted row.

**UX-026** (MUST) Amounts MUST be shown with an explicit unit label and sign. Amounts MUST be presented from the integer values returned by the interface, without rounding or reformatting that changes the value. Presentation MUST NOT use wording that states or implies monetary value, as defined in [ROL-007](./02-personas-roles-journeys.md) and [_context.md](./_context.md).

**UX-027** (MUST) Every date and time MUST be displayed with an explicit time zone or with the time zone named once on the same screen. The time zone rule is an open decision, see OD-08-2.

**UX-028** (MUST) When displayed data may be stale, the screen MUST show a text indicator with the time the data was last updated. This applies to balance, entry, and transaction views.
- Governing requirement: [ROL-046](./02-personas-roles-journeys.md).

### Responsive layout

**UX-030** (MUST) Every MVP screen MUST be usable at a viewport width of 320 CSS pixels with no two-dimensional scrolling of the page, except for content that cannot reasonably reflow (such as a data table, which MUST follow UX-031). The minimum supported width is an open decision, see OD-08-5.

**UX-031** (MUST) At narrow widths, no information or function may be removed. Content MUST reflow. Data tables (entry history, transaction list) MUST either present each row as a labeled group of fields or sit in a horizontally scrollable region that is itself keyboard-operable and whose column headers remain associated with their cells.

**UX-032** (MUST) At 200% browser zoom and at text spacing increased as permitted by the target level in OD-08-1, no content or function MUST be lost or overlap.

**UX-033** (MUST) Interactive controls MUST present a target of at least 24 by 24 CSS pixels, or have adequate spacing from adjacent targets, as required by the target level in OD-08-1.

**UX-034** (SHOULD) Screens SHOULD remain usable in both portrait and landscape orientations.

### Keyboard operation

**UX-040** (MUST) Every function available on an MVP screen MUST be operable using a keyboard alone. Keyboard focus MUST NOT become trapped in any component, except within a modal dialog that is open.

**UX-041** (MUST) Focus order MUST follow the reading and logical order of the screen. Every focusable element MUST show a visible focus indicator.

**UX-042** (MUST) Each MVP screen MUST provide a skip link, or an equivalent mechanism, that moves focus to the main content.

**UX-043** (MUST) When a modal dialog opens, focus MUST move into it. While open, focus MUST remain inside it. On close, focus MUST return to the control that opened it. Every dialog MUST be dismissible by the keyboard, and dismissal MUST NOT submit or complete an action.

**UX-044** (MUST) Any single-key or keyboard shortcut MUST NOT be the only way to perform a function. Shortcuts that use a single printable character MUST be turn-off-able or remappable by the member.

**UX-045** (SHOULD) If a session will expire after a period of inactivity, the product SHOULD warn the member before expiry and SHOULD allow the member to extend the session without losing entered data. The timeout behavior is an open decision, see OD-08-6.

### Validation and error messages

**UX-050** (MUST) Each validation error MUST identify the field, state what is wrong, and state how to correct it. The message MUST appear in text adjacent to the field. On submit, a summary of all errors MUST appear, and each summary item MUST move focus to, or link to, its field.

**UX-051** (MUST) Error state MUST NOT be conveyed by color alone. Each error message MUST be programmatically associated with its field so that assistive technology announces it with the field.

**UX-052** (MUST) When the platform refuses an action on a policy or rule basis (for example, an acceptance that would breach a credit limit, as in [FR-021](./04-functional-requirements.md)), the message MUST state the reason category in plain language and the next available action. The message MUST NOT disclose another member's balance, entries, or identity beyond what [FR-030](./04-functional-requirements.md) permits.

**UX-053** (MUST) When an action may already have taken effect (for example, after a timeout or a retry, see [API-040](./07-api-integration-contract.md) to [API-042](./07-api-integration-contract.md)), the message MUST show the current status of the transaction before offering any further action. The product MUST NOT invite the member to resubmit an action that has already completed.
- Rationale: prevents apparent duplicate postings from user action; see [FR-023](./04-functional-requirements.md).

**UX-054** (MUST) A validation failure MUST NOT clear data the member has entered.

**UX-055** (MUST) Error messages MUST NOT display raw internal error text, stack traces, or identifiers that are not meaningful to the member. Errors returned by the interface MUST be mapped from the error categories in [07](./07-api-integration-contract.md).

**UX-056** (MUST) Required fields MUST be identified in text, not only by a visual marker. Field-level validation MUST occur on leaving a field or on submit; it SHOULD NOT run on every keystroke.

### Accessible presentation

**UX-060** (MUST) The product MUST meet the target accessibility conformance level recorded in OD-08-1 before the MVP is declared ready for release. Until verification is recorded, no document, screen, or release note MUST state that a conformance level is achieved.

**UX-061** (MUST) Text contrast MUST meet the contrast ratio required by the target level in OD-08-1. Non-text indicators that are needed to understand content or to identify focus MUST meet the non-text contrast ratio required by that level.

**UX-062** (MUST) Information MUST NOT be conveyed by color alone. This applies at minimum to balance direction (UX-020), transaction status (UX-024), entry sign (UX-025), and error state (UX-051).

**UX-063** (MUST) Every control MUST expose a name, role, and state to assistive technology. Dynamic changes to transaction status and validation summaries MUST be announced through a status mechanism without moving keyboard focus unexpectedly.

**UX-064** (MUST) Tables MUST associate each cell with its column and row headers. Sortable columns MUST expose their current sort state.

**UX-065** (MUST) Each page MUST declare its language. Content in a different language MUST be marked with its language.

**UX-066** (SHOULD) Non-essential motion SHOULD be avoided. Where motion is used, it SHOULD stop or be suppressible when the member's system indicates a reduced-motion preference.

**UX-067** (MUST) Each MVP screen MUST be checked by an automated accessibility scan and by manual keyboard and screen-reader passes before release. The test method for each UX requirement in this spec is recorded in the quality documentation, not here.

## Open decisions

**OD-08-1 Target conformance level.** Choose the accessibility level that UX-032, UX-033, UX-060, and UX-061 reference.
- Option A: WCAG 2.2 Level AA (recommended default).
- Option B: WCAG 2.1 Level AA.
- Option C: a higher level for selected screens, with AA as the baseline for all others.
- Default: Option A. Selecting any option does not establish conformance; verification does.

**OD-08-2 Time zone display.** Choose how dates and times are shown (UX-027).
- Option A: network's configured time zone, labeled on the screen (recommended default).
- Option B: member's device time zone, labeled on the screen.
- Option C: both, with the network time zone shown alongside the local time.
- Default: Option A.

**OD-08-3 Balance direction wording.** UX-020 requires direction in text but does not fix the words. Choose the approach.
- Option A: signed numeric value plus a fixed sentence that states the meaning of positive, zero, and negative values (recommended default).
- Option B: signed numeric value plus short status labels only.
- Option C: wording chosen per network by its administrator, within a platform-set required meaning.
- Default: Option A. Final wording is set in the copy process and MUST satisfy [ROL-007](./02-personas-roles-journeys.md).

**OD-08-4 Projected balance.** UX-023 forbids pending transactions in the balance figure. Choose whether a separately labeled projection is shown.
- Option A: no projection; pending transactions appear only in their own list (recommended default).
- Option B: a separately labeled projection figure that includes accepted but not completed transactions.
- Default: Option A, to keep the balance figure equal to posted entries (see [FR-029](./04-functional-requirements.md)).

**OD-08-5 Minimum supported width.** UX-030 sets 320 CSS pixels. Choose whether a different minimum applies.
- Option A: 320 CSS pixels (recommended default).
- Option B: 360 CSS pixels, if design research shows a different minimum is needed.
- Default: Option A.

**OD-08-6 Session timeout behavior.** UX-045 is a SHOULD. Choose the timeout policy, which may be a network setting.
- Option A: warn before expiry and allow extension, with no fixed timeout set by the platform (recommended default).
- Option B: fixed timeout set per network through the policy in [FR-032](./04-functional-requirements.md), with the warning required by UX-045.
- Default: Option A.

## Related docs
- [_context.md](./_context.md): shared terms, credit semantics, ledger invariants, normative terms.
- [01-product-vision-scope.md](./01-product-vision-scope.md): product scope and boundaries.
- [02-personas-roles-journeys.md](./02-personas-roles-journeys.md): roles, journeys, and role-based visibility.
- [03-domain-model-rules.md](./03-domain-model-rules.md): balances, credit limits, transaction states, and entries.
- [04-functional-requirements.md](./04-functional-requirements.md): MVP functional requirements that the screens expose.
- [05-system-architecture.md](./05-system-architecture.md): system boundaries for the interface layer.
- [06-data-ledger-audit.md](./06-data-ledger-audit.md): data and audit records shown in history views.
- [07-api-integration-contract.md](./07-api-integration-contract.md): interface contract for data, errors, and idempotency.
- [13-assumptions-risks-decisions-roadmap.md](./13-assumptions-risks-decisions-roadmap.md): assumptions, risks, and roadmap.
