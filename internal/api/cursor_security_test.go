package api

import (
	"strings"
	"testing"
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
