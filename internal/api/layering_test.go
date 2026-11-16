package api

import (
	"go/parser"
	"go/token"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

// ARC-002: the HTML and JSON adapters reach state only through
// internal/service, and the operator commands only through the service or the
// storage boundary in cmd/circula/schema.go (plus serve.go, the composition
// root). depguard enforces the same rules in CI (.golangci.yml); this test
// keeps them in `go test` so a local run without the linter also fails.
func TestAdaptersDoNotImportStorage(t *testing.T) {
	const mod = "github.com/djshellshoxxx/circula/"
	cases := []struct {
		dir    string
		deny   []string
		exempt map[string]bool
	}{
		{".", []string{mod + "internal/store", "modernc.org/sqlite", "database/sql"}, nil},
		{filepath.Join("..", "web"), []string{mod + "internal/store", "modernc.org/sqlite", "database/sql"}, nil},
		{filepath.Join("..", "..", "cmd", "circula"), []string{mod + "internal/store", "modernc.org/sqlite"},
			map[string]bool{"schema.go": true, "serve.go": true}},
	}
	for _, c := range cases {
		paths, err := filepath.Glob(filepath.Join(c.dir, "*.go"))
		if err != nil || len(paths) == 0 {
			t.Fatalf("no Go files in %s: %v", c.dir, err)
		}
		for _, p := range paths {
			base := filepath.Base(p)
			if strings.HasSuffix(base, "_test.go") || c.exempt[base] {
				continue
			}
			f, err := parser.ParseFile(token.NewFileSet(), p, nil, parser.ImportsOnly)
			if err != nil {
				t.Fatal(err)
			}
			for _, im := range f.Imports {
				ip, _ := strconv.Unquote(im.Path.Value)
				for _, d := range c.deny {
					if ip == d {
						t.Errorf("%s imports %s: go through internal/service (ARC-002)", p, ip)
					}
				}
			}
		}
	}
}
