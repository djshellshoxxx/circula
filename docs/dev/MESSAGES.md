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

The bounded pagination completion reuses `pager.previous`, `pager.next` and `pager.first` inside a named navigation region. `sessions.bulk_*` covers the separate other-session confirmation and result; `sessions.help_*` explains the complete flow on `/help#sessions`. Missing confirmation uses the existing error-summary and labelled-field rendering. These additions use the existing English catalogue and do not claim translations in other locales.

`TestSessionMessages` and the catalogue completeness check cover the wording. `TestAccountSessionsRequireOwnerCSRFAndConfirmation` and `TestAccountSessionsOmitExpiredSignIns` exercise the real handlers/templates, owner scope, credential secrecy, confirmation, current-browser sign-out and expired-session filtering. Runtime verification is recorded in the session-flow review report.
## Shared validation and administrator password-reset continuation

This continuation migrates the plain field labels, fixed-list corrections,
`fieldDetail`, `fieldFix` and `refuse` text, plus the administrator password-reset
confirmation and one-time temporary-password page. Its 123 new entries use
`validation.*`, `admin.reset.*` and `admin.member_action.*`. The shared member
action template also resolves its heading, required-reason hint and back link
through the catalogue; the other member actions and their policies are unchanged.

Complete refusal sentences use positional arguments. Available-credit and
largest-trade amounts stay exact preformatted strings, including values above
the floating-point exact-integer range. Service-provided field details and
blocker descriptions remain ordinary text and are not translated in this slice.
Unknown field names retain the existing underscore-to-space fallback. Existing
English wording, status codes, authorization, audit behavior, CSRF fields and
password-reset lifetime rules are unchanged. Names, field details, retained form
values and the temporary password remain subject to contextual HTML escaping.

`TestValidationMessages` and `TestAdminResetMessages` check message formatting,
exact amounts, positional values, Unicode and literal percent signs. The
catalogue completeness check supplies every new interpolation argument.
`TestValidationCatalogueRendersEscapedFieldDetails` and
`TestAdminResetPagesCatalogue` render the real templates to check escaping and
accessible form associations. Existing `TestValidationMessagesNameFieldAndFix`,
`TestRefusalCoversEveryCode` and `TestWebAdminReset` cover the unchanged refusal
and reset behavior. This is a partial UX-047 continuation; service/API messages,
other flows, JavaScript and language scope remain separate work.

Local receipt for this continuation: the unchanged catalogue suite passed.
The added message regression failed before implementation because its first
new catalogue entry did not exist. Post-change catalogue race and focused web
runs were interrupted during workspace resource contention without test
results; they are not passing evidence. The combined candidate still needs
those checks and current-revision CI before integration.

## Marketplace and directory continuation

This continuation adds 66 entries under `listing.*`, `profile.*` and
`directory.*` for listing browsing, creation and editing, member profiles and
the directory. It covers page titles, filters, form labels, status labels,
optional-price hints, validation summaries, success notices, empty states and
complete member/listing action names. Visible action labels stay unchanged;
the full accessible names now come from catalogue-backed `aria-label` values
instead of joining visible labels with screen-reader-only fragments. Stable
member references remain unchanged identifiers, including their existing
strong emphasis on profiles.

The listing and profile templates now use the existing per-amount `unitFor`
helper. A price of one credit no longer says "1.00 credits", and a unit already
named "Hours" no longer gains another `s`. Prices remain exact strings from
the minor-unit formatter; the catalogue does not convert them to floating
point or group them as counts. Directory result counts use English CLDR plural
and number formatting (`1 member`, `1,000 members`), and the price correction
uses "1 decimal place" for singular precision. Names, titles, descriptions,
unit names, retained form values and accessible labels remain ordinary text
escaped by `html/template`. Routes, query handling, pagination, CSRF,
authorization and ledger behavior are unchanged by this slice.

`TestMarketplaceMessages` checks plural/count formatting, exact large amounts,
Unicode, percent signs and reference strings. `TestMarketplacePriceUnits`
renders prices in both actual page templates. `TestMarketplacePagesCatalogue`
and `TestDirectoryCatalogueCountsAndReferences` render the public/own listing
states, form controls, profile actions, directory references and malicious
interpolated values.

Local evidence on 2026-11-19 with Go 1.26.11: the new price regression failed
before the change on both views for singular credits and `Hourss`; the new
message regression failed on the absent `directory.results` entry. After the
change, the complete message package passed with `-race -count=1` (1.074s).
A file-list race run containing every production web Go file and only
`marketplace_catalog_test.go` passed (10.473s), exercising the production
templates without compiling unrelated SQLite-backed fixtures. That selective
render run is not a complete web-package, HTTP, browser or accessibility-suite
qualification. Those checks remain required for the combined candidate. This
is further partial UX-047 completion; other flows, JavaScript, service/API
messages and additional languages remain outside this slice.

## Freshness and session-warning continuation

This continuation moves the seven remaining browser-enhancement messages into
`freshness.*` and `session_warning.*`. Each entry is a complete sentence or
complete set of sentences with a positional string argument where needed. The
shared shell supplies the browser messages in contextually escaped data
attributes; `app.js` substitutes the time or saved snapshot into the supplied
text and writes it with `textContent`. Function replacement keeps dollar
sequences in substituted text literal. The existing `session.*` button and
heading labels remain in use.

The server-rendered snapshot keeps its existing `<time>` element and works
without JavaScript. `messages.Around` requires exactly one string slot and
returns ordinary `Before` and `After` strings around that element. The template
escapes those strings and inserts its existing trusted timestamp separately;
catalogue content never becomes trusted HTML.

The change preserves offline/history warnings, the last server timestamp,
the rule that returning online does not imply fresh figures, session warning
timing, dismissal, extension requests and server expiry handling. Expiry does
not redirect or discard unsent form values. It adds no session policy,
endpoint, language or locale negotiation.

Local verification: `node --test scripts/test_app_messages.cjs` passes all seven
tests against the production script. Six of those tests failed against the
original script because it ignored the supplied messages. The tests cover
literal dollar sequences and HTML-like text, offline/history state, warning
timing, fixed sessions, extension success and failure, safe return links, and
retained form values. CI runs this dependency-free check in the lint job.
`go test -race -p 1 ./internal/web/messages` passes. The two
`TestProgressiveCatalogue*` tests pass with the race detector using a file-list
run of the production web sources and their test file; this checks the actual
templates, complete English text, the existing time element and escaping of
all six new message attributes. It is not a full web-package or browser run.
Combined-candidate web and browser checks remain integration work. This
continuation does not complete the remaining UX-047 flows or approve new
languages.
