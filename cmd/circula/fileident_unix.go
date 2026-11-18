//go:build unix

package main

import (
	"fmt"
	"os"
	"syscall"
)

// privateParent reports whether only this process's user can add, rename or
// remove entries in the directory described by fi (AUD4-D02). Sweeping
// leftover work directories trusts their names and contents, which is safe
// only when nobody else can create or swap them.
func privateParent(fi os.FileInfo) bool {
	st, ok := fi.Sys().(*syscall.Stat_t)
	return ok && fi.IsDir() && int(st.Uid) == os.Geteuid() && fi.Mode().Perm()&0o022 == 0
}

// checkStagingParent refuses a backup destination directory in which other
// users could rename this process's private work directory and substitute
// their own (AUD4-D02). A sticky shared directory such as /tmp is accepted:
// there, only the owner of an entry may rename or remove it.
func checkStagingParent(dir string) error {
	fi, err := os.Stat(dir) // #nosec G703 -- dir is the operator-chosen backup destination; only its mode is read.
	if err != nil {
		return err
	}
	if fi.Mode().Perm()&0o022 != 0 && fi.Mode()&os.ModeSticky == 0 {
		return fmt.Errorf("backup destination directory %s is writable by other users; use a private directory", dir)
	}
	return nil
}

// singleLink reports whether the regular file described by fi has exactly
// one name. Overwriting a file with further hard links would also destroy
// the data seen through its other names (AUD4-D02).
func singleLink(fi os.FileInfo) bool {
	st, ok := fi.Sys().(*syscall.Stat_t)
	return ok && st.Nlink == 1
}

// overwriteOpenFlags make the overwrite open fail on a final symlink and
// return at once on a FIFO, instead of following or blocking (AUD4-D02).
const overwriteOpenFlags = syscall.O_NOFOLLOW | syscall.O_NONBLOCK
