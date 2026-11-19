package feed

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"errors"
	"reflect"
	"sort"
	"strings"
	"time"
	"unicode/utf8"
)

// Feed cursors are opaque, HMAC-SHA256 signed pagination positions. They follow
// the core contract in internal/api/cursor.go (API-034, API-035): the MAC binds
// the position to the network, the viewer and a canonical filter digest, so a
// cursor minted for one of those is refused for any other. Unlike the core
// cursor, the bound identifiers are not written into the token. They enter only
// the MAC input, so a token reveals just its key ID, issue time and position
// (the kind, ID and time of an item the viewer was just shown). The position is
// readable, not secret: "opaque" means clients must not parse or build tokens.
//
// Wire format, version 2:
//
//	<key id> "." base64url(payload JSON) "." base64url(HMAC-SHA256)
//
// The MAC input is the length-prefixed sequence label, key ID, network, viewer,
// filter digest, payload. The label (cursorMACLabel) separates feed cursor MACs
// from core cursor MACs, which are taken over raw JSON, even if an operator
// reuses one secret for both; it also versions the construction.

const (
	cursorMACLabel    = "circula/future/feed-cursor/v2"
	filterDigestLabel = "circula/future/feed-filter/v1"
	cursorVersion     = 2

	// MinKeyLen is the minimum cursor MAC key length, in bytes.
	MinKeyLen = 32

	// DefaultCursorTTL is how long a cursor stays valid after it is issued
	// when CodecOptions.TTL is zero. PROVISIONAL: owner to confirm the value.
	DefaultCursorTTL = time.Hour

	// MaxCursorTTL bounds a configured TTL so that no configuration makes
	// cursors effectively permanent.
	MaxCursorTTL = 24 * time.Hour

	// maxIssueSkew is how far in the future an issue time may be (a verifier
	// clock behind the signer's) before the cursor is refused.
	maxIssueSkew = time.Minute

	maxTokenLen    = 2048
	maxKeyIDLen    = 32
	maxBindingLen  = 256
	maxSourceIDLen = 128
	maxFilterKinds = 16
)

// sourceKinds are the only source kinds a feed may show, in canonical order.
var sourceKinds = []string{"announcement", "listing"}

func isSourceKind(k string) bool { return k == "announcement" || k == "listing" }

// ErrInvalidCursor is the only error Open returns, whatever was wrong with the
// token (malformed, tampered, unknown key, expired, or minted for another
// network, viewer or filter), so callers cannot learn why a cursor failed.
var ErrInvalidCursor = errors.New("feed: cursor is not valid for this request")

var (
	errSeal           = errors.New("feed: cannot seal cursor")
	errInvalidBinding = errors.New("feed: invalid network, viewer or filter")
)

// b64 decodes strictly: non-zero trailing bits are refused, so every payload
// and MAC has exactly one accepted encoding.
var b64 = base64.RawURLEncoding.Strict()

// KeySet supplies cursor MAC keys. New cursors are signed with SigningKey. A
// cursor verifies only under the key that VerificationKey returns for the key
// ID the cursor carries. Rotation is: add the new key, make it the signing key,
// keep the old key verifiable for at least one TTL, then remove it.
//
// Where keys come from, how they persist across restarts and how often they
// rotate are owner decisions left to a future adapter (see README.md).
// Implementations must be safe for concurrent use. Return a nil key to refuse.
type KeySet interface {
	SigningKey() (id string, key []byte)
	VerificationKey(id string) (key []byte, ok bool)
}

// KeyRing is an immutable, in-memory KeySet.
type KeyRing struct {
	current string
	keys    map[string][]byte
}

// NewKeyRing returns a KeyRing that signs with keys[current] and verifies with
// any key in keys. Key IDs are 1 to 32 characters of [A-Za-z0-9_-]; keys are at
// least MinKeyLen bytes. Keys are copied.
func NewKeyRing(current string, keys map[string][]byte) (*KeyRing, error) {
	if _, ok := keys[current]; !ok {
		return nil, errors.New("feed: signing key id is not in the key ring")
	}
	r := &KeyRing{current: current, keys: make(map[string][]byte, len(keys))}
	for id, k := range keys {
		if !validKeyID(id) || len(k) < MinKeyLen {
			return nil, errors.New("feed: invalid cursor key or key id")
		}
		r.keys[id] = append([]byte(nil), k...)
	}
	return r, nil
}

// SigningKey returns a copy of the current key. Mutating or clearing the copy
// cannot change the ring or race with another caller.
func (r *KeyRing) SigningKey() (string, []byte) {
	if r == nil {
		return "", nil
	}
	return r.current, append([]byte(nil), r.keys[r.current]...)
}

// VerificationKey returns a copy of the key with this ID.
func (r *KeyRing) VerificationKey(id string) ([]byte, bool) {
	if r == nil {
		return nil, false
	}
	k, ok := r.keys[id]
	return append([]byte(nil), k...), ok
}

