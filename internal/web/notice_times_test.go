package web

import (
	"html"
	"testing"
	"time"

	"github.com/djshellshoxxx/circula/internal/domain"
)

// A2-13: composing notices must keep trusted time elements separate from
// ordinary text, which still needs the template's escaping.
func TestFrozenAndCheckTimesRenderAsElements(t *testing.T) {
	w := newWorld(t)
	if err := w.env.Svc.SetTimeZone(w.env.Ctx, w.admin, "Asia/Kolkata"); err != nil {
		t.Fatal(err)
	}
	rep, err := w.env.Svc.VerifyLedger(w.env.Ctx, domain.SystemActor("test"), w.net.ID)
	if err != nil {
		t.Fatal(err)
	}
	if err := w.env.Svc.RecordScheduledConsistencyRun(w.env.Ctx, domain.SystemActor("test"), rep); err != nil {
		t.Fatal(err)
	}
	w.env.Clock.Advance(15 * time.Minute)
	reason := "<img src=x onerror=alert(1)> & check the ledger"
	w.env.Svc.FreezeNetwork(w.net.ID, reason)

	admin := w.browser(t)
	admin.login("admin@example.org")
	p := admin.get("/n/town/admin")
	if p.Status != 200 {
		t.Fatalf("admin page status = %d", p.Status)
	}
	admin.mustContain(p,
		`since <time datetime="2026-03-01T12:15:00Z">1 Mar 2026, 17:45 IST</time>.`,
		`The last ledger check passed. <time datetime="2026-03-01T12:00:00Z">1 Mar 2026, 17:30 IST</time>`,
		"Reason: "+html.EscapeString(reason))
	admin.mustNotContain(p, "&lt;time", "<img")

	member := w.browser(t)
	member.login("alice@example.org")
	p = member.get("/n/town")
	if p.Status != 200 {
		t.Fatalf("member page status = %d", p.Status)
	}
	member.mustContain(p, `since <time datetime="2026-03-01T12:15:00Z">1 Mar 2026, 12:15 UTC</time>.`)
	member.mustNotContain(p, "&lt;time", "check the ledger", "<img")
}
