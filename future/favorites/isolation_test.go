package favorites_test

import (
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

const modulePath = "github.com/djshellshoxxx/circula/future/favorites"

// TestSourceIsolation enforces the HOLD contract mechanically over every Go
// file in the module: standard library and this module only (plus the
// test-only cgo driver behind its build tag), no other future module, no root
// application import, no init(), no goroutines or global registration in
// shipped (non-test) code.
func TestSourceIsolation(t *testing.T) {
	fset := token.NewFileSet()
	files := 0
	err := filepath.WalkDir(".", func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.HasSuffix(path, ".go") {
			return err
		}
		files++
		f, err := parser.ParseFile(fset, path, nil, parser.ParseComments)
		if err != nil {
			t.Errorf("%s: %v", path, err)
			return nil
		}
		isTest := strings.HasSuffix(path, "_test.go")
		cgoTagged := false
		for _, cg := range f.Comments {
			for _, c := range cg.List {
				if strings.HasPrefix(c.Text, "//go:build") && strings.Contains(c.Text, "sqlite_cgo_test") {
					cgoTagged = true
				}
			}
		}
		for _, imp := range f.Imports {
			p, _ := strconv.Unquote(imp.Path.Value)
			first := strings.SplitN(p, "/", 2)[0]
			switch {
			case !strings.Contains(first, "."): // standard library
			case p == modulePath || strings.HasPrefix(p, modulePath+"/"):
			case p == "github.com/mattn/go-sqlite3" && cgoTagged:
			default:
				t.Errorf("%s imports %q: only stdlib, this module, and the tagged test-only driver are allowed", path, p)
			}
			if strings.Contains(p, "circula") && !strings.HasPrefix(p, modulePath) {
				t.Errorf("%s imports another Circula package %q", path, p)
			}
		}
		if isTest {
			return nil
		}
		for _, decl := range f.Decls {
			if fn, ok := decl.(*ast.FuncDecl); ok && fn.Recv == nil && fn.Name.Name == "init" && !cgoTagged {
				t.Errorf("%s: init() in shipped code", path)
			}
		}
		ast.Inspect(f, func(n ast.Node) bool {
			switch x := n.(type) {
			case *ast.GoStmt:
				t.Errorf("%s: goroutine in shipped code (no background execution allowed)", path)
			case *ast.SelectorExpr:
				if id, ok := x.X.(*ast.Ident); ok && id.Name == "http" && (x.Sel.Name == "Handle" || x.Sel.Name == "HandleFunc" || x.Sel.Name == "DefaultServeMux" || x.Sel.Name == "ListenAndServe") {
					t.Errorf("%s: global HTTP registration or listener (%s)", path, x.Sel.Name)
				}
			}
			return true
		})
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if files < 10 {
		t.Fatalf("walked only %d files; test is not looking at the module", files)
	}
}
