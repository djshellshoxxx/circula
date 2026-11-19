package analytics

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

// HostAuth resolves current identity and membership. Source.Read separately
// enforces admin privilege for aggregate snapshots on every request.
type HostAuth func(*http.Request) (Actor, error)
type Handler struct {
	Service DashboardService
	Auth    HostAuth
}

func (h Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("Content-Security-Policy", "default-src 'none'; form-action 'self'; frame-ancestors 'none'")
	problem := func(code int, title string) {
		w.Header().Set("Content-Type", "application/problem+json")
		w.WriteHeader(code)
		json.NewEncoder(w).Encode(map[string]any{"type": "about:blank", "status": code, "title": title})
	}
	if r.Method != "GET" {
		w.Header().Set("Allow", "GET")
		problem(405, "Method not allowed")
		return
	}
	if h.Auth == nil {
		problem(503, "Analytics unavailable")
		return
	}
	// Bound work before host callbacks can parse attacker-controlled query data.
	if len(r.URL.RawQuery) > 2048 {
		problem(400, "Query too large")
		return
	}
	a, e := h.Auth(r)
	if e != nil || a.Network == "" || a.Member == "" || len(r.Header.Values("X-Network")) > 1 || r.Header.Get("X-Network") != "" && r.Header.Get("X-Network") != a.Network {
		problem(403, "Access denied")
		return
	}
	query, e := url.ParseQuery(r.URL.RawQuery)
	if e != nil {
		problem(400, "Invalid query")
		return
	}
	for key, values := range query {
		if len(values) != 1 {
			problem(400, "Duplicate query field")
			return
		}
		switch key {
		case "period", "count", "view", "format":
		default:
			problem(400, "Unsupported analytics slice")
			return
		}
	}
	format := query.Get("format")
	if format == "" && strings.Contains(r.Header.Get("Accept"), "text/html") {
		format = "html"
	}
	if format != "" && format != "json" && format != "html" && format != "csv" {
		problem(400, "Invalid format")
		return
	}
	kind := query.Get("period")
	if kind == "" {
		kind = h.Service.Granularity
		if kind == "" {
			kind = "month"
		}
	}
	count := 1
	if raw := query.Get("count"); raw != "" {
		count, e = strconv.Atoi(raw)
		if e != nil {
			problem(400, "Invalid period count")
			return
		}
	}
	if count < 1 || count > 12 || (kind != "week" && kind != "month") {
		problem(400, "Invalid closed period request")
		return
	}
	view := query.Get("view")
	if view == "own" {
		data, err := h.Service.Own(r.Context(), a, kind, count)
		if err != nil {
			problem(403, "Trend unavailable")
			return
		}
		switch format {
		case "html":
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			RenderOwn(w, data)
		case "csv":
			w.Header().Set("Content-Type", "text/csv; charset=utf-8")
			ExportOwn(w, data)
		default:
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(data)
		}
		return
	}
	if view != "" && view != "admin" {
		problem(400, "Invalid view")
		return
	}
	data, err := h.Service.Admin(r.Context(), a, kind, count)
	if err != nil {
		problem(403, "Aggregate unavailable")
		return
	}
	switch format {
	case "html":
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		RenderDashboard(w, data)
	case "csv":
		w.Header().Set("Content-Type", "text/csv; charset=utf-8")
		Export(w, data)
	case "", "json":
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(data)
	default:
		problem(400, "Invalid format")
	}
}

// RunCLI delegates to the same authorized read-only source. Credentials are
// provided by the host, never accepted as command arguments or stored here.
func RunCLI(ctx context.Context, s DashboardService, a Actor, args []string, w io.Writer) error {
	if len(args) != 3 {
		return errors.New("usage: admin|admin-json|own week|month PERIOD_COUNT")
	}
	count, e := strconv.Atoi(args[2])
	if e != nil {
		return e
	}
	switch args[0] {
	case "admin":
		data, err := s.Admin(ctx, a, args[1], count)
		if err != nil {
			return err
		}
		return Export(w, data)
	case "admin-json":
		data, err := s.Admin(ctx, a, args[1], count)
		if err != nil {
			return err
		}
		return json.NewEncoder(w).Encode(data)
	case "own":
		data, err := s.Own(ctx, a, args[1], count)
		if err != nil {
			return err
		}
		return json.NewEncoder(w).Encode(data)
	default:
		return errors.New("unknown analytics command")
	}
}
