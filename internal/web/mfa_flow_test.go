package web

import (
	"crypto/hmac"
	"crypto/sha1" //nolint:gosec // RFC 6238 test oracle
	"encoding/base32"
	"encoding/binary"
	"fmt"
	"net/url"
	"regexp"
	"strings"
	"testing"
	"time"
)

func webTOTP(t *testing.T, b32 string, at time.Time) string {
	t.Helper()
	key, err := base32.StdEncoding.WithPadding(base32.NoPadding).DecodeString(b32)
	if err != nil {
		t.Fatal(err)
	}
	var msg [8]byte
	binary.BigEndian.PutUint64(msg[:], uint64(at.Unix()/30)) //nolint:gosec // positive
	m := hmac.New(sha1.New, key)
	m.Write(msg[:])
	s := m.Sum(nil)
	o := s[19] & 15
	return fmt.Sprintf("%06d", (binary.BigEndian.Uint32(s[o:o+4])&0x7fffffff)%1000000)
}

var (
	secretRe = regexp.MustCompile(`<code>([A-Z2-7]{32})</code>`)
	codeRe   = regexp.MustCompile(`<code>([A-Z2-7]{5}-[A-Z2-7]{5})</code>`)
)

// enrolWeb enrols alice through the real pages and returns secret + codes.
func enrolWeb(t *testing.T, w *world, b *browser) (string, []string) {
	t.Helper()
	b.login("alice@example.org")
	pg1 := b.get("/account/security/mfa/enrol")
	b.mustContain(pg1, "Start setup")
	pg2 := b.post("/account/security/mfa/enrol", pg1, nil)
	m := secretRe.FindStringSubmatch(pg2.Body)
	if pg2.Status != 200 || m == nil {
		t.Fatalf("no secret shown: %d %s", pg2.Status, snippet(pg2.Body))
	}
	secret := m[1]
	// wrong code keeps the same secret on the page
	bad := b.post("/account/security/mfa/enrol/confirm", pg2, url.Values{"code": {"000000"}})
	if bad.Status != 422 || !strings.Contains(bad.Body, secret) || !strings.Contains(bad.Body, "id=\"error-summary\"") {
		t.Fatalf("bad code: %d %s", bad.Status, snippet(bad.Body))
	}
	ok := b.post("/account/security/mfa/enrol/confirm", bad, url.Values{"code": {webTOTP(t, secret, w.env.Clock.Now())}})
	cs := codeRe.FindAllStringSubmatch(ok.Body, -1)
	if ok.Status != 200 || len(cs) != 10 {
		t.Fatalf("codes: %d %d %s", ok.Status, len(cs), snippet(ok.Body))
	}
	var codes []string
	for _, c := range cs {
		codes = append(codes, c[1])
	}
	w.env.Clock.Advance(31 * time.Second)
	return secret, codes
}

func TestWebMFAEnrolAndLogin(t *testing.T) {
	w := newWorld(t)
	secret, codes := enrolWeb(t, w, w.browser(t))

	// secrets are not rendered by the security page afterwards
	b := w.browser(t)
	b.login("bob@example.org") // plain member still signs in with the password alone
	if p := b.get("/account/security"); !strings.Contains(p.Body, "is <strong>off</strong>") {
		t.Errorf("bob page: %s", snippet(p.Body))
	}

	b2 := w.browser(t)
	f := b2.get("/login")
	step := b2.post("/login", f, url.Values{"login": {"alice@example.org"}, "password": {testPW}})
	b2.mustContain(step, "Enter your authentication code")
	if step.Status != 200 {
		t.Fatalf("status %d", step.Status)
	}
	// no session yet
	if p := b2.get("/account/security"); p.URL != "/login" {
		t.Errorf("reached %s without second factor", p.URL)
	}
	wrong := b2.post("/login/mfa", step, url.Values{"challenge": {challengeOf(t, step)}, "code": {"111111"}})
	if wrong.Status != 401 || !strings.Contains(wrong.Body, "Enter your authentication code") {
		t.Fatalf("wrong code: %d %s", wrong.Status, snippet(wrong.Body))
	}
	good := b2.post("/login/mfa", wrong, url.Values{"challenge": {challengeOf(t, wrong)}, "code": {webTOTP(t, secret, w.env.Clock.Now())}})
	if good.Status != 200 || good.URL == "/login/mfa" {
		t.Fatalf("good code: %d %s", good.Status, good.URL)
	}
	sec := b2.get("/account/security")
	b2.mustContain(sec, "is <strong>on</strong>", "10 unused recovery codes", "Two-step sign-in was turned on.")
	_ = sec

	// recovery code logs in once
	b3 := w.browser(t)
	st := b3.post("/login", b3.get("/login"), url.Values{"login": {"alice@example.org"}, "password": {testPW}})
	rc := b3.post("/login/mfa", st, url.Values{"challenge": {challengeOf(t, st)}, "code": {strings.ToLower(codes[0])}})
	if rc.Status != 200 || rc.URL == "/login/mfa" {
		t.Fatalf("recovery login: %d %s", rc.Status, rc.URL)
	}
	b3.mustContain(b3.get("/account/security"), "A recovery code was used to sign in.")
	b4 := w.browser(t)
	st = b4.post("/login", b4.get("/login"), url.Values{"login": {"alice@example.org"}, "password": {testPW}})
	re := b4.post("/login/mfa", st, url.Values{"challenge": {challengeOf(t, st)}, "code": {codes[0]}})
	if re.Status != 401 {
		t.Errorf("used recovery code: %d", re.Status)
	}
}

