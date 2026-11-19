# Notification source reconciliation

## Disposition

Preserve both implementations and their histories. The recovered notification module is substantial implemented source, not a missing feature or a documentation-only checkpoint. It is also not a superset of the currently reviewed module. A whole-folder replacement would discard current identity binding, session-specific subscriptions, email verification, and the reviewed transport implementation. No schema, event taxonomy, retention, unsubscribe, or activation policy is selected by this report.

This was a read-only source comparison. No source, reference, PR, workflow, or dependency was changed, and no additional build or runtime probe was started. Fresh archive qualification belongs to the separate serial runner. Historical archive passes and current PR103 passes remain evidence for their own source only.

## Exact provenance

| Item | Identity |
|---|---|
| Current reviewed local commit | `4a59bd1e740c2fcf5877de2a90594439cb9cd570` |
| Published current commit / PR | `13669485b641475fce526633f47055ef31997098` / PR103 |
| Current complete tree | `198d2e6d9e40922080eb91f3ab73d011808feea9` |
| Current notification subtree | `32ebe377c0710d2e6528628e35cf6e0c02028ed8` |
| Recovered commit | `ba454e1c14d35bdd90dca8379e890f2bca6281d7` |
| Recovered complete tree | `14fa7ab8962fa2282c3877027aae08d19df47b73` |
| Recovered notification subtree | `be42783f6bbb7cc1fdc443cbcff15fa54e540fc3` |
| Recovered recorded tested runtime | `9f4c4da33eba7a668fb8e8ddbe7f2a7718d0bc5e` |
| Common ancestor of the compared commits | `5290916d116387a283a9ec344df112f00bccf7c9` |

All 20 recovered Go files read from the qualification extraction match their archived Git blobs. Those Go blobs also match the archive's recorded tested runtime. That identity check does not turn dated test records into fresh execution evidence.

Relative to the reviewed current module, the archive has **78 added, 8 modified, and 27 deleted paths**. The Go-only difference is **16 added, 3 modified, and 25 deleted files**. Much of the added path count is documentation and dependency notices. The earlier 5/15-source comparison remains unchanged: it compared the archive with `9c794a5` and recorded 78 additions, 8 modifications, and 25 deletions. The two additional deletions in this comparison are the transport tests added by PR103. The original comparison file and its hash are recorded in `source-provenance.json`.

In the discussion below, **C** means `future/notifications/` at current `4a59bd1`; **A** means that same prefix at archive `ba454e1`. Exact blobs, all changed paths, and the readable archived copies are listed in `source-provenance.json` and `module-name-status.txt`.

## Useful recovered implementation and tests

