// Package logx is the application-log boundary (SEC-020, OPS-032). The HTTP
// adapters and the serve command log only through a logger wrapped by New or
// Wrap, which, whatever the underlying slog handler:
//
//   - replaces the value of any attribute whose key names a credential
//     (password, secret, token, session, cookie, CSRF, TOTP, OTP, recovery
//     code, API key, authorization) with Redacted;
//   - redacts invitation and API tokens, bearer values and key=value
//     credentials inside free text, and masks email addresses and
//     international phone numbers;
//   - escapes newlines, other control characters, Unicode line separators and
//     bidirectional or invisible format characters (and the backslash that
//     introduces an escape), so user-controlled text cannot end an entry,
//     forge a second one or hide part of a line;
//   - stamps every entry in UTC.
//
// Errors, Stringers and other values are rendered to text and cleaned the
// same way. Integers, booleans, durations and times pass unchanged.
package logx

import (
	"context"
	"fmt"
	"log/slog"
	"regexp"
	"strings"
	"unicode"
	"unicode/utf8"
)

// Redacted replaces the value of a sensitive attribute.
const Redacted = "[REDACTED]"

// New returns l (slog.Default() when nil) with its handler wrapped by Wrap.
// A logger that is already wrapped is returned unchanged.
func New(l *slog.Logger) *slog.Logger {
	if l == nil {
		l = slog.Default()
	}
	if _, ok := l.Handler().(*handler); ok {
		return l
	}
	return slog.New(&handler{inner: l.Handler()})
}

// Wrap returns h behind the redacting, escaping handler. Wrapping twice adds
// no second layer.
func Wrap(h slog.Handler) slog.Handler {
	if _, ok := h.(*handler); ok {
		return h
	}
	return &handler{inner: h}
}

type handler struct{ inner slog.Handler }

func (h *handler) Enabled(ctx context.Context, l slog.Level) bool { return h.inner.Enabled(ctx, l) }

func (h *handler) Handle(ctx context.Context, r slog.Record) error {
	out := slog.NewRecord(r.Time.UTC(), r.Level, Clean(r.Message), r.PC)
	r.Attrs(func(a slog.Attr) bool {
		out.AddAttrs(cleanAttr(a))
		return true
	})
	return h.inner.Handle(ctx, out)
}

func (h *handler) WithAttrs(as []slog.Attr) slog.Handler {
	clean := make([]slog.Attr, len(as))
	for i, a := range as {
		clean[i] = cleanAttr(a)
	}
	return &handler{inner: h.inner.WithAttrs(clean)}
}

func (h *handler) WithGroup(name string) slog.Handler {
	return &handler{inner: h.inner.WithGroup(Clean(name))}
}

func cleanAttr(a slog.Attr) slog.Attr {
	key := Clean(a.Key)
	if sensitiveKey(a.Key) {
		return slog.String(key, Redacted)
	}
	v := a.Value.Resolve()
	switch v.Kind() {
	case slog.KindString:
		return slog.String(key, Clean(v.String()))
	case slog.KindTime:
		return slog.Time(key, v.Time().UTC())
	case slog.KindGroup:
		g := v.Group()
		out := make([]slog.Attr, len(g))
		for i, x := range g {
			out[i] = cleanAttr(x)
		}
		return slog.Attr{Key: key, Value: slog.GroupValue(out...)}
	case slog.KindAny:
		// fmt recovers from a panicking Error or String method.
		return slog.String(key, Clean(fmt.Sprintf("%+v", v.Any())))
	default:
		return slog.Attr{Key: key, Value: v}
	}
}

// sensitiveFragments mark a credential when they appear anywhere in the key
// (lower-cased, separators removed); sensitiveWords only as a whole word.
var (
	sensitiveFragments = []string{"password", "passwd", "passphrase", "secret", "token", "csrf", "xsrf", "session",
		"cookie", "authorization", "bearer", "totp", "credential", "apikey", "privatekey", "recoverycode"}
	sensitiveWords = map[string]bool{"otp": true, "sid": true, "pwd": true, "pw": true, "auth": true}
	keyWord        = regexp.MustCompile(`[^a-z0-9]+`)
)

