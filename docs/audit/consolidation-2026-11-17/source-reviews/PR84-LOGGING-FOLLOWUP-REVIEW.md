# PR84: independent clock-anomaly logging follow-up

Review completed 2026-11-18 09:15 UTC. Reviewer: core_recovery (C), using Caveman investigate-first / review / verify-and-stop. Read-only tracked source; no remote mutation, new test/probe code, or broad suite. Only the two new supplied logging regression functions were executed, sequentially before and after the production change, with bounded resources and an explicit workspace Go cache.

## Verdict

**The remaining A3-F10 logging portion is implemented and independently verified for the future-dated-backup scenario. No correctness blocker was found in this logging delta.** The new behavior satisfies the existing OPS-033 requirement to log the observed clock anomaly without changing cadence, retention counts, policy or permissions. The wider OPS-033 requirement correctly remains partial; this is not a claim of general host-clock detection or complete failure-recovery qualification.

PR84 is still an externally owned draft. Required exact-integration CI and readiness controls remain the integration coordinator's gate. This review does not approve a release or substitute a narrow test result for the aggregate suite.

| Item | Exact value |
|---|---|
| PR | https://github.com/djshellshoxxx/circula/pull/84 |
| Branch | `claude/sharp-archimedes-tu76rh` |
| Reviewed head | `eb4bb08b15d41295aa2c4b9f93040124ce4d66e9` |
| Tree | `f6d1abf5be0935f5bca5c7f1e3cda56918565693` |
| Metadata base / merge base against cdc8112 | `cdc8112cf80ca51fdd220cd2f86c5180180cbe2a` |
| Previous independently checked head | `7f98b3db8d032fc107599a3da4a6e1747c3e00aa` |
| Production logging commit | `4bbc871` |
| New source snapshot | `/workspace/scratch/9ceb45b04668/audit-c-pr84/fixed-eb4bb08` |
| Final metadata | Open, draft, not merged; same head at 09:15:40 UTC |

GitHub's `mergeable` field changed from true at the initial refresh to false at the final refresh, with the same reported head/base. **This is not a confirmed source conflict.** Root separately reports that `git merge-tree` of current main `b4398074` and this head is clean, producing tree `717b32995b82772e3d6ff644297b7d05a74464b5`. No repair is justified by the observed Git result; refresh the remote status/gates before merge. The reason for the differing metadata was not established here.

Metadata snapshots are `PR84-EB4BB08-METADATA.json` and `PR84-EB4BB08-FINAL-METADATA.json`.

## Changes after the prior verified head

The delta is six files, 187 insertions / 22 deletions. Production code changes are confined to 14 added lines in `cmd/circula/serve.go:untilNextBackupAt`.

- Count entries with a parsed filename time strictly after the supplied current time.
- If any exist, emit one `Warn` for the entire scheduling evaluation. Fields are the count, newest matching backup's basename, filename time and current time. Both times are converted to UTC.
- Continue through the existing scheduling loop unchanged. No backup is written, removed or rescheduled by the logging block itself.
- Add the two focused tests described below, and supply a discard logger to one existing direct-call test fixture that previously needed none.
- Update monitoring documentation, the OPS-018 status note, audit evidence and generated traceability. OPS-033 stays partial.

`listBackups` sorts parsed backup times newest-first, so when at least one entry is in the future, `entries[0]` really is a future entry and the newest one. Its parser accepts the existing plain/encrypted and legacy timestamp naming forms. The log uses `filepath.Base`, not the full directory, and does not expose backup contents or credentials. Production supplies the existing logx-wrapped `newLogger`, which escapes/redacts output and is non-nil on the production call path. No new setting, tolerance, approval or retention policy is introduced.

The previous catalogue additions, catalogue regression, retention implementation and pruning regression file are byte-identical to the earlier verified head. `jobs_regress_test.go` adds the logging tests and changes the existing future-scheduling fixture's logger; the previously tested retention function body is unchanged. Earlier fail-before/pass-after records remain precisely attributed to 7f98b3d, not relabeled as an aggregate pass for eb4bb08.

## Independent regression evidence

Existing supplied tests only:

1. `TestJobUntilNextBackupLogsFutureDatedBackups`: six scenarios (no backups, recent, exactly now, future only, one nanosecond ahead, several future files including encrypted and legacy names). Runs each scheduling evaluation twice; checks unchanged wait duration, zero warnings for non-future cases, one warning per evaluation otherwise, exact count/newest basename, UTC values and production JSON logger field shape.
2. `TestJobBackupLoopLogsFutureDatedBackupEachEvaluation`: runs the actual background backup loop, verifies the future file does not delay the first backup, and checks exactly one warning at startup and one after that backup. The existing log sink is mutex-protected.

