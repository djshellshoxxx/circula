# Light QA run

Date: 2026-10-21. Build: `go build -o <scratch>/circula ./cmd/circula` with `GOTOOLCHAIN=go1.25.9`
(`circula version` reports `dev`, no commit or build date were injected). Target: `http://127.0.0.1:8080`,
database and backups in a scratch directory outside the repository. Repository code was not modified.

Evidence (scratch, not committed): `<scratch>/qa/out/` (JSON and text responses), `<scratch>/qa/shots/` (screenshots),
`<scratch>/run/serve*.log` (service logs), `<scratch>/qa/out/browser.json` (browser results).
`<scratch>` is `/tmp/claude-0/-home-user-GrooveMorph/20548ad6-35bf-51c1-a0ea-92cdfaf82a2e/scratchpad`.

Scope limits: not run: `go test`, `make check`, the in-progress idempotency case (API-024), the verify-ledger
exit code 3 path, admin credit-limit changes, invitations, announcements (not implemented), token creation,
Bearer-token API calls, mail, and any load or timing measurement. No audio or DAW checks apply to this service.

## Setup

| # | Step | Result | Evidence |
|---|---|---|---|
| S1 | `serve -check-config` with env (setup secret set to `<set>`) | PASS | effective settings printed, exit 0 |
| S2 | `migrate` on empty DB (pre-migration backup, v1 to v2) | PASS | exit 0, backup written 0600 |
| S3 | `admin create-operator -login ops-admin` via stdin | PASS | "created person ..." |
| S4 | `serve` starts; `/healthz` 200 `{"status":"ok"}`, `/readyz` 200 `{"status":"ready"}`, `/metrics` 200 from loopback | PASS | `serve.log` |
| S5 | `/metrics` with `X-Forwarded-For` from a non-allowed client | PASS (403) | `curl` status |

## Web and JSON API flows

Persons: alice (network founder, admin), bob, carol. Network `qa-net`, join mode `approval`, decimals 2,
currency name `Credits`, default credit limit 100.00. Forms used CSRF hidden field; API calls used
`X-CSRF-Token`, `X-Network`, `Idempotency-Key`.

