package main

import (
	"bufio"
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

// Citation sources.
const (
	srcMention = "mention" // the file names the ID (comment, string, route comment, test title)
	srcName    = "name"    // a Go test function name carries the ID without the hyphen (TestUX002...)
	srcSpec    = "spec"    // the spec's status note names the test function
	srcLegacy  = "legacy"  // frozen row of scripts/traceability/legacy-citations.tsv whose target still exists
)

// Cite is one code or test citation of a requirement ID.
type Cite struct {
	ID, Path, Symbol string
	Test             bool
	Source           string
}

var (
	compactRe = regexp.MustCompile(`(LED|` + strings.Join(prefixes[1:], "|") + `)_?([0-9]{1,3})(?:[^0-9]|$)`)
	jsTitleRe = regexp.MustCompile(`\b(?:test|it)(?:\.(?:only|skip|fixme|describe|serial))*\(\s*['"` + "`" + `]([^'"` + "`" + `]*)`)
	skipDirs  = map[string]bool{"node_modules": true, "bin": true, "docs": true, "prompts": true, "testdata": true, "vendor": true}
	scanExts  = map[string]bool{".go": true, ".sql": true, ".html": true, ".tmpl": true, ".gohtml": true, ".sh": true, ".py": true, ".js": true}
)

// selfDir is not scanned: its tests and sources contain sample IDs.
const selfDir = "scripts/traceability"

// scanTree walks root and returns every ID mention in code, test, template and script files, plus the index
// of Go test functions (name -> files) used to resolve test names quoted in spec notes.
func scanTree(root string) ([]Cite, map[string][]string, error) {
	var cites []Cite
	testFuncs := map[string][]string{}
	err := filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(root, p)
		rel = filepath.ToSlash(rel)
		if d.IsDir() {
			if rel != "." && (strings.HasPrefix(d.Name(), ".") || skipDirs[d.Name()] || rel == selfDir) {
				return filepath.SkipDir
			}
			return nil
		}
		if !d.Type().IsRegular() || !scanExts[filepath.Ext(rel)] {
			return nil
		}
		src, err := os.ReadFile(p)
		if err != nil {
			return err
		}
		switch filepath.Ext(rel) {
		case ".go":
			cites = append(cites, scanGo(rel, src, testFuncs)...)
		case ".js":
			cites = append(cites, scanJS(rel, string(src))...)
		default:
			for _, id := range uniqueIDs(string(src)) {
				cites = append(cites, Cite{ID: id, Path: rel, Source: srcMention})
			}
		}
		return nil
	})
	for k := range testFuncs {
		sort.Strings(testFuncs[k])
	}
	return cites, testFuncs, err
}

// findIDs returns the [start, end) offsets of requirement IDs in src, skipping a match that continues a longer
// hyphenated label (OQ-API-001, SA-DOM-010).
func findIDs(src []byte) [][]int {
	var out [][]int
	for _, loc := range idRe.FindAllIndex(src, -1) {
		if loc[0] > 0 && src[loc[0]-1] == '-' {
			continue
		}
		out = append(out, loc)
	}
	return out
}

func uniqueIDs(s string) []string {
	seen := map[string]bool{}
	var out []string
	for _, loc := range findIDs([]byte(s)) {
		id := s[loc[0]:loc[1]]
		if !seen[id] {
			seen[id] = true
			out = append(out, id)
		}
	}
	return out
}

type span struct {
	start, end int
	name       string
}

func declSpans(fset *token.FileSet, f *ast.File) []span {
	off := func(p token.Pos) int { return fset.Position(p).Offset }
	var spans []span
	for _, d := range f.Decls {
		switch d := d.(type) {
		case *ast.FuncDecl:
			start := d.Pos()
			if d.Doc != nil {
				start = d.Doc.Pos()
			}
			spans = append(spans, span{off(start), off(d.End()), d.Name.Name})
		case *ast.GenDecl:
			start := d.Pos()
			if d.Doc != nil {
				start = d.Doc.Pos()
			}
			first := ""
			for _, s := range d.Specs {
				name, sStart := "", s.Pos()
				switch s := s.(type) {
				case *ast.TypeSpec:
					name = s.Name.Name
					if s.Doc != nil {
						sStart = s.Doc.Pos()
					}
				case *ast.ValueSpec:
					if len(s.Names) > 0 {
						name = s.Names[0].Name
					}
					if s.Doc != nil {
						sStart = s.Doc.Pos()
					}
				}
				if name == "" || name == "_" {
					continue
				}
				if first == "" {
					first = name
				}
				end := s.End()
				if c := specComment(s); c != nil && c.End() > end {
					end = c.End()
				}
				spans = append(spans, span{off(sStart), off(end), name})
			}
			if first != "" {
				spans = append(spans, span{off(start), off(d.End()), first})
			}
		}
	}
	return spans
}

