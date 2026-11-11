package main

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/netip"
	"strings"
	"testing"
	"time"

	"github.com/djshellshoxxx/circula/internal/domain"
	"github.com/djshellshoxxx/circula/internal/store"
)

// breakLedger changes a cached balance so verification fails for the network.
func breakLedger(t *testing.T, db string) {
	t.Helper()
	raw, err := openRaw(db)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = raw.Close() }()
	if _, err := raw.Exec(`UPDATE accounts SET balance = balance + 7 WHERE rowid = (SELECT MIN(rowid) FROM accounts)`); err != nil {
		t.Fatal(err)
	}
}

// CMD-07 / OPS-033: a failing network is frozen, not the whole instance, and
// thaws when verification passes again.
func TestApplyFreezeFreezesAndThaws(t *testing.T) {
	ctx := context.Background()
	db := seedDatabase(t, t.TempDir())
	breakLedger(t, db)
	st, err := store.Open(ctx, db)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = st.Close() }()
	svc, err := newService(st, testConfigFor(db))
	if err != nil {
		t.Fatal(err)
	}
	checks, err := verifyAllNetworks(ctx, svc, st.DB(), "t", "")
	if err != nil || len(checks) != 1 || checks[0].OK {
		t.Fatalf("checks = %+v err=%v", checks, err)
	}
	applyFreeze(svc, checks, discardLogger())
	if !svc.IsFrozen(checks[0].ID) {
		t.Fatal("failing network not frozen")
	}
	_, err = svc.ExpireProposals(ctx, domain.SystemActor("t"), 10)
	if err != nil {
		t.Fatalf("expiry must skip frozen networks, not fail: %v", err)
	}
	// repaired: the same network passes and thaws
	checks[0].OK = true
	applyFreeze(svc, checks, discardLogger())
	if svc.IsFrozen(checks[0].ID) {
		t.Fatal("passing network still frozen")
	}
}

// The instance starts and serves although a network fails verification.
func TestServeStartsWithFailingNetwork(t *testing.T) {
	db := seedDatabase(t, t.TempDir())
	breakLedger(t, db)
	cfg := testConfigFor(db)
	cfg.Listen = freeAddr(t)
	cfg.PublicBaseURL = "https://x.test"
	cfg.MigrateMode = "explicit"
	cfg.MaxBodyBytes = 1 << 20
	cfg.RequestTimeout = 5 * time.Second
	cfg.BackupInterval = 0
	cfg.MetricsAllow = []netip.Prefix{netip.MustParsePrefix("127.0.0.0/8")}

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- runServe(ctx, cfg, discardLogger()) }()

	base := "http://" + cfg.Listen
	deadline := time.Now().Add(10 * time.Second)
	var ready bool
	for time.Now().Before(deadline) && !ready {
		select {
		case err := <-done:
			t.Fatalf("serve exited: %v", err)
		default:
		}
		resp, err := http.Get(base + "/readyz") //nolint:gosec,noctx // local test server
		if err == nil {
			ready = resp.StatusCode == 200
			_ = resp.Body.Close()
		}
		time.Sleep(50 * time.Millisecond)
	}
	if !ready {
		t.Fatal("instance never became ready")
	}
	resp, err := http.Get(base + "/metrics") //nolint:gosec,noctx // local test server
	if err != nil {
		t.Fatal(err)
	}
	body, _ := io.ReadAll(resp.Body)
	_ = resp.Body.Close()
	if !strings.Contains(string(body), "circula_frozen_networks 1") {
		t.Errorf("frozen network gauge missing:\n%s", body)
	}
	cancel()
	if err := <-done; err != nil {
		t.Fatalf("serve returned %v", err)
	}
}

func freeAddr(t *testing.T) string {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = ln.Close() }()
	return fmt.Sprint(ln.Addr())
}
