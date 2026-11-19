# Independent source review

**No unresolved blocking source finding remains in the reviewed candidates. They are not yet cleared for merge.** Current-head CI, the generated traceability handoff and normal repository approval/protection gates remain outstanding. This is an engineering review record, not a GitHub approval or a policy exception.

## Exact reviewed publications

| Candidate | Published code head | Local reviewed head | Scope |
|---|---|---|---|
| Core export verification [#104](https://github.com/djshellshoxxx/circula/pull/104) | [c3c991aa3c12](https://github.com/djshellshoxxx/circula/commit/c3c991aa3c12a11edce2874e9444b9d2a99f746c) | `6e8f5f1473084e7a14beccb92d3db9ea547554d4` | Five files; tree `f97a12c04853c298b56dec0bfde72fd0b500b722` |
| Isolated recurring [#97](https://github.com/djshellshoxxx/circula/pull/97) | [73a911a9bec3](https://github.com/djshellshoxxx/circula/commit/73a911a9bec387381633c031d8a4083c9fc724ad) | `cdcd2cb38b3af0275e5ceb58242384cfa916be45` | Eight files; tree `c6a4c5dabdbc482967a44353f14926a24066ad56` |
| Isolated notification transports [#103](https://github.com/djshellshoxxx/circula/pull/103) | [13669485b641](https://github.com/djshellshoxxx/circula/commit/13669485b641475fce526633f47055ef31997098) | `4a59bd1e740c2fcf5877de2a90594439cb9cd570` | Four files; tree `198d2e6d9e40922080eb91f3ab73d011808feea9` |
| Isolated federation [#105](https://github.com/djshellshoxxx/circula/pull/105) | [059b0e4508a6](https://github.com/djshellshoxxx/circula/commit/059b0e4508a6c4f8e29f6a4cccd06f92b14ea115) | `db21ea0a0008309d6e8d116ccb00eb77ac6873d9` | Three files; tree `4d9f608041e65a2a1a297f17614438790240bc04` |

The reviewer read the complete source/test diffs, fetched every changed published file at the exact SHA, and checked both the complete path sets and all Git blobs against the reviewed local commits. The Core comparison uses main `ead63b5`. Recurring uses its original isolated base `0437e548`. Notifications publishes on merged target `797dddf`, whose tree is independently confirmed equivalent to the original reviewed base `9c794a5`. Federation uses its original isolated base `1b3c92e`. The JSON contains the exact receipts.

## Core export verifier review

Reviewed paths:

- `cmd/circula/verify_export.go` — blob `bb00a63f67b0c2beee23db030643e7d0f55f26ae`.
- `cmd/circula/verify_export_ambiguity_test.go` — blob `8d3e9d71e108035c693f04588db99291fbc61dc4`.
- `cmd/circula/verify_export_boundary_test.go` — blob `9ce07c566ab4130f22b05f53100eb749ff763955`.
- `scripts/read-export.py` — blob `d6d97a0c19c8a4c30bd97845bdcd4d23346f4dac`.
- `scripts/test_read_export.py` — blob `afd4ceaae544a44f1515d2752e52fc48a985dff5`.

The Go files match author commit `bf02b29b514a5fd0f7c8cc33aaf5d0ddb352f5ff`. The Python files match corrected reader commit `9b97bf94e5bf820bd0013939e1d487635ea5ce35`. All four local integration commits preserve those contents; no API-036, feed, storage, shared workflow or generated-document source is included in the five-file diff.

The review covered portable member identities, duplicate members and manifest declarations, special-file metadata, schema versions, explicit versus missing/null file arrays, metadata size preflight, actual bounded reads, error-associated partial bytes, ZIP handles and backward compatibility for valid packages. An explicit empty manifest array remains supported; nil/missing arrays do not.

### Review findings that were repaired

| Finding | Observed defect | Resolution checked |
|---|---|---|
| Raw ZIP NUL names | Python's normalized filename could conceal a suffix before validation. | Validate `orig_filename` and require unchanged identity; regressions patch real local and central ZIP names. |
| Conflicting directory metadata | A DOS directory bit could disagree with other creator/mode metadata. | Both readers reject it for all tested creators, including Unix-mode regular entries. |
| Schema and array shape | Go accepted absent/unsupported schema or nil files; Python accepted boolean `true` as version 1. | Supported schema, exact Python integer type and explicit array requirements. |
| Unchecked read suffix | A cap could return a valid prefix without proving that the source ended. | Size preflight and cap-plus-one reads; partial data is discarded on failure. Tiny-budget tests avoid enormous allocations. |
| Actual ZIP size | Python's ZIP reader truncated output to attacker-supplied metadata before the outer cap could see a suffix. Overstated lengths also passed. | Bounded full decompression through a copied `ZipInfo`, followed by exact original-size comparison. |

### Independent archive checks

At Python commit `778d534`, both stored and deflated members with a 25-byte valid JSON prefix plus a hidden 25-byte suffix returned `export ok` when size and CRC described only the prefix. Declaring 26 bytes for an actual 25-byte member also returned success. These are observed false successes, not hypothetical parser concerns.

At corrected commit `9b97bf94`, the reviewer reran the same six existing fixtures:

| Fixture | Expected | Observed |
|---|---|---|
| `deflated-size-overstated.zip` | Reject malformed archive | Exit 1; PASS |
| `deflated-size-understated.zip` | Reject malformed archive | Exit 1; PASS |
| `deflated-valid.zip` | Accept valid archive | Exit 0; PASS |
| `stored-size-overstated.zip` | Reject malformed archive | Exit 1; PASS |
| `stored-size-understated.zip` | Reject malformed archive | Exit 1; PASS |
| `stored-valid.zip` | Accept valid archive | Exit 0; PASS |

No failing fixture emitted a traceback. The source/test review also checked the complete-data-CRC regression, which ensures the explicit size comparison is required even when CRC verification itself succeeds. The separate Go author checked the same four malformed archives through `archive/zip`: understated data fails with `ErrFormat`, and overstated data with `unexpected EOF`; the bounded Go helper discards those error-associated bytes.

## Isolated recurring review

The original `Due` API, `Template` layout and valid daily logical keys are preserved. The new structured daily/weekly/monthly rules reject contradictory selectors, follow the existing missing-month-day skip policy, and inspect only the current local date without catch-up. Invalid UTF-8 identities and host-dependent `Local` are rejected.

The bounded interval resolver walks timezone interval starts backward, computes candidate instants using exact second offsets, and round-trips civil date/time fields before selecting the earliest fold occurrence. Gaps are skipped. An explicit found flag preserves a valid zero instant. The review checked the independent 3,456-case minute oracle, separate historical second-offset fixture, DST/calendar regressions and fuzz invariants.

The dedicated workflow uses immutable existing action pins, read-only repository permissions, no persisted checkout credentials and explicit deadlines. Ubuntu/macOS run race tests; Windows runs the ordinary suite with CGO disabled. All three check formatting, vet and disabled/HOLD/module boundaries. Bounded fuzzing runs on Linux only. There is no implied Windows race pass or production adapter qualification. Durable scheduling, authorization adapters, crash recovery, UI acceptance and policy adoption remain open.

## Isolated notification transport review

The reviewer read both complete transport implementations, both new test files, the synthetic SMTP server, independent Web Push decryption fixture, existing egress code, and pinned `go-mail v0.8.2` dial/TLS/authentication paths.

The push change returns a true nil error for successful HTTP statuses and clamps numeric `Retry-After` values before duration multiplication, including integer overflow. Tests exercise actual encrypted payload decryption, checked public-address dialing, private-address refusal, no redirect following, bounded/closed response bodies, error redaction, cancellation and provider status classification.

The SMTP custom-dialer path now completes name/CA-verified TLS with a minimum TLS 1.2 before any connection is returned. Failed handshakes close the underlying connection. The dependency's `TLSOpportunistic` option is used only to discover this already-encrypted connection state; there is no plaintext fallback. Tests verify STARTTLS and implicit-TLS success, certificate/name/downgrade failures before AUTH or MAIL, the unchanged default implicit dialer, fixed message headers/body, error hygiene and concurrent message identities.

These tests use synthetic loopback providers. They do not establish real SMTP/Web Push provider acceptance or enable this disabled/HOLD module. The four published blobs exactly match the reviewed local source.

## Isolated federation review — subsequent publication

The complete three-file diff at local `db21ea0a0008309d6e8d116ccb00eb77ac6873d9` was reviewed after the initial three candidates. Every published blob and the complete path set at [PR #105](https://github.com/djshellshoxxx/circula/pull/105), head `059b0e4508a6c4f8e29f6a4cccd06f92b14ea115`, matches that reviewed source. The shared tree is `4d9f608041e65a2a1a297f17614438790240bc04`.

This is a signing-input fidelity repair. `Sign` previously accepted malformed UTF-8 and returned a normalized envelope that did not preserve the caller's string bytes. **The existing verifier already rejected those envelopes through exact JSON re-encoding; no signature-verification or replay-identity bypass was reproduced.** The repair checks all five signed strings before marshaling and returns `ErrInvalid` with an empty envelope. Valid payload encoding, domain prefix, key handling and `Verify` behavior remain unchanged.

The review checked 30 malformed-input regressions, exact valid Unicode/literal U+FFFD/decomposed-text encoding, independently signed malformed representations that retain existing verifier rejection, and bounded property tests for exact string round-trip, canonical payload bytes and Ed25519 authentication. The new dedicated workflow follows the reviewed native-platform, permission and deadline pattern; its root toolchain pin is Go 1.26.11. Federation remains disabled/HOLD. Broader replay, queue, trust, transport and lifecycle work is outside this narrow PR; the later recovered implementation is recorded separately and requires reconciliation and qualification. Full product acceptance is not established.

## Subsequent Python manifest review

The complete two-file follow-up from `6e8f5f1473084e7a14beccb92d3db9ea547554d4` through `56c09536a01fe6658ee98fe573029eac77c6ae82` to **`9e95b73861be23d0f46121f534e6c3e46d83bc1a`** was independently reviewed. Only `scripts/read-export.py` and its test file change; their exact blobs are in the JSON. There is no remaining blocking finding in this repaired scope.

The first follow-up rejects wrong manifest field types, out-of-range integers and malformed timestamps before earlier duplicate occurrences can be hidden. Review then reproduced additional real ZIP false passes: Go-recognized uppercase/long-s field aliases bypassed validation, a later `ROWS` value was ignored, and nonstandard `NaN`/`Infinity` values in otherwise ignored fields were accepted. The final follow-up fixes all three cases. It uses the pinned Go field-folding equivalences, retains original member order and scalar-versus-collection null rules, and rejects nonstandard constants in both manifest and member JSON. Valid duplicate fields, unknown extensions, ligatures, valid Unicode and the pinned Go timestamp decoder's existing legacy spellings remain covered by controls.

The reviewer independently replayed eight small alias/constant ZIPs: **seven malformed packages reject and the unknown-ligature control succeeds**. The six previous stored/deflated ZIP-length fixtures were rerun against the same reader: **both valid controls succeed and all four mismatches reject**. The author also supplied an actual standard-library Go decoder run with the exact manifest structs; its six type/syntax errors reject, its ligature control accepts, and its later `ROWS` value decodes to 1. That probe is explicitly a decoder check, not a full CLI qualification.

The author reports all **38 reader tests passing in 23.950 seconds** at `9e95b738`. The parent composed those changes into local `3cc231542e7c46793f27b2fc406b9220fabf1bff`, with tree `ca441d471000c1ae24ca87c28255a0c467c6e2f9`, and reports another **38-test pass in 17.805 seconds**. Its three Go blobs remain equal to the earlier reviewed source and its two Python blobs equal this follow-up. This integration report is kept separate from a future remote publication check; the earlier code-stage CI observations do not qualify a later head.

## Relationship to recovered future source

[ARCHIVE_RECOVERY.md](ARCHIVE_RECOVERY.md) records the separately verified unpublished source archive. Broader services and persistence found there are separate from the narrow primitive/transport candidates reviewed above. The archived recurring daily helper and federation v1 envelope helper retain their old defects, so a recovery composition must preserve this pass's fixes and tests. A proposed recurring Calendar adapter also needs its own review and fresh qualification. No whole archived implementation is approved by the earlier source receipts, and historical tests are not current passes.

## Recovered analytics workflow and provenance review

The independent review covered all 36 changed paths at local `e1f81f989c24ec3eaea73594083c8d1d2957a6e3` and the final two-line documentary correction at **`06a14e4bc6e5bca8bb0d9b857d26f1d280821aba`**, tree `5bee54b3ed39c4fe680f02dfc2b3bc48c64c36d6`. All 30 archived module files remain; 29 are byte-identical. All eight Go source/test files and `go.mod` are unchanged from archive `ef4bdf87d2bc915b26d84e548ea4e6a478079006`, whose ancestry contains the advertised target. The README now explicitly identifies the dedicated qualification workflow; its exact candidate blob is reflected in the provenance manifest.

The reviewer read the complete new workflow and recovery documentation. The workflow retains disabled/HOLD and module-isolation checks, immutable action pins, read-only permissions, native Linux/macOS race tests and Windows ordinary tests, and a bounded Linux 100,000-execution fuzz command. Historical coverage/completion figures remain separate from the new receipts. Fresh independent Linux race testing produced 19 test pass events and no skips, with 86.4% statement coverage; vet, formatting, checksums and unchanged-source verification passed. The parent separately reviewed all recovered runtime/test source and ran 100,000 actual fuzz executions, passing in 13.740 seconds (156.507 seconds including its cold build). There is no blocking finding in this assigned review scope. The reviewer subsequently verified [draft PR #108](https://github.com/djshellshoxxx/circula/pull/108), head `29a0312c5c4d446eb47c4f1d4555b58ff52c99b8`, with exact reviewed tree `5bee54b3ed39c4fe680f02dfc2b3bc48c64c36d6` and sole parent `0b2965101e1d5d1e3e6447e7c24197efc7f0cb9f`. Its complete 36-path remote set matches the reviewed boundary. Dedicated native run `44522482254` and Core run `44522482048` were both queued at this observation; no provider pass is claimed.

## Recovered recurring composition review

The reviewer inspected the complete new `Calendar` delta, reconciliation tests, dedicated workflow and provenance at **`048c7156133bef0b6a7d394c4e7814a975f1d8b2`**, tree `e074e3a49b7a6271483831e576d9ef2f4df90f8e`, module tree `f24dfcdbf115f65ad662df59ac02c624defc21ce`. All 82 archived module files remain, 75 byte-identical. The previously reviewed `schedule.go`, its original tests and the new regression suite are preserved byte-for-byte. The prior `calendar_test.go` is retained under `due_with_schedule_test.go`, alongside the archive's distinct calendar tests. No unique tests were discarded to resolve that filename collision.

The direct `Calendar` API now rejects malformed UTF-8 identities before JSON key construction. Its service boundary already rejected these identities, so no service authorization bypass was reproduced. `Window` uses the retained exact timezone-interval resolver. `Next` uses cron only to enumerate eligible Gregorian dates at UTC noon, resolves each date's exact local time, and stops after at most five candidate dates. This fixes both the original historical-offset failure and the reviewer's reproduced next-date failure after the scheduled time had passed. Existing zero-clock rejection, first-fold selection, missing-day/gap skipping and key compatibility are retained.

The tests include 24 malformed-identity negatives with valid Unicode controls, 12 historical second-offset schedules at before/exact/after boundaries, an independent 10,368-comparison minute oracle across three cadences, and 12 explicit next-date gap/fold/dateline/month-end/leap cases. The original 3,456-comparison helper oracle remains intact. The author reports the combined calendar-only race command passing in 3.343 seconds. That command excludes SQL; it does not qualify the recovered service. The workflow now checks exact recovered dependency pins and imports, runs full real-SQLite suites separately from two parser-only fuzz commands, and preserves disabled/HOLD, native-platform, permission and timeout requirements.

There is no remaining blocking finding in the assigned Calendar/workflow/provenance delta. The parent is separately reviewing recovered service/store/HTTP/CLI production code and identified a `Cleanup` timestamp-ordering defect requiring its own correction and real-driver regressions. Full composed-module qualification and that follow-up remain pending. This receipt does not approve the entire recovered service for publication or integration.

## Qualification evidence and limits

The reviewer independently ran the six Python archive probes and performed the source, diff, path-set and remote-blob checks above. Broad suites were not redundantly rerun by this reviewer. The following results were reported by the implementation owners or parent and remain clearly separate from provider CI:

- The earlier code-stage combined Python suite passed all 22 tests in 9.901 seconds; the subsequent 38-test follow-up and its exact source identities are recorded above.
- The Go author's final focused real-command export tests passed in 0.030 seconds with `-vet=off`, after an explicitly recorded compiler-memory failure. This does not substitute for a complete package/repository run.
- The full final Core ordinary test attempt failed: the CLI test process, API linker and service test compiler were killed. Contract, store, web, domain and tooling selections passed, but a complete Core pass is not established.
- The recurring author reported Linux race/vet and 100,000 fuzz executions on the final runtime/test contents. Native provider results remain separate.
- The notification author reported adapter race/coverage and vet passes, then 10,000 Retry-After fuzz inputs in 2.567 seconds after recording an interrupted attempt. The exact-current full tagged notification race suite and full tagged vet also passed, as separately reported by the author and parent.
- The federation author reported a committed-head Linux race pass in 1.071 seconds at 87.5% statement coverage, vet, and 100,000 fuzz executions in 5.682 seconds. These were not independently rerun by this reviewer; native provider results remain separate.

### Provider observations at the code-stage heads

| PR | Workflow | Run | Observed state |
|---|---|---|---|
| [#97](https://github.com/djshellshoxxx/circula/pull/97) | future recurring module | [44508676983](https://github.com/djshellshoxxx/circula/actions/runs/44508676983) | queued; conclusion not available |
| [#97](https://github.com/djshellshoxxx/circula/pull/97) | ci | [44508676822](https://github.com/djshellshoxxx/circula/actions/runs/44508676822) | pending; conclusion not available |
| [#103](https://github.com/djshellshoxxx/circula/pull/103) | ci | [44511002032](https://github.com/djshellshoxxx/circula/actions/runs/44511002032) | queued; conclusion not available |
| [#103](https://github.com/djshellshoxxx/circula/pull/103) | future notifications module | [44511002005](https://github.com/djshellshoxxx/circula/actions/runs/44511002005) | queued; conclusion not available |
| [#104](https://github.com/djshellshoxxx/circula/pull/104) | ci | [44511241324](https://github.com/djshellshoxxx/circula/actions/runs/44511241324) | queued; conclusion not available |
| [#105](https://github.com/djshellshoxxx/circula/pull/105) | ci | [44513782075](https://github.com/djshellshoxxx/circula/actions/runs/44513782075) | queued; conclusion not available |

The subsequent #105 lookup at 11:20:38 UTC listed only the queued Core run; its dedicated federation workflow was not yet listed. That absence does not establish a pass.

The connector returns a first page of PR-triggered workflow runs; absence is never treated as success. These observations belong to the exact recorded code-stage SHAs. Any evidence-only follow-up is a new head and must receive its own required checks.

## Generated traceability handoff

The reviewer inspected [pending-traceability.patch](pending-traceability.patch), SHA-256 `59a9258c670952bb2faed2aa45ab30b76bf7d73824c034bb4ab775d9661f37ea`. It contains exactly the expected provenance-head change and the additional DAT-040 citation for `TestVerifyExportRejectsDuplicateZIPMembers`, preserving all existing citations and unrelated rows.

`docs/TRACEABILITY.md` itself remains untouched because the active worker owns that generated/shared path. The legitimate test and citation remain in the source. The owning worker should regenerate at the eventual combined head, taking the other active traceability recovery into account; a stale patch should not be applied blindly. **The traceability gate remains failed until the handoff lands and qualification is rerun.** The original shared CI fuzz configuration also remains with its active owner.

## Review boundary

This record does not authorize a merge, protection bypass, feature activation, release or deployment. Normal repository approvals and current-head checks remain required. Later commits must be checked for source drift; a documentation-only follow-up must preserve the reviewed blobs and still receive its required CI. See [BRANCH_DELTA.md](BRANCH_DELTA.md) for the separately frozen remote ownership inventory.


## Archived federation and notification follow-up reviews

Federation composition `bcaf407bfeccb3946d72b298c502bb081c29ba24` (tree `b246abf02b0e35914b4874ce830d7b41685848c0`) was independently compared against all 70 archived files. Sixty-five are byte-identical with the archive README preserved under its new recovery name; the five intended deltas are provenance/feature metadata and the preserved v1 signing repair/package comment. Legacy v1 production body and both test files remain exact. The externally owned README remains byte-identical to PR #105. The complete new v2 signature-fidelity tests, protocol validation, native workflow and recovery documents were read. No blocking finding exists in that composition/test/workflow scope; full recovered service review and fresh combined qualification remain separate.

The later federation head `5cf90f4ea3f9bcd17d4c7eaa611759080895c4dc` (tree `efc696eca11420d14ab526433af267eb6068347c`) changes exactly two recovery scope documents and two accepted-main baseline files. The latter are exact main blobs `e2a0d8ee5d2d340b5222fcc2b86ad50104255587` and `3b117ecfe2f01e7e2363b9d201503cde6e0f5595`. Module runtime/test/dependency/workflow bytes and the reserved README remain unchanged. This bounded delta has no review blocker and establishes no fresh Core pass.

Archived notification identity repair `b96100d2a53cf696f16b8c34305eb132c673863c` (tree `36122c54a1a11333f36700d448a41aedafa7d631`) was reviewed in full: three production files and one new real-driver regression file. Dedupe hashing, target identity/AAD/endpoint marshaling and unsubscribe capability creation reject malformed UTF-8 before lossy encoding. Existing mail parsing and canonical base64 validation already protect the other target fields. Tests preserve valid Unicode, literal U+FFFD and percent-encoded endpoint representations, and check absence of SQL rows after invalid input. No authorization or replay bypass is claimed. The author's whole ordinary module/vet passes are distinct from the still-unqualified race suite; the initial linker kill and successful `-ldflags=-w` resource adjustment remain in the receipts. No blocking issue was found in this narrow repair.

Separate archived notification parser repair `3c734042159899c9d1dc737f70c535e82010c40f` (tree `f3668a89acb046d159525cddf8cd5192467a26de`) changes only `strictjson.go` and its new regression file. The simple-fold orbit comparison covers long-s, Kelvin and Greek sigma aliases while preserving single aliases and distinct sharp-s, ligature and dotted/dotless-I names. The new string builder also avoids repeated string concatenation. Seven actual failing alias subcases were reproduced before the change; the author's production-source-file race/vet result is correctly limited to that parser. Full SQLite/composed-runtime qualification remains separate. No blocking issue was found in the two-file delta.

These review receipts do not bypass current-head CI, normal approval, ownership handoff or integration HOLD. The original notification runtime comparison and PR #103 source remain separate.


## Core checkpoint and completed earlier provider evidence

The reviewer independently verified draft [PR #104](https://github.com/djshellshoxxx/circula/pull/104) at **`707a4363a86922d152ed2d203c9ea184fe4b323f`**, full tree `7ccb0188df8416eb8fa57b8740eba400a5e01363`, with sole parent `c3c991aa3c12a11edce2874e9444b9d2a99f746c`. The tree exactly equals local `87cc4b229d39dbc8c00785048024f0e7296c015e`. Its three Go runtime/test blobs equal the previously reviewed `6e8f5f1` source, and its two Python runtime/test blobs equal the reviewed `9e95b738` follow-up. The new native reader workflow has a separate independent receipt in the published checkpoint. No source-equivalent Go test was unnecessarily repeated.

The earlier [provider run 44511241324](https://github.com/djshellshoxxx/circula/actions/runs/44511241324) has now completed for source **`c3c991aa3c12a11edce2874e9444b9d2a99f746c`**, tested merge `ae7b54a77f24b7ea71099220fe6b50c17b6dc2dc`. Seven of nine jobs succeeded: the Linux/macOS/Windows test jobs, release builds, reproducible build, API contract/load smoke and fuzz smoke. The run still failed. Generated DAT-040 traceability freshness blocked the lint job and its later steps; an APT HTTP 503 blocked browser dependency installation before the e2e suite. [CORE_PROVIDER_CHECKPOINT.json](CORE_PROVIDER_CHECKPOINT.json) preserves exact job and failure excerpts. These results supersede the earlier queued observation for that run, not the new `707a4363` head's required checks. No complete current-head provider pass is inferred.

## Recurring and federation qualification follow-up

The complete five-path federation delta at `18978139341a1d066f58d632de757bbdb4d00999`, tree `0586f86d213b6ccfefdf12fd645ceacef1122219`, was independently read. It moves the exact original `FuzzDiscoveryWire` body, seed literals and `fixtureNow` declaration into `protocol_fuzz_test.go`, and names the actual envelope/protocol/fuzz files in the bounded fuzz command. Production, dependencies and externally owned README bytes remain unchanged. The whole-module tests still contain the same target and fixture. No blocking issue was found in this test/workflow extraction.

The later recurring and federation metadata fixes add `-buildvcs=false` only to dependency enumeration after an actual VCS-status failure, preserving separate Git identity recording and normal suite/vet flags. The complete workflow/documentation diffs were read. The subsequent routing-only commits `45af2b2e31fa9bde532e6952c0b02da6d6e6a231` and `175283209e930d4bc4baeeaa8d66b7d489dd4229` add the existing owned primitive PR source branch as an explicit permitted base for each stacked recovery; runtime, test commands, action pins and permissions remain unchanged.

Fresh combined-module receipts from the author now supersede the previously pending local-qualification state. Recurring at `e79cd7a3d8bc1830aeb5ee255c044ddfef72f278` passed the full real-driver race suite with 48 top-level tests, no skips and 80.0% statement coverage; vet, checksums, formatting and isolation passed, and both actual parser fuzz targets completed 100,000 executions. Federation at `ca9af7543cc560d4a798b6e446d85ec38f740595` passed the full real-driver race suite with 36 top-level tests, no skips and 74.5% coverage; the same ancillary gates and all three 100,000-execution fuzz campaigns passed. The exact commands, resource profile and source SHA-256 maps are retained in the JSON-linked receipts. These results were read, not redundantly rerun by this reviewer. They do not qualify a later publication head's provider jobs or resolve external README/PR #110 ownership gates.

## Areas, advisory governance and messaging recovery

This reviewer authored the three isolated recoveries and read all 19 recovered production files against their contracts. **Independent review was performed by `/root/independent_triage`**, covering the complete new parser/CLI repairs and regressions, relevant callers, every archived-file/test/metadata preservation check, dedicated workflows and the two exact accepted-main baseline blobs. This is distinct from a second independent security audit of every unchanged archived production file. No blocking source/workflow finding remained. The mixed areas fail-before result was corrected without discarding its original receipt or raw log.

| Candidate | Frozen local head | Full tree | Current Linux qualification |
|---|---|---|---|
| Areas | `3275ac6016ca7426427da92e7bfa9f620e3b55ad` | `f5de54ced2f6fd363372a4588c292ab900eeef65` | Complete real-driver race, vet, checksums, formatting/isolation; 100,000 actual parser fuzz executions. |
| Advisory governance | `7ec8147db9fb06674d20983dfcd3b7ac765717ed` | `36ca2f142be776095a97d51cf1caee15d04926e6` | Complete real-driver race, vet, checksums, formatting/isolation; 100,000 actual parser fuzz executions. |
| Messaging | `d0e28040c27f3d336076fdaee51b84869f5f4153` | `951e68d1316d0efd8503408d15a3fc555dd4848e` | Complete real-driver race, vet, checksums and formatting/isolation. Full-package fuzz remains unqualified. |

The final commits change documentation and portable evidence only; their runtime, tests, dependency graph and workflows remain exact to the independently reviewed source. Areas carries forward completed race/vet/checksum/format checks only after complete Go/test/dependency SHA-256 dictionary equality. Later areas parser fuzz and all governance/messaging current checks have their own terminal receipts. The original interrupted governance call and areas resource failures remain preserved rather than recast as passes. All module manifests and historical README bodies remain unchanged.

Each candidate changes its own module, a new dedicated native workflow, and only the coordinator-authorized accepted-main `auditanchor.go` and departure fixture blobs. No current GOV-021/Core governance, active matching/feed/PWA or shared CI work is integrated. The existing messaging private-mode check is expected from source inspection to reject native Windows regular-file modes; no Windows execution or permission workaround is approved by this recovery receipt. A separate Windows adaptation is under review in a different worktree.

The current master directive permits reviewed isolated publication and normal gated merges into the module targets. Old remote-publication prohibitions in preserved manifests describe their earlier authorization state. All three modules remain disabled and on integration HOLD; current-head native/provider checks, normal approvals, host integration policy and manual acceptance remain separate gates. Their local freeze is not a remote publication or merge claim.


### Subsequent remote identity verification for the three recoveries

The reviewer directly fetched each published Git commit, PR metadata and complete paginated changed-path set. Areas [#115](https://github.com/djshellshoxxx/circula/pull/115) at `684ca5132690e24104aa389aac16870c18bbc7fd`, advisory governance [#114](https://github.com/djshellshoxxx/circula/pull/114) at `fbd92a22f06af7bcd96957397958b33e2706807a`, and messaging [#116](https://github.com/djshellshoxxx/circula/pull/116) at `62623635d9711fbafc9b9660ac8860f1eb916dbb` each have their exact advertised target as sole parent. Their full trees and all 104/73/90 changed paths respectively match the frozen local candidates above. All remain open drafts; the original target heads are unchanged. This is a completed remote source-identity check, with no provider or approval bypass.


### Terminal unchanged-archive notification race result

A single justified bounded retry completed the full archived notification race suite at `ba454e1c14d35bdd90dca8379e890f2bca6281d7`, module tree `be42783f6bbb7cc1fdc443cbcff15fa54e540fc3`: 35 top-level tests, 58 pass events, no failures/skips, 176.503 seconds wall time. Library/CLI statement coverage was 71.0%/83.8%. The complete 89-file archive tree was checked before and after; exact dependencies, real SQLite and assertions were unchanged. The private warmed cache, serial low-memory profile and `-ldflags=-w` resolved the previous build resource gap without a driver/version/tag substitution. All fifteen unchanged archived module suites now have passing existing Linux checks. The earlier 14-of-15 snapshot and failed attempts remain preserved. This result does not qualify the later notification repairs or transport composition, and no broader fuzz/native/provider/integration pass is inferred.


### Stacked recurring/federation remote identity verification

The reviewer independently verified open draft [#118](https://github.com/djshellshoxxx/circula/pull/118), recurring head `035dd1c0d7c3de901cb1d2e1854ac4fde4cbdad4`, sole parent `73a911a9bec387381633c031d8a4083c9fc724ad`, tree `d41e47b2a4961983f8881e5162096c588973158c`, and its complete 88-path delta. Open draft [#117](https://github.com/djshellshoxxx/circula/pull/117), federation head `f50aaa4b599c2c0272d37cdc910f567b76c451f8`, has sole parent `059b0e4508a6c4f8e29f6a4cccd06f92b14ea115`, tree `0873c4516d633cc56304c6d35a54fe1e812ab71d` and 78 exact changed paths. Each full remote tree equals its frozen local candidate. Runtime/test/dependency files remain exact to the corresponding tested e79/ca9 source. Descriptive manifest fields changed without changing ID, disabled/HOLD, dependencies, scope or remaining policy/integration gates. This verifies the published source; normal current-head CI, approval and external #110/#106 handoffs remain separate.
