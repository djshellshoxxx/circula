package messages

import "testing"

// UX-047, DAT-016, FR-052: the event total is a count; the ledger sequence
// remains an exact, ungrouped reference, including values above 2^53.
func TestAuditCheckSummaryKeepsExactReference(t *testing.T) {
	const seq = "9223372036854775807"
	for _, tc := range []struct {
		count int64
		want  string
	}{
		{0, "Ledger sequence " + seq + "; 0 audit events checked."},
		{1, "Ledger sequence " + seq + "; 1 audit event checked."},
		{1000, "Ledger sequence " + seq + "; 1,000 audit events checked."},
	} {
		if got := Text("admin.check.summary", tc.count, seq); got != tc.want {
			t.Errorf("count %d: got %q, want %q", tc.count, got, tc.want)
		}
	}
}

// UX-047: catalogue wording preserves the audit/check/export consequences and
// the member view's privacy explanation without changing technical identifiers.
func TestAuditExportMessages(t *testing.T) {
	for key, want := range map[string]string{
		"admin.audit.title":             "Audit and checks",
		"admin.audit.system":            "System",
		"admin.audit.check_explanation": "Recomputes balances and checks the audit chain. The run is recorded.",
		"admin.check.passed":            "Check passed: no problems found.",
		"admin.check.failed":            "Check failed: problems found.",
		"admin.exports.explanation":     "Exports are recorded in the audit log and download as one zip file.",
		"admin.exports.scope.ledger":    "Ledger: accounts, trades, entries",
		"admin.exports.scope.members":   "Members and roles",
		"admin.exports.scope.audit":     "Audit history",
		"admin.exports.scope.full":      "Everything",
		"admin.exports.format.csv":      "CSV",
		"admin.exports.format.json":     "JSON",
		"my_audit.explanation":          "Events recorded about your account. Other people's names and details are left out.",
		"my_audit.staff":                "Network staff",
		"my_audit.system":               "System",
	} {
		if got := Text(key); got != want {
			t.Errorf("%s: got %q, want %q", key, got, want)
		}
	}
}
