package receipts

import (
	"bytes"
	"context"
	"github.com/gowebpki/jcs"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestCanonicalExactAndRoundTrip(t *testing.T) {
	r, s := fixture()
	r.snapshot.Amount = 1 << 50
	r.snapshot.Revision = 1 << 50
	d, e := s.Receipt(context.Background(), Actor{"n", "a"}, DownloadRequest{"tx", "json", true})
	if e != nil {
		t.Fatal(e)
	}
	canonical, e := d.Canonical()
	if e != nil || !bytes.Contains(canonical, []byte(`"amount_minor":1125899906842624`)) {
		t.Fatal(string(canonical), e)
	}
	decoded, e := DecodeDocument(bytes.NewReader(canonical))
	if e != nil || decoded.Amount != d.Amount {
		t.Fatal(decoded, e)
	}
	hash, e := d.Digest()
	if e != nil || len(hash) != 64 {
		t.Fatal(hash, e)
	}
	again, e := decoded.Digest()
	if e != nil || hash != again {
		t.Fatal(hash, again, e)
	}
	d.Revision = 1<<53 + 1
	if _, e = d.Canonical(); e == nil {
		t.Fatal("rounded revision accepted")
	}
}
func TestCanonicalInputValidation(t *testing.T) {
	_, s := fixture()
	d, e := s.Receipt(context.Background(), Actor{"n", "a"}, DownloadRequest{"tx", "json", true})
	if e != nil {
		t.Fatal(e)
	}
	b, _ := d.JSON()
	for name, raw := range map[string]string{"duplicate": `{"schema":"one","schema":"two"}`, "huge": strings.Repeat(" ", MaxDocumentBytes+1), "unknown": strings.TrimSuffix(string(b), "}") + `,"private_note":"hidden"}`, "trailing": string(b) + `{}`, "invalid unicode": `{"schema":"\ud800"}`} {
		t.Run(name, func(t *testing.T) {
			if _, e := DecodeDocument(strings.NewReader(raw)); e == nil {
				t.Fatal("invalid accepted")
			}
		})
	}
	got, e := jcs.Transform([]byte(`{"z":1.0,"a":[true,"é",null]}`))
	if e != nil || string(got) != `{"a":[true,"é",null],"z":1}` {
		t.Fatal(string(got), e)
	}
}
func TestHTTPRejectsDuplicateAndCaseVariant(t *testing.T) {
	_, s := fixture()
	h := Handler{s, hostFixture{csrf: true}}
	for _, body := range []string{`{"reference":"missing","reference":"tx","format":"json","explicit":true}`, `{"Reference":"tx","format":"json","explicit":true}`} {
		w := httptestRecorder(t, h, body)
		if w != 400 {
			t.Fatal(body, w)
		}
	}
}
func FuzzDecodeDocument(f *testing.F) {
	_, s := fixture()
	d, _ := s.Receipt(context.Background(), Actor{"n", "a"}, DownloadRequest{"tx", "json", true})
	b, _ := d.JSON()
	f.Add(b)
	f.Add([]byte(`{"schema":"bad"}`))
	f.Fuzz(func(t *testing.T, b []byte) {
		d, e := DecodeDocument(bytes.NewReader(b))
		if e == nil {
			if _, e = d.Canonical(); e != nil {
				t.Fatal(e)
			}
		}
	})
}

func httptestRecorder(t *testing.T, h Handler, body string) int {
	t.Helper()
	w := httptest.NewRecorder()
	r := httptest.NewRequest("POST", "/receipt", strings.NewReader(body))
	r.Header.Set("Content-Type", "application/json")
	h.ServeHTTP(w, r)
	return w.Code
}
