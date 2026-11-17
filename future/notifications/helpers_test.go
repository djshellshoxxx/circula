package notifications_test

import (
	"bytes"
	"context"
	"database/sql"
	"log/slog"
	"path/filepath"
	"sync"
	"testing"
	"time"

	_ "time/tzdata"

	n "github.com/djshellshoxxx/circula/future/notifications"
	"github.com/djshellshoxxx/circula/future/notifications/notifytest"
)

const (
	baseURL = "https://circula.example"
	net1    = "net-1"
	net2    = "net-2"
)

var t0 = time.Date(2026, 11, 16, 18, 0, 0, 0, time.UTC) // 10:00 in Vancouver

func testSecrets() n.Secrets {
	kr, err := n.NewKeyring("k1", map[string][]byte{"k1": bytes.Repeat([]byte{1}, 32)})
	if err != nil {
		panic(err)
	}
	return n.Secrets{Keyring: kr, IndexKey: bytes.Repeat([]byte{2}, 32), TokenKey: bytes.Repeat([]byte{3}, 32)}
}

// fakeEmail / fakePush record sends and let a test script outcomes.
type fakeEmail struct {
	mu   sync.Mutex
	sent []n.EmailMessage
	fn   func(m n.EmailMessage) error
}

func (f *fakeEmail) SendEmail(_ context.Context, m n.EmailMessage) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.fn != nil {
		if err := f.fn(m); err != nil {
			return err
		}
	}
	f.sent = append(f.sent, m)
	return nil
}
func (f *fakeEmail) Sent() []n.EmailMessage {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]n.EmailMessage(nil), f.sent...)
}

type fakePush struct {
	mu   sync.Mutex
	sent []n.PushMessage
	fn   func(m n.PushMessage) error
}

func (f *fakePush) SendPush(_ context.Context, m n.PushMessage) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.fn != nil {
		if err := f.fn(m); err != nil {
			return err
		}
	}
	f.sent = append(f.sent, m)
	return nil
}
func (f *fakePush) Sent() []n.PushMessage {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]n.PushMessage(nil), f.sent...)
}

type env struct {
	t     *testing.T
	path  string
	db    *sql.DB
	st    *n.Store
	dir   *notifytest.Directory
	clk   *notifytest.Clock
	svc   *n.Service
	log   *bytes.Buffer
	logMu *sync.Mutex
	email *fakeEmail
	push  *fakePush
	sec   n.Secrets
	audit []n.AuditRecord
	amu   sync.Mutex
}

type lockedWriter struct {
	mu *sync.Mutex
	b  *bytes.Buffer
}

func (w lockedWriter) Write(p []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.b.Write(p)
}

func openDB(t *testing.T, path string) *sql.DB {
	t.Helper()
	if !haveDriver {
		t.Skip("no SQLite driver linked: run with -tags sqlite_cgo_test (see VALIDATION.md)")
	}
	db, err := sql.Open(driverName, "file:"+path+dsnOpts)
	if err != nil {
		t.Fatal(err)
	}
	db.SetMaxOpenConns(8)
	t.Cleanup(func() { db.Close() })
	return db
}

func newEnv(t *testing.T) *env {
	t.Helper()
	e := &env{t: t, path: filepath.Join(t.TempDir(), "n.db"), sec: testSecrets(), email: &fakeEmail{}, push: &fakePush{}}
	e.db = openDB(t, e.path)
	e.st = n.NewStore(e.db)
	if err := e.st.Migrate(context.Background()); err != nil {
		t.Fatal(err)
	}
	e.dir = notifytest.NewDirectory()
	for _, nw := range []string{net1, net2} {
		for _, m := range []string{"alice", "bob"} {
			e.dir.SetActive(nw, m, true)
		}
	}
	e.clk = notifytest.NewClock(t0)
	e.log, e.logMu = &bytes.Buffer{}, &sync.Mutex{}
	e.svc = e.newService()
	return e
}

