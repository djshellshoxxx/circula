//go:build sqlite_cgo_test

package cli_test

import (
	"context"
	"database/sql"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"

	_ "github.com/mattn/go-sqlite3"

	qr "github.com/djshellshoxxx/circula/future/qr"
	"github.com/djshellshoxxx/circula/future/qr/cli"
	"github.com/djshellshoxxx/circula/future/qr/qrfixture"
)

func newService(t *testing.T) (*qr.Service, *sql.DB) {
	t.Helper()
	db, err := sql.Open("sqlite3", "file:"+filepath.Join(t.TempDir(), "c.db")+"?_busy_timeout=5000&_journal_mode=WAL")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	ctx := context.Background()
	for _, e := range []error{qrfixture.Migrate(ctx, db), qrfixture.AddNetwork(ctx, db, "n1", "Alpha", true),
		qrfixture.AddMember(ctx, db, "n1", "a", "Alice", 1000), qrfixture.AddMember(ctx, db, "n1", "b", "Bob", 1000), qrfixture.AddMember(ctx, db, "n1", "c", "Carol", 1000)} {
		if e != nil {
			t.Fatal(e)
		}
	}
	st := qr.NewStore(db)
	if err := st.Migrate(ctx); err != nil {
		t.Fatal(err)
	}
	svc, err := qr.NewService(qr.Config{Origin: origin, AliasSecret: []byte("0123456789abcdef0123456789abcdef-cli"), Now: func() time.Time { return time.Date(2026, 11, 20, 12, 0, 0, 0, time.UTC) }}, st, &qrfixture.Fixture{})
	if err != nil {
		t.Fatal(err)
	}
	return svc, db
}

func TestStatefulCommands(t *testing.T) {
	svc, _ := newService(t)
	env := cli.Env{Service: svc}
	dir := t.TempDir()
	a := []string{"--network", "n1", "--member", "a"}
	b := []string{"--network", "n1", "--member", "b"}
	with := func(base []string, rest ...string) []string { return append(append([]string{}, rest...), base...) }

	code, out, e := run(t, env, with(a, "alias", "show", "--out", filepath.Join(dir, "alias.png"))...)
	if code != 0 || !strings.Contains(out, "generation=1") || !strings.Contains(out, origin+"/qr/v1/a/") {
		t.Fatal(code, out, e)
	}
	code, out, e = run(t, env, with(a, "alias", "rotate", "--key", "cli-rot-0001")...)
	if code != 0 || !strings.Contains(out, "generation=2") || !strings.Contains(out, "old_codes=revoked") {
		t.Fatal(code, out, e)
	}
	// create, with stable idempotency
	create := with(a, "request", "create", "--amount", "300", "--note", "cli", "--ttl", "1h", "--key", "cli-create-001", "--out", filepath.Join(dir, "req.png"))
	code, out, e = run(t, env, create...)
	if code != 0 || !strings.Contains(out, "warning=shown once") || !strings.Contains(out, "status=open") || !strings.Contains(out, "completed=false") {
		t.Fatal(code, out, e)
	}
	link := regexp.MustCompile(`url=(\S+)`).FindStringSubmatch(out)[1]
	id := regexp.MustCompile(`id=(qrr_\S+)`).FindStringSubmatch(out)[1]
	_, out2, _ := run(t, env, with(a, "request", "create", "--amount", "300", "--note", "cli", "--ttl", "1h", "--key", "cli-create-001")...)
	if !strings.Contains(out2, "token_available=false") || strings.Contains(out2, "url=") {
		t.Fatal(out2)
	}
	// an existing PNG is never overwritten
	if code, _, e := run(t, env, with(a, "alias", "show", "--out", filepath.Join(dir, "alias.png"))...); code != 1 || !strings.Contains(e, "error") {
		t.Fatal("overwrote an existing file", code, e)
	}
	code, out, e = run(t, env, with(b, "request", "preview", link)...)
	if code != 0 || !strings.Contains(out, "payee=Alice") || !strings.Contains(out, "amount_minor=300") || !strings.Contains(out, "not acceptance") {
		t.Fatal(code, out, e)
	}
	code, out, e = run(t, env, with(b, "claim", link, "--key", "cli-claim-0001")...)
	if code != 0 || !strings.Contains(out, "status=pending") || strings.Contains(out, "completed") {
		t.Fatal(code, out, e)
	}
	code, out, _ = run(t, env, with(a, "request", "show", id)...)
	if code != 0 || !strings.Contains(out, "status=pending") || !strings.Contains(out, "completed=false") {
		t.Fatal(out)
	}
	if code, _, e := run(t, env, with(a, "request", "revoke", id)...); code != 1 || !strings.Contains(e, "already claimed") {
		t.Fatal(code, e)
	}
	if code, out, _ := run(t, env, with(a, "request", "list")...); code != 0 || !strings.Contains(out, id) {
		t.Fatal(out)
	}
	// error mapping and usage
	for _, c := range []struct {
		args []string
		code int
		msg  string
	}{
		{[]string{"claim", link, "--key", "cli-claim-0002", "--network", "n1", "--member", "c"}, 1, "already claimed"},
		{with(b, "claim", "garbage", "--key", "k-12345678"), 1, "invalid"},
		{with(b, "claim", link, "--key", "cli-claim-0001", "--amount", "5"), 1, "fixes its own amount"},
		{with(a, "request", "revoke", "qrr_00000000000000000000000000000000"), 1, "not found"},
		{[]string{"request", "list"}, 2, "--network and --member"},
		{with(a, "request", "create", "--amount", "0", "--key", "cli-create-002"), 1, "amount"},
		{with(a, "request", "show"), 2, "usage"},
		{with(a, "request", "frobnicate"), 2, ""},
	} {
		code, _, e := run(t, env, c.args...)
		if code != c.code || !strings.Contains(e, c.msg) {
			t.Fatalf("%v -> %d %q", c.args, code, e)
		}
	}
	// alias claim by a member
	_, aout, _ := run(t, env, with(a, "alias", "show")...)
	aliasLink := regexp.MustCompile(`url=(\S+)`).FindStringSubmatch(aout)[1]
	code, out, e = run(t, env, with(b, "claim", aliasLink, "--amount", "50", "--note", "x", "--key", "cli-claim-0003")...)
	if code != 0 || !strings.Contains(out, "status=pending") {
		t.Fatal(code, out, e)
	}
}
