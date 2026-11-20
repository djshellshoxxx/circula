package storetest

import (
	"fmt"
	"testing"
	"time"

	"github.com/djshellshoxxx/circula/internal/domain"
	store "github.com/djshellshoxxx/circula/internal/repository"
)

// tokenInventoryBound enforces the bounded repository result itself: an HTTP
// response-only truncation does not satisfy API-036. The read must preserve
// owner isolation, stable order and retained credentials beyond the cap.
func tokenInventoryBound(t *testing.T, s store.Store) {
	seed(t, s, "a")
	id := func(owner string, n int) string { return fmt.Sprintf("%03d-%s", n, owner) }
	must(t, s.Update(ctx, func(tx store.Tx) error {
		for n := 134; n >= 0; n-- {
			for _, owner := range []string{"apa", "apb"} {
				tokenID := id(owner, n)
				if err := tx.InsertToken(&domain.APIToken{ID: tokenID, PersonID: owner,
					Name: tokenID, TokenHash: "hash-" + tokenID,
					Scopes: []domain.Scope{domain.ScopeRead}, CreatedAt: t0}); err != nil {
					return err
				}
			}
		}
		_, err := tx.RevokeToken("apa", id("apa", 0), t0.Add(time.Minute))
		return err
	}))
	view(t, s, func(tx store.Tx) error {
		for _, owner := range []string{"apa", "apb", "missing"} {
			rows, err := tx.ListTokens(owner)
			if err != nil {
				return err
			}
			want := 100
			if owner == "missing" {
				want = 0
			}
			if len(rows) != want {
				t.Fatalf("owner %s: %d rows, want %d", owner, len(rows), want)
			}
			for n, row := range rows {
				if row.ID != id(owner, n) || row.PersonID != owner {
					t.Fatalf("owner %s row %d: %+v", owner, n, row)
				}
			}
			if owner == "apa" && rows[0].RevokedAt == nil {
				t.Fatal("bounded inventory dropped retained revoked metadata")
			}
		}
		return nil
	})
	outside := id("apa", 134)
	must(t, s.Update(ctx, func(tx store.Tx) error {
		changed, err := tx.RevokeToken("apb", outside, t0.Add(time.Minute))
		if err != nil {
			return err
		}
		if changed {
			t.Fatal("foreign owner revoked a token beyond the listing cap")
		}
		changed, err = tx.RevokeToken("apa", outside, t0.Add(time.Minute))
		if err == nil && !changed {
			t.Fatal("owner could not revoke a token beyond the listing cap")
		}
		return err
	}))
	view(t, s, func(tx store.Tx) error {
		row, err := tx.GetTokenByHash("hash-" + outside)
		if err == nil && (row.ID != outside || row.RevokedAt == nil) {
			t.Fatalf("token beyond cap was lost or not revoked: %+v", row)
		}
		return err
	})
}