| # | Step | Result | Evidence / note |
|---|---|---|---|
| F1 | Register three persons (`POST /register`) | PASS | 303 to `/login` |
| F2 | Sign in (`POST /login`), session cookie set | PASS | 303 to `/` |
| F3 | Create network via `/setup` with setup secret | PASS | 303 to `/n/qa-net`, "Network created and open" |
| F4 | Bob and Carol request to join (`POST /n/qa-net/join`) | PASS | members `pending` |
| F5 | Admin approves Bob and Carol (`POST /api/v1/admin/members/{id}/approve`) | PASS | 200, status `active` |
| F6 | Approve replay with same key | PASS | 200, same body |
| F7 | Approve without `Idempotency-Key` | PASS | 400 `idempotency-key-required` |
| F8 | Bob creates offer listing (`POST /api/v1/listings`) | PASS | 201 |
| F9 | Alice lists listings | PASS | listing visible, `active` |
| F10 | Alice proposes pay 25.00 to Bob | PASS | 201 `proposed` |
| F11 | Proposer accepts own proposal | PASS (403) | "the proposer cannot accept their own proposal" |
| F12 | Bob accepts | PASS | 200 `completed`, `posted_ledger_seq` 1 |
| F13 | Balances after trade | PASS | Alice -2500 `in_debt`, Bob +2500 `in_credit`, Carol 0 `zero`; sum 0 |
| F14 | Carol proposes pay 5.00 to Bob; Bob declines | PASS | status `declined`, no entries |
| F15 | Carol proposes pay 3.00 to Bob; Carol cancels | PASS | status `cancelled` |
| F16 | Bob tries to cancel Carol's proposal | PASS (403) | "only the proposer can withdraw; decline instead" |
| F17 | Alice proposes pay 80.00 to Bob, available 75.00 | PASS (422) | `insufficient-available-credit`, "payer has 75.00 available" |
| F18 | Bob proposes pay 12.00 to Carol; Carol accepts | PASS | completed |
| F19 | Carol proposes receive 5.00 from Bob; Bob accepts | PASS | `my_direction` `receive` for Carol; payer is Bob |
| F20 | Balance sum after F18/F19 | PASS | -2500 + 800 + 1700 = 0 |
| F21 | Carol (party) opens dispute `quality` on the 12.00 trade | PASS | 201 `open` |
| F22 | Non-party Alice opens dispute on that trade | PASS (404) | `not-found` |
| F23 | Admin grants herself reviewer role | PASS (403) | "administrators cannot perform this action on themselves" |
| F24 | Alice (admin, no reviewer role) resolves with `reversal` | see BUG-3 | 200, `upheld`, reversal 1200 |
| F25 | Balances after dispute reversal | PASS | Alice -2500, Bob +2000, Carol +500; sum 0 |
| F26 | Admin credit adjustment +1.00 to Carol with reason | PASS | 201 completed, `posted_ledger_seq` 5 |
| F27 | Adjustment without reason | PASS (422) | `validation-failed` at `/reason` |
| F28 | Admin reversal of 5.00 trade with reason | PASS | 201, `posted_ledger_seq` 6 |
| F29 | Second reversal of same trade | PASS (409) | `invalid-transaction-state`, `current_status` `reversed` |
| F30 | Member sum after F26/F28 | PASS | member sum +100 = system account -100 (adjustment offset); `verify-ledger` ok |
| F31 | Admin reverses trade where admin is party | PASS (403) | DOM-037 |
| F32 | Admin adjusts own account | PASS (403) | DOM-037 |
| F33 | Audit view as admin | PASS | 200, 24 events in first page, `network.created` onward |
| F34 | Audit view as member | PASS (403) | "administrator role required" |
| F35 | Audit unknown query param | PASS (400) | `invalid-query` |
| F36 | Member export (`scope mine`, csv) | PASS | 201 `complete`, sha256 matches downloaded artefact (`b7f92cc7...`) |
| F37 | Admin export `ledger` json; download | PASS | 200 zip with `ledger.json`, `manifest.json` |
| F38 | Member export `ledger` and `full` | PASS (403) | "administrator role required" |
| F39 | Invalid export scope or format | PASS (422) | `invalid-enum` |
| F40 | Idempotent retry: same key, same body | PASS | 201, `Idempotent-Replayed: true`, same id, no new trade |
| F41 | Same key, different body | PASS (422) | `idempotency-key-reused` |
| F42 | Key reused after a failed command (F17) | PASS | 201; failed commands not recorded |
| F43 | Unknown body field; number instead of string; leading zero | PASS (400, 400, 422) | `invalid-body`, `invalid-body`, `invalid-amount` |
| F44 | Credential in query string | PASS (400) | `credentials-in-query` |
| F45 | Cookie POST without `X-CSRF-Token` | PASS (403) | `csrf-invalid` |
| F46 | Non-party member reads a trade | PASS (404) | Carol on Alice-Bob trade |
| F47 | Web propose: review then confirm | PASS | review page "You pay Carol 3.00 Creditss", then redirect to trade |
| F48 | Web propose double submit with same `idem` | PASS | one trade only |
| F49 | Web accept: Carol review, confirm | PASS | status `completed`, posted |
| F50 | Member sum after F49 | PASS | -2500 + 2200 + 400 = 100 (adjustment offset as in F30) |
| F51 | Password change wrong current password | PASS (422) | `/current_password` `incorrect` |
| F52 | Password change new password too short | PASS (422) | min 12 characters |
| F53 | Password change correct | PASS | 204 |
| F54 | Old session after change | PASS (401) | |
| F55 | Old password sign-in | PASS (401) | sign-in page re-rendered |
| F56 | New password sign-in | PASS | 303 to `/` |
| F57 | Logout without CSRF | PASS (403) | |
| F58 | Logout with CSRF | PASS | 303 to `/login`; API then 401; dashboard redirects to login |

## Backup, restore, lifecycle

| # | Step | Result | Evidence / note |
|---|---|---|---|
| B1 | `backup <dir>` while serve runs | PASS | `ledger: ok`, manifest `integrity_check` ok, 0600 files |
| B2 | Manifest sha256 check of the backup | PASS | `sha256sum -c` OK |
| B3 | `verify-ledger` on a copy of the backup | PASS | `network qa-net: ok` |
| B4 | Second `serve` on same DB, different port | PASS (exit 1) | "another circula process holds the database lock" |
| B5 | Second `serve` on same port | PASS (exit 1) | same message |
| B6 | `migrate -dry-run` and `restore` while serving | PASS | "schema up to date"; restore refused, lock held |
| B7 | SIGTERM `serve` (first instance) | PASS | log "shutdown requested" then "circula stopped" |
| B8 | SIGTERM `serve` (second instance) | PASS | exit 0 recorded by wrapper |
| B9 | `restore` without `-overwrite` on existing DB | PASS (exit 1) | refused, DB unchanged |
| B10 | `restore -overwrite` of a byte-altered backup | PASS (exit 1) | "does not match its manifest checksum or size" |
| B11 | `restore -overwrite` of the good backup | PASS (exit 0) | pre-restore backup written (ledger ok); `restore-<time>.json` written; warning that the backup is not authenticated (no backup key) |
| B12 | `verify-ledger` live DB after restore | PASS | ok, exit 0 |
| B13 | `verify-ledger -network nope` | PASS (exit 1) | "no network matches" |
| B14 | Data after restore | PASS | Alice balance -2500 as before the backup |
| B15 | `/healthz`, `/readyz`, `/metrics` on restarted serve | PASS | 200 each; metrics include `circula_schema_version 2` |
| B16 | First `/healthz` probe about 3 s after start | NOTE | returned no response, next probe OK; the 3 s wait was a fixed sleep, not a measured startup time |

Note on B11: the backup was taken before any writes that followed it, so the restore did not lose data in this run.
Data loss after a backup time was not exercised.

## Browser (headless Chromium via Playwright 1.56.1, chromium-1194)