| Recovered capability | Exact source and regression entry points | Composition consideration |
|---|---|---|
| Explicit scoped HTTP and native preference form adapters | A `api.go`: `Handler.ServeHTTP`, `decode`, `RenderPreferenceForm`, `localFormAction`; `api_test.go`, `audit_test.go` | Current C `Service` has preference and target operations but no equivalent module HTTP/form adapter. Recover the adapter behavior through an explicitly chosen interface; A handlers directly depend on `Repository`, different preference shapes, and a different event set. |
| Host callback and input ordering defenses | A `master_audit_test.go`: `TestMasterQueryValidationBeforeHost`, `TestMasterCallbacksRequireCurrentSession`, `TestMasterMalformedFormCannotBeCachedByActor`; A `Handler.ServeHTTP` | Valuable regressions for malformed/ambiguous queries, cached partial form input, and actor changes during CSRF callbacks. Current production auth/session adapters are still absent; these tests do not qualify those adapters. |
| Strict body parsing and parser fuzz seed corpus | A `strictjson.go`: `uniqueJSON`, `validUnicodeEscapes`; `strictjson_fuzz_test.go`: `FuzzStrictPreferenceJSON`; A `decode` | The 64 KiB body cap, trailing-value rejection, Unicode escape checks, and depth bound are useful. The duplicate-name preflight folds ASCII only; see the finite compatibility finding below before adopting it. |
| Consent cancellation with write postconditions | A `Repository.SetPreference`, `revoke`, `requireOne`, `requireNone`; `consent_lifecycle_test.go`; `TestMasterRevocationScopeAndRestart` | A checks failed or silently ignored SQL writes and cancels ready/retry/sending rows transactionally. C already cancels ready deliveries on opt-out and rechecks consent before send. Port the failure/postcondition tests and intended guarantees, not A SQL or mutex assumptions unchanged. |
| Consent grant does not resurrect disabled-period work | A `Repository.SetPreference`; `TestMasterOptInDoesNotResurrectDisabledInterval` | A `EnqueueTx` can enqueue regardless of preference. C `Service.EnqueueTx` already skips disabled or missing preferences and inactive members. The exact archived bug must not be asserted against C without a new failing case; any port must account for C's claimed-row/lease semantics. |
| Lifecycle operations and recovery tests | A `operations.go`: `RotateEncryptionKey`, `Depart`, `TargetVersion`, `TargetFeedback`; `TestKeyRotationRollbackFeedbackDepartureAndRetention`, `TestRotationFailureAndConcurrentCapabilities`, `TestRotatedSecretClosedBackupRestore` | Recover tests for atomic rotation, failed writes, closed backup/restore, departure, and stale provider feedback. C uses a keyring, separate token/index keys, row IDs, and multiple push targets, so A's single-cipher implementation cannot replace C rotation directly. |
| Retention replay tombstones | A `RetainUntil`, `EnqueueTx`, `notification_tombstones` schema; lifecycle/retention tests | C `Purge` removes terminal rows and their dedupe identity and retains audit rows. A retains opaque IDs permanently and can remove old audit rows. This is a policy and schema decision, not an automatic repair or a demonstrated current replay exploit. |
| Private unsubscribe confirmation adapter | A `unsubscribe.go`, `unsubscribe_http.go`; `TestUnsubscribeExpiryTamperAndTargetVersion`, `TestCapabilityConfirmationEndpointAndUnicode`, `TestUnsubscribeAuditFailureRollsBackConsentAndTarget` | Useful GET-without-mutation, private confirmation, target-generation, and transactional consent tests. A's seven-day, target-bound, encryption-key-bound capability is incompatible with C's existing non-expiring channel link and automatic email headers/body. Preserve both designs pending an explicit contract decision. |
| Read-only operator command | A `cmd/notifications-status/main.go`: `run`; `TestDiagnosticsScopeRedactionAndReadOnly`, `TestDiagnosticsUsage` | C has `Status` and `ListDeliveries` service primitives but no equivalent standalone executable. Preserve A's read-only open, scoped query, bounded rows, escaped database path, and redacted failure tests while adapting SQL/output and the chosen driver. |
| Additional persistence and worker tests | A `TestDurableDedupeRollbackPrivacy`, `TestRetryCapAndCrashLease`, `TestCrashOnFinalAttemptIsTerminal`, `TestQuietHoursDoNotSpendAttempts`, `TestPermanentFailureRevokesTarget` | C already has durable leases, bounded retry, crash hooks, multi-target receipts, and dispatcher tests. These archived cases are valuable additional scenarios, not evidence that the current implementation lacks all of those mechanisms. |

## Current protections and behavior that must survive composition

### PR93 identity and target binding

C `targets.go:sealAAD` binds kind, network, member, and target-row identity. Pipe-free tuples retain the compatible `ntf/v1` representation; delimiter-bearing tuples use length-prefixed raw bytes in a disjoint `ntf/v2` namespace, without fallback to ambiguous legacy AAD. `target_binding_test.go` proves separation, every component's authentication, retained safe legacy data, and refusal of ambiguous old ciphertext.

C `Actor.validate`, `Event.validate`, `DedupeKey`, and `signToken` reject malformed UTF-8 before JSON can change an identity. `identity_encoding_test.go` includes positive valid Unicode/U+FFFD controls. C `OnLogout` and the dispatcher retain session-bound push subscriptions and check `Directory.SessionActive` before sends. These capabilities are removed by an archive folder replacement: A has no corresponding Actor/session target model or keyring file.

A's JSON tuple AAD is unambiguous for ordinary valid Unicode strings, including delimiter-bearing names. It should not be described as having C's old pipe-delimiter collision. Its identity-validation gap and incompatible ciphertext format are separate concerns, detailed below.

### All four PR103 transport paths

Preserve these exact reviewed files and the interfaces/support code they use:

| Path beneath `future/notifications/` | Reviewed blob |
|---|---|
| `pushsend/pushsend.go` | `cf4e6d876137f5756d0bba6a57536596f5541025` |
| `pushsend/pushsend_test.go` | `e4e82c4be240f8c5314e855ae710bf395f9f8c85` |
| `smtpsend/smtpsend.go` | `e4a473915835db4d219a2118a6fa448f91921537` |
| `smtpsend/smtpsend_test.go` | `356b32bce1661324e7f2cb721d05b18589fa750a` |

