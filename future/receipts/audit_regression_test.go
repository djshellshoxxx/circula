package receipts

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"
	"time"
)

type readFunc func(context.Context, Actor, string) (Trade, error)

func (f readFunc) ReadReceipt(ctx context.Context, a Actor, id string) (Trade, error) {
	return f(ctx, a, id)
}

type auditFunc func(context.Context, string) error

func (f auditFunc) ReceiptOutcome(ctx context.Context, outcome string) error { return f(ctx, outcome) }
func TestAuditRejectsReceiptCancellationInsideCallbacks(t *testing.T) {
	for _, stage := range []string{"reader", "audit", "network"} {
		t.Run(stage, func(t *testing.T) {
			reader, s := fixture()
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			switch stage {
			case "reader":
				s.Reader = readFunc(func(context.Context, Actor, string) (Trade, error) { cancel(); return reader.snapshot, nil })
			case "audit":
				s.Audit = auditFunc(func(context.Context, string) error { cancel(); return nil })
			case "network":
				s.NetworkEnabled = func(context.Context, string) bool { cancel(); return true }
			}
			var out bytes.Buffer
			if e := s.Download(ctx, Actor{"n", "a"}, DownloadRequest{"tx", "json", true}, &out); !errors.Is(e, context.Canceled) || out.Len() != 0 {
				t.Fatal(e, out.String())
			}
		})
	}
}
func TestAuditRejectsLateCertificateAccessRevocation(t *testing.T) {
	for _, stage := range []string{"membership", "receipt-enable", "certificate-enable", "projection"} {
		t.Run(stage, func(t *testing.T) {
			issuer, _, _, handle := certificateFixture(t)
			reader := issuer.Receipts.Reader.(*fixtureReader)
			enabled := true
			certificateEnabled := true
			issuer.Receipts.NetworkEnabled = func(context.Context, string) bool { return enabled }
			issuer.NetworkEnabled = func(context.Context, string) bool { return certificateEnabled }
			audits := 0
			issuer.Receipts.Audit = auditFunc(func(context.Context, string) error { audits++; return nil })
			handle.before = func() {
				switch stage {
				case "membership":
					reader.denial = true
				case "receipt-enable":
					enabled = false
				case "certificate-enable":
					certificateEnabled = false
				case "projection":
					reader.snapshot.Supplement = &StatusSupplement{"reversal", "reversed", time.Unix(150, 0).UTC().Format(time.RFC3339Nano)}
				}
			}
			token, e := issuer.Issue(context.Background(), Actor{"n", "a"}, CertificateRequest{"tx", true})
			if !errors.Is(e, ErrCertificateUnavailable) || token != "" {
				t.Fatal(e, token)
			}
			if audits != 1 {
				t.Fatal("recheck duplicated audit", audits)
			}
		})
	}
}
func TestAuditSuccessfulFinalRecheckDoesNotDuplicateAudit(t *testing.T) {
	issuer, _, trust, _ := certificateFixture(t)
	audits := 0
	issuer.Receipts.Audit = auditFunc(func(context.Context, string) error { audits++; return nil })
	token := issueFixture(t, issuer)
	if audits != 1 || issuer.Receipts.Reader.(*fixtureReader).calls.Load() != 2 || verifyFixture(token, trust).Reason != "valid" {
		t.Fatal(audits)
	}
}

type panicReader struct{}

func (panicReader) Read([]byte) (int, error) { panic("cancelled verifier must not read") }
func TestAuditCancelledVerifierDoesNotRead(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if v := VerifyFile(ctx, panicReader{}, "deployment", "n", nil, time.Unix(300, 0)); v.Reason != "cancelled" {
		t.Fatal(v)
	}
}

func TestAuditPartyAndExplicitScopedAdminReaderContract(t *testing.T) {
	original, s := fixture()
	s.Reader = readFunc(func(_ context.Context, a Actor, ref string) (Trade, error) {
		if a.Network != "n" || ref != "tx" || (a.Member != "a" && a.Member != "b" && a.Member != "scoped-admin") {
			return Trade{}, ErrUnavailable
		}
		return original.snapshot, nil
	})
	for _, member := range []string{"a", "b", "scoped-admin"} {
		for _, format := range []string{"txt", "json", "html"} {
			var out bytes.Buffer
			if e := s.Download(context.Background(), Actor{"n", member}, DownloadRequest{"tx", format, true}, &out); e != nil || out.Len() == 0 {
				t.Fatal(member, format, e)
			}
		}
	}
	for _, a := range []Actor{{"n", "reviewer"}, {"n", "unscoped-admin"}, {"other", "a"}} {
		var out bytes.Buffer
		if e := s.Download(context.Background(), a, DownloadRequest{"tx", "json", true}, &out); !errors.Is(e, ErrUnavailable) || out.Len() != 0 {
			t.Fatal(a, e, out.String())
		}
	}
}

func TestAuditMalformedOfflineDocumentReturnsNoPartialProjection(t *testing.T) {
	_, s := fixture()
	d, e := s.Receipt(context.Background(), Actor{"n", "a"}, DownloadRequest{"tx", "json", true})
	if e != nil {
		t.Fatal(e)
	}
	raw, _ := d.JSON()
	malformed := strings.Replace(string(raw), `"precision":2`, `"precision":"invalid"`, 1)
	decoded, e := DecodeDocument(strings.NewReader(malformed))
	if e == nil || decoded.Schema != "" || decoded.Network != "" || decoded.Amount != 0 {
		t.Fatal(decoded, e)
	}
}
