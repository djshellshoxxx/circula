package discovery

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"strings"
	"time"
)

// Position is the keyset position of the last item on a page.
type Position struct {
	Score   int64  `json:"sc"`
	Updated int64  `json:"u"`
	ID      string `json:"id"`
}

type cursorPayload struct {
	V    int      `json:"v"`
	KID  string   `json:"k"`
	A    string   `json:"a"` // actor
	N    string   `json:"n"` // network
	F    string   `json:"f"` // filter hash (includes sort, scope, limit)
	Snap int64    `json:"s"` // snapshot revision
	P    Position `json:"p"`
	Exp  int64    `json:"e"`
}

// cursors signs and verifies page cursors with HMAC-SHA256. Cursors are bound
// to the actor, network, complete filter state, sort and snapshot revision, and
// expire. They authorize nothing: every page re-runs fresh authorization.
type cursors struct {
	keys map[string][]byte
	cur  string
	ttl  time.Duration
}

func kid(key []byte) string {
	h := sha256.Sum256(append([]byte("kid:"), key...))
	return hex.EncodeToString(h[:4])
}

func newCursors(keys [][]byte, ttl time.Duration) (*cursors, error) {
	if len(keys) == 0 {
		return nil, invalid("CursorKeys", "at least one signing key is required")
	}
	c := &cursors{keys: map[string][]byte{}, ttl: ttl}
	for i, k := range keys {
		if len(k) < 32 {
			return nil, invalid("CursorKeys", "keys must be at least 32 bytes")
		}
		id := kid(k)
		c.keys[id] = append([]byte(nil), k...)
		if i == 0 {
			c.cur = id
		}
	}
	return c, nil
}

func (c *cursors) mac(key []byte, data string) []byte {
	m := hmac.New(sha256.New, key)
	m.Write([]byte(data))
	return m.Sum(nil)
}

func (c *cursors) seal(a Actor, filterHash string, snap int64, p Position, now time.Time) string {
	pl := cursorPayload{V: 1, KID: c.cur, A: a.ID, N: a.Network, F: filterHash, Snap: snap, P: p, Exp: now.Add(c.ttl).Unix()}
	raw, _ := json.Marshal(pl)
	body := base64.RawURLEncoding.EncodeToString(raw)
	return body + "." + base64.RawURLEncoding.EncodeToString(c.mac(c.keys[c.cur], body))
}

func (c *cursors) open(tok string, a Actor, filterHash string, now time.Time) (int64, Position, error) {
	body, sig, ok := strings.Cut(tok, ".")
	if !ok || len(tok) > 1024 {
		return 0, Position{}, ErrBadCursor
	}
	raw, err := base64.RawURLEncoding.DecodeString(body)
	if err != nil {
		return 0, Position{}, ErrBadCursor
	}
	var pl cursorPayload
	if json.Unmarshal(raw, &pl) != nil {
		return 0, Position{}, ErrBadCursor
	}
	key, known := c.keys[pl.KID]
	got, err := base64.RawURLEncoding.DecodeString(sig)
	if !known || err != nil || !hmac.Equal(got, c.mac(key, body)) {
		return 0, Position{}, ErrBadCursor
	}
	if pl.V != 1 || pl.A != a.ID || pl.N != a.Network || pl.F != filterHash || now.Unix() > pl.Exp || pl.Snap < 0 || pl.P.ID == "" {
		return 0, Position{}, ErrBadCursor
	}
	return pl.Snap, pl.P, nil
}