func specComment(s ast.Spec) *ast.CommentGroup {
	switch s := s.(type) {
	case *ast.TypeSpec:
		return s.Comment
	case *ast.ValueSpec:
		return s.Comment
	}
	return nil
}

// symbolAt returns the name of the smallest declaration span holding offset o ("" at file level).
func symbolAt(spans []span, o int) string {
	best, bestLen := "", -1
	for _, s := range spans {
		if o >= s.start && o < s.end && (bestLen < 0 || s.end-s.start < bestLen) {
			best, bestLen = s.name, s.end-s.start
		}
	}
	return best
}

func scanGo(rel string, src []byte, testFuncs map[string][]string) []Cite {
	isTest := strings.HasSuffix(rel, "_test.go")
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, rel, src, parser.ParseComments)
	if err != nil {
		var out []Cite
		for _, id := range uniqueIDs(string(src)) {
			out = append(out, Cite{ID: id, Path: rel, Test: isTest, Source: srcMention})
		}
		return out
	}
	spans := declSpans(fset, f)
	var out []Cite
	for _, loc := range findIDs(src) {
		out = append(out, Cite{ID: string(src[loc[0]:loc[1]]), Path: rel, Symbol: symbolAt(spans, loc[0]), Test: isTest, Source: srcMention})
	}
	if isTest {
		for _, d := range f.Decls {
			fd, ok := d.(*ast.FuncDecl)
			if !ok || fd.Recv != nil {
				continue
			}
			name := fd.Name.Name
			for _, p := range []string{"Test", "Fuzz", "Benchmark"} {
				if strings.HasPrefix(name, p) {
					testFuncs[name] = append(testFuncs[name], rel)
					for _, m := range compactRe.FindAllStringSubmatch(name[len(p):], -1) {
						out = append(out, Cite{ID: m[1] + "-" + m[2], Path: rel, Symbol: name, Test: true, Source: srcName})
					}
					break
				}
			}
		}
	}
	return out
}

// scanJS cites the nearest enclosing or preceding test(...) title for each mention in a JavaScript file.
func scanJS(rel, src string) []Cite {
	isTest := strings.HasSuffix(rel, ".spec.js") || strings.HasPrefix(rel, "e2e/tests/")
	var out []Cite
	cur := ""
	sc := bufio.NewScanner(strings.NewReader(src))
	sc.Buffer(make([]byte, 0, 64*1024), 4*1024*1024)
	for sc.Scan() {
		line := sc.Text()
		if m := jsTitleRe.FindStringSubmatch(line); m != nil {
			cur = shorten(m[1], 70)
		}
		for _, loc := range findIDs([]byte(line)) {
			id := line[loc[0]:loc[1]]
			sym := ""
			if isTest {
				sym = cur
			}
			out = append(out, Cite{ID: id, Path: rel, Symbol: sym, Test: isTest, Source: srcMention})
		}
	}
	return out
}

// legacyRow is one frozen citation from the hand-made matrix.
type legacyRow struct {
	ID, Kind, Path, Symbol string
}

func parseLegacy(src string) []legacyRow {
	var out []legacyRow
	for _, line := range strings.Split(src, "\n") {
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		f := strings.Split(line, "\t")
		for len(f) < 4 {
			f = append(f, "")
		}
		out = append(out, legacyRow{ID: f[0], Kind: f[1], Path: f[2], Symbol: f[3]})
	}
	return out
}

// resolveLegacy keeps the legacy rows whose file exists and, for a Go test, whose function is declared in it;
// for other rows with a symbol, the file must contain that symbol as a word (Go) or as text (other files).
func resolveLegacy(root string, rows []legacyRow) []Cite {
	cache := map[string]*string{}
	read := func(p string) *string {
		if c, ok := cache[p]; ok {
			return c
		}
		b, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(p)))
		var s *string
		if err == nil {
			t := string(b)
			s = &t
		}
		cache[p] = s
		return s
	}
	var out []Cite
	for _, r := range rows {
		src := read(r.Path)
		if src == nil {
			continue
		}
		if r.Symbol != "" {
			q := regexp.QuoteMeta(r.Symbol)
			switch {
			case strings.HasSuffix(r.Path, ".go") && r.Kind == "test":
				if !regexp.MustCompile(`(?m)^func\s+` + q + `\s*\(`).MatchString(*src) {
					continue
				}
			case strings.HasSuffix(r.Path, ".go"):
				if !regexp.MustCompile(`\b` + q + `\b`).MatchString(*src) {
					continue
				}
			default:
				if !strings.Contains(*src, r.Symbol) {
					continue
				}
			}
		}
		out = append(out, Cite{ID: r.ID, Path: r.Path, Symbol: r.Symbol, Test: r.Kind == "test", Source: srcLegacy})
	}
	return out
}

func shorten(s string, n int) string {
	s = strings.Join(strings.Fields(s), " ")
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	cut := string(r[:n])
	if i := strings.LastIndex(cut, " "); i > n/2 {
		cut = cut[:i]
	}
	return cut + "..."
}
