package api

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
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
	env            *testutil.Env
	netID          string
	net            *domain.Network
	admin, alice   domain.Actor // behind tok and memberTok
	mu             sync.Mutex
	now            time.Time
}

// spoolEnv is a network with one trade and admin and member tokens; the
// handler's clock is controllable, its artefacts spool into dir and member
// erasure reaches it (OnMemberErased).
func spoolEnv(t *testing.T, dir string) *spoolWorld {
	t.Helper()
	env := testutil.NewEnvConfig(t, nil, func(c *service.Config) { c.OnMemberErased = OnMemberErased })
	admin := env.Person("admin")
	net, _ := env.Network(admin, "acme", nil)
	alice, aliceM := env.Member(admin, net, "alice")
	bob, bobM := env.Member(admin, net, "bob")
	env.Trade(net.ID, alice, bob, aliceM, bobM, 250)
	w := &spoolWorld{t: t, person: admin.PersonID, env: env, netID: net.ID, net: net, admin: admin, alice: alice, now: time.Now()}
	w.h = NewHandlerWithOptions(env.Svc, Options{ExportSpoolDir: dir,
		Now: func() time.Time { w.mu.Lock(); defer w.mu.Unlock(); return w.now }}).(*server)
	// Release every artefact before dir is removed (Windows keeps open files).
	t.Cleanup(func() {
		CloseHandler(w.h)
	})
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
	return w.exportScope("ledger")
}

