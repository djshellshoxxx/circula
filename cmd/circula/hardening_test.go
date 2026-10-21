package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/netip"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/djshellshoxxx/circula/internal/httpx"
	"github.com/djshellshoxxx/circula/internal/store"
)

func discardLogger() *slog.Logger { return slog.New(slog.NewTextHandler(io.Discard, nil)) }

func sha(t *testing.T, path string) string {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}

func writeSecret(t *testing.T, content string, mode os.FileMode) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), "secret")
	if err := os.WriteFile(p, []byte(content), mode); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(p, mode); err != nil {
		t.Fatal(err)
	}
	return p
}

// SEC-022: plain http on a non-loopback address is refused without the explicit flag.
func TestPlainHTTPRefusedUnlessExplicit(t *testing.T) {
	cases := []struct {
		name, base, listen, allow string
		ok                        bool
	}{
		{"https public", "https://circula.example.org", "0.0.0.0:8080", "", true},
		{"http public host", "http://circula.example.org", "127.0.0.1:8080", "", false},
		{"http public host, listen all", "http://circula.example.org", "0.0.0.0:8080", "", false},
		{"http public host, explicit", "http://circula.example.org", "0.0.0.0:8080", "true", true},
		{"http loopback", "http://localhost:8080", "127.0.0.1:8080", "", true},
		{"http loopback ip", "http://127.0.0.1:8080", "[::1]:8080", "", true},
		{"http loopback url, listen all", "http://localhost:8080", "0.0.0.0:8080", "", false},
	}
	for _, c := range cases {
		extra := map[string]string{envName("public-base-url"): c.base, envName("listen"): c.listen}
		if c.allow != "" {
			extra[envName("allow-insecure-http")] = c.allow
		}
		_, _, err := resolveWith(t, nil, baseEnv(t, extra))
		if c.ok && err != nil {
			t.Errorf("%s: refused: %v", c.name, err)
		}
		if !c.ok {
			if err == nil || !strings.Contains(err.Error(), "public-base-url") || !strings.Contains(err.Error(), "SEC-022") {
				t.Errorf("%s: accepted or unclear: %v", c.name, err)
			}
		}
	}
	// -check-config exits non-zero for the refused combination (startup refusal).
	var out, errOut bytes.Buffer
	code := run([]string{"serve", "-check-config", "-public-base-url", "http://circula.example.org", "-database", filepath.Join(t.TempDir(), "x.db")}, &out, &errOut)
	if code != exitUsage {
		t.Errorf("serve with http public URL: code %d", code)
	}
}

func TestTrustedProxiesSetting(t *testing.T) {
	cfg, _, err := resolveWith(t, nil, baseEnv(t, nil))
	if err != nil || len(cfg.TrustedProxies) != 0 {
		t.Fatalf("default: %v %v", cfg.TrustedProxies, err)
	}
	cfg, _, err = resolveWith(t, nil, baseEnv(t, map[string]string{envName("trusted-proxies"): "10.0.0.0/8, 192.168.1.5"}))
	if err != nil || len(cfg.TrustedProxies) != 2 || !cfg.TrustedProxies[0].Contains(netip.MustParseAddr("10.9.9.9")) {
		t.Fatalf("list: %v %v", cfg.TrustedProxies, err)
	}
	_, _, err = resolveWith(t, nil, baseEnv(t, map[string]string{envName("trusted-proxies"): "bogus"}))
	if err == nil || !strings.Contains(err.Error(), "trusted-proxies") {
		t.Errorf("bad value: %v", err)
	}
}

