package main

import (
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

// webKeyFile is the persistent signing key for CSRF tokens and signed web
// cookies (SEC-027). It lives in state-dir, next to the database by default.
const webKeyFile = "circula-web.key"

const webKeyLen = 32

// loadOrCreateWebKey returns the persistent 32-byte web signing key, creating
// it on first start. Creation is atomic and race-safe: the key is written to a
// private temporary file (O_EXCL, 0600) and hard-linked into place, which
// fails if another process got there first, in which case that key is loaded.
// An existing file that is readable by group or others, is not a regular
// file, or does not hold exactly 32 base64 bytes is refused rather than
// replaced, since replacing it would silently sign out every user and hide
// tampering.
func loadOrCreateWebKey(stateDir string) ([]byte, error) {
	if stateDir == "" {
		return nil, errors.New("no state directory configured")
	}
	if fi, err := os.Stat(stateDir); err != nil {
		if !errors.Is(err, fs.ErrNotExist) {
			return nil, err
		}
		if err := os.MkdirAll(stateDir, 0o700); err != nil {
			return nil, err
		}
	} else if !fi.IsDir() {
		return nil, fmt.Errorf("state-dir %s is not a directory", stateDir)
	}
	path := filepath.Join(stateDir, webKeyFile)
	for attempt := 0; attempt < 3; attempt++ {
		key, err := readWebKey(path)
		if err == nil {
			return key, nil
		}
		if !errors.Is(err, fs.ErrNotExist) {
			return nil, err
		}
		if err := createWebKey(path); err != nil && !errors.Is(err, fs.ErrExist) {
			return nil, err
		}
	}
	return nil, fmt.Errorf("web key file %s could not be created", path)
}

func readWebKey(path string) ([]byte, error) {
	fi, err := os.Lstat(path)
	if err != nil {
		return nil, err
	}
	if !fi.Mode().IsRegular() {
		return nil, fmt.Errorf("web key file %s is not a regular file", path)
	}
	if runtime.GOOS != "windows" && fi.Mode().Perm()&0o077 != 0 {
		return nil, fmt.Errorf("web key file %s is readable by group or others (mode %o); chmod 600 it, or delete it to sign everyone out and start a new key", path, fi.Mode().Perm())
	}
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	key, err := base64.StdEncoding.DecodeString(strings.TrimSpace(string(b)))
	if err != nil || len(key) != webKeyLen {
		return nil, fmt.Errorf("web key file %s is corrupt (expected %d base64-encoded bytes); delete it to sign everyone out and start a new key", path, webKeyLen)
	}
	return key, nil
}

func createWebKey(path string) error {
	key := make([]byte, webKeyLen)
	if _, err := rand.Read(key); err != nil {
		return fmt.Errorf("web key: %w", err)
	}
	tmp, err := os.CreateTemp(filepath.Dir(path), ".circula-web.key.*")
	if err != nil {
		return err
	}
	defer func() { _ = os.Remove(tmp.Name()) }()
	if err := tmp.Chmod(0o600); err != nil && runtime.GOOS != "windows" {
		_ = tmp.Close()
		return err
	}
	if _, err := tmp.WriteString(base64.StdEncoding.EncodeToString(key) + "\n"); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Sync(); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Link(tmp.Name(), path)
}
