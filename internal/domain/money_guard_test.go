package domain_test

import (
	"go/ast"
	"go/parser"
	"go/token"
	"regexp"
	"strconv"
	"testing"
)

var moneyFloatText = regexp.MustCompile(`\bfloat(32|64)\b|strconv\.(ParseFloat|FormatFloat)`)

// Exact aggregate verification needs big.Int when an int64 prefix overflows.
// Keep the existing float prohibition and permit only the integer type and
// constructor from math/big, including under an explicit import alias.
func nonIntegerMoneyArithmetic(src []byte) (string, error) {
	if use := moneyFloatText.Find(src); use != nil {
		return string(use), nil
	}
	f, err := parser.ParseFile(token.NewFileSet(), "money.go", src, 0)
	if err != nil {
		return "", err
	}
	bigNames := make(map[string]bool)
	for _, imp := range f.Imports {
		path, err := strconv.Unquote(imp.Path.Value)
		if err != nil {
			return "", err
		}
		if path != "math/big" {
			continue
		}
		name := "big"
		if imp.Name != nil {
			name = imp.Name.Name
		}
		if name == "." || name == "_" {
			return "math/big import " + name, nil
		}
		bigNames[name] = true
	}
	use := ""
	ast.Inspect(f, func(n ast.Node) bool {
		if use != "" {
			return false
		}
		sel, ok := n.(*ast.SelectorExpr)
		if !ok {
			return true
		}
		// Inferred results must not hide a float conversion from the text
		// check, including big.Int.Float64 and aliased strconv calls.
		switch sel.Sel.Name {
		case "Float32", "Float64", "SetFloat64", "ParseFloat", "FormatFloat":
			use = sel.Sel.Name
			return false
		}
		if pkg, ok := sel.X.(*ast.Ident); ok && bigNames[pkg.Name] && sel.Sel.Name != "Int" && sel.Sel.Name != "NewInt" {
			use = pkg.Name + "." + sel.Sel.Name
			return false
		}
		return true
	})
	return use, nil
}

// Validate the QA guard itself against aliases and inferred floating-point
// conversions, so allowing exact integer sums cannot exempt math/big wholesale.
func TestIntegerOnlyMoneyArithmeticGuard(t *testing.T) {
	cases := []struct {
		name, src string
		allowed   bool
	}{
		{"integer amount", `package p; var amount int64`, true},
		{"exact integer", `package p; import "math/big"; var total big.Int`, true},
		{"integer constructor", `package p; import "math/big"; var total = big.NewInt(1)`, true},
		{"integer alias", `package p; import exact "math/big"; var total exact.Int; var one = exact.NewInt(1)`, true},
		{"float type", `package p; var amount float64`, false},
		{"big float", `package p; import "math/big"; var amount big.Float`, false},
		{"aliased big float", `package p; import exact "math/big"; var amount exact.Float`, false},
		{"aliased float constructor", `package p; import exact "math/big"; var amount = exact.NewFloat(1)`, false},
		{"rational type", `package p; import exact "math/big"; var amount exact.Rat`, false},
		{"rational constructor", `package p; import exact "math/big"; var amount = exact.NewRat(1, 2)`, false},
		{"dot import", `package p; import . "math/big"; var total Int`, false},
		{"blank import", `package p; import _ "math/big"`, false},
		{"inferred float", `package p; import "math/big"; var amount, _ = big.NewInt(1).Float64()`, false},
		{"aliased float parser", `package p; import text "strconv"; var amount, _ = text.ParseFloat("1", 64)`, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			use, err := nonIntegerMoneyArithmetic([]byte(tc.src))
			if err != nil {
				t.Fatal(err)
			}
			if (use == "") != tc.allowed {
				t.Fatalf("arithmetic guard reported %q, allowed=%v", use, tc.allowed)
			}
		})
	}
}
