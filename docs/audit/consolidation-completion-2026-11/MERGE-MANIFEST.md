# Merge manifest, 2026-11-18

Each merge was made only after these checks:

- `main` was re-fetched, and the PR head SHA matched `expectedHeadSha`.
- Every required CI job passed on that exact head.

Local qualification is specific to each row below. In particular, #77 has local diff review and CI evidence;
this manifest does not claim a separate local test run for that tree. Other rows identify the exact tree and
the local checks reported for it. These merge commits were observed by the parallel integration coordinator,
which did not issue those merge calls. Local-check entries here are author-reported; the Git merge/tree and
CI receipts are independently inspectable.

Merge method: merge commit, as in the repository's history.

| Order | PR | Head merged | CI run (exact head) | Merge commit | `main` tree after merge | Local qualification of that tree |
|---|---|---|---|---|---|---|
| 1 | [#77](https://github.com/djshellshoxxx/circula/pull/77) colour-mode parity test | `2f6b3e4f1b70d8c787c4de886cf085f5a3295002` | [44256413287](https://github.com/djshellshoxxx/circula/actions/runs/44256413287): 7/7 (cross-platform not requested) | `78c695f202e4b9440a4d78ac3b4b181aa81cabac` | — | Test-only change; diff reviewed |
| 2 | [#71](https://github.com/djshellshoxxx/circula/pull/71) E-02 `trusted_schema(0)` | `b1cd8f38450ae966cfea99d59a131175d7b798dd` | [44256822964](https://github.com/djshellshoxxx/circula/actions/runs/44256822964): 9/9, including native Windows and macOS | `28ed7166a14616163bf49ed248dcce0a01414850` | `756fed2f` (= PR head tree) | Merge tree identical to the tested head tree |
| 3 | [#72](https://github.com/djshellshoxxx/circula/pull/72) NEW-01 private backup staging | `faed4906f5419f573181bd7cf3ffdf0bdfb9992b` | [44256964805](https://github.com/djshellshoxxx/circula/actions/runs/44256964805): 9/9, including native Windows and macOS | `a62071d2c4e65027f4918c90d63e804b1deceb60` | `bddf8035` | Tested as main + #71 + #72 before merge. `go build`, `go vet`, `go test -count=1 ./...` (all packages), `-race` for `cmd/circula` and `internal/store`, golangci-lint v2.15.0: 0 issues |
| 4 | [#79](https://github.com/djshellshoxxx/circula/pull/79) UX-047 authentication catalogue | `dd3e82e27ddb3d1cbf03460e6a5fc879f16691db` | [44260832164](https://github.com/djshellshoxxx/circula/actions/runs/44260832164): 7/7 | `cdc8112cf80ca51fdd220cd2f86c5180180cbe2a` | `f41365f8` | Tested as main `a62071d` + #79 before merge. `go test` for `internal/web/...`, `internal/api/...`, `cmd/circula`; traceability check |
| 5 | [#81](https://github.com/djshellshoxxx/circula/pull/81) AUD4-D01/D02 backup path hardening | `09ef8a5aa25f11ee4d96f043e7d686e97598b7e3` | [44264463917](https://github.com/djshellshoxxx/circula/actions/runs/44264463917): 9/9, including native Windows and macOS | `b4398074ae4ca72559235ca50242cfcb7a934eec` | `1e2923ef` | Tested as main `cdc8112` + #81 before merge: build, vet, `cmd/circula`, `internal/store`, `internal/web` tests, traceability, hygiene |

`main` was `ffa2758ffc2fe29133a46b1d8d3f0a6138c404e4` at baseline. It is `b4398074ae4ca72559235ca50242cfcb7a934eec` after merge 5 (as of 2026-11-18T09:41Z). The final `main` is recorded in HANDOFF when this PR merges last.

## Open, not merged

| PR | State | Gate |
|---|---|---|
| [#78](https://github.com/djshellshoxxx/circula/pull/78) | Draft, docs only | This record; merges last |
| [#80](https://github.com/djshellshoxxx/circula/pull/80) | Draft, future module only | Stays out of `main`. Base-branch CI fails on stale core code (`cmd/circula/auditanchor.go: undefined: os`), not on this change |
| [#82](https://github.com/djshellshoxxx/circula/pull/82) | Open, AUD4-F06/F04 plus password-proof follow-up at `e127acc` | Current-head provider CI pending; independent review, focused regressions and local combined race/lint qualification at `17152dad` passed |
| [#83](https://github.com/djshellshoxxx/circula/pull/83) | Draft, AUD4-F02 | Owner decision; not to be merged without one |
| [#67](https://github.com/djshellshoxxx/circula/pull/67) | Draft, future module | Unchanged; #80 stacks on it |
| [#84](https://github.com/djshellshoxxx/circula/pull/84) | Open, ready for review; another worker; head `eb4bb08` | Current-head provider CI pending; included in the separately qualified local combined candidate `17152dad` |
| [#85](https://github.com/djshellshoxxx/circula/pull/85) | Draft, GOV-021 | Owner gates (see BLOCKERS) |
| [#86](https://github.com/djshellshoxxx/circula/pull/86) | Draft, SEC-026, stacked on #85 | Owner gates (see BLOCKERS) |
| [#87](https://github.com/djshellshoxxx/circula/pull/87) | Open, ready for review; AUD4-F03 (partial); head `679a301` | Current-head provider CI pending; local combined race/lint qualification at `17152dad` passed; timestamp residuals remain |

At 2026-11-18T10:07:00.462200Z, the single full local race invocation completed successfully on
`17152dadad91d4e1c994ea807e7d6e2497513aa8`, tree `1e47c9c30165ca27866887533586c90323e1630e`.
This is the unmerged #82/#84/#87 candidate; the latest recorded `main` remains `b4398074`. The provider
snapshot at 10:07:24Z still has all seven required jobs queued, zero assigned runners and zero executed
steps for each of the three PRs. Their local qualification does not substitute for current-head CI or a
merge receipt. See [TEST-EVIDENCE](TEST-EVIDENCE.md).
