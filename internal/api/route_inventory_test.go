package api

import (
	"bufio"
	"go/ast"
	"go/parser"
	"go/token"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"testing"

	"github.com/djshellshoxxx/circula/internal/domain"
	"github.com/djshellshoxxx/circula/internal/httpx"
	"github.com/djshellshoxxx/circula/internal/service"
	"github.com/djshellshoxxx/circula/internal/testutil"
	"github.com/djshellshoxxx/circula/internal/web"
)

// QA-012 / ARC-008 / SEC-009: every HTTP route the binary serves is listed in
// the reviewed authorization table testdata/route_authz.txt with the least
// principal allowed through. The inventory is read from the registration
// calls in the source (JSON API, HTML interface and the operational mux in
// cmd/circula); a route missing from the table, a table row that is not
// registered, or a gate that disagrees with the registration fails the build.
// TestRouteAuthorizationMatrix then sends every route as anonymous, as a
// person of no network and as a plain member, and fails on any unexpected
// allow.

const authzTablePath = "testdata/route_authz.txt"

// Gates of the table, per surface. There is deliberately no operator gate:
// the platform operator has no HTTP route (ARC-008, API-019).
var authzGates = map[string]map[string]bool{
	"api": {"public": true, "person": true, "member": true, "reviewer": true, "admin": true},
	"web": {"public": true, "person": true, "member": true, "reviewer": true, "admin": true},
	"ops": {"mount": true, "health": true},
}

// routeKey identifies a route: surface, method ("*" for any) and path pattern.
type routeKey struct{ surface, method, path string }

func (k routeKey) String() string { return k.surface + " " + k.method + " " + k.path }

// scannedRoute is one registration found in the source.
type scannedRoute struct {
	routeKey
	derived string // gate implied by the registration itself
	pos     string
}

type authzRow struct {
	routeKey
	gate string
	line int
}

func splitRoutePattern(p string) (method, path string) {
	if m, rest, ok := strings.Cut(p, " "); ok && !strings.HasPrefix(p, "/") {
		return m, strings.TrimSpace(rest)
	}
	return "*", p
}

// constStringExpr evaluates a string constant expression: literals, "+" and
// package-level string constants.
func constStringExpr(e ast.Expr, consts map[string]string) (string, bool) {
	switch x := e.(type) {
	case *ast.BasicLit:
		if x.Kind != token.STRING {
			return "", false
		}
		s, err := strconv.Unquote(x.Value)
		return s, err == nil
	case *ast.Ident:
		s, ok := consts[x.Name]
		return s, ok
	case *ast.ParenExpr:
		return constStringExpr(x.X, consts)
	case *ast.BinaryExpr:
		if x.Op != token.ADD {
			return "", false
		}
		a, ok1 := constStringExpr(x.X, consts)
		b, ok2 := constStringExpr(x.Y, consts)
		return a + b, ok1 && ok2
	}
	return "", false
}

// parseGoDir parses the non-test Go files of dir and collects its package-level
// string constants.
func parseGoDir(t *testing.T, fset *token.FileSet, dir string) ([]*ast.File, map[string]string) {
	t.Helper()
	paths, err := filepath.Glob(filepath.Join(dir, "*.go"))
	if err != nil || len(paths) == 0 {
		t.Fatalf("no Go files in %s: %v", dir, err)
	}
	var files []*ast.File
	consts := map[string]string{}
	for _, p := range paths {
		if strings.HasSuffix(p, "_test.go") {
			continue
		}
		f, err := parser.ParseFile(fset, p, nil, parser.SkipObjectResolution)
		if err != nil {
			t.Fatal(err)
		}
		files = append(files, f)
		for _, d := range f.Decls {
			g, ok := d.(*ast.GenDecl)
			if !ok || g.Tok != token.CONST {
				continue
			}
			for _, sp := range g.Specs {
				vs := sp.(*ast.ValueSpec)
				for i, n := range vs.Names {
					if i < len(vs.Values) {
						if s, ok := constStringExpr(vs.Values[i], consts); ok {
							consts[n.Name] = s
						}
					}
				}
			}
		}
	}
	return files, consts
}

func routeCallee(fun ast.Expr) (recv, name string) {
	switch f := fun.(type) {
	case *ast.Ident:
		return "", f.Name
	case *ast.SelectorExpr:
		switch x := f.X.(type) {
		case *ast.Ident:
			return x.Name, f.Sel.Name
		case *ast.SelectorExpr:
			return x.Sel.Name, f.Sel.Name
		}
		return "?", f.Sel.Name
	}
	return "", ""
}

