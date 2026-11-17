package pwa

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"io/fs"
	"strconv"
	"strings"
	"testing"
	"testing/fstest"
)

var testConfig = Config{Enabled: true, PublicOrigin: "https://c.test", BasePath: "/app/"}

// realFS copies the embedded build into a mutable in-memory filesystem.
func realFS(t *testing.T) fstest.MapFS {
	t.Helper()
	m := fstest.MapFS{}
	entries, err := fs.ReadDir(files, "public")
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		data, err := fs.ReadFile(files, "public/"+e.Name())
		if err != nil {
			t.Fatal(err)
		}
		m["public/"+e.Name()] = &fstest.MapFile{Data: data}
	}
	return m
}

func catalogOf(t *testing.T, m fstest.MapFS) catalog {
	t.Helper()
	var c catalog
	if err := json.Unmarshal(m["public/catalog.json"].Data, &c); err != nil {
		t.Fatal(err)
	}
	return c
}

func putCatalog(t *testing.T, m fstest.MapFS, c catalog) {
	t.Helper()
	raw, err := json.Marshal(c)
	if err != nil {
		t.Fatal(err)
	}
	m["public/catalog.json"] = &fstest.MapFile{Data: raw}
}

func fingerprinted(name, ext string, data []byte) (string, asset) {
	h := sha256.Sum256(data)
	file := name + "." + hex.EncodeToString(h[:8]) + ext
	types := map[string]string{".css": "text/css", ".js": "application/javascript", ".png": "image/png"}
	return file, asset{File: file, SHA256: base64.StdEncoding.EncodeToString(h[:]), Type: types[ext], Size: len(data)}
}

func TestBuiltCatalogRecordsExactSizes(t *testing.T) {
	m := realFS(t)
	c := catalogOf(t, m)
	if len(c.Assets) != 4 {
		t.Fatal(len(c.Assets))
	}
	for _, a := range c.Assets {
		if a.Size <= 0 || a.Size != len(m["public/"+a.File].Data) || a.Size > MaxAssetBytes {
			t.Fatalf("%s size %d", a.File, a.Size)
		}
	}
	if _, err := newBundle(testConfig, m, "public/"); err != nil {
		t.Fatal(err)
	}
}

func TestWorkerConfigCarriesBoundedSizes(t *testing.T) {
	b, err := New(testConfig)
	if err != nil {
		t.Fatal(err)
	}
	w := string(b.worker)
	for _, a := range b.catalog.Assets {
		if !strings.Contains(w, `"size":`+itoa(a.Size)) {
			t.Fatalf("worker config lacks size for %s", a.File)
		}
	}
}

func itoa(n int) string { return strconv.Itoa(n) }

func TestCatalogRejectsHostileMetadata(t *testing.T) {
	mutations := map[string]func(*testing.T, fstest.MapFS, *catalog){
		"size understated": func(_ *testing.T, _ fstest.MapFS, c *catalog) { c.Assets[0].Size-- },
		"size overstated":  func(_ *testing.T, _ fstest.MapFS, c *catalog) { c.Assets[0].Size++ },
		"size missing":     func(_ *testing.T, _ fstest.MapFS, c *catalog) { c.Assets[0].Size = 0 },
		"size negative":    func(_ *testing.T, _ fstest.MapFS, c *catalog) { c.Assets[0].Size = -1 },
		"wrong mime":       func(_ *testing.T, _ fstest.MapFS, c *catalog) { c.Assets[0].Type = "text/html" },
		"empty mime":       func(_ *testing.T, _ fstest.MapFS, c *catalog) { c.Assets[0].Type = "" },
		"hash mismatch":    func(_ *testing.T, _ fstest.MapFS, c *catalog) { c.Assets[0].SHA256 = c.Assets[1].SHA256 },
		"duplicate asset":  func(_ *testing.T, _ fstest.MapFS, c *catalog) { c.Assets = append(c.Assets, c.Assets[0]) },
		"empty catalog":    func(_ *testing.T, _ fstest.MapFS, c *catalog) { c.Assets = nil },
		"bad version":      func(_ *testing.T, _ fstest.MapFS, c *catalog) { c.Version = `x","evil":"` },
		"empty version":    func(_ *testing.T, _ fstest.MapFS, c *catalog) { c.Version = "" },
		"path traversal":   func(_ *testing.T, _ fstest.MapFS, c *catalog) { c.Assets[0].File = "../" + c.Assets[0].File },
		"subdirectory":     func(_ *testing.T, _ fstest.MapFS, c *catalog) { c.Assets[0].File = "x/" + c.Assets[0].File },
		"unfingerprinted":  func(_ *testing.T, _ fstest.MapFS, c *catalog) { c.Assets[0].File = "style.css" },
		"unknown ext": func(t *testing.T, m fstest.MapFS, c *catalog) {
			file, a := fingerprinted("x", ".html", []byte("<p>"))
			_ = file
			c.Assets[0] = a
		},
		"too many assets": func(t *testing.T, m fstest.MapFS, c *catalog) {
			for i := 0; i <= MaxAssets; i++ {
				data := []byte(strings.Repeat("a", i+1))
				file, a := fingerprinted("extra", ".css", data)
				m["public/"+file] = &fstest.MapFile{Data: data}
				c.Assets = append(c.Assets, a)
			}
		},
		"oversize asset": func(t *testing.T, m fstest.MapFS, c *catalog) {
			data := make([]byte, MaxAssetBytes+1)
			file, a := fingerprinted("big", ".css", data)
			m["public/"+file] = &fstest.MapFile{Data: data}
			c.Assets[0] = a
		},
	}
	for name, mutate := range mutations {
		t.Run(name, func(t *testing.T) {
			m := realFS(t)
			c := catalogOf(t, m)
			mutate(t, m, &c)
			putCatalog(t, m, c)
			if _, err := newBundle(testConfig, m, "public/"); err == nil {
				t.Fatal("accepted hostile catalog")
			}
		})
	}
}

func TestCatalogAtExactBoundaryIsAccepted(t *testing.T) {
	m := realFS(t)
	c := catalogOf(t, m)
	data := make([]byte, MaxAssetBytes)
	file, a := fingerprinted("edge", ".css", data)
	m["public/"+file] = &fstest.MapFile{Data: data}
	c.Assets[0] = a
	c.Assets[0].File = file
	putCatalog(t, m, c)
	b, err := newBundle(testConfig, m, "public/")
	if err == nil && b.style == "" {
		t.Fatal("style asset replaced without detection")
	}
	// Replacing the stylesheet removes the style.* asset, so the build is incomplete; the size check itself must pass.
	if err == nil {
		t.Fatal("expected incomplete build because style prefix is gone")
	}
	if !strings.Contains(err.Error(), "incomplete") {
		t.Fatalf("size boundary rejected unexpectedly: %v", err)
	}
}

func TestMissingBuildFilesFailClosed(t *testing.T) {
	for _, name := range []string{"public/catalog.json", "public/worker.js"} {
		m := realFS(t)
		delete(m, name)
		if _, err := newBundle(testConfig, m, "public/"); err == nil {
			t.Fatal(name)
		}
	}
	m := realFS(t)
	m["public/worker.js"] = &fstest.MapFile{Data: []byte("no placeholder")}
	if _, err := newBundle(testConfig, m, "public/"); err == nil {
		t.Fatal("worker without config placeholder accepted")
	}
	m = realFS(t)
	m["public/catalog.json"] = &fstest.MapFile{Data: []byte("{not json")}
	if _, err := newBundle(testConfig, m, "public/"); err == nil {
		t.Fatal("malformed catalog accepted")
	}
}
