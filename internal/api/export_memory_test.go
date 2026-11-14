package api_test

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"hash"
	"math/rand"
	"net/http"
	"net/http/httptest"
	"runtime"
	"runtime/debug"
	"sync"
	"testing"
	"time"

	"github.com/djshellshoxxx/circula/internal/api"
	"github.com/djshellshoxxx/circula/internal/domain"
	"github.com/djshellshoxxx/circula/internal/service"
	"github.com/djshellshoxxx/circula/internal/store"
	"github.com/djshellshoxxx/circula/internal/testutil"
)

// bigLedger appends n synthetic, pairwise balanced rows to the network-wide
// entries table, so a large export costs no postings. Ids are random hex, so
// the archive compresses about as poorly as real ids do.
type bigLedger struct {
	store.Store
	n int
}

func (b *bigLedger) View(ctx context.Context, fn func(store.Tx) error) error {
	return b.Store.View(ctx, func(tx store.Tx) error { return fn(bigTx{Tx: tx, n: b.n}) })
}

type bigTx struct {
	store.Tx
	n int
}

func (t bigTx) StreamTable(tbl store.ExportTable, netID, memberID string, fn func([]string, []any) error) error {
	var cols []string
	var last []any
	err := t.Tx.StreamTable(tbl, netID, memberID, func(c []string, v []any) error {
		cols, last = c, v
		return fn(c, v)
	})
	if err != nil || tbl != store.TblEntries || memberID != "" || last == nil {
		return err
	}
	rng := rand.New(rand.NewSource(1))
	id := func() string { return fmt.Sprintf("%016x-%016x", rng.Uint64(), rng.Uint64()) }
	for i := 0; i < t.n; i++ {
		row := append([]any(nil), last...)
		amt := int64(1 + i/2%997)
		if i%2 == 1 {
			amt = -amt
		}
		for j, c := range cols {
			switch c {
			case "id", "tx_id", "account_id":
				row[j] = id()
			case "amount":
				row[j] = amt
			}
		}
		if err := fn(cols, row); err != nil {
			return err
		}
	}
	return nil
}

// peakHeap runs fn and returns the largest growth of the live heap over its
// starting point, sampled every millisecond with the collector made eager.
func peakHeap(fn func()) uint64 {
	defer debug.SetGCPercent(debug.SetGCPercent(5))
	runtime.GC()
	var ms runtime.MemStats
	runtime.ReadMemStats(&ms)
	base, peak := ms.HeapAlloc, ms.HeapAlloc
	done := make(chan struct{})
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		var m runtime.MemStats
		for {
			runtime.ReadMemStats(&m)
			peak = max(peak, m.HeapAlloc)
			select {
			case <-done:
				return
			case <-time.After(time.Millisecond):
			}
		}
	}()
	fn()
	close(done)
	wg.Wait()
	return peak - base
}

// hashingWriter is a ResponseWriter that keeps nothing but a digest.
type hashingWriter struct {
	hdr  http.Header
	code int
	h    hash.Hash
	n    int64
}

func (w *hashingWriter) Header() http.Header  { return w.hdr }
func (w *hashingWriter) WriteHeader(code int) { w.code = code }
func (w *hashingWriter) Write(p []byte) (int, error) {
	w.h.Write(p)
	w.n += int64(len(p))
	return len(p), nil
}

// FR-095: the API export streams. Creating and downloading a multi-megabyte
// archive raises the live heap by less than the archive's own size, while
// the same measurement over a buffering sink sees the whole export. Peak
// growth is recorded; product size and duration thresholds are not set here
// (OQ-FR-3).
func TestExportStreamsWithBoundedMemory(t *testing.T) {
	if testing.Short() {
		t.Skip("writes a large export")
	}
	const rows = 150_000
	env := testutil.NewEnvWith(t, func(s store.Store) store.Store { return &bigLedger{Store: s, n: rows} })
	admin := env.Person("admin")
	net, _ := env.Network(admin, "acme", nil)
	alice, aliceM := env.Member(admin, net, "alice")
	bob, bobM := env.Member(admin, net, "bob")
	env.Trade(net.ID, alice, bob, aliceM, bobM, 100)
	w := &world{t: t, env: env, h: api.NewHandlerWithOptions(env.Svc, api.Options{CSRFKey: testCSRFKey})}
	adminT := w.token(admin, domain.ScopeRead, domain.ScopeWrite)

	type created struct {
		SHA256   *string `json:"sha256"`
		Bytes    int64   `json:"bytes"`
		Artefact string  `json:"artefact_url"`
	}
	var ex created
	create := peakHeap(func() {
		rec := w.post(adminT, "/exports", map[string]any{"scope": "ledger", "format": "csv"}, "")
		wantStatus(t, rec, 201)
		ex = decode[created](t, rec)
	})
	if ex.SHA256 == nil || ex.Bytes < 4<<20 {
		t.Fatalf("test premise: archive of %d bytes is too small to measure", ex.Bytes)
	}
	dl := &hashingWriter{hdr: http.Header{}, h: sha256.New()}
	download := peakHeap(func() {
		r := httptest.NewRequest("GET", ex.Artefact, nil)
		r.Header.Set("Authorization", "Bearer "+adminT)
		w.h.ServeHTTP(dl, r)
	})
	if dl.code != 200 || dl.n != ex.Bytes || hex.EncodeToString(dl.h.Sum(nil)) != *ex.SHA256 {
		t.Fatalf("download: %d, %d bytes, checksum match %v", dl.code, dl.n, hex.EncodeToString(dl.h.Sum(nil)) == *ex.SHA256)
	}
	// Control: the measurement does see an export that is held in memory.
	var held int64
	buffered := peakHeap(func() {
		sink := &service.MemSink{}
		if _, err := env.Svc.Export(env.Ctx, admin, net.ID, service.ExportLedger, service.FormatCSV, sink); err != nil {
			t.Error(err)
		}
		for _, n := range sink.Names() {
			held += int64(len(sink.Bytes(n)))
		}
		runtime.KeepAlive(sink)
	})
	t.Logf("FR-095 peak heap growth: create %.1f MiB, download %.1f MiB, buffered control %.1f MiB; archive %.1f MiB, uncompressed %.1f MiB, %d entry rows",
		mib(create), mib(download), mib(buffered), mib(uint64(ex.Bytes)), mib(uint64(held)), rows)
	if buffered < uint64(held)/2 {
		t.Fatalf("test premise: the sampler missed a buffered export (%d of %d bytes)", buffered, held)
	}
	// Structural bound, not a product threshold: holding the archive (let
	// alone the uncompressed files) would by itself exceed it.
	if limit := uint64(ex.Bytes); create >= limit || download >= limit {
		t.Errorf("export not streamed: heap grew %d (create) / %d (download) bytes for a %d byte archive", create, download, ex.Bytes)
	}
}

func mib(n uint64) float64 { return float64(n) / (1 << 20) }
