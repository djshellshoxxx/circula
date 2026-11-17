// Package storetest is a conformance suite every favorites.Store must pass.
package storetest

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/djshellshoxxx/circula/future/favorites"
)

var t0 = time.Date(2026, 11, 16, 12, 0, 0, 0, time.UTC)

func entry(n, o, tg string, pinned bool, order int) favorites.Entry {
	return favorites.Entry{Favorite: favorites.Favorite{Network: n, Owner: o, Target: tg, Nickname: "nick-" + tg}, Pinned: pinned, Order: order, CreatedAt: t0, UpdatedAt: t0}
}

func must(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
}

// Run executes the suite; newStore must return a fresh empty store.
func Run(t *testing.T, newStore func(t *testing.T) favorites.Store) {
	ctx := context.Background()
	t.Run("crud_scoping_order", func(t *testing.T) {
		s := newStore(t)
		must(t, s.Update(ctx, func(tx favorites.Tx) error {
			for _, e := range []favorites.Entry{
				entry("n1", "a", "z", false, 1), entry("n1", "a", "y", false, 1), entry("n1", "a", "p", true, 9),
				entry("n1", "b", "z", false, 1), entry("n2", "a", "z", false, 1),
			} {
				if err := tx.InsertFavorite(ctx, e); err != nil {
					return err
				}
			}
			if err := tx.InsertFavorite(ctx, entry("n1", "a", "z", false, 5)); !errors.Is(err, favorites.ErrDuplicate) {
				t.Errorf("duplicate: %v", err)
			}
			return nil
		}))
		must(t, s.View(ctx, func(tx favorites.Tx) error {
			l, err := tx.ListFavorites(ctx, "n1", "a", 10)
			if err != nil {
				return err
			}
			var got []string
			for _, e := range l {
				got = append(got, e.Target)
			}
			if len(got) != 3 || got[0] != "p" || got[1] != "y" || got[2] != "z" {
				t.Errorf("order %v", got)
			}
			if l, _ := tx.ListFavorites(ctx, "n1", "a", 2); len(l) != 2 {
				t.Errorf("limit %d", len(l))
			}
			if l, _ := tx.ListFavorites(ctx, "n3", "a", 10); len(l) != 0 {
				t.Error("unknown network")
			}
			e, ok, _ := tx.GetFavorite(ctx, "n1", "b", "z")
			if !ok || e.Owner != "b" || e.Nickname != "nick-z" || !e.CreatedAt.Equal(t0) {
				t.Errorf("get %+v", e)
			}
			if _, ok, _ := tx.GetFavorite(ctx, "n2", "b", "z"); ok {
				t.Error("cross-network get")
			}
			return nil
		}))
		must(t, s.Update(ctx, func(tx favorites.Tx) error {
			u := entry("n1", "a", "z", true, 3)
			u.Nickname = "changed"
			u.UpdatedAt = t0.Add(time.Minute)
			if ok, err := tx.UpdateFavorite(ctx, u); !ok || err != nil {
				t.Errorf("update %v %v", ok, err)
			}
			if ok, _ := tx.UpdateFavorite(ctx, entry("n9", "a", "z", true, 3)); ok {
				t.Error("update matched wrong network")
			}
			e, _, _ := tx.GetFavorite(ctx, "n1", "a", "z")
			if e.Nickname != "changed" || !e.Pinned || e.Order != 3 || !e.UpdatedAt.Equal(t0.Add(time.Minute)) || !e.CreatedAt.Equal(t0) {
				t.Errorf("after update %+v", e)
			}
			o, _, _ := tx.GetFavorite(ctx, "n1", "b", "z")
			if o.Nickname != "nick-z" {
				t.Error("update touched other owner")
			}
			if ok, _ := tx.DeleteFavorite(ctx, "n1", "nobody", "z"); ok {
				t.Error("delete wrong owner")
			}
			if ok, _ := tx.DeleteFavorite(ctx, "n1", "a", "z"); !ok {
				t.Error("delete")
			}
			if ok, _ := tx.DeleteFavorite(ctx, "n1", "a", "z"); ok {
				t.Error("second delete")
			}
			return nil
		}))
	})

	t.Run("rollback_and_readonly", func(t *testing.T) {
		s := newStore(t)
		boom := errors.New("boom")
		err := s.Update(ctx, func(tx favorites.Tx) error {
			must(t, tx.InsertFavorite(ctx, entry("n1", "a", "x", false, 1)))
			_, err := tx.BumpRevision(ctx, "n1", "a")
			must(t, err)
			return boom
		})
		if !errors.Is(err, boom) {
			t.Fatal(err)
		}
		func() {
			defer func() { _ = recover() }()
			_ = s.Update(ctx, func(tx favorites.Tx) error {
				must(t, tx.InsertFavorite(ctx, entry("n1", "a", "p", false, 1)))
				panic("mid-transaction panic")
			})
		}()
		must(t, s.View(ctx, func(tx favorites.Tx) error {
			l, _ := tx.ListFavorites(ctx, "n1", "a", 10)
			r, _ := tx.Revision(ctx, "n1", "a")
			if len(l) != 0 || r != 0 {
				t.Errorf("rollback incomplete: %d rows, revision %d", len(l), r)
			}
			if err := tx.InsertFavorite(ctx, entry("n1", "a", "w", false, 1)); err == nil {
				t.Error("write in View must fail")
			}
			if _, err := tx.BumpRevision(ctx, "n1", "a"); err == nil {
				t.Error("bump in View must fail")
			}
			return nil
		}))
		// the store is still usable after a panic rolled back
		must(t, s.Update(ctx, func(tx favorites.Tx) error { return tx.InsertFavorite(ctx, entry("n1", "a", "ok", false, 1)) }))
		cctx, cancel := context.WithCancel(ctx)
		cancel()
		if err := s.Update(cctx, func(favorites.Tx) error { return nil }); err == nil {
			t.Error("cancelled context must fail")
		}
	})

	t.Run("revisions", func(t *testing.T) {
		s := newStore(t)
		must(t, s.Update(ctx, func(tx favorites.Tx) error {
			if r, _ := tx.Revision(ctx, "n", "o"); r != 0 {
				t.Error("initial")
			}
			for want := int64(1); want <= 3; want++ {
				if r, err := tx.BumpRevision(ctx, "n", "o"); err != nil || r != want {
					t.Errorf("bump %d %v", r, err)
				}
			}
			if r, _ := tx.Revision(ctx, "n", "p"); r != 0 {
				t.Error("owner scoping")
			}
			if r, _ := tx.Revision(ctx, "m", "o"); r != 0 {
				t.Error("network scoping")
			}
			return nil
		}))
	})

	t.Run("intents", func(t *testing.T) {
		s := newStore(t)
		mk := func(o, k, intent string, exp time.Duration) favorites.Intent {
			return favorites.Intent{Network: "n", Owner: o, Key: k, TradeID: "t", Target: "x", Description: "d", Intent: intent, CreatedAt: t0, ExpiresAt: t0.Add(exp)}
		}
		must(t, s.Update(ctx, func(tx favorites.Tx) error {
			must(t, tx.InsertIntent(ctx, mk("a", "k1", "i1", time.Hour)))
			if err := tx.InsertIntent(ctx, mk("a", "k1", "i2", time.Hour)); !errors.Is(err, favorites.ErrDuplicate) {
				t.Errorf("dup key: %v", err)
			}
			if err := tx.InsertIntent(ctx, mk("b", "k1", "i1", time.Hour)); !errors.Is(err, favorites.ErrDuplicate) {
				t.Errorf("dup intent value: %v", err)
			}
			must(t, tx.InsertIntent(ctx, mk("b", "k1", "i3", time.Minute)))
			in, ok, _ := tx.GetIntent(ctx, "n", "b", "k1")
			if !ok || in.Intent != "i3" || in.TradeID != "t" || in.Description != "d" || !in.ExpiresAt.Equal(t0.Add(time.Minute)) {
				t.Errorf("get %+v", in)
			}
			if _, ok, _ := tx.GetIntent(ctx, "n", "c", "k1"); ok {
				t.Error("owner scoping")
			}
			if n, _ := tx.CountIntents(ctx, "n", "b", t0); n != 1 {
				t.Errorf("count %d", n)
			}
			if n, _ := tx.CountIntents(ctx, "n", "b", t0.Add(time.Minute)); n != 0 {
				t.Errorf("expired counted %d", n)
			}
			if n, err := tx.PurgeExpiredIntents(ctx, t0.Add(time.Minute)); err != nil || n != 1 {
				t.Errorf("purge expired %d %v", n, err)
			}
			must(t, tx.DeleteIntent(ctx, "n", "a", "k1"))
			if _, ok, _ := tx.GetIntent(ctx, "n", "a", "k1"); ok {
				t.Error("delete")
			}
			return nil
		}))
	})

	t.Run("purge_member", func(t *testing.T) {
		s := newStore(t)
		must(t, s.Update(ctx, func(tx favorites.Tx) error {
			for _, e := range []favorites.Entry{
				entry("n", "a", "d", false, 1), entry("n", "b", "d", false, 1), entry("n", "d", "a", false, 1),
				entry("n", "a", "k", false, 2), entry("m", "a", "d", false, 1),
			} {
				must(t, tx.InsertFavorite(ctx, e))
			}
			must(t, tx.InsertIntent(ctx, favorites.Intent{Network: "n", Owner: "a", Key: "k", Target: "d", Intent: "i1", CreatedAt: t0, ExpiresAt: t0.Add(time.Hour)}))
			must(t, tx.InsertIntent(ctx, favorites.Intent{Network: "n", Owner: "d", Key: "k", Target: "a", Intent: "i2", CreatedAt: t0, ExpiresAt: t0.Add(time.Hour)}))
			must(t, tx.InsertIntent(ctx, favorites.Intent{Network: "n", Owner: "a", Key: "k2", Target: "k", Intent: "i3", CreatedAt: t0, ExpiresAt: t0.Add(time.Hour)}))
			_, err := tx.BumpRevision(ctx, "n", "d")
			return err
		}))
		var res favorites.PurgeResult
		must(t, s.Update(ctx, func(tx favorites.Tx) (err error) {
			res, err = tx.PurgeMember(ctx, "n", "d")
			return err
		}))
		if res.Favorites != 3 || res.Intents != 2 || res.States != 1 {
			t.Fatalf("%+v", res)
		}
		must(t, s.View(ctx, func(tx favorites.Tx) error {
			l, _ := tx.ListFavorites(ctx, "n", "a", 10)
			if len(l) != 1 || l[0].Target != "k" {
				t.Errorf("a's remaining %v", l)
			}
			if l, _ := tx.ListFavorites(ctx, "m", "a", 10); len(l) != 1 {
				t.Error("other network purged")
			}
			if r, _ := tx.Revision(ctx, "n", "a"); r != 1 {
				t.Errorf("affected owner revision %d", r)
			}
			if r, _ := tx.Revision(ctx, "n", "b"); r != 1 {
				t.Errorf("affected owner revision %d", r)
			}
			if _, ok, _ := tx.GetIntent(ctx, "n", "a", "k2"); !ok {
				t.Error("unrelated intent purged")
			}
			return nil
		}))
	})
}
