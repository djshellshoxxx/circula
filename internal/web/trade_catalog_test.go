package web

import (
	"bytes"
	"html"
	"strings"
	"testing"
	"time"

	"github.com/djshellshoxxx/circula/internal/domain"
)

func renderTradePage(t *testing.T, name string, p *page) string {
	t.Helper()
	var out bytes.Buffer
	if err := mustLoadTemplates()[name].ExecuteTemplate(&out, "base", p); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(out.String(), "%!") {
		t.Fatal("trade catalogue interpolation was corrupted")
	}
	return out.String()
}

func wantTradeText(t *testing.T, body string, wants ...string) {
	t.Helper()
	for _, want := range wants {
		if !strings.Contains(html.UnescapeString(body), want) {
			t.Errorf("trade page lacks %q", want)
		}
	}
}

// UX-047, UX-015, UX-022: both actual templates must preserve fixed-point
// precision, inflect one unit correctly and never double-suffix a unit name.
func TestTradeAmountUnitTemplates(t *testing.T) {
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
		{"zero decimal singular", "credit", 1, 0, "1 credit"},
		{"zero decimal plural", "Hours", 2, 0, "2 Hours"},
		{"HTML-sensitive unit", "<hours>&", 100, 2, "1.00 <hours>&"},
		{"exact large amount", "credit", 9007199254740993, 2, "90071992547409.93 credits"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			tx := &domain.Transaction{ID: "trade-id", Amount: tc.amount, LedgerSeq: 9007199254740993}
			for name, data := range map[string]any{
				"trade_accept": &actData{Tx: tx},
				"trade_detail": &tradeDetail{Tx: tx, Row: txRow{Ledger: "ledger reference 9007199254740993"}},
			} {
				t.Run(name, func(t *testing.T) {
					body := renderTradePage(t, name, &page{Slug: "town", Unit: tc.unit, Dec: tc.dec, D: data})
					wantTradeText(t, body, "<dt>Amount</dt><dd>"+tc.want+"</dd>")
					if name == "trade_detail" {
						wantTradeText(t, body, "<dt>Ledger reference</dt><dd>9007199254740993</dd>")
					}
					if strings.Contains(body, "Hourss") || strings.Contains(body, "<hours>") {
						t.Error("unit was double-suffixed or rendered as markup")
					}
					if tc.unit == "<hours>&" && !strings.Contains(body, "&lt;hours&gt;&amp;</dd>") {
						t.Error("custom unit was not preserved and escaped")
					}
				})
			}
		})
	}
}

