package main

import (
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const membersJSON = `{"members":[
 {"Network":"n1","ID":"alice","Name":"Alice","Active":true,"Visible":true},
 {"Network":"n1","ID":"bob","Name":"Bob","Active":true,"Visible":true},
 {"Network":"n1","ID":"carol","Name":"Carol","Active":true,"Visible":true}],
 "blocks":[],"trades":[{"ID":"t1","Network":"n1","Payer":"alice","Payee":"bob","Description":"bike","Completed":true}]}`

func capture(t *testing.T, fn func() int) (int, string) {
	t.Helper()
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w
	code := fn()
	w.Close()
	os.Stdout = old
	b, _ := io.ReadAll(r)
	return code, string(b)
}

func TestHelpAndMissingFlags(t *testing.T) {
	if c, o := capture(t, func() int { return run([]string{"help"}) }); c != 0 || !strings.Contains(o, "development harness") {
		t.Errorf("%d %q", c, o)
	}
	if c := run([]string{"list"}); c != 2 {
		t.Errorf("missing flags: %d", c)
	}
}

// With the test-only cgo driver tag this exercises the real binary path over a
// real SQLite file across separate invocations (restart durability).
func TestEndToEndAcrossInvocations(t *testing.T) {
	if driverName == "" {
		t.Skip("built without -tags sqlite_cgo_test: no SQLite driver linked")
	}
	dir := t.TempDir()
	m := filepath.Join(dir, "m.json")
	if err := os.WriteFile(m, []byte(membersJSON), 0o600); err != nil {
		t.Fatal(err)
	}
	db := filepath.Join(dir, "f.db")
	g := []string{"--db", db, "--members", m}
	call := func(extra ...string) (int, string) {
		return capture(t, func() int { return run(append(append([]string{}, g...), extra...)) })
	}
	id := []string{"--network", "n1", "--member", "alice"}
	if c, o := call(append([]string{"add", "--target", "bob", "--nickname", "B"}, id...)...); c != 0 || !strings.Contains(o, "saved bob") {
		t.Fatalf("%d %q", c, o)
	}
	if c, o := call(append([]string{"list"}, id...)...); c != 0 || !strings.Contains(o, "bob") || !strings.Contains(o, "Bob") {
		t.Fatalf("list after reopen: %d %q", c, o)
	}
	if c, o := call(append([]string{"draft", "--trade", "t1", "--key", "k"}, id...)...); c != 0 || !strings.Contains(o, "nothing has been proposed") {
		t.Fatalf("%d %q", c, o)
	}
	if c, _ := call(append([]string{"list"}, "--network", "n1", "--member", "bob")...); c != 0 {
		t.Fatal("bob list")
	}
	if _, o := call(append([]string{"list"}, "--network", "n1", "--member", "bob")...); strings.Contains(o, "Bob") {
		t.Fatalf("bob sees alice's list: %q", o)
	}
}
