package main

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/djshellshoxxx/circula/internal/service"
)

func TestVerifyExportRequiresSupportedSchema(t *testing.T) {
	for _, data := range []string{
		"null",
		"{}",
		`{"schema_version":0,"files":[]}`,
		`{"schema_version":2,"files":[]}`,
		`{"schema_version":1}`,
		`{"schema_version":1,"files":null}`,
		`{"schema_version":true,"files":[]}`,
		`{"schema_version":"1","files":[]}`,
	} {
		t.Run(data, func(t *testing.T) {
			dir := t.TempDir()
			if err := os.WriteFile(filepath.Join(dir, "manifest.json"), []byte(data), 0o600); err != nil {
				t.Fatal(err)
			}
			if code, out := runVerifyExport(dir); code != exitFailure {
				t.Fatalf("invalid schema accepted: code=%d output=%s", code, out)
			}
		})
	}
	t.Run("supported empty package", func(t *testing.T) {
		dir := t.TempDir()
		man := service.Manifest{SchemaVersion: service.ExportSchemaVersion, Files: []service.ManifestFile{}}
		data, err := json.Marshal(man)
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, "manifest.json"), data, 0o600); err != nil {
			t.Fatal(err)
		}
		if code, out := runVerifyExport(dir); code != exitOK {
			t.Fatalf("supported empty package: code=%d output=%s", code, out)
		}
	})
}

// ZIP metadata alone can advertise an enormous member. Refuse it before
// opening its body; this fixture allocates no large data buffer or file.
func TestOpenExportRejectsAdvertisedOversizeZIPMember(t *testing.T) {
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	h := &zip.FileHeader{Name: "entries.csv", Method: zip.Store, UncompressedSize64: maxExportFile + 1}
	if _, err := zw.CreateRaw(h); err != nil {
		t.Fatal(err)
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "oversize.zip")
	if err := os.WriteFile(path, buf.Bytes(), 0o600); err != nil {
		t.Fatal(err)
	}
	r, err := openExport(path)
	if r != nil && r.close != nil {
		defer r.close()
	}
	if err == nil {
		t.Fatal("oversized ZIP member metadata was accepted")
	}
}

func TestOpenExportRejectsDOSDirectoryAttributes(t *testing.T) {
	for _, creator := range []uint16{0, 3, 10, 14, 19} {
		var buf bytes.Buffer
		zw := zip.NewWriter(&buf)
		h := &zip.FileHeader{Name: "entries.csv", Method: zip.Store}
		h.SetMode(0o600)
		h.CreatorVersion = h.CreatorVersion&0xff | creator<<8
		h.ExternalAttrs |= 0x10
		if _, err := zw.CreateRaw(h); err != nil {
			t.Fatal(err)
		}
		if err := zw.Close(); err != nil {
			t.Fatal(err)
		}
		path := filepath.Join(t.TempDir(), "directory-attributes.zip")
		if err := os.WriteFile(path, buf.Bytes(), 0o600); err != nil {
			t.Fatal(err)
		}
		r, err := openExport(path)
		if r != nil && r.close != nil {
			r.close()
		}
		if err == nil {
			t.Errorf("creator %d: DOS directory metadata was accepted as a file", creator)
		}
	}
}

type countedExportReader struct {
	reader io.Reader
	read   int
}

func (r *countedExportReader) Read(p []byte) (int, error) {
	n, err := r.reader.Read(p)
	r.read += n
	return n, err
}

func TestReadExportLimited(t *testing.T) {
	for _, tc := range []struct {
		name, input, want string
		limit             int64
		fail              bool
	}{
		{name: "empty at zero", limit: 0},
		{name: "nonempty at zero", input: "x", limit: 0, fail: true},
		{name: "below limit", input: "ab", want: "ab", limit: 3},
		{name: "exact limit", input: "abc", want: "abc", limit: 3},
		{name: "above limit", input: "abcd", limit: 3, fail: true},
		{name: "bounded reading", input: strings.Repeat("x", 256), limit: 3, fail: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r := &countedExportReader{reader: strings.NewReader(tc.input)}
			got, err := readExportLimited(r, tc.limit)
			if tc.fail {
				if err == nil || got != nil {
					t.Fatalf("oversize read returned data=%q error=%v", got, err)
				}
			} else if err != nil || string(got) != tc.want {
				t.Fatalf("read=%q error=%v, want %q", got, err, tc.want)
			}
			if int64(r.read) > tc.limit+1 {
				t.Fatalf("read %d bytes with limit %d", r.read, tc.limit)
			}
		})
	}
}

func TestReadExportLimitedRejectsInvalidBounds(t *testing.T) {
	for _, limit := range []int64{-1, maxExportFile + 1} {
		r := &countedExportReader{reader: strings.NewReader("must not be read")}
		got, err := readExportLimited(r, limit)
		if err == nil || got != nil || r.read != 0 {
			t.Fatalf("limit=%d: data=%q error=%v read=%d", limit, got, err, r.read)
		}
	}
}

type failingExportReader struct{ err error }

func (r failingExportReader) Read(p []byte) (int, error) { return copy(p, "x"), r.err }

func TestReadExportLimitedDiscardsPartialDataOnError(t *testing.T) {
	failure := errors.New("synthetic read failure")
	got, err := readExportLimited(failingExportReader{err: failure}, 3)
	if got != nil || !errors.Is(err, failure) {
		t.Fatalf("read error retained data=%q or lost cause: %v", got, err)
	}
}
