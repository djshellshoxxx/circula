# Independent areas audit — 2026-11-16

Reviewed implementation baseline `31eae07` against SPEC-SNAPSHOT, upstream handoff, adopted named-area scope and actual source. Integration remains **HOLD**. This review adds no core registration, geometry, map, provider or ledger authority.

## Findings fixed

1. **Ignored safety writes:** SQLite `RAISE(IGNORE)` could acknowledge area/association/sharing/retry/audit writes without persistence. Single-row writes now verify affected rows. Archive/departure cleanup verifies the scoped associations actually changed, with atomic rollback on incomplete deletion.
2. **Schema and connection assumptions:** migration accepted incompatible existing objects and writes could use a connection without foreign keys. Migration now compares the owned table/index definitions before DDL and rejects tables/views/indexes with incompatible shapes; every mutation checks the supplied connection's foreign-key setting. Initialization remains atomic.
3. **Historical retry disclosure/state confusion:** cached assignment replies could report active selections after archival/restoration/cleanup. Replays bind all payload/scope/revision facts, recheck live taxonomy and current assignment/sharing state, and reject stale/corrupt results. Listing cleanup retires cached coarse-location replies across former owners into replay-preserving fingerprint tombstones, atomically with association deletion. A failed retirement rolls back cleanup. Tombstones prevent forgotten-key recreation.
4. **Opt-out availability:** exhausted generic admin retry quota could block stopping network sharing. `DisableSharing` is a naturally idempotent, freshly authorized safety operation that allocates no retry records. Native admin form, HTTP and CLI paths expose it. Saturated-quota regression proves opt-out remains possible; no new rows are allocated by repeated disable.
5. **Ambiguous request decoding:** ordinary JSON decoding accepted duplicate semantic keys, invalid UTF-8 and lone surrogate escapes. A bounded depth-limited strict decoder now rejects these, unknown fields, non-object/trailing input, and invalid request types/charsets. Canonical local routes reject protocol-relative/encoded/backslash/trailing-slash variants. Existing form actions remain fixed relative paths, never caller-derived authorities.
6. **Cancellation:** listing policy callbacks could cancel a read while returning nil, leaving already accumulated identifiers eligible for output. Both filtering paths now stop without returning a partial page.
7. **Revision and corrupt reply handling:** requests cannot overflow revision increments; stored revisions and cached scope/payload facts are checked. Failed mutations return zero-value result DTOs rather than partially applied reply data.

## Requirement-to-evidence matrix

| Requirement | Evidence | Disposition |
|---|---|---|
| Admin-defined named taxonomy, current admin/MFA authority | Existing principal fixtures plus `TestAuditSilentWritesRollback`, schema/FK regressions | Standalone tests pass; actual core policy adapter remains unqualified |
| Optional zero/one listing association, current owner/tenant checks | Existing two-same/one-different/unset fixture; concurrent stale writes; composite FK; current projection | Pass for named areas; no exact coordinate or geometry field exists |
| Hide/archive immediately; restoring does not restore selections | Existing lifecycle tests and `TestAuditArchiveCleanupAndReplay`, restoration replay regression | Pass |
| Bounded private filtering and paging | Existing encrypted cursor scope/tamper/expiry tests; strict base64; cancellation regression | Pass; fresh keyset semantics remain explicit, not historical snapshot paging |
| Default-off network federation sharing | Existing export/role tests; `TestAuditSharingOptOutSurvivesQuotaAndStaleReplay` | Pass; exports contain taxonomy only, no members/assignments |
| Authenticated native forms and CLI | Existing no-JS service/CSRF/CLI tests; safe route/strict body tests; native sharing safety form+CLI regression | Automated/static checks pass; real keyboard/screen-reader/reflow checks pending |
| Restart, migration, rollback and backup/restore | Existing reopen/closed backup fixtures; incompatible schema rollback, ignored-write rollback, cache-retirement rollback | Standalone pass; actual core restore/transaction compatibility pending |
| Retention/privacy cleanup and replay safety | Cleanup remains available at >10,000 retry rows; cached coarse-location replies become tombstones; no auto-reapplication | Pass for explicit listing cleanup; host retention/compaction must preserve fingerprints |
| No GPS, geocoder, map service, tracking or ledger write | Source/type/import review; no image/upload or external request API; only own module diff | Pass for adopted named-area scope; image EXIF qualification remains external if connected |
| Optional maps/geometry/meetup/distance | Explicitly not adopted, as permitted by spec | Not claimed |

## Observed validation

Go 1.26.11 linux/amd64:

- `GOGC=20 GOMEMLIMIT=450MiB GOMAXPROCS=1 go test -p 1 -race -cover -timeout 90s ./...`: PASS, **70.6%** statements (4.487s).
- `GOMAXPROCS=1 go vet ./...`: PASS.
- `go mod verify`: all modules verified.
- Strict request parser fuzz, isolated from SQLite instrumentation: `go test -run '^$' -fuzz FuzzStrictRequest -fuzztime=4s -parallel=1 json.go json_fuzz_test.go`: PASS, **11,378** executions.
- Runtime `govulncheck` v1.1.4: no vulnerabilities found (frozen-source rerun recorded in validation).
- `git diff --check`: PASS for changes; existing upstream notices are unchanged.

## Remaining qualification

Actual Circula principal/MFA/store transactions, API/CLI/help registration, replay-preserving operational retention, deployment/session/CSRF configuration, full ledger regression and production backup/restore remain integration gates. Real browser, keyboard/zoom/reflow, screen-reader and deny-geolocation platform checks were not executed. Image EXIF/GPS removal is not implemented by a module accepting no photos; it must be qualified by a future media adapter. Human administrators must choose broad labels: absence of coordinates is not an anonymity proof. No full feature release or core completion is claimed. No push, merge or activation occurred.
