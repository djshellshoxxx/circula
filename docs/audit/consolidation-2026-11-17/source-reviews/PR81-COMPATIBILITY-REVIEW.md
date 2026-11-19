# PR #81 compatibility review — 2026-11-18

Reviewed head: `09ef8a5aa25f11ee4d96f043e7d686e97598b7e3`.

Reviewed tree: `36326959c19c93fdcc8a415d3152e1cc4f183138`.

Delta base: `5776be13613ddbf172bc73c97a9fd83e26c9f989`.

## Verdict

No blocking source-correctness finding within the requested D01 and ordinary
compatibility scope. The URI implementation is unchanged from the independently
verified `3dcb3f8` version. The Windows test correction remains intact. The new
tests exercise the changed failure behavior directly. Root owns the separate
D02 cleanup-identity and platform-documentation security review.

Tracked source was read only. No additional adversarial probes were created.
The detached worktree is clean.

## D01 and Windows coverage

`internal/store/uri.go`, `sqlite.go` and `inspect.go` have no diff from
`3dcb3f8` through this head. Percent, question mark and fragment markers are
escaped before appending driver query parameters. Both store entry points use
the helper and preserve all E-02 and durability pragmas.

The filesystem regressions omit only names containing `?` on Windows:

- Store coverage still opens `plain`, `a#b`, `a%41b` and `a%` directories
  through both normal and raw pools. It compares actual file identity and
  required connection settings.
- Backup coverage retains `bk#x` and `bk%41x` on Windows. It checks the actual
  destination, schema version, integrity, ledger status and absence of a
  truncated-path stray file.
- `TestFileURIEscaping` runs on every platform. Its six cases include ordinary
  Unix and Windows paths, `?`, `#`, `%41`, and query-like filename content.

This preserves valid native filename coverage while testing `?` encoding
without requiring an invalid Windows filesystem name. Native Windows execution
is not claimed by this Linux review.

## New delta since 5776be1

`backup_work.go` records the created directory identity and passes it to the
cleanup function. Both the normal cleanup callback and stale-work path use
the same identity-checking cleanup entry point. `fileident_unix.go` adds the
intended immediate-parent owner restriction while retaining the sticky-mode
condition. No D01 source or unrelated behavior changes appear in this delta.

The two added tests are focused:

- `TestLeasedWorkCleanupRefusesSubstitutedDirectory` leases a work directory,
  renames it, replaces its name with a symlink, and checks that the unrelated
  target data remains unchanged after cleanup. Its assertion detects both
  erasure and deletion of the target data.
- `TestBackupStagingRefusesForeignOwnedParent` creates a directory owned by a
  different UID and requires refusal for both sticky 0777 and ordinary 0755
  modes. It explicitly requires Unix root privileges; it is skipped here
  because the reviewer runs as UID 1000. This run therefore does not supply
  execution evidence for that owner-refusal assertion.

The existing private/non-sticky/sticky-parent regression retains the accepted
same-owner cases and verifies the ordinary compatibility behavior alongside
the new restrictions.

## Verification evidence

Using Go 1.26.11 at the exact head:

- Four focused store tests pass (0.105 s): helper escaping, literal filename
  routing, every pooled connection's trusted-schema setting, and schema-call
  refusal with direct function compatibility. Log: `PR81-09ef8a5-store.log`.
- The new substitution regression and existing private/sticky-parent test
  pass under `-race`; the foreign-owner test is explicitly skipped (1.031 s).
  Log: `PR81-09ef8a5-new-regressions-race.log`.
- A broader optional backup compatibility race command passed all three URI
  backup cases, private/sticky staging, the new substitution case, plaintext
  and encrypted staging, and stale-work cleanup before the process received
  `signal: killed` during the existing
  `TestSnapshotWritesVerifiedManifestAndRestores`. No assertion or race error
  was reported before termination. That overall command did not pass; its
  remaining tests were not completed and it was not repeated. Log:
  `PR81-09ef8a5-backup-race.log`.

Aggregate and native-platform qualification remain the integration lead's
gates. No root-only coverage is inferred from a skipped test.
