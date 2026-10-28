# 08 UX, Information Architecture and Accessibility Requirements

Status: Draft v0.1 · Owner/role: UX lead (with accessibility reviewer) · Last updated: 2026-10-24
Scope: MVP information architecture, core screens (requirements level only), presentation of balances and transactions (direction, pending versus completed, negative balances, available credit), validation and error handling, responsive and keyboard requirements, accessibility targets against WCAG 2.2 Level AA. Excludes visual design, brand, and feature-level interaction specifications.
Dependencies: [00-conventions.md](00-conventions.md) (glossary, LED-1..LED-7, DEC-002); [02-personas-journeys.md](02-personas-journeys.md) (roles and journeys); [03-domain-model.md](03-domain-model.md) (state definitions); [04-functional-requirements.md](04-functional-requirements.md) (FR screens); [05-architecture.md](05-architecture.md) (ARC-017); [07-api-contract.md](07-api-contract.md) (error codes and balance fields); [09-security-privacy.md](09-security-privacy.md) (session handling); [12-quality-test-plan.md](12-quality-test-plan.md) (accessibility tests).
Open questions: see section 12.

Normative terms follow [00-conventions.md](00-conventions.md) section 1. WCAG references are to WCAG 2.2 (W3C Recommendation, 5 October 2023; success criteria cited by number, to be verified against the W3C text by the accessibility reviewer before release, as recorded in section 12).

---





## Build status (as built)

Snapshot: HEAD `925f637` (branch claude/circula-build). Statuses were first read at `31b1015` and re-checked at `925f637` (see TRACEABILITY.md). **built** = behaviour exists in code; **partial** = exists with the gaps stated, or the required verification was not run; **not built**; **deferred** = out of MVP by this spec or a recorded decision. Code paths and test names: [TRACEABILITY.md](../TRACEABILITY.md). Open owner decisions and resolved questions: [DECISIONS.md](../DECISIONS.md). Status is implementation status, not a compliance claim; no release gate has been run.

Review note (2026-10-24): The build-status snapshot above is retained as historical evidence and has not been re-audited as a whole at this checkpoint. See [the 2026-10-24 continuation review](../audit/RESUME-2026-10-24.md). Historical-balance presentation below clarifies UX-061; it does not claim that the feature or its acceptance evidence exists.

Summary: built 31, not built 7, partial 22.