// routePlumbing are the registration calls that forward a pattern held in a
// variable; every other registration must use a constant pattern.
var routePlumbing = map[string]bool{
	"api/route.go:add":     true, // s.mux.HandleFunc(rt.pattern, s.wrap(rt))
	"api/routes.go:routes": true, // R's body: s.add(rt)
}

// scanRoutes lists the routes registered by the non-test files of dir.
func scanRoutes(t *testing.T, surface, dir string) []scannedRoute {
	t.Helper()
	fset := token.NewFileSet()
	files, consts := parseGoDir(t, fset, dir)
	var out []scannedRoute
	for _, f := range files {
		file := filepath.Base(fset.Position(f.Pos()).Filename)
		for _, d := range f.Decls {
			site := surface + "/" + file + ":" // package-level declarations (function literals in vars)
			if fd, ok := d.(*ast.FuncDecl); ok {
				site += fd.Name.Name
			}
			ast.Inspect(d, func(n ast.Node) bool {
				call, ok := n.(*ast.CallExpr)
				if !ok || len(call.Args) == 0 {
					return true
				}
				recv, name := routeCallee(call.Fun)
				isReg := false
				switch name {
				case "HandleFunc", "Handle":
					isReg = true
					if recv != "mux" {
						t.Errorf("%s: route registered on %q, not on a mux the inventory reads", fset.Position(call.Pos()), recv)
						return true
					}
				case "R":
					isReg = recv == "" && surface == "api"
				case "add":
					isReg = recv == "s" && surface == "api"
				}
				if !isReg {
					return true
				}
				arg := call.Args[0]
				if cl, ok := arg.(*ast.CompositeLit); ok && name == "add" {
					arg = nil
					for _, el := range cl.Elts {
						if kv, ok := el.(*ast.KeyValueExpr); ok {
							if k, ok := kv.Key.(*ast.Ident); ok && k.Name == "pattern" {
								arg = kv.Value
							}
						}
					}
					if arg == nil {
						t.Errorf("%s: route literal without a pattern", fset.Position(call.Pos()))
						return true
					}
				}
				pat, ok := constStringExpr(arg, consts)
				if !ok {
					if !routePlumbing[site] {
						t.Errorf("%s: route pattern is not a constant; the inventory cannot read it (register with a constant pattern)",
							fset.Position(call.Pos()))
					}
					return true
				}
				m, p := splitRoutePattern(pat)
				out = append(out, scannedRoute{routeKey{surface, m, p}, derivedGate(surface, name, p, call), fset.Position(call.Pos()).String()})
				return true
			})
		}
	}
	return out
}

// derivedGate reads the gate the registration itself enforces.
func derivedGate(surface, name, path string, call *ast.CallExpr) string {
	switch surface {
	case "api":
		if name != "R" {
			return "public" // the not-found fallback
		}
		gate := "member"
		for _, a := range call.Args[2:] {
			if id, ok := a.(*ast.Ident); ok {
				switch id.Name {
				case "noNet":
					gate = "person"
				case "public":
					return "public"
				}
			}
		}
		return gate
	case "web":
		if len(call.Args) < 2 {
			return "public"
		}
		if c, ok := call.Args[1].(*ast.CallExpr); ok {
			if _, n := routeCallee(c.Fun); n != "" {
				switch n {
				case "auth":
					return "person"
				case "netH":
					return "member"
				case "adminH":
					return "admin"
				}
			}
		}
		return "public"
	default: // ops
		if path == "/" || path == "/api/" {
			return "mount"
		}
		return "health"
	}
}

// gateMatches reports whether the table's gate is compatible with what the
// registration enforces. Roles finer than membership are decided by the
// service (api) or the handler (web reviewer pages).
func gateMatches(surface, derived, table, path string) bool {
	if surface == "api" && strings.HasPrefix(path, v1+"/admin/") && table != "admin" {
		return false
	}
	switch derived {
	case "member":
		return table == "member" || table == "reviewer" || (table == "admin" && surface == "api")
	default:
		return table == derived
	}
}

