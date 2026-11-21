# UX-047 setup precision qualification

Checkpoint: 2026-11-20 Pacific. This note records local qualification of the
bounded setup amount-validation correction on
`codex/ux047-setup-precision-20261120`, based on
`87651d36558a8e421e41fbd76c514045d88c74ed`. The resulting PR's exact-head checks
and subsequent actual-main checks are separate integration evidence. UX-047
remains **partial**.

The preceding Core consolidation is complete. Its exact main qualification
and historical PR dispositions are preserved in the
[post-merge continuation record](https://github.com/djshellshoxxx/circula/blob/8f936b6eb4eefbe85250ee5cf44a839ad361e4ce/docs/consolidation/CORE_POSTMERGE_CONTINUATION_20261120.md).
Those earlier results do not qualify this new source change.

## Defect and correction

The setup form offers precision one, but an invalid default credit limit or
sign-up grant previously produced "1 decimal places" in both the linked error
summary and the field correction. The existing `setup.amount.invalid` entry
had only an `other` form, and the caller supplied the field label first.

The correction supplies `one` and `other` forms and passes integer precision
as argument 1, as required by the existing catalogue formatter. The localized
label is argument 2 and the exact fixed-point example is argument 3. No key
or dependency is added; the catalogue remains at 1,458 entries. Parsing,
precision bounds, amount formatting, authentication, CSRF, setup-secret
controls and the early 422 return remain unchanged.

Two new regressions cover both amount fields at precisions 0, 1, 2 and 4:

- `TestSetupAmountValidationMessages` checks complete sentences and exact
  examples `1250`, `125.0`, `12.50` and `0.1250`.
- `TestSetupAmountValidationUsesSelectedPrecision` uses the real authenticated
  setup handler and CSRF flow. It checks HTTP 422, retained selected precision,
  escaped submitted values, exact summary links and inline errors, field IDs,
  `aria-invalid`, `aria-describedby` and absence of a created network.

Precision three is accepted by the unchanged bound and uses the same plural
branch and formatter; it is not a separate new case. No new browser or screen
reader execution is claimed by these local checks.

## Reproducible red-to-green evidence

The [evidence archive](evidence/UX047-setup-precision-2026-11-20.tar.gz) contains
the test-only baseline patch, raw test events, commands, timestamps, file
hashes, build diagnostics and both independent review records. Its
`MANIFEST.json` records each archived file's SHA-256 and size. It contains no
compiled executables.

The red execution ran against the base production source plus the two new
test files. Only the precision-one catalogue cases for the two fields and the
precision-one HTTP case failed. Precisions zero, two and four and the existing
malformed/out-of-range precision controls passed. The two parent test failure
events are included in the raw named-event counts; they are not additional
failing scenarios.

The HTTP regression is byte-identical between red and green, SHA-256
`f1eb0bc3b1b78e8566bc811e714440942d663e148ba638c694b37e7cd39395ef`.
The final five-file source diff reviewed independently has SHA-256
`f5078e1b03843cac7ab1944abb4456274d7be743381a653912e2fdaf68431a00`.
All post-fix test, vet and build executions retained those five final source
hashes.

| Local check | Result |
| --- | --- |
| Baseline regression selection | Expected exit 1; only the three precision-one scenarios fail. 18 named pass events and five named fail events, including the two failing parents. |
| Same focused selection after correction | Exit 0; 23 named pass events, zero failures or skips. |
| Complete messages package with race detector | Exit 0; 49 top-level tests, 351 named pass events including subtests, zero failures or skips. |
| Setup/onboarding handler selection with race detector | Exit 0; seven top-level tests, 17 named pass events including subtests, zero failures or skips. Includes setup-secret enforcement, restart persistence and concurrent first use. |
| Vet for both affected packages | Exit 0. |
| Executable build | Exit 0 from an identical full Git checkout, with normal VCS stamping enabled. The initial linked-worktree metadata failure is preserved below. |

Commands used the verified Go 1.26.11 toolchain. The test commands were:

```sh
go test -json -count=1 -p 1 -timeout=10m \
  -run '^(TestSetupAmountValidationMessages|TestSetupAmountValidationUsesSelectedPrecision|TestOnboardingMessages|TestEveryCatalogueEntryFormats|TestSetupRejectsInvalidPrecisionBeforeFormatting)$' \
  ./internal/web/messages ./internal/web
go test -json -race -count=1 -p 1 -timeout=10m ./internal/web/messages
go test -json -race -count=1 -p 1 -timeout=10m \
  -run '^Test(Setup|Onboarding)' ./internal/web
go vet ./internal/web/messages ./internal/web
go build -o /tmp/circula-ux047 ./cmd/circula
```

The archive preserves the exact absolute executable and output paths used;
the build command above substitutes a portable output location. Execution
used `GOTOOLCHAIN=local`, `GOMAXPROCS=2`, `GOMEMLIMIT=1GiB`, `GOGC=40` and
`CGO_ENABLED=1`, with task-specific dependency and build caches.

## Initial build failure and qualified rerun

The first executable build failed while obtaining VCS status, exit status
128. A diagnostic build with `-x` showed `git status --porcelain` running in
`/workspace`, rather than the linked worktree. The installed toolchain's Git
root rule requires a `.git` directory; the linked worktree has a `.git` file.
Direct Git status and revision commands in that worktree succeeded.

A full local Git checkout at the same base received exactly the five reviewed
files. Its binary diff and all five file hashes matched the feature worktree.
The unchanged normal build then passed with default VCS stamping enabled.
The binary records base revision `87651d36558a8e421e41fbd76c514045d88c74ed`
and `vcs.modified=true`, accurately identifying the uncommitted correction.
No source change or stamping override was used to obtain that result. Both
attempts and the successful binary's hash/metadata are retained in the archive.

## Independent reviews and remaining scope

Separate code/standards and acceptance/security reviews approved the bounded
five-file source snapshot with zero blocking findings. The acceptance review
independently reconstructed the red patch, matched the source hashes and
verified the full-checkout build equivalence and binary metadata. Reviewers
inspected evidence and did not rerun product suites concurrently.

The current handoff, UX-047 status note and generated traceability inventory
now reflect this continuation. Older dated catalogue records keep their
original results and scope, explicitly marked as historical. Source citations
are not execution claims or whole-requirement acceptance.

This slice introduces no language selection, translation scope, new policy or
release approval. Continue UX-047 from current integrated source and a bounded
gap investigation. Do not treat historical missing-flow lists as a current
inventory or this correction as completion of the full requirement.
