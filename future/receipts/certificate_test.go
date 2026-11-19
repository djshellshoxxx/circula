package receipts

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	jose "github.com/go-jose/go-jose/v4"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

// Synthetic ephemeral test handles generate private material only in RAM.
type testHandle struct {
	private   ed25519.PrivateKey
	before    func()
	fail      bool
	badPublic bool
}

func (h *testHandle) Public() *jose.JSONWebKey {
	if h.badPublic {
		return &jose.JSONWebKey{Key: []byte("not Ed25519")}
	}
	return &jose.JSONWebKey{Key: h.private.Public().(ed25519.PublicKey)}
}
func (h *testHandle) Algs() []jose.SignatureAlgorithm { return []jose.SignatureAlgorithm{jose.EdDSA} }
func (h *testHandle) SignPayload(payload []byte, alg jose.SignatureAlgorithm) ([]byte, error) {
	if h.before != nil {
		h.before()
	}
	if h.fail {
		return nil, errors.New("synthetic failure")
	}
	return ed25519.Sign(h.private, payload), nil
}

type signingFixture struct {
	key   SigningKey
	calls atomic.Int64
}

func (k *signingFixture) Current(context.Context, string) (SigningKey, error) {
	k.calls.Add(1)
	return k.key, nil
}

type trustFixture struct {
	keys   map[string]TrustedKey
	calls  atomic.Int64
	before func()
}

