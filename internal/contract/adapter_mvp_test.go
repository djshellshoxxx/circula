package contract

import (
	"archive/zip"
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http/httptest"
	"regexp"
	"strings"
	"testing"

	"github.com/djshellshoxxx/circula/internal/domain"
)

// ARC-002 / API-071: the same business operations run through real HTML and
// JSON handlers. Every step compares full ledger/status state and every audit
// event produced by the operation; no action class is filtered from the audit.
func TestMVPAdapterEquivalence(t *testing.T) {
	p := newAdapterPair(t)
	read := func(key, who, ap, wp string) {
		p.step(adapterStep{key: key, who: who, apiMethod: "GET", apiPath: ap, webMethod: "GET", webPath: wp})
	}
	read("API-040-network", "alice", "/network", "{base}/policy")
	p.step(adapterStep{key: "API-041-identity", who: "alice", apiMethod: "GET", apiPath: "/me", webMethod: "GET", webPath: "/", webStatus: 303})
	read("API-042-directory", "alice", "/members", "{base}/directory")
	read("API-043-member", "alice", "/members/{bob}", "{base}/members/{bob}")
	read("API-044-account", "alice", "/accounts/me", "{base}/")
	read("API-045-entries", "alice", "/accounts/me/entries", "{base}/history")
	p.step(adapterStep{key: "API-046-create", who: "bob", apiMethod: "POST", apiPath: "/listings", webMethod: "POST", webPath: "{base}/listings/new", apiStatus: 201,
		json: m{"kind": "offer", "title": "Garden help", "description": "Two hours", "price_minor": "2500"},
		form: map[string]string{"type": "offer", "title": "Garden help", "description": "Two hours", "price": "25.00"}})
	p.remember("listing", "listings", "title=?", "Garden help")
	read("API-046-list", "bob", "/listings", "{base}/listings")
	p.step(adapterStep{key: "API-047-edit", who: "bob", apiMethod: "PATCH", apiPath: "/listings/{listing}", webMethod: "POST", webPath: "{base}/listings/{listing}/edit",
		json: m{"title": "Garden work", "description": "Three hours", "price_minor": "3000", "status": "active"},
		form: map[string]string{"title": "Garden work", "description": "Three hours", "price": "30.00", "status": "active"}})
	propose := func(alias, amount string) {
		p.step(adapterStep{key: "API-048-" + alias, who: "alice", apiMethod: "POST", apiPath: "/transactions", webMethod: "POST", webPath: "{base}/trades/new", apiStatus: 201, review: true,
			json: m{"counterparty_member_id": "{bob}", "direction": "pay", "amount_minor": amount, "description": alias, "listing_id": "{listing}"},
			form: map[string]string{"counterparty_id": "{bob}", "direction": "pay", "amount": minorDecimal(amount), "note": alias, "listing_id": "{listing}"}})
		p.remember(alias, "transactions", "note=?", alias)
	}
	act := func(key, who, alias, action string) {
		p.step(adapterStep{key: key, who: who, apiMethod: "POST", apiPath: "/transactions/{" + alias + "}/" + action, webMethod: "POST", webPath: "{base}/trades/{" + alias + "}/" + action})
	}
	propose("completed", "2500")
	read("API-049-list", "alice", "/transactions", "{base}/trades")
	read("API-049-detail", "bob", "/transactions/{completed}", "{base}/trades/{completed}")
	read("API-050-events", "bob", "/transactions/{completed}/events", "{base}/trades/{completed}")
	act("API-051-accept", "bob", "completed", "accept")
	propose("declined", "100")
	act("API-052-decline", "bob", "declined", "decline")
	propose("cancelled", "100")
	act("API-053-cancel", "alice", "cancelled", "cancel")
	p.step(adapterStep{key: "API-054-open", who: "alice", apiMethod: "POST", apiPath: "/disputes", webMethod: "POST", webPath: "{base}/trades/{completed}/dispute", apiStatus: 201, review: true,
		json: m{"transaction_id": "{completed}", "category": "quality", "statement": "Not as agreed"}, form: map[string]string{"category": "quality", "statement": "Not as agreed"}})
	p.remember("dispute", "disputes", "statement=?", "Not as agreed")
	read("API-054-list", "alice", "/disputes", "{base}/disputes")
	read("API-054-detail", "bob", "/disputes/{dispute}", "{base}/disputes/{dispute}")
	p.step(adapterStep{key: "API-055-resolve", who: "admin", apiMethod: "POST", apiPath: "/disputes/{dispute}/resolve", webMethod: "POST", webPath: "{base}/disputes/{dispute}/resolve", review: true,
		json: m{"outcome": "upheld", "reason": "Evidence confirms the complaint"}, form: map[string]string{"outcome": "upheld", "rationale": "Evidence confirms the complaint"}})
	propose("reversible", "700")
	act("API-051-accept-second", "bob", "reversible", "accept")
	p.step(adapterStep{key: "API-056-reverse", who: "admin", apiMethod: "POST", apiPath: "/admin/transactions/{reversible}/reversal", webMethod: "POST", webPath: "{base}/admin/reverse", apiStatus: 201, review: true,
		apiAccountReads: []string{"{alice}", "{bob}"}, json: m{"amount_minor": "700", "reason": "Duplicate trade"}, form: map[string]string{"tx": "{reversible}", "amount": "7.00", "reason": "Duplicate trade"}})
	p.step(adapterStep{key: "API-057-adjust", who: "admin", apiMethod: "POST", apiPath: "/admin/adjustments", webMethod: "POST", webPath: "{base}/admin/adjust", apiStatus: 201, review: true,
		apiAccountReads: []string{"{bob}"}, json: m{"member_id": "{bob}", "direction": "credit", "amount_minor": "250", "reason": "Community work"}, form: map[string]string{"member_id": "{bob}", "direction": "credit", "amount": "2.50", "reason": "Community work"}})
	read("API-058-read", "admin", "/admin/policy", "{base}/admin/policy")
	policy := equivalentPolicyForm(p.web.net.Policy)
	policy["default_credit_limit"], policy["reason"] = "200.00", "Raise default limit"
	p.step(adapterStep{key: "API-058-update", who: "admin", apiMethod: "PATCH", apiPath: "/admin/policy", webMethod: "POST", webPath: "{base}/admin/policy",
		json: m{"default_credit_limit_minor": "20000", "reason": "Raise default limit"}, form: policy})
	read("API-059-members", "admin", "/admin/members", "{base}/admin/members")
	for _, a := range []struct{ key, member, api, web string }{{"approve", "pending", "approve", "approve"}, {"deny", "denied", "deny", "deny"}, {"suspend", "carol", "suspend", "suspend"}, {"reinstate", "carol", "reinstate", "restore"}} {
		p.step(adapterStep{key: "API-059-" + a.key, who: "admin", apiMethod: "POST", apiPath: "/admin/members/{" + a.member + "}/" + a.api, webMethod: "POST", webPath: "{base}/admin/members/{" + a.member + "}/" + a.web,
			json: m{"reason": "Membership review"}, form: map[string]string{"reason": "Membership review"}})
	}
	p.step(adapterStep{key: "API-059-remove", who: "admin", apiMethod: "POST", apiPath: "/admin/members/{carol}/remove", webMethod: "POST", webPath: "{base}/admin/members/{carol}/remove",
		json: m{"reason_code": "rule_breach", "reason": "Repeated breach"}, form: map[string]string{"reason_code": "rule_breach", "reason": "Repeated breach"}})
	p.step(adapterStep{key: "API-060-limit", who: "admin", apiMethod: "PUT", apiPath: "/admin/accounts/{alice}/credit-limit", webMethod: "POST", webPath: "{base}/admin/members/{alice}/limit",
		json: m{"credit_limit_minor": "15000", "reason": "Agreed limit"}, form: map[string]string{"amount": "150.00", "reason": "Agreed limit"}})
	r := p.step(adapterStep{key: "API-061-invite", who: "admin", apiMethod: "POST", apiPath: "/admin/invitations", webMethod: "POST", webPath: "{base}/admin/invitations", apiStatus: 201, webStatus: 200,
		json: m{"ttl_seconds": 604800}, form: map[string]string{"days": "7"}})
	p.api.ids["invitation"] = get[idOnly](t, r[0]).Token
	match := regexp.MustCompile(`/join/(inv_[A-Za-z0-9_-]+)`).FindStringSubmatch(r[1].Body.String())
	if len(match) != 2 {
		t.Fatalf("invitation link missing: %s", trunc(r[1].Body.String()))
	}
	p.web.ids["invitation"] = match[1]
	p.step(adapterStep{key: "API-061-accept", who: "invitee", apiMethod: "POST", apiPath: "/invitations/{invitation}/accept", webMethod: "POST", webPath: "/join/{invitation}", apiStatus: 201,
		json: m{"display_name": "Invited member"}, form: map[string]string{"display_name": "Invited member"}})
	read("API-062-admin-audit", "admin", "/admin/audit-events", "{base}/admin/audit")
	read("API-062a-member-audit", "alice", "/audit-events", "{base}/audit")
	r = p.step(adapterStep{key: "API-063-check", who: "admin", apiMethod: "POST", apiPath: "/admin/consistency-checks", webMethod: "POST", webPath: "{base}/admin/check", apiStatus: 201, webStatus: 200})
	check := get[idOnly](t, r[0]).ID
	t.Run("API-063-unpaired-result", func(t *testing.T) {
		p.api.t = t
		mark := p.api.auditMark()
		before := p.api.normalize(p.api.ledger())
		w := p.api.request("admin", "GET", "/api/v1/admin/consistency-checks/"+check, nil, nil)
		p.api.requireStatus(w, 200)
		var body struct {
			OK bool `json:"ok"`
		}
		if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil || !body.OK {
			t.Fatalf("stored check failed: %s", w.Body.String())
		}
		if before != p.api.normalize(p.api.ledger()) || len(p.api.auditAfter(mark)) != 0 {
			t.Fatal("stored result read changed state")
		}
	})
	p.api.t = t
	p.step(adapterStep{key: "API-064-publish", who: "admin", apiMethod: "POST", apiPath: "/admin/announcements", webMethod: "POST", webPath: "{base}/admin/announcements", apiStatus: 201,
		json: m{"title": "Working party", "body": "Saturday in the garden"}, form: map[string]string{"title": "Working party", "body": "Saturday in the garden"}})
	read("API-064-admin-list", "admin", "/admin/announcements", "{base}/admin/announcements")
	read("API-064-member-list", "alice", "/network/announcements", "{base}/")
	p.remember("announcement", "announcements", "title=?", "Working party")
	p.step(adapterStep{key: "API-064-retract", who: "admin", apiMethod: "DELETE", apiPath: "/admin/announcements/{announcement}", webMethod: "POST", webPath: "{base}/admin/announcements/{announcement}/delete", apiStatus: 204})
	for _, scope := range []string{"mine", "ledger"} {
		who, wp := "alice", "{base}/export"
		if scope == "ledger" {
			who, wp = "admin", "{base}/admin/exports"
		}
		r = p.step(adapterStep{key: "API-065-export-" + scope, who: who, apiMethod: "POST", apiPath: "/exports", webMethod: "POST", webPath: wp, apiStatus: 201, webStatus: 200,
			json: m{"scope": scope, "format": "json"}, form: map[string]string{"scope": scope, "format": "json"}})
		t.Run("API-065-unpaired-lifecycle-"+scope, func(t *testing.T) { p.api.t = t; assertExportLifecycle(t, p.api, who, r[0], r[1]) })
		p.api.t = t
	}
	p.step(adapterStep{key: "API-067-departure", who: "dave", apiMethod: "POST", apiPath: "/me/membership/departure", webMethod: "POST", webPath: "{base}/leave",
		json: m{"reason": "Moving away"}, form: map[string]string{"reason": "Moving away", "confirm": "1"}})
	p.step(adapterStep{key: "API-067a-password", who: "dave", apiMethod: "POST", apiPath: "/me/password", webMethod: "POST", webPath: "/account/password", apiStatus: 204,
		json: m{"current_password": equivalencePassword, "new_password": "a different strong password"}, form: map[string]string{"current_password": equivalencePassword, "password": "a different strong password", "password2": "a different strong password"}})
	t.Run("API-066-unpaired-token-lifecycle", func(t *testing.T) { p.api.t = t; assertTokenLifecycle(t, p.api) })
	p.api.t = t
	assertMVPInventory(t, p.api.seen, p.web.seen)
}

