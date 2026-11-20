package recurring

import (
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"unicode/utf8"
)

// Session must authenticate current membership and validate cookie CSRF on POST.
// It returns the token to render in no-JS forms. JSON callers normally send the
// host's CSRF header. No route, login or session storage is registered here.
type Session func(*http.Request, bool) (Actor, string, error)
type Handler struct {
	Service Service
	Session Session
}
type command struct {
	Action, ID, Key string
	Expected        int64
	Record          Record
}

func problem(w http.ResponseWriter, code int, title string) {
	w.Header().Set("Content-Type", "application/problem+json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(map[string]any{"type": "about:blank", "title": title, "status": code})
}
func (h Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("Content-Security-Policy", "default-src 'none'; style-src 'none'; form-action 'self'; base-uri 'none'; frame-ancestors 'none'")
	if r.Method != "GET" && r.Method != "POST" {
		w.Header().Set("Allow", "GET, POST")
		problem(w, 405, "Method not allowed")
		return
	}
	if !localAction(r.URL.Path) {
		problem(w, 400, "Invalid local path")
		return
	}
	if h.Session == nil || h.Service.ready() != nil {
		problem(w, 503, "Recurring templates unavailable")
		return
	}
	// Validate pagination before any host parsing or state-changing action.
	if len(r.URL.RawQuery) > 8192 || r.Method == "POST" && r.URL.RawQuery != "" {
		problem(w, 400, "Invalid query")
		return
	}
	query, queryErr := url.ParseQuery(r.URL.RawQuery)
	if queryErr != nil {
		problem(w, 400, "Invalid query")
		return
	}
	for key, values := range query {
		if (key != "after" && key != "limit") || len(values) != 1 {
			problem(w, 400, "Invalid query")
			return
		}
	}
	after := query.Get("after")
	if after != "" && !validID(after) {
		problem(w, 400, "Invalid cursor")
		return
	}
	limit := 20
	if raw := query.Get("limit"); raw != "" {
		var limitErr error
		limit, limitErr = strconv.Atoi(raw)
		if limitErr != nil || limit < 1 || limit > 100 {
			problem(w, 400, "Invalid page limit")
			return
		}
	}
	formRequest := strings.HasPrefix(r.Header.Get("Content-Type"), "application/x-www-form-urlencoded")
	var formInput command
	if r.Method == "POST" {
		r.Body = http.MaxBytesReader(w, r.Body, 65536)
		defer r.Body.Close()
		if formRequest {
			// Validate before a CSRF/session callback can hide ParseForm's first
			// error behind cached partial fields. Keep the command independent
			// of callback mutations while exposing parsed fields for host CSRF.
			var err error
			formInput, err = parseForm(r)
			if err != nil {
				var tooLarge *http.MaxBytesError
				if errors.As(err, &tooLarge) {
					problem(w, 413, "Request too large")
				} else {
					problem(w, 400, "Invalid request")
				}
				return
			}
		}
	}
	actor, csrf, err := h.Session(r, r.Method == "POST")
	if err != nil || !actorOK(actor) || r.Header.Get("X-Network") != "" && r.Header.Get("X-Network") != actor.Network {
		problem(w, 403, "Access denied")
		return
	}
	html := strings.Contains(r.Header.Get("Accept"), "text/html") || strings.HasPrefix(r.Header.Get("Content-Type"), "application/x-www-form-urlencoded")
	var response any
	if r.Method == "POST" {
		var input command
		if formRequest {
			input = formInput
		} else if strings.HasPrefix(r.Header.Get("Content-Type"), "application/json") {
			raw, readErr := io.ReadAll(r.Body)
			err = readErr
			if err == nil {
				err = decodeJSON(raw, &input)
			}
			input.Key = r.Header.Get("Idempotency-Key")
		} else {
			problem(w, 415, "Unsupported request body")
			return
		}
		if err != nil {
			var tooLarge *http.MaxBytesError
			if errors.As(err, &tooLarge) {
				problem(w, 413, "Request too large")
			} else {
				problem(w, 400, "Invalid request")
			}
			return
		}
		switch input.Action {
		case "save":
			response, err = h.Service.Save(r.Context(), actor, input.Record, input.Expected, input.Key)
		case "pause", "resume", "skip", "cancel":
			response, err = h.Service.Change(r.Context(), actor, input.ID, input.Action, input.Expected, input.Key)
		case "draft":
			response, err = h.Service.CreateDraft(r.Context(), actor, input.ID, input.Key)
		default:
			problem(w, 400, "Invalid template action")
			return
		}
		if err != nil {
			switch {
			case errors.Is(err, ErrDenied):
				problem(w, 403, "Action unavailable")
			case errors.Is(err, ErrConflict):
				problem(w, 409, "Revision or retry conflict")
			default:
				problem(w, 400, "Template request failed")
			}
			return
		}
		if !html {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(response)
			return
		}
	}
	rows, next, err := h.Service.List(r.Context(), actor, after, limit)
	if err != nil {
		problem(w, 403, "Templates unavailable")
		return
	}
	inbox, err := h.Service.Inbox(r.Context(), actor, 20)
	if err != nil {
		problem(w, 403, "Reminders unavailable")
		return
	}
	if html {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		if err = Render(w, Page{Path: r.URL.Path, CSRF: csrf, Rows: rows, Next: next, Reminders: inbox, Response: response}); err != nil {
			return
		}
	} else {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(struct {
			Rows      []Record   `json:"templates"`
			Next      string     `json:"next_cursor,omitempty"`
			Reminders []Reminder `json:"reminders"`
		}{rows, next, inbox})
	}
}
func parseForm(r *http.Request) (command, error) {
	var input command
	if err := r.ParseForm(); err != nil {
		return input, err
	}
	allowed := map[string]bool{}
	for _, key := range strings.Fields("action id key expected csrf target note amount frequency zone hour minute weekday monthday active") {
		allowed[key] = true
	}
	for key, values := range r.PostForm {
		if !allowed[key] || len(values) != 1 {
			return input, errors.New("unknown or duplicate form field")
		}
	}
	input.Action = r.PostForm.Get("action")
	input.ID = r.PostForm.Get("id")
	input.Key = r.PostForm.Get("key")
	var err error
	if raw := r.PostForm.Get("expected"); raw != "" {
		input.Expected, err = strconv.ParseInt(raw, 10, 64)
		if err != nil {
			return input, err
		}
	}
	if input.Action != "save" {
		return input, nil
	}
	r0 := Record{ID: input.ID, Target: r.PostForm.Get("target"), Note: r.PostForm.Get("note"), Active: r.PostForm.Get("active") == "on"}
	if raw := r.PostForm.Get("amount"); raw != "" {
		amount, e := strconv.ParseInt(raw, 10, 64)
		if e != nil {
			return input, e
		}
		r0.Amount = &amount
	}
	r0.Calendar.Frequency = r.PostForm.Get("frequency")
	r0.Calendar.Zone = r.PostForm.Get("zone")
	fields := []struct {
		key  string
		dest *int
	}{{"hour", &r0.Calendar.Hour}, {"minute", &r0.Calendar.Minute}, {"weekday", &r0.Calendar.Weekday}, {"monthday", &r0.Calendar.MonthDay}}
	for _, field := range fields {
		raw := r.PostForm.Get(field.key)
		if raw == "" {
			continue
		}
		*field.dest, err = strconv.Atoi(raw)
		if err != nil {
			return input, err
		}
	}
	input.Record = r0
	return input, nil
}

type Page struct {
	Path, CSRF string
	Rows       []Record
	Next       string
	Reminders  []Reminder
	Response   any
}

var page = template.Must(template.New("page").Funcs(template.FuncMap{"key": randomID, "amount": func(n *int64) string {
	if n == nil {
		return ""
	}
	return fmt.Sprint(*n)
}}).Parse(`<!doctype html><html lang="en"><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1"><title>Recurring trade reminders</title><main><h1>Recurring trade reminders</h1><p>Reminders never pay automatically. Each trade requires fresh review and acceptance. Times use your named time zone; missing local times are skipped and repeated times remind once.</p>{{if .Response}}<p role="status">Template action completed.</p>{{with .Draft}}<section aria-label="Draft for review"><h2>Draft for review</h2><p>This draft has not proposed or paid a trade. Use the normal trade review flow to continue.</p><dl><dt>Counterparty</dt><dd>{{.Target}}</dd><dt>Note</dt><dd>{{.Note}}</dd><dt>Amount in minor units</dt><dd>{{amount .Amount}}</dd></dl><input type="hidden" name="intent" value="{{.Intent}}"></section>{{end}}{{end}}<section><h2>Recent reminders</h2><ul>{{range .Reminders}}<li>Review template {{.Template}}. Scheduled {{.Scheduled}}.</li>{{else}}<li>No reminders.</li>{{end}}</ul></section><section><h2>Create template</h2><form method="post" action="{{.Path}}"><input type="hidden" name="csrf" value="{{.CSRF}}"><input type="hidden" name="key" value="{{key}}"><input type="hidden" name="action" value="save">{{template "fields" .Empty}}<button>Create reminder template</button></form></section>{{range .Rows}}<section><h2>Template {{.ID}}</h2><p>Revision {{.Version}}. {{if .Cancelled}}Cancelled{{else if .Active}}Active{{else}}Paused{{end}}</p>{{if not .Cancelled}}<form method="post" action="{{$.Path}}"><input type="hidden" name="csrf" value="{{$.CSRF}}"><input type="hidden" name="key" value="{{key}}"><input type="hidden" name="action" value="save"><input type="hidden" name="id" value="{{.ID}}"><input type="hidden" name="expected" value="{{.Version}}">{{template "fields" .}}<button>Save changes</button></form><form method="post" action="{{$.Path}}"><input type="hidden" name="csrf" value="{{$.CSRF}}"><input type="hidden" name="key" value="{{key}}"><input type="hidden" name="id" value="{{.ID}}"><input type="hidden" name="expected" value="{{.Version}}"><button name="action" value="pause">Pause</button><button name="action" value="resume">Resume</button><button name="action" value="skip">Skip next occurrence</button><button name="action" value="cancel">Cancel template</button><button name="action" value="draft">Create draft for review</button></form>{{end}}</section>{{end}}{{if .Next}}<a href="{{.Path}}?after={{.Next}}">Next page</a>{{end}}</main></html>{{define "fields"}}<label>Counterparty <input name="target" maxlength="256" required value="{{.Target}}"></label><label>Private note <textarea name="note" maxlength="1024">{{.Note}}</textarea></label><label>Optional amount in minor units <input name="amount" type="number" min="1" max="1125899906842624" value="{{amount .Amount}}"></label><label>Frequency (daily, weekly or monthly) <input name="frequency" required value="{{.Calendar.Frequency}}"></label><label>IANA time zone <input name="zone" required value="{{.Calendar.Zone}}"></label><label>Hour <input name="hour" type="number" min="0" max="23" value="{{.Calendar.Hour}}"></label><label>Minute <input name="minute" type="number" min="0" max="59" value="{{.Calendar.Minute}}"></label><label>Weekday (Sunday 0, weekly only) <input name="weekday" type="number" min="0" max="6" value="{{.Calendar.Weekday}}"></label><label>Day of month (monthly only) <input name="monthday" type="number" min="0" max="31" value="{{.Calendar.MonthDay}}"></label><label><input name="active" type="checkbox" {{if .Active}}checked{{end}}> Active reminders</label>{{end}}`))

func (Page) Empty() Record { return Record{Calendar: Calendar{Frequency: "daily", Zone: "UTC"}} }
func Render(w io.Writer, p Page) error {
	if !localAction(p.Path) {
		return errors.New("form action must be a local path")
	}
	return page.Execute(w, p)
}
func localAction(path string) bool {
	return utf8.ValidString(path) && (path == "" || strings.HasPrefix(path, "/") && !strings.HasPrefix(path, "//")) && !strings.ContainsAny(path, "\\\x00?#\r\n")
}

func (p Page) Draft() *Draft {
	if d, ok := p.Response.(Draft); ok {
		return &d
	}
	return nil
}