// SA-02/SEC-027: secrets come from the environment or a 0600 file, never argv,
// and never appear in the effective configuration.
func TestSecretsLoadingAndRedaction(t *testing.T) {
	const sec = "correct-horse-battery-staple"
	cfg, _, err := resolveWith(t, nil, baseEnv(t, map[string]string{envName("setup-secret"): sec}))
	if err != nil || cfg.SetupSecret != sec {
		t.Fatalf("env secret: %q %v", cfg.SetupSecret, err)
	}
	f := writeSecret(t, sec+"\n", 0o600)
	cfg, vals, err := resolveWith(t, []string{"-metrics-token-file", f}, baseEnv(t, nil))
	if err != nil || cfg.MetricsToken != sec {
		t.Fatalf("file secret: %q %v", cfg.MetricsToken, err)
	}
	cfg, _, err = resolveWith(t, nil, baseEnv(t, map[string]string{envName("backup-key"): sec}))
	if err != nil || string(cfg.BackupKey) != sec {
		t.Fatalf("backup key: %v", err)
	}
	var sb strings.Builder
	writeEffective(&sb, vals)
	writeSecretStatus(&sb, cfg)
	if strings.Contains(sb.String(), sec) || !strings.Contains(sb.String(), "setup-secret = <unset>") || !strings.Contains(sb.String(), "backup-key = <set>") {
		t.Errorf("effective output:\n%s", sb.String())
	}
	// No flag carries the secret itself, so it cannot reach argv or the process list.
	if err := resolveWithFlagErr(t, []string{"-setup-secret", sec}); err == nil {
		t.Error("-setup-secret flag accepted")
	}
	// Not accepted as a configuration file key either.
	cf := writeSecret(t, `{"setup-secret": "`+sec+`"}`, 0o600)
	if _, _, err := resolveWith(t, []string{"-config", cf}, baseEnv(t, nil)); err == nil {
		t.Error("secret accepted inside the config file")
	}
	// Ambiguous, too short, group-readable.
	if _, _, err := resolveWith(t, []string{"-setup-secret-file", f}, baseEnv(t, map[string]string{envName("setup-secret"): sec})); err == nil {
		t.Error("env and file together accepted")
	}
	if _, _, err := resolveWith(t, nil, baseEnv(t, map[string]string{envName("setup-secret"): "short"})); err == nil {
		t.Error("short secret accepted")
	}
	if wide := writeSecret(t, sec, 0o644); true {
		if _, _, err := resolveWith(t, []string{"-setup-secret-file", wide}, baseEnv(t, nil)); err == nil && !isWindows() {
			t.Error("world-readable secret file accepted")
		}
	}
}

// resolveWithFlagErr parses args against the real flag set and returns the parse error.
func resolveWithFlagErr(t *testing.T, args []string) error {
	t.Helper()
	fs := newFlagSet("t", io.Discard)
	bindSettingFlags(fs)
	return fs.Parse(args)
}

func isWindows() bool { return os.PathSeparator == '\\' }

// SA-15: /metrics needs the token when configured, refuses proxy hops and any
// forwarding header, even from loopback.
func TestMetricsGate(t *testing.T) {
	ops, _ := testOps(t)
	ops.cfg.TrustedProxies = []netip.Prefix{netip.MustParsePrefix("127.0.0.0/8")}
	h := ops.routes(stubHandler("api"), stubHandler("web"))
	if c := do(t, h, "GET", "/metrics", "127.0.0.1:5", nil).Code; c != 403 {
		t.Errorf("trusted proxy hop reached /metrics: %d", c)
	}
	ops.cfg.TrustedProxies = nil
	if c := do(t, h, "GET", "/metrics", "127.0.0.1:5", nil).Code; c != 200 {
		t.Errorf("loopback scrape: %d", c)
	}
	for _, hdr := range []string{"X-Forwarded-For", "Forwarded", "X-Real-Ip", "X-Forwarded-Proto", "X-Forwarded-Host", "Via"} {
		if c := do(t, h, "GET", "/metrics", "127.0.0.1:5", map[string]string{hdr: "1.2.3.4"}).Code; c != 403 {
			t.Errorf("%s accepted: %d", hdr, c)
		}
	}
	ops.cfg.MetricsToken = "0123456789abcdef-token"
	if c := do(t, h, "GET", "/metrics", "127.0.0.1:5", nil).Code; c != 401 {
		t.Errorf("no token: %d", c)
	}
	if c := do(t, h, "GET", "/metrics", "127.0.0.1:5", map[string]string{"Authorization": "Bearer wrong-wrong-wrong-x"}).Code; c != 401 {
		t.Errorf("wrong token: %d", c)
	}
	if c := do(t, h, "GET", "/metrics", "127.0.0.1:5", map[string]string{"Authorization": "Bearer 0123456789abcdef-token"}).Code; c != 200 {
		t.Errorf("right token: %d", c)
	}
	if c := do(t, h, "GET", "/metrics", "203.0.113.5:5", map[string]string{"Authorization": "Bearer 0123456789abcdef-token"}).Code; c != 403 {
		t.Errorf("right token from outside the allow list: %d", c)
	}
}