Those files need C `transport.go`'s `EmailSender`, `PushSender`, message types, `SendError`, and retry cap; C `egress.go`'s policy/client; C secret handling; and the synthetic `notifytest` fixtures. Copying only the four files into A will not compile against A's `Sender.Send(context.Context,string,Target)` interface.

C retains actual verified SMTP TLS, custom-dial implicit TLS handshaking, before-credential failure tests, Web Push success as a true nil interface, overflow-safe capped Retry-After, response limits, checked dial addresses, redirect refusal, payload decryption, and concurrent message identities. The existing qualification receipt and independent remote review remain attached to the exact PR103 tree.

A also has real transport source. Its `Transports.Send` uses mandatory STARTTLS, returns literal nil on success, and `sendPush` returns literal nil for 2xx. It has no Retry-After parser and no implicit-TLS/custom-dial mode. Therefore it is inaccurate to claim the same typed-nil, overflow, or custom-dial TLS bugs exist in A; replacing C would instead drop those configured capabilities and their stronger regression coverage. A's SMTP test checks downgrade refusal, and its push test checks encrypted request construction without decrypting the received payload. These do not replace C's full adapter tests.

## Finite source findings and public-entry reachability

### Archived JSON identity normalization

This is a source-level parity finding. No authorization bypass, cross-tenant replay, or end-to-end exploit was demonstrated during this read-only task.

| Symbol | Reachability and existing checks | Boundary that remains |
|---|---|---|
| A `delivery.go:DedupeKey` | Exported pure helper. It checks nonempty network/event/member/kind and a recognized channel, then JSON-encodes all five strings. | No UTF-8 rejection. Distinct malformed byte strings can receive the same JSON replacement encoding before hashing. C explicitly rejects those inputs. |
| A `store.go:EnqueueTx` | Public source-event entry point. It checks nonnil transaction, recognized kind, time/length bounds, then calls `DedupeKey` before inserting. | Network/member/event are not checked for valid UTF-8 here or in the archived helper. A minimal helper guard can protect this call path without changing event or expiry policy. |
| A `store.go:SetTarget` and `aad` | Public method checks nonempty network/member, channel, and target syntax, then calls the supplied `Authorize` callback. `aad` JSON-encodes network/member/channel before AES-GCM sealing. | No module UTF-8 guard for those identities. Reachability requires the trusted callback to accept the supplied strings; an actual host may reject them, but that adapter is unqualified. A `Handler` obtains these IDs from its trusted `Actor` callback, so strict request-body UTF-8 checks do not establish identity validation. |
| A `unsubscribe.go:UnsubscribeToken` | Public method calls `Authorize`, reads the target by the supplied IDs, and JSON-encodes those IDs into an encrypted grant. | No UTF-8 check before grant serialization. Normalized identity bytes can differ from caller input. Downstream target-version comparison can still reject a mismatching grant; no successful unauthorized revocation is claimed. |
| A `operations.go:RotateEncryptionKey` / private `Repository.target` | Read existing identity columns from storage or the outbox and feed them into `aad`. | A future repair/migration must consider existing data. New-entry guards alone do not prove that all historically stored identities are valid. Do not guess or silently rebind old ciphertext. |

The pinned Go encoder's invalid-byte branch appends `\ufffd` (`encoding/json/encode.go`, `appendString`). A `json.Marshal([3]string{n,m,c})` AAD does not preserve arbitrary raw invalid bytes. This reasoning is supported by inspected source and C's already-qualified identity regressions; a new archived runtime probe was deliberately deferred while the shared SQLite build slot was occupied.

The parent has subsequently authorized a separate, narrowly scoped archival-source repair for reachable malformed-UTF-8 inputs. That repair must have its own exact base, fail-before evidence, valid Unicode/U+FFFD controls, and review; it is not included in this source comparison or published onto either current runtime branch.

### Archived strict-JSON duplicate aliases

A `strictjson.go:uniqueJSON` folds only ASCII A-Z before checking duplicates. Go's typed decoder also matches long-s and Kelvin equivalents of ASCII letters. For example, the valid JSON object members `"Kind"` and `"Kind"` are distinct in A's preflight but refer to the same field in A `Handler`'s typed input. Both values can be valid notification kinds, so the later value can silently replace the earlier value despite the duplicate-field contract. `"QuietStart"` and `"Quietſtart"` have the same issue.

This is a source-confirmed parser-contract discrepancy, with actual Go alias semantics independently established during the export task. It was not executed through the archived HTTP handler here. Actor, membership, and CSRF checks still apply; it is not described as an authorization bypass. Preserve intended unknown Unicode fields and use Go-compatible simple folding if this parser is adopted. This issue is outside the subsequently authorized identity-only patch.