func (w *spoolWorld) exportScope(scope string) (exportDTO, *exportEntry) {
	w.t.Helper()
	rec := w.do("POST", "/api/v1/exports", `{"scope":"`+scope+`","format":"csv"}`)
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
	if e.spool == nil || e.spool.size != d.Bytes || d.Bytes == 0 || w.h.jobs.bytes != d.Bytes {
		t.Fatalf("artefact not spooled or miscounted: spool=%v dto=%d store=%d", e.spool != nil, d.Bytes, w.h.jobs.bytes)
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
	w.h.jobs.dropErased("", w.person)
	if _, err := e.spool.f.ReadAt(make([]byte, 1), 0); !errors.Is(err, os.ErrClosed) {
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
	tk, why := w.h.jobs.beginExport(w.netID, w.person, true)
	if why != "" {
		t.Fatalf("slot after expiry: %q", why)
	}
	tk.abort()
	if _, err := e2.spool.f.ReadAt(make([]byte, 1), 0); !errors.Is(err, os.ErrClosed) || w.h.jobs.bytes != 0 || dirEntries(t, dir) != 0 {
		t.Errorf("expired artefact not released: %v, %d bytes", err, w.h.jobs.bytes)
	}
}

func itoa64(n int64) string { b, _ := json.Marshal(n); return string(b) }

// storeExport puts e into j with data as its spooled artefact, through the
// ticket path createExport uses.
func storeExport(t *testing.T, j *jobStore, e *exportEntry, data []byte) {
	t.Helper()
	tk, why := j.beginExport(e.net, e.person, e.dto.Scope != "mine")
	if why != "" {
		t.Fatalf("begin: %s", why)
	}
	a, err := newArtefactWriter(j.dir, tk)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := a.Write(data); err != nil {
		t.Fatal(err)
	}
	a.finish(e)
	t.Cleanup(e.release) // Windows cannot remove a temporary directory holding an open file
	if !tk.commit(e) {
		t.Fatal("commit refused")
	}
}

// assertIdle fails unless no export is running or holds a reservation.
func assertIdle(t *testing.T, j *jobStore) {
	t.Helper()
	j.mu.Lock()
	defer j.mu.Unlock()
	if j.nRunning != 0 || len(j.running) != 0 || j.inflight != 0 {
		t.Errorf("running exports left behind: %d running, %v per person, %d bytes reserved", j.nRunning, j.running, j.inflight)
	}
}

// BR-06: without a usable spool directory the export fails closed with 503
// and a clear reason; the archive is never collected in memory instead.
func TestExportRefusedWithoutSpoolDir(t *testing.T) {
	w := spoolEnv(t, filepath.Join(t.TempDir(), "missing"))
	rec := w.do("POST", "/api/v1/exports", `{"scope":"ledger","format":"csv"}`)
	if rec.Code != http.StatusServiceUnavailable || rec.Header().Get("Retry-After") == "" ||
		!strings.Contains(rec.Body.String(), "export storage is unavailable") {
		t.Fatalf("export without spool dir: %d %s", rec.Code, rec.Body.String())
	}
	if n := len(w.h.jobs.exps); n != 0 || w.h.jobs.bytes != 0 {
		t.Errorf("%d exports (%d bytes) stored without a spool file", n, w.h.jobs.bytes)
	}
	assertIdle(t, w.h.jobs)
}

// BR-06: an export that would pass the byte cap is stopped while it writes,
// before the excess reaches the disk, and leaves no file or reservation
// behind; so does an export refused by the service.
func TestExportRefusedLeavesNoSpool(t *testing.T) {
	dir := t.TempDir()
	w := spoolEnv(t, dir)
	w.h.jobs.capBytes = 100 // smaller than any archive
	rec := w.do("POST", "/api/v1/exports", `{"scope":"ledger","format":"csv"}`)
	if rec.Code != 503 || !strings.Contains(rec.Body.String(), "export storage is full") {
		t.Fatalf("over the cap: %d %s", rec.Code, rec.Body.String())
	}
	if n := dirEntries(t, dir); n != 0 || len(w.h.jobs.exps) != 0 || w.h.jobs.bytes != 0 {
		t.Errorf("refused export left %d files, %d entries, %d bytes", n, len(w.h.jobs.exps), w.h.jobs.bytes)
	}
	assertIdle(t, w.h.jobs)
	// Refused by the service: a member may not export the ledger.
	w.h.jobs.capBytes = maxExpBytes
	w.tok = w.memberTok
	if rec := w.do("POST", "/api/v1/exports", `{"scope":"ledger","format":"csv"}`); rec.Code != 403 {
		t.Fatalf("member ledger export: %d", rec.Code)
	}
	if n := dirEntries(t, dir); n != 0 {
		t.Errorf("%d files left behind by the forbidden export", n)
	}
	assertIdle(t, w.h.jobs)
}

// BR-06: parallel exports reserve every byte before writing it, so what they
// have on disk together never passes the cap, however large each one would
// grow. The disk use is measured from the spool files themselves.
func TestParallelExportWritesStayWithinCap(t *testing.T) {
	j := newJobStore(time.Now)
	j.dir = t.TempDir()
	const capBytes, want, chunk = 1 << 20, 1 << 20, 64 << 10 // each export alone would fill the cap
	j.capBytes = capBytes
	type run struct {
		tk  *exportTicket
		a   *artefactWriter
		err error
	}
	runs := make([]*run, maxExportsRunning)
	for i := range runs {
		tk, why := j.beginExport("n", fmt.Sprintf("p%d", i), true)
		if why != "" {
			t.Fatalf("begin %d: %s", i, why)
		}
		a, err := newArtefactWriter(j.dir, tk)
		if err != nil {
			t.Fatal(err)
		}
		runs[i] = &run{tk: tk, a: a}
	}
	if _, why := j.beginExport("n", "one-more", true); why != "busy" {
		t.Errorf("export beyond the running cap admitted: %q", why)
	}
	var wg sync.WaitGroup
	buf := make([]byte, chunk)
	for _, r := range runs {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for r.a.n < want {
				if _, err := r.a.Write(buf); err != nil {
					r.err = err
					return
				}
			}
		}()
	}
	wg.Wait()
	var disk int64
	refused := 0
	for _, r := range runs {
		fi, err := r.a.spool.f.Stat()
		if err != nil {
			t.Fatal(err)
		}
		disk += fi.Size()
		if errors.Is(r.err, errExportFull) {
			refused++
			if !r.a.full {
				t.Error("refused write not flagged")
			}
		} else if r.err != nil {
			t.Fatal(r.err)
		}
	}
	t.Logf("BR-06: %d parallel exports wrote %d bytes in total under a %d byte cap; %d refused", len(runs), disk, capBytes, refused)
	if disk > capBytes {
		t.Errorf("parallel exports wrote %d bytes, over the %d byte cap", disk, capBytes)
	}
	if refused < len(runs)-1 {
		t.Errorf("%d exports refused, want at least %d", refused, len(runs)-1)
	}
	for _, r := range runs {
		r.a.discard()
		r.tk.abort()
	}
	assertIdle(t, j)
}

// BR-06: running exports count against the per-person cap, so parallel
// requests by one person cannot all pass the check before any is stored.
func TestRunningExportsHoldTheirSlot(t *testing.T) {
	j := newJobStore(time.Now)
	j.dir = t.TempDir()
	stored := maxExportsPerPerson - 2
	for i := 0; i < stored; i++ {
		storeExport(t, j, &exportEntry{net: "n", person: "p", at: time.Now(), dto: exportDTO{ID: fmt.Sprint(i), Scope: "mine"}}, []byte("x"))
	}
	var tks []*exportTicket
	for i := stored; i < maxExportsPerPerson; i++ {
		tk, why := j.beginExport("n", "p", false)
		if why != "" {
			t.Fatalf("export %d: %q", i+1, why)
		}
		tks = append(tks, tk)
	}
	if _, why := j.beginExport("n", "p", false); why != "person" {
		t.Errorf("export beyond the person's cap while two run: %q, want person", why)
	}
	if tk, why := j.beginExport("n", "q", false); why != "" {
		t.Errorf("another person refused: %q", why)
	} else {
		tk.abort()
	}
	for _, tk := range tks {
		tk.abort()
	}
	assertIdle(t, j)
	if tk, why := j.beginExport("n", "p", false); why != "" {
		t.Errorf("after the running exports ended: %q", why)
	} else {
		tk.abort()
	}
}

// BR-04: erasing a member drops every export artefact of that
// network, since even member exports can hold counterparties' data.
// Other networks' exports stay. An export still
// running when the erasure happens is not stored.
func TestErasureDropsNetworkExports(t *testing.T) {
	w := spoolEnv(t, t.TempDir())
	ctx := context.Background()
	carol, _ := w.env.Member(w.admin, w.net, "carol")
	var admin []*exportEntry
	for _, scope := range []string{"full", "ledger", "members", "audit"} {
		_, e := w.exportScope(scope)
		admin = append(admin, e)
	}
	_, mine := w.exportScope("mine")
	other := &exportEntry{net: "other-network", person: "someone", at: time.Now(), dto: exportDTO{ID: "o", Scope: "full"}}
	storeExport(t, w.h.jobs, other, []byte("other"))
	running, why := w.h.jobs.beginExport(w.netID, "another-admin", true)
	if why != "" {
		t.Fatal(why)
	}
	runningMine, why := w.h.jobs.beginExport(w.netID, "another-member", false)
	if why != "" {
		t.Fatal(why)
	}

	if _, err := w.env.Svc.DepartMember(ctx, carol, w.netID, service.DepartInput{}); err != nil {
		t.Fatal(err)
	}
	if _, err := w.env.Svc.EraseMyData(ctx, carol, w.netID, ""); err != nil {
		t.Fatal(err)
	}

	for _, e := range admin {
		if w.h.jobs.getExport(w.netID, w.person, e.dto.ID) != nil {
			t.Errorf("%s export holding the erased member survives", e.dto.Scope)
		}
		if _, err := e.spool.f.ReadAt(make([]byte, 1), 0); !errors.Is(err, os.ErrClosed) {
			t.Errorf("%s export file still open: %v", e.dto.Scope, err)
		}
	}
	if w.h.jobs.getExport(w.netID, w.person, mine.dto.ID) != nil {
		t.Error("the member export survived erasure")
	}
	if w.h.jobs.getExport("other-network", "someone", "o") == nil {
		t.Error("another network's export was dropped")
	}
	if want := other.size(); w.h.jobs.bytes != want {
		t.Errorf("byte count %d, want %d", w.h.jobs.bytes, want)
	}
	e := &exportEntry{net: w.netID, person: "another-admin", at: time.Now(), dto: exportDTO{ID: "late", Scope: "full"}}
	if running.commit(e) {
		t.Error("admin-scope export running across the erasure was stored")
	}
	if runningMine.commit(&exportEntry{dto: exportDTO{ID: "late-mine", Scope: "mine"}}) {
		t.Error("member export running across the erasure was stored")
	}
	assertIdle(t, w.h.jobs)
	// A new export after the erasure is stored as usual.
	w.exportScope("full")
}
