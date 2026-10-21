package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/netip"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
)

// envPrefix is prepended to every upper-cased setting key to form its
// environment variable name (for example listen -> CIRCULA_LISTEN).
const envPrefix = "CIRCULA_"

// configFileEnv names the environment variable that selects a configuration file.
const configFileEnv = envPrefix + "CONFIG"

// settingDef is one documented setting. The same key is used as the flag name
// (-key), the configuration file key, and the environment variable suffix.
type settingDef struct {
	key      string
	def      string // "" means no default (the setting must be supplied)
	required bool
	help     string
}

// settingDefs is the authoritative setting list; docs/ops/configuration.md
// mirrors it and config_test.go checks that every entry has a sample value.
var settingDefs = []settingDef{
	{key: "listen", def: "127.0.0.1:8080", help: "host:port to listen on; loopback by default"},
	{key: "public-base-url", required: true, help: "absolute http(s) URL users reach, used in links"},
	{key: "database", required: true, help: "path to the SQLite database file"},
	{key: "backup-dir", help: "directory for scheduled and pre-migration backups (default: <database dir>/backups)"},
	{key: "backup-interval", def: "24h", help: "time between scheduled backups; 0 disables them"},
	{key: "backup-retain-daily", def: "30", help: "newest backups kept"},
	{key: "backup-retain-monthly", def: "12", help: "months for which the newest backup of each month is also kept; 0 disables"},
	{key: "migrate-mode", def: "explicit", help: "explicit: refuse to serve with pending migrations; auto: back up and migrate at start-up"},
	{key: "session-idle", def: "720h", help: "idle session timeout (DEC-005)"},
	{key: "session-max", def: "2160h", help: "absolute session lifetime (DEC-005)"},
	{key: "idempotency-ttl", def: "168h", help: "retention of idempotency records"},
	{key: "min-password-length", def: "12", help: "minimum password length (SEC-002); not below 12"},
	{key: "max-body-bytes", def: "1048576", help: "maximum request body size in bytes"},
	{key: "request-timeout", def: "30s", help: "per-request deadline for API and UI routes"},
	{key: "log-level", def: "info", help: "debug, info, warn or error"},
	{key: "log-format", def: "json", help: "json or text"},
	{key: "cookie-secure", def: "true", help: "Secure attribute for session cookies (SEC-022); reserved until the web adapter reads it"},
	{key: "metrics-allow", def: "127.0.0.1/32,::1/128", help: "comma-separated CIDRs allowed to read /metrics; none to deny all"},
}

// Config is the typed, validated configuration.
type Config struct {
	Listen              string
	PublicBaseURL       string
	Database            string
	BackupDir           string
	BackupInterval      time.Duration
	BackupRetainDaily   int
	BackupRetainMonthly int
	MigrateMode         string
	SessionIdle         time.Duration
	SessionMax          time.Duration
	IdempotencyTTL      time.Duration
	MinPasswordLength   int
	MaxBodyBytes        int64
	RequestTimeout      time.Duration
	LogLevel            slog.Level
	LogFormat           string
	CookieSecure        bool
	MetricsAllow        []netip.Prefix
}

// resolved is one effective setting value and where it came from.
type resolved struct {
	value  string
	source string // default, file, env or flag
}

// settingsFlags holds the flag bindings shared by every subcommand.
type settingsFlags struct {
	fs         *flag.FlagSet
	configPath string
	values     map[string]*string
}

// bindSettingFlags registers --config and one flag per setting on fs.
func bindSettingFlags(fs *flag.FlagSet) *settingsFlags {
	sf := &settingsFlags{fs: fs, values: map[string]*string{}}
	fs.StringVar(&sf.configPath, "config", "", "path to a JSON configuration file (env "+configFileEnv+")")
	for _, d := range settingDefs {
		sf.values[d.key] = fs.String(d.key, "", d.help+" (env "+envName(d.key)+")")
	}
	return sf
}

// envName returns the environment variable for a setting key.
func envName(key string) string {
	return envPrefix + strings.ToUpper(strings.ReplaceAll(key, "-", "_"))
}

// lookupEnv is injectable for tests.
type lookupEnv func(string) (string, bool)

