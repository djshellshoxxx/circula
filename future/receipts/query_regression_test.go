package receipts

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

type queryHost struct {
	hostFixture
	calls int
}

func (h *queryHost) Authorize(r *http.Request, unsafe bool) (Actor, error) {
	h.calls++
	return h.hostFixture.Authorize(r, unsafe)
}

func TestDownloadQueryBoundaryBeforeHost(t *testing.T) {
	for _, certificate := range []bool{false, true} {
		for _, method := range []string{"GET", "POST"} {
			for _, query := range []string{"reference=tx&reference=other", "reference=tx&amount=1", "reference=tx&bad=%zz", "reference=", "reference=" + strings.Repeat("a", 8192)} {
				_, service := fixture()
				host := &queryHost{hostFixture: hostFixture{csrf: true}}
				var handler http.Handler = Handler{service, host}
				if certificate {
					handler = CertificateHandler{Issuer: Issuer{Enabled: true, Receipts: service, NetworkEnabled: func(context.Context, string) bool { return true }}, Host: host}
				}
				r := httptest.NewRequest(method, "/receipt?"+query, strings.NewReader("reference=tx&format=txt&explicit=true"))
				r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
				w := httptest.NewRecorder()
				handler.ServeHTTP(w, r)
				if w.Code != 400 || host.calls != 0 || r.Form != nil {
					t.Fatalf("certificate=%t method=%s query rejected late: status=%d calls=%d", certificate, method, w.Code, host.calls)
				}
			}
		}
	}
}

func TestReceiptChoicePostQueryAgreement(t *testing.T) {
	for _, reference := range []string{"", "tx", "other"} {
		reader, service := fixture()
		h := Handler{service, hostFixture{csrf: true}}
		path := "/receipt"
		if reference != "" {
			path += "?reference=" + reference
		}
		r := httptest.NewRequest("POST", path, strings.NewReader("reference=tx&format=txt&explicit=true&csrf_token=synthetic-csrf"))
		r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		if reference == "other" {
			if w.Code != 400 || reader.calls.Load() != 0 {
				t.Fatalf("conflicting URL/body reached reader: %d", w.Code)
			}
		} else if w.Code != 200 || reader.calls.Load() != 1 {
			t.Fatalf("valid no-JS submission failed: %d %s", w.Code, w.Body.String())
		}
	}
}

func TestCertificateChoicePostQueryAgreement(t *testing.T) {
	for _, reference := range []string{"", "tx", "other"} {
		issuer, keys, trust, _ := certificateFixture(t)
		reader := issuer.Receipts.Reader.(*fixtureReader)
		h := CertificateHandler{issuer, hostFixture{csrf: true}}
		path := "/certificate"
		if reference != "" {
			path += "?reference=" + reference
		}
		r := httptest.NewRequest("POST", path, strings.NewReader("reference=tx&format=jws&explicit=true&csrf_token=synthetic-csrf"))
		r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		if reference == "other" {
			if w.Code != 400 || reader.calls.Load() != 0 || keys.calls.Load() != 0 {
				t.Fatalf("conflicting URL/body reached private source or signer: status=%d reads=%d keys=%d", w.Code, reader.calls.Load(), keys.calls.Load())
			}
		} else if w.Code != 200 || verifyFixture(w.Body.String(), trust).Reason != "valid" {
			t.Fatalf("valid certificate form submission failed: status=%d", w.Code)
		}
	}
}

type parsingQueryHost struct{ hostFixture }

func (h parsingQueryHost) Authorize(r *http.Request, unsafe bool) (Actor, error) {
	// Hosts often read form fields while checking cookie-authenticated CSRF.
	_ = r.ParseForm()
	return h.hostFixture.Authorize(r, unsafe)
}

func TestMalformedFormCannotBeHiddenByHostParsing(t *testing.T) {
	for _, malformed := range []string{"discarded=%zz", "reference=%zz", "discarded=a;b"} {
		for _, certificate := range []bool{false, true} {
			reader, service := fixture()
			var handler http.Handler = Handler{service, parsingQueryHost{hostFixture{csrf: true}}}
			format := "txt"
			if certificate {
				issuer, _, _, _ := certificateFixture(t)
				issuer.Receipts.Reader = reader
				handler = CertificateHandler{issuer, parsingQueryHost{hostFixture{csrf: true}}}
				format = "jws"
			}
			r := httptest.NewRequest("POST", "/download?reference=tx", strings.NewReader("reference=tx&format="+format+"&explicit=true&csrf_token=synthetic-csrf&"+malformed))
			r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			w := httptest.NewRecorder()
			handler.ServeHTTP(w, r)
			if w.Code != 400 || reader.calls.Load() != 0 {
				t.Fatalf("malformed form accepted after host parsing: certificate=%t status=%d source_reads=%d", certificate, w.Code, reader.calls.Load())
			}
		}
	}
}