// CodecOptions configures a CursorCodec.
type CodecOptions struct {
	// TTL is how long a cursor stays valid after it is issued, from one second
	// to MaxCursorTTL. Zero selects DefaultCursorTTL (provisional; owner to
	// confirm). The verifier's TTL applies, so shortening it also shortens
	// cursors already issued.
	TTL time.Duration
	// Now is the clock. Nil selects time.Now.
	Now func() time.Time
}

// CursorCodec seals and opens feed cursors. It is immutable and safe for
// concurrent use when its KeySet and clock are. Its zero value refuses all
// operations; construct a usable codec with NewCursorCodec.
type CursorCodec struct {
	keys KeySet
	ttl  time.Duration
	now  func() time.Time
}

func (c *CursorCodec) ready() bool {
	if c == nil || c.keys == nil || c.now == nil || c.ttl < time.Second || c.ttl > MaxCursorTTL {
		return false
	}
	// An interface containing a typed nil is not itself nil. Refuse it before
	// invoking a custom KeySet whose methods might dereference that receiver.
	v := reflect.ValueOf(c.keys)
	switch v.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Ptr, reflect.Slice:
		return !v.IsNil()
	default:
		return true
	}
}

// NewCursorCodec returns a codec over keys. It fails closed: there is no
// unsigned or keyless mode.
func NewCursorCodec(keys KeySet, opt CodecOptions) (*CursorCodec, error) {
	ttl := opt.TTL
	if ttl == 0 {
		ttl = DefaultCursorTTL
	}
	if keys == nil || ttl < time.Second || ttl > MaxCursorTTL {
		return nil, errors.New("feed: invalid cursor codec options")
	}
	now := opt.Now
	if now == nil {
		now = time.Now
	}
	return &CursorCodec{keys: keys, ttl: ttl, now: now}, nil
}

// Binding is everything besides the position that a cursor is valid for.
type Binding struct {
	Network string // network (tenant) the page was served in
	Viewer  string // authenticated person the page was served to
	Filter  Filter // viewer-chosen shape of the page
}

// Filter is the viewer-chosen shape of a feed page. Filters that select the
// same items share one canonical digest.
type Filter struct {
	// Kinds restricts items to these source kinds ("listing",
	// "announcement"). Empty means every kind; order and repeats are ignored.
	// Any other kind is refused rather than ignored.
	Kinds []string
	// PreferenceRevision is an opaque version of the viewer's feed
	// preferences (categories, mute, follow) supplied by a future adapter, at
	// most 256 bytes. Changing it invalidates outstanding cursors. What it
	// covers is an owner decision; empty means no preferences are applied.
	PreferenceRevision string
}

// canonical returns the sorted, de-duplicated kinds this filter selects and
// its digest.
func (f Filter) canonical() ([]string, [sha256.Size]byte, error) {
	var digest [sha256.Size]byte
	if len(f.Kinds) > maxFilterKinds || len(f.PreferenceRevision) > maxBindingLen {
		return nil, digest, errInvalidBinding
	}
	set := map[string]bool{}
	for _, k := range f.Kinds {
		if !isSourceKind(k) {
			return nil, digest, errInvalidBinding
		}
		set[k] = true
	}
	kinds := sourceKinds
	if len(set) > 0 {
		kinds = make([]string, 0, len(set))
		for k := range set {
			kinds = append(kinds, k)
		}
		sort.Strings(kinds)
	}
	buf := appendField(nil, []byte(filterDigestLabel))
	buf = binary.BigEndian.AppendUint32(buf, uint32(len(kinds)))
	for _, k := range kinds {
		buf = appendField(buf, []byte(k))
	}
	buf = appendField(buf, []byte(f.PreferenceRevision))
	return kinds, sha256.Sum256(buf), nil
}

// canonical validates the binding and returns the filter's kinds and digest.
func (b Binding) canonical() ([]string, [sha256.Size]byte, error) {
	if b.Network == "" || len(b.Network) > maxBindingLen || b.Viewer == "" || len(b.Viewer) > maxBindingLen {
		return nil, [sha256.Size]byte{}, errInvalidBinding
	}
	return b.Filter.canonical()
}

type cursorPayload struct {
	V   int    `json:"v"`
	K   string `json:"k"`   // position: source kind
	I   string `json:"i"`   // position: source ID
	T   int64  `json:"t"`   // position: source time, Unix nanoseconds
	Iat int64  `json:"iat"` // issue time, Unix seconds
}

// Seal signs pos for b with the current signing key and the codec's clock.
func (c *CursorCodec) Seal(b Binding, pos Cursor) (string, error) {
	if !c.ready() {
		return "", errSeal
	}
	kid, key := c.keys.SigningKey()
	return sealWith(kid, key, b, pos, c.now())
}

func sealWith(kid string, key []byte, b Binding, pos Cursor, now time.Time) (string, error) {
	_, digest, err := b.canonical()
	iat := now.Unix()
	if err != nil || !validKeyID(kid) || len(key) < MinKeyLen || !validPosition(pos) || iat <= 0 {
		return "", errSeal
	}
	p, err := json.Marshal(cursorPayload{V: cursorVersion, K: pos.Kind, I: pos.ID, T: pos.Created.UnixNano(), Iat: iat})
	if err != nil {
		return "", errSeal
	}
	tok := kid + "." + b64.EncodeToString(p) + "." + b64.EncodeToString(cursorMAC(key, kid, b, digest, p))
	if len(tok) > maxTokenLen {
		return "", errSeal
	}
	return tok, nil
}

