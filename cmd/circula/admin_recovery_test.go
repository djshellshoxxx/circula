package main

import (
	"context"
	"database/sql"
	"go/ast"
	"go/parser"
	"go/token"
	"path/filepath"
	"strings"
	"testing"

	"github.com/djshellshoxxx/circula/internal/domain"
	"github.com/djshellshoxxx/circula/internal/service"
	"github.com/djshellshoxxx/circula/internal/store"
)

// seedOrphanedNetwork builds an active network whose only administrator is
// suspended (GOV-020) and in which member "ana" has asked for recovery. It
// returns the database path and the request id.
func seedOrphanedNetwork(t *testing.T) (string, string) {
	t.Helper()
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "circula.db")
	st, err := store.Open(ctx, path)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = st.Close() }()
	svc, err := service.New(st, service.Config{Argon: service.Argon2Params{MemoryKiB: 64, Time: 1, Threads: 1, KeyLen: 16, SaltLen: 8}})
	if err != nil {
		t.Fatal(err)
	}
	person := func(login string) domain.Actor {
		p, err := svc.RegisterPerson(ctx, service.RegisterInput{Login: login, Password: "correct horse battery"})
		if err != nil {
			t.Fatal(err)
		}
		return domain.Actor{PersonID: p.ID, Scopes: []domain.Scope{domain.ScopeRead, domain.ScopeWrite}}
	}
	founder := person("founder01")
	net, fm, err := svc.CreateNetwork(ctx, founder, service.CreateNetworkInput{Name: "Test", Slug: "test-net", CreatorDisplayName: "Founder"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.ActivateNetwork(ctx, founder, net.ID); err != nil {
		t.Fatal(err)
	}
	ana := person("ana01")
	tok, _, err := svc.CreateInvitation(ctx, founder, net.ID, service.InviteInput{})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.RedeemInvitation(ctx, ana, tok, "Ana"); err != nil {
		t.Fatal(err)
	}
	if _, err := st.DB().Exec(`UPDATE members SET status = 'suspended' WHERE id = ?`, fm.ID); err != nil {
		t.Fatal(err)
	}
	r, err := svc.RequestAdminRecovery(ctx, ana, net.ID, service.AdminRecoveryInput{IdempotencyKey: "cli-test"})
	if err != nil {
		t.Fatal(err)
	}
	return path, r.ID
}

// GOV-021: the operator lists, approves and refuses through the CLI; one
// operator alone cannot apply a request, and every step is audited.
func TestGOV021AdminRecoveryCLI(t *testing.T) {
	db, id := seedOrphanedNetwork(t)
	code, out, errOut := runCLI("admin-recovery", "list", "-database", db)
	if code != exitOK || !strings.Contains(out, id) || !strings.Contains(out, "requested") || !strings.Contains(out, "Ana") ||
		!strings.Contains(out, "test-net") {
		t.Fatalf("list: %d %s %s", code, out, errOut)
	}
	approve := func(op string) (int, string, string) {
		return runCLI("admin-recovery", "approve", "-database", db, "-network", "test-net", "-request", id,
			"-operator", op, "-reason", "ticket 12: identity confirmed")
	}
	code, out, errOut = approve("Olga")
	if code != exitOK || !strings.Contains(out, "first approval") {
		t.Fatalf("first approval: %d %s %s", code, out, errOut)
	}
	if code, _, errOut = approve("OLGA"); code != exitFailure || !strings.Contains(errOut, "different operator") {
		t.Fatalf("same operator twice: %d %s", code, errOut)
	}
	raw, err := sql.Open("sqlite", db)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = raw.Close() }()
	admins := func() int {
		var n int
		if err := raw.QueryRow(`SELECT COUNT(*) FROM role_assignments r JOIN members m ON m.id = r.member_id
			WHERE r.role = 'admin' AND r.revoked_at IS NULL AND m.status = 'active'`).Scan(&n); err != nil {
			t.Fatal(err)
		}
		return n
	}
	if admins() != 0 {
		t.Fatal("one operator appointed an administrator")
	}
	code, out, errOut = approve("Piet")
	if code != exitOK || !strings.Contains(out, "applied") || !strings.Contains(out, "Olga and Piet") {
		t.Fatalf("second approval: %d %s %s", code, out, errOut)
	}
	if admins() != 1 {
		t.Fatal("dual-approved request did not appoint")
	}
	if code, _, _ = approve("Ravi"); code != exitFailure {
		t.Fatalf("approval after apply: %d", code)
	}
	code, _, errOut = runCLI("admin-recovery", "refuse", "-database", db, "-network", "test-net", "-request", id,
		"-operator", "Ravi", "-reason", "late")
	if code != exitFailure || !strings.Contains(errOut, "applied") {
		t.Fatalf("refuse after apply: %d %s", code, errOut)
	}
	code, out, _ = runCLI("admin-recovery", "list", "-database", db, "-network", "test-net", "-all")
	if code != exitOK || !strings.Contains(out, "applied") || !strings.Contains(out, "Olga") {
		t.Fatalf("list -all: %d %s", code, out)
	}
	var platform, network int
	if err := raw.QueryRow(`SELECT COUNT(*) FILTER (WHERE network_id = ''), COUNT(*) FILTER (WHERE network_id <> '')
		FROM audit_events WHERE action LIKE 'admin_recovery.%' AND actor_type = 'platform_operator'`).Scan(&platform, &network); err != nil {
		t.Fatal(err)
	}
	if platform != 2 || network != 2 {
		t.Fatalf("operator audit rows: platform %d network %d, want 2 and 2", platform, network)
	}
}

