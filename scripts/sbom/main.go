// Command sbom writes a CycloneDX 1.5 JSON software bill of materials for one built circula binary (SEC-029,
// OPS-035). The module list comes from the build information the Go linker embeds in every binary (the data
// `go version -m` prints), so the bill names exactly the modules compiled into that target, with the go.sum
// hash each was built from. Standard library only; the output is deterministic for the same inputs.
//
//	go run ./scripts/sbom -bin stage/circula -version v1.2.3 -timestamp 2026-01-02T03:04:05Z \
//	    -modcache "$(go env GOMODCACHE)" -graph modgraph.txt -licenses docs/legal/DEPENDENCIES.md -o x.cdx.json
//
// -modcache adds the SHA-256 of each module zip from the download cache (a missing zip is an error),
// -graph (output of `go mod graph`) adds module-to-module dependency edges, -licenses (the ARC-024 manifest)
// adds SPDX licence identifiers. Without -graph the root component depends on every module directly.
package main

import (
	"bufio"
	"bytes"
	"crypto/sha1" //nolint:gosec // RFC 4122 version 5 UUIDs are defined over SHA-1; not a security use
	"crypto/sha256"
	"debug/buildinfo"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"runtime/debug"
	"sort"
	"strings"
	"time"
)

// CycloneDX 1.5 JSON shapes (only the fields this tool writes).
type bom struct {
	BOMFormat    string       `json:"bomFormat"`
	SpecVersion  string       `json:"specVersion"`
	SerialNumber string       `json:"serialNumber"`
	Version      int          `json:"version"`
	Metadata     metadata     `json:"metadata"`
	Components   []component  `json:"components"`
	Dependencies []dependency `json:"dependencies"`
}

type metadata struct {
	Timestamp string    `json:"timestamp"`
	Tools     tools     `json:"tools"`
	Component component `json:"component"`
}

type tools struct {
	Components []component `json:"components"`
}

type component struct {
	Type       string     `json:"type"`
	BOMRef     string     `json:"bom-ref,omitempty"`
	Name       string     `json:"name"`
	Version    string     `json:"version,omitempty"`
	PURL       string     `json:"purl,omitempty"`
	Hashes     []hash     `json:"hashes,omitempty"`
	Licenses   []license  `json:"licenses,omitempty"`
	Properties []property `json:"properties,omitempty"`
}

type hash struct {
	Alg     string `json:"alg"`
	Content string `json:"content"`
}

type license struct {
	License licenseID `json:"license"`
}

type licenseID struct {
	ID string `json:"id"`
}

type property struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

type dependency struct {
	Ref       string   `json:"ref"`
	DependsOn []string `json:"dependsOn"`
}

// options are the inputs besides the build information.
type options struct {
	Version   string            // release version of the root component; empty uses the embedded module version
	Timestamp string            // RFC 3339; empty uses vcs.time from the build information
	BinarySHA string            // hex SHA-256 of the binary file
	ZipSHA    map[string]string // "path@version" -> hex SHA-256 of the module zip
	Graph     [][2]string       // `go mod graph` edges, "path[@version]" pairs
	Licenses  map[string]string // "path@version" -> SPDX identifier
}

func main() {
	if err := run(os.Args[1:], os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, "sbom:", err)
		os.Exit(1)
	}
}

