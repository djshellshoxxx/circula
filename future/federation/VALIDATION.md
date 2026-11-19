# Federation recovery composition validation — 2026-11-19

Dependency isolation lists packages with `go list -buildvcs=false -deps -test`. This disables only Go's incidental VCS stamping for the dependency query, after the managed worktree produced a VCS-status error despite successful direct Git reads. The dedicated workflow separately records the exact Git head and module tree; the dependency allowlist, disabled/HOLD assertions and actual test/vet flags are unchanged. The initial failed recurring isolation attempt is preserved in the coordinator's evidence and is not a test pass.

**Current combined local qualification: PASS at `ca9af7543cc560d4a798b6e446d85ec38f740595`.** The full real-driver race suite passed 36 top-level tests with zero skips and 74.5% statement coverage. Vet, checksums, formatting, exact dependency/HOLD checks and all three 100,000-execution fuzz targets passed. `RECOVERY-QUALIFICATION-20261119.md` and its JSON companion record commands, source fingerprints and review scopes. `RECOVERY-20261119.md` records archive and external README ownership provenance. Native provider and handoff gates remain.

The candidate contains this nested module, its dedicated `.github/workflows/future-federation.yml`, and exactly two coordinator-authorized accepted-main baseline restorations: the missing `os` import in `cmd/circula/auditanchor.go` and the departure regression fixture cap in `internal/service/departure_pagination_regression_test.go`. Their exact blobs and introducing commits are recorded in `RECOVERY-20261119.md`. Root dependencies, shared workflows, migration registries, routes and active session/auth/store work are unchanged. The feature remains disabled and integration HOLD; no core integration, activation, peer traffic or settlement authority is introduced.

The complete local suite used the actual pinned `modernc.org/sqlite v1.60.1` driver and default build tags. Independent review covered all eight production files at `5cf90f4ea3f9bcd17d4c7eaa611759080895c4dc`, plus the composition, protocol tests, fuzz extraction and workflow metadata follow-ups. Production bytes are unchanged from that complete source review. The three-platform provider workflow must still execute at the new publication head. Windows runs ordinary tests; Linux/macOS run race tests.

New reconciliation controls cover all ten v2 signed string/key-selector inputs with six malformed UTF-8 patterns each, exact valid public Unicode/U+FFFD/decomposed text/JWS bytes, and the existing ASCII-only v2 identifier grammar. A new bounded property target checks that every successful v2 signature preserves exact input JSON and verifies the compact signing input independently. No v2 production bug is asserted merely because the legacy v1 signer needed a repair; current v2 source already validates identifiers and public UTF-8 before marshaling. These new controls passed in the combined suite and exact 100,000-execution v2 fidelity fuzz run recorded above.

The original `FuzzDiscoveryWire` target is extracted from `store_test.go` into `protocol_fuzz_test.go` together with its unchanged `fixtureNow` declaration. Both the function body, including seeds, and the clock declaration are byte-identical to reviewed source `5cf90f4ea3f9bcd17d4c7eaa611759080895c4dc`. Its bounded command now names `envelope.go protocol.go protocol_fuzz_test.go`; no production code, driver, assertion or full-module race/vet requirement is removed. This follow-up avoids the shared environment's unrelated SQLite fuzz-instrumentation resource limit; the extracted unchanged parser target subsequently passed exactly 100,000 executions on the recorded production source.

`SIGNING-VALIDATION-20261119.md` preserves the earlier v1 repair evidence. The following archive validation, audit and handoff references are historical and must not be read as new combined-source passes or current completion percentages.

---

# Standalone Phase-A implementation validation — 2026-11-16

Toolchain: Go1.26.11 linux/amd64. Only `future/federation/` changes. Integration remains HOLD, no production endpoint/worker registered and no push/merge/release attempted.

Observed commands and results:

