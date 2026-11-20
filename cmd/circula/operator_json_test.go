package main

import (
	"bytes"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestConfigFileRequiresExactlyOneObject(t *testing.T) {
	for _, tc := range []struct {
		name string
		body string
	}{
		{"second object", `{"maintenance":true} {"maintenance":false}`},
		{"second array", `{} []`},
		{"trailing null", `{} null`},
		{"trailing text", `{} copied twice`},
		{"incomplete second value", `{} {`},
		{"null", `null`},
		{"array", `[]`},
		{"scalar", `true`},
		{"unknown field", `{"unexpected-setting":true}`},
		{"unsupported setting value", `{"maintenance":null}`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "config.json")
			if err := os.WriteFile(path, []byte(tc.body), 0o600); err != nil {
				t.Fatal(err)
			}
			values, err := readConfigFile(path)
			if err == nil || values != nil {
				t.Fatalf("invalid complete file accepted: values=%v err=%v", values, err)
			}
		})
	}
}

func TestConfigFilePreservesWhitespaceAndScalarValues(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")
	// Parsing must preserve numeric precision before buildConfig applies each
	// setting's own range, and it must retain ordinary boolean/string values.
	body := " \n\t" + `{"signup-grant-cap":9007199254740993,"maintenance":false,"listen":"127.0.0.1:9090"}` + "\r\n\t "
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	values, err := readConfigFile(path)
	want := map[string]string{"signup-grant-cap": "9007199254740993", "maintenance": "false", "listen": "127.0.0.1:9090"}
	if err != nil || !reflect.DeepEqual(values, want) {
		t.Fatalf("valid object: values=%v err=%v, want %v", values, err, want)
	}
	if err := os.WriteFile(path, []byte("{}\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if values, err := readConfigFile(path); err != nil || values == nil || len(values) != 0 {
		t.Fatalf("empty object: values=%v err=%v", values, err)
	}
}

func TestConfigPreflightRejectsIncompleteFileValidation(t *testing.T) {
	for _, tc := range []struct {
		name string
		body string
		code int
	}{
		{"whole object", "{\"maintenance\":true}\n\t", exitOK},
		{"concatenated objects", `{"maintenance":true} {"maintenance":false}`, exitUsage},
		{"trailing text", `{} operator notes`, exitUsage},
		{"null", `null`, exitUsage},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			path, db := filepath.Join(dir, "config.json"), filepath.Join(dir, "unused.db")
			if err := os.WriteFile(path, []byte(tc.body), 0o600); err != nil {
				t.Fatal(err)
			}
			var out, errOut bytes.Buffer
			code := run([]string{"serve", "-check-config", "-config", path, "-database", db,
				"-public-base-url", "https://circula.example.test"}, &out, &errOut)
			if code != tc.code {
				t.Fatalf("preflight code=%d, want %d; stderr=%s", code, tc.code, errOut.String())
			}
			if tc.code == exitUsage && (out.Len() != 0 || !strings.Contains(errOut.String(), "invalid JSON object")) {
				t.Fatalf("invalid file must not print an effective configuration: stdout=%s stderr=%s", out.String(), errOut.String())
			}
			if _, err := os.Stat(db); !os.IsNotExist(err) {
				t.Fatalf("configuration preflight accessed the database: %v", err)
			}
		})
	}
}

func TestAuditAnchorRequiresExactlyOneObject(t *testing.T) {
	db := seedDatabase(t, t.TempDir())
	keyFile := writeSecret(t, strings.Repeat("anchor", 8), 0o600)
	t.Setenv(envName("audit-anchor-key-file"), keyFile)
	path := filepath.Join(t.TempDir(), "original.json")
	if code, out := runAnchor(t, db, "write", path); code != exitOK {
		t.Fatalf("write: code=%d %s", code, out)
	}
	body, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	before := sha(t, db)
	for _, tc := range []struct {
		name string
		body string
		code int
	}{
		{"whole signed object", string(body), exitOK},
		{"surrounding whitespace", " \n\t" + string(body) + "\r\n\t ", exitOK},
		{"second object", string(body) + `{}`, exitFailure},
		{"second array", string(body) + `[]`, exitFailure},
		{"trailing null", string(body) + `null`, exitFailure},
		{"trailing text", string(body) + `operator notes`, exitFailure},
		{"incomplete second value", string(body) + `{`, exitFailure},
		{"null", `null`, exitFailure},
		{"array", `[]`, exitFailure},
		{"scalar", `true`, exitFailure},
		{"unknown field", strings.TrimSuffix(strings.TrimSpace(string(body)), "}") + `,"extra":true}`, exitFailure},
		{"unknown platform field", strings.Replace(string(body), `"platform": {`, `"platform": {"extra":true,`, 1), exitFailure},
		{"unsupported version", `{"version":999}`, exitLedgerFailed},
		{"missing platform coverage", `{"version":2}`, exitLedgerFailed},
	} {
		t.Run(tc.name, func(t *testing.T) {
			candidate := filepath.Join(t.TempDir(), "candidate.json")
			if err := os.WriteFile(candidate, []byte(tc.body), 0o600); err != nil {
				t.Fatal(err)
			}
			code, out := runAnchor(t, db, "verify", candidate)
			if code != tc.code {
				t.Fatalf("verify code=%d, want %d; output=%s", code, tc.code, out)
			}
			if tc.code == exitFailure && (!strings.Contains(out, "not an anchor file") || strings.Contains(out, "audit anchor: ok")) {
				t.Fatalf("invalid file must not be reported as verified: %s", out)
			}
		})
	}
	if after := sha(t, db); after != before {
		t.Error("anchor verification changed the database")
	}
}