Screenshots: `<scratch>/qa/shots/01` to `15`. Console and page errors were collected per context.

| # | Step | Result | Evidence / note |
|---|---|---|---|
| U1 | Login page loads, desktop screenshot | PASS | `01-login-desktop.png` |
| U2 | Sign in as alice with form fields | PASS | redirect to `/n/qa-net/` |
| U3 | Dashboard shows direction text | PASS | "In debt by 25.00", "In credit by 25.00" (Bob) |
| U4 | Dashboard unit text | FAIL | "Creditss", see BUG-1 |
| U5 | Pages load: trades, trade form, history, directory, disputes, admin audit, admin adjust, admin exports, admin members, password, help | PASS | HTTP 200 each, screenshots `03` to `13` |
| U6 | Console errors on desktop pass | FAIL (minor) | one `404` for `/favicon.ico`, see BUG-2 |
| U7 | 320px login: no horizontal scroll | PASS | scrollWidth 320 = clientWidth 320 |
| U8 | 320px login, Tab sequence (12 stops) | PASS | skip link first; every stop had a visible focus outline |
| U9 | Keyboard-only sign-in at 320px (type login, Tab, type password, Tab to Sign in, Enter) | PASS | redirected to dashboard |
| U10 | 320px trade form: no horizontal scroll | PASS | 320 = 320 |
| U11 | 320px trade form Tab sequence (25 stops) | PASS | all stops had a visible focus outline; the second direction radio is skipped by Tab (normal radio-group behaviour, arrow keys reach it) |
| U12 | Console errors on 320px pass | PASS | none |

## Bugs and observations

BUG-1 (UI, low). Doubled plural unit text. Repro: `setup` with currency name `Credits` (plural). The dashboard shows
`100.00 Creditss`, the pending list shows `You pay Bob 10.00 Creditss`, and the level text uses `unit+"s"`. Cause:
`internal/web/format.go` `unitFor` appends `s` to any name that is not exactly one unit. A singular currency name
(`Credit`) works. Expected: the currency name is not pluralized, or plural forms come from configuration.

BUG-2 (minor). `GET /favicon.ico` returns 404 on every page, which logs a console error in the browser. Repro:
`curl -i http://127.0.0.1:8080/favicon.ico`. Expected: a 204 or an icon link in the page head.

BUG-3 (spec conflict, needs decision). Admin without the reviewer role resolved a dispute. Repro: admin Alice (roles
`["admin"]`, role grant to self refused with 403) calls `POST /api/v1/disputes/{id}/resolve` with `outcome: reversal`
and gets 200 with `reviewer_id` = Alice. `API-018` and `FR-065` say the reviewer role is required for reviewer
operations and that reviewer assignment does not grant administrator rights. `DOM-038` and the service comment
(`internal/service/disputes_decide.go`) allow "reviewer or administrator". Decide which rule holds and align the spec
or the code.

OBS-4 (spec and API inconsistency, needs decision). `GET /api/v1/transactions/{id}` and `/events` return 200 to an
administrator who is not a party to the trade. `API-049` says a non-party gets 404, and `GET /transactions` lists only
the caller's own trades. Repro: Alice (admin) reads Bob-Carol trade `01a15e0a-52e5-...`. If admin oversight is intended,
the spec should say so; otherwise restrict the read.

OBS-5 (spec drift, not a defect). `API-004` and `API-021` require printable ASCII keys. A key containing a space was
accepted (201), which matches RFC printable ASCII. The step was written expecting 400, so the expectation was wrong.

OBS-6 (not in scope of the service). Adjustments post an equal and opposite entry on the system account. Member
balances therefore sum to the adjustment amount, not 0. Sums of zero hold only when the system account is included
(verify-ledger ok). The check "balances sum to 0" must include the system account after an adjustment.

## Not run or not verified

- Unit, race and integration tests (`go test`, `make check`): not run in this pass.
- Native Windows, macOS and Linux arm64 builds and CLAP or VST3 checks: not applicable to this repository.
- Concurrent acceptance on one account and in-progress idempotency (409): not run.
- Backup authenticity (`CIRCULA_BACKUP_KEY`) and encryption: not configured; not run.
- Restore audit, metrics for posting counters: not run, documented as known gaps in `docs/ops/README.md`.
- Screen-reader behaviour, colour contrast ratios, and 200% zoom: not measured.
- Only Chromium was used. No other browser or device was tested.

## Triage (2026-10-21)
- BUG-1 fixed (no doubled plural; test added). BUG-2 fixed (`/favicon.ico` returns 204).
- BUG-3 / OBS-4: kept as built. DOM-038 lets a network administrator resolve disputes and read any transaction in their network; API-018, FR-065 and API-049 wording should be aligned to DOM-038 (owner to confirm; logged as an open decision).
- OBS-6: expected. Adjustments post against the system account (LED-1 holds per network including the system account).
- Not run: concurrency under load, bearer-token API, invitations, backup HMAC key, exit code 3 path, browsers other than Chromium, screen readers, any performance measurement.
