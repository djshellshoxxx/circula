package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/djshellshoxxx/circula/internal/domain"
	store "github.com/djshellshoxxx/circula/internal/repository"
)

// ListingInput creates a listing (FR-021). Price is optional minor units.
type ListingInput struct {
	Type           domain.ListingType
	Title          string
	Description    string
	Price          *int64
	IdempotencyKey string `json:"-"`
}

// MaxActiveListingsPerMember caps the active listings one member can have, so
// one member cannot flood the directory (A3-08). Paused and archived listings
// do not count.
const MaxActiveListingsPerMember = 100

// cleanListingText refuses control characters and invisible format runes
// (zero-width joiners and spaces, bidi overrides, BOM, line separators), which
// let a listing spoof another text or hide content (A3-08). In multi-line text
// only newline, carriage return and tab are allowed.
func cleanListingText(field, v string, multiline bool) error {
	for _, r := range v {
		if multiline && (r == '\n' || r == '\r' || r == '\t') {
			continue
		}
		if unicode.IsControl(r) || unicode.Is(unicode.Cf, r) || unicode.Is(unicode.Zl, r) || unicode.Is(unicode.Zp, r) {
			return domain.Invalid(field, "must not contain control or invisible formatting characters")
		}
	}
	return nil
}

func validateListing(title, desc string, price *int64) (string, error) {
	title = strings.TrimSpace(title)
	if c := utf8.RuneCountInString(title); c < 1 || c > 120 {
		return "", domain.Invalid("title", "must be 1 to 120 characters")
	}
	if err := cleanListingText("title", title, false); err != nil {
		return "", err
	}
	if err := boundedText("description", desc, maxTextLen); err != nil {
		return "", err
	}
	if err := cleanListingText("description", desc, true); err != nil {
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
			if err := activeListingRoom(tx, netID, p.member.ID); err != nil {
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

// activeListingRoom refuses another active listing once the member holds
// MaxActiveListingsPerMember.
func activeListingRoom(tx store.Tx, netID, memberID string) error {
	ls, err := tx.ListListings(netID, store.ListingFilter{MemberID: memberID, Status: domain.ListingActive, Limit: MaxActiveListingsPerMember})
	if err != nil {
		return err
	}
	if len(ls) >= MaxActiveListingsPerMember {
		return domain.Invalid("listing", "at most 100 active listings per member; pause or archive some first")
	}
	return nil
}

// ListingUpdate edits the owner's listing; nil fields are unchanged.
type ListingUpdate struct {
	ListingID   string
	Title       *string
	Description *string
	Price       *int64 // set price; use ClearPrice to remove
	ClearPrice  bool
	Status      *domain.ListingStatus // active | paused | archived
	// ExpectedRevision, when supplied, must match inside the write unit.
	// Empty preserves unconditional edits, including wildcard API updates.
	ExpectedRevision string `json:"-"`
}

// ErrListingChanged means a conditional owner edit lost its precondition
// inside the write transaction. No listing or audit change was written.
var ErrListingChanged = errors.New("listing changed since it was read")

// ListingRevision fingerprints the editable values and stored update time.
// This retains the existing API validator's UTC millisecond representation,
// field order and 96-bit digest so issued listing ETags remain compatible.
func ListingRevision(l *domain.Listing) string {
	price := "-"
	if l.Price != nil {
		price = strconv.FormatInt(*l.Price, 10)
	}
	h := sha256.New()
	for _, part := range []string{"listing", l.ID, l.UpdatedAt.UTC().Format("2006-01-02T15:04:05.000Z"),
		string(l.Status), l.Title, l.Description, price} {
		h.Write([]byte(part))
		h.Write([]byte{0})
	}
	return hex.EncodeToString(h.Sum(nil)[:12])
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
		if !domain.ValidRecordID(in.ListingID) {
			return domain.NotFound("listing")
		}
		l, err := tx.GetListing(netID, in.ListingID)
		if err != nil || l.MemberID != p.member.ID {
			return domain.NotFound("listing")
		}
		if in.ExpectedRevision != "" && in.ExpectedRevision != ListingRevision(l) {
			return ErrListingChanged
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
		// A status-only change (pause, archive) must stay possible for an older
		// listing whose text predates the A3-08 character rules.
		if in.Title != nil || in.Description != nil || in.Price != nil || in.ClearPrice {
			if l.Title, err = validateListing(l.Title, l.Description, l.Price); err != nil {
				return err
			}
		}
		if l.Status == domain.ListingActive && before != domain.ListingActive {
			if err := activeListingRoom(tx, netID, p.member.ID); err != nil {
				return err
			}
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
		if q.MemberID != "" && !domain.ValidRecordID(q.MemberID) {
			return domain.Invalid("member_id", "must be a record identifier")
		}
		f := store.ListingFilter{Type: q.Type, Query: q.Query, MemberID: q.MemberID, AfterID: q.Page.AfterID, Limit: clampPage(q.Page)}
		if q.MemberID != p.member.ID {
			f.Status, f.ActiveOwnersOnly = domain.ListingActive, true
		}
		if err := validatePageIDs(q.Page.AfterID); err != nil {
			return err
		}
		out, err = tx.ListListings(netID, f)
		return err
	})
	return out, err
}
