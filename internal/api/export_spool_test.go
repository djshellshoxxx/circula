package api

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/djshellshoxxx/circula/internal/domain"
	"github.com/djshellshoxxx/circula/internal/service"
	"github.com/djshellshoxxx/circula/internal/testutil"
)

type spoolWorld struct {
	t              *testing.T
	h              *server
	tok, memberTok string // admin's and a member's bearer tokens
	person         string // the admin's person id
	mu             sync.Mutex
	now            time.Time
}

// spoolEnv is a network with one trade and admin and member tokens; the
// handler's clock is controllable and its artefacts spool into dir.
func spoolEnv(t *testing.T, dir string) *spoolWorld {
	t.Helper()
	env := testutil.NewEnv(t)
	admin := env.Person("admin")
	net, _ := env.Network(admin, "acme", nil)
	alice, aliceM := env.Member(admin, net, "alice")
	bob, bobM := env.Member(admin, net, "bob")
	env.Trade(net.ID, alice, bob, aliceM, bobM, 250)
	w := &spoolWorld{t: t, person: admin.PersonID, now: time.Now()}
	w.h = NewHandlerWithOptions(env.Svc, Options{Now: func() time.Time { w.mu.Lock(); defer w.mu.Unlock(); return w.now }}).(*server)
	w.h.jobs.dir = dir
	for _, p := range []struct {
		a   domain.Actor
		tok *string
	}{{admin, &w.tok}, {alice, &w.memberTok}} {
		raw, _, err := env.Svc.CreateAPIToken(context.Background(), p.a, service.TokenInput{Name: "t",
			Scopes: []domain.Scope{domain.ScopeRead, domain.ScopeWrite}})
		if err != nil {
			t.Fatal(err)
		}
		*p.tok = raw
	}
	return w
}

func (w *spoolWorld) do(method, path, body string) *httptest.ResponseRecorder {
	r := httptest.NewRequest(method, path, strings.NewReader(body))
	r.Header.Set("Authorization", "Bearer "+w.tok)
	if body != "" {
		r.Header.Set("Content-Type", "application/json")
	}
	rec := httptest.NewRecorder()
	w.h.ServeHTTP(rec, r)
	return rec
}

func (w *spoolWorld) export() (exportDTO, *exportEntry) {
	w.t.Helper()
	rec := w.do("POST", "/api/v1/exports", `{"scope":"ledger","format":"csv"}`)
	if rec.Code != 201 {
		w.t.Fatalf("create: %d %s", rec.Code, rec.Body.String())
	}
	var d exportDTO
	if err := json.Unmarshal(rec.Body.Bytes(), &d); err != nil {
		w.t.Fatal(err)
	}
	w.h.jobs.mu.Lock()
	e := w.h.jobs.exps[d.ID]
	w.h.jobs.mu.Unlock()
	if e == nil {
		w.t.Fatal("export not stored")
	}
	return d, e
}

func dirEntries(t *testing.T, dir string) int {
	t.Helper()
	ents, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	return len(ents)
}

