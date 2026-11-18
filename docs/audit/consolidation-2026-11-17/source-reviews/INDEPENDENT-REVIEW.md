# Independent adversarial review — focused consolidation fixes

Reviewer: independent agent G. Review date: 2026-11-18 UTC.

Source repository: `/workspace/scratch/9ceb45b04668/circula-integration`.
Review used exact objects, not the moving root checkout. Review probes ran in a separate detached worktree and temporary fixtures. No production source or remote state was changed by this reviewer.

## Verdicts

| Exact local commit | Scope | Review verdict |
|---|---|---|
| `cdbfefc836eff50788712e92a8bca540ef67e1a5` | Original NEW-01 backup staging and exclusive file copies | G-02 found; superseded by corrected head below |
| `eb17b6c28f5ff25d3ae4942c4500d233fbea1f93` | Corrected backup staging with active-work leases | **No blocking findings**; G-02 resolved |
| `59e12f34a8b90ea02b2ea5f4dfaad82c235db5a0` | Backup lint follow-up | **No blocking findings**; annotations remain local and justified |
| `c357ae3d6c4f7e513c08284e8378e4edf2ff695d` | Remaining backup cleanup lint annotations | **No blocking findings**; exact-version full-repository lint is recorded as clean |
| `66a0d4a94e3b2650a444037c7e6874d172d7a11d` | E-02 SQLite schema trust | **No blocking findings**, subject to aggregate and required remote CI gates |
| `4c26a7ed443d47e339d3bdac7b8f17d3b4e6fe58` | Administrator grant notification pagination | **No blocking findings**, subject to aggregate and required remote CI gates |
| `ef8c3a609f74fd9b7e2a0b1f7452b8f078b35009` | Original release threshold decision parser | G-01 found; superseded by corrected head below |
| `284d756a28d2b0a0e453da650446cd5b844c39db` | Corrected release threshold decision parser | **No blocking findings**; G-01 resolved |
| `24b5a9e6c052e961725ac8d8953f26a8cc3ee9bb` including parent `f2954c80181c958818b11d9e1e7a9f19d83dd881` | Factual documentation reconciliation | **No blocking findings** |
| `9193b320701690182970ed33757aac4031af9895` | Deterministic erasure privacy test fixture | **No blocking findings** |
| `4013b4d3cdd0eca389eb3a2dd1ee8d2244822acc` | Initial browser clock-flake repair | Later-page false pass found; superseded by corrected head below |
| `80ef09389238189b2b738f7805844aa9d913b5af` | Browser clock repair with preserved cross-page content guards | **No blocking findings** |

The integration lead recorded these publication mappings. The local trees below were also independently checked by this reviewer; remote identity checks are recorded by the lead:

| Pull request | Published head | Reviewed local head | Tree |
|---|---|---|---|
| PR71 | `1a0202001e422a3aefb09139f1237fc0b19b5747` | `66a0d4a94e3b2650a444037c7e6874d172d7a11d` | `450c3a47480e1da2b71e5b20a42bcfc2c7bd3388` |
| PR72, corrected | `946089539c57469241ab294e16377086e0add6b4` | `eb17b6c28f5ff25d3ae4942c4500d233fbea1f93` | `96a1357cef28b30e8be512c3f6c07e3d365fe6b7` |

The source verdicts apply to those published trees while the recorded identities remain true. None substitutes for current-head CI, remote head refresh, or merge conflict review.

## G-01 — P2: Missing owner-decision rows produce a threshold PASS

Affected commit: `ef8c3a609f74fd9b7e2a0b1f7452b8f078b35009`.

Location: `scripts/check_thresholds.py`, lines 43–44, 53–54 and 65–67. The parser ignores any non-pipe line, grants the absolute exemption based only on value/source/decision, and requires only a nonzero total row count.

Reproduction starts from the actual `docs/qa/THRESHOLDS.md` at this commit. Delete only its seven table rows whose decision is `UNCONFIRMED`. The three absolute rows, the `Status: PROPOSED` statement, the assertion that the owner has not confirmed DEC-023, and the instructional footer all remain. `check_thresholds` returns an empty error list. A full isolated `scripts/release-gate.sh` fixture records `| thresholds | PASS | confirmed decision rows |` and returns zero when unrelated automatic checks are explicitly skipped. Converting the same seven rows to tab-separated prose produces the same false pass. A single invented threshold row with `0 (absolute)`, `QA-040`, and `n/a (absolute)` also passes.

