package notifications

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"strings"
	"time"
)

// Secret holds a credential (SMTP password, VAPID private key). Every textual
// rendering is redacted so a logged struct cannot leak it; call Reveal at the
// single point of use.
type Secret struct{ v string }

// NewSecret wraps v.
func NewSecret(v string) Secret { return Secret{v: v} }

// Reveal returns the plaintext. Do not log it.
func (s Secret) Reveal() string { return s.v }

// Set reports whether a value is present.
func (s Secret) Set() bool { return s.v != "" }

func (s Secret) String() string               { return redacted(s) }
func (s Secret) GoString() string             { return redacted(s) }
func (s Secret) MarshalText() ([]byte, error) { return []byte(redacted(s)), nil }
func (s Secret) MarshalJSON() ([]byte, error) { return json.Marshal(redacted(s)) }
func (s Secret) Format(f fmt.State, _ rune)   { _, _ = io.WriteString(f, redacted(s)) }
func (s Secret) LogValue() slog.Value         { return slog.StringValue(redacted(s)) }

func redacted(s Secret) string {
	if s.v == "" {
		return "[unset]"
	}
	return "[redacted]"
}

// Keyring seals and opens target blobs with AES-256-GCM. Blobs name the key
// that sealed them so keys can rotate: the active key seals, any retained key
// opens. Additional authenticated data binds a blob to its network, member and
// row, so a blob copied into another tenant's row fails to open.
type Keyring struct {
	active string
	keys   map[string][]byte
}

// NewKeyring requires 32-byte keys and an active id that exists.
func NewKeyring(active string, keys map[string][]byte) (*Keyring, error) {
	if len(keys) == 0 || keys[active] == nil {
		return nil, errors.New("notifications: keyring needs an active key")
	}
	k := &Keyring{active: active, keys: map[string][]byte{}}
	for id, key := range keys {
		if id == "" || len(id) > 32 || len(key) != 32 {
			return nil, errors.New("notifications: keyring keys must be 32 bytes with short ids")
		}
		k.keys[id] = append([]byte(nil), key...)
	}
	return k, nil
}

// ActiveID is the id new blobs are sealed with.
func (k *Keyring) ActiveID() string { return k.active }

const sealVersion = 1

// Seal encrypts plaintext under the active key with the given AAD.
func (k *Keyring) Seal(plaintext, aad []byte) ([]byte, error) {
	gcm, err := newGCM(k.keys[k.active])
	if err != nil {
		return nil, err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}
	out := []byte{sealVersion, byte(len(k.active))}
	out = append(out, k.active...)
	out = append(out, nonce...)
	return gcm.Seal(out, nonce, plaintext, aad), nil
}

// ErrSealed is the single failure for a blob that cannot be opened (tampered,
// wrong AAD, unknown key). It deliberately carries no detail.
var ErrSealed = errors.New("notifications: sealed value cannot be opened")

// Open decrypts a blob produced by Seal. It also returns the id of the key.
func (k *Keyring) Open(blob, aad []byte) (plain []byte, keyID string, err error) {
	if len(blob) < 2 || blob[0] != sealVersion {
		return nil, "", ErrSealed
	}
	n := int(blob[1])
	if len(blob) < 2+n+12+16 {
		return nil, "", ErrSealed
	}
	id := string(blob[2 : 2+n])
	key, ok := k.keys[id]
	if !ok {
		return nil, "", ErrSealed
	}
	gcm, err := newGCM(key)
	if err != nil {
		return nil, "", ErrSealed
	}
	nonce := blob[2+n : 2+n+12]
	p, err := gcm.Open(nil, nonce, blob[2+n+12:], aad)
	if err != nil {
		return nil, "", ErrSealed
	}
	return p, id, nil
}

func newGCM(key []byte) (cipher.AEAD, error) {
	b, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	return cipher.NewGCM(b)
}

// Secrets is everything credential-like the module needs. None of it is
// stored in the database.
type Secrets struct {
	Keyring  *Keyring
	IndexKey []byte // keyed hash of endpoints/addresses (>= 32 bytes)
	TokenKey []byte // signs unsubscribe/confirm tokens (>= 32 bytes)

	VAPIDPublic  string
	VAPIDPrivate Secret
	VAPIDSubject string // mailto: or https: contact for push services

	SMTPPassword Secret
}

