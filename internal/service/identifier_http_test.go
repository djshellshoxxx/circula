package service_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/djshellshoxxx/circula/internal/api"
	"github.com/djshellshoxxx/circula/internal/domain"
	"github.com/djshellshoxxx/circula/internal/httpx"
	"github.com/djshellshoxxx/circula/internal/service"
	tu "github.com/djshellshoxxx/circula/internal/testutil"
	"github.com/djshellshoxxx/circula/internal/web"
)

var identifierCSRFKey = []byte("0123456789abcdef0123456789abcdef")

func identifierHTTP(h http.Handler, method, path, session string, body any, form url.Values, csrf bool) *httptest.ResponseRecorder {
	var data []byte
	if body != nil {
		data, _ = json.Marshal(body)
	}
	if form != nil {
		data = []byte(form.Encode())
	}
	r := httptest.NewRequest(method, path, bytes.NewReader(data))
	if body != nil {
		r.Header.Set("Content-Type", "application/json")
	}
	if form != nil {
		r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	}
	if session != "" {
		r.AddCookie(&http.Cookie{Name: httpx.SessionCookie, Value: session})
	}
	if csrf {
		r.Header.Set("X-CSRF-Token", httpx.CSRF{Key: identifierCSRFKey}.Token(session))
	}
	r.Header.Set("Idempotency-Key", "identifier-"+method+"-"+path)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, r)
	return rec
}