// Resolve applies precedence flags > environment > configuration file >
// defaults and validates the result. It reports every invalid setting, not
// just the first. Empty environment and file values count as unset.
func (sf *settingsFlags) Resolve(env lookupEnv, forServe bool) (*Config, map[string]resolved, error) {
	explicit := map[string]bool{}
	sf.fs.Visit(func(f *flag.Flag) { explicit[f.Name] = true })

	vals := map[string]resolved{}
	for _, d := range settingDefs {
		if d.def != "" {
			vals[d.key] = resolved{value: d.def, source: "default"}
		}
	}

	path := ""
	if explicit["config"] && sf.configPath != "" {
		path = sf.configPath
	} else if v, ok := env(configFileEnv); ok && v != "" {
		path = v
	}
	if path != "" {
		fileVals, err := readConfigFile(path)
		if err != nil {
			return nil, nil, err
		}
		for k, v := range fileVals {
			vals[k] = resolved{value: v, source: "file"}
		}
	}

	for _, d := range settingDefs {
		if v, ok := env(envName(d.key)); ok && v != "" {
			vals[d.key] = resolved{value: v, source: "env"}
		}
	}
	for _, d := range settingDefs {
		if explicit[d.key] {
			vals[d.key] = resolved{value: *sf.values[d.key], source: "flag"}
		}
	}

	cfg, err := buildConfig(vals, forServe)
	if err != nil {
		return nil, nil, err
	}
	if _, ok := vals["backup-dir"]; !ok {
		vals["backup-dir"] = resolved{value: cfg.BackupDir, source: "derived"}
	}
	return cfg, vals, nil
}

// readConfigFile parses a JSON object whose keys are setting names. Unknown
// keys are rejected so that typos cannot silently fall back to defaults.
func readConfigFile(path string) (map[string]string, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("config file %s: %w", path, err)
	}
	dec := json.NewDecoder(bytes.NewReader(b))
	dec.UseNumber()
	var raw map[string]any
	if err := dec.Decode(&raw); err != nil {
		return nil, fmt.Errorf("config file %s: invalid JSON object: %w", path, err)
	}
	known := map[string]bool{}
	for _, d := range settingDefs {
		known[d.key] = true
	}
	out := map[string]string{}
	var problems []string
	for k, v := range raw {
		if !known[k] {
			problems = append(problems, fmt.Sprintf("unknown setting %q", k))
			continue
		}
		switch t := v.(type) {
		case string:
			out[k] = t
		case json.Number:
			out[k] = t.String()
		case bool:
			out[k] = strconv.FormatBool(t)
		default:
			problems = append(problems, fmt.Sprintf("setting %q must be a string, number or boolean", k))
		}
	}
	if len(problems) > 0 {
		sort.Strings(problems)
		return nil, fmt.Errorf("config file %s: %s", path, strings.Join(problems, "; "))
	}
	return out, nil
}