func TestGOV021AdminRecoveryCLIUsage(t *testing.T) {
	db, id := seedOrphanedNetwork(t)
	for _, args := range [][]string{
		{"admin-recovery"},
		{"admin-recovery", "bogus", "-database", db},
		{"admin-recovery", "approve", "-database", db, "-network", "test-net", "-request", id, "-operator", "Olga"},
		{"admin-recovery", "refuse", "-database", db, "-network", "test-net", "-request", id, "-reason", "r"},
		{"admin-recovery", "approve", "-database", db, "-request", id, "-operator", "Olga", "-reason", "r"},
	} {
		if code, _, _ := runCLI(args...); code != exitUsage {
			t.Errorf("%v: code %d, want usage", args, code)
		}
	}
	code, _, errOut := runCLI("admin-recovery", "approve", "-database", db, "-network", "nope", "-request", id, "-operator", "Olga", "-reason", "r")
	if code != exitFailure || !strings.Contains(errOut, "not found") {
		t.Errorf("unknown network: %d %s", code, errOut)
	}
	code, out, errOut := runCLI("admin-recovery", "refuse", "-database", db, "-network", "test-net", "-request", id, "-operator", "Olga", "-reason", "no")
	if code != exitOK || !strings.Contains(out, "refused") {
		t.Errorf("refuse: %d %s %s", code, out, errOut)
	}
	if _, out, _ := runCLI("help"); !strings.Contains(out, "admin-recovery") {
		t.Error("help lacks admin-recovery")
	}
}

// GOV-021: the platform operator must not appoint an administrator on its own
// authority. No operator command calls the service's role or break-glass
// grant methods other than the documented break-glass command, and the only
// permanent appointment an operator command can reach is the dual-approved
// ApproveAdminRecovery.
func TestGOV021NoOperatorCommandAppointsDirectly(t *testing.T) {
	allowed := map[string]string{"GrantBreakGlassAdmin": "breakglass.go"}
	files, err := filepath.Glob("*.go")
	if err != nil {
		t.Fatal(err)
	}
	for _, f := range files {
		if strings.HasSuffix(f, "_test.go") {
			continue
		}
		fset := token.NewFileSet()
		file, err := parser.ParseFile(fset, f, nil, 0)
		if err != nil {
			t.Fatal(err)
		}
		ast.Inspect(file, func(n ast.Node) bool {
			sel, ok := n.(*ast.SelectorExpr)
			if !ok {
				return true
			}
			switch sel.Sel.Name {
			case "AssignRole", "GrantRole", "GrantBreakGlassAdmin":
				if allowed[sel.Sel.Name] != f {
					t.Errorf("%s: operator command calls %s; appointments go through admin-recovery (GOV-021)", fset.Position(sel.Pos()), sel.Sel.Name)
				}
			}
			return true
		})
	}
}
