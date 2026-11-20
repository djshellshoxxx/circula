package federation

import (
	"context"
	"crypto/ed25519"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"errors"
	"html/template"
	"io"
	"mime"
	"net/http"
	"net/url"
	"strconv"
	"sync"
	"time"
)

type ingressLimit struct {
	sync.Mutex
	window int64
	count  int
}
type Ingress struct {
	Store         Store
	Local, Remote string
	limit         ingressLimit
}

func (h *Ingress) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	if r.Method != "POST" {
		w.Header().Set("Allow", "POST")
		http.Error(w, "method not allowed", 405)
		return
	}
	kind, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if err != nil || kind != "application/circula-discovery+jws" {
		http.Error(w, "unsupported media", 415)
		return
	}
	now := time.Now()
	h.limit.Lock()
	window := now.Unix() / 60
	if h.limit.window != window {
		h.limit.window = window
		h.limit.count = 0
	}
	h.limit.count++
	allowed := h.limit.count <= 240
	h.limit.Unlock()
	if !allowed {
		http.Error(w, "rate limited", 429)
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, MaxWire)
	defer r.Body.Close()
	raw, e := io.ReadAll(r.Body)
	if e != nil {
		http.Error(w, "body rejected", 413)
		return
	}
	duplicate, e := h.Store.Receive(r.Context(), h.Local, h.Remote, string(raw))
	if e != nil {
		http.Error(w, "event rejected", 400)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(struct {
		Duplicate bool `json:"duplicate"`
	}{duplicate})
}

// HostAuth authenticates current identity and verifies cookie CSRF on mutations.
type HostAuth func(*http.Request, bool) (Actor, error)
type Admin struct {
	Store     Store
	Auth      HostAuth
	CSRFToken func(*http.Request) string
}
type PeerStatus struct {
	Remote, Endpoint string
	Enabled          bool
}
type QueueStatus struct {
	Remote, ID, Listing, Kind, State string
	Attempts                         int
}

func (s Store) Status(ctx context.Context, a Actor) ([]PeerStatus, []QueueStatus, error) {
	peers := []PeerStatus{}
	queue := []QueueStatus{}
	err := s.transaction(ctx, func(c *sql.Conn) error {
		if e := s.auth(ctx, c, a, true); e != nil {
			return e
		}
		rows, e := c.QueryContext(ctx, `SELECT remote,endpoint,enabled FROM discovery_peers WHERE local=? ORDER BY remote LIMIT 100`, a.Network)
		if e != nil {
			return e
		}
		for rows.Next() {
			var p PeerStatus
			if e = rows.Scan(&p.Remote, &p.Endpoint, &p.Enabled); e != nil {
				rows.Close()
				return e
			}
			peers = append(peers, p)
		}
		if e = rows.Err(); e != nil {
			rows.Close()
			return e
		}
		rows.Close()
		rows, e = c.QueryContext(ctx, `SELECT remote,id,listing,kind,state,attempts FROM discovery_outbox WHERE local=? ORDER BY remote,id LIMIT 100`, a.Network)
		if e != nil {
			return e
		}
		defer rows.Close()
		for rows.Next() {
			var q QueueStatus
			if e = rows.Scan(&q.Remote, &q.ID, &q.Listing, &q.Kind, &q.State, &q.Attempts); e != nil {
				return e
			}
			queue = append(queue, q)
		}
		return rows.Err()
	})
	return peers, queue, err
}

var adminPage = template.Must(template.New("admin").Parse(`<!doctype html><html lang="en"><meta charset="utf-8"><title>Discovery federation</title><main><h1>Discovery federation</h1><p>Remote listings are suggestions. Federation cannot transfer credit or administer this network.</p><form method="post"><input type="hidden" name="csrf" value="{{.CSRF}}"><label>Action <select name="action"><option value="enable">Enable network</option><option value="disable">Disable network</option><option value="approve">Approve or rotate peer key</option><option value="revoke">Revoke peer</option><option value="revoke-key">Revoke key</option></select></label><label>Peer ID <input name="remote" maxlength="128"></label><label>HTTPS endpoint <input name="endpoint" type="url"></label><label>Key ID <input name="kid" maxlength="128"></label><label>Ed25519 public key (base64) <input name="key"></label><label>Key valid from (RFC3339) <input name="from"></label><label>Key valid until (RFC3339) <input name="until"></label><button>Apply reviewed action</button></form><h2>Peers</h2><ul>{{range .Peers}}<li>{{.Remote}} — {{.Endpoint}} — enabled: {{.Enabled}}</li>{{else}}<li>No configured peers.</li>{{end}}</ul><h2>Outbox status</h2><ul>{{range .Queue}}<li>{{.Remote}} / {{.ID}} / {{.State}} / attempts: {{.Attempts}}</li>{{else}}<li>Empty.</li>{{end}}</ul></main></html>`))
var listingsPage = template.Must(template.New("listings").Parse(`<!doctype html><html lang="en"><meta charset="utf-8"><title>Remote listings</title><main><h1>Remote listings</h1><p>Remote content is untrusted. These listings cannot authorize local trades.</p>{{range .}}<article><h2>{{.Content.Title}}</h2><p>Remote network: {{.Peer}}</p><p>{{.Content.Description}}</p><p>Category: {{.Content.Category}}. Coarse area: {{.Content.Area}}.</p></article>{{else}}<p>No shared remote listings.</p>{{end}}</main></html>`))

