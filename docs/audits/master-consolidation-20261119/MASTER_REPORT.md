# Master autonomous consolidation — engineering report

Date: 19 November 2026 (America/Los_Angeles).

## Outcome and integration status

Current published work includes the following independently reviewed changesets. Further recovered candidates and exact qualification are recorded below and in the execution checkpoint:

1. **Core export-package verification**: [PR #104](https://github.com/djshellshoxxx/circula/pull/104), targeting `main`.
2. **Recurring calendar computation**: [PR #97](https://github.com/djshellshoxxx/circula/pull/97), targeting only `future/fnet-207-recurring-20261116`.
3. **Notification transports**: [PR #103](https://github.com/djshellshoxxx/circula/pull/103), targeting only `future/fux-104-notifications-20261116`.
4. **Federation signing input fidelity**: [PR #105](https://github.com/djshellshoxxx/circula/pull/105), targeting only `future/fnet-209-federation-20261116`.
5. **Recovered analytics implementation**: [PR #108](https://github.com/djshellshoxxx/circula/pull/108), targeting only `future/fnet-205-analytics-20261116`.
6. **QR camera and image lifecycle**: [PR #120](https://github.com/djshellshoxxx/circula/pull/120), targeting only `future/fux-102-qr-20261116`.

**Additional source recovered:** the saved `Circula-Future-Recovery.zip` version 2
was retrieved and verified against its recorded SHA-256. Its self-contained Git
bundle passed complete-history verification, all 48 refs matched the inventory,
and a fresh bare-object check passed. It contains newer implementations for all
15 future modules and 164 historical evidence files. Seven compared targets are
ancestors of the archived heads; eight require deliberate reconciliation. None
of these older test records is treated as current-head qualification.

Fresh archive checks and selected recovery are recorded separately. The analytics
candidate restores the archived production/test blobs unchanged and adds native
qualification in PR #108. Recurring and federation are now reconciled, independently
reviewed, and published as stacked recovery drafts #118 and #117 with full current
Linux race/vet/fuzz qualification. Replacing their folders wholesale would regress
retained repairs. Receipts, areas, advisory governance and messaging recoveries are
published as #113, #115, #114 and #116. Archived
notifications also removes the current reviewed transport packages and therefore
requires a separate design/source reconciliation. See `ARCHIVE_FEATURE_STATUS.md`
before concluding that a feature missing from an older advertised branch must be
implemented from scratch.

The source fixes are complete within their reviewed scope. The master integration
is **not merge-ready**: required current-head provider qualification, active-worker
handoffs and the shared generated-traceability reconciliation remain outstanding.
Passing module or focused tests do not establish complete Core qualification.
All future modules remain disabled and on integration HOLD.

No main merge, deployment, release, source-branch deletion, shared force push,
protection bypass or policy adoption was performed by this workstream. Other
workers' independently performed merges are distinguished below.

## Repository inventory and provenance

| Item | Recorded result |
|---|---|
| Repository | `djshellshoxxx/circula` |
| Starting main | `ead63b5ce24b2bcf8352d75fd0d1ff7be711f625` |
| Initial advertised remote branches | **114**, each classified in `BRANCH_INVENTORY.json` and `.md` |
| Initial PR records examined | **96**, including the separately recovered metadata gaps |
| Initial remote tag refs | **0** advertised |
| Additional existing local branches/worktrees | **36 branches / 33 worktrees**, inspected read-only for ownership |
| Final reviewed local Core code | `3cc231542e7c46793f27b2fc406b9220fabf1bff` |
| Final local code tree, before evidence files | `ca441d471000c1ae24ca87c28255a0c467c6e2f9` |
| Earlier published Core code stage | `c3c991aa3c12a11edce2874e9444b9d2a99f746c`, tree `f97a12c04853c298b56dec0bfde72fd0b500b722` |
| Published Core checkpoint | PR #104 at `707a4363a86922d152ed2d203c9ea184fe4b323f`, exact tree `7ccb0188df8416eb8fa57b8740eba400a5e01363`, matching local `87cc4b229d39dbc8c00785048024f0e7296c015e`; later final-report publication is recorded separately |
| Additional recovered history | 48 saved refs, all preserved in a verified self-contained bundle; all 15 future-module trees differ from the compared advertised snapshots |
| Live changes after initial inventory | Recorded separately in `BRANCH_DELTA.json` and `.md`; the initial snapshot is preserved |

Each original branch record includes its head, merge base, unique commits,
changed paths, diff digest, tests, candidate security commits, source containment,
CI observations, ownership overlap and disposition. Merge simulations supplement
ancestry and patch comparison. A title or historical green check alone never
establishes merge eligibility. The first-page PR-triggered workflow lookup's
coverage limitation is explicit in the inventory.

The initial classifications are:

| Classification | Branches |
|---|---:|
| Already integrated | 52 |
| Blocked by active worker | 11 |
| Experimental future module | 17 |
| Blocked by policy/design | 5 |
| Duplicate or superseded | 29 |
| **Total** | **114** |

No additional historical Core branch was independently eligible to merge outside
the already active Core streams. In particular, the ledger-auth guard, MFA decider
and dashboard-amount meter historical topics clean-merge to the existing main
tree; merging them again would add history without new code. GOV-021's technical
fixture and the favorites follow-up were already contained in their isolated
targets. Useful outstanding Core history is preserved in the active consolidation
streams, including PR #90 and its descendants. It was not copied while unfinished.

The exact source provenance is retained even where connector publication gives a
commit a different identity. Each publication was checked by full Git tree
equality, not by approximate text comparison.

| Source | Reviewed local commit | Published head / PR |
|---|---|---|
| Go initial export boundary | `4e4d42eadf860891b54b3d6802db013b09029f2b` | Composed in #104 |
| Go complete manifest/bounded reads | `bf02b29b514a5fd0f7c8cc33aaf5d0ddb352f5ff` | Composed in #104 |
| Python package identity/limits | `778d53488b01ef7e27a3e94e47c043b38258d706` | Composed in #104 |
| Python complete physical ZIP members | `9b97bf94e5bf820bd0013939e1d487635ea5ce35` | Composed in #104 |
| Python manifest types, ordered aliases and strict JSON constants | `56c09536a01fe6658ee98fe573029eac77c6ae82` then `9e95b73861be23d0f46121f534e6c3e46d83bc1a` | Independently reviewed, composed in local `3cc2315`, and published with native-reader workflow/evidence in #104 at `707a4363` |
| Recurring source and native workflow | `cdcd2cb38b3af0275e5ceb58242384cfa916be45` | `73a911a9bec387381633c031d8a4083c9fc724ad`, #97 |
| Notification transports | `4a59bd1e740c2fcf5877de2a90594439cb9cd570` | `13669485b641475fce526633f47055ef31997098`, #103 |
| Federation signing inputs | `db21ea0a0008309d6e8d116ccb00eb77ac6873d9` | `059b0e4508a6c4f8e29f6a4cccd06f92b14ea115`, #105 |

## Core export repair

### Defects and behavior

Both verifiers now require each package member to have one unambiguous flat
identity. They reject duplicate archive names, duplicate manifest declarations,
declaration of the reserved manifest itself, malformed names and nonregular
members. ZIP directory metadata is checked even when its creator-system metadata
is contradictory. Python checks the original ZIP filename before the standard
library can silently trim a NUL suffix.

Both require the supported manifest schema and an explicitly present files list.
A valid empty list remains accepted. Python additionally rejects boolean/float
substitutes for its integer schema value. Unsupported or incomplete manifests fail
before ledger reconciliation.

Go's existing **4 GiB per-member maximum** is preserved; Python now enforces the
same cap. Size metadata is checked before a read, and bounded reads detect one byte
beyond the limit. A Go reader
error discards the associated partial bytes. The Go command still buffers member
data; this work does not claim a streaming ledger verifier or a lower memory
budget. It also does not claim protection against every concurrent filesystem
replacement between inspection and opening.

Independent review found a further Python-specific failure: `ZipExtFile` can stop
at a ZIP's declared uncompressed length, allowing a valid manifest and CRC for a
prefix to hide a physical suffix. The reader now opens a bounded copy of the entry
metadata, consumes its complete content with the normal CRC check, and requires
actual length to equal the original declaration. Both understated and overstated
stored/deflated members fail. Valid controls continue to pass. Go's archive reader
already rejects these malformed-size fixtures; its bounded-reader repair preserves
those errors rather than returning partial data.

Further exact-Go-decoder probes reproduced manifest differences in the Python
reader: boolean/float substitutes for integer fields, invalid optional field
types, and malformed earlier duplicate values could be accepted. Validation now
checks every occurrence before duplicate values collapse, while preserving valid
duplicate and nullable-field behavior. Known field names follow Go's simple
case-fold matching, including long-s and Kelvin aliases; unrelated Unicode
ligatures remain ordinary unknown extensions. Later case aliases supply the
effective value. Non-JSON `NaN`, `Infinity` and `-Infinity` are rejected throughout
manifest and member JSON. Legacy timestamp forms accepted by the pinned Go parser
remain compatible. These are parser-conformance repairs, without a blanket ban
on valid duplicate JSON fields or unknown extensions.

### Changed files

- `cmd/circula/verify_export.go`
- `cmd/circula/verify_export_ambiguity_test.go`
- `cmd/circula/verify_export_boundary_test.go`
- `scripts/read-export.py`
- `scripts/test_read_export.py`

The reviewed source diff contains **1,058 additions and 24 deletions** in these five
files. There are no migrations, root dependency changes, ledger arithmetic changes,
API changes, authentication edits or future-module activation in this Core patch.

### Regression and review evidence

The initial Go tests reproduced duplicate-member and duplicate-manifest acceptance.
A later missing/null-files regression also failed before its repair. The normal
focused production-package verifier suite then passed. A separate limited harness
using the exact production verifier source passed race testing; that result is
explicitly narrower than the complete CLI package race suite.

The Python suite retains the original ten tests and now has **38 top-level tests**,
with additional parameterized malformed-input cases. The final author source
passed all 38 in **23.950 seconds**; the final composed Core source reran all 38 in
**17.805 seconds**. The earlier complete-ZIP source passed its then-current 22 tests
in 10.662 seconds, and its earlier composed tree passed those 22 in 9.901 seconds;
these are separately preserved historical stages. Independent probes used two valid stored/deflated controls and
four malformed-size counterparts: both controls returned success; all four invalid
archives returned failure without a traceback. Tests also cover full-data CRC
variants, ensuring the explicit actual-length comparison is exercised.

The manifest follow-up includes two distinct fail-before campaigns, each exposing
48 failing subcases at its preceding source stage. The final author replay matched
all 27 actual ZIP fixture expectations, including five valid controls. Independent
review reran eight alias/constant probes and the six prior size probes, inspected
the strict parser and ordered-field behavior, and found no remaining blocker in
this selected repair. Decoder-only Go probes are explicitly not a whole CLI test.

An independent reviewer read the complete combined diff and checked all five blobs
against the reviewed author commits. No blocking source finding remains within
this selected repair scope. The reviewer did not grant a repository-required
approval or waive provider checks.

## Isolated feature completion

### FNET-207 recurring computation

PR #97 adds structured daily, weekly and monthly `DueWithSchedule` rules while
preserving the existing `Template` layout, `Due` API and valid daily logical keys.
It keeps the provisional module's current-local-date behavior, missing-month-day
skip semantics, DST gap skip and earliest-fold selection. It also repairs invalid
UTF-8 identity normalization, host-dependent `Local` zones, historical offsets
containing seconds, and a valid zero instant being mistaken for no occurrence.

The bounded timezone-interval resolver was checked against an independent
minute-based oracle: **3,456 comparisons across six zones**. The final local tree
passed race testing at **94.4% statement coverage** and vet. The final runtime/test
contents completed exactly **100,000 fuzz executions**; the later native-matrix
change affected only the workflow. A new dedicated workflow uses native Ubuntu, macOS Intel and Windows;
Linux/macOS run race tests, Windows runs the ordinary suite with CGO disabled.
The dedicated module provider matrix passed on all three native systems at the
recorded publication. The separate inherited Core workflow failed; its jobs and
external #110 repair remain separate gates.

That published helper snapshot does not contain the whole product. The recovered
archive does contain durable templates/firings, atomic reminder insertion,
authorization ports, edits, retention and HTTP/CLI source; those must be reviewed
and reconciled rather than reimplemented. Fresh review of its Calendar reproduced
invalid-identity and historical second-offset defects; the dedicated recovery
candidate retains the earlier helper repair and all its tests. Actual Core
adapters, host scheduler recovery/fairness, browser/accessibility and operational
qualification remain open. Calendar computation never grants authority to post or
accept a trade.

### FUX-104 notification transports

PR #103 repairs three independently reproduced adapter defects:

- Successful HTTP 200/201/204 Web Push responses returned a typed-nil error.
- Large decimal `Retry-After` values could overflow a duration or be reduced to
  zero; the decimal is now bounded before conversion.
- The pinned SMTP library's custom-dial path did not perform implicit TLS. The
  wrapper now establishes and verifies TLS before handing the connection to it.

The tests exercise real Web Push encryption/decryption with synthetic subscriber
keys and a synthetic loopback HTTPS server, plus synthetic loopback SMTP servers.
Injected no-network HTTP clients cover response statuses and body bounds.
They cover certificate rejection before AUTH, cancellation, error redaction,
response-body bounds, DNS/redirect policy and concurrent immutable sends. Focused
race coverage is **85.3% for pushsend** and **84.2% for smtpsend**; vet passed. The
Retry-After fuzz target completed exactly **10,000 executions** in **2.567 seconds**.
The full final notifications module also passed its explicit SQLite-tagged race
suite (notifications 1.480s, Web Push 1.848s, SMTP 1.201s) and tagged vet. The vet
retry took 110.903s after a failed cache-initialization attempt. The earlier
interrupted fuzz and failed cache-initialization attempts are not passes.

The initial repair changes four transport source/test files. Publication starts from the normal
PR #93 merge, `797dddf21adddfb34a49e7bda1871a202968dcda`; its tree equals the reviewed
local parent. The resulting full tree
`198d2e6d9e40922080eb91f3ab73d011808feea9` equals the independently reviewed local
repair. The new follow-up at `f831c45de9137b20abe3d31590433bda6348112b`
adds only a dedicated native transport workflow and qualification note, with exact
whole-tree equality `d95dd114d7c39914c39d81ad8e79be033c9f8446` to local
`f4555898e832c94c19fb69139327230af44af3ba`. The four source/test blobs
and dependencies remain unchanged. The preceding source passed all seven executed
Core jobs and the complete Linux SQLite module job; the optional native Core job
was skipped. The follow-up requests actual transport tests on three native systems
and opts into the existing native Core matrix. Its newly queued jobs are not passes.
PR #100's separate Core refresh and owned workflows remain untouched. Adapter qualification
does not establish production credentials, provider interoperability, deliverability,
key custody, retention or a complete authorized Core notification integration.

### FNET-209 federation signing inputs

PR #105 rejects malformed UTF-8 in all five signed string fields before JSON
marshaling. The reproduced defect is false success at the signing boundary: the
old signer changed invalid bytes and returned an envelope that its existing
exact-JSON verifier already rejected. No accepted-identity, signature-verification
or replay bypass was reproduced. Valid Unicode, literal U+FFFD, decomposed text,
wire bytes, domain separation and all existing verifier behavior are preserved.

Thirty malformed-input cases reproduce the old failure. The committed repair
passed race tests in 1.071s at 87.5% statement coverage, vet and exactly 100,000
string-fidelity fuzz executions in 5.682s. A separate reviewer checked the complete
three-file source/workflow diff and every published blob. Whole-tree equality is
`4d9f608041e65a2a1a297f17614438790240bc04`. The dedicated new workflow qualifies
Linux/macOS race, native Windows ordinary tests and bounded Linux fuzzing. The
dedicated provider matrix passed on all three systems. The separate inherited
Core workflow failed and remains a gate; this does not qualify the newer recovered
federation source automatically. The verified archive additionally contains durable
transport/replay state, trust/consent, key lifecycle and a v2 discovery protocol.
Its recovery must preserve this v1 signing-input repair and qualify the composed
source with the actual pinned SQLite driver. Independent-peer interoperability,
key custody, capacity policy and product acceptance remain open.

### FUX-102 QR camera and image lifecycle

PR #120 selectively repairs the current QR scanner wrapper without replacing the
service, persistence, API or vendor distribution. A per-operation generation check
rejects stale camera permission, playback, import and decode completions. Attached
tracks stop immediately on cancellation. Each bundled scanner owns a separate
video element so delayed cleanup cannot stop a later stream. New image selection
and actual manual form submission cancel earlier work, preserving normal preview
submission and the no-new-selection control.

The initial unchanged source failed 21 of 23 deterministic lifecycle tests. The
first repair passed all 23. Independent review reproduced stale results after an
oversized replacement image and manual form submission; the same expanded suite
then failed three cases before the follow-up and passed all 27 afterwards, with
zero skips. Root independently reran all 27 and both syntax checks. A separate
reviewer replayed its original two probes and confirmed both pass. Two tests run
the exact bundled Nimiq lifecycle implementation with synthetic media boundaries.

The final published head is `2f0c83bc34a52551281769ec17561b18920bd2cf`,
matching local `3c1ce7a7dc190271a9a830abc924f4f938049801` at whole tree
`6c5f55e7fe658d5144c3c09c010f74923397c7df`. Six paths change: scanner,
lifecycle tests, scope note, a dedicated three-OS Node workflow and the same two
exact accepted-main Core baseline files described below. The qualified camera
script/test/workflow blobs remain unchanged by that baseline follow-up.

This establishes synthetic camera lifetime and preview-navigation behavior. It is
not real browser/camera/CSP/accessibility, host-session or ledger qualification;
no payment or authorization bypass is claimed. Actual native provider jobs and
normal merge requirements remain open. The module remains disabled/HOLD.

### FNET-205 recovered analytics

The isolated analytics candidate `150ab4ed3c2f9911f45fd07e8676edd03aa31179`
is published as PR #108 at `38b9bcd0e049d97167d793baba933d5e79d8030d`,
with exact full-tree equality `926205aae18016ef55f882fda6415e1ac2b7fd94`. It
restores archived head `ef4bdf87d2bc915b26d84e548ea4e6a478079006` onto its
published ancestor `0b2965101e1d5d1e3e6447e7c24197efc7f0cb9f`. All eight Go
source/test files and the standard-library-only module remain byte-identical to
the archive. New documentation separates historical milestones from current
qualification; a dedicated workflow requests native Linux/macOS/Windows checks.
A separately reviewed six-file follow-up restores exactly two accepted-main Core
blobs (audit-anchor `os` import and departure-test fixture cap), with provenance
and accurate scope documentation. The analytics Go/module/workflow files are
unchanged by that follow-up; no unfinished worker source is copied.

Recovered behavior includes ledger-verified ordinary trades and partial reversals,
scoped own balances, exact integer output, closed calendar periods, cohort and
dominance suppression, and read-only HTTP/no-JS/CSV/CLI interfaces. Review covered
all three production files and their tests. Fresh race testing passed at 86.4%
coverage, with 19 passing test events and no skips; vet and module verification
passed. A separate actual fuzz campaign passed exactly 100,000 `FuzzClosedPeriods`
executions in 13.740 seconds (156.507 seconds including cold compilation).

Immutable-release persistence, query-budget/correction policy, production budgets,
manual acceptance and the authenticated historical Core adapter remain unfinished.
Restoring existing source does not complete or activate that feature.

### Current recovered candidates and reproduced follow-ups

Seven substantial recovered candidates are now published separately. The table
records the current runtime qualification, rather than transferring an earlier
prototype or archive pass to changed source. Exact local/remote trees, parents and
targets are in `PUBLICATIONS.json`; complete source and review receipts accompany
each module and this report.

| Recovery | Published PR / head | Current Linux qualification | Remaining immediate boundary |
|---|---|---|---|
| Analytics | #108 / `38b9bcd0` | Original Go/module source exact; race 86.4%, vet/checksums and 100,000 closed-period fuzz executions | Exact-head native/provider gates; immutable releases/query-budget and host adapter decisions |
| Receipts | #113 / `3cc55f79` | Repaired full race: 40 top-level tests, 118 pass events, no skips, 84.6%; vet/checksums and all three 100,000-execution fuzz targets | Native/provider, independent verifier/keys and real Core adapters |
| Areas | #115 / `684ca513` | Full real-driver race: 68 pass events, no skips; vet/checksums/isolation and 100,000 actual parser fuzz executions | Native/provider and host/manual acceptance |
| Advisory governance | #114 / `fbd92a22` | Full real-driver race: 59 pass events, no skips, 75.7%; vet/checksums/isolation and 100,000 actual parser fuzz executions | Native/provider, host authority and manual/policy integration |
| Messaging | #116 / `62623635` | Full real-driver race: 107 pass events, no skips; library 76.6% / CLI 80.9%; vet/checksums/isolation | Full-package fuzz; separate Windows privacy repair and native verification; host/manual acceptance |
| Recurring | #118 / `035dd1c0`, stacked on #97 | Full real-driver race: 48 top-level tests, no skips, 80.0%; vet/checksums and both 100,000-execution fuzz targets | External #110 baseline/digest handoff and renamed-test reconciliation; native/provider/host/manual gates |
| Federation | #117 / `f50aaa4b`, stacked on #105 | Full real-driver race: 36 top-level tests, no skips, 74.5%; vet/checksums and all three 100,000-execution fuzz targets | External #106 README handoff; native/provider, live peer/TLS and production key/host acceptance |

**Receipts exact numbers.** The plain offline parser fed raw numeric tokens through
JCS before integer decoding. Actual unchanged-source tests accepted all eleven
fractional/underflow cases, including `1125899906842624.01` as the allowed maximum
and nonzero precision `1e-400` as zero. The repair compares original significant
decimal digits and exponent displacement with the bounded integer DTO, under the
existing 16 KiB input cap. It preserves all eighteen exact decimal/exponent/zero
controls and changes no issuer, verifier, signed format or dependency. This is an
offline parsing-integrity defect; no signature forgery or authorization bypass
was demonstrated. Independent root code review and separate composition review
preceded publication. Later full race and three fuzz campaigns passed with every
published source/dependency hash unchanged.

**Recurring chronology and calendar.** The recovered Calendar implementation
needed independent repairs in addition to the earlier helper: malformed identity
checks, historical second-offset wall times, and the next eligible local date.
The combined suite retains the earlier 3,456-case oracle and adds 10,368 independent
all-cadence comparisons. Actual pinned-driver regressions also demonstrated
premature/late cleanup from variable-width RFC3339Nano string ordering, one-nanosecond
loss from `julianday`, and an older inbox item selected ahead of a later fractional
timestamp. Canonical UTC fractional padding now preserves exact strict cutoffs,
equality, tenant scope, tombstones and stable ID ordering. The original and
recovered test suites coexist; no test was dropped for the filename collision.

**Unicode JSON aliases and messaging input.** Areas, advisory governance, messaging
and recovered recurrence used lowercase conversion for duplicate names, which did
not cover all of Go's Unicode simple-fold field aliases. Actual regressions now
exercise the production parsers and their caller boundaries. Cycle-representative
comparison rejects duplicate aliases without changing a valid single alias or
performing multi-rune expansion. Messaging additionally rejected eighteen malformed
UTF-8 stdin cases before JSON construction while preserving valid Unicode and
literal U+FFFD. No corresponding authorization bypass was claimed.

**Preserved failures and source review scope.** The original areas red run contains
five actual assertion failures and a killed dependency-vet subprocess. Its mixed
classification is corrected without replacing the original bytes. An executor
reattachment left other outstanding commands without a terminal result; those are
incomplete, not presumed passes. Subsequent serial runs have new receipts. Areas'
prior current-runtime race/vet results are carried only after complete Go/test/module
hash equality. Parser-only fuzz names actual production files and remains distinct
from full real-driver race coverage. The three new recovery deltas/workflows received
independent review; that scope is separate from the author's complete archived-source
read. Federation additionally received a separate complete eight-file production review.

**Accepted-main baseline.** Six recovered candidates and the QR camera candidate restore exactly the already-merged
audit-anchor import and departure-test fixture cap, with source/provenance proof.
No production cap, root dependency, shared CI, migration or unfinished worker code
is changed by those two-file restorations. Recurring's corresponding three-path
handoff remains with #110 and is not copied prematurely.

**Additional independent work.** The Windows messaging follow-up is frozen for
review and continuing qualification. The existing mode guard fails closed on
Windows; its replacement establishes an owner-only process-user/directory/DB/WAL/SHM/
backup ACL profile and rejects unsafe existing states without modifying them.
Independent review reproduced three lifetime gaps involving concurrent close,
an active transaction and asynchronous cancellation rollback. Earlier races and
cross-compilation remain exact-source checkpoints; the latest lifetime design
requires its own completed qualification and actual native Windows execution.

The four unreserved divergent modules have now been compared against their richer
current implementations in `FOUR_MODULE_RECONCILIATION.md`. The concrete QR camera
repair is published. Current onboarding strict-before retention is being probed
against its existing SQLite store; until reproduced, its timestamp-ordering
concern remains a hypothesis. The report separates compatible technical work
from actual schema, policy and API decisions. No future module is enabled.

### All fifteen recovered archive snapshots

The current `ARCHIVE_MODULE_QUALIFICATION.json` records successful complete Linux
race, vet, checksum, formatting and source-integrity checks for all **15 unchanged
archived module snapshots**, using their actual pinned dependency graphs. The last
notifications retry passed 35 top-level tests / 58 pass events with zero failures
or skips, library 71.0% / CLI 83.8% coverage, in 176.503 seconds wall. All 89 archived
files remained exact. It used the original x/sys v0.49 graph, bounded memory and
`-ldflags=-w`; no substitute driver, tag or dependency was introduced.

The earlier 14/15 aggregate remains byte-preserved in `verification/`, along with
the original failures and the final notifications receipt/log. The retry was one
new, resource-monitored attempt after the executor reattached and shared caches
were warm. These archive results do not qualify later patches, native platforms,
manual acceptance or product integration automatically. Known archived defects are
tracked in the repaired candidates and preserved notification patches.

### Earlier advertised module snapshots

`MODULE_QUALIFICATION.json` preserves the exact source SHA, manifest, command,
exit status, duration and output for fourteen separately extracted module
snapshots. All fourteen completed their default race suite, vet, module checksum
verification and formatting check. The five explicit SQLite-tagged suites for
favorites, QR, discovery, notifications and onboarding also passed. PWA's Node
tests passed. The repaired recurring module was qualified separately, giving
**fifteen tested module snapshots**.

These results are revision-specific. The baseline notification receipt predates
the transport repair; the feed receipt predates the active completion branch. A
manifest marked incomplete remains incomplete after tests pass. The later recovered
source is separately described in `ARCHIVE_FEATURE_STATUS.md` and qualified in
`ARCHIVE_MODULE_QUALIFICATION.json`. Read the older `FUTURE_MODULE_STATUS.md`
against its recorded published snapshots; its missing-code observations do not
erase implementation found in the verified archive.

## Verification commands and limits

The local toolchain was Go 1.26.11 on Linux. Resource controls included
`GOTOOLCHAIN=local`, `GOMAXPROCS=1`, `GOMEMLIMIT=512MiB` and package parallelism one.
Explicit SQLite module runs used `CGO_ENABLED=1` and `CGO_CFLAGS='-O0 -g0'`.
These controls do not turn a killed or incomplete command into a pass.

| Check | Actual result / scope |
|---|---|
| `go test -p=1 ./cmd/circula -run '^TestVerifyExport' -count=1 -timeout=2m` | Initial actual package verifier regression suite passed; later expanded focused production-source run with `-vet=off` passed on the final Go contents |
| `python3 -m unittest discover -s scripts -p 'test_read_export.py'` | Final 38 tests passed on reviewed author and composed Core source; composed suite17.805s, then Python3.12.14 suite17.199s after executor reattachment |
| `go test -count=1 -p=1 -timeout=10m ./...` on earlier composed source | **FAIL**: contract dependencies could not be cached because workspace storage was full; runnable cmd/API/domain/repository/service/store/web/tooling packages passed |
| Same full command at reviewed Core `6e8f5f1` | **FAIL**: CLI process and API link/service compilation were killed; contract, domain, repository, store, web and tooling completed successfully. Final `3cc2315` retains the exact same three Go verifier blobs; that equivalence does not turn the failed invocation into a whole-Core pass. |
| Full CLI vet/race attempts | Compiler resource failures; not passes |
| `go test -race -count=1 ./...`, `go vet ./...`, `make check` | Complete local aggregate qualification is not established. The preceding published c3c991a subsequently passed provider full Linux/macOS race and Windows suites. Traceability remains a real blocker; current final-head provider/lint gates are still required. |
| `make traceability-check` | **FAIL / owner handoff required** for the new DAT-040 regression citation; see pending patch below |
| `scripts/check-action-pins.sh` | PASS |
| `scripts/check-hygiene.sh` | PASS, including migration/static and fixture checks |
| Core runtime diff and changed Go formatting | PASS at the recorded source. Authored audit summaries pass after EOF cleanup; this audit checkpoint preserves cosmetic whitespace in two portable patches and two verbatim provider logs. See REPORT_PUBLICATION_QUALIFICATION.json for the exact comparison and exceptions. |
| Default nested module commands | `go test -race -p=1 -count=1 -cover -timeout=90s ./...`, `go vet ./...`, `go mod verify`, `gofmt -l .`; exact receipts attached |
| SQLite module commands | Same race/vet selections with `-tags=sqlite_cgo_test`; five baseline module suites PASS |
| Recurring fuzz | `go test -run '^$' -fuzz '^FuzzScheduleInvariants$' -fuzztime=100000x -parallel=2 -timeout=90s ./...`: PASS |
| Transport fuzz | `go test -p=1 ./pushsend -run '^$' -fuzz '^FuzzRetryAfter$' -fuzztime=10000x -parallel=1 -timeout=30s`: PASS |
| Recovered analytics fuzz | `go test -run '^$' -fuzz '^FuzzClosedPeriods$' -fuzztime=100000x -parallel=1 -timeout=150s ./...`: PASS, 100,000 executions |

The executor temporarily disconnected and later reattached. Actual local source
and commits survived; no source reconstruction was published. Interrupted commands
are retained as incomplete. A later reattachment again returned unknown process
for an outstanding runner; its unfinished commands are not assigned an invented
exit status or pass, and resumed executions use separate attempt records. Shared storage later filled; only this workstream's
reproducible download archives were removed and its build cache was relocated.
No other worker's files or caches were removed. Process/compiler kills persist
as recorded failures; provider jobs must supply the complete qualification.

The starting main was not completely green: scheduled run
[44296161788](https://github.com/djshellshoxxx/circula/actions/runs/44296161788)
passed eight jobs but failed fuzz smoke. The shared duration-to-execution-budget
repair belongs to the active Core stream and is not duplicated here. Historical
main, PR #93 and PWA job passes are evidence about their exact revisions, not about
the final #104 head.

### Provider Core checkpoint and native reader coverage

Run [44511241324](https://github.com/djshellshoxxx/circula/actions/runs/44511241324)
completed for the preceding published Core head `c3c991a`, tested as merge
`ae7b54a77f24b7ea71099220fe6b50c17b6dc2dc` into recorded main. Seven jobs passed:
full Linux/macOS race suites, Windows ordinary tests, release-target builds,
reproducibility/local smoke, bounded fuzz and contract/load. Two jobs failed:
DAT-040 generated traceability was stale, and browser dependencies failed to
install after Ubuntu package-server HTTP503 responses. Browser tests did not run.
Vet, action policy, hygiene, licences and secret scan passed before traceability;
later Python, golangci-lint and vulnerability steps were skipped. Complete job
metadata and selected diagnostic excerpts are in `CORE_PROVIDER_CHECKPOINT.json`.

The final Python follow-up adds an independently reviewed dedicated native reader
workflow without modifying shared CI or adopting a new action. Each of Linux,
macOS and Windows runs the same38-test suite using its hosted interpreter and
records actual source/runtime provenance. The repository action policy passed.
The new current source and later report commits require their own provider gates;
previous-head success is not treated as current-head approval.

## Ownership, handoffs and remaining blockers

1. **API-036 remains owned.** PR #101 was included in PR #90's topic. At the
   later live refresh PR #90 had advanced to `839e86935767288e594914ad711486616a604eca`,
   with draft continuations #109 (UI/catalogue) and #111 (API-036 qualification).
   PR #90 remained draft. Its report identifies a live-token-authority race repaired
   in the independently active #99 descendant. Topic inclusion does not establish
   complete current-head qualification or handoff. No unfinished commit was copied.
2. **FNET-204 remains owned.** PR #92 advanced to
   `db80ed34af09ad6b28e533da6d4b82b8250f3711` after its owner's inclusion of
   PR #98. It remains draft, isolated and disabled/HOLD, subject to its own
   current-head completion, security and qualification gates. This workstream did not change
   their source, workflow, base or target.
3. **Other active recovery remains separate.** PRs #99 and #102 contain independent
   transaction/store/export-admission/catalogue work. Source comparisons found no
   change in the five #104 export-verifier paths at the inspected #99 head. Their
   moving heads require reconciliation after owner handoff, not an opportunistic
   merge of their in-progress branches.
4. **Generated traceability requires the shared owner.**
   `pending-traceability.patch` contains the exact generator-produced delta on this
   earlier `6e8f5f1` Core source: its input-head header and the additional DAT-040 test citation.
   The legitimate regression citation is retained. The reserved generated file is
   untouched, and its failing freshness gate is a real merge blocker. After an
   eligible handoff, regenerate once against the combined final tree rather than
   blindly applying this patch to a newer report.
5. **Current-head provider gates remain mandatory.** The Core PR has the
   `ci:cross-platform` label. Its evidence follow-up triggers Linux, native Windows
   and macOS qualification plus contract/load, browser, lint/vulnerability, fuzz,
   build and reproducibility jobs. Queued, skipped, cancelled, superseded or failed
   jobs do not count as passes. The module PRs retain their configured Core gates
   as well as dedicated module gates.
6. **Normal review/protection requirements remain mandatory.** Branch-protection
   reads returned insufficient integration permissions; the ruleset collection
   alone does not prove there are no protections. Independent engineering review
   is not represented as a provider approval. Any eventual merge must use the
   ordinary expected-head PR procedure after the applicable checks/approvals pass.
7. **Product and policy work remains.** Governance recovery, permanent operator
   authority, security-signal visibility, retention/erasure and reference-hardware
   release criteria are not invented or silently approved. The module status
   report distinguishes these decisions from remaining technical implementation.

## Evidence index and completion boundary

- `BRANCH_INVENTORY.json` / `.md`: preserved initial complete branch assessment.
- `BRANCH_DELTA.json` / `.md`: separately timestamped live-ref/PR changes.
- `OWNERSHIP.md`: exclusive paths, agents, dependencies and publication boundaries.
- `PUBLICATIONS.json`: exact current publication checkpoint, original target baselines, actual commit parents and verified whole trees.
- `FOUR_MODULE_RECONCILIATION.md`: current/archive feature, API and policy comparisons for favorites, QR, discovery and onboarding.
- `PROVIDER_STATUS_20261119T1353Z.json`: dated current-head provider observations, with separate newer notification and QR publication receipts.
- `REVIEW.json` / `.md`: exact-source independent review and probe receipts.
- `MODULE_QUALIFICATION.json`: fourteen default and five explicit SQLite module
  suites, including exact command outputs.
- `FUTURE_MODULE_STATUS.md`: all fifteen implementation states and dependencies.
- `ARCHIVE_RECOVERY.json` / `.md`: verified archive hashes, refs and source comparisons.
- `ARCHIVE_FEATURE_STATUS.md`: recovered implementation inventory and remaining gates,
  superseding any global missing-code inference from older published snapshots.
- `ARCHIVE_MODULE_QUALIFICATION.json`: fresh archive checks, explicit failed attempts
  and resource limitations; the archived source uses its real pinned dependencies.
- `MANIFEST_PARITY_EVIDENCE.json` and `MANIFEST_PARITY_QUALIFICATION.md`: all 56
  original probe/fixture/log files, with exact bytes and hashes, plus source/review scope.
- `pending-traceability.patch`: unapplied generated reconciliation for owner handoff.
- `verification/qualification-evidence-20261119.zip` and its manifest preserve 205 original receipts, logs and review probes losslessly (3,739,552 uncompressed bytes); every archived member hash was verified. Older failures remain included.
- Additional `*_QUALIFICATION.json` and current-CI receipts record later observed
  transport/module/provider results without replacing earlier failed evidence.

Completed deliverables include the reviewed export boundary repair, notification
adapter repairs, seven published recovered candidates, complete initial and dated
branch inventories, all fifteen archived Linux qualifications, and preserved
source/review/failure evidence, the reviewed QR camera repair and the four-module
divergent-source comparison. Windows messaging and current onboarding retention
investigation remain active. The full definition
of done remains open until eligible worker handoffs, shared-artifact reconciliation,
normal approvals and current-head qualification are complete. This report does not
claim that every Core or future feature is complete.
