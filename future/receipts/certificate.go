package receipts

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"strings"
	"time"

	jose "github.com/go-jose/go-jose/v4"
)

const MaxCertificateBytes = 65536
const certificateType = "circula-receipt+jws"
const certificateSchema = "circula-certificate/v1"

type KeyState string

const (
	KeyActive  KeyState = "active"
	KeyRetired KeyState = "retired"
	KeyRevoked KeyState = "revoked"
)

// KeyMetadata is immutable, trusted local policy. Retirement is administrative;
// issued_at is a signed assertion, not trusted external time. Compromise therefore
// requires full revocation, which rejects historic certificates as well.
type KeyMetadata struct {
	Issuer, Network, ID            string
	State                          KeyState
	NotBefore, NotAfter, RetiredAt time.Time
}
type SigningKey struct {
	Metadata KeyMetadata
	Handle   jose.OpaqueSigner
}
type SigningKeys interface {
	Current(context.Context, string) (SigningKey, error)
}
type TrustedKey struct {
	Metadata KeyMetadata
	Public   ed25519.PublicKey
}

// TrustStore must be a local public-only store, with no network lookup. It must
// preserve historic public keys and return immutable snapshots of key metadata.
type TrustStore interface {
	Lookup(context.Context, string, string, string) (TrustedKey, error)
}

var ErrUnknownKey = errors.New("unknown certificate key")
var ErrCertificateUnavailable = errors.New("certificate unavailable")

type Manifest struct {
	Schema             string   `json:"schema"`
	Issuer             string   `json:"issuer"`
	Network            string   `json:"network"`
	KeyID              string   `json:"key_id"`
	IssuedAt           string   `json:"issued_at"`
	SignatureAlgorithm string   `json:"signature_algorithm"`
	Canonicalization   string   `json:"canonicalization"`
	DigestAlgorithm    string   `json:"digest_algorithm"`
	Digest             string   `json:"digest"`
	Receipt            Document `json:"receipt"`
}
type CertificateRequest struct {
	Reference string
	Explicit  bool
}
type Issuer struct {
	Receipts       Service
	Enabled        bool
	NetworkEnabled func(context.Context, string) bool
	IssuerID       string
	Keys           SigningKeys
	Now            func() time.Time
}

func metadataValid(m KeyMetadata) bool {
	return safe(m.Issuer) && safe(m.Network) && safe(m.ID) && len(m.ID) <= 128 && len(m.Issuer) <= 128 && !m.NotBefore.IsZero() && !m.NotAfter.IsZero() && m.NotBefore.Before(m.NotAfter) && (m.State == KeyActive || m.State == KeyRetired || m.State == KeyRevoked)
}
func inWindow(m KeyMetadata, t time.Time) bool { return !t.Before(m.NotBefore) && t.Before(m.NotAfter) }

// boundSigner fixes public metadata for one signing operation and checks context
// around the opaque call. The host key boundary must itself honor deadlines;
// go-jose's opaque interface cannot interrupt a hardware operation mid-call.
type boundSigner struct {
	ctx    context.Context
	handle jose.OpaqueSigner
	public ed25519.PublicKey
}

func (b boundSigner) Public() *jose.JSONWebKey {
	return &jose.JSONWebKey{Key: b.public, Algorithm: string(jose.EdDSA), Use: "sig"}
}
func (b boundSigner) Algs() []jose.SignatureAlgorithm { return []jose.SignatureAlgorithm{jose.EdDSA} }
func (b boundSigner) SignPayload(payload []byte, alg jose.SignatureAlgorithm) ([]byte, error) {
	if e := b.ctx.Err(); e != nil {
		return nil, e
	}
	if alg != jose.EdDSA {
		return nil, ErrCertificateUnavailable
	}
	signature, e := b.handle.SignPayload(payload, alg)
	if e != nil {
		return nil, e
	}
	if e = b.ctx.Err(); e != nil {
		return nil, e
	}
	return signature, nil
}

