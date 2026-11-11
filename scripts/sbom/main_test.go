package main

import (
	"bytes"
	"crypto/sha256"
	"debug/buildinfo"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"runtime/debug"
	"strings"
	"testing"
)

func sampleInfo() *debug.BuildInfo {
	return &debug.BuildInfo{
		GoVersion: "go1.26.10",
		Path:      "github.com/djshellshoxxx/circula/cmd/circula",
		Main:      debug.Module{Path: "github.com/djshellshoxxx/circula", Version: "(devel)"},
		Deps: []*debug.Module{
			{Path: "modernc.org/sqlite", Version: "v1.60.1", Sum: "h1:/blz53O951KWFOso4QQvEs/Fq6cDBKLtMVrYNSeJVKw="},
			{Path: "github.com/BurntSushi/Toml", Version: "v1.0.0+incompatible", Sum: "h1:AAAA"},
			{Path: "golang.org/x/sys", Version: "v0.49.0", Sum: "h1:XbzkgYJHdqh/8m2Uu0W/dQv8nktxx4BFHp1M0gROTXA="},
			{Path: "example.org/replaced", Version: "v1.0.0", Replace: &debug.Module{Path: "example.org/fork", Version: "v1.0.1", Sum: "h1:BBBB"}},
		},
		Settings: []debug.BuildSetting{
			{Key: "GOOS", Value: "linux"}, {Key: "GOARCH", Value: "arm64"}, {Key: "CGO_ENABLED", Value: "0"},
			{Key: "-trimpath", Value: "true"}, {Key: "-ldflags", Value: "-s -w -X main.version=v1.2.3"},
			{Key: "vcs.revision", Value: "abc"}, {Key: "vcs.time", Value: "2026-01-02T03:04:05Z"},
		},
	}
}

// checkWellFormed fails the test when validate rejects the bill.
func checkWellFormed(t *testing.T, b *bom) {
	t.Helper()
	if err := validate(b); err != nil {
		t.Fatal(err)
	}
}

func find(t *testing.T, b *bom, name string) component {
	t.Helper()
	for _, c := range b.Components {
		if c.Name == name {
			return c
		}
	}
	t.Fatalf("component %q not in bill", name)
	return component{}
}

func TestBuildListsEveryLinkedModule(t *testing.T) {
	b, err := build(sampleInfo(), options{Version: "v1.2.3", BinarySHA: strings.Repeat("ab", 32)})
	if err != nil {
		t.Fatal(err)
	}
	checkWellFormed(t, b)
	root := b.Metadata.Component
	if root.Name != "github.com/djshellshoxxx/circula" || root.Version != "v1.2.3" || root.Type != "application" {
		t.Fatalf("root = %+v", root)
	}
	if want := "pkg:golang/github.com/djshellshoxxx/circula@v1.2.3?goarch=arm64&goos=linux&type=module"; root.PURL != want {
		t.Fatalf("root purl = %q, want %q", root.PURL, want)
	}
	if len(root.Hashes) != 1 || root.Hashes[0].Alg != "SHA-256" || root.Hashes[0].Content != strings.Repeat("ab", 32) {
		t.Fatalf("root hashes = %+v", root.Hashes)
	}
	if prop(root, "circula:go:build:GOOS") != "linux" || prop(root, "circula:go:build:CGO_ENABLED") != "0" ||
		prop(root, "circula:go:build:vcs.revision") != "abc" || prop(root, "circula:go:toolchain") != "go1.26.10" {
		t.Fatalf("root build properties = %+v", root.Properties)
	}
	if prop(root, "circula:go:build:-ldflags") != "" {
		t.Fatal("ldflags must not be copied into the bill")
	}
	if b.Metadata.Timestamp != "2026-01-02T03:04:05Z" {
		t.Fatalf("timestamp = %q, want vcs.time", b.Metadata.Timestamp)
	}
	if len(b.Components) != 5 { // std + 4 modules
		t.Fatalf("components = %d, want 5", len(b.Components))
	}
	sq := find(t, b, "modernc.org/sqlite")
	if sq.PURL != "pkg:golang/modernc.org/sqlite@v1.60.1?type=module" || prop(sq, "circula:go:sum") != "h1:/blz53O951KWFOso4QQvEs/Fq6cDBKLtMVrYNSeJVKw=" {
		t.Fatalf("sqlite = %+v", sq)
	}
	toml := find(t, b, "github.com/BurntSushi/Toml")
	if toml.PURL != "pkg:golang/github.com/BurntSushi/Toml@v1.0.0%2Bincompatible?type=module" {
		t.Fatalf("purl escaping: %q", toml.PURL)
	}
	rep := find(t, b, "example.org/replaced")
	if rep.Version != "v1.0.1" || rep.PURL != "pkg:golang/example.org/fork@v1.0.1?type=module" ||
		prop(rep, "circula:go:replaced-by") != "example.org/fork v1.0.1" || prop(rep, "circula:go:sum") != "h1:BBBB" {
		t.Fatalf("replaced module = %+v", rep)
	}
	std := find(t, b, "std")
	if std.Version != "go1.26.10" {
		t.Fatalf("std = %+v", std)
	}
	// Without a graph the root depends on every component.
	if got := len(b.Dependencies[0].DependsOn); b.Dependencies[0].Ref != root.BOMRef || got != 5 {
		t.Fatalf("root dependsOn = %d entries, want 5", got)
	}
}

