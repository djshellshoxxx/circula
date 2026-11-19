# Four divergent future modules: reconciliation assessment

Observed 2026-11-19. Scope: favorites (FNET-202), QR (FUX-102), discovery (FUX-103), and onboarding (FUX-106).

## Disposition

Preserve the current implementation as the integration base for all four modules. Each current target contains substantial service, storage, presentation and test work absent from its archived alternative. Their old README/feature/VALIDATION descriptions still describe earlier prototypes and cannot be used as an inventory of current code.

The verified archive also contains useful implementation and regression work. Divergence alone does not prevent adaptation. The first concrete adaptation is the QR camera cancellation repair, frozen at 2ee089e36c9f2036c797d2be5bcce5a0650f78cb with 23 passing synthetic lifecycle cases. Independent review subsequently demonstrated an additional oversized replacement-image case; a separate correction and final qualification remain pending. Do not treat the first 23-case green as final acceptance.

All four modules remain disabled/HOLD. No recovered alternative is selected as the authoritative replacement, no root integration is performed, and historical or separately executed snapshot tests do not qualify a new composition.

## Exact identities

| Module | Current target head | Current whole tree | Current module tree | Archive head |
|---|---|---|---|---|
| Favorites | efc9d696eea2dd3260ecb6c5091802ef31e20572 | c98d6aa875435577bd6ce9a25e899fe217ee5e79 | ccb4a304384585caf409e329206936af0a270ea0 | 47cef5958b7fcdd5d3cbfb5878d74b38b327ce68 |
| QR | 9db8a323d7dcf430d1a58a4a4d10ca3628aeb401 | 8bd3334467754c571a5a35ffb182d4f065ae1610 | dadd80b3321d2ac663990d1378251305c9e65e9e | 08a09b2bb196403b26e87de32e3b0ac167cfa75e |
| Discovery | 34d0f889631e2d133149a89a276cf9c71fd261a9 | c095e71e22d3ef10b3ab15f1d6feb5ef103063b1 | 85d431619d438ea6cf7d3c3eebb6b5111a2dd2ad | 204dc2691f3a8c914ff2623b714f96e6d8b82968 |
| Onboarding | 473cab2fb45a7b67ea0001f9fdc7833ffda45eb9 | bdf730b32e8f11308c571f6a3b56ff44ce6331d8 | 688f2dc061be04ef17c3f654a0fcca1a346bcb9c | d91c66e8f4285944f2c780ea21801ca819c3b5eb |

The canonical verified archive is evidence/branch-inventory/legacy-recovery/recovered.git. The full object/path comparison is module-comparisons.json; the supplementary symbol inventory is divergent-four-source-map.json. The current objects are read through the shared Circula object directory as an alternate, without changing archive refs.

## Ownership and qualification

A fresh survey read all changed-file pages for 32 open PRs, then the two newly opened recovery PRs #117/#118. None of the 34 observed PRs touched these four module directories. All 134 remote branches were listed; the only extra favorites branch, chatgpt/favorites-owner-scoped-collection-20261116 at 372a8f97ae01c0af3b4a96c4a5ce8b1bcc75cd8d, is already an ancestor of the current favorites target. Root explicitly designated these four streams unreserved. This is a dated ownership observation, not proof that no unpublished external work exists; recheck before further edits.

MODULE_QUALIFICATION.json records ordinary race/vet and separately tagged sqlite_cgo_test race/vet passes for the four exact current target heads. The ordinary run alone omits substantial opt-in SQLite coverage in these targets. The archive receipts at evidence/recovered-qualification/{module}-receipt.json record successful latest required checks on the four exact archive heads and unchanged source. Earlier failed/interrupted archive attempts remain in those receipts. These results qualify their separate tested snapshots only.

During this comparison no Go build or test was launched. The QR repair used Node v24.22.3 on Linux, with synthetic DOM/media promises; two cases additionally executed the unchanged bundled Nimiq class. This is not real browser, camera, CSP, mobile or accessibility qualification.

## Reconciliation matrix