func run(args []string, stdout io.Writer) error {
	fs := flag.NewFlagSet("sbom", flag.ContinueOnError)
	bin := fs.String("bin", "", "built binary to describe (required)")
	version := fs.String("version", "", "release version of the root component")
	timestamp := fs.String("timestamp", "", "RFC 3339 timestamp for metadata (default: vcs.time of the build)")
	modcache := fs.String("modcache", "", "Go module cache (go env GOMODCACHE); adds module zip SHA-256 hashes")
	graphFile := fs.String("graph", "", "file holding `go mod graph` output; adds dependency edges")
	licFile := fs.String("licenses", "", "ARC-024 licence manifest (docs/legal/DEPENDENCIES.md); adds SPDX ids")
	out := fs.String("o", "", "output file (default stdout)")
	verifyFile := fs.String("verify", "", "check an existing bill against -bin instead of writing one")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *bin == "" {
		return errors.New("-bin is required")
	}
	bi, err := buildinfo.ReadFile(*bin)
	if err != nil {
		return fmt.Errorf("read build information from %s: %w", *bin, err)
	}
	if *verifyFile != "" {
		return verify(*verifyFile, *bin, bi)
	}
	opt := options{Version: *version, Timestamp: *timestamp}
	if opt.BinarySHA, err = fileSHA256(*bin); err != nil {
		return err
	}
	if *modcache != "" {
		if opt.ZipSHA, err = zipHashes(*modcache, bi.Deps); err != nil {
			return err
		}
	}
	if *graphFile != "" {
		if opt.Graph, err = readGraph(*graphFile); err != nil {
			return err
		}
	}
	if *licFile != "" {
		if opt.Licenses, err = readLicenses(*licFile); err != nil {
			return err
		}
	}
	doc, err := build(bi, opt)
	if err != nil {
		return err
	}
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false) // purl qualifiers keep their '&'
	enc.SetIndent("", "  ")
	if err := enc.Encode(doc); err != nil {
		return err
	}
	if *out == "" {
		_, err = stdout.Write(buf.Bytes())
		return err
	}
	return os.WriteFile(*out, buf.Bytes(), 0o644)
}

// effective returns the module that was actually compiled in place of m.
func effective(m *debug.Module) *debug.Module {
	if m.Replace != nil {
		return m.Replace
	}
	return m
}

func modKey(path, version string) string { return path + "@" + version }

