package federation

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"database/sql"
	"fmt"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestAuditSafetyWritesCannotSilentlyDisappear(t *testing.T) {
	ctx := context.Background()
	for _, table := range []string{"discovery_inbox", "discovery_projection"} {
		t.Run(table, func(t *testing.T) {
			s := fixtureStore(t, filepath.Join(t.TempDir(), "db"))
			pub, key, _ := ed25519.GenerateKey(rand.Reader)
			fixturePeer(t, s, "local", "remote", "key", pub)
			if _, err := s.DB.Exec(`CREATE TRIGGER ignored_write BEFORE INSERT ON ` + table + ` BEGIN SELECT RAISE(IGNORE); END`); err != nil {
				t.Fatal(err)
			}
			wire, _ := SignDiscovery(fixtureEvent("event", "listing.publish", 1), "key", key)
			if _, err := s.Receive(ctx, "local", "remote", wire); err == nil {
				t.Fatal("silent write acknowledged")
			}
			var count int
			s.DB.QueryRow(`SELECT count(*) FROM discovery_inbox`).Scan(&count)
			if count != 0 {
				t.Fatal("inbox committed without projection")
			}
		})
	}
	s := fixtureStore(t, filepath.Join(t.TempDir(), "outbox"))
	pub, key, _ := ed25519.GenerateKey(rand.Reader)
	fixturePeer(t, s, "local", "remote", "key", pub)
	s.SigningKey = func(context.Context, string, string) (string, ed25519.PrivateKey, error) { return "key", key, nil }
	owner := Actor{"local", "owner"}
	if err := s.Consent(ctx, owner, "remote", "listing", true); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Queue(ctx, owner, "remote", "listing", "event", false); err != nil {
		t.Fatal(err)
	}
	if _, err := s.DB.Exec(`CREATE TRIGGER ignored_lease BEFORE UPDATE ON discovery_outbox BEGIN SELECT RAISE(IGNORE); END`); err != nil {
		t.Fatal(err)
	}
	sent := false
	err := s.Dispatch(ctx, "local", "remote", "event", deliveryFunc(func(context.Context, string, string) error { sent = true; return nil }))
	if err == nil || sent {
		t.Fatal("network send without durable lease", sent, err)
	}
}

