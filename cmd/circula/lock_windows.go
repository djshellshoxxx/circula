//go:build windows

package main

import (
	"errors"
	"fmt"
	"os"
	"syscall"
)

// errSharingViolation is ERROR_SHARING_VIOLATION (32), which the syscall
// package does not export.
const errSharingViolation = syscall.Errno(32)

// acquireLock opens path with no sharing mode, so a second process that tries
// to open it fails with a sharing violation for as long as this process runs.
func acquireLock(path string) (*dbLock, error) {
	p, err := syscall.UTF16PtrFromString(path)
	if err != nil {
		return nil, err
	}
	h, err := syscall.CreateFile(p, syscall.GENERIC_READ|syscall.GENERIC_WRITE, 0,
		nil, syscall.OPEN_ALWAYS, syscall.FILE_ATTRIBUTE_NORMAL, 0)
	if err != nil {
		if errors.Is(err, errSharingViolation) {
			return nil, ErrLocked
		}
		return nil, fmt.Errorf("lock %s: %w", path, err)
	}
	return newDBLock(os.NewFile(uintptr(h), path))
}

func unlockFile(f *os.File) error {
	return nil // closing the handle releases the lock
}
