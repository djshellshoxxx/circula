# Release gate (QA-039)

Run `make release-gate` (or `scripts/release-gate.sh [-o DIR] [-s check,check]`). It never tags, publishes or pushes.
It writes `gate-<UTC>.md` (default `docs/qa/evidence/`) with commit, tree state, Go version, host and one row per check.

| Check | What runs | Needs |
|---|---|---|
| fmt | `gofmt -l .` empty | Go |
| vet | `go vet ./...` | Go |
| race | `go test -race -count=1 ./...` (about 10 min on 4 vCPU) | cgo |
| lint | `golangci-lint run` (0 issues; `$HOME/go/bin` preferred) | golangci-lint 2.15 |
| contract | `TestAPIMatchesOpenAPI` | Go |
| golden | golden migration tests (QA-027, 10,000-entry fixture) | Go |
| e2e | `make e2e`: axe + keyboard + ARIA proxy, 93 tests | Node 20+, Chromium |
| load | small `CIRCULA_LOAD` smoke (4 workers, 20 members, 25 trades) | Go |
| vuln | `govulncheck` v1.1.4 | network to vuln.go.dev |
| pins | `scripts/check-action-pins.sh` | none |
| hygiene | `scripts/check-hygiene.sh` (QA-043, QA-037) | none |
| drill | `scripts/restore-drill.sh`, writes `drill-<UTC>.json` (QA-029) | none |
| defects | no open `A-block` rows in docs/qa/DEFECTS.md (UX-060) | none |
| thresholds | no `UNCONFIRMED` in docs/qa/THRESHOLDS.md (QA-040, DEC-023) | owner decision |

Manual items, always printed as MANUAL and never auto-passed: screen-reader run (docs/qa/SCREEN-READER.md), L14
exploratory records, SEC-033 independent review, L13 run smoke per DEC-003 target, release notes (OPS-026).

## Evidence file format
Markdown: header bullets (generated UTC, commit, tree clean/DIRTY, Go, host, FAIL count) then a table
`| Check | Result | Note |`. Result is PASS, FAIL, SKIPPED or MANUAL. PASS means exit 0 on that host only.
For a release, copy the file to `docs/qa/evidence/<version>/` and link it, the drill record and the manual records from
`docs/releases/<version>.md` (docs/qa/EVIDENCE.md).

## Known state
The gate currently FAILS by design: thresholds are unconfirmed (QA-040), and govulncheck cannot reach its database from
the sandbox. The sample record `docs/qa/evidence/gate-20261021T160332Z.md` was produced in that sandbox; its lint FAIL
was a PATH problem (older golangci-lint first on PATH), since fixed in the script.
