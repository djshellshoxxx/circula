package web

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/djshellshoxxx/circula/internal/domain"
	"github.com/djshellshoxxx/circula/internal/httpx"
	"github.com/djshellshoxxx/circula/internal/store"
	"github.com/djshellshoxxx/circula/internal/testutil"
)

type logoutFaultStore struct {
	store.Store
	fail  atomic.Bool
	calls atomic.Int32
	err   error
}

func (s *logoutFaultStore) Update(ctx context.Context, fn func(store.Tx) error) error {
	return s.Store.Update(ctx, func(tx store.Tx) error {
		return fn(&logoutFaultTx{Tx: tx, s: s})
	})
}

type logoutFaultTx struct {
	store.Tx
	s *logoutFaultStore
}

func (tx *logoutFaultTx) RevokeSession(id string, at time.Time) error {
	tx.s.calls.Add(1)
	if tx.s.fail.Load() {
		return tx.s.err
	}
	return tx.Tx.RevokeSession(id, at)
}

// A2-06 / SEC-003: a failed server-side revocation cannot be reported as a
// successful sign-out. Keep the browser session so the user can retry, then
// verify that a successful retry invalidates the original cookie server-side.
func TestLogoutFailurePreservesSessionForRetry(t *testing.T) {
	for _, tc := range []struct {
		name   string
		err    error
		status int
	}{
		{"write failure", errors.New("injected session revocation failure"), 500},
		{"busy", &domain.Error{Code: domain.CodeUnavailable, Msg: "database busy"}, 503},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var fs *logoutFaultStore
			env := testutil.NewEnvWith(t, func(s store.Store) store.Store {
				fs = &logoutFaultStore{Store: s, err: tc.err}
				return fs
			})
			w := &world{env: env}
			w.admin = env.Person("admin@example.org")
			w.net, w.adminM = env.Network(w.admin, "town", nil)
			w.alice, w.aliceM = env.Member(w.admin, w.net, "alice@example.org")
			w.srv = httptest.NewServer(NewHandlerWith(env.Svc, Options{
				Clock: env.Clock, CSRFKey: []byte("0123456789abcdef0123456789abcdef"),
			}))
			t.Cleanup(w.srv.Close)
			alice := w.browser(t)
			alice.login("alice@example.org")
			token := sessionValue(t, alice)
			if token == "" {
				t.Fatal("login did not issue a session")
			}
			home := alice.get(w.base())
			fs.fail.Store(true)
			alice.c.CheckRedirect = func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }
			failed := alice.post("/logout", home, nil)
			if failed.Status != tc.status || failed.Header.Get("Location") != "" {
				t.Errorf("failed logout status %d, location %q; want %d without redirect", failed.Status, failed.Header.Get("Location"), tc.status)
			}
			if strings.Contains(failed.Body, "You are signed out") {
				t.Error("failed logout announced successful sign-out")
			}
			for _, c := range (&http.Response{Header: failed.Header}).Cookies() {
				if (c.Name == httpx.SessionCookie || c.Name == httpx.SessionCookieSecure) && (c.MaxAge < 0 || c.Value == "") {
					t.Error("failed logout cleared the session cookie")
				}
				if c.Name == cookieFlash && c.Value != "" {
					t.Error("failed logout set a success flash")
				}
			}
			if sessionValue(t, alice) != token || fs.calls.Load() != 1 {
				t.Errorf("session or revocation call changed unexpectedly: same session %v, calls %d", sessionValue(t, alice) == token, fs.calls.Load())
			}
			if got := replay(t, w, token); got == "/login" {
				t.Error("failed revocation unexpectedly invalidated the server session")
			}
			fs.fail.Store(false)
			alice.c.CheckRedirect = nil
			alice.mustContain(alice.post("/logout", home, nil), "You are signed out")
			if sessionValue(t, alice) != "" || replay(t, w, token) != "/login" {
				t.Error("successful retry did not clear and revoke the original session")
			}
		})
	}
}
