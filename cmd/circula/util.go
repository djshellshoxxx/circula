package main

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"io/fs"
	"os"
	"os/signal"
	"runtime"
	"syscall"

	"github.com/djshellshoxxx/circula/internal/store"
)

// copyFileSync copies src to dst and flushes dst to stable storage before returning.
func copyFileSync(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer func() { _ = in.Close() }()
	out, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o600)
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, in); err != nil {
		_ = out.Close()
		return err
	}
	if err := out.Sync(); err != nil {
		_ = out.Close()
		return err
	}
	return out.Close()
}

// writeJSONFile writes v as indented JSON atomically.
func writeJSONFile(path string, v any) error {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	b = append(b, '\n')
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, b, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, path)
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

// storeOptions maps the configuration to store.Options (SA2-20).
func storeOptions(cfg *Config) store.Options {
	return store.Options{Synchronous: cfg.Synchronous}
}