func (s *trustFixture) Lookup(ctx context.Context, issuer, network, id string) (TrustedKey, error) {
	s.calls.Add(1)
	if s.before != nil {
		s.before()
	}
	key, ok := s.keys[id]
	if !ok {
		return TrustedKey{}, ErrUnknownKey
	}
	return key, nil
}
func certificateFixture(t testing.TB) (Issuer, *signingFixture, *trustFixture, *testHandle) {
	t.Helper()
	pub, private, e := ed25519.GenerateKey(rand.Reader)
	if e != nil {
		t.Fatal(e)
	}
	_, receipts := fixture()
	now := time.Unix(200, 0)
	metadata := KeyMetadata{"deployment", "n", "key1", KeyActive, time.Unix(1, 0), time.Unix(1000, 0), time.Time{}}
	handle := &testHandle{private: private}
	keys := &signingFixture{key: SigningKey{metadata, handle}}
	trust := &trustFixture{keys: map[string]TrustedKey{"key1": {metadata, pub}}}
	issuer := Issuer{Receipts: receipts, Enabled: true, NetworkEnabled: func(context.Context, string) bool { return true }, IssuerID: "deployment", Keys: keys, Now: func() time.Time { return now }}
	return issuer, keys, trust, handle
}
func issueFixture(t testing.TB, issuer Issuer) string {
	t.Helper()
	token, e := issuer.Issue(context.Background(), Actor{"n", "a"}, CertificateRequest{"tx", true})
	if e != nil {
		t.Fatal(e)
	}
	return token
}
func verifyFixture(token string, trust TrustStore) Verification {
	return VerifyCertificate(context.Background(), token, "deployment", "n", trust, time.Unix(300, 0))
}
func TestCertificateValidDeterministicAndIndependent(t *testing.T) {
	issuer, _, trust, handle := certificateFixture(t)
	token := issueFixture(t, issuer)
	again := issueFixture(t, issuer)
	if token != again {
		t.Fatal("nondeterministic certificate")
	}
	result := verifyFixture(token, trust)
	if result.Reason != "valid" || result.Manifest == nil || result.Manifest.Receipt.Amount != 123 {
		t.Fatal(result)
	}
	fromFile := VerifyFile(context.Background(), strings.NewReader(token), "deployment", "n", trust, time.Unix(300, 0))
	if fromFile.Reason != "valid" {
		t.Fatal(fromFile)
	}
	handle.fail = true
	if _, e := issuer.Issue(context.Background(), Actor{"n", "a"}, CertificateRequest{"tx", true}); e == nil {
		t.Fatal("sign failure")
	}
	var out bytes.Buffer
	if e := issuer.Receipts.Download(context.Background(), Actor{"n", "a"}, DownloadRequest{"tx", "txt", true}, &out); e != nil || out.Len() == 0 {
		t.Fatal("certificate failure blocked receipt", e)
	}
}
func TestCertificateConsentAndDisabled(t *testing.T) {
	issuer, keys, _, _ := certificateFixture(t)
	for _, change := range []func(*Issuer, *CertificateRequest){func(i *Issuer, q *CertificateRequest) { q.Explicit = false }, func(i *Issuer, q *CertificateRequest) { i.Enabled = false }, func(i *Issuer, q *CertificateRequest) { i.NetworkEnabled = nil }, func(i *Issuer, q *CertificateRequest) {
		i.NetworkEnabled = func(context.Context, string) bool { return false }
	}} {
		copy := issuer
		q := CertificateRequest{"tx", true}
		change(&copy, &q)
		if _, e := copy.Issue(context.Background(), Actor{"n", "a"}, q); !errors.Is(e, ErrCertificateUnavailable) {
			t.Fatal(e)
		}
	}
	if keys.calls.Load() != 0 {
		t.Fatal("key provider called absent consent")
	}
}
func TestKeyLifecycleAndRotation(t *testing.T) {
	issuer, keys, trust, _ := certificateFixture(t)
	token := issueFixture(t, issuer)
	key := trust.keys["key1"]
	key.Metadata.State = KeyRetired
	key.Metadata.RetiredAt = time.Unix(250, 0)
	trust.keys["key1"] = key
	if result := verifyFixture(token, trust); result.Reason != "valid_retired_key" {
		t.Fatal(result)
	}
	key.Metadata.RetiredAt = time.Unix(200, 0)
	trust.keys["key1"] = key
	if result := verifyFixture(token, trust); result.Reason != "retired_key" {
		t.Fatal(result)
	}
	key.Metadata.State = KeyRevoked
	trust.keys["key1"] = key
	if result := verifyFixture(token, trust); result.Reason != "revoked_key" {
		t.Fatal(result)
	}
	key.Metadata.State = KeyActive
	key.Metadata.RetiredAt = time.Time{}
	trust.keys["key1"] = key
	pub, private, e := ed25519.GenerateKey(rand.Reader)
	if e != nil {
		t.Fatal(e)
	}
	keys.key.Metadata.ID = "key2"
	keys.key.Handle = &testHandle{private: private}
	trust.keys["key2"] = TrustedKey{keys.key.Metadata, pub}
	second := issueFixture(t, issuer)
	if verifyFixture(second, trust).Reason != "valid" || verifyFixture(token, trust).Reason != "valid" {
		t.Fatal("rotation lost historical trust")
	}
	keys.key.Metadata.State = KeyRetired
	if _, e = issuer.Issue(context.Background(), Actor{"n", "a"}, CertificateRequest{"tx", true}); e == nil {
		t.Fatal("retired issued")
	}
	keys.key.Metadata.State = KeyRevoked
	if _, e = issuer.Issue(context.Background(), Actor{"n", "a"}, CertificateRequest{"tx", true}); e == nil {
		t.Fatal("revoked issued")
	}
}
func TestCertificateKeyWindowsAndCancellation(t *testing.T) {
	issuer, keys, trust, handle := certificateFixture(t)
	token := issueFixture(t, issuer)
	if VerifyCertificate(context.Background(), token, "deployment", "n", trust, time.Unix(199, 0)).Reason != "invalid_time" {
		t.Fatal("future issue accepted")
	}
	key := trust.keys["key1"]
	key.Metadata.NotBefore = time.Unix(201, 0)
	trust.keys["key1"] = key
	if verifyFixture(token, trust).Reason != "invalid_key_window" {
		t.Fatal("before key activation accepted")
	}
	keys.key.Metadata.NotAfter = time.Unix(200, 0)
	if _, e := issuer.Issue(context.Background(), Actor{"n", "a"}, CertificateRequest{"tx", true}); e == nil {
		t.Fatal("expiry boundary")
	}
	keys.key.Metadata.NotAfter = time.Unix(1000, 0)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, e := issuer.Issue(ctx, Actor{"n", "a"}, CertificateRequest{"tx", true}); !errors.Is(e, context.Canceled) {
		t.Fatal(e)
	}
	if VerifyCertificate(ctx, token, "deployment", "n", trust, time.Unix(300, 0)).Reason != "cancelled" {
		t.Fatal("cancelled verifier")
	}
	ctx, cancel = context.WithCancel(context.Background())
	handle.before = cancel
	if _, e := issuer.Issue(ctx, Actor{"n", "a"}, CertificateRequest{"tx", true}); !errors.Is(e, context.Canceled) {
		t.Fatal("sign cancellation", e)
	}
	ctx, cancel = context.WithCancel(context.Background())
	trust.before = cancel
	if VerifyCertificate(ctx, token, "deployment", "n", trust, time.Unix(300, 0)).Reason != "cancelled" {
		t.Fatal("cancelled lookup")
	}
}

