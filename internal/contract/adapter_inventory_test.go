package contract

import (
	"os"
	"regexp"
	"strings"
	"testing"
)

// This inventory expands every conceptual operation in spec07 section 8's MVP
// range, including each membership action. HTML shows some read resources in
// a composite page. These mappings qualify ledger/audit outcomes, not equality
// of HTML presentation and JSON response schemas.
var mvpAdapterInventory = []struct{ id, api, html, exception string }{
	{"API-040", "GET /network", "GET /n/{slug}/policy", ""},
	{"API-041", "GET /me", "GET /", ""},
	{"API-042", "GET /members", "GET /n/{slug}/directory", ""},
	{"API-043", "GET /members/{id}", "GET /n/{slug}/members/{id}", ""},
	{"API-044", "GET /accounts/me", "GET /n/{slug}/", ""},
	{"API-045", "GET /accounts/me/entries", "GET /n/{slug}/history", ""},
	{"API-046", "GET /listings", "GET /n/{slug}/listings", ""},
	{"API-046", "POST /listings", "POST /n/{slug}/listings/new", ""},
	{"API-047", "PATCH /listings/{id}", "POST /n/{slug}/listings/{id}/edit", ""},
	{"API-048", "POST /transactions", "POST /n/{slug}/trades/new (review, confirm)", ""},
	{"API-049", "GET /transactions", "GET /n/{slug}/trades", ""},
	{"API-049", "GET /transactions/{id}", "GET /n/{slug}/trades/{id}", ""},
	{"API-050", "GET /transactions/{id}/events", "GET /n/{slug}/trades/{id} (history section)", ""},
	{"API-051", "POST /transactions/{id}/accept", "POST /n/{slug}/trades/{id}/accept", ""},
	{"API-052", "POST /transactions/{id}/decline", "POST /n/{slug}/trades/{id}/decline", ""},
	{"API-053", "POST /transactions/{id}/cancel", "POST /n/{slug}/trades/{id}/cancel", ""},
	{"API-054", "POST /disputes", "POST /n/{slug}/trades/{id}/dispute (review, confirm)", ""},
	{"API-054", "GET /disputes", "GET /n/{slug}/disputes", ""},
	{"API-054", "GET /disputes/{id}", "GET /n/{slug}/disputes/{id}", ""},
	{"API-055", "POST /disputes/{id}/resolve", "POST /n/{slug}/disputes/{id}/resolve (review, confirm)", ""},
	{"API-056", "POST /admin/transactions/{id}/reversal", "POST /n/{slug}/admin/reverse (review, confirm; API reads both accounts at each stage)", ""},
	{"API-057", "POST /admin/adjustments", "POST /n/{slug}/admin/adjust (review, confirm; API reads target account at each stage)", ""},
	{"API-058", "GET /admin/policy", "GET /n/{slug}/admin/policy", ""},
	{"API-058", "PATCH /admin/policy", "POST /n/{slug}/admin/policy", ""},
	{"API-059", "GET /admin/members", "GET /n/{slug}/admin/members", ""},
	{"API-059", "POST /admin/members/{id}/approve", "POST /n/{slug}/admin/members/{id}/approve", ""},
	{"API-059", "POST /admin/members/{id}/deny", "POST /n/{slug}/admin/members/{id}/deny", ""},
	{"API-059", "POST /admin/members/{id}/suspend", "POST /n/{slug}/admin/members/{id}/suspend", ""},
	{"API-059", "POST /admin/members/{id}/reinstate", "POST /n/{slug}/admin/members/{id}/restore", ""},
	{"API-059", "POST /admin/members/{id}/remove", "POST /n/{slug}/admin/members/{id}/remove", ""},
	{"API-060", "PUT /admin/accounts/{id}/credit-limit", "POST /n/{slug}/admin/members/{id}/limit", ""},
	{"API-061", "POST /admin/invitations", "POST /n/{slug}/admin/invitations", ""},
	{"API-061", "POST /invitations/{id}/accept", "POST /join/{token}", ""},
	{"API-062", "GET /admin/audit-events", "GET /n/{slug}/admin/audit", ""},
	{"API-062a", "GET /audit-events", "GET /n/{slug}/audit", ""},
	{"API-063", "POST /admin/consistency-checks", "POST /n/{slug}/admin/check", ""},
	{"API-063", "GET /admin/consistency-checks/{id}", "", "HTML POST renders the result directly; API stored-result read is separately asserted read-only and successful."},
	{"API-064", "POST /admin/announcements", "POST /n/{slug}/admin/announcements", ""},
	{"API-064", "GET /admin/announcements", "GET /n/{slug}/admin/announcements", ""},
	{"API-064", "GET /network/announcements", "GET /n/{slug}/ (announcement section)", ""},
	{"API-064", "DELETE /admin/announcements/{id}", "POST /n/{slug}/admin/announcements/{id}/delete", ""},
	{"API-065", "POST /exports", "POST /n/{slug}/export or /n/{slug}/admin/exports", ""},
	{"API-065", "GET /exports/{id}", "", "HTML streams the ZIP from POST; API job status is checked against creation response."},
	{"API-065", "GET /exports/{id}/artefact", "", "HTML streams the ZIP from POST; API download size, checksum, ZIP and ownership are asserted."},
	{"API-066", "GET /tokens", "", "No HTML token management surface; API create/list/revoke/authentication/audit lifecycle is tested."},
	{"API-066", "POST /tokens", "", "No HTML token management surface; API create/list/revoke/authentication/audit lifecycle is tested."},
	{"API-066", "DELETE /tokens/{id}", "", "No HTML token management surface; API create/list/revoke/authentication/audit lifecycle is tested."},
	{"API-067", "POST /me/membership/departure", "POST /n/{slug}/leave", ""},
	{"API-067a", "POST /me/password", "POST /account/password", ""},
}

