package feed

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/binary"
	"encoding/json"
	"strings"
	"time"
)

const (
	snapshotVersion  = 3
	snapshotKeyLabel = "circula/future/feed-snapshot-key/v3"
	snapshotAADLabel = "circula/future/feed-snapshot/v3"
	// The fixed plaintext length hides the kind, identifier length and
	// high-water digit count of an unauthorized scan frontier. Two bytes
	// encode the canonical JSON length; all remaining bytes are zero padding.
	snapshotPlaintextSize = 1024
	snapshotHeaderSize    = 2
)

// SnapshotCursor is a bounded traversal's private state. Position is the last
// scanned source, which need not have been visible to the viewer. HighWater is
// the source store's committed sequence ceiling. IssuedAt is set once when the
// traversal begins and is preserved across every page; paging never renews it.
// This structure must only cross an untrusted boundary through SealSnapshot.
type SnapshotCursor struct {
	Position  Cursor
	HighWater int64
	IssuedAt  time.Time
}

type snapshotPayload struct {
	V   int    `json:"v"`
	K   string `json:"k"`
	I   string `json:"i"`
	T   int64  `json:"t"`
	H   int64  `json:"h"`
	Iat int64  `json:"iat"` // original traversal issue time, Unix nanoseconds
}

// SealSnapshot encrypts and authenticates a bounded traversal cursor with the
// current signing key. It requires a valid position, positive high-water mark
// and explicit, unexpired IssuedAt. Callers must retain the original IssuedAt
// when advancing Position. All state is encrypted because a bounded scan may
// advance across an unauthorized source that must not be exposed to the reader.
//
// The version 3 wire format is "3.<key id>.<base64url nonce+ciphertext+tag>".
// AES-256-GCM uses a fresh random 96-bit nonce and a separately HMAC-derived key.
// Its additional authenticated data binds version, key ID, network, viewer and
// canonical filter digest. Fixed-size padding hides the lengths of the private
// fields. No position, high-water mark or issue time is clear.
func (c *CursorCodec) SealSnapshot(b Binding, snapshot SnapshotCursor) (string, error) {
	if !c.ready() {
		return "", errSeal
	}
	_, digest, err := b.canonical()
	if err != nil || !validSnapshot(snapshot, c.now(), c.ttl) {
		return "", errSeal
	}
	kid, key := c.keys.SigningKey()
	if !validKeyID(kid) || len(key) < MinKeyLen {
		return "", errSeal
	}
	aead, err := snapshotAEAD(key, kid)
	if err != nil {
		return "", errSeal
	}
	payload, err := json.Marshal(snapshotPayload{
		V: snapshotVersion, K: snapshot.Position.Kind, I: snapshot.Position.ID,
		T: snapshot.Position.Created.UnixNano(), H: snapshot.HighWater,
		Iat: snapshot.IssuedAt.UnixNano(),
	})
	if err != nil {
		return "", errSeal
	}
	plaintext, err := padSnapshotPayload(payload)
	if err != nil {
		return "", errSeal
	}
	nonce := make([]byte, aead.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return "", errSeal
	}
	wire := aead.Seal(nonce, nonce, plaintext, snapshotAAD(kid, b, digest))
	token := "3." + kid + "." + b64.EncodeToString(wire)
	if len(token) > maxTokenLen {
		return "", errSeal
	}
	return token, nil
}

