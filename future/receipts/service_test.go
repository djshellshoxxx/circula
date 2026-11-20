package receipts

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

type fixtureReader struct {
	calls    atomic.Int64
	denial   bool
	snapshot Trade
}

func (f *fixtureReader) ReadReceipt(ctx context.Context, a Actor, id string) (Trade, error) {
	f.calls.Add(1)
	if ctx.Err() != nil {
		return Trade{}, ctx.Err()
	}
	if f.denial || a.Network != "n" || a.Member != "a" || id != "tx" {
		return Trade{}, ErrUnavailable
	}
	return f.snapshot, nil
}
func fixture() (*fixtureReader, Service) {
	r := &fixtureReader{snapshot: Trade{Snapshot: Snapshot{"n", "tx", "a", "b", "credits", "completed", 123, time.Unix(100, 0)}, Precision: 2, PayerEntry: "entry1", PayeeEntry: "entry2", Revision: 1}}
	return r, Service{Reader: r, Enabled: true, NetworkEnabled: func(context.Context, string) bool { return true }}
}
func TestServiceConsentDisabledAndCancellation(t *testing.T) {
	r, s := fixture()
	a := Actor{"n", "a"}
	for _, change := range []func(*Service, *DownloadRequest){func(s *Service, q *DownloadRequest) { q.Explicit = false }, func(s *Service, q *DownloadRequest) { s.Enabled = false }, func(s *Service, q *DownloadRequest) { s.NetworkEnabled = nil }, func(s *Service, q *DownloadRequest) {
		s.NetworkEnabled = func(context.Context, string) bool { return false }
	}} {
		copy := s
		q := DownloadRequest{"tx", "json", true}
		change(&copy, &q)
		if _, e := copy.Receipt(context.Background(), a, q); !errors.Is(e, ErrUnavailable) {
			t.Fatal(e)
		}
	}
	if r.calls.Load() != 0 {
		t.Fatal("reader called without consent/enablement")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, e := s.Receipt(ctx, a, DownloadRequest{"tx", "json", true}); !errors.Is(e, context.Canceled) {
		t.Fatal(e)
	}
}
func TestServiceDeterministicScopeAndReversal(t *testing.T) {
	r, s := fixture()
	a := Actor{"n", "a"}
	q := DownloadRequest{"tx", "json", true}
	d, e := s.Receipt(context.Background(), a, q)
	if e != nil {
		t.Fatal(e)
	}
	before, _ := d.JSON()
	for _, format := range []string{"txt", "json", "html"} {
		one, _, e := d.Export(format)
		if e != nil {
			t.Fatal(e)
		}
		two, _, e := d.Export(format)
		if e != nil || !bytes.Equal(one, two) || !bytes.Contains(one, []byte("123")) {
			t.Fatal(format, e)
		}
	}
	r.snapshot.Supplement = &StatusSupplement{"reversal1", "reversed", time.Unix(101, 0).Format(time.RFC3339Nano)}
	fresh, e := s.Receipt(context.Background(), a, q)
	if e != nil || fresh.Status != "completed" || fresh.Supplement == nil {
		t.Fatal(fresh, e)
	}
	after, _ := d.JSON()
	if !bytes.Equal(before, after) {
		t.Fatal("historical document mutated")
	}
	r.snapshot.Supplement.Status = "adjusted"
	if fresh.Supplement.Status != "reversed" {
		t.Fatal("supplement aliases reader state")
	}
	for _, actor := range []Actor{{"other", "a"}, {"n", "other"}} {
		if _, e = s.Receipt(context.Background(), actor, q); !errors.Is(e, ErrUnavailable) {
			t.Fatal(actor, e)
		}
	}
	if _, e = s.Receipt(context.Background(), a, DownloadRequest{"missing", "json", true}); !errors.Is(e, ErrUnavailable) {
		t.Fatal(e)
	}
}
func TestProjectionBoundsAndAllowlist(t *testing.T) {
	r, s := fixture()
	for name, mutate := range map[string]func(*Trade){"overflow": func(v *Trade) { v.Amount = 1<<50 + 1 }, "negative": func(v *Trade) { v.Amount = -1 }, "precision": func(v *Trade) { v.Precision = 9 }, "same entry": func(v *Trade) { v.PayeeEntry = v.PayerEntry }, "missing entry": func(v *Trade) { v.PayerEntry = "" }, "status": func(v *Trade) { v.Status = "pending" }, "revision": func(v *Trade) { v.Revision = 0 }, "control": func(v *Trade) { v.Unit = "credits\nforged" }, "date": func(v *Trade) { v.CompletedAt = time.Time{} }, "cross transaction": func(v *Trade) { v.Transaction = "other" }, "supplement": func(v *Trade) { v.Supplement = &StatusSupplement{"tx", "reversed", "bad"} }} {
		t.Run(name, func(t *testing.T) {
			original := r.snapshot
			mutate(&r.snapshot)
			defer func() { r.snapshot = original }()
			if _, e := s.Receipt(context.Background(), Actor{"n", "a"}, DownloadRequest{"tx", "json", true}); !errors.Is(e, ErrUnavailable) {
				t.Fatal(e)
			}
		})
	}
	r.snapshot.Payer = "<img src=x onerror=alert(1)>"
	d, e := s.Receipt(context.Background(), Actor{"n", "a"}, DownloadRequest{"tx", "html", true})
	if e != nil {
		t.Fatal(e)
	}
	html, _, _ := d.Export("html")
	if bytes.Contains(html, []byte("<img")) {
		t.Fatal("HTML injection")
	}
	raw, _ := d.JSON()
	var fields map[string]any
	if e = json.Unmarshal(raw, &fields); e != nil {
		t.Fatal(e)
	}
	if len(fields) != 13 {
		t.Fatalf("unexpected DTO fields: %v", fields)
	}
	if _, _, e = d.Export("pdf"); e == nil {
		t.Fatal("unsupported format")
	}
}
func TestReadOnlyConcurrentDownloads(t *testing.T) {
	r, s := fixture()
	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			var b bytes.Buffer
			if e := s.Download(context.Background(), Actor{"n", "a"}, DownloadRequest{"tx", "txt", true}, &b); e != nil {
				t.Error(e)
			}
		}()
	}
	wg.Wait()
	if r.calls.Load() != 20 {
		t.Fatal(r.calls.Load())
	}
	if r.snapshot.Amount != 123 || r.snapshot.Revision != 1 {
		t.Fatal("snapshot mutation")
	}
}