// UX-020..022, SEC-016: review retains exact submitted values and recognizable
// member references through both directions without changing form mechanics.
func TestTradeProposalTemplates(t *testing.T) {
	const memberID = "018f0000-0000-7000-8000-00000000000a"
	const unsafe = `<img src=x onerror="alert(1)"> & Zoë 100%`
	for _, dir := range []string{"pay", "request"} {
		p := &page{Slug: "town", Dec: 2, Unit: "Hours", CSRF: "csrf-value", Idem: "idem-value",
			V: map[string]string{"counterparty_id": memberID, "direction": dir, "amount": "90071992547409.93", "note": unsafe, "listing_id": "listing-id"},
			D: &proposeData{Members: []option{{"", "Choose a member"}, {memberID, unsafe + " (member reference 0000-000A)"}}}}
		body := renderTradePage(t, "propose", p)
		wantTradeText(t, body, `<form method="post" action="/n/town/trades/new" novalidate>`,
			`name="csrf" value="csrf-value"`, `name="idem" value="idem-value"`, `name="step" value="review"`,
			`name="listing_id" value="listing-id"`, `for="f-counterparty_id">Who is the trade with?</label>`,
			`value="`+memberID+`" selected>`+unsafe+" (member reference 0000-000A)",
			"<legend>Direction</legend>", " I pay them</label>", " They pay me</label>",
			`value="`+dir+`" checked`, `for="f-amount">Amount</label>`, "Amount in credits, up to 2 decimal places.",
			`inputmode="decimal" aria-describedby="h-amount" required`, "What is it for? (optional)",
			`maxlength="500"`, ">Review trade</button>", `href="/n/town">Cancel</a>`)
		if strings.Contains(body, "<img src=x") || !strings.Contains(body, "&lt;img src=x") {
			t.Fatal("proposal options or retained form values crossed the escaping boundary")
		}

		reserve := dir == "pay"
		rv := &review{Sentence: "You pay " + unsafe + " 90071992547409.93 Hours", Counter: unsafe,
			Amount: "90071992547409.93", Unit: "Hours", Before: makeBalance(0, 2, "Hours"),
			After: makeBalance(-9007199254740993, 2, "Hours"), AvailNow: "12.50", AvailAfter: "−90071992547397.43",
			Reserve: reserve, TooMuch: reserve, Expires: 1}
		if !reserve {
			rv.Sentence = unsafe + " pays you 90071992547409.93 Hours"
			rv.After = makeBalance(9007199254740993, 2, "Hours")
			rv.AvailAfter = "90071992547422.43"
		}
		p.D = rv
		body = renderTradePage(t, "propose_review", p)
		wantTradeText(t, body, rv.Sentence, "<dt>With</dt><dd>"+unsafe+" (member reference 0000-000A)</dd>",
			"<dt>Amount</dt><dd>90071992547409.93 Hours</dd>", "Your balance now", "Your balance if accepted",
			"Available to spend now", "Available to spend after sending", "after 1 day if not answered",
			"Note: "+unsafe, "Nothing is posted until the other member accepts.",
			`<form method="post" action="/n/town/trades/new">`, `name="csrf" value="csrf-value"`, `name="idem" value="idem-value"`,
			`name="counterparty_id" value="`+memberID+`"`, `name="direction" value="`+dir+`"`,
			`name="amount" value="90071992547409.93"`, `name="note" value="`+unsafe+`"`, `name="listing_id" value="listing-id"`,
			`name="step" value="confirm">Send trade</button>`, `name="step" value="back" class="secondary">Go back and edit</button>`)
		if reserve {
			wantTradeText(t, body, "This is more than your available to spend (12.50). It will be refused if you send it. Go back and enter a smaller amount.",
				"−90071992547397.43 (the amount is set aside while the trade is pending)")
		} else {
			wantTradeText(t, body, "12.50 (it rises only once the trade is accepted)")
			if strings.Contains(body, "This is more than") || strings.Contains(body, "set aside while the trade") {
				t.Fatal("receive-side review gained the outgoing reservation warning")
			}
		}
		if strings.Contains(body, "<img src=x") || !strings.Contains(body, "&lt;img src=x") {
			t.Fatal("review text or hidden values crossed the escaping boundary")
		}
	}
}

