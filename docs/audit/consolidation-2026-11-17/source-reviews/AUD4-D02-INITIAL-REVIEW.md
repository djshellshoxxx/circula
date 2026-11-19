# Independent review of concurrent AUD4-D02 backup cleanup

Reviewed commit: `3dcb3f8bfd07e0a8382d06762d7e9ba32c0aacbd`.
Reviewed tree: `fdaa1ab68cd931a361bb4c69d4fe99215c7926bb`.
Reviewer: integration Agent G. Source and remote state were not modified.

Latest-head follow-up: PR #81 head `5776be13613ddbf172bc73c97a9fd83e26c9f989`, tree `7de852707f23b06be231ce0cc9f69ad0c675c99c`. Compared the complete follow-up diff and re-executed G-03 at this head: **still fails**, with the unrelated canary deleted in 0.004s. Evidence: `backup-path-lease-substitution-5776be1.log` and `backup-path-lease-substitution-5776be1-command.json`. D02 runtime source and documentation are unchanged. The intervening commits add Windows guards around invalid `?` filesystem cases and a direct URI-encoding test; the previously reported invalid-name test issue is addressed in source, with native qualification still required. G-03 and G-04 remain blocking at the latest reviewed head.

## Verdict

**Changes requested before integration of this exact patch.** The Unix overwrite hardening is useful and the previously accepted private staging/live leases still pass focused race tests. However, the accepted-parent check does not establish the stated protection against another user replacing the work directory, and the real leased-cleanup entry point still follows a replacement symlink. New operating guidance also states Unix-only protections without qualifying Windows. Agent C separately reviewed the SQLite URI changes and identified invalid Windows filename test cases.

## G-03: accepted sticky-parent ownership does not protect leased cleanup

`cmd/circula/fileident_unix.go:20–32` accepts any sticky directory, without checking its owner. Its comment says that only the entry owner can rename or remove an entry. Sticky directories also permit the **directory owner** to do so. An operator who selects a sticky directory owned by another unprivileged user therefore accepts a parent whose owner can replace the process's work-directory name.

`cmd/circula/backup_work.go:33–49` still calls `os.ReadDir(work)` and then cleans joined paths without checking that `work` remains the leased directory. Holding a lock on the original directory's `active.lock` does not prevent its parent-directory entry from being renamed. The `secureRemoveAll` guard added by this commit does not protect this initial `ReadDir` in the producer's returned cleanup closure.

Deterministic reproduction at the exact reviewed head:

1. Create an accepted sticky parent and call `newBackupWork`.
2. Create a separate directory containing a canary file.
3. Rename the leased work directory and place a symlink to the canary directory at its old name.
4. Invoke the returned cleanup function.
5. The canary is deleted; its final read reports `no such file or directory`.

The probe is `backup-path-review-probe_test.go`, added only through a Go overlay. `backup-path-lease-substitution-command.json` pins the source head, command and exit 1; `backup-path-lease-substitution.log` records the expected failure in 0.010s. **This is a deterministic name-substitution test, not an execution under two different UIDs.** The other-user reachability follows from the unchecked sticky-parent owner and directory-owner rename permission. A hostile process sharing the operator's UID remains outside the stated threat model and is not required for the identified ownership gap.

Recommended bounded correction: reject a parent owned by an untrusted UID, retaining the intended trusted root-owned sticky temporary-directory case; ensure leased cleanup skips a work name whose original directory identity has been replaced. Keep the trusted ancestor/parent assumption explicit, because checking a name once does not by itself eliminate later substitutions by a hostile parent owner. Retain release of the lease on all cleanup paths, and preserve active-operation protection, crash reclamation and Windows close-before-delete ordering.

## G-04: platform and filesystem guarantees are overstated

`docs/ops/backup-restore.md:62–64` says cleanup never overwrites a multiply linked file, shared non-sticky destinations are refused, and shared directories are never swept. `docs/security/security-review.md` labels the cleanup race closed and repeats unconditional claims.

Those parent/link checks exist only in `fileident_unix.go`. In `cmd/circula/fileident_other.go:9–21`, `privateParent` accepts any directory, `checkStagingParent` always succeeds, `singleLink` always returns true and the no-follow/nonblocking flags are absent. On Windows the existing file-handle identity recheck is useful, but it cannot establish the omitted ownership, link-count or no-follow guarantees. The detailed audit note acknowledges these limits; the runbook and security register must state them consistently. Avoid describing AUD4-D02 as fully closed across platforms unless the missing native protections are implemented and qualified.

The new `secureRemoveAll` pre-`ReadDir` `Lstat` rejects a symlink already present at inspection. It is not an atomic directory traversal primitive and does not justify an unrestricted claim about a parent that another actor can replace concurrently. Trusted directory ownership/ancestors remain a boundary.

## Preserved behavior and positive verification

The Unix overwrite change opens without following a final symlink, avoids blocking on a FIFO, and checks the opened handle's regular-file type, identity and link count before writing. It introduces no `O_TRUNC`, so failed identity checks do not truncate a substitute. Existing caller cleanup scope, snapshot formats, hashes, authentication, schemas and ledger semantics are not changed by D02.

Independent `go test -p 2 -race -count=1` at this head passed twelve focused backup tests in 8.030s. Coverage includes the four new D02 tests, the original active-aged-work regression, a real subprocess hold/crash/reclaim sequence, lease symlink rejection, private snapshot success/failure, exclusive copy ownership, predictable-path preservation and stale/fresh cleanup. Exact command/exit 0: `backup-path-focused-race-command.json`; output: `backup-path-focused-race.log`.

This passing set does not cover the failing G-03 scenario, native Windows/macOS runtime behavior or a full aggregate suite. The SQLite URI portion is independently covered by Agent C's `AUD4-D01-INVESTIGATION.md`; that review also identifies the unguarded `?` path cases in both new test files as a Windows qualification blocker. No native Windows execution is claimed here.

The final sentence above describes the original `3dcb3f8` review state. The `5776be1` follow-up addresses those invalid filename cases as noted at the top; it does not address the two D02 review findings.
