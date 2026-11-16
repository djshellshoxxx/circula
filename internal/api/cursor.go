package api

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"strconv"
	"strings"

	"github.com/djshellshoxxx/circula/internal/service"
)

// Cursors are opaque, HMAC-signed and bound to the route, network, caller and
// filters that produced them (API-034, API-035).

type cursorBody struct {
	P string `json:"p"` // position: last id or sequence
	F string `json:"f"` // filter fingerprint
}

func (s *server) mac(payload []byte) []byte {
	m := hmac.New(sha256.New, s.cursor)
	m.Write(payload)
	return m.Sum(nil)
}

func (s *server) sealCursor(pos, filter string) string {
	b, _ := json.Marshal(cursorBody{P: pos, F: filter})
	return base64.RawURLEncoding.EncodeToString(b) + "." + base64.RawURLEncoding.EncodeToString(s.mac(b))
}

func (s *server) openCursor(tok, filter string) (string, error) {
	bad := &apiError{status: 400, code: "invalid-cursor", detail: "cursor is not valid for this request"}
	a, b, ok := strings.Cut(tok, ".")
	if !ok {
		return "", bad
	}
	payload, err1 := base64.RawURLEncoding.DecodeString(a)
	sig, err2 := base64.RawURLEncoding.DecodeString(b)
	if err1 != nil || err2 != nil || !hmac.Equal(sig, s.mac(payload)) {
		return "", bad
	}
	var cb cursorBody
	if json.Unmarshal(payload, &cb) != nil || cb.F != filter {
		return "", bad
	}
	return cb.P, nil
}

// pageReq is the parsed limit/cursor of a list request.
type pageReq struct {
	limit  int
	after  string
	filter string
}

// page parses limit (1..100, default 25) and cursor. filter must capture
// everything that shapes the result (route, network, caller, query filters).
func (c *reqCtx) page(filter string) (pageReq, error) {
	n, err := c.qInt("limit", 25, 1, 100)
	if err != nil {
		return pageReq{}, err
	}
	f := c.r.URL.Path + "|" + c.netID + "|" + c.actor.PersonID + "|" + filter
	p := pageReq{limit: n, filter: f}
	if tok := c.query.Get("cursor"); tok != "" {
		if p.after, err = c.s.openCursor(tok, f); err != nil {
			return p, err
		}
	}
	return p, nil
}

// listResponse is the API-034 envelope.
type listResponse struct {
	Data       any     `json:"data"`
	NextCursor *string `json:"next_cursor"`
}

// collect gathers up to limit+1 matching items from a keyset-paged service
// call, bounded in work, and returns the items plus the next cursor.
// fetch returns raw rows after an id; keep filters them in the adapter.
func collect[T any](c *reqCtx, p pageReq, fetch func(pg service.Page) ([]T, error),
	idOf func(T) string, keep func(T) bool) ([]T, *string, error) {

	const batch, maxRounds = 100, 20
	var out []T
	after := p.after
	exhausted := false
	last := ""
	for round := 0; round < maxRounds && len(out) <= p.limit; round++ {
		rows, err := fetch(service.Page{AfterID: after, Limit: batch})
		if err != nil {
			return nil, nil, err
		}
		for _, r := range rows {
			last = idOf(r)
			if keep == nil || keep(r) {
				out = append(out, r)
				if len(out) > p.limit {
					break
				}
			}
		}
		if len(rows) < batch {
			exhausted = true
			break
		}
		after = idOf(rows[len(rows)-1])
	}
	if len(out) > p.limit {
		out = out[:p.limit]
		n := c.s.sealCursor(idOf(out[len(out)-1]), p.filter)
		return out, &n, nil
	}
	if !exhausted && last != "" { // work cap hit: continue from the last scanned row
		n := c.s.sealCursor(last, p.filter)
		return out, &n, nil
	}
	return out, nil, nil
}

func ok(c *reqCtx, status int, v any) error {
	writeJSON(c.w, status, v)
	return nil
}

func listOK[T any](c *reqCtx, items []T, next *string) error {
	if items == nil {
		items = []T{}
	}
	return ok(c, 200, listResponse{Data: items, NextCursor: next})
}

func atoiNonNeg(s string) (int, bool) {
	n, err := strconv.Atoi(s)
	return n, err == nil && n >= 0
}

func itoa(n int) string { return strconv.Itoa(n) }
