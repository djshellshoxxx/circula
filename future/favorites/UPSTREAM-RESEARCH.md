# FNET-202 upstream research and implementation handoff

Research date: 2026-11-16. Phase: research only. **Integration HOLD remains unchanged.** No upstream source has been copied into this module by this phase.

## Decision

Adapt **linkding's owner-scoped private saved-item CRUD and authorization regression patterns**, plus the **GOV.UK Frontend native-button and summary-list template patterns**. Implement the resulting service in Go/SQLite with Circula's network and actor boundaries. Do not import the Django application, its asynchronous URL-fetching tasks, shared-bookmark behavior, or a browser-only bookmark app. No examined upstream provides Circula's financial consent semantics; quick-repeat must remain an unpriced draft using the existing proposal/acceptance service.

The feature is small enough that a large dependency or wholesale application transplant would add more unrelated behavior than reusable code. The meaningful portable parts are query scoping, deletion/ownership test cases, and accessible form markup. If any literal or translated substantial code is adapted, retain the full applicable MIT notice in `THIRD-PARTY-NOTICES.md`, with a file-by-file origin/changes map. Do not add or choose a license for Circula itself.

## Verified candidates

GitHub repository search located these official repositories. Source, license, and metadata were inspected from shallow clones at the exact commits below, not inferred from search summaries. Activity is a maintenance signal, not a security assurance; no comprehensive advisory audit was performed in this research phase.

