# Duplicate-definition qualification recovery

Date: 2026-11-19, America/Los_Angeles. Source baseline: `main@ead63b5ce24b2bcf8352d75fd0d1ff7be711f625`; the preceding inventory commit changes documentation only.

## Recovered behavior

The historical `claude/quirky-babbage-y9fpmm@af69702be08aaf14cbe9a705ddf7f08796d5e976` Python generator rejects duplicate requirement status IDs. The current Go generator already detects conflicting definitions, but its `-check` command previously checked only whether the generated report was current. Regenerating a report with duplicate IDs therefore made an invalid requirement set pass qualification.

The recovery adds a guard to the existing Go `-check` path. It uses the existing sorted `Duplicate` records and returns a nonzero exit status naming the conflicting IDs. Normal report generation still succeeds and includes diagnostic locations, so maintainers can inspect and repair the source. The parser, source precedence, ordinary references, and consecutive rule-row semantics are unchanged. No historical Python generator, obsolete workflow, or bytecode cache is imported.

## Verification

| Check | Observed result |
|---|---|
| New CLI regressions against the unmodified Go implementation | Failed in both duplicate cases: a freshly generated duplicate report was accepted as up to date. [Fail-before log](traceability-duplicates-before.log). |
| `go test -p 1 -race -count=1 ./scripts/traceability` after the guard | Passed, 7.743 seconds. [Race log](traceability-duplicates-after.log). |
| `go vet -p 1 ./scripts/traceability` | Passed. |
| `go run -p 1 ./scripts/traceability -check` on the actual repository | Passed; the existing inventory is current and has no duplicate definitions. |
| `gofmt` and `git diff --check` | Passed for the changed Go source and tests. |

`TestCheckRejectsRegeneratedDuplicateStatusIDs` invokes the real command in a subprocess, covering duplicates within one status table and across separate specification files. It requires a nonzero exit, a useful ID-bearing diagnostic, and an unchanged report. `TestCheckAllowsSharedIDsAndConsecutiveRuleRows` confirms that a status row plus its requirement heading, a cross-file reference, and consecutive UX rule rows remain valid. The existing package tests continue to cover duplicate parsing and report generation.

The original executor was lost before the first attempted test run returned. Those interrupted results are not used here. The logs above come from the recovered workspace and completed commands. These checks qualify this tooling change; they do not substitute for the complete integration candidate's CI, manual accessibility, external security review, or release requirements.
