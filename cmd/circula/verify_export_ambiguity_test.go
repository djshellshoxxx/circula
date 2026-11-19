package main

import (
	"archive/zip"
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/djshellshoxxx/circula/internal/service"
)

// DAT-040: a verified package must identify one unambiguous member per name.
// ZIP readers disagree about which duplicate member they select, so matching
// only the last occurrence cannot establish the integrity of the whole package.
func TestVerifyExportRejectsDuplicateZIPMembers(t *testing.T) {
	for _, name := range []string{"entries.csv", "manifest.json"} {
		for _, identical := range []bool{false, true} {
			label := name + "/conflicting"
			if identical {
				label = name + "/identical"
			}
			t.Run(label, func(t *testing.T) {
				dir := t.TempDir()
				writeExportPkg(t, dir)
				duplicate := []byte("unverified duplicate\n")
				if identical {
					var err error
					duplicate, err = os.ReadFile(filepath.Join(dir, name))
					if err != nil {
						t.Fatal(err)
					}
				}
				path := writeAmbiguousExportZIP(t, dir, name, duplicate)
				if code, out := runVerifyExport(path); code != exitFailure || !strings.Contains(out, "duplicate") {
					t.Fatalf("duplicate member must fail verification: code=%d output=%s", code, out)
				}
			})
		}
	}
}

func writeAmbiguousExportZIP(t *testing.T, dir, duplicateName string, duplicate []byte) string {
	t.Helper()
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	write := func(name string, data []byte) {
		t.Helper()
		w, err := zw.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := w.Write(data); err != nil {
			t.Fatal(err)
		}
	}
	// The original verifier selected the last occurrence, allowing this
	// earlier, potentially conflicting content to escape verification.
	write(duplicateName, duplicate)
	ents, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range ents {
		data, err := os.ReadFile(filepath.Join(dir, e.Name()))
		if err != nil {
			t.Fatal(err)
		}
		write(e.Name(), data)
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "duplicate.zip")
	if err := os.WriteFile(path, buf.Bytes(), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestVerifyExportRejectsDuplicateManifestDeclarations(t *testing.T) {
	dir := t.TempDir()
	writeExportPkg(t, dir)
	path := filepath.Join(dir, "manifest.json")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var man service.Manifest
	if err := json.Unmarshal(data, &man); err != nil {
		t.Fatal(err)
	}
	man.Files = append(man.Files, man.Files[0])
	data, err = json.Marshal(man)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatal(err)
	}
	if code, out := runVerifyExport(dir); code != exitFailure || !strings.Contains(out, "duplicate") {
		t.Fatalf("duplicate declaration must fail verification: code=%d output=%s", code, out)
	}
}

func TestVerifyExportRejectsUnsafeZIPAndManifestNames(t *testing.T) {
	for _, name := range []string{"", ".", "..", "nested/entries.csv", `nested\entries.csv`, "C:entries.csv", "entries\x00.csv"} {
		t.Run(name, func(t *testing.T) {
			path := writeSingleMemberExportZIP(t, name, 0)
			if code, out := runVerifyExport(path); code != exitFailure {
				t.Fatalf("unsafe member name must fail verification: code=%d output=%s", code, out)
			}
		})
	}
}

func TestVerifyExportRejectsNonRegularZIPMembers(t *testing.T) {
	for _, mode := range []os.FileMode{os.ModeSymlink | 0o600, os.ModeDir | 0o700, os.ModeNamedPipe | 0o600} {
		t.Run(mode.String(), func(t *testing.T) {
			path := writeSingleMemberExportZIP(t, "entries.csv", mode)
			if code, out := runVerifyExport(path); code != exitFailure || !strings.Contains(out, "regular file") {
				t.Fatalf("nonregular member must fail verification: code=%d output=%s", code, out)
			}
		})
	}
}

// The manifest and CSV are otherwise valid. This makes successful validation
// depend on detecting the member name/type, rather than a checksum mismatch.
func writeSingleMemberExportZIP(t *testing.T, name string, mode os.FileMode) string {
	t.Helper()
	data := []byte("id,amount\ne1,5\ne2,-5\n")
	sum := sha256.Sum256(data)
	man := service.Manifest{
		SchemaVersion: 1,
		Scope:         service.ExportFull,
		LedgerSeq:     9,
		Files: []service.ManifestFile{{
			Name: name, Rows: 2, Bytes: int64(len(data)), SHA256: hex.EncodeToString(sum[:]),
		}},
	}
	manifest, err := json.Marshal(man)
	if err != nil {
		t.Fatal(err)
	}
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	h := &zip.FileHeader{Name: name}
	if mode != 0 {
		h.SetMode(mode)
	}
	w, err := zw.CreateHeader(h)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := w.Write(data); err != nil {
		t.Fatal(err)
	}
	w, err = zw.Create("manifest.json")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := w.Write(manifest); err != nil {
		t.Fatal(err)
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "member.zip")
	if err := os.WriteFile(path, buf.Bytes(), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestVerifyExportRejectsNonRegularDirectoryMembers(t *testing.T) {
	t.Run("directory", func(t *testing.T) {
		dir := t.TempDir()
		writeExportPkg(t, dir)
		if err := os.Mkdir(filepath.Join(dir, "unverified"), 0o700); err != nil {
			t.Fatal(err)
		}
		if code, out := runVerifyExport(dir); code != exitFailure || !strings.Contains(out, "regular file") {
			t.Fatalf("directory member must fail verification: code=%d output=%s", code, out)
		}
	})
	t.Run("symlink", func(t *testing.T) {
		dir := t.TempDir()
		writeExportPkg(t, dir)
		member := filepath.Join(dir, "entries.csv")
		target := filepath.Join(t.TempDir(), "external.csv")
		if err := os.Rename(member, target); err != nil {
			t.Fatal(err)
		}
		if err := os.Symlink(target, member); err != nil {
			t.Skipf("creating a symlink is unavailable on this host: %v", err)
		}
		if code, out := runVerifyExport(dir); code != exitFailure || !strings.Contains(out, "regular file") {
			t.Fatalf("symlink member must fail verification: code=%d output=%s", code, out)
		}
	})
}
