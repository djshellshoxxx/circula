package web

import (
	"html"
	"net/url"
	"regexp"
	"strings"
	"testing"

	"github.com/djshellshoxxx/circula/internal/domain"
	"github.com/djshellshoxxx/circula/internal/service"
	"github.com/djshellshoxxx/circula/internal/testutil"
)

// GOV-021: a member of a network with no administrator asks for recovery from
// the HTML interface (CSRF and idempotency key on the form); every member
// reads the requests and the recorded steps; nothing on the page approves.
func TestGOV021AdminRecoveryPage(t *testing.T) {
	w := newWorld(t)
	alice := w.browser(t)
	alice.login("alice@example.org")
	p := alice.get(w.base() + "/admin-recovery")
	if p.Status != 200 || strings.Contains(p.Body, `name="confirm"`) {
		t.Fatalf("page with an active administrator offers a request: %d", p.Status)
	}
	alice.mustContain(p, "has an active administrator")

	if _, err := w.env.Store.DB().Exec(`UPDATE members SET status = 'suspended' WHERE id = ?`, w.adminM.ID); err != nil {
		t.Fatal(err)
	}
	home := alice.get(w.base() + "/")
	alice.mustContain(home, `href="/n/town/admin-recovery"`, "Ask for administrator recovery")
	form := alice.get(w.base() + "/admin-recovery")
	checkPage(t, "admin recovery form", form)
	alice.mustContain(form, `name="confirm"`, `name="idem"`, "Ask to become administrator")

	if r := alice.post(w.base()+"/admin-recovery", nil, url.Values{"confirm": {"1"}}); r.Status != 403 {
		t.Fatalf("post without CSRF token: %d", r.Status)
	}
	bad := alice.post(w.base()+"/admin-recovery", form, url.Values{})
	if bad.Status != 422 {
		t.Fatalf("unconfirmed post: %d", bad.Status)
	}
	checkPage(t, "admin recovery unconfirmed", bad)
	idem := form.idem()
	done := alice.post(w.base()+"/admin-recovery", form, url.Values{"confirm": {"1"}, "idem": {idem}})
	if done.Status != 200 || !strings.Contains(done.Body, "Your request was sent") {
		t.Fatalf("request: %d %s", done.Status, snippet(done.Body))
	}
	alice.mustContain(done, "You", "Waiting for a first approval", "A member asked to be appointed administrator")
	if strings.Contains(done.Body, `name="confirm"`) {
		t.Error("the form is still offered with an open request")
	}
	// A resubmitted form (same idempotency key) creates nothing new.
	alice.post(w.base()+"/admin-recovery", form, url.Values{"confirm": {"1"}, "idem": {idem}})
	if got := testutil.Must(w.env.Svc.ListAdminRecoveryRequests(w.env.Ctx, w.bob, w.net.ID, service.Page{})); len(got) != 1 {
		t.Fatalf("requests after a resubmission: %d", len(got))
	}

	// The operator approves twice in the CLI; bob sees each step, without labels or reasons.
	op := domain.SystemActor("cli-admin-recovery")
	req := testutil.Must(w.env.Svc.ListAdminRecoveryRequests(w.env.Ctx, w.bob, w.net.ID, service.Page{}))[0]
	for _, label := range []string{"Olga", "Piet"} {
		testutil.Must(w.env.Svc.ApproveAdminRecovery(w.env.Ctx, op, service.AdminRecoveryDecision{Network: "town", RequestID: req.ID,
			Operator: label, Reason: "ticket 5 identity confirmed"}))
	}
	bob := w.browser(t)
	bob.login("bob@example.org")
	view := bob.get(w.base() + "/admin-recovery")
	checkPage(t, "admin recovery history", view)
	bob.mustContain(view, "alice@example.org", "Approved twice: the member is now an administrator", "First approval",
		"Second approval by a different operator; administrator appointed", "Platform operator")
	for _, secret := range []string{"Olga", "Piet", "ticket 5"} {
		if strings.Contains(view.Body, secret) {
			t.Errorf("members' page shows %q", secret)
		}
	}
	inbox := bob.get(w.base() + "/inbox")
	bob.mustContain(inbox, "An administrator was appointed", "Administrator recovery requested")
	// Alice is now an ordinary, permanent administrator.
	if a := alice.get(w.base() + "/admin"); a.Status != 200 {
		t.Fatalf("appointed administrator cannot open the admin page: %d", a.Status)
	}
	if strings.Contains(alice.get(w.base()+"/").Body, "Ask for administrator recovery") {
		t.Error("recovery link still shown after an administrator was appointed")
	}
}

// GOV-021: the members' page continues both lists past their first page
// (50 requests, 100 recorded steps), so no request or step is out of reach.
func TestGOV021AdminRecoveryPagePaging(t *testing.T) {
	w := newWorld(t)
	if _, err := w.env.Store.DB().Exec(`UPDATE members SET status = 'suspended' WHERE id = ?`, w.adminM.ID); err != nil {
		t.Fatal(err)
	}
	op := domain.SystemActor("cli-admin-recovery")
	const cycles = 51 // 51 requests and 102 recorded steps
	var firstID string
	for i := 0; i < cycles; i++ {
		r := testutil.Must(w.env.Svc.RequestAdminRecovery(w.env.Ctx, w.alice, w.net.ID, service.AdminRecoveryInput{IdempotencyKey: w.env.Key()}))
		if i == 0 {
			firstID = r.ID
		}
		testutil.Must(w.env.Svc.RefuseAdminRecovery(w.env.Ctx, op, service.AdminRecoveryDecision{Network: "town", RequestID: r.ID,
			Operator: "Olga", Reason: "ticket"}))
	}
	bob := w.browser(t)
	bob.login("bob@example.org")
	first := bob.get(w.base() + "/admin-recovery")
	checkPage(t, "admin recovery first page", first)
	rows := strings.Count(first.Body, `data-label="Requested by"`)
	steps := strings.Count(first.Body, `data-label="When"`)
	if rows != 50 || steps != 100 {
		t.Fatalf("first page: %d requests, %d steps", rows, steps)
	}
	nextRows := regexpFind(first.Body, `href="(/n/town/admin-recovery\?after=[^"]+)"`)
	nextSteps := regexpFind(first.Body, `href="(/n/town/admin-recovery\?events_after=[0-9]+)"`)
	if nextRows == "" || nextSteps == "" {
		t.Fatalf("no continuation links: %q %q", nextRows, nextSteps)
	}
	more := bob.get(nextRows)
	if n := strings.Count(more.Body, `data-label="Requested by"`); n != 1 || !strings.Contains(more.Body, "Refused") {
		t.Fatalf("second requests page: %d rows", n)
	}
	if strings.Contains(more.Body, `admin-recovery?after=`) {
		t.Error("continuation offered after the last request")
	}
	if got := testutil.Must(w.env.Svc.ListAdminRecoveryRequests(w.env.Ctx, w.bob, w.net.ID, service.Page{AfterID: strings.TrimPrefix(nextRows, "/n/town/admin-recovery?after="), Limit: 50})); len(got) != 1 || got[0].ID != firstID {
		t.Fatalf("the oldest request is not on the second page: %+v", got)
	}
	tail := bob.get(nextSteps)
	if n := strings.Count(tail.Body, `data-label="When"`); n != 2 {
		t.Fatalf("second steps page: %d steps, want 2", n)
	}
}

func regexpFind(body, pattern string) string {
	m := regexp.MustCompile(pattern).FindStringSubmatch(body)
	if m == nil {
		return ""
	}
	return html.UnescapeString(m[1])
}