// WEB-01 / SA-02 / API-01: serve gives both adapters the operator's settings
// and one session scheme.
func TestServeWiresSettingsIntoAdapters(t *testing.T) {
	ctx := context.Background()
	db := seedDatabase(t, t.TempDir())
	st, err := store.Open(ctx, db)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = st.Close() }()
	cfg := testConfigFor(db)
	cfg.PublicBaseURL = "https://circula.example.org"
	cfg.CookieSecure = true
	cfg.SetupSecret = "0123456789abcdef-setup"
	svc, err := newService(st, cfg)
	if err != nil {
		t.Fatal(err)
	}
	apiH, webH, err := newHandlers(svc, cfg, discardLogger())
	if err != nil {
		t.Fatal(err)
	}
	ops := &opsServer{cfg: &Config{MaxBodyBytes: 1 << 20, RequestTimeout: 5 * time.Second}, db: st.DB(), logger: discardLogger(), metrics: newMetrics(time.Now())}
	srv := httptest.NewServer(ops.routes(apiH, webH))
	defer srv.Close()

	get := func(path string, cookies ...*http.Cookie) (*http.Response, string) {
		req, _ := http.NewRequest("GET", srv.URL+path, nil)
		for _, c := range cookies {
			req.AddCookie(c)
		}
		resp, err := http.DefaultTransport.RoundTrip(req)
		if err != nil {
			t.Fatal(err)
		}
		b, _ := io.ReadAll(resp.Body)
		_ = resp.Body.Close()
		return resp, string(b)
	}
	resp, body := get("/login") //nolint:bodyclose // get closes the body
	if resp.Header.Get("Strict-Transport-Security") == "" {
		t.Error("no HSTS with an https public URL")
	}
	var csrfCookie *http.Cookie
	for _, c := range resp.Cookies() {
		if c.Name == httpx.CSRFCookieSecure {
			csrfCookie = c
			if !c.Secure {
				t.Error("csrf cookie not Secure although cookie-secure is on")
			}
		}
	}
	tok := regexp.MustCompile(`name="csrf" value="([^"]+)"`).FindStringSubmatch(body)[1]
	form := url.Values{"login": {"founder01"}, "password": {"correct horse battery"}, "csrf": {tok}}
	req, _ := http.NewRequest("POST", srv.URL+"/login", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Origin", "https://circula.example.org") // the public origin, not the test server's
	req.AddCookie(csrfCookie)
	lr, err := http.DefaultTransport.RoundTrip(req)
	if err != nil {
		t.Fatal(err)
	}
	_ = lr.Body.Close()
	var sess *http.Cookie
	for _, c := range lr.Cookies() {
		if c.Name == httpx.SessionCookieSecure {
			sess = c
		}
	}
	if lr.StatusCode != 303 || sess == nil || !sess.Secure || sess.Path != "/" {
		t.Fatalf("login: %d %v", lr.StatusCode, lr.Cookies())
	}
	// The same cookie authenticates the API.
	ar, abody := get("/api/v1/me", sess) //nolint:bodyclose // get closes the body
	if ar.StatusCode != 200 || !strings.Contains(abody, "person_id") {
		t.Errorf("API with the web session: %d %s", ar.StatusCode, abody)
	}
	if ar.Header.Get("Strict-Transport-Security") == "" {
		t.Error("no HSTS on the API")
	}
	// The setup secret is wired: the form asks for it.
	_, setup := get("/setup", sess) //nolint:bodyclose // get closes the body
	if !strings.Contains(setup, "setup_secret") {
		t.Errorf("setup form does not ask for the secret:\n%s", setup)
	}
	// A foreign Origin is refused although it matches the Host header.
	bad, _ := http.NewRequest("POST", srv.URL+"/login", strings.NewReader(form.Encode()))
	bad.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	bad.Header.Set("Origin", srv.URL)
	bad.AddCookie(csrfCookie)
	br, err := http.DefaultTransport.RoundTrip(bad)
	if err != nil {
		t.Fatal(err)
	}
	_ = br.Body.Close()
	if br.StatusCode != 403 {
		t.Errorf("origin other than the public URL accepted: %d", br.StatusCode)
	}
}

