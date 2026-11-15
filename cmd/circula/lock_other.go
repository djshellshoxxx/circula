//go:build !unix && !windows

package main

import (
	"errors"
	"os"
)

// acquireLock is unsupported on this platform; serve refuses to run without
// the single-writer guarantee (ARC-011).
func acquireLock(path string) (*dbLock, error) {
	return nil, errors.New("single-process locking is not implemented on this platform")
}

func unlockFile(f *os.File) error { return nil }