The repository's current unmodified proposed table correctly fails. This finding is about accidental row deletion/malformed conversion causing the new gate to certify a table with no owner decisions. It is not a claim that DEC-023 has been approved. The previous literal marker check would still reject these real-document deletion/conversion fixtures because their prose retains `UNCONFIRMED`.

Evidence: `release-parser-probe.py` and `release-parser-probe.json` in this directory. Four fixtures exercise both the imported function and the full shell entry point. All non-threshold checks are visibly `SKIPPED` or `MANUAL` in the isolated gate fixture; no actual release qualification is claimed.

Recommended correction: require the existing ten core threshold identities exactly once; only the existing three absolute identities may use the precise absolute exemption. Continue validating added rows by the dated DEC-023 rule. This protects completeness without choosing or changing any numeric policy. Add parser and full-gate regressions for a missing required row, all owner rows removed, rows converted to non-table prose, and an invented absolute exemption. Preserve the real document's unconfirmed status.

### G-01 follow-up — resolved at `284d756a28d2b0a0e453da650446cd5b844c39db`

The follow-up implements the existing ten required core identities, rejects duplicate identities and restricts the absolute exemption to the existing three names with the existing value/source/decision tuple. Additional rows still require valid dated DEC-023 references. It does not change real threshold values, the decision register, owner status, manual-gate classifications or skip behavior.

Independent re-review read the complete follow-up diff and ran all 26 Python tooling tests, which passed in 2.289s. The original real-document deletion, prose-conversion and invented-absolute probes were repeated at the exact corrected head. All now produce errors and full-shell `thresholds FAIL`/exit 1. The current unmodified `PROPOSED` document remains blocked. The valid synthetic confirmation fixture continues to pass the threshold classification while preserving visible manual/skipped classifications.

Evidence: `release-checker-corrected-tests.log` and `release-parser-corrected-probe.json`. **No remaining blocking finding on the corrected parser head.** Current-head CI and aggregate qualification remain the integration lead's gates.

## G-02 — P2: A second backup can sweep an active plaintext backup

Affected commit: `cdbfefc836eff50788712e92a8bca540ef67e1a5`.

Location: new unconditional `sweepStaleTemp(filepath.Dir(dest))` at `cmd/circula/snapshot.go:243`, interacting with the existing age-only deletion at lines 447–464. That helper treats a prefixed directory older than one hour as abandoned. A directory's mtime is not a liveness signal, and neither the manual backup call nor the scheduled backup call imposes a maximum execution time below one hour.

The deterministic probe starts a normal plaintext snapshot. During its verification callback, it sets that active work directory's mtime two hours in the past, representing a long-running backup, then starts a second plaintext snapshot into the same parent directory. The second backup succeeds. The first backup's verification file and snapshot have been removed; the first backup fails with `ENOENT`. No source database damage was observed or claimed. This is a backup availability regression and an ownership error in cleanup.

The same probe passes at parent `163fe1596bd7cfb8c2a22891629fc4459e6b639c` and fails at the reviewed commit. Encrypted snapshots already used the flawed age heuristic; this change extends it to all plaintext snapshots, which also now occupy prefixed work directories. The code comment that age filtering means a running backup is never disturbed is inaccurate.

Evidence:

- `backup-active-sweep-test.go`: exact deterministic Go probe; copied from the reviewer's detached worktree.
- `backup-active-sweep-baseline.log`: parent passes (`0.431s`).
- `backup-active-sweep.log`: reviewed commit fails (`0.434s`) with both missing verification file and missing snapshot.

Command: `go test -count=1 -run '^TestAuditGActiveBackupSurvivesStaleSweep$' ./cmd/circula`, using Go 1.26.11 at `/workspace/scratch/b4e4b2242853/toolchain/go/bin/go`.

Recommended correction: make stale cleanup require proof that the work is not active, such as a per-work-directory lock held for its entire lifetime and a nonblocking lock attempt by the sweeper. Failures to prove non-activity must skip cleanup. Existing native lock helpers may be reusable, but Windows close/delete ordering and old directories without a lease need explicit treatment. Retain cleanup of abandoned work and the successful/failing snapshot cleanup tests. Do not impose an unapproved one-hour backup cutoff merely to justify the heuristic.

