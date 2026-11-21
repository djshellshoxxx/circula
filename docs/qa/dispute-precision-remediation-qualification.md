# Dispute precision remediation qualification

Source review of `9061e7f` found a small presentation regression in the
singular dispute-amount validation message. For a network with one decimal
place, the message now correctly said `decimal place`, while `rc.fieldFix`
looked only for `decimal places`. Passing that error through `rc.refuse`
therefore selected `Check the value and try again.` instead of the existing
precision-specific correction. The direct catalogue test covered singular
wording, but the strengthened HTTP tests used a two-decimal network.

This follow-up starts at `0492d798` and changes only the remediation selector
to match `decimal place` (covering both singular and plural forms) and adds
the singular form of `validation.fix.decimals`. The plural correction text
is unchanged. Decimal parsing, valid ranges, positive-amount checks,
confirmation/proof checks, service calls, permissions and financial effects
are untouched. Missing network/context still uses the generic correction.

The test-only snapshot `0daa4d6` preserves the regression before the fix.
`TestRefusePrecisionCatalogue` calls the actual `rc.refuse` path using
`domain.Invalid` and the dispute amount message, without any database or
service fixture. It covers precision 0, 1 and 4, a missing network, and a nil
request context. The existing validation-message table gains a singular
correction expectation.

Before the fix, only the precision-1 refusal case failed (0.010s); zero, four
and both fallback cases passed. The singular correction message also failed
(0.011s). After the fix, the complete messages package passed (0.054s), and
all five actual refusal cases passed (0.012s). Both runs used Go 1.26.11,
default vet, `-p 1 -count=1`, `GOMAXPROCS=1`, `GOMEMLIMIT=250MiB` and `GOGC=40`.
The refusal invocation listed all production web Go files plus only
`refusal_precision_catalog_test.go`; it did not include SQLite-backed fixtures
or run the full web, HTTP, browser or race suites.

The raw commands and output are retained in
`docs/consolidation/evidence/continuation/precision-remediation-{messages,refuse}-{before,after}.log`.
