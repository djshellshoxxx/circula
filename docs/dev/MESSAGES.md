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

Not migrated in that authentication-only continuation: the `setup`, `join`, `join_anon` and `joinnet` pages (covered by the onboarding continuation below), the shared validation text produced by `refuse` and `fieldDetail`, the admin "Reset password" confirm text and temporary-password page (covered by the administrator-reset continuation below), JavaScript text, API/service errors, other templates and any locale decision. UX-047 remains open.

Verification: `TestAuthenticationMessages` (plural, zero and grouped counts, verbatim refusals) and the catalogue completeness test; `auth_catalog_test.go` renders each migrated page through the real templates and checks catalogue text, accessible labels, uniform failure wording and HTML escaping of interpolated values.

## Setup and joining continuation

The 2026-11-18 Pacific audit externalises 51 additional messages for `setup`, `join`, `join_anon` and `joinnet`, their handler titles, local validation/refusal messages and success notices, and the shared terms heading. Network names and terms stay ordinary template data. Amount examples remain exact minor-unit formatting; terms versions stay ungrouped record identifiers. Join throttling uses complete messages for its default ten-minute window and configurable window instead of assembling sentence fragments. Labels, route choices, HTTP statuses, CSRF and membership rules are unchanged by this catalogue migration. The separate precision validation repair rejects malformed or out-of-range setup precision before formatting any amount example.

`TestOnboardingMessages` checks exact amounts, Unicode, literal percent signs, identifier formatting and both throttle messages. `TestOnboardingPagesCatalogue` renders all four pages and all three join modes, including malicious names, terms and return paths. The message package's race suite and the actual production web templates with the authentication/onboarding render tests passed locally. The latter used a file-list Go test invocation to exclude unrelated SQLite-backed fixtures; full handler integration and browser execution remain subject to the combined PR CI. The shared validation and later administrator, marketplace and browser-message continuations below extend this original scope; remaining product flows are still incomplete. This is additional partial UX-047 completion; it does not claim full localisation or approve other languages.

## Active-session continuation

The account-security session inventory now has a complete English catalogue flow (`sessions.*`): owner-only active sign-ins, current-session label, timestamp labels, the confirmation screen, consequences, validation and post-action messages. The page is linked from account security. Every state change retains CSRF protection, and the confirmation names the sign-in by its creation and last-active times. Existing bearer values and stored session hashes are never rendered. No new locale or timeout policy is introduced.

The bounded pagination completion reuses `pager.previous`, `pager.next` and `pager.first` inside a named navigation region. `sessions.bulk_*` covers the separate other-session confirmation and result; `sessions.help_*` explains the complete flow on `/help#sessions`. Missing confirmation uses the existing error-summary and labelled-field rendering. These additions use the existing English catalogue and do not claim translations in other locales.

`TestSessionMessages` and the catalogue completeness check cover the wording. `TestAccountSessionsRequireOwnerCSRFAndConfirmation` and `TestAccountSessionsOmitExpiredSignIns` exercise the real handlers/templates, owner scope, credential secrecy, confirmation, current-browser sign-out and expired-session filtering. Runtime verification is recorded in the session-flow review report.

## Administrator-reset continuation

The 2026-11-19 UTC continuation originally externalised 14 messages (`auth.admin_reset.*`) for the administrator password-reset confirmation and one-time result. The handler titles, action label, complete consequence text, reason label and hint, navigation, 403/429 summaries, and temporary-password explanation now resolve through the same English catalogue. The initial slice selected these messages only for `reset-password`. During composition with PR #102, the equivalent messages were consolidated under `admin.reset.*` and the shared `admin.member_action.*` / `validation.*` keys; the duplicate namespace was removed. The complete forbidden-reset explanation and all additional escaping/refusal assertions are retained.

The 403 explanation now covers every existing refusal in this handler: self-reset uses Change password, while another refusal directs the administrator to the platform operator. It no longer incorrectly describes every forbidden reset as self-reset. The service still decides authorization and returns the same status codes; no reset authority, policy, CSRF, credential, audit or forced-change behavior changes.

`TestAdministratorResetMessages` checks complete interpolations with Unicode, literal percent signs and markup-like member names, plus the refusal wording. `TestAdministratorResetPagesCatalogue` renders the real confirmation, reason-error, refusal and one-time-result states, checks the native reason label and error associations, and verifies escaping of the member name, entered reason and temporary password. It also covers an unrelated action using the shared template. After workspace recovery, the full message package and the actual administrator-reset template tests passed locally with the race detector against draft `e1ea0852`. The template invocation included every production web Go file and the existing authentication render helper while excluding unrelated database-backed test fixtures. `TestWebAdminReset` remains the handler integration gate for the actual reset, one-time password display and forced password change; full combined-source and browser qualification still require the PR checks.

