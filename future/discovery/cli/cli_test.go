package cli_test

import (
	"bytes"
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	d "github.com/djshellshoxxx/circula/future/discovery"
	"github.com/djshellshoxxx/circula/future/discovery/cli"
	"github.com/djshellshoxxx/circula/future/discovery/imgsan"
	"github.com/djshellshoxxx/circula/future/discovery/internal/testkit"
	"github.com/djshellshoxxx/circula/future/discovery/sqlstore"
	"github.com/djshellshoxxx/circula/future/discovery/sqlstore/sqlitetest"
)

func run(t *testing.T, args ...string) (int, string, string) {
	t.Helper()
	var out, errb bytes.Buffer
	code := cli.Run(args, &out, &errb, cli.Env{Open: func(p string) (*sql.DB, error) { return sqlitetest.OpenFile(p) }})
	return code, out.String(), errb.String()
}

func needDriver(t *testing.T) {
	if !sqlitetest.Available() {
		t.Skip("run with -tags=sqlite_cgo_test")
	}
}

func TestHelpVersionAndUsageErrors(t *testing.T) {
	code, out, _ := run(t, "help")
	if code != 0 || !strings.Contains(out, "HOLD") {
		t.Fatal(code, out)
	}
	for _, c := range []string{"selftest", "sanitize", "migrate", "categories", "areas", "tags", "search", "retention", "verify", "audit", "version"} {
		if !strings.Contains(out, c) {
			t.Errorf("help lacks %s", c)
		}
		if code, h, _ := run(t, "help", c); code != 0 || !strings.Contains(h, "discovery "+c) {
			t.Errorf("help %s: %d %q", c, code, h)
		}
	}
	if code, _, errs := run(t); code != 2 || !strings.Contains(errs, "Usage") {
		t.Fatal(code, errs)
	}
	if code, _, _ := run(t, "bogus"); code != 2 {
		t.Fatal(code)
	}
	if code, _, _ := run(t, "help", "bogus"); code != 2 {
		t.Fatal(code)
	}
	if code, out, _ := run(t, "version"); code != 0 || !strings.Contains(out, "FUX-103") {
		t.Fatal(code, out)
	}
	for _, args := range [][]string{{"categories"}, {"categories", "list"}, {"categories", "enable", "--network", "n", "--db", "x"}, {"areas", "add"},
		{"tags", "approve", "--network", "n", "--db", "x"}, {"search", "--db", "x"}, {"retention"}, {"verify"}, {"audit"}, {"migrate"}, {"sanitize"}, {"categories", "list", "--nope"}} {
		if code, _, _ := run(t, args...); code != 2 && code != 1 {
			t.Errorf("%v -> %d", args, code)
		}
	}
	if code, _, errs := run(t, "categories", "list"); code != 2 || !strings.Contains(errs, "required") {
		t.Fatal(code, errs)
	}
}

func TestNoDriverMessage(t *testing.T) {
	var out, errb bytes.Buffer
	code := cli.Run([]string{"migrate", "--db", "x.db"}, &out, &errb, cli.Env{})
	if code != 1 || !strings.Contains(errb.String(), "no SQLite driver") {
		t.Fatal(code, errb.String())
	}
}

func TestSelftestAndSanitize(t *testing.T) {
	if code, out, _ := run(t, "selftest"); code != 0 || !strings.Contains(out, "PASS") {
		t.Fatal(code, out)
	}
	dir := t.TempDir()
	in := filepath.Join(dir, "in.jpg")
	os.WriteFile(in, testkit.WithGPSEXIF(testkit.JPEG(t, 300, 200, 1), 1), 0o600)
	outf, th := filepath.Join(dir, "out.jpg"), filepath.Join(dir, "thumb.jpg")
	code, out, errs := run(t, "sanitize", "--in", in, "--out", outf, "--thumb", th, "--max-edge", "100")
	if code != 0 || !strings.Contains(out, "ok image/jpeg 100x67") {
		t.Fatal(code, out, errs)
	}
	b, _ := os.ReadFile(outf)
	if bytes.Contains(b, []byte("GPS")) {
		t.Fatal("gps in output")
	}
	if _, err := os.Stat(th); err != nil {
		t.Fatal(err)
	}
	bad := filepath.Join(dir, "x.svg")
	os.WriteFile(bad, []byte("<svg onload=alert(1)/>"), 0o600)
	if code, _, errs := run(t, "sanitize", "--in", bad, "--out", filepath.Join(dir, "y")); code != 1 || !strings.Contains(errs, "rejected") {
		t.Fatal(code, errs)
	}
	if _, err := os.Stat(filepath.Join(dir, "y")); err == nil {
		t.Fatal("output written for rejected file")
	}
	if code, _, _ := run(t, "sanitize", "--in", filepath.Join(dir, "missing"), "--out", outf); code != 1 {
		t.Fatal(code)
	}
}

