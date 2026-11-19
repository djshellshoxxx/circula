package service_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/djshellshoxxx/circula/internal/domain"
	"github.com/djshellshoxxx/circula/internal/service"
)

// SEC-014 acceptance: an over-length free-text value is rejected with a
// field-level error. One case per member-typed text field of the personal-data
// inventory (docs/security/personal-data-inventory.md) that a member can set.
func TestOverLengthFreeTextRejectedPerField(t *testing.T) {
	w := newWorld(t, nil)
	done := w.Trade(w.net.ID, w.a, w.b, w.aM, w.bM, 100)
	long := func(n int) string { return strings.Repeat("x", n) }
	cases := []struct {
		field string
		call  func() error
	}{
		{"login", func() error {
			_, err := w.Svc.RegisterPerson(w.Ctx, service.RegisterInput{Login: long(255), Password: "correct horse battery staple 1"})
			return err
		}},
		{"display_name", func() error {
			_, _, err := w.Svc.CreateNetwork(w.Ctx, w.a, service.CreateNetworkInput{Name: "Second", Slug: "second-net",
				CreatorDisplayName: long(81)})
			return err
		}},
		{"title", func() error {
			_, err := w.Svc.CreateListing(w.Ctx, w.a, w.net.ID, service.ListingInput{Type: domain.ListingOffer,
				Title: long(121), IdempotencyKey: w.Key()})
			return err
		}},
		{"description", func() error {
			_, err := w.Svc.CreateListing(w.Ctx, w.a, w.net.ID, service.ListingInput{Type: domain.ListingOffer,
				Title: "Bread", Description: long(2001), IdempotencyKey: w.Key()})
			return err
		}},
		{"note", func() error {
			_, err := w.Svc.ProposeTransaction(w.Ctx, w.a, w.net.ID, service.ProposeInput{CounterpartyID: w.bM.ID,
				Direction: service.ProposerPays, Amount: 1, Note: long(501), IdempotencyKey: w.Key()})
			return err
		}},
		{"statement", func() error {
			_, err := w.Svc.OpenDispute(w.Ctx, w.a, w.net.ID, service.OpenDisputeInput{TxID: done.ID, Category: "other",
				Statement: long(2001), IdempotencyKey: w.Key()})
			return err
		}},
		{"note", func() error {
			_, err := w.Svc.CreateReport(w.Ctx, w.a, w.net.ID, service.ReportInput{TargetType: service.ReportMember,
				TargetID: w.bM.ID, Category: service.ReportCategories[0], Note: long(501), IdempotencyKey: w.Key()})
			return err
		}},
		{"name", func() error {
			_, _, err := w.Svc.CreateAPIToken(w.Ctx, w.a, service.TokenInput{Name: long(81), Scopes: []domain.Scope{domain.ScopeRead}})
			return err
		}},
	}
	for _, c := range cases {
		err := c.call()
		var de *domain.Error
		if !errors.As(err, &de) || de.Code != domain.CodeValidation {
			t.Errorf("%s: over-length value: err = %v, want a validation error", c.field, err)
			continue
		}
		if len(de.Fields) != 1 || de.Fields[0].Field != c.field {
			t.Errorf("%s: want one field error naming %q, got %+v", c.field, c.field, de.Fields)
		}
	}
}