// CMD-01: a backup holds exactly what the source held: verification writes
// nothing into it.
func TestBackupEqualsSource(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	db := seedDatabase(t, dir)
	cfg := testConfigFor(db)
	count := func(path, q string) int {
		raw, err := openRaw(path)
		if err != nil {
			t.Fatal(err)
		}
		defer func() { _ = raw.Close() }()
		var n int
		if err := raw.QueryRowContext(ctx, q).Scan(&n); err != nil {
			t.Fatal(err)
		}
		return n
	}
	srcAudit := count(db, `SELECT COUNT(*) FROM audit_events`)
	srcRuns := count(db, `SELECT COUNT(*) FROM consistency_runs`)
	mf, path, err := backupTo(ctx, cfg, filepath.Join(dir, "bk"), "manual", nil)
	if err != nil {
		t.Fatal(err)
	}
	if mf.Ledger.Status != "ok" {
		t.Fatalf("ledger %+v", mf.Ledger)
	}
	if n := count(path, `SELECT COUNT(*) FROM audit_events`); n != srcAudit {
		t.Errorf("backup has %d audit events, source %d", n, srcAudit)
	}
	if n := count(path, `SELECT COUNT(*) FROM audit_events WHERE request_id = 'backup-verify'`); n != 0 {
		t.Errorf("%d verification events were written into the backup", n)
	}
	if n := count(path, `SELECT COUNT(*) FROM consistency_runs`); n != srcRuns {
		t.Errorf("backup has %d consistency runs, source %d", n, srcRuns)
	}
	// The recorded digest is that of the file as it lies on disk.
	if got := sha(t, path); got != mf.SHA256 {
		t.Errorf("manifest digest %s != file %s", mf.SHA256, got)
	}
	if leftovers, _ := filepath.Glob(path + ".*"); len(leftovers) != 1 { // only the manifest
		t.Errorf("leftover files: %v", leftovers)
	}
}

// tamper edits a backup file in place and fixes up its manifest checksum, as
// someone with write access to the backup storage could.
func tamper(t *testing.T, path string, statements ...string) {
	t.Helper()
	raw, err := openRaw(path)
	if err != nil {
		t.Fatal(err)
	}
	for _, s := range statements {
		if _, err := raw.Exec(s); err != nil {
			t.Fatalf("%s: %v", s, err)
		}
	}
	_ = raw.Close()
	removeSideFiles(path)
	m, err := readManifest(path + ".manifest.json")
	if err != nil {
		t.Fatal(err)
	}
	m.SHA256, m.Bytes, _ = func() (string, int64, error) { s, n, e := fileSHA256(path); return s, n, e }()
	if err := writeManifest(path+".manifest.json", m); err != nil {
		t.Fatal(err)
	}
}

// SA-07: restore verifies the ledger and audit chain inside the backup, not
// just the self-attested manifest, and leaves the current database alone.
func TestRestoreRefusesBackupFailingEmbeddedChecks(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	db := seedDatabase(t, dir)
	cfg := testConfigFor(db)
	_, path, err := backupTo(ctx, cfg, filepath.Join(dir, "bk"), "manual", nil)
	if err != nil {
		t.Fatal(err)
	}
	tamper(t, path, `DROP TRIGGER audit_no_update`, `UPDATE audit_events SET reason = 'forged' WHERE seq = 1`)
	before := sha(t, db)
	var out, errOut bytes.Buffer
	code := run([]string{"restore", "-database", db, "-overwrite", path}, &out, &errOut)
	if code != exitLedgerFailed || !strings.Contains(errOut.String(), "FAILS ledger or audit-chain verification") || !strings.Contains(errOut.String(), "not touched") {
		t.Fatalf("code %d\nstdout %s\nstderr %s", code, out.String(), errOut.String())
	}
	if sha(t, db) != before {
		t.Error("a refused restore changed the current database")
	}
	if _, err := os.Stat(db + ".restore-partial"); err == nil {
		t.Error("partial file left behind")
	}
}

