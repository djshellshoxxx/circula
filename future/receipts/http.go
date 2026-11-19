package receipts

import (
	"bytes"
	"context"
	"encoding/json"
	"html/template"
	"io"
	"net/http"
	"net/url"
	"strings"
)

// Host uses existing authentication, current membership, rate policy, and CSRF
// for cookie-authenticated POST. The body cap is applied before this hook runs.
type Host interface {
	Authorize(*http.Request, bool) (Actor, error)
	RateLimit(*http.Request, Actor) bool
	CSRFToken(*http.Request) string
}
type Handler struct {
	Service Service
	Host    Host
}
type CertificateHandler struct {
	Issuer Issuer
	Host   Host
}

var choice = template.Must(template.New("choice").Parse(`<!doctype html><html lang="en"><meta charset="utf-8"><title>Optional trade receipt</title><main><h1>Optional trade receipt</h1><p>Choose a server-generated download. No JavaScript is required. A receipt is optional and contains private transaction data. Downloaded copies cannot be remotely recalled.</p><form method="post"><input type="hidden" name="csrf_token" value="{{.CSRF}}"><input type="hidden" name="reference" value="{{.Reference}}"><input type="hidden" name="explicit" value="true"><label for="format">Format</label><select id="format" name="format"><option value="txt">Plain text</option><option value="json">JSON</option><option value="html">Printable HTML</option></select><button type="submit">Download receipt</button></form><p>Continue without a receipt by returning to transaction history. No certificate is generated.</p></main></html>`))
var certificateChoice = template.Must(template.New("certificate-choice").Parse(`<!doctype html><html lang="en"><meta charset="utf-8"><title>Optional trade certificate</title><main><h1>Optional trade certificate</h1><p>This separately requested certificate includes private receipt data and a server signature. No JavaScript is required. It verifies only the issuer's historical snapshot, not cash settlement, delivery, identity or current status. Downloaded copies cannot be remotely recalled.</p><form method="post"><input type="hidden" name="csrf_token" value="{{.CSRF}}"><input type="hidden" name="reference" value="{{.Reference}}"><input type="hidden" name="explicit" value="true"><input type="hidden" name="format" value="jws"><button type="submit">Download optional certificate</button></form><p>Continue without a certificate by returning to transaction history. Ordinary receipt downloads remain separate.</p></main></html>`))

type downloadInput struct {
	Reference string `json:"reference"`
	Format    string `json:"format"`
	Explicit  bool   `json:"explicit"`
}

