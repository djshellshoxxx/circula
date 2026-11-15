# Addendum: SEC-027 runtime secrets persisted; SEC-028 setup secret

Status: implemented. Operator guide: [configuration.md](../../ops/configuration.md#runtime-state-the-web-signing-key-sec-027).

## Web signing key

- Problem: `newHandlers` drew a new random CSRF/HMAC key each start, so signed cookies and form tokens broke on restart.
- Fix: `cmd/circula/statekey.go`. Key = 32 random bytes, base64, in `<state-dir>/circula-web.key` (`state-dir` defaults
  to the database directory; created 0700 when missing). Written to a private temp file (0600) and hard-linked into
  place, so creation is atomic and, when processes race, exactly one key wins and all load it. Loading refuses a file
  readable by group/others, a non-regular file (symlink) or content that is not exactly 32 base64 bytes; it is never
  silently replaced. The API and the UI share the key as before.
- Rotation: stop, delete the file, start. Existing signed cookies and tokens stop validating; ledger data is untouched.
  The key is not in the database and not in backups; back it up with the configuration.

## Setup secret persistence (SEC-028 bootstrap)

- Problem: the "setup secret already used" flag was an in-memory atomic, so a restart re-armed the secret.
- Fix without a migration: the durable state is the platform-scope audit log. `setup.secret_used` is appended (same
  transaction, hash-chained) when the secret creates a network; `setup.secret_reset` re-arms it. The secret is spent when
  the newest of the two is a use. `service.SetupSecretUsed`, `MarkSetupSecretUsed` (conflict if already used, so exactly
  one racer wins) and `ResetSetupSecret` (system actor only) live in `internal/service/setup_state.go`; the store adds
  `LastPlatformAuditSeq` (`internal/store/sqlite_platform.go`). No new table, so no migration number is consumed
  (the offered 0008 was not needed, and a lone 0008 would break the contiguous-version check on this branch).
- Web: `setupPost` serialises check, create and mark under a mutex; the atomic remains as an in-process fast path.
  A database error fails closed (500). Changed message: ask the operator to reset.
- Operator: `circula setup-reset` (needs `serve` stopped; audited with the system actor) allows exactly one more network.
- Known window: the network is created before the use is recorded (separate transactions). A crash between the two
  leaves the secret unspent although a network exists; the operator can see it in the audit log.

## Tests

| Criterion | Test |
|---|---|
| Key created, persisted, 0600, no stray files | `TestWebKeyCreatedPersistedAndPrivate` |
| State dir 0700 | `TestWebKeyStateDirCreatedPrivate` |
| Open permissions refused | `TestWebKeyRefusesOpenPermissions` |
| Corrupt / wrong size refused, not overwritten | `TestWebKeyRefusesCorruptOrWrongSize` |
| Symlink refused | `TestWebKeyRefusesNonRegularFile` |
| Concurrent first start, one key | `TestWebKeyConcurrentFirstStart` |
| Restart keeps key; corrupt key stops start | `TestNewHandlersKeySurvivesRestart` |
| Used flag survives restart; reset re-arms once | `TestSetupSecretUsedSurvivesRestart` (web) |
| Concurrent first use creates one network | `TestSetupSecretConcurrentFirstUse` (web), `TestSetupSecretMarkRace` (service) |
| Audit events, authz of reset | `TestSetupSecretStateLifecycle` |
| `setup-reset` command | `TestSetupResetCommand` |