// placeholderKey keeps the work for an unknown key ID equal to the work for a
// known one. A MAC under it is never accepted.
var placeholderKey [MinKeyLen]byte

// Open verifies token for b and returns its position. Every refusal is
// ErrInvalidCursor. The MAC is checked in constant time before the payload is
// parsed.
func (c *CursorCodec) Open(b Binding, token string) (Cursor, error) {
	if !c.ready() || len(token) > maxTokenLen || !isTokenText(token) {
		return Cursor{}, ErrInvalidCursor
	}
	kid, rest, _ := strings.Cut(token, ".")
	enc, sigEnc, ok := strings.Cut(rest, ".")
	if !ok || strings.Contains(sigEnc, ".") || !validKeyID(kid) {
		return Cursor{}, ErrInvalidCursor
	}
	_, digest, err := b.canonical()
	if err != nil {
		return Cursor{}, ErrInvalidCursor
	}
	payload, err := b64.DecodeString(enc)
	if err != nil || len(payload) == 0 {
		return Cursor{}, ErrInvalidCursor
	}
	sig, err := b64.DecodeString(sigEnc)
	if err != nil || len(sig) != sha256.Size {
		return Cursor{}, ErrInvalidCursor
	}
	key, known := c.keys.VerificationKey(kid)
	if !known || len(key) < MinKeyLen {
		key, known = placeholderKey[:], false
	}
	if !hmac.Equal(sig, cursorMAC(key, kid, b, digest, payload)) || !known {
		return Cursor{}, ErrInvalidCursor
	}
	var v cursorPayload
	if json.Unmarshal(payload, &v) != nil || v.V != cursorVersion || v.T <= 0 || v.Iat <= 0 {
		return Cursor{}, ErrInvalidCursor
	}
	// Require the exact representation Seal emits. encoding/json otherwise
	// accepts duplicate or case-aliased fields and repairs invalid Unicode,
	// which can make the decoded position differ from the authenticated bytes.
	canonical, err := json.Marshal(v)
	if err != nil || !bytes.Equal(payload, canonical) {
		return Cursor{}, ErrInvalidCursor
	}
	pos := Cursor{Created: time.Unix(0, v.T).UTC(), Kind: v.K, ID: v.I}
	issued, now := time.Unix(v.Iat, 0), c.now()
	if !validPosition(pos) || now.Before(issued.Add(-maxIssueSkew)) || !now.Before(issued.Add(c.ttl)) {
		return Cursor{}, ErrInvalidCursor
	}
	return pos, nil
}

func cursorMAC(key []byte, kid string, b Binding, digest [sha256.Size]byte, payload []byte) []byte {
	return labelledMAC(cursorMACLabel, key, kid, b, digest, payload)
}

func labelledMAC(label string, key []byte, kid string, b Binding, digest [sha256.Size]byte, payload []byte) []byte {
	buf := appendField(nil, []byte(label))
	buf = appendField(buf, []byte(kid))
	buf = appendField(buf, []byte(b.Network))
	buf = appendField(buf, []byte(b.Viewer))
	buf = appendField(buf, digest[:])
	buf = appendField(buf, payload)
	m := hmac.New(sha256.New, key)
	m.Write(buf)
	return m.Sum(nil)
}

// appendField appends v with a 4-byte big-endian length prefix, so that no
// two different field sequences produce the same bytes.
func appendField(dst, v []byte) []byte {
	dst = binary.BigEndian.AppendUint32(dst, uint32(len(v)))
	return append(dst, v...)
}

// validPosition reports whether pos can be sealed and round-trips exactly.
func validPosition(pos Cursor) bool {
	if pos.Created.IsZero() || !isSourceKind(pos.Kind) || pos.ID == "" || len(pos.ID) > maxSourceIDLen || !utf8.ValidString(pos.ID) {
		return false
	}
	n := pos.Created.UnixNano()
	return n > 0 && time.Unix(0, n).Equal(pos.Created)
}

func isBase64URL(c byte) bool {
	return c >= 'A' && c <= 'Z' || c >= 'a' && c <= 'z' || c >= '0' && c <= '9' || c == '-' || c == '_'
}

// isTokenText reports whether s holds only base64url characters and dots, so
// the decoder's tolerance of line breaks and padding never applies.
func isTokenText(s string) bool {
	for i := 0; i < len(s); i++ {
		if !isBase64URL(s[i]) && s[i] != '.' {
			return false
		}
	}
	return true
}

func validKeyID(id string) bool {
	if id == "" || len(id) > maxKeyIDLen {
		return false
	}
	for i := 0; i < len(id); i++ {
		if !isBase64URL(id[i]) {
			return false
		}
	}
	return true
}
