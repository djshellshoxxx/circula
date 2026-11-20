package federation

import (
	"bytes"
	"crypto/ed25519"
	"encoding/base64"
	"encoding/json"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	jose "github.com/go-jose/go-jose/v4"
)

const ProtocolType = "circula-discovery-v2+jws"
const MaxWire = 16384

// SharedListing is an explicit public-content allowlist. It has no URL, member
// profile, exact coordinates, balance or settlement fields.
type SharedListing struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	Category    string `json:"category"`
	Area        string `json:"area"`
}
type Discovery struct {
	Version  int            `json:"version"`
	ID       string         `json:"id"`
	Issuer   string         `json:"issuer"`
	Audience string         `json:"audience"`
	Kind     string         `json:"kind"`
	Listing  string         `json:"listing"`
	Revision int64          `json:"revision"`
	Issued   int64          `json:"issued"`
	Expires  int64          `json:"expires"`
	Content  *SharedListing `json:"content"`
}

func identifier(s string) bool {
	if len(s) == 0 || len(s) > 128 {
		return false
	}
	for _, r := range s {
		if !((r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '-' || r == '_' || r == '.') {
			return false
		}
	}
	return true
}
func publicText(s string, max int) bool {
	if len(s) > max || !utf8.ValidString(s) {
		return false
	}
	for _, r := range s {
		if unicode.IsControl(r) && r != '\n' && r != '\t' {
			return false
		}
	}
	return true
}
func validShared(c SharedListing) bool {
	return strings.TrimSpace(c.Title) != "" && publicText(c.Title, 120) && publicText(c.Description, 2000) && publicText(c.Category, 80) && publicText(c.Area, 80)
}
func validDiscovery(e Discovery, local, remote string, now time.Time) bool {
	if e.Version != 2 || !identifier(e.ID) || !identifier(e.Listing) || !identifier(e.Issuer) || !identifier(e.Audience) || e.Issuer != remote || e.Audience != local || local == remote || e.Revision < 1 || e.Issued <= 0 || e.Expires <= e.Issued || e.Expires-e.Issued > 86400 || e.Issued > now.Unix()+300 || e.Expires <= now.Unix() {
		return false
	}
	switch e.Kind {
	case "listing.publish":
		return e.Content != nil && validShared(*e.Content)
	case "listing.revoke":
		return e.Content == nil
	default:
		return false
	}
}

// SignDiscovery is experimental v2; it never accepts the prototype v1 framing.
func SignDiscovery(e Discovery, kid string, key ed25519.PrivateKey) (string, error) {
	if !identifier(kid) || len(key) != ed25519.PrivateKeySize || !validDiscovery(e, e.Audience, e.Issuer, time.Unix(e.Issued, 0)) {
		return "", ErrInvalid
	}
	payload, err := json.Marshal(e)
	if err != nil {
		return "", ErrInvalid
	}
	options := (&jose.SignerOptions{}).WithType(ProtocolType).WithHeader("kid", kid)
	signer, err := jose.NewSigner(jose.SigningKey{Algorithm: jose.EdDSA, Key: key}, options)
	if err != nil {
		return "", err
	}
	signed, err := signer.Sign(payload)
	if err != nil {
		return "", err
	}
	return signed.CompactSerialize()
}

// WireKeyID only extracts a bounded untrusted selector. It confers no trust.
func WireKeyID(wire string) (string, error) {
	if len(wire) == 0 || len(wire) > MaxWire {
		return "", ErrInvalid
	}
	parts := strings.Split(wire, ".")
	if len(parts) != 3 || len(parts[0]) > 1024 {
		return "", ErrInvalid
	}
	for i, part := range parts {
		decoded, e := base64.RawURLEncoding.DecodeString(part)
		if e != nil || len(decoded) == 0 || base64.RawURLEncoding.EncodeToString(decoded) != part || i == 2 && len(decoded) != ed25519.SignatureSize {
			return "", ErrInvalid
		}
	}
	raw, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return "", ErrInvalid
	}
	var header map[string]json.RawMessage
	if json.Unmarshal(raw, &header) != nil || len(header) != 3 {
		return "", ErrInvalid
	}
	var alg, typ, kid string
	if json.Unmarshal(header["alg"], &alg) != nil || json.Unmarshal(header["typ"], &typ) != nil || json.Unmarshal(header["kid"], &kid) != nil || alg != "EdDSA" || typ != ProtocolType || !identifier(kid) {
		return "", ErrInvalid
	}
	canonical, err := json.Marshal(map[string]string{"alg": alg, "kid": kid, "typ": typ})
	if err != nil || !bytes.Equal(canonical, raw) {
		return "", ErrInvalid
	}
	return kid, nil
}
func VerifyDiscovery(wire, local, remote, kid string, key ed25519.PublicKey, now time.Time) (Discovery, error) {
	selector, err := WireKeyID(wire)
	if err != nil || selector != kid || len(key) != ed25519.PublicKeySize {
		return Discovery{}, ErrInvalid
	}
	signed, err := jose.ParseSignedCompact(wire, []jose.SignatureAlgorithm{jose.EdDSA})
	if err != nil {
		return Discovery{}, ErrInvalid
	}
	payload, err := signed.Verify(key)
	if err != nil || len(payload) > 8192 || !utf8.Valid(payload) {
		return Discovery{}, ErrInvalid
	}
	var event Discovery
	if json.Unmarshal(payload, &event) != nil {
		return Discovery{}, ErrInvalid
	}
	canonical, err := json.Marshal(event)
	if err != nil || !bytes.Equal(canonical, payload) || !validDiscovery(event, local, remote, now) {
		return Discovery{}, ErrInvalid
	}
	return event, nil
}
