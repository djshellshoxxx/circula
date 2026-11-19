package governance

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"html/template"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"
)

// WebAuth must authenticate current membership, bind X-Network and verify cookie
// CSRF for every mutation. Service independently rechecks within its transaction.
type WebAuth interface {
	Authorize(*http.Request, bool) (Identity, error)
	CSRF(*http.Request) string
}
type Web struct {
	Service   Service
	Auth      WebAuth
	CursorKey []byte
}

func (w Web) cursor(id Identity, after string) string {
	data := []byte(after)
	mac := hmac.New(sha256.New, w.CursorKey)
	mac.Write([]byte(id.Network + "\x00" + id.Member + "\x00" + after))
	return base64.RawURLEncoding.EncodeToString(data) + "." + base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}
func (w Web) ServeHTTP(out http.ResponseWriter, r *http.Request) {
	out.Header().Set("Cache-Control", "no-store")
	out.Header().Set("X-Content-Type-Options", "nosniff")
	out.Header().Set("Content-Security-Policy", "default-src 'none'; style-src 'unsafe-inline'; form-action 'self'; frame-ancestors 'none'")
	out.Header().Set("Referrer-Policy", "no-referrer")
	problem := func(code int, title string) {
		out.Header().Set("Content-Type", "application/problem+json")
		out.WriteHeader(code)
		json.NewEncoder(out).Encode(map[string]any{"type": "about:blank", "title": title, "status": code})
	}
	if r.URL.IsAbs() || r.URL.Host != "" || !localRequestPath(r.URL.Path) {
		problem(400, "Invalid governance path")
		return
	}
	if w.Auth == nil || len(w.CursorKey) < 32 {
		problem(503, "Governance unavailable")
		return
	}
	if r.Method != "GET" && r.Method != "POST" {
		out.Header().Set("Allow", "GET, POST")
		problem(405, "Method not allowed")
		return
	}
	// Validate query syntax before host hooks can parse it or a command mutates state.
	if len(r.URL.RawQuery) > 8192 {
		problem(400, "Invalid governance query")
		return
	}
	query, queryErr := url.ParseQuery(r.URL.RawQuery)
	if queryErr != nil {
		problem(400, "Invalid governance query")
		return
	}
	for key, values := range query {
		if len(values) != 1 || key != "proposal" && key != "format" && key != "cursor" {
			problem(400, "Invalid governance query")
			return
		}
	}
	if proposal := query.Get("proposal"); proposal != "" && !ident(proposal) {
		problem(400, "Invalid proposal")
		return
	}
	if format := query.Get("format"); format != "" && format != "html" && format != "json" {
		problem(400, "Invalid format")
		return
	}
	if query.Get("proposal") != "" && query.Get("cursor") != "" {
		problem(400, "Ambiguous governance query")
		return
	}
	if r.Method == "POST" {
		r.Body = http.MaxBytesReader(out, r.Body, 65536)
		defer r.Body.Close()
	}
	// Parse and validate before a host CSRF lookup can cache partial form values
	// while discarding ParseForm's one-time error.
	form := r.Method == "POST" && strings.HasPrefix(r.Header.Get("Content-Type"), "application/x-www-form-urlencoded")
	if form {
		if e := r.ParseForm(); e != nil {
			var oversized *http.MaxBytesError
			if errors.As(e, &oversized) {
				problem(413, "Request too large")
				return
			}
			problem(400, "Invalid form")
			return
		}
		for key, values := range r.PostForm {
			if len(values) != 1 || !utf8.ValidString(key) || !utf8.ValidString(values[0]) {
				problem(400, "Ambiguous form")
				return
			}
			switch key {
			case "action", "proposal", "intent", "revision", "title", "body", "category", "comments", "choice", "text", "comment", "dispute", "challenged", "policy_version", "opens", "closes", "csrf":
			default:
				problem(400, "Unknown form field")
				return
			}
		}
	}
	id, e := w.Auth.Authorize(r, r.Method == "POST")
	if e != nil || !ident(id.Network) || !ident(id.Member) || (r.Header.Get("X-Network") != "" && r.Header.Get("X-Network") != id.Network) {
		problem(403, "Access denied")
		return
	}
	after := ""
	if cursor := query.Get("cursor"); cursor != "" {
		if len(cursor) > 512 {
			problem(400, "Invalid cursor")
			return
		}
		parts := strings.Split(cursor, ".")
		if len(parts) != 2 {
			problem(400, "Invalid cursor")
			return
		}
		decoded, err := base64.RawURLEncoding.DecodeString(parts[0])
		if err != nil {
			problem(400, "Invalid cursor")
			return
		}
		after = string(decoded)
		if !ident(after) || !hmac.Equal([]byte(cursor), []byte(w.cursor(id, after))) {
			problem(400, "Invalid cursor")
			return
		}
	}
	// Token providers can revoke or rotate sessions. Read private state only after
	// the callback and a fresh same-identity authorization decision.
	csrfToken := func() (string, bool) {
		token := w.Auth.CSRF(r)
		current, err := w.Auth.Authorize(r, false)
		return token, err == nil && current == id && r.Context().Err() == nil
	}
	if r.Method == "POST" {
		var c Command
		if form {
			c.Action = r.PostForm.Get("action")
			c.Proposal = r.PostForm.Get("proposal")
			c.Intent = r.PostForm.Get("intent")
			if raw := r.PostForm.Get("revision"); raw != "" {
				c.Revision, e = strconv.ParseInt(raw, 10, 64)
				if e != nil || c.Revision < 0 {
					problem(400, "Invalid revision")
					return
				}
			}
			c.Title = r.PostForm.Get("title")
			c.Body = r.PostForm.Get("body")
			c.Category = r.PostForm.Get("category")
			c.Comments = r.PostForm.Get("comments") == "on"
			c.Choice = Choice(r.PostForm.Get("choice"))
			c.Text = r.PostForm.Get("text")
			c.Comment = r.PostForm.Get("comment")
			c.Dispute = r.PostForm.Get("dispute")
			c.Challenged = r.PostForm.Get("challenged")
			c.PolicyVersion = r.PostForm.Get("policy_version")
			if v := r.PostForm.Get("opens"); v != "" {
				c.Opens, e = time.Parse(time.RFC3339, v)
				if e != nil {
					problem(400, "Invalid opening time")
					return
				}
			}
			if v := r.PostForm.Get("closes"); v != "" {
				c.Closes, e = time.Parse(time.RFC3339, v)
				if e != nil {
					problem(400, "Invalid closing time")
					return
				}
			}
		} else {
			raw, err := io.ReadAll(r.Body)
			if err != nil {
				var oversized *http.MaxBytesError
				if errors.As(err, &oversized) {
					problem(413, "Request too large")
				} else {
					problem(400, "Invalid command")
				}
				return
			}
			if e = strictDecode(raw, &c); e != nil {
				problem(400, "Invalid command")
				return
			}

			if c.Intent != r.Header.Get("Idempotency-Key") {
				problem(400, "Intent header mismatch")
				return
			}
		}
		if pid := query.Get("proposal"); pid != "" && pid != c.Proposal {
			problem(400, "Proposal mismatch")
			return
		}
		reply, err := w.Service.Execute(r.Context(), id, c)
		if err != nil {
			code := 403
			if errors.Is(err, ErrConflict) {
				code = 409
			}
			if form {
				token, ok := csrfToken()
				if !ok {
					problem(403, "Access denied")
					return
				}
				detail, readErr := w.Service.Read(r.Context(), id, c.Proposal)
				if readErr != nil && c.Proposal != "" {
					problem(403, "Proposal unavailable")
					return
				}
				intent, _ := randomID()
				out.Header().Set("Content-Type", "text/html; charset=utf-8")
				out.WriteHeader(code)
				Render(out, Page{Detail: detail, CSRF: token, Intent: intent, Member: id.Member, Error: "The action could not be completed. Refresh the proposal and check the current status, permissions and required fields."})
				return
			}
			problem(code, "Governance action unavailable")
			return
		}
		if form {
			http.Redirect(out, r, "?proposal="+reply.Proposal+"&format=html", http.StatusSeeOther)
			return
		}
		out.Header().Set("Content-Type", "application/json")
		json.NewEncoder(out).Encode(reply)
		return
	}
	token := ""
	if query.Get("format") == "html" {
		var ok bool
		token, ok = csrfToken()
		if !ok {
			problem(403, "Access denied")
			return
		}
	}
	if pid := query.Get("proposal"); pid != "" {
		detail, err := w.Service.Read(r.Context(), id, pid)
		if err != nil {
			problem(403, "Proposal unavailable")
			return
		}
		if query.Get("format") == "html" {
			out.Header().Set("Content-Type", "text/html; charset=utf-8")
			intent, err := randomID()
			if err != nil {
				problem(503, "Form unavailable")
				return
			}
			Render(out, Page{Detail: detail, CSRF: token, Intent: intent, Member: id.Member})
			return
		}
		out.Header().Set("Content-Type", "application/json")
		json.NewEncoder(out).Encode(detail)
		return
	}
	proposals, err := w.Service.List(r.Context(), id, after, 21)
	if err != nil {
		problem(403, "Proposals unavailable")
		return
	}
	next := ""
	if len(proposals) > 20 {
		proposals = proposals[:20]
		next = w.cursor(id, proposals[19].ID)
	}
	if query.Get("format") == "html" {
		out.Header().Set("Content-Type", "text/html; charset=utf-8")
		intent, _ := randomID()
		Render(out, Page{Proposals: proposals, CSRF: token, Intent: intent, Member: id.Member, Next: next})
		return
	}
	out.Header().Set("Content-Type", "application/json")
	json.NewEncoder(out).Encode(struct {
		Proposals []Proposal `json:"proposals"`
		Next      string     `json:"next_cursor,omitempty"`
	}{proposals, next})
}

