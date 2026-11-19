# UI messages (UX-047)

`internal/web/messages/en.json` is the first English message catalogue. It is
embedded in the binary; there is no runtime file, language selector or new
dependency. The existing pinned `golang.org/x/text` supplies CLDR plural rules
and English integer formatting. No locale expansion is approved by this work.

Use `messages.Text(key, args...)` for **complete sentences**, including the
count. A plural entry supplies `one` and `other` forms, selecting on integer
argument 1. Positional placeholders (`%[1]d`, `%[2]s`) allow argument reordering
without rebuilding sentences in a handler. Count values use locale formatting
(for example, `1,000`); amounts remain exact strings from the domain's fixed
minor-unit formatter. Never convert an amount to floating point for translation.

The formatter returns ordinary strings. Keep automatic `html/template` escaping
for member names, currency names and other interpolated content; never cast the
result to trusted HTML. Unknown keys fail visibly instead of printing a key to
the user. The embedded catalogue is validated on load. Each new entry needs a
formatting test, including its plural forms and argument ordering where relevant.

Migrated: all six administrator-overview count sentences, overview actions and
check messages, overview/queue titles, queue party descriptions and row status.
The count remains a separate numeric field only for link visibility, not for
sentence construction. Existing English wording is preserved for ordinary counts.

Not migrated: the remaining templates, JavaScript, other handler messages,
currency-unit inflection and API/service errors. In particular, this first
catalogue does not make the entire UI localizable or complete UX-047. Migrate
one complete flow at a time with rendered-page regressions and keep policy,
ledger and API semantics unchanged. Additional languages need their own plural
rules and product scope; do not add automatic language negotiation as a side effect.

Verification: `go test -race ./internal/web/messages` exercises singular, zero,
plural and grouped counts, positional arguments, exact large amounts, Unicode,
literal percent signs, escaping and missing keys. `TestAdminOverviewPage` and
`TestAdminStaleProposalQueue` exercise the actual handler/template integration.

## Financial-message continuation

The candidate financial-message change extends the catalogue to balance state and spoken descriptions, credit-limit usage, neutral member names, transaction statuses/explanations, reversal links and full row sentences for trade/reversal/adjustment/grant kinds. It preserves exact fixed-point amount strings and formats ledger references as unchanged identifier strings rather than grouped quantities. Zero-balance speech reuses the existing unit inflector, avoiding an extra suffix for names such as `Hours`.

`TestBalanceCatalogueMessages` and `TestTransactionCatalogueRows` are new regression checks, and the catalogue completeness test supplies the new interpolation arguments. These checks have not run locally because the execution environment is unavailable; candidate CI is authoritative. This is still a partial UX-047 migration, not a locale-scope decision or completion of all UI text.

## Shared-shell and dashboard continuation

The next candidate exposes `msg` as a plain-text template helper backed by the same English catalogue. It migrates shared labels, pagination labels, dashboard headings/actions/empty states, main navigation, persistent network notices and unread/new counts. User-controlled values remain subject to contextual HTML escaping. The dashboard uses `Card.Unit`, already inflected by the formatter, instead of appending another `s` to the network unit. `TestDashboardCatalogueAndUnitNames` renders the real shell/dashboard with ordinary, already-plural and HTML-sensitive unit names.

Timestamp-rich notices still retain their existing `time` markup; they are not converted to trusted catalogue HTML. Remaining templates, other handlers, JavaScript and complete locale/number/date scope are still open. This continuation does not complete UX-047 or approve translations.

## Authentication-flow continuation

This continuation migrates the authentication and account-security pages: sign-in (`login`), second sign-in step (`login_mfa`), account creation (`register`), recovery (`recover`), recovery codes (`recovery_codes`), forced and voluntary password change (`password`), two-step enrolment/disable/replacement (`mfa_enrol`, `mfa_recovery`) and account security with security notices (`security`). It covers page titles, labels, hints, headings, buttons, link text, the 429/401/422/409 summaries and field errors, flash messages for sign-out, registration, recovery, password change, forced password change, two-step turn-off and notices read, and the security-notice texts (121 entries, keys `auth.*` and `security.*`). Counted sentences (`security.mfa.recovery_left`) use `one`/`other`.

Rules kept: the uniform sign-in failure text is unchanged (`auth.login.failed`, `auth.login.failed_field`) and no account-enumeration distinction was added; throttling text, status codes, CSRF and redirects are unchanged; accessible names are unchanged. Existing English was preserved. Two rendering notes: sentences that contain a link or `<strong>` are split at the markup boundary into separate entries (for example `auth.login.new_here` plus link text), and the recovery-codes count keeps its `<strong>` count with a plural fragment (`auth.codes.unused`) because existing tests and the page design depend on that markup. An apostrophe in catalogue text (`network's`) is now emitted as `&#39;` by `html/template`; the visible text is identical.

Not migrated in that authentication-only continuation: the `setup`, `join`, `join_anon` and `joinnet` pages (covered by the onboarding continuation below), the shared validation text produced by `refuse` and `fieldDetail`, the admin "Reset password" confirm text and temporary-password page, JavaScript text, API/service errors, other templates and any locale decision. UX-047 remains open.

Verification: `TestAuthenticationMessages` (plural, zero and grouped counts, verbatim refusals) and the catalogue completeness test; `auth_catalog_test.go` renders each migrated page through the real templates and checks catalogue text, accessible labels, uniform failure wording and HTML escaping of interpolated values.

## Setup and joining continuation

The 2026-11-18 Pacific audit externalises 51 additional messages for `setup`, `join`, `join_anon` and `joinnet`, their handler titles, local validation/refusal messages and success notices, and the shared terms heading. Network names and terms stay ordinary template data. Amount examples remain exact minor-unit formatting; terms versions stay ungrouped record identifiers. Join throttling uses complete messages for its default ten-minute window and configurable window instead of assembling sentence fragments. Labels, route choices, HTTP statuses, CSRF and membership rules are unchanged by this catalogue migration. The separate precision validation repair rejects malformed or out-of-range setup precision before formatting any amount example.

`TestOnboardingMessages` checks exact amounts, Unicode, literal percent signs, identifier formatting and both throttle messages. `TestOnboardingPagesCatalogue` renders all four pages and all three join modes, including malicious names, terms and return paths. The message package's race suite and the actual production web templates with the authentication/onboarding render tests passed locally. The latter used a file-list Go test invocation to exclude unrelated SQLite-backed fixtures; full handler integration and browser execution remain subject to the combined PR CI. Shared `refuse`/`fieldDetail` text, administrator reset pages, remaining product templates and JavaScript are still outside the migrated flows. This is additional partial UX-047 completion; it does not claim full localisation or approve other languages.

## Active-session continuation

The account-security session inventory now has a complete English catalogue flow (`sessions.*`): owner-only active sign-ins, current-session label, timestamp labels, the confirmation screen, consequences, validation and post-action messages. The page is linked from account security. Every state change retains CSRF protection, and the confirmation names the sign-in by its creation and last-active times. Existing bearer values and stored session hashes are never rendered. No new locale or timeout policy is introduced.

`TestSessionMessages` and the catalogue completeness check cover the wording. `TestAccountSessionsRequireOwnerCSRFAndConfirmation` and `TestAccountSessionsOmitExpiredSignIns` exercise the real handlers/templates, owner scope, credential secrecy, confirmation, current-browser sign-out and expired-session filtering. Runtime verification is recorded in the session-flow review report.
