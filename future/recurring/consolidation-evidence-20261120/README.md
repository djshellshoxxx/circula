# Recurring consolidation evidence

This directory distinguishes the initial source composition from the later executed qualification. The tested source is `1e0838e04310bb2a19a585d69e40426b3c50fb71`, tree `d34913b1dc28aa735bc180d3b152b3394144ddb2`, with ordered parents `94deb0a20df07a3b38c8a905254645d6cd3019f2` (accepted future target) and `1e0e7e2f12a77d6d8969d1593ecdb6309bb13c48` (recovery source). The qualification follow-up changes only documentation and evidence.

## Files and interpretation

| Files | Scope |
| --- | --- |
| `provider-initial.json`, `composition-source.json`, `static-review.json` | Discovery and source checkpoint before any composed-source runtime run. Their then-current pending-qualification wording is deliberately preserved. |
| `gitleaks-receipt.json`, `gitleaks.log` | Initial full-worktree scan with unchanged repository configuration. |
| `independent-source-review.json` | Independent read-only disposition pinned to the tested source commit and ordered parents. The reviewer did not run runtime tests. |
| `qualification-module.json` and six referenced logs | Actually executed isolation guard, full real-driver module race/coverage, vet, checksums and two 100,000-execution calendar fuzz campaigns. |
| `qualification-core.json`, `core-fixtures.log` | Four focused accepted Core regression tests with race instrumentation. The command package compiled, with no tests matching the filter. |
| `provider-final.json` | Read-only source/target refresh after qualification. The original PR remains unpublished at the composed head and still targets the older topic. |
| `gitleaks-final-receipt.json`, `gitleaks-final.log` | Final assembled-worktree scan with the same unchanged configuration, version and redaction settings. The output is captured outside the scanned tree before being copied here. |
| `SHA256SUMS` | SHA-256 hashes of this evidence directory and the parent consolidation report, excluding this checksum file itself. Paths are relative to the repository root. |

## Results

The module race log has 48 ordinary `Test...` functions and 2 `Fuzz...` seed entrypoints, or 50 named top-level entries. All 160 test/subtest terminal events passed; zero failed and zero skipped. Statement coverage is 80.0%. Both calendar campaigns reached exactly 100,000 executions and passed. Module vet and checksum verification exited zero. The focused Core race log has 4 ordinary top-level tests and 6 passing test/subtest terminal events, with zero failed or skipped. Package-level pass events are not counted as tests.

The qualification commands used the actual pinned SQLite drivers and default build tags. Every command's exit code and source-hash check is recorded. Resource limits were `GOMAXPROCS=1`, `GOGC=20`, `GOMEMLIMIT=256MiB` and serial package compilation. The race commands omitted DWARF using `-ldflags=-w`; no driver, assertions or race instrumentation were removed.

## Limits and independent reproduction

The seven original provider Core jobs and the three native recurring jobs must qualify the eventual published composition. Native Core jobs are separately opt-in through the accepted `ci:cross-platform` label. Historical successful module jobs and the local Linux evidence do not qualify an unpublished head on macOS or Windows. No full Core suite, complete Core lint, browser, reproducibility or release-build campaign was run by this worker.

The inherited original source log excerpts and SHA-256 hashes are in `composition-source.json`; full provider logs remain linked by their original job IDs. Those original failures were diagnosed from the logs without rerunning a pre-fix vulnerability reproduction. The sole original merge whitespace finding is an unmodified upstream license notice's blank final line; its source bytes remain preserved. The authored qualification follow-up has no whitespace errors.

Verify the recorded file digests from the repository root using:

```sh
sha256sum -c future/recurring/consolidation-evidence-20261120/SHA256SUMS
```

All module and selected Core/workflow input hashes in both qualification receipts can be checked against the tested Git commit. The module remains disabled and on integration HOLD, with no Core activation, foreign future-module change, remote branch update, release or deployment performed by this worker.
