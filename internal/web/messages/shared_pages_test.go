package messages

import (
	"bytes"
	"go/ast"
	"go/parser"
	"go/token"
	"html"
	"html/template"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"
)

// UX-047, UX-061: identifiers and money stay literal strings; arbitrary data
// never becomes a message key, format string or template-owned element.
func TestSharedPagesCatalogueMessages(t *testing.T) {
	const unsafe = `Zoë 100% %! "><img src=x>` + "\x00"
	for _, tc := range []struct {
		key  string
		args []any
		want string
	}{
		{"error.fallback.internal", nil, "internal error"},
		{"error.fallback.reference", []any{unsafe}, "internal error, reference " + unsafe},
		{"error.reference", []any{unsafe}, "Reference: " + unsafe},
		{"error.not_found.body", nil, "We could not find that page. It may not exist, or you may not have access to it."},
		{"error.internal.body", nil, "The server could not complete the request. Nothing is shown here about the cause. If you contact the administrator, quote the reference below."},
		{"notice.terms_changed", nil, "The network's rules and terms changed. Please read and accept them before you trade."},
		{"notice.membership_pending.body", []any{unsafe}, "Your request to join " + unsafe + " is waiting for an administrator to decide. You cannot trade until it is approved. Check back later."},
		{"notice.credit_limit_used", []any{"90071992547409.93", "90071992547409.94", "90"}, "You are using 90071992547409.93 of your 90071992547409.94 credit limit (90 percent or more). A trade that would take you past the limit is refused."},
		{"histbal.hint", []any{"2026-03-01T12:00:00Z"}, "Use a full date and time with Z or a numeric offset, for example 2026-03-01T12:00:00Z."},
		{"histbal.field_error", []any{unsafe}, "Date and time: " + unsafe},
		{"histbal.reason.missing", nil, "enter a date and time"},
		{"histbal.reason.format", []any{"2026-03-01T12:00:00Z"}, "use a full RFC 3339 date-time with Z or a numeric offset, for example 2026-03-01T12:00:00Z"},
		{"histbal.reason.precision", nil, "use at most millisecond precision (3 fractional digits)"},
		{"histbal.reason.future", nil, "choose an instant that is not in the future"},
	} {
		t.Run(tc.key, func(t *testing.T) {
			if _, ok := keys[tc.key]; !ok {
				t.Fatalf("shared-page sentence is not in the catalogue: %s", tc.key)
			}
			if got := Text(tc.key, tc.args...); got != tc.want {
				t.Fatalf("got %q, want %q", got, tc.want)
			}
		})
	}
}

func sharedPagePath(t *testing.T, path string) string {
	t.Helper()
	_, source, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("locate actual shared-page source")
	}
	return filepath.Join(filepath.Dir(source), "..", path)
}

// The fragment is embedded in production Go source. Read that exact constant,
// rather than maintaining a second template in this lightweight test package.
func sharedHistBalanceSource(t *testing.T) string {
	t.Helper()
	f, err := parser.ParseFile(token.NewFileSet(), sharedPagePath(t, "histbal_view.go"), nil, 0)
	if err != nil {
		t.Fatal(err)
	}
	for _, decl := range f.Decls {
		g, ok := decl.(*ast.GenDecl)
		if !ok || g.Tok != token.CONST {
			continue
		}
		for _, spec := range g.Specs {
			v := spec.(*ast.ValueSpec)
			for i, name := range v.Names {
				if name.Name != "histBalanceSrc" {
					continue
				}
				literal, ok := v.Values[i].(*ast.BasicLit)
				if !ok {
					t.Fatal("historical-balance source is not a string literal")
				}
				src, err := strconv.Unquote(literal.Value)
				if err != nil {
					t.Fatal(err)
				}
				return src
			}
		}
	}
	t.Fatal("historical-balance template not found")
	return ""
}