var operationPlaceholder = regexp.MustCompile(`\{[^}]+\}|[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}`)

func canonicalOperation(s string) string { return operationPlaceholder.ReplaceAllString(s, "{id}") }

func assertMVPInventory(t *testing.T, seen, htmlSeen map[string]bool) {
	t.Helper()
	for _, op := range mvpAdapterInventory {
		if !seen[op.api] {
			t.Errorf("unexercised MVP operation %s %s; exercised=%v", op.id, op.api, sortedKeys(seen))
		}
		if op.html != "" {
			primary := strings.SplitN(strings.SplitN(op.html, " (", 2)[0], " or ", 2)[0]
			if !htmlSeen[canonicalOperation(primary)] {
				t.Errorf("unexercised HTML counterpart %s %s", op.id, primary)
			}
		}
	}
}

// Adding an MVP conceptual requirement or a full method/path to its operation
// column requires a coverage entry. Shorthand membership actions are expanded
// explicitly in the inventory above rather than inferred from prose.
func TestMVPAdapterInventoryMatchesSpec(t *testing.T) {
	b, err := os.ReadFile("../../docs/specs/07-api-contract.md")
	if err != nil {
		t.Fatal(err)
	}
	_, section, found := strings.Cut(string(b), "## 8. Operations (MVP)")
	if !found {
		t.Fatal("MVP operations section missing")
	}
	section, _, found = strings.Cut(section, "## 9. Ledger")
	if !found {
		t.Fatal("MVP operations section end missing")
	}
	ids := map[string]bool{}
	ops := map[string]bool{}
	for _, op := range mvpAdapterInventory {
		if (op.html == "") == (op.exception == "") {
			t.Errorf("%s needs either an HTML pair or explicit exception", op.api)
		}
		if ops[op.api] {
			t.Errorf("duplicate operation %s", op.api)
		}
		ops[op.api] = true
		ids[op.id] = true
	}
	method := regexp.MustCompile("`((?:GET|POST|PATCH|PUT|DELETE) /[^`]+)`")
	for _, line := range strings.Split(section, "\n") {
		if !strings.HasPrefix(line, "| API-") {
			continue
		}
		cols := strings.Split(line, "|")
		id := strings.TrimSpace(cols[1])
		if id == "API-083" {
			continue
		} // explicitly labelled later extension, outside MVP conceptual set
		if !ids[id] {
			t.Errorf("new MVP requirement %s needs adapter coverage", id)
		}
		for _, match := range method.FindAllStringSubmatch(cols[2], -1) {
			if !ops[canonicalOperation(match[1])] {
				t.Errorf("spec operation %s (%s) absent from inventory", match[1], id)
			}
		}
	}
}
