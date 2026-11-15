# Crypto conformance review

Date 2026-10-21, base f8278a0. Code read: `internal/service/totp.go`, `password.go`, `auth.go`, `service.go`, `internal/httpx/httpx.go`, `cmd/circula/backupcrypt.go`.

Source access note: rfc-editor.org, datatracker.ietf.org, go.dev and cheatsheetseries.owasp.org were blocked by the sandbox proxy. RFC vectors were taken from the pquerna/otp test files (raw.githubusercontent.com, which embed the RFC 4226 App. D and RFC 6238 App. B tables) and cross-checked by an independent HOTP implementation in the new tests. OWASP text came from the cheat sheet's GitHub markdown. Argon2 guidance came from pkg.go.dev/golang.org/x/crypto/argon2 (v0.58.0), which quotes RFC 9106 section 4. Items not read in the primary RFC text are marked (secondary).

Verdict key: OK, NOTE (works, improvement advised), FIXED, FINDING.

## 1. TOTP / HOTP

| Item | Reference | Code | Verdict |
|---|---|---|---|
| HOTP truncation, big-endian counter, 31-bit mask, mod 10^6 | RFC 4226 5.3-5.4 | `hotp` | OK |
| RFC 4226 App. D, counters 0-9 (755224 ... 520489), HMAC-SHA-1 counter 0 = cc93cf18... | RFC 4226 App. D | `TestHOTPRFC4226AppendixD` | OK, test added |
| RFC 6238 App. B, 8 digits, SHA1/SHA256/SHA512 seeds of 20/32/64 bytes, 6 times each | RFC 6238 App. B | `TestTOTPRFC6238AppendixB`: reference impl for all three hashes; production `hotp` checked against the SHA-1 column (6 digits = low six of the 8-digit value) | OK, test added |
| Step = floor(unix/30), T0 = 0 | RFC 6238 4.2 | `totpStep` | OK |
| Production algorithm is SHA-1 / 6 digits / 30 s only | RFC 6238 allows SHA-256/512, but Google Authenticator-compatible apps assume SHA-1/6/30 | URI states them explicitly | OK (SHA-256/512 deliberately unsupported; HMAC-SHA1 is not collision-sensitive) |
| Secret length 160 bits | RFC 4226 R6 (min 128, recommended 160) | `totpSecretLn = 20`, crypto/rand | OK |
| Validation window | RFC 6238 5.2: accept at most one step back for transmission delay; resync is a server decision | skew +-1 (previous, current, next) | NOTE: the +1 (future) step is beyond the RFC's wording but common for client clock drift; tested in `TestMatchTOTPSkewAndReplay` |
| Replay | RFC 6238 5.2: a verifier MUST NOT accept the second attempt of an OTP after a successful validation | `matchTOTP(last)` accepts only steps > last accepted | OK |
| Brute force | RFC 4226 7.3 / RFC 6238 5.2 throttling | login throttle exists; its design is SA2-10 | see SA2-10 |
| Comparison | constant time | `subtle.ConstantTimeCompare` over all 3 candidates, no early exit | OK |
| Input shape | exactly 6 ASCII digits | `isDigits` | OK |
| Base32 handling | RFC 4648 5, unpadded, upper case (Key URI format convention) | `base32.StdEncoding` without padding; 20 bytes -> 32 chars; round trip tested | OK |
| otpauth URI | Google Authenticator Key URI Format (secondary: github.com/google/google-authenticator/wiki/Key-Uri-Format): `otpauth://totp/Issuer:account?secret=&issuer=&algorithm=&digits=&period=`, label percent-encoded, issuer param must match label prefix | `otpauthURI`; tested incl. hostile login | OK. NOTE: `url.PathEscape` leaves `:` and `@` literal in the label; apps accept this |
| Seed at rest | not specified | AES-256-GCM, 96-bit random nonce, person id as AAD; key = SHA-256(domain tag + SecretKey) | OK if SecretKey is high-entropy (NOTE: use HKDF-style derivation or require >= 32 random bytes; SHA-256 of a weak passphrase gives no stretching). Without SecretKey the seed is plain base32 in the DB (documented fallback) |
| Recovery codes | none | 50 random bits, hashed with SHA-256 bound to person id | NOTE (SA-REC): 50 bits under an unsalted-cost hash is brute-forceable offline in about a day on one GPU if the database leaks (2^50 / ~10^10 H/s). A leak also yields the Argon2id password hash and, without SecretKey, the TOTP seed, so the marginal loss is low. Hardening: HMAC with a server key, or raise to 80+ bits. Not changed (format/UX change, owner decision) |

