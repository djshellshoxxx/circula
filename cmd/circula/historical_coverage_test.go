package main

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"math"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/djshellshoxxx/circula/internal/service"
	tu "github.com/djshellshoxxx/circula/internal/testutil"
)

// OPS-016: evidence coverage and invariant success are independent. Unknown
// historical coverage must survive aggregation, including an otherwise good run.
func TestHistoricalLimitSummaryCoverage(t *testing.T) {
	complete := service.HistoricalLimitCoverage{Status: "complete", CheckedDebits: 2, SystemDebits: 1}
	partial := service.HistoricalLimitCoverage{Status: "partial", CheckedDebits: 3, LegacyUnverifiedDebits: 4, UnverifiableDebits: 1}
	for _, tc := range []struct {
		name   string
		checks []networkCheck
		want   *service.HistoricalLimitCoverage
		status string
	}{
		{"complete", []networkCheck{{OK: true, HistoricalLimits: &complete}}, &complete, "ok"},
		{"partial passes", []networkCheck{{OK: true, HistoricalLimits: &partial}}, &partial, "ok"},
		{"complete has finding", []networkCheck{{OK: false, HistoricalLimits: &complete}}, &complete, "failed"},
		{"aggregate", []networkCheck{{OK: true, HistoricalLimits: &complete}, {OK: true, HistoricalLimits: &partial}},
			&service.HistoricalLimitCoverage{Status: "partial", CheckedDebits: 5, SystemDebits: 1, LegacyUnverifiedDebits: 4, UnverifiableDebits: 1}, "ok"},
		{"platform excluded", []networkCheck{{OK: true, HistoricalLimits: &complete}, {Platform: true, OK: false}}, &complete, "failed"},
		{"one unknown network", []networkCheck{{OK: true, HistoricalLimits: &complete}, {OK: true}}, nil, "ok"},
		{"platform only", []networkCheck{{Platform: true, OK: true}}, nil, "ok"},
		{"no scopes", nil, nil, "ok"},
		{"invalid status", []networkCheck{{OK: true, HistoricalLimits: &service.HistoricalLimitCoverage{Status: ""}}}, nil, "ok"},
		{"negative count", []networkCheck{{OK: true, HistoricalLimits: &service.HistoricalLimitCoverage{Status: "complete", CheckedDebits: -1}}}, nil, "ok"},
		{"count overflow", []networkCheck{{OK: true, HistoricalLimits: &service.HistoricalLimitCoverage{Status: "complete", CheckedDebits: math.MaxInt64}},
			{OK: true, HistoricalLimits: &complete}}, nil, "ok"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got := summarise(tc.checks)
			if got.Status != tc.status || !reflect.DeepEqual(got.HistoricalLimits, tc.want) {
				t.Fatalf("summary = %+v coverage=%+v, want status=%s coverage=%+v", got, got.HistoricalLimits, tc.status, tc.want)
			}
			if got.HistoricalLimits != nil && (got.HistoricalLimits == &complete || got.HistoricalLimits == &partial) {
				t.Fatal("aggregation retained a mutable source pointer")
			}
		})
	}
}

func TestHistoricalLimitCheckOutputKeepsResultSeparate(t *testing.T) {
	for _, tc := range []struct {
		name     string
		ok       bool
		coverage *service.HistoricalLimitCoverage
		want     string
	}{
		{"old unknown", true, nil, "unknown (not recorded)"},
		{"legacy partial", true, &service.HistoricalLimitCoverage{Status: "partial", CheckedDebits: 2, SystemDebits: 1, LegacyUnverifiedDebits: 3},
			"partial (checked debits: 2; system debits: 1; legacy unverified debits: 3; unverifiable debits: 0)"},
		{"complete failure", false, &service.HistoricalLimitCoverage{Status: "complete", CheckedDebits: 2}, "complete (checked debits: 2;"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var out bytes.Buffer
			checks := []networkCheck{{Slug: "town", OK: tc.ok, HistoricalLimits: tc.coverage}, {Platform: true, OK: true}}
			if got := printChecks(&out, checks); got != tc.ok {
				t.Fatalf("coverage changed command success: %v, want %v", got, tc.ok)
			}
			if !strings.Contains(out.String(), "historical credit-limit coverage: "+tc.want) ||
				strings.Count(out.String(), "historical credit-limit coverage:") != 1 {
				t.Fatalf("missing financial coverage or invented platform coverage: %s", &out)
			}
		})
	}
}

