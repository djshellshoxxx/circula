# UX-047 platform-suspension inbox wording

Source checkpoint: 2026-11-26 Pacific, based on
`main@3faeed76c41467a75b09bc75e7185a98a92f8219`, on
`fix/ux047-platform-notice-categories-20261126`. UX-047 remains **partial**.
This record describes local source qualification; provider CI and any later
main integration have their own results on the resulting pull request.

## Defect and behavior

`PlatformSuspendNetwork` stores category values such as `security_incident`
inside its notification body. The administrator network-status page already
uses the readable English catalogue labels, but the inbox previously printed
the internal value. The new renderer uses the same four labels: Legal order,
Security incident, Platform abuse and Hosting agreement.

Recognition requires the exact existing notification kind, platform title,
known category prefix and complete producer suffix, an empty link, no event
date, and a nonempty already-trimmed reason. The extracted reason is passed
unchanged to a complete catalogue sentence, with ordinary template escaping.
It may contain category codes, repeated envelope text, Unicode, percent signs
or markup-like text without being reinterpreted. Older notices retain their
own categories after later suspension changes because no live suspension is
consulted. Unrecognized, changed and erased records retain their stored text.

The change adds two catalogue keys and preserves all 1,482 previous entries.
There is no migration, service/API/storage change, new language, authorization
change or policy decision. The deliberate compatibility boundary is one
existing stored English format; a future producer format falls back intact.

## Regression and verification

The regression-only source is preserved in
[`35c62199`](https://github.com/djshellshoxxx/circula/commit/35c62199cb9b6c88c6ecc3b8779911926498e930).
Before the product fix, the real inbox test failed the human-readable label
expectations for all four categories; its API/storage assertions and all 13
unrecognized-record cases passed. The new catalogue test failed on the absent
message key. The same regression file then passed unchanged after the fix.
Its SHA-256 is
`4d46a14f3d717c665ddfbbacecb7fa47c6fe17a5f01d5f43d2bdfa13a5ccd00c`.

The two integration tests use real service producers, authentication,
production HTTP middleware, SQLite storage, templates and the JSON API through
`httptest.ResponseRecorder`. They require no TCP listener. They cover four
suspension categories, replacement and resumption, administrator pauses, exact
escaped reasons, unchanged stored records and API bytes, plus 13 fallback
forms including redaction, unexpected links and future event context.

| Local check, Go 1.26.11 | Result |
| --- | --- |
| Focused message/inbox regression selection | 6 top-level tests, 44 named pass events including subtests; no failures or skips |
| Same selection with race detector | 44 named pass events; no failures or skips |
| Complete message package with race detector | 49 top-level tests, 359 named pass events; no failures or skips |
| `go vet ./...` | Passed |
| `go build -o bin/circula ./cmd/circula` | Passed with normal VCS stamping |
| Catalogue uniqueness and preservation | Exactly two new keys; every previous entry unchanged |
| Repository hygiene and whitespace | Passed |

The focused selection was:

```sh
go test -race -count=1 -timeout=5m \
  -run '^(TestPlatformSuspensionInbox.*|TestInboxCatalogueCountsAndActions|TestReportCatalogueCategoryValues|TestInboxReportMessages|TestEveryCatalogueEntryFormats)$' \
  ./internal/web/messages ./internal/web
```

An independent source reviewer found no actionable blockers and verified the
recognition boundaries, escaping, preserved evidence/API behavior and test
coverage. This is an engineering review, not the independent external security
acceptance gate.

The local runtime denies socket creation, so installed Chromium could not
start and the complete HTTP/browser suites were not run locally. The existing
unmodified native CI checks must supply those results. No assertion, workflow
or protection was weakened. The five policy-held PRs, all future modules and
manual/external/release gates remain unchanged. Main merge still requires
separate owner authorization.