// Validate checks the pieces every deployment needs. SMTP/VAPID parts are
// optional: no provider is required for core operation.
func (s Secrets) Validate() error {
	switch {
	case s.Keyring == nil:
		return errors.New("notifications: encryption keyring missing")
	case len(s.IndexKey) < 32:
		return errors.New("notifications: index key must be at least 32 bytes")
	case len(s.TokenKey) < 32:
		return errors.New("notifications: token key must be at least 32 bytes")
	case bytes.Equal(s.IndexKey, s.TokenKey):
		return errors.New("notifications: index and token keys must differ")
	}
	return nil
}

// Secret names read by LoadSecrets. Each is taken from NAME_FILE (a path to a
// regular file that must not be group/world accessible) or, if that is unset,
// from NAME itself. File form is the recommended one.
const (
	EnvEncryptionKeys = "NOTIFY_ENCRYPTION_KEYS" // lines "id:base64-32-bytes"; first line is active
	EnvIndexKey       = "NOTIFY_INDEX_KEY"       // base64, >= 32 bytes
	EnvTokenKey       = "NOTIFY_TOKEN_KEY"       // base64, >= 32 bytes
	EnvVAPIDPrivate   = "NOTIFY_VAPID_PRIVATE_KEY"
	EnvVAPIDPublic    = "NOTIFY_VAPID_PUBLIC_KEY"
	EnvVAPIDSubject   = "NOTIFY_VAPID_SUBJECT"
	EnvSMTPPassword   = "NOTIFY_SMTP_PASSWORD"
)

const maxSecretFile = 64 << 10

// LoadSecrets resolves secrets through getenv (usually os.Getenv). Errors name
// the variable, never the value.
func LoadSecrets(getenv func(string) string) (Secrets, error) {
	var s Secrets
	read := func(name string, required bool) (string, error) {
		if path := getenv(name + "_FILE"); path != "" {
			return readSecretFile(name, path)
		}
		v := strings.TrimSpace(getenv(name))
		if v == "" && required {
			return "", fmt.Errorf("notifications: %s (or %s_FILE) is required", name, name)
		}
		return v, nil
	}
	raw, err := read(EnvEncryptionKeys, true)
	if err != nil {
		return s, err
	}
	if s.Keyring, err = parseKeyring(raw); err != nil {
		return s, fmt.Errorf("%s: %w", EnvEncryptionKeys, err)
	}
	for _, it := range []struct {
		name string
		dst  *[]byte
	}{{EnvIndexKey, &s.IndexKey}, {EnvTokenKey, &s.TokenKey}} {
		v, err := read(it.name, true)
		if err != nil {
			return s, err
		}
		b, err := decodeB64(v)
		if err != nil || len(b) < 32 {
			return s, fmt.Errorf("notifications: %s must be base64 of at least 32 bytes", it.name)
		}
		*it.dst = b
	}
	pw, err := read(EnvSMTPPassword, false)
	if err != nil {
		return s, err
	}
	s.SMTPPassword = NewSecret(pw)
	vp, err := read(EnvVAPIDPrivate, false)
	if err != nil {
		return s, err
	}
	s.VAPIDPrivate = NewSecret(vp)
	if s.VAPIDPublic, err = read(EnvVAPIDPublic, false); err != nil {
		return s, err
	}
	if s.VAPIDSubject, err = read(EnvVAPIDSubject, false); err != nil {
		return s, err
	}
	if (s.VAPIDPrivate.Set() || s.VAPIDPublic != "") && (!s.VAPIDPrivate.Set() || s.VAPIDPublic == "" || s.VAPIDSubject == "") {
		return s, errors.New("notifications: VAPID needs private key, public key and subject together")
	}
	return s, s.Validate()
}