func TestHistoricalLimitVerificationAndSnapshotPropagation(t *testing.T) {
	e := tu.NewEnv(t)
	admin := e.Person("historical-admin")
	net, adminM := e.Network(admin, "historical", nil)
	member, memberM := e.Member(admin, net, "historical-member")
	e.Trade(net.ID, admin, member, adminM, memberM, 100)
	want := &service.HistoricalLimitCoverage{Status: "complete", CheckedDebits: 1}
	checks := tu.Must(verifyAllNetworks(e.Ctx, e.Svc, "historical-consumer-check", ""))
	if len(checks) != 2 || checks[0].Platform || !reflect.DeepEqual(checks[0].HistoricalLimits, want) || checks[1].HistoricalLimits != nil {
		t.Fatalf("network coverage did not reach the check projection: %+v", checks)
	}
	dest := filepath.Join(t.TempDir(), "coverage.db")
	manifest := tu.Must(snapshot(e.Ctx, e.Store.DB(), dest, "manual", e.Clock.Now(), func(ctx context.Context, path string) (LedgerSummary, error) {
		return verifyFile(ctx, path, testConfigFor(path))
	}, manifestVersionKey))
	if manifest.Ledger.Status != "ok" || !reflect.DeepEqual(manifest.Ledger.HistoricalLimits, want) {
		t.Fatalf("snapshot coverage = %+v", manifest.Ledger)
	}
	loaded := tu.Must(readManifest(dest + ".manifest.json"))
	if !reflect.DeepEqual(loaded.Ledger.HistoricalLimits, want) {
		t.Fatalf("saved manifest lost coverage: %+v", loaded.Ledger.HistoricalLimits)
	}
	if ok, err := checkManifestAuth(loaded, manifestVersionKey); !ok || err != nil {
		t.Fatalf("new covered snapshot did not authenticate: %v", err)
	}
}

// CRYPTO-02: these canonical v2 payloads predate historical_limit_coverage but
// already contain platform coverage. Their bytes and signatures stay valid.
// The older v1 payload vectors remain in TestManifestSignatureLegacyCompatibility.
func TestHistoricalLimitManifestPreservesOlderV2Signatures(t *testing.T) {
	const prefix = `{"format":1,"created_at":"2026-01-02T03:04:05Z","reason":"manual","app_version":"v1.0","schema_version":23,"file":"","bytes":123,"sha256":"abcdef","integrity_check":"ok","ledger_verification":{"status":"ok","networks":2,"failing_networks":0,"platform_audit":"ok"},`
	for _, tc := range []struct{ name, suffix string }{
		{"plain", `"encrypted":false,"hmac_version":2}`},
		{"encrypted", `"encrypted":true,"cipher":"AES-256-GCM","plaintext_bytes":100,"plaintext_sha256":"012345","hmac_version":2}`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			oldPayload := prefix + tc.suffix
			var manifest Manifest
			if err := json.Unmarshal([]byte(oldPayload), &manifest); err != nil {
				t.Fatal(err)
			}
			if manifest.Ledger.HistoricalLimits != nil {
				t.Fatal("older manifest gained a historical coverage claim")
			}
			if got := string(tu.Must(json.Marshal(manifest))); got != oldPayload {
				t.Fatalf("older canonical payload changed: %s", got)
			}
			mac := hmac.New(sha256.New, manifestVersionKey)
			_, _ = mac.Write([]byte("circula-backup-v2\n" + oldPayload))
			manifest.HMAC = hex.EncodeToString(mac.Sum(nil))
			if ok, err := checkManifestAuth(&manifest, manifestVersionKey); !ok || err != nil {
				t.Fatalf("older signature failed: %v", err)
			}
		})
	}
}

func TestHistoricalLimitManifestAuthenticatesEveryCoverageField(t *testing.T) {
	for _, encrypted := range []bool{false, true} {
		manifest := manifestVersionFixture()
		manifest.HMACVersion, manifest.Encrypted = 2, encrypted
		if encrypted {
			manifest.Cipher, manifest.PlaintextBytes, manifest.PlaintextSHA256 = "AES-256-GCM", 100, "012345"
		}
		manifest.Ledger.HistoricalLimits = &service.HistoricalLimitCoverage{Status: "partial", CheckedDebits: 1, SystemDebits: 2, LegacyUnverifiedDebits: 3, UnverifiableDebits: 4}
		manifest.HMAC = manifestMAC(&manifest, manifestVersionKey)
		for name, mutate := range map[string]func(*service.HistoricalLimitCoverage){
			"status":       func(c *service.HistoricalLimitCoverage) { c.Status = "complete" },
			"checked":      func(c *service.HistoricalLimitCoverage) { c.CheckedDebits++ },
			"system":       func(c *service.HistoricalLimitCoverage) { c.SystemDebits++ },
			"legacy":       func(c *service.HistoricalLimitCoverage) { c.LegacyUnverifiedDebits++ },
			"unverifiable": func(c *service.HistoricalLimitCoverage) { c.UnverifiableDebits++ },
		} {
			copy := manifest
			coverage := *manifest.Ledger.HistoricalLimits
			copy.Ledger.HistoricalLimits = &coverage
			mutate(copy.Ledger.HistoricalLimits)
			if ok, err := checkManifestAuth(&copy, manifestVersionKey); ok || err == nil {
				t.Errorf("encrypted=%v: altered %s authenticated", encrypted, name)
			}
		}
		copy := manifest
		copy.Ledger.HistoricalLimits = nil
		if ok, err := checkManifestAuth(&copy, manifestVersionKey); ok || err == nil {
			t.Errorf("encrypted=%v: omitted coverage authenticated", encrypted)
		}
		var roundTrip Manifest
		if err := json.Unmarshal(tu.Must(json.Marshal(manifest)), &roundTrip); err != nil {
			t.Fatal(err)
		}
		if ok, err := checkManifestAuth(&roundTrip, manifestVersionKey); !ok || err != nil {
			t.Fatalf("encrypted=%v: unchanged coverage failed: %v", encrypted, err)
		}
	}
}