| Module | Useful archived work absent from current implementation | Current work that replacement would remove | Safe continuation | Actual contract decisions |
|---|---|---|---|---|
| Favorites | Owner export/confirmed erasure; private-payload scrubbing with retained replay tombstones; MAC-protected cursors bound to owner/network/visible snapshot; final callback, persisted-result and expiry checks; strict form/JSON audit cases. | Store/Tx abstraction and memory/SQLite conformance; explicit Enabled gate; favorite-based drafts and draft lookup; current CLI and complete web/API adapters; configurable limits; display names and creation timestamps. | Adapt strict decoder, callback/expiry and rollback tests to existing methods; repair only demonstrated failures. Owner export can be added using current storage contracts rather than replacing Service. | SQL schema and Policy signatures; current partial-prefix reorder versus complete visible-list reorder; cursor format; 200 default favorites versus fixed 500; 100 live drafts versus 1000 including tombstones; expiration/reuse versus retained replay identity. |
| QR | Explicit alias disable without replacement; deterministic secret-bound issuance retries; bounded session-private confirmations; strict body and fresh-session callback audits; generation-aware scanner lifecycle. | Transaction-scoped Core API, explicit network opt-in, request lifecycle/status/list operations, same-write-unit proposal fixture and acceptance ledger tests, CLI, OpenAPI fragment, current encoder/Nimiq distributions, media/print browser harness. | Camera cancellation is demonstrated and being repaired against current source. Adapt callback and strict-input cases next without switching vendors, schema or navigation. | Recoverable issuance-token/secret lifecycle; schema/routes; 20 default active requests versus 1000; 7-day versus 30-day maximum TTL; note/amount limits; renderer/scanner selection. |
| Discovery | Atomic bounded private JSON catalog and failure/restore tests; strict body/form/query validation; current-session callback audit cases; visible-result fingerprint pagination; alias-copy, ownership, retention and malformed-state regression scenarios. | SQLite listing projection and taxonomy/search tables; separate blob storage, pending upload activation and garbage collection; rotating-key signed cursors; SQL search and coarse-area distance filtering; CLI/API; JPEG/PNG container checks, rescaling and media quotas. | Adapt parser/trailing-body, callback and media/control tests to current Repository/Service/web boundaries. Keep current SQL/blob architecture. | Archive acts as a catalog owner while current SyncListing mirrors Core-owned listing state; JSON versus SQL/blob persistence; snapshot continuation semantics; field/enum mappings; output image representation and quotas; authority for area IDs. |
| Onboarding | Private JSON checkpoint/process-crash recovery; deletion generations rejecting stale learning/consent forms; dynamic validated LTR/RTL catalog; strict forms/JSON and fresh-auth callback scenarios; local practice/operator CLI. | SQLStore and MemStore contracts; existing English/French catalogs; complete bounded practice World and session server; preview/proposal/accept/cancel controls; JSON/CSV exports; current HTTP/API and anonymous per-step counters. | Adapt boundary tests without replacing stores or practice. Investigate the current SQL retention timestamp comparison with an actual current-store regression. Add extensible catalog/CLI features around current interfaces if selected. | Progress/revision schema and deletion replay semantics; per-step cumulative counters versus participant/completion aggregates and complementary suppression; JSON versus SQL persistence; dynamic catalog API; practice-session persistence and HTTPS/origin deployment contract. |

### Favorites: concrete source and test mapping

Current collection and drafts already persist through Store.Update/View, with implementations in memstore/memstore.go and sqlstore/. Service.Config requires Policy, Store and the explicit per-owner/network Enabled callback. draft.go provides RepeatTrade, DraftFromFavorite and GetDraft. The CLI, no-JS forms and JSON routes have their own lifecycle/isolation tests. Removing these files to use the archive would reduce existing functionality.

Archive privacy.go adds Export and Erase. It exports currently authorized data without proposal/request keys and preserves bounded intent identity when erasing private payloads. Archive paging.go signs a visible-list fingerprint with owner/network scope. Audit cases include TestAuditLatePolicyAndCancellation, TestAuditDraftPersistenceMutationRollsBack, TestAuditExpiryDuringLatePolicyCallback, TestFormAuditCallbacksRequireCurrentSession and TestOwnerErasureRollback. These are reusable requirements/test scenarios, not source-compatible drop-ins.