func minorDecimal(s string) string {
	for len(s) < 3 {
		s = "0" + s
	}
	return s[:len(s)-2] + "." + s[len(s)-2:]
}

func equivalentPolicyForm(p domain.Policy) map[string]string {
	v := map[string]string{"currency_name": p.CurrencyName, "join_mode": string(p.JoinMode), "departure_rule": string(p.DepartureRule)}
	for k, n := range map[string]int64{"default_credit_limit": p.DefaultCreditLimit, "max_tx_amount": p.MaxTxAmount, "signup_grant": p.SignupGrant} {
		v[k] = domain.FormatAmount(n, p.Decimals)
	}
	for k, n := range map[string]int{"proposal_expiry_days": p.ProposalExpiryDays, "dispute_window_days": p.DisputeWindowDays, "limit_warning_days": p.LimitWarningDays, "response_period_days": p.ResponsePeriodDays, "re_review_window_days": p.ReReviewWindowDays, "limit_warning_percent": p.LimitWarningPercent, "inactivity_days": p.InactivityDays} {
		v[k] = fmt.Sprint(n)
	}
	if p.DefaultFlagBlocksPayer {
		v["default_flag_blocks_payer"] = "1"
	}
	return v
}

func assertExportLifecycle(t *testing.T, f *adapterFixture, who string, created, htmlDownload *httptest.ResponseRecorder) {
	t.Helper()
	var job struct {
		ID, Status, SHA256 string
		Bytes              int64
		URL                string `json:"artefact_url"`
	}
	if err := json.Unmarshal(created.Body.Bytes(), &job); err != nil || job.Status != "complete" || job.URL == "" {
		t.Fatalf("bad completed export: %s", created.Body.String())
	}
	mark := f.auditMark()
	before := f.normalize(f.ledger())
	status := f.request(who, "GET", "/api/v1/exports/"+job.ID, nil, nil)
	f.requireStatus(status, 200)
	if !bytes.Equal(status.Body.Bytes(), created.Body.Bytes()) {
		t.Fatal("stored export status changed")
	}
	dl := f.request(who, "GET", job.URL, nil, nil)
	f.requireStatus(dl, 200)
	sum := sha256.Sum256(dl.Body.Bytes())
	if int64(dl.Body.Len()) != job.Bytes || hex.EncodeToString(sum[:]) != job.SHA256 {
		t.Fatal("download differs from checksum/size in completed job")
	}
	for _, b := range [][]byte{dl.Body.Bytes(), htmlDownload.Body.Bytes()} {
		z, err := zip.NewReader(bytes.NewReader(b), int64(len(b)))
		if err != nil || len(z.File) == 0 {
			t.Fatalf("invalid export ZIP: %v", err)
		}
		found := false
		for _, file := range z.File {
			r, err := file.Open()
			if err != nil {
				t.Fatal(err)
			}
			data, err := io.ReadAll(r)
			_ = r.Close()
			if err != nil {
				t.Fatal(err)
			}
			if strings.Contains(file.Name, "manifest") {
				found = true
				if !json.Valid(data) {
					t.Fatal("invalid manifest")
				}
			}
		}
		if !found {
			t.Fatal("export lacks manifest")
		}
	}
	if before != f.normalize(f.ledger()) || len(f.auditAfter(mark)) != 0 {
		t.Fatal("job/download read changed ledger or audit")
	}
	f.requireStatus(f.request("bob", "GET", job.URL, nil, nil), 404)
}