## Contract decisions that cannot be silently combined

| Area | Current C contract | Archived A contract |
|---|---|---|
| Runtime API | `Service`, transaction-capable `Store`/`DBTX`, `Directory`, `Dispatcher` | One shared `Repository`, exported `*sql.DB`, `Authorize`, serialized `DispatchOne` |
| Schema | Versioned migration files; `network_id/member_id`; separate email/push targets; `notification_delivery` and per-target receipts | Inline `Schema`; `network/member`; one `notification_targets` row per channel; `notification_outbox` and tombstones |
| Events | Five selectable kinds; `proposal_resolved` combines accepted/declined | Six selectable kinds; accepted and declined are separate |
| Target ownership | Multiple push subscriptions with session binding; email confirmation and rate limits | One target per channel; email must be host-verified; no stored subscription session ID |
| Secret format | Versioned keyring ciphertext; raw-byte AAD rules; independent token/index keys | One AES-GCM cipher and JSON-tuple AAD; same cipher also protects unsubscribe grants |
| Unsubscribe | Non-expiring channel-scoped signed link, automatically placed in email body and headers | Seven-day encrypted target-version grant, invalidated on replacement/rotation; private delivery still unfinished |
| Retention | Purges terminal deliveries/old revoked targets; retains append-only audit | Deletes old terminal metadata/audit and retains permanent opaque replay tombstones |
| Transport/payload | STARTTLS or implicit TLS; typed failure/retry classes; versioned JSON push body; configured provider policy | STARTTLS only; generic/permanent failure split; fixed text push body, TTL60; exact provider-host list |
| SQLite qualification | Tagged `sqlite_cgo_test` test driver; current published module workflow | Default modernc SQLite in tests and operator CLI; different nested dependency graph |

Both dependency files pin the same go-mail and Web Push versions. A additionally pins modernc SQLite and a different x/crypto pseudo-version; C has the tagged mattn driver and a Go toolchain directive. This report does not infer a current vulnerability from that version difference and does not recommend an unreviewed dependency or workflow change.

## Ordered composition plan

1. **Pin both histories and evidence.** Keep `ba454e1` and its documented runtime/evidence intact. Keep PR93/current identity tests and the four PR103 transport blobs intact. Do not overwrite either runtime or the existing 5/15 source inventory.
2. **Make the independently authorized identity repair on an isolated archival copy only.** Prove failures at the exact archived base; reject only malformed UTF-8 on reachable serialization paths; preserve valid Unicode/U+FFFD, current archive schemas, kinds, TTLs, and retention. Save a portable patch and qualification receipt; do not publish a competing runtime implementation.
3. **Record the unresolved API/data decisions before composing executable source.** Resolve target cardinality/session binding, verified email authority, event mapping, token lifetime/delivery, audit/tombstone retention, and ciphertext migration explicitly. Neither folder's schema or public API is a safe implicit default. Existing experimental ciphertext must have a documented compatibility path or explicit re-registration procedure; never attempt ambiguous fallback.
4. **Carry security invariants into the selected composition first.** Keep PR93's identity/legacy-binding controls, session-specific logout behavior, and per-target authorization checks. Preserve all four PR103 transport files and either preserve their supporting types/policy/fixtures or introduce an explicit tested adapter. Do not hide a transport or ciphertext format change behind a file move.
5. **Adapt recovered behavior in small units.** Bring over strict HTTP/form and CLI cases through the selected service interface, correcting the known duplicate-alias discrepancy. Adapt consent/postcondition, rotation/backup, feedback, departure, and retention tests to the selected schema. C already filters disabled events during enqueue; A's grant-time cancellation should not be copied as if that same defect were demonstrated in C. Claimed-row cancellation must be reconciled with C's lease checks and send boundary, not just SQL status names.
6. **Qualify the resulting exact tree and retain HOLD.** Run the selected actual SQLite driver suite, race/vet, parser fuzz, both transport acceptance suites, original identity/AAD/session tests, and meaningful lifecycle failure/restart tests. Compare the complete changed-path set and preservation blobs independently. Reuse dated evidence only for provenance. Actual host transaction/auth/session/CSRF, verified email, browser/accessibility, provider interoperability, private link delivery, authenticated DSN correlation, VAPID operations, per-attempt audit, and secret/backup custody remain separate acceptance gates. No production registration or activation follows from this report.

The plan deliberately preserves both implemented designs while identifying reusable source and tests. It neither reports the recovered module as missing nor converts its historical completion score into present feature or release approval.
