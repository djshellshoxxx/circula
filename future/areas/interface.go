package areas

import (
	"context"
	"encoding/json"
	"errors"
	"html/template"
	"io"
	"mime"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

// HostAuth must reject revoked/suspended sessions and verify CSRF for cookie
// mutations. Domain policy is checked again inside the database transaction.
type HostAuth func(*http.Request, bool) (Identity, error)
type Web struct {
	Service   Service
	Auth      HostAuth
	CSRFToken func(*http.Request) string
	CursorKey []byte
}
type page struct {
	Areas      []Record
	CSRF       string
	Assignment Assignment
}

var areaPage = template.Must(template.New("areas").Parse(`<!doctype html><html lang="en"><meta charset="utf-8"><title>Community areas</title><main><h1>Community areas</h1><p>Choosing an area is optional. No GPS permission or exact address is needed.</p><ul>{{range .Areas}}<li>{{.Label}}</li>{{else}}<li>No areas available.</li>{{end}}</ul><form method="get" action="filter"><label for="filter-area">Filter by area</label><select id="filter-area" name="area">{{range .Areas}}<option value="{{.ID}}">{{.Label}}</option>{{end}}</select><button>Show listings</button></form><form method="get" action="catalog"><label for="lookup-listing">Load your listing area</label><input id="lookup-listing" name="listing" required maxlength="128"><button>Load current selection</button></form><form method="post" action="assign"><input type="hidden" name="csrf" value="{{.CSRF}}"><label for="listing">Your listing ID</label><input id="listing" name="listing" value="{{.Assignment.Listing}}" required maxlength="128"><label for="selected-area">Optional area</label><select id="selected-area" name="area"><option value="">No area</option>{{range .Areas}}<option value="{{.ID}}"{{if eq $.Assignment.AreaID .ID}} selected{{end}}>{{.Label}}</option>{{end}}</select><label for="revision">Current association revision (0 for new)</label><input id="revision" name="revision" type="number" min="0" value="{{.Assignment.Revision}}" required><label for="key">Request key (keep for retries)</label><input id="key" name="key" required maxlength="128"><button>Save area</button></form></main></html>`))

func Render(w io.Writer, rows []Record, csrf string) error {
	return areaPage.Execute(w, page{Areas: rows, CSRF: csrf})
}

var adminPage = template.Must(template.New("admin").Parse(`<!doctype html><html lang="en"><meta charset="utf-8"><title>Manage community areas</title><main><h1>Manage community areas</h1><p>Archiving removes area associations from public results. Restoring does not restore old listing selections.</p>{{range .Areas}}<form method="post" action="admin"><input type="hidden" name="csrf" value="{{$.CSRF}}"><input type="hidden" name="id" value="{{.ID}}"><input type="hidden" name="revision" value="{{.Revision}}"><label>Area label<input name="label" value="{{.Label}}" required maxlength="100"></label><label>Visible<select name="visible"><option>{{.Visible}}</option><option>true</option><option>false</option></select></label><label>Archived<select name="archived"><option>{{.Archived}}</option><option>true</option><option>false</option></select></label><label>Request key<input name="key" required maxlength="128"></label><button>Save {{.Label}}</button></form>{{end}}<h2>Federation sharing</h2><p>Named areas shared: {{.Enabled}}. Current policy revision: {{.Revision}}. Listing associations are never exported.</p><form method="post" action="sharing"><input type="hidden" name="csrf" value="{{.CSRF}}"><input type="hidden" name="revision" value="{{.Revision}}"><label>Share named areas<select name="enabled"><option>false</option><option>true</option></select></label><label>Request key<input name="key" required></label><button>Save sharing policy</button></form><form method="post" action="disable-sharing"><input type="hidden" name="csrf" value="{{.CSRF}}"><input type="hidden" name="revision" value="0"><button>Stop sharing immediately</button></form><h2>Create an area</h2><form method="post" action="admin"><input type="hidden" name="csrf" value="{{.CSRF}}"><input type="hidden" name="revision" value="0"><input type="hidden" name="visible" value="true"><input type="hidden" name="archived" value="false"><label>Area ID<input name="id" required maxlength="80"></label><label>Area label<input name="label" required maxlength="100"></label><label>Request key<input name="key" required maxlength="128"></label><button>Create area</button></form></main></html>`))

var listingPageTemplate = template.Must(template.New("listings").Parse(`<!doctype html><html lang="en"><meta charset="utf-8"><title>Listings by area</title><main><h1>Listings in selected area</h1><ul>{{range .Page.Listings}}<li><code>{{.}}</code></li>{{else}}<li>No visible listings in this page.</li>{{end}}</ul>{{if .Page.Next}}<a href="filter?area={{.Area}}&amp;cursor={{.Page.Next}}">Next page</a>{{end}}<p><a href="catalog">Choose another area</a></p></main></html>`))

func (w Web) ServeHTTP(out http.ResponseWriter, r *http.Request) {
	out.Header().Set("Cache-Control", "no-store")
	out.Header().Set("X-Content-Type-Options", "nosniff")
	out.Header().Set("Content-Security-Policy", "default-src 'none'; form-action 'self'; base-uri 'none'; frame-ancestors 'none'")
	problem := func(code int, title string) {
		out.Header().Set("Content-Type", "application/problem+json")
		out.WriteHeader(code)
		json.NewEncoder(out).Encode(map[string]any{"type": "about:blank", "status": code, "title": title})
	}
	if w.Auth == nil {
		problem(503, "Areas unavailable")
		return
	}
	if r.Method != "GET" && r.Method != "POST" {
		out.Header().Set("Allow", "GET, POST")
		problem(405, "Method not allowed")
		return
	}
	if len(r.URL.RawQuery) > 8192 {
		problem(400, "Invalid query")
		return
	}
	if r.URL.IsAbs() || r.URL.Host != "" || strings.Contains(r.URL.Path, "\\") || strings.Contains(r.URL.EscapedPath(), "%") || strings.Contains(r.URL.Path, "//") || (r.URL.Path != "/" && strings.HasSuffix(r.URL.Path, "/")) {
		problem(400, "Invalid local route")
		return
	}
	path := strings.Trim(r.URL.Path, "/")
	query, queryErr := url.ParseQuery(r.URL.RawQuery)
	if queryErr != nil {
		problem(400, "Invalid query")
		return
	}
	for key, values := range query {
		allowed := r.Method == "GET" && (((path == "" || path == "catalog" || path == "assignment") && key == "listing") || (path == "filter" && (key == "area" || key == "cursor")))
		if !allowed || len(values) != 1 {
			problem(400, "Invalid query")
			return
		}
	}
	if r.Method == "POST" {
		media, params, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
		if err != nil || (media != "application/json" && media != "application/x-www-form-urlencoded") || (params["charset"] != "" && !strings.EqualFold(params["charset"], "utf-8")) {
			problem(415, "Unsupported request type")
			return
		}
		r.Body = http.MaxBytesReader(out, r.Body, 65536)
		defer r.Body.Close()
		// Parse before host CSRF helpers can cache a partial form and discard errors.
		if media == "application/x-www-form-urlencoded" {
			if err := r.ParseForm(); err != nil {
				problem(400, "Invalid form")
				return
			}
			allowed := map[string]bool{"csrf": true, "listing": true, "area": true, "revision": true, "key": true, "id": true, "label": true, "visible": true, "archived": true, "enabled": true}
			for key, values := range r.PostForm {
				if !allowed[key] || len(values) != 1 {
					problem(400, "Invalid form fields")
					return
				}
			}
		}
	}
	a, e := w.Auth(r, r.Method == "POST")
	if e != nil || !validIdentity(a) || r.Header.Get("X-Network") != "" && r.Header.Get("X-Network") != a.Network {
		problem(403, "Access denied")
		return
	}
	if r.Method == "GET" {
		token := ""
		formPage := path == "admin" || ((path == "" || path == "catalog") && strings.Contains(r.Header.Get("Accept"), "text/html"))
		if formPage {
			if w.CSRFToken == nil {
				problem(503, "Form security unavailable")
				return
			}
			token = w.CSRFToken(r)
			current, err := w.Auth(r, false)
			if err != nil || current != a || r.Context().Err() != nil {
				problem(403, "Access denied")
				return
			}
		}
		switch path {
		case "", "catalog":
			rows, err := w.Service.Catalog(r.Context(), a)
			if err != nil {
				problem(403, "Catalog unavailable")
				return
			}
			if strings.Contains(r.Header.Get("Accept"), "text/html") {
				if w.CSRFToken == nil {
					problem(503, "Form security unavailable")
					return
				}
				out.Header().Set("Content-Type", "text/html; charset=utf-8")
				data := page{Areas: rows, CSRF: token}
				if listing := query.Get("listing"); listing != "" {
					data.Assignment, err = w.Service.Current(r.Context(), a, listing)
					if err != nil {
						problem(403, "Listing unavailable")
						return
					}
				}
				areaPage.Execute(out, data)
				return
			}
			out.Header().Set("Content-Type", "application/json")
			json.NewEncoder(out).Encode(rows)
		case "admin":
			rows, err := w.Service.AdminCatalog(r.Context(), a)
			if err != nil {
				problem(403, "Admin catalog unavailable")
				return
			}
			if w.CSRFToken == nil {
				problem(503, "Form security unavailable")
				return
			}
			out.Header().Set("Content-Type", "text/html; charset=utf-8")
			sharing, err := w.Service.GetSharing(r.Context(), a)
			if err != nil {
				problem(403, "Sharing policy unavailable")
				return
			}
			adminPage.Execute(out, struct {
				page
				Sharing
			}{page{Areas: rows, CSRF: token}, sharing})
		case "assignment":
			data, err := w.Service.Current(r.Context(), a, query.Get("listing"))
			if err != nil {
				problem(403, "Listing unavailable")
				return
			}
			out.Header().Set("Content-Type", "application/json")
			json.NewEncoder(out).Encode(data)
		case "filter":
			ids, err := w.Service.FilterPage(r.Context(), a, query.Get("area"), query.Get("cursor"), 100, w.CursorKey)
			if err != nil {
				problem(403, "Area unavailable")
				return
			}
			if strings.Contains(r.Header.Get("Accept"), "text/html") {
				out.Header().Set("Content-Type", "text/html; charset=utf-8")
				listingPageTemplate.Execute(out, struct {
					Area string
					Page ListingPage
				}{query.Get("area"), ids})
				return
			}
			out.Header().Set("Content-Type", "application/json")
			json.NewEncoder(out).Encode(ids)
		default:
			problem(404, "Not found")
		}
		return
	}

	var result any
	if strings.HasPrefix(r.Header.Get("Content-Type"), "application/x-www-form-urlencoded") {
		revision, err := strconv.ParseInt(r.PostForm.Get("revision"), 10, 64)
		if err != nil {
			problem(400, "Invalid revision")
			return
		}
		switch path {
		case "assign":
			result, e = w.Service.Assign(r.Context(), a, r.PostForm.Get("key"), Assign{r.PostForm.Get("listing"), r.PostForm.Get("area"), revision})
		case "disable-sharing":
			result, e = w.Service.DisableSharing(r.Context(), a)
		case "sharing":
			enabled, err := strconv.ParseBool(r.PostForm.Get("enabled"))
			if err != nil {
				problem(400, "Invalid sharing policy")
				return
			}
			result, e = w.Service.SetSharing(r.Context(), a, r.PostForm.Get("key"), enabled, revision)
		case "admin":
			visible, err := strconv.ParseBool(r.PostForm.Get("visible"))
			if err != nil {
				problem(400, "Invalid visibility")
				return
			}
			archived, err := strconv.ParseBool(r.PostForm.Get("archived"))
			if err != nil {
				problem(400, "Invalid archival state")
				return
			}
			result, e = w.Service.Edit(r.Context(), a, r.PostForm.Get("key"), Edit{r.PostForm.Get("id"), r.PostForm.Get("label"), visible, archived, revision})
		default:
			problem(404, "Not found")
			return
		}
	} else {
		decode := func(value any) error {
			data, err := io.ReadAll(r.Body)
			if err != nil {
				return err
			}
			return strictJSON(data, value)
		}
		key := r.Header.Get("Idempotency-Key")
		switch path {
		case "disable-sharing":
			var input struct{}
			e = decode(&input)
			if e == nil {
				result, e = w.Service.DisableSharing(r.Context(), a)
			}
		case "sharing":
			var input Sharing
			e = decode(&input)
			if e == nil {
				result, e = w.Service.SetSharing(r.Context(), a, key, input.Enabled, input.Revision)
			}
		case "admin":
			var input Edit
			e = decode(&input)
			if e == nil {
				result, e = w.Service.Edit(r.Context(), a, key, input)
			}
		case "assign":
			var input Assign
			e = decode(&input)
			if e == nil {
				result, e = w.Service.Assign(r.Context(), a, key, input)
			}
		default:
			problem(404, "Not found")
			return
		}
	}
	if e != nil {
		code := 400
		var tooLarge *http.MaxBytesError
		if errors.As(e, &tooLarge) {
			code = 413
		}
		if errors.Is(e, ErrDenied) {
			code = 403
		}
		if errors.Is(e, ErrConflict) {
			code = 409
		}
		problem(code, "Area operation rejected")
		return
	}
	out.Header().Set("Content-Type", "application/json")
	json.NewEncoder(out).Encode(result)
}

// RunCLI receives the same host identity/service as HTTP, never a second login.
func RunCLI(ctx context.Context, s Service, a Identity, args []string, out io.Writer) error {
	if len(args) < 1 {
		return errors.New("usage: catalog | filter AREA | assign LISTING AREA REVISION KEY | edit ID LABEL REVISION VISIBLE ARCHIVED KEY")
	}
	var result any
	var err error
	switch args[0] {
	case "unshare":
		if len(args) != 1 {
			return errors.New("unexpected arguments")
		}
		result, err = s.DisableSharing(ctx, a)
	case "current":
		if len(args) != 2 {
			return errors.New("current requires listing")
		}
		result, err = s.Current(ctx, a, args[1])
	case "sharing":
		if len(args) != 1 {
			return errors.New("unexpected arguments")
		}
		result, err = s.GetSharing(ctx, a)
	case "share":
		if len(args) != 4 {
			return errors.New("share requires enabled revision key")
		}
		enabled, e := strconv.ParseBool(args[1])
		if e != nil {
			return e
		}
		revision, e := strconv.ParseInt(args[2], 10, 64)
		if e != nil {
			return e
		}
		result, err = s.SetSharing(ctx, a, args[3], enabled, revision)
	case "catalog":
		if len(args) != 1 {
			return errors.New("unexpected arguments")
		}
		result, err = s.Catalog(ctx, a)
	case "filter":
		if len(args) != 2 {
			return errors.New("filter requires area")
		}
		result, err = s.Filter(ctx, a, args[1])
	case "assign":
		if len(args) != 5 {
			return errors.New("assign requires listing area revision key")
		}
		revision, e := strconv.ParseInt(args[3], 10, 64)
		if e != nil {
			return e
		}
		result, err = s.Assign(ctx, a, args[4], Assign{args[1], args[2], revision})
	case "edit":
		if len(args) != 7 {
			return errors.New("edit requires id label revision visible archived key")
		}
		revision, e := strconv.ParseInt(args[3], 10, 64)
		if e != nil {
			return e
		}
		visible, e := strconv.ParseBool(args[4])
		if e != nil {
			return e
		}
		archived, e := strconv.ParseBool(args[5])
		if e != nil {
			return e
		}
		result, err = s.Edit(ctx, a, args[6], Edit{args[1], args[2], visible, archived, revision})
	default:
		return errors.New("unknown areas command")
	}
	if err != nil {
		return err
	}
	return json.NewEncoder(out).Encode(result)
}

// RunPagedCLI uses the same opaque paging contract as HTTP. Cursor key belongs
// to the host secret store, never command arguments or output.
func RunPagedCLI(ctx context.Context, s Service, a Identity, key []byte, args []string, out io.Writer) error {
	if len(args) > 0 && args[0] == "filter" {
		if len(args) < 2 || len(args) > 3 {
			return errors.New("filter requires area and optional cursor")
		}
		cursor := ""
		if len(args) == 3 {
			cursor = args[2]
		}
		result, e := s.FilterPage(ctx, a, args[1], cursor, 100, key)
		if e != nil {
			return e
		}
		return json.NewEncoder(out).Encode(result)
	}
	return RunCLI(ctx, s, a, args, out)
}
