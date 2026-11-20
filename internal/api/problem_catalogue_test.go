package api

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"testing"
)

// A2-08: every problem code this package (the JSON API adapter) can send has a
// row in the error code catalogue of spec 07 section 7.2, with the status the
// adapter sends it with. The codes come from codeStatus (the service codes
// fromErr maps) and from every apiError literal in the package's sources (the
// codes the adapter builds itself), so a code cannot ship without its row.
func TestProblemCodesHaveCatalogueRows(t *testing.T) {
	catalogue := specErrorCatalogue(t)
	literals := apiErrorLiterals(t)
	if len(literals) < 20 {
		t.Fatalf("found only %d apiError literals; the scan no longer sees the adapter's problems", len(literals))
	}
	type source struct {
		status int
		where  string
	}
	sent := map[string][]source{}
	for code, status := range codeStatus {
		sent[string(code)] = append(sent[string(code)], source{status, "codeStatus"})
	}
	for _, l := range literals {
		sent[l.code] = append(sent[l.code], source{l.status, l.pos})
	}
	codes := make([]string, 0, len(sent))
	for code := range sent {
		codes = append(codes, code)
	}
	sort.Strings(codes)
	for _, code := range codes {
		want, listed := catalogue[code]
		for _, s := range sent[code] {
			switch {
			case !listed:
				t.Errorf("%s (%d, %s) can be sent but has no row in spec 07 section 7.2", code, s.status, s.where)
			case s.status != want:
				t.Errorf("%s is sent with %d (%s) but spec 07 section 7.2 says %d", code, s.status, s.where, want)
			}
		}
	}
}

// specErrorCatalogue reads the code and status columns of the table in
// spec 07 section 7.2.
func specErrorCatalogue(t *testing.T) map[string]int {
	t.Helper()
	b, err := os.ReadFile(filepath.Join("..", "..", "docs", "specs", "07-api-contract.md"))
	if err != nil {
		t.Fatal(err)
	}
	_, section, found := strings.Cut(string(b), "\n### 7.2 Error code catalogue")
	if !found {
		t.Fatal("spec 07 section 7.2 not found")
	}
	section, _, _ = strings.Cut(section, "\n### ")
	out := map[string]int{}
	for _, line := range strings.Split(section, "\n") {
		if !strings.HasPrefix(line, "| `") {
			continue
		}
		cells := strings.Split(line, "|")
		if len(cells) < 4 {
			t.Fatalf("malformed catalogue row %q", line)
		}
		code := strings.Trim(strings.TrimSpace(cells[1]), "`")
		status, err := strconv.Atoi(strings.TrimSpace(cells[2]))
		if err != nil {
			t.Fatalf("catalogue row %q: status: %v", line, err)
		}
		if _, dup := out[code]; dup {
			t.Errorf("code %s has two rows in spec 07 section 7.2", code)
		}
		out[code] = status
	}
	if len(out) < 20 {
		t.Fatalf("read only %d rows from spec 07 section 7.2", len(out))
	}
	return out
}

type problemLiteral struct {
	code, pos string
	status    int
}

// apiErrorLiterals returns the code and status of every apiError composite
// literal in this package's non-test sources. Only fromErr may build one whose
// code is not a string literal: it takes code and status from codeStatus,
// which the test checks directly. Any other literal this scan cannot read
// fails the test instead of being skipped.
func apiErrorLiterals(t *testing.T) []problemLiteral {
	t.Helper()
	paths, err := filepath.Glob("*.go")
	if err != nil {
		t.Fatal(err)
	}
	fset := token.NewFileSet()
	var out []problemLiteral
	for _, p := range paths {
		if strings.HasSuffix(p, "_test.go") {
			continue
		}
		f, err := parser.ParseFile(fset, p, nil, 0)
		if err != nil {
			t.Fatal(err)
		}
		for _, decl := range f.Decls {
			fn, isFunc := decl.(*ast.FuncDecl)
			inFromErr := isFunc && fn.Name.Name == "fromErr"
			ast.Inspect(decl, func(n ast.Node) bool {
				lit, ok := n.(*ast.CompositeLit)
				if !ok {
					return true
				}
				if id, ok := lit.Type.(*ast.Ident); !ok || id.Name != "apiError" {
					return true
				}
				pos := fset.Position(lit.Pos()).String()
				fields := map[string]ast.Expr{}
				for _, e := range lit.Elts {
					kv, ok := e.(*ast.KeyValueExpr)
					if !ok {
						t.Errorf("%s: apiError literal without field names; this test cannot read it", pos)
						return true
					}
					if k, ok := kv.Key.(*ast.Ident); ok {
						fields[k.Name] = kv.Value
					}
				}
				code, ok := fields["code"].(*ast.BasicLit)
				if !ok || code.Kind != token.STRING {
					if !inFromErr {
						t.Errorf("%s: apiError whose code is not a string literal; this test cannot check it against spec 07 section 7.2", pos)
					}
					return true
				}
				c, err := strconv.Unquote(code.Value)
				if err != nil {
					t.Fatalf("%s: code %s: %v", pos, code.Value, err)
				}
				st, ok := fields["status"].(*ast.BasicLit)
				if !ok || st.Kind != token.INT {
					t.Errorf("%s: apiError %q has no integer literal status; this test cannot check it", pos, c)
					return true
				}
				status, err := strconv.Atoi(st.Value)
				if err != nil {
					t.Fatalf("%s: status %s: %v", pos, st.Value, err)
				}
				out = append(out, problemLiteral{code: c, pos: pos, status: status})
				return true
			})
		}
	}
	return out
}