func TestTradeActionTemplates(t *testing.T) {
	const unsafe = `<img src=x onerror="alert(1)"> & Zoë 100%`
	tx := &domain.Transaction{ID: "trade-id", PayerMemberID: "018f0000-0000-7000-8000-00000000000a",
		PayeeMemberID: "018f0000-0000-7000-8000-00000000000b", Amount: 100, Note: unsafe}
	for _, tc := range []struct {
		action, verb, tmpl string
		askReason          bool
	}{
		{"accept", "Accept", "trade_accept", false},
		{"decline", "Decline", "trade_confirm", false},
		{"cancel", "Cancel", "trade_confirm", false},
		{"cancel", "Cancel", "trade_confirm", true},
	} {
		t.Run(tc.action+"/"+tc.tmpl+"/"+map[bool]string{true: "administrator", false: "party"}[tc.askReason], func(t *testing.T) {
			a := &actData{Action: tc.action, Verb: tc.verb, Tx: tx, Payer: unsafe, Payee: "Ben", Sentence: unsafe + " pays Ben 1.00 credit",
				Before: makeBalance(0, 2, "credit"), After: makeBalance(100, 2, "credit"), AvailNow: "0.00", AvailAfter: "1.00",
				AskReason: tc.askReason, Consequence: "Nothing is posted and the amount set aside for this trade is released."}
			body := renderTradePage(t, tc.tmpl, &page{Slug: "town", Dec: 2, Unit: "credit", CSRF: "csrf-value", Idem: "idem-value",
				V: map[string]string{"reason": unsafe}, D: a})
			wantTradeText(t, body, "<dt>Payer</dt><dd>"+unsafe+" (member reference 0000-000A)</dd>",
				"<dt>Payee</dt><dd>Ben (member reference 0000-000B)</dd>",
				`<form method="post" action="/n/town/trades/trade-id/`+tc.action+`">`,
				`name="csrf" value="csrf-value"`, `name="idem" value="idem-value"`, `href="/n/town/trades/trade-id">Go back</a>`)
			if tc.action == "accept" {
				wantTradeText(t, body, "<dt>Amount</dt><dd>1.00 credit</dd>", "<dt>Note</dt><dd>"+unsafe+"</dd>",
					"Your balance after posting", "Available to spend after posting",
					"Accepting posts this trade to both accounts straight away.", ">Confirm and accept</button>")
			} else {
				wantTradeText(t, body, a.Consequence, `class="danger">`+tc.verb+" this trade</button>")
			}
			if tc.askReason {
				wantTradeText(t, body, `for="f-reason">Reason</label>`, "Required when you cancel a trade you are not part of.",
					`name="reason" type="text" value="`+unsafe+`" aria-describedby="h-reason" required`)
			} else if strings.Contains(body, `name="reason"`) {
				t.Fatal("party confirmation gained an administrator-only reason field")
			}
			if strings.Contains(body, "<img src=x") || !strings.Contains(body, "&lt;img src=x") {
				t.Fatal("action confirmation crossed the escaping boundary")
			}
		})
	}
	// The system account has no member reference; no empty marker is introduced.
	tx.PayerMemberID = ""
	body := renderTradePage(t, "trade_confirm", &page{Slug: "town", D: &actData{Tx: tx, Payer: "Network", Payee: "Ben", Action: "cancel", Verb: "Cancel"}})
	wantTradeText(t, body, "<dt>Payer</dt><dd>Network</dd>")
}