## 2. Backup container (AES-256-GCM, chunked)

Layout: `"CIRCBK1" | kdf(1) | chunk(4) | salt(16) | nonce prefix(8)`, chunk i = GCM(key, nonce = prefix|i(4), aad = header|i(8)|final(1)).

| Item | Reference | Analysis | Verdict |
|---|---|---|---|
| Key size | AES-256 | 32-byte key from HKDF-SHA256 (high-entropy 32-byte secret) or Argon2id (passphrase) | OK |
| Key per file | NIST SP 800-38D 8.3 limits invocations per key; unique key avoids cross-file IV reuse | A fresh 16-byte random salt gives a fresh key for every file (both KDFs), so IV uniqueness only has to hold within one file | OK |
| IV construction | SP 800-38D 8.2.1 deterministic construction, 96-bit IV = fixed field || counter | IV = 8-byte per-file prefix || 32-bit big-endian chunk index; within a key the counter is strictly increasing, so no IV repeats. The random prefix is redundant given the per-file key but harmless | OK |
| Invocation limit | SP 800-38D 8.3: at most 2^32 invocations with random IVs; deterministic IV allows up to 2^32 here by counter width | 2^32 chunks x 64 KiB = 256 TiB per file; `flush` refuses at idx = 2^32-1 | OK |
| Plaintext per invocation | SP 800-38D 5.2.1.1: P <= 2^39-256 bits (~64 GiB) | max chunk 16 MiB | OK |
| Tag length | SP 800-38D 5.2.1.2: 128 bits preferred | 16 bytes | OK |
| AAD | Binds context | Whole header (magic, kdf id, chunk size, salt, prefix) + chunk index + final flag | OK; downgrade of kdf byte or chunk size breaks every tag |
| Reorder, drop, truncation, append | STREAM online AE (Hoang, Reyhanitabar, Rogaway, Vizarova, CRYPTO 2015, eprint 2015/189) puts a last-block bit in the nonce/AD; Google Tink streaming AEAD does the same | index and final flag both in AAD; decoder derives `final` from EOF and retries with the opposite flag only to classify the error | OK; equivalent security to STREAM (flag in AD instead of nonce) |
| Decrypt-then-use | STREAM releases plaintext before the end is verified | `decryptFile` removes the output on error; `decryptTo` doc says caller must discard | OK (documented) |
| Passphrase KDF | RFC 9106 section 4 second recommended option (t=3, m=64 MiB, p=4), 32-byte tag | `argon2.IDKey(pass, salt, 3, 64<<10, 4, 32)`, salt 16 bytes | OK. NOTE: parameters are implied by the kdf byte, not stored; changing them needs kdf id 3 |
| Secret classification | | 32 bytes base64 -> HKDF, else passphrase | NOTE: no minimum passphrase length; a short passphrase is Argon2-limited only |
| Tests | | `backupcrypt_vectors_test.go` pins header length, nonce layout, AAD layout, per-file fresh salt/prefix, KDF output length and kind check; existing `backupcrypt_test.go` covers tamper/truncate/append/reorder | tests added |

## 3. Password hashing, tokens, CSRF

