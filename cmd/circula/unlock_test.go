package main

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/djshellshoxxx/circula/internal/store"
)

// SA2-10: unlock needs -login, clears the login's rows for every address and
// the recovery rows, leaves other logins alone, audits, and prints no secret.
func TestUnlockCommand(t *testing.T) {
	db := seedDatabase(t, t.TempDir())
	var out, errOut bytes.Buffer
	if code := run([]string{"unlock", "-database", db}, &out, &errOut); code != exitUsage || out.Len() != 0 {
		t.Fatalf("missing -login: %d %q", code, out.String())
	}
	if code := run([]string{"unlock", "-database", db, "-login", "ghost"}, &out, &errOut); code != exitFailure {
		t.Fatalf("unknown login: %d", code)
	}

	st, err := store.Open(context.Background(), db)
	if err != nil {
		t.Fatal(err)
	}
	for _, k := range []string{"founder01", "founder01 ip:203.0.113.9", "recover: founder01", "recover: founder01 ip:203.0.113.9", "founder010", "other"} {
		if _, err := st.DB().Exec(`INSERT INTO login_throttle(login,failures,locked_until) VALUES (?,9,9999999999999)`, k); err != nil {
			t.Fatal(err)
		}
	}
	_ = st.Close()

	out.Reset()
	errOut.Reset()
	if code := run([]string{"unlock", "-database", db, "-login", "Founder01"}, &out, &errOut); code != exitOK {
		t.Fatalf("unlock: %d %s", code, errOut.String())
	}
	if !strings.Contains(out.String(), "cleared 4 ") {
		t.Errorf("output: %q", out.String())
	}

	st, err = store.Open(context.Background(), db)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = st.Close() }()
	var left int
	if err := st.DB().QueryRow(`SELECT COUNT(*) FROM login_throttle`).Scan(&left); err != nil || left != 2 {
		t.Errorf("rows left = %d (%v), want 2 (other logins untouched)", left, err)
	}
	var typ string
	if err := st.DB().QueryRow(`SELECT actor_type FROM audit_events WHERE action = 'person.login_unlocked'`).Scan(&typ); err != nil || typ == "" {
		t.Errorf("audit event missing: %v", err)
	}
}
