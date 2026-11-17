package web_test

import (
	"net"
	"net/http"
	"os"
	"strconv"
	"testing"
	"time"
)

// TestServeForBrowser is a manual-gate helper, skipped unless
// FAVORITES_BROWSER_ADDR is set. It serves the handler over synthetic data with
// the test cookie sessions ("sid" = alice | bob | alice-n2) so
// browser/check.mjs can drive a real browser. It is test code, not a
// constructor: production code has no demo authentication.
func TestServeForBrowser(t *testing.T) {
	addr := os.Getenv("FAVORITES_BROWSER_ADDR")
	if addr == "" {
		t.Skip("set FAVORITES_BROWSER_ADDR to serve for browser checks")
	}
	e := newEnv(t)
	for _, id := range []string{"bob", "carol", "dave"} {
		e.add(t, "alice", id, "Nick "+id)
	}
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		t.Fatal(err)
	}
	secs, _ := strconv.Atoi(os.Getenv("FAVORITES_BROWSER_SECONDS"))
	if secs <= 0 {
		secs = 120
	}
	srv := &http.Server{Handler: e.h, ReadHeaderTimeout: 5 * time.Second}
	go func() { _ = srv.Serve(ln) }()
	time.Sleep(time.Duration(secs) * time.Second)
	_ = srv.Close()
}
