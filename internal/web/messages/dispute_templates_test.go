package messages

import (
	"bytes"
	"fmt"
	"html"
	"html/template"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// These fixtures render the actual dispute and base partial templates without
// a database. Handler/service authorization and posting tests remain separate.
type disputeTemplatePage struct {
	D                any
	Slug, Unit, CSRF string
	Idem             string
	Dec              int
	V, Errors        map[string]string
}

func (p disputeTemplatePage) Val(key string) string { return p.V[key] }
func (p disputeTemplatePage) Err(key string) string { return p.Errors[key] }

func renderDisputeTemplate(t *testing.T, name string, p disputeTemplatePage) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("locate actual templates")
	}
	dir := filepath.Join(filepath.Dir(file), "..", "templates")
	funcs := template.FuncMap{
		"msg": Text, "msgAround": Around,
		"dict": func(pairs ...any) (map[string]any, error) {
			if len(pairs)%2 != 0 {
				return nil, fmt.Errorf("odd dictionary arguments")
			}
			m := make(map[string]any)
			for i := 0; i < len(pairs); i += 2 {
				m[pairs[i].(string)] = pairs[i+1]
			}
			return m, nil
		},
		"eq2":  func(a, b any) bool { return fmt.Sprint(a) == fmt.Sprint(b) },
		"time": func(v any) any { return v },
		"categoryLabel": func(v string) string {
			if v == "quality" {
				return "Not as described or poor quality"
			}
			return v
		},
	}
	tmpl, err := template.New(name).Funcs(funcs).ParseFiles(filepath.Join(dir, "base.html"), filepath.Join(dir, name+".html"))
	if err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	if err := tmpl.ExecuteTemplate(&out, "content", p); err != nil {
		t.Fatal(err)
	}
	return out.String()
}

func requireDisputeText(t *testing.T, body string, wants ...string) {
	t.Helper()
	for _, want := range wants {
		if !strings.Contains(html.UnescapeString(body), want) {
			t.Errorf("missing %q in dispute page", want)
		}
	}
}

// UX-047: exact large amounts, singular/default and custom unit names, and
// hostile names/statements remain plain text in the real detail template.
func TestDisputeDetailCatalogueUnitsAndEscaping(t *testing.T) {
	const unsafe = `Zoë 100% $& "><img src=x>`
	for _, tc := range []struct{ amount, networkUnit, unit string }{
		{"1", "credit", "credit"},
		{"90071992547409.93", "Hours", "Hours"},
	} {
		t.Run(tc.unit, func(t *testing.T) {
			d := map[string]any{
				"V": map[string]any{"Tx": map[string]any{"ID": "trade-123"}, "PayerName": unsafe, "PayeeName": "Ana",
					"Dispute": map[string]any{"ID": "dispute-123", "Statement": unsafe, "Rationale": unsafe}},
				"Status": "Under review", "Amount": tc.amount, "Unit": tc.unit, "Opener": unsafe,
				"Category": unsafe, "ResponseBy": unsafe, "Response": unsafe,
				"OpenedAt": template.HTML(`<time datetime="2026-11-19T12:00:00Z">19 Nov 2026, 12:00 UTC</time>`),
				"Events":   []any{},
			}
			body := renderDisputeTemplate(t, "dispute_view", disputeTemplatePage{D: d, Slug: "town", Unit: tc.networkUnit})
			requireDisputeText(t, body, unsafe+" paid Ana "+tc.amount+" "+tc.unit+"</a>",
				`href="/n/town/trades/trade-123"`, `datetime="2026-11-19T12:00:00Z"`, "Response from "+unsafe)
			if strings.Contains(body, "<img") || !strings.Contains(body, template.HTMLEscapeString(unsafe)) {
				t.Fatal("dispute data crossed the contextual escaping boundary")
			}
		})
	}
}

// UX-047, FR-069: the actual opening/review forms retain literal IDs, form
// guards, submitted statement, the deadline time element and error association.
func TestDisputeOpeningCatalogueFormControls(t *testing.T) {
	const unsafe = `Statement 100% "><img src=x>`
	deadline := template.HTML(`<time datetime="2026-11-20T12:00:00Z">20 Nov 2026, 12:00 UTC</time>`)
	d := map[string]any{"TxID": "trade-123", "Row": map[string]any{"Text": unsafe, "Status": "Completed"}, "Deadline": deadline,
		"Cats": []any{map[string]any{"Value": "quality", "Label": "Not as described or poor quality"}}, "Confirm": false}
	p := disputeTemplatePage{D: d, Slug: "town", CSRF: "csrf-value", Idem: "idem-value",
		V: map[string]string{"category": "quality", "statement": unsafe}, Errors: map[string]string{"statement": "Statement is required."}}
	body := renderDisputeTemplate(t, "dispute_form", p)
	requireDisputeText(t, body, `action="/n/town/trades/trade-123/dispute" novalidate`, `name="csrf" value="csrf-value"`,
		`name="idem" value="idem-value"`, `name="step" value="review"`, `<label for="f-statement">What went wrong?</label>`,
		`rows="4" aria-invalid="true" aria-describedby="h-statement e-statement" required`,
		`id="h-statement">Up to 2000 characters. The other party and the reviewer will read this.</p>`,
		`<option value="quality" selected>Not as described or poor quality</option>`, string(deadline), unsafe+"</textarea>")
	if strings.Contains(body, "<img") {
		t.Fatal("opening form rendered statement as markup")
	}
	d["Confirm"] = true
	body = renderDisputeTemplate(t, "dispute_form", p)
	requireDisputeText(t, body, "Not as described or poor quality", `name="category" value="quality"`,
		`name="statement" value="`+unsafe+`"`, `name="step" value="confirm">Open dispute</button>`,
		`name="step" value="back" class="secondary">Go back and edit</button>`)
	if strings.Contains(body, "<img") {
		t.Fatal("review form rendered statement as markup")
	}
}