type hostFixture struct {
	denied, limited bool
	csrf            bool
}

func (h hostFixture) Authorize(r *http.Request, unsafe bool) (Actor, error) {
	if h.denied || unsafe && !h.csrf {
		return Actor{}, errors.New("denied")
	}
	return Actor{"n", "a"}, nil
}
func (h hostFixture) RateLimit(*http.Request, Actor) bool { return !h.limited }
func (h hostFixture) CSRFToken(*http.Request) string      { return "synthetic-csrf" }
func TestHTTPNoJSAndBoundary(t *testing.T) {
	r, s := fixture()
	h := Handler{s, hostFixture{csrf: true}}
	get := httptest.NewRecorder()
	h.ServeHTTP(get, httptest.NewRequest("GET", "/receipt?reference=tx", nil))
	if get.Code != 200 || strings.Contains(get.Body.String(), "<script") || r.calls.Load() != 0 || !strings.Contains(get.Body.String(), "synthetic-csrf") {
		t.Fatal(get.Code, get.Body.String())
	}
	form := url.Values{"reference": {"tx"}, "format": {"txt"}, "explicit": {"true"}, "csrf_token": {"synthetic-csrf"}}
	post := httptest.NewRequest("POST", "/receipt", strings.NewReader(form.Encode()))
	post.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, post)
	if w.Code != 200 || w.Header().Get("Cache-Control") != "private, no-store" || !strings.Contains(w.Header().Get("Content-Disposition"), "circula-receipt.txt") || !strings.Contains(w.Body.String(), "123") {
		t.Fatal(w.Code, w.Body.String(), w.Header())
	}
	for _, reference := range []string{"missing", "tx"} {
		s.Reader = &fixtureReader{denial: true}
		denied := Handler{s, hostFixture{csrf: true}}
		v := httptest.NewRecorder()
		req := httptest.NewRequest("POST", "/receipt", strings.NewReader(fmt.Sprintf(`{"reference":%q,"format":"json","explicit":true}`, reference)))
		req.Header.Set("Content-Type", "application/json")
		denied.ServeHTTP(v, req)
		if v.Code != 404 || strings.Contains(v.Body.String(), reference) {
			t.Fatal(v.Code, v.Body.String())
		}
	}
}
func TestHTTPInvalidRequests(t *testing.T) {
	_, s := fixture()
	for name, test := range map[string]struct {
		method, body, typ string
		host              hostFixture
		network           string
		want              int
	}{"bad csrf": {"POST", `{"reference":"tx","format":"json","explicit":true}`, "application/json", hostFixture{}, "", 404}, "body facts": {"POST", `{"reference":"tx","format":"json","explicit":true,"amount":999}`, "application/json", hostFixture{csrf: true}, "", 400}, "no consent": {"POST", `{"reference":"tx","format":"json"}`, "application/json", hostFixture{csrf: true}, "", 400}, "cross tenant": {"POST", `{}`, "application/json", hostFixture{csrf: true}, "other", 404}, "rate limit": {"GET", "", "", hostFixture{limited: true}, "", 429}, "huge": {"POST", strings.Repeat(" ", 65537), "application/json", hostFixture{csrf: true}, "", 400}, "method": {"DELETE", "", "", hostFixture{}, "", 405}, "content": {"POST", "x", "text/plain", hostFixture{csrf: true}, "", 415}, "trailing": {"POST", `{"reference":"tx","format":"json","explicit":true}{}`, "application/json", hostFixture{csrf: true}, "", 400}} {
		t.Run(name, func(t *testing.T) {
			h := Handler{s, test.host}
			w := httptest.NewRecorder()
			req := httptest.NewRequest(test.method, "/receipt?reference=tx", strings.NewReader(test.body))
			req.Header.Set("Content-Type", test.typ)
			req.Header.Set("X-Network", test.network)
			h.ServeHTTP(w, req)
			if w.Code != test.want {
				t.Fatal(w.Code, w.Body.String())
			}
		})
	}
}
func TestCLIRequiresExplicitChoice(t *testing.T) {
	r, s := fixture()
	var out, help bytes.Buffer
	e := RunCLI(context.Background(), s, Actor{"n", "a"}, []string{"--transaction", "tx"}, &out, &help)
	if e == nil || r.calls.Load() != 0 {
		t.Fatal(e)
	}
	e = RunCLI(context.Background(), s, Actor{"n", "a"}, []string{"--transaction", "tx", "--format", "json", "--request-receipt"}, &out, &help)
	if e != nil || !json.Valid(out.Bytes()) {
		t.Fatal(e, out.String())
	}
	e = RunCLI(context.Background(), s, Actor{"n", "a"}, []string{"--help"}, &out, &help)
	if e == nil || !strings.Contains(help.String(), "request-receipt") {
		t.Fatal(e, help.String())
	}
}
func FuzzCreateReceipt(f *testing.F) {
	f.Add("n", "credits", int64(1))
	f.Add("\x00", "bad\nunit", int64(1<<53))
	f.Fuzz(func(t *testing.T, network, unit string, amount int64) {
		s := Snapshot{network, "tx", "a", "b", unit, "completed", amount, time.Unix(1, 0)}
		r, e := Create(Request{network, "a", true, true}, s)
		if e == nil {
			if amount <= 0 || amount > 1<<50 || !safe(network) || !safe(unit) {
				t.Fatal("invalid accepted")
			}
			b, e := r.JSON()
			if e != nil || !json.Valid(b) {
				t.Fatal(e)
			}
		}
	})
}

