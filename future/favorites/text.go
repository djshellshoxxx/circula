package favorites

import (
	"strings"
	"unicode"
	"unicode/utf8"
)

// Text limits follow the core application's member-visible text rules
// (internal/service/validate.go and listings.go): bounded rune counts, no
// control characters, no invisible format runes (zero-width, bidi overrides,
// BOM) and no line/paragraph separators.
const (
	MaxNicknameRunes    = 80
	MaxDescriptionRunes = 2000
)

// cleanText validates v and returns it trimmed of surrounding white space.
// multiline allows newline, carriage return and tab.
func cleanText(field, v string, max int, multiline bool) (string, error) {
	if !utf8.ValidString(v) {
		return "", invalid(field, "must be valid UTF-8")
	}
	v = strings.TrimSpace(v)
	if utf8.RuneCountInString(v) > max {
		return "", invalid(field, "too long")
	}
	for _, r := range v {
		if multiline && (r == '\n' || r == '\r' || r == '\t') {
			continue
		}
		if unicode.IsControl(r) || unicode.Is(unicode.Cf, r) || unicode.Is(unicode.Zl, r) || unicode.Is(unicode.Zp, r) || r == utf8.RuneError {
			return "", invalid(field, "must not contain control or invisible formatting characters")
		}
	}
	return v, nil
}

// CleanNickname validates and trims a private nickname. An empty nickname is allowed.
func CleanNickname(v string) (string, error) {
	return cleanText("nickname", v, MaxNicknameRunes, false)
}

// CleanDescription validates and trims a copied trade description.
func CleanDescription(v string) (string, error) {
	return cleanText("description", v, MaxDescriptionRunes, true)
}

// validID reports whether s is an acceptable opaque identifier: non-empty,
// bounded, printable ASCII without whitespace. Real member IDs are supplied by
// the host application; this only bounds what flows into storage and markup.
func validID(s string) bool {
	if s == "" || len(s) > 128 {
		return false
	}
	for i := 0; i < len(s); i++ {
		if s[i] <= 0x20 || s[i] >= 0x7f {
			return false
		}
	}
	return true
}

func validKey(s string) bool {
	if s == "" || len(s) > 64 {
		return false
	}
	for i := 0; i < len(s); i++ {
		c := s[i]
		if !(c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c >= '0' && c <= '9' || c == '-' || c == '_') {
			return false
		}
	}
	return true
}
