package federation

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"html/template"
	"net/http"
	"net/url"
)

type Members struct {
	Store     Store
	Auth      HostAuth
	CSRFToken func(*http.Request) string
}

var memberPage = template.Must(template.New("member").Parse(`<!doctype html><html lang="en"><meta charset="utf-8"><title>Shared discovery</title><main><h1>Shared discovery</h1><p>Sharing is optional and limited to one peer. It never transfers credit. Withdrawing queues a signed removal; disconnected peers may retain the prior listing until its expiry.</p><form method="post"><input type="hidden" name="csrf" value="{{.CSRF}}"><input type="hidden" name="intent" value="{{.Intent}}"><label>Peer network <input name="remote" required maxlength="128"></label><label>Listing reference <input name="listing" required maxlength="128"></label><label>Action <select name="action"><option value="consent">Allow this peer</option><option value="share">Share current listing</option><option value="withdraw">Withdraw listing</option></select></label><button>Confirm</button></form><h2>Remote listings</h2>{{range .Rows}}<article><h3>{{.Content.Title}}</h3><p>Remote network: {{.Peer}}</p><p>{{.Content.Description}}</p><p>{{.Content.Category}} — {{.Content.Area}}</p></article>{{else}}<p>No shared remote listings.</p>{{end}}{{if .Next}}<a href="{{.Next}}">Next listings</a>{{end}}</main></html>`))

func (h Members) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	problem := func(code int, title string) {
		w.Header().Set("Content-Type", "application/problem+json")
		w.WriteHeader(code)
		json.NewEncoder(w).Encode(map[string]any{"type": "about:blank", "title": title, "status": code})
	}
	if h.Auth == nil {
		problem(503, "Unavailable")
		return
	}
	if r.Method != "GET" && r.Method != "POST" {
		w.Header().Set("Allow", "GET, POST")
		problem(405, "Method not allowed")
		return
	}
	peer, listing, limit, pageErr := ParsePage(r)
	if pageErr != nil {
		problem(400, "Invalid page")
		return
	}
	if r.Method == "POST" {
		r.Body = http.MaxBytesReader(w, r.Body, 65536)
		defer r.Body.Close()
		if e := strictForm(r, "csrf", "intent", "remote", "listing", "action"); e != nil {
			status := 400
			var tooLarge *http.MaxBytesError
			if errors.As(e, &tooLarge) {
				status = 413
			}
			problem(status, "Invalid or excessive form")
			return
		}
	}
	a, e := h.Auth(r, r.Method == "POST")
	if e != nil || r.Header.Get("X-Network") != "" && r.Header.Get("X-Network") != a.Network {
		problem(403, "Access denied")
		return
	}
	if r.Method == "POST" {
		remote, listing := r.PostForm.Get("remote"), r.PostForm.Get("listing")
		switch r.PostForm.Get("action") {
		case "consent":
			e = h.Store.Consent(r.Context(), a, remote, listing, true)
		case "share":
			_, e = h.Store.Queue(r.Context(), a, remote, listing, r.PostForm.Get("intent"), false)
		case "withdraw":
			e = h.Store.Consent(r.Context(), a, remote, listing, false)
		default:
			e = ErrInvalid
		}
		if e != nil {
			problem(400, "Action unavailable")
			return
		}
		w.WriteHeader(204)
		return
	}
	token := ""
	if h.CSRFToken != nil {
		token = h.CSRFToken(r)
	}
	current, authErr := h.Auth(r, false)
	if authErr != nil || current != a || r.Context().Err() != nil {
		problem(403, "Access denied")
		return
	}
	rows, e := h.Store.BrowsePage(r.Context(), a, peer, listing, limit)
	if e != nil {
		problem(403, "Listings unavailable")
		return
	}
	next := ""
	if len(rows) == limit {
		last := rows[len(rows)-1]
		query := url.Values{"after_peer": {last.Peer}, "after_listing": {last.Listing}}
		next = "?" + query.Encode()
	}
	seed := make([]byte, 16)
	if _, e = rand.Read(seed); e != nil {
		problem(503, "Unavailable")
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	memberPage.Execute(w, struct {
		CSRF, Intent, Next string
		Rows               []RemoteListing
	}{token, hex.EncodeToString(seed), next, rows})
}
