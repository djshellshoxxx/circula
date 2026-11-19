package main

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"io/fs"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"
	"syscall"
)

// copyFileSync copies src to a new dst and flushes dst to stable storage before
// returning. Exclusive creation refuses existing files and planted symlinks.
func copyFileSync(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer func() { _ = in.Close() }()
	out, err := os.OpenFile(dst, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	complete := false
	defer func() {
		if !complete {
			_ = out.Close()
			_ = os.Remove(dst)
		}
	}()
	if _, err := io.Copy(out, in); err != nil {
		return err
	}
	if err := out.Sync(); err != nil {
		return err
	}
	if err := out.Close(); err != nil {
		return err
	}
	complete = true
	return nil
}

// writeJSONFile writes v as indented JSON atomically.
func writeJSONFile(path string, v any) error {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	b = append(b, '\n')
	return writeAtomicPrivateFile(path, b)
}

// writeAtomicPrivateFile writes through an unpredictable, exclusively created file
// in the destination directory, avoiding predictable .tmp symlink attacks.
func writeAtomicPrivateFile(path string, data []byte) error {
	dir := filepath.Dir(path)
	f, err := os.CreateTemp(dir, "."+filepath.Base(path)+".tmp-*")
	if err != nil {
		return err
	}
	tmp := f.Name()
	defer func() { _ = os.Remove(tmp) }() // #nosec G703 -- tmp is the exclusively created temporary file, not request input.
	if err := f.Chmod(0o600); err != nil {
		_ = f.Close()
		return err
	}
	if _, err := f.Write(data); err != nil {
		_ = f.Close()
		return err
	}
	if err := f.Sync(); err != nil {
		_ = f.Close()
		return err
	}
	if err := f.Close(); err != nil {
		return err
	}
	return os.Rename(tmp, path) // #nosec G703 -- destination is an operator-configured output path; tmp is created exclusively alongside it.
}

// signalContext returns a context cancelled on Ctrl-C or SIGTERM.
func signalContext(parent context.Context) (context.Context, func()) {
	return signal.NotifyContext(parent, os.Interrupt, syscall.SIGTERM)
}

// ensurePrivateDir creates dir (and parents) and makes sure the directory
// itself is accessible by its owner only (SA-06). Backups hold password
// hashes and every member's data, so a pre-existing wider directory is
// tightened. Use it only for directories circula manages.
func ensurePrivateDir(dir string) error {
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	fi, err := os.Stat(dir)
	if err != nil {
		return err
	}
	if runtime.GOOS != "windows" && fi.Mode().Perm()&0o077 != 0 {
		return os.Chmod(dir, 0o700)
	}
	return nil
}

// secureDatabase makes sure the database file and its WAL side files are
// owner-only (SA-06). A missing database is created empty with mode 0600 so
// the driver cannot create it with the umask default. It reports whether it
// had to tighten an existing file.
func secureDatabase(path string) (tightened bool, err error) {
	_, err = os.Stat(path)
	if errors.Is(err, fs.ErrNotExist) {
		f, cerr := os.OpenFile(path, os.O_CREATE|os.O_WRONLY, 0o600)
		if cerr != nil {
			return false, cerr
		}
		return false, f.Close()
	}
	if err != nil {
		return false, err
	}
	for _, p := range []string{path, path + "-wal", path + "-shm"} {
		fi, serr := os.Stat(p)
		if serr != nil {
			continue
		}
		if runtime.GOOS != "windows" && fi.Mode().Perm()&0o077 != 0 {
			if err := os.Chmod(p, 0o600); err != nil {
				return tightened, err
			}
			tightened = true
		}
	}
	return tightened, nil
}
