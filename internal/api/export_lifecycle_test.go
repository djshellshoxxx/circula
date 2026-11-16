package api

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"
)

func TestExpiredExportLookupReleasesFile(t *testing.T) {
	w := spoolEnv(t, t.TempDir())
	d, e := w.export()
	w.mu.Lock()
	w.now = w.now.Add(jobTTL)
	w.mu.Unlock()
	if got := w.do("GET", "/api/v1/exports/"+d.ID, ""); got.Code != 404 {
		t.Fatalf("expired export: %d", got.Code)
	}
	if _, err := e.spool.f.Stat(); !errors.Is(err, os.ErrClosed) {
		t.Fatalf("expired archive remains open: %v", err)
	}
	if w.h.jobs.bytes != 0 {
		t.Fatal("expired archive remains counted")
	}
}

func TestExportTimerReleasesFileWithoutTraffic(t *testing.T) {
	w := spoolEnv(t, t.TempDir())
	_, e := w.export()
	// Exercise the installed callback without waiting for the production TTL.
	e.timer.Reset(time.Millisecond)
	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		w.h.jobs.mu.Lock()
		gone := len(w.h.jobs.exps) == 0 && w.h.jobs.bytes == 0
		w.h.jobs.mu.Unlock()
		if gone {
			if _, err := e.spool.f.Stat(); !errors.Is(err, os.ErrClosed) {
				t.Fatalf("timer left file open: %v", err)
			}
			return
		}
		time.Sleep(time.Millisecond)
	}
	t.Fatal("timer did not expire the archive")
}

type exportHeaderHook struct {
	*httptest.ResponseRecorder
	onHeader func()
}

func (w *exportHeaderHook) WriteHeader(code int) {
	w.ResponseRecorder.WriteHeader(code)
	if code == http.StatusOK && w.onHeader != nil {
		f := w.onHeader
		w.onHeader = nil
		f()
	}
}

func TestExportExpiryDoesNotTruncateAdmittedDownload(t *testing.T) {
	w := spoolEnv(t, t.TempDir())
	d, e := w.export()
	w.h.jobs.capBytes = e.size()
	r := httptest.NewRequest("GET", "/api/v1/exports/"+d.ID+"/artefact", nil)
	r.Header.Set("Authorization", "Bearer "+w.tok)
	rec := &exportHeaderHook{ResponseRecorder: httptest.NewRecorder(), onHeader: func() {
		w.mu.Lock()
		w.now = w.now.Add(jobTTL)
		w.mu.Unlock()
		w.h.jobs.expire(e)
		// An expired but pinned archive must keep its byte reservation.
		if tk, why := w.h.jobs.beginExport(w.netID, "other", false); why != "full" {
			if tk != nil {
				tk.abort()
			}
			t.Errorf("pinned archive no longer bounds storage: %q", why)
		}
	}}
	w.h.ServeHTTP(rec, r)
	sum := sha256.Sum256(rec.Body.Bytes())
	if rec.Code != 200 || int64(rec.Body.Len()) != d.Bytes || hex.EncodeToString(sum[:]) != *d.SHA256 {
		t.Fatalf("truncated download: status %d, bytes %d/%d", rec.Code, rec.Body.Len(), d.Bytes)
	}
	if _, err := e.spool.f.Stat(); !errors.Is(err, os.ErrClosed) || w.h.jobs.bytes != 0 {
		t.Fatalf("completed expired download not released: %v, bytes %d", err, w.h.jobs.bytes)
	}
}

func TestExportErasureAbortsAdmittedDownload(t *testing.T) {
	w := spoolEnv(t, t.TempDir())
	d, _ := w.exportScope("mine")
	r := httptest.NewRequest("GET", "/api/v1/exports/"+d.ID+"/artefact", nil)
	r.Header.Set("Authorization", "Bearer "+w.tok)
	rec := &exportHeaderHook{ResponseRecorder: httptest.NewRecorder(), onHeader: func() {
		w.h.jobs.dropErased(w.netID, "counterparty")
	}}
	defer func() {
		if got := recover(); got != http.ErrAbortHandler {
			t.Errorf("erased stream was not aborted: %v", got)
		}
		if rec.Body.Len() != 0 || w.h.jobs.bytes != 0 {
			t.Error("erased archive was retained or sent")
		}
	}()
	w.h.ServeHTTP(rec, r)
}

func TestCloseHandlerReleasesExportsAndRejectsRunningCommit(t *testing.T) {
	w := spoolEnv(t, t.TempDir())
	_, e := w.export()
	tk, why := w.h.jobs.beginExport(w.netID, "other", false)
	if why != "" {
		t.Fatal(why)
	}
	CloseHandler(w.h)
	CloseHandler(w.h)
	if tk.commit(&exportEntry{dto: exportDTO{ID: "late"}}) {
		t.Fatal("closed handler accepted a running export")
	}
	if _, err := e.spool.f.Stat(); !errors.Is(err, os.ErrClosed) || w.h.jobs.bytes != 0 {
		t.Fatalf("shutdown retained archive: %v", err)
	}
	if tk, why := w.h.jobs.beginExport(w.netID, "other", false); why == "" || tk != nil {
		t.Fatal("closed handler accepted a new export")
	}
	assertIdle(t, w.h.jobs)
}

type exportWriteHook struct {
	*httptest.ResponseRecorder
	onWrite func()
}

func (w *exportWriteHook) Write(p []byte) (int, error) {
	if w.onWrite != nil {
		f := w.onWrite
		w.onWrite = nil
		f() // the archive chunk is already in memory when erasure happens
	}
	return w.ResponseRecorder.Write(p)
}

func TestExportErasureDuringFinalWriteAbortsResponse(t *testing.T) {
	w := spoolEnv(t, t.TempDir())
	d, _ := w.exportScope("mine")
	r := httptest.NewRequest("GET", "/api/v1/exports/"+d.ID+"/artefact", nil)
	r.Header.Set("Authorization", "Bearer "+w.tok)
	rec := &exportWriteHook{ResponseRecorder: httptest.NewRecorder(), onWrite: func() {
		w.h.jobs.dropErased(w.netID, "counterparty")
	}}
	defer func() {
		if got := recover(); got != http.ErrAbortHandler {
			t.Errorf("revoked final write was reported as complete: %v", got)
		}
	}()
	w.h.ServeHTTP(rec, r)
}