func renderSharedPage(t *testing.T, name, entry string, data any, extra string) string {
	t.Helper()
	funcs := template.FuncMap{"msg": Text, "msgAround": Around,
		"dict": func(kv ...any) map[string]any {
			m := map[string]any{}
			for i := 0; i < len(kv); i += 2 {
				m[kv[i].(string)] = kv[i+1]
			}
			return m
		},
		"eq2": func(a, b any) bool { return a == b }, "time": func(v any) any { return v },
	}
	tmpl := template.New(name).Funcs(funcs)
	if name != "histbal-only" {
		for _, file := range []string{"base.html", name + ".html"} {
			src, err := os.ReadFile(sharedPagePath(t, filepath.Join("templates", file)))
			if err != nil {
				t.Fatal(err)
			}
			if _, err := tmpl.Parse(string(src)); err != nil {
				t.Fatal(err)
			}
		}
	}
	if extra != "" {
		if _, err := tmpl.Parse(extra); err != nil {
			t.Fatal(err)
		}
	}
	var out bytes.Buffer
	if err := tmpl.ExecuteTemplate(&out, entry, data); err != nil {
		t.Fatal(err)
	}
	return out.String()
}

func sharedWant(t *testing.T, body string, values ...string) {
	t.Helper()
	for _, value := range values {
		if !strings.Contains(html.UnescapeString(body), value) {
			t.Errorf("missing %q", value)
		}
	}
}

func TestSharedPagesCatalogueHelpAndPendingTemplates(t *testing.T) {
	body := renderSharedPage(t, "help", "content", nil, "")
	sharedWant(t, body, "What credits are", "nobody is obliged to accept them",
		"<strong>In credit</strong> (shown with a plus sign)", "<strong>In debt</strong> (shown with a minus sign)",
		"<strong>Level</strong>: you have given and received the same.", "It is not your balance.",
		`<h2 id="sessions">Review and end your sign-ins</h2>`, `href="/account/sessions"`,
		`<h2 id="support-boundaries">Support boundaries</h2>`, "documented reason and a second staff approver",
		"technical enforcement is incomplete", "can technically read data stored by the deployment")
	for _, claim := range []string{"cannot technically access", "cryptographically prevented", "guaranteed response time"} {
		if strings.Contains(strings.ToLower(body), claim) {
			t.Fatalf("help acquired an unsupported claim: %s", claim)
		}
	}
	const unsafe = `Town 100% %! "><img src=x>`
	body = renderSharedPage(t, "pending", "content", map[string]any{"NetName": unsafe}, "")
	sharedWant(t, body, "Your request to join "+unsafe+" is waiting for an administrator to decide.", "You cannot trade until it is approved.")
	if strings.Contains(body, "<img") || !strings.Contains(body, template.HTMLEscapeString(unsafe)) {
		t.Fatal("pending notice interpreted the network name as markup")
	}
}

func TestSharedPagesCatalogueErrorAndFrozenTemplates(t *testing.T) {
	const unsafe = `Reference 100% %! "><img src=x>`
	const frozen = `<time datetime="2026-03-01T12:15:00Z">1 Mar 2026, 17:45 IST</time>`
	for _, status := range []int{403, 404, 409, 422, 423, 429, 500, 503} {
		p := map[string]any{"Status": status, "ReqID": unsafe, "Text": unsafe}
		body := renderSharedPage(t, "error", "content", p, "")
		sharedWant(t, body, `href="/">Return home</a>`, unsafe)
		if strings.Contains(body, `href="/login"`) != (status == 403) {
			t.Fatalf("status %d changed sign-in action visibility", status)
		}
		if strings.Contains(body, "<code>") != (status == 500) {
			t.Fatalf("status %d changed error reference visibility", status)
		}
		if strings.Contains(body, "<img") || !strings.Contains(body, template.HTMLEscapeString(unsafe)) {
			t.Fatal("error page data became markup")
		}
	}
	for _, isFrozen := range []bool{false, true} {
		p := map[string]any{"Title": "Home", "Notice": unsafe}
		if isFrozen {
			p["FrozenSince"] = template.HTML(frozen)
		}
		body := renderSharedPage(t, "pending", "base", p, "")
		if isFrozen {
			sharedWant(t, body, "since "+frozen+".", "anything that would change them is refused until the check passes.")
			if strings.Contains(body, "&lt;time") || strings.Contains(body, "Reference 100%") {
				t.Fatal("frozen notice lost its time element or exposed the replaced notice")
			}
		} else if !strings.Contains(body, template.HTMLEscapeString(unsafe)) || strings.Contains(body, "<img") {
			t.Fatal("ordinary notice lost escaping")
		}
	}
}

