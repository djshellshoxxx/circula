# Shared-page review and positional message correction

Reviewed source: `f8a9d7efdd94602685c1b98bea3455ccb65e3f57` against its
`40277310094ac16b34969fc3b002b2182983d8b5` base. This follow-up starts from the
reviewed source and changes only the catalogue formats listed below. It does
not replace or reattribute the original batch's verification receipts.

## Read-only review

No behavior findings were identified in the requested escaping, HTTP status,
fallback and historical-balance boundaries. The review checked the production
diff and its focused test source; it did not repeat the original HTTP tests.

- The refusal mapper retains HTTP statuses, `Retry-After`, authentication and
  terms redirects, generic-error privacy and request references. Buffered
  rendering still falls back to a plain 500 response for missing or broken
  templates. Recovery middleware and `http.ErrAbortHandler` are unchanged.
- Catalogue strings remain ordinary text. Names, references, raw historical
  input and unknown reasons retain contextual HTML escaping; the templates
  own the existing strong, code and time elements. Markup-slot helpers receive
  catalogue keys, without inserting user text into their marker validation.
- Historical input parsing and service behavior are unchanged. All four known
  reasons are mapped in the web adapter, including submitted blank input in
  the History handler; unknown reasons remain ordinary text. Exact signed
  amounts, raw offsets, millisecond precision, future-bound checks and UTC
  normalization are retained. Both historical template loaders have the
  helpers they use.
- Support disclosures and the existing session-help block are unchanged.

## Correction scope

The review found twelve newly added messages using implicit `%s` arguments.
They now use explicit positional string arguments, consistent with the
catalogue convention. There are fourteen replaced slots in total:

| Message key | Positional slots |
|---|---:|
| `error.fallback.reference` | 1 |
| `error.reference` | 1 |
| `help.balance.credit` | 1 |
| `help.balance.debt` | 1 |
| `help.balance.level` | 1 |
| `histbal.at_utc` | 1 |
| `histbal.field_error` | 1 |
| `histbal.hint` | 1 |
| `histbal.reason.format` | 1 |
| `notice.credit_limit_used` | 3 |
| `notice.frozen` | 1 |
| `notice.membership_pending.body` | 1 |

Each one-slot format uses `%[1]s`; the credit-limit notice uses `%[1]s`,
`%[2]s` and `%[3]s` in the existing argument order. English output, argument
types and every other character of the values are unchanged. No keys were
added or removed: this isolated catalogue still contains **1151 entries**.
No unrelated historical formats, helpers, handlers or templates were edited.

## Observed verification

An in-memory comparison with the reviewed source verified the exact twelve-key
change set and that removing the explicit argument indices reproduces each
original value exactly. It also verified the unchanged complete key set.
`git diff --check` passed.

Go **1.26.11**, default vet enabled:

```text
GOMAXPROCS=1 GOMEMLIMIT=250MiB GOGC=40 \
GOCACHE=/tmp/circula-go-cache-93 \
GOMODCACHE=/tmp/circula-go-modcache-93 \
/workspace/scratch/beb70d57deba/toolchain/go/bin/go test \
  -p 1 -count=1 ./internal/web/messages

ok  github.com/djshellshoxxx/circula/internal/web/messages  0.061s
```

The complete existing messages package passed, including its formatting and
actual-template cases. Raw output is retained locally in
`bin/shared-positional-messages-after.log`. No separate web, SQLite, race,
browser or full-suite run was performed for this format-only correction.
Combined qualification remains with the integrating parent candidate.