// resign produces adversarial but correctly signed profiles to separate schema
// checks from cryptographic authentication. Only ephemeral keys are used.
func resign(t testing.TB, handle *testHandle, payload []byte, headers map[jose.HeaderKey]any) string {
	t.Helper()
	options := &jose.SignerOptions{ExtraHeaders: headers}
	signer, e := jose.NewSigner(jose.SigningKey{Algorithm: jose.EdDSA, Key: handle.private}, options)
	if e != nil {
		t.Fatal(e)
	}
	signed, e := signer.Sign(payload)
	if e != nil {
		t.Fatal(e)
	}
	out, e := signed.CompactSerialize()
	if e != nil {
		t.Fatal(e)
	}
	return out
}
func manifestBytes(t testing.TB, token string) []byte {
	t.Helper()
	parts := strings.Split(token, ".")
	raw, e := base64.RawURLEncoding.DecodeString(parts[1])
	if e != nil {
		t.Fatal(e)
	}
	return raw
}
func TestCertificateAdversarialInputs(t *testing.T) {
	issuer, _, trust, handle := certificateFixture(t)
	token := issueFixture(t, issuer)
	parts := strings.Split(token, ".")
	payload := manifestBytes(t, token)
	standard := map[jose.HeaderKey]any{"typ": certificateType, "kid": "key1"}
	var m Manifest
	if e := json.Unmarshal(payload, &m); e != nil {
		t.Fatal(e)
	}
	m.Digest = strings.Repeat("0", 64)
	wrongDigest, _ := json.Marshal(m)
	wrongDigest, _ = canonicalJSON(wrongDigest, 32768)
	m.IssuedAt = time.Unix(99, 0).Format(time.RFC3339Nano)
	early, _ := json.Marshal(m)
	early, _ = canonicalJSON(early, 32768)
	for name, test := range map[string]struct{ token, reason string }{"too big": {strings.Repeat("x", MaxCertificateBytes+1), "malformed"}, "json serialization": {`{"payload":"x"}`, "malformed"}, "padded": {parts[0] + "=." + parts[1] + "." + parts[2], "malformed"}, "missing sig": {parts[0] + "." + parts[1] + ".", "malformed"}, "wrong digest": {resign(t, handle, wrongDigest, standard), "invalid_digest"}, "early": {resign(t, handle, early, standard), "invalid_time"}, "remote key": {resign(t, handle, payload, map[jose.HeaderKey]any{"typ": certificateType, "kid": "key1", "jku": "https://not-used.invalid/key"}), "unsupported"}, "noncanonical payload": {resign(t, handle, append([]byte(" "), payload...), standard), "malformed"}, "duplicate": {resign(t, handle, []byte(`{"schema":"one","schema":"two"}`), standard), "malformed"}, "alg confusion": {base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"HS256","kid":"key1","typ":"circula-receipt+jws"}`)) + "." + parts[1] + "." + parts[2], "unsupported"}, "depth": {resign(t, handle, []byte(strings.Repeat("[", 17)+"0"+strings.Repeat("]", 17)), standard), "malformed"}} {
		t.Run(name, func(t *testing.T) {
			r := verifyFixture(test.token, trust)
			if r.Reason != test.reason || r.Manifest != nil {
				t.Fatal(r, test.reason)
			}
		})
	}
	if r := VerifyCertificate(context.Background(), token, "wrong", "n", trust, time.Unix(300, 0)); r.Reason != "invalid_scope" || r.Manifest != nil {
		t.Fatal(r)
	}
	unknown := &trustFixture{keys: map[string]TrustedKey{}}
	if verifyFixture(token, unknown).Reason != "unknown_key" {
		t.Fatal("unknown key")
	}
	pub, _, _ := ed25519.GenerateKey(rand.Reader)
	key := trust.keys["key1"]
	key.Public = pub
	trust.keys["key1"] = key
	if verifyFixture(token, trust).Reason != "invalid_signature" {
		t.Fatal("wrong key")
	}
}
func TestCertificateExactKeyTypeAndHandleBinding(t *testing.T) {
	issuer, keys, _, handle := certificateFixture(t)
	handle.badPublic = true
	if _, e := issuer.Issue(context.Background(), Actor{"n", "a"}, CertificateRequest{"tx", true}); e == nil {
		t.Fatal("wrong public type")
	}
	handle.badPublic = false
	_, other, _ := ed25519.GenerateKey(rand.Reader)
	handle.before = func() { handle.private = other }
	if _, e := issuer.Issue(context.Background(), Actor{"n", "a"}, CertificateRequest{"tx", true}); e == nil {
		t.Fatal("rotating handle escaped")
	}
	keys.key.Metadata.Network = "other"
	if _, e := issuer.Issue(context.Background(), Actor{"n", "a"}, CertificateRequest{"tx", true}); e == nil {
		t.Fatal("network key mismatch")
	}
}
func FuzzVerifyCertificate(f *testing.F) {
	issuer, _, trust, _ := certificateFixture(f)
	f.Add(issueFixture(f, issuer))
	f.Add("bad.token.here")
	f.Fuzz(func(t *testing.T, token string) {
		r := verifyFixture(token, trust)
		if r.Manifest != nil && r.Reason != "valid" && r.Reason != "valid_retired_key" {
			t.Fatal("unverified receipt returned")
		}
	})
}

func TestStandardJWSIndependentVerification(t *testing.T) {
	issuer, _, trust, _ := certificateFixture(t)
	token := issueFixture(t, issuer)
	parts := strings.Split(token, ".")
	signature, e := base64.RawURLEncoding.DecodeString(parts[2])
	if e != nil {
		t.Fatal(e)
	}
	public := trust.keys["key1"].Public
	if !ed25519.Verify(public, []byte(parts[0]+"."+parts[1]), signature) {
		t.Fatal("standard Ed25519 JWS verification failed")
	}
}
func TestKeyRevokedDuringSigningAndLateExpiry(t *testing.T) {
	issuer, keys, _, handle := certificateFixture(t)
	handle.before = func() { keys.key.Metadata.State = KeyRevoked }
	if _, e := issuer.Issue(context.Background(), Actor{"n", "a"}, CertificateRequest{"tx", true}); e == nil {
		t.Fatal("key revoked during operation")
	}
	keys.key.Metadata.State = KeyActive
	handle.before = nil
	calls := 0
	issuer.Now = func() time.Time {
		calls++
		if calls == 1 {
			return time.Unix(200, 0)
		}
		return time.Unix(1000, 0)
	}
	if _, e := issuer.Issue(context.Background(), Actor{"n", "a"}, CertificateRequest{"tx", true}); e == nil {
		t.Fatal("certificate returned after expiry")
	}
}
func TestCertificateHTTPAndCLI(t *testing.T) {
	issuer, _, trust, _ := certificateFixture(t)
	h := CertificateHandler{issuer, hostFixture{csrf: true}}
	get := httptest.NewRecorder()
	h.ServeHTTP(get, httptest.NewRequest("GET", "/certificate?reference=tx", nil))
	if get.Code != 200 || strings.Contains(get.Body.String(), "<script") || !strings.Contains(get.Body.String(), "No JavaScript") {
		t.Fatal(get.Code, get.Body.String())
	}
	post := httptest.NewRequest("POST", "/certificate", strings.NewReader(`{"reference":"tx","format":"jws","explicit":true}`))
	post.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, post)
	if w.Code != 200 || w.Header().Get("Content-Type") != "application/jose" || verifyFixture(w.Body.String(), trust).Reason != "valid" {
		t.Fatal(w.Code, w.Body.String())
	}
	var out, help bytes.Buffer
	if e := RunCertificateCLI(context.Background(), issuer, Actor{"n", "a"}, []string{"--transaction", "tx"}, &out, &help); e == nil {
		t.Fatal("certificate CLI implicit consent")
	}
	if e := RunCertificateCLI(context.Background(), issuer, Actor{"n", "a"}, []string{"--transaction", "tx", "--request-certificate"}, &out, &help); e != nil || verifyFixture(out.String(), trust).Reason != "valid" {
		t.Fatal(e)
	}
	path := filepath.Join(t.TempDir(), "certificate.jws")
	if e := os.WriteFile(path, out.Bytes(), 0600); e != nil {
		t.Fatal(e)
	}
	out.Reset()
	if e := RunVerifierCLI(context.Background(), []string{"--file", path}, "deployment", "n", trust, time.Unix(300, 0), &out, &help); e != nil || !strings.Contains(out.String(), "valid\n") || strings.Contains(out.String(), "amount_minor") {
		t.Fatal(e, out.String())
	}
}
func TestTrustSetLoader(t *testing.T) {
	issuer, _, trust, _ := certificateFixture(t)
	token := issueFixture(t, issuer)
	key := trust.keys["key1"]
	record := publicKeyRecord{key.Metadata.Issuer, key.Metadata.Network, key.Metadata.ID, key.Metadata.State, key.Metadata.NotBefore.UTC().Format(time.RFC3339Nano), key.Metadata.NotAfter.UTC().Format(time.RFC3339Nano), "", base64.RawURLEncoding.EncodeToString(key.Public)}
	set := keySet{"circula-trusted-keys/v1", []publicKeyRecord{record}}
	raw, e := json.Marshal(set)
	if e != nil {
		t.Fatal(e)
	}
	local, e := LoadTrustStore(bytes.NewReader(raw))
	if e != nil || verifyFixture(token, local).Reason != "valid" {
		t.Fatal(e)
	}
	first, e := local.Lookup(context.Background(), "deployment", "n", "key1")
	if e != nil {
		t.Fatal(e)
	}
	first.Public[0] ^= 255
	if verifyFixture(token, local).Reason != "valid" {
		t.Fatal("public key slice escaped")
	}
	if _, e := local.Lookup(context.Background(), "deployment", "other", "key1"); !errors.Is(e, ErrUnknownKey) {
		t.Fatal(e)
	}
	set.Keys = append(set.Keys, record)
	duplicate, _ := json.Marshal(set)
	for _, input := range [][]byte{duplicate, []byte(`{"schema":"circula-trusted-keys/v1","keys":[]}`), []byte(strings.Repeat(" ", 65537)), []byte(`{"schema":"one","schema":"two"}`)} {
		if _, e := LoadTrustStore(bytes.NewReader(input)); e == nil {
			t.Fatal("invalid trust set accepted")
		}
	}
}
func TestRejectNoncanonicalBase64AndExtraEntry(t *testing.T) {
	issuer, _, trust, handle := certificateFixture(t)
	token := issueFixture(t, issuer)
	parts := strings.Split(token, ".")
	alphabet := "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789-_"
	last := strings.IndexByte(alphabet, parts[2][len(parts[2])-1])
	if last < 0 {
		t.Fatal("base64 alphabet")
	}
	mutated := parts[2][:len(parts[2])-1] + string(alphabet[last|1])
	if verifyFixture(parts[0]+"."+parts[1]+"."+mutated, trust).Reason != "malformed" {
		t.Fatal("noncanonical low base64 bits accepted")
	}
	payload := manifestBytes(t, token)
	payload = bytes.Replace(payload, []byte(`"ledger_entries":["entry1","entry2"]`), []byte(`"ledger_entries":["entry1","entry2","hidden"]`), 1)
	payload, _ = canonicalJSON(payload, 32768)
	bad := resign(t, handle, payload, map[jose.HeaderKey]any{"typ": certificateType, "kid": "key1"})
	if verifyFixture(bad, trust).Reason != "malformed" {
		t.Fatal("overlong fixed array")
	}
}

func TestCertificateHealthNoPrivateMetadata(t *testing.T) {
	issuer, keys, _, _ := certificateFixture(t)
	if h := issuer.Health(context.Background(), "n"); !h.Ready || h.Reason != "ready" {
		t.Fatal(h)
	}
	keys.key.Metadata.State = KeyRevoked
	if h := issuer.Health(context.Background(), "n"); h.Ready || h.Reason != "key_ineligible" {
		t.Fatal(h)
	}
	issuer.Enabled = false
	if h := issuer.Health(context.Background(), "n"); h.Enabled || h.Ready || h.Reason != "disabled" {
		t.Fatal(h)
	}
}
