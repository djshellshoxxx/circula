package api

import (
	"bytes"
	"log/slog"
	"path/filepath"
	"strings"
	"testing"
)

// BR-06: bytes of exports still being written count against the same cap as
// stored artefacts, so parallel large exports cannot exhaust disk or memory
// before the size check at the end. A refused export leaves nothing behind.
func TestExportInFlightBytesAreBounded(t *testing.T) {
	dir := t.TempDir()
	w := spoolEnv(t, dir)
	w.h.jobs.limit = 64 // smaller than any ledger archive
	if rec := w.do("POST", "/api/v1/exports", `{"scope":"ledger","format":"csv"}`); rec.Code != 503 {
		t.Fatalf("export over the in-flight budget: %d %s", rec.Code, rec.Body.String())
	}
	if w.h.jobs.inflight != 0 || w.h.jobs.bytes != 0 || dirEntries(t, dir) != 0 {
		t.Fatalf("refused export left state: inflight=%d bytes=%d files=%d", w.h.jobs.inflight, w.h.jobs.bytes, dirEntries(t, dir))
	}

	// Two writers in flight share one budget: the second is stopped while
	// writing, not after it has finished.
	w.h.jobs.limit = 100
	a := w.h.jobs.newArtefactWriter()
	if _, err := a.Write(make([]byte, 90)); err != nil {
		t.Fatalf("first writer within budget: %v", err)
	}
	b := w.h.jobs.newArtefactWriter()
	if n, err := b.Write(make([]byte, 20)); err == nil || n != 0 {
		t.Fatalf("second writer over the shared budget: n=%d err=%v", n, err)
	}
	b.discard()
	a.discard()
	if w.h.jobs.inflight != 0 {
		t.Fatalf("discarded writers still reserve %d bytes", w.h.jobs.inflight)
	}

	// A completed export moves its reservation to the stored total.
	w.h.jobs.limit = maxExpBytes
	d, _ := w.export()
	if w.h.jobs.inflight != 0 || w.h.jobs.bytes != d.Bytes {
		t.Fatalf("after store: inflight=%d bytes=%d want %d", w.h.jobs.inflight, w.h.jobs.bytes, d.Bytes)
	}
}

// BR-06: without a usable spool directory the memory fallback is bounded by
// the same budget and is logged, not silent.
func TestExportMemoryFallbackIsBoundedAndLogged(t *testing.T) {
	w := spoolEnv(t, filepath.Join(t.TempDir(), "missing"))
	var buf bytes.Buffer
	w.h.log = slog.New(slog.NewTextHandler(&buf, nil))
	w.h.jobs.limit = 64
	if rec := w.do("POST", "/api/v1/exports", `{"scope":"ledger","format":"csv"}`); rec.Code != 503 {
		t.Fatalf("memory fallback over the budget: %d", rec.Code)
	}
	if w.h.jobs.inflight != 0 {
		t.Fatalf("inflight=%d", w.h.jobs.inflight)
	}
	if !strings.Contains(buf.String(), "export spool unavailable") {
		t.Fatalf("fallback not logged: %q", buf.String())
	}
}

// BR-06: the spool directory is an option, not always os.TempDir().
func TestExportDirOption(t *testing.T) {
	dir := t.TempDir()
	h := NewHandlerWithOptions(nil, Options{ExportDir: dir}).(*server)
	if h.jobs.dir != dir {
		t.Fatalf("jobs.dir = %q, want %q", h.jobs.dir, dir)
	}
}