func TestTradeDetailTemplateActionsAndTimeline(t *testing.T) {
	const unsafe = `<img src=x onerror="alert(1)"> & Zoë 100%`
	at := time.Date(2026, time.November, 19, 12, 34, 56, 0, time.UTC)
	tx := &domain.Transaction{ID: "trade-id", Amount: 100, Note: unsafe, Reason: unsafe,
		Status: domain.TxCompleted, LedgerSeq: 9007199254740993, CompletedAt: &at,
		OriginalTxID: "original-id", RelatedTxID: "related-id"}
	d := &tradeDetail{Tx: tx, Party: true, Payer: unsafe, Payee: "Ben", Proposed: fmtTime(at), Posted: fmtTime(at),
		Explain: "This trade is posted.", Row: txRow{Text: "You paid " + unsafe + " 1.00 credit", Status: "Completed",
			Ledger: "ledger reference 9007199254740993", ReversedBy: []txLink{{ID: "reversal-id", Text: "Reversed by " + unsafe}}},
		History: []histRow{{Label: "Proposed", At: fmtTime(at)}}}
	body := renderTradePage(t, "trade_detail", &page{Slug: "town", Dec: 2, Unit: "credit", D: d})
	wantTradeText(t, body, "You paid "+unsafe+" 1.00 credit", "<dt>Status</dt><dd><span class=\"tag\">Completed</span>",
		"<dt>Payer</dt><dd>"+unsafe+"</dd>", "<dt>Payee</dt><dd>Ben</dd>",
		"<dt>Note</dt><dd>"+unsafe+"</dd>", "<dt>Reason</dt><dd>"+unsafe+"</dd>", "<dt>Posted</dt>",
		"<dt>Ledger reference</dt><dd>9007199254740993</dd>",
		`<dt>Reverses</dt><dd><a href="/n/town/trades/original-id">The original entry</a></dd>`,
		`<dt>Adjusts</dt><dd><a href="/n/town/trades/related-id">The original entry</a></dd>`,
		`<dt>Reversals</dt>`, `href="/n/town/trades/reversal-id">Reversed by `+unsafe,
		`<h2>Timeline</h2>`, `<li>Proposed: <time datetime="2026-11-19T12:34:56Z">`, `href="/n/town/trades">Back to trades</a>`)
	if strings.Contains(body, "<img src=x") || strings.Contains(body, "&lt;time") {
		t.Fatal("detail content crossed the escaping boundary or the timeline lost its time element")
	}
	actions := []struct {
		label, href string
		enable      func(*tradeDetail)
	}{
		{"Review and accept", "/n/town/trades/trade-id/accept", func(v *tradeDetail) { v.Accept = true }},
		{"Decline", "/n/town/trades/trade-id/decline", func(v *tradeDetail) { v.Decline = true }},
		{"Cancel this proposal", "/n/town/trades/trade-id/cancel", func(v *tradeDetail) { v.Cancel = true }},
		{"Raise a dispute", "/n/town/trades/trade-id/dispute", func(v *tradeDetail) { v.Dispute = true }},
		{"View the dispute", "/n/town/disputes/dispute-id", func(v *tradeDetail) { v.DisputeID = "dispute-id" }},
		{"Reverse this trade (administrator)", "/n/town/admin/reverse?tx=trade-id", func(v *tradeDetail) { v.Reverse = true }},
	}
	for i, action := range actions {
		t.Run(action.label, func(t *testing.T) {
			v := &tradeDetail{Tx: tx, WhyNot: unsafe}
			action.enable(v)
			body := renderTradePage(t, "trade_detail", &page{Slug: "town", Dec: 2, Unit: "credit", D: v})
			wantTradeText(t, body, `href="`+action.href+`">`+action.label+"</a>", "No status changes recorded yet.")
			for j, other := range actions {
				if i != j && strings.Contains(body, `href="`+other.href+`"`) {
					t.Errorf("%s also exposed %s", action.label, other.label)
				}
			}
			if strings.Contains(body, "<img src=x") {
				t.Fatal("unavailable-action explanation crossed the escaping boundary")
			}
		})
	}
	tx.Status, tx.ExpiresAt, tx.CompletedAt = domain.TxProposed, &at, nil
	d = &tradeDetail{Tx: tx, Expires: fmtTime(at)}
	body = renderTradePage(t, "trade_detail", &page{Slug: "town", Dec: 2, Unit: "credit", D: d})
	wantTradeText(t, body, `<dt>Expires</dt><dd><time datetime="2026-11-19T12:34:56Z">`)
	if strings.Contains(body, "<dt>Posted</dt>") {
		t.Fatal("pending trade gained a posting date")
	}
}

