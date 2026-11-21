package messages

import "testing"

// SEC-018, UX-047: this migration preserves the existing scope, retention and
// irreversible-action wording, including the conditional account-login effect.
func TestErasureMessages(t *testing.T) {
	const name = `Zoë 100% $& "><img src=x>` + "\x00%!"
	for _, tc := range []struct {
		key  string
		args []any
		want string
	}{
		{"erasure.member_title", []any{name}, "Erase personal data of " + name + "?"},
		{"erasure.member_heading", []any{name}, "Erase the personal data of " + name},
		{"erasure.self_title", nil, "Erase my personal data?"},
		{"erasure.self_heading", nil, "Erase my personal data in this network"},
		{"erasure.action", nil, "Erase personal data"},
		{"erasure.consequence", nil, "Irreversible. The member's name, listings, trade notes and dispute statements are replaced or cleared. Balances and the ledger stay exactly as they are."},
		{"erasure.irreversible", nil, "This cannot be undone."},
		{"erasure.name_effect", nil, `The display name is replaced by "Former member" and a short code.`},
		{"erasure.listings_effect", nil, "Listings are archived and their text is removed."},
		{"erasure.notes_effect", nil, "Notes written on trades and statements written in disputes are removed."},
		{"erasure.login_effect", nil, "If this is the person's only membership, the login is erased and the password stops working."},
		{"erasure.records_effect", nil, "Balances, trades, amounts and the audit log stay as they are, because the network's accounts must keep adding up. Anything still open (a proposed trade or a dispute) blocks erasure until it is settled."},
		{"erasure.reason_optional", nil, "Optional. It is written to the audit log, which is never rewritten, so do not put names or contact details in it."},
		{"erasure.reason_required", nil, "Required. It is written to the audit log, which is never rewritten, so do not put names or contact details in it."},
		{"erasure.confirm_label", nil, "I understand this cannot be undone"},
		{"erasure.confirm_required", nil, "Not done. Tick the box to confirm. Nothing changed."},
		{"erasure.confirm_error", nil, "Confirm: tick the box to show you understand this cannot be undone."},
		{"erasure.member_done", nil, "Personal data erased. The member now appears as a former member."},
		{"erasure.self_done", nil, "Your personal data in this network has been erased."},
		{"erasure.go_back", nil, "Go back"},
	} {
		if got := Text(tc.key, tc.args...); got != tc.want {
			t.Errorf("%s: got %q, want %q", tc.key, got, tc.want)
		}
	}
}