| Candidate | Exact inspected revision | License and observed maintenance | Disposition |
|---|---|---|---|
| [sissbruecker/linkding](https://github.com/sissbruecker/linkding) | `27b7303baf41bb28babc610ac8eaa486e1ddfab5` | MIT, copyright 2019 Sascha Ißbrücker; HEAD 2026-09-13; changelog v1.47.0 on that date; extensive Django owner-isolation tests | Selected semantic/code-port reference for owner-scoped operations and adversarial tests. No runtime dependency. |
| [alphagov/govuk-frontend](https://github.com/alphagov/govuk-frontend) | `a11bd3316363ab4702fcef81d6ce06a947c464ef` | MIT, 2017 Crown Copyright (Government Digital Service); HEAD 2026-11-12; package reports 6.6.0; component tests and current accessibility fixes | Selected markup reference, limited to native buttons and summary-list actions. Exact source snapshot, not an assertion that HEAD equals the v6.6.0 release tag. |
| [go-shiori/shiori](https://github.com/go-shiori/shiori) | `aebb4021a56048eed4c8feb325399be5f46e5470` | MIT, 2018-present Radhi Fadlillah; HEAD 2026-10-26; Go application with SQLite and database tests | Rejected as a persistence transplant: inspected bookmark schema has global URL uniqueness and no owner/network columns. Shared application semantics do not satisfy private favorite relations. |

### Exact source evidence and reusable boundaries

**Linkding**

- [LICENSE.txt](https://github.com/sissbruecker/linkding/blob/27b7303baf41bb28babc610ac8eaa486e1ddfab5/LICENSE.txt): inspected full MIT permission/copyright text; retain it with substantial adaptations.
- [CHANGELOG.md](https://github.com/sissbruecker/linkding/blob/27b7303baf41bb28babc610ac8eaa486e1ddfab5/CHANGELOG.md): v1.47.0 includes SSRF and asset-memory handling fixes. These demonstrate active maintenance but are not capabilities needed by favorites.
- [bookmarks/models.py](https://github.com/sissbruecker/linkding/blob/27b7303baf41bb28babc610ac8eaa486e1ddfab5/bookmarks/models.py): `Bookmark.owner`, owner-specific duplicate lookup, private-by-default `shared` field. Adapt ownership into mandatory `(network_id, owner_member_id, target_member_id)` keys. Do **not** bring the sharing field into Circula favorites.
- [bookmarks/services/bookmarks.py](https://github.com/sissbruecker/linkding/blob/27b7303baf41bb28babc610ac8eaa486e1ddfab5/bookmarks/services/bookmarks.py): `delete_bookmarks` and archive/update functions filter by authenticated owner in the SQL operation. Port that predicate structure into every Go query/write, adding network. Do not copy URL metadata loading, archive downloads, tags, tasks, or Django models.
- [bookmarks/queries.py](https://github.com/sissbruecker/linkding/blob/27b7303baf41bb28babc610ac8eaa486e1ddfab5/bookmarks/queries.py): `_base_bookmarks_query` applies an owner filter when a user is supplied. Circula must be stricter: a missing owner/network is an error, never an unfiltered fallback.
- [bookmarks/tests/test_bookmark_action_view.py](https://github.com/sissbruecker/linkding/blob/27b7303baf41bb28babc610ac8eaa486e1ddfab5/bookmarks/tests/test_bookmark_action_view.py): translate `test_delete_can_only_delete_own_bookmarks` and related archive/unshare ownership scenarios into Go tests; extend them to cross-network, forged owner, mixed-validity bulk/reorder input, and unchanged unrelated rows.

**GOV.UK Frontend**

- [LICENSE.txt](https://github.com/alphagov/govuk-frontend/blob/a11bd3316363ab4702fcef81d6ce06a947c464ef/LICENSE.txt): full MIT notice inspected. Do not copy government logos, branding, or other unrelated assets.
- [button/template.njk](https://github.com/alphagov/govuk-frontend/blob/a11bd3316363ab4702fcef81d6ce06a947c464ef/packages/govuk-frontend/src/govuk/components/button/template.njk): port only the native `<button>` form variant into Go `html/template`. Use explicit POST action, submit type, and accessible label. Do not port Nunjucks `safe` bypasses, arbitrary attributes, role-button links, or user-controlled trusted HTML. Double-click suppression is not idempotency.
- [summary-list/template.njk](https://github.com/alphagov/govuk-frontend/blob/a11bd3316363ab4702fcef81d6ce06a947c464ef/packages/govuk-frontend/src/govuk/components/summary-list/template.njk): reference owner-visible member/nickname/action layout. Translate accessible action context to normal escaped Go text; use existing Circula styling or an explicitly attributed isolated fragment, not the entire Sass/macro stack.
- [official button guidance](https://design-system.service.gov.uk/components/button/) and [summary-list guidance](https://design-system.service.gov.uk/components/summary-list/) are follow-up browser qualification references; exact source templates above were inspected directly.

**Shiori rejection evidence**

- [LICENSE](https://github.com/go-shiori/shiori/blob/aebb4021a56048eed4c8feb325399be5f46e5470/LICENSE).
- [SQLite initial schema](https://github.com/go-shiori/shiori/blob/aebb4021a56048eed4c8feb325399be5f46e5470/internal/database/migrations/sqlite/0001_initial.up.sql) and [SQLite database code](https://github.com/go-shiori/shiori/blob/aebb4021a56048eed4c8feb325399be5f46e5470/internal/database/sqlite.go): bookmark URL uniqueness is global; direct adoption would collapse separate owners' records. Its crawler/offline archive and account model are also outside scope.

## Existing Circula seams inspected

- `internal/service/member_page.go`: `ListDirectoryPage` authenticates through `s.open`, requires active membership, and reads active members in the same view transaction. Favorite list resolution must preserve this current-status visibility and must not expose a stored nickname to the target.
- `internal/service/transactions.go`: `ProposeTransaction` handles current party rules, idempotency, amount/limits, reservation and notification. A favorite never bypasses it. `Direction` includes both `pay` and `request`; the current isolated `Repeat` only supports historical payer-to-payee repetition. Keep that narrowed behavior explicit or add a documented direction field before supporting historical payee-side repetition.
- `internal/web/h_trades.go`: current proposal form and `renderPropose` are the later UI seam. Prefilling must leave amount blank and preserve existing CSRF, error rendering and fresh confirmation.
- `internal/service/departure.go`: departure is an atomic service command. Future cleanup must remove relations owned by or targeting the departing member in that transaction, or immediately filter unavailable targets and perform idempotent cleanup; do not delay access denial until a background job.
- `future/favorites/repeat.go`: current trusted-snapshot validator is a preflight, not live authorization or durable persistence. Nickname validation currently rejects only CR/LF/NUL; align with core text controls including invalid UTF-8, bidi/invisible controls, and length limits before exposing UI.

## Full-feature adaptation plan for the fresh implementation agent

Keep all new code, SQL, templates, tests and notices under `future/favorites/`; no main application registration or migration is authorized now.

1. **Repository and owner-private state.** Add a narrow transactional repository interface and SQLite implementation with additive module-local migration. Composite primary/unique key must include network, owner and target. Add bounded nickname, pinned/position state and timestamps; cap favorites per owner, and use stable bounded pagination. Every statement requires owner and network, including rename, delete, reorder and lookup. Never accept owner from request JSON as authority. Tests must persist/reopen a temporary DB; an in-memory map alone is not feature completion.
2. **Authorization service.** Inject a fresh membership/visibility/block policy provider and transaction boundary. Validate owner and target at save, read resolution and repeat; missing or errored policy fails closed. Do not depend on the future messaging module for blocks: inject a minimal policy interface. Owner deletion must remain possible for an obsolete/blocked target without exposing target information.
3. **Pin/unpin and reorder.** Explicit idempotent commands, atomic reorder scoped to the complete expected owner list or a documented subset; reject duplicates, foreign IDs, missing IDs, oversized inputs and stale revisions. Pin/unpin must not silently expose a social graph, reputation or target notification.
4. **Repeat drafts.** Authoritatively load the selected historical trade, verify requester is the allowed party, revalidate the target, copy only approved description/recipient fields, and leave amount unset. A distinct user intent gets a fresh key; a retry of the same intent returns the same draft/key from durable intent storage. Never copy the original transaction key. No draft action posts or accepts anything. Preserve existing service checks at eventual submission because membership/credit can change after draft creation.
5. **Presentation and contracts.** Expose explicit constructors/handlers or view-model renderers suitable for later mounting behind Circula authentication. No self-registering routes or demo auth in production constructors. Build owner-private escaped no-JS list/edit/pin/delete and repeat-confirmation forms, with POST+CSRF for writes, no-store responses, keyboard reorder alternatives and fallback directory link. Add module-scoped API contract/CLI adapter where appropriate; future root OpenAPI updates wait for integration.
6. **Cleanup and portability.** Document schema/index/retention/backup semantics; an owner or target departure invalidates visibility immediately. Test cleanup with more entries than one page, reopening DB and rollback-on-failure. Module remains optional and independent; no credit, ledger, reputation, messages, analytics or notification mutation capability belongs in its interface.
7. **Traceability.** Add exact upstream notice/provenance for adapted tests/markup; distinguish translated source from original Circula-specific logic. Update README, feature manifest and validation to describe real shipped-in-branch behavior and remaining gates; retain HOLD and disabled state.

## Required tests and honest completion boundaries

- Positive save/update/pin/unpin/reorder/remove, idempotent retries, separate new repeat intents, blank amount, full explicit confirmation.
- Same owner ID in two networks remains isolated; different owners' nicknames and relation existence never leak. Forged/mixed reorder IDs leave all rows unchanged. No nil-actor wildcard.
- Current inactive, hidden, departed or blocked target is omitted/refused immediately; failed policy lookup fails closed. Removal of one's own stale favorite still works.
- Concurrent add/reorder/draft retry, uniqueness, transaction rollback, close/reopen durability and cleanup beyond pagination limits.
- Template malicious nickname/description and URL tests, CSRF/no-store response tests, no-JS actions, account/network switching and browser back-cache privacy.
- Assert a fake ledger/proposal sink is never called by favorite operations. Actual proposal/acceptance regression belongs to later core integration.
- Run module `go test -race ./...`, `go vet ./...`, schema tests and parser/property/fuzz tests where meaningful. Existing root modernc SQLite v1.60.1 requires Go 1.26; root coordinator confirmed Go 1.26.11 is available. If introducing that dependency, pin it and keep root go.mod unchanged.

A standalone repository/service/UI adapter can be completed and tested here. Actual main-route mounting, current-core service authorization wiring, full ledger regression, browser/manual assistive-technology acceptance and release selection remain explicit integration gates, not claims made by this research report. No suitable upstream makes those Circula-specific checks unnecessary.
