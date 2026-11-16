// Synthetic loopback-only acceptance harness. Never deploy as authentication.
package main

import (
	"flag"
	"fmt"
	pwa "github.com/djshellshoxxx/circula/future/pwa"
	"html/template"
	"log"
	"net/http"
	"sync/atomic"
)

func main() {
	port := flag.Int("port", 18761, "loopback-only port")
	flag.Parse()
	origin := fmt.Sprintf("http://127.0.0.1:%d", *port)
	b, e := pwa.New(pwa.Config{Enabled: true, PublicOrigin: origin, BasePath: "/app/", AllowLoopbackHTTP: true})
	if e != nil {
		log.Fatal(e)
	}
	var trades atomic.Int64
	page := template.Must(template.New("page").Parse(`<!doctype html><html lang="en"><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1"><meta name="circula-pwa" content="/app/"><title>Circula synthetic practice</title><link rel="manifest" href="/app/manifest.webmanifest"><link rel="stylesheet" href="{{.Style}}"><script defer src="{{.Client}}"></script><body><a href="#main">Skip to content</a><p>Synthetic PWA harness — no real accounts or credits.</p><nav aria-label="Main"><a href="/app/find">Find</a> <a href="/app/trade">Trade</a> <a href="/app/balance">Balance</a> <a href="/app/inbox">Inbox</a> <a href="/app/install">Install help</a></nav><main id="main"><h1>Circula practice</h1><p data-session-notice hidden role="status"></p>{{if .Member}}<section data-private><h2>Member {{.Member}}</h2><p>Private synthetic balance for {{.Member}}: 100</p><form method="post" action="/app/trade"><input type="hidden" name="intent" value="synthetic"><button>Send synthetic trade</button></form></section><form data-signout method="post" action="/app/logout"><button>Sign out</button></form>{{else}}<form method="post" action="/app/login"><label>Practice member <select name="member"><option>A</option><option>B</option></select></label><button>Sign in</button></form>{{end}}</main></body></html>`))
	http.HandleFunc("/app/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-store")
		w.Header().Set("Content-Security-Policy", pwa.CSP)
		w.Header().Set("X-Content-Type-Options", "nosniff")
		member := ""
		if c, e := r.Cookie("demo"); e == nil && (c.Value == "A" || c.Value == "B") {
			member = c.Value
		}
		if r.Method == "POST" && r.Header.Get("Origin") != origin {
			http.Error(w, "wrong origin", 403)
			return
		}
		switch r.URL.Path {
		case "/app/login":
			if r.Method == "POST" {
				_ = r.ParseForm()
				v := r.Form.Get("member")
				if v != "A" && v != "B" {
					http.Error(w, "invalid", 400)
					return
				}
				http.SetCookie(w, &http.Cookie{Name: "demo", Value: v, Path: "/app/", HttpOnly: true, SameSite: http.SameSiteStrictMode})
				http.Redirect(w, r, "/app/", 303)
				return
			}
		case "/app/logout":
			if r.Method != "POST" {
				http.Error(w, "POST required", 405)
				return
			}
			http.SetCookie(w, &http.Cookie{Name: "demo", Value: "", Path: "/app/", MaxAge: -1, HttpOnly: true, SameSite: http.SameSiteStrictMode})
			http.Redirect(w, r, "/app/install", 303)
			return
		case "/app/counts":
			w.Header().Set("Content-Type", "text/plain")
			fmt.Fprint(w, trades.Load())
			return
		case "/app/trade":
			if r.Method == "POST" {
				if member == "" {
					http.Error(w, "sign in", 401)
					return
				}
				trades.Add(1)
				http.Redirect(w, r, "/app/", 303)
				return
			}
		case "/app/", "/app/find", "/app/balance", "/app/inbox":
		default:
			b.ServeHTTP(w, r)
			return
		}
		if r.Method != "GET" {
			http.Error(w, "method", 405)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_ = page.Execute(w, struct{ Member, Style, Client string }{member, b.StylePath(), b.ClientPath()})
	})
	log.Printf("Synthetic PWA harness at %s/app/", origin)
	log.Fatal(http.ListenAndServe(fmt.Sprintf("127.0.0.1:%d", *port), nil))
}
