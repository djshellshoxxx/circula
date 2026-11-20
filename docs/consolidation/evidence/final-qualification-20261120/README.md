# Final qualification evidence — 2026-11-20

This directory preserves actual Core execution outputs, summaries and dated provider/ref observations supporting the [consolidation checkpoint](../../QUALIFICATION_CHECKPOINT_20261120.md). Packaging and reporting do not rerun a check or alter its result. The evidence-only publication branch is `audit/core127-final-evidence-20261120`, based on the unchanged published Core candidate.

## Source identities

| Role | Exact commit or tree |
|---|---|
| Published Core PR #127 | `be9a5731bc93dcbf7b57301a8d2538a13af360bd` |
| Published complete tree | `7ba56914025bccada81f90a326e05233c45189a7` |
| Complete race execution | `36112a8614f4719415ec2e208521aef28ff3e69c` |
| Static/fuzz/load checkout | `69ca540dd9a1c01471de613e476b76babb87b31a`, complete tree equal to published tree |
| Original normal execution | `a9cdcb80179df1469b1443dcb79f1df00b1857fb` |
| Unchanged main | `aa01563fcc891da60fefebb8335eca27855678e8` |

Independent composition review found only documentation/evidence differences between the race source and published Core; their executable source is identical. The [source-equivalence receipt](source-equivalence.json) records those seventeen documentary paths and preservation of all twenty-three original audit files. Whole-tree equality applies to static/fuzz/load qualification. The historical normal failure retains its original source.

## Contents and interpretation

| Files | Evidence |
|---|---|
| [Race receipt](full-race-receipt.json), [summary](full-race-summary.json), [raw gzip](full-race.jsonl.gz) | Full command/environment/times, exit 0 and unmodified JSONL. 1,293 top-level passes, 3,255 passing events, zero failures, six skips; 16 passing packages and two no-test helpers. |
| [Normal receipt](full-normal-receipt.json), [summary](full-normal-summary.json), [raw gzip](full-normal.jsonl.gz) | Original exit 1: 1,292 top-level passes, one anchor-fixture failure, six skips. The complete race run includes the corrected fixture. |
| [Fuzz campaign summary](critical-fuzz/campaign-summary.json), [command summary](critical-fuzz/summary.json), adjacent logs/receipts | Five successful count-bounded campaigns, 1,400,000 executions; separate from race fuzz-seed entrypoints. |
| [Load receipt](load-smoke-receipt.json), [summary](load-smoke-summary.json), [raw JSONL](load-smoke.jsonl) | Two passing load/QA008 cases; a correctness probe, not production capacity qualification. |
| [Static README](CORE_STATIC_README.md), [summary](core-static-summary.json), [exact bundle](core-static-20261120.tar.gz) | Full configured lint with zero issues, govulncheck against the published Go advisory database with zero findings, successful module verification; original outputs and source/tool metadata. |
| [Archive verification](archive-verification.json) | Raw-log lengths/hashes and verification of the static bundle's original eighteen-entry manifest. |
| [Provider snapshot](provider-snapshot.json) | Individual exact-head jobs at 18:14:31 UTC; pending/skipped results remain distinct from success. |
| [Original merge receipt](verified-merges.json) | Seven ordinary merges observed at 18:19:52, with parents/trees/refs; includes onboarding after the provider snapshot. |
| [Analytics gates](analytics-final-gates.json), [merge verification](analytics-merge-verification.json) | Later eighth merge at 18:30:20; seven applicable Core and three native successes, one conditional Core skip. Original snapshots remain unchanged. |
| [Ref preservation](ref-preservation-snapshot.json) | 18:23:46 local Git snapshot: 148 remote-tracking branches, 18 local branches, zero tags, 15 disabled/HOLD future targets and unchanged main. |

The static README is copied outside the archive for convenient reading. Its original internal manifest covers eighteen evidence files and does not hash itself; the archive contains nineteen regular members including that manifest. The enclosing directory's `SHA256SUMS` covers the completed report and packaged artifacts separately; it does not hash itself.

## Lossless compression and verification

Gzip provides storage compression only. Decompression restores original bytes without line normalization, filtering, sorting or omitted failures:

| Uncompressed file | Bytes | SHA-256 |
|---|---:|---|
| `full-race.jsonl` | 2,916,856 | `31b2f4b9ed3b6b6750bef6052abb28261d0d1e9db33d04ed4042b9204bfeba52` |
| `full-normal.jsonl` | 3,096,970 | `8287d0efdbbbbc2807dc0e1ef856b2dbb2fd979dcc8ce3b10f6b7fa4ea8c9ea8` |

From this directory, verify the completed package manifest and compare the decompressed hashes and lengths with the table:

```sh
sha256sum -c SHA256SUMS
gzip -cd full-race.jsonl.gz | sha256sum
gzip -cd full-normal.jsonl.gz | sha256sum
gzip -cd full-race.jsonl.gz | wc -c
gzip -cd full-normal.jsonl.gz | wc -c
```

The static archive preserves the original `core-static-20261120/` layout. Its original checksum manifest has SHA-256 `cb7a2d199c1e0381ca0fdb6d1d8578f080f0d6edc0462e2d97e3cf7a94c87b13`. After verifying the enclosing archive checksum, extract into a fresh temporary directory and verify all eighteen original entries:

```sh
qualification_tmp=$(mktemp -d)
tar -xzf core-static-20261120.tar.gz -C "$qualification_tmp"
(
  cd "$qualification_tmp/core-static-20261120" || exit 1
  sha256sum SHA256SUMS
  sha256sum -c SHA256SUMS
)
```

`archive-verification.json` records that all eighteen archived contents match the original manifest. Exact command arrays, raw logs, source fingerprints, timestamps, tool versions/build information and the execution harness remain intact. Absolute workspace paths are provenance; the copied bytes/hashes allow verification without those workspaces.

## Limits

Top-level tests, subtest events, fuzz campaigns, package outcomes and command outcomes are different measures and must not be added into one test total. Skipped, queued and running results are never promoted to passes. The earlier killed lint attempt and Chromium's pre-page-creation EPERM failure remain historical limitations. Complete final-head provider/native/browser acceptance remains required.

Vulnerability qualification is bounded by the actual database response, last modified `2026-11-11T02:51:11Z`. Root-module results do not qualify isolated future modules or absent Core adapters. Missing historical federation logs are not recreated here. The checkpoint identifies that provenance limit and the separate policy, peer, device, hardware and production acceptance work.
