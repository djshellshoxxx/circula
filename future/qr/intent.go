// Package qr implements versioned QR identifiers and one-time trade requests
// for Circula as an isolated module. It never posts ledger entries: a claim
// creates an ordinary proposal through an injected same-write-unit Core adapter,
// and the proposal still needs the counterparty's normal acceptance.
package qr

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"net/url"
	"strings"
	"time"
)

// ErrInvalid is returned for malformed, tampered or semantically unusable input.
var ErrInvalid = errors.New("invalid QR request")

const (
	// MaxLinkLen bounds every scanned or pasted code before parsing.
	MaxLinkLen = 2048
	// LinkVersionPrefix is the path prefix of the first (v1) link format.
	LinkVersionPrefix = "/qr/v1/"
	aliasSegment      = "a/"
)

// Kind says what a parsed code refers to.
type Kind string

// Code kinds.
const (
	KindRequest Kind = "request" // one-time trade request token
	KindAlias   Kind = "alias"   // static member alias
)

// NewToken returns a 128-bit random opaque code (unpadded base64url, 22 chars).
func NewToken() (string, error) {
	b := make([]byte, 16)
	if _, e := rand.Read(b); e != nil {
		return "", e
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

// TokenHash validates the canonical shape of a code and returns the SHA-256 hash
// that is the only form ever persisted.
func TokenHash(token string) ([32]byte, error) {
	b, e := base64.RawURLEncoding.DecodeString(token)
	if e != nil || len(b) != 16 || base64.RawURLEncoding.EncodeToString(b) != token {
		return [32]byte{}, ErrInvalid
	}
	return sha256.Sum256([]byte(token)), nil
}

func origin(raw string) (*url.URL, error) {
	u, e := url.Parse(raw)
	if e != nil || u.Scheme != "https" || u.Host == "" || u.User != nil || u.Path != "" || u.RawQuery != "" || u.ForceQuery || u.Fragment != "" || u.Opaque != "" {
		return nil, ErrInvalid
	}
	u.Host = strings.ToLower(u.Host)
	return u, nil
}

// ValidateOrigin reports whether raw is a usable trusted https origin
// (scheme and host[:port] only).
func ValidateOrigin(raw string) error {
	_, e := origin(raw)
	return e
}

// Link builds the one-time request URL on the trusted origin.
func Link(trustedOrigin, token string) (string, error) {
	return link(trustedOrigin, LinkVersionPrefix, token)
}

// AliasLink builds the static member alias URL on the trusted origin.
func AliasLink(trustedOrigin, alias string) (string, error) {
	return link(trustedOrigin, LinkVersionPrefix+aliasSegment, alias)
}

func link(trustedOrigin, prefix, code string) (string, error) {
	u, e := origin(trustedOrigin)
	if e != nil {
		return "", e
	}
	if _, e = TokenHash(code); e != nil {
		return "", e
	}
	u.Path = prefix + code
	return u.String(), nil
}

// Parsed is the result of parsing a scanned or pasted code.
type Parsed struct {
	Kind Kind
	Code string
}

// ParseLink parses a full QR URL. The scheme and host must match the trusted
// origin exactly; userinfo, query, fragment, encoded separators and anything
// except the two v1 shapes are rejected. It never panics.
func ParseLink(trustedOrigin, raw string) (Parsed, error) {
	base, e := origin(trustedOrigin)
	if e != nil || len(raw) > MaxLinkLen || strings.Contains(raw, "#") {
		return Parsed{}, ErrInvalid
	}
	u, e := url.Parse(raw)
	if e != nil || u.Scheme != base.Scheme || u.Host != base.Host || u.User != nil || u.Opaque != "" ||
		u.RawQuery != "" || u.ForceQuery || u.Fragment != "" || u.RawPath != "" || !strings.HasPrefix(u.Path, LinkVersionPrefix) {
		return Parsed{}, ErrInvalid
	}
	rest := strings.TrimPrefix(u.Path, LinkVersionPrefix)
	kind := KindRequest
	if strings.HasPrefix(rest, aliasSegment) {
		kind, rest = KindAlias, strings.TrimPrefix(rest, aliasSegment)
	}
	if _, e = TokenHash(rest); e != nil {
		return Parsed{}, ErrInvalid
	}
	return Parsed{Kind: kind, Code: rest}, nil
}

// Parse returns the request token from a one-time request URL. Alias URLs are
// rejected; use ParseLink for both.
func Parse(trustedOrigin, raw string) (string, error) {
	p, e := ParseLink(trustedOrigin, raw)
	if e != nil || p.Kind != KindRequest {
		return "", ErrInvalid
	}
	return p.Code, nil
}

// CompactCode renders the printable manual-entry form of a code: "r." (one-time
// request) or "a." (static alias) followed by the 22-character code.
func CompactCode(k Kind, code string) string {
	if k == KindAlias {
		return "a." + code
	}
	return "r." + code
}

// ParseManual accepts what a person may paste or type: a full QR URL or a
// compact code. Surrounding whitespace is ignored; embedded whitespace is not.
func ParseManual(trustedOrigin, raw string) (Parsed, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" || len(raw) > MaxLinkLen {
		return Parsed{}, ErrInvalid
	}
	if len(raw) == 24 && raw[1] == '.' && (raw[0] == 'r' || raw[0] == 'a') {
		k := KindRequest
		if raw[0] == 'a' {
			k = KindAlias
		}
		if _, e := TokenHash(raw[2:]); e != nil {
			return Parsed{}, ErrInvalid
		}
		return Parsed{Kind: k, Code: raw[2:]}, nil
	}
	return ParseLink(trustedOrigin, raw)
}

// RedactPath replaces any code in a /qr/v1/... or /api/v1/qr-... request path
// with "[redacted]" so access logs never hold bearer-like tokens.
func RedactPath(p string) string {
	if i := strings.Index(p, LinkVersionPrefix); i >= 0 {
		head := p[:i+len(LinkVersionPrefix)]
		rest := p[i+len(LinkVersionPrefix):]
		if strings.HasPrefix(rest, aliasSegment) {
			head += aliasSegment
		}
		return head + "[redacted]"
	}
	for _, pre := range []string{"/api/v1/qr-requests/", "/api/v1/qr-aliases/"} {
		if strings.HasPrefix(p, pre) {
			rest := p[len(pre):]
			if j := strings.IndexByte(rest, '/'); j >= 0 {
				return pre + "[redacted]" + rest[j:]
			}
			return pre + "[redacted]"
		}
	}
	return p
}

// Request is a caller-supplied snapshot used only by the pure Validate preflight.
type Request struct {
	Network, Recipient, ClaimedTransaction string
	Amount                                 int64
	Expires                                time.Time
	Revoked                                bool
}

// Validate is a preflight only. Recheck and claim inside the same DB transaction
// as proposal creation (Service.ClaimRequest does). A caller-supplied request
// is never an authoritative record.
func (r Request) Validate(network, payer string, now time.Time) error {
	if network == "" || r.Network != network || payer == "" || r.Recipient == "" || payer == r.Recipient || r.Amount <= 0 || r.Revoked || r.ClaimedTransaction != "" || !now.Before(r.Expires) {
		return ErrInvalid
	}
	return nil
}
