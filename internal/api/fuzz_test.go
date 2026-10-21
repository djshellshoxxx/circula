package api

import (
	"encoding/json"
	"strconv"
	"testing"
)

// RB-017: decoder and header validators must never panic and must keep their
// documented guarantees for any input.

func FuzzValidKey(f *testing.F) {
	for _, s := range []string{"", "a", "abc-123", "with space", "tab\t", "\x7f", "é", string(make([]byte, 256))} {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, s string) {
		if !validKey(s) {
			return
		}
		if len(s) < 1 || len(s) > 255 {
			t.Fatalf("accepted length %d", len(s))
		}
		for i := 0; i < len(s); i++ {
			if s[i] < 0x20 || s[i] > 0x7e {
				t.Fatalf("accepted non-printable byte %#x", s[i])
			}
		}
	})
}

func FuzzRequestIDPattern(f *testing.F) {
	for _, s := range []string{"", "short", "abcdefgh", "a.b_c-d.e12345", "has space 12345", "line\nbreak1234", "é-unicode-1234"} {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, s string) {
		if !ridPattern.MatchString(s) {
			return
		}
		if len(s) < 8 || len(s) > 64 {
			t.Fatalf("accepted length %d", len(s))
		}
		for i := 0; i < len(s); i++ {
			c := s[i]
			ok := c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c >= '0' && c <= '9' || c == '.' || c == '_' || c == '-'
			if !ok {
				t.Fatalf("accepted byte %q", c)
			}
		}
	})
}

type fuzzBody struct {
	A string          `json:"a"`
	B *int64          `json:"b"`
	C []string        `json:"c"`
	D map[string]any  `json:"d"`
	E json.RawMessage `json:"e"`
}

func FuzzStrictUnmarshal(f *testing.F) {
	for _, s := range []string{`{}`, `{"a":"x"}`, `{"a":"x","a":"y"}`, `{"z":1}`, `{"a":1}`, `{"a":"x"} {}`, `{"d":{"k":1,"k":2}}`,
		`[1,2]`, ``, `{`, `{"e":[{"x":1,"x":2}]}`, "{\"a\":\"\\ud800\"}", `{"b":99999999999999999999}`} {
		f.Add([]byte(s))
	}
	f.Fuzz(func(t *testing.T, b []byte) {
		var v fuzzBody
		if err := strictUnmarshal(b, &v); err != nil {
			return
		}
		// Accepted means: valid JSON, no duplicate keys anywhere, one document.
		if !json.Valid(b) {
			t.Fatalf("accepted invalid JSON %q", b)
		}
		if err := checkDuplicateKeys(b); err != nil {
			t.Fatalf("accepted duplicate keys %q: %v", b, err)
		}
	})
}

func FuzzParseMinor(f *testing.F) {
	for _, s := range []string{"", "0", "1", "01", "-1", "+1", "1.0", "1e2", " 1", "9223372036854775807", "9223372036854775808", "99999999999999999999"} {
		f.Add(s, false)
		f.Add(s, true)
	}
	f.Fuzz(func(t *testing.T, s string, allowZero bool) {
		v, err := parseMinor("amount", s, allowZero)
		if err != nil {
			return
		}
		if v < 0 || (v == 0 && !allowZero) {
			t.Fatalf("accepted %q as %d", s, v)
		}
		if strconv.FormatInt(v, 10) != s {
			t.Fatalf("non-canonical %q accepted as %d", s, v)
		}
	})
}
