package discovery

import (
	"crypto/rand"
	"encoding/hex"
	"strings"
	"unicode"
	"unicode/utf8"

	"golang.org/x/text/cases"
	"golang.org/x/text/runes"
	"golang.org/x/text/transform"
	"golang.org/x/text/unicode/norm"
)

// Fold mirrors Circula core's domain.FoldName so that search semantics match:
// NFKD, drop combining marks, Unicode case fold, NFC, dotless i to i.
func Fold(s string) string {
	t := transform.Chain(norm.NFKD, runes.Remove(runes.In(unicode.Mn)), cases.Fold(), norm.NFC)
	out, _, err := transform.String(t, s)
	if err != nil {
		out = s
	}
	return strings.Map(func(r rune) rune {
		if r == 'ı' {
			return 'i'
		}
		return r
	}, out)
}

func randomID(prefix string) string {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		panic("discovery: crypto/rand failed: " + err.Error())
	}
	return prefix + hex.EncodeToString(b[:])
}

func isBidiControl(r rune) bool {
	return r >= 0x202A && r <= 0x202E || r >= 0x2066 && r <= 0x2069 || r == 0x200E || r == 0x200F
}

// cleanText trims, collapses whitespace and rejects control, bidi override,
// angle-bracket and invalid UTF-8 content. Lengths are in runes.
func cleanText(field, s string, min, max int) (string, error) {
	if !utf8.ValidString(s) {
		return "", invalid(field, "must be valid UTF-8")
	}
	for _, r := range s {
		if (unicode.IsControl(r) && !unicode.IsSpace(r)) || isBidiControl(r) || r == '<' || r == '>' || r == 0xFFFD {
			return "", invalid(field, "contains disallowed characters")
		}
	}
	s = strings.Join(strings.Fields(s), " ")
	n := utf8.RuneCountInString(s)
	if n < min || n > max {
		return "", invalid(field, "length out of range")
	}
	return s, nil
}

// NormalizeTag returns the canonical tag form (folded, hyphenated) or an error.
func NormalizeTag(raw string) (string, error) {
	s, err := cleanText("tag", raw, 1, 48)
	if err != nil {
		return "", err
	}
	s = Fold(s)
	var b strings.Builder
	prevHyphen := true
	for _, r := range s {
		switch {
		case unicode.IsLetter(r) || unicode.IsDigit(r):
			b.WriteRune(r)
			prevHyphen = false
		case r == ' ' || r == '-' || r == '_':
			if !prevHyphen {
				b.WriteByte('-')
				prevHyphen = true
			}
		default:
			return "", invalid("tag", "tags may contain letters, digits and hyphens only")
		}
	}
	out := strings.TrimRight(b.String(), "-")
	if n := utf8.RuneCountInString(out); n < 2 || n > 24 {
		return "", invalid("tag", "tags must be 2 to 24 characters")
	}
	return out, nil
}

// validID accepts core-style opaque identifiers (ULID/UUID/slug shapes).
func validID(s string) bool {
	if len(s) == 0 || len(s) > 64 {
		return false
	}
	for i := 0; i < len(s); i++ {
		c := s[i]
		if !(c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c >= '0' && c <= '9' || c == '_' || c == '-') {
			return false
		}
	}
	return true
}

// ValidID reports whether s is an acceptable opaque identifier.
func ValidID(s string) bool { return validID(s) }
