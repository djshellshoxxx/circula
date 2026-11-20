package api_test

import (
	"fmt"
	"net/http"
	"sort"
	"strings"
	"testing"

	"github.com/djshellshoxxx/circula/internal/domain"
	store "github.com/djshellshoxxx/circula/internal/repository"
)

func TestTokenInventoryRetainsBoundOrderOwnerAndPrivacy(t *testing.T) { // API-036, API-066
	w := newWorld(t)
	existing, err := w.env.Svc.ListAPITokens(w.env.Ctx, w.alice)
	if err != nil {
		t.Fatal(err)
	}
	var expected []string
	for _, token := range existing {
		expected = append(expected, token.ID)
	}
	lastID := ""
	if err := w.env.Store.Update(w.env.Ctx, func(tx store.Tx) error {
		for n := 119; n >= 0; n-- {
			for ownerN, owner := range []string{w.alice.PersonID, w.bob.PersonID} {
				id := fmt.Sprintf("018f0000-0000-7000-8000-%012x", 10000+n*2+ownerN)
				if err := tx.InsertToken(&domain.APIToken{ID: id, PersonID: owner,
					Name: "inventory", TokenHash: "private-token-hash-" + id,
					Scopes: []domain.Scope{domain.ScopeRead}, CreatedAt: w.env.Clock.Now()}); err != nil {
					return err
				}
				if ownerN == 0 {
					expected = append(expected, id)
					if n == 119 {
						lastID = id
					}
				}
			}
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	sort.Strings(expected)
	response := w.get(w.aliceT, "/tokens")
	wantStatus(t, response, http.StatusOK)
	list := decode[struct {
		Data       []map[string]any `json:"data"`
		NextCursor *string          `json:"next_cursor"`
	}](t, response)
	if len(list.Data) != 100 || list.NextCursor != nil {
		t.Fatalf("unpaginated token contract: %d rows, next=%v", len(list.Data), list.NextCursor)
	}
	for n, row := range list.Data {
		if row["id"] != expected[n] {
			t.Fatalf("row %d ID=%v, want owner token %s", n, row["id"], expected[n])
		}
		for _, field := range []string{"person_id", "token", "token_hash"} {
			if _, ok := row[field]; ok {
				t.Fatalf("private token field %s exposed", field)
			}
		}
	}
	for _, secret := range []string{w.aliceT, w.bobT, "private-token-hash-"} {
		if strings.Contains(response.Body.String(), secret) {
			t.Fatal("token inventory exposed a credential or hash")
		}
	}
	// The inventory cap does not revoke, delete or prevent direct management
	// of a known owner token beyond the first 100 rows.
	wantProblem(t, w.do(req{method: "DELETE", path: "/api/v1/tokens/" + lastID, token: w.bobT}), http.StatusNotFound, "not-found")
	wantStatus(t, w.do(req{method: "DELETE", path: "/api/v1/tokens/" + lastID, token: w.aliceT}), http.StatusNoContent)
}
