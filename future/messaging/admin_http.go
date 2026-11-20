package messaging

import (
	"database/sql"
	"fmt"
	"html/template"
	"net/http"
	"net/url"
	"strings"
)

func isAdminPage(path string) bool {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	return path == "/messaging-maintenance" || len(parts) == 3 && parts[0] == "conversations" && (parts[2] == "moderation" || parts[2] == "moderate")
}

type adminData struct {
	Token, ID, NextMessages, NextReports string
	Messages                             []Message
	Reports                              []Report
}

func (h *handler) adminPage(w http.ResponseWriter, r *http.Request, a Actor) {
	ctx := r.Context()
	parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	maintenance := r.URL.Path == "/messaging-maintenance"
	id := ""
	if !maintenance {
		if len(parts) != 3 || !validID(parts[1]) {
			respondError(w, ErrInvalid, false)
			return
		}
		id = parts[1]
	}
	if r.Method == "POST" {
		var e error
		if maintenance {
			switch r.FormValue("action") {
			case "purge":
				_, e = h.store.Purge(ctx, a)
			case "shorten":
				days, err := pageNumber(r.FormValue("days"), 0)
				if err != nil || days > 30 {
					e = ErrInvalid
				} else {
					e = h.store.ShortenRetention(ctx, a, int(days))
				}
			default:
				e = ErrInvalid
			}
		} else {
			sequence, err := pageNumber(r.FormValue("sequence"), 0)
			if err != nil {
				e = err
			} else {
				e = h.store.Moderate(ctx, a, id, r.FormValue("action"), sequence, r.FormValue("reporter"))
			}
		}
		if e != nil {
			respondError(w, e, false)
			return
		}
		target := "/messaging-maintenance"
		if !maintenance {
			target = "/conversations/" + id + "/moderation"
		}
		http.Redirect(w, r, target, http.StatusSeeOther)
		return
	}
	token, e := h.formToken(r, a)
	if e != nil {
		respondError(w, ErrDenied, false)
		return
	}
	data := adminData{Token: token, ID: id}
	page := maintenancePage
	if maintenance {
		e = h.store.unit(ctx, a, "retention", "", func(_tx *sql.Tx) error { return nil })
	} else {
		var afterMessages, afterReports int64
		afterMessages, e = pageNumber(r.URL.Query().Get("message_after"), 0)
		if e == nil {
			afterReports, e = pageNumber(r.URL.Query().Get("report_after"), 0)
		}
		if e == nil {
			data.Messages, e = h.store.HeldMessages(ctx, a, id, afterMessages, 100)
		}
		if e == nil {
			data.Reports, e = h.store.ReportsPage(ctx, a, id, afterReports, r.URL.Query().Get("reporter"), 100)
		}
		if len(data.Messages) == 100 {
			query := r.URL.Query()
			query.Set("message_after", fmt.Sprint(data.Messages[99].Sequence))
			data.NextMessages = "?" + query.Encode()
		}
		if len(data.Reports) == 100 {
			query := url.Values{}
			for key, values := range r.URL.Query() {
				query[key] = append([]string(nil), values...)
			}
			last := data.Reports[99]
			query.Set("report_after", fmt.Sprint(last.Sequence))
			query.Set("reporter", last.Reporter)
			data.NextReports = "?" + query.Encode()
		}
		page = moderationPage
	}
	if e != nil {
		respondError(w, e, false)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	page.Execute(w, data)
}

var maintenancePage = template.Must(template.New("maintenance").Parse(`<!doctype html><html lang="en"><meta charset="utf-8"><title>Messaging retention</title><main><h1>Messaging retention</h1><p>Separate network maintenance permission is required. Shortening never extends existing deadlines; held evidence remains restricted.</p><form method="post"><input type="hidden" name="csrf" value="{{.Token}}"><input type="hidden" name="action" value="shorten"><label>Maximum inactivity retention days <input type="number" name="days" min="1" max="30" required></label><button>Shorten existing retention</button></form><form method="post"><input type="hidden" name="csrf" value="{{.Token}}"><input type="hidden" name="action" value="purge"><button>Purge expired unheld content</button></form></main></html>`))
var moderationPage = template.Must(template.New("moderation").Parse(`<!doctype html><html lang="en"><meta charset="utf-8"><title>Message moderation</title><main><h1>Message moderation</h1><p>Restricted server-readable evidence. Expired content is available only under a hold placed before expiry.</p><h2>Evidence</h2><ol>{{range .Messages}}<li value="{{.Sequence}}"><strong>{{.Sender}}</strong><pre>{{.Body}}</pre></li>{{else}}<li>No evidence.</li>{{end}}</ol>{{if .NextMessages}}<a href="{{.NextMessages}}">Next evidence</a>{{end}}<h2>Reports</h2><ul>{{range .Reports}}<li>Message {{.Sequence}}, reporter {{.Reporter}}, status {{.State}}<p>{{.Reason}}</p><form method="post" action="/conversations/{{$.ID}}/moderate"><input type="hidden" name="csrf" value="{{$.Token}}"><input type="hidden" name="sequence" value="{{.Sequence}}"><input type="hidden" name="reporter" value="{{.Reporter}}"><button name="action" value="resolve">Resolve report</button><button name="action" value="dismiss">Dismiss report</button></form></li>{{else}}<li>No reports.</li>{{end}}</ul>{{if .NextReports}}<a href="{{.NextReports}}">Next reports</a>{{end}}<form method="post" action="/conversations/{{.ID}}/moderate"><input type="hidden" name="csrf" value="{{.Token}}"><label>Legal hold <select name="action"><option value="hold">Place hold</option><option value="release">Release hold</option></select></label><button>Apply legal hold decision</button></form></main></html>`))