### G-02 follow-up — resolved at `eb17b6c28f5ff25d3ae4942c4500d233fbea1f93`

The corrected commit adds `backup_work.go` and uses the same native nonblocking lock for snapshot, restore verification and encrypted pre-migration work directories. The lock is held across the entire operation. Stale cleanup must acquire the same lock, rejects symlink/nonregular lease paths, and checks the directory identity before and after acquisition. Failed acquisition preserves the candidate. Data cleanup happens while the lease is held; the handle is then released before lease-file and empty-directory removal, accommodating Windows sharing semantics.

The source review covered the full follow-up diff, all three producers, Unix and Windows lock implementations, process-crash release, legacy orphan behavior, and cleanup order. The documented operator-controlled parent-directory boundary remains applicable. Older binaries' directories cannot retroactively gain leases; the audit note explicitly preserves that limitation rather than claiming protection for them.

Independent verification at this exact commit passed under the race detector in 3.800s: the original deterministic G-02 regression, a real child process holding an aged directory, reclamation after force-killing that child, directory/lease symlink rejection, plaintext stale/fresh/unrelated preservation and the existing stale/fresh cleanup case. Evidence: `backup-lease-corrected-review-race.log`. `git diff --check` passed. **No remaining blocking finding on the corrected backup head.** Aggregate verification and native Windows/macOS runtime CI remain integration gates.

### Backup lint follow-up — reviewed at `59e12f34a8b90ea02b2ea5f4dfaad82c235db5a0`

The exact follow-up contains one checked deferred test-database close, three sink-local `#nosec G703` explanations, and the audit note documenting their provenance. The original provider log reported exactly that unchecked close and those three filesystem sinks. No runtime path, cleanup, lease, scanner configuration or global exclusion changed.

Independent caller tracing confirms that the two removals in `backup_work.go` receive private `MkdirTemp` directories or prefix-selected, identity-checked and leased backup-directory entries, with a constant lease filename. `secureRemove` receives those work entries, exclusively created decrypt outputs or restore staging derived from operator database configuration. Neither HTTP/member input nor a backup manifest chooses those cleanup destinations. `Lstat` itself does not follow symlinks; the annotation does not claim race resistance for later operations against a hostile same-identity process or replaceable parent. The manual-parent and filesystem ownership boundaries remain explicit.

**No blocking findings.** `git diff --check` passes. Agent D's five targeted race tests and vet are recorded by the integration lead; this reviewer did not duplicate those tests for the annotation/test-cleanup-only diff. The actual golangci-lint 2.15 and current-head CI results remain required before merge and were not yet claimed successful at this review.

### Remaining backup lint sinks — reviewed at `c357ae3d6c4f7e513c08284e8378e4edf2ff695d`

The next provider run exposed the two final `os.Remove` sinks in `secureRemove` and `secureRemoveAll`, beyond the first run's capped diagnostics. This exact follow-up, tree `399e1e7854e2342bc851b64f6a320b2211dc2edc`, adds only two further local G703 provenance comments and updates the audit note. The already-reviewed operator/owned-work provenance and threat boundary apply. `Remove` unlinks a final symlink itself and removes directories only when empty; no stronger race-defense claim is added.

**No blocking findings.** No runtime code, scanner configuration or test gate changed, and diff whitespace checks pass. Agent D ran the CI-pinned golangci-lint 2.15.0 across the complete repository with `--max-same-issues=0 --max-issues-per-linter=0 ./...`; the preserved `audit-d/backup-golangci-final.log` reports `0 issues.` These flags remove diagnostic-output limits, not checks. The integration lead records exit 0. Required remote CI still qualifies the eventual integrated head.

## Other backup findings

The focused hardening is otherwise sound within its documented operator-controlled filesystem boundary:

- `copyFileSync` uses exclusive creation and registers cleanup only after it owns a new file. An existing regular destination or symlink is refused without cleanup; failed copies remove their newly created output.
- Plaintext, verification and ciphertext stages are inside a private unpredictable directory; both success and error cleanup target that invocation's own directory.
- `verifyCopy` stops deleting a predictable external `<source>.verify` name.
- The restore caller no longer deletes an external competitor after `copyFileSync` refuses exclusive creation. Existing deliberate removal of restore's own old staging path is unchanged.
- Hash comparison, manifest authentication, encryption verification and final backup formats are preserved. No ledger or database migration change appears in this diff.