func TestOperatorWorkflow(t *testing.T) {
	needDriver(t)
	dir := t.TempDir()
	db := filepath.Join(dir, "d.db")
	blobs := filepath.Join(dir, "blobs")
	base := []string{"--db", db, "--network", "n1"}
	mustRun := func(args ...string) string {
		t.Helper()
		code, out, errs := run(t, args...)
		if code != 0 {
			t.Fatalf("%v -> %d\n%s\n%s", args, code, out, errs)
		}
		return out
	}
	if out := mustRun("migrate", "--db", db); !strings.Contains(out, "applied") {
		t.Fatal(out)
	}
	if out := mustRun(append([]string{"categories", "add", "--seed"}, base...)...); !strings.Contains(out, "added 12") {
		t.Fatal(out)
	}
	out := mustRun(append([]string{"categories", "add", "--name", "Bikes", "--kind", "goods"}, base...)...)
	id := strings.Fields(out)[1]
	if code, _, errs := run(t, append([]string{"categories", "add", "--name", "bikes"}, base...)...); code != 1 || !strings.Contains(errs, "duplicate") {
		t.Fatal(code, errs)
	}
	mustRun(append([]string{"categories", "disable", "--id", id}, base...)...)
	if out := mustRun(append([]string{"categories", "list"}, base...)...); !strings.Contains(out, "Bikes") || !strings.Contains(out, "false") {
		t.Fatal(out)
	}
	mustRun(append([]string{"categories", "rename", "--id", id, "--name", "Cycles"}, base...)...)
	// networks are separate
	if out := mustRun("categories", "list", "--db", db, "--network", "n2"); strings.Contains(out, "Cycles") {
		t.Fatal("cross-network leak")
	}
	mustRun(append([]string{"areas", "add", "--name", "Downtown", "--lat", "49.2827", "--lon", "-123.1207"}, base...)...)
	if out := mustRun(append([]string{"areas", "list"}, base...)...); !strings.Contains(out, "49.3,-123.1") {
		t.Fatal(out)
	}
	if code, _, _ := run(t, append([]string{"areas", "add", "--name", "Bad", "--lat", "95", "--lon", "0"}, base...)...); code != 2 {
		t.Fatal(code)
	}

	// seed a listing + photo + pending tag via the service, then use the CLI
	conn, _ := sqlitetest.OpenFile(db)
	st := sqlstore.New(conn)
	now := time.Now().UTC()
	svc, _ := d.NewService(st, mustBlobs(t, blobs), d.Config{CursorKeys: [][]byte{bytes.Repeat([]byte{1}, 32)}, Now: func() time.Time { return now }}, imgsan.Limits{})
	owner := d.Actor{ID: "alice", Network: "n1", Role: d.RoleMember, Active: true}
	svc.SyncListing(context.Background(), d.ListingRecord{ID: "L1", Network: "n1", OwnerID: "alice", Kind: d.KindOffer, Title: "Road bike", Status: d.StatusActive, OwnerActive: true, Created: now, Updated: now})
	svc.UpdateDiscovery(context.Background(), owner, "L1", d.DiscoveryInput{Tags: []string{"fixie"}})
	m, err := svc.AddMedia(context.Background(), owner, "L1", "bike", bytes.NewReader(testkit.JPEG(t, 80, 60, 1)))
	if err != nil {
		t.Fatal(err)
	}
	conn.Close()
	if out := mustRun(append([]string{"tags", "list", "--status", "pending"}, base...)...); !strings.Contains(out, "fixie") {
		t.Fatal(out)
	}
	if out := mustRun("search", "--db", db, "--network", "n1", "--member", "bob", "--tag", "fixie"); !strings.Contains(out, "no listings") {
		t.Fatal("pending tag searchable:", out)
	}
	mustRun(append([]string{"tags", "approve", "--tag", "fixie"}, base...)...)
	if out := mustRun("search", "--db", db, "--network", "n1", "--member", "bob", "--q", "ROAD", "--json"); !strings.Contains(out, `"L1"`) {
		t.Fatal(out)
	}
	if out := mustRun("search", "--db", db, "--network", "n2", "--member", "carol", "--q", "road"); !strings.Contains(out, "no listings") {
		t.Fatal(out)
	}
	if code, _, _ := run(t, "search", "--db", db, "--network", "n1", "--member", "bob", "--scope", "weird"); code != 1 {
		t.Fatal(code)
	}
	// verify: clean, then damage the blob store
	if out := mustRun("verify", "--db", db, "--blobs", blobs); !strings.Contains(out, "1 active photos, 0 problem") {
		t.Fatal(out)
	}
	os.WriteFile(filepath.Join(blobs, m.StoragePath), []byte("corrupt"), 0o600)
	os.WriteFile(filepath.Join(blobs, "media", "stray.jpg"), []byte("x"), 0o600)
	os.Remove(filepath.Join(blobs, m.ThumbPath))
	code, out, _ := run(t, "verify", "--db", db, "--blobs", blobs)
	if code != 1 || !strings.Contains(out, "CHECKSUM") || !strings.Contains(out, "MISSING thumbnail") || !strings.Contains(out, "ORPHAN file media/stray.jpg") {
		t.Fatal(code, out)
	}
	// retention does nothing now (nothing eligible) and audit shows operator actions
	if out := mustRun("retention", "--db", db, "--blobs", blobs); !strings.Contains(out, "claimed 0") {
		t.Fatal(out)
	}
	if out := mustRun(append([]string{"audit"}, base...)...); !strings.Contains(out, "category.create") || !strings.Contains(out, "cli-operator") || !strings.Contains(out, "tag.moderate") {
		t.Fatal(out)
	}
	// rollback removes module tables only
	if out := mustRun("migrate", "--db", db, "--rollback"); !strings.Contains(out, "rolled back") {
		t.Fatal(out)
	}
	if code, _, _ := run(t, append([]string{"categories", "list"}, base...)...); code != 0 {
		t.Fatal("categories list re-migrates")
	}
}

func mustBlobs(t *testing.T, dir string) d.BlobStore {
	b, err := d.NewFSBlobs(dir)
	if err != nil {
		t.Fatal(err)
	}
	return b
}
