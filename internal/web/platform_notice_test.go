package web

import (
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"

	"github.com/djshellshoxxx/circula/internal/api"
	"github.com/djshellshoxxx/circula/internal/domain"
	"github.com/djshellshoxxx/circula/internal/httpx"
	"github.com/djshellshoxxx/circula/internal/repository"
	"github.com/djshellshoxxx/circula/internal/service"
	"github.com/djshellshoxxx/circula/internal/testutil"
)

// UX-047, GOV-024, FR-038: the inbox describes the historical suspension in
// human terms without altering its reason, stored evidence or API representation.
func TestPlatformSuspensionInboxPreservesHistoryAndAPI(t *testing.T) {
	w, inbox := newPlatformNoticeWorld(t)
	token, _, err := w.env.Svc.CreateAPIToken(w.env.Ctx, w.admin, service.TokenInput{
		Name: "notification regression", Scopes: []domain.Scope{domain.ScopeRead}, NetworkID: w.net.ID,
	})
	if err != nil {
		t.Fatal(err)
	}
	h := api.NewHandlerWithOptions(w.env.Svc, api.Options{Now: w.env.Clock.Now})
	t.Cleanup(func() { api.CloseHandler(h) })
	apiBody := func() string {
		t.Helper()
		r := httptest.NewRequest(http.MethodGet, "/api/v1/notifications", nil)
		r.Header.Set("Authorization", "Bearer "+token)
		r.Header.Set("X-Network", w.net.ID)
		out := httptest.NewRecorder()
		h.ServeHTTP(out, r)
		if out.Code != http.StatusOK {
			t.Fatalf("notification API returned %d: %s", out.Code, out.Body.String())
		}
		return out.Body.String()
	}
	stored := func() []*service.Notification {
		t.Helper()
		ns, err := w.env.Svc.ListNotifications(w.env.Ctx, w.admin, w.net.ID, service.NotificationQuery{})
		if err != nil {
			t.Fatal(err)
		}
		return ns
	}
	wants := map[string]string{}
	check := func() {
		t.Helper()
		before, beforeAPI := stored(), apiBody()
		body := inbox()
		for id, want := range wants {
			row := platformNoticeHTMLRow(t, body, id)
			if !strings.Contains(row, "<p>"+template.HTMLEscapeString(want)+" &middot; ") {
				t.Errorf("notice %s did not render its exact escaped historical body %q", id, want)
			}
			if strings.Contains(row, "<img") {
				t.Error("operator reason became HTML")
			}
		}
		if after := stored(); !reflect.DeepEqual(after, before) {
			t.Fatal("reading the inbox changed stored notifications")
		}
		if afterAPI := apiBody(); afterAPI != beforeAPI {
			t.Fatal("reading the inbox changed the notification API response")
		}
		var response struct {
			Data []struct{ ID, Kind, Title, Body string }
		}
		if err := json.Unmarshal([]byte(beforeAPI), &response); err != nil {
			t.Fatal(err)
		}
		if len(response.Data) != len(before) {
			t.Fatalf("API has %d notices; stored history has %d", len(response.Data), len(before))
		}
		for i, n := range before {
			got := response.Data[i]
			if got.ID != n.ID || got.Kind != n.Kind || got.Title != n.Title || got.Body != n.Body {
				t.Fatalf("API notification text differs from stored history: %#v", got)
			}
		}
	}
	const suffix = ". Members can still read balances and history and download their data."
	sys := domain.SystemActor("platform-notice-regression")
	for _, tc := range []struct{ category, label string }{
		{domain.SuspendLegalOrder, "Legal order"},
		{domain.SuspendSecurityIncident, "Security incident"},
		{domain.SuspendPlatformAbuse, "Platform abuse"},
		{domain.SuspendHostingAgreement, "Hosting agreement"},
	} {
		reason := tc.category + ` stays literal; Zoë 100% %! $& "><img src=x>. Category: security_incident. Reason: quoted` + suffix
		if err := w.env.Svc.PlatformSuspendNetwork(w.env.Ctx, sys, service.PlatformSuspendInput{
			Network: w.net.ID, Operator: "test operator", Category: tc.category, Reason: reason,
		}); err != nil {
			t.Fatal(err)
		}
		rawBody := "Category: " + tc.category + ". Reason: " + reason + suffix
		found := false
		for _, n := range stored() {
			if n.Kind == service.NotifNetworkSuspended && n.Title == "The platform has suspended this network" && n.Body == rawBody {
				wants[n.ID] = "Category: " + tc.label + ". Reason: " + reason + suffix
				found = true
			}
		}
		if !found {
			t.Fatalf("producer did not retain the original %s notice", tc.category)
		}
		check() // Earlier notices must not take this newer suspension's category.
		if err := w.env.Svc.PlatformResumeNetwork(w.env.Ctx, sys, w.net.ID, "test operator", "resolved"); err != nil {
			t.Fatal(err)
		}
		check() // A lifted suspension must still retain its historical category.
	}
	const adminReason = "Category: security_incident. Reason: administrator text"
	if err := w.env.Svc.SuspendNetwork(w.env.Ctx, w.admin, w.net.ID, adminReason); err != nil {
		t.Fatal(err)
	}
	for _, n := range stored() {
		if n.Kind == service.NotifNetworkSuspended && n.Title == "Trading is paused" {
			wants[n.ID] = "An administrator has paused trading: " + adminReason + ". You can still read balances and history and download your data."
		}
	}
	if len(wants) != 5 {
		t.Fatalf("expected four platform notices and one administrator notice, got %d", len(wants))
	}
	check()
}

