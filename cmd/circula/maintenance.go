package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

// Maintenance mode (OPS-024). While active, reads (GET, HEAD, OPTIONS) on the
// web UI and the API keep working; every other method (including sign-in and
// sign-out) answers 503 with Retry-After and a friendly page (HTML) or problem
// document (API).
// /healthz, /readyz and /metrics keep working so monitoring and load balancers
// still see the instance. Two switches, either one turns it on:
//
//   - the `maintenance` setting (start in maintenance mode), and
//   - the file <database>.maintenance, created and removed by
//     `circula maintenance on|off` while the server runs, no restart needed.
//     The file may hold a Retry-After in seconds.
const (
	defaultMaintenanceRetry = 300
	maintenanceStatTTL      = time.Second
)

// maintenancePath returns the toggle file for a database path.
func maintenancePath(database string) string { return database + ".maintenance" }

// maintenance reports whether maintenance mode is active.
type maintenance struct {
	cfgOn bool
	path  string

	mu      sync.Mutex
	checked time.Time
	fileOn  bool
	retry   int
	now     func() time.Time
}

func newMaintenance(cfg *Config) *maintenance {
	return &maintenance{cfgOn: cfg.Maintenance, path: maintenancePath(cfg.Database), now: time.Now}
}

// state returns whether maintenance is active and the Retry-After in seconds.
// The file is re-checked at most once per second.
func (m *maintenance) state() (bool, int) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if t := m.now(); t.Sub(m.checked) >= maintenanceStatTTL {
		m.checked = t
		m.fileOn, m.retry = readMaintenanceFile(m.path)
	}
	retry := m.retry
	if retry <= 0 {
		retry = defaultMaintenanceRetry
	}
	return m.cfgOn || m.fileOn, retry
}

func readMaintenanceFile(path string) (bool, int) {
	b, err := os.ReadFile(path)
	if err != nil {
		return !errors.Is(err, fs.ErrNotExist), 0 // present but unreadable: fail closed
	}
	n, _ := strconv.Atoi(strings.TrimSpace(string(b)))
	if n < 0 || n > 86400 {
		n = 0
	}
	return true, n
}

// readOnlyMethod reports whether a method only reads. Reads keep working in
// maintenance mode (OPS-024); everything else, sign-in and sign-out included,
// is refused.
func readOnlyMethod(method string) bool {
	return method == http.MethodGet || method == http.MethodHead || method == http.MethodOptions
}

// gate wraps a handler so mutating requests answer 503 while maintenance is active.
func (m *maintenance) gate(next http.Handler, api bool) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		on, retry := m.state()
		if !on || readOnlyMethod(r.Method) {
			next.ServeHTTP(w, r)
			return
		}
		w.Header().Set("Retry-After", strconv.Itoa(retry))
		w.Header().Set("Cache-Control", "no-store")
		if api {
			w.Header().Set("Content-Type", "application/problem+json; charset=utf-8")
			w.WriteHeader(http.StatusServiceUnavailable)
			_ = json.NewEncoder(w).Encode(map[string]any{
				"type": "urn:circula:problem:maintenance", "title": "Maintenance", "status": 503,
				"code": "maintenance", "detail": "Circula is in maintenance mode. Nothing was changed; retry after the Retry-After interval.",
			})
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusServiceUnavailable)
		if r.Method != http.MethodHead {
			_, _ = io.WriteString(w, maintenancePage)
		}
	})
}

const maintenancePage = `<!doctype html>
<html lang="en"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width, initial-scale=1">
<title>Back soon</title>
<style>
:root{color-scheme:light dark}
body{font:16px/1.5 system-ui,sans-serif;margin:0;padding:16px;display:grid;place-items:center;min-height:100vh}
main{max-width:32rem}
h1{font-size:1.4rem;margin:0 0 .5rem}
</style></head><body><main>
<h1>Back soon</h1>
<p>Circula is down for planned maintenance. Your balances and history are safe, and nothing you were doing was changed.</p>
<p>Please try again in a few minutes.</p>
</main></body></html>
`

// cmdMaintenance toggles maintenance mode on a running or stopped instance:
// circula maintenance [-retry-after SECONDS] on | off | status.
func cmdMaintenance(ctx context.Context, args []string, stdout, stderr io.Writer) int {
	_ = ctx
	var retry int
	cfg, rest, code, done := resolveFlags("maintenance", args, stderr, func(fs *flag.FlagSet) {
		fs.IntVar(&retry, "retry-after", defaultMaintenanceRetry, "Retry-After seconds sent to clients (with on)")
	})
	if done {
		return code
	}
	if len(rest) != 1 || (rest[0] != "on" && rest[0] != "off" && rest[0] != "status") {
		fmt.Fprintln(stderr, "usage: circula maintenance [flags] on|off|status")
		return exitUsage
	}
	path := maintenancePath(cfg.Database)
	switch rest[0] {
	case "on":
		if retry < 1 || retry > 86400 {
			fmt.Fprintln(stderr, "circula: -retry-after must be 1 to 86400 seconds")
			return exitUsage
		}
		if err := os.WriteFile(path, []byte(strconv.Itoa(retry)+"\n"), 0o600); err != nil {
			fmt.Fprintf(stderr, "circula: maintenance: %v\n", err)
			return exitFailure
		}
		fmt.Fprintf(stdout, "maintenance mode on (%s); a running server applies it within a second\n", path)
	case "off":
		if err := os.Remove(path); err != nil && !errors.Is(err, fs.ErrNotExist) {
			fmt.Fprintf(stderr, "circula: maintenance: %v\n", err)
			return exitFailure
		}
		fmt.Fprintln(stdout, "maintenance mode off (the file switch; the maintenance setting is unchanged)")
		if cfg.Maintenance {
			fmt.Fprintln(stdout, "note: the maintenance setting is true, so a server using this configuration stays in maintenance mode")
		}
	default:
		on, _ := readMaintenanceFile(path)
		fmt.Fprintf(stdout, "file switch: %t\nsetting: %t\n", on, cfg.Maintenance)
	}
	return exitOK
}
