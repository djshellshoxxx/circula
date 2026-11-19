# Messaging Windows storage-privacy assessment

## Decision

The existing contract clearly requires private database storage and private backup/restore. It does **not** specify a complete Windows ACL profile or native storage acceptance. A platform-contract extension is therefore still needed before claiming Windows support. The narrowest practical option is a single execution-identity profile in a dedicated private directory, followed by actual Windows tests. The coordinator can adopt that engineering profile without choosing new messaging, retention, network-governance or E2EE policy.

Reviewed candidate: `7e9708d8e50a1959af399aa5956fb2a54ab880b6`, tree `a5556af33aca22f91ee5371d1d979217bf4936b8`. The worktree remains clean and unchanged. This was read-only source review: no Go jobs or Windows execution occurred, and no content disclosure or authorization bypass was demonstrated.

## Existing contract and observed implementation

`CONTRACT.md` specifies a private regular SQLite file, mode 0600 for new files, rejection of unsafe existing files, and private backup permissions. `MIGRATION-PLAN.md` requires consistent SQLite backup, private permissions and migration rerun. `SPEC-SNAPSHOT.md` explicitly permits the hosting operator to access message content and leaves operational integration/qualification separate. None names permitted Windows ACE principals, directory inheritance, ownership validation or a backup ACL procedure.

The concrete current behavior is fail-closed: `Open` creates/opens the file, then rejects `info.Mode().Perm() & 0077` before opening SQLite. Go 1.26.11 synthesizes Windows regular-file permissions as 0666 or 0444, both rejected. Its `Chmod` API uses only the owner-writable bit to alter the Windows read-only attribute; it does not implement a private DACL. Consequently the existing restore fixture's `os.Chmod(backup,0600)` cannot serve as Windows ACL qualification.

The pinned `modernc.org/sqlite v1.60.1` Windows implementation supplies a zero security-attributes pointer to `CreateFileW/A` in `_winOpen` and `_winHandleOpen`. The latter explicitly opens SHM files. The module enables WAL mode. Windows assigns default/inherited file security when explicit creation security is absent; SQLite's WAL is separate persistent database state. Restricting only the main database file would leave the privacy of newly created sidecars dependent on the directory's inheritable ACL. Existing child files need their own validation because directory security does not retroactively control their access. These are source-based constraints on a future adapter, not evidence that the currently rejecting constructor disclosed data. [Microsoft file security](https://learn.microsoft.com/en-us/windows/win32/fileio/file-security-and-access-rights), [new-object security](https://learn.microsoft.com/en-us/windows/win32/secauthz/security-descriptors-for-new-objects), [SQLite WAL](https://www.sqlite.org/wal.html).

## Smallest defensible follow-up

1. **Adopt an explicit standalone profile.** Recommend the process-token user SID as the storage identity, with a protected owner-only DACL. Document how services select their execution identity. Additional SYSTEM, Administrators or service-group ACEs should be deliberate rather than inferred from a default ACL. The intended boundary is ordinary other-account access, consistent with the existing owner-private file contract and operator-readable messaging model.
2. **Protect the directory and sidecars.** Use a dedicated private directory with inheritable owner-only child ACLs. Provision it through an explicit helper or documented operator step, and validate before SQLite initialization. Avoid changing the ACL of an arbitrary supplied parent directory. Validate existing database, WAL and SHM files separately; inherited protection applies to newly created objects.
3. **Preserve fail-closed existing-file behavior.** Create new files with their restrictive descriptor atomically. Inspect existing file ownership, DACL and regular-file identity through handles before SQLite writes. Reject null/missing or unsafe security descriptors and unsupported forms without silently repairing existing data. A supplied creation security descriptor is ignored for an existing file, so it is not an existing-file check. [CreateFile semantics](https://learn.microsoft.com/en-us/windows/win32/api/fileapi/nf-fileapi-createfilea).
4. **Define filesystem and path assumptions.** Require a filesystem that supports the selected security model, and protect the path namespace against check/use replacement or reparse surprises. A private file descriptor alone does not bind SQLite's later path-based reopen. The exact supported local/service storage profile and relevant handle/path behavior need acceptance tests. [Creating and opening files](https://learn.microsoft.com/en-us/windows/desktop/FileIO/creating-and-opening-files).
5. **Apply the same profile to backup/restore.** Use explicit ACL setup/validation for the snapshot destination and restored database. Retain the actual VACUUM snapshot, reopen, migration and visibility assertions. A Windows `Chmod(0600)` call is insufficient.

Keep POSIX behavior in its existing helper path. A separate Windows helper can use the already pinned `golang.org/x/sys/windows v0.48.0` APIs for security descriptors, token SID and handle-based security inspection. Directly importing it would require declaring its directness and updating the corresponding exact-dependency guard, without changing its version or the SQLite driver. This is a separate reviewed follow-up; none of those files or guards was edited here.

`NewStore` already allows a host-owned pool, but replacing constructor tests with that path would qualify neither standalone `Open` nor Windows file privacy. It remains a later host integration option with its own acceptance.

## Meaningful native acceptance

| Test group | Required evidence |
|---|---|
| Current regression and successful lifecycle | Actual Windows fail-before for a legitimate private fixture; then new-file create, migrate, write/read, close and reopen with the pinned driver. |
| Resulting object security | Inspect owner/DACL on the database and live WAL/SHM after real writes. A directory-only assertion is insufficient. |
| Unsafe existing state | Reject null/missing DACL, unsafe ordinary-principal grants, foreign ownership, unsafe inheritance and unsupported ACE forms before writes; preserve existing content and ACL on rejection. |
| Directory, filesystem and path handling | Unsafe parent/sidecar rejection; relevant reparse and replacement cases; creation collision without truncation; unsupported security capability rejected. |
| Backup/restore | Real consistent snapshot and restored message/own-view data, with Windows ACL setup and inspection of backup/database/sidecars. |
| Ordinary-account isolation | Where the native harness can provide a second unprivileged token, verify owner access succeeds and unrelated ordinary access fails. |
| Existing behavior | Complete current Windows suite and Linux/macOS regression/race gates, preserving tests and avoiding skips or a driver substitution. |

No native pass follows from this plan or from cross-compilation. Current recovery stays frozen and disabled/HOLD. The machine-readable `assessment.json` records exact Git blobs, pinned implementation-source SHA-256 values, primary documentation sources, scope limits and the proposed tests.