// Exercise the actual embedded fragment alone and inside the actual History
// page. Handler/service execution remains a separate web-package gate.
func TestSharedPagesCatalogueHistoricalTemplates(t *testing.T) {
	const unsafe = `Bad 100% %! "><img src=x>`
	const instant = "2026-03-01T12:00:00.000Z"
	const rawInstant = "2026-03-01T14:00:00+02:00"
	for _, page := range []bool{false, true} {
		for _, state := range []string{"bare", "invalid", "posted", "no-postings"} {
			vm := map[string]any{"Raw": "", "FieldError": "", "Hint": "Use a full date and time with Z or a numeric offset, for example 2026-03-01T12:00:00Z.",
				"Submitted": false, "NoPostings": false, "AsOf": instant,
				"Explanation": "This is the posted balance of your account at this instant. It counts only postings made up to then.",
				"Balance":     map[string]any{"Signed": "−90071992547409.93", "State": "In debt", "Spoken": "In debt by 90071992547409.93 Hours"}}
			switch state {
			case "invalid":
				vm["Raw"], vm["FieldError"] = unsafe, unsafe
			case "posted":
				vm["Raw"], vm["Submitted"] = rawInstant, true
			case "no-postings":
				vm["Raw"], vm["Submitted"], vm["NoPostings"] = rawInstant, true, true
				vm["Balance"] = map[string]any{"Signed": "0.00", "State": "Level", "Spoken": "Level, zero credits"}
			}
			name, entry, data := "histbal-only", "histbal", any(vm)
			if page {
				name, entry = "history", "content"
				data = map[string]any{"Slug": "exact-network-id", "CSRF": "csrf-exact", "D": map[string]any{"Filter": "completed", "HistBal": vm}}
			}
			body := renderSharedPage(t, name, entry, data, sharedHistBalanceSource(t))
			if page {
				sharedWant(t, body, `action="/n/exact-network-id/history"`, `name="csrf" value="csrf-exact"`)
			}
			start := strings.Index(body, `<section aria-labelledby="histbal-h">`)
			if start < 0 {
				t.Fatal("missing historical-balance section")
			}
			section := body[start:]
			section = section[:strings.Index(section, "</section>")]
			sharedWant(t, section, `<form method="get" action="">`, `<label for="histbal-at">Date and time (with Z or offset)</label>`,
				`id="histbal-at" name="at" type="text" autocomplete="off" spellcheck="false"`, "Show balance")
			if strings.Contains(section, `aria-invalid="true"`) != (state == "invalid") || strings.Contains(section, `id="histbal-result"`) != (state == "posted" || state == "no-postings") {
				t.Fatalf("historical state %s changed result/error visibility", state)
			}
			if state == "invalid" {
				sharedWant(t, section, `id="histbal-err" class="error" role="alert"`, `aria-describedby="histbal-hint histbal-err"`, "Date and time: "+unsafe, `value="`+unsafe+`"`)
				if strings.Contains(section, "<img") {
					t.Fatal("raw timestamp or unknown reason became markup")
				}
			}
			if state == "posted" || state == "no-postings" {
				sharedWant(t, section, `value="`+rawInstant+`"`, `At <time datetime="`+instant+`">`+instant+`</time> (UTC)`, `aria-live="polite"`)
			}
			if state == "posted" {
				sharedWant(t, section, "−90071992547409.93", `aria-label="In debt by 90071992547409.93 Hours"`)
			}
			if strings.Contains(section, "No postings as of this instant") != (state == "no-postings") {
				t.Fatal("historical no-postings explanation changed")
			}
			for _, currentValue := range []string{"available", "credit limit", "reserved"} {
				if strings.Contains(strings.ToLower(section), currentValue) {
					t.Fatalf("historical result acquired current value %q", currentValue)
				}
			}
		}
	}
}