### Before: exact previous production file via Go overlay

The new tests ran on the exact eb4bb08 snapshot with only `cmd/circula/serve.go` supplied from 7f98b3d through an external Go overlay. No tracked file was edited. The before file is a literal prior production file, not new probe code.

- Prior `serve.go` blob: `e8ff373b13bf0cc011829a35dca21b6700a2bf8f`.
- New `serve.go` blob: `4791177040987b0e3cffd8de8feef47dab8ada0d`.
- New supplied test-file blob: `e4db2ea073cd8b6fce613f7c4770b8a080ea7a63`.
- Overlay: `PR84-logging-before-overlay.json`; prior file: `PR84-serve-7f98b3d.go`.

**Expected failure, exit 1, package 10.589 s.** The three future-file scenarios each fail because zero warnings were emitted instead of one. The three non-future cases pass, and the production-loop test reaches the first backup but times out waiting for the two absent warnings. Log: `PR84-logging-before.log`.

### After: exact reviewed head, race enabled

**Pass, exit 0, package 3.622 s; no data race reported.** All six scenarios and the background-loop regression pass on the unmodified eb4bb08 snapshot. Log: `PR84-logging-after-race.log`.

Both commands ran from `/workspace/scratch/9ceb45b04668/audit-c-pr84/fixed-eb4bb08`:

```sh
GOMAXPROCS=2 GOGC=50 GOCACHE=/workspace/scratch/9ceb45b04668/audit-c-pr84/go-build-cache /workspace/scratch/b4e4b2242853/toolchain/go/bin/go test -p=1 -count=1 -timeout=3m -overlay=/workspace/scratch/9ceb45b04668/audit-c/PR84-logging-before-overlay.json -run '^(TestJobUntilNextBackupLogsFutureDatedBackups|TestJobBackupLoopLogsFutureDatedBackupEachEvaluation)$' -v ./cmd/circula

GOMAXPROCS=2 GOGC=50 CGO_ENABLED=1 GOCACHE=/workspace/scratch/9ceb45b04668/audit-c-pr84/go-build-cache /workspace/scratch/b4e4b2242853/toolchain/go/bin/go test -race -p=1 -count=1 -timeout=3m -run '^(TestJobUntilNextBackupLogsFutureDatedBackups|TestJobBackupLoopLogsFutureDatedBackupEachEvaluation)$' -v ./cmd/circula
```

The six recorded source-file hashes remained unchanged after execution. Exact blobs and hashes are in `PR84-logging-snapshot.json`. No process termination or cache-space failure occurred in this follow-up, and no Go process remains from this workstream.

## Readiness and scope limits

The earlier A2-08/retention verification remains in `PR84-REGRESSION-VERIFICATION.md`. A2-08's missing four catalogue rows and A3-F10's retention/logging changes remain branch-local until integration. The original A3-F10 report explicitly described the absent anomaly log; this follow-up closes that part of the original finding, rather than broadening its scope.

The branch also documents a pre-existing discrepancy: backup documentation says pruned filenames are logged, while the pruning job logs a count. The new anomaly warning does not change that pruning message. This is separate from whether the future-date anomaly is now reported and is not introduced by the logging delta.

No broader Low/Info disposition ledger, owner-policy question or author-reported full-suite result was requalified here. A3-F02 / DEC-014 and other explicitly owner-held decisions remain unchanged. This source review and targeted race gate are suitable evidence for the integration coordinator; refresh the final head, current main, draft/readiness status and required CI before merging.

## Separate AUD4-F03 work discovered

Draft **PR87**, https://github.com/djshellshoxxx/circula/pull/87, now addresses dispute timelines. At discovery:

- Branch: `claude/aud4-f03-dispute-timelines`.
- Head: `1a3ef0fad836067b71857d961659459d976e7e15`.
- Metadata base: `cdc8112cf80ca51fdd220cd2f86c5180180cbe2a`.
- Status: open, draft.

The author describes using the policy in force when a dispute opened, with no migration, and acknowledges a same-millisecond policy-order ambiguity. Those are PR-body claims, not a source verdict from this workstream. Discovery metadata is preserved in `PR87-DISCOVERY-METADATA.json`. Root has assigned its independent review to agent B; no PR87 ownership, source edits, tests or remote mutations were undertaken here.
