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