func TestAuditSchemaMismatchRollsBackMigration(t *testing.T) {
	ctx := context.Background()
	db, err := sql.Open("sqlite", filepath.Join(t.TempDir(), "schema"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if _, err = db.Exec(`CREATE TABLE discovery_inbox(local TEXT,remote TEXT,id TEXT,hash TEXT,received INTEGER)`); err != nil {
		t.Fatal(err)
	}
	s := Store{DB: db}
	if err = s.Migrate(ctx); err == nil || !strings.Contains(err.Error(), "incompatible federation schema") {
		t.Fatal(err)
	}
	var count int
	if err = db.QueryRow(`SELECT count(*) FROM sqlite_schema WHERE name='discovery_network'`).Scan(&count); err != nil || count != 0 {
		t.Fatal("partial migration", count, err)
	}
}

func TestAuditDepartureCanWithdrawAllOrdinaryCapacity(t *testing.T) {
	ctx := context.Background()
	s := fixtureStore(t, filepath.Join(t.TempDir(), "reserve"))
	tx, err := s.DB.Begin()
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 1000; i++ {
		id := fmt.Sprintf("listing%d", i)
		if _, err = tx.Exec(`INSERT INTO discovery_outbox VALUES('local','remote',?,'owner',?,1,1,'listing.publish','synthetic-prior-wire',?,'sent',1,0)`, id, id, fixtureNow.Add(time.Hour).Unix()); err != nil {
			t.Fatal(err)
		}
		if _, err = tx.Exec(`INSERT INTO discovery_sequence VALUES('local','remote',?,1)`, id); err != nil {
			t.Fatal(err)
		}
		if _, err = tx.Exec(`INSERT INTO discovery_consent VALUES('local','remote',?,'owner',1)`, id); err != nil {
			t.Fatal(err)
		}
	}
	if err = tx.Commit(); err != nil {
		t.Fatal(err)
	}
	if err = s.Depart(ctx, Actor{"local", "admin"}, "owner"); err != nil {
		t.Fatal("withdrawal reserve blocks departure", err)
	}
	var consent, withdrawals, total int
	s.DB.QueryRow(`SELECT count(*) FROM discovery_consent WHERE enabled=1`).Scan(&consent)
	s.DB.QueryRow(`SELECT count(*) FROM discovery_outbox WHERE kind='listing.revoke'`).Scan(&withdrawals)
	s.DB.QueryRow(`SELECT count(*) FROM discovery_outbox`).Scan(&total)
	if consent != 0 || withdrawals != 1000 || total != 2000 {
		t.Fatal(consent, withdrawals, total)
	}
	if err = s.Depart(ctx, Actor{"local", "admin"}, "owner"); err != nil {
		t.Fatal("repeated departure", err)
	}
}

func TestAuditConsentTogglesWithoutPublicationConsumeNoReserve(t *testing.T) {
	ctx := context.Background()
	s := fixtureStore(t, filepath.Join(t.TempDir(), "toggle"))
	pub, _, _ := ed25519.GenerateKey(rand.Reader)
	fixturePeer(t, s, "local", "remote", "key", pub)
	owner := Actor{"local", "owner"}
	for i := 0; i < 150; i++ {
		if err := s.Consent(ctx, owner, "remote", "listing", true); err != nil {
			t.Fatal(err)
		}
		if err := s.Consent(ctx, owner, "remote", "listing", false); err != nil {
			t.Fatal(err)
		}
	}
	var count int
	s.DB.QueryRow(`SELECT count(*) FROM discovery_outbox`).Scan(&count)
	if count != 0 {
		t.Fatal("empty consent generated withdrawals", count)
	}
}

func TestAuditInboxPeerAndOwnerQuotaBoundaries(t *testing.T) {
	ctx := context.Background()
	s := fixtureStore(t, filepath.Join(t.TempDir(), "quota"))
	pub, key, _ := ed25519.GenerateKey(rand.Reader)
	fixturePeer(t, s, "local", "remote", "key", pub)
	if _, err := s.DB.Exec(`WITH RECURSIVE n(i) AS (VALUES(1) UNION ALL SELECT i+1 FROM n WHERE i<10000) INSERT INTO discovery_inbox SELECT 'local','remote','seed'||i,'hash',0 FROM n`); err != nil {
		t.Fatal(err)
	}
	wire, _ := SignDiscovery(fixtureEvent("overflow", "listing.publish", 1), "key", key)
	if _, err := s.Receive(ctx, "local", "remote", wire); err == nil || !strings.Contains(err.Error(), "inbox quota") {
		t.Fatal("inbox boundary", err)
	}
	var count int
	s.DB.QueryRow(`SELECT count(*) FROM discovery_inbox`).Scan(&count)
	if count != 10000 {
		t.Fatal(count)
	}
	for i := 1; i < 100; i++ {
		if _, err := s.DB.Exec(`INSERT INTO discovery_peers VALUES('local',?,'https://example.com/events',0)`, fmt.Sprintf("peer%d", i)); err != nil {
			t.Fatal(err)
		}
	}
	err := s.ApprovePeer(ctx, Actor{"local", "admin"}, PeerConfig{"local", "extra", "https://example.com/events", true}, TrustedKey{"key", pub, fixtureNow, fixtureNow.Add(time.Hour), false})
	if err == nil || !strings.Contains(err.Error(), "peer quota") {
		t.Fatal("peer boundary", err)
	}
}

func TestAuditCorruptProjectionFailsClosed(t *testing.T) {
	ctx := context.Background()
	s := fixtureStore(t, filepath.Join(t.TempDir(), "corrupt"))
	pub, key, _ := ed25519.GenerateKey(rand.Reader)
	fixturePeer(t, s, "local", "remote", "key", pub)
	wire, _ := SignDiscovery(fixtureEvent("event", "listing.publish", 1), "key", key)
	if _, err := s.Receive(ctx, "local", "remote", wire); err != nil {
		t.Fatal(err)
	}
	if _, err := s.DB.Exec(`UPDATE discovery_projection SET content='{"title":"x","description":"","category":"","area":"","private":"unexpected"}'`); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Browse(ctx, Actor{"local", "owner"}); err == nil {
		t.Fatal("corrupt schema leaked")
	}
}

func TestAuditConsentAndOrdinaryOutboxLimits(t *testing.T) {
	ctx := context.Background()
	s := fixtureStore(t, filepath.Join(t.TempDir(), "bounds"))
	pub, key, _ := ed25519.GenerateKey(rand.Reader)
	fixturePeer(t, s, "local", "remote", "key", pub)
	owner := Actor{"local", "owner"}
	if err := s.Consent(ctx, owner, "unapproved", "listing", true); err == nil {
		t.Fatal("unapproved peer consent")
	}
	if err := s.Consent(ctx, owner, "unapproved", "listing", false); err != nil {
		t.Fatal(err)
	}
	var count int
	s.DB.QueryRow(`SELECT count(*) FROM discovery_consent`).Scan(&count)
	if count != 0 {
		t.Fatal("absent optout allocated data")
	}
	if _, err := s.DB.Exec(`WITH RECURSIVE n(i) AS(VALUES(1) UNION ALL SELECT i+1 FROM n WHERE i<1000) INSERT INTO discovery_consent SELECT 'local','remote','seed'||i,'owner',1 FROM n`); err != nil {
		t.Fatal(err)
	}
	if err := s.Consent(ctx, owner, "remote", "listing", true); err == nil || !strings.Contains(err.Error(), "consent quota") {
		t.Fatal("consent bound", err)
	}
	s.DB.Exec(`DELETE FROM discovery_consent`)
	if err := s.Consent(ctx, owner, "remote", "listing", true); err != nil {
		t.Fatal(err)
	}
	s.SigningKey = func(context.Context, string, string) (string, ed25519.PrivateKey, error) { return "key", key, nil }
	if _, err := s.DB.Exec(`WITH RECURSIVE n(i) AS(VALUES(1) UNION ALL SELECT i+1 FROM n WHERE i<120) INSERT INTO discovery_outbox SELECT 'local','remote','seed'||i,'owner','listing',i,i,'listing.publish','wire',?,'sent',1,0 FROM n`, fixtureNow.Add(24*time.Hour).Unix()); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Queue(ctx, owner, "remote", "listing", "hour-overflow", false); err == nil || !strings.Contains(err.Error(), "rate limit") {
		t.Fatal("owner rate", err)
	}
	if _, err := s.DB.Exec(`WITH RECURSIVE n(i) AS(VALUES(121) UNION ALL SELECT i+1 FROM n WHERE i<1000) INSERT INTO discovery_outbox SELECT 'local','remote','seed'||i,'owner','listing',i,i,'listing.publish','wire',?,'sent',1,0 FROM n`, fixtureNow.Add(24*time.Hour).Unix()); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Queue(ctx, owner, "remote", "listing", "outbox-overflow", false); err == nil || !strings.Contains(err.Error(), "outbox quota") {
		t.Fatal("ordinary outbox bound", err)
	}
}

func TestAuditCanonicalBase64Required(t *testing.T) {
	_, key, _ := ed25519.GenerateKey(rand.Reader)
	wire, err := SignDiscovery(fixtureEvent("event", "listing.publish", 1), "key", key)
	if err != nil {
		t.Fatal(err)
	}
	parts := strings.Split(wire, ".")
	const alphabet = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789-_"
	last := parts[2][len(parts[2])-1]
	index := strings.IndexByte(alphabet, last)
	parts[2] = parts[2][:len(parts[2])-1] + string(alphabet[index+1])
	if _, err = WireKeyID(strings.Join(parts, ".")); err == nil {
		t.Fatal("noncanonical base64")
	}
}

func TestAuditMigrationRejectsViewCollision(t *testing.T) {
	db, err := sql.Open("sqlite", filepath.Join(t.TempDir(), "view"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if _, err = db.Exec(`CREATE VIEW discovery_network AS SELECT 'local' AS local,1 AS enabled`); err != nil {
		t.Fatal(err)
	}
	if err = (Store{DB: db}).Migrate(context.Background()); err == nil {
		t.Fatal("view accepted as mutable safety table")
	}
}
func TestAuditEndpointRejectsEmptyQuery(t *testing.T) {
	if ValidateEndpoint("https://example.com/events?") == nil {
		t.Fatal("empty query marker outside fixed endpoint profile")
	}
}

func TestAuditDeletedListingWithdrawsWithoutSourceOrSigner(t *testing.T) {
	ctx := context.Background()
	s := fixtureStore(t, filepath.Join(t.TempDir(), "delete"))
	pub, key, _ := ed25519.GenerateKey(rand.Reader)
	fixturePeer(t, s, "local", "remote", "key", pub)
	owner := Actor{"local", "owner"}
	if err := s.Consent(ctx, owner, "remote", "listing", true); err != nil {
		t.Fatal(err)
	}
	s.SigningKey = func(context.Context, string, string) (string, ed25519.PrivateKey, error) { return "key", key, nil }
	if _, err := s.Queue(ctx, owner, "remote", "listing", "publish", false); err != nil {
		t.Fatal(err)
	}
	s.SigningKey = nil
	s.Project = func(context.Context, *sql.Conn, Actor, string) (LocalListing, error) {
		return LocalListing{}, ErrInvalid
	}
	if err := s.WithdrawListing(ctx, owner, "listing"); err == nil {
		t.Fatal("ordinary member invoked system withdrawal")
	}
	if err := s.WithdrawListing(ctx, Actor{"other", "admin"}, "listing"); err != nil {
		t.Fatal(err)
	}
	var enabled bool
	s.DB.QueryRow(`SELECT enabled FROM discovery_consent WHERE local='local' AND remote='remote' AND listing='listing'`).Scan(&enabled)
	if !enabled {
		t.Fatal("cross-tenant withdrawal")
	}
	if _, err := s.DB.Exec(`CREATE TRIGGER ignore_audit BEFORE INSERT ON discovery_audit BEGIN SELECT RAISE(IGNORE);END`); err != nil {
		t.Fatal(err)
	}
	if err := s.WithdrawListing(ctx, Actor{"local", "admin"}, "listing"); err == nil {
		t.Fatal("missing audit not detected")
	}
	s.DB.QueryRow(`SELECT enabled FROM discovery_consent WHERE local='local' AND remote='remote' AND listing='listing'`).Scan(&enabled)
	if !enabled {
		t.Fatal("partially committed lifecycle operation")
	}
	s.DB.Exec(`DROP TRIGGER ignore_audit`)
	if err := RunCLI(ctx, s, Actor{"local", "admin"}, []string{"withdraw-listing", "listing"}, nil); err != nil {
		t.Fatal(err)
	}
	rows, err := s.Pending(ctx, Actor{"local", "admin"}, "", "", 20)
	if err != nil || len(rows) != 1 || rows[0].State != "unsealed" {
		t.Fatal(rows, err)
	}
	if err = s.WithdrawListing(ctx, Actor{"local", "admin"}, "listing"); err != nil {
		t.Fatal(err)
	}
}

func TestAuditCorruptOutboxCannotBypassProtocolBinding(t *testing.T) {
	ctx := context.Background()
	s := fixtureStore(t, filepath.Join(t.TempDir(), "outgoing"))
	pub, key, _ := ed25519.GenerateKey(rand.Reader)
	fixturePeer(t, s, "local", "remote", "key", pub)
	s.SigningKey = func(context.Context, string, string) (string, ed25519.PrivateKey, error) { return "key", key, nil }
	owner := Actor{"local", "owner"}
	if err := s.Consent(ctx, owner, "remote", "listing", true); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Queue(ctx, owner, "remote", "listing", "event", false); err != nil {
		t.Fatal(err)
	}
	event := Discovery{Version: 2, ID: "different-event", Issuer: "local", Audience: "remote", Kind: "listing.revoke", Listing: "listing", Revision: 1, Issued: fixtureNow.Unix(), Expires: fixtureNow.Add(time.Hour).Unix()}
	wire, err := SignDiscovery(event, "key", key)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = s.DB.Exec(`UPDATE discovery_outbox SET kind='listing.revoke',wire=?,expires=? WHERE id='event'`, wire, event.Expires); err != nil {
		t.Fatal(err)
	}
	sent := false
	if err = s.Dispatch(ctx, "local", "remote", "event", deliveryFunc(func(context.Context, string, string) error { sent = true; return nil })); err == nil || sent {
		t.Fatal("corrupt outbound identity released", sent, err)
	}
}
