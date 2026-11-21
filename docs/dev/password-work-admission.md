# Shared password-work admission (AUTH-04)

Circula Core admits at most **two password-work operations per process**. The
default is deliberately small and has **zero queued waiters**: when both tickets
are occupied, a new operation immediately returns the existing `unavailable`
domain error. HTTP adapters map that error to **503** with `Retry-After`; command
line callers receive the retryable error. This implementation default does not
resolve the owner's minimum-hardware benchmark decision, DEC-006.

## Ownership and placement

`internal/passwordwork.Process()` owns one fixed-capacity limiter. Every normal
service instance and the backup passphrase derivation use that same object.
`service.New` does not resize it. `Config.PasswordWork` permits an explicitly
isolated scope for deterministic tests; application construction leaves it nil.
An embedding that supplies custom scopes is responsible for their composition
and cannot claim the application's shared default guarantee.

A service operation acquires its ticket before credential-specific repository
reads, durable guessing reservations, recovery-code consumption or credential
writes. Public input validation and checks of the already supplied actor may
precede admission. Sign-in acquires before even distinguishing a malformed,
known or unknown login, so occupied capacity returns the same refusal in all
three cases. The service's authentication observer does not count a capacity
refusal as a failed password proof or a throttled guess.

| Production path | Ticket lifetime |
|---|---|
| Service startup | One ticket for the dummy-password hash |
| Registration, including operator provisioning | Before hashing through registration completion |
| Sign-in, including dummy and unusable-PHC paths | Before durable attempt reservation through session/challenge issuance |
| Password change | Before account lookup through old-password proof, replacement hashing and the protected write |
| Account recovery-code generation | Before the current-password reservation through code replacement |
| Account password recovery | Before account/code lookup and reservation through replacement hashing and atomic consumption |
| Administrator and operator password reset | Before temporary-password generation/hashing through the reset write |
| MFA begin, confirm and disable | Before credential reservation or lookup through the protected factor change |
| Backup passphrase encryption and decryption | Around the synchronous Argon2 key derivation |

The ticket covers a complete operation, not only its individual Argon2 calls.
Password change uses one ticket for its two sequential evaluations. No caller
reacquires between those evaluations. A ticket refuses simultaneous reuse;
legitimate nested operations must obtain their own ordinary ticket. Limiter
bookkeeping never holds a mutex while calling cryptographic work or the test
observer, preserving the stale-proof tests that reenter a password change/reset.

Existing transport controls still run at their original boundaries. Session
authentication, CSRF validation and adapter request rate limits can reject a
request before the password service is called. In particular, the JSON password
route retains its independent request-count limiter. A capacity refusal does
not reserve a **durable service guessing attempt**, consume recovery material,
replace a password, revoke a session or issue new credentials. The HTML sign-in
failure counters also exclude `unavailable`.

## Cancellation and release

Admission checks context cancellation before acquiring capacity and again after
acquisition. `Ticket.Run` checks before starting work and after the synchronous
function returns. Service Argon2 also checks after its test observation hook,
before invoking the primitive. Every production owner defers ticket release.

Argon2 has no interruptible context API. Cancellation during an evaluation does
not terminate it in another goroutine and does not free its capacity early.
The operation waits for that evaluation to return, then propagates the context
error without continuing its credential transition. Even if a ticket owner
requests release while synchronous work is running, its slot remains occupied
until that work returns. A canceled admitted attempt retains any guessing
reservation already committed; cancellation does not grant additional guesses.

Startup has no caller context. The existing backup streaming constructors also
have no context parameter; they call the context-aware derivation wrapper with
`context.Background()`. They therefore retain their existing synchronous
completion behavior while sharing the process budget. The new helper can refuse
a canceled context before work and report cancellation after work, but this
change does not claim end-to-end cancellation of the existing backup stream.
Raw-key backup derivation uses HKDF and does not consume an Argon2 ticket.

## Compatibility and remaining decisions

The limit counts operations, **not bytes**. The default password hash remains
Argon2id with 64 MiB, three passes and two lanes; backup passphrase parameters
also remain unchanged. Each verifier continues to honor the recorded parameters
of an accepted historical PHC. A historical hash can request substantially more
memory than the current default, so two admitted operations are not a promise
of a 128 MiB process ceiling. No PHC format, accepted parameter range, credential
strength, database schema or backup format changes in this milestone.

The existing durable guessing budgets, unknown-account dummy work, current-hash
rechecks, session/token authority and atomic recovery-code use remain in place.
The work does not impose a new global guessing allowance or weaken existing
throttles. Byte-based admission or new historical-PHC limits would need an
explicit compatibility/migration decision and minimum-hardware measurements.
The standalone Argon2 parallel benchmark remains a primitive benchmark using
isolated test admission; its old measurements are not a production load result.

## Verification approach

The new tests occupy a small number of **inert tickets** and use channel-driven
callbacks to verify ownership without expensive concurrent hashes. They cover
prompt refusal, cancellation before and after synchronous work, deferred release,
idempotent cleanup, sequential work, panic cleanup and ordinary reentry.
Service tests use cheap existing fixture parameters and record repository calls
to show that every password entry point refuses before credential reads or
effects. Additional tests cover shared startup/service/backup scope, preserved
recovery codes, HTML and JSON 503 responses, and unchanged password-change proofs.

Existing tests of parallel guessing, recovery-code consumption and registration
uniqueness receive explicit isolated fixture capacity so that all intended
operations still reach the invariant they test. Independently sharded ledger
property tests likewise own isolated fixture admission. These test scopes do
not alter the application's shared default. The existing reentrant stale-proof
tests use an explicit isolated two-operation capacity; both overlapping calls
retain ordinary ticket accounting, with no observer-specific bypass.

Execution receipts are recorded separately; the test descriptions above are the
intended checks and are not claims that an unexecuted suite passed.
