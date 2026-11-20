package messaging

import (
	"bytes"
	"encoding/json"
	"errors"
	"html/template"
	"io"
	"mime"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"
)

// HTTPAuth is an explicit host boundary. Resolve authenticates a session/API key
// and selected network on EVERY request. VerifyCSRF must validate cookie-form
// requests and may exempt a separately authenticated non-cookie API credential.
// Token returns a session-bound token for no-JS forms. None may be nil.
type HTTPAuth struct {
	Resolve    func(*http.Request) (Actor, error)
	VerifyCSRF func(*http.Request) error
	Token      func(*http.Request) (string, error)
}
type handler struct {
	store *Store
	auth  HTTPAuth
}

// Handler returns an unmounted handler; it starts no listener or background job.
// Mount under the documented paths on a TLS-only origin. Host middleware owns
// secure cookies, sessions, network opt-in, CSRF and principal revocation.
func Handler(s *Store, auth HTTPAuth) (http.Handler, error) {
	if s == nil || auth.Resolve == nil || auth.VerifyCSRF == nil || auth.Token == nil {
		return nil, ErrInvalid
	}
	return &handler{s, auth}, nil
}

func (h *handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Pragma", "no-cache")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("Content-Security-Policy", "default-src 'none'; form-action 'self'; frame-ancestors 'none'; base-uri 'none'")
	api := strings.HasPrefix(r.URL.Path, "/api/v1/")
	// Bound query parsing before host callbacks. Keep existing route-specific
	// pagination fields; this transport guard rejects malformed encoding only.
	if len(r.URL.RawQuery) > 8192 {
		respondError(w, ErrInvalid, api)
		return
	}
	if _, err := url.ParseQuery(r.URL.RawQuery); err != nil {
		respondError(w, ErrInvalid, api)
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, 64<<10)
	defer r.Body.Close()
	// Parse before a host CSRF/session lookup can hide ParseForm's one-time
	// error behind cached partial values. Valid PostForm remains available.
	if r.Method == "POST" && !api {
		if err := r.ParseForm(); err != nil {
			respondError(w, ErrInvalid, false)
			return
		}
	}
	a, e := h.auth.Resolve(r)
	if e != nil {
		respondError(w, ErrDenied, api)
		return
	}
	allowed := "GET, POST"
	if api {
		allowed = apiMethods(r.URL.Path)
	}
	validMethod := false
	for _, method := range strings.Split(allowed, ", ") {
		if r.Method == method {
			validMethod = true
		}
	}
	if !validMethod {
		w.Header().Set("Allow", allowed)
		if api {
			w.Header().Set("Content-Type", "application/problem+json")
			w.WriteHeader(http.StatusMethodNotAllowed)
			json.NewEncoder(w).Encode(map[string]any{"type": "about:blank", "title": "Method not allowed", "status": 405})
		} else {
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
		return
	}
	if r.Method == "POST" {
		if e = h.auth.VerifyCSRF(r); e != nil || h.currentActor(r, a) != nil {
			respondError(w, ErrDenied, api)
			return
		}
	}
	if api {
		h.api(w, r, a)
		return
	}
	h.page(w, r, a)
}

// currentActor rejects session revocation/rotation during host callbacks.
// Store operations still enforce fresh membership and content policy in their
// existing transactions; this check does not replace those authorization gates.
func (h *handler) currentActor(r *http.Request, actor Actor) error {
	if r.Context().Err() != nil {
		return ErrDenied
	}
	current, err := h.auth.Resolve(r)
	if err != nil || current != actor || !validID(current.Network) || !validID(current.Member) || r.Context().Err() != nil {
		return ErrDenied
	}
	return nil
}

func (h *handler) formToken(r *http.Request, actor Actor) (string, error) {
	token, err := h.auth.Token(r)
	if err != nil || h.currentActor(r, actor) != nil {
		return "", ErrDenied
	}
	return token, nil
}

func apiMethods(path string) string {
	path = strings.TrimPrefix(path, "/api/v1/")
	if path == "block" || path == "purge" || path == "retention" {
		return "POST"
	}
	parts := strings.Split(path, "/")
	if len(parts) == 3 && parts[0] == "conversations" && validID(parts[1]) {
		switch parts[2] {
		case "export", "reports", "held":
			return "GET"
		case "hide", "close", "read", "report", "moderate", "search":
			return "POST"
		}
	}
	return "GET, POST"
}

func respondError(w http.ResponseWriter, e error, api bool) {
	status := http.StatusInternalServerError
	title := "Messaging request failed"
	switch {
	case errors.Is(e, ErrDenied):
		status = http.StatusForbidden
		title = "Message unavailable"
	case errors.Is(e, ErrInvalid):
		status = http.StatusBadRequest
		title = "Invalid messaging input"
	case errors.Is(e, ErrConflict):
		status = http.StatusConflict
		title = "Idempotency conflict"
	case errors.Is(e, ErrQuota):
		status = http.StatusTooManyRequests
		title = "Messaging quota exceeded"
	}
	if api {
		w.Header().Set("Content-Type", "application/problem+json")
		w.WriteHeader(status)
		_ = json.NewEncoder(w).Encode(map[string]any{"type": "about:blank", "title": title, "status": status})
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(status)
	_ = errorPage.Execute(w, title)
}

var errorPage = template.Must(template.New("error").Parse(`<!doctype html><html lang="en"><meta charset="utf-8"><title>Messaging error</title><main><h1>Messaging error</h1><p role="alert">{{.}}</p><a href="/conversations">Return to conversations</a></main></html>`))

func jsonReply(w http.ResponseWriter, value any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(value)
}
func decode(r *http.Request, v any) error {
	media, _, e := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if e != nil || media != "application/json" {
		return ErrInvalid
	}
	raw, err := io.ReadAll(io.LimitReader(r.Body, (64<<10)+1))
	if err != nil || len(raw) > 64<<10 || !utf8.Valid(raw) || !validJSONUnicodeEscapes(raw) {
		return ErrInvalid
	}
	raw = bytes.TrimSpace(raw)
	if len(raw) == 0 || raw[0] != '{' || !uniqueJSONNames(raw) {
		return ErrInvalid
	}
	d := json.NewDecoder(bytes.NewReader(raw))
	d.DisallowUnknownFields()
	if e := d.Decode(v); e != nil {
		return ErrInvalid
	}
	var extra any
	if e := d.Decode(&extra); e != io.EOF {
		return ErrInvalid
	}
	return nil
}

// JSON permits UTF-16 escapes; reject unpaired surrogates rather than letting
// encoding/json silently replace them before policy validation.
func validJSONUnicodeEscapes(raw []byte) bool {
	for i := 0; i < len(raw); {
		if raw[i] != '\\' {
			i++
			continue
		}
		if i+1 >= len(raw) {
			return false
		}
		if raw[i+1] != 'u' {
			i += 2
			continue
		}
		if i+6 > len(raw) {
			return false
		}
		value, err := strconv.ParseUint(string(raw[i+2:i+6]), 16, 16)
		if err != nil {
			return false
		}
		if value >= 0xD800 && value <= 0xDBFF {
			if i+12 > len(raw) || raw[i+6] != '\\' || raw[i+7] != 'u' {
				return false
			}
			low, err := strconv.ParseUint(string(raw[i+8:i+12]), 16, 16)
			if err != nil || low < 0xDC00 || low > 0xDFFF {
				return false
			}
			i += 12
			continue
		}
		if value >= 0xDC00 && value <= 0xDFFF {
			return false
		}
		i += 6
	}
	return true
}

// uniqueJSONNames rejects ambiguous duplicate names, including case aliases
// that encoding/json otherwise maps to the same Go field. Depth is bounded.
func uniqueJSONNames(raw []byte) bool {
	decoder := json.NewDecoder(bytes.NewReader(raw))
	var value func(int) bool
	value = func(depth int) bool {
		if depth > 16 {
			return false
		}
		token, err := decoder.Token()
		if err != nil {
			return false
		}
		delimiter, container := token.(json.Delim)
		if !container {
			return true
		}
		switch delimiter {
		case '{':
			seen := map[string]bool{}
			for decoder.More() {
				key, err := decoder.Token()
				if err != nil {
					return false
				}
				name, ok := key.(string)
				if !ok {
					return false
				}
				name = jsonNameKey(name)
				if seen[name] {
					return false
				}
				seen[name] = true
				if !value(depth + 1) {
					return false
				}
			}
			end, err := decoder.Token()
			return err == nil && end == json.Delim('}')
		case '[':
			for decoder.More() {
				if !value(depth + 1) {
					return false
				}
			}
			end, err := decoder.Token()
			return err == nil && end == json.Delim(']')
		default:
			return false
		}
	}
	if !value(0) {
		return false
	}
	_, err := decoder.Token()
	return err == io.EOF
}

func pageNumber(raw string, defaultValue int64) (int64, error) {
	if raw == "" {
		return defaultValue, nil
	}
	v, e := strconv.ParseInt(raw, 10, 64)
	if e != nil || v < 0 {
		return 0, ErrInvalid
	}
	return v, nil
}
func (h *handler) api(w http.ResponseWriter, r *http.Request, a Actor) {
	path := strings.TrimPrefix(r.URL.Path, "/api/v1/")
	parts := strings.Split(path, "/")
	ctx := r.Context()
	var result any
	var e error
	switch {
	case path == "conversations" && r.Method == "GET":
		result, e = h.store.Threads(ctx, a, r.URL.Query().Get("after"), 100)
	case path == "conversations" && r.Method == "POST":
		var in struct {
			Target    string `json:"target"`
			Reference string `json:"reference"`
			Days      *int   `json:"retention_days"`
		}
		if e = decode(r, &in); e == nil {
			days := 30
			if in.Days != nil {
				days = *in.Days
			}
			result, e = h.store.Create(ctx, a, in.Target, in.Reference, r.Header.Get("Idempotency-Key"), days)
		}
	case len(parts) == 3 && parts[0] == "conversations" && validID(parts[1]):
		id := parts[1]
		switch parts[2] {
		case "messages":
			if r.Method == "GET" {
				var after int64
				after, e = pageNumber(r.URL.Query().Get("after"), 0)
				if e == nil {
					result, e = h.store.Messages(ctx, a, id, after, 100)
				}
			} else {
				var in struct {
					Body string `json:"body"`
				}
				if e = decode(r, &in); e == nil {
					result, e = h.store.Send(ctx, a, id, in.Body, r.Header.Get("Idempotency-Key"))
				}
			}
		case "search":
			var in struct {
				Query string `json:"query"`
				After int64  `json:"after"`
			}
			if e = decode(r, &in); e == nil {
				result, e = h.store.Search(ctx, a, id, in.Query, in.After, 100)
			}
		case "export":
			if r.Method != "GET" {
				e = ErrInvalid
				break
			}
			w.Header().Set("Content-Type", "application/json")
			var after int64
			after, e = pageNumber(r.URL.Query().Get("after"), 0)
			if e == nil {
				e = h.store.ExportPage(ctx, a, id, after, w)
			}
			if e == nil {
				return
			}
		case "reports":
			if r.Method != "GET" {
				e = ErrInvalid
				break
			}
			var after int64
			after, e = pageNumber(r.URL.Query().Get("after"), 0)
			if e == nil {
				result, e = h.store.ReportsPage(ctx, a, id, after, r.URL.Query().Get("reporter"), 100)
			}
		case "held":
			var after int64
			if r.Method != "GET" {
				e = ErrInvalid
				break
			}
			after, e = pageNumber(r.URL.Query().Get("after"), 0)
			if e == nil {
				result, e = h.store.HeldMessages(ctx, a, id, after, 100)
			}
		case "hide", "close":
			if r.Method != "POST" {
				e = ErrInvalid
				break
			}
			var in struct{}
			if e = decode(r, &in); e != nil {
				break
			}
			if parts[2] == "hide" {
				e = h.store.Hide(ctx, a, id)
			} else {
				e = h.store.CloseThread(ctx, a, id)
			}
		case "read":
			if r.Method != "POST" {
				e = ErrInvalid
				break
			}
			var in struct {
				Sequence *int64 `json:"sequence"`
			}
			if e = decode(r, &in); e == nil {
				if in.Sequence == nil {
					e = ErrInvalid
				} else {
					e = h.store.MarkRead(ctx, a, id, *in.Sequence)
				}
			}
		case "report":
			if r.Method != "POST" {
				e = ErrInvalid
				break
			}
			var in struct {
				Sequence int64  `json:"sequence"`
				Reason   string `json:"reason"`
			}
			if e = decode(r, &in); e == nil {
				e = h.store.Report(ctx, a, id, in.Sequence, in.Reason)
			}
		case "moderate":
			if r.Method != "POST" {
				e = ErrInvalid
				break
			}
			var in struct {
				Action   string `json:"action"`
				Sequence int64  `json:"sequence"`
				Reporter string `json:"reporter"`
			}
			if e = decode(r, &in); e == nil {
				e = h.store.Moderate(ctx, a, id, in.Action, in.Sequence, in.Reporter)
			}
		default:
			e = ErrInvalid
		}
	case path == "block" && r.Method == "POST":
		var in struct {
			Target  string `json:"target"`
			Enabled *bool  `json:"enabled"`
		}
		if e = decode(r, &in); e == nil {
			if in.Enabled == nil {
				e = ErrInvalid
			} else {
				e = h.store.Block(ctx, a, in.Target, *in.Enabled)
			}
		}
	case path == "retention" && r.Method == "POST":
		var in struct {
			Days *int `json:"days"`
		}
		if e = decode(r, &in); e == nil {
			if in.Days == nil {
				e = ErrInvalid
			} else {
				e = h.store.ShortenRetention(ctx, a, *in.Days)
			}
		}
	case path == "purge" && r.Method == "POST":
		var in struct{}
		if e = decode(r, &in); e == nil {
			result, e = h.store.Purge(ctx, a)
		}
	default:
		e = ErrInvalid
	}
	if e != nil {
		respondError(w, e, true)
		return
	}
	if result == nil {
		result = map[string]bool{"ok": true}
	}
	jsonReply(w, result)
}

type pageData struct {
	Token, Key, ID, After, Query string
	Member                       string
	Threads                      []Conversation
	Messages                     []Message
	Next                         int64
	Target                       string
}

var inboxPage = template.Must(template.New("inbox").Parse(`<!doctype html><html lang="en"><meta charset="utf-8"><title>Private messages</title><main><h1>Private messages</h1><p>Messages can be read by the server operator; this is not end-to-end encrypted.</p><h2>Conversations</h2><ul>{{range .Threads}}<li><a href="/conversations/{{.ID}}">{{.A}} and {{.B}}</a> — {{.Unread}} unread</li>{{else}}<li>No conversations.</li>{{end}}</ul>{{if .After}}<a href="/conversations?after={{.After}}">Next conversations</a>{{end}}<h2>Start conversation</h2><form method="post" action="/conversations"><input type="hidden" name="csrf" value="{{.Token}}"><input type="hidden" name="key" value="{{.Key}}"><label>Member ID <input name="target" required maxlength="128"></label><label>Optional listing or transaction reference <input name="reference" maxlength="128"></label><label>Retention days (1–30, subject to network policy) <input name="days" type="number" min="1" max="30" value="30" required></label><button>Start conversation</button></form><h2>Block or unblock member</h2><form method="post" action="/block"><input type="hidden" name="csrf" value="{{.Token}}"><label>Member ID <input name="target" required maxlength="128"></label><label>Action <select name="enabled"><option value="true">Block</option><option value="false">Unblock</option></select></label><button>Save block setting</button></form></main></html>`))
var threadPage = template.Must(template.New("thread").Parse(`<!doctype html><html lang="en"><meta charset="utf-8"><title>Conversation</title><main><h1>Conversation</h1><p>Messages can be read by the server operator; this is not end-to-end encrypted. No attachments, remote images or link previews.</p><a href="/conversations">All conversations</a><h2>Search visible messages</h2><p>Literal, case-sensitive search. Search text stays in the request body.</p><form method="post" action="/conversations/{{.ID}}/search"><input type="hidden" name="csrf" value="{{.Token}}"><label>Search text (maximum 128 UTF-8 bytes) <input name="query" value="{{.Query}}" required maxlength="128"></label><button>Search messages</button></form>{{if .Query}}<h2>Search results</h2>{{end}}<ol>{{range .Messages}}<li value="{{.Sequence}}"><strong>{{.Sender}}</strong><pre>{{.Body}}</pre>{{if ne .Sender $.Member}}<form method="post" action="/conversations/{{$.ID}}/report"><input type="hidden" name="csrf" value="{{$.Token}}"><input type="hidden" name="sequence" value="{{.Sequence}}"><label>Report reason <input name="reason" required maxlength="1024"></label><button>Report message {{.Sequence}}</button></form>{{end}}</li>{{else}}<li>No visible messages.</li>{{end}}</ol>{{if .Next}}{{if .Query}}<form method="post" action="/conversations/{{.ID}}/search"><input type="hidden" name="csrf" value="{{.Token}}"><input type="hidden" name="query" value="{{.Query}}"><input type="hidden" name="after" value="{{.Next}}"><button>Next search results</button></form>{{else}}<a href="/conversations/{{.ID}}?after={{.Next}}">Next messages</a>{{end}}{{end}}<h2>Send message</h2><form method="post" action="/conversations/{{.ID}}/send"><input type="hidden" name="csrf" value="{{.Token}}"><input type="hidden" name="key" value="{{.Key}}"><label>Message (maximum 4096 UTF-8 bytes) <textarea name="body" maxlength="4096" required></textarea></label><button>Send message</button></form><form method="post" action="/conversations/{{.ID}}/read"><input type="hidden" name="csrf" value="{{.Token}}"><label>Last read sequence <input name="sequence" type="number" min="0" required></label><button>Mark read</button></form><form method="post" action="/conversations/{{.ID}}/hide"><input type="hidden" name="csrf" value="{{.Token}}"><button>Hide current history from my view</button></form><form method="post" action="/conversations/{{.ID}}/close"><input type="hidden" name="csrf" value="{{.Token}}"><button>Close conversation for sending</button></form><a href="/api/v1/conversations/{{.ID}}/export">Export my visible messages (JSON)</a></main></html>`))

func (h *handler) page(w http.ResponseWriter, r *http.Request, a Actor) {
	if isAdminPage(r.URL.Path) {
		h.adminPage(w, r, a)
		return
	}
	ctx := r.Context()
	parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	var e error
	redirect := "/conversations"
	if r.Method == "POST" {
		switch {
		case r.URL.Path == "/conversations":
			var days int64
			days, e = pageNumber(r.FormValue("days"), 30)
			if e == nil && days <= 30 {
				var t Conversation
				t, e = h.store.Create(ctx, a, r.FormValue("target"), r.FormValue("reference"), r.FormValue("key"), int(days))
				if e == nil {
					redirect += "/" + t.ID
				}
			} else {
				e = ErrInvalid
			}
		case r.URL.Path == "/block":
			enabled := r.FormValue("enabled")
			if enabled != "true" && enabled != "false" {
				e = ErrInvalid
			} else {
				e = h.store.Block(ctx, a, r.FormValue("target"), enabled == "true")
			}
		case len(parts) == 3 && parts[0] == "conversations" && validID(parts[1]):
			id := parts[1]
			redirect += "/" + id
			switch parts[2] {
			case "search":
				h.searchPage(w, r, a, id)
				return
			case "send":
				_, e = h.store.Send(ctx, a, id, r.FormValue("body"), r.FormValue("key"))
			case "hide":
				e = h.store.Hide(ctx, a, id)
				redirect = "/conversations"
			case "close":
				e = h.store.CloseThread(ctx, a, id)
			case "read", "report":
				var seq int64
				seq, e = pageNumber(r.FormValue("sequence"), 0)
				if e == nil {
					if parts[2] == "read" {
						e = h.store.MarkRead(ctx, a, id, seq)
					} else {
						e = h.store.Report(ctx, a, id, seq, r.FormValue("reason"))
					}
				}
			default:
				e = ErrInvalid
			}
		default:
			e = ErrInvalid
		}
		if e != nil {
			respondError(w, e, false)
			return
		}
		http.Redirect(w, r, redirect, http.StatusSeeOther)
		return
	}
	token, e := h.formToken(r, a)
	if e != nil {
		respondError(w, ErrDenied, false)
		return
	}
	key, e := newID()
	if e != nil {
		respondError(w, e, false)
		return
	}
	data := pageData{Token: token, Key: key, Member: a.Member}
	tmpl := inboxPage
	switch {
	case r.URL.Path == "/conversations":
		data.Threads, e = h.store.Threads(ctx, a, r.URL.Query().Get("after"), 100)
		if len(data.Threads) == 100 {
			data.After = data.Threads[99].ID
		}
	case len(parts) == 2 && parts[0] == "conversations" && validID(parts[1]):
		data.ID = parts[1]
		var after int64
		after, e = pageNumber(r.URL.Query().Get("after"), 0)
		if e == nil {
			data.Messages, e = h.store.Messages(ctx, a, data.ID, after, 100)
		}
		if len(data.Messages) == 100 {
			data.Next = data.Messages[99].Sequence
		}
		tmpl = threadPage
	default:
		e = ErrInvalid
	}
	if e != nil {
		respondError(w, e, false)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_ = tmpl.Execute(w, data)
}

// jsonNameKey matches encoding/json's Unicode simple-fold field equivalence.
// It is only a comparison key: input names and valid single aliases are not
// rewritten, and distinct names requiring multi-rune expansions stay distinct.
func jsonNameKey(name string) string {
	return strings.Map(func(r rune) rune {
		least := r
		for folded := unicode.SimpleFold(r); folded != r; folded = unicode.SimpleFold(folded) {
			if folded < least {
				least = folded
			}
		}
		return least
	}, name)
}