// OpenSnapshot verifies and decrypts a version 3 cursor for b. Every refusal
// returns exactly ErrInvalidCursor and a zero SnapshotCursor. Legacy version 2
// cursors intentionally do not open here: they lack a fixed snapshot ceiling
// and traversal lifetime, so a bounded traversal must restart from its first
// page. Existing version 2 callers continue to use Open and Seal unchanged.
func (c *CursorCodec) OpenSnapshot(b Binding, token string) (SnapshotCursor, error) {
	if !c.ready() || len(token) > maxTokenLen || !isTokenText(token) {
		return SnapshotCursor{}, ErrInvalidCursor
	}
	version, rest, _ := strings.Cut(token, ".")
	kid, encoded, ok := strings.Cut(rest, ".")
	if version != "3" || !ok || !validKeyID(kid) || strings.Contains(encoded, ".") {
		return SnapshotCursor{}, ErrInvalidCursor
	}
	_, digest, err := b.canonical()
	if err != nil {
		return SnapshotCursor{}, ErrInvalidCursor
	}
	wire, err := b64.DecodeString(encoded)
	if err != nil {
		return SnapshotCursor{}, ErrInvalidCursor
	}
	key, known := c.keys.VerificationKey(kid)
	if !known || len(key) < MinKeyLen {
		key, known = placeholderKey[:], false
	}
	aead, err := snapshotAEAD(key, kid)
	if err != nil || len(wire) != aead.NonceSize()+snapshotPlaintextSize+aead.Overhead() {
		return SnapshotCursor{}, ErrInvalidCursor
	}
	nonce, ciphertext := wire[:aead.NonceSize()], wire[aead.NonceSize():]
	plaintext, err := aead.Open(nil, nonce, ciphertext, snapshotAAD(kid, b, digest))
	if err != nil || !known {
		return SnapshotCursor{}, ErrInvalidCursor
	}
	payload, err := unpadSnapshotPayload(plaintext)
	if err != nil {
		return SnapshotCursor{}, ErrInvalidCursor
	}
	var p snapshotPayload
	if json.Unmarshal(payload, &p) != nil || p.V != snapshotVersion || p.T <= 0 || p.Iat <= 0 {
		return SnapshotCursor{}, ErrInvalidCursor
	}
	canonical, err := json.Marshal(p)
	if err != nil || !bytes.Equal(payload, canonical) {
		return SnapshotCursor{}, ErrInvalidCursor
	}
	snapshot := SnapshotCursor{
		Position:  Cursor{Created: time.Unix(0, p.T).UTC(), Kind: p.K, ID: p.I},
		HighWater: p.H, IssuedAt: time.Unix(0, p.Iat).UTC(),
	}
	if !validSnapshot(snapshot, c.now(), c.ttl) {
		return SnapshotCursor{}, ErrInvalidCursor
	}
	return snapshot, nil
}

func padSnapshotPayload(payload []byte) ([]byte, error) {
	if len(payload) == 0 || len(payload) > snapshotPlaintextSize-snapshotHeaderSize {
		return nil, errSeal
	}
	plaintext := make([]byte, snapshotPlaintextSize)
	binary.BigEndian.PutUint16(plaintext, uint16(len(payload)))
	copy(plaintext[snapshotHeaderSize:], payload)
	return plaintext, nil
}

func unpadSnapshotPayload(plaintext []byte) ([]byte, error) {
	if len(plaintext) != snapshotPlaintextSize {
		return nil, ErrInvalidCursor
	}
	n := int(binary.BigEndian.Uint16(plaintext))
	if n == 0 || n > snapshotPlaintextSize-snapshotHeaderSize {
		return nil, ErrInvalidCursor
	}
	var nonzero byte
	for _, value := range plaintext[snapshotHeaderSize+n:] {
		nonzero |= value
	}
	if nonzero != 0 {
		return nil, ErrInvalidCursor
	}
	return plaintext[snapshotHeaderSize : snapshotHeaderSize+n], nil
}

// validSnapshot uses saturating duration subtraction rather than adding to an
// attacker-controlled issue time. The explicit Unix-nanosecond round-trip
// checks reject overflowed or unrepresentable clocks before evaluating age.
func validSnapshot(snapshot SnapshotCursor, now time.Time, ttl time.Duration) bool {
	if !validPosition(snapshot.Position) || snapshot.HighWater <= 0 ||
		!validSnapshotInstant(snapshot.IssuedAt) || !validSnapshotInstant(now) {
		return false
	}
	age := now.Sub(snapshot.IssuedAt)
	return age >= -maxIssueSkew && age < ttl
}

func validSnapshotInstant(instant time.Time) bool {
	n := instant.UnixNano()
	return n > 0 && time.Unix(0, n).Equal(instant)
}

// snapshotAEAD derives a purpose- and key-ID-specific AES key. Cursor MAC keys
// are never used directly as encryption keys, even when operators use the same
// KeySet for legacy signed positions and encrypted snapshot cursors.
func snapshotAEAD(key []byte, kid string) (cipher.AEAD, error) {
	m := hmac.New(sha256.New, key)
	m.Write(appendField(appendField(nil, []byte(snapshotKeyLabel)), []byte(kid)))
	block, err := aes.NewCipher(m.Sum(nil))
	if err != nil {
		return nil, err
	}
	return cipher.NewGCM(block)
}

func snapshotAAD(kid string, b Binding, digest [sha256.Size]byte) []byte {
	data := appendField(nil, []byte(snapshotAADLabel))
	data = appendField(data, []byte(kid))
	data = appendField(data, []byte(b.Network))
	data = appendField(data, []byte(b.Viewer))
	return appendField(data, digest[:])
}
