package main

import (
	"archive/zip"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/djshellshoxxx/circula/internal/domain"
	"github.com/djshellshoxxx/circula/internal/httpx"
	"github.com/djshellshoxxx/circula/internal/service"
	"github.com/djshellshoxxx/circula/internal/store"
	"github.com/djshellshoxxx/circula/internal/testutil"
	"github.com/djshellshoxxx/circula/internal/web"
)

// committedExportFaultStore waits for the client to confirm receipt of ZIP
// bytes before failing a real SQLite export read. It qualifies the HTTP
// interruption contract, not the storage driver's device-failure behavior.
type committedExportFaultStore struct {
	store.Store
	release <-chan struct{}
	fired   atomic.Bool
}

func (s *committedExportFaultStore) View(ctx context.Context, fn func(store.Tx) error) error {
	return s.Store.View(ctx, func(tx store.Tx) error {
		return fn(committedExportFaultTx{Tx: tx, ctx: ctx, fault: s})
	})
}

type committedExportFaultTx struct {
	store.Tx
	ctx   context.Context
	fault *committedExportFaultStore
}

func (tx committedExportFaultTx) StreamTable(table store.ExportTable, netID, memberID string, fn func([]string, []any) error) error {
	if err := tx.Tx.StreamTable(table, netID, memberID, fn); err != nil {
		return err
	}
	if table != store.TblListings {
		return nil
	}
	select {
	case <-tx.fault.release:
		tx.fault.fired.Store(true)
		return errors.New("injected export failure after client received bytes")
	case <-tx.ctx.Done():
		return tx.ctx.Err()
	}
}

// Both web recovery and the outer operations middleware must let an interrupted
// download reach net/http. A recorder-only test cannot prove that the client
// receives a truncated HTTP body after the successful headers were committed.
func TestWebExportCommittedReadFailureAbortsHTTP(t *testing.T) {
	for _, format := range []service.ExportFormat{service.FormatCSV, service.FormatJSON} {
		t.Run(string(format), func(t *testing.T) {
			releaseFault := make(chan struct{})
			var releaseOnce sync.Once
			release := func() { releaseOnce.Do(func() { close(releaseFault) }) }
			defer release() // also unblock the handler if a client assertion fails
			var fault *committedExportFaultStore
			env := testutil.NewEnvWith(t, func(st store.Store) store.Store {
				fault = &committedExportFaultStore{Store: st, release: releaseFault}
				return fault
			})
			admin := env.Person("export-http@example.org")
			net, _ := env.Network(admin, "export-http", nil)
			// Deterministic, poorly compressible descriptions exceed the ZIP
			// compressor and HTTP output buffers without artificial response writes.
			for i := 0; i < 64; i++ {
				var desc strings.Builder
				for desc.Len() < 2000 {
					digest := sha256.Sum256([]byte(fmt.Sprintf("%d/%d", i, desc.Len())))
					desc.WriteString(base64.RawStdEncoding.EncodeToString(digest[:]))
				}
				testutil.Must(env.Svc.CreateListing(env.Ctx, admin, net.ID, service.ListingInput{
					Type: domain.ListingOffer, Title: fmt.Sprintf("Export fixture %02d", i),
					Description: desc.String()[:2000], IdempotencyKey: env.Key(),
				}))
			}
			session, _ := testutil.Must2(env.Svc.Login(env.Ctx, "export-http@example.org", "correct horse battery"))
			key := []byte("0123456789abcdef0123456789abcdef")
			logger := slog.New(slog.NewTextHandler(io.Discard, nil))
			ui := web.NewHandlerWith(env.Svc, web.Options{CSRFKey: key, Clock: env.Clock, Logger: logger})
			ops := &opsServer{
				cfg:     &Config{MaxBodyBytes: 1024, RequestTimeout: 15 * time.Second},
				db:      env.Store.DB(),
				logger:  logger,
				metrics: newMetrics(time.Now()),
			}
			srv := httptest.NewServer(ops.routes(http.NotFoundHandler(), ui))
			t.Cleanup(srv.Close)
			client := srv.Client()
			client.Timeout = 10 * time.Second
			form := url.Values{"scope": {"full"}, "format": {string(format)}}
			req, err := http.NewRequest(http.MethodPost, srv.URL+"/n/export-http/admin/exports", strings.NewReader(form.Encode()))
			if err != nil {
				t.Fatal(err)
			}
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			req.Header.Set("X-CSRF-Token", (httpx.CSRF{Key: key}).Token(session))
			req.AddCookie(&http.Cookie{Name: httpx.SessionCookie, Value: session})
			resp, err := client.Do(req)
			if err != nil {
				t.Fatalf("export failed before a response was committed: %v", err)
			}
			defer func() { _ = resp.Body.Close() }()
			if resp.StatusCode != http.StatusOK || resp.Header.Get("Content-Type") != "application/zip" {
				t.Fatalf("export response: status %d, headers %v", resp.StatusCode, resp.Header)
			}
			prefix := make([]byte, 4096)
			if _, err := io.ReadFull(resp.Body, prefix); err != nil {
				t.Fatalf("client did not receive ZIP bytes before the fault: %v", err)
			}
			if !bytes.HasPrefix(prefix, []byte{'P', 'K', 3, 4}) || fault.fired.Load() {
				t.Fatal("fault must wait until the client receives the beginning of the ZIP")
			}
			release()
			rest, readErr := io.ReadAll(resp.Body)
			if !fault.fired.Load() {
				t.Fatal("download ended without reaching the injected read failure")
			}
			if !errors.Is(readErr, io.ErrUnexpectedEOF) {
				t.Fatalf("interrupted HTTP body read = %v, want unexpected EOF", readErr)
			}
			body := append(prefix, rest...)
			if _, err := zip.NewReader(bytes.NewReader(body), int64(len(body))); err == nil {
				t.Fatal("interrupted export was delivered as a valid partial ZIP")
			}
		})
	}
}
