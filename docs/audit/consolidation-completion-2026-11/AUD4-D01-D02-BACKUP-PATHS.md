# AUD4-D01 and AUD4-D02: backup path hardening

Found by this pass's independent data/backup audit. Tree reviewed: `main@ffa2758` plus PR #71 and PR #72.
Both findings are Low. Both need an operator to choose an unusual directory (D01) or a shared one (D02).

## AUD4-D01: SQLite URI metacharacters in paths (CWE-116)

`store.Open` and `store.OpenRaw` put the path into a `file:` URI without escaping it. A3-05 refuses `?`, `#`
and `%` only in the `database` setting. Backup work paths come from `backup-dir` and from
`circula backup <dest>`, and those can contain these characters.

| Path component | Pre-fix behaviour (probe) |
|---|---|
| `a?b` | Opened `…/a`. The driver parsed `b/x.db` as query text and `trusted_schema` was 1 |
| `a#b` | Opened `…/a`, a different file |
| `a?&_fk=0&x=` | `foreign_keys` = 0 |
| `a%41b` | Open failed or decoded to another name |

In backups the effect was a manifest with `integrity: ok` and `schema: 0` from an empty stray file, or the ledger
check running on the live database. Pre-migration verification then failed closed on the schema mismatch.

Fix: `internal/store/uri.go` `fileURI` percent-encodes `%`, `?` and `#`. SQLite decodes `%HH` in URI paths. Every
other character is unchanged, so Windows and existing paths behave as before. The `database` refusal (A3-05)
stays as defence in depth.

Regressions:

- `TestSQLiteOpensExactPathWithURIMetacharacters` (`internal/store`) checks that the opened file is exactly the
  requested one and that `trusted_schema`, `foreign_keys` and `secure_delete` hold, for both the normal and the
  raw pool.
- `TestBackupDirectoryWithURIMetacharacters` (`cmd/circula`) checks that integrity, schema and ledger are recorded
  from the snapshot itself and that no stray file is created.

## AUD4-D02: check-then-use in backup cleanup (CWE-367 / CWE-59)

`secureRemove` checked an entry with `Lstat`, then opened it for overwriting by name. `secureRemoveAll` called
`ReadDir`, which follows symlinks. `sweepStaleTemp` swept every old `circula-*` directory in the destination
directory. When another local user can write to the destination of a manual `circula backup <dest>`,
they could plant or swap entries so that cleanup zeroes or deletes data outside the work area. Scheduled,
pre-migration and pre-restore backups already used a 0700 directory (`ensurePrivateDir`) and were not exposed.
The sweep predates PR #72. PR #72 narrowed the gap but did not close it.

Fix (`cmd/circula/backupcrypt.go`, `snapshot.go`, `backup_work.go`, `fileident_{unix,other}.go`):

- `zeroFile` opens with `O_NOFOLLOW|O_NONBLOCK` (Unix) and writes only if the open handle is the same regular,
  single-link file that was inspected.
- `secureRemoveAll` unlinks a directory that has been replaced by a symlink and never descends into it.
- `sweepStaleTemp` sweeps only a directory owned by the effective user with no group/other write bit.
- `newBackupWork` refuses a destination owned by anyone other than the effective user or root. It also refuses
  one that others can write, unless it is sticky (such as `/tmp`). Only the directory's owner, or the owner of an
  entry, can rename that entry.
- Cleanup records the work directory's identity at creation. If the name now refers to something else, cleanup
  does not follow it and only releases the lease.

Review follow-up (independent review on PR #81, findings G03/G04):

- G03: a sticky directory owned by another unprivileged user was still accepted. That user could rename the work
  directory and replace its name with a symlink, and cleanup would then delete files outside the work area. Fixed
  by the owner rule and the identity re-check. Regressions: `TestLeasedWorkCleanupRefusesSubstitutedDirectory` and
  `TestBackupStagingRefusesForeignOwnedParent`, which runs only as root on Unix. Both fail before the fix.
- G04: the runbook and the security register now say which checks are Unix-only and state the Windows ACL
  boundary.

Regressions: `TestZeroFileRefusesReplacedName`, `TestSecureRemoveLeavesHardLinkedDataIntact`,
`TestSecureRemoveAllDoesNotFollowDirectorySymlink`, `TestBackupStagingRequiresPrivateOrStickyParent`.

## Fail-before evidence

With the production changes removed and the new tests kept:

- the store test failed;
- the `bk?x` and `bk%41x` backup subtests failed;
- the hard-link, directory-symlink and shared-directory tests failed.

`zeroFile` is new, so its test cannot run before the fix. With the fix, all tests pass.

## Residual limits

- Windows: there are no `O_NOFOLLOW`, link-count or owner/mode checks. Protection relies on NTFS ACLs on the
  destination directory, the identity re-checks, and the privilege Windows requires to create symlinks.
- Only the immediate destination directory is checked. A writable ancestor directory lets its owner replace the
  whole destination. Between the identity re-check and `ReadDir`, a race remains that only the directory's owner
  (this user or root) can win.
- An attacker running as the same user as the operator is out of scope: they can read the database directly (TA-5).
- Overwriting before unlinking does not guarantee physical erasure on SSD or copy-on-write file systems
  (see `docs/ops/backup-restore.md`).