func RenderListings(w io.Writer, listings []RemoteListing) error {
	return listingsPage.Execute(w, listings)
}
func (h Admin) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	problem := func(code int, title string) {
		w.Header().Set("Content-Type", "application/problem+json")
		w.WriteHeader(code)
		json.NewEncoder(w).Encode(map[string]any{"type": "about:blank", "status": code, "title": title})
	}
	if r.Method != "GET" && r.Method != "POST" {
		w.Header().Set("Allow", "GET, POST")
		problem(405, "Method not allowed")
		return
	}
	if h.Auth == nil {
		problem(503, "Unavailable")
		return
	}
	if r.URL.RawQuery != "" || r.URL.ForceQuery {
		problem(400, "Invalid query")
		return
	}
	if r.Method == "POST" {
		r.Body = http.MaxBytesReader(w, r.Body, 65536)
		defer r.Body.Close()
		if err := strictForm(r, "csrf", "action", "remote", "endpoint", "kid", "key", "from", "until"); err != nil {
			status := 400
			var tooLarge *http.MaxBytesError
			if errors.As(err, &tooLarge) {
				status = 413
			}
			problem(status, "Invalid or excessive form")
			return
		}
	}
	a, err := h.Auth(r, r.Method == "POST")
	if err != nil || r.Header.Get("X-Network") != "" && r.Header.Get("X-Network") != a.Network {
		problem(403, "Access denied")
		return
	}
	if r.Method == "GET" {
		token := ""
		if h.CSRFToken != nil {
			token = h.CSRFToken(r)
		}
		current, authErr := h.Auth(r, false)
		if authErr != nil || current != a || r.Context().Err() != nil {
			problem(403, "Access denied")
			return
		}
		peers, queue, e := h.Store.Status(r.Context(), a)
		if e != nil {
			problem(403, "Access denied")
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		adminPage.Execute(w, struct {
			CSRF  string
			Peers []PeerStatus
			Queue []QueueStatus
		}{token, peers, queue})
		return
	}
	switch r.PostForm.Get("action") {
	case "enable":
		err = h.Store.EnableNetwork(r.Context(), a, true)
	case "disable":
		err = h.Store.EnableNetwork(r.Context(), a, false)
	case "revoke":
		err = h.Store.RevokePeer(r.Context(), a, r.PostForm.Get("remote"))
	case "revoke-key":
		err = h.Store.RevokeKey(r.Context(), a, r.PostForm.Get("remote"), r.PostForm.Get("kid"))
	case "approve":
		key, e := base64.StdEncoding.DecodeString(r.PostForm.Get("key"))
		from, e1 := time.Parse(time.RFC3339, r.PostForm.Get("from"))
		until, e2 := time.Parse(time.RFC3339, r.PostForm.Get("until"))
		if e != nil || e1 != nil || e2 != nil || len(key) != ed25519.PublicKeySize {
			err = ErrInvalid
			break
		}
		err = h.Store.ApprovePeer(r.Context(), a, PeerConfig{a.Network, r.PostForm.Get("remote"), r.PostForm.Get("endpoint"), true}, TrustedKey{r.PostForm.Get("kid"), key, from, until, false})
	default:
		err = ErrInvalid
	}
	if err != nil {
		problem(400, "Action rejected")
		return
	}
	w.WriteHeader(204)
}