| Item | Reference | Code | Verdict |
|---|---|---|---|
| Argon2id parameters | OWASP: m=19 MiB t=2 p=1 minimum (also 46 MiB t=1, 12 MiB t=3, ...); RFC 9106 s4: first option t=1 p=4 m=2 GiB, second option t=3 p=4 m=64 MiB | m=64 MiB, t=3, p=2, 32-byte tag, 16-byte salt | OK: above the OWASP minimum, memory/time equal to RFC second option, p=2 below the RFC's 4 (fewer lanes cost an attacker no more; DEC-006 benchmark still owed) |
| Salt and tag | RFC 9106 3.1: 128-bit salt recommended, 128-bit tag minimum | 16 / 32 bytes | OK |
| PHC string | github.com/P-H-C/phc-string-format: `$argon2id$v=19$m=..,t=..,p=..$salt$hash`, unpadded base64 | matches; tested | OK |
| Verification | constant time, params from the hash | `subtle.ConstantTimeCompare`; memory capped at 2 GiB | FIXED: a stored hash with `t=0` made `argon2.IDKey` panic, and unbounded `t` could stall a login. Now rejected for t=0 or t>64 (`password.go`, regression `TestVerifyPasswordRejectsDegenerateParams`). Only reachable by database tampering, so Low |
| NOTE | | `fmt.Sscanf` ignores trailing text in `m=..,t=..,p=..`; harmless, hash must still match | |
| Session tokens | OWASP session management: >= 64 bits entropy (128 recommended in practice) | 32 random bytes (256 bits), `crypto/rand`, base64url | OK |
| Token storage | Hashing a 256-bit random token with plain SHA-256 is sufficient: brute force needs 2^256 work, so a slow KDF adds nothing; storage as hash means a DB read does not yield usable sessions | `hashToken` = hex SHA-256 | OK |
| API / join / MFA challenge tokens | same | 32 bytes (recovery token 15 bytes = 120 bits) | OK |
| CSRF | OWASP CSRF cheat sheet: HMAC-signed token bound to the session | HMAC-SHA256(key, "circula-csrf-v2\0" + binding), binding = session token or pre-login cookie; random per-process key when none configured | OK. NOTE: a per-process random key invalidates forms on restart (acceptable) |
| Constant-time compares | | `subtle.ConstantTimeCompare` / `hmac.Equal` everywhere seen (CSRF, flash, cursor, ops bearer via hash, secretEqual via SHA-256) | OK |
| Cookie hardening | | SA2-14 | open |

## 4. Fixes made

1. `internal/service/password.go` `verifyPassword`: reject `t=0` and `t>64` (panic / DoS on a damaged row).

Everything else is test-only: `totp_vectors_test.go`, `password_vectors_test.go`, `backupcrypt_vectors_test.go`.

## 5. Open advice (no code change made)

- Recovery codes: add server-keyed HMAC or more bits (see table).
- SealKey derivation: document that `SecretKey` must be 32 random bytes, or run it through HKDF.
- Backup passphrase: enforce a minimum length; reserve kdf id 3 for future Argon2 parameters.
- TOTP forward skew (+1 step) exceeds RFC wording; consider -1..0 plus resync UI if desired.

## Sources

- RFC 4226 https://www.rfc-editor.org/rfc/rfc4226 ; RFC 6238 https://www.rfc-editor.org/rfc/rfc6238 ; RFC 9106 https://www.rfc-editor.org/rfc/rfc9106 (hosts unreachable here; vectors via https://github.com/pquerna/otp hotp_test.go / totp_test.go)
- NIST SP 800-38D https://csrc.nist.gov/pubs/sp/800/38/d/final (from knowledge, not fetched)
- STREAM: https://eprint.iacr.org/2015/189 (from knowledge, not fetched)
- OWASP Password Storage https://cheatsheetseries.owasp.org/cheatsheets/Password_Storage_Cheat_Sheet.html (fetched via GitHub markdown)
- https://pkg.go.dev/golang.org/x/crypto/argon2
