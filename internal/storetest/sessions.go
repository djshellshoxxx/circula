package storetest

import (
	"fmt"
	"sort"
	"strings"
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
		rows, err := tx.ListSessions("pa", nil, 100)
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
		_, err = tx.GetOwnedSession("pa", foreign.PublicID)
		wantCode(t, "foreign selector lookup", err, domain.CodeNotFound)
		_, err = tx.GetOwnedSession("pb", "foreign")
		wantCode(t, "credential hash is not a public selector", err, domain.CodeNotFound)
		got, err := tx.GetOwnedSession("pb", foreign.PublicID)
		if err != nil || got.IDHash != "foreign" || len(got.PublicID) != 64 {
			t.Fatalf("owned selector lookup: %+v, %v", got, err)
		}
		return nil
	}))
}

// Pagination is over retained rows, not only active sessions: filtering dead
// rows before LIMIT would allow a request to scan an unbounded backlog.
func sessionPagination(t *testing.T, s store.Store) {
	seed(t, s, "")
	var all []*domain.Session
	must(t, s.Update(ctx, func(tx store.Tx) error {
		for i := 0; i < 207; i++ {
			sess := &domain.Session{IDHash: fmt.Sprintf("own-%03d", i), PersonID: "pa",
				CreatedAt: t0.Add(time.Duration(i/70) * time.Millisecond), LastSeenAt: t0, ExpiresAt: t0.Add(time.Hour)}
			if err := tx.InsertSession(sess); err != nil {
				return err
			}
			if len(sess.PublicID) != 64 || strings.Trim(sess.PublicID, "0123456789abcdef") != "" {
				t.Fatalf("insert did not assign a canonical public ID: %+v", sess)
			}
			all = append(all, sess)
		}
		return tx.InsertSession(&domain.Session{IDHash: "foreign", PersonID: "pb",
			CreatedAt: t0.Add(time.Hour), LastSeenAt: t0, ExpiresAt: t0.Add(time.Hour)})
	}))
	sort.Slice(all, func(i, j int) bool {
		if all[i].CreatedAt.Equal(all[j].CreatedAt) {
			return all[i].PublicID > all[j].PublicID
		}
		return all[i].CreatedAt.After(all[j].CreatedAt)
	})
	must(t, s.View(ctx, func(tx store.Tx) error {
		for _, limit := range []int{0, -1, 101, 10000} {
			rows, err := tx.ListSessions("pa", nil, limit)
			if err != nil || len(rows) != 100 {
				t.Fatalf("hard cap for %d: len=%d err=%v", limit, len(rows), err)
			}
		}
		var before *store.SessionPosition
		count := 0
		for {
			rows, err := tx.ListSessions("pa", before, 7)
			if err != nil {
				return err
			}
			if len(rows) > 7 {
				t.Fatal("repository exceeded requested row bound")
			}
			for _, row := range rows {
				if count >= len(all) || row.PublicID != all[count].PublicID || row.PersonID != "pa" {
					t.Fatalf("page order, duplicate or owner mismatch at %d: %+v", count, row)
				}
				count++
				before = &store.SessionPosition{CreatedAt: row.CreatedAt, PublicID: row.PublicID}
			}
			if len(rows) < 7 {
				break
			}
		}
		if count != len(all) {
			t.Fatalf("page traversal skipped rows: %d, want %d", count, len(all))
		}
		return nil
	}))
	boundary := all[6]
	must(t, s.Update(ctx, func(tx store.Tx) error {
		if err := tx.TouchSession(boundary.IDHash, t0.Add(time.Minute)); err != nil {
			return err
		}
		if _, err := tx.RevokeOwnedSession("pa", boundary.IDHash, t0); err != nil {
			return err
		}
		got, err := tx.GetOwnedSession("pa", boundary.PublicID)
		if err != nil || got.PublicID != boundary.PublicID || got.RevokedAt == nil {
			t.Fatalf("touch/revoke changed public ID or lost retained row: %+v, %v", got, err)
		}
		rows, err := tx.ListSessions("pa", nil, 7)
		if err != nil || len(rows) != 7 || rows[6].PublicID != boundary.PublicID {
			t.Fatalf("repository filtered a dead row before LIMIT: %+v, %v", rows, err)
		}
		n, err := tx.DeleteDeadSessions(t0.Add(-time.Hour), t0.Add(-time.Hour), t0, 1)
		if err != nil || n != 1 {
			t.Fatalf("delete boundary fixture: %d, %v", n, err)
		}
		return nil
	}))
	must(t, s.View(ctx, func(tx store.Tx) error {
		rows, err := tx.ListSessions("pa", &store.SessionPosition{CreatedAt: boundary.CreatedAt, PublicID: boundary.PublicID}, 7)
		if err != nil || len(rows) != 7 || rows[0].PublicID != all[7].PublicID {
			t.Fatalf("deleted boundary broke continuation: %+v, %v", rows, err)
		}
		return nil
	}))
}
