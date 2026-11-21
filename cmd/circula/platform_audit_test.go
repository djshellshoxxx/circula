package main

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"log/slog"
	"strings"
	"testing"
	"time"

	"github.com/djshellshoxxx/circula/internal/store"
	tu "github.com/djshellshoxxx/circula/internal/testutil"
)

// Only repository responses differ in this fixture; the persisted audit and
// ledger remain unchanged and keep all of their append-only protections.
type platformAuditStore struct {
	store.Store
	mismatch bool
	failure  error
	writes   int
}

func (s *platformAuditStore) View(ctx context.Context, fn func(store.Tx) error) error {
	return s.Store.View(ctx, func(tx store.Tx) error {
		return fn(&platformAuditTx{Tx: tx, GovRepo: tx.(store.GovRepo), control: s})
	})
}

func (s *platformAuditStore) Update(context.Context, func(store.Tx) error) error {
	s.writes++
	return errors.New("routine verification attempted a write")
}

type platformAuditTx struct {
	store.Tx
	store.GovRepo
	control *platformAuditStore
}

func (tx *platformAuditTx) AuditHead(scope string) (int64, string, bool, error) {
	seq, hash, found, err := tx.Tx.AuditHead(scope)
	if scope != "" || err != nil {
		return seq, hash, found, err
	}
	if tx.control.failure != nil {
		return 0, "", false, tx.control.failure
	}
	if tx.control.mismatch {
		return seq + 1, hash, true, nil
	}
	return seq, hash, found, nil
}

func TestRoutineVerificationIncludesPlatformWithoutFreezingNetworks(t *testing.T) {
	ctx := context.Background()
	db := seedDatabase(t, t.TempDir())
	st := tu.Must(store.Open(ctx, db))
	defer func() { _ = st.Close() }()
	wrapped := &platformAuditStore{Store: st}
	svc := tu.Must(newService(wrapped, testConfigFor(db)))
	before := tu.Snapshot(t, st.DB())
	var logs bytes.Buffer
	j := jobRunner{svc: svc, metrics: newMetrics(time.Now()), logger: slog.New(slog.NewTextHandler(&logs, nil))}
	for _, inconsistent := range []bool{false, true, false} {
		wrapped.mismatch = inconsistent
		checks := tu.Must(verifyAllNetworks(ctx, svc, "routine-test", ""))
		summary := summarise(checks)
		wantStatus := "ok"
		wantMetric := 1
		if inconsistent {
			wantStatus, wantMetric = "failed", 0
		}
		if summary.Status != wantStatus || summary.PlatformAudit != wantStatus || summary.Networks != 1 || summary.Failing != 0 {
			t.Fatalf("platform result changed network coverage or counts: %+v", summary)
		}
		var output bytes.Buffer
		if printChecks(&output, checks) == inconsistent || !strings.Contains(output.String(), "platform audit:") {
			t.Fatalf("platform result missing from operator output: %s", output.String())
		}
		if err := j.verify(ctx); err != nil {
			t.Fatal(err)
		}
		if j.metrics.ledgerOK != wantMetric || j.metrics.ledgerFailing != 0 || len(svc.FrozenNetworks()) != 0 {
			t.Fatalf("platform result must affect the overall metric without freezing a network: ok=%d failing=%d frozen=%v",
				j.metrics.ledgerOK, j.metrics.ledgerFailing, svc.FrozenNetworks())
		}
	}
	if !strings.Contains(logs.String(), "platform audit verification failed") {
		t.Fatal("platform finding was not logged")
	}
	wrapped.failure = errors.New("platform audit read unavailable")
	if _, err := verifyAllNetworks(ctx, svc, "read-error", ""); !errors.Is(err, wrapped.failure) {
		t.Fatalf("platform read error was replaced: %v", err)
	}
	if err := j.verify(ctx); !errors.Is(err, wrapped.failure) {
		t.Fatalf("scheduled verification did not report its read error: %v", err)
	}
	if wrapped.writes != 0 {
		t.Fatalf("verification entered %d write units", wrapped.writes)
	}
	if changes := before.Changed(tu.Snapshot(t, st.DB())); len(changes) != 0 {
		t.Fatalf("routine verification changed stored state: %v", changes)
	}
}

