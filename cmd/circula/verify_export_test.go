package main

import (
	"archive/zip"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/djshellshoxxx/circula/internal/service"
)

func writeExportPkg(t *testing.T, dir string) {
	t.Helper()
	files := map[string]string{
		"entries.csv": "id,amount\ne1,5\ne2,-5\n",
		"full.json":   `{"tables":{"a":[{"x":1},{"x":2}],"b":[{"y":3}]}}` + "\n",
	}
	rows := map[string]int64{"entries.csv": 2, "full.json": 3}
	man := service.Manifest{SchemaVersion: 1, Scope: service.ExportFull, LedgerSeq: 9}
	for n, c := range files {
		if err := os.WriteFile(filepath.Join(dir, n), []byte(c), 0o600); err != nil {
			t.Fatal(err)
		}
		sum := sha256.Sum256([]byte(c))
		man.Files = append(man.Files, service.ManifestFile{Name: n, Rows: rows[n], Bytes: int64(len(c)), SHA256: hex.EncodeToString(sum[:])})
	}
	b, _ := json.Marshal(man)
	if err := os.WriteFile(filepath.Join(dir, "manifest.json"), b, 0o600); err != nil {
		t.Fatal(err)
	}
}

func runVerifyExport(path string) (int, string) {
	var out, errb bytes.Buffer
	code := cmdVerifyExport(context.Background(), []string{path}, &out, &errb)
	return code, out.String() + errb.String()
}

// DAT-040: checksum or row count mismatch fails export verification.
func TestVerifyExport(t *testing.T) {
	dir := t.TempDir()
	writeExportPkg(t, dir)
	if code, out := runVerifyExport(dir); code != exitOK {
		t.Fatalf("clean package: %d %s", code, out)
	}
	// zip form
	zp := filepath.Join(t.TempDir(), "e.zip")
	var zb bytes.Buffer
	zw := zip.NewWriter(&zb)
	ents, _ := os.ReadDir(dir)
	for _, e := range ents {
		w, _ := zw.Create(e.Name())
		b, _ := os.ReadFile(filepath.Join(dir, e.Name()))
		_, _ = w.Write(b)
	}
	_ = zw.Close()
	if err := os.WriteFile(zp, zb.Bytes(), 0o600); err != nil {
		t.Fatal(err)
	}
	if code, out := runVerifyExport(zp); code != exitOK {
		t.Fatalf("zip package: %d %s", code, out)
	}
	// tamper: same length, different content
	if err := os.WriteFile(filepath.Join(dir, "entries.csv"), []byte("id,amount\ne1,5\ne2,-9\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if code, out := runVerifyExport(dir); code != exitFailure || !strings.Contains(out, "checksum mismatch") {
		t.Fatalf("tampered: %d %s", code, out)
	}
	// extra unlisted file
	writeExportPkg(t, dir)
	if err := os.WriteFile(filepath.Join(dir, "stray.csv"), []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	if code, out := runVerifyExport(dir); code != exitFailure || !strings.Contains(out, "not listed") {
		t.Fatalf("stray: %d %s", code, out)
	}
	// missing manifest
	if code, _ := runVerifyExport(t.TempDir()); code != exitFailure {
		t.Fatalf("no manifest: %d", code)
	}
}
