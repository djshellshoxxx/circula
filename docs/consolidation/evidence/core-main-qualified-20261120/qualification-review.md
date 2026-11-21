# Circula actual-main qualification receipt

**COMPLETE: all thirteen applicable jobs in three actual-main push workflows succeeded, and all thirteen decoded job logs explicitly identify the merged main checkout.** The final independent provider snapshot was taken at **2026-11-21 03:23:28 UTC** (**November 20, 2026, 7:23:28 p.m. Pacific Standard Time**).

| Identity | Verified value |
|---|---|
| Main commit | `87651d36558a8e421e41fbd76c514045d88c74ed` |
| Main tree | `43e65ee91a90ff70b70cc69d0b517b8367a2585f` |
| First parent | `aa01563fcc891da60fefebb8335eca27855678e8` |
| Second parent | `3ef43ac1e05360d2a1641742dc056afe8871879b` |
| Independent source SHA-256 | `9de685071744f0b9e260d53fc7d934cb4289c3b10f84a020db8e319227b77100` |

The final API read confirms the unchanged main ref, tree, and ordered parents. Every expected workflow is a completed successful `push` run at this exact main commit. Each of the thirteen job records also has that `head_sha`, and each retained log has the full hash immediately following its checkout `git log -1 --format=%H` command. The earlier synthetic premerge checkout `6eb49aa2b2c7859fd2a8ceda207ddbb01eaa9d65` was not substituted for any execution in this receipt.

## Thirteen qualified jobs