func challengeOf(t *testing.T, p *pg) string {
	t.Helper()
	m := regexp.MustCompile(`name="challenge" value="([^"]+)"`).FindStringSubmatch(p.Body)
	if m == nil {
		t.Fatalf("no challenge in page: %s", snippet(p.Body))
	}
	return m[1]
}

func TestWebMFAChallengeCSRFAndGet(t *testing.T) {
	w := newWorld(t)
	enrolWeb(t, w, w.browser(t))
	b := w.browser(t)
	st := b.post("/login", b.get("/login"), url.Values{"login": {"alice@example.org"}, "password": {testPW}})
	// no csrf token: refused before any attempt is counted
	if p := b.post("/login/mfa", nil, url.Values{"challenge": {challengeOf(t, st)}, "code": {"123456"}}); p.Status != 403 {
		t.Errorf("missing csrf: %d", p.Status)
	}
	// GET does not expose the form
	if p := b.get("/login/mfa"); p.URL != "/login" {
		t.Errorf("GET /login/mfa -> %s", p.URL)
	}
}

func TestWebSecurityNoticesAndDisable(t *testing.T) {
	w := newWorld(t)
	b := w.browser(t)
	secret, codes := enrolWeb(t, w, b)
	home := b.get("/account/security")
	b.mustContain(home, "1 new", "Mark all as read")
	// header carries the count on other pages too
	b.mustContain(b.get("/"), "1 new")
	ack := b.post("/account/security/ack", home, nil)
	if strings.Contains(ack.Body, "Mark all as read") || strings.Contains(ack.Body, "1 new") {
		t.Errorf("notices not acknowledged: %s", snippet(ack.Body))
	}
	// regenerate codes
	reg := b.post("/account/security/mfa/recovery", ack, url.Values{"code": {webTOTP(t, secret, w.env.Clock.Now())}})
	if got := codeRe.FindAllString(reg.Body, -1); reg.Status != 200 || len(got) != 10 {
		t.Fatalf("regen: %d %d", reg.Status, len(got))
	}
	// disable: wrong password first
	page := b.get("/account/security")
	bad := b.post("/account/security/mfa/disable", page, url.Values{"password": {"nope nope nope nope"}, "code": {codes[0]}})
	if bad.Status != 422 || !strings.Contains(bad.Body, "Password: not correct") {
		t.Fatalf("bad password: %d %s", bad.Status, snippet(bad.Body))
	}
	w.env.Clock.Advance(31 * time.Second)
	done := b.post("/account/security/mfa/disable", page, url.Values{"password": {testPW}, "code": {webTOTP(t, secret, w.env.Clock.Now())}})
	b.mustContain(done, "is <strong>off</strong>", "Two-step sign-in was turned off.")
}

func TestWebSecurityPagesNeedLogin(t *testing.T) {
	w := newWorld(t)
	b := w.browser(t)
	for _, path := range []string{"/account/security", "/account/security/mfa/enrol"} {
		if p := b.get(path); p.URL != "/login" {
			t.Errorf("%s -> %s", path, p.URL)
		}
	}
}