// buildConfig parses effective string values into a Config. The database
// path is required for every command; public-base-url only when forServe is set.
func buildConfig(vals map[string]resolved, forServe bool) (*Config, error) {
	var errs []error
	get := func(key string) string { return vals[key].value }
	fail := func(key, format string, a ...any) {
		errs = append(errs, &settingError{key: key, msg: fmt.Sprintf(format, a...)})
	}
	for _, d := range settingDefs {
		if !d.required || (d.key == "public-base-url" && !forServe) {
			continue
		}
		if _, ok := vals[d.key]; !ok {
			fail(d.key, "required; set flag -%s, env %s or file key %q", d.key, envName(d.key), d.key)
		}
	}
	cfg := &Config{}

	cfg.Listen = get("listen")
	if _, port, err := net.SplitHostPort(cfg.Listen); err != nil {
		fail("listen", "must be host:port")
	} else if p, err := strconv.Atoi(port); err != nil || p < 1 || p > 65535 {
		fail("listen", "port must be 1 to 65535")
	}

	if raw, ok := vals["public-base-url"]; ok {
		u, err := url.Parse(raw.value)
		switch {
		case err != nil, u.Host == "":
			fail("public-base-url", "must be an absolute URL")
		case u.Scheme != "http" && u.Scheme != "https":
			fail("public-base-url", "scheme must be http or https")
		default:
			cfg.PublicBaseURL = strings.TrimRight(raw.value, "/")
		}
	}

	if raw, ok := vals["database"]; ok {
		p := filepath.Clean(raw.value)
		if fi, err := os.Stat(filepath.Dir(p)); err != nil || !fi.IsDir() {
			fail("database", "directory %s does not exist", filepath.Dir(p))
		}
		cfg.Database = p
	}

	if raw, ok := vals["backup-dir"]; ok && raw.value != "" {
		cfg.BackupDir = filepath.Clean(raw.value)
	} else if cfg.Database != "" {
		cfg.BackupDir = filepath.Join(filepath.Dir(cfg.Database), "backups")
	}

	cfg.BackupInterval = parseDuration(get("backup-interval"), "backup-interval", fail, 0, 366*24*time.Hour)
	cfg.BackupRetainDaily = parseInt(get("backup-retain-daily"), "backup-retain-daily", fail, 1, 10000)
	cfg.BackupRetainMonthly = parseInt(get("backup-retain-monthly"), "backup-retain-monthly", fail, 0, 1200)

	cfg.MigrateMode = get("migrate-mode")
	if cfg.MigrateMode != "explicit" && cfg.MigrateMode != "auto" {
		fail("migrate-mode", "must be explicit or auto")
	}

	cfg.SessionIdle = parseDuration(get("session-idle"), "session-idle", fail, time.Minute, 366*24*time.Hour)
	cfg.SessionMax = parseDuration(get("session-max"), "session-max", fail, time.Minute, 366*24*time.Hour)
	if cfg.SessionMax < cfg.SessionIdle {
		fail("session-max", "must not be shorter than session-idle")
	}
	cfg.IdempotencyTTL = parseDuration(get("idempotency-ttl"), "idempotency-ttl", fail, time.Hour, 90*24*time.Hour)
	cfg.MinPasswordLength = parseInt(get("min-password-length"), "min-password-length", fail, 12, 1024)
	cfg.MaxBodyBytes = int64(parseInt(get("max-body-bytes"), "max-body-bytes", fail, 1024, 1<<30))
	cfg.RequestTimeout = parseDuration(get("request-timeout"), "request-timeout", fail, time.Second, 10*time.Minute)

	switch strings.ToLower(get("log-level")) {
	case "debug":
		cfg.LogLevel = slog.LevelDebug
	case "info":
		cfg.LogLevel = slog.LevelInfo
	case "warn":
		cfg.LogLevel = slog.LevelWarn
	case "error":
		cfg.LogLevel = slog.LevelError
	default:
		fail("log-level", "must be debug, info, warn or error")
	}
	cfg.LogFormat = get("log-format")
	if cfg.LogFormat != "json" && cfg.LogFormat != "text" {
		fail("log-format", "must be json or text")
	}

	if b, err := strconv.ParseBool(get("cookie-secure")); err != nil {
		fail("cookie-secure", "must be true or false")
	} else {
		cfg.CookieSecure = b
	}

	cfg.MetricsAllow = parsePrefixes(get("metrics-allow"), fail)

	if len(errs) > 0 {
		return nil, errors.Join(errs...)
	}
	return cfg, nil
}

// settingError names the setting that failed validation.
type settingError struct {
	key string
	msg string
}

func (e *settingError) Error() string { return fmt.Sprintf("setting %s: %s", e.key, e.msg) }

func parseDuration(s, key string, fail func(string, string, ...any), min, max time.Duration) time.Duration {
	d, err := time.ParseDuration(s)
	if err != nil {
		fail(key, "must be a duration such as 30s or 24h")
		return 0
	}
	if d == 0 && min == 0 {
		return 0
	}
	if d < min || d > max {
		fail(key, "must be between %s and %s", min, max)
	}
	return d
}

func parseInt(s, key string, fail func(string, string, ...any), min, max int) int {
	n, err := strconv.Atoi(s)
	if err != nil {
		fail(key, "must be an integer")
		return 0
	}
	if n < min || n > max {
		fail(key, "must be between %d and %d", min, max)
	}
	return n
}

// parsePrefixes accepts CIDRs or single addresses, comma separated. "none"
// or an empty list denies everything.
func parsePrefixes(s string, fail func(string, string, ...any)) []netip.Prefix {
	if s == "none" {
		return nil
	}
	var out []netip.Prefix
	for _, part := range strings.Split(s, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		if p, err := netip.ParsePrefix(part); err == nil {
			out = append(out, p.Masked())
			continue
		}
		if a, err := netip.ParseAddr(part); err == nil {
			out = append(out, netip.PrefixFrom(a, a.BitLen()))
			continue
		}
		fail("metrics-allow", "%q is not an address or CIDR", part)
		return nil
	}
	return out
}

// writeEffective prints every setting, its value and its source. Settings are
// all non-secret; secret values, when added, must be redacted here.
func writeEffective(w io.Writer, vals map[string]resolved) {
	for _, d := range settingDefs {
		r, ok := vals[d.key]
		if !ok {
			if d.required {
				fmt.Fprintf(w, "%s = <unset> (required)\n", d.key)
			} else {
				fmt.Fprintf(w, "%s = <unset>\n", d.key)
			}
			continue
		}
		fmt.Fprintf(w, "%s = %s (%s)\n", d.key, r.value, r.source)
	}
}