func TestBuildIsDeterministic(t *testing.T) {
	opt := options{Version: "v1.2.3", BinarySHA: "00"}
	a, _ := build(sampleInfo(), opt)
	b, _ := build(sampleInfo(), opt)
	ja, _ := json.Marshal(a)
	jb, _ := json.Marshal(b)
	if !bytes.Equal(ja, jb) {
		t.Fatal("two builds from the same inputs differ")
	}
	c, _ := build(sampleInfo(), options{Version: "v1.2.3", BinarySHA: "01"})
	if c.SerialNumber == a.SerialNumber {
		t.Fatal("a different binary must get a different serial number")
	}
}

func TestBuildUsesGraphEdgesBetweenLinkedModules(t *testing.T) {
	graph := [][2]string{
		{"github.com/djshellshoxxx/circula", "modernc.org/sqlite@v1.60.1"},
		{"modernc.org/sqlite@v1.60.1", "golang.org/x/sys@v0.40.0"},                       // required version differs; selected is linked
		{"modernc.org/sqlite@v1.50.0", "github.com/BurntSushi/Toml@v1.0.0+incompatible"}, // not the selected sqlite
		{"modernc.org/sqlite@v1.60.1", "example.org/not-linked@v1.0.0"},
	}
	b, err := build(sampleInfo(), options{Version: "v1.2.3", Graph: graph})
	if err != nil {
		t.Fatal(err)
	}
	checkWellFormed(t, b)
	deps := map[string][]string{}
	for _, d := range b.Dependencies {
		deps[d.Ref] = d.DependsOn
	}
	sq := find(t, b, "modernc.org/sqlite").BOMRef
	sys := find(t, b, "golang.org/x/sys").BOMRef
	if got := deps[sq]; len(got) != 1 || got[0] != sys {
		t.Fatalf("sqlite dependsOn = %v, want [%s]", got, sys)
	}
	root := deps[b.Metadata.Component.BOMRef]
	has := func(list []string, ref string) bool {
		for _, r := range list {
			if r == ref {
				return true
			}
		}
		return false
	}
	if !has(root, sq) || has(root, sys) {
		t.Fatalf("root dependsOn = %v: want sqlite (direct) and not x/sys (reached through sqlite)", root)
	}
	// Modules the graph does not reach are attached to the root rather than left orphaned.
	if !has(root, find(t, b, "github.com/BurntSushi/Toml").BOMRef) {
		t.Fatalf("unreached module not attached to root: %v", root)
	}
}

func TestZipHashesReadTheModuleCache(t *testing.T) {
	mc := t.TempDir()
	content := []byte("zip bytes")
	dir := filepath.Join(mc, "cache", "download", "github.com", "!burnt!sushi", "!toml", "@v")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "v1.0.0+incompatible.zip"), content, 0o644); err != nil {
		t.Fatal(err)
	}
	deps := []*debug.Module{{Path: "github.com/BurntSushi/Toml", Version: "v1.0.0+incompatible"}, {Path: "example.org/local", Replace: &debug.Module{Path: "../local"}}}
	got, err := zipHashes(mc, deps)
	if err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(content)
	if got["github.com/BurntSushi/Toml@v1.0.0+incompatible"] != hex.EncodeToString(sum[:]) || len(got) != 1 {
		t.Fatalf("hashes = %v", got)
	}
	if _, err := zipHashes(mc, []*debug.Module{{Path: "example.org/missing", Version: "v1.0.0"}}); err == nil {
		t.Fatal("a missing module zip must be an error, not a bill without hashes")
	}
}

