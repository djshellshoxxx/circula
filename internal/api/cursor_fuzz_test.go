package api

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"unicode/utf8"
)

// ARC-010 (TB1) / API-034 / API-035: the pagination cursor is attacker input.
// Decoding any byte string must not panic; a cursor opens only for the filter
// it was minted for and only when this server minted it; every refusal is the
// same 400 invalid-cursor problem, whatever was wrong with it.

var (
	fuzzCursorKey  = bytes.Repeat([]byte{0x5a}, 32)
	fuzzCursorKey2 = bytes.Repeat([]byte{0xa5}, 32)
)

func wantInvalidCursor(t *testing.T, err error) {
	t.Helper()
	var ae *apiError
	if !errors.As(err, &ae) || ae.status != 400 || ae.code != "invalid-cursor" || ae.detail != "cursor is not valid for this request" {
		t.Fatalf("refusal = %#v, want the uniform 400 invalid-cursor problem", err)
	}
}

func FuzzOpenCursor(f *testing.F) {
	s := &server{cursor: fuzzCursorKey}
	other := &server{cursor: fuzzCursorKey2}
	filter := "/api/v1/transactions|net-1|person-1|completed|"
	valid := s.sealCursor("018f0000-0000-7000-8000-000000000042", filter)
	payload, sig, _ := strings.Cut(valid, ".")
	for _, tok := range []string{
		valid, "", ".", "..", valid + ".", "." + sig, payload + ".", payload, valid + "A", valid[:len(valid)-1],
		strings.ToUpper(valid), strings.Replace(valid, ".", "..", 1), payload + "." + sig + "=", payload + "=." + sig,
		payload + "\n." + sig, other.sealCursor("x", filter), s.sealCursor("x", filter+"x"), s.sealCursor("", filter),
		base64.RawURLEncoding.EncodeToString([]byte(`{"p":"x","f":"` + filter + `"}`)) + "." + sig,
		base64.RawURLEncoding.EncodeToString([]byte(`not json`)) + "." + base64.RawURLEncoding.EncodeToString(s.mac([]byte(`not json`))),
		"%00", "\x00.\x00", strings.Repeat("A", 4096) + "." + strings.Repeat("B", 4096),
	} {
		f.Add(tok, filter, "pos")
	}
	f.Add(valid, "", "")
	f.Add(valid, "\xff", "\xfe")
	f.Fuzz(func(t *testing.T, tok, filter, pos string) {
		// Arbitrary tokens: either the uniform refusal, or a token that this
		// key minted for exactly this filter (the fuzzer cannot forge the MAC,
		// so the only accepted inputs are re-encodings of minted payloads).
		got, err := s.openCursor(tok, filter)
		if err != nil {
			wantInvalidCursor(t, err)
		} else {
			enc, _, _ := strings.Cut(tok, ".")
			raw, derr := base64.RawURLEncoding.DecodeString(enc)
			want, _ := json.Marshal(cursorBody{P: got, F: filter})
			if derr != nil || !bytes.Equal(raw, want) {
				t.Fatalf("accepted %q for filter %q with a payload this server never minted: %q", tok, filter, raw)
			}
			if _, err := other.openCursor(tok, filter); err == nil {
				t.Fatalf("token %q opened under a different cursor key", tok)
			}
		}

		// Minted tokens: URL-safe, and bound to filter and key. Cursors only
		// carry ids and filter strings; JSON cannot carry invalid UTF-8, so the
		// round trip is only promised for valid UTF-8 (see the trust-boundary
		// checklist, TB1 notes).
		minted := s.sealCursor(pos, filter)
		if strings.Trim(minted, "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789-_.") != "" {
			t.Fatalf("minted cursor %q is not URL-safe", minted)
		}
		if _, err := other.openCursor(minted, filter); err == nil {
			t.Fatal("cursor opened under a different key")
		}
		if _, err := s.openCursor(minted, filter+"|x"); err == nil {
			t.Fatal("cursor opened for a different filter")
		}
		if !utf8.ValidString(pos) || !utf8.ValidString(filter) {
			return
		}
		back, err := s.openCursor(minted, filter)
		if err != nil || back != pos {
			t.Fatalf("round trip of %q for %q = %q, %v", pos, filter, back, err)
		}
	})
}
