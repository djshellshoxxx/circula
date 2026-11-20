package web

import (
	"net/url"
	"testing"
	"time"

	"github.com/djshellshoxxx/circula/internal/passwordwork"
	"github.com/djshellshoxxx/circula/internal/service"
	"github.com/djshellshoxxx/circula/internal/testutil"
	"github.com/djshellshoxxx/circula/internal/web/messages"
)

func TestPasswordAdmissionHTMLReturnsRetryableServiceBusy(t *testing.T) {
	limit := testutil.Must(passwordwork.New(1))
	w := newWorldCfg(t, func(c *service.Config) { c.PasswordWork = limit }, func(o *Options) {
		// One wrongly classified capacity refusal would lock the next request.
		o.Limits.SigninAddress = Limit{Max: 1, Window: time.Minute}
		o.Limits.SigninAccount = Limit{Max: 1, Window: time.Minute}
	})
	b := w.browser(t)
	form := b.get("/login")
	held := testutil.Must(limit.Acquire(w.env.Ctx))
	defer held.Release()
	for _, login := range []string{"alice@example.org", "unknown@example.org", "not a login"} {
		page := b.post("/login", form, url.Values{"login": {login}, "password": {testPW}})
		if page.Status != 503 || page.Header.Get("Retry-After") != "5" {
			t.Fatalf("busy sign-in %q: status=%d retry-after=%q", login, page.Status, page.Header.Get("Retry-After"))
		}
		b.mustContain(page, messages.Text("error.busy.title"), messages.Text("error.busy.body"))
	}
	held.Release()
	b.login("alice@example.org")
}