| ID | Priority | Status | Note |
|---|---|---|---|
| UX-001 | P0 | built | Navigation by role; hidden pages return not-found or forbidden |
| UX-002 | P0 | partial | Home reachable in three activations not measured |
| UX-003 | P1 | built | Network switcher and separate network pages |
| UX-004 | P0 | built | In credit, In debt, Level labels |
| UX-005 | P0 | partial | Sign and words used; greyscale check not run |
| UX-006 | P0 | built | Accessible name spells out direction in words |
| UX-007 | P0 | built | Help text explains sign convention |
| UX-008 | P0 | built | Network unit name and decimals; no currency symbols |
| UX-009 | P0 | built | Balance card shows balance, limit, reserved, available |
| UX-010 | P0 | built | Available labelled as available to spend with help link |
| UX-011 | P0 | built | Text states credit in use against limit |
| UX-012 | P1 | built | Credit-use warning at the policy percent (default 80); OQ-UX-02 threshold default still to confirm |
| UX-013 | P0 | built | Direction in words per transaction line |
| UX-014 | P0 | built | Pending section excluded from balance |
| UX-015 | P0 | built | Completed rows show posted time |
| UX-016 | P0 | built | Status labels and explanations; accepted not shown |
| UX-017 | P1 | partial | Reversal and adjustment rows exist; link to original not verified |
| UX-018 | P0 | built | Review step before accept, propose, dispute |
| UX-019 | P0 | built | Confirmation message after posting |
| UX-020 | P0 | built | Refusal copy names cause and that nothing changed |
| UX-021 | P0 | partial | Error summary and field messages present; not verified against all codes |
| UX-022 | P0 | partial | Error summary present; focus move and value retention not verified |
| UX-023 | P0 | partial | inputmode decimal on amount fields; format text in helper |
| UX-024 | P1 | not built | Member time zone not supported; times shown in server zone |
| UX-025 | P0 | built | Generic 5xx page with request reference |
| UX-026 | P0 | partial | Live region on some pages; asynchronous announcements not verified |
| UX-027 | P0 | partial | Disabled controls explained in text; not audited for all |
| UX-028 | P0 | partial | Loading state not implemented; empty and error states present |
| UX-029 | P0 | built | Permission failure page with link home |
| UX-030 | P0 | built | Two-minute warning with Stay signed in extension |
| UX-031 | P1 | partial | Suspended notice not verified in templates |
| UX-032 | P0 | built | Confirmation step names object for remove and cancel |
| UX-033 | P0 | partial | Single-column layout with media queries; 320px checked by manual QA only |
| UX-034 | P0 | partial | Resize and reflow not measured |
| UX-035 | P0 | partial | Target sizes not measured |
| UX-036 | P0 | partial | No orientation lock; not tested |
| UX-037 | P1 | not built | Stacked rows for wide tables not built |
| UX-038 | P0 | partial | Keyboard operation verified manually for sign-in and trade form only |
| UX-039 | P0 | partial | Reading order by template; not automated |
| UX-040 | P0 | partial | Focus outline present in CSS; not automated |
| UX-041 | P0 | built | Skip link to main |
| UX-042 | P0 | built | No drag interactions |
| UX-043 | P0 | built | Actions fire on submit and link activation |
| UX-044 | P0 | built | Page title includes network name and screen |
| UX-045 | P0 | built | Semantic lists, tables, landmarks |
| UX-046 | P0 | built | Document language en |
| UX-047 | P1 | not built | Strings not externalised into catalogues |
| UX-048 | P0 | built | Visible labels and autocomplete |
| UX-049 | P1 | partial | Help link in same place not verified |
| UX-050 | P0 | built | Copy avoids money and cash wording for credits |
| UX-051 | P0 | built | Sign-in form allows paste; no cognitive test |
| UX-052 | P0 | partial | Recovery not built; registration response uniform |
| UX-053 | P0 | not built | Second factor not built |
| UX-054 | P0 | partial | No automated WCAG checker in CI; structural tests only |
| UX-055 | P0 | not built | Keyboard-only walkthrough not recorded for release |
| UX-056 | P0 | not built | Screen-reader pass not recorded |
| UX-057 | P0 | partial | Dark scheme token set present; contrast not measured |
| UX-058 | P1 | built | Reduced motion media query present |
| UX-059 | P1 | built | Session warning can be extended |
| UX-060 | P0 | partial | Defect logging process not evidenced in repo |

## 1. Principles

1. **Direction is unmistakable.** A reader MUST be able to tell whether a balance or a transaction increases or decreases their balance from text alone (section 4).
2. **Consent before posting.** A member MUST review and confirm any action that posts to the ledger before it is submitted (3.3.4).
3. **Same meaning everywhere.** A status, balance field or error code means the same thing on every screen and in the API (07 section 2.1).
4. **Core flows work without JavaScript and without a mouse.** DEC-002 and ARC-017.
5. **No overclaiming.** Copy MUST NOT state or imply that credits are money, can be redeemed, or are guaranteed to be accepted (00 section 3, 01). Legal copy is subject to counsel review (section 12).

## 2. Information architecture

### 2.1 Top-level sections

| Section | Visible to | Contents |
|---|---|---|
| Home | Active member | Balance summary, available credit, trades needing action, recent activity |
| Trades | Active or suspended member | Proposals incoming and outgoing, active and past trades, detail |
| Directory | Members with directory access | Member list and profiles |
| Listings | Members | Offers and requests; create and edit own listings |
| History | Member | Own account statement (entries, running balance optional); historical balance lookup at an exact instant (UX-061) |
| Disputes | Parties to a dispute; dispute reviewers | Dispute cases and queue |
| Network admin | Network administrator | Members, policy, adjustments, audit, checks, announcements, exports, invitations, roles |
| Account | Any signed-in person | Profile, sign-in and security, API tokens, sign out, network memberships |
| Help | Everyone | Plain-language explanation of credits, balances, statuses and rules (network-specific rules linked) |