func readAuthzTable(t *testing.T) []authzRow {
	t.Helper()
	f, err := os.Open(authzTablePath)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = f.Close() }()
	var rows []authzRow
	sc := bufio.NewScanner(f)
	for n := 1; sc.Scan(); n++ {
		line := sc.Text()
		if i := strings.Index(line, "#"); i >= 0 {
			line = line[:i]
		}
		fs := strings.Fields(line)
		if len(fs) == 0 {
			continue
		}
		if len(fs) != 4 {
			t.Fatalf("%s:%d: want 4 fields (surface method path gate), got %q", authzTablePath, n, sc.Text())
		}
		rows = append(rows, authzRow{routeKey{fs[0], fs[1], fs[2]}, fs[3], n})
	}
	if err := sc.Err(); err != nil {
		t.Fatal(err)
	}
	return rows
}

func routeInventory(t *testing.T) []scannedRoute {
	t.Helper()
	var all []scannedRoute
	all = append(all, scanRoutes(t, "api", ".")...)
	all = append(all, scanRoutes(t, "web", filepath.Join("..", "web"))...)
	all = append(all, scanRoutes(t, "ops", filepath.Join("..", "..", "cmd", "circula"))...)
	return all
}

func TestRouteInventoryMatchesAuthorizationTable(t *testing.T) {
	inv := routeInventory(t)
	rows := readAuthzTable(t)
	table := map[routeKey]authzRow{}
	for _, r := range rows {
		if prev, dup := table[r.routeKey]; dup {
			t.Errorf("%s:%d: %s already listed on line %d", authzTablePath, r.line, r.routeKey, prev.line)
		}
		if !authzGates[r.surface][r.gate] {
			t.Errorf("%s:%d: %s has gate %q; allowed for %s: %v (there is no operator gate, ARC-008)",
				authzTablePath, r.line, r.routeKey, r.gate, r.surface, sortedGates(authzGates[r.surface]))
		}
		table[r.routeKey] = r
	}
	seen := map[routeKey]bool{}
	for _, s := range inv {
		if seen[s.routeKey] {
			t.Errorf("%s: %s registered twice", s.pos, s.routeKey)
		}
		seen[s.routeKey] = true
		r, ok := table[s.routeKey]
		if !ok {
			t.Errorf("%s: route %s is missing from %s; add it with its gate after review (QA-012)", s.pos, s.routeKey, authzTablePath)
			continue
		}
		if !gateMatches(s.surface, s.derived, r.gate, s.path) {
			t.Errorf("%s:%d: %s is listed as %q but its registration enforces %q", authzTablePath, r.line, s.routeKey, r.gate, s.derived)
		}
	}
	for _, r := range rows {
		if !seen[r.routeKey] {
			t.Errorf("%s:%d: %s is not registered; remove the row", authzTablePath, r.line, r.routeKey)
		}
	}
	if len(inv) < 150 {
		t.Errorf("inventory found only %d routes; the scanner is missing registrations", len(inv))
	}
}

func sortedGates(m map[string]bool) []string {
	var out []string
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

// ARC-008: no HTTP route can act as the platform operator. The operator is
// the system actor (domain.Actor.System), so neither adapter may construct
// one; and the operational endpoints in cmd/circula cannot reach the service
// at all, so they can read or change no network data.
func TestNoOperatorScopedHTTPRoute(t *testing.T) {
	for _, dir := range []string{".", filepath.Join("..", "web")} {
		fset := token.NewFileSet()
		files, _ := parseGoDir(t, fset, dir)
		for _, f := range files {
			ast.Inspect(f, func(n ast.Node) bool {
				switch x := n.(type) {
				case *ast.SelectorExpr:
					if x.Sel.Name == "SystemActor" {
						t.Errorf("%s: an HTTP adapter builds the system (operator) actor", fset.Position(x.Pos()))
					}
				case *ast.KeyValueExpr:
					if k, ok := x.Key.(*ast.Ident); ok && k.Name == "System" {
						t.Errorf("%s: an HTTP adapter sets Actor.System", fset.Position(x.Pos()))
					}
				case *ast.AssignStmt:
					for _, l := range x.Lhs {
						if s, ok := l.(*ast.SelectorExpr); ok && s.Sel.Name == "System" {
							t.Errorf("%s: an HTTP adapter sets Actor.System", fset.Position(x.Pos()))
						}
					}
				}
				return true
			})
		}
	}
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, filepath.Join("..", "..", "cmd", "circula", "ops_http.go"), nil, parser.ImportsOnly)
	if err != nil {
		t.Fatal(err)
	}
	for _, im := range f.Imports {
		if p, _ := strconv.Unquote(im.Path.Value); strings.HasSuffix(p, "/internal/service") {
			t.Errorf("cmd/circula/ops_http.go imports %s: the operational endpoints must not reach network data", p)
		}
	}
	for _, r := range readAuthzTable(t) {
		if r.gate == "health" && r.method != "*" && r.method != http.MethodGet {
			t.Errorf("%s:%d: operational route %s is not read-only", authzTablePath, r.line, r.routeKey)
		}
	}
	// Every authenticated API caller is a person; tokens and sessions never
	// yield the system actor.
	env := testutil.NewEnv(t)
	who := env.Person("someone")
	tok, _, err := env.Svc.CreateAPIToken(env.Ctx, who, service.TokenInput{Name: "t", Scopes: []domain.Scope{domain.ScopeRead, domain.ScopeWrite}})
	if err != nil {
		t.Fatal(err)
	}
	a, err := env.Svc.AuthenticateToken(env.Ctx, tok)
	if err != nil || a.System || a.PersonID != who.PersonID {
		t.Errorf("token actor = %+v, %v; want the person, never the system actor", a, err)
	}
	sess, _, err := env.Svc.Login(env.Ctx, "someone", "correct horse battery")
	if err != nil {
		t.Fatal(err)
	}
	if a, err := env.Svc.Authenticate(env.Ctx, sess); err != nil || a.System || a.PersonID != who.PersonID {
		t.Errorf("session actor = %+v, %v; want the person, never the system actor", a, err)
	}
}

