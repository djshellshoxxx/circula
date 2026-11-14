package web

import (
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/djshellshoxxx/circula/internal/domain"
	"github.com/djshellshoxxx/circula/internal/service"
	"github.com/djshellshoxxx/circula/internal/testutil"
)

// queueAdjustment sets a dual-control threshold, a second administrator and one
// queued credit of 25.00 to bob proposed by the first administrator.
func queueAdjustment(t *testing.T, w *world) (pendingID string, adm *browser) {
	t.Helper()
	ctx := w.env.Ctx
	_, adm2M := w.env.Member(w.admin, w.net, "admin2@example.org")
	testutil.Must(0, w.env.Svc.AssignRole(ctx, domain.SystemActor("fixture"), w.net.ID, adm2M.ID, domain.RoleAdmin))
	w.env.Clock.Advance(25 * time.Hour) // A3-01: a new administrator waits before deciding
	n := testutil.Must(w.env.Svc.GetNetwork(ctx, w.admin, w.net.ID))
	pol := n.Policy
	pol.DualControlThreshold = 1000
	testutil.Must(w.env.Svc.UpdatePolicy(ctx, w.admin, w.net.ID, service.PolicyChange{Policy: pol, Reason: "dual", IdempotencyKey: w.env.Key()}))
	adm = w.browser(t)
	adm.login("admin@example.org")
	vals := url.Values{"member_id": {w.bobM.ID}, "direction": {"credit"}, "amount": {"25.00"}, "reason": {"big correction"}, "step": {"review"}}
	rev := adm.post(w.base()+"/admin/adjust", adm.get(w.base()+"/admin/adjust"), vals)
	vals.Set("step", "confirm")
	adm.post(w.base()+"/admin/adjust", rev, vals)
	return testutil.Must(w.env.Svc.ListPendingAdjustments(ctx, w.admin, w.net.ID, nil, service.Page{}))[0].ID, adm
}

func bobBalance(t *testing.T, w *world) int64 {
	t.Helper()
	return testutil.Must(w.env.Svc.GetBalance(w.env.Ctx, w.bob, w.net.ID)).Balance
}

// Dual control: reject and cancel through the pages, every refusal leaves the
// balance alone.
func TestWebDualControlRejectCancelNegative(t *testing.T) {
	w := newWorld(t)
	b := w.base()
	id, adm := queueAdjustment(t, w)
	two := w.browser(t)
	two.login("admin2@example.org")
	list := two.get(b + "/admin/adjustments/pending")

	// No CSRF token: nothing happens, for each action.
	for _, act := range []string{"approve", "reject", "cancel"} {
		if p := two.post(b+"/admin/adjustments/"+id+"/"+act, nil, nil); p.Status != 403 {
			t.Errorf("%s without csrf: %d", act, p.Status)
		}
	}
	// Unknown action.
	if p := two.post(b+"/admin/adjustments/"+id+"/delete", list, nil); p.Status != 404 {
		t.Errorf("unknown action: %d", p.Status)
	}
	// A plain member and an anonymous visitor cannot decide.
	al := w.browser(t)
	al.login("alice@example.org")
	if p := al.post(b+"/admin/adjustments/"+id+"/approve", al.get(b), nil); p.Status != 403 {
		t.Errorf("member approve: %d", p.Status)
	}
	if p := w.browser(t).post(b+"/admin/adjustments/"+id+"/approve", nil, nil); p.Status == 200 {
		t.Errorf("anonymous approve: %d", p.Status)
	}
	// The administrator of another network cannot decide it either.
	other := w.env.Person("boss@other.example")
	w.env.Network(other, "elsewhere", nil)
	ob := w.browser(t)
	ob.login("boss@other.example")
	if p := ob.post(b+"/admin/adjustments/"+id+"/approve", ob.get("/"), nil); p.Status != 404 {
		t.Errorf("foreign admin approve: %d", p.Status)
	}
	// Only the proposer cancels.
	if p := two.post(b+"/admin/adjustments/"+id+"/cancel", list, nil); p.Status != 409 {
		t.Errorf("approver cancel: %d", p.Status)
	}
	if bobBalance(t, w) != 0 {
		t.Fatal("balance moved by a refused action")
	}

	// Reject, then every later decision is refused.
	rej := two.post(b+"/admin/adjustments/"+id+"/reject", list, nil)
	two.mustContain(rej, "Adjustment request rejected")
	if p := two.post(b+"/admin/adjustments/"+id+"/approve", list, nil); p.Status != 409 {
		t.Errorf("approve after reject: %d", p.Status)
	}
	// Repeating the same decision is an idempotent replay, not an error and not a second effect.
	if p := two.post(b+"/admin/adjustments/"+id+"/reject", list, nil); p.Status != 200 {
		t.Errorf("repeat reject: %d", p.Status)
	}
	if p := adm.post(b+"/admin/adjustments/"+id+"/cancel", adm.get(b+"/admin/adjustments/pending"), nil); p.Status != 409 {
		t.Errorf("cancel after reject: %d", p.Status)
	}
	two.mustContain(two.get(b+"/admin/adjustments/pending?status=rejected"), "rejected by", "big correction")
	if bobBalance(t, w) != 0 {
		t.Fatal("rejected request moved the balance")
	}
}