// UX-047, FR-038: unrecognized, redacted and noncanonical historical records
// stay ordinary escaped data, even if they resemble the known platform notice.
func TestPlatformSuspensionInboxKeepsUnrecognizedRecords(t *testing.T) {
	w, inbox := newPlatformNoticeWorld(t)
	const title = "The platform has suspended this network"
	const suffix = ". Members can still read balances and history and download their data."
	const body = "Category: security_incident. Reason: stored <img src=x> %!" + suffix
	cases := []struct{ name, kind, title, body, link string }{
		{"other kind", service.NotifNetworkResumed, title, body, ""},
		{"administrator title", service.NotifNetworkSuspended, "Trading is paused", body, ""},
		{"changed title", service.NotifNetworkSuspended, title + " <img src=x>", body, ""},
		{"unknown category", service.NotifNetworkSuspended, title, strings.Replace(body, "security_incident", "future_category", 1), ""},
		{"missing prefix", service.NotifNetworkSuspended, title, strings.TrimPrefix(body, "Category: "), ""},
		{"extra prefix", service.NotifNetworkSuspended, title, "Archived: " + body, ""},
		{"missing suffix", service.NotifNetworkSuspended, title, strings.TrimSuffix(body, suffix), ""},
		{"extra suffix", service.NotifNetworkSuspended, title, body + " archived", ""},
		{"empty reason", service.NotifNetworkSuspended, title, "Category: security_incident. Reason: " + suffix, ""},
		{"noncanonical reason", service.NotifNetworkSuspended, title, "Category: security_incident. Reason:  preserved whitespace " + suffix, ""},
		{"redacted body", service.NotifNetworkSuspended, title, "", ""},
		{"unexpected link", service.NotifNetworkSuspended, title, body, "/admin/network"},
		{"future event context", service.NotifNetworkSuspended, title, body, ""},
	}
	notices := make([]repository.Notification, 0, len(cases))
	for i, tc := range cases {
		n := repository.Notification{ID: fmt.Sprintf("018f0000-0000-7000-8000-%012x", 10000+i), NetworkID: w.net.ID, MemberID: w.adminM.ID,
			Kind: tc.kind, Title: tc.title, Body: tc.body, Link: tc.link, CreatedAt: w.env.Clock.Now()}
		if tc.name == "future event context" {
			n.EventDate = &domain.NotificationDate{Event: "future.event", At: w.env.Clock.Now()}
		}
		notices = append(notices, n)
	}
	if err := w.env.Store.Update(w.env.Ctx, func(tx repository.Tx) error {
		for _, n := range notices {
			if inserted, err := tx.InsertNotification(n); err != nil || !inserted {
				return fmt.Errorf("insert historical notice: inserted=%v, error=%v", inserted, err)
			}
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	page := inbox()
	for i, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			row := platformNoticeHTMLRow(t, page, notices[i].ID)
			if !strings.Contains(row, "<p>"+template.HTMLEscapeString(tc.body)+" &middot; ") ||
				!strings.Contains(row, ">"+template.HTMLEscapeString(tc.title)+"</") {
				t.Fatal("unrecognized or redacted stored text was reinterpreted")
			}
			if strings.Contains(row, "<img") {
				t.Fatal("historical notification became HTML")
			}
		})
	}
}

// Exercise the production HTTP middleware and templates without opening a TCP
// listener. The session is issued by the normal service authentication path.
func newPlatformNoticeWorld(t *testing.T) (*world, func() string) {
	t.Helper()
	env := testutil.NewEnv(t)
	w := &world{env: env, admin: env.Person("admin@example.org")}
	w.net, w.adminM = env.Network(w.admin, "town", nil)
	session, _, err := env.Svc.Login(env.Ctx, "admin@example.org", testPW)
	if err != nil {
		t.Fatal(err)
	}
	h := NewHandlerWith(env.Svc, Options{CSRFKey: []byte("0123456789abcdef0123456789abcdef"), Clock: env.Clock})
	return w, func() string {
		t.Helper()
		r := httptest.NewRequest(http.MethodGet, "/n/town/inbox", nil)
		r.AddCookie(&http.Cookie{Name: httpx.SessionCookie, Value: session})
		out := httptest.NewRecorder()
		h.ServeHTTP(out, r)
		if out.Code != http.StatusOK {
			t.Fatalf("inbox returned %d: %s", out.Code, out.Body.String())
		}
		return out.Body.String()
	}
}

func platformNoticeHTMLRow(t *testing.T, body, id string) string {
	t.Helper()
	field := `name="id" value="` + id + `"`
	i := strings.Index(body, field)
	if i < 0 {
		t.Fatalf("notification %s has no mark-read control", id)
	}
	start, end := strings.LastIndex(body[:i], "<li>"), strings.Index(body[i:], "</li>")
	if start < 0 || end < 0 {
		t.Fatalf("notification %s has no list row", id)
	}
	return body[start : i+end+len("</li>")]
}