type Page struct {
	Detail                            Detail
	Proposals                         []Proposal
	CSRF, Intent, Member, Next, Error string
}

// Static radios/fieldset/legend/labels adapt the pinned GOV.UK Frontend patterns
// listed in third_party/PROVENANCE.md. User content always uses escaped template.
var pageTemplate = template.Must(template.New("governance").Funcs(template.FuncMap{"actions": func() []string { return []string{"withdraw", "close", "certify", "link"} }}).Parse(`<!doctype html><html lang="en"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1"><title>Community governance</title><style>body{font:1rem system-ui;max-width:65rem;margin:auto;padding:1rem}label{display:block;margin:.5rem 0}input,textarea,button{font:inherit}textarea{width:95%;min-height:6rem}:focus-visible{outline:3px solid #6b4d00;outline-offset:3px}fieldset{margin:1rem 0}button{padding:.5rem}a{color:#164f97}</style></head><body><main><h1>Community governance</h1>{{if .Error}}<div id="error-summary" role="alert" tabindex="-1"><h2>There is a problem</h2><p><a href="#governance-content">{{.Error}}</a></p></div>{{end}}<div id="governance-content"></div><p>Advisory voting only. Administrators apply policy separately through the existing authorized process. Ballots are readable by database operators; this service does not provide anonymous or coercion-resistant voting.</p>{{if .Detail.Proposal.ID}}{{$root := .}}{{$p := .Detail.Proposal}}<h2>{{$p.Title}}</h2><p>{{$p.Body}}</p><dl><dt>Category</dt><dd>{{$p.Category}}</dd><dt>Status</dt><dd>{{$p.State}}</dd><dt>Eligible snapshot count</dt><dd>{{$p.Eligible}}</dd></dl>{{if .Detail.OwnVote}}<p>Your recorded choice: {{.Detail.OwnVote.Choice}}. It cannot be changed.</p>{{end}}{{if .Detail.Result}}<h3>Advisory result</h3><p>Yes {{.Detail.Result.Outcome.Yes}}; no {{.Detail.Result.Outcome.No}}; abstain {{.Detail.Result.Outcome.Abstain}}; eligible {{.Detail.Result.Outcome.Eligible}}. No quorum or binding pass rule is imposed.</p>{{if .Detail.Result.Certified}}<p>Certified by {{.Detail.Result.Certified.Member}}.</p>{{end}}{{if .Detail.Result.PolicyVersion}}<p>Separately applied policy version: {{.Detail.Result.PolicyVersion}}</p>{{end}}{{end}}
{{if eq $p.State "open"}}<form method="post"><input type="hidden" name="csrf" value="{{.CSRF}}"><input type="hidden" name="intent" value="{{.Intent}}"><input type="hidden" name="proposal" value="{{$p.ID}}"><input type="hidden" name="revision" value="{{$p.Revision}}"><input type="hidden" name="action" value="vote"><fieldset aria-describedby="ballot-hint"><legend>Choose your ballot</legend><p id="ballot-hint">Choose one option. There is no default choice. Every ballot is advisory.</p><label for="vote-yes"><input id="vote-yes" name="choice" type="radio" value="yes" required>Yes</label><label for="vote-no"><input id="vote-no" name="choice" type="radio" value="no">No</label><label for="vote-abstain"><input id="vote-abstain" name="choice" type="radio" value="abstain">Abstain</label></fieldset><button type="submit">Record ballot</button></form>{{end}}
{{if and (eq $p.State "draft") (eq $p.Author .Member)}}<form method="post"><input type="hidden" name="csrf" value="{{.CSRF}}"><input type="hidden" name="intent" value="{{.Intent}}-edit"><input type="hidden" name="proposal" value="{{$p.ID}}"><input type="hidden" name="revision" value="{{$p.Revision}}"><input type="hidden" name="action" value="edit"><label for="edit-title">Title</label><input id="edit-title" name="title" value="{{$p.Title}}" maxlength="200" required><label for="edit-category">Category</label><input id="edit-category" name="category" value="{{$p.Category}}" maxlength="64" required><label for="edit-body">Context</label><textarea id="edit-body" name="body" maxlength="12000" required>{{$p.Body}}</textarea><label><input type="checkbox" name="comments" {{if $p.Comments}}checked{{end}}>Allow comments</label><button>Save draft</button></form><form method="post"><input type="hidden" name="csrf" value="{{.CSRF}}"><input type="hidden" name="intent" value="{{.Intent}}-open"><input type="hidden" name="proposal" value="{{$p.ID}}"><input type="hidden" name="revision" value="{{$p.Revision}}"><input type="hidden" name="action" value="open"><label for="opens">Opening time (RFC3339)</label><input id="opens" name="opens" required><label for="closes">Closing time (RFC3339)</label><input id="closes" name="closes" required><button>Freeze electorate and open</button></form>{{end}}
<h3>Comments</h3>{{range .Detail.Comments}}<article><p>{{.Text}}</p><small>By {{.Author}}</small><form method="post"><input type="hidden" name="csrf" value="{{$root.CSRF}}"><input type="hidden" name="intent" value="{{$root.Intent}}-hide-{{.ID}}"><input type="hidden" name="proposal" value="{{$p.ID}}"><input type="hidden" name="revision" value="{{$p.Revision}}"><input type="hidden" name="action" value="hide"><input type="hidden" name="comment" value="{{.ID}}"><label>Moderation/deletion reason<input name="text" maxlength="1000" required></label><button>Hide comment</button></form></article>{{else}}<p>No visible comments.</p>{{end}}{{if $p.Comments}}<form method="post"><input type="hidden" name="csrf" value="{{.CSRF}}"><input type="hidden" name="intent" value="{{.Intent}}-comment"><input type="hidden" name="proposal" value="{{$p.ID}}"><input type="hidden" name="revision" value="{{$p.Revision}}"><input type="hidden" name="action" value="comment"><label for="comment-text">Comment</label><textarea id="comment-text" name="text" maxlength="2048" required></textarea><button>Add comment</button></form>{{end}}
<h3>Process disputes</h3><p>Unresolved process disputes: {{.Detail.OpenDisputes}}</p>{{range .Detail.Disputes}}<article><p>{{.Reason}}</p>{{if .Resolution}}<p>Resolution: {{.Resolution}}</p>{{else}}<form method="post"><input type="hidden" name="csrf" value="{{$root.CSRF}}"><input type="hidden" name="intent" value="{{$root.Intent}}-resolve-{{.ID}}"><input type="hidden" name="proposal" value="{{$p.ID}}"><input type="hidden" name="revision" value="{{$p.Revision}}"><input type="hidden" name="action" value="resolve"><input type="hidden" name="dispute" value="{{.ID}}"><label>Resolution<textarea name="text" maxlength="2000" required></textarea></label><button>Resolve dispute</button></form>{{end}}</article>{{end}}<form method="post"><input type="hidden" name="csrf" value="{{.CSRF}}"><input type="hidden" name="intent" value="{{.Intent}}-dispute"><input type="hidden" name="proposal" value="{{$p.ID}}"><input type="hidden" name="revision" value="{{$p.Revision}}"><input type="hidden" name="action" value="dispute"><label for="challenged">Member whose process action is challenged</label><input id="challenged" name="challenged" maxlength="128" required><label for="dispute-text">Process concern</label><textarea id="dispute-text" name="text" maxlength="2000" required></textarea><button>Record process dispute</button></form>
<details><summary>Lifecycle and administrative actions</summary>{{range $action := actions}}<form method="post"><input type="hidden" name="csrf" value="{{$root.CSRF}}"><input type="hidden" name="intent" value="{{$root.Intent}}-{{$action}}"><input type="hidden" name="proposal" value="{{$p.ID}}"><input type="hidden" name="revision" value="{{$p.Revision}}"><input type="hidden" name="action" value="{{$action}}">{{if eq $action "withdraw"}}<label>Withdrawal reason<textarea name="text" required maxlength="1000"></textarea></label>{{end}}{{if eq $action "link"}}<label>Existing authorized policy version<input name="policy_version" required maxlength="128"></label>{{end}}<button>{{$action}}</button></form>{{end}}</details>
{{else}}<h2>Proposals</h2><ul>{{range .Proposals}}<li><a href="?proposal={{.ID}}&amp;format=html">{{.Title}}</a> — {{.State}}</li>{{else}}<li>No proposals.</li>{{end}}</ul>{{if .Next}}<a href="?cursor={{.Next}}&amp;format=html">Next proposals</a>{{end}}<h2>Create a draft</h2><form method="post"><input type="hidden" name="csrf" value="{{.CSRF}}"><input type="hidden" name="intent" value="{{.Intent}}"><input type="hidden" name="action" value="create"><label for="title">Title</label><input id="title" name="title" maxlength="200" required><label for="category">Category</label><input id="category" name="category" maxlength="64" required><label for="body">Context</label><textarea id="body" name="body" maxlength="12000" required></textarea><label><input name="comments" type="checkbox">Allow comments</label><button>Create draft</button></form>{{end}}</main></body></html>`))

func Render(out io.Writer, p Page) error { return pageTemplate.Execute(out, p) }

func localRequestPath(path string) bool {
	if !strings.HasPrefix(path, "/") || strings.HasPrefix(path, "//") || strings.ContainsAny(path, "\\\r\n") || !utf8.ValidString(path) {
		return false
	}
	for _, part := range strings.Split(path, "/") {
		if part == "." || part == ".." {
			return false
		}
	}
	return true
}
