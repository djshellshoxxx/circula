// Package qr validates opaque request links. It cannot claim or post a trade.
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

var ErrInvalid = errors.New("invalid QR request")

func NewToken() (string, error) {
	b := make([]byte, 16)
	if _, e := rand.Read(b); e != nil {
		return "", e
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}
func TokenHash(token string) ([32]byte, error) {
	b, e := base64.RawURLEncoding.DecodeString(token)
	if e != nil || len(b) != 16 || base64.RawURLEncoding.EncodeToString(b) != token {
		return [32]byte{}, ErrInvalid
	}
	return sha256.Sum256([]byte(token)), nil
}
func origin(raw string) (*url.URL, error) {
	u, e := url.Parse(raw)
	if e != nil || u.Scheme != "https" || u.Host == "" || u.User != nil || u.Path != "" || u.RawQuery != "" || u.ForceQuery || u.Fragment != "" {
		return nil, ErrInvalid
	}
	return u, nil
}
func Link(trustedOrigin, token string) (string, error) {
	u, e := origin(trustedOrigin)
	if e != nil {
		return "", e
	}
	if _, e = TokenHash(token); e != nil {
		return "", e
	}
	u.Path = "/qr/v1/" + token
	return u.String(), nil
}
func Parse(trustedOrigin, raw string) (string, error) {
	base, e := origin(trustedOrigin)
	if e != nil || len(raw) > 2048 || strings.Contains(raw, "#") {
		return "", ErrInvalid
	}
	u, e := url.Parse(raw)
	if e != nil || u.Scheme != base.Scheme || u.Host != base.Host || u.User != nil || u.RawQuery != "" || u.ForceQuery || u.Fragment != "" || u.RawPath != "" || !strings.HasPrefix(u.Path, "/qr/v1/") {
		return "", ErrInvalid
	}
	token := strings.TrimPrefix(u.Path, "/qr/v1/")
	_, e = TokenHash(token)
	return token, e
}

type Request struct {
	Network, Recipient, ClaimedTransaction string
	Amount                                 int64
	Expires                                time.Time
	Revoked                                bool
}

// Validate is a preflight only. Recheck and claim inside the same DB transaction
// as proposal creation. A caller-supplied request is never an authoritative record.
func (r Request) Validate(network, payer string, now time.Time) error {
	if network == "" || r.Network != network || payer == "" || r.Recipient == "" || payer == r.Recipient || r.Amount <= 0 || r.Revoked || r.ClaimedTransaction != "" || !now.Before(r.Expires) {
		return ErrInvalid
	}
	return nil
}
