package web

import (
	"bytes"
	"html"
	"strings"
	"testing"
	"time"

	"github.com/djshellshoxxx/circula/internal/domain"
)

func renderAdminAmountPage(t *testing.T, name string, p *page) string {
	t.Helper()
	var out bytes.Buffer
	if err := mustLoadTemplates()[name].ExecuteTemplate(&out, "base", p); err != nil {
		t.Fatal(err)
	}
	return out.String()
}

func wantAdminAmountText(t *testing.T, body string, wants ...string) {
	t.Helper()
	for _, want := range wants {
		if !strings.Contains(html.UnescapeString(body), want) {
			t.Errorf("administrator financial page lacks %q", want)
		}
	}
}

// UX-047, UX-022: every administrator amount uses the corresponding minor-unit
// value for inflection while its displayed decimal string stays exact.
func TestAdminAmountUnitTemplates(t *testing.T) {
	for _, tc := range []struct {
		name, unit string
		amount     int64
		dec        int
		want       string
	}{
		{"singular", "credit", 100, 2, "1.00 credit"},
		{"plural", "credit", 250, 2, "2.50 credits"},
		{"already plural singular", "Hours", 100, 2, "1.00 Hours"},
		{"already plural", "Hours", 250, 2, "2.50 Hours"},
		{"zero decimal", "credit", 1, 0, "1 credit"},
		{"four decimals", "credit", 10000, 4, "1.0000 credit"},
		{"HTML-sensitive", "<hours>&", 100, 2, "1.00 <hours>&"},
		{"exact large amount", "Hours", 9007199254740993, 2, "90071992547409.93 Hours"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			amount := domain.FormatAmount(tc.amount, tc.dec)
			rv := adjustReview{Name: "Zoë", Amount: amount, AmountMinor: tc.amount}
			trade := reverseTrade{ID: "trade-id", Payer: "Ana", Payee: "Ben", Amount: amount, AmountMinor: tc.amount,
				Reversed: domain.FormatAmount(0, tc.dec), Remaining: amount, RemainingMinor: tc.amount}
			for name, data := range map[string]any{
				"admin_adjust_review":  &rv,
				"admin_adjust_approve": &pendingApprovalData{adjustReview: rv, ID: "request-id"},
				"admin_adjust_pending": &pendingData{Threshold: amount, ThresholdMinor: tc.amount,
					Rows: []pendingRow{{ID: "request-id", Dir: "Credit (balance goes up)", Amount: amount, AmountMinor: tc.amount}}},
				"admin_reverse":        &reverseForm{Trade: &trade},
				"admin_reverse_review": &reverseForm{Review: &reverseReview{reverseTrade: trade}},
			} {
				t.Run(name, func(t *testing.T) {
					body := renderAdminAmountPage(t, name, &page{Slug: "town", Dec: tc.dec, Unit: tc.unit, D: data})
					switch name {
					case "admin_adjust_pending":
						wantAdminAmountText(t, body, "Adjustments above "+tc.want+" need approval", "Credit (balance goes up): "+tc.want+"</td>")
					case "admin_reverse_review":
						wantAdminAmountText(t, body, "<dt>Trade</dt><dd>Ana paid Ben "+tc.want+"</dd>")
					default:
						wantAdminAmountText(t, body, "<dt>Amount</dt><dd>"+tc.want+"</dd>")
					}
					if strings.Contains(body, "Hourss") || strings.Contains(body, "<hours>") {
						t.Fatal("unit was double-suffixed or rendered as markup")
					}
					if tc.unit == "<hours>&" && !strings.Contains(body, "&lt;hours&gt;&amp;") {
						t.Fatal("custom unit was not escaped")
					}
				})
			}
		})
	}
}

