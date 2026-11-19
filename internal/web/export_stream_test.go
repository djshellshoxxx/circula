package web

import (
	"archive/zip"
	"bytes"
	"context"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/djshellshoxxx/circula/internal/service"
	"github.com/djshellshoxxx/circula/internal/store"
	"github.com/djshellshoxxx/circula/internal/testutil"
)

// exportReadFailureStore injects a read failure after real SQLite rows have
// reached the export writer. This qualifies the adapter's failure behavior,
// not the SQLite driver's behavior on an underlying device failure.
type exportReadFailureStore struct {
	store.Store
	fail      atomic.Bool
	triggered atomic.Bool
}

func (s *exportReadFailureStore) View(ctx context.Context, fn func(store.Tx) error) error {
	return s.Store.View(ctx, func(tx store.Tx) error {
		return fn(exportReadFailureTx{Tx: tx, fault: s})
	})
}

type exportReadFailureTx struct {
	store.Tx
	fault *exportReadFailureStore
}

func (tx exportReadFailureTx) StreamTable(table store.ExportTable, netID, memberID string, fn func([]string, []any) error) error {
	if err := tx.Tx.StreamTable(table, netID, memberID, fn); err != nil {
		return err
	}
	if tx.fault.fail.Load() {
		tx.fault.triggered.Store(true)
		return errors.New("injected export read failure")
	}
	return nil
}

func exportHTTPFixture(t *testing.T) (*testutil.Env, *server, *rc, *exportReadFailureStore, *bytes.Buffer) {
	t.Helper()
	var fault *exportReadFailureStore
	env := testutil.NewEnvWith(t, func(st store.Store) store.Store {
		fault = &exportReadFailureStore{Store: st}
		return fault
	})
	admin := env.Person("export-admin@example.org")
	net, member := env.Network(admin, "export", nil)
	logs := &bytes.Buffer{}
	s := newServer(env.Svc, Options{
		CSRFKey: []byte("0123456789abcdef0123456789abcdef"),
		Clock:   env.Clock,
		Logger:  slog.New(slog.NewJSONHandler(logs, nil)),
	})
	c := &rc{s: s, ctx: env.Ctx, actor: admin, Net: net, Me: member}
	return env, s, c, fault, logs
}

// invokeWebExport runs the download through the actual web recovery layer.
// ErrAbortHandler must escape that layer so net/http can interrupt the stream.
func invokeWebExport(s *server, c *rc, w http.ResponseWriter, scope service.ExportScope, format service.ExportFormat) (panicked any) {
	defer func() { panicked = recover() }()
	h := s.wrap(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		s.sendExport(w, r, c, scope, format)
	}))
	r := httptest.NewRequest(http.MethodGet, "/n/export/export", nil)
	r.Header.Set("X-Request-Id", "web-export-test")
	h.ServeHTTP(w, r)
	return nil
}

func assertWebExportAborted(t *testing.T, panicked any, rec *httptest.ResponseRecorder, logs *bytes.Buffer, reason string) {
	t.Helper()
	if panicked != http.ErrAbortHandler {
		t.Fatalf("interrupted export returned normally or with the wrong panic: %v", panicked)
	}
	if _, err := zip.NewReader(bytes.NewReader(rec.Body.Bytes()), int64(rec.Body.Len())); err == nil {
		t.Fatal("interrupted export was finalized as a valid partial ZIP")
	}
	if strings.Contains(strings.ToLower(rec.Body.String()), "<html") {
		t.Fatal("recovery appended an HTML error page to the ZIP response")
	}
	entries := logEntries(t, logs.String())
	if len(entries) != 1 || entries[0]["msg"] != "web export interrupted" ||
		entries[0]["request_id"] != rec.Header().Get("X-Request-Id") || entries[0]["err"] != reason {
		t.Fatalf("interruption must produce one request-correlated error: %s", logs.String())
	}
}