func TestWebDualControlCancelAndExpiry(t *testing.T) {
	w := newWorld(t)
	b := w.base()
	id, adm := queueAdjustment(t, w)
	list := adm.get(b + "/admin/adjustments/pending")
	adm.mustContain(adm.post(b+"/admin/adjustments/"+id+"/cancel", list, nil), "Adjustment request cancelled")
	two := w.browser(t)
	two.login("admin2@example.org")
	if p := two.post(b+"/admin/adjustments/"+id+"/approve", two.get(b+"/admin/adjustments/pending"), nil); p.Status != 409 {
		t.Errorf("approve after cancel: %d", p.Status)
	}

	// An expired request cannot be approved and nothing posts.
	vals := url.Values{"member_id": {w.bobM.ID}, "direction": {"credit"}, "amount": {"30.00"}, "reason": {"second"}, "step": {"review"}}
	rev := adm.post(b+"/admin/adjust", adm.get(b+"/admin/adjust"), vals)
	vals.Set("step", "confirm")
	adm.post(b+"/admin/adjust", rev, vals)
	pa := testutil.Must(w.env.Svc.ListPendingAdjustments(w.env.Ctx, w.admin, w.net.ID, nil, service.Page{}))[0]
	w.env.Clock.Advance(8 * 24 * time.Hour) // past the 7 day lifetime; sessions are renewed by signing in again
	two = w.browser(t)
	two.login("admin2@example.org")
	p := two.post(b+"/admin/adjustments/"+pa.ID+"/approve", two.get(b+"/admin/adjustments/pending?status=expired"), nil)
	if p.Status != 409 {
		t.Errorf("approve after expiry: %d", p.Status)
	}
	two.mustContain(two.get(b+"/admin/adjustments/pending?status=expired"), "second", "expired")
	if bobBalance(t, w) != 0 {
		t.Fatal("expired request posted")
	}
}