func sensitiveKey(k string) bool {
	lower := strings.ToLower(k)
	joined := keyWord.ReplaceAllString(lower, "")
	for _, f := range sensitiveFragments {
		if strings.Contains(joined, f) {
			return true
		}
	}
	for _, w := range keyWord.Split(lower, -1) {
		if sensitiveWords[w] {
			return true
		}
	}
	return false
}

var (
	// Invitation tokens (inv_...) and API tokens (cir_...), as service.RedactSecrets.
	bearerToken = regexp.MustCompile(`\b(inv|cir)_[A-Za-z0-9_-]{8,}`)
	bearerValue = regexp.MustCompile(`(?i)\b(bearer|basic)\s+[A-Za-z0-9._~+/=-]+`)
	// key=value, key: value and "key":"value" credentials in free text.
	credentialKV = regexp.MustCompile(`(?i)((?:password|passwd|passphrase|secret|token|csrf|xsrf|session(?:[_-]?id)?|cookie|` +
		`totp|otp|recovery[_ -]?codes?|api[_-]?key|authorization)"?\s*[=:]\s*)("[^"]*"|'[^']*'|[^\s"',;&]+)`)
	email = regexp.MustCompile(`[A-Za-z0-9._%+-]+@[A-Za-z0-9-]+(?:\.[A-Za-z0-9-]+)*\.[A-Za-z]{2,}\b`)
	phone = regexp.MustCompile(`\+\d[\d ().-]{6,}\d`)
)

// Clean returns s with credentials redacted, email addresses and
// international phone numbers masked, and every character that could break
// or disguise a log line escaped (\n, \r, \t, \xNN, \uNNNN; a backslash
// becomes \\). Text that needs none of this is returned unchanged.
func Clean(s string) string {
	s = bearerToken.ReplaceAllString(s, "${1}_REDACTED")
	s = bearerValue.ReplaceAllString(s, "${1} "+Redacted)
	s = credentialKV.ReplaceAllString(s, "${1}"+Redacted)
	s = email.ReplaceAllString(s, "[email]")
	s = phone.ReplaceAllString(s, "[phone]")
	if !needsEscape(s) {
		return s
	}
	var b strings.Builder
	b.Grow(len(s) + 16)
	for i := 0; i < len(s); {
		r, n := utf8.DecodeRuneInString(s[i:])
		switch {
		case r == utf8.RuneError && n <= 1:
			b.WriteString(`\ufffd`)
		case r == '\\':
			b.WriteString(`\\`)
		case r == '\n':
			b.WriteString(`\n`)
		case r == '\r':
			b.WriteString(`\r`)
		case r == '\t':
			b.WriteString(`\t`)
		case r < 0x80 && unicode.IsControl(r):
			fmt.Fprintf(&b, `\x%02x`, r)
		case unsafeRune(r):
			fmt.Fprintf(&b, `\u%04x`, r)
		default:
			b.WriteString(s[i : i+n])
		}
		i += n
	}
	return b.String()
}

// unsafeRune reports runes outside ASCII that end or disguise a line: C1
// controls (including NEL), line and paragraph separators, and format
// characters such as bidirectional overrides and zero-width joiners.
func unsafeRune(r rune) bool {
	return unicode.IsControl(r) || unicode.In(r, unicode.Zl, unicode.Zp, unicode.Cf)
}

func needsEscape(s string) bool {
	for i := 0; i < len(s); i++ {
		if c := s[i]; c < 0x20 || c == 0x7f || c == '\\' || c >= 0x80 {
			if c >= 0x80 {
				return needsEscapeUnicode(s[i:])
			}
			return true
		}
	}
	return false
}

func needsEscapeUnicode(s string) bool {
	for i := 0; i < len(s); {
		r, n := utf8.DecodeRuneInString(s[i:])
		if (r == utf8.RuneError && n <= 1) || r < 0x20 || r == 0x7f || r == '\\' || unsafeRune(r) {
			return true
		}
		i += n
	}
	return false
}