func TestServiceRecreationRetainsNoArtifactOrAuthorization(t *testing.T) {
	r, s := fixture()
	q := DownloadRequest{"tx", "json", true}
	first, e := s.Receipt(context.Background(), Actor{"n", "a"}, q)
	if e != nil {
		t.Fatal(e)
	}
	fresh := Service{Reader: r, Enabled: true, NetworkEnabled: s.NetworkEnabled}
	second, e := fresh.Receipt(context.Background(), Actor{"n", "a"}, q)
	if e != nil {
		t.Fatal(e)
	}
	a, _ := first.JSON()
	b, _ := second.JSON()
	if !bytes.Equal(a, b) {
		t.Fatal("unstable stateless download")
	}
	r.denial = true
	if _, e = fresh.Receipt(context.Background(), Actor{"n", "a"}, q); !errors.Is(e, ErrUnavailable) {
		t.Fatal("stale authorization", e)
	}
}

type brokenAudit struct{}

func (brokenAudit) ReceiptOutcome(context.Context, string) error { return errors.New("unavailable") }
func TestAuditAndBadWriterFailuresAreIsolated(t *testing.T) {
	r, s := fixture()
	s.Audit = brokenAudit{}
	if _, e := s.Receipt(context.Background(), Actor{"n", "a"}, DownloadRequest{"tx", "json", true}); !errors.Is(e, ErrUnavailable) {
		t.Fatal(e)
	}
	if r.snapshot.Amount != 123 || r.snapshot.Status != "completed" {
		t.Fatal("trade changed")
	}
	s.Audit = nil
	if e := s.Download(context.Background(), Actor{"n", "a"}, DownloadRequest{"tx", "pdf", true}, &bytes.Buffer{}); e == nil {
		t.Fatal("format")
	}
	if e := s.Download(context.Background(), Actor{"n", "a"}, DownloadRequest{"tx", "txt", true}, badWriter{}); e == nil {
		t.Fatal("write error ignored")
	}
}

type badWriter struct{}

func (badWriter) Write([]byte) (int, error) { return 0, errors.New("closed") }
