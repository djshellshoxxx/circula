# FNET-209 combined recovery qualification — 2026-11-19

**Local combined-source qualification passed. The feature remains disabled and integration HOLD.** The results below apply to the exact nested-module source recorded here. Native provider and Core checks at the new published head remain separate gates.

| Identity | Value |
|---|---|
| Tested local commit | `ca9af7543cc560d4a798b6e446d85ec38f740595` |
| Tested whole tree | `2b1400be19b42ee63c86980666a71d1c88e07811` |
| Tested module tree | `81a58c2b2b05972f977e1997ed0ce978f6ca4a12` |
| Toolchain | Go 1.26.11, Linux/amd64 |
| SQLite driver | Actual pinned `modernc.org/sqlite v1.60.1`, default build tags |
| Source stability | All tracked qualification fingerprints remained unchanged during execution |

## Commands and observed results

The environment was `GOTOOLCHAIN=local`, `GOMAXPROCS=1`, `GOGC=10`, `GOMEMLIMIT=256MiB`, `CGO_ENABLED=1`, `GOWORK=off`, and `GOFLAGS=-mod=readonly -p=1`. The full race command used `-ldflags=-w` for the constrained shared environment. Every command ran serially with a separate temporary directory; module sources and dependency pins were unchanged. The JSON companion records the actual executable, environment, commands, per-command outcomes, test names and SHA-256 fingerprints.

| Check | Observed result | Wall time including build |
|---|---|---|
| toolchain | PASS | 0.008s |
| isolation | PASS | 0.266s |
| format | PASS | 0.066s |
| race | PASS; 36 top-level tests, zero skips, 74.5% statement coverage; package 24.790s | 34.537s |
| vet | PASS | 0.873s |
| checksums | PASS | 2.127s |
| fuzz-legacy | PASS; `FuzzSignPreservesEventStrings`; exactly 100,000 executions; package 8.572s | 14.379s |
| fuzz-v2-strings | PASS; `FuzzRecoveredV2StringFidelity`; exactly 100,000 executions; package 8.651s | 53.114s |
| fuzz-wire | PASS; `FuzzDiscoveryWire`; exactly 100,000 executions; package 5.401s | 6.394s |
| dependencies | PASS | 0.018s |

Exact runtime commands, using the recorded Go toolchain:

```sh
go test -p 1 -race -count=1 -cover -json -ldflags=-w -timeout=2m ./...
go vet -p 1 ./...
go mod verify
go test -p 1 -run '^$' -fuzz '^FuzzSignPreservesEventStrings$' -fuzztime=100000x -parallel=1 -timeout=150s envelope.go envelope_test.go envelope_utf8_test.go
go test -p 1 -run '^$' -fuzz '^FuzzRecoveredV2StringFidelity$' -fuzztime=100000x -parallel=1 -timeout=150s envelope.go protocol.go protocol_recovery_test.go
go test -p 1 -run '^$' -fuzz '^FuzzDiscoveryWire$' -fuzztime=100000x -parallel=1 -timeout=150s envelope.go protocol.go protocol_fuzz_test.go
```

The full race suite has a two-minute Go test deadline and a five-minute outer deadline. Each fuzz target has a 150-second Go deadline, a 180-second outer deadline, and an exact 100,000-execution budget. The fuzz commands compile the actual production files named in each invocation; complete real-driver race and vet qualification ran separately over `./...`.

Dependency isolation and disabled/HOLD checks ran the exact dedicated-workflow guard. Its `go list -buildvcs=false -deps -test` query avoids incidental managed-worktree VCS stamping; separate Git commands record the tested commit and module tree. The first recurring attempt stopped at that metadata lookup, before the full suite, and is preserved as a failed attempt. It is not counted as a pass. Formatting, exact requirements/indirectness, no replacements/exclusions, package boundaries and module checksums all passed.

## Qualified scope and independent review

The complete suite covers retained legacy v1 signing/verification and recovered v2 protocol, SQLite trust/replay/consent/projection/outbox, withdrawal/lifecycle, HTTP/member/admin/CLI and synthetic bounded transport fixtures. New v2 tests reject malformed bytes in all ten signed string/key-selector inputs, preserve valid public Unicode and exact JSON/JWS bytes, and retain ASCII-only v2 identifiers. The legacy repair fixes false-success signing of malformed input; the original verifier already rejected those lossy envelopes. No signature-verification or replay bypass is claimed. The extracted wire target retains its original seeds, body and clock declaration.

The JSON companion lists the independent full review of all eight production files at `5cf90f4ea3f9bcd17d4c7eaa611759080895c4dc`, the composition/protocol-test/workflow review, the exact fuzz-target extraction review at `18978139341a1d066f58d632de757bbdb4d00999`, and root's separate VCS metadata-adjustment review. No production source changed after the complete eight-file review.

After these tests, the dedicated workflow adds the owned PR source branch as a second explicit target for the new stacked recovery draft. That single routing line leaves all test/guard scripts, action pins, permissions and deadlines unchanged. Documentation and descriptive manifest status follow the tested source; all Go files, tests, `go.mod` and `go.sum` retain their tested bytes. No test rerun is inferred from these documentary successors.

## Remaining gates

External PR #106 still owns `README.md`; its current published bytes are preserved. The two separately reviewed Core baseline files are exact accepted-main restores. Actual Core adapters, live independent-peer/TLS interoperability, real browser/manual accessibility, lifetime-capacity/compaction strategy, production key custody and final protocol/rate/peer approvals remain open. Cross-network settlement remains excluded.

The dedicated workflow requires native Linux/macOS race tests and native Windows ordinary tests at the final published head; Windows does not supply race coverage. The provider has not qualified this new recovery tree yet. Historical prototype, archived baseline and prior native results retain their original scope. Coverage percentages measure exercised statements and do not establish feature completion, host integration, policy approval or release readiness.
