# NEW-01: private backup staging and exclusive copies

Reviewed and reproduced against PR #69 head
`163fe1596bd7cfb8c2a22891629fc4459e6b639c`. This focused change addresses the
predictable staging paths recorded in `docs/security/security-review.md`.

## Reproduction

The original `copyFileSync` opened its destination with `O_TRUNC`, so both an
existing regular file and the target of a planted symlink were overwritten.
The original unencrypted snapshot placed its verification copy directly beside
the destination at `<dest>.partial.verify`. Snapshot and verification cleanup
also deleted pre-existing predictable `.partial` and `.verify` names.

The three new regressions failed before the fix:

- `TestCopyFileSyncRefusesExistingDestination`: the copy succeeded and replaced
  the protected data, for both a regular destination and a symlink.
- `TestSnapshotUsesPrivateStaging`: the plaintext verification file was outside
  a private staging directory; both plaintext and encrypted paths modified
  pre-existing temporary names. Success and injected verification failure are
  covered separately.
- `TestVerifyCopyLeavesPredictablePathUntouched`: verification deleted the
  pre-existing `<source>.verify` file.

Review of the first fix identified two cleanup cases. Their regressions also
failed before the final correction:

- `TestPlainSnapshotSweepsOnlyStaleBackupWork`: a plaintext-only installation
  did not sweep old private backup directories after a crash.
- `TestCopyFileSyncRemovesOnlyItsFailedCopy`: a failed read left the newly
  created output behind.

## Change

Every snapshot now stages its plaintext, verification copy and optional
ciphertext inside an unpredictable `os.MkdirTemp` directory beside the final
backup. The directory is mode `0700`; copied files are mode `0600`. The same
private-directory approach protects the throw-away copies used during restore
verification. Normal return and verification failure remove this invocation's
temporary directory without touching legacy predictable names.

All backup modes use the stale-directory sweep. Directories with the backup
prefix older than one hour are considered only after obtaining an exclusive
process-held lease; age alone does not establish that a directory is abandoned.
Fresh directories, active leases and directories without that prefix remain
untouched. Legacy orphan directories without lease files are still reclaimed.

`copyFileSync` uses `O_EXCL` and removes an incomplete output only after it has
successfully created that output. Restore no longer deletes the destination
after an exclusive-creation refusal. Existing production callers expect a new
destination; restore explicitly removes its old staging path before copying.

Backup names, manifest fields, signing and encryption formats, restore
credential invalidation, ledger rules and migrations remain unchanged.

## Validation

Toolchain: Go 1.26.11 on Linux.

- All five new regressions pass with the race detector.
- Existing snapshot/manifest verification, overwrite refusal, restore drill and
  signed-backup substitution tests pass with the race detector.
- The broader backup/restore/migration race run completed with one local
  environment failure: the restore-drill subprocess's Go VCS discovery selected
  `/workspace` instead of this nested worktree. Re-running that test and the
  focused final set with `GOFLAGS=-buildvcs=false` passed. That flag applies only
  to this local validation process; repository build and release settings were
  not changed.
- `go vet ./cmd/circula` and `git diff --check` pass.
- Root integration must run aggregate checks and independent review on the
  final integrated commit before merge. Native Windows/macOS qualification is
  supplied by the repository CI, not claimed from this Linux run.

## Threat boundary

This closes predictable-file overwrite and staging exposure within the normal
operator-controlled filesystem boundary. Scheduled backup directories already
use `ensurePrivateDir`. An operator-selected manual backup directory still
needs appropriate filesystem ownership. This is not a defense against an
administrator, a hostile process under the same operating-system account, or
an attacker allowed to rename the private directory through a writable parent.
It does not prove forensic erasure on copy-on-write storage. A3-F02 / DEC-014 and
the privacy-retention policy questions remain unresolved.

## Independent review correction: active backup leases

Reviewer G identified a regression in `cdbfefc`: a backup running for more than
one hour could lose its private directory when another backup swept old
directories. `TestAuditGActiveBackupSurvivesStaleSweep` deterministically ages an
active backup directory and starts a second backup. It failed before this
follow-up because the second backup removed the first backup's verification
file and snapshot.

All private backup directories now acquire the existing native `acquireLock`
lease immediately after creation and retain it through snapshot, verification,
or encrypted pre-migration work. The sweeper must obtain the same nonblocking
lease before deleting an old directory. A process crash releases the native
lock, so its orphan remains reclaimable. Legacy directories without a lease
file retain the original age-based eligibility.

Cleanup holds the lease while removing snapshot contents. It then releases the
handle before removing the lease file and empty directory, because Windows
disallows deleting the exclusively opened lease file. Sweep candidates whose
directory identity changed or whose lease path is a symlink/nonregular file are
left untouched. No new permanent process registry or timestamp heartbeat is
required.

Follow-up regressions cover the originally reported active-backup failure, a
real subprocess holding an aged directory, reclamation after killing that
process, and directory/lease symlinks. The existing fresh-directory and legacy
orphan cleanup tests remain. Execution is on Linux; Windows and macOS runtime
qualification remains the native CI gate. The filesystem ownership boundary
above still applies, and legacy directories from older binaries do not acquire
leases retroactively.

## CI lint qualification

The PR #72 lint job at published head
`946089539c57469241ab294e16377086e0add6b4` reported one unchecked test cleanup
and three G703 taint-analysis findings. The test now reports an error if closing
its database fails.

The three filesystem findings were independently traced before adding
source-local `#nosec G703` explanations:

- The two removals in `removeLeasedBackupWork` receive either a private
  `MkdirTemp` result or a backup-directory entry selected by prefix, directory
  identity and an acquired lease. The lease filename is a constant.
- `secureRemove` receives these owned work entries, exclusive-created decrypt
  outputs, or the fixed restore-staging path derived from the operator's
  database configuration. Its `Lstat` inspection does not follow symlinks.
- HTTP/member input and backup-manifest filenames do not choose these cleanup
  destinations. Manual backup parents remain operator-selected; not every
  parent is made private automatically. The existing filesystem ownership and
  concurrent-writer limitations above still apply.

The next lint run at `cac2e0e148abde6c7666c4064f792e3acdaf1eb3` exposed two
additional G703 reports at the final `os.Remove` calls in `secureRemove` and
`secureRemoveAll`. They have the same traced provenance: an operator-controlled
staging file or an entry enumerated inside owned backup work. `os.Remove`
unlinks a final symlink rather than following its target and removes a directory
only when empty. Those two individual sinks also carry provenance comments.

Only these five reported sinks have an annotation. Scanner configuration and
all other security checks remain enabled. The runtime cleanup, lease and path
validation behavior is unchanged by these lint corrections.

Validation used the CI-pinned golangci-lint v2.15.0 with Go 1.26.11 against the
complete repository and the existing configuration:
`golangci-lint run --max-same-issues=0 --max-issues-per-linter=0 ./...`.
The result was `0 issues.` (exit 0). The command-line flags remove output caps
for this verification run; they do not disable checks or change configuration.
