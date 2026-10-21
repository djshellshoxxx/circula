package main

import (
	"flag"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// sampleValues gives every setting a valid non-default value. TestEverySettingAcceptedByEverySource
// fails when a setting is added without one.
var sampleValues = map[string]string{
	"listen":                "127.0.0.1:9090",
	"public-base-url":       "https://circula.example.test",
	"database":              "", // filled per test: a file in a temp dir
	"backup-dir":            "", // filled per test
	"backup-interval":       "12h",
	"backup-retain-daily":   "7",
	"backup-retain-monthly": "3",
	"migrate-mode":          "auto",
	"session-idle":          "1h",
	"session-max":           "1000h",
	"idempotency-ttl":       "48h",
	"min-password-length":   "14",
	"max-body-bytes":        "2048",
	"request-timeout":       "5s",
	"log-level":             "debug",
	"log-format":            "text",
	"cookie-secure":         "false",
	"metrics-allow":         "10.0.0.0/8",
	"trusted-proxies":       "10.0.0.0/8, 192.168.1.1",
	"allow-insecure-http":   "true",
	"setup-secret-file":     "", // filled per test: a 0600 file
	"metrics-token-file":    "",
	"backup-key-file":       "",
}

// baseEnv returns a lookup that supplies the two required settings.
func baseEnv(t *testing.T, extra map[string]string) lookupEnv {
	t.Helper()
	dir := t.TempDir()
	env := map[string]string{
		envName("public-base-url"): "https://circula.example.test",
		envName("database"):        filepath.Join(dir, "circula.db"),
	}
	for k, v := range extra {
		env[k] = v
	}
	return func(k string) (string, bool) {
		v, ok := env[k]
		return v, ok
	}
}

func resolveWith(t *testing.T, args []string, env lookupEnv) (*Config, map[string]resolved, error) {
	t.Helper()
	fs := flag.NewFlagSet("test", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	sf := bindSettingFlags(fs)
	if err := fs.Parse(args); err != nil {
		t.Fatal(err)
	}
	return sf.Resolve(env, true)
}

func TestDefaultsAndDerivedBackupDir(t *testing.T) {
	env := baseEnv(t, nil)
	cfg, vals, err := resolveWith(t, nil, env)
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	if cfg.Listen != "127.0.0.1:8080" {
		t.Errorf("listen default = %q", cfg.Listen)
	}
	if cfg.BackupDir != filepath.Join(filepath.Dir(cfg.Database), "backups") {
		t.Errorf("backup dir = %q", cfg.BackupDir)
	}
	if cfg.MigrateMode != "explicit" || cfg.LogFormat != "json" || cfg.BackupRetainDaily != 30 || cfg.BackupRetainMonthly != 12 {
		t.Errorf("unexpected defaults: %+v", cfg)
	}
	if vals["listen"].source != "default" {
		t.Errorf("listen source = %q", vals["listen"].source)
	}
}

// TestPrecedence checks flags > env > file > defaults for one setting.
func TestPrecedence(t *testing.T) {
	dir := t.TempDir()
	cfgFile := filepath.Join(dir, "circula.json")
	if err := os.WriteFile(cfgFile, []byte(`{"listen": "127.0.0.1:1111", "request-timeout": "10s",
		"database": "`+filepath.ToSlash(filepath.Join(dir, "a.db"))+`", "public-base-url": "https://x.test"}`), 0o600); err != nil {
		t.Fatal(err)
	}

	// File only.
	cfg, vals, err := resolveWith(t, []string{"-config", cfgFile}, baseEnv(t, nil))
	if err != nil {
		t.Fatalf("file only: %v", err)
	}
	_ = cfg
	if vals["listen"].value != "127.0.0.1:1111" || vals["listen"].source != "file" {
		t.Errorf("file value not applied: %+v", vals["listen"])
	}

	// File and env: env wins.
	env := func(k string) (string, bool) {
		if k == envName("listen") {
			return "127.0.0.1:2222", true
		}
		return "", false
	}
	_, vals, err = resolveWith(t, []string{"-config", cfgFile}, env)
	if err != nil {
		t.Fatalf("file+env: %v", err)
	}
	if vals["listen"].value != "127.0.0.1:2222" || vals["listen"].source != "env" {
		t.Errorf("env did not override file: %+v", vals["listen"])
	}

	// File, env and flag: flag wins.
	cfg, vals, err = resolveWith(t, []string{"-config", cfgFile, "-listen", "127.0.0.1:3333"}, env)
	if err != nil {
		t.Fatalf("all three: %v", err)
	}
	if cfg.Listen != "127.0.0.1:3333" || vals["listen"].source != "flag" {
		t.Errorf("flag did not override: %+v", vals["listen"])
	}
}

// TestEverySettingAcceptedByEverySource proves each documented setting works
// through the flag, environment and file sources (OPS-006).
func TestEverySettingAcceptedByEverySource(t *testing.T) {
	dir := t.TempDir()
	for _, d := range settingDefs {
		v, ok := sampleValues[d.key]
		if !ok {
			t.Fatalf("sampleValues has no entry for setting %q", d.key)
		}
		if d.key == "database" {
			v = filepath.Join(dir, "x.db")
		}
		if d.key == "backup-dir" {
			v = filepath.Join(dir, "bk")
		}
		if strings.HasSuffix(d.key, "-file") {
			v = filepath.Join(dir, d.key+".secret")
			if err := os.WriteFile(v, []byte("0123456789abcdef-secret\n"), 0o600); err != nil {
				t.Fatal(err)
			}
		}
		base := baseEnv(t, map[string]string{envName(d.key): ""})
		// Environment source.
		env := func(k string) (string, bool) {
			if k == envName(d.key) {
				return v, true
			}
			return base(k)
		}
		if _, _, err := resolveWith(t, nil, env); err != nil {
			t.Errorf("env %s: %v", d.key, err)
		}
		// Flag source.
		if _, _, err := resolveWith(t, []string{"-" + d.key, v}, base); err != nil {
			t.Errorf("flag %s: %v", d.key, err)
		}
		// File source.
		fp := filepath.Join(dir, "f-"+d.key+".json")
		body := `{"` + d.key + `": "` + strings.ReplaceAll(v, `\`, `/`) + `"}`
		if err := os.WriteFile(fp, []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
		if _, _, err := resolveWith(t, []string{"-config", fp}, base); err != nil {
			t.Errorf("file %s: %v", d.key, err)
		}
	}
}

func TestRequiredSettingsAreNamed(t *testing.T) {
	_, _, err := resolveWith(t, nil, func(string) (string, bool) { return "", false })
	if err == nil {
		t.Fatal("expected error")
	}
	for _, want := range []string{"public-base-url", "database"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("error %q does not name %s", err, want)
		}
	}
	// Commands other than serve need only the database path.
	fs := flag.NewFlagSet("t", flag.ContinueOnError)
	sf := bindSettingFlags(fs)
	if err := fs.Parse(nil); err != nil {
		t.Fatal(err)
	}
	_, _, err = sf.Resolve(baseEnv(t, map[string]string{envName("public-base-url"): ""}), false)
	if err != nil {
		t.Errorf("non-serve command needs public-base-url: %v", err)
	}
}

func TestInvalidValuesNameEachSetting(t *testing.T) {
	env := baseEnv(t, map[string]string{
		envName("request-timeout"):     "forever",
		envName("migrate-mode"):        "sometimes",
		envName("min-password-length"): "8",
		envName("listen"):              "no-port",
	})
	_, _, err := resolveWith(t, nil, env)
	if err == nil {
		t.Fatal("expected error")
	}
	for _, want := range []string{"request-timeout", "migrate-mode", "min-password-length", "listen"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("error does not name %s: %v", want, err)
		}
	}
}

func TestUnknownFileKeyRejected(t *testing.T) {
	dir := t.TempDir()
	fp := filepath.Join(dir, "c.json")
	if err := os.WriteFile(fp, []byte(`{"lisen": ":8080"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	_, _, err := resolveWith(t, []string{"-config", fp}, baseEnv(t, nil))
	if err == nil || !strings.Contains(err.Error(), "lisen") {
		t.Fatalf("expected unknown key error naming lisen, got %v", err)
	}
}

func TestEmptyEnvIsUnset(t *testing.T) {
	cfg, vals, err := resolveWith(t, nil, baseEnv(t, map[string]string{envName("listen"): ""}))
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Listen != "127.0.0.1:8080" || vals["listen"].source != "default" {
		t.Errorf("empty env should be ignored: %+v", vals["listen"])
	}
}

func TestSessionOrderingAndPasswordFloor(t *testing.T) {
	_, _, err := resolveWith(t, nil, baseEnv(t, map[string]string{
		envName("session-idle"): "10h",
		envName("session-max"):  "1h",
	}))
	if err == nil || !strings.Contains(err.Error(), "session-max") {
		t.Errorf("session-max shorter than idle accepted: %v", err)
	}
	_, _, err = resolveWith(t, nil, baseEnv(t, map[string]string{envName("min-password-length"): "11"}))
	if err == nil || !strings.Contains(err.Error(), "min-password-length") {
		t.Errorf("password floor below 12 accepted: %v", err)
	}
}

func TestMetricsAllowParsing(t *testing.T) {
	cfg, _, err := resolveWith(t, nil, baseEnv(t, map[string]string{envName("metrics-allow"): "none"}))
	if err != nil || len(cfg.MetricsAllow) != 0 {
		t.Fatalf("none: %v %v", cfg.MetricsAllow, err)
	}
	cfg, _, err = resolveWith(t, nil, baseEnv(t, map[string]string{envName("metrics-allow"): "10.1.2.3, 192.168.0.0/16"}))
	if err != nil || len(cfg.MetricsAllow) != 2 {
		t.Fatalf("list: %v %v", cfg.MetricsAllow, err)
	}
	if _, _, err = resolveWith(t, nil, baseEnv(t, map[string]string{envName("metrics-allow"): "not-an-ip"})); err == nil {
		t.Error("bad CIDR accepted")
	}
}

func TestEffectiveConfigListsEverySettingWithSource(t *testing.T) {
	_, vals, err := resolveWith(t, nil, baseEnv(t, nil))
	if err != nil {
		t.Fatal(err)
	}
	var sb strings.Builder
	writeEffective(&sb, vals)
	for _, d := range settingDefs {
		if !strings.Contains(sb.String(), d.key+" = ") {
			t.Errorf("effective output lacks %s", d.key)
		}
	}
}

func TestCheckConfigDoesNotCreateDatabase(t *testing.T) {
	dir := t.TempDir()
	db := filepath.Join(dir, "never.db")
	var out, errOut strings.Builder
	code := run([]string{"serve", "-check-config", "-public-base-url", "https://x.test", "-database", db}, &out, &errOut)
	if code != exitOK {
		t.Fatalf("code %d: %s", code, errOut.String())
	}
	if _, err := os.Stat(db); err == nil {
		t.Fatal("check-config created the database")
	}
}
