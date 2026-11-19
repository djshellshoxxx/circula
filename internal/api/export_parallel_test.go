package api_test

import (
	"sync"
	"testing"
	"time"

	"github.com/djshellshoxxx/circula/internal/api"
	"github.com/djshellshoxxx/circula/internal/domain"
	"github.com/djshellshoxxx/circula/internal/store"
	"github.com/djshellshoxxx/circula/internal/testutil"
)

// BR-06: parallel large exports neither exhaust memory nor pass the byte cap.
// With the cap set to one and a half archives, eight parallel requests end in
// 201 or 503 only; what the store holds, stored and in flight together, never
// passes the cap while they run; nothing stays reserved afterwards; and the
// live heap grows by less than two archives.
func TestParallelLargeExportsStayBounded(t *testing.T) {
	if testing.Short() {
		t.Skip("writes several large exports")
	}
	const rows = 150_000
	env := testutil.NewEnvWith(t, func(s store.Store) store.Store { return &bigLedger{Store: s, n: rows} })
	admin := env.Person("admin")
	net, _ := env.Network(admin, "acme", nil)
	alice, aliceM := env.Member(admin, net, "alice")
	bob, bobM := env.Member(admin, net, "bob")
	env.Trade(net.ID, alice, bob, aliceM, bobM, 100)
	body := map[string]any{"scope": "ledger", "format": "csv"}

	// The archive's size, from a handler of its own.
	initial := api.NewHandlerWithOptions(env.Svc, api.Options{CSRFKey: testCSRFKey, ExportSpoolDir: t.TempDir()})
	t.Cleanup(func() { api.CloseHandler(initial) })
	w := &world{t: t, env: env, h: initial}
	adminT := w.token(admin, domain.ScopeRead, domain.ScopeWrite)
	rec := w.post(adminT, "/exports", body, "")
	wantStatus(t, rec, 201)
	size := decode[struct {
		Bytes int64 `json:"bytes"`
	}](t, rec).Bytes
	if size < 4<<20 {
		t.Fatalf("test premise: archive of %d bytes is too small", size)
	}

	h := api.NewHandlerWithOptions(env.Svc, api.Options{CSRFKey: testCSRFKey, ExportSpoolDir: t.TempDir()})
	t.Cleanup(func() { api.CloseHandler(h) })
	w.h = h
	capBytes := size * 3 / 2
	api.SetExportByteCap(h, capBytes)
	const parallel = 8
	codes := make([]int, parallel)
	var peak int64
	stop := make(chan struct{})
	var sampler sync.WaitGroup
	sampler.Add(1)
	go func() {
		defer sampler.Done()
		for {
			s, in, _ := api.ExportStoreUsage(h)
			peak = max(peak, s+in)
			select {
			case <-stop:
				return
			case <-time.After(200 * time.Microsecond):
			}
		}
	}()
	heap := peakHeap(func() {
		var wg sync.WaitGroup
		for i := range codes {
			wg.Add(1)
			go func() {
				defer wg.Done()
				codes[i] = w.post(adminT, "/exports", body, "").Code
			}()
		}
		wg.Wait()
	})
	close(stop)
	sampler.Wait()
	ok, refused := 0, 0
	for _, c := range codes {
		switch c {
		case 201:
			ok++
		case 503:
			refused++
		default:
			t.Errorf("parallel export answered %d", c)
		}
	}
	stored, inflight, running := api.ExportStoreUsage(h)
	t.Logf("BR-06: %d parallel exports of %.1f MiB under a %.1f MiB cap: %d stored, %d refused; peak held %.1f MiB, heap growth %.1f MiB",
		parallel, mib(uint64(size)), mib(uint64(capBytes)), ok, refused, mib(uint64(peak)), mib(heap))
	if refused == 0 || int64(ok)*size > capBytes || stored != int64(ok)*size {
		t.Errorf("%d stored (%d bytes), %d refused, cap %d", ok, stored, refused, capBytes)
	}
	if peak > capBytes {
		t.Errorf("stored and in-flight exports reached %d bytes, over the %d byte cap", peak, capBytes)
	}
	if inflight != 0 || running != 0 {
		t.Errorf("left behind: %d bytes reserved, %d running", inflight, running)
	}
	if limit := uint64(2 * size); heap >= limit {
		t.Errorf("parallel exports grew the heap by %d bytes, at least two %d byte archives", heap, size)
	}
}
