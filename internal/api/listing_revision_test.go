package api_test

import (
	"context"
	"testing"
	"time"

	"github.com/djshellshoxxx/circula/internal/api"
	"github.com/djshellshoxxx/circula/internal/domain"
	"github.com/djshellshoxxx/circula/internal/service"
	"github.com/djshellshoxxx/circula/internal/store"
	"github.com/djshellshoxxx/circula/internal/testutil"
)

// API-027 / WAPI-02: another owner tab can commit through the service while
// an API edit holds its adapter mutex. Bind explicit validators to the write
// transaction as well as the initial read; wildcard remains existence-only.
func TestListingConditionalWriteRejectsInterveningOwnerEdit(t *testing.T) {
	for _, change := range []string{
		"title", "description", "price", "clear-price", "paused", "timestamp", "etag-list",
		"wildcard", "wildcard-list", "archived", "archived-wildcard",
	} {
		t.Run(change, func(t *testing.T) {
			var wrapped *listingRevisionStore
			env := testutil.NewEnvWith(t, func(s store.Store) store.Store {
				wrapped = &listingRevisionStore{Store: s}
				return wrapped
			})
			w := &world{t: t, env: env, h: api.NewHandlerWithOptions(env.Svc, api.Options{Now: env.Clock.Now})}
			t.Cleanup(func() { api.CloseHandler(w.h) })
			w.admin = env.Person("admin")
			w.net, w.adminM = env.Network(w.admin, "listing-revision", nil)
			w.alice, w.aliceM = env.Member(w.admin, w.net, "alice")
			w.aliceT = w.token(w.alice, domain.ScopeRead, domain.ScopeWrite)
			price := int64(500)
			listing := testutil.Must(env.Svc.CreateListing(env.Ctx, w.alice, w.net.ID, service.ListingInput{
				Type: domain.ListingOffer, Title: "Bike repair", Description: "Original description", Price: &price, IdempotencyKey: env.Key(),
			}))
			path := "/api/v1/listings/" + listing.ID
			current := w.do(req{method: "GET", path: path, token: w.aliceT})
			wantStatus(t, current, 200)
			etag := current.Header().Get("ETag")
			switch change {
			case "etag-list":
				etag = `"another-version", ` + etag
			case "wildcard", "archived-wildcard":
				etag = "*"
			case "wildcard-list":
				etag = `"another-version", *`
			}
			var afterEdit testutil.State
			intervened := false
			wrapped.afterListingRead = func() {
				u := service.ListingUpdate{ListingID: listing.ID}
				switch change {
				case "title", "etag-list":
					title := "Owner's newer title"
					u.Title = &title
				case "description", "wildcard", "wildcard-list":
					description := "Owner's newer description"
					u.Description = &description
				case "price":
					newPrice := int64(700)
					u.Price = &newPrice
				case "clear-price":
					u.ClearPrice = true
				case "paused":
					status := domain.ListingPaused
					u.Status = &status
				case "archived", "archived-wildcard":
					status := domain.ListingArchived
					u.Status = &status
				case "timestamp":
					env.Clock.Advance(time.Millisecond)
				}
				// Except for the timestamp case, both owner edits have the same
				// stored time. The revision must also cover their editable values.
				testutil.Must(env.Svc.UpdateListing(env.Ctx, w.alice, w.net.ID, u))
				afterEdit = testutil.Snapshot(t, env.Store.DB())
				intervened = true
			}
			result := w.do(req{method: "PATCH", path: path, token: w.aliceT,
				body: m{"title": "Owner's first-tab edit"}, hdr: map[string]string{"If-Match": etag}})
			if !intervened {
				t.Fatal("test did not place the owner edit between the conditional read and write")
			}
			if change == "wildcard" || change == "wildcard-list" {
				wantStatus(t, result, 200)
				got := testutil.Must(env.Svc.GetListing(env.Ctx, w.alice, w.net.ID, listing.ID))
				if got.Title != "Owner's first-tab edit" || got.Description != "Owner's newer description" ||
					got.Price == nil || *got.Price != price || got.Status != domain.ListingActive {
					t.Fatalf("wildcard edit did not preserve the other current fields: %+v", got)
				}
				latest := w.do(req{method: "GET", path: path, token: w.aliceT})
				wantStatus(t, latest, 200)
				if result.Header().Get("ETag") == "" || result.Header().Get("ETag") != latest.Header().Get("ETag") {
					t.Fatal("successful edit and subsequent read disagree on ETag")
				}
				after := testutil.Snapshot(t, env.Store.DB())
				if after.Ledger != afterEdit.Ledger || after.AuditEvents != afterEdit.AuditEvents+1 {
					t.Fatal("successful listing edit must append one audit and leave the ledger unchanged")
				}
				return
			}
			if change == "archived-wildcard" {
				wantProblem(t, result, 409, "invalid-transaction-state")
			} else {
				wantProblem(t, result, 412, "precondition-failed")
			}
			if changed := afterEdit.Changed(testutil.Snapshot(t, env.Store.DB())); len(changed) != 0 {
				t.Fatalf("refused conditional edit changed durable state: %v", changed)
			}
		})
	}
}

type listingRevisionStore struct {
	store.Store
	afterListingRead func()
}

func (s *listingRevisionStore) View(ctx context.Context, fn func(store.Tx) error) error {
	if s.afterListingRead == nil {
		return s.Store.View(ctx, fn)
	}
	read := false
	err := s.Store.View(ctx, func(tx store.Tx) error {
		return fn(&listingRevisionReadTx{Tx: tx, GovRepo: tx.(store.GovRepo), read: &read})
	})
	if err == nil && read && s.afterListingRead != nil {
		f := s.afterListingRead
		s.afterListingRead = nil
		f()
	}
	return err
}

type listingRevisionReadTx struct {
	store.Tx
	store.GovRepo
	read *bool
}

func (t *listingRevisionReadTx) GetListing(netID, listingID string) (*domain.Listing, error) {
	*t.read = true
	return t.Tx.GetListing(netID, listingID)
}
