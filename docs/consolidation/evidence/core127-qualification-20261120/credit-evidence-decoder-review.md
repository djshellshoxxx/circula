# Core #127 independent review and credit-evidence decoder repair

## Source and scope

Reviewed candidate `21933a3c650690db5a8e1e44cc236d8d1b5cdb24` against merged
main `aa01563fcc891da60fefebb8335eca27855678e8`. The review focused on the new
migration integrity gate, immutable historical credit evidence and its coverage
consumers, and transaction/member/listing report privacy and explicit dispute
context. It used a detached checkout, the accepted contracts, and earlier review
receipts. The parent owns combined-source qualification and provider CI.

The bounded repair is on `fix/core-credit-evidence-null-20261120`, based directly
on the reviewed candidate. The two source hashes below identify the tested
implementation without depending on the eventual integration commit hash.

## Finding and repair

**Low: historical policy null validation did not match the decoder's field-name
matching.** At the reviewed candidate,
`internal/store/sqlite_credit_limits.go:90` inspected only the exact lowercase
`default_credit_limit` key. The following struct decode at line 95 recognizes
case-equivalent field names, while JSON null leaves a primitive integer's
existing value unchanged. A noncanonical explicit-null field could therefore
retain the historical default and be treated as usable policy evidence.

This is a narrow extension of the earlier policy-object repair. Ordinary policy
writes already produce validated canonical JSON. The finding concerns the
historical evidence decoder's treatment of damaged or imported policy data;
the review did not establish a remote policy-history mutation path.

The repair applies the existing explicit-null check to every case-equivalent
field name using `strings.EqualFold`, consistent with the subsequent policy
decoder. It retains legitimate omitted historical defaults and valid numeric
aliases. Posting decisions, immutable storage guards, evidence coverage rules,
and other policy readers are unchanged. Six ordinary post-fix acceptance cases
cover uppercase and mixed-case null names, a numeric canonical field alongside
a null alias in either order, and valid numeric aliases. The tests use constant
SELECT rows and do not alter ledger history or disable database protections.

## Other reviewed boundaries

No additional concrete blocking source defect was identified within these
reviewed boundaries:

- Migration admission uses the same immediate transaction for the precheck,
  migration, and postcheck. Read/check errors and newly introduced findings
  refuse the migration before the version record is committed.
- Posting credit evidence is inserted in the same write unit as the ledger
  effects. Historical reconstruction uses exact integer arithmetic; legacy,
  system, checked, and unusable evidence remain distinct. Coverage does not
  replace the invariant result, and unknown older coverage remains unknown.
- Report-subject restrictions are applied to queue queries, direct service
  access and handling, audit/export projections, and HTTP replay. Reviewer
  projection stays limited to the accepted three-field case summary and uses
  current authority and independence checks.
- Explicit member/listing dispute context is validated in the write unit,
  immutable, included in idempotency semantics, and constrained to the same
  network. Existing policy holds for system-account report subjects and
  all-conflicted escalation remain unresolved product decisions.

These are scoped source-review conclusions, not exhaustive security approval
or a substitute for combined-source tests and required repository checks.

## Executed qualification

Executed after the repair on 2026-11-20 at approximately 08:08 Pacific with
`go version go1.26.11 linux/amd64`:

```text
go test -race -count=1 -p 1 -timeout 3m -json -run '^TestHistoricalCreditEvidenceDecoder' ./internal/store
```

The run used the parent-provided Go runtime and caches, with `GOTOOLCHAIN=local`,
`GOMAXPROCS=1`, `GOGC=20`, and `GOMEMLIMIT=512MiB`. Observed exit status: **0**.
Both top-level tests and all **28 subtests** passed, with **0 failures**, **0
skips**, and no race report. Package output reported **1.126 seconds**. Existing
legacy-marker, missing-policy, and omitted-default controls passed. The
architecture-conditional 32-bit integer case was not selected on amd64.

`gofmt` completed successfully and `git diff --check` passed. This worker did
not run the full suite, fuzzing, provider CI, or any pre-repair reproduction.

| Tested file | SHA-256 |
|---|---|
| `internal/store/sqlite_credit_limits.go` | `aa74f349544b6898c0b92cb8ae9a468045bd328e03398e146ebffd41496315b4` |
| `internal/store/credit_limit_decode_test.go` | `a7be1f244ed7ce1b620c504f7a83f62f6d958084b61066f942bf02196fa0f05e` |