func build(bi *debug.BuildInfo, opt options) (*bom, error) {
	if bi == nil || bi.Main.Path == "" {
		return nil, errors.New("binary carries no main module information")
	}
	settings := map[string]string{}
	for _, s := range bi.Settings {
		settings[s.Key] = s.Value
	}
	ts := opt.Timestamp
	if ts == "" {
		ts = settings["vcs.time"]
	}
	if ts == "" {
		ts = time.Now().UTC().Format(time.RFC3339)
	}
	t, err := time.Parse(time.RFC3339, ts)
	if err != nil {
		return nil, fmt.Errorf("timestamp %q is not RFC 3339: %w", ts, err)
	}
	ts = t.UTC().Format(time.RFC3339)

	rootVersion := opt.Version
	if rootVersion == "" {
		rootVersion = bi.Main.Version
	}
	quals := []string{"type=module"}
	if v := settings["GOARCH"]; v != "" {
		quals = append(quals, "goarch="+escape(v))
	}
	if v := settings["GOOS"]; v != "" {
		quals = append(quals, "goos="+escape(v))
	}
	sort.Strings(quals) // purl qualifiers are sorted by key
	root := component{
		Type:    "application",
		Name:    bi.Main.Path,
		Version: rootVersion,
		PURL:    purl(bi.Main.Path, rootVersion, quals...),
	}
	root.BOMRef = root.PURL
	if opt.BinarySHA != "" {
		root.Hashes = []hash{{Alg: "SHA-256", Content: opt.BinarySHA}}
	}
	root.Properties = append(root.Properties, property{"circula:go:package", bi.Path}, property{"circula:go:toolchain", bi.GoVersion})
	keys := make([]string, 0, len(settings))
	for k := range settings {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		if k == "-ldflags" {
			continue // carries the version stamps already shown above; keep the bill independent of flag spelling
		}
		root.Properties = append(root.Properties, property{"circula:go:build:" + k, settings[k]})
	}

	std := component{Type: "library", Name: "std", Version: bi.GoVersion, PURL: purl("std", bi.GoVersion, "type=module")}
	std.BOMRef = std.PURL

	deps := append([]*debug.Module(nil), bi.Deps...)
	sort.Slice(deps, func(i, j int) bool { return deps[i].Path < deps[j].Path })
	comps := []component{std}
	refByPath := map[string]string{} // module path as required in go.mod -> bom-ref
	selected := map[string]string{}  // module path -> selected version
	for _, d := range deps {
		e := effective(d)
		c := component{Type: "library", Name: d.Path, Version: e.Version, PURL: purl(e.Path, e.Version, "type=module")}
		if e.Version == "" { // replaced by a local directory: no module version or checksum exists
			c.Version = "(local)"
			c.PURL = ""
			c.BOMRef = "local:" + e.Path
		} else {
			c.BOMRef = c.PURL
		}
		if h, ok := opt.ZipSHA[modKey(e.Path, e.Version)]; ok {
			c.Hashes = []hash{{Alg: "SHA-256", Content: h}}
		}
		if id, ok := opt.Licenses[modKey(d.Path, d.Version)]; ok && id != "" && id != "UNKNOWN" {
			c.Licenses = []license{{licenseID{id}}}
		}
		if e.Sum != "" {
			c.Properties = append(c.Properties, property{"circula:go:sum", e.Sum})
		}
		if d.Replace != nil {
			c.Properties = append(c.Properties, property{"circula:go:replaced-by", strings.TrimSuffix(e.Path+" "+e.Version, " ")})
		}
		comps = append(comps, c)
		refByPath[d.Path] = c.BOMRef
		selected[d.Path] = d.Version
	}

	edges := map[string]map[string]bool{root.BOMRef: {std.BOMRef: true}}
	add := func(from, to string) {
		if edges[from] == nil {
			edges[from] = map[string]bool{}
		}
		edges[from][to] = true
	}
	if len(opt.Graph) == 0 {
		for _, d := range deps {
			add(root.BOMRef, refByPath[d.Path])
		}
	} else {
		for _, e := range opt.Graph {
			fromPath, fromVer, _ := strings.Cut(e[0], "@")
			toPath, _, _ := strings.Cut(e[1], "@")
			to, ok := refByPath[toPath]
			if !ok {
				continue // not linked into this binary
			}
			switch {
			case fromVer == "" && fromPath == bi.Main.Path:
				add(root.BOMRef, to)
			case selected[fromPath] == fromVer && refByPath[fromPath] != "":
				if refByPath[fromPath] != to {
					add(refByPath[fromPath], to)
				}
			}
		}
		// A linked module the graph does not reach still ships: attach it to the root so nothing is orphaned.
		reached := map[string]bool{}
		for _, tos := range edges {
			for to := range tos {
				reached[to] = true
			}
		}
		for _, d := range deps {
			if !reached[refByPath[d.Path]] {
				add(root.BOMRef, refByPath[d.Path])
			}
		}
	}
	var dependencies []dependency
	for _, c := range append([]component{root}, comps...) {
		on := []string{}
		for to := range edges[c.BOMRef] {
			on = append(on, to)
		}
		sort.Strings(on)
		dependencies = append(dependencies, dependency{Ref: c.BOMRef, DependsOn: on})
	}

	tool := component{Type: "application", Name: "circula scripts/sbom", Version: rootVersion}
	return &bom{
		BOMFormat:    "CycloneDX",
		SpecVersion:  "1.5",
		SerialNumber: serial(root.BOMRef, opt.BinarySHA, ts),
		Version:      1,
		Metadata:     metadata{Timestamp: ts, Tools: tools{Components: []component{tool}}, Component: root},
		Components:   comps,
		Dependencies: dependencies,
	}, nil
}

// purl builds pkg:golang/<path>@<version>?<qualifiers>; each path segment and the version are percent-encoded.
func purl(path, version string, quals ...string) string {
	segs := strings.Split(path, "/")
	for i, s := range segs {
		segs[i] = escape(s)
	}
	p := "pkg:golang/" + strings.Join(segs, "/")
	if version != "" {
		p += "@" + escape(version)
	}
	if len(quals) > 0 {
		p += "?" + strings.Join(quals, "&")
	}
	return p
}

