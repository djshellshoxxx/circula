package api

import (
	"go/ast"
	"go/parser"
	"go/token"
	"net/http"
	"testing"
	"time"

	"github.com/djshellshoxxx/circula/internal/service"
)

// Every cached route must have an explicit replay gate. Read registration
// sites, rather than copying their list, so new cached endpoints fail closed
// until their authority is reviewed. Unknown registration modifiers also fail
// this inventory instead of silently hiding another idempotency annotation.
func TestCachedRouteReplayAuthorizationInventory(t *testing.T) {
	fset := token.NewFileSet()
	files, consts := parseGoDir(t, fset, ".")
	seen := map[string]bool{}
	for _, file := range files {
		ast.Inspect(file, func(n ast.Node) bool {
			call, ok := n.(*ast.CallExpr)
			if !ok || len(call.Args) < 2 {
				return true
			}
			recv, name := routeCallee(call.Fun)
			if recv != "" || name != "R" {
				return true
			}
			cached := false
			for _, arg := range call.Args[2:] {
				caches, reviewed := replayInventoryModifier(arg)
				if !reviewed {
					t.Errorf("%s: unrecognized registration modifier; review its replay behavior", fset.Position(arg.Pos()))
				}
				cached = cached || caches
			}
			if !cached {
				return true
			}
			pattern, ok := constStringExpr(call.Args[0], consts)
			if !ok {
				t.Errorf("%s: cached route pattern is not constant", fset.Position(call.Pos()))
				return true
			}
			seen[pattern] = true
			if replayPermission(pattern) == service.ReplayDenied {
				t.Errorf("%s: cached route lacks replay authorization: %s", fset.Position(call.Pos()), pattern)
			}
			return true
		})
	}
	if len(seen) < 29 {
		t.Fatalf("inventory found only %d cached routes; expected at least the existing 29", len(seen))
	}
	if replayPermission("POST /api/v1/unreviewed-write") != service.ReplayDenied {
		t.Fatal("unreviewed route must fail closed")
	}
}

// queryParams only wraps the handler with query-name validation. Recognize its
// exact registration shape, while unknown calls, nested modifiers and dynamic
// argument expansion still require review rather than hiding replay caching.
func replayInventoryModifier(arg ast.Expr) (cached, reviewed bool) {
	switch expr := arg.(type) {
	case *ast.Ident:
		switch expr.Name {
		case "idem", "idemOpt":
			return true, true
		case "noNet", "cookieOnly":
			return false, true
		}
	case *ast.CallExpr:
		callee, ok := expr.Fun.(*ast.Ident)
		if !ok || callee.Name != "queryParams" || expr.Ellipsis.IsValid() {
			return false, false
		}
		for _, allowed := range expr.Args {
			literal, ok := allowed.(*ast.BasicLit)
			if !ok || literal.Kind != token.STRING {
				return false, false
			}
		}
		return false, true
	}
	return false, false
}

func TestReplayInventoryModifiersFailClosed(t *testing.T) {
	for _, tc := range []struct {
		src              string
		cached, reviewed bool
	}{
		{"idem", true, true}, {"idemOpt", true, true},
		{"noNet", false, true}, {"cookieOnly", false, true},
		{"queryParams()", false, true}, {`queryParams("at")`, false, true},
		{"unreviewed", false, false}, {"unreviewed()", false, false},
		{"wrap(idem)", false, false}, {"queryParams(idem)", false, false},
		{"queryParams(computed())", false, false}, {"queryParams(names...)", false, false},
		{"other.queryParams()", false, false}, {"queryParams(1)", false, false},
	} {
		t.Run(tc.src, func(t *testing.T) {
			expr, err := parser.ParseExpr(tc.src)
			if err != nil {
				t.Fatal(err)
			}
			cached, reviewed := replayInventoryModifier(expr)
			if cached != tc.cached || reviewed != tc.reviewed {
				t.Fatalf("cached=%t reviewed=%t; want %t %t", cached, reviewed, tc.cached, tc.reviewed)
			}
		})
	}
}

// The reviewed wrapper must remain neutral to the metadata that controls
// authentication and replay. If it starts setting these flags, the inventory's
// classification above must be reviewed as well.
func TestQueryParamsPreservesAuthorizationMetadata(t *testing.T) {
	for _, initial := range []route{
		{pattern: "GET /example"},
		{pattern: "GET /example", idem: idemRequired},
		{pattern: "GET /example", idem: idemOptional, noNet: true, cookieOnly: true},
	} {
		for _, allowed := range [][]string{nil, {"at"}} {
			rt := initial
			queryParams(allowed...)(&rt)
			if rt.pattern != initial.pattern || rt.idem != initial.idem || rt.noNet != initial.noNet || rt.cookieOnly != initial.cookieOnly {
				t.Fatal("queryParams changed authentication or replay metadata")
			}
		}
	}
}

// Erasure can drop an in-flight record before a caller reuses its key. The
// old handler must not complete or delete the new request's reservation.
func TestReplayStaleFinishDoesNotAdoptReusedKey(t *testing.T) {
	for _, keep := range []bool{true, false} {
		t.Run(map[bool]string{true: "success", false: "failure"}[keep], func(t *testing.T) {
			st := newIdemStore(time.Hour, 10)
			now := time.Now()
			state, old := st.begin("key", "old-fingerprint", "person", "network", now)
			if state != idemNew || old == nil {
				t.Fatal("new request did not receive its own reservation")
			}
			st.dropErased("network", "erased-counterparty")
			state, current := st.begin("key", "new-fingerprint", "person", "network", now)
			if state != idemNew || current == nil || current == old {
				t.Fatal("replacement request did not receive a new reservation")
			}
			st.finish("key", old, http.StatusCreated, http.Header{}, []byte("erased body"), keep)
			if state, _ := st.begin("key", "new-fingerprint", "person", "network", now); state != idemBusy {
				t.Fatalf("stale completion changed the new request: state=%v", state)
			}
			st.finish("key", current, http.StatusCreated, http.Header{}, []byte("current body"), true)
			state, replay := st.begin("key", "new-fingerprint", "person", "network", now)
			if state != idemReplay || replay != current || string(replay.body) != "current body" {
				t.Fatal("the current request did not retain its own response")
			}
		})
	}
}
