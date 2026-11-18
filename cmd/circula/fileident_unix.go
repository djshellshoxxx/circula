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

// checkStagingParent refuses a backup destination directory in which another
// user could rename this process's private work directory and substitute
// their own (AUD4-D02). The directory's owner can always rename its entries,
// so it must be this user or root. Other users may write to it only if it is
// sticky (such as /tmp): there, only an entry's owner, or the directory's
// owner, may rename or remove it. Only the immediate directory is checked;
// its ancestors must be trusted by the operator.
func checkStagingParent(dir string) error {
	fi, err := os.Stat(dir) // #nosec G703 -- dir is the operator-chosen backup destination; only its owner and mode are read.
	if err != nil {
		return err
	}
	st, ok := fi.Sys().(*syscall.Stat_t)
	if !ok || (int(st.Uid) != os.Geteuid() && st.Uid != 0) {
		return fmt.Errorf("backup destination directory %s is owned by another user; use a directory owned by this user", dir)
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
