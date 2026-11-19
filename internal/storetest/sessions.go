package storetest

import (
	"testing"
	"time"

	"github.com/djshellshoxxx/circula/internal/domain"
	store "github.com/djshellshoxxx/circula/internal/repository"
)

func sessionOwnership(t *testing.T, s store.Store) {
	seed(t, s, "")
	must(t, s.Update(ctx, func(tx store.Tx) error {
		for _, row := range []struct{ id, owner string }{{"first", "pa"}, {"second", "pa"}, {"foreign", "pb"}} {
			if err := tx.InsertSession(&domain.Session{IDHash: row.id, PersonID: row.owner,
				CreatedAt: t0, LastSeenAt: t0, ExpiresAt: t0.Add(time.Hour)}); err != nil {
				return err
			}
		}
		if changed, err := tx.RevokeOwnedSession("pa", "foreign", t0); err != nil || changed {
			t.Fatalf("owner-scoped revoke matched foreign session: changed=%v err=%v", changed, err)
		}
		for i := 0; i < 2; i++ {
			if changed, err := tx.RevokeOwnedSession("pa", "second", t0); err != nil || changed != (i == 0) {
				t.Fatalf("repeat revoke: changed=%v err=%v", changed, err)
			}
		}
		return nil
	}))
	must(t, s.View(ctx, func(tx store.Tx) error {
		rows, err := tx.ListSessions("pa")
		if err != nil {
			return err
		}
		if len(rows) != 2 {
			t.Fatalf("own sessions: %+v", rows)
		}
		for _, row := range rows {
			if row.PersonID != "pa" || (row.RevokedAt != nil) != (row.IDHash == "second") {
				t.Errorf("session scope or revoke state: %+v", row)
			}
		}
		foreign, err := tx.GetSession("foreign")
		if err != nil {
			return err
		}
		if foreign.RevokedAt != nil {
			t.Fatal("foreign session was revoked")
		}
		return nil
	}))
}