- `GOGC=20 GOMEMLIMIT=450MiB GOMAXPROCS=1 go test -p 1 -race -cover -timeout 90s ./...`: PASS, 69.4% statements, final run3.247s.
- `go vet ./...` with constrained compiler environment: PASS.
- `go mod verify`: all modules verified.
- `go test -p 1 -run '^$' -fuzz FuzzDiscoveryWire -fuzztime=5s -parallel=1 -timeout 45s ./...`: PASS,14,683 executions; performed before later quota/backup tests, with unchanged protocol parser.
- `go run golang.org/x/vuln/cmd/govulncheck@v1.1.4 ./...`: PASS, “No vulnerabilities found.” This is point-in-time advisory evidence, not a security certification.
- `git diff --cached --check`: PASS after preserving legal text and normalizing extra EOF blank lines in copied notices.

The suite tests signed malformed/duplicate/unknown payload/header rejection; issuer/audience/algorithm/type/expiry/settlement rejection; concurrent replay and restart; SQL trigger failure rollback; ID and revision equivocation; stale publication after revocation; rotation and revoked-key visibility; explicit consent/idempotent outgoing requests; partition/backoff; signer-outage withdrawal surviving departure/retention; expiry body trimming; trusted-peer rate and inbox quota; closed-DB backup restore; no-JS/CSRF/admin/private-tenant checks; XSS escaping; request bounds; SSRF/mixed-DNS/reserved IP denial; synthetic HTTP redirect/oversize/slow/error behavior and secure TLS/proxy/timeouts by construction.

Synthetic transport fixtures do not establish interoperability with a live external peer or validate production certificate/key custody. No manual browser/screen-reader test or actual Circula auth/store/departure/suspension adapter and ledger regression was run. Independent audit, protocol/key ADR approval and remaining acceptance gaps in ACCEPTANCE.md are real gates. Unit coverage does not establish completion or release readiness.

## Independent audit continuation

See AUDIT.md for findings, fixes, exact final evidence and remaining gates. Final nested race suite PASS71.3% (17.962s), vet/modverify PASS, wire fuzz9,504 inputs PASS and independent govulncheck no vulnerabilities. Critical safety writes require one affected row, schema/view corruption fails closed, capacity permits1000-publication departure, and source-independent lifecycle withdrawal is tested. HOLD remains.

## Master continuation — frozen source a833a8636724ac437d9680f9689f28a7257c7419

Date: 2026-11-16. Go 1.26.11, GOGC=20, GOMAXPROCS=1, GOMEMLIMIT=450MiB, shared GOCACHE=/workspace/scratch/0afb7942ff6b/go-build-cache. Commands ran inside future/federation only.

- `go test -p 1 -run '^TestMaster' -count=1 -timeout 90s ./...`: PASS, 0.026s.
- `go test -p 1 -race -cover -count=1 -timeout 90s ./...`: PASS, 15.312s, 74.2% statements.
- `go vet -p 1 ./...`: PASS, no diagnostics.
- `go mod verify`: PASS, all modules verified.
- Static notice inventory: 26 modules, 33 retained files; all referenced files exist. No dependency changed.

Raw output is under recovery-evidence/. An initial focused build found an unused strings import; it was removed. An intermediate successful run overlapped a small response-contract edit and is not used as final qualification. The results above are from the subsequent frozen committed source.

New regressions exercise malformed/duplicate/unknown/oversized queries and bodies before host callbacks, discarded malformed form values, PostForm-only mutation data, valid paginated native form submission, and token callbacks revoking/replacing sessions, revoking member authority or canceling context. Oversized bodies retain HTTP 413; malformed/ambiguous forms return 400. Existing cryptographic, persistence, concurrency, withdrawal, transport and failure suites ran in the full race command.

No new advisory scan, protocol fuzz, live peer, browser, manual accessibility, production key-operation or core-integration claim is made. Historical scans/fuzz remain at the sources recorded above and in AUDIT.md. The unchanged protocol/transport does not justify claiming refreshed external qualification.
