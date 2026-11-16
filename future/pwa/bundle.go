package pwa

import (
	"crypto/sha256"
	"embed"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"html/template"
	"net"
	"net/http"
	"net/url"
	"regexp"
	"strings"
)

//go:embed public/*
var files embed.FS

type Config struct {
	// Enabled must be explicitly set. No package init or automatic route mounting.
	Enabled           bool
	PublicOrigin      string
	BasePath          string
	AllowLoopbackHTTP bool
	// Relative application routes. Empty fields use synthetic conventional names;
	// the production adapter must supply its actual route mapping.
	Find, Trade, Balance, Inbox string
}
type asset struct {
	File   string `json:"file"`
	SHA256 string `json:"sha256"`
	Type   string `json:"type"`
}
type catalog struct {
	Version string  `json:"version"`
	Assets  []asset `json:"assets"`
}
type Bundle struct {
	config        Config
	assets        map[string]asset
	catalog       catalog
	worker        []byte
	style, client string
	icons         map[int]string
}

var basePattern = regexp.MustCompile(`^/(?:[A-Za-z0-9_-]+/)*$`)
var fingerprint = regexp.MustCompile(`\.[0-9a-f]{16}\.(?:css|js|png)$`)

func New(config Config) (*Bundle, error) {
	b := &Bundle{config: config, assets: map[string]asset{}, icons: map[int]string{}}
	if !config.Enabled {
		return b, nil
	}
	u, e := url.Parse(config.PublicOrigin)
	if e != nil || u.Host == "" || u.User != nil || u.Path != "" || u.RawQuery != "" || u.ForceQuery || u.Fragment != "" || strings.Contains(config.PublicOrigin, "#") {
		return nil, errors.New("invalid PWA origin")
	}
	loopback := u.Hostname() == "localhost"
	if ip := net.ParseIP(u.Hostname()); ip != nil {
		loopback = ip.IsLoopback()
	}
	if u.Scheme != "https" && !(u.Scheme == "http" && config.AllowLoopbackHTTP && loopback) {
		return nil, errors.New("PWA requires HTTPS")
	}
	if !basePattern.MatchString(config.BasePath) {
		return nil, errors.New("invalid PWA base path")
	}
	routes := []*string{&config.Find, &config.Trade, &config.Balance, &config.Inbox}
	defaults := []string{"find", "trade", "balance", "inbox"}
	for i, p := range routes {
		if *p == "" {
			*p = config.BasePath + defaults[i]
		}
		v, err := url.Parse(*p)
		if err != nil || v.IsAbs() || v.Host != "" || !strings.HasPrefix(v.Path, config.BasePath) || strings.Contains(v.Path, "..") || v.RawPath != "" || v.RawQuery != "" || v.Fragment != "" || strings.ContainsAny(*p, "?#") {
			return nil, errors.New("invalid PWA navigation")
		}
	}
	b.config = config
	raw, _ := files.ReadFile("public/catalog.json")
	if e = json.Unmarshal(raw, &b.catalog); e != nil {
		return nil, e
	}
	for _, a := range b.catalog.Assets {
		data, err := files.ReadFile("public/" + a.File)
		if err != nil {
			return nil, err
		}
		h := sha256.Sum256(data)
		if !fingerprint.MatchString(a.File) || a.SHA256 != base64.StdEncoding.EncodeToString(h[:]) || !strings.Contains(a.File, "."+hex.EncodeToString(h[:8])+".") {
			return nil, errors.New("invalid build integrity")
		}
		path := config.BasePath + "static/" + a.File
		b.assets[path] = a
		switch {
		case strings.HasPrefix(a.File, "style."):
			b.style = path
		case strings.HasPrefix(a.File, "client."):
			b.client = path
		case strings.HasPrefix(a.File, "icon-192."):
			b.icons[192] = path
		case strings.HasPrefix(a.File, "icon-512."):
			b.icons[512] = path
		}
	}
	if b.style == "" || b.client == "" || len(b.icons) != 2 {
		return nil, errors.New("incomplete build")
	}
	h := sha256.Sum256([]byte(config.BasePath))
	prefix := "circula-pwa-" + hex.EncodeToString(h[:6]) + "-"
	type entry struct {
		URL    string `json:"url"`
		SHA256 string `json:"sha256"`
		Type   string `json:"type"`
	}
	entries := []entry{}
	for _, a := range b.catalog.Assets {
		entries = append(entries, entry{config.BasePath + "static/" + a.File, a.SHA256, a.Type})
	}
	data, _ := json.Marshal(struct {
		Base   string  `json:"base"`
		Prefix string  `json:"cachePrefix"`
		Name   string  `json:"cacheName"`
		Assets []entry `json:"assets"`
	}{config.BasePath, prefix, prefix + b.catalog.Version, entries})
	worker, _ := files.ReadFile("public/worker.js")
	if strings.Count(string(worker), "__CIRCULA_CONFIG__") != 1 {
		return nil, errors.New("invalid worker build")
	}
	b.worker = []byte(strings.Replace(string(worker), "__CIRCULA_CONFIG__", string(data), 1))
	return b, nil
}
func (b *Bundle) ClientPath() string { return b.client }
func (b *Bundle) StylePath() string  { return b.style }
func (b *Bundle) BasePath() string   { return b.config.BasePath }