// escape percent-encodes everything outside the purl unreserved set.
func escape(s string) string {
	var b strings.Builder
	for i := 0; i < len(s); i++ {
		c := s[i]
		switch {
		case c >= 'a' && c <= 'z', c >= 'A' && c <= 'Z', c >= '0' && c <= '9', c == '.', c == '-', c == '_', c == '~':
			b.WriteByte(c)
		default:
			fmt.Fprintf(&b, "%%%02X", c)
		}
	}
	return b.String()
}

// serial returns a deterministic RFC 4122 version 5 UUID URN for the bill.
func serial(parts ...string) string {
	// RFC 4122 URL namespace.
	ns := [16]byte{0x6b, 0xa7, 0xb8, 0x11, 0x9d, 0xad, 0x11, 0xd1, 0x80, 0xb4, 0x00, 0xc0, 0x4f, 0xd4, 0x30, 0xc8}
	h := sha1.New() //nolint:gosec // see import
	h.Write(ns[:])
	h.Write([]byte("circula-sbom:" + strings.Join(parts, "|")))
	u := h.Sum(nil)[:16]
	u[6] = (u[6] & 0x0f) | 0x50
	u[8] = (u[8] & 0x3f) | 0x80
	x := hex.EncodeToString(u)
	return "urn:uuid:" + x[0:8] + "-" + x[8:12] + "-" + x[12:16] + "-" + x[16:20] + "-" + x[20:32]
}

func fileSHA256(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer func() { _ = f.Close() }()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

// escapeModPath applies the module cache case encoding: each upper-case letter becomes '!' and its lower case.
func escapeModPath(s string) string {
	var b strings.Builder
	for _, r := range s {
		if r >= 'A' && r <= 'Z' {
			b.WriteByte('!')
			r += 'a' - 'A'
		}
		b.WriteRune(r)
	}
	return b.String()
}

// zipHashes hashes the downloaded zip of every linked module. Modules replaced by a local directory have none.
func zipHashes(modcache string, deps []*debug.Module) (map[string]string, error) {
	out := map[string]string{}
	for _, d := range deps {
		e := effective(d)
		if e.Version == "" {
			continue
		}
		zip := filepath.Join(modcache, "cache", "download", filepath.FromSlash(escapeModPath(e.Path)), "@v", escapeModPath(e.Version)+".zip")
		h, err := fileSHA256(zip)
		if err != nil {
			return nil, fmt.Errorf("module zip for %s %s: %w (run go mod download)", e.Path, e.Version, err)
		}
		out[modKey(e.Path, e.Version)] = h
	}
	return out, nil
}

func readGraph(path string) ([][2]string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer func() { _ = f.Close() }()
	var edges [][2]string
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		fields := strings.Fields(sc.Text())
		if len(fields) == 0 {
			continue
		}
		if len(fields) != 2 {
			return nil, fmt.Errorf("%s: unexpected go mod graph line %q", path, sc.Text())
		}
		edges = append(edges, [2]string{fields[0], fields[1]})
	}
	return edges, sc.Err()
}

// readLicenses parses table rows "| `path` | version | SPDX |" of the ARC-024 manifest.
func readLicenses(path string) (map[string]string, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	out := map[string]string{}
	for _, line := range strings.Split(string(b), "\n") {
		cells := strings.Split(line, "|")
		if len(cells) != 5 || !strings.HasPrefix(strings.TrimSpace(cells[1]), "`") {
			continue
		}
		mod := strings.Trim(strings.TrimSpace(cells[1]), "`")
		out[modKey(mod, strings.TrimSpace(cells[2]))] = strings.TrimSpace(cells[3])
	}
	return out, nil
}