A hostile process with the same OS identity, replacement of an operator-controlled parent directory, secure-erasure guarantees on copy-on-write media, and the pre-existing final-destination publication race are not solved by this patch and are not represented as solved in this review.

## E-02 SQLite schema trust

`internal/store/sqlite.go:106–117` adds `trusted_schema(0)` to the same DSN used to open every normal physical connection. `internal/store/inspect.go:82–89` does the same for raw inspection/backup connections. Setting it in the DSN avoids the common error of changing one pooled connection only. Existing pool limits are eight and two respectively.

The tests hold all pool connections open simultaneously, then check the setting on each, and exercise the production `circula_fold` function directly and through a stored view. Direct calls remain allowed; the view fails with SQLite's unsafe-use refusal. All production uses of this function are direct search queries in `member_page.go` and `sqlite_listing.go`; the canonical migrations do not invoke it in schema expressions. No migration, ledger semantics, policy, or function registration was added.

Independent verification: `go test -race -count=1 -run '^TestSQLite(DistrustsSchemaOnEveryConnection|RefusesSchemaFunctionCalls)$' ./internal/store` passed at exact commit `66a0d4a...` in `1.926s`; see `schema-review-race.log`. Agent C's audit document records failing-before results on both pools and the behavioral view test. Aggregate results belong to the root's qualification record.

## Administrator notification pagination

`internal/service/membership.go:334–358` uses the existing `ListMembers` cursor within the caller's write transaction. `internal/store/sqlite_member.go:82–96` enforces network equality, active-status filtering, `id > AfterID`, ascending ID order and the requested limit. Page size 500 matches `limitOf`'s upper bound. The cursor advances on the last row, so nonempty full pages strictly advance and an exact multiple of 500 terminates on a final empty page.

Each row remains filtered to active administrators in the specified network except the grantee. There is one membership per person per network, no offset-based duplication, and notifications retain the same commit/rollback boundary as role changes and audit events. No approval, governance or recipient policy changes were introduced.

The regression checks exactly one notice for an early and a later-page administrator, and zero for an ordinary member, grantee, inactive administrator and foreign-network administrator. It uses deltas around the actual role grant so setup notices do not inflate the assertions. Agent C records the failing-before case and repeated after-fix/race success. Independent `go test -race -count=1 -run '^Test(AdminGrantNoticesBeyondFirstMemberPage|A3_01_AdminGrantNotifiesOtherAdmins)$' ./internal/service` passed at exact commit `4c26a7e...` in `2.683s`; see `admin-notice-review-race.log`.

## Scope and limitations

Review focused on the listed diffs and their production callers. It did not re-audit all historical branches, validate native macOS/Windows behavior locally, approve a release, or resolve owner policy. A3-F02 / DEC-014 remains unresolved. No future module integration, deployment, tag, force push, branch deletion, or remote mutation was performed.

## Additional documentation review

At the lead's request, reviewed `ef54238ee9ebc2184919b6a8ec30c3398df55ed5..24b5a9e6c052e961725ac8d8953f26a8cc3ee9bb`, comprising documentation commits `f2954c80181c958818b11d9e1e7a9f19d83dd881` and `24b5a9e6c052e961725ac8d8953f26a8cc3ee9bb`. All nine changed files were inspected. **No blocking findings.**

Source corroboration:

- `throttle.go`, `throttle_config.go` and `mfa.go` match the fixed finite lock, configurable bounds, half-allowance restoration, decay, separate MFA counters, and family clearing.
- `recovery.go` checks codes before Argon2 hashing, restricts ordinary administrator resets to active non-admin members with no non-departed membership elsewhere, and implements System-only operator reset without `RequireBreakGlass`.
- `erasure.go` limits person-level erasure to self/System and no other non-departed membership. The MFA and both recovery-code deletions are present; the corresponding SQL helpers issue the stated deletes. `TestErasureRemovesMFAAndNotices` does not seed both code tables, exactly as the new note discloses.
- `principal.go` and role-change callers implement the stated partial `requireLasting` restriction. `breakglass.go` applies the hard TTL and emits network/platform audit records. The documentation explicitly leaves remaining powers and approval/notice policy unresolved.
- `config.go` defaults MFA enforcement off and log retention to 2160 hours; `log_retention.go` implements the described pruning command.

