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

All backup modes use the existing age-bounded stale-directory sweep. Directories
with the backup prefix older than one hour are eligible; fresh directories and
directories without that prefix remain untouched. This preserves the existing
backup-work ownership convention.

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
