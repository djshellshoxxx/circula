# Archived notification identity boundary repair

## Frozen source

Base: `ba454e1c14d35bdd90dca8379e890f2bca6281d7`. Local repair: `b96100d2a53cf696f16b8c34305eb132c673863c`. Tree: `36122c54a1a11333f36700d448a41aedafa7d631`. The isolated branch is `master/notifications-archive-identity-20261119`; it remains local and is not a competing published runtime. The portable patch is `0001-notifications-archive-identity-utf8.patch`.

The four changed files are:

- `future/notifications/archive_identity_utf8_test.go` — `7c2cb746f6b7b0377a2d2b8b0be80f03275aa18d`
- `future/notifications/delivery.go` — `d422e124a8ed12b246c16f765c949ba9694b372c`
- `future/notifications/store.go` — `13ae8929f9ca429fcca3fdc1e1102a3b59ad8f2b`
- `future/notifications/unsubscribe.go` — `c4c14bcfdb884b0548d2f163a194a4e13ab7e868`

## Demonstrated behavior and minimal repair

The archived public entry points allowed malformed UTF-8 to reach JSON encoding. The encoder replaces invalid bytes with U+FFFD, so distinct invalid values could lose their exact representation in deduplication material, authenticated target metadata, stored endpoint values, and unsubscribe grants. Tests use the actual repository and original SQLite driver; the supplied authorization callback intentionally accepts identities so the module's own boundary is tested. No authorization or replay bypass is claimed.

`DedupeKey` now rejects malformed inputs before encoding; `EnqueueTx` reaches that guard before writing an outbox row. `SetTarget` rejects malformed network/member and raw endpoint strings before sealing. `UnsubscribeToken` rejects malformed identity strings before producing a capability, including when an old malformed row already exists. The patch does not rewrite stored data or change valid AAD serialization. Positive controls preserve Unicode, literal U+FFFD, separator characters, raw Unicode endpoint paths and ASCII percent escapes.

## Actual validation

All commands use the pinned archived module and unmodified `go.mod`/`go.sum`, `GOTOOLCHAIN=local`, `GOMAXPROCS=1`, `CGO_ENABLED=1`, `GOGC=10`, `GOMEMLIMIT=256MiB`, and `GOFLAGS='-mod=readonly -p=1'`. Each command has its complete environment, hashes and output in the corresponding JSON/text receipt. Private temporary directories were recreated for each command.

| Check | Result | Evidence |
|---|---|---|
| Initial targeted ordinary test link | Linker killed before assertions; 40.227s | `initial-linker-killed.json/.txt` |
| Same targeted regression with DWARF disabled, unchanged archived production | Four negative methods / 12 subcases failed; valid Unicode controls passed; wall 2.618s, package 0.084s | `fail-before-dwarf-disabled.json/.txt` |
| `go test -p 1 -ldflags=-w -count=1 -timeout=90s ./...` | Full module PASS: library 0.604s, diagnostics CLI 0.008s; wall 8.517s | `green-full-dwarf-disabled.json/.txt` |
| `go vet -p 1 ./...` | Full module PASS; wall 1.820s | `green-vet.json/.txt` |

The linker flag omits DWARF debugging data to fit this executor; no alternate database driver, build tag, dependency replacement or production behavior was introduced. Source hashes at the frozen commit exactly match the green test/vet receipts. `git diff --check` passed and the worktree is clean. A current race pass is not claimed: the independent archive qualification's full race attempt was killed while compiling SQLite/libc before tests; this patch did not repeat the same resource profile.

## Review and integration boundary

Independent source review by `/root/branch_recovery` approved the full frozen four-file diff with no blocking finding. The review traced public entry paths and confirmed existing `net/mail` malformed UTF-8 checks and canonical ASCII base64 validation cover the remaining target fields. This is source approval; a full race pass remains unqualified. Keep this narrow archival patch separate from the later JSON alias repair and from deliberate reconciliation of the two notification runtime designs. No event/schema/retention/unsubscribe policy, current PR103 transport source, PR100 workflow or Core source was changed. The module remains on HOLD.