func assertTokenLifecycle(t *testing.T, f *adapterFixture) {
	t.Helper()
	before := f.normalize(f.ledger())
	w := f.request("alice", "POST", "/api/v1/tokens", m{"name": "equivalence token", "scopes": []string{"read"}}, nil)
	f.requireStatus(w, 201)
	tok := get[idOnly](t, w)
	if tok.ID == "" || tok.Token == "" {
		t.Fatal("token create did not return one-time credential")
	}
	listed := f.request("alice", "GET", "/api/v1/tokens", nil, nil)
	f.requireStatus(listed, 200)
	if !strings.Contains(listed.Body.String(), tok.ID) || strings.Contains(listed.Body.String(), tok.Token) {
		t.Fatal("token list lost metadata or disclosed the credential")
	}
	f.requireStatus(f.request("alice", "GET", "/api/v1/network", nil, map[string]string{"Authorization": "Bearer " + tok.Token}), 200)
	f.requireStatus(f.request("alice", "DELETE", "/api/v1/tokens/"+tok.ID, nil, nil), 204)
	f.requireStatus(f.request("alice", "GET", "/api/v1/network", nil, map[string]string{"Authorization": "Bearer " + tok.Token}), 401)
	if before != f.normalize(f.ledger()) {
		t.Fatal("token lifecycle changed network domain state")
	}
	for _, action := range []string{domain.ActTokenCreated, domain.ActTokenRevoked} {
		rows := f.rows("SELECT COUNT(*) FROM audit_events WHERE action=? AND target_id=?", action, tok.ID)
		if rows[0][0].(int64) != 1 {
			t.Fatalf("token lifecycle audit %s missing or duplicated", action)
		}
	}
}
