# Circula: Continue Specs (Low-Cost Runner)

Paste into agent with repo open. Continues `prompts/create-foundational-specs.md` (the SOURCE; do not re-paste it, read by line range).

## State (verified)
- Repo = 1 commit, only `prompts/create-foundational-specs.md`. No README, no `docs/`, no code.
- So: 0 of 13 specs done. Job = run SOURCE pass. Specs only, no code.

## Mode
- Run `caveman:caveman` now. Caveman for ALL chat, worker prompts, worker replies, status. Drop it only for a warning user must not misread.
- Spec file bodies = normal terse technical prose (developer-ready), NOT caveman.

## Model/credit rules
- Orchestrator = Sonnet. Workers = Haiku (`model: haiku`). No Opus/Fable unless Haiku output fails review twice.
- No web search, no research agents. Repo evidence only; unknowns -> "Open decisions" list.
- Never read whole SOURCE twice. Orchestrator reads it once. Worker gets only its section's lines (`Read offset/limit`) + `_context.md`.
- Workers return: file path + <=5 lines. Never echo spec bodies. Orchestrator never re-reads a spec in full; use Grep for IDs/headings.
- Cache: keep stable prefix identical across worker prompts (role block, then `_context.md` ref, then section lines, then task). Variable part last.
- Parallel: independent workers in one message (`superpowers:dispatching-parallel-agents`). Max 4 at once.
- One review pass total (Sonnet, grep-based). No rewrite loops; fix small things with Edit.

## Steps
1. **Foundation (orchestrator, Sonnet, once).** Create `docs/specs/_context.md` (<=60 lines): MUST/SHOULD/MAY meaning, ID prefix list (PRD, ROL, DOM, FR, ARC, DAT, API, UX, SEC, GOV, OPS, QA, RSK), glossary (network, account, balance, credit limit, offer, request, transaction, dispute, adjustment, audit event), ledger invariants (equal+opposite postings, append-only, no silent mutation, idempotent, reconstructable), "credit = accounting unit, no monetary value/redemption promised", spec header template (status, owner, date from `date`, scope, deps, open questions). Every worker reads this; it is the single source of shared terms.
2. **Wave 1 (4 Haiku, parallel):** 01 vision/scope, 02 personas/journeys, 03 domain/rules, 13 assumptions/risks/decisions/roadmap. Skill hint to worker: `engineering:documentation`.
3. **Wave 2 (4 Haiku, parallel, after wave 1 exists):** 04 functional reqs (+acceptance), 05 architecture (`engineering:system-design`), 06 data/ledger/audit, 07 API.
4. **Wave 3 (4 Haiku, parallel):** 08 UX/a11y, 09 security/privacy, 10 governance/ops, 11 deploy/ops (`engineering:architecture` for decision records where options exist).
5. **Wave 4 (1 Haiku):** 12 quality/test plan (`engineering:testing-strategy`); must cite FR/DOM/DAT/API IDs from earlier docs via Grep.
6. **Index (orchestrator):** write `docs/specs/README.md` (list + 1-line purpose each). Add README pointer only if README exists/needed.
7. **Review (1 Sonnet, grep only):** (`superpowers:verification-before-completion`) check: all 13 files exist; no duplicate IDs (`grep -ho '[A-Z]\{2,4\}-[0-9]\{3\}'| sort | uniq -d`); each FR has acceptance criteria; relative links resolve; ledger rules agree across DOM/DAT/API/UX/SEC/QA; no implied monetary redemption; no legal-compliance claims. Fix with Edit. Real tool output only, no "assumed passing".
8. **Ship:** commit per wave (small msgs), push to the assigned branch, open draft PR (`superpowers:finishing-a-development-branch` only if choices unclear). Do not merge.

## Worker prompt template (copy, fill 3 fields)
```
caveman. Haiku worker. Write ONE spec: docs/specs/<NN-name>.md.
Read: docs/specs/_context.md ; prompts/create-foundational-specs.md lines <A-B> ; (deps: <files, Grep headings only>).
Rules: specs only, no code. Use _context.md terms + header template. Stable IDs <PREFIX>-001.. , each testable. MUST/SHOULD/MAY. Unknowns -> "Open decisions" w/ options+default. No legal/compliance claims, no vendor picks. Link, don't copy, other docs.
Reply: path + <=5 lines (open decisions only). No body echo.
```
Section line ranges in SOURCE: 01=40-43, 02=44-48, 03=49-55, 04=56-60, 05=61-66, 06=67-72, 07=73-77, 08=78-83, 09=84-89, 10=90-94, 11=95-99, 12=100-104, 13=105-109. Cross-doc rules: lines 110-120.

## Final response (caveman, <=15 lines)
Files made; key decisions; open decisions needing owner; contradictions/gaps; "specs only, no code, checks actually run: <list>".