func parseDownload(r *http.Request) (downloadInput, int) {
	var input downloadInput
	var err error
	switch strings.Split(r.Header.Get("Content-Type"), ";")[0] {
	case "application/json":
		raw, e := io.ReadAll(r.Body)
		if e != nil {
			return input, 400
		}
		canonical, e := canonicalJSON(raw, 65536)
		if e != nil {
			return input, 400
		}
		var keys map[string]json.RawMessage
		if json.Unmarshal(canonical, &keys) != nil {
			return input, 400
		}
		for key := range keys {
			if key != "reference" && key != "format" && key != "explicit" {
				return input, 400
			}
		}
		decoder := json.NewDecoder(bytes.NewReader(canonical))
		decoder.DisallowUnknownFields()
		err = decoder.Decode(&input)
	case "application/x-www-form-urlencoded":
		err = r.ParseForm()
		if err == nil {
			for key, values := range r.PostForm {
				if len(values) != 1 || (key != "reference" && key != "format" && key != "explicit" && key != "csrf_token") {
					return input, 400
				}
			}
			input.Reference = r.PostForm.Get("reference")
			input.Format = r.PostForm.Get("format")
			input.Explicit = r.PostForm.Get("explicit") == "true"
		}
	default:
		return input, 415
	}
	if err != nil || !input.Explicit || !safe(input.Reference) {
		return input, 400
	}
	return input, 0
}
func problem(w http.ResponseWriter, status int, title string) {
	w.Header().Set("Content-Type", "application/problem+json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]any{"type": "about:blank", "status": status, "title": title})
}

type downloadAdapter struct {
	enabled        bool
	networkEnabled func(context.Context, string) bool
	host           Host
	page           *template.Template
	formats        map[string]bool
	download       func(context.Context, Actor, downloadInput) ([]byte, string, string, error)
}

func (a downloadAdapter) serve(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "private, no-store")
	w.Header().Set("Pragma", "no-cache")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("Referrer-Policy", "no-referrer")
	w.Header().Set("Content-Security-Policy", "default-src 'none'; style-src 'unsafe-inline'; form-action 'self'; frame-ancestors 'none'; base-uri 'none'")
	if !a.enabled || a.host == nil || a.networkEnabled == nil {
		problem(w, 404, "Receipt unavailable")
		return
	}
	if r.Method != "GET" && r.Method != "POST" {
		w.Header().Set("Allow", "GET, POST")
		problem(w, 405, "Method not allowed")
		return
	}
	// Bound and validate URL input before a host hook can parse or merge it.
	if len(r.URL.RawQuery) > 8192 {
		problem(w, 400, "Invalid request")
		return
	}
	query, queryErr := url.ParseQuery(r.URL.RawQuery)
	if queryErr != nil {
		problem(w, 400, "Invalid request")
		return
	}
	for key, values := range query {
		if key != "reference" || len(values) != 1 || !safe(values[0]) {
			problem(w, 400, "Invalid request")
			return
		}
	}
	if r.Method == "POST" {
		r.Body = http.MaxBytesReader(w, r.Body, 65536)
		defer r.Body.Close()
		// ParseForm caches partial values but does not retain its first error.
		// Validate before a host CSRF hook can consume and cache a malformed form.
		if strings.Split(r.Header.Get("Content-Type"), ";")[0] == "application/x-www-form-urlencoded" {
			if err := r.ParseForm(); err != nil {
				problem(w, 400, "Invalid request")
				return
			}
		}
	}
	actor, e := a.host.Authorize(r, r.Method == "POST")
	if e != nil || !actorValid(actor) || (r.Header.Get("X-Network") != "" && r.Header.Get("X-Network") != actor.Network) || !a.networkEnabled(r.Context(), actor.Network) {
		problem(w, 404, "Receipt unavailable")
		return
	}
	if !a.host.RateLimit(r, actor) {
		problem(w, 429, "Too many requests")
		return
	}
	if r.Method == "GET" {
		ref := query.Get("reference")
		if !safe(ref) {
			problem(w, 400, "Invalid request")
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		a.page.Execute(w, struct{ CSRF, Reference string }{a.host.CSRFToken(r), ref})
		return
	}
	input, status := parseDownload(r)
	if status != 0 {
		if status == 415 {
			problem(w, 415, "Unsupported content type")
		} else {
			problem(w, 400, "Invalid request")
		}
		return
	}
	if query.Has("reference") && query.Get("reference") != input.Reference {
		problem(w, 400, "Invalid request")
		return
	}
	if !a.formats[input.Format] {
		problem(w, 400, "Invalid request")
		return
	}
	body, contentType, filename, e := a.download(r.Context(), actor, input)
	if e != nil {
		problem(w, 404, "Receipt unavailable")
		return
	}
	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Content-Disposition", `attachment; filename="`+filename+`"`)
	w.Write(body)
}
func (h Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	downloadAdapter{h.Service.Enabled, h.Service.NetworkEnabled, h.Host, choice, map[string]bool{"txt": true, "json": true, "html": true}, func(ctx context.Context, actor Actor, q downloadInput) ([]byte, string, string, error) {
		d, e := h.Service.Receipt(ctx, actor, DownloadRequest{q.Reference, q.Format, true})
		if e != nil {
			return nil, "", "", e
		}
		body, typ, e := d.Export(q.Format)
		return body, typ, "circula-receipt." + q.Format, e
	}}.serve(w, r)
}

// CertificateHandler must be registered separately by a future host. It never
// changes ordinary receipt routes or requires browser code.
func (h CertificateHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	networkEnabled := func(ctx context.Context, network string) bool {
		return h.Issuer.NetworkEnabled != nil && h.Issuer.Receipts.NetworkEnabled != nil && h.Issuer.NetworkEnabled(ctx, network) && h.Issuer.Receipts.NetworkEnabled(ctx, network)
	}
	downloadAdapter{h.Issuer.Enabled && h.Issuer.Receipts.Enabled, networkEnabled, h.Host, certificateChoice, map[string]bool{"jws": true}, func(ctx context.Context, actor Actor, q downloadInput) ([]byte, string, string, error) {
		token, e := h.Issuer.Issue(ctx, actor, CertificateRequest{q.Reference, true})
		return []byte(token), "application/jose", "circula-certificate.jws", e
	}}.serve(w, r)
}
