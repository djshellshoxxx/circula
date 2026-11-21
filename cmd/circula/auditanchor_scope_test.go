package main

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"testing"
	"time"

	tu "github.com/djshellshoxxx/circula/internal/testutil"
)

// A minimal read fixture supplies only checkpoint data. It does not alter a
// Circula database or disable any audit/ledger integrity protection.
func anchorScopeFixture(t *testing.T, platform, network []anchorCheckpoint) (*sql.DB, string) {
	t.Helper()
	path := filepath.Join(t.TempDir(), "heads.db")
	db := tu.Must(openRaw(path))
	t.Cleanup(func() { _ = db.Close() })
	_, err := db.Exec(`CREATE TABLE networks(id TEXT PRIMARY KEY, slug TEXT NOT NULL);
		CREATE TABLE audit_events(id TEXT PRIMARY KEY, network_id TEXT NOT NULL, seq INTEGER NOT NULL,
		occurred_at INTEGER NOT NULL, hash TEXT NOT NULL, UNIQUE(network_id, seq));`)
	if err != nil {
		t.Fatal(err)
	}
	if network != nil {
		if _, err := db.Exec(`INSERT INTO networks VALUES ('network-one', 'test-net')`); err != nil {
			t.Fatal(err)
		}
	}
	for _, scope := range []struct {
		id    string
		base  int64
		heads []anchorCheckpoint
	}{{"", 1000, platform}, {"network-one", 2000, network}} {
		for _, head := range scope.heads {
			_, err := db.Exec(`INSERT INTO audit_events(id, network_id, seq, occurred_at, hash) VALUES (?, ?, ?, ?, ?)`,
				fmt.Sprintf("%s-event-%d", scope.id, head.Seq), scope.id, head.Seq, scope.base+head.Seq, head.Hash)
			if err != nil {
				t.Fatal(err)
			}
		}
	}
	return db, path
}

func TestAuditAnchorV2ExplicitScopesAndGrowth(t *testing.T) {
	platform := []anchorCheckpoint{{1, strings.Repeat("a", 64)}, {2, strings.Repeat("c", 64)}}
	network := []anchorCheckpoint{{1, strings.Repeat("b", 64)}, {2, strings.Repeat("d", 64)}}
	db, _ := anchorScopeFixture(t, platform, network)
	ctx := context.Background()
	a := tu.Must(buildAnchor(ctx, db, nil, time.Now()))
	if a.Version != 2 || a.Platform == nil || *a.Platform != platform[1] || len(a.Networks) != 1 || a.Networks[0].Seq != 2 {
		t.Fatalf("new anchor omitted a covered scope: %+v", a)
	}
	if findings := tu.Must(verifyAnchor(ctx, db, a, nil)); len(findings) != 0 {
		t.Fatalf("current checkpoints did not verify: %v", findings)
	}
	// This historical checkpoint covers the first event of each complete
	// fixture scope. Both scopes have since legitimately grown.
	a.Platform = &platform[0]
	a.Networks[0].Seq, a.Networks[0].Hash = network[0].Seq, network[0].Hash
	if findings := tu.Must(verifyAnchor(ctx, db, a, nil)); len(findings) != 0 {
		t.Fatalf("legitimate growth did not preserve the checkpoint: %v", findings)
	}
	ctx, cancel := context.WithCancel(ctx)
	cancel()
	if findings, err := verifyAnchor(ctx, db, a, nil); !errors.Is(err, context.Canceled) || len(findings) != 0 {
		t.Fatalf("read cancellation did not propagate: findings=%v err=%v", findings, err)
	}
}

func TestAuditAnchorV2ReportsPlatformCheckpointProblems(t *testing.T) {
	for _, tc := range []struct {
		name, finding string
		live          []anchorCheckpoint
		checkpoint    anchorCheckpoint
	}{
		{"different hash", "platform audit: event 1 hash differs", []anchorCheckpoint{{1, strings.Repeat("a", 64)}}, anchorCheckpoint{1, strings.Repeat("b", 64)}},
		{"missing event", "platform audit: event 2 is missing", []anchorCheckpoint{{1, strings.Repeat("a", 64)}}, anchorCheckpoint{2, strings.Repeat("b", 64)}},
		{"no platform events", "platform audit: event 1 is missing", nil, anchorCheckpoint{1, strings.Repeat("b", 64)}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			db, _ := anchorScopeFixture(t, tc.live, []anchorCheckpoint{{1, strings.Repeat("c", 64)}})
			a := tu.Must(buildAnchor(context.Background(), db, nil, time.Now()))
			a.Platform = &tc.checkpoint
			findings := tu.Must(verifyAnchor(context.Background(), db, a, nil))
			if !strings.Contains(strings.Join(findings, "\n"), tc.finding) {
				t.Fatalf("platform checkpoint problem was not reported: %v", findings)
			}
		})
	}
}

