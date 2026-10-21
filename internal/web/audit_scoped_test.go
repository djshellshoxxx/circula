package web

import (
	"strings"
	"testing"
)

// FR-082, DAT-033: a member sees an activity log about their own account only.
func TestMemberAuditPage(t *testing.T) {
	w := newWorld(t)
	b := w.base()
	al := w.browser(t)
	al.login("alice@example.org")
	p := al.get(b + "/audit")
	if p.Status != 200 {
		t.Fatalf("member audit: %d", p.Status)
	}
	al.mustContain(p, "Account activity log")
	if strings.Contains(p.Body, "bob@example.org") {
		t.Error("member audit leaks another member")
	}
	ad := w.browser(t)
	ad.login("admin@example.org")
	if p := ad.get(b + "/audit"); p.Status != 200 {
		t.Errorf("admin audit view: %d", p.Status)
	}
}