// concretePath turns a route pattern into a request path.
func concretePath(path string, vals map[string]string) string {
	path = strings.ReplaceAll(path, "{$}", "")
	var b strings.Builder
	for {
		i := strings.Index(path, "{")
		if i < 0 {
			b.WriteString(path)
			return b.String()
		}
		j := strings.Index(path[i:], "}")
		b.WriteString(path[:i])
		name := strings.TrimSuffix(path[i+1:i+j], "...")
		if v, ok := vals[name]; ok {
			b.WriteString(v)
		} else {
			b.WriteString("x")
		}
		path = path[i+j+1:]
	}
}

// routeProbe is one principal's expectation for a gate.
type routeProbe struct {
	who  string
	deny func(status int) bool
	want string
}

// TestRouteAuthorizationMatrix: anonymous callers, people of no network and
// plain members are refused on every route the table closes to them, and a
// refusal for a network the caller does not belong to looks like not-found
// (QA-012, SEC-009, API-017). Nothing is ever a 5xx.
func TestRouteAuthorizationMatrix(t *testing.T) {
	key := []byte("0123456789abcdef0123456789abcdef")
	env := testutil.NewEnv(t)
	admin := env.Person("admin")
	net, _ := env.Network(admin, "acme", nil)
	alice, _ := env.Member(admin, net, "alice")
	_, bobM := env.Member(admin, net, "bob")
	outsider := env.Person("outsider")
	apiH := NewHandlerWithOptions(env.Svc, Options{CSRFKey: key})
	webH := web.NewHandlerWith(env.Svc, web.Options{CSRFKey: key})
	srv := apiH.(*server)

	token := func(a domain.Actor) string {
		raw, _, err := env.Svc.CreateAPIToken(env.Ctx, a, service.TokenInput{Name: "t", Scopes: []domain.Scope{domain.ScopeRead, domain.ScopeWrite}})
		if err != nil {
			t.Fatal(err)
		}
		return raw
	}
	login := func(name string) string {
		s, _, err := env.Svc.Login(env.Ctx, name, "correct horse battery")
		if err != nil {
			t.Fatal(err)
		}
		return s
	}
	tokens := map[string]string{"alice": token(alice), "outsider": token(outsider)}
	sessions := map[string]string{"alice": login("alice"), "outsider": login("outsider")}
	apiVals := map[string]string{"role": "reviewer"}
	webVals := map[string]string{"slug": net.Slug}

	is4xx := func(s int) bool { return s >= 400 && s < 500 }
	api := map[string][]routeProbe{
		"person":   {{"anonymous", func(s int) bool { return s == 401 }, "401"}},
		"member":   {{"anonymous", func(s int) bool { return s == 401 }, "401"}, {"outsider", func(s int) bool { return s == 404 }, "404 not-found"}},
		"reviewer": {{"anonymous", func(s int) bool { return s == 401 }, "401"}, {"outsider", func(s int) bool { return s == 404 }, "404 not-found"}, {"alice", is4xx, "a 4xx refusal"}},
		"admin":    {{"anonymous", func(s int) bool { return s == 401 }, "401"}, {"outsider", func(s int) bool { return s == 404 }, "404 not-found"}, {"alice", is4xx, "a 4xx refusal"}},
	}
	toLogin := func(s int) bool { return s == http.StatusSeeOther }
	webP := map[string][]routeProbe{
		"person":   {{"anonymous", toLogin, "303 to /login"}},
		"member":   {{"anonymous", toLogin, "303 to /login"}, {"outsider", func(s int) bool { return s == 404 }, "404"}},
		"reviewer": {{"anonymous", toLogin, "303 to /login"}, {"outsider", func(s int) bool { return s == 404 }, "404"}, {"alice", func(s int) bool { return s == 403 }, "403"}},
		"admin":    {{"anonymous", toLogin, "303 to /login"}, {"outsider", func(s int) bool { return s == 404 }, "404"}, {"alice", func(s int) bool { return s == 403 }, "403"}},
	}
	csrf := httpx.CSRF{Key: key}
	anonSecret := strings.Repeat("a", 32)

	for _, r := range readAuthzTable(t) {
		switch r.surface {
		case "api":
			vals := apiVals
			if strings.Contains(r.path, "/members/{id}") || strings.Contains(r.path, "/accounts/{id}") {
				vals = map[string]string{"id": bobM.ID, "role": "reviewer"} // a real member: the refusal must come from the role check
			}
			p := concretePath(r.path, vals)
			if r.method != "*" {
				hr := httptest.NewRequest(r.method, p, nil)
				if _, pat := srv.mux.Handler(hr); pat != r.method+" "+r.path {
					t.Errorf("%s: %s %s resolves to %q, want the listed pattern", r.routeKey, r.method, p, pat)
				}
			}
			for _, pr := range api[r.gate] {
				if r.method == "*" {
					continue
				}
				hr := httptest.NewRequest(r.method, p, nil)
				if isUnsafe(r.method) && r.method != http.MethodDelete {
					hr = httptest.NewRequest(r.method, p, strings.NewReader("{}"))
					hr.Header.Set("Content-Type", "application/json")
					if r.method == http.MethodPatch {
						hr.Header.Set("Content-Type", "application/merge-patch+json")
					}
				}
				hr.Header.Set("Idempotency-Key", "route-matrix-"+pr.who+"-"+strconv.Itoa(r.line))
				if pr.who != "anonymous" {
					hr.Header.Set("Authorization", "Bearer "+tokens[pr.who])
				}
				probeRoute(t, r, pr, apiH, hr)
			}
		case "web":
			if r.method == "*" {
				continue
			}
			p := concretePath(r.path, webVals)
			for _, pr := range webP[r.gate] {
				hr := httptest.NewRequest(r.method, p, nil)
				if pr.who == "anonymous" {
					hr.AddCookie(&http.Cookie{Name: httpx.CSRFCookie, Value: anonSecret})
					hr.Header.Set("X-CSRF-Token", csrf.Token(anonSecret))
				} else {
					hr.AddCookie(&http.Cookie{Name: httpx.SessionCookie, Value: sessions[pr.who]})
					hr.Header.Set("X-CSRF-Token", csrf.Token(sessions[pr.who]))
				}
				rec := probeRoute(t, r, pr, webH, hr)
				if pr.who == "anonymous" && rec.Code == http.StatusSeeOther && !strings.HasPrefix(rec.Header().Get("Location"), "/login") {
					t.Errorf("%s as anonymous: redirected to %q, want /login", r.routeKey, rec.Header().Get("Location"))
				}
			}
		}
	}
}

func probeRoute(t *testing.T, r authzRow, pr routeProbe, h http.Handler, hr *http.Request) *httptest.ResponseRecorder {
	t.Helper()
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, hr)
	if !pr.deny(rec.Code) || rec.Code >= 500 {
		what := "unexpected response"
		if rec.Code < 400 && rec.Code != http.StatusSeeOther {
			what = "unexpected allow"
		}
		t.Errorf("%s (gate %s) as %s: status %d, want %s (%s; %s:%d)", r.routeKey, r.gate, pr.who, rec.Code, pr.want,
			what, authzTablePath, r.line)
	}
	if pr.who == "outsider" && r.surface == "api" && rec.Code == 404 && !strings.Contains(rec.Body.String(), `"code":"not-found"`) {
		t.Errorf("%s as outsider: 404 body %s is not the plain not-found problem", r.routeKey, rec.Body.String())
	}
	return rec
}
