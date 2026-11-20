package federation

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"database/sql"
	"errors"
	_ "modernc.org/sqlite"
	"net/http"
	"net/http/httptest"
	"net/netip"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

func fixtureStore(t *testing.T, path string) Store {
	t.Helper()
	db, e := sql.Open("sqlite", path+"?_pragma=busy_timeout(5000)")
	if e != nil {
		t.Fatal(e)
	}
	db.SetMaxOpenConns(4)
	t.Cleanup(func() { db.Close() })
	s := Store{DB: db, Now: func() time.Time { return fixtureNow }, Authorize: func(_ context.Context, _ *sql.Conn, a Actor, admin bool) error {
		if a.Member != "owner" && a.Member != "admin" || admin && a.Member != "admin" {
			return ErrInvalid
		}
		return nil
	}, Project: func(_ context.Context, _ *sql.Conn, a Actor, id string) (LocalListing, error) {
		if id != "listing" {
			return LocalListing{}, ErrInvalid
		}
		return LocalListing{"owner", true, true, 1, SharedListing{"Bike", "<script>private</script>", "tools", "north"}}, nil
	}}
	if e = s.Migrate(context.Background()); e != nil {
		t.Fatal(e)
	}
	return s
}
func fixturePeer(t *testing.T, s Store, local, remote, kid string, key ed25519.PublicKey) {
	t.Helper()
	a := Actor{local, "admin"}
	ctx := context.Background()
	if e := s.EnableNetwork(ctx, a, true); e != nil {
		t.Fatal(e)
	}
	if e := s.ApprovePeer(ctx, a, PeerConfig{local, remote, "https://example.com/federation/v2/events", true}, TrustedKey{kid, key, fixtureNow.Add(-time.Hour), fixtureNow.Add(48 * time.Hour), false}); e != nil {
		t.Fatal(e)
	}
}
func fixtureEvent(id, kind string, revision int64) Discovery {
	e := Discovery{2, id, "remote", "local", kind, "listing", revision, fixtureNow.Unix(), fixtureNow.Add(time.Hour).Unix(), nil}
	if kind == "listing.publish" {
		e.Content = &SharedListing{"Bike", "Description", "tools", "north"}
	}
	return e
}
func TestV2ProtocolFailClosed(t *testing.T) {
	pub, key, _ := ed25519.GenerateKey(rand.Reader)
	e := fixtureEvent("event", "listing.publish", 1)
	wire, err := SignDiscovery(e, "key", key)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = VerifyDiscovery(wire, "local", "remote", "key", pub, fixtureNow); err != nil {
		t.Fatal(err)
	}
	for _, args := range [][3]string{{"wrong", "remote", "key"}, {"local", "wrong", "key"}, {"local", "remote", "wrong"}} {
		if _, err = VerifyDiscovery(wire, args[0], args[1], args[2], pub, fixtureNow); err == nil {
			t.Fatal("identity")
		}
	}
	if _, err = VerifyDiscovery(wire, "local", "remote", "key", pub, fixtureNow.Add(time.Hour)); err == nil {
		t.Fatal("expiry")
	}
	e.Kind = "ledger.post"
	if _, err = SignDiscovery(e, "key", key); err == nil {
		t.Fatal("settlement")
	}
	if _, err = WireKeyID(strings.Repeat("x", MaxWire+1)); err == nil {
		t.Fatal("bound")
	}
}
func TestInboxConcurrencyRestartAndRevocation(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "federation.db")
	s := fixtureStore(t, path)
	pub, key, _ := ed25519.GenerateKey(rand.Reader)
	fixturePeer(t, s, "local", "remote", "key", pub)
	wire, _ := SignDiscovery(fixtureEvent("event", "listing.publish", 1), "key", key)
	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if _, err := s.Receive(ctx, "local", "remote", wire); err != nil {
				t.Error(err)
			}
		}()
	}
	wg.Wait()
	var count int
	if e := s.DB.QueryRow(`SELECT count(*) FROM discovery_inbox`).Scan(&count); e != nil || count != 1 {
		t.Fatal(count, e)
	}
	s.DB.Close()
	s = fixtureStore(t, path)
	if duplicate, e := s.Receive(ctx, "local", "remote", wire); e != nil || !duplicate {
		t.Fatal(duplicate, e)
	}
	revoke, _ := SignDiscovery(fixtureEvent("revoked", "listing.revoke", 3), "key", key)
	if _, e := s.Receive(ctx, "local", "remote", revoke); e != nil {
		t.Fatal(e)
	}
	stale, _ := SignDiscovery(fixtureEvent("stale", "listing.publish", 2), "key", key)
	if _, e := s.Receive(ctx, "local", "remote", stale); e != nil {
		t.Fatal(e)
	}
	rows, e := s.Browse(ctx, Actor{"local", "owner"})
	if e != nil || len(rows) != 0 {
		t.Fatal(rows, e)
	}
	conflict := fixtureEvent("conflict", "listing.publish", 3)
	wireConflict, _ := SignDiscovery(conflict, "key", key)
	if _, e = s.Receive(ctx, "local", "remote", wireConflict); e == nil {
		t.Fatal("revision conflict")
	}
	if _, e = s.Receive(ctx, "wrong", "remote", wire); e == nil {
		t.Fatal("tenant")
	}
	if e = s.RevokeKey(ctx, Actor{"local", "admin"}, "remote", "key"); e != nil {
		t.Fatal(e)
	}
	if _, e = s.Receive(ctx, "local", "remote", wire); e == nil {
		t.Fatal("revoked key")
	}
}
func TestInboxAtomicRollbackAndEquivocation(t *testing.T) {
	ctx := context.Background()
	s := fixtureStore(t, filepath.Join(t.TempDir(), "db"))
	pub, key, _ := ed25519.GenerateKey(rand.Reader)
	fixturePeer(t, s, "local", "remote", "key", pub)
	event := fixtureEvent("event", "listing.publish", 1)
	wire, _ := SignDiscovery(event, "key", key)
	if _, e := s.DB.Exec(`CREATE TRIGGER fail_projection BEFORE INSERT ON discovery_projection BEGIN SELECT RAISE(ABORT,'synthetic failure');END`); e != nil {
		t.Fatal(e)
	}
	if _, e := s.Receive(ctx, "local", "remote", wire); e == nil {
		t.Fatal("rollback")
	}
	var count int
	s.DB.QueryRow(`SELECT count(*) FROM discovery_inbox`).Scan(&count)
	if count != 0 {
		t.Fatal("partial replay journal")
	}
	s.DB.Exec(`DROP TRIGGER fail_projection`)
	if _, e := s.Receive(ctx, "local", "remote", wire); e != nil {
		t.Fatal(e)
	}
	event.Content.Title = "Changed"
	changed, _ := SignDiscovery(event, "key", key)
	if _, e := s.Receive(ctx, "local", "remote", changed); e == nil {
		t.Fatal("equivocation")
	}
}

