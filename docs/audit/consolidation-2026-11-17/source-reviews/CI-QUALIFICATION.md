# Independent CI and release qualification review

Baseline: `ef54238ee9ebc2184919b6a8ec30c3398df55ed5` (merge of PR #69), whose tree equals reviewed PR head `163fe1596bd7cfb8c2a22891629fc4459e6b639c`.

## Verified provider evidence

- [PR run 44226365456](https://github.com/djshellshoxxx/circula/actions/runs/44226365456): seven successful Linux jobs; the native matrix placeholder is intentionally skipped.
- [Native dispatch 44228060824](https://github.com/djshellshoxxx/circula/actions/runs/44228060824): all nine jobs successful at the same exact head. Windows runs `go test -count=1 ./...`; Linux and macOS use race detection. No Windows race or ACL-qualification claim is supported.
- E2E log [134546021540](https://github.com/djshellshoxxx/circula/actions/runs/44228060824/job/134546021540): **584 passed**, not the older 579-test figure. Browser is Chromium, with axe and keyboard checks. CSP is bypassed for axe injection; security-header unit tests are separate. This does not replace manual screen-reader testing.
- Lint log [134546021993](https://github.com/djshellshoxxx/circula/actions/runs/44228060824/job/134546021993): `0 issues.` and `No vulnerabilities found.` Go 1.26.11. Includes action pins, hygiene, licenses, gitleaks, vet, Python tooling and traceability.
- Reproducibility log [134546021947](https://github.com/djshellshoxxx/circula/actions/runs/44228060824/job/134546021947): two linux/amd64 binaries both hash `bf0cc9d1a4b2f3ef4746c5510aa35deef0a5394616d3858919c2ed421200a6c8`.
- `ci-provider-evidence.json` records jobs, steps and observed statuses. PR #70 and merge-main runs were still executing/queued at this snapshot; do not label them successful from this record.

## Coverage and limits

| Area | Verified mechanism | Remaining boundary |
|---|---|---|
| Go tests / race | Linux full race; macOS bounded parallel full race; Windows ordinary full suite | Windows race, Linux arm64 native runtime and Darwin arm64 native runtime are not included |
| Builds / packaging | All five OPS-003 targets, readonly module sums, per-target CycloneDX SBOM, archive checksums and secret/PII scan | Native execution on arm64 targets and real tag publication not performed |
| Reproducibility | Two linux/amd64 binaries, separate paths/caches, `-buildvcs=false` | Does not establish archive or all-target reproducibility; package-release uses a different GOFLAGS setup |
| Migrations | Full Go suite includes sequential/golden migrations, version refusal and backup-before-upgrade tests; hygiene runs destructive-SQL checks | Current specification status and owner thresholds still govern acceptance |
| Backup / restore | CLI tests include plaintext/encrypted snapshots, manifest tampering, restore TOCTOU, credential invalidation, swap crash recovery, QA-028 load and drill script | New file-creation hardening needs fresh exact-head tests; Windows POSIX-mode/symlink tests deliberately skip |
| SQLite failure recovery | Real SQLITE_FULL, Linux pre-commit WAL-write error and checkpoint database-write error; reopen/idempotency checks | Read, fsync, truncate, arbitrary device faults and uncertain commit outcomes remain unqualified |
| Fuzz / load | All seed corpora plus ten-second amount/strict-JSON fuzz; small concurrent contract/load workload | Not an exhaustive fuzz campaign or approved performance budget |
| Release gate | Explicit PASS/FAIL/SKIPPED/MANUAL evidence | DEC-023 unconfirmed, manual screen reader, independent security review, exploratory/native smoke and release notes remain gates |

## Reproducible defect and local repair

`scripts/release-gate.sh:61` grepped the whole thresholds document for `UNCONFIRMED`, which also appears in its permanent instructional footer. A synthetic copy with every genuine table decision populated still failed the full gate's threshold classification. The approved focused repair replaces this grep with a Decision-cell validator, retains fail-closed behavior for unconfirmed/missing/malformed decisions, adds full-script regressions, and wires tests into CI. The real thresholds document remains unchanged and continues to fail. See `docs/audit/RELEASE-THRESHOLD-CHECK-2026-11-18.md` in the repair branch.

After independent review identified omitted-row and invented-absolute-row bypasses in the first unpublished revision, the checker was tightened to require all ten existing core names exactly once and restrict the absolute exemption to its three existing names. Eight regression methods (including ten failing-before parser/full-gate subcases) now pass. All 26 Python tooling tests passed at the final local head `284d756a28d2b0a0e453da650446cd5b844c39db`. Shell syntax, hygiene, action pins, generated traceability and diff whitespace checks pass. No unrelated Go code changed, so the existing exact-tree Go results are not rerun locally for this tooling-only repair. The integration owner still runs the configured PR gates at the published repair head.

## Recommended gates for pending backup/store hardening

Run failing-before reproductions, focused backup/store suites and their race versions; preserve restore/upgrade and ledger tests; run aggregate Go, vet, traceability, hygiene and existing PR CI. Add `ci:cross-platform` before the relevant PR event when practical because file creation and rename semantics differ on Windows. Successful build artifacts alone do not prove native behavior. These recommendations do not authorize or trigger release publication.

No release, deployment, tag, remote branch update, workflow dispatch or remote review mutation was performed by this workstream.