func (e *env) newService() *n.Service {
	e.t.Helper()
	lg := slog.New(slog.NewTextHandler(lockedWriter{e.logMu, e.log}, &slog.HandlerOptions{Level: slog.LevelDebug}))
	svc, err := n.NewService(n.Config{
		Store: e.st, Secrets: e.sec, Directory: e.dir, Clock: e.clk, BaseURL: baseURL, Logger: lg,
		Endpoints: n.EndpointPolicy{AllowedHosts: n.DefaultPushProviders},
		Audit: func(_ context.Context, r n.AuditRecord) {
			e.amu.Lock()
			e.audit = append(e.audit, r)
			e.amu.Unlock()
		},
	})
	if err != nil {
		e.t.Fatal(err)
	}
	return svc
}

func (e *env) disp(owner string, h n.Hooks) *n.Dispatcher {
	e.t.Helper()
	d, err := e.svc.NewDispatcher(n.DispatcherOptions{Owner: owner, Email: e.email, Push: e.push, Hooks: h})
	if err != nil {
		e.t.Fatal(err)
	}
	return d
}

func (e *env) logs() string { e.logMu.Lock(); defer e.logMu.Unlock(); return e.log.String() }

func actor(net, member string) n.Actor { return n.Actor{NetworkID: net, MemberID: member} }

func ptr[T any](v T) *T { return &v }

// optIn enables the kind on the channel with no quiet hours (UTC).
func (e *env) optIn(a n.Actor, channel string, kinds ...string) {
	e.t.Helper()
	var ups []n.PrefUpdate
	for _, k := range kinds {
		ups = append(ups, n.PrefUpdate{EventType: k, Channel: channel, Enabled: true})
	}
	if err := e.svc.SetPreferences(context.Background(), a, ups); err != nil {
		e.t.Fatal(err)
	}
}

// verifiedEmail registers and confirms an address by running the real flow.
func (e *env) verifiedEmail(a n.Actor, addr string) {
	e.t.Helper()
	ctx := context.Background()
	if _, err := e.svc.AddEmailTarget(ctx, a, addr); err != nil {
		e.t.Fatal(err)
	}
	d := e.disp("verify-"+a.MemberID, n.Hooks{})
	if _, err := d.RunOnce(ctx); err != nil {
		e.t.Fatal(err)
	}
	sent := e.email.Sent()
	if len(sent) == 0 {
		e.t.Fatal("no verification mail")
	}
	tok := tokenFrom(e.t, sent[len(sent)-1].Body, "/notifications/confirm-email?t=")
	if err := e.svc.ConfirmEmail(ctx, tok); err != nil {
		e.t.Fatal(err)
	}
}

func tokenFrom(t *testing.T, body, marker string) string {
	t.Helper()
	i := bytes.Index([]byte(body), []byte(marker))
	if i < 0 {
		t.Fatalf("no %q in body", marker)
	}
	rest := body[i+len(marker):]
	end := 0
	for end < len(rest) && rest[end] != '\n' && rest[end] != ' ' && rest[end] != '\r' {
		end++
	}
	return rest[:end]
}

// enqueue commits an event through a caller-owned transaction, like the host.
func (e *env) enqueue(ev n.Event) []n.Enqueued {
	e.t.Helper()
	var out []n.Enqueued
	err := e.st.Update(context.Background(), func(ctx context.Context, q n.DBTX) error {
		var err error
		out, err = e.svc.EnqueueTx(ctx, q, ev)
		return err
	})
	if err != nil {
		e.t.Fatal(err)
	}
	return out
}

func (e *env) count(query string, args ...any) int {
	e.t.Helper()
	var c int
	if err := e.db.QueryRow(query, args...).Scan(&c); err != nil {
		e.t.Fatal(err)
	}
	return c
}

func (e *env) status(id string) (string, string) {
	e.t.Helper()
	var s string
	var le sql.NullString
	if err := e.db.QueryRow(`SELECT status,last_error FROM notification_delivery WHERE id=?`, id).Scan(&s, &le); err != nil {
		e.t.Fatal(err)
	}
	return s, le.String
}

func (e *env) pushSub(a n.Actor, endpoint string) n.PushSubscriptionInfo {
	e.t.Helper()
	sub := notifytest.NewSubscriber()
	info, err := e.svc.AddPushSubscription(context.Background(), a, n.PushSubscriptionInput{Endpoint: endpoint, P256dh: sub.P256dh(), Auth: sub.AuthB64()})
	if err != nil {
		e.t.Fatal(err)
	}
	return info
}

const fcm = "https://fcm.googleapis.com/fcm/send/abc123-secret-token"