| Rule | Requirement |
|---|---|
| UX-001 · P0 | Navigation MUST show only sections the current person's role permits. A hidden section MUST NOT be reachable by URL with a different result than a 403 or 404 page (API-017 rule). |
| UX-002 · P0 | Every MVP core screen in section 3 MUST be reachable from Home in no more than three activations (links or buttons). |
| UX-003 · P1 | A person who belongs to more than one network MUST see a network switcher, and each network's data MUST be presented as separate (no combined totals). |

## 3. Core screens (MVP, requirements level)

| Screen | Required content | Primary actions |
|---|---|---|
| Sign-in and join | Sign-in form; join entry according to join mode (invite, approval, open) | Sign in, request to join, accept invitation |
| Home | Balance card (UX-004), available credit, trades awaiting my action, latest history | Go to trade, propose trade |
| Directory | Searchable, paginated list; profile with listings | View profile, start proposal |
| Listings | Filterable list by kind (offer or request); create/edit form | Create, edit own, propose from listing |
| Propose trade | Counterparty, direction (pay or receive from my view), amount, description, optional listing; review step | Review, submit, cancel |
| Trades list | Tabs or filters for status and role; empty state | Open trade |
| Trade detail | Parties, direction, amount, status with timeline, expiry, actions allowed in current state | Accept, decline, cancel, raise dispute |
| Confirm trade acceptance | Review of amount, both names, direction, resulting balance and available credit after posting | Confirm, go back |
| History | Entries with date, description, counterparty, signed amount, status, ledger reference | Filter by period, open trade |
| Disputes | Dispute detail with timeline; reviewer queue | Open dispute, add reason, reviewer decision |
| Account and security | Display name, sign-out, session list, password change, recovery method, API tokens | Change, revoke |
| Network admin: members | Pending approvals, members by status | Approve, suspend, reinstate, remove, edit credit limit |
| Network admin: policy | Current values, history with reasons and effective dates | Edit (with reason), view versions |
| Network admin: adjustments | Form with member, direction, amount and mandatory reason; effect preview | Submit after review |
| Network admin: audit and checks | Filterable audit log; consistency check results | Filter, run check, export |
| Errors | Not found, forbidden, expired session, server error | Return home, sign in, copy reference |

## 4. Balance and transaction presentation

This section is the single UI authority for balance direction. Field names match 07 section 8 (API-044) and the balance definitions in [00-conventions.md](00-conventions.md) section 3.

### 4.1 Balance states

| Requirement | Text (visible) | Accessible name adds | Example visible text |
|---|---|---|---|
| UX-004 · P0 | `In credit` when balance > 0 | the amount and the phrase "in credit" | `+12.50 · In credit` |
| UX-004 · P0 | `In debt` when balance < 0 | the amount and the phrase "in debt" | `−12.50 · In debt` |
| UX-004 · P0 | `Level` when balance = 0 | "level, zero credits" | `0.00 · Level` |

Rules that apply to all three states:

- **UX-005 · P0 (no colour alone).** Direction MUST be conveyed by the sign character and the word. Colour MAY reinforce but MUST NOT be the only cue. Test: the screen remains fully understandable in greyscale and with colours disabled.
- **UX-006 · P0 (screen reader text).** The accessible name of a balance MUST spell out the sign in words, for example "In debt by 12.50 credits", so that the minus sign is not read as an unknown symbol.
- **UX-007 · P0 (meaning).** Help text MUST explain the sign convention in plain language: "In credit" means the member has provided more value than it has received in this network; "In debt" means the member has received more value than it has provided and is using some of its credit limit. The help text MUST NOT describe these as money owed by or to the network.
- **UX-008 · P0 (units).** Amounts MUST be shown with the network's credit unit name and the decimal places in the network policy. Currency symbols (for example dollar or euro signs) MUST NOT be used for credits.

### 4.2 Balance card

| Requirement | Content |
|---|---|
| UX-009 · P0 | The Home balance card MUST show, as separate labelled values: current balance with direction (UX-004); credit limit; reserved amount for pending outgoing trades; available to spend (API-044 `available_minor`). |
| UX-010 · P0 | Available to spend MUST NOT be presented as the balance. Its label MUST include the words "available to spend", and the help link MUST explain that pending outgoing trades reduce it. |
| UX-011 · P0 | When the member is in debt, the card MUST show the portion of the credit limit in use as text ("You are using 25.00 of your 50.00 credit limit") and MAY show a meter. The meter MUST have a text equivalent and MUST NOT be the only indicator. |
| UX-012 · P1 | When the member's in-use credit reaches the network's warning threshold (default proposed 80 percent, product decision), an inline notice MUST appear on Home and on the propose form. As built: threshold is the network policy `limit_warning_percent` (default 80, 1 to 100); the notice shows on Home and the propose form, and the hourly job also sends an inbox notification once per ISO week. |