Current request keys deliberately become reusable after their configured TTL and cleanup deletes expired intents. Archive Cleanup retains expired/scrubbed tombstones and Draft returns ErrExpired for the old key. Current Reorder accepts a unique prefix; archive Reorder expects the complete visible set. Neither behavior should be silently replaced while adapting a parser or callback fix. The replay retention, cursor migration and reorder decisions are genuine API/data-contract decisions. They do not block strict input or callback regression work.

### QR: concrete adaptation and remaining archive value

Current service.go already implements creation, preview, alias resolution/rotation, atomic ClaimRequest/ProposeToAlias, revocation, status and request pagination through a transaction-scoped Core interface. The qrfixture package tests proposed versus completed state and balanced synthetic ledger effects; it is not the production Core adapter. The archive's generic Authorize/Propose callbacks do not replace this contract without losing current tests and interfaces.

Current creation returns the one-time raw token on initial success; an idempotent replay returns the stored view without recovering that raw token. Archive issuance.go derives a reproducible token from an external secret, stable intent and payload fingerprint. Adopting that capability requires an explicit secret/compatibility design for previously generated tokens; it is not equivalent to a parser bug fix. The archive also has RevokeAlias separate from rotation. A corresponding local service operation could be added without changing scanner vendors, but its API and replay semantics must be reviewed.

The current camera's cancellation defect was reproduced against unchanged source. Four simple deferred-promise comparisons showed the archive correctly cancelling late media acquisition/results while current scan.js did not. The first adapted candidate at 2ee089e adds attempt checks, immediate media release and per-attempt video ownership. It preserves the current automatic navigation to the server preview; the archive's fill-only interaction was not copied. The formal original suite failed 21/23 before repair and passed 23/23 afterwards. Independent review then reproduced valid image A followed by oversized image B: A still completed because B's size rejection preceded cancellation. That follow-up is tracked separately in evidence/qr-camera-independent-review/invalid-replacement-red.json and must be corrected before final signoff.

### Discovery: preserve the projection and media architecture

The current repository is a Core listing read model: SyncListing is a trusted host seam, while normal clients edit discovery metadata. Its SQL store supports snapshot/keyset search, rotating signed cursors, categories/areas/tags and idempotency. Media uses reserved metadata, separate blobs, activation, integrity verification, retention and cleanup, with explicit upload/decode budgets. Current tests already cover restart, concurrent quotas, discarded upload recovery, stale/expired visibility, owner/network scoping, cursor rotation and image metadata stripping.

The archive instead serializes an authoritative bounded Catalog to a private JSON file and embeds clean media in that state. Its failure/restore tests are useful, but copying catalog.go would remove the SQL/BlobStore design rather than supply a missing durable implementation. SearchPage binds continuation to the authorized visible projection; current cursors use a stored sequence snapshot and position. Different continuation behavior needs API review.

Specific portable scenarios include TestAuditStrictHTTPBodyAndMultipartCleanup, TestCallbackAuditPrivatePagesUseCurrentSession, TestQualificationQueryBudgetBeforeAuthentication, TestQualificationEqualFormTimestampsPreserved, TestQualificationWithdrawnTagsDoNotPreventArchival and TestQualificationPhotoRemovalRejectsZeroTime. Adapt each to actual current ownership: archiving a Core listing is not an arbitrary new discovery authority. Full image-pipeline replacement is unnecessary because current imgsan already has all-orientation, GPS stripping, container, allocation, concurrency and output-budget tests.

### Onboarding: preserve existing practice and localization

Current onboarding has a five-step Tracker, compare-and-swap persistence in both memory and SQLite, configurable network copy/link policy, fixed English/French catalogs, no-JS HTTP and JSON adapters, and a bounded independent practice session server. The practice World supports terms/credit lessons, listings, partner selection, proposal preview, acceptance/decline/cancellation, idempotency and synthetic balance invariants. Current tests include concurrent tabs/counters, host-database byte identity, session/credential separation, CSV injection prevention and complete five-action learning flow.