// RunCLI uses the same current authorization and never accepts/stores secrets.
func RunCLI(ctx context.Context, s Store, a Actor, args []string, w io.Writer) error {
	if len(args) == 0 {
		return errors.New("usage: status|browse|enable|disable|revoke PEER|revoke-key PEER KID|consent PEER LISTING on|off|queue PEER LISTING EVENT_ID publish|revoke")
	}
	switch args[0] {
	case "approve":
		if len(args) != 7 {
			return ErrInvalid
		}
		key, e := base64.StdEncoding.DecodeString(args[4])
		from, e1 := time.Parse(time.RFC3339, args[5])
		until, e2 := time.Parse(time.RFC3339, args[6])
		if e != nil || e1 != nil || e2 != nil {
			return ErrInvalid
		}
		return s.ApprovePeer(ctx, a, PeerConfig{a.Network, args[1], args[2], true}, TrustedKey{args[3], key, from, until, false})
	case "retry":
		if len(args) != 3 {
			return ErrInvalid
		}
		return s.Retry(ctx, a, args[1], args[2])
	case "maintenance":
		if len(args) != 1 {
			return ErrInvalid
		}
		return s.Maintenance(ctx, a)
	case "withdraw-listing":
		if len(args) != 2 {
			return ErrInvalid
		}
		return s.WithdrawListing(ctx, a, args[1])
	case "depart":
		if len(args) != 2 {
			return ErrInvalid
		}
		return s.Depart(ctx, a, args[1])
	case "status":
		if len(args) != 1 {
			return ErrInvalid
		}
		p, q, e := s.Status(ctx, a)
		if e != nil {
			return e
		}
		return json.NewEncoder(w).Encode(struct {
			Peers []PeerStatus
			Queue []QueueStatus
		}{p, q})
	case "browse":
		if len(args) != 1 {
			return ErrInvalid
		}
		data, e := s.Browse(ctx, a)
		if e != nil {
			return e
		}
		return json.NewEncoder(w).Encode(data)
	case "enable", "disable":
		if len(args) != 1 {
			return ErrInvalid
		}
		return s.EnableNetwork(ctx, a, args[0] == "enable")
	case "revoke":
		if len(args) != 2 {
			return ErrInvalid
		}
		return s.RevokePeer(ctx, a, args[1])
	case "revoke-key":
		if len(args) != 3 {
			return ErrInvalid
		}
		return s.RevokeKey(ctx, a, args[1], args[2])
	case "consent":
		if len(args) != 4 || args[3] != "on" && args[3] != "off" {
			return ErrInvalid
		}
		return s.Consent(ctx, a, args[1], args[2], args[3] == "on")
	case "queue":
		if len(args) != 5 || args[4] != "publish" && args[4] != "revoke" {
			return ErrInvalid
		}
		_, e := s.Queue(ctx, a, args[1], args[2], args[3], args[4] == "revoke")
		return e
	default:
		return errors.New("unknown federation command")
	}
}

// ParsePage validates bounded public keyset values for a host browse route.
func ParsePage(r *http.Request) (string, string, int, error) {
	if len(r.URL.RawQuery) > 8192 {
		return "", "", 0, ErrInvalid
	}
	query, err := url.ParseQuery(r.URL.RawQuery)
	if err != nil {
		return "", "", 0, ErrInvalid
	}
	for key, values := range query {
		if (key != "after_peer" && key != "after_listing" && key != "limit") || len(values) != 1 {
			return "", "", 0, ErrInvalid
		}
	}
	limit := 20
	if raw := query.Get("limit"); raw != "" {
		n, e := strconv.Atoi(raw)
		if e != nil {
			return "", "", 0, ErrInvalid
		}
		limit = n
	}
	peer, listing := query.Get("after_peer"), query.Get("after_listing")
	if limit < 1 || limit > 100 || (peer != "" && !identifier(peer)) || (listing != "" && !identifier(listing)) {
		return "", "", 0, ErrInvalid
	}
	return peer, listing, limit, nil
}

// strictForm validates only body fields; query context never supplies mutation data.
func strictForm(r *http.Request, fields ...string) error {
	kind, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if err != nil || kind != "application/x-www-form-urlencoded" {
		return ErrInvalid
	}
	if err = r.ParseForm(); err != nil {
		return err
	}
	allowed := make(map[string]bool, len(fields))
	for _, field := range fields {
		allowed[field] = true
	}
	for field, values := range r.PostForm {
		if !allowed[field] || len(values) != 1 {
			return ErrInvalid
		}
	}
	return nil
}