// FR-095 / DEC-047: the artefact is spooled to disk, not held in memory; the
// download is served from the file and still verified (DAT-040); dropping or
// expiring the export releases the file.
func TestExportArtefactIsSpooled(t *testing.T) {
	dir := t.TempDir()
	w := spoolEnv(t, dir)
	d, e := w.export()
	if e.spool == nil || e.file != nil || e.spool.size != d.Bytes || d.Bytes == 0 || w.h.jobs.bytes != d.Bytes {
		t.Fatalf("artefact held in memory or miscounted: spool=%v file=%d dto=%d store=%d", e.spool != nil, len(e.file), d.Bytes, w.h.jobs.bytes)
	}
	if runtime.GOOS != "windows" && dirEntries(t, dir) != 0 {
		t.Error("spooled artefact still has a name on disk; it should be unlinked at once")
	}
	got := w.do("GET", "/api/v1/exports/"+d.ID+"/artefact", "")
	sum := sha256.Sum256(got.Body.Bytes())
	if got.Code != 200 || hex.EncodeToString(sum[:]) != *d.SHA256 || got.Header().Get("Content-Length") != itoa64(d.Bytes) {
		t.Fatalf("download: %d len %d (header %s) sha %x want %s", got.Code, got.Body.Len(), got.Header().Get("Content-Length"), sum, *d.SHA256)
	}
	// A corrupted spool file is never served.
	if _, err := e.spool.f.WriteAt([]byte("XX"), 40); err != nil {
		t.Fatal(err)
	}
	if bad := w.do("GET", "/api/v1/exports/"+d.ID+"/artefact", ""); bad.Code != 500 {
		t.Errorf("corrupted artefact served: %d", bad.Code)
	}
	// Erasure drops the export and closes its file.
	w.h.jobs.dropPerson(w.person)
	if _, err := e.spool.f.Stat(); !errors.Is(err, os.ErrClosed) {
		t.Errorf("spool file still open after drop: %v", err)
	}
	if w.h.jobs.bytes != 0 || dirEntries(t, dir) != 0 {
		t.Errorf("after drop: %d bytes counted, %d files", w.h.jobs.bytes, dirEntries(t, dir))
	}
	if gone := w.do("GET", "/api/v1/exports/"+d.ID+"/artefact", ""); gone.Code != 404 {
		t.Errorf("dropped export: %d", gone.Code)
	}
	// Expiry releases the file too.
	_, e2 := w.export()
	w.mu.Lock()
	w.now = w.now.Add(jobTTL + time.Minute)
	w.mu.Unlock()
	if why := w.h.jobs.exportSlot(w.person); why != "" {
		t.Fatalf("slot after expiry: %q", why)
	}
	if _, err := e2.spool.f.Stat(); !errors.Is(err, os.ErrClosed) || w.h.jobs.bytes != 0 || dirEntries(t, dir) != 0 {
		t.Errorf("expired artefact not released: %v, %d bytes", err, w.h.jobs.bytes)
	}
}

func itoa64(n int64) string { b, _ := json.Marshal(n); return string(b) }

// Without a usable temporary directory the export still works, in memory as
// before (DEC-047).
func TestExportArtefactFallsBackToMemory(t *testing.T) {
	w := spoolEnv(t, filepath.Join(t.TempDir(), "missing"))
	d, e := w.export()
	if e.spool != nil || int64(len(e.file)) != d.Bytes || d.Bytes == 0 {
		t.Fatalf("fallback: spool=%v file=%d dto=%d", e.spool != nil, len(e.file), d.Bytes)
	}
	got := w.do("GET", "/api/v1/exports/"+d.ID+"/artefact", "")
	if sum := sha256.Sum256(got.Body.Bytes()); got.Code != http.StatusOK || hex.EncodeToString(sum[:]) != *d.SHA256 {
		t.Fatalf("download: %d", got.Code)
	}
}

// An export refused after it was spooled (the byte cap is reached only once
// its size is known) or refused by the service leaves no file behind.
func TestExportRefusedLeavesNoSpool(t *testing.T) {
	dir := t.TempDir()
	w := spoolEnv(t, dir)
	// Another person's artefact almost fills the byte cap (SA-11).
	other := newArtefactWriter(t.TempDir())
	if other.spool == nil {
		t.Fatal("test premise: no spool file")
	}
	big := &exportEntry{net: "n", person: "someone-else", at: w.now, dto: exportDTO{ID: "big"}}
	other.finish(big)
	big.spool.size = maxExpBytes - 10
	if !w.h.jobs.putExport(big) {
		t.Fatal("test premise: filler refused")
	}
	if rec := w.do("POST", "/api/v1/exports", `{"scope":"ledger","format":"csv"}`); rec.Code != 503 {
		t.Fatalf("full store: %d %s", rec.Code, rec.Body.String())
	}
	if n := dirEntries(t, dir); n != 0 {
		t.Errorf("%d files left behind by the refused export", n)
	}
	// Refused by the service: a member may not export the ledger.
	w.h.jobs.dropPerson("someone-else")
	w.tok = w.memberTok
	if rec := w.do("POST", "/api/v1/exports", `{"scope":"ledger","format":"csv"}`); rec.Code != 403 {
		t.Fatalf("member ledger export: %d", rec.Code)
	}
	if n := dirEntries(t, dir); n != 0 {
		t.Errorf("%d files left behind by the forbidden export", n)
	}
}