type deliveryFunc func(context.Context, string, string) error

func (f deliveryFunc) Send(c context.Context, e, w string) error { return f(c, e, w) }
func TestOutboxConsentIdempotencyAndPartition(t *testing.T) {
	ctx := context.Background()
	s := fixtureStore(t, filepath.Join(t.TempDir(), "db"))
	pub, key, _ := ed25519.GenerateKey(rand.Reader)
	fixturePeer(t, s, "local", "remote", "key", pub)
	s.SigningKey = func(context.Context, string, string) (string, ed25519.PrivateKey, error) { return "key", key, nil }
	a := Actor{"local", "owner"}
	if _, e := s.Queue(ctx, a, "remote", "listing", "event", false); e == nil {
		t.Fatal("missing consent")
	}
	if e := s.Consent(ctx, a, "remote", "listing", true); e != nil {
		t.Fatal(e)
	}
	wire, e := s.Queue(ctx, a, "remote", "listing", "event", false)
	if e != nil {
		t.Fatal(e)
	}
	retry, e := s.Queue(ctx, a, "remote", "listing", "event", false)
	if e != nil || wire != retry {
		t.Fatal("idempotency", e)
	}
	calls := 0
	f := deliveryFunc(func(context.Context, string, string) error { calls++; return errors.New("partition") })
	if e = s.Dispatch(ctx, "local", "remote", "event", f); e == nil || calls != 1 {
		t.Fatal(e, calls)
	}
	if e = s.Dispatch(ctx, "local", "remote", "event", f); e == nil || calls != 1 {
		t.Fatal("backoff", e, calls)
	}
	s.Now = func() time.Time { return fixtureNow.Add(2 * time.Minute) }
	success := deliveryFunc(func(context.Context, string, string) error { calls++; return nil })
	if e = s.Dispatch(ctx, "local", "remote", "event", success); e != nil {
		t.Fatal(e)
	}
	if e = s.Dispatch(ctx, "local", "remote", "event", success); e != nil || calls != 2 {
		t.Fatal("sent retry", e, calls)
	}
	if _, e = s.Queue(ctx, a, "remote", "listing", "revoke", true); e != nil {
		t.Fatal(e)
	}
	if _, e = s.Queue(ctx, a, "remote", "listing", "another", false); e == nil {
		t.Fatal("withdrawal")
	}
}
func TestInterfacesAndSSRF(t *testing.T) {
	for _, raw := range []string{"http://example.com/x", "https://127.0.0.1/x", "https://[::ffff:10.0.0.1]/x", "https://169.254.169.254/x", "https://user:pass@example.com/x", "https://example.com:8443/x", "https://example.com/x?token=a"} {
		if ValidateEndpoint(raw) == nil {
			t.Fatal(raw)
		}
	}
	for _, raw := range []string{"10.0.0.1", "100.64.0.1", "192.0.2.1", "2001:db8::1", "::1", "ff02::1", "::ffff:127.0.0.1"} {
		if PublicAddress(netip.MustParseAddr(raw)) {
			t.Fatal(raw)
		}
	}
	if !PublicAddress(netip.MustParseAddr("1.1.1.1")) {
		t.Fatal("public")
	}
	s := fixtureStore(t, filepath.Join(t.TempDir(), "db"))
	h := Admin{Store: s, Auth: func(r *http.Request, mutate bool) (Actor, error) {
		if mutate && r.Form.Get("csrf") != "token" {
			return Actor{}, ErrInvalid
		}
		return Actor{"local", "owner"}, nil
	}}
	req := httptest.NewRequest("POST", "/", strings.NewReader("action=enable"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Code != 403 {
		t.Fatal(w.Code)
	}
	var out bytes.Buffer
	if e := RenderListings(&out, []RemoteListing{{Content: SharedListing{Title: "<script>"}}}); e != nil || strings.Contains(out.String(), "<script>") {
		t.Fatal(out.String(), e)
	}
	ingress := &Ingress{Store: s, Local: "local", Remote: "remote"}
	w = httptest.NewRecorder()
	ingress.ServeHTTP(w, httptest.NewRequest("GET", "/", nil))
	if w.Code != 405 {
		t.Fatal(w.Code)
	}
}

type fakeResolver struct{ ips []netip.Addr }

func (f fakeResolver) LookupNetIP(context.Context, string, string) ([]netip.Addr, error) {
	return f.ips, nil
}
func TestNoPrivateEgressMixedDNS(t *testing.T) {
	h := HTTPSDelivery{fakeResolver{[]netip.Addr{netip.MustParseAddr("1.1.1.1"), netip.MustParseAddr("127.0.0.1")}}}
	if e := h.Send(context.Background(), "https://example.com/federation/v2/events", "fakewire"); e == nil {
		t.Fatal("mixed private address accepted")
	}
	if PublicAddress(netip.MustParseAddr("3fff::1")) || PublicAddress(netip.MustParseAddr("4000::1")) {
		t.Fatal("reserved IPv6")
	}
}
func TestWithdrawalSurvivesDepartureAndSignerOutage(t *testing.T) {
	ctx := context.Background()
	s := fixtureStore(t, filepath.Join(t.TempDir(), "db"))
	pub, key, _ := ed25519.GenerateKey(rand.Reader)
	fixturePeer(t, s, "local", "remote", "key", pub)
	a := Actor{"local", "owner"}
	if e := s.Consent(ctx, a, "remote", "listing", true); e != nil {
		t.Fatal(e)
	}
	s.SigningKey = func(context.Context, string, string) (string, ed25519.PrivateKey, error) { return "key", key, nil }
	if _, e := s.Queue(ctx, a, "remote", "listing", "prior-publish", false); e != nil {
		t.Fatal(e)
	}
	s.SigningKey = nil // withdrawal must not depend on signer availability.
	if e := s.Consent(ctx, a, "remote", "listing", false); e != nil {
		t.Fatal(e)
	}
	pending, e := s.Pending(ctx, Actor{"local", "admin"}, "", "", 20)
	if e != nil || len(pending) != 1 || pending[0].State != "unsealed" {
		t.Fatal(pending, e)
	}
	if e = s.Consent(ctx, a, "remote", "listing", false); e != nil {
		t.Fatal(e)
	}
	s.Now = func() time.Time { return fixtureNow.Add(48 * time.Hour) }
	if e = s.Maintenance(ctx, Actor{"local", "admin"}); e != nil {
		t.Fatal(e)
	}
	pending, e = s.Pending(ctx, Actor{"local", "admin"}, "", "", 20)
	if e != nil || len(pending) != 1 || pending[0].State != "unsealed" {
		t.Fatal(pending, e)
	}
	s.SigningKey = func(context.Context, string, string) (string, ed25519.PrivateKey, error) { return "key", key, nil }
	s.Authorize = func(_ context.Context, _ *sql.Conn, a Actor, admin bool) error {
		if a.Member == "admin" {
			return nil
		}
		return ErrInvalid
	}
	received := false
	delivery := deliveryFunc(func(_ context.Context, endpoint, wire string) error {
		event, e := VerifyDiscovery(wire, "remote", "local", "key", pub, s.Now())
		if e != nil {
			return e
		}
		if event.Kind != "listing.revoke" || event.Content != nil {
			t.Fatal(event)
		}
		received = true
		return nil
	})
	if e = s.Dispatch(ctx, "local", "remote", pending[0].ID, delivery); e != nil || !received {
		t.Fatal(received, e)
	}
}
func TestRotationRevocationAndReadPrivacy(t *testing.T) {
	ctx := context.Background()
	s := fixtureStore(t, filepath.Join(t.TempDir(), "db"))
	pub, key, _ := ed25519.GenerateKey(rand.Reader)
	fixturePeer(t, s, "local", "remote", "old", pub)
	wire, _ := SignDiscovery(fixtureEvent("event", "listing.publish", 1), "old", key)
	if _, e := s.Receive(ctx, "local", "remote", wire); e != nil {
		t.Fatal(e)
	}
	rows, e := s.Browse(ctx, Actor{"local", "owner"})
	if e != nil || len(rows) != 1 {
		t.Fatal(rows, e)
	}
	if e = s.RevokeKey(ctx, Actor{"local", "admin"}, "remote", "old"); e != nil {
		t.Fatal(e)
	}
	rows, e = s.Browse(ctx, Actor{"local", "owner"})
	if e != nil || len(rows) != 0 {
		t.Fatal(rows, e)
	}
	newpub, newkey, _ := ed25519.GenerateKey(rand.Reader)
	if e = s.ApprovePeer(ctx, Actor{"local", "admin"}, PeerConfig{"local", "remote", "https://example.com/federation/v2/events", true}, TrustedKey{"new", newpub, fixtureNow.Add(-time.Hour), fixtureNow.Add(48 * time.Hour), false}); e != nil {
		t.Fatal(e)
	}
	updated, _ := SignDiscovery(fixtureEvent("updated", "listing.publish", 2), "new", newkey)
	if _, e = s.Receive(ctx, "local", "remote", updated); e != nil {
		t.Fatal(e)
	}
	rows, e = s.BrowsePage(ctx, Actor{"local", "owner"}, "", "", 1)
	if e != nil || len(rows) != 1 || rows[0].Revision != 2 {
		t.Fatal(rows, e)
	}
	if e = s.RevokePeer(ctx, Actor{"local", "admin"}, "remote"); e != nil {
		t.Fatal(e)
	}
	rows, e = s.Browse(ctx, Actor{"local", "owner"})
	if e != nil || len(rows) != 0 {
		t.Fatal(rows, e)
	}
}
func TestMembersNoJSAndAdminControls(t *testing.T) {
	ctx := context.Background()
	s := fixtureStore(t, filepath.Join(t.TempDir(), "db"))
	pub, key, _ := ed25519.GenerateKey(rand.Reader)
	fixturePeer(t, s, "local", "remote", "key", pub)
	s.SigningKey = func(context.Context, string, string) (string, ed25519.PrivateKey, error) { return "key", key, nil }
	auth := func(r *http.Request, mutation bool) (Actor, error) {
		if mutation && r.Form.Get("csrf") != "token" {
			return Actor{}, ErrInvalid
		}
		member := r.Header.Get("X-Fixture-Member")
		if member == "" {
			member = "owner"
		}
		return Actor{"local", member}, nil
	}
	h := Members{Store: s, Auth: auth, CSRFToken: func(*http.Request) string { return "token" }}
	r := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != 200 || !strings.Contains(w.Body.String(), `method="post"`) {
		t.Fatal(w.Code, w.Body.String())
	}
	for _, form := range []string{"csrf=token&action=consent&remote=remote&listing=listing", "csrf=token&action=share&remote=remote&listing=listing&intent=event", "csrf=token&action=withdraw&remote=remote&listing=listing"} {
		r = httptest.NewRequest("POST", "/", strings.NewReader(form))
		r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		w = httptest.NewRecorder()
		h.ServeHTTP(w, r)
		if w.Code != 204 {
			t.Fatal(w.Code, w.Body.String())
		}
	}
	r = httptest.NewRequest("POST", "/", strings.NewReader("action=share"))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w = httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != 403 {
		t.Fatal(w.Code)
	}
	admin := Admin{Store: s, Auth: auth}
	r = httptest.NewRequest("GET", "/", nil)
	r.Header.Set("X-Fixture-Member", "admin")
	w = httptest.NewRecorder()
	admin.ServeHTTP(w, r)
	if w.Code != 200 {
		t.Fatal(w.Code, w.Body.String())
	}
	for _, args := range [][]string{{"status"}, {"browse"}, {"disable"}, {"enable"}, {"maintenance"}} {
		var out bytes.Buffer
		if e := RunCLI(ctx, s, Actor{"local", "admin"}, args, &out); e != nil {
			t.Fatal(args, e)
		}
	}
	if e := s.Depart(ctx, Actor{"local", "admin"}, "owner"); e != nil {
		t.Fatal(e)
	}
}
func TestIngressResourceAndTrustBoundaries(t *testing.T) {
	ctx := context.Background()
	s := fixtureStore(t, filepath.Join(t.TempDir(), "db"))
	pub, key, _ := ed25519.GenerateKey(rand.Reader)
	fixturePeer(t, s, "local", "remote", "key", pub)
	wire, _ := SignDiscovery(fixtureEvent("event", "listing.publish", 1), "key", key)
	h := &Ingress{Store: s, Local: "local", Remote: "remote"}
	for _, c := range []struct {
		kind, body string
		status     int
	}{{"text/plain", wire, 415}, {"application/circula-discovery+jws", strings.Repeat("a", MaxWire+1), 413}, {"application/circula-discovery+jws", "a.b.c", 400}, {"application/circula-discovery+jws", wire, 200}} {
		r := httptest.NewRequest("POST", "/", strings.NewReader(c.body))
		r.Header.Set("Content-Type", c.kind)
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		if w.Code != c.status {
			t.Fatal(c.status, w.Code, w.Body.String())
		}
	}
	h.limit.window = time.Now().Unix() / 60
	h.limit.count = 240
	r := httptest.NewRequest("POST", "/", strings.NewReader(wire))
	r.Header.Set("Content-Type", "application/circula-discovery+jws")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != 429 {
		t.Fatal(w.Code)
	}
	if e := s.EnableNetwork(ctx, Actor{"local", "admin"}, false); e != nil {
		t.Fatal(e)
	}
	if _, e := s.Receive(ctx, "local", "remote", wire); e == nil {
		t.Fatal("disabled")
	}
}
func TestDepartureAndExpiredBodies(t *testing.T) {
	ctx := context.Background()
	s := fixtureStore(t, filepath.Join(t.TempDir(), "db"))
	pub, key, _ := ed25519.GenerateKey(rand.Reader)
	fixturePeer(t, s, "local", "remote", "key", pub)
	s.SigningKey = func(context.Context, string, string) (string, ed25519.PrivateKey, error) { return "key", key, nil }
	a := Actor{"local", "owner"}
	if e := s.Consent(ctx, a, "remote", "listing", true); e != nil {
		t.Fatal(e)
	}
	if _, e := s.Queue(ctx, a, "remote", "listing", "event", false); e != nil {
		t.Fatal(e)
	}
	if e := s.Depart(ctx, Actor{"local", "admin"}, "owner"); e != nil {
		t.Fatal(e)
	}
	pending, e := s.Pending(ctx, Actor{"local", "admin"}, "", "", 20)
	if e != nil || len(pending) != 1 || pending[0].Kind != "listing.revoke" {
		t.Fatal(pending, e)
	}
	sent := false
	if e = s.Dispatch(ctx, "local", "remote", "event", deliveryFunc(func(context.Context, string, string) error { sent = true; return nil })); e == nil || sent {
		t.Fatal(e, sent)
	}
	s.Now = func() time.Time { return fixtureNow.Add(48 * time.Hour) }
	if e = s.Maintenance(ctx, Actor{"local", "admin"}); e != nil {
		t.Fatal(e)
	}
	var wire, state string
	if e = s.DB.QueryRow(`SELECT wire,state FROM discovery_outbox WHERE id='event'`).Scan(&wire, &state); e != nil || wire != "" || state != "expired" {
		t.Fatal(wire, state, e)
	}
}

func TestQuotaAndTrustedPeerRate(t *testing.T) {
	ctx := context.Background()
	s := fixtureStore(t, filepath.Join(t.TempDir(), "db"))
	pub, key, _ := ed25519.GenerateKey(rand.Reader)
	fixturePeer(t, s, "local", "remote", "key", pub)
	wire, _ := SignDiscovery(fixtureEvent("event", "listing.publish", 1), "key", key)
	for i := 0; i < 120; i++ {
		if _, e := s.Receive(ctx, "local", "remote", wire); e != nil {
			t.Fatal(i, e)
		}
	}
	if _, e := s.Receive(ctx, "local", "remote", wire); e == nil {
		t.Fatal("trusted-peer rate limit")
	}
	s.Now = func() time.Time { return fixtureNow.Add(time.Minute) }
	if duplicate, e := s.Receive(ctx, "local", "remote", wire); e != nil || !duplicate {
		t.Fatal(duplicate, e)
	}
	_, e := s.DB.Exec(`WITH RECURSIVE n(x) AS (VALUES(1) UNION ALL SELECT x+1 FROM n WHERE x<9999) INSERT INTO discovery_inbox SELECT 'local','remote','old-'||x,'fixture-hash',0 FROM n`)
	if e != nil {
		t.Fatal(e)
	}
	fresh, _ := SignDiscovery(fixtureEvent("fresh", "listing.publish", 2), "key", key)
	if _, e = s.Receive(ctx, "local", "remote", fresh); e == nil {
		t.Fatal("inbox capacity")
	}
	if _, e = s.Receive(ctx, "local", "remote", wire); e != nil {
		t.Fatal("duplicate should remain idempotent at quota", e)
	}
}
func TestClosedDBBackupRestore(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "source.db")
	s := fixtureStore(t, path)
	pub, key, _ := ed25519.GenerateKey(rand.Reader)
	fixturePeer(t, s, "local", "remote", "key", pub)
	wire, _ := SignDiscovery(fixtureEvent("event", "listing.publish", 1), "key", key)
	if _, e := s.Receive(ctx, "local", "remote", wire); e != nil {
		t.Fatal(e)
	}
	if e := s.DB.Close(); e != nil {
		t.Fatal(e)
	}
	backup, e := os.ReadFile(path)
	if e != nil {
		t.Fatal(e)
	}
	restored := filepath.Join(t.TempDir(), "restored.db")
	if e = os.WriteFile(restored, backup, 0600); e != nil {
		t.Fatal(e)
	}
	s = fixtureStore(t, restored)
	if duplicate, e := s.Receive(ctx, "local", "remote", wire); e != nil || !duplicate {
		t.Fatal(duplicate, e)
	}
	rows, e := s.Browse(ctx, Actor{"local", "owner"})
	if e != nil || len(rows) != 1 {
		t.Fatal(rows, e)
	}
}