The broader shared-validation, marketplace and browser-message work below supersedes the original slice boundary. The earlier `e1ea0852` focused results remain attributed to that exact source; the reconciled catalogue requires its own combined checks. UX-047 remains partial.

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
Unknown field names retain the existing underscore-to-space fallback. Status codes, authorization, audit behavior, CSRF fields and password-reset
lifetime rules are unchanged. The separately reviewed forbidden-reset explanation
now covers self-reset and other existing operator-only refusals instead of
mislabeling every forbidden reset as self-reset. Names, field details, retained form
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

## Inbox and report continuation

This continuation adds 51 entries under `inbox.*` and `report.*`. It covers
inbox headings, read actions and feedback, counts and empty states, dispute
links and their response sentence, report form labels and explanations,
category labels, duplicate/throttled summaries, success notices and the
administrator report queue. Notification titles and bodies already stored by
the service remain ordinary data; service/API messages and the unrelated
credit-limit notice in `h_notify.go` are outside this slice.

Complete accessible names for marking a notification read or a report handled
now come from catalogue-backed `aria-label` values. Their visible button labels
and spoken wording are unchanged. Inbox counts use the catalogue's English
integer formatting, including `1,000 unread.`; zero still says
`No unread notifications.`. Report privacy, the distinction between review and
action, routes, filter/category values, pagination, status codes, CSRF,
idempotency and policy are unchanged.

The inbox dispute timestamp and report form target retain their existing
markup through `messages.Around`. Its narrow extension accepts preceding
trusted catalogue labels and reserves the final string argument for exactly
one template-owned element. The report queue passes only labels selected from
the five report categories and the two known target types. Target names never
enter the helper: the template escapes them inside the existing link or strong
element. An unknown category or type uses a complete plain-text row instead,
so arbitrary stored values, including NUL and `%!`, cannot trip the helper's
sentinel or formatting-error guard. Both surrounding strings remain ordinary
text and are still escaped by `html/template`.

`TestInboxReportMessages` checks complete messages, counts, positional names
and the unchanged reporting/privacy explanations. The catalogue completeness
check supplies every new interpolation argument. `TestReportAroundUsesKnownLabelsAndOneElementSlot`
and the existing slot-rejection tests preserve the exactly-one-element rule.
The focused `inbox_report_catalog_test.go` rendering tests cover empty and
populated inboxes, read/unread state, filter/cursor links, report form values,
known category choices, open/handled queues, target/timestamp markup, CSRF and
idempotency fields, escaping and complete accessible names. The enum-provenance
regression includes NUL and `%!` in target names and unknown category/type
values. Existing `TestInboxAndBadge` and `TestReportPages` retain the HTTP and
authorization checks for the combined candidate.

Fail-before evidence was observed on base `5da8bc7`: the message regression
failed on the absent `inbox.unread_count` entry, and the isolated production-file
inbox render failed on the ungrouped large count and missing complete
`aria-label`. A runtime restart then cleared the raw temporary logs; a receipt
was retained from the observed output. Post-change checks passed with Go
1.26.11: the complete messages package with `-race -p 1 -count=1` (1.063s), and
all five new rendering/category/provenance tests with the same flags (4.137s).
The latter was a file-list run of every production web Go file plus only
`inbox_report_catalog_test.go`, without SQLite-backed test fixtures. Formatting,
catalogue JSON/duplicate-key validation and `git diff --check` also passed.
Full web-package, HTTP, browser and combined-candidate validation remain
integration work. This is a partial UX-047 continuation and introduces no new
languages.

## Trade and history continuation

This continuation adds 108 `trade.*` entries for trade lists, proposals,
proposal review, acceptance, decline/cancellation confirmation, trade detail
and account history. It includes the handlers' titles, validation text,
payment descriptions, action notices and timeline status labels. Complete
messages preserve interpolated member names, stable member references,
available-credit amounts, notes and final balance descriptions. History's
reversal and adjustment sentences use `msgAround` to retain their existing
links while keeping catalogue text subject to HTML escaping.