The archive contributes a different private JSON checkpoint store with strict loading, atomic rename and process restart/crash scenarios. Its deletion generations preserve a replay boundary for stale learning/metrics forms; current deletion removes progress and subsequent actions use the current curriculum without this archived generation protocol. The archive's Catalog supports a complete per-request plain-text message set and RTL/LTR direction, while current already has English/French localization. Dynamic catalog support is an extensibility opportunity, not evidence that current localization is absent.

Archive Aggregate releases participant/completion totals only with complementary suppression; current Aggregate reports opt-in per-step cumulative counts and explicitly documents remaining differencing risk. Selecting one aggregate definition/retention behavior requires a privacy/API decision. It does not block parser or chronological retention repairs.

## Boundary hypotheses: separate from executed findings

| Boundary | Actual source observation | Qualification/disposition |
|---|---|---|
| Current favorites JSON | web/api.go decode uses DisallowUnknownFields and EOF but no duplicate-name or raw Unicode preflight. CleanNickname validates the already-decoded string. | Candidate duplicate/Unicode-fidelity regression; no Go reproduction in this comparison. Do not claim an authorization bypass. |
| Current QR JSON | web/web.go decodeJSON similarly uses the standard decoder; note validation runs after decoding. | Candidate malformed Unicode/duplicate-input fidelity test; camera repair does not alter this decoder. |
| Current discovery JSON | web/api.go readJSON uses DisallowUnknownFields and dec.More as the trailing-data check. | Test actual decoder with trailing closing delimiters, duplicate aliases, valid single aliases and valid controls before editing. Current cleanText rejects U+FFFD as well as malformed UTF-8, so do not generalize favorites' possible text normalization acceptance to all discovery values. |
| Current onboarding JSON | APIHandler decodes curriculum/action/step with DisallowUnknownFields and EOF; no duplicate-name preflight. Action/step then pass enum validation. | Duplicate-alias probe remains unrun. There is no reproduced signing/hash boundary in this handler. |
| Archived duplicate handling | Favorites and discovery use strings.ToLower for the seen-key map; onboarding locates child types with EqualFold but still lowers seen keys; QR tracks exact spelling only. | Do not copy these validators verbatim as a solved SimpleFold boundary. Adapt to encoding/json's actual field equivalence and preserve map-key distinctions. |
| Current onboarding retention | SQLStore serializes UTC RFC3339Nano and PurgeStale executes updated_at < ? over TEXT. | Same source pattern as the demonstrated recurring chronology defect. Needs actual current SQLStore red test with whole/fractional seconds, equality and 1ns controls, using its pinned SQLite test driver. No new Go run or production edit has occurred. |

## Action order and gates

1. Finish the independently reproduced QR replacement-image cancellation follow-up; review the exact final diff, execute the dedicated three-OS provider job and retain separate real-browser/CSP/manual gates.
2. At the next coordinated Go slot, reproduce current onboarding retention ordering and the current HTTP parser cases. These are narrowly scoped engineering checks; architecture divergence, translation review and final policy adoption do not prevent them.
3. Adapt useful archive callback/rollback/ownership scenarios onto current APIs one at a time. Do not replace already implemented durable storage or practice flows to import tests.
4. Reconcile documentation with actual current runtime and tested source identities. The stale prototype manifests and VALIDATION prose are independent documentation debt.
5. Resolve deliberate feature/API choices separately: replay tombstones and quotas in favorites; raw-token recovery and independent alias revoke in QR; projection authority/cursor behavior in discovery; progress generations/catalog extension/metric definition in onboarding.
6. Keep host authentication, authoritative Core transaction/listing integration, OpenAPI/CLI registration, policy/license acceptance, real browser/mobile/accessibility and deployment approval as explicit gates. None licenses activation of these held modules.

No changes were made to favorites, discovery or onboarding sources. The archive refs remain intact. The only implementation begun by this comparison is the separately authorized QR wrapper/test/workflow/documentation candidate.
