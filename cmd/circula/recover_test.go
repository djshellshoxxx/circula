package main

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/djshellshoxxx/circula/internal/store"
)

// SEC-008: reset-password needs -login and -reason, prints the temporary
// password once, audits as the system actor, refuses beside a running process.
func TestResetPasswordCommand(t *testing.T) {
	db := seedDatabase(t, t.TempDir())
	var out, errOut bytes.Buffer
	for _, args := range [][]string{
		{"reset-password", "-database", db},
		{"reset-password", "-database", db, "-login", "founder01"},
		{"reset-password", "-database", db, "-reason", "x y z"},
	} {
		out.Reset()
		if code := run(args, &out, &errOut); code != exitUsage || out.Len() != 0 {
			t.Errorf("%v: code %d out %q", args, code, out.String())
		}
	}
	if code := run([]string{"reset-password", "-database", db, "-login", "ghost", "-reason", "no such person"}, &out, &errOut); code != exitFailure {
		t.Errorf("unknown login: %d", code)
	}

	held, err := acquireLock(lockPath(db))
	if err != nil {
		t.Fatal(err)
	}
	errOut.Reset()
	if code := run([]string{"reset-password", "-database", db, "-login", "founder01", "-reason", "lost it"}, &out, &errOut); code != exitFailure ||
		!strings.Contains(errOut.String(), "in use by another circula process") {
		t.Errorf("lock: %d %s", code, errOut.String())
	}
	_ = held.Release()

	out.Reset()
	errOut.Reset()
	if code := run([]string{"reset-password", "-database", db, "-login", "Founder01", "-reason", "ticket 7: lost password"}, &out, &errOut); code != exitOK {
		t.Fatalf("reset: %d %s", code, errOut.String())
	}
	lines := strings.Split(strings.TrimSpace(out.String()), "\n")
	temp := lines[len(lines)-1]
	if len(temp) < 12 || strings.Contains(errOut.String(), temp) {
		t.Fatalf("temp password not printed once on stdout: %q / %q", out.String(), errOut.String())
	}

	st, err := store.Open(context.Background(), db)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = st.Close() }()
	var typ, reason, summary string
	if err := st.DB().QueryRow(`SELECT actor_type, reason, summary_after FROM audit_events WHERE action = 'person.password_reset'`).Scan(&typ, &reason, &summary); err != nil {
		t.Fatal(err)
	}
	if typ != "platform_operator" || reason != "ticket 7: lost password" || strings.Contains(summary, temp) {
		t.Errorf("audit: %s %q %s", typ, reason, summary)
	}
	var must int
	_ = st.DB().QueryRow(`SELECT must_change_password FROM persons WHERE login = 'founder01'`).Scan(&must)
	if must != 1 {
		t.Error("must_change_password not set")
	}
}