func TestDisputeCatalogueListAndQueueControls(t *testing.T) {
	const unsafe = `Unknown 100% "><img src=x>`
	stamp := template.HTML(`<time datetime="2026-11-19T12:00:00Z">19 Nov 2026, 12:00 UTC</time>`)
	for _, name := range []string{"disputes", "review_queue"} {
		d := map[string]any{"Rows": []any{map[string]any{"ID": "dispute-id", "Category": unsafe, "Status": "Under review", "Opened": stamp}},
			"Next": "cursor-123", "Filter": "under_review", "Filters": []any{map[string]any{"Value": "under_review", "Label": "Under review"}}}
		body := renderDisputeTemplate(t, name, disputeTemplatePage{D: d, Slug: "town"})
		requireDisputeText(t, body, `href="/n/town/disputes/dispute-id"`, "Dispute: "+unsafe, string(stamp), "cursor-123")
		if name == "review_queue" {
			requireDisputeText(t, body, `<label for="f-status">Show</label>`, `<option value="under_review" selected>Under review</option>`,
				`href="/n/town/review?status=under_review&after=cursor-123"`, "You cannot decide a dispute on a trade you took part in.")
		}
		if strings.Contains(body, "<img") {
			t.Fatal("dispute list enum fallback rendered as markup")
		}
		d["Rows"] = nil
		body = renderDisputeTemplate(t, name, disputeTemplatePage{D: d, Slug: "town"})
		if name == "disputes" {
			requireDisputeText(t, body, `href="/n/town/trades">trades</a>`, "and choose Raise a dispute.")
		} else {
			requireDisputeText(t, body, "The queue is empty. Disputes that need a decision appear here.")
		}
	}
}

func TestDisputeDetailCatalogueConditionalControls(t *testing.T) {
	const unsafe = `Reason 100% "><img src=x>`
	stamp := template.HTML(`<time datetime="2026-11-20T12:00:00Z">20 Nov 2026, 12:00 UTC</time>`)
	for _, tc := range []struct {
		flag, action string
		want         []string
	}{
		{"CanRespond", "respond", []string{`<label for="f-response">Your response</label>`, `maxlength="2000"`, ">Send response</button>"}},
		{"CanReReview", "rereview", []string{`<label for="f-reason">Why should it be looked at again?</label>`, `maxlength="500"`, string(stamp), ">Request re-review</button>"}},
		{"Decide", "resolve", []string{`<label for="f-outcome">Outcome</label>`, `<option value="partially_upheld" selected>Partly upheld: reverse part</option>`, `name="waive_response" value="1" checked`, `name="waive_reason"`, ">Review decision</button>"}},
		{"DecideReRev", "rereview-decide", []string{`<option value="confirmed">Confirmed: the original decision stands</option>`, `name="step" value="review">Review re-review decision</button>`}},
	} {
		t.Run(tc.flag, func(t *testing.T) {
			d := map[string]any{"V": map[string]any{"Tx": map[string]any{"ID": "trade-123"}, "PayerName": "Ana", "PayeeName": "Ben",
				"Dispute": map[string]any{"ID": "dispute-123", "Statement": "Existing statement"}},
				"Status": "Under review", "Amount": "1", "Unit": "credit", "ReReviewUntil": stamp, tc.flag: true}
			p := disputeTemplatePage{D: d, Slug: "town", Unit: "credit", Dec: 2, CSRF: "csrf-value", Idem: "idem-value",
				V: map[string]string{"reason": unsafe, "response": unsafe, "rationale": unsafe, "waive_reason": unsafe,
					"outcome": "partially_upheld", "amount": "90071992547409.93", "waive_response": "1"}}
			body := renderDisputeTemplate(t, "dispute_view", p)
			requireDisputeText(t, body, `action="/n/town/disputes/dispute-123/`+tc.action+`" novalidate`,
				`name="csrf" value="csrf-value"`, `name="idem" value="idem-value"`)
			requireDisputeText(t, body, tc.want...)
			if tc.flag == "Decide" || tc.flag == "DecideReRev" {
				requireDisputeText(t, body, `value="90071992547409.93" inputmode="decimal"`, "Amount in credits, up to 2 decimal places.",
					`<textarea id="f-rationale" name="rationale" rows="4" aria-describedby="h-rationale" required`,
					`id="h-rationale">Required. Both parties will see it.</p>`, unsafe+"</textarea>")
			}
			if strings.Contains(body, "<img") || !strings.Contains(body, template.HTMLEscapeString(unsafe)) {
				t.Fatal("retained form text crossed the escaping boundary")
			}
		})
	}
}
