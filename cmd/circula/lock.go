package main

import (
	"errors"
	"fmt"
	"os"
)

// ErrLocked means another process holds the database's writer lock.
var ErrLocked = errors.New("another circula process holds the database lock")

// dbLock is the single-writer guard for one database file (ARC-011). The lock
// file sits next to the database as <database>.lock and records the owner pid.
type dbLock struct {
	f *os.File
}

func newDBLock(f *os.File) (*dbLock, error) {
	if err := f.Truncate(0); err != nil {
		_ = f.Close()
		return nil, err
	}
	if _, err := fmt.Fprintf(f, "%d\n", os.Getpid()); err != nil {
		_ = f.Close()
		return nil, err
	}
	return &dbLock{f: f}, nil
}

// lockPath returns the lock file path for a database path.
func lockPath(database string) string { return database + ".lock" }

// Release drops the lock.
func (l *dbLock) Release() error {
	if l == nil || l.f == nil {
		return nil
	}
	_ = unlockFile(l.f)
	err := l.f.Close()
	l.f = nil
	return err
}