// SA-07: with backup-key, a forged manifest cannot pass; an unsigned or
// wrongly signed one is refused; without a key restore says it is unauthenticated.
func TestSignedBackupManifest(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	db := seedDatabase(t, dir)
	cfg := testConfigFor(db)
	cfg.BackupKey = []byte("0123456789abcdef-backup-key")
	mf, path, err := backupTo(ctx, cfg, filepath.Join(dir, "bk"), "manual", nil)
	if err != nil {
		t.Fatal(err)
	}
	if mf.HMAC == "" {
		t.Fatal("manifest not signed")
	}
	m, _ := readManifest(path + ".manifest.json")
	if ok, err := checkManifestAuth(m, cfg.BackupKey); !ok || err != nil {
		t.Fatalf("valid signature: %v %v", ok, err)
	}
	if _, err := checkManifestAuth(m, []byte("another-key-0123456789")); err == nil {
		t.Error("wrong key accepted")
	}
	unsigned := *m
	unsigned.HMAC = ""
	if _, err := checkManifestAuth(&unsigned, cfg.BackupKey); err == nil {
		t.Error("stripped signature accepted")
	}
	if ok, err := checkManifestAuth(&unsigned, nil); ok || err != nil {
		t.Errorf("no key: %v %v", ok, err)
	}

	// The credential-swap forgery from the audit: change a password hash, fix
	// the digest. Ledger checks cannot see it; the signature does.
	tamper(t, path, `UPDATE persons SET password_hash = 'attacker'`)
	keyFile := writeSecret(t, string(cfg.BackupKey), 0o600)
	var out, errOut bytes.Buffer
	code := run([]string{"restore", "-database", db, "-overwrite", "-backup-key-file", keyFile, path}, &out, &errOut)
	if code != exitFailure || !strings.Contains(errOut.String(), "signature") {
		t.Fatalf("forged manifest with key set: code %d, %s", code, errOut.String())
	}
	// Same file, no key configured: restore proceeds but says plainly it is unauthenticated.
	out.Reset()
	errOut.Reset()
	code = run([]string{"restore", "-database", db, "-overwrite", path}, &out, &errOut)
	if code != exitOK || !strings.Contains(out.String(), "not authenticated") {
		t.Fatalf("unkeyed restore: code %d\n%s\n%s", code, out.String(), errOut.String())
	}
}

// CMD-08: when the swapped-in database fails its checks the previous one is
// put back unchanged.
func TestSwapDatabaseRollsBack(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	db := seedDatabase(t, dir)
	before := sha(t, db)
	newFile := filepath.Join(dir, "incoming.db")
	if err := os.WriteFile(newFile, []byte("not a database"), 0o600); err != nil {
		t.Fatal(err)
	}
	err := swapDatabase(ctx, db, newFile, true, func(context.Context) error { return errors.New("checks failed") })
	if err == nil {
		t.Fatal("swap succeeded")
	}
	if sha(t, db) != before {
		t.Error("previous database not restored byte for byte")
	}
	for _, p := range []string{db + ".pre-restore-swap", newFile} {
		if _, err := os.Stat(p); err == nil && p != newFile {
			t.Errorf("%s left behind", p)
		}
	}
	// And a good swap replaces it and drops the aside copy.
	good := seedDatabase(t, t.TempDir())
	goodSum := sha(t, good)
	if err := swapDatabase(ctx, db, good, true, func(context.Context) error { return nil }); err != nil {
		t.Fatal(err)
	}
	if sha(t, db) != goodSum {
		t.Error("new database not in place")
	}
	if _, err := os.Stat(db + ".pre-restore-swap"); err == nil {
		t.Error("aside copy kept after success")
	}
}

// CMD-02: verify-ledger and create-operator refuse to run beside another
// process that holds the database.
func TestOperatorCommandsTakeTheLock(t *testing.T) {
	db := seedDatabase(t, t.TempDir())
	held, err := acquireLock(lockPath(db))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = held.Release() }()
	before := sha(t, db)
	t.Setenv(operatorPasswordEnv, "correct horse battery")
	for _, args := range [][]string{
		{"verify-ledger", "-database", db},
		{"admin", "create-operator", "-database", db, "-login", "ops01"},
	} {
		var out, errOut bytes.Buffer
		if code := run(args, &out, &errOut); code != exitFailure || !strings.Contains(errOut.String(), "in use by another circula process") {
			t.Errorf("%v: code %d, %s", args, code, errOut.String())
		}
	}
	if sha(t, db) != before {
		t.Error("a refused command wrote to the database")
	}
	_ = held.Release()
	var out, errOut bytes.Buffer
	if code := run([]string{"admin", "create-operator", "-database", db, "-login", "ops01"}, &out, &errOut); code != exitOK {
		t.Errorf("create-operator without a holder: %d %s", code, errOut.String())
	}
}

func testNow() time.Time { return time.Now() }
