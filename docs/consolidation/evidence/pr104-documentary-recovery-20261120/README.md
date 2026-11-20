# PR #104 historical documentary recovery — 2026-11-20

This package preserves **all 102 documentary files missing from the reviewed Core candidate**, byte for byte, from [PR #104](https://github.com/djshellshoxxx/circula/pull/104). Their original contents total **5,206,983 bytes**. It also restores access to the ten federation qualification logs previously recorded as unavailable.

**The archive contains historical evidence.** Words such as “current,” “passed,” and “complete” inside original reports retain their original source and date. They do not describe present Core or future-module qualification. Original failures, intermediate reviews, patches, and source snapshots remain unchanged. No archived command or product test was executed during recovery.

## Source and inclusion decision

| Identity | Exact value |
|---|---|
| Source branch | `integration/circula-master-consolidation-20261119` |
| Source commit | `bcdc569cbc483761bf2d0e2d7594636ac1713d7d` |
| Source tree | `629fdb415e4181ee349947526f39a7387e2d4d40` |
| Compared Core commit | `eef0c6d8c832ce994db1019eb5e7d6a41800b8ef` |
| Core tree | `029e5fdda2d03347389397576ec08701ae04cec2` |
| Common ancestor | `c3c991aa3c12a11edce2874e9444b9d2a99f746c` |
| Earlier source-only repair | `707a4363a86922d152ed2d203c9ea184fe4b323f` |
| Evidence-branch base | `49dc8acc781790635fca8ed32882a7a27971c8a1` |

The complete source delta changes 115 paths: eleven are byte-identical in the Core candidate; two older Python alternatives, `scripts/read-export.py` and `scripts/test_read_export.py`, differ and are excluded; and 102 documentary paths are missing. Those 102 paths are exactly the documentation-only delta from `707a4363` to `bcdc569c`, all under `docs/audits/master-consolidation-20261119/`, all mode `100644`.

**Documentary verdict: preserve this package without merging the older PR #104 runtime.** The documentary gap is closed. Equivalence or supersession of the two Python alternatives remains the separate Core containment review's decision. This recovery changes no remote refs and does not itself close the PR.

## Package and integrity

| File | Purpose |
|---|---|
| [historical-evidence.tar.gz](historical-evidence.tar.gz) | All 102 original files; 1,205,996 compressed bytes. |
| [payload-manifest.json](payload-manifest.json) | Every original path, Git blob, SHA256, length, archive entry, source commit, and excluded path. |
| [archive-verification.json](archive-verification.json) | All 205 original ZIP payload checks and the old 20-file index reconciliation. |
| [federation-independent-review.json](federation-independent-review.json) | Per-log hashes, event recount, ten recorded commands/statuses, all 21 source comparisons, and independent scanner review. |
| [secret-scan-review.json](secret-scan-review.json) | Complete payload scan, actual exit 1, 24 redacted findings, and classification. |
| [historical-object-remote-check.json](historical-object-remote-check.json) | Read-only checks for the exact historical commit and trees. |
| [verify_recovery.py](verify_recovery.py) | Read-only integrity verifier and optional text viewer. |
| [verification-result.json](verification-result.json) | Actual final recovery checks and execution scope. |
| [SHA256SUMS](SHA256SUMS) | Hashes of all package files except this checksum file. |

The deterministic USTAR/gzip wrapper preserves original bytes and relative paths. Wrapper metadata uses regular files, mode `0644`, zero mtime/UID/GID, and empty owner/group names; these do not describe original execution-host metadata. Every payload was compared byte for byte with its Git blob, and Git blob SHA1 plus SHA256 were recomputed. All 102 wrapper entries verify. All 56 original JSON documents and 44 nested JSON members parse.

The original ZIP remains unchanged: blob `55c4e9817609abb9269923f86814911703165639`, 522,344 bytes, SHA256 `ba01ee9653a3bfbc07734348594f878b45cbd5ca87ad797bff64fb09930f5d3f`. Its 206 entries comprise 205 payloads plus the internal manifest. Every payload length/hash/CRC matches. Internal and external manifests are byte-identical, SHA256 `3592091bfac5373e843121ee1a478ddb37dc4793f5936075a7ce94ee0efa805a`. No duplicate/unsafe paths, symlinks, encrypted members, or directories were found.

All 20 entries of the old `VERIFICATION_FILES.json` match source blobs, including four already present in the Core candidate and not duplicated here. The original manifest's excluded symlink and separately referenced full Git recovery bundle remain recorded. That bundle was never among these 102 files.

## Recovered federation results and limits

The ten logs and `receipt.json` reside inside the original ZIP under `evidence/branch-inventory/legacy-recovery/federation-combined-qualification/`. All eleven entries match their manifest lengths/hashes. Receipt SHA256 is `ee9a61947ead62d9c1531a81a697345cfb30e221f9fe47e844c8f1bde35e29d6`. The recorded run occurred on 2026-11-19, 13:12:06–13:13:58 UTC, with Go 1.26.11 on Linux/amd64 and `modernc.org/sqlite v1.60.1`.

| Historical event category | Passed | Failed | Skipped |
|---|---:|---:|---:|
| Top-level tests | 36 | 0 | 0 |
| Ordinary subtests | 105 | 0 | 0 |
| Top-level fuzz targets | 3 | 0 | 0 |
| Fuzz seed cases | 137 | 0 | 0 |
| **Named terminal events** | **281** | **0** | **0** |
| Package terminal result | 1 | 0 | 0 |

The race log contains 1,129 valid JSON events. All run/terminal names reconcile, all 36 top-level test names match the receipt, and coverage is recorded twice as 74.5%. No race-warning or explicit failure marker was found. Seed cases comprise 55 legacy-string, 80 V2-string, and 2 discovery-wire cases. Each of the three separate fuzz logs ends at **100,000 executions**, `PASS`, and package `ok`; all counter sequences match the receipt.

All ten recorded steps have exit 0, no outer timeout, and unchanged source. Empty formatting and vet logs have no success text; their status comes from the retained receipt. These counts describe recovered historical records, not new product executions.

The receipt declares historical commit `ca9af7543cc560d4a798b6e446d85ec38f740595`, tree `2b1400be19b42ee63c86980666a71d1c88e07811`, and federation module tree `81a58c2b2b05972f977e1997ed0ce978f6ca4a12`. Those objects were unavailable in the local database. The separate remote-check receipt records GitHub API 404/NOT_FOUND for all three exact commit/tree endpoints through the active connection. No source/tree relationship is assumed from a missing object.

Accepted federation commit `0aeb5a1446798019141a3f6f5bcb1e7a24b70c47` matches **19 of 21** historical fingerprints: all 17 Go files, `go.mod`, and `go.sum`. The differences are `.github/workflows/future-federation.yml` and `future/federation/feature.json`.

The historical log-availability/hash gap is closed. This does not qualify accepted-head metadata, workflow, provider checks, platform matrix, or current dependency vulnerabilities. Hash agreement establishes retained-evidence consistency, not independent attestation of the original host/invocation. All fifteen future streams retain their independent HOLD paths.

## Scanner result

The unchanged repository Gitleaks configuration scanned **307 text payloads / 8,465,994 bytes**: 101 original non-ZIP documents plus all 206 ZIP members. Members were materialized only after path/type/length/hash/CRC checks; no archived source was executed.

The full scan **exited 1 with 24 generic-api-key detections across four payloads**. Independent review reproduced and classified every detection: twelve occurrences of four locally verified Git commit IDs, three occurrences of the ordinary disabled/HOLD phrase, one VAPID/encryption topic phrase, and eight notification source SHA256 fields across two byte-identical receipts.

No rule, configuration, inline exception, or original payload changed. This is a reviewed false-positive result, **not a clean raw-payload scan**. The notification source objects/bytes were unavailable locally: the review identifies explicit before/after checksum fields and does not claim source-byte recomputation. The separate scan of authored metadata is recorded in `verification-result.json`.

## Verify or read evidence without extraction

From the repository root:

```sh
python3 docs/consolidation/evidence/pr104-documentary-recovery-20261120/verify_recovery.py
```

The verifier checks package hashes, all 102 original SHA256/Git blob identities, all 205 ZIP payloads, manifest equality, safe entry types/paths, CRCs, and JSON parsing. It invokes no subprocess, network request, archived command, or product test. To view a historical log after verification:

```sh
python3 docs/consolidation/evidence/pr104-documentary-recovery-20261120/verify_recovery.py \
  --show-zip evidence/branch-inventory/legacy-recovery/federation-combined-qualification/fuzz-wire.log
```

Use `--show-original` with an exact inventory path to view an original document. Old relative links and absolute scratch paths remain historical data. The [pinned original directory](https://github.com/djshellshoxxx/circula/tree/bcdc569cbc483761bf2d0e2d7594636ac1713d7d/docs/audits/master-consolidation-20261119) remains a navigation source.

Historical source objects and the separately referenced full Git bundle would be needed to connect every recorded source/tree identity independently. Their local absence does not block preservation of verified PR #104 bytes. Other modules' logs are preserved and checksum-verified; this recovery does not rerun products or declare new qualification. Core integration, PR disposition, and publication remain separate consolidation decisions.
