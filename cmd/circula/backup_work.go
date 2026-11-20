package main

import (
	"fmt"
	"os"
	"path/filepath"
)

const backupWorkLease = "active.lock"

// newBackupWork keeps a process-held lease for the entire lifetime of a private
// staging directory. Directory age alone cannot distinguish a slow backup from
// a crashed one. The operating system releases the lease after a process crash.
func newBackupWork(parent string) (string, func(), error) {
	if err := checkStagingParent(parent); err != nil {
		return "", nil, err
	}
	work, err := os.MkdirTemp(parent, tempDirPrefix)
	if err != nil {
		return "", nil, err
	}
	created, err := os.Lstat(work) // #nosec G703 -- work was just created by MkdirTemp inside the checked parent; Lstat only records its identity.
	if err != nil {
		return "", nil, err
	}
	lease, err := acquireLock(filepath.Join(work, backupWorkLease))
	if err != nil {
		secureRemoveAll(work)
		return "", nil, fmt.Errorf("lease backup work directory: %w", err)
	}
	return work, func() { removeLeasedBackupWork(work, lease, created) }, nil
}

// removeLeasedBackupWork removes data while the lease is held. Windows refuses
// to delete the exclusively opened lease file, so release it only after all
// other contents are gone, then remove the lease file and empty directory.
//
// want is the identity the work directory had when it was created or
// inspected. If the name now refers to something else (renamed and replaced,
// AUD4-D02), nothing is followed or removed; only the lease is released.
// The remaining window between this check and ReadDir needs the parent's
// owner (this user or root, checkStagingParent) to act, which is outside the
// threat model (TA-5).
func removeLeasedBackupWork(work string, lease *dbLock, want os.FileInfo) {
	if cur, err := os.Lstat(work); err != nil || !cur.IsDir() || !os.SameFile(want, cur) { // #nosec G703 -- identity check of owned work; Lstat does not follow symlinks.
		_ = lease.Release()
		return
	}
	if entries, err := os.ReadDir(work); err == nil {
		for _, entry := range entries {
			if entry.Name() == backupWorkLease {
				continue
			}
			path := filepath.Join(work, entry.Name())
			if entry.IsDir() {
				secureRemoveAll(path)
			} else {
				secureRemove(path)
			}
		}
	}
	_ = lease.Release()
	_ = os.Remove(filepath.Join(work, backupWorkLease)) // #nosec G703 -- constant lease name inside owned MkdirTemp work or an identity-checked, leased backup-directory entry.
	_ = os.Remove(work)                                 // #nosec G703 -- work has the same checked provenance; this removes only the now-empty owned directory.
}

// trySweepBackupWork also handles legacy orphan directories without a lease
// file. Refuse symlink/non-regular lease paths before acquireLock can truncate
// them. Any inability to acquire a lease leaves the directory untouched.
func trySweepBackupWork(work string, original os.FileInfo) {
	current, err := os.Lstat(work)
	if err != nil || !current.IsDir() || !os.SameFile(original, current) {
		return
	}
	path := filepath.Join(work, backupWorkLease)
	if info, err := os.Lstat(path); err == nil {
		if !info.Mode().IsRegular() {
			return
		}
	} else if !os.IsNotExist(err) {
		return
	}
	lease, err := acquireLock(path)
	if err != nil {
		return
	}
	removeLeasedBackupWork(work, lease, original)
}