func readSecretFile(name, path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", fmt.Errorf("notifications: %s_FILE unreadable", name)
	}
	defer f.Close()
	fi, err := f.Stat()
	if err != nil || !fi.Mode().IsRegular() {
		return "", fmt.Errorf("notifications: %s_FILE is not a regular file", name)
	}
	if fi.Mode().Perm()&0o077 != 0 {
		return "", fmt.Errorf("notifications: %s_FILE must not be group or world accessible (mode %04o)", name, fi.Mode().Perm())
	}
	b, err := io.ReadAll(io.LimitReader(f, maxSecretFile+1))
	if err != nil || len(b) > maxSecretFile {
		return "", fmt.Errorf("notifications: %s_FILE unreadable or too large", name)
	}
	return strings.TrimSpace(string(b)), nil
}

func decodeB64(v string) ([]byte, error) {
	v = strings.TrimSpace(v)
	for _, enc := range []*base64.Encoding{base64.StdEncoding, base64.RawStdEncoding, base64.URLEncoding, base64.RawURLEncoding} {
		if b, err := enc.DecodeString(v); err == nil {
			return b, nil
		}
	}
	return nil, errors.New("bad base64")
}

func parseKeyring(raw string) (*Keyring, error) {
	keys := map[string][]byte{}
	active := ""
	for _, line := range strings.Split(raw, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		id, val, ok := strings.Cut(line, ":")
		if !ok {
			return nil, errors.New("lines must be id:base64key")
		}
		b, err := decodeB64(val)
		if err != nil || len(b) != 32 {
			return nil, errors.New("keys must be base64 of exactly 32 bytes")
		}
		if _, dup := keys[id]; dup {
			return nil, errors.New("duplicate key id")
		}
		keys[id] = b
		if active == "" {
			active = id
		}
	}
	return NewKeyring(active, keys)
}

// ---- keyed hashes and signed tokens ----

// indexHash is a keyed hash for uniqueness lookups. Unkeyed hashes of e-mail
// addresses can be brute-forced from a stolen database; this cannot.
func indexHash(key []byte, domain string, parts ...string) string {
	m := hmac.New(sha256.New, key)
	m.Write([]byte(domain))
	for _, p := range parts {
		m.Write([]byte{0})
		m.Write([]byte(p))
	}
	return hex.EncodeToString(m.Sum(nil))
}

// Token purposes.
const (
	tokUnsubscribe = "unsub"
	tokConfirm     = "confirm"
)

type tokenPayload struct {
	P string `json:"p"`
	N string `json:"n"`
	M string `json:"m"`
	C string `json:"c,omitempty"`
	T string `json:"t,omitempty"`
	E int64  `json:"e,omitempty"`
}

func signToken(key []byte, p tokenPayload) string {
	body, _ := json.Marshal(p)
	b := base64.RawURLEncoding.EncodeToString(body)
	m := hmac.New(sha256.New, key)
	m.Write([]byte("ntf/token/v1."))
	m.Write([]byte(b))
	return "v1." + b + "." + base64.RawURLEncoding.EncodeToString(m.Sum(nil))
}

// verifyToken checks the signature, purpose and (if set) expiry in constant
// time with respect to the signature.
func verifyToken(key []byte, tok, purpose string, now time.Time) (tokenPayload, error) {
	var p tokenPayload
	if len(tok) > 1024 {
		return p, ErrSealed
	}
	parts := strings.Split(tok, ".")
	if len(parts) != 3 || parts[0] != "v1" {
		return p, ErrSealed
	}
	mac, err := base64.RawURLEncoding.DecodeString(parts[2])
	if err != nil {
		return p, ErrSealed
	}
	m := hmac.New(sha256.New, key)
	m.Write([]byte("ntf/token/v1."))
	m.Write([]byte(parts[1]))
	if subtle.ConstantTimeCompare(mac, m.Sum(nil)) != 1 {
		return p, ErrSealed
	}
	body, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil || json.Unmarshal(body, &p) != nil || p.P != purpose {
		return p, ErrSealed
	}
	if p.E != 0 && now.Unix() > p.E {
		return p, ErrSealed
	}
	return p, nil
}

func newID() string {
	var b [16]byte
	if _, err := io.ReadFull(rand.Reader, b[:]); err != nil {
		panic("notifications: entropy source failed")
	}
	return hex.EncodeToString(b[:])
}
