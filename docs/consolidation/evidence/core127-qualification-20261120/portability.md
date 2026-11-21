# Native qualification portability repair — 2026-11-20

## Source and observed failures

Base: PR #127 at `21933a3c650690db5a8e1e44cc236d8d1b5cdb24`.
Repair source: `cf4148470ac2060e1b5628b541f00065b5edbb5d`.

[CI run 44712003434](https://github.com/djshellshoxxx/circula/actions/runs/44712003434)
completed with five successful and four failed jobs. This receipt covers only
the two native-platform failures and the proposed narrow repairs.

The [Windows job](https://github.com/djshellshoxxx/circula/actions/runs/44712003434/job/136205746896)
failed `TestMigrationsExpandThenContract`: the exact-byte review digests for
migrations 0023 and 0025 differed after checkout. Other listed test packages
passed. Both stored Git blobs use LF; no repository attributes previously fixed
the checkout representation. The local Git conversion control below reproduces
that discrepancy without changing either reviewed SQL source or expected digest.

The [macOS job](https://github.com/djshellshoxxx/circula/actions/runs/44712003434/job/136205747018)
exhausted the 25-minute service-package timeout. The running parent test was
19 seconds old and its current subtest was two seconds old. The captured stack
was runnable inside SQLite parsing during a newly created fixture's migration;
it did not show that subtest waiting for 25 minutes. Other observed native race
package durations were 483.246 seconds (command), 630.603 (API), 1000.287 (store),
and 649.294 (web). The full command started at 10:34:31 UTC and finished failed
at 11:13:04 UTC. The service deadline occurred at 11:09:31 UTC.

## Change

- `.gitattributes` pins only `/internal/store/migrations/*.sql` to
  `text eol=lf`. The byte-level review registry and SQL files remain unchanged.
- The macOS Go package limit increases from 25 to 45 minutes, and its job limit
  from 45 to 60 minutes. The observed scheduling and earlier-package work need
  headroom outside the service-package budget. These bounded allowances are
  a qualification budget, not a measured new full-suite completion time.
- The complete `./...` selection, `-race`, `-count=1`, `-p 2`, all assertions,
  native runner names, and Windows/Linux limits remain unchanged.

## Executed local verification

The actual Git checkout experiment used an isolated temporary repository with
the two original LF migration blobs and `core.autocrlf=true`. After committing
those blobs, it removed their worktree copies and ran:

```sh
git checkout-index --force --all
```

Without attributes, both worktree files contained CRLF and both SHA-256
comparisons failed against the existing review registry: **2/2 reproduced**.
After committing the proposed attributes, removing the worktree copies, and
running the same checkout command, both files contained LF and both original
digests matched: **2/2 passed**. A changed SQL source still failed the exact
digest comparison for each file. `git check-attr text eol -- <both paths>`
reported `text: set` and `eol: lf`; the temporary worktree was clean.

The existing Go review and scanner tests executed using Go 1.26.11,
`GOTOOLCHAIN=local`, from `internal/store`:

```sh
go test -count=1 -v expand_contract_test.go
```

Result: exit **0**, `TestMigrationsExpandThenContract` and
`TestContractScannerForms` both passed; **2 passed, 0 failed, 0 skipped**.
Reported package time: **0.009 seconds**. This focused command compiles the
existing standard-library-only test file; it does not execute the entire
SQLite store suite.

PyYAML 6.0.3 parsed the original and repaired workflows. Recursive comparison
found exactly two semantic changes: `jobs.test-cross.timeout-minutes` and
the macOS step's Go timeout. The Windows and Linux job definitions are equal.
`bash scripts/check-action-pins.sh` exited **0** with
`action pins and permissions: ok`. `git diff --check` exited **0**.

## Remaining qualification

The checkout experiment ran on Linux with actual Git CRLF conversion, not on
a native Windows host. The new Windows job must confirm the same migration
digests under the provider's checkout. The enlarged macOS race suite must
complete within the new bounds. Neither native result is claimed here.
Current-head combined CI and independent review remain required before merge.
