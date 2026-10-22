package main

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/djshellshoxxx/circula/internal/domain"
	"github.com/djshellshoxxx/circula/internal/service"
	"github.com/djshellshoxxx/circula/internal/store"
)

// A3-03: reset-mfa needs -login and -reason, refuses a person without a second
// factor, removes it otherwise and audits as the operator.
func TestResetMFACommand(t *testing.T) {
	db := seedDatabase(t, t.TempDir())
	var out, errOut bytes.Buffer
	for _, args := range [][]string{
		{"reset-mfa", "-database", db},
		{"reset-mfa", "-database", db, "-login", "founder01"},
		{"reset-mfa", "-database", db, "-reason", "x y z"},
	} {
		if code := run(args, &out, &errOut); code != exitUsage {
			t.Errorf("%v: code %d", args, code)
		}
	}
	if code := run([]string{"reset-mfa", "-database", db, "-login", "founder01", "-reason", "lost phone"}, &out, &errOut); code != exitFailure {
		t.Errorf("no second factor: %d", code)
	}

	ctx := context.Background()
	st, err := store.Open(ctx, db)
	if err != nil {
		t.Fatal(err)
	}
	svc, err := service.New(st, service.Config{})
	if err != nil {
		t.Fatal(err)
	}
	var pid string
	if err := st.DB().QueryRow(`SELECT id FROM persons WHERE login = 'founder01'`).Scan(&pid); err != nil {
		t.Fatal(err)
	}
	actor := domain.Actor{PersonID: pid, Scopes: []domain.Scope{domain.ScopeRead, domain.ScopeWrite}}
	if _, err := svc.BeginMFAEnrolment(ctx, actor, "correct horse battery"); err != nil {
		t.Fatal(err)
	}
	_ = st.Close()

	out.Reset()
	errOut.Reset()
	if code := run([]string{"reset-mfa", "-database", db, "-login", "Founder01", "-reason", "ticket 9: lost phone"}, &out, &errOut); code != exitOK {
		t.Fatalf("reset-mfa: %d %s", code, errOut.String())
	}
	if !strings.Contains(out.String(), "founder01") && !strings.Contains(out.String(), "Founder01") {
		t.Errorf("output: %q", out.String())
	}
	st, err = store.Open(ctx, db)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = st.Close() }()
	var n int
	_ = st.DB().QueryRow(`SELECT COUNT(*) FROM person_mfa`).Scan(&n)
	var typ, reason string
	if err := st.DB().QueryRow(`SELECT actor_type, reason FROM audit_events WHERE action = ?`, domain.ActMFAReset).Scan(&typ, &reason); err != nil {
		t.Fatal(err)
	}
	if n != 0 || typ != "platform_operator" || reason != "ticket 9: lost phone" {
		t.Errorf("mfa rows %d, audit %s %q", n, typ, reason)
	}
}