func TestWebExportAbortsAfterReadFailure(t *testing.T) {
	for _, format := range []service.ExportFormat{service.FormatCSV, service.FormatJSON} {
		t.Run(string(format), func(t *testing.T) {
			_, s, c, fault, logs := exportHTTPFixture(t)
			fault.fail.Store(true)
			rec := httptest.NewRecorder()
			panicked := invokeWebExport(s, c, rec, service.ExportFull, format)
			if !fault.triggered.Load() {
				t.Fatal("export never reached the injected read failure")
			}
			file := "members.csv"
			if format == service.FormatJSON {
				file = "full.json"
			}
			assertWebExportAborted(t, panicked, rec, logs, "export "+file+": injected export read failure")
		})
	}
}

// The end-of-central-directory record is emitted by zip.Writer.Close. Track
// bytes across Write boundaries so this fault targets archive finalization.
type exportCloseFailureWriter struct {
	*httptest.ResponseRecorder
	observed []byte
	failed   bool
}

func (w *exportCloseFailureWriter) Write(p []byte) (int, error) {
	w.observed = append(w.observed, p...)
	if bytes.Contains(w.observed, []byte{'P', 'K', 5, 6}) {
		w.failed = true
		return 0, errors.New("injected export close failure")
	}
	return w.ResponseRecorder.Write(p)
}

func TestWebExportAbortsAfterCloseFailure(t *testing.T) {
	for _, format := range []service.ExportFormat{service.FormatCSV, service.FormatJSON} {
		t.Run(string(format), func(t *testing.T) {
			_, s, c, _, logs := exportHTTPFixture(t)
			w := &exportCloseFailureWriter{ResponseRecorder: httptest.NewRecorder()}
			panicked := invokeWebExport(s, c, w, service.ExportFull, format)
			if !w.failed {
				t.Fatal("export never reached the injected ZIP finalization failure")
			}
			assertWebExportAborted(t, panicked, w.ResponseRecorder, logs, "injected export close failure")
		})
	}
}

func TestWebExportCompletesValidArchive(t *testing.T) {
	for _, format := range []service.ExportFormat{service.FormatCSV, service.FormatJSON} {
		t.Run(string(format), func(t *testing.T) {
			_, s, c, _, logs := exportHTTPFixture(t)
			rec := httptest.NewRecorder()
			if panicked := invokeWebExport(s, c, rec, service.ExportFull, format); panicked != nil {
				t.Fatalf("successful export panicked: %v", panicked)
			}
			if rec.Code != http.StatusOK || rec.Header().Get("Content-Type") != "application/zip" {
				t.Fatalf("export response: status %d, headers %v", rec.Code, rec.Header())
			}
			zr, err := zip.NewReader(bytes.NewReader(rec.Body.Bytes()), int64(rec.Body.Len()))
			if err != nil {
				t.Fatal(err)
			}
			manifest := false
			for _, f := range zr.File {
				r, err := f.Open()
				if err != nil {
					t.Fatal(err)
				}
				_, readErr := io.Copy(io.Discard, r)
				closeErr := r.Close()
				if readErr != nil || closeErr != nil {
					t.Fatalf("archive file %s: read %v, close %v", f.Name, readErr, closeErr)
				}
				manifest = manifest || f.Name == "manifest.json"
			}
			if !manifest || len(zr.File) < 2 {
				t.Fatalf("incomplete archive: manifest %v, files %d", manifest, len(zr.File))
			}
			if logs.Len() != 0 {
				t.Fatalf("successful export logged an error: %s", logs.String())
			}
		})
	}
}

func TestWebExportEarlyRefusalKeepsErrorPage(t *testing.T) {
	for _, format := range []service.ExportFormat{service.FormatCSV, service.FormatJSON} {
		t.Run(string(format), func(t *testing.T) {
			env, s, c, _, _ := exportHTTPFixture(t)
			c.actor, c.Me = env.Member(c.actor, c.Net, "export-member@example.org")
			rec := httptest.NewRecorder()
			if panicked := invokeWebExport(s, c, rec, service.ExportFull, format); panicked != nil {
				t.Fatalf("early refusal aborted the response: %v", panicked)
			}
			if rec.Code != http.StatusForbidden || !strings.HasPrefix(rec.Header().Get("Content-Type"), "text/html") {
				t.Fatalf("early refusal: status %d, headers %v", rec.Code, rec.Header())
			}
			if rec.Header().Get("Content-Disposition") != "" || rec.Body.Len() == 0 {
				t.Fatal("early refusal must render an error page without download headers")
			}
		})
	}
}