func TestVerifyLedgerCLIReportsPlatformCoverage(t *testing.T) {
	db := seedDatabase(t, t.TempDir())
	for _, filter := range []string{"", "test-net", "missing-network"} {
		var output, errOut bytes.Buffer
		code := run([]string{"verify-ledger", "-database", db, "-network", filter}, &output, &errOut)
		if filter == "missing-network" {
			if code != exitFailure || !strings.Contains(errOut.String(), "no network matches") {
				t.Fatalf("missing filter was treated as platform-only success: %d %s", code, errOut.String())
			}
			continue
		}
		if code != exitOK || !strings.Contains(output.String(), "network test-net: ok") || !strings.Contains(output.String(), "platform audit: ok") {
			t.Fatalf("operator output omitted a verified scope: %d %s %s", code, output.String(), errOut.String())
		}
	}
}

func TestRoutineVerificationCoversEmptyPlatformWithoutNetworks(t *testing.T) {
	ctx := context.Background()
	db := t.TempDir() + "/empty.db"
	st := tu.Must(store.Open(ctx, db))
	defer func() { _ = st.Close() }()
	svc := tu.Must(newService(st, testConfigFor(db)))
	checks := tu.Must(verifyAllNetworks(ctx, svc, "empty-verification", ""))
	summary := summarise(checks)
	if summary.Status != "ok" || summary.PlatformAudit != "ok" || summary.Networks != 0 || summary.Failing != 0 {
		t.Fatalf("empty platform coverage is not explicit: %+v", summary)
	}
	var output bytes.Buffer
	if !printChecks(&output, checks) || !strings.Contains(output.String(), "platform audit: ok") || !strings.Contains(output.String(), "no networks to verify") {
		t.Fatalf("empty verification output: %s", output.String())
	}
}

func TestManifestPlatformCoveragePreservesLegacySignature(t *testing.T) {
	// Exact canonical v2 payload produced before platform_audit existed.
	const legacy = `{"format":1,"created_at":"2026-01-02T03:04:05Z","reason":"manual","app_version":"v1.0","schema_version":16,"file":"","bytes":123,"sha256":"abcdef","integrity_check":"ok","ledger_verification":{"status":"not-run","networks":2,"failing_networks":0,"reason":"older schema"},"encrypted":false,"hmac_version":2}`
	var manifest Manifest
	if err := json.Unmarshal([]byte(legacy), &manifest); err != nil {
		t.Fatal(err)
	}
	if manifest.Ledger.PlatformAudit != "" {
		t.Fatal("historical manifest was relabelled as platform-covered")
	}
	canonical := tu.Must(json.Marshal(manifest))
	if string(canonical) != legacy {
		t.Fatalf("historical canonical bytes changed: %s", canonical)
	}
	mac := hmac.New(sha256.New, manifestVersionKey)
	_, _ = mac.Write([]byte("circula-backup-v2\n" + legacy))
	manifest.HMAC = hex.EncodeToString(mac.Sum(nil))
	if authenticated, err := checkManifestAuth(&manifest, manifestVersionKey); !authenticated || err != nil {
		t.Fatalf("historical signature stopped verifying: %v", err)
	}
	manifest.Ledger.PlatformAudit = "ok"
	if authenticated, err := checkManifestAuth(&manifest, manifestVersionKey); authenticated || err == nil {
		t.Fatal("added platform coverage was not authenticated")
	}
	manifest.HMAC = manifestMAC(&manifest, manifestVersionKey)
	if authenticated, err := checkManifestAuth(&manifest, manifestVersionKey); !authenticated || err != nil {
		t.Fatalf("new explicit coverage did not authenticate: %v", err)
	}
	manifest.Ledger.PlatformAudit = "failed"
	if authenticated, err := checkManifestAuth(&manifest, manifestVersionKey); authenticated || err == nil {
		t.Fatal("changed platform coverage did not invalidate its signature")
	}
}
