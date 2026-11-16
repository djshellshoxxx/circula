package domain

import (
	"unicode"

	"golang.org/x/text/cases"
	"golang.org/x/text/runes"
	"golang.org/x/text/transform"
	"golang.org/x/text/unicode/norm"
)

// FoldName reduces a display name or a search term to a comparison key so that
// directory search ignores case and accents: it decomposes (NFKD), drops
// combining marks, applies full Unicode case folding and maps the Turkish
// dotless i to i. "Émile", "EMILE" and "émile" all fold to "emile";
// "İstanbul", "ISTANBUL" and "ıstanbul" all fold to "istanbul". The result is
// for matching only and is never shown or stored as the name.
func FoldName(s string) string {
	// Transformers are stateful, so each call builds its own chain.
	t := transform.Chain(norm.NFKD, runes.Remove(runes.In(unicode.Mn)), cases.Fold(), norm.NFC)
	out, _, err := transform.String(t, s)
	if err != nil {
		return s
	}
	// U+0131 (dotless i) has no case-fold partner outside Turkic locales; treat it as i for search.
	rs := []rune(out)
	for i, r := range rs {
		if r == 'ı' {
			rs[i] = 'i'
		}
	}
	return string(rs)
}
