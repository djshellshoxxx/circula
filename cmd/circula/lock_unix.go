//go:build unix

package main

import (
	"errors"
	"fmt"
	"os"
	"syscall"
)

// acquireLock takes an exclusive, non-blocking advisory lock on path. The
// lock is released when the process exits or Release is called.
func acquireLock(path string) (*dbLock, error) {
	f, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0o600) //nolint:gosec // G703: operator-supplied database path
	if err != nil {
		return nil, fmt.Errorf("open lock file %s: %w", path, err)
	}
	if err := syscall.Flock(int(f.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); err != nil {
		_ = f.Close()
		if errors.Is(err, syscall.EWOULDBLOCK) {
			return nil, ErrLocked
		}
		return nil, fmt.Errorf("lock %s: %w", path, err)
	}
	return newDBLock(f)
}

func unlockFile(f *os.File) error {
	_ = syscall.Flock(int(f.Fd()), syscall.LOCK_UN)
	return nil
}
