package main

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
)

// exportSpoolPattern matches the archive files the API spools exports into
// (internal/api, os.CreateTemp pattern circula-export-*.zip).
const exportSpoolPattern = "circula-export-*.zip"

// prepareExportSpool makes export-spool-dir ready before serving (BR-06): it
// creates the directory (0700) when missing, removes archives a stopped
// process left behind (on POSIX they are unlinked at once, so only other
// platforms leave any), and proves that a file can be created there. A
// directory that fails any step stops the start, rather than every export
// failing later.
func prepareExportSpool(dir string) error {
	if dir == "" {
		return errors.New("no directory configured")
	}
	if fi, err := os.Stat(dir); err != nil {
		if !errors.Is(err, fs.ErrNotExist) {
			return err
		}
		if err := os.MkdirAll(dir, 0o700); err != nil {
			return err
		}
	} else if !fi.IsDir() {
		return fmt.Errorf("%s is not a directory", dir)
	}
	stale, err := filepath.Glob(filepath.Join(dir, exportSpoolPattern))
	if err != nil {
		return err
	}
	for _, p := range stale {
		if err := os.Remove(p); err != nil {
			return fmt.Errorf("remove stale export archive: %w", err)
		}
	}
	f, err := os.CreateTemp(dir, exportSpoolPattern)
	if err != nil {
		return fmt.Errorf("%s is not writable: %w", dir, err)
	}
	name := f.Name()
	cerr := f.Close()
	if err := os.Remove(name); err != nil {
		return err
	}
	return cerr
}