func TestAdminReversalIndependentUnitAmounts(t *testing.T) {
	body := renderAdminAmountPage(t, "admin_reverse", &page{Slug: "town", Dec: 2, Unit: "credit",
		D: &reverseForm{Trade: &reverseTrade{Amount: "2.50", AmountMinor: 250, Reversed: "1.00", ReversedMinor: 100,
			Remaining: "1.50", RemainingMinor: 150}}})
	wantAdminAmountText(t, body, "<dt>Amount</dt><dd>2.50 credits</dd>", "<dt>Already reversed</dt><dd>1.00 credit</dd>",
		"<dt>Left to reverse</dt><dd>1.50 credits</dd>")
}

// SEC-011, UX-022: the review/back controls retain submitted values, while an
// approval confirmation carries only the stored-request proof and CSRF token.
func TestAdminAdjustmentFormAndConfirmTemplates(t *testing.T) {
	const unsafe = `<img src=x onerror="alert(1)"> & Zoë 100%`
	for _, dir := range []string{"credit", "debit"} {
		p := &page{Slug: "town", Dec: 2, Unit: "Hours", CSRF: "csrf-value", Idem: "idem-value",
			V: map[string]string{"member_id": "member-id", "direction": dir, "amount": "90071992547409.93", "reason": unsafe, "related_transaction_id": "original-id"},
			D: &adjustData{Members: []option{{"", "Choose a member"}, {"member-id", unsafe}}}}
		body := renderAdminAmountPage(t, "admin_adjust", p)
		wantAdminAmountText(t, body, "Every adjustment needs a reason. You will review the effect before it is posted.",
			`<form method="post" action="/n/town/admin/adjust" novalidate>`, `name="csrf" value="csrf-value"`, `name="idem" value="idem-value"`,
			`name="step" value="review"`, `for="f-member_id">Member</label>`, `value="member-id" selected>`+unsafe,
			"<legend>Direction</legend>", `value="`+dir+`" checked`, "Credit: the member's balance goes up", "Debit: the member's balance goes down",
			"Amount in credits, up to 2 decimal places.", `inputmode="decimal" aria-describedby="h-amount" required`,
			`for="f-reason">Reason</label>`, "Required. Recorded in the audit log.", `maxlength="500" aria-describedby="h-reason" required`,
			"Related transaction (optional)", "Transaction ID of a posted entry involving this member. Leave blank for a standalone correction.",
			`name="related_transaction_id" type="text" value="original-id"`, ">Review adjustment</button>")
		if strings.Contains(body, "<img src=x") || !strings.Contains(body, "&lt;img src=x") {
			t.Fatal("adjustment form content crossed the escaping boundary")
		}
		delta, direction := int64(9007199254740993), "Credit (balance goes up)"
		if dir == "debit" {
			delta, direction = -delta, "Debit (balance goes down)"
		}
		rv := adjustReview{Name: unsafe, Dir: direction, Amount: "90071992547409.93", AmountMinor: 9007199254740993,
			Reason: unsafe, RelatedTxID: "original-id", Before: makeBalance(0, 2, "Hours"), After: makeBalance(delta, 2, "Hours")}
		p.D = &rv
		body = renderAdminAmountPage(t, "admin_adjust_review", p)
		wantAdminAmountText(t, body, "<dt>Member</dt><dd>"+unsafe+"</dd>", "<dt>Direction</dt><dd>"+direction+"</dd>",
			"<dt>Amount</dt><dd>90071992547409.93 Hours</dd>", "<dt>Reason</dt><dd>"+unsafe+"</dd>",
			`<dt>Related transaction</dt><dd><a href="/n/town/trades/original-id">The original entry</a></dd>`,
			"Their balance now", "Their balance after", rv.After.Signed,
			`<form method="post" action="/n/town/admin/adjust">`, `name="csrf" value="csrf-value"`, `name="idem" value="idem-value"`,
			`name="member_id" value="member-id"`, `name="direction" value="`+dir+`"`, `name="amount" value="90071992547409.93"`,
			`name="related_transaction_id" value="original-id"`, `name="reason" value="`+unsafe+`"`,
			`name="step" value="confirm" class="danger">Post adjustment</button>`, `name="step" value="back" class="secondary">Go back and edit</button>`)
		if strings.Contains(body, "<img src=x") {
			t.Fatal("adjustment review content crossed the escaping boundary")
		}
		p.D = &pendingApprovalData{adjustReview: rv, ID: "request-id", Proposer: unsafe, Proof: `proof<&"`}
		body = renderAdminAmountPage(t, "admin_adjust_approve", p)
		wantAdminAmountText(t, body, "Nothing is posted until you confirm approval.",
			"The request, permissions and available credit are checked again when you confirm. Balances may change while you review.",
			"<dt>Requested by</dt><dd>"+unsafe+"</dd>", "<dt>Reason</dt><dd>"+unsafe+"</dd>", rv.After.Signed,
			`href="/n/town/trades/original-id">The original entry</a>`,
			`<form method="post" action="/n/town/admin/adjustments/request-id/approve">`, `name="csrf" value="csrf-value"`,
			`name="review_proof" value="proof<&""`, `name="step" value="confirm" class="danger">Approve and post</button>`,
			`name="step" value="back" class="secondary">Go back without approving</button>`)
		for _, field := range []string{"member_id", "direction", "amount", "reason", "related_transaction_id", "idem"} {
			if strings.Contains(body, `name="`+field+`"`) {
				t.Errorf("approval confirmation gained a submitted %s field", field)
			}
		}
		if strings.Contains(body, "<img src=x") {
			t.Fatal("approval names or reason crossed the escaping boundary")
		}
	}
}

