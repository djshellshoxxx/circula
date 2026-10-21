package api_test

import (
	"archive/zip"
	"bytes"
	"context"
	"database/sql"
	"encoding/csv"
	"fmt"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/djshellshoxxx/circula/internal/api"
	"github.com/djshellshoxxx/circula/internal/domain"
	"github.com/djshellshoxxx/circula/internal/service"
	"github.com/djshellshoxxx/circula/internal/store"
	"github.com/djshellshoxxx/circula/internal/testutil"
)

// CMD-05 / OPS-011: while another process holds the write lock, a write fails
// with a retryable 503 and leaves nothing behind; once the lock is released the
// same request (same Idempotency-Key) succeeds.
func TestBusyDatabaseIs503AndRetrySucceeds(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "circula.db")
	st, err := store.OpenWith(ctx, path, store.Options{BusyTimeout: 100 * time.Millisecond})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = st.Close() })
	svc, err := service.New(st, service.Config{Argon: service.Argon2Params{MemoryKiB: 64, Time: 1, Threads: 1, KeyLen: 16, SaltLen: 8}})
	if err != nil {
		t.Fatal(err)
	}
	admin := registerPerson(t, svc, "admin-busy")
	if _, _, err := svc.CreateNetwork(ctx, admin, service.CreateNetworkInput{Name: "Busy", Slug: "busy", CreatorDisplayName: "Admin"}); err != nil {
		t.Fatal(err)
	}
	nets, err := svc.MyMemberships(ctx, admin)
	if err != nil || len(nets) != 1 {
		t.Fatalf("memberships: %v %v", nets, err)
	}
	netID := nets[0].NetworkID
	if _, err := svc.ActivateNetwork(ctx, admin, netID); err != nil {
		t.Fatal(err)
	}
	raw, _, err := svc.CreateAPIToken(ctx, admin, service.TokenInput{Name: "t", Scopes: []domain.Scope{domain.ScopeRead, domain.ScopeWrite}})
	if err != nil {
		t.Fatal(err)
	}
	h := api.NewHandlerWithOptions(svc, api.Options{CSRFKey: []byte("0123456789abcdef0123456789abcdef")})
	post := func() *httptest.ResponseRecorder {
		r := httptest.NewRequest("POST", "/api/v1/listings", strings.NewReader(`{"kind":"offer","title":"ladder"}`))
		r.Header.Set("Authorization", "Bearer "+raw)
		r.Header.Set("Content-Type", "application/json")
		r.Header.Set("Idempotency-Key", "busy-key-1")
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, r)
		return rec
	}

	// A second connection, as the operator CLI or a backup would hold it.
	holder, err := sql.Open("sqlite", "file:"+path+"?_pragma=busy_timeout(0)")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = holder.Close() })
	conn, err := holder.Conn(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := conn.ExecContext(ctx, `BEGIN IMMEDIATE`); err != nil {
		t.Fatal(err)
	}
	rec := post()
	if _, err := conn.ExecContext(ctx, `ROLLBACK`); err != nil {
		t.Fatal(err)
	}
	_ = conn.Close()
	if rec.Code != http.StatusServiceUnavailable || !strings.Contains(rec.Body.String(), `"unavailable"`) {
		t.Fatalf("busy write: %d %s", rec.Code, rec.Body.String())
	}
	if rec.Header().Get("Retry-After") == "" {
		t.Error("503 without Retry-After")
	}
	var n int
	if err := st.DB().QueryRowContext(ctx, `SELECT COUNT(*) FROM listings`).Scan(&n); err != nil || n != 0 {
		t.Fatalf("busy write left %d listing(s): %v", n, err)
	}

	rec = post()
	if rec.Code != http.StatusCreated {
		t.Fatalf("retry after lock released: %d %s", rec.Code, rec.Body.String())
	}
}

func registerPerson(t *testing.T, svc *service.Service, login string) domain.Actor {
	t.Helper()
	p, err := svc.RegisterPerson(context.Background(), service.RegisterInput{Login: login, Password: "correct horse battery"})
	if err != nil {
		t.Fatal(err)
	}
	return domain.Actor{PersonID: p.ID, Scopes: []domain.Scope{domain.ScopeRead, domain.ScopeWrite}}
}

// walkPages follows next_cursor from path with the given page size and returns
// every item id in order. It fails on duplicates and on pages over limit.
func walkPages(t *testing.T, w *world, token, path string, limit int) []string {
	t.Helper()
	var ids []string
	seen := map[string]bool{}
	next := fmt.Sprintf("%s?limit=%d", path, limit)
	for pages := 0; next != "" && pages < 10000; pages++ {
		rec := w.get(token, next[len("/api/v1"):])
		wantStatus(t, rec, 200)
		pg := decode[page](t, rec)
		if len(pg.Data) > limit {
			t.Fatalf("page of %d exceeds limit %d", len(pg.Data), limit)
		}
		for _, it := range pg.Data {
			id, _ := it["id"].(string)
			if seen[id] {
				t.Fatalf("duplicate item %s across pages", id)
			}
			seen[id] = true
			ids = append(ids, id)
		}
		next = ""
		if pg.NextCursor != nil {
			next = fmt.Sprintf("%s?limit=%d&cursor=%s", path, limit, *pg.NextCursor)
		}
	}
	return ids
}