No executable source, schema, configuration defaults or normative SEC-006 requirements changed. DEC-007, DEC-014 and DEC-019 remain open; A3-F02 remains unresolved. Dated implementation descriptions do not claim owner approval. `git diff --check` passed. The author's 13 plus 15 targeted existing tests are documented; this reviewer checked their relevant source and did not duplicate their execution for a documentation-only diff.

## Additional erasure fixture review

At the lead's request, reviewed exact commit `9193b320701690182970ed33757aac4031af9895`, based on PR69 merge `ef54238ee9ebc2184919b6a8ec30c3398df55ed5`. The two-file diff changes `internal/service/erasure_test.go` and adds `docs/audit/ERASURE-FIXTURE-STABILITY-2026-11-18.md`. **No blocking findings.**

The original global SQLite `LIKE '%Ana%'` scan could match a random unrelated Argon2 salt/digest. The author reproduced that exact failure using a valid Ben credential with the selected salt `AnaAAAAAAAE`, while proving that the credential still authenticated. The corrected fixture uses complete authored markers containing spaces or non-base64 punctuation, retains scans of every database column, and asserts all markers are present before erasure and absent afterward. Exports are checked against the same markers.

The test additionally verifies that the target hash becomes `!erased`, the original target hash disappears from the entire database, Ben's and the administrator's hashes remain byte-for-byte unchanged, and the deliberately colliding Ben credential still authenticates. Existing ledger, account identity, session, authorization, audit and irreversible-erasure checks remain. No runtime source, database schema or erasure policy changes.

Reviewed the author's before/after evidence, 20 consecutive repaired-test runs, 11 related tests repeated normally and under race, and ten mutation cases. The mutation harness requires the expected real assertion failure for each retained field/credential defect and restores the exact original test checksum afterward. It covers name, login, memo, name-only memo fragment, shared-text-only memo fragment, listing title/description, dispute statement, target password hash and unrelated password hash.

Independent `go test -race -count=1 -run '^TestAnonymiseMemberKeepsLedger$' ./internal/service` passed at the exact committed head in 3.969s; see `erasure-fixture-review-race.log`. `git diff --check` passed. The test-only fix is approved for integration subject to aggregate/current-head CI gates.

## Additional browser clock repair review

The provider browser failure compared separately generated pages at 06:33 and 06:34. The original test at `9c1b38d566305e0b577ff0555cdda6498eebed18` compared full visible text across those requests. Initial repair `4013b4d3cdd0eca389eb3a2dd1ee8d2244822acc` correctly compared each page before and after its colour mode, but dropped the independent cross-page content guard.

Independent review reproduced an introduced false pass: when the initial page had the expected trade but later pages had only a sign-in message and empty semantic collections, the original test failed and the initial repair passed all three modes. `greyscale-later-page-probe.json` pins both Git inputs and includes the passing original control. An initial exploratory run accidentally read moving HEAD after the first repair was committed; it was not evidence of this finding and was replaced with the explicit-commit reproduction.

Corrected head `80ef09389238189b2b738f7805844aa9d913b5af`, tree `04b19d714cb1e89c1d3827469fc285a4cf1b6878`, restores cross-request comparison while substituting only the visible string from `#figure-freshness time` inside its dedicated paragraph. The time node and nonempty value are required. All other main text, including the remaining freshness statement and nonfinancial body words, stays in that comparison. Full same-page before/after equality still includes the exact timestamp. Existing signs, spoken balances, statuses, visibility, contrast, fresh-page modes and expected-content checks remain.

Independent `greyscale-final-review-probe.cjs` executes the actual assertion body and `readScreen` callback using minimal controlled DOM objects. It confirms that a legitimate minute rollover passes all three modes and that missing/blank time, missing freshness paragraph, changed footer wording, omitted nonfinancial text, amount/time changes during mode application and inadequate contrast all fail. These nine scenarios have the expected outcomes in `greyscale-final-review-probe.json`. The tested source SHA-256 is `d0de3ff70b26c2d2051b18589abec83784a15af100992defdf1ea7f0d5a821a7`, matching committed Git blob `5494214521b0110fd37ad8f4ac97fb12e966f641`.

