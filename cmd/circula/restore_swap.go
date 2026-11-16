package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sync/atomic"
)

const restoreMarkerSuffix = ".restore-in-progress"

var restoreSidecars = []string{"", "-wal", "-shm", "-journal"}

type restoreValidationKey struct{}

type restoreValidationPermit struct {
	path   string
	active atomic.Bool
}

type restoreMarker struct {
	Version     int    `json:"version"`
	Database    string `json:"database"`
	Staged      string `json:"staged"`
	Previous    string `json:"previous"`
	HadDatabase bool   `json:"had_database"`
}

// checkInterruptedRestore runs before every command-owned SQLite open. A
// marker can be corrupt or empty after a crash; its existence alone blocks
// startup. Saved files from older versions without markers also fail closed.
// Only swapDatabase's validation callback receives the path-scoped bypass.
func checkInterruptedRestore(ctx context.Context, path string) error {
	absolute, err := filepath.Abs(path)
	if err != nil {
		return err
	}
	if allowed, ok := ctx.Value(restoreValidationKey{}).(*restoreValidationPermit); ok && allowed.path == absolute && allowed.active.Load() {
		return nil
	}
	candidates := []string{path + restoreMarkerSuffix}
	for _, suffix := range restoreSidecars {
		candidates = append(candidates, path+".pre-restore-swap"+suffix)
	}
	for _, candidate := range candidates {
		if _, err := os.Lstat(candidate); err == nil { // #nosec G703 -- inspect only fixed restore suffixes of the operator-configured database path.
			return interruptedRestoreError(path)
		} else if !errors.Is(err, fs.ErrNotExist) {
			return fmt.Errorf("checking restore state: %w", err)
		}
	}
	return nil
}

func interruptedRestoreError(path string) error {
	return fmt.Errorf("interrupted restore detected for %s: keep the service stopped; preserve %s, %s.restore-partial, %s.pre-restore-swap and their -wal/-shm/-journal files, plus %s%s; recover and verify one coherent database set before removing the marker or saved files (see docs/ops/backup-restore.md, Interrupted restore recovery)", path, path, path, path, path, restoreMarkerSuffix)
}

func writeRestoreMarker(path, newFile string, hadDB bool) error {
	target, err := filepath.Abs(path)
	if err != nil {
		return err
	}
	staged, err := filepath.Abs(newFile)
	if err != nil {
		return err
	}
	b, err := json.MarshalIndent(restoreMarker{Version: 1, Database: target, Staged: staged, Previous: target + ".pre-restore-swap", HadDatabase: hadDB}, "", "  ")
	if err != nil {
		return err
	}
	f, err := os.OpenFile(path+restoreMarkerSuffix, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600) // #nosec G703 -- exclusive marker beside the operator-configured database; existing paths are never overwritten.
	if err != nil {
		return err
	}
	_, writeErr := f.Write(append(b, '\n'))
	if writeErr == nil {
		writeErr = f.Sync()
	}
	if err := errors.Join(writeErr, f.Close()); err != nil {
		return err
	}
	return syncDir(filepath.Dir(path))
}

func syncRestoreFile(path string) error {
	f, err := os.OpenFile(path, os.O_RDWR, 0) // #nosec G703 -- flush the existing operator-selected database; no create or truncate flags.
	if err != nil {
		return err
	}
	return errors.Join(f.Sync(), f.Close())
}

func clearRestoreMarker(path string) error {
	if err := os.Remove(path + restoreMarkerSuffix); err != nil { // #nosec G703 -- fixed marker created for this operator-selected restore.
		return err
	}
	return syncDir(filepath.Dir(path))
}

func removeRestoreFiles(path string) error {
	for _, suffix := range restoreSidecars {
		if err := os.Remove(path + suffix); err != nil && !errors.Is(err, fs.ErrNotExist) { // #nosec G703 -- remove only this restore's database and fixed SQLite sidecar names.
			return err
		}
	}
	return nil
}

