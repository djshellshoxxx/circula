# Final Core static and dependency qualification — 2026-11-20

All three requested commands executed successfully against the exact published Core source. No source repair is indicated by these results.

| Identity | Value |
| --- | --- |
| Published commit | `be9a5731bc93dcbf7b57301a8d2538a13af360bd` |
| Published and tested whole tree | `7ba56914025bccada81f90a326e05233c45189a7` |
| Equivalent local checkout | `69ca540dd9a1c01471de613e476b76babb87b31a` |
| Working directory | `/workspace/scratch/bfba682cba37/circula` |
| Tracked source fingerprints | 2,024; unchanged before and after every command |

The local checkout was clean at the start and after each command. The published and local commits have identical complete trees. Evidence is stored outside the repository worktree so recording it changes no tested source file.

## Actual results

| Command | Result | Wall time | Completed UTC |
| --- | --- | ---: | --- |
| Full `golangci-lint run --concurrency 1 --timeout 10m --verbose ./...` | Exit 0; 12 configured linters active; 0 reported issues | 46.477 s | 18:02:22 |
| Actual-database `govulncheck -json -db https://vuln.go.dev ./...` | Exit 0; source/symbol scan; no finding events | 23.233 s | 18:03:00 |
| `go mod verify` | Exit 0; all modules verified | 4.235 s | 18:03:31 |

There were zero failed or timed-out commands and no retries in this qualification attempt. These three static/dependency commands are not unit-test case counts.

### Full lint scope

The command covered the complete root module using the repository's unchanged `.golangci.yml`. It used no changed-file selector, analyzer override, new exclusion or automatic fix. All twelve configured linters were active: bodyclose, depguard, errcheck, gofmt, gosec, govet, ineffassign, misspell, nilerr, staticcheck, unconvert and unused.

The existing configuration filtered 107 test-file gosec diagnostics and 23 existing nolint-covered diagnostics; the final reported count was zero. Those filters were not introduced or changed by this qualification. The tool reported peak memory of 360.2 MB. The preceding killed full-lint attempt remains a separate historical failure; this successful command does not relabel it.

### Actual vulnerability database

The scanner connected to the explicit actual endpoint `https://vuln.go.dev`; no empty fixture, local substitute, alternate database or dependency rewrite was used. Its configuration record reports:

- Scanner: govulncheck v1.1.4.
- Project Go version: go1.26.11.
- Mode: source; level: symbol.
- Database last modified: **2026-11-11T02:51:11Z**.

The complete JSON stream contains one configuration event, one SBOM event, two progress events and 227 OSV database records. It contains **zero finding events**. The OSV records are fetched advisory data, not 227 findings in Circula. The SBOM lists eighteen root packages and twelve module entries including Circula and the standard library. Results are bounded by the database response and reported last-modified timestamp.

## Toolchain and resource profile

Project commands use the supplied Go 1.26.11 executable at `/tmp/circula-runtime-4d30/go/bin/go`, with the existing GOPATH and GOCACHE under that runtime directory. The supplied golangci-lint binary is version 2.15.0, built with Go 1.27.0; that binary-build version is distinct from the project's Go 1.26.11 analysis toolchain. The supplied govulncheck binary is v1.1.4, built with Go 1.26.11. Complete build metadata and binary hashes are retained.

All three commands ran sequentially with `GOMAXPROCS=1`, `GOGC=20`, `GOMEMLIMIT=1536MiB`, `GOFLAGS='-mod=readonly -p=1'`, `GOTOOLCHAIN=local`, `GOWORK=off` and `CGO_ENABLED=1`. Lint had a ten-minute analyzer deadline and an eleven-minute outer cap. Govulncheck had a ten-minute outer cap; module verification had a three-minute outer cap. No resource or network error occurred.

## Evidence index

- `preflight.json`: published/local identities, source manifest hash, resource profile, configuration hash, tool versions and binary hashes.
- `source-sha256.json`: complete tracked-file SHA-256 map.
- `go-version.log`, `golangci-version.log`, `golangci-buildinfo.log`, `govulncheck-buildinfo.log`: complete version and build-metadata outputs.
- `lint.json`, `govulncheck.json`, `mod-verify.json`: command arrays, timestamps, exit codes, source checks, memory snapshots and log hashes.
- Corresponding `.log` files: complete unmodified command output.
- Corresponding `-running.json` files: preserved initial start checkpoints; final state is in the completed receipt of the same phase.
- `qualification-summary.json`: parsed results and explicit scope limitations.
- `run-qualification.py`: exact execution and recording harness.
- `SHA256SUMS`: hashes of every evidence file except that checksum file itself; paths are relative to this directory.

To check the evidence hashes, run `sha256sum -c SHA256SUMS` from this directory.

## Boundaries

This execution qualifies static analysis and dependencies of the root Go module. Independent nested future modules are separately scoped. No unit/race/fuzz/native/browser suite or provider workflow was started by this worker, and no source, configuration, dependency, tool version, remote branch or PR was changed. The coordinator owns final integration and publication decisions.