**No blocking findings at the corrected head.** JavaScript syntax and diff whitespace checks pass. These probes establish assertion wiring and normalization, not actual browser rendering, emulation or axe qualification. Local Chromium's socket restriction is not a passing browser run; the real required browser CI remains a merge gate. The repair changes no production code, clocks, retries or timeouts.

After the revised browser repair passes its required gates, updating the security PRs with accepted current main through normal forward merge commits is appropriate. Preserve the security changes and run required gates at each resulting exact head/tree. Prior-head checks cannot stand in for the refreshed merge candidates; no history rewrite or force push is needed.

## Refreshed security integration candidates

Independently reviewed the ordinary forward merge candidates against accepted main `ffa2758ffc2fe29133a46b1d8d3f0a6138c404e4`. **No blocking integration findings.** Both working trees were clean. Complete tracked-tree comparisons, including every file mode, type and object ID, show exactly the approved security files plus the two approved browser-repair files; all other accepted-main entries are unchanged.

| Candidate | Local reviewed commit | Published commit independently checked | Exact tree | Diff against accepted main |
| --- | --- | --- | --- | --- |
| PR #71 | `813faddb47c0c0c080f8c9479b8ef7105cf18746` | `b1cd8f38450ae966cfea99d59a131175d7b798dd` | `756fed2f703fd9ec882b0ed3d92ef3dab99b2ffc` | Four E-02 files plus two browser repair files |
| PR #72 | `2f9da51c6bde79b1031ebd2ab0701f649ff39a3e` | `faed4906f5419f573181bd7cf3ffdf0bdfb9992b` | `b10a55a019ab37b149d33649fba1f6615acfeac4` | Nine backup hardening files plus two browser repair files |

Each expected tree was constructed from the complete accepted-main tree and the exact approved blobs: E-02 `66a0d4a94e3b2650a444037c7e6874d172d7a11d`, backup final lint head `c357ae3d6c4f7e513c08284e8378e4edf2ff695d`, and browser repair `80ef09389238189b2b738f7805844aa9d913b5af`. There were no unexpected entries, deletions, modes or content changes. This confirms that the lease fix, all five reviewed local G703 explanations, schema-trust connection coverage and the corrected cross-page UI guard are retained without replacing newer accepted work.

Local ancestry checks confirm accepted main, each prior security head and published browser head `2f6b3e4f1b70d8c787c4de886cf085f5a3295002` are parents/ancestors of their candidates. Independent GitHub commit GETs confirm the published commits have the same respective trees and forward-merge parents. PR #72's published security parent `6dd208660969f02100b2b5bbed04dcb6780c0b99` has tree `399e1e7854e2342bc851b64f6a320b2211dc2edc`, identical to the approved local `c357ae3d...`; its different commit ID does not conceal a source difference.

Evidence: `integration-refresh-review.json` records all changed paths, approved source/blob comparisons and full-tree results; `integration-refresh-remote.json` records the independently retrieved published tree/parent metadata. This is an integration-content approval, not a passing CI claim. PR #77 must pass and merge, and the refreshed PR #71/#72 heads must pass their required and native qualification gates before main integration. The lead's combined-tree aggregate run remains separate evidence; no redundant test rerun was required merely to compare byte-identical reviewed blobs.

## Concurrent PR #79 authentication catalogue review

Reviewed the complete 19-file diff at exact head `dd3e82e27ddb3d1cbf03460e6a5fc879f16691db`, tree `e6d94f9fbc59016bfe7350ef6d9a8334031ef725`, based on `ffa2758ffc2fe29133a46b1d8d3f0a6138c404e4`. **No blocking findings.** This is valuable core message externalization; no future-module, ledger, migration, dependency, workflow or owner-policy change is included.

The four handler diffs substitute catalogue strings without changing service calls, control-flow branches, HTTP status codes, authentication/authorization checks, throttle accounting, session/CSRF rotation, redirects or safe-next validation. Unknown/wrong login failures still share the same message and field error. Unknown/wrong/used recovery codes retain their shared refusal. MFA challenge refusal and rate-limit distinctions remain as before. Security notice kinds retain their mapping and the existing generic fallback for unrecognized kinds. The package initializes its English catalogue before these package-level notice strings; no request-dependent locale is introduced.

