# UX-047 member role lists — 2026-11-28 Pacific

Fresh base: `3faeed76c41467a75b09bc75e7185a98a92f8219`.
Branch: `fix/ux047-semantic-member-roles-20261128`.
Regression-only ancestor: `10be0aa232b03d4ad1758d90b8cfdb3684ca2832`.

The regression ancestor adds only an actual-template test. Its prospective
list-valued view contract fails against the old template; this is not normal
handler/service/HTTP execution or proof of wrong current-English wording.

`newMemberRow` previously joined translated roles with a hard-coded comma and
space. The repair retains a label slice and renders semantic `ul`/`li` elements,
removing locale-dependent list punctuation without new catalogue entries.
Order, empty-role `member` fallback and contextual escaping are preserved.
Scoped logical padding removes default list margins. No shared catalogue, inbox,
shell, session or account path changes. #142/#143 owner merge gates remain.
Domain roles, state/actions, routes, authorization, API, ledger, storage and audit
are unchanged. This is not new-locale or whole UX-047/024 acceptance.

## Executed locally

Official checksum-verified native Go 1.26.11:

- Focused regression-only run: exit 1; empty-role control passed; one/two/hostile
  role subtests plus parent failed (four fail events).
- Corrected `go test -json -count=1 -race ./internal/web/messages`: passed,
  362 named passes, zero failures/skips.
- `go vet ./internal/web/messages`, gofmt on all three changed Go files and
  `git diff --check` passed.

The new fixture renders actual table content using real catalogue/html-template
code, with prospective view data and pagination stubbed. It does not execute the
real row builder, service, auth, HTTP or shell. Existing full-web state/action,
filter/paging and hostile-role checks remain, with exact slice/list expectations
replacing only their two obsolete flat-string expectations. Full web was NOT
run locally; native CI must compile/execute it. No local browser, responsive,
keyboard, screen-reader or new full-Core qualification is claimed. Earlier killed
SQLite builds were not repeated.

Independent design and implemented-source review found no actionable blockers;
reviewer ran no tests. A disk-full event interrupted their review-file append;
complete findings were delivered separately and preserved with the evidence.
No product edit was made by the reviewer. Verify publication readbacks.

## Continue this same slice

Inspect narrow draft exact-head native Core CI; fix actual failures without
weakening assertions. Full Linux race/web, browser and all applicable jobs remain
required. After any authorized normal merge, actual-main native checks remain
separate. At acceptance update handoff/traceability with exact source, PR, runs,
results and limits, then stop at the bounded boundary. HANDOFF/TODO are not edited
in this source because they overlap the excluded draft-owned paths.

Do not select another feature or repeat #142/#143 qualification, the prior plural
scan or listing-validation proof. Shared en.json remains excluded; listing-detail
repair still needs coordination. Held #83/#85/#86/#94/#107, SEC-026/GOV-021 and
all fifteen future modules remain untouched. No licence/deploy/release/force-push,
history rewrite or branch deletion. Retain six existing Core skips and manual,
external, locale and bounded fuzz/load/repro/packaging limits.

Evidence archive `evidence/UX047-member-role-lists-2026-11-28.tar.gz`: 26810 bytes, SHA256 `a87830a0c7fb700e0a923a84b93826c888efbbda84644c70f723478817829f21`; four payloads plus digest manifest. The original design review includes its interrupted append; the complete delivered source conclusion is preserved separately. This is not a CI artifact.