// validate checks the CycloneDX 1.5 rules the release relies on: the header, a schema-valid serial number, unique
// bom-refs, one dependency entry per component and no dangling dependency reference.
func validate(b *bom) error {
	if b.BOMFormat != "CycloneDX" || b.SpecVersion != "1.5" || b.Version != 1 {
		return fmt.Errorf("header is %q %q version %d, want CycloneDX 1.5 version 1", b.BOMFormat, b.SpecVersion, b.Version)
	}
	if !serialRE.MatchString(b.SerialNumber) {
		return fmt.Errorf("serialNumber %q does not match the CycloneDX 1.5 pattern", b.SerialNumber)
	}
	if _, err := time.Parse(time.RFC3339, b.Metadata.Timestamp); err != nil {
		return fmt.Errorf("metadata.timestamp: %w", err)
	}
	refs := map[string]bool{}
	for _, c := range append([]component{b.Metadata.Component}, b.Components...) {
		if c.Type == "" || c.Name == "" || c.BOMRef == "" {
			return fmt.Errorf("component without type, name or bom-ref: %+v", c)
		}
		if refs[c.BOMRef] {
			return fmt.Errorf("duplicate bom-ref %q", c.BOMRef)
		}
		refs[c.BOMRef] = true
	}
	seen := map[string]bool{}
	for _, d := range b.Dependencies {
		if !refs[d.Ref] {
			return fmt.Errorf("dependency ref %q names no component", d.Ref)
		}
		seen[d.Ref] = true
		for _, on := range d.DependsOn {
			if !refs[on] {
				return fmt.Errorf("%q depends on unknown ref %q", d.Ref, on)
			}
		}
	}
	for r := range refs {
		if !seen[r] {
			return fmt.Errorf("component %q has no dependency entry", r)
		}
	}
	return nil
}

var serialRE = regexp.MustCompile(`^urn:uuid:[0-9a-f]{8}-[0-9a-f]{4}-[1-5][0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`)

// verify checks that the bill in path is well formed and describes exactly the binary bin: its SHA-256, target
// and toolchain, and the set of linked modules with their versions and go.sum hashes.
func verify(path, bin string, bi *debug.BuildInfo) error {
	raw, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	var b bom
	dec := json.NewDecoder(bytes.NewReader(raw))
	if err := dec.Decode(&b); err != nil {
		return fmt.Errorf("%s: %w", path, err)
	}
	if err := validate(&b); err != nil {
		return fmt.Errorf("%s: %w", path, err)
	}
	sum, err := fileSHA256(bin)
	if err != nil {
		return err
	}
	root := b.Metadata.Component
	if len(root.Hashes) != 1 || root.Hashes[0].Alg != "SHA-256" || root.Hashes[0].Content != sum {
		return fmt.Errorf("%s: root SHA-256 %v does not match %s (%s)", path, root.Hashes, bin, sum)
	}
	want, err := build(bi, options{BinarySHA: sum, Version: root.Version, Timestamp: b.Metadata.Timestamp})
	if err != nil {
		return err
	}
	if root.PURL != want.Metadata.Component.PURL {
		return fmt.Errorf("%s: root purl %q, binary is %q", path, root.PURL, want.Metadata.Component.PURL)
	}
	key := func(c component) string { return c.Name + " " + c.Version + " " + prop(c, "circula:go:sum") }
	got := map[string]bool{}
	for _, c := range b.Components {
		got[key(c)] = true
	}
	for _, c := range want.Components {
		if !got[key(c)] {
			return fmt.Errorf("%s: linked module %s is missing from the bill", path, key(c))
		}
		delete(got, key(c))
	}
	if len(got) > 0 {
		extra := make([]string, 0, len(got))
		for k := range got {
			extra = append(extra, k)
		}
		sort.Strings(extra)
		return fmt.Errorf("%s: bill lists %s, not linked into %s", path, strings.Join(extra, ", "), bin)
	}
	return nil
}

func prop(c component, name string) string {
	for _, p := range c.Properties {
		if p.Name == name {
			return p.Value
		}
	}
	return ""
}
