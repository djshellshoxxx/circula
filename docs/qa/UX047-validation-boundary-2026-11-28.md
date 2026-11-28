# UX-047 listing validation-detail boundary — 2026-11-28 Pacific

## Confirmed gap and exact source

Fresh Git fetch and connected GitHub reads agreed on main
`3faeed76c41467a75b09bc75e7185a98a92f8219`. The isolated worktree was clean.
No AGENTS.md was found in this sparse Core checkout. Current HANDOFF/TODO,
UX-046/047 specification, decisions, prior investigation and current owner
messages were read. Historical missing-flow lists were not treated as current gaps.

The listing form uses `novalidate`. Blank/whitespace or over-120-rune titles cause
`internal/service/listings.go:validateListing` to return the application-authored
detail `must be 1 to 120 characters`. The normal listing-create source path sends
that error through `formFail`, `rc.refuse` and `fieldDetail` in
`internal/web/refusal.go`. Title is not a fixed-choice field, so `fieldDetail`
returns the English detail unchanged inside `validation.field.message`.
There is no matching complete detail entry in the current English catalogue.

This is a structural externalization gap under UX-046/047: a translated wrapper
and correction cannot translate the inner application-authored detail. Current
English wording, domain validation, authorization and API behavior are not alleged
to be incorrect. Reachability is established by source tracing, not HTTP execution.

## Executed failing proof

Official native Go 1.26.11 built a small AST extractor and ran:

```
go test -json -count=1 extracted.go probe_test.go
```

Exit **1**: **two tests passed, one failed, zero skipped**. Existing choice errors
called the synthetic catalogue printer. Unknown arbitrary text remained literal
and escaped under html/template. The known title detail failed the proposed
catalogue-boundary contract, returning `must be 1 to 120 characters` with no
catalogue call.

The extractor takes the actual `choiceFixes`, `machineChoices` and `fieldDetail`
declarations from the pinned production AST. Its only semantic substitutions are
`messages.Text` to an instrumented printer and `domain.FieldError` to the same-shaped
local type. Map-initialization calls are reset before each test. No supported
translation is selected. This is an isolated pure-function regression probe, not
compilation/execution of the production web package or actual messages printer.

Original source SHA256:
`0ad664cc1ec1fae397778faa1c096659e5d1d96b46abbbba0bc41193d681c7fc`.
Original JSONL SHA256:
`5488ce62bdf524e2099e4d9f53c49ff37972b98bda8dfa3babf89049be2a92b9`.

An initial `go run` command rejected source-path arguments before extraction or
tests. Building the extractor executable corrected that invocation. No SQLite
compile loop, Go web/race suite, HTTP/browser suite, CI run or green repair is
claimed. The independent investigator reviewed producer reachability and then
the exact generator, extracted code, tests and original JSONL, approving this
bounded finding without rerunning it. Their review is archived separately.

## Ownership gate and implementation handoff

The repair needs a new complete message in shared `internal/web/messages/en.json`,
an explicitly excluded #142-owned path. No product file was changed; no alternate
catalogue loader was introduced to bypass that restriction. Shared-catalogue
ownership coordination is needed before implementing this slice.

After coordination, map only the exact known field/detail combination to a complete
catalogue message, preserving the current English text unless deliberately reviewed.
Do not broadly translate arbitrary details or change domain/API/audit values.
Keep field association, summary, correction and contextual escaping. Carry the
isolated red contract forward, add real service/HTTP regressions for create/edit,
and retain unknown-detail and escaping controls. Then run focused native checks,
required exact-head CI and independent review. No implementation acceptance exists
yet, and this probe does not replace those gates.

On the fresh provider read, #142 remains open/draft at
`78e254f4f1b8283c124274e0a6a1ba2b82959a0f` and #143 open/draft at
`bfbfb79020942394457966c1e72b329e65bb06dd`, both based on main above.
Their separate owner merge authorizations remain absent. No ready/merge action,
requalification or status-only CI loop was performed. Their original qualification
and runtime limits remain in their published provider records.

This run stops at the written ownership-gate handoff for this bounded investigation.
Next fresh run must not repeat this proof or the completed plural scan. Check owner
messages first. If coordination is absent, assess one different independent UX-047
boundary outside the owned paths; do not switch to held policy features or infer
that all unblocked Core engineering is exhausted.

UX-047 remains partial. Preserve #142/#143 integration gates, six existing skips,
manual screen-reader/external-review/locale and bounded fuzz/load/repro/packaging
limits. Held #83/#85/#86/#94/#107, SEC-026/GOV-021 and all fifteen future modules
remain untouched. No licence, deployment, release, force-push, history rewrite or
branch deletion occurred.

## Preserved evidence

Companion archive: `evidence/UX047-validation-boundary-2026-11-28.tar.gz`.
Seven payloads plus SHA256SUMS; 8495 bytes; SHA256 `90369b542d112b02e5060b205f2e94add6d4645d90546e90267f1a23c7826dc3`.
Archive contains the exact extractor, extracted source, probe, original source,
original JSONL, reproducibility instructions and independent review. No compiled
binary is included. Report and archive are documentation/evidence only, not a
product repair or a passing CI branch.