// MFA: state-changing pages need a CSRF token, a challenge is single use and a
// one-time code cannot be replayed.
func TestWebMFANegative(t *testing.T) {
	w := newWorld(t)
	b := w.browser(t)
	secret, codes := enrolWeb(t, w, b)
	sec := b.get("/account/security")

	for _, path := range []string{"/account/security/mfa/enrol", "/account/security/mfa/enrol/confirm", "/account/security/mfa/disable",
		"/account/security/mfa/recovery", "/account/security/ack"} {
		if p := b.post(path, nil, url.Values{"code": {webTOTP(t, secret, w.env.Clock.Now())}, "password": {testPW}}); p.Status != 403 {
			t.Errorf("%s without csrf: %d", path, p.Status)
		}
	}
	// Still on, codes untouched.
	b.mustContain(b.get("/account/security"), "is <strong>on</strong>", "10 unused recovery codes")

	// Start enrolment again while on: refused, nothing changes.
	if p := b.post("/account/security/mfa/enrol", sec, url.Values{"password": {testPW}}); p.Status != 409 {
		t.Errorf("enrol while on: %d", p.Status)
	}
	// Disable with a correct code but a wrong password.
	w.env.Clock.Advance(31 * time.Second)
	if p := b.post("/account/security/mfa/disable", sec, url.Values{"password": {"wrong wrong wrong"}, "code": {webTOTP(t, secret, w.env.Clock.Now())}}); p.Status != 422 {
		t.Errorf("disable wrong password: %d", p.Status)
	}
	b.mustContain(b.get("/account/security"), "is <strong>on</strong>")

	// A challenge completes once; a replay of the same challenge and code fails.
	c := w.browser(t)
	step := c.post("/login", c.get("/login"), url.Values{"login": {"alice@example.org"}, "password": {testPW}})
	ch := challengeOf(t, step)
	code := webTOTP(t, secret, w.env.Clock.Now())
	if p := c.post("/login/mfa", step, url.Values{"challenge": {ch}, "code": {code}}); p.URL == "/login/mfa" || p.Status != 200 {
		t.Fatalf("good code: %d %s", p.Status, p.URL)
	}
	d := w.browser(t)
	if p := d.post("/login/mfa", step, url.Values{"challenge": {ch}, "code": {code}}); p.Status == 200 && p.URL != "/login/mfa" && p.URL != "/login" {
		t.Errorf("challenge reused: %d %s", p.Status, p.URL)
	}
	if p := d.get("/account/security"); p.URL != "/login" {
		t.Errorf("session from a replayed challenge: %s", p.URL)
	}
	// The same TOTP code cannot start a second session.
	e := w.browser(t)
	st2 := e.post("/login", e.get("/login"), url.Values{"login": {"alice@example.org"}, "password": {testPW}})
	if p := e.post("/login/mfa", st2, url.Values{"challenge": {challengeOf(t, st2)}, "code": {code}}); p.Status != 401 {
		t.Errorf("replayed TOTP code: %d", p.Status)
	}
	// A garbage challenge is refused without leaking why.
	if p := e.post("/login/mfa", st2, url.Values{"challenge": {"garbage"}, "code": {codes[1]}}); p.Status != 401 {
		t.Errorf("garbage challenge: %d", p.Status)
	}
	// A recovery code works once, upper or lower case alike.
	f := w.browser(t)
	st3 := f.post("/login", f.get("/login"), url.Values{"login": {"alice@example.org"}, "password": {testPW}})
	if p := f.post("/login/mfa", st3, url.Values{"challenge": {challengeOf(t, st3)}, "code": {codes[2]}}); p.Status != 200 || p.URL == "/login/mfa" {
		t.Fatalf("recovery code: %d %s", p.Status, p.URL)
	}
	g := w.browser(t)
	st4 := g.post("/login", g.get("/login"), url.Values{"login": {"alice@example.org"}, "password": {testPW}})
	if p := g.post("/login/mfa", st4, url.Values{"challenge": {challengeOf(t, st4)}, "code": {strings.ToLower(codes[2])}}); p.Status != 401 {
		t.Errorf("reused recovery code: %d", p.Status)
	}
}

// Recovery: CSRF is required, a code works once and not for another login.
func TestWebRecoveryNegative(t *testing.T) {
	w := newWorld(t)
	b := w.browser(t)
	b.login("alice@example.org")
	cf := b.get("/account/recovery-codes")
	if p := b.post("/account/recovery-codes", nil, url.Values{"password": {testPW}}); p.Status != 403 {
		t.Errorf("make codes without csrf: %d", p.Status)
	}
	codes := recCodeRe.FindAllStringSubmatch(b.post("/account/recovery-codes", cf, url.Values{"password": {testPW}}).Body, -1)
	if len(codes) != 10 {
		t.Fatalf("codes: %d", len(codes))
	}
	pw := "a brand new passphrase"
	anon := w.browser(t)
	f := anon.get("/recover")
	form := func(login, code string) url.Values {
		return url.Values{"login": {login}, "recovery_code": {code}, "password": {pw}, "password2": {pw}}
	}
	if p := anon.post("/recover", nil, form("alice@example.org", codes[0][1])); p.Status != 403 {
		t.Errorf("recover without csrf: %d", p.Status)
	}
	// Alice's code does not work for bob, and the failure looks the same.
	if p := anon.post("/recover", f, form("bob@example.org", codes[0][1])); p.Status != 401 {
		t.Errorf("another person's code: %d", p.Status)
	}
	// Short and weak new passwords are refused without spending the code.
	if p := anon.post("/recover", f, url.Values{"login": {"alice@example.org"}, "recovery_code": {codes[0][1]}, "password": {"short"}, "password2": {"short"}}); p.Status != 422 {
		t.Errorf("weak password: %d", p.Status)
	}
	if p := anon.post("/recover", f, form("alice@example.org", codes[0][1])); p.URL != "/login" {
		t.Fatalf("code spent by a refused attempt, at %s (%d)", p.URL, p.Status)
	}
	// The rate limit stops guessing.
	var last int
	for i := 0; i < 8; i++ {
		last = anon.post("/recover", f, form("alice@example.org", "AAAA-AAAA-AAAA-AAAA")).Status
	}
	if last != 429 {
		t.Errorf("guessing was never limited: %d", last)
	}
}

