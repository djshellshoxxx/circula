package api

import (
	"go/ast"
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
				id, ok := arg.(*ast.Ident)
				if !ok {
					t.Errorf("%s: unrecognized registration modifier", fset.Position(arg.Pos()))
					continue
				}
				switch id.Name {
				case "idem", "idemOpt":
					cached = true
				case "noNet", "cookieOnly":
				default:
					t.Errorf("%s: review registration modifier %q for replay caching", fset.Position(arg.Pos()), id.Name)
				}
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