Both trade amount displays now use the existing per-amount unit inflector:
one credit remains singular, and a unit already named `Hours` does not gain
another `s`. Amounts still come from the domain's fixed minor-unit formatter;
ledger and member references retain their identifier formatting. Decimal-place
and proposal-expiry messages select the singular wording for one. No arithmetic,
posting rules, authorization, action eligibility, status codes, CSRF,
idempotency, routes, submitted form values or pagination behavior changes.
The historical-balance lookup fragment remains a separate, unmigrated flow.

`TestTradeMessages` and `TestTradeMessagesRemainPlainText` cover complete
messages, singular/plural forms, exact large amounts, Unicode, literal percent
signs, references and hostile interpolated text. `trade_catalog_test.go`
renders all seven production page templates. It checks both amount-unit
regressions, proposal/review form controls, both directions, confirmation
references, the administrator-only reason field, independently gated detail
actions, history links, time markup, export controls and contextual escaping.
It also checks existing status mappings/fallback and amount-help precision.

Local evidence on 2026-11-19 with Go 1.26.11: against the original production
sources, the message regression failed on the absent `trade.sentence.pay` key,
and the amount regression failed in both templates for singular, already-plural,
zero-decimal and HTML-sensitive units. Both controls compiled successfully.
After implementation, the complete message package passed with `-race -count=1`
(1.034s), and a file-list race run of every production web Go file plus
`trade_catalog_test.go` passed (11.536s). Formatting, catalogue key coverage and
unchanged pager expressions also passed. The selective render run excludes
unrelated SQLite-backed fixtures; it is not a complete web-package, HTTP,
browser or accessibility-suite qualification. Combined-candidate tests and
current-revision CI remain integration work. This is further partial UX-047
completion and adds no language scope.
## Network-picker continuation and onboarding ownership

