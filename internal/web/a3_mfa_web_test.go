package web

import (
	"net/url"
	"strings"
	"testing"
)

// A3-03: starting setup needs the current password; a wrong one shows no secret.
func TestWebMFAEnrolNeedsPassword(t *testing.T) {
	w := newWorld(t)
	b := w.browser(t)
	b.login("alice@example.org")
	pg1 := b.get("/account/security/mfa/enrol")
	b.mustContain(pg1, `autocomplete="current-password"`)
	bad := b.post("/account/security/mfa/enrol", pg1, url.Values{"password": {"not the password"}})
	if bad.Status != 422 || secretRe.MatchString(bad.Body) || !strings.Contains(bad.Body, "id=\"error-summary\"") {
		t.Fatalf("wrong password: %d %s", bad.Status, snippet(bad.Body))
	}
	none := b.post("/account/security/mfa/enrol", bad, nil)
	if none.Status != 422 || secretRe.MatchString(none.Body) {
		t.Fatalf("no password: %d %s", none.Status, snippet(none.Body))
	}
	ok := b.post("/account/security/mfa/enrol", none, url.Values{"password": {testPW}})
	m := secretRe.FindStringSubmatch(ok.Body)
	if ok.Status != 200 || m == nil {
		t.Fatalf("right password: %d %s", ok.Status, snippet(ok.Body))
	}
	// a right code without the password enables nothing
	noPW := b.post("/account/security/mfa/enrol/confirm", ok, url.Values{"code": {webTOTP(t, m[1], w.env.Clock.Now())}})
	if noPW.Status != 422 || strings.Contains(noPW.Body, "Save your recovery codes") {
		t.Fatalf("confirm without password: %d %s", noPW.Status, snippet(noPW.Body))
	}
}