// SEC-011: preserve the current template gates, including the common decision
// gate for approve/reject and the separate proposer-cancellation gate.
func TestAdminAdjustmentPendingActionTemplates(t *testing.T) {
	const unsafe = `<img src=x onerror="alert(1)"> & Zoë 100%`
	at := time.Date(2026, time.November, 19, 12, 0, 0, 0, time.UTC)
	for _, tc := range []struct {
		name, status                            string
		approve, reject, cancel, expired, other bool
	}{
		{"decider", "pending", true, true, false, false, false},
		{"proposer", "pending", false, false, true, false, false},
		{"ineligible", "pending", false, false, false, false, true},
		{"expired", "pending", false, false, false, true, false},
		{"decided", "approved", false, false, false, false, false},
		{"decision gate", "pending", true, false, false, false, false},
		{"reject flag alone", "pending", false, true, false, false, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			row := pendingRow{ID: "request-id", Target: unsafe, Proposer: unsafe, Dir: "Debit (balance goes down)",
				Amount: "90071992547409.93", AmountMinor: 9007199254740993, Reason: unsafe, RelatedTxID: "original-id",
				Created: at, Expires: at.Add(time.Hour), Status: tc.status, CanApprove: tc.approve, CanReject: tc.reject, CanCancel: tc.cancel, Expired: tc.expired}
			if tc.status == "approved" {
				row.Decider = unsafe
			}
			body := renderAdminAmountPage(t, "admin_adjust_pending", &page{Slug: "town", NetName: unsafe, Dec: 2, Unit: "Hours", CSRF: "csrf-value",
				V: map[string]string{"status": "pending"}, D: &pendingData{Threshold: "1.00", ThresholdMinor: 100, Next: "next-id", Rows: []pendingRow{row}}})
			wantAdminAmountText(t, body, "Adjustments above 1.00 Hours need approval from a second administrator before anything is posted.",
				"A request that is not decided in time expires and posts nothing.", `for="f-status">Show requests</label>`,
				`value="pending" selected>Waiting for approval</option>`, `aria-label="Adjustment requests table"`,
				"<caption>Adjustment requests in "+unsafe+"</caption>", "Debit (balance goes down): 90071992547409.93 Hours",
				`Adjusts <a href="/n/town/trades/original-id">the original entry</a>`, `<time datetime="2026-11-19T12:00:00Z">`,
				`href="/n/town/admin/adjustments/pending?status=pending&after=next-id">Next page</a>`)
			for _, action := range []struct {
				name, label string
				want        bool
			}{
				{"approve", "Review approval for " + unsafe, tc.approve},
				{"reject", "Reject adjustment for " + unsafe, tc.approve},
				{"cancel", "Cancel request for " + unsafe, tc.cancel},
			} {
				has := strings.Contains(body, `action="/n/town/admin/adjustments/request-id/`+action.name+`"`)
				if has != action.want {
					t.Errorf("%s action visibility = %v, want %v", action.name, has, action.want)
				}
				if has {
					wantAdminAmountText(t, body, `aria-label="`+action.label+`"`, `name="csrf" value="csrf-value"`)
				}
			}
			if strings.Contains(body, "Another administrator must decide this.") != tc.other {
				t.Error("independent-administrator notice changed its eligibility gate")
			}
			if tc.expired {
				wantAdminAmountText(t, body, "pending: expired, nothing will be posted")
			} else if tc.status == "pending" {
				wantAdminAmountText(t, body, `pending; expires <time datetime="2026-11-19T13:00:00Z">`)
			} else {
				wantAdminAmountText(t, body, "approved by "+unsafe)
			}
			if strings.Contains(body, "<img src=x") || strings.Contains(body, "&lt;time") {
				t.Fatal("pending request content crossed the escaping boundary or timestamps lost their markup")
			}
		})
	}
	body := renderAdminAmountPage(t, "admin_adjust_pending", &page{Slug: "town", D: &pendingData{}})
	wantAdminAmountText(t, body, "Dual control is off for this network, so adjustments post at once. Requests made earlier are listed here.",
		"No requests match this filter.")
	// User-controlled names and unknown status strings never enter msgAround.
	odd := "\x00%! " + unsafe
	for _, status := range []string{"pending", odd} {
		body = renderAdminAmountPage(t, "admin_adjust_pending", &page{Slug: "town", Dec: 2, Unit: "credit",
			D: &pendingData{Rows: []pendingRow{{Status: status, Target: odd, Proposer: odd, Decider: odd, Expires: at, CanApprove: true}}}})
		if !strings.Contains(body, "%!") || strings.Contains(body, "<img src=x") {
			t.Fatal("untrusted status/name was interpreted as catalogue formatting or markup")
		}
	}
}