// Issue obtains its own fresh server-authoritative receipt. No caller-supplied
// receipt, private key export, production registration, file retention or ledger
// mutation is exposed. Separate certificate consent is mandatory.
func (i Issuer) Issue(ctx context.Context, a Actor, q CertificateRequest) (string, error) {
	if ctx.Err() != nil {
		return "", ctx.Err()
	}
	if !q.Explicit || !i.Enabled || i.NetworkEnabled == nil || !actorValid(a) || !safe(i.IssuerID) || len(i.IssuerID) > 128 || i.Keys == nil || i.Now == nil || !i.NetworkEnabled(ctx, a.Network) {
		return "", ErrCertificateUnavailable
	}
	receipt, e := i.Receipts.Receipt(ctx, a, DownloadRequest{q.Reference, "json", true})
	if e != nil {
		return "", ErrCertificateUnavailable
	}
	key, e := i.Keys.Current(ctx, a.Network)
	if ctx.Err() != nil {
		return "", ctx.Err()
	}
	if e != nil || key.Handle == nil {
		return "", ErrCertificateUnavailable
	}
	now := i.Now().UTC()
	if now.Year() < 1 || now.Year() > 9999 {
		return "", ErrCertificateUnavailable
	}
	m := key.Metadata
	if !metadataValid(m) || m.State != KeyActive || !m.RetiredAt.IsZero() || m.Network != a.Network || m.Issuer != i.IssuerID || !inWindow(m, now) {
		return "", ErrCertificateUnavailable
	}
	public := key.Handle.Public()
	if public == nil {
		return "", ErrCertificateUnavailable
	}
	pub, ok := public.Key.(ed25519.PublicKey)
	if !ok || len(pub) != ed25519.PublicKeySize || (public.Algorithm != "" && public.Algorithm != string(jose.EdDSA)) || (public.KeyID != "" && public.KeyID != m.ID) {
		return "", ErrCertificateUnavailable
	}
	pub = append(ed25519.PublicKey(nil), pub...)
	supported := false
	for _, alg := range key.Handle.Algs() {
		if alg == jose.EdDSA {
			supported = true
		}
	}
	if !supported {
		return "", ErrCertificateUnavailable
	}
	completed, e := time.Parse(time.RFC3339Nano, receipt.CompletedAt)
	if e != nil || now.Before(completed) {
		return "", ErrCertificateUnavailable
	}
	if receipt.Supplement != nil {
		observed, _ := time.Parse(time.RFC3339Nano, receipt.Supplement.ObservedAt)
		if observed.After(now) {
			return "", ErrCertificateUnavailable
		}
	}
	digest, e := receipt.Digest()
	if e != nil {
		return "", ErrCertificateUnavailable
	}
	manifest := Manifest{certificateSchema, m.Issuer, m.Network, m.ID, now.Format(time.RFC3339Nano), string(jose.EdDSA), "RFC8785", "SHA-256", digest, receipt}
	raw, e := json.Marshal(manifest)
	if e != nil {
		return "", ErrCertificateUnavailable
	}
	payload, e := canonicalJSON(raw, 32768)
	if e != nil {
		return "", ErrCertificateUnavailable
	}
	signer, e := jose.NewSigner(jose.SigningKey{Algorithm: jose.EdDSA, Key: boundSigner{ctx, key.Handle, pub}}, (&jose.SignerOptions{}).WithType(certificateType).WithHeader(jose.HeaderKey("kid"), m.ID))
	if e != nil {
		return "", ErrCertificateUnavailable
	}
	signed, e := signer.Sign(payload)
	if e != nil {
		if ctx.Err() != nil {
			return "", ctx.Err()
		}
		return "", ErrCertificateUnavailable
	}
	// Detect provider rotation/misbinding during the opaque operation before any
	// certificate can escape. This uses only captured public material.
	verified, e := signed.Verify(pub)
	if e != nil || !bytes.Equal(verified, payload) {
		return "", ErrCertificateUnavailable
	}
	// An opaque signer can block while access or consent changes. Reauthorize
	// through the existing Reader and require the exact same projection before
	// releasing the signature. This is not a second issuance/download audit.
	if !i.NetworkEnabled(ctx, a.Network) {
		return "", ErrCertificateUnavailable
	}
	fresh, e := i.Receipts.receipt(ctx, a, DownloadRequest{q.Reference, "json", true}, false)
	if e != nil {
		if ctx.Err() != nil {
			return "", ctx.Err()
		}
		return "", ErrCertificateUnavailable
	}
	freshDigest, e := fresh.Digest()
	if e != nil || freshDigest != digest {
		return "", ErrCertificateUnavailable
	}
	latest, e := i.Keys.Current(ctx, a.Network)
	if ctx.Err() != nil {
		return "", ctx.Err()
	}
	if e != nil || !metadataValid(latest.Metadata) || latest.Metadata.State != KeyActive || !latest.Metadata.RetiredAt.IsZero() || latest.Metadata.ID != m.ID || latest.Metadata.Issuer != m.Issuer || latest.Metadata.Network != m.Network || !latest.Metadata.NotBefore.Equal(m.NotBefore) || !latest.Metadata.NotAfter.Equal(m.NotAfter) {
		return "", ErrCertificateUnavailable
	}
	after := i.Now().UTC()
	if after.Before(now) || !inWindow(latest.Metadata, after) || !inWindow(latest.Metadata, now) {
		return "", ErrCertificateUnavailable
	}
	compact, e := signed.CompactSerialize()
	if e != nil || len(compact) > MaxCertificateBytes {
		return "", ErrCertificateUnavailable
	}
	if ctx.Err() != nil {
		return "", ctx.Err()
	}
	if !i.NetworkEnabled(ctx, a.Network) || !i.Receipts.NetworkEnabled(ctx, a.Network) {
		return "", ErrCertificateUnavailable
	}
	if ctx.Err() != nil {
		return "", ctx.Err()
	}
	return compact, nil
}

