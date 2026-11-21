package messages

import (
	"html/template"
	"strings"
	"testing"
)

// UX-047: decision messages keep exact money and member text, and distinguish
// an initial decision's deferred reversal from an atomic re-review refusal.
func TestDisputeDecisionCatalogueMessages(t *testing.T) {
	const name = `Zoë 100% %! $& "><img src=x>`
	const amount = "90071992547409.93"
	for _, tc := range []struct {
		key  string
		args []any
		want string
	}{
		{"dispute.decision.reversal_sentence", []any{amount, "Hours", name, "Ana"}, amount + " Hours go back from " + name + " to Ana."},
		{"dispute.decision.waiting", []any{name}, "If " + name + " does not have the credit available, the decision is recorded and the reversal waits as \"Decided, reversal not yet posted\" until you retry it."},
		{"dispute.decision.no_reversal", nil, "None. No credit moves."},
		{"dispute.decision.change_for", []any{name}, "Change for " + name},
		{"dispute.decision.amount_value", []any{"−" + amount, "Hours"}, "−" + amount + " Hours"},
		{"dispute.decision.balance_after", []any{name}, name + ": balance after"},
		{"dispute.decision.amount_max", []any{amount}, "must be at most the remaining reversible amount " + amount},
		{"dispute.rereview.amount_max", []any{amount}, "choose a positive amount no greater than the remaining reversible amount " + amount},
		{"dispute.rereview.atomic_explanation", nil, "Balances and permissions are checked again when you confirm. If the reversal cannot be posted, nothing is recorded."},
		{"dispute.rereview.upheld_posted", []any{"1", "credit"}, "Re-review decided, upheld: a further reversal of 1 credit has been posted."},
		{"dispute.rereview.partial_posted", []any{amount, "Hours"}, "Re-review decided, partly upheld: a further reversal of " + amount + " Hours has been posted."},
		{"dispute.rereview.confirmed", nil, "Re-review decided: the original decision stands."},
		{"dispute.rereview.decided", nil, "Re-review decided."},
	} {
		t.Run(tc.key, func(t *testing.T) {
			if _, ok := keys[tc.key]; !ok {
				t.Fatalf("decision sentence is not in the catalogue: %s", tc.key)
			}
			if got := Text(tc.key, tc.args...); got != tc.want {
				t.Fatalf("got %q, want %q", got, tc.want)
			}
		})
	}
}

// Both actual confirmation templates retain their financial controls. Full
// balance rows appear only when the handler's existing ShowBalances flag is set.
func TestDisputeDecisionCatalogueConfirmationTemplates(t *testing.T) {
	const unsafe = `Zoë 100% $& "><img src=x>`
	const amount = "90071992547409.93"
	balance := func(label string) map[string]any {
		return map[string]any{"Signed": label, "State": "available", "Spoken": label + " available"}
	}
	for _, name := range []string{"dispute_decide_review", "dispute_rereview_confirm"} {
		for _, reverses := range []bool{false, true} {
			for _, showBalances := range []bool{false, true} {
				d := map[string]any{"ID": "dispute-exact-id", "Outcome": "Partly upheld: part of the trade is reversed.",
					"Amount": amount, "Unit": "Hours", "Payer": unsafe, "Payee": "Ana", "Rationale": unsafe,
					"Reverses": reverses, "Proof": `signed-proof-100%<>&"`, "PayerChange": "+" + amount, "PayeeChange": "−" + amount,
					"ShowBalances": showBalances, "PayerBefore": balance("private-payer-before"), "PayerAfter": balance("private-payer-after"),
					"PayeeBefore": balance("private-payee-before"), "PayeeAfter": balance("private-payee-after")}
				p := disputeTemplatePage{D: d, Slug: "town", CSRF: "csrf-value", Idem: "idem-value",
					V: map[string]string{"outcome": "partially_upheld", "amount": amount, "rationale": unsafe, "waive_response": "1", "waive_reason": unsafe}}
				body := renderDisputeTemplate(t, name, p)
				action := "resolve"
				if name == "dispute_rereview_confirm" {
					action = "rereview-decide"
				}
				requireDisputeText(t, body, `action="/n/town/disputes/dispute-exact-id/`+action+`"`,
					`name="csrf" value="csrf-value"`, `name="idem" value="idem-value"`,
					`name="outcome" value="partially_upheld"`, `name="amount" value="`+amount+`"`,
					`name="rationale" value="`+unsafe+`"`, `name="step" value="confirm" class="danger"`,
					`name="step" value="back" class="secondary">Go back and edit</button>`)
				if reverses {
					requireDisputeText(t, body, amount+" Hours go back from Ana to "+unsafe+".")
				} else {
					requireDisputeText(t, body, "None. No credit moves.")
				}
				if name == "dispute_rereview_confirm" {
					requireDisputeText(t, body, `name="review_proof" value="`+d["Proof"].(string)+`"`,
						"Change for "+unsafe, "+"+amount+" Hours", "−"+amount+" Hours",
						"If the reversal cannot be posted, nothing is recorded.")
					if strings.Contains(body, "private-payer-before") != showBalances || strings.Contains(body, "private-payee-after") != showBalances {
						t.Fatal("re-review full-balance visibility changed")
					}
				} else {
					requireDisputeText(t, body, `name="waive_response" value="1"`, `name="waive_reason" value="`+unsafe+`"`)
					if reverses {
						requireDisputeText(t, body, `the reversal waits as "Decided, reversal not yet posted" until you retry it.`)
					}
				}
				if strings.Contains(body, "<img") || !strings.Contains(body, template.HTMLEscapeString(unsafe)) {
					t.Fatal("decision data crossed the contextual escaping boundary")
				}
			}
		}
	}
}