// API-004 / WEB-02: listings and the member directory page correctly past the
// 100-row page and well past 2000 rows, with no gaps, duplicates or early end.
func TestPaginationBeyondOneHundredAndTwoThousandRows(t *testing.T) {
	if testing.Short() {
		t.Skip("creates 2100+ rows")
	}
	env := testutil.NewEnv(t)
	admin := env.Person("admin")
	net, _ := env.Network(admin, "acme", nil)
	alice, _ := env.Member(admin, net, "alice")
	const listings = 2105
	for i := 0; i < listings; i++ {
		testutil.Must(env.Svc.CreateListing(env.Ctx, alice, net.ID, service.ListingInput{Type: domain.ListingOffer, Title: fmt.Sprintf("item %d", i), IdempotencyKey: env.Key()}))
	}
	const members = 120
	for i := 0; i < members; i++ {
		env.Member(admin, net, fmt.Sprintf("member%03d", i))
	}
	w := &world{t: t, env: env, h: api.NewHandlerWithOptions(env.Svc, api.Options{CSRFKey: testCSRFKey})}
	aliceT := w.token(alice, domain.ScopeRead, domain.ScopeWrite)

	for _, limit := range []int{25, 100} {
		ids := walkPages(t, w, aliceT, "/api/v1/listings", limit)
		if len(ids) != listings {
			t.Errorf("listings with limit %d: %d items, want %d", limit, len(ids), listings)
		}
	}
	if ids := walkPages(t, w, aliceT, "/api/v1/members", 100); len(ids) != members+2 {
		t.Errorf("members: %d, want %d", len(ids), members+2)
	}
}

// API-065 / DAT-040: a ledger export of more than 2000 entries completes, is
// served once with a matching checksum, and its entries file holds every row.
// Each trade posts two entries, so 1005 trades give 2010.
func TestLargeExportRoundTrip(t *testing.T) {
	if testing.Short() {
		t.Skip("posts 1005 trades")
	}
	env := testutil.NewEnv(t)
	admin := env.Person("admin")
	net, _ := env.Network(admin, "acme", nil)
	alice, aliceM := env.Member(admin, net, "alice")
	bob, bobM := env.Member(admin, net, "bob")
	const trades = 1005
	for i := 0; i < trades; i++ {
		env.Trade(net.ID, alice, bob, aliceM, bobM, 1)
	}
	w := &world{t: t, env: env, h: api.NewHandlerWithOptions(env.Svc, api.Options{CSRFKey: testCSRFKey})}
	adminT := w.token(admin, domain.ScopeRead, domain.ScopeWrite)

	rec := w.post(adminT, "/exports", map[string]any{"scope": "ledger", "format": "csv"}, "")
	wantStatus(t, rec, 201)
	ex := decode[struct {
		ID, Status string
		SHA256     *string `json:"sha256"`
		Artefact   string  `json:"artefact_url"`
	}](t, rec)
	if ex.Status != "complete" || ex.SHA256 == nil {
		t.Fatalf("export = %s", rec.Body.String())
	}
	got := get(t, w, adminT, ex.Artefact)
	if got.Code != http.StatusOK || got.Header().Get("X-Content-SHA256") != *ex.SHA256 {
		t.Fatalf("artefact: %d sha %q want %q", got.Code, got.Header().Get("X-Content-SHA256"), *ex.SHA256)
	}
	zr, err := zip.NewReader(bytes.NewReader(got.Body.Bytes()), int64(got.Body.Len()))
	if err != nil {
		t.Fatalf("zip: %v", err)
	}
	var rows int
	for _, f := range zr.File {
		if f.Name != "entries.csv" {
			continue
		}
		rc, err := f.Open()
		if err != nil {
			t.Fatal(err)
		}
		recs, err := csv.NewReader(rc).ReadAll()
		_ = rc.Close()
		if err != nil {
			t.Fatalf("entries.csv: %v", err)
		}
		rows = len(recs) - 1 // header
	}
	if rows < 2*trades {
		t.Errorf("entries.csv has %d rows for %d trades", rows, trades)
	}
	// The export belongs to its creator: a member cannot fetch an admin's ledger export.
	bobT := w.token(bob, domain.ScopeRead, domain.ScopeWrite)
	if r := get(t, w, bobT, ex.Artefact); r.Code != http.StatusNotFound {
		t.Errorf("another member fetched the export: %d", r.Code)
	}
}

func get(t *testing.T, w *world, token, path string) *httptest.ResponseRecorder {
	t.Helper()
	return w.do(req{method: "GET", path: path, token: token})
}
