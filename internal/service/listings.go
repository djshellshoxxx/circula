package service

import (
	"context"
	"strings"
	"unicode/utf8"

	"github.com/djshellshoxxx/circula/internal/domain"
	"github.com/djshellshoxxx/circula/internal/store"
)

// ListingInput creates a listing (FR-021). Price is optional minor units.
type ListingInput struct {
	Type           domain.ListingType
	Title          string
	Description    string
	Price          *int64
	IdempotencyKey string `json:"-"`
}

func validateListing(title, desc string, price *int64) (string, error) {
	title = strings.TrimSpace(title)
	if c := utf8.RuneCountInString(title); c < 1 || c > 120 {
		return "", domain.Invalid("title", "must be 1 to 120 characters")
	}
	if err := boundedText("description", desc, maxTextLen); err != nil {
		return "", err
	}
	if price != nil && (*price <= 0 || *price > domain.MaxPolicyAmount) {
		return "", domain.Invalid("price", "must be a positive amount")
	}
	return title, nil
}

// CreateListing publishes an offer or request. Listings never affect
// balances, reservations or entries (DOM-040). Active members only.
func (s *Service) CreateListing(ctx context.Context, actor domain.Actor, netID string, in ListingInput) (*domain.Listing, error) {
	var out *domain.Listing
	err := s.update(ctx, func(tx store.Tx) error {
		p, err := s.open(tx, actor, netID, domain.ScopeWrite)
		if err != nil {
			return err
		}
		if err := p.requireActive(); err != nil {
			return err
		}
		out, err = idempotent(s, tx, netID, actor, "listing.create", in.IdempotencyKey, in, func() (string, error) {
			if err := p.requireNetworkActive(); err != nil {
				return "", err
			}
			if in.Type != domain.ListingOffer && in.Type != domain.ListingRequest {
				return "", domain.Invalid("type", "must be offer or request")
			}
			title, err := validateListing(in.Title, in.Description, in.Price)
			if err != nil {
				return "", err
			}
			now := s.now()
			l := &domain.Listing{ID: s.id(), NetworkID: netID, MemberID: p.member.ID, Type: in.Type, Title: title,
				Description: in.Description, Price: in.Price, Status: domain.ListingActive, CreatedAt: now, UpdatedAt: now}
			if err := tx.InsertListing(l); err != nil {
				return "", err
			}
			return l.ID, s.audit(tx, p, auditEntry{Action: domain.ActListingCreated, TargetType: "listing", TargetID: l.ID,
				IdemRef: keyRef(in.IdempotencyKey), After: map[string]any{"type": string(in.Type), "status": "active"}})
		}, func(id string) (*domain.Listing, error) { return tx.GetListing(netID, id) })
		return err
	})
	return out, err
}

// ListingUpdate edits the owner's listing; nil fields are unchanged.
type ListingUpdate struct {
	ListingID   string
	Title       *string
	Description *string
	Price       *int64 // set price; use ClearPrice to remove
	ClearPrice  bool
	Status      *domain.ListingStatus // active | paused | archived
}

// UpdateListing edits, pauses or archives the caller's own listing (FR-022).
// Archived listings are final. Other members' listings read as not-found.
func (s *Service) UpdateListing(ctx context.Context, actor domain.Actor, netID string, in ListingUpdate) (*domain.Listing, error) {
	var out *domain.Listing
	err := s.update(ctx, func(tx store.Tx) error {
		p, err := s.open(tx, actor, netID, domain.ScopeWrite)
		if err != nil {
			return err
		}
		if err := p.requireActive(); err != nil {
			return err
		}
		l, err := tx.GetListing(netID, in.ListingID)
		if err != nil || l.MemberID != p.member.ID {
			return domain.NotFound("listing")
		}
		before := l.Status
		if l.Status == domain.ListingArchived {
			return domain.InvalidState("listing is archived")
		}
		if in.Title != nil {
			l.Title = *in.Title
		}
		if in.Description != nil {
			l.Description = *in.Description
		}
		if in.ClearPrice {
			l.Price = nil
		} else if in.Price != nil {
			l.Price = in.Price
		}
		if in.Status != nil {
			switch *in.Status {
			case domain.ListingActive, domain.ListingPaused, domain.ListingArchived:
				l.Status = *in.Status
			default:
				return domain.Invalid("status", "must be active, paused or archived")
			}
		}
		if l.Title, err = validateListing(l.Title, l.Description, l.Price); err != nil {
			return err
		}
		l.UpdatedAt = s.now()
		if err := tx.UpdateListing(l); err != nil {
			return err
		}
		out = l
		return s.audit(tx, p, auditEntry{Action: domain.ActListingUpdated, TargetType: "listing", TargetID: l.ID,
			Before: map[string]any{"status": string(before)}, After: map[string]any{"status": string(l.Status)}})
	})
	return out, err
}

// ListingQuery filters the directory of listings.
type ListingQuery struct {
	Type     domain.ListingType
	Query    string
	MemberID string // set to browse one member; own listings show every status
	Page     Page
}

// ListListings returns active listings to active members (FR-023, FR-025).
// Paused and archived listings are visible only to their owner.
func (s *Service) ListListings(ctx context.Context, actor domain.Actor, netID string, q ListingQuery) ([]*domain.Listing, error) {
	var out []*domain.Listing
	err := s.view(ctx, func(tx store.Tx) error {
		p, err := s.open(tx, actor, netID, domain.ScopeRead)
		if err != nil {
			return err
		}
		if err := p.requireActive(); err != nil {
			return err
		}
		f := store.ListingFilter{Type: q.Type, Query: q.Query, MemberID: q.MemberID, AfterID: q.Page.AfterID, Limit: clampPage(q.Page)}
		if q.MemberID != p.member.ID {
			f.Status, f.ActiveOwnersOnly = domain.ListingActive, true
		}
		out, err = tx.ListListings(netID, f)
		return err
	})
	return out, err
}