// swapDatabase leaves a durable marker throughout replacement and validation.
// A crash may split the old DB/WAL across two names; startup must refuse to
// open either set until an operator recovers it. No saved file is overwritten.
// Directory sync uses the existing platform helper (no-op on Windows), so
// process-crash refusal is portable; power-loss durability is filesystem-specific.
func swapDatabase(ctx context.Context, path, newFile string, hadDB bool, check func(context.Context) error) error {
	return swapDatabaseSteps(ctx, path, newFile, hadDB, check, nil)
}

// checkpoint is a test seam for terminating a subprocess at exact file-system
// transitions. Production never selects crash behavior from flags/environment.
func swapDatabaseSteps(ctx context.Context, path, newFile string, hadDB bool, check func(context.Context) error, checkpoint func(string)) error {
	if err := checkInterruptedRestore(ctx, path); err != nil {
		return err
	}
	if err := syncRestoreFile(newFile); err != nil {
		return fmt.Errorf("syncing staged database: %w", err)
	}
	if err := writeRestoreMarker(path, newFile, hadDB); err != nil {
		return fmt.Errorf("preparing restore marker: %w", err)
	}
	step := func(name string) {
		if checkpoint != nil {
			checkpoint(name)
		}
	}
	step("prepared")
	aside := path + ".pre-restore-swap"
	moved := []string{}
	installed := false
	rollback := func(cause error) error {
		if installed {
			if err := removeRestoreFiles(path); err != nil {
				return errors.Join(cause, err, interruptedRestoreError(path))
			}
		}
		for _, suffix := range moved {
			if err := os.Rename(aside+suffix, path+suffix); err != nil { // #nosec G703 -- rollback only the files moved aside by this restore.
				return errors.Join(cause, err, interruptedRestoreError(path))
			}
		}
		if err := syncDir(filepath.Dir(path)); err != nil {
			return errors.Join(cause, err, interruptedRestoreError(path))
		}
		if err := clearRestoreMarker(path); err != nil {
			return errors.Join(cause, err, interruptedRestoreError(path))
		}
		return fmt.Errorf("%w; restore rolled back to the previous database state", cause)
	}
	if hadDB {
		for _, suffix := range restoreSidecars {
			err := os.Rename(path+suffix, aside+suffix) // #nosec G703 -- preserve operator-selected database files under fixed recovery names.
			if err != nil {
				if suffix != "" && errors.Is(err, fs.ErrNotExist) {
					continue
				}
				return rollback(fmt.Errorf("moving the current database aside: %w", err))
			}
			moved = append(moved, suffix)
			if err := syncDir(filepath.Dir(path)); err != nil {
				return rollback(err)
			}
			stage := "old-database"
			if suffix != "" {
				stage = "old" + suffix
			}
			step(stage)
		}
	}
	if err := os.Rename(newFile, path); err != nil { // #nosec G703 -- install the verified operator-selected staging file under the restore lock/marker.
		return rollback(fmt.Errorf("replacing the database: %w", err))
	}
	installed = true
	if err := syncDir(filepath.Dir(path)); err != nil {
		return rollback(err)
	}
	if filepath.Dir(path) != filepath.Dir(newFile) {
		if err := syncDir(filepath.Dir(newFile)); err != nil {
			return rollback(err)
		}
	}
	step("installed")
	absolute, err := filepath.Abs(path)
	if err != nil {
		return rollback(err)
	}
	permit := &restoreValidationPermit{path: absolute}
	validationCtx := context.WithValue(ctx, restoreValidationKey{}, permit)
	checkErr := func() error {
		permit.active.Store(true)
		defer permit.active.Store(false)
		return check(validationCtx)
	}()
	if checkErr != nil {
		return rollback(checkErr)
	}
	if err := syncRestoreFile(path); err != nil {
		return rollback(err)
	}
	if err := syncDir(filepath.Dir(path)); err != nil {
		return rollback(err)
	}
	step("verified")
	if hadDB {
		if err := removeRestoreFiles(aside); err != nil {
			return errors.Join(err, interruptedRestoreError(path))
		}
		if err := syncDir(filepath.Dir(path)); err != nil {
			return errors.Join(err, interruptedRestoreError(path))
		}
	}
	step("cleaned")
	if err := clearRestoreMarker(path); err != nil {
		return fmt.Errorf("removing completed restore marker: %w", err)
	}
	return nil
}