// Break-glass history and erasure: wrong role, foreign admin, missing CSRF.
func TestWebBreakGlassErasureNegative(t *testing.T) {
	w := newWorld(t)
	b := w.base()
	ctx := w.env.Ctx
	testutil.Must(w.env.Svc.OpenBreakGlass(ctx, domain.SystemActor("t"), service.OpenBreakGlassInput{Network: "town", Operator: "Olga",
		Reason: "restore after disk fault", TTL: time.Hour}))
	other := w.env.Person("boss@other.example")
	w.env.Network(other, "elsewhere", nil)
	ob := w.browser(t)
	ob.login("boss@other.example")
	if p := ob.get(b + "/admin/break-glass"); p.Status != 404 || strings.Contains(p.Body, "Olga") {
		t.Errorf("foreign admin break-glass: %d", p.Status)
	}
	// The sessions of the other network's own page show nothing of this one.
	if p := ob.get("/n/elsewhere/admin/break-glass"); strings.Contains(p.Body, "Olga") {
		t.Error("operator access leaked across networks")
	}

	testutil.Must(w.env.Svc.DepartMember(ctx, w.admin, w.net.ID, service.DepartInput{MemberID: w.aliceM.ID, Reason: "left"}))
	testutil.Must(w.env.Svc.DepartMember(ctx, w.admin, w.net.ID, service.DepartInput{MemberID: w.bobM.ID, Reason: "left"}))
	adm := w.browser(t)
	adm.login("admin@example.org")
	f := adm.get(b + "/admin/members/" + w.aliceM.ID + "/erase")
	if p := adm.post(b+"/admin/members/"+w.aliceM.ID+"/erase", nil, url.Values{"confirm": {"yes"}, "reason": {"request 12"}}); p.Status != 403 {
		t.Errorf("admin erase without csrf: %d", p.Status)
	}
	if p := ob.post(b+"/admin/members/"+w.aliceM.ID+"/erase", ob.get("/"), url.Values{"confirm": {"yes"}, "reason": {"request 12"}}); p.Status != 404 {
		t.Errorf("foreign admin erase: %d", p.Status)
	}
	m := testutil.Must(w.env.Svc.GetMember(ctx, w.admin, w.net.ID, w.aliceM.ID))
	if strings.HasPrefix(m.DisplayName, "Former member") {
		t.Fatal("a refused request erased the member")
	}
	// Another person's membership id on the self-service route does nothing to them.
	bob := w.browser(t)
	bob.login("bob@example.org")
	sf := bob.get(b + "/erase-my-data")
	if p := bob.post(b+"/erase-my-data", nil, url.Values{"confirm": {"yes"}}); p.Status != 403 {
		t.Errorf("self erase without csrf: %d", p.Status)
	}
	bob.post(b+"/erase-my-data", sf, url.Values{"confirm": {"yes"}, "member_id": {w.aliceM.ID}})
	if a := testutil.Must(w.env.Svc.GetMember(ctx, w.admin, w.net.ID, w.aliceM.ID)); strings.HasPrefix(a.DisplayName, "Former member") {
		t.Error("self erase affected another member")
	}
	_ = f
}