type Verification struct {
	Reason   string    `json:"reason"`
	Manifest *Manifest `json:"manifest,omitempty"`
}

func invalid(reason string) Verification { return Verification{Reason: reason} }
func strictBase64(segment string, limit int) ([]byte, error) {
	if segment == "" || len(segment) > (limit*4+2)/3 {
		return nil, errors.New("invalid segment")
	}
	raw, e := base64.RawURLEncoding.Strict().DecodeString(segment)
	if e != nil || len(raw) > limit || base64.RawURLEncoding.EncodeToString(raw) != segment {
		return nil, errors.New("noncanonical base64")
	}
	return raw, nil
}

// VerifyCertificate is offline. No unverified manifest or receipt is returned.
// IssuedAt is an issuer assertion, not independently trusted time; retirement
// preserves historic assertions, while full revocation rejects them all.
func VerifyCertificate(ctx context.Context, compact, expectedIssuer, expectedNetwork string, trust TrustStore, now time.Time) Verification {
	if ctx.Err() != nil {
		return invalid("cancelled")
	}
	if trust == nil || !safe(expectedIssuer) || !safe(expectedNetwork) || len(expectedIssuer) > 128 || now.IsZero() {
		return invalid("invalid_configuration")
	}
	if len(compact) > MaxCertificateBytes {
		return invalid("malformed")
	}
	parts := strings.Split(compact, ".")
	if len(parts) != 3 {
		return invalid("malformed")
	}
	header, e := strictBase64(parts[0], 2048)
	if e != nil {
		return invalid("malformed")
	}
	payload, e := strictBase64(parts[1], 32768)
	if e != nil {
		return invalid("malformed")
	}
	signature, e := strictBase64(parts[2], ed25519.SignatureSize)
	if e != nil || len(signature) != ed25519.SignatureSize {
		return invalid("malformed")
	}
	canonHeader, e := canonicalJSON(header, 2048)
	if e != nil {
		return invalid("malformed")
	}
	var headers map[string]json.RawMessage
	if e = json.Unmarshal(canonHeader, &headers); e != nil || len(headers) != 3 {
		return invalid("unsupported")
	}
	for k := range headers {
		if k != "alg" && k != "typ" && k != "kid" {
			return invalid("unsupported")
		}
	}
	var algorithm, typ, kid string
	if json.Unmarshal(headers["alg"], &algorithm) != nil || json.Unmarshal(headers["typ"], &typ) != nil || json.Unmarshal(headers["kid"], &kid) != nil {
		return invalid("malformed")
	}
	if algorithm != string(jose.EdDSA) || typ != certificateType {
		return invalid("unsupported")
	}
	if !safe(kid) || len(kid) > 128 {
		return invalid("malformed")
	}
	canonical, e := canonicalJSON(payload, 32768)
	if e != nil {
		return invalid("malformed")
	}
	if !bytes.Equal(canonical, payload) {
		return invalid("malformed")
	}
	var manifest Manifest
	decoder := json.NewDecoder(bytes.NewReader(payload))
	decoder.DisallowUnknownFields()
	if e = decoder.Decode(&manifest); e != nil {
		return invalid("malformed")
	}
	if manifest.Schema != certificateSchema || manifest.SignatureAlgorithm != string(jose.EdDSA) || manifest.Canonicalization != "RFC8785" || manifest.DigestAlgorithm != "SHA-256" {
		return invalid("unsupported")
	}
	if manifest.Issuer != expectedIssuer || manifest.Network != expectedNetwork || manifest.KeyID != kid || manifest.Receipt.Network != expectedNetwork {
		return invalid("invalid_scope")
	}
	if !safe(manifest.Issuer) || !safe(manifest.Network) || !safe(manifest.KeyID) || len(manifest.Issuer) > 128 || len(manifest.KeyID) > 128 {
		return invalid("malformed")
	}
	issued, e := time.Parse(time.RFC3339Nano, manifest.IssuedAt)
	if e != nil || len(manifest.IssuedAt) > 40 || issued.UTC().Format(time.RFC3339Nano) != manifest.IssuedAt || issued.After(now) {
		return invalid("invalid_time")
	}
	if e = validateDocument(manifest.Receipt); e != nil {
		return invalid("malformed")
	}
	completed, _ := time.Parse(time.RFC3339Nano, manifest.Receipt.CompletedAt)
	if issued.Before(completed) {
		return invalid("invalid_time")
	}
	if manifest.Receipt.Supplement != nil {
		observed, _ := time.Parse(time.RFC3339Nano, manifest.Receipt.Supplement.ObservedAt)
		if observed.After(issued) {
			return invalid("invalid_time")
		}
	}
	reconstructed, _ := json.Marshal(manifest)
	recanonical, e := canonicalJSON(reconstructed, 32768)
	if e != nil || !bytes.Equal(recanonical, payload) {
		return invalid("malformed")
	}
	digest, e := manifest.Receipt.Digest()
	if e != nil || len(manifest.Digest) != 64 {
		return invalid("malformed")
	}
	decodedDigest, e := hex.DecodeString(manifest.Digest)
	if e != nil || len(decodedDigest) != sha256.Size || hex.EncodeToString(decodedDigest) != manifest.Digest {
		return invalid("malformed")
	}
	if ctx.Err() != nil {
		return invalid("cancelled")
	}
	trusted, e := trust.Lookup(ctx, expectedIssuer, expectedNetwork, kid)
	if ctx.Err() != nil {
		return invalid("cancelled")
	}
	if errors.Is(e, ErrUnknownKey) {
		return invalid("unknown_key")
	}
	if e != nil {
		return invalid("key_unavailable")
	}
	if ctx.Err() != nil {
		return invalid("cancelled")
	}
	meta := trusted.Metadata
	public := append(ed25519.PublicKey(nil), trusted.Public...)
	if !metadataValid(meta) || meta.Issuer != expectedIssuer || meta.Network != expectedNetwork || meta.ID != kid || len(public) != ed25519.PublicKeySize {
		return invalid("invalid_key")
	}
	if meta.State == KeyRevoked {
		return invalid("revoked_key")
	}
	if !inWindow(meta, issued) {
		return invalid("invalid_key_window")
	}
	if meta.State == KeyRetired && (meta.RetiredAt.IsZero() || !issued.Before(meta.RetiredAt)) {
		return invalid("retired_key")
	}
	if meta.State == KeyActive && !meta.RetiredAt.IsZero() {
		return invalid("invalid_key")
	}
	signed, e := jose.ParseSignedCompact(compact, []jose.SignatureAlgorithm{jose.EdDSA})
	if e != nil {
		return invalid("malformed")
	}
	verified, e := signed.Verify(public)
	if e != nil || !bytes.Equal(verified, payload) {
		return invalid("invalid_signature")
	}
	if digest != manifest.Digest {
		return invalid("invalid_digest")
	}
	if ctx.Err() != nil {
		return invalid("cancelled")
	}
	reason := "valid"
	if meta.State == KeyRetired {
		reason = "valid_retired_key"
	}
	return Verification{reason, &manifest}
}

// VerifyFile bounds offline input before parsing. A successful signature checks
// only the historical snapshot; it makes no online current-status request.
func VerifyFile(ctx context.Context, r io.Reader, issuer, network string, trust TrustStore, now time.Time) Verification {
	if ctx.Err() != nil {
		return invalid("cancelled")
	}
	if r == nil {
		return invalid("malformed")
	}
	b, e := io.ReadAll(io.LimitReader(r, MaxCertificateBytes+1))
	if e != nil || len(b) > MaxCertificateBytes {
		return invalid("malformed")
	}
	return VerifyCertificate(ctx, string(b), issuer, network, trust, now)
}
