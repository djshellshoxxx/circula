package areas

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"errors"
	"time"
)

type ListingPage struct {
	Listings []string `json:"listings"`
	Next     string   `json:"next_cursor,omitempty"`
}
type listingCursor struct {
	Network, Member, Area, After string
	Expires                      int64
}

func cursorCipher(key []byte) (cipher.AEAD, error) {
	if len(key) != 32 {
		return nil, errors.New("a 32-byte host cursor key is required")
	}
	block, e := aes.NewCipher(key)
	if e != nil {
		return nil, e
	}
	return cipher.NewGCM(block)
}
func sealCursor(key []byte, c listingCursor) (string, error) {
	aead, e := cursorCipher(key)
	if e != nil {
		return "", e
	}
	nonce := make([]byte, aead.NonceSize())
	if _, e = rand.Read(nonce); e != nil {
		return "", e
	}
	body, e := json.Marshal(c)
	if e != nil {
		return "", e
	}
	return base64.RawURLEncoding.EncodeToString(aead.Seal(nonce, nonce, body, []byte("circula-area-listings-v1"))), nil
}
func openCursor(key []byte, encoded string) (listingCursor, error) {
	c := listingCursor{}
	aead, e := cursorCipher(key)
	if e != nil {
		return c, e
	}
	if len(encoded) > 4096 {
		return c, errors.New("cursor too large")
	}
	raw, e := base64.RawURLEncoding.Strict().DecodeString(encoded)
	if e != nil || len(raw) < aead.NonceSize()+aead.Overhead() {
		return c, errors.New("invalid cursor")
	}
	body, e := aead.Open(nil, raw[:aead.NonceSize()], raw[aead.NonceSize():], []byte("circula-area-listings-v1"))
	if e != nil {
		return c, errors.New("invalid cursor")
	}
	e = json.Unmarshal(body, &c)
	return c, e
}

// FilterPage uses encrypted actor/network/area-bound keysets, expiring after 15
// minutes. Every page rechecks visibility; concurrent edits may change membership
// between pages. Scanning is bounded to 1000 references even for hidden listings.
// The opaque next cursor never discloses the ID of a hidden candidate.
func (s Service) FilterPage(ctx context.Context, a Identity, area, cursor string, limit int, key []byte) (ListingPage, error) {
	result := ListingPage{Listings: []string{}}
	if s.DB == nil || s.Policy == nil || s.Now == nil || !validIdentity(a) || !validKey(area) || limit < 1 || limit > 100 {
		return result, errors.New("invalid page request")
	}
	if _, e := cursorCipher(key); e != nil {
		return result, e
	}
	after := ""
	if cursor != "" {
		c, e := openCursor(key, cursor)
		if e != nil || c.Network != a.Network || c.Member != a.Member || c.Area != area || !validKey(c.After) || c.Expires <= s.Now().Unix() {
			return result, errors.New("invalid or expired cursor")
		}
		after = c.After
	}
	tx, e := s.DB.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if e != nil {
		return result, e
	}
	defer tx.Rollback()
	if e = s.Policy.Check(ctx, tx, a, "read", ""); e != nil {
		return result, e
	}
	var visible, archived bool
	if e = tx.QueryRowContext(ctx, `SELECT visible,archived FROM future_areas WHERE network=? AND id=?`, a.Network, area).Scan(&visible, &archived); e != nil || !visible || archived {
		return result, ErrUnavailable
	}
	rows, e := tx.QueryContext(ctx, `SELECT listing FROM future_listing_areas WHERE network=? AND area_id=? AND active=1 AND listing>? ORDER BY listing LIMIT 1001`, a.Network, area, after)
	if e != nil {
		return result, e
	}
	ids := []string{}
	for rows.Next() {
		var id string
		if e = rows.Scan(&id); e != nil {
			rows.Close()
			return result, e
		}
		ids = append(ids, id)
	}
	e = rows.Err()
	rows.Close()
	if e != nil {
		return result, e
	}
	more := len(ids) > 1000
	if more {
		ids = ids[:1000]
	}
	for i, id := range ids {
		if !validKey(id) {
			return result, errors.New("invalid stored listing reference")
		}
		if s.Policy.Check(ctx, tx, a, "listing", id) == nil {
			result.Listings = append(result.Listings, id)
		}
		if e = ctx.Err(); e != nil {
			return ListingPage{}, e
		}
		if len(result.Listings) == limit || i == len(ids)-1 {
			if i+1 < len(ids) || more {
				result.Next, e = sealCursor(key, listingCursor{a.Network, a.Member, area, id, s.Now().Add(15 * time.Minute).Unix()})
				if e != nil {
					return result, e
				}
			}
			if len(result.Listings) == limit {
				break
			}
		}
	}
	return result, nil
}