All nine templates preserve form methods/actions, CSRF includes, input names and IDs, hidden challenge/next fields, password types, required markers, autocomplete, semantic markup and link destinations. An independent complete-source expansion check replaces the new static message calls with their exact English values and restores only the three reviewed plural/count expressions: every resulting template is byte-identical to its previous version. Evidence: `pr79-template-equivalence.json`. This structural comparison does not claim browser qualification.

Escaping remains contextual `html/template` escaping: `msg` still maps to `messages.Text`, which returns an ordinary string, and no new trusted HTML/URL conversion is present. Login, challenge, return path, recovery code, enrolment secret/URI and notice-detail values retain their original escaped interpolation positions. Moving the static apostrophe in `network's` through the helper changes its HTML encoding to `&#39;` while preserving visible text. The new rendering tests exercise hostile values in those positions; real endpoint tests also retain CSRF and authentication refusal coverage.

Independent catalogue inspection found exactly 121 additions, each referenced in production handlers/templates, no duplicate keys, no missing referenced keys, no empty `other` values and no changes/removals of existing entries. The two new plural entries use existing English `one`/`other` selection; counted security-notice formatting uses the existing integer formatter. Unknown catalogue keys still fail visibly rather than leaking raw keys, and all new fixed lookups resolve. Evidence: `pr79-catalogue-completeness.json`.

Independent focused `go test -race -count=1` passed ten web tests in 5.870s and three catalogue tests in 1.019s at this exact head. These include the four new render tests, registration uniformity, negative MFA/recovery routes, MFA challenge CSRF/GET behavior, general CSRF rejection, login requirements for security pages, authentication plural/refusal formatting, complete-catalogue formatting and unknown-key behavior. Exact command and exit 0 are preserved in `pr79-review-race-command.json`; output is in `pr79-review-race.log`. Diff whitespace checks passed. Full combined-tree Go verification and required current-head/browser CI remain the integration lead's gates; these focused tests do not replace them.

UX-047 correctly remains partial. Sentences split around existing links/strong markup, the raw emphasized recovery count and the explicitly unmigrated surfaces still limit full language/word-order flexibility. The documentation discloses those limitations and approves no locale or translation. These are follow-up scope, not a reason to discard the reviewed English catalogue extraction. Approval is pinned to this head and subject to required integration qualification and preservation of concurrent owners' work.

## Concurrent AUD4-D02 backup-path review

Reviewed `3dcb3f8bfd07e0a8382d06762d7e9ba32c0aacbd`, tree `fdaa1ab68cd931a361bb4c69d4fe99215c7926bb`. **Changes requested at this head.** `AUD4-D02-REVIEW.md` records two findings: G-03 reproduces deletion outside a replaced leased-work directory while the new parent check accepts sticky directories owned by another user; G-04 identifies unconditional runbook/security claims for ownership, no-follow and hardlink protections that are implemented only on Unix or depend on a trusted parent boundary. The real producer cleanup still begins with an unchecked `ReadDir(work)`; the newly added `secureRemoveAll` guard does not cover it.

The deterministic Go-overlay probe fails with the canary deleted (`backup-path-lease-substitution.log`); it simulates the name replacement and is explicitly not a two-UID execution test. Twelve focused existing/new backup tests independently pass under race in 8.030s (`backup-path-focused-race.log`), preserving private staging, live leases, crash reclamation and exclusive-copy behavior. Agent C independently handles D01 and identified additional Windows-invalid `?` test paths. No implementation or remote mutation was performed by this reviewer. Corrected code, accurately bounded documentation and native/current-head qualification are required before approval.

Latest PR #81 follow-up reviewed at `5776be13613ddbf172bc73c97a9fd83e26c9f989`, tree `7de852707f23b06be231ce0cc9f69ad0c675c99c`: only three test files changed, adding Windows invalid-filename guards and direct URI-encoding coverage. That test portability issue is addressed in source, pending native results. D02 runtime source and documentation remain unchanged, and the G-03 probe fails again in 0.004s (`backup-path-lease-substitution-5776be1.log`, matching command JSON). G-03 and G-04 remain **changes requested** at this latest exact head. The integration lead was notified before submitting a remote review; no reviewer remote mutation occurred.