| Workflow / job | Job ID and provider log | Conclusion | Recorded result |
|---|---|---|---|
| Core: build release targets | [136698603681](https://github.com/djshellshoxxx/circula/actions/runs/44856110228/job/136698603681) | Success | Packaged build targets produced; `ci-dist` artifact uploaded. |
| Core: Windows | [136698603787](https://github.com/djshellshoxxx/circula/actions/runs/44856110228/job/136698603787) | Success | Sixteen package passes from the Windows test command, without the race detector. |
| Core: reproducible build | [136698603858](https://github.com/djshellshoxxx/circula/actions/runs/44856110228/job/136698603858) | Success | Two builds had the same SHA-256; isolated runner deployment smoke succeeded. |
| Core: Linux race | [136698603892](https://github.com/djshellshoxxx/circula/actions/runs/44856110228/job/136698603892) | Success | Actual-main artifact independently classified: 3,258 named test/subtest passes, six named skips, zero failures. |
| Core: fuzz smoke | [136698603901](https://github.com/djshellshoxxx/circula/actions/runs/44856110228/job/136698603901) | Success | Five campaigns passed: one million plus four times 100,000 executions, totaling 1.4 million. |
| Core: vet / format / lint / vulnerability checks | [136698603910](https://github.com/djshellshoxxx/circula/actions/runs/44856110228/job/136698603910) | Success | No secret leaks found, zero lint issues, no vulnerabilities found by the recorded scan; helper suites passed, including 24 Node tests with no failures or skips. |
| Core: macOS Intel race | [136698603986](https://github.com/djshellshoxxx/circula/actions/runs/44856110228/job/136698603986) | Success | Sixteen package passes using uncached race tests with bounded parallelism; no failure or race-report markers. |
| Core: browser / accessibility / keyboard | [136698604023](https://github.com/djshellshoxxx/circula/actions/runs/44856110228/job/136698604023) | Success | 597 passed, including all four loading feedback scenarios. |
| Core: API contract and load smoke | [136698604084](https://github.com/djshellshoxxx/circula/actions/runs/44856110228/job/136698604084) | Success | Contract and QA008 passed; 100 completed trades, zero insufficient-credit refusals, 200 committed write operations. |
| Native reader: macOS | [136698603597](https://github.com/djshellshoxxx/circula/actions/runs/44856110230/job/136698603597) | Success | 49 tests, no failures or skips. |
| Native reader: Linux | [136698603854](https://github.com/djshellshoxxx/circula/actions/runs/44856110230/job/136698603854) | Success | 49 tests, no failures or skips. |
| Native reader: Windows | [136698603942](https://github.com/djshellshoxxx/circula/actions/runs/44856110230/job/136698603942) | Success | 49 tests, no failures or skips. |
| Scorecard analysis | [136698602978](https://github.com/djshellshoxxx/circula/actions/runs/44856110197/job/136698602978) | Success | Execution and analysis upload succeeded; score 5.9 across eighteen criteria, with findings retained. |

The final macOS race step succeeded at **03:22:46 UTC**, and its job completed at **03:22:54 UTC**. Its test step ran from **02:43:16 UTC** with `go test -race -count=1 -p 2 -timeout=45m ./...`. Linux's race step ran from **02:42:57 UTC** to **03:13:39 UTC**. The conditional Windows-only/macOS-only and optional diagnostic steps that were skipped are listed in `result.json`; those skips are distinct from failed steps. No failed step was present.

## Original Linux artifact and independent counts

The original actual-main archive is retained as `linux-artifact/circula-actual-main-linux-test-evidence.zip`, artifact **14689307829** from Core run **44856110228**. Its **138,870 bytes** hash to:

`df5b84d700ad1193afe6244178384fc8ae7fa02eeab2fbe8482220661af89a54`

This equals both the provider artifact digest and the upload log's digest. The archive records main `87651d36558a8e421e41fbd76c514045d88c74ed`, `dirty=false`, exit code zero, and the command `go test -json -count=1 -race -timeout 30m ./...`. Its source digest independently matches the clean main fingerprint over 2,023 inputs. Only generated `docs/TRACEABILITY.md` is excluded from that fingerprint.

| Terminal result category | Pass | Fail | Skip |
|---|---:|---:|---:|
| All 3,282 terminal events | 3,274 | 0 | 8 |
| Named tests and subtests | 3,258 | 0 | 6 |
| Top-level named entries | 1,293 | 0 | 6 |
| Subtests | 1,965 | 0 | 0 |
| Packages | 16 | 0 | 2 |

Top-level passes comprise 1,270 `Test` functions and 23 `Fuzz` entries. Top-level entries and subtests partition named results; named results and packages partition all results. Counts were independently calculated from this downloaded actual-main archive, even though they happen to match the earlier premerge totals. No duplicate terminal identity, unknown terminal action, or failure was found. Member hashes and all twelve artifact validation results appear in `linux-artifact/result.json`.

The six named skips remain explicit. The inspected source guards cover a Unix ownership test needing root, three subprocess helper entry points, an opt-in load test, and an opt-in statistical sign-in timing test. Two helper packages have no tests. The separate API/load job executes its small load scenario. Source guard interpretation is labeled separately from the compact artifact's recorded terminal actions because the artifact omits runtime skip text.

## Scope and remaining limits

This receipt establishes the successful actual-main CI executions and their checkout identities. It does not establish production deployment, reference-hardware performance acceptance, manual acceptance, policy adoption, release approval, or acceptance of every requirement merely because a cited test passed. The runner deployment smoke was isolated CI activity. No product suite was executed again during this review, no shared checkout was changed, and no remote state was modified.

Scorecard job success is execution success. Its retained findings include missing main branch protection; no GitHub approval reviews in the two sampled changesets; no detected license; partial security-policy evidence; two container image digest-pin gaps; and a SAST detector finding. These findings remain visible alongside the actual Go lint/vulnerability execution and are not relabeled as passed controls. All eighteen criteria and their details are preserved in `native-and-scorecard/result.json`.

## Evidence package

- `result.json`: final normalized qualification, twelve passing gates, all thirteen job records, explicit checkout verification, skipped steps, and test summaries.
- `final-provider-observation.json`: full final main-ref/commit, three run records, thirteen job records, and Core artifact inventory.
- `log-manifest.json`: thirteen saved log paths, exact checkout witnesses, decoded UTF-8 byte lengths/SHA-256 hashes, and gzip byte lengths/SHA-256 hashes.
- `core/logs/` and `native-and-scorecard/logs/`: full decoded provider logs, encoded as UTF-8 and compressed with deterministic gzip timestamps; original line endings and ANSI sequences retained. These hashes describe the decoded provider text bytes, not an original HTTP transport encoding.
- `linux-artifact/`: original ZIP, provider metadata, independent counts, archive/member hashes, skip guard review, and focused receipt.
- `source-digest/`: separate read-only source-fingerprint calculation, exact modes and counts, clean/head/tree checks, and reproducible script.
- `native-and-scorecard/`: independent reader/Scorecard subreview, complete Scorecard findings, and its own checksum manifest.
- `inspect_linux_artifact.py` and `compile_receipt.py`: standard-library review tools used to produce the classifications and aggregate receipt; they do not execute product suites.
- `core/observation-*.json`: historical provider observations. Later entries label the separate earlier artifact-inventory observation time explicitly.
- `core/review-pending-at-20261121T025510Z.json`: the clearly timestamped historical pending review, superseded by the final `result.json` in this directory.
- `SHA256SUMS`: final hashes for every other file in this package, including nested manifests and the retained original ZIP.

The package is frozen after final checksum verification for deterministic archival and publication by the parent finalization task.
