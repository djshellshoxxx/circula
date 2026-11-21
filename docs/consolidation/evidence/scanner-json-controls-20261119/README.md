# Scanner controls after Core evidence reconciliation

Date: 2026-11-19 Pacific. Starting Core source: `22c767a90f69d286642ef96e6b3c174836637a3c` (PR #125).

## Failure and repair

The [current-head quality job](https://github.com/djshellshoxxx/circula/actions/runs/44660259330/job/136036460029) failed in the scanner-control harness before the repository scan. The harness expected 18 literal-value findings. Main's preserved cursor, catalogue and branch-inventory evidence stores those same values with JSON Unicode escapes, leaving eight literal findings in the original bytes. The old harness failure was reproduced locally with the pinned Gitleaks v8.28.0 scanner.

The repair checks both representations explicitly. The untouched repository bytes must produce exactly eight findings before the exceptions and zero after. Temporary copies normalize JSON string encodings, while asserting identical JSON values and unchanged physical lines. Those copies must reproduce all 18 findings before exceptions and zero after. Every replacement, extension, path, adjacent-credential and independent-rule control still runs against the decoded copies. Every excepted value must actually be present in its test input.

Neither `.gitleaks.toml`, the scanner version, default rules, entropy thresholds, workflow permissions nor historical evidence files were changed. The existing post-snapshot SHA-256 manifest remains authoritative for its original bytes.

## Executed evidence

- Original harness: exit 1, reproducing the provider's 18-versus-eight mismatch.
- Revised harness: exit 0, all ten real-scanner controls passed; [complete metadata-only output](controls.log).
- Whole-repository directory scan after the script repair: exit 0, zero findings over approximately 22.56 MB. This scan predates subsequent continuation documentation and requires a final scan after integration.
- Python compilation, exception structure, JSON value/line preservation and `git diff --check`: passed.

Commands used the checksum-verified Go 1.26.11 toolchain and `go install github.com/zricethezav/gitleaks/v8@v8.28.0`. The runner used the resulting executable; CI continues to invoke the same version through `go run`.

```sh
python3 scripts/check_gitleaks_config.py -- /path/to/gitleaks
gitleaks dir . --no-banner --redact
```

This record concerns this source and directory-scan scope. It does not assert that unrelated repositories, Git history or external systems were scanned.
