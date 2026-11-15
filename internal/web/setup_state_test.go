package web

import (
	"net/http/httptest"
	"net/url"
	"sync"
	"testing"

	"github.com/djshellshoxxx/circula/internal/domain"
)

func setupVals(secret, slug string) url.Values {
	return url.Values{"setup_secret": {secret}, "name": {"New"}, "slug": {slug}, "display_name": {"Founder"}, "currency_name": {"credit"},
		"decimals": {"2"}, "default_credit_limit": {"0.00"}, "join_mode": {"invite"}, "signup_grant": {"0.00"}}
}

func hasNetwork(w *world, slug string) bool {
	_, err := w.env.Svc.NetworkBySlug(w.env.Ctx, slug)
	return err == nil
}

// SEC-028: a restart (a new handler over the same store) does not re-arm the
// setup secret; only the operator reset does.
func TestSetupSecretUsedSurvivesRestart(t *testing.T) {
	const secret = "s3cret-value-123"
	w := newWorldWith(t, func(o *Options) { o.SetupSecret = secret })
	w.env.Person("founder@example.org")
	b := w.browser(t)
	b.login("founder@example.org")
	form := b.get("/setup")
	if p := b.post("/setup", form, setupVals(secret, "one")); p.URL != "/n/one/" {
		t.Fatalf("first use: %d %s", p.Status, p.URL)
	}

	// "Restart": a second server with fresh in-memory state over the same store.
	restart := func() *browser {
		srv := httptest.NewServer(NewHandlerWith(w.env.Svc, Options{
			SetupSecret: secret, CSRFKey: []byte("0123456789abcdef0123456789abcdef"), Clock: w.env.Clock}))
		t.Cleanup(srv.Close)
		old := w.srv
		w.srv = srv
		defer func() { w.srv = old }()
		return w.browser(t)
	}
	b2 := restart()
	b2.login("founder@example.org")
	form2 := b2.get("/setup")
	if p := b2.post("/setup", form2, setupVals(secret, "two")); p.Status != 403 {
		t.Fatalf("secret reusable after restart: %d", p.Status)
	}
	if hasNetwork(w, "two") {
		t.Fatal("second network created after restart")
	}

	// Operator reset re-arms it, and then it works exactly once more.
	if _, err := w.env.Svc.ResetSetupSecret(w.env.Ctx, domain.SystemActor("test")); err != nil {
		t.Fatal(err)
	}
	b3 := restart() // reset is an offline operator step; the service is restarted after it
	b3.login("founder@example.org")
	form3 := b3.get("/setup")
	if p := b3.post("/setup", form3, setupVals(secret, "two")); p.URL != "/n/two/" {
		t.Fatalf("after reset: %d %s", p.Status, p.URL)
	}
	if p := b3.post("/setup", form3, setupVals(secret, "three")); p.Status != 403 {
		t.Errorf("reused after reset: %d", p.Status)
	}
}

// Concurrent first use by two people with the right secret creates exactly one network.
func TestSetupSecretConcurrentFirstUse(t *testing.T) {
	const secret = "s3cret-value-123"
	w := newWorldWith(t, func(o *Options) { o.SetupSecret = secret })
	const n = 6
	var wg sync.WaitGroup
	slugs := make([]string, n)
	status := make([]int, n)
	type prepared struct {
		b    *browser
		form *pg
	}
	preps := make([]prepared, n)
	for i := 0; i < n; i++ {
		login := "founder" + string(rune('a'+i)) + "@example.org"
		w.env.Person(login)
		b := w.browser(t)
		b.login(login)
		preps[i] = prepared{b, b.get("/setup")}
		slugs[i] = "net-" + string(rune('a'+i))
	}
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			status[i] = preps[i].b.post("/setup", preps[i].form, setupVals(secret, slugs[i])).Status
		}()
	}
	wg.Wait()
	created := 0
	for _, s := range slugs {
		if hasNetwork(w, s) {
			created++
		}
	}
	if created != 1 {
		t.Fatalf("%d networks created from one secret (statuses %v)", created, status)
	}
}