// FR-066, UX-022: reversal review keeps both projected balances, required
// rationale, the final-action notice, and complete full/partial consequences.
func TestAdminReversalFormAndReviewTemplates(t *testing.T) {
	const unsafe = `<img src=x onerror="alert(1)"> & Zoë 100%`
	trade := reverseTrade{ID: "trade-id", Payer: unsafe, Payee: "Ben", Amount: "90071992547409.93", AmountMinor: 9007199254740993,
		Reversed: "0.00", Remaining: "90071992547409.93", RemainingMinor: 9007199254740993}
	p := &page{Slug: "town", Dec: 2, Unit: "Hours", CSRF: "csrf-value", Idem: "idem-value",
		V: map[string]string{"tx": "trade-id", "amount": "1.00", "reason": unsafe}, D: &reverseForm{Trade: &trade}}
	body := renderAdminAmountPage(t, "admin_reverse", p)
	wantAdminAmountText(t, body, "It is posted as a new entry, the original trade stays on record, and every reversal needs a reason.",
		"You cannot reverse a trade you took part in. Disputed trades are decided from the review queue.",
		"<dt>Payer</dt><dd>"+unsafe+"</dd>", "<dt>Payee</dt><dd>Ben</dd>", "<dt>Amount</dt><dd>90071992547409.93 Hours</dd>",
		"<dt>Already reversed</dt><dd>0.00 Hours</dd>", "<dt>Left to reverse</dt><dd>90071992547409.93 Hours</dd>",
		`<form method="post" action="/n/town/admin/reverse" novalidate>`, `name="csrf" value="csrf-value"`, `name="idem" value="idem-value"`,
		`name="step" value="review"`, `for="f-tx">Trade reference</label>`, "The reference of the completed trade, as in the address of its page.",
		`name="tx" type="text" value="trade-id" aria-describedby="h-tx" required`, `for="f-amount">Amount to reverse</label>`,
		"Optional. Up to 2 decimal places. Leave empty to reverse everything that is left.", `inputmode="decimal" aria-describedby="h-amount">`,
		`for="f-reason">Reason</label>`, "Required. Recorded in the audit log and shown on the reversal.",
		`maxlength="500" aria-describedby="h-reason" required`, ">Review reversal</button>")
	if strings.Contains(body, "<img src=x") || !strings.Contains(body, "&lt;img src=x") {
		t.Fatal("reversal form content crossed the escaping boundary")
	}
	for _, full := range []bool{false, true} {
		amount := int64(100)
		if full {
			amount = trade.RemainingMinor
		}
		rv := &reverseReview{reverseTrade: trade, Reverse: domain.FormatAmount(amount, 2), Unit: "Hours", Reason: unsafe, Full: full,
			PayerBefore: makeBalance(-trade.AmountMinor, 2, "Hours"), PayerAfter: makeBalance(-trade.AmountMinor+amount, 2, "Hours"),
			PayeeBefore: makeBalance(trade.AmountMinor, 2, "Hours"), PayeeAfter: makeBalance(trade.AmountMinor-amount, 2, "Hours")}
		p.D = &reverseForm{Trade: &trade, Review: rv}
		p.V["amount"] = rv.Reverse
		body = renderAdminAmountPage(t, "admin_reverse_review", p)
		wantAdminAmountText(t, body, "Nothing is posted until you confirm. A posted reversal cannot be undone.",
			"<dt>Trade</dt><dd>"+unsafe+" paid Ben 90071992547409.93 Hours</dd>", rv.Reverse+" Hours go back from Ben to "+unsafe,
			"<dt>Reason</dt><dd>"+unsafe+"</dd>", "<dt>"+unsafe+" now</dt>", "<dt>"+unsafe+" after</dt>",
			"<dt>Ben now</dt>", "<dt>Ben after</dt>", rv.PayerBefore.Signed, rv.PayerAfter.Signed, rv.PayeeBefore.Signed, rv.PayeeAfter.Signed,
			`<form method="post" action="/n/town/admin/reverse">`, `name="csrf" value="csrf-value"`, `name="idem" value="idem-value"`,
			`name="tx" value="trade-id"`, `name="amount" value="`+rv.Reverse+`"`, `name="reason" value="`+unsafe+`"`,
			`name="step" value="confirm" class="danger">Post reversal</button>`, `name="step" value="back" class="secondary">Go back and edit</button>`)
		if full {
			wantAdminAmountText(t, body, "This reverses everything that is left, so the trade becomes Reversed.")
			if strings.Contains(body, "This is a partial reversal") {
				t.Fatal("full reversal gained the partial-reversal consequence")
			}
		} else {
			wantAdminAmountText(t, body, "This is a partial reversal; the trade stays Completed.")
			if strings.Contains(body, "This reverses everything that is left") {
				t.Fatal("partial reversal gained the full-reversal consequence")
			}
		}
		if strings.Contains(body, "<img src=x") {
			t.Fatal("reversal review content crossed the escaping boundary")
		}
	}
}

func TestAdminAdjustmentCatalogueStatusFallback(t *testing.T) {
	for _, status := range []string{"pending", "approved", "rejected", "cancelled", "expired", "", "future\x00%! <img src=x>"} {
		if got := (pendingRow{Status: status}).StatusText(); got != status {
			t.Errorf("stored status %q rendered as %q", status, got)
		}
	}
}