// SEC-024 / A2-16: both HTTP adapters must reject malformed target text before its
// repository use while retaining their existing authentication and CSRF order.
func TestIdentifierBoundariesAcrossHTTPAdapters(t *testing.T) {
	f := newIdentifierFixture(t)
	sessions := map[string]string{}
	for _, login := range []string{"admin", "alice"} {
		tok, _, err := f.Svc.Login(f.Ctx, login, "correct horse battery")
		if err != nil {
			t.Fatal(err)
		}
		sessions[login] = tok
	}
	ah := api.NewHandlerWithOptions(f.Svc, api.Options{CSRFKey: identifierCSRFKey, Now: f.Clock.Now, IPPerMinute: -1, PrincipalPerMinute: -1})
	defer api.CloseHandler(ah)
	wh := web.NewHandlerWith(f.Svc, web.Options{CSRFKey: identifierCSRFKey, Clock: f.Clock})
	const bad = "x' OR '1'='1"
	type obj = map[string]any
	// Each request has a distinct path/key; none can be cached as successful.
	for _, tc := range []struct {
		name, path string
		body       obj
		status     int
	}{
		{"counterparty", "/transactions", obj{"counterparty_member_id": bad, "direction": "pay", "amount_minor": "1"}, 404},
		{"listing", "/transactions", obj{"counterparty_member_id": f.bobM.ID, "listing_id": bad, "direction": "pay", "amount_minor": "1"}, 422},
		{"dispute", "/disputes", obj{"transaction_id": bad, "category": "other", "statement": "reason"}, 404},
		{"adjustment", "/admin/adjustments", obj{"member_id": bad, "direction": "credit", "amount_minor": "1", "reason": "reason"}, 404},
		{"report", "/reports", obj{"target_type": "member", "target_id": bad, "category": "other"}, 404},
		{"notifications", "/notifications/read", obj{"ids": []string{bad}}, 422},
		{"token_network", "/tokens", obj{"name": "test", "network_id": bad, "scopes": []string{"read"}}, 404},
		{"challenge", "/challenges", obj{"kind": "listing_removal", "subject_id": bad, "reason": "reason"}, 404},
		{"announcement", "/admin/announcements", obj{"title": "Title", "body": "Body", "supersedes_id": bad}, 422},
		{"removal_matter", "/admin/members/" + f.bobM.ID + "/remove", obj{"reason": "reason", "reason_code": "confirmed_fraud", "transaction_id": bad}, 422},
	} {
		t.Run("api_"+tc.name, func(t *testing.T) {
			rec := identifierHTTP(ah, "POST", "/api/v1"+tc.path, sessions["admin"], tc.body, nil, true)
			if rec.Code != tc.status {
				t.Fatalf("status=%d want=%d body=%s", rec.Code, tc.status, rec.Body.String())
			}
			if got := f.spy.count(bad); got != 0 {
				t.Fatalf("body target reached repository %d times", got)
			}
		})
	}
	for _, tc := range []struct{ method, path string }{
		{"GET", "/trades/{id}"}, {"GET", "/members/{id}"}, {"GET", "/listings/{id}/edit"},
		{"GET", "/disputes/{id}"}, {"POST", "/disputes/{id}/start-review"},
		{"GET", "/admin/members/{id}/suspend"}, {"POST", "/admin/invitations/{id}/revoke"},
		{"POST", "/admin/announcements/{id}/delete"}, {"POST", "/admin/reports/{id}/handle"},
		{"POST", "/admin/adjustments/{id}/approve"},
	} {
		t.Run("web_path_"+tc.method+tc.path, func(t *testing.T) {
			path := "/n/town" + strings.ReplaceAll(tc.path, "{id}", url.PathEscape(bad))
			rec := identifierHTTP(wh, tc.method, path, sessions["admin"], nil, url.Values{}, true)
			if rec.Code != 404 {
				t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
			}
			if got := f.spy.count(bad); got != 0 {
				t.Fatalf("path target reached repository %d times", got)
			}
		})
	}
	for _, tc := range []struct {
		name, method, path string
		form               url.Values
		status             int
	}{
		{"counterparty", "POST", "/trades/new", url.Values{"counterparty_id": {bad}, "direction": {"pay"}, "amount": {"1"}}, 422},
		{"member", "POST", "/admin/adjust", url.Values{"member_id": {bad}, "direction": {"credit"}, "amount": {"1"}, "reason": {"reason"}}, 422},
		{"reversal", "POST", "/admin/reverse", url.Values{"tx": {bad}, "reason": {"reason"}}, 422},
		{"report", "POST", "/report", url.Values{"type": {"member"}, "id": {bad}, "category": {"other"}}, 404},
		{"notifications", "POST", "/inbox/read", url.Values{"id": {bad}}, 422},
		{"report_query", "GET", "/report?type=member&id=" + url.QueryEscape(bad), nil, 404},
		{"history_cursor", "GET", "/history?after=" + url.QueryEscape(bad), nil, 422},
		{"directory_cursor", "GET", "/directory?before=" + url.QueryEscape(bad), nil, 422},
		{"pending_cursor", "GET", "/admin/adjustments/pending?after=" + url.QueryEscape(bad), nil, 422},
	} {
		t.Run("web_"+tc.name, func(t *testing.T) {
			rec := identifierHTTP(wh, tc.method, "/n/town"+tc.path, sessions["admin"], nil, tc.form, true)
			if rec.Code != tc.status {
				t.Fatalf("status=%d want=%d body=%s", rec.Code, tc.status, rec.Body.String())
			}
			if got := f.spy.count(bad); got != 0 {
				t.Fatalf("form/filter target reached repository %d times", got)
			}
		})
	}
	// Auth, CSRF and endpoint role checks must keep their refusal precedence.
	adminPath := "/n/town/admin/members/" + url.PathEscape(bad) + "/suspend"
	for _, tc := range []struct {
		h                     http.Handler
		method, path, session string
		body                  any
		csrf                  bool
		want                  int
	}{
		{wh, "GET", adminPath, "", nil, false, 303},
		{wh, "GET", adminPath, sessions["alice"], nil, false, 403},
		{wh, "POST", adminPath, sessions["admin"], nil, false, 403},
		{ah, "POST", "/api/v1/admin/adjustments", "", obj{"member_id": bad}, false, 401},
		{ah, "POST", "/api/v1/admin/adjustments", sessions["admin"], obj{"member_id": bad}, false, 403},
		{ah, "POST", "/api/v1/admin/adjustments", sessions["alice"], obj{"member_id": bad, "direction": "credit", "amount_minor": "1", "reason": "reason"}, true, 403},
	} {
		rec := identifierHTTP(tc.h, tc.method, tc.path, tc.session, tc.body, nil, tc.csrf)
		if rec.Code != tc.want {
			t.Fatalf("precedence %s %s: status=%d want=%d body=%s", tc.method, tc.path, rec.Code, tc.want, rec.Body.String())
		}
	}
}