The onboarding slice was compared against immutable active heads
`47446b33b01f4993a1caa406516746858bbf0a11` (#99) and
`baedf4b6670f6bf39ec593978ce38780c5640215` (#101), using base `03fc500`.
Both heads already catalogue the setup/invitation/join sections of `h_auth.go`
and `setup.html`, `join.html`, `join_anon.html` and `joinnet.html` under
`setup.*` and `join.*`. They also validate the setup decimal scale before
formatting amount-error examples. Those sections, their behavior and their
regressions remain with the active work; this continuation does not duplicate
or replace them. Help, session, authentication/proof, terms/version checks,
setup-secret handling, rate limits and amount parsing are untouched.

The complete `home()` function and `networks.html` were byte-identical at the
base and both compared heads. This disjoint continuation adds seven entries
under `onboarding.networks.*` for the picker title, complete empty-state
guidance, creation link and the four existing membership labels. It preserves
their lowercase English spelling, membership filtering, single-network
redirect, local links and network-name display. Unknown status values retain
the existing plain-text fallback. Names, unknown statuses and catalogue text
remain subject to `html/template` escaping.

Before implementation, `TestNetworkPickerMessages` failed on a missing
catalogue entry and `TestNetworkPickerCatalogueTextIsEscaped` failed because
the creation label did not use the catalogue. After implementation, the full
messages package passed with `-race -p 1 -count=1` (1.052s), and both new picker
rendering tests passed with those flags (2.072s). The render command used all
production web Go files plus only `network_picker_catalog_test.go`; it did not
include SQLite-backed fixtures. The tests cover empty and populated states,
all four status labels, exact links and escaping of names, unknown status
values and substituted catalogue text. Formatting, JSON/duplicate-key checks
and `git diff --check` also passed. These are local checks with Go 1.26.11,
not full HTTP/browser or combined-candidate qualification. UX-047 remains a
partial migration and this change introduces no new language scope.

## Audit, consistency-check and export continuation

This continuation adds 49 entries under `admin.audit.*`, `admin.check.*`,
`admin.exports.*` and `my_audit.*`. It covers the administrator and member audit
page titles, fixed actor labels, filter controls, table headings and responsive
labels, empty states, consistency-check explanation/results, and the export
form. Audit action names, target references, actor names, reasons and
service-generated findings/remediation remain ordinary data. No service/API
messages are translated in this slice.

The consistency summary is one complete positional message. The audit-event
total uses English count formatting and the singular `1 audit event`; the
ledger sequence is passed as an exact decimal string so it is never grouped or
converted through floating point. Audit row sequences, target IDs and amounts
inside findings are unchanged, including values above 2^53. Timestamps and
links remain template-owned markup, and every catalogue result and data value
keeps contextual HTML escaping.

Authorization, role scoping, service calls, filters, pagination, status codes,
CSRF fields, export scope/format values and download behavior are unchanged.
The policy/adjustment block and the complete ZIP/export-post section of
`h_admin2.go` were checked byte-for-byte against base `03fc500`; both are
unchanged. The existing interrupted-download handling is preserved.

The tests were committed first as `175324b`, retaining a reproducible test-only
baseline. With Go 1.26.11, the new summary message test failed before the change
on the absent `admin.check.summary` key. The production-template render run
failed on `1 audit event` and `1,000 audit events`; its audit-data/control and
export-form checks already passed. After the change, the complete message
package passed with `-race -p 1 -count=1` (1.060s), and all three new production
render tests passed with the same flags (2.984s). The latter was a file-list run
of every production web Go file plus `audit_export_catalog_test.go`; it did not
compile unrelated SQLite-backed test fixtures. Catalogue duplicate-key and
namespace checks, formatting and `git diff --check` also passed.

This is further partial UX-047 completion. Full web-package, HTTP, browser and
combined-candidate validation remain integration work. Other administrator
flows, policy, service-generated remediation wording and additional languages
remain outside this slice.

## Administrator adjustment and reversal continuation

This continuation adds 119 `admin.adjust.*` and `admin.reverse.*` entries for
adjustment entry/review, pending requests, approval confirmation and
administrative reversal entry/review. It covers the existing required-reason,
dual-control, refusal, expiry and irreversible-action notices, full payment and
reversal descriptions, titles, labels and result messages. Pending-list buttons
retain their visible labels and full spoken names through catalogue-backed
`aria-label` values. Their existing approve/reject and cancellation gates stay
unchanged.

Exact integer minor-unit values accompany the existing formatted strings only
to select the existing unit inflector. Adjustment amounts, the dual-control
threshold, original trade amounts, previously reversed amounts and remaining
amounts now inflect independently. One credit stays singular, and `Hours` does
not gain another `s`. Displayed amounts and balance calculations retain their
existing fixed-point formatting and arithmetic. Required fields, review/back
steps, stored-request approval proofs, CSRF/idempotency controls, routes,
filters, pagination and service calls are unchanged.

The pending table retains its existing status punctuation and timestamp
markup. Only the timestamp slot and related-entry link use `msgAround`;
member names, deciders and unknown status strings remain ordinary escaped
text. Known status display labels do not replace the stored status codes used
for eligibility. The policy form, export handling, member administration and
service/API error catalogues remain outside this slice.

Before editing, immutable heads for active #99 (`0a73ee4`) and #101 (`f7ac6d2`)
were compared with their #90 base (`f25d10b`). The only selected-file overlap
was #99's later export block in `h_admin2.go`; that block is unchanged here.
The adjustment presentation code lives in `h_admin2.go`, so `h_admin.go` is
also unchanged.

`TestAdminAmountMessages` and `TestAdminAmountMessagesRemainPlainText` check
exact large amounts, positional sentences, precision plurals and escaping.
`admin_amount_catalog_test.go` renders all six production templates, checking
singular/custom/already-plural units, independently inflected reversal amounts,
both adjustment directions, complete full/partial reversal consequences,
review proofs, required reasons, retained values, individual action gates,
timestamp/link markup and malicious names/status text (including NUL and `%!`).

Local evidence on 2026-11-19 with Go 1.26.11: the message control failed on the
absent `admin.adjust.amount_value` key. The compile-valid template control
restored the six original templates while retaining the additive view fields;
it failed on singular/already-plural unit rendering, independent reversal
amounts and complete accessible labels (3.340s). The original catalogue was
restored independently for the message control, and the source checkpoint was
committed before either restoration.

After implementation, the complete message package passed with
`-race -p 1 -count=1` (1.044s). A file-list race run containing all production
web Go files and only `admin_amount_catalog_test.go` passed (14.837s).
Formatting, catalogue key coverage, unchanged pager expressions and the
untouched audit/export block also passed static checks. This selective render
run excludes unrelated SQLite-backed fixtures and does not constitute full
web-package, HTTP, browser or accessibility qualification. Combined tests and
current-revision CI remain integration work. UX-047 and language scope remain
open beyond this bounded continuation.

## Account time-zone continuation

This continuation adds six `time_zone.*` messages for the complete display
paragraph, field hint, save button, validation summary/correction and saved
notice. The page title and field label reuse `account.time_zone`. The selected
files, `h_account_f2.go` and `time_zone.html`, were byte-identical between base
`0fb5304a2caeb375d1791bf583744ce2a81cc3ba` and the available immutable active
heads `0a73ee42b96255601cf2227b1ddf6599ec07a0ef` (#99) and
`f7ac6d22a1bd1718f6f4e3a840da7a28c83d0cb0` (#101).

The display paragraph retains its existing `time` element and emphasized zone
ID. `messages.AroundTwo` accepts only a catalogue key and returns three
ordinary strings around exactly two ordered element slots. It rejects missing,
repeated, reversed, unexpected or invalidly formatted markers. No user value
enters that formatter, and no catalogue output becomes trusted HTML. The
template inserts the existing timestamp and escapes the zone ID separately.

Service validation, IANA identifiers, trimming, empty-value UTC behavior,
the existing text input and 64-character limit, raw retained invalid values,
CSRF, authentication, response status codes and redirect remain unchanged.
The API and audit timestamp behavior is unchanged. No other account,
membership, governance or session flow is migrated by this slice.

Before implementation, the message check failed on the missing
`time_zone.display` entry; a compile-valid production-file rendering control
failed because the label, hint and button ignored catalogue text (0.150s).
After implementation, the complete messages package passed with
`-p 1 -count=1` (0.012s), including the two-slot negative guards. All three
`TestTimeZoneCatalogue*` tests passed with those flags in a file-list run of
the production web Go files and only `time_zone_catalog_test.go` (0.288s).
They cover UTC and Asia/Kolkata display, unchanged machine-readable UTC time,
retained invalid input, error-summary/field/hint associations, CSRF, and
escaping of catalogue text and zone values containing quotes, HTML, NUL and
`%!`. Formatting, JSON/duplicate-key checks and `git diff --check` passed.
These Go 1.26.11 checks used the existing caches and did not run the race,
SQLite-backed, full web, HTTP or browser suites. Existing `TestTimeZoneDisplay`
and combined-candidate qualification remain integration checks.

## Terms, rules and departure continuation

This bounded continuation adds 65 entries under `terms.*`, `rules.*` and
`leave.*` for the existing `h_gov.go` flow and its four templates: `terms`,
`admin_terms`, `rules` and `leave`. The selected paths match the available
immutable #99 `0a73ee4` and #101 `f7ac6d2` snapshots at base `ea63ff6`.
No service, domain or API logic changes, governance feature activation, or
policy-form migration are included.

Versions remain ungrouped decimal identifiers, including hidden acceptance
fields and successful publication/acceptance notices. Rule-change amounts and
departure balances remain exact strings from the existing fixed-point
formatters. Authored terms, reasons, blocker text and network names remain
ordinary escaped values, including retained form values. Read/accept gates,
safe return paths, membership eligibility, departure blockers, consent fields,
CSRF, status codes and redirects are unchanged. Self-export still precedes the
departure form and remains available when departure is blocked. The default
persisted cancellation reason remains unchanged; only its visible success
notice is catalogued.

Complete sentences retain timestamp and History-link markup through the
existing plain-parts helpers. Version strings passed to `Around` come only from
application decimal-integer formatting. `AroundTwoCount` adds a typed integer
count before the two existing timestamp slots and uses the same strict marker
and order guard. User-authored text never enters those marker paths. Proposal
counts now use singular/plural English instead of `proposal(s)`; a single day
uses `day's notice`. These grammar changes do not change departure or notice
policy.

`TestTermsRulesLeaveMessages` checks exact versions/amounts, plural counts,
literal names and complete feedback. `TestTermsRulesMarkupSlots` checks numeric
identifier provenance, timestamp slots, notice plurals and invalid counted
formats. The focused rendering file covers accepted/unaccepted/missing terms,
publication and retained editor values, all existing rule-log states, exact
changes, both departure balance rules, blockers, zero/one/multiple proposals,
consent/CSRF/export controls and escaping of raw authored/catalogue text.

After source checkpoint `8f5e854`, restoring only the base catalogue produced
the expected missing `terms.accepted_flash` failure. Restoring the four base
templates produced a compile-valid negative control: all four pages ignored
the injected catalogue labels and failed `TestTermsRulesLeaveCatalogueEscapesMessages`
(0.283s). The committed source was restored after each control.

With Go 1.26.11, the complete messages package passed with `-p 1 -count=1`
(0.009s). The four new rendering tests passed with the same flags in a file-list
run containing all production web Go files and only
`terms_rules_leave_catalog_test.go` (1.224s). JSON/duplicate-key validation,
unchanged existing catalogue entries, Go formatting and `git diff --check`
also passed. Raw before/after receipts are retained in this worktree's `bin/`
directory. These focused normal runs use the existing caches and exclude
race, SQLite-backed, full web, HTTP and browser suites; combined HTTP/service
qualification remains integration work.
