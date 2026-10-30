# Interim verification results: circula

Backup of the independent verification pass as of 2026-10-29 03:15 UTC. The pass stopped twice on a usage limit and is being resumed; the consolidated report supersedes this file.

Findings extracted: 55; batch-verified: 55; findings with extra lens results: 5.

| ID | Auditor severity | Verifier verdict | Verifier severity | Extra checks | Summary |
|---|---|---|---|---|---|
| A1-01 | Medium | confirmed | Medium | refute: not refuted; impact: real (Medium) | Settle-to-zero departure posts an adjustment without any network-state gate, even on closed, closing, suspended or administrator-required networks, breaking the |
| A1-02 | Medium | confirmed | Medium | refute: not refuted; impact: real (Medium) | The decimals guard (LedgerSeq > 0) is checked only when a change is requested, so a decimals change scheduled with notice takes effect after postings exist and  |
| A1-03 | Medium | confirmed | Medium | refute: not refuted; impact: real (Medium) | A re-review posts a reversal on a trade that another live (serial) dispute holds as disputed, leaving a posting_pending dispute that can never post or close, or |
| A1-04 | Low | confirmed | Low | - | ResolveDispute and RetryDisputePosting post reversals in an administrator-required network because reverse() checks only Suspended(), unlike DecideReReview whic |
| A1-05 | Low | confirmed | Low | - | The expiry sweep pages the oldest N expired proposals across all networks and skips frozen ones, so 500+ elapsed proposals in frozen networks starve every other |
| A1-06 | Low | confirmed | Low | - | Departure cancels only one page of 500 open proposals, leaving the rest proposed with a departed party and holding the counterparty's reservation until expiry. |
| A1-07 | Low | confirmed | Low | - | Closure commits closed status in one unit and writes the final export in later units; if the export fails once, the closure is never listed again so no final ch |
| A1-08 | Info | confirmed | Info | - | OperatorSetDualControl writes policy version effective+1, which collides with an already scheduled future version, so the operator cannot change the threshold w |
| A1-09 | Info | confirmed | Info | - | Reversal transactions posted by a dispute decision, retry or re-review have no audit event targeting the reversal id; only the dispute is the target. |
| A1-10 | Info | confirmed | Info | - | ExpirePendingAdjustments writes status changes and audit events into frozen networks, whereas proposal expiry and closure finalisation skip frozen networks. |
| A1-11 | Low | confirmed | Low | - | FR-005 and GOV-004 are marked built although creation-time decimals immutability is unimplemented, and TestPolicyChanges asserts the non-conforming behaviour. |
| A2-01 | Low | confirmed | Low | - | Case-variant duplicate JSON keys pass the duplicate check (last spelling wins) and wrongly-cased field names are accepted instead of refused as unknown. |
| A2-02 | Low | confirmed | Low | - | A token restricted to one network lists the person's other tokens (with other network ids), revokes them, and changes the person-wide time zone. |
| A2-03 | Low | confirmed | Low | - | The replay cache is keyed only by person+key, so a read-only token (or demoted admin) replays a cached write response such as a plaintext invitation token. |
| A2-04 | Low | confirmed | Low | - | A break-glass grant-admin administrator is refused by every web admin page and by API adminOnly pre-checks, while the service accepts them. |
| A2-05 | Low | confirmed | Low | - | GET /admin/consistency-checks/{id} and GET /admin/policy are authorised only by the adapter role check, so an admin without the mandatory second factor can read |
| A2-06 | Low | confirmed | Low | - | POST /logout ignores a failed session revocation, reports 'You are signed out.' and clears the cookie while the server session stays valid. |
| A2-07 | Low | confirmed | Low | - | Credit-limit and policy ETags omit the override/expiry and scheduled policy changes, so a stale If-Match succeeds and silently undoes another admin's change. |
| A2-08 | Low | confirmed | Low | - | Runtime 423 network-frozen, 429, 503 busy and some 413/415 statuses are not documented in openapi.yaml, so kin-openapi validation rejects them. |
| A2-09 | Low | confirmed | Low | - | GET /network/policy-log returns every version and GET /challenges up to 200 rows, both with next_cursor null, ignoring limit, cursor and status. |
| A2-10 | Low | confirmed | Low | - | /n/{slug}/join tells any signed-in person whether a slug exists and shows the name of an invitation-only network, while other /n/{slug} pages answer 404. |
| A2-11 | Low | confirmed | Low | - | Sections hidden from a suspended or departed member's navigation still answer 200 by URL. |
| A2-12 | Low | confirmed | Low | - | A re-review decision posting a reversal is recorded on first submit with no review step, and dual-control 'Approve and post' also posts without a review screen. |
| A2-13 | Info | confirmed | Info | - | Frozen-network notice and admin overview last-check line display escaped <time datetime=...> markup as text. |
| A2-14 | Low | confirmed | Low | - | Custom qs (url.QueryEscape) inside href is escaped again by html/template, so the double-encoded next fails SafeNext and invitation sign-in lands on '/'. |
| A2-15 | Info | confirmed | Info | - | /static/*, /robots.txt and /favicon.ico are Cache-Control: public yet carry a fresh Set-Cookie cm_csrf pre-login CSRF secret. |
| A2-16 | Info | confirmed | Info | - | Path ids reach the store unvalidated; SQL is parameterised so not exploitable, but SEC-024 requires rejection before any database call. |
| A2-17 | Info | confirmed | Info | - | A person who must change their password can still load '/' and '/join/{token}' instead of being redirected. |
| A2-18 | Info | confirmed | Info | - | invalid-body problem detail echoes Go internals and several reads ignore unknown query parameters. |
| A2-19 | Info | confirmed | Info | - | OpenAPI declares an optional Idempotency-Key on POST /tokens but the route ignores it, so a retry with the same key mints a second credential. |
| A2-20 | Info | confirmed | Info | - | The web reference shown to users is a second random id unrelated to the access-log request_id, and the API stores a client-chosen X-Request-Id in audit events. |
| A2-21 | Info | confirmed | Info | - | API tokens and session ids are stored as plain unsalted, unkeyed SHA-256 whereas the spec requires a salted or keyed hash. |
| A3-F01 | Medium | confirmed | Medium | refute: not refuted; impact: real (Medium) | With backup-encryption-key set, the pre-migration gate runs integrity_check on the .db.enc ciphertext, so circula migrate and serve (migrate-mode=auto) can neve |
| A3-F02 | Medium | confirmed | Medium | refute: refuted; impact: real (Medium) | Break-glass 'no lasting changes' rule (requireLasting) covers only about 10 methods, so a break-glass admin can post adjustments, approve dual-control requests  |
| A3-F16 | Medium | confirmed | Medium | - | Second-factor throttle is renewable by one failure at expiry and is shared with session-only DisableMFA, so a stolen session keeps the owner out of MFA sign-in  |
| A3-F03 | Low | confirmed | Info | - | require-mfa-admins is not applied to admin powers reached through requireDecider, so an admin without a second factor resolves disputes and decides challenges. |
| A3-F05 | Low | confirmed | Low | - | Erasure leaves the member's challenge text (challenges.reason) and the login in 'recover: <login>' throttle rows. |
| A3-F06 | Low | confirmed | Low | - | DeleteThrottleFamily compares a Go byte length with SQLite character substr, so per-address throttle rows of non-ASCII logins are never removed. |
| A3-F09 | Low | confirmed | Low | - | Restore resurrects sessions and API tokens revoked after the backup (and, by trace, closed break-glass sessions), and the documented remedy (password change) le |
| A3-F10 | Low | confirmed | Low | - | Backup schedule and retention trust the timestamp in backup file names, so one future-dated backup stops scheduled backups for years and is always retained. |
| A3-F11 | Low | confirmed | Low | - | circula dual-control fails with 'conflict: duplicate record' while a policy change waits out its notice because both use version in-force+1. |
| A3-F13 | Low | confirmed | Low | - | Docs say rotating circula-web.key signs everyone out, but sessions are unsigned DB tokens that survive and no mass session revocation exists. |
| A3-F14 | Low | confirmed | Low | - | Migration static check flags only DROP TABLE/COLUMN and DELETE FROM, missing UPDATE entries/audit_events and DROP TRIGGER. |
| A3-F15 | Low | confirmed | Low | - | Hourly scheduled verification writes a consistency_runs row every run (and audit events on failure/recovery), contrary to the accepted read-only rule. |
| A3-F17 | Low | confirmed | Low | - | Manifest/record/anchor writers use a fixed <path>.tmp with os.WriteFile which follows symlinks, allowing file clobber when backing up into a shared directory. |
| A3-F18 | Low | confirmed | Low | - | Per-member credit-limit cuts take effect at once although the policy table requires GOV-003 notice, sidestepping the 14-day notice on default-limit cuts. |
| A3-F19 | Low | confirmed | Low | - | Authentication throttling limits are constants, not operator settings, although OPS-008 lists them as required settings and it is marked built. |
| A3-F25 | Low | confirmed | Low | - | Erasure blanks every inbox body in the network containing the display name as a substring, so a 1-character name wipes other members' notifications via self-ser |
| A3-F07 | Info | confirmed | Info | - | Erased values remain in -wal frames after the normal PASSIVE checkpoint; the RB-003 test passes only because it forces TRUNCATE. |
| A3-F12 | Info | confirmed | Info | - | A schema_migrations row with version <= 0 makes every open panic (all[v-1]) instead of failing cleanly. |
| A3-F20 | Info | confirmed | Info | - | Forbidden-name scan runs tar -tzf on .zip archives, so the Windows archive is never inspected. |
| A3-F21 | Info | confirmed | Info | - | Docs say '15 min doubling to 24 h' but code locks a fixed 15 min (throttleMax). |
| A3-F22 | Info | confirmed | Info | - | break-glass grant-admin -login is not normalised for case, unlike every other operator command. |
| A3-F23 | Info | confirmed | Info | - | admin create-operator is audited as the new person registering themself; nothing marks it as an operator action. |
| A3-F24 | Info | confirmed | Info | - | Erasure drops only the erased person's in-memory artefacts; another administrator's full export made before the erasure stays downloadable for up to 24 h. |
