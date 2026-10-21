package main

import (
	"bytes"
	"database/sql"
	"strings"
	"testing"
)

func runCLI(args ...string) (int, string, string) {
	var out, errOut bytes.Buffer
	code := run(args, &out, &errOut)
	return code, out.String(), errOut.String()
}

// SEC-013: the operator CLI opens, lists and closes sessions; the TTL is capped at four hours.
func TestBreakGlassCLI(t *testing.T) {
	db := seedDatabase(t, t.TempDir())
	code, out, errOut := runCLI("break-glass", "open", "-database", db, "-network", "test-net", "-operator", "Olga",
		"-reason", "ticket 9", "-ttl", "99h")
	if code != exitOK || !strings.Contains(out, "open for") {
		t.Fatalf("open: %d %s %s", code, out, errOut)
	}
	raw, err := sql.Open("sqlite", db)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = raw.Close() }()
	var id string
	var span int64
	if err := raw.QueryRow(`SELECT id, expires_at - opened_at FROM break_glass_sessions`).Scan(&id, &span); err != nil {
		t.Fatal(err)
	}
	if span != 4*3600*1000 {
		t.Errorf("ttl %d ms, want the four hour cap", span)
	}
	code, out, _ = runCLI("break-glass", "list", "-database", db, "-network", "test-net")
	if code != exitOK || !strings.Contains(out, "Olga") || !strings.Contains(out, "open") || !strings.Contains(out, "ticket 9") {
		t.Errorf("list: %d %s", code, out)
	}
	var n int
	if err := raw.QueryRow(`SELECT COUNT(*) FROM audit_events WHERE action = 'break_glass.opened'`).Scan(&n); err != nil || n != 2 {
		t.Errorf("opened audit rows (platform + network): %d %v", n, err)
	}
	if code, _, errOut = runCLI("break-glass", "close", "-database", db, "-session", id); code != exitOK {
		t.Errorf("close: %d %s", code, errOut)
	}
	if code, _, _ = runCLI("break-glass", "close", "-database", db, "-session", id); code != exitFailure {
		t.Errorf("second close: %d", code)
	}
	// grant-admin needs an open session.
	code, _, errOut = runCLI("break-glass", "grant-admin", "-database", db, "-network", "test-net", "-login", "nobody", "-reason", "x")
	if code != exitFailure || !strings.Contains(errOut, "break-glass") {
		t.Errorf("grant without session: %d %s", code, errOut)
	}
}

func TestBreakGlassAndEraseUsage(t *testing.T) {
	db := seedDatabase(t, t.TempDir())
	for _, args := range [][]string{
		{"break-glass"},
		{"break-glass", "bogus", "-database", db},
		{"break-glass", "open", "-database", db, "-network", "test-net"},
		{"break-glass", "grant-admin", "-database", db, "-network", "test-net"},
		{"erase-member", "-database", db, "-network", "test-net", "-member", "x", "-reason", "r"}, // no -confirm
	} {
		if code, _, _ := runCLI(args...); code != exitUsage {
			t.Errorf("%v: code %d, want usage", args, code)
		}
	}
	// Refused cleanly for an unknown or active member.
	code, _, errOut := runCLI("erase-member", "-database", db, "-network", "test-net", "-member", "x", "-reason", "r", "-confirm")
	if code != exitFailure || !strings.Contains(errOut, "not found") {
		t.Errorf("unknown member: %d %s", code, errOut)
	}
	if _, out, _ := runCLI("help"); !strings.Contains(out, "break-glass") || !strings.Contains(out, "erase-member") {
		t.Error("help lacks the new commands")
	}
}