// SEC-024 / API-017: missing and hidden targets stay indistinguishable, while
// identifiers with separate contracts and empty optional references still work.
func TestIdentifierValidationPreservesOpaqueMissingAndNonRecordInputs(t *testing.T) {
	f := newIdentifierFixture(t)
	const missing = "018f0000-0000-7000-8000-ffffffffffff"
	const bad = "x' OR '1'='1"
	tu.Must(0, f.Svc.SuspendMember(f.Ctx, f.admin, f.net.ID, f.bobM.ID, "hide directory entry"))
	var reference string
	for _, id := range []string{f.bobM.ID, missing, bad} {
		_, err := f.Svc.GetMember(f.Ctx, f.alice, f.net.ID, id)
		if domain.CodeOf(err) != domain.CodeNotFound {
			t.Fatalf("id=%q err=%v", id, err)
		}
		if reference == "" {
			reference = err.Error()
		} else if err.Error() != reference {
			t.Fatalf("opaque missing response differs: %v", err)
		}
	}
	if f.spy.count(bad) != 0 || f.spy.count(missing) == 0 {
		t.Fatal("validation bypassed malformed or blocked well-formed missing reference")
	}
	// Nil marks all; malformed non-empty input must never become that sentinel.
	tu.Must(0, f.Svc.RestoreMember(f.Ctx, f.admin, f.net.ID, f.bobM.ID, "restore"))
	tu.Must(f.Svc.ProposeTransaction(f.Ctx, f.alice, f.net.ID, service.ProposeInput{CounterpartyID: f.bobM.ID, Direction: service.ProposerPays, Amount: 1, IdempotencyKey: f.Key()}))
	before := tu.Must(f.Svc.CountUnreadNotifications(f.Ctx, f.bob, f.net.ID))
	ns := tu.Must(f.Svc.ListNotifications(f.Ctx, f.bob, f.net.ID, service.NotificationQuery{}))
	if _, err := f.Svc.MarkNotificationsRead(f.Ctx, f.bob, f.net.ID, []string{ns[0].ID, bad}); domain.CodeOf(err) != domain.CodeValidation {
		t.Fatalf("malformed batch: %v", err)
	}
	if after := tu.Must(f.Svc.CountUnreadNotifications(f.Ctx, f.bob, f.net.ID)); after != before {
		t.Fatal("malformed batch partially marked notifications")
	}
	if n := tu.Must(f.Svc.MarkNotificationsRead(f.Ctx, f.bob, f.net.ID, []string{missing})); n != 0 {
		t.Fatal("unknown UUID unexpectedly marked a notification")
	}
	if n := tu.Must(f.Svc.MarkNotificationsRead(f.Ctx, f.bob, f.net.ID, nil)); n != int64(before) {
		t.Fatalf("empty-list sentinel changed: %d/%d", n, before)
	}
	// Network slugs, invitation credentials, and numeric audit cursors retain
	// their contracts; an absent optional listing reference remains usable above.
	tu.Must(f.Svc.NetworkBySlug(f.Ctx, "town"))
	guest := f.Person("guest")
	token, _ := tu.Must2(f.Svc.CreateInvitation(f.Ctx, f.admin, f.net.ID, service.InviteInput{}))
	if domain.ValidRecordID(token) {
		t.Fatal("fixture invitation unexpectedly resembles UUID")
	}
	tu.Must(f.Svc.RedeemInvitation(f.Ctx, guest, token, "Guest"))
	tu.Must(f.Svc.ListAudit(f.Ctx, f.admin, f.net.ID, service.AuditQuery{AfterSeq: 1, Limit: 5}))
	session, _, err := f.Svc.Login(f.Ctx, "admin", "correct horse battery")
	if err != nil {
		t.Fatal(err)
	}
	wh := web.NewHandlerWith(f.Svc, web.Options{CSRFKey: identifierCSRFKey, Clock: f.Clock})
	for _, path := range []string{"/n/town/admin/audit?after=1", "/n/town/audit?after=1"} {
		if rec := identifierHTTP(wh, "GET", path, session, nil, nil, false); rec.Code != 200 {
			t.Fatalf("numeric cursor refused: status=%d", rec.Code)
		}
	}
}