### 4.3 Transactions and pending versus completed

| Requirement | Content |
|---|---|
| UX-013 · P0 | Each transaction line MUST state the viewer's direction in words: "You paid Alex 5.00" (balance decreased) or "You received 5.00 from Alex" (balance increased), with the signed amount displayed (`−5.00` or `+5.00`). |
| UX-014 · P0 | Pending trades (status `proposed`) MUST be listed in a section labelled "Pending" and MUST NOT be included in the balance figure. Their reserved effect appears only as the reserved amount (UX-009). |
| UX-015 · P0 | Completed trades MUST show "Completed" with the posting date and time (member's time zone, UX-024) and a ledger reference. A completed trade MUST NOT show the word "pending". |
| UX-016 · P0 | Each status MUST have a text label and a one-line explanation in the detail view. The labels are: `Waiting for <name> to respond` (proposed, incoming or outgoing), `Completed`, `Declined`, `Cancelled`, `Expired`, `Disputed`, `Reversed`. The internal state `accepted` MUST NOT be shown as a separate label in the MVP (assumption OQ-ARC-04). |
| UX-017 · P1 | Reversals and adjustments MUST be shown as separate lines linked to the original, with their own date, reason, and signed effect. The original line MUST remain unchanged in appearance except for a link "Reversed by …". |

### 4.4 Historical balance

| Requirement | Content |
|---|---|
| UX-061 · P1 | The member History screen MUST let a member request the historical balance of their own account at an exact instant, using an explicitly offset RFC 3339 date-time input. The interface MUST NOT infer an instant from a date-only value or the member's display time zone. On success, it MUST show the signed amount, direction and normalized UTC instant (with `Z`), and explain that this is the posted balance at that instant. It MUST NOT label current credit limit, reserved amount, or available-to-spend values as historical. Malformed or future input MUST identify the timestamp field and state the correction. When no account postings satisfy the requested timestamp cutoff, it MUST show zero and an explicit "No postings as of this instant" state, irrespective of network creation time. If clock rollback gives an account posting a timestamp earlier than network creation, that posting MUST still contribute whenever it satisfies the cutoff, consistent with FR-054. |

## 5. Decisions, confirmation and errors

| Requirement | Content |
|---|---|
| UX-018 · P0 (3.3.4) | Accepting a trade, proposing a trade, creating a reversal or adjustment, and raising a dispute MUST show a review step before submission. The review step MUST show: direction, amounts, both member names, expiry where applicable, and the balance and available credit that will result. A clear "Go back" action MUST be present and MUST NOT lose entered values. |
| UX-019 · P0 | After a successful posting the confirmation message MUST state the new balance and direction (UX-004 wording). |
| UX-020 · P0 | Refusals MUST state the cause in plain language and the fact that nothing changed, for example: "Not sent. This trade would take your available credit below zero. Available to spend is 12.00; this trade is 20.00." Each refusal code in 07 section 7.2 MUST have a matching message. |
| UX-021 · P0 (3.3.1, 3.3.3) | Validation errors MUST identify the field in text, describe the problem, and suggest a correction. Colour MUST NOT be the only indicator of an error. |
| UX-022 · P0 | On a failed submission, the page MUST show an error summary at the top that links to each invalid field; focus MUST move to the summary. Values entered by the member MUST be retained (3.3.7 redundant entry). |
| UX-023 · P0 | Amount inputs MUST state the accepted format in the label or helper text ("Amount in credits, up to 2 decimal places"), use the numeric keyboard where available (`inputmode="decimal"`), and reject values with more decimal places than the policy allows with a message that names the limit. |
| UX-024 · P1 | Dates and times MUST be shown in the member's chosen time zone with its abbreviation, in an unambiguous format (for example "28 Oct 2026, 17:00 CET"). Machine-readable values MUST use the `time` element with a `datetime` attribute. |
| UX-025 · P0 | Server failures (5xx) MUST show a generic message with the request reference (07 API-030) and no internal detail. |
| UX-026 · P0 | Asynchronous results (for example "Trade sent" after a background change) MUST be announced through a live region (4.1.3). Focus MUST NOT move without a user action unless the user has just submitted a form. |
| UX-027 · P0 | Disabled controls MUST be accompanied by visible text explaining why they are disabled and what would enable them. A control MUST NOT be disabled silently. |

## 6. Navigation, permissions and states

| Requirement | Content |
|---|---|
| UX-028 · P0 | Every list and detail screen MUST define four states: loading (announced to assistive technology), empty (explanation and next step), error (reason and retry), and content. |
| UX-029 · P0 | A permission failure MUST show a page that explains the person does not have access, with a link to Home. A not-found response for another network or a hidden resource MUST look the same as a non-existent resource (07 API-017). |
| UX-030 · P0 (2.2.1) | Sessions that expire MUST warn the person at least 2 minutes before expiry (default; product decision) with an option to extend. Expiry MUST send the person to sign in and return them to the page afterwards. Unsent form values SHOULD be preserved where technically feasible. |
| UX-031 · P1 | Suspended members MUST see a persistent notice explaining that they can view but not propose, accept or post, and how to contact the network administrator (per 03). |
| UX-032 · P0 | Destructive or irreversible-looking actions (remove member, cancel trade, revoke token, close network) MUST use a confirmation step that names the object and states the consequence. |

## 7. Responsive layout

| Requirement | Content |
|---|---|
| UX-033 · P0 (1.4.10) | All core screens MUST reflow to a single column at a 320 CSS pixel viewport width without horizontal scrolling, except for data tables that use a labelled, keyboard-focusable scroll region. |
| UX-034 · P0 (1.4.4, 1.4.12) | Text MUST remain readable and all functionality available when text is resized to 200 percent, and when line height, paragraph spacing, letter spacing and word spacing are increased per 1.4.12. |
| UX-035 · P0 (2.5.8) | Interactive targets MUST be at least 24 by 24 CSS pixels; targets SHOULD be at least 44 by 44 CSS pixels on touch layouts. |
| UX-036 · P0 (1.3.4) | The layout MUST NOT lock orientation; portrait and landscape MUST both be usable. |
| UX-037 · P1 | Tables with more than four columns MUST collapse into labelled stacked rows at narrow widths, with the same information available. |

## 8. Keyboard and focus

| Requirement | Content |
|---|---|
| UX-038 · P0 (2.1.1, 2.1.2) | Every function reachable with a pointer MUST be reachable and operable with the keyboard alone. Focus MUST NOT be trapped except in modal dialogs that close with Escape and return focus to the trigger. |
| UX-039 · P0 (2.4.3) | Focus order MUST follow reading order. |
| UX-040 · P0 (2.4.7, 2.4.11) | A visible focus indicator MUST be present on every focusable element, and focused elements MUST NOT be fully hidden by sticky headers or banners. |
| UX-041 · P0 (2.4.1) | Each page MUST provide a skip link to the main content. |
| UX-042 · P0 (2.5.7) | Any action that uses dragging MUST have a single-pointer or keyboard alternative. The MVP SHOULD avoid drag-based interactions altogether. |
| UX-043 · P0 (2.5.1, 2.5.2) | Multi-point or path-based gestures MUST NOT be required. Actions MUST fire on pointer release or on keyboard activation with a means to cancel. |

## 9. Semantics, language and content

| Requirement | Content |
|---|---|
| UX-044 · P0 (2.4.2, 2.4.6) | Each page MUST have a unique title that includes the network name and screen name. Headings MUST describe their sections and form a logical outline. |
| UX-045 · P0 (1.3.1, 4.1.2) | Structures (lists, tables with header cells and scope, forms with labels, landmarks: header, nav, main, footer) MUST be marked up semantically. Custom components MUST expose name, role and state. |
| UX-046 · P0 (3.1.1) | The document language MUST be set. The MVP language is English; all strings MUST be externalised so that translation is possible (UX-047). |
| UX-047 · P1 | The UI MUST be structured for localisation: no concatenated sentences in code, plural and number formatting via message catalogues. Translation scope is a product decision. |
| UX-048 · P0 (3.3.2, 2.5.3) | Form labels MUST be visible text; the accessible name MUST contain the visible label text. Input purposes MUST be declared with `autocomplete` where applicable (1.3.5). |
| UX-049 · P1 (3.2.6) | Help links MUST appear in the same location on every page of the same section. |
| UX-050 · P0 | Copy MUST NOT use the words "money", "cash", "coins", "tokens" (except for the API token screen) or "investment" to describe credits. Copy MUST describe credits as accounting units of the network, consistent with 00 section 3. Final copy MUST be reviewed for legal and regulatory claims before launch (section 12). |

## 10. Authentication and accessibility of sign-in

| Requirement | Content |
|---|---|
| UX-051 · P0 (3.3.8) | Sign-in and recovery MUST NOT require the person to solve a cognitive test (for example, transcribing or puzzles) unless an alternative that does not require such a test is also available. Paste into password fields MUST be allowed, and password managers MUST work. |
| UX-052 · P0 | Account recovery flows MUST use the same error message for a recognised and an unrecognised identifier where needed to prevent account enumeration (detail in 09). |
| UX-053 · P0 | Two-factor or other additional verification, if adopted, MUST be available with a non-cognitive method and documented recovery (09 decides). |

## 11. Accessibility verification

| Requirement | Content |
|---|---|
| UX-054 · P0 | Automated accessibility checks MUST run on every page template in CI with the WCAG 2.2 AA rule set; zero violations are required to merge (ARC-017). Automated tools MUST NOT be the only evidence. |
| UX-055 · P0 | Each release MUST pass a keyboard-only walkthrough of the core flows: sign-in, propose trade, accept trade with review, decline, history view, admin member approval. Results MUST be recorded in the release checklist. |
| UX-056 · P0 | Each release MUST have a screen-reader pass of the same core flows on at least one desktop and one mobile combination of screen reader and browser. Results MUST be recorded. Combinations used MUST be listed. |
| UX-057 · P0 | Colour contrast MUST meet 4.5:1 for body text and 3:1 for large text and for non-text UI boundaries and focus indicators (1.4.3, 1.4.11). The check MUST be run for every colour token in the design system. |
| UX-058 · P1 | Motion MUST respect the operating system "reduce motion" setting. No content MAY flash more than three times per second (2.3.1). |
| UX-059 · P1 | The application MUST provide a way to close or extend time-limited content (session warning, UX-030) (2.2.1). |
| UX-060 · P0 | Accessibility defects found in release testing MUST be logged and classified. Any WCAG 2.2 AA failure on a core flow is release-blocking (12). |

## 12. Open questions

- **OQ-UX-01 (resolved as built).** Question moved to [DECISIONS.md](../DECISIONS.md): see DEC-055.
- **OQ-UX-02 (decision needed):** Warning threshold for credit use (UX-012, default 80 percent) and session warning lead time (UX-030, default 2 minutes).
- **OQ-UX-03 (resolved as built).** Question moved to [DECISIONS.md](../DECISIONS.md): see DEC-054.
- **OQ-UX-04 (decision needed, counsel):** Final wording of balance and credit explanations (UX-007, UX-050). Copy must not assert monetary value, redemption, or regulatory status. Qualified counsel review is required before launch.
- **OQ-UX-05 (resolved as built).** Question moved to [DECISIONS.md](../DECISIONS.md): see DEC-054.
- **OQ-UX-06 (verification):** The WCAG 2.2 success-criterion numbers cited in this document are taken from the W3C Recommendation and MUST be checked against the published text by the accessibility reviewer before the first release. Access date and URL to be recorded in that review.
- **OQ-UX-07 (resolved as built).** Question moved to [DECISIONS.md](../DECISIONS.md): see DEC-054.

Cross-references: domain definitions [03-domain-model.md](03-domain-model.md); functional requirements [04-functional-requirements.md](04-functional-requirements.md); API operations and error codes [07-api-contract.md](07-api-contract.md); security and sign-in [09-security-privacy.md](09-security-privacy.md); governance announcements [10-governance-operations.md](10-governance-operations.md); test plan [12-quality-test-plan.md](12-quality-test-plan.md); decisions [13-assumptions-risks-decisions.md](13-assumptions-risks-decisions.md).
