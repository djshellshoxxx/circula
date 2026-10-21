package api

import (
	"encoding/json"

	"github.com/djshellshoxxx/circula/internal/domain"
	"github.com/djshellshoxxx/circula/internal/service"
)

func (s *server) listListings(c *reqCtx) error { // API-046
	if err := c.only("limit", "cursor", "kind", "q", "member_id"); err != nil {
		return err
	}
	kind := domain.ListingType(c.query.Get("kind"))
	if kind != "" && kind != domain.ListingOffer && kind != domain.ListingRequest {
		return invalidQuery("kind must be offer or request")
	}
	q, mid := c.query.Get("q"), c.query.Get("member_id")
	if len(q) > 200 {
		return invalidQuery("q is too long")
	}
	p, err := c.page(string(kind) + "|" + q + "|" + mid)
	if err != nil {
		return err
	}
	rows, next, err := collect(c, p, func(pg service.Page) ([]*domain.Listing, error) {
		return s.svc.ListListings(c.r.Context(), c.actor, c.netID,
			service.ListingQuery{Type: kind, Query: q, MemberID: mid, Page: pg})
	}, func(l *domain.Listing) string { return l.ID }, nil)
	if err != nil {
		return err
	}
	out := make([]listingDTO, 0, len(rows))
	for _, l := range rows {
		out = append(out, toListing(l))
	}
	return listOK(c, out, next)
}

type createListingReq struct {
	Kind        string  `json:"kind"`
	Title       string  `json:"title"`
	Description string  `json:"description"`
	PriceMinor  *string `json:"price_minor"`
}

func (s *server) createListing(c *reqCtx) error { // API-046
	var in createListingReq
	if err := c.decode(&in); err != nil {
		return err
	}
	var price *int64
	if in.PriceMinor != nil {
		v, err := parseMinor("price_minor", *in.PriceMinor, false)
		if err != nil {
			return err
		}
		price = &v
	}
	l, err := s.svc.CreateListing(c.r.Context(), c.actor, c.netID, service.ListingInput{Type: domain.ListingType(in.Kind),
		Title: in.Title, Description: in.Description, Price: price, IdempotencyKey: c.idemKey})
	if err != nil {
		return err
	}
	c.w.Header().Set("ETag", listingETag(l))
	return ok(c, 201, toListing(l))
}

// ownListing finds one of the caller's listings (any status); the service has
// no single-listing read. Not found covers non-owners (API-017).
func (s *server) ownListing(c *reqCtx, id string) (*domain.Listing, error) {
	after := ""
	for i := 0; i < 20; i++ {
		rows, err := s.svc.ListListings(c.r.Context(), c.actor, c.netID,
			service.ListingQuery{MemberID: c.member.ID, Page: service.Page{AfterID: after, Limit: 100}})
		if err != nil {
			return nil, err
		}
		for _, l := range rows {
			if l.ID == id {
				return l, nil
			}
		}
		if len(rows) < 100 {
			break
		}
		after = rows[len(rows)-1].ID
	}
	return nil, notFoundErr()
}

type patchListingReq struct {
	Title       json.RawMessage `json:"title"`
	Description json.RawMessage `json:"description"`
	PriceMinor  json.RawMessage `json:"price_minor"`
	Status      json.RawMessage `json:"status"`
}

func isNull(r json.RawMessage) bool { return string(r) == "null" }

func rawString(field string, r json.RawMessage) (string, error) {
	var v string
	if isNull(r) || json.Unmarshal(r, &v) != nil {
		return "", validation(field, "invalid-type", "must be a string")
	}
	return v, nil
}

func (s *server) patchListing(c *reqCtx) error { // API-047 (merge-patch + If-Match)
	var in patchListingReq
	if err := c.decode(&in); err != nil {
		return err
	}
	s.casMu.Lock()
	defer s.casMu.Unlock()
	cur, err := s.ownListing(c, c.r.PathValue("id"))
	if err != nil {
		return err
	}
	if err := c.requireIfMatch(listingETag(cur)); err != nil {
		return err
	}
	u := service.ListingUpdate{ListingID: cur.ID}
	if in.Title != nil {
		v, err := rawString("title", in.Title)
		if err != nil {
			return err
		}
		u.Title = &v
	}
	if in.Description != nil {
		v, err := rawString("description", in.Description)
		if err != nil {
			return err
		}
		u.Description = &v
	}
	if in.Status != nil {
		v, err := rawString("status", in.Status)
		if err != nil {
			return err
		}
		st := domain.ListingStatus(v)
		u.Status = &st
	}
	if in.PriceMinor != nil {
		if isNull(in.PriceMinor) {
			u.ClearPrice = true
		} else {
			v, err := rawString("price_minor", in.PriceMinor)
			if err != nil {
				return err
			}
			n, err := parseMinor("price_minor", v, false)
			if err != nil {
				return err
			}
			u.Price = &n
		}
	}
	l, err := s.svc.UpdateListing(c.r.Context(), c.actor, c.netID, u)
	if err != nil {
		return err
	}
	c.w.Header().Set("ETag", listingETag(l))
	return ok(c, 200, toListing(l))
}