func TestAuditAnchorV2EmptyScopeSemantics(t *testing.T) {
	ctx := context.Background()
	empty, _ := anchorScopeFixture(t, nil, nil)
	a := tu.Must(buildAnchor(ctx, empty, nil, time.Now()))
	if a.Platform == nil || a.Platform.Seq != 0 || a.Platform.Hash != "" || a.DBID != "" || len(a.Networks) != 0 {
		t.Fatalf("empty database must have an explicit unbound zero checkpoint: %+v", a)
	}
	encoded := tu.Must(json.Marshal(a))
	if !bytes.Contains(encoded, []byte(`"platform":{"seq":0,"hash":""}`)) {
		t.Fatalf("empty platform coverage was omitted: %s", encoded)
	}
	if findings := tu.Must(verifyAnchor(ctx, empty, a, nil)); len(findings) != 0 {
		t.Fatal(findings)
	}
	// An empty checkpoint covers no past events and has no database identity
	// yet, so ordinary first writes cannot contradict it.
	grown, _ := anchorScopeFixture(t, []anchorCheckpoint{{1, strings.Repeat("a", 64)}}, nil)
	if findings := tu.Must(verifyAnchor(ctx, grown, a, nil)); len(findings) != 0 {
		t.Fatalf("first events contradicted an empty checkpoint: %v", findings)
	}
}

func TestAuditAnchorV1CanonicalMACAndLimitedCoverage(t *testing.T) {
	ctx := context.Background()
	networkHash := strings.Repeat("b", 64)
	db, path := anchorScopeFixture(t, []anchorCheckpoint{{1, strings.Repeat("a", 64)}}, []anchorCheckpoint{{1, networkHash}})
	dbID := tu.Must(anchorDBID(ctx, db))
	// Exact field order and omissions of the original v1 writer.
	legacy := fmt.Sprintf(`{"version":1,"created_at":"2026-11-20T12:00:00Z","db_id":%q,"networks":[{"network_id":"network-one","slug":"test-net","seq":1,"hash":%q}]}`, dbID, networkHash)
	var a auditAnchor
	if err := json.Unmarshal([]byte(legacy), &a); err != nil {
		t.Fatal(err)
	}
	if got := tu.Must(json.Marshal(a)); string(got) != legacy || a.Platform != nil {
		t.Fatalf("legacy canonical representation changed: %s", got)
	}
	key := []byte("historical-anchor-test-key")
	mac := hmac.New(sha256.New, key)
	_, _ = mac.Write([]byte(legacy))
	a.MAC = hex.EncodeToString(mac.Sum(nil))
	if findings := tu.Must(verifyAnchor(ctx, db, &a, key)); len(findings) != 0 {
		t.Fatalf("legacy signature no longer verifies: %v", findings)
	}
	emptyLegacy := a
	emptyLegacy.Networks = append([]anchorNetwork(nil), a.Networks...)
	emptyLegacy.Networks[0].Seq, emptyLegacy.Networks[0].Hash = 0, ""
	emptyLegacy.MAC = anchorMAC(emptyLegacy, key)
	if findings := tu.Must(verifyAnchor(ctx, db, &emptyLegacy, key)); len(findings) == 0 {
		t.Fatal("legacy v1 unexpectedly acquired zero-checkpoint semantics")
	}
	file := filepath.Join(t.TempDir(), "legacy.json")
	if err := writeAnchorFile(file, &a); err != nil {
		t.Fatal(err)
	}
	t.Setenv(envName("audit-anchor-key-file"), writeSecret(t, string(key), 0o600))
	if code, output := runAnchor(t, path, "verify", file); code != exitOK || !strings.Contains(output, "legacy v1 network-only coverage, platform audit not covered") {
		t.Fatalf("legacy verification overstated its coverage: %d %s", code, output)
	}
}

func TestAuditAnchorScopeValidationAndMAC(t *testing.T) {
	ctx := context.Background()
	db, _ := anchorScopeFixture(t, []anchorCheckpoint{{1, strings.Repeat("a", 64)}}, []anchorCheckpoint{{1, strings.Repeat("b", 64)}})
	key := []byte("scope-authentication-test-key")
	original := tu.Must(buildAnchor(ctx, db, key, time.Now()))
	if findings := tu.Must(verifyAnchor(ctx, db, original, key)); len(findings) != 0 {
		t.Fatal(findings)
	}
	for name, change := range map[string]func(*auditAnchor){
		"missing platform":      func(a *auditAnchor) { a.Platform = nil },
		"missing identity":      func(a *auditAnchor) { a.DBID = "" },
		"legacy platform":       func(a *auditAnchor) { a.Version = 1 },
		"negative sequence":     func(a *auditAnchor) { a.Platform.Seq = -1 },
		"empty sequence hash":   func(a *auditAnchor) { a.Platform.Seq = 0 },
		"nonempty sequence gap": func(a *auditAnchor) { a.Platform.Hash = "" },
		"changed platform":      func(a *auditAnchor) { a.Platform.Hash = strings.Repeat("c", 64) },
		"platform as network":   func(a *auditAnchor) { a.Networks[0].NetworkID = "" },
		"duplicate network":     func(a *auditAnchor) { a.Networks = append(a.Networks, a.Networks[0]) },
	} {
		t.Run(name, func(t *testing.T) {
			candidate := *original
			platform := *original.Platform
			candidate.Platform = &platform
			candidate.Networks = append([]anchorNetwork(nil), original.Networks...)
			change(&candidate)
			if findings := tu.Must(verifyAnchor(ctx, db, &candidate, key)); len(findings) == 0 {
				t.Fatal("invalid or unauthenticated scope metadata was accepted")
			}
		})
	}
}