func TestReadLicensesFromManifest(t *testing.T) {
	got, err := readLicenses(filepath.Join("..", "..", "docs", "legal", "DEPENDENCIES.md"))
	if err != nil {
		t.Fatal(err)
	}
	if len(got) == 0 {
		t.Fatal("no rows parsed from docs/legal/DEPENDENCIES.md")
	}
	for k, v := range got {
		if !strings.Contains(k, "@v") || v == "" {
			t.Fatalf("bad row %q -> %q", k, v)
		}
	}
	b, _ := build(&debug.BuildInfo{Main: debug.Module{Path: "m"}, Deps: []*debug.Module{{Path: "a", Version: "v1.0.0"}, {Path: "b", Version: "v1.0.0"}}},
		options{Timestamp: "2026-01-02T03:04:05Z", Licenses: map[string]string{"a@v1.0.0": "MIT", "b@v1.0.0": "UNKNOWN"}})
	if l := find(t, b, "a").Licenses; len(l) != 1 || l[0].License.ID != "MIT" {
		t.Fatalf("licence of a = %+v", l)
	}
	if l := find(t, b, "b").Licenses; len(l) != 0 {
		t.Fatalf("UNKNOWN must not be written as a licence id: %+v", l)
	}
}

func TestRunOnARealBinary(t *testing.T) {
	exe, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	out := filepath.Join(t.TempDir(), "x.cdx.json")
	if err := run([]string{"-bin", exe, "-version", "v0.0.0-test", "-timestamp", "2026-01-02T03:04:05+01:00", "-o", out}, nil); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(out)
	if err != nil {
		t.Fatal(err)
	}
	var b bom
	if err := json.Unmarshal(raw, &b); err != nil {
		t.Fatal(err)
	}
	checkWellFormed(t, &b)
	want, err := fileSHA256(exe)
	if err != nil {
		t.Fatal(err)
	}
	if h := b.Metadata.Component.Hashes; len(h) != 1 || h[0].Content != want {
		t.Fatalf("binary hash = %+v, want %s", h, want)
	}
	if b.Metadata.Timestamp != "2026-01-02T02:04:05Z" {
		t.Fatalf("timestamp not normalised to UTC: %q", b.Metadata.Timestamp)
	}
	if bytes.Contains(raw, []byte{'\\', 'u', '0', '0', '2', '6'}) || !bytes.Contains(raw, []byte("goarch=")) {
		t.Fatal("purl qualifiers must keep a literal '&', not an HTML escape")
	}
	if err := run([]string{"-bin", filepath.Join(t.TempDir(), "missing")}, nil); err == nil {
		t.Fatal("missing binary must fail")
	}
	if err := run([]string{"-bin", exe, "-timestamp", "yesterday"}, &bytes.Buffer{}); err == nil {
		t.Fatal("bad timestamp must fail")
	}
}

func TestVerifyMatchesBillToBinary(t *testing.T) {
	exe, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	good := filepath.Join(dir, "good.cdx.json")
	if err := run([]string{"-bin", exe, "-version", "v1.2.3", "-o", good}, nil); err != nil {
		t.Fatal(err)
	}
	if err := run([]string{"-bin", exe, "-verify", good}, nil); err != nil {
		t.Fatalf("a freshly written bill must verify: %v", err)
	}
	raw, err := os.ReadFile(good)
	if err != nil {
		t.Fatal(err)
	}
	write := func(name string, mutate func(*bom)) string {
		t.Helper()
		var c bom
		if err := json.Unmarshal(raw, &c); err != nil {
			t.Fatal(err)
		}
		mutate(&c)
		out, _ := json.Marshal(&c)
		p := filepath.Join(dir, name)
		if err := os.WriteFile(p, out, 0o644); err != nil {
			t.Fatal(err)
		}
		return p
	}
	cases := map[string]func(*bom){
		"hash": func(c *bom) { c.Metadata.Component.Hashes[0].Content = strings.Repeat("0", 64) },
		"extra": func(c *bom) {
			c.Components = append(c.Components, component{Type: "library", Name: "example.org/x", Version: "v1.0.0", BOMRef: "x"})
			c.Dependencies = append(c.Dependencies, dependency{Ref: "x", DependsOn: []string{}})
		},
		"target": func(c *bom) {
			c.Metadata.Component.PURL = strings.Replace(c.Metadata.Component.PURL, "goos=", "goos=x", 1)
		},
		"spec":     func(c *bom) { c.SpecVersion = "1.4" },
		"dangling": func(c *bom) { c.Dependencies[0].DependsOn = append(c.Dependencies[0].DependsOn, "nowhere") },
	}
	for name, mutate := range cases {
		if err := run([]string{"-bin", exe, "-verify", write(name+".json", mutate)}, nil); err == nil {
			t.Errorf("%s: a bill that does not match the binary must fail verification", name)
		}
	}
	// A linked module missing from the bill: the binary's build information plus modules the bill does not list.
	bi, err := buildinfo.ReadFile(exe)
	if err != nil {
		t.Fatal(err)
	}
	bi.Deps = append(bi.Deps, sampleInfo().Deps...)
	if err := verify(good, exe, bi); err == nil || !strings.Contains(err.Error(), "missing from the bill") {
		t.Fatalf("missing module: err = %v", err)
	}
}
