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