// Handler serves only module assets, manifest, worker and public installation help.
// It is not an authentication layer and handles no member/API/ledger routes.
func (b *Bundle) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	if !b.config.Enabled {
		http.NotFound(w, r)
		return
	}
	if r.Method != "GET" && r.Method != "HEAD" {
		w.Header().Set("Allow", "GET, HEAD")
		http.Error(w, "method not allowed", 405)
		return
	}
	expected, _ := url.Parse(b.config.PublicOrigin)
	if r.Host != expected.Host || r.URL.RawQuery != "" || r.URL.ForceQuery || r.URL.RawPath != "" {
		http.NotFound(w, r)
		return
	}
	path := r.URL.Path
	if a, ok := b.assets[path]; ok {
		w.Header().Set("Content-Type", a.Type)
		w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
		data, _ := files.ReadFile("public/" + a.File)
		if r.Method != "HEAD" {
			_, _ = w.Write(data)
		}
		return
	}
	switch path {
	case b.config.BasePath + "sw.js":
		w.Header().Set("Content-Type", "application/javascript")
		w.Header().Set("Service-Worker-Allowed", b.config.BasePath)
		if r.Method != "HEAD" {
			_, _ = w.Write(b.worker)
		}
	case b.config.BasePath + "manifest.webmanifest":
		w.Header().Set("Content-Type", "application/manifest+json")
		if r.Method != "HEAD" {
			_ = json.NewEncoder(w).Encode(map[string]any{"id": b.config.BasePath, "name": "Circula", "short_name": "Circula", "start_url": b.config.BasePath, "scope": b.config.BasePath, "display": "standalone", "background_color": "#ffffff", "theme_color": "#172333", "icons": []map[string]string{{"src": b.icons[192], "sizes": "192x192", "type": "image/png", "purpose": "any"}, {"src": b.icons[512], "sizes": "512x512", "type": "image/png", "purpose": "any"}}})
		}
	case b.config.BasePath + "install":
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Header().Set("Content-Security-Policy", CSP)
		if r.Method != "HEAD" {
			_ = help.Execute(w, b)
		}
	default:
		http.NotFound(w, r)
	}
}

// CSP is a proposed narrow policy for the isolated harness, not a root change.
const CSP = "default-src 'none'; script-src 'self'; style-src 'self'; img-src 'self'; manifest-src 'self'; worker-src 'self'; connect-src 'self'; base-uri 'none'; form-action 'self'; frame-ancestors 'none'"

func (b *Bundle) Navigation() []struct{ Label, URL string } {
	out := []struct{ Label, URL string }{}
	for i, u := range []string{b.config.Find, b.config.Trade, b.config.Balance, b.config.Inbox} {
		out = append(out, struct{ Label, URL string }{[]string{"Find", "Trade", "Balance", "Inbox"}[i], u})
	}
	return out
}

var help = template.Must(template.New("help").Parse(`<!doctype html><html lang="en"><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1,viewport-fit=cover"><meta name="theme-color" content="#172333"><meta name="circula-pwa" content="{{.BasePath}}"><title>Install Circula</title><link rel="manifest" href="{{.BasePath}}manifest.webmanifest"><link rel="stylesheet" href="{{.StylePath}}"><script defer src="{{.ClientPath}}"></script><body><a href="#main">Skip to content</a><nav aria-label="Main"><ul>{{range .Navigation}}<li><a href="{{.URL}}">{{.Label}}</a></li>{{end}}</ul></nav><main id="main"><h1>Install Circula</h1><p>Installation is optional. All online trading remains available in your browser without JavaScript.</p><h2>Android and desktop</h2><p>Open your browser menu and choose Install app or Add to Home screen if available.</p><h2>iPhone and iPad</h2><p>In Safari, use Share, then Add to Home Screen.</p><h2>Connection required</h2><p>Circula cannot show balances or send, accept, decline or reverse trades offline. Nothing is queued for later. Reconnect and review a new action explicitly.</p><p>Sign out before sharing your device. Installation does not keep you signed in indefinitely.</p><p data-session-notice hidden role="status"></p></main></body></html>`))
