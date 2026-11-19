package api

import (
	"errors"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/djshellshoxxx/circula/internal/domain"
)

// Pagination rejects oversized or noncanonical encodings before accepting an
// otherwise valid HMAC. A canonical cursor must have exactly one spelling.
func TestPaginationCursorSecurityBoundsAndCanonicalEncoding(t *testing.T) {
	s := &server{cursor: fuzzCursorKey}
	filter := "/api/v1/me/sessions||owner|session-public-id-v2"
	position := "0:" + strings.Repeat("a", 64)
	valid := s.sealCursor(position, filter)
	payload, signature, ok := strings.Cut(valid, ".")
	if !ok {
		t.Fatal("fixture cursor has no signature")
	}
	const alphabet = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789-_"
	last := strings.IndexByte(alphabet, signature[len(signature)-1])
	if last < 0 || last&3 != 0 {
		t.Fatal("fixture signature is not canonical")
	}
	alias := signature[:len(signature)-1] + string(alphabet[last|1])
	tests := []struct {
		name, token string
	}{
		{"payload newline", payload[:1] + "\n" + payload[1:] + "." + signature},
		{"signature newline", payload + "." + signature[:1] + "\n" + signature[1:]},
		{"signature spare bits", payload + "." + alias},
		{"oversized signed payload", s.sealCursor(strings.Repeat("a", 8192), filter)},
		{"oversized unsigned payload", strings.Repeat("a", 8192) + "." + signature},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := s.openCursor(tt.token, filter)
			if err == nil {
				t.Fatal("noncanonical or oversized cursor accepted")
			}
			wantInvalidCursor(t, err)
		})
	}
	if got, err := s.openCursor(valid, filter); err != nil || got != position {
		t.Fatalf("canonical cursor rejected: %q, %v", got, err)
	}
}

// Bound filter growth before querying, so the API never returns a next cursor
// that the size-limited decoder refuses. The largest supported listing query
// remains usable even when every search byte needs JSON escaping.
func TestPaginationFilterBudgetKeepsSupportedQueries(t *testing.T) {
	s := &server{cursor: fuzzCursorKey}
	c := &reqCtx{s: s, r: httptest.NewRequest("GET", "/api/v1/listings", nil),
		netID: "018f0000-0000-7000-8000-000000000001",
		actor: domain.Actor{PersonID: "018f0000-0000-7000-8000-000000000002"}}
	filter := "request|" + strings.Repeat("\x00", 200) + "|018f0000-0000-7000-8000-000000000003"
	p, err := c.page(filter)
	if err != nil {
		t.Fatalf("maximum supported listing filter rejected: %v", err)
	}
	position := strings.Repeat("a", 256)
	token := s.sealCursor(position, p.filter)
	if len(token) > 4096 {
		t.Fatalf("maximum supported filter exceeds cursor budget: %d bytes", len(token))
	}
	if got, err := s.openCursor(token, p.filter); err != nil || got != position {
		t.Fatalf("maximum supported filter cannot resume: %q, %v", got, err)
	}
	for _, oversized := range []string{strings.Repeat("a", 8192), strings.Repeat("\x00", 700)} {
		_, err := c.page(oversized)
		var ae *apiError
		if !errors.As(err, &ae) || ae.status != 400 || ae.code != "invalid-query" {
			t.Fatalf("oversized filter returned %v, want 400 invalid-query", err)
		}
	}
}

func TestPaginationCursorExactSizeBoundary(t *testing.T) {
	s := &server{cursor: fuzzCursorKey}
	const filter = "boundary"
	var boundary, over string
	for n := 2800; n < 4096; n++ {
		token := s.sealCursor(strings.Repeat("x", n), filter)
		if len(token) == 4096 {
			boundary = token
		}
		if len(token) > 4096 {
			over = token
			break
		}
	}
	if boundary == "" || over == "" {
		t.Fatal("size-boundary fixtures not found")
	}
	if _, err := s.openCursor(boundary, filter); err != nil {
		t.Fatalf("4096-byte canonical cursor rejected: %v", err)
	}
	_, err := s.openCursor(over, filter)
	wantInvalidCursor(t, err)
}

func BenchmarkPaginationCursorRejectedOversize(b *testing.B) {
	s := &server{cursor: fuzzCursorKey}
	token := strings.Repeat("A", 1<<20) + "." + strings.Repeat("A", 43)
	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		if _, err := s.openCursor(token, "filter"); err == nil {
			b.Fatal("oversized cursor accepted")
		}
	}
}
