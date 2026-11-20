# Standalone validation — 2026-11-16

Go 1.26.11 linux/amd64. Module tests explicitly run from future/governance; no root Circula build/test or ledger-release claim is made.

```
GOGC=20 GOMEMLIMIT=450MiB GOMAXPROCS=1 go test -p 1 -race -cover -timeout 90s ./...
PASS; coverage 67.5% of statements

go vet ./...
PASS, no diagnostics

go mod verify
all modules verified

govulncheck ./...
No vulnerabilities found.
```

`GOFLAGS=-buildvcs=false govulncheck -test ./...` also passed: `No vulnerabilities found.` The first test-inclusive attempt failed only while loading generated test-main VCS metadata; disabling build VCS stamping allowed the actual test-import scan to complete. This scanner result is not an independent security audit or a promise that no vulnerabilities exist.

Tests cover persistent lifecycle/current membership/frozen eligibility, duplicate/changed-intent/concurrent votes, deterministic abstain and empty outcomes, wrong-network/disabled/suspended/new-member denials, protected policy-link separation, moderation/dispute/reviewer conflict, audit-failure rollback, process restart, schema-version rollback, actual SQLite VACUUM backup restore, escaped native ballot labels/no default selection, cookie-CSRF interface/header checks, body bounds, PRG, account-bound cursors and shared CLI limits. See ACCEPTANCE.md for mapping.

The current implementation is an isolated candidate awaiting fresh independent source/security review. The test principal/electorate/policy resolver is explicitly synthetic, not a production authorization adapter. Browser automation/manual keyboard/screen-reader tests were not performed here. Real core-store transaction adaptation, API/ledger regressions, translations, deployment/retention approval and owner-selected rollout remain pending. Coverage is not proof of feature completion or security. Integration remains HOLD; no push, merge or activation occurred.

## Fresh independent audit continuation

Reviewed initial implementation `6eed7eb`; findings, scope and requirement-to-evidence mapping are in AUDIT.md. No root application file or other future module changed.

Final frozen-source checks with Go 1.26.11 linux/amd64:

```
GOGC=20 GOMEMLIMIT=450MiB GOMAXPROCS=1 go test -p 1 -race -cover -timeout 90s ./...
ok github.com/djshellshoxxx/circula/future/governance 6.923s coverage: 70.4% of statements

go vet ./...
PASS

go mod verify
all modules verified

govulncheck ./...
No vulnerabilities found.
```

Parser-only fuzz run `go test -run=^$ -fuzz=FuzzAuditStrictJSON -fuzztime=3s -parallel=1 strict.go strict_fuzz_test.go` passed 18,890 executions. It tests bounded duplicate/Unicode/depth parsing without compiling unrelated SQLite fuzz instrumentation.

Added regressions for ignored ballot/result/audit/intent writes and transaction rollback, incompatible indexes/constraint literals, corrupt tenant/state/frozen-roster metadata, copied provider electorate, revoked administrator replay, retained proposal churn, saturated-intent safety withdrawal, duplicate unresolved concerns, deadline immutability, hostile form paths, duplicate/invalid-Unicode inputs, and valid maximum escaped stored content. Existing concurrency, restart, VACUUM restore and migration rollback evidence is reproduced by the full suite.

Actual Circula transactional auth/admin-MFA/store/policy-history compatibility, full core/API/ledger regression, real browser/accessibility/screen-reader/translation checks, deployment/retention policy and owner-selected single-feature integration remain pending. No browser or full product completion is inferred from these standalone checks. Integration remains HOLD.

## Master continuation — 2026-11-16

Recovered predecessor: `e4381aa19292f1d4b24f09e1de51f3e2e5a2930b`. Final tested implementation: `8b74e7db0e237df15acb46a27609f6015eadb4ed`. Toolchain Go 1.26.11, shared build cache, `GOGC=20 GOMAXPROCS=1 GOMEMLIMIT=450MiB`, all checks serialized.

- `go test -p 1 -count=1 -run '^TestMaster' ./...`: PASS, 0.126s.
- `go test -p 1 -race -cover -count=1 -timeout 90s ./...`: PASS, 5.824s, 75.3% statements.
- `go vet -p 1 ./...`: PASS, exit 0.
- `go mod verify`: PASS, all modules verified.
- `go test -run=^$ -fuzz=FuzzAuditStrictJSON -fuzztime=3s -parallel=1 strict.go strict_fuzz_test.go`: PASS, 1,742 executions. This ran at correction `86624d5`; both explicitly selected parser files are unchanged at the final tested implementation. It is a bounded parser-only run, not a whole-package fuzz claim.
- Static checks: `git diff --check`, HOLD/disabled JSON assertions, 15 enumerated imported dependency notice paths all passed. Dependency versions/checksums are unchanged.

The first focused run exposed an ignored private reload failure on the form-error path and a mistaken expected status in the unchanged conflict control. Correction `86624d5` fails closed on reload denial and preserves the existing 409 conflict response when authority remains valid. An intermediate full suite passed 74.4%; it is superseded by the final result above.

A fresh receipts auditor prompted a shared parser review. Regression `TestMasterHostCannotHideMalformedForm` reproduced malformed body acceptance when a host CSRF lookup called PostFormValue: status303, one host call, one persisted proposal. Correction `8b74e7d` parses and validates the bounded form before that callback. The final suite rejects malformed/duplicate/unknown fields before host or mutation, while native valid forms remain functional.

Other new regressions cover strict query parsing before auth, cursor validation before mutation, same-URL native form compatibility, URL/body proposal conflict without mutation, single empty cursor first pages, and revoked/replaced/member-disabled/canceled callback decisions across detail/list/error rendering. No browser, assistive-technology, core ledger, production auth/MFA, live deployment or new network advisory scan ran in this continuation. Earlier advisory results retain their original source/date scope.