func TestTradeListAndHistoryTemplates(t *testing.T) {
	const unsafe = `<img src=x onerror="alert(1)"> & Zoë 100%`
	at := time.Date(2026, time.November, 19, 12, 34, 56, 0, time.UTC)
	row := txRow{ID: "trade-id", Text: "You paid " + unsafe + " 90071992547409.93 Hours", Signed: "−90071992547409.93",
		Status: "Completed", When: at, Ledger: "ledger reference 9007199254740993", Reason: unsafe,
		Original: "original-id", Related: "related-id", ReversedBy: []txLink{{ID: "reversal-id", Text: unsafe}}}
	for _, populated := range []bool{false, true} {
		trades := &tradesData{Filter: "declined", Next: "next-id"}
		history := &historyData{Filter: "declined", Next: "next-id"}
		if populated {
			trades.Needs, trades.Waiting, trades.Past = []txRow{row}, []txRow{row}, []txRow{row}
			history.Pending, history.Done = []txRow{row}, []txRow{row}
		}
		for name, data := range map[string]any{"trades": trades, "history": history} {
			body := renderTradePage(t, name, &page{Slug: "town", CSRF: "csrf-value", D: data})
			wantTradeText(t, body, `<form method="get" action="/n/town/`+name+`">`, `for="f-status">Show</label>`,
				`value="">All statuses</option>`, `value="declined" selected>Declined</option>`, ">Apply filter</button>",
				`href="/n/town/`+name+`?status=declined&after=next-id">Next page</a>`)
			for _, status := range []string{"Completed", "Cancelled", "Expired", "Disputed", "Reversed"} {
				wantTradeText(t, body, ">"+status+"</option>")
			}
			if name == "trades" {
				wantTradeText(t, body, `href="/n/town/trades/new">Propose a trade</a>`, "Waiting for you", "Past trades")
				if !populated {
					wantTradeText(t, body, "Nothing is waiting for you.", "No trades are waiting for someone else.",
						"No past trades yet. Completed and closed trades appear here.")
				}
			} else {
				wantTradeText(t, body, `value="proposed">Pending</option>`, "Posted and closed",
					`href="/n/town/audit">Account activity log</a>`, "Download your data", `<form method="post" action="/n/town/export">`,
					`name="csrf" value="csrf-value"`, `for="f-format">File format</label>`, `value="csv">CSV (spreadsheet)</option>`,
					`value="json">JSON</option>`, ">Download my transactions</button>")
				if populated {
					wantTradeText(t, body, "Pending trades are not part of your balance.", "Reason: "+unsafe,
						`Reverses <a href="/n/town/trades/original-id">the original entry</a>`,
						`Adjusts <a href="/n/town/trades/related-id">the original entry</a>`,
						`href="/n/town/trades/reversal-id">`+unsafe+"</a>")
				} else {
					wantTradeText(t, body, "No pending trades.", "Nothing here yet. Once you trade, each entry shows who paid whom, the signed amount, its status and ledger reference.")
				}
			}
			if populated {
				wantTradeText(t, body, row.Text, "−90071992547409.93", "ledger reference 9007199254740993",
					`<time datetime="2026-11-19T12:34:56Z">`)
				if strings.Contains(body, "<img src=x") || !strings.Contains(body, "&lt;img src=x") {
					t.Fatal("trade/history row content crossed the escaping boundary")
				}
			}
		}
	}
}

func TestTradeCatalogueStatusAndAmountHelp(t *testing.T) {
	for _, tc := range []struct {
		status domain.TxStatus
		want   string
	}{
		{domain.TxProposed, "Pending"}, {domain.TxAccepted, "Completed"}, {domain.TxCompleted, "Completed"},
		{domain.TxDeclined, "Declined"}, {domain.TxCancelled, "Cancelled"}, {domain.TxExpired, "Expired"},
		{domain.TxDisputed, "Disputed"}, {domain.TxReversed, "Reversed"}, {"future_status", "Future_status"},
	} {
		if got := statusWord(tc.status); got != tc.want {
			t.Errorf("status %s: got %q, want %q", tc.status, got, tc.want)
		}
		if got := historyWord("", tc.status); got != "Proposed" {
			t.Errorf("initial %s history label: %q", tc.status, got)
		}
		if got := historyWord(domain.TxProposed, tc.status); got != tc.want {
			t.Errorf("transition to %s: got %q, want %q", tc.status, got, tc.want)
		}
	}
	for _, tc := range []struct {
		dec  int
		want string
	}{
		{0, "Amount: enter a number with up to 0 decimal places, for example 1250."},
		{1, "Amount: enter a number with up to 1 decimal place, for example 125.0."},
		{4, "Amount: enter a number with up to 4 decimal places, for example 0.1250."},
	} {
		c := &rc{Net: &domain.Network{Policy: domain.Policy{Decimals: tc.dec}}}
		if got := c.amountHelp(); got != tc.want {
			t.Errorf("precision %d: got %q, want %q", tc.dec, got, tc.want)
		}
	}
}
