package favorites_test

import (
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/djshellshoxxx/circula/future/favorites"
)

func addTrades(fx *fixture) {
	fx.dir.AddTrade("t1", favorites.HistoricalTrade{Network: "n1", Payer: "alice", Payee: "bob", Description: "bike repair", Completed: true})
	fx.dir.AddTrade("t2", favorites.HistoricalTrade{Network: "n1", Payer: "alice", Payee: "carol", Description: "soup", Completed: true})
	fx.dir.AddTrade("t3", favorites.HistoricalTrade{Network: "n1", Payer: "bob", Payee: "alice", Description: "reverse", Completed: true})
	fx.dir.AddTrade("t4", favorites.HistoricalTrade{Network: "n1", Payer: "alice", Payee: "bob", Description: "pending", Completed: false})
	fx.dir.AddTrade("t5", favorites.HistoricalTrade{Network: "n1", Payer: "carol", Payee: "dave", Description: "not alice", Completed: true})
	fx.dir.AddTrade("t6", favorites.HistoricalTrade{Network: "n1", Payer: "alice", Payee: "bob", Description: "bad‮text", Completed: true})
}

func TestTwoRepeatsAreTwoIndependentIntents(t *testing.T) {
	eachStore(t, func(t *testing.T, f storeFactory) {
		fx := newFixture(t, f)
		addTrades(fx)
		a, err := fx.svc.RepeatTrade(ctx, alice, "t1", "page-render-1")
		if err != nil {
			t.Fatal(err)
		}
		b, err := fx.svc.RepeatTrade(ctx, alice, "t1", "page-render-2")
		if err != nil {
			t.Fatal(err)
		}
		c, err := fx.svc.RepeatTrade(ctx, alice, "t1", "") // no key: always fresh
		if err != nil {
			t.Fatal(err)
		}
		if a.Intent == b.Intent || a.Intent == c.Intent || b.Intent == c.Intent || a.Intent == "" {
			t.Fatal("repeats must carry distinct idempotency intents")
		}
		if a.Amount != nil || b.Amount != nil || c.Amount != nil {
			t.Fatal("amount must stay blank")
		}
		if a.Recipient != "bob" || a.Description != "bike repair" || a.Network != "n1" || a.TradeID != "t1" || a.RequestKey != "page-render-1" {
			t.Fatalf("%+v", a)
		}
		if a.Intent == "t1" || len(a.Intent) != 32 {
			t.Fatalf("intent must be fresh random, got %q", a.Intent)
		}
		// same deliberate action retried: same draft and key
		a2, err := fx.svc.RepeatTrade(ctx, alice, "t1", "page-render-1")
		if err != nil || a2 != a {
			t.Fatalf("retry: %+v %v", a2, err)
		}
		g, err := fx.svc.GetDraft(ctx, alice, "page-render-1")
		if err != nil || g != a {
			t.Fatalf("get: %+v %v", g, err)
		}
		// key reuse for another trade
		if _, err := fx.svc.RepeatTrade(ctx, alice, "t2", "page-render-1"); !errors.Is(err, favorites.ErrKeyReuse) {
			t.Fatalf("key reuse: %v", err)
		}
	})
}

func TestRepeatRefusals(t *testing.T) {
	eachStore(t, func(t *testing.T, f storeFactory) {
		fx := newFixture(t, f)
		addTrades(fx)
		for _, id := range []string{"t3", "t4", "t5", "t6", "missing", "", "bad id"} {
			if _, err := fx.svc.RepeatTrade(ctx, alice, id, ""); !errors.Is(err, favorites.ErrTradeUnavailable) {
				t.Errorf("trade %q: %v", id, err)
			}
		}
		if _, err := fx.svc.RepeatTrade(ctx, alice, "t1", "bad key!"); err == nil {
			t.Error("bad key accepted")
		}
		// someone else's trade is indistinguishable from a missing one
		if _, err := fx.svc.RepeatTrade(ctx, favorites.Actor{Network: "n1", Member: "dave"}, "t1", ""); !errors.Is(err, favorites.ErrTradeUnavailable) {
			t.Errorf("foreign trade: %v", err)
		}
		// cross-network
		if _, err := fx.svc.RepeatTrade(ctx, favorites.Actor{Network: "n2", Member: "alice"}, "t1", ""); !errors.Is(err, favorites.ErrTradeUnavailable) {
			t.Errorf("cross network: %v", err)
		}
		// blocked / suspended / departed / hidden partner cannot be paid again
		m := fx.dir.Get("n1", "bob")
		m.Blocked = true
		fx.dir.Set(m)
		if _, err := fx.svc.RepeatTrade(ctx, alice, "t1", ""); !errors.Is(err, favorites.ErrUnavailable) {
			t.Errorf("suspended: %v", err)
		}
		fx.dir.Remove("n1", "bob")
		if _, err := fx.svc.RepeatTrade(ctx, alice, "t1", ""); !errors.Is(err, favorites.ErrUnavailable) {
			t.Errorf("departed: %v", err)
		}
		// no trade source configured
		nofx := newFixture(t, f, func(c *favorites.Config) { c.Trades = nil })
		if _, err := nofx.svc.RepeatTrade(ctx, alice, "t1", ""); !errors.Is(err, favorites.ErrTradeUnavailable) {
			t.Errorf("no source: %v", err)
		}
	})
}

func TestDraftsExpireAndAreCapped(t *testing.T) {
	eachStore(t, func(t *testing.T, f storeFactory) {
		fx := newFixture(t, f, func(c *favorites.Config) { c.MaxDrafts = 2; c.DraftTTL = time.Hour })
		addTrades(fx)
		d1, _ := fx.svc.RepeatTrade(ctx, alice, "t1", "k1")
		if _, err := fx.svc.RepeatTrade(ctx, alice, "t1", "k2"); err != nil {
			t.Fatal(err)
		}
		if _, err := fx.svc.RepeatTrade(ctx, alice, "t1", "k3"); !errors.Is(err, favorites.ErrLimit) {
			t.Fatalf("cap: %v", err)
		}
		// another owner has an independent cap and namespace
		if _, err := fx.svc.RepeatTrade(ctx, favorites.Actor{Network: "n1", Member: "carol"}, "t5", "k1"); err != nil {
			t.Fatal(err)
		}
		fx.clk.Add(2 * time.Hour)
		if _, err := fx.svc.GetDraft(ctx, alice, "k1"); !errors.Is(err, favorites.ErrNotFound) {
			t.Fatalf("expired: %v", err)
		}
		d1b, err := fx.svc.RepeatTrade(ctx, alice, "t1", "k1")
		if err != nil || d1b.Intent == d1.Intent {
			t.Fatalf("expired key must mint a new intent: %+v %v", d1b, err)
		}
		n, err := fx.svc.PurgeExpiredDrafts(ctx)
		if err != nil || n < 2 {
			t.Fatalf("purge %d %v", n, err)
		}
	})
}

func TestGetDraftScopedAndRevalidated(t *testing.T) {
	eachStore(t, func(t *testing.T, f storeFactory) {
		fx := newFixture(t, f)
		addTrades(fx)
		if _, err := fx.svc.RepeatTrade(ctx, alice, "t1", "mine"); err != nil {
			t.Fatal(err)
		}
		if _, err := fx.svc.GetDraft(ctx, bob, "mine"); !errors.Is(err, favorites.ErrNotFound) {
			t.Fatalf("other owner: %v", err)
		}
		if _, err := fx.svc.GetDraft(ctx, favorites.Actor{Network: "n2", Member: "alice"}, "mine"); !errors.Is(err, favorites.ErrNotFound) {
			t.Fatalf("other network: %v", err)
		}
		if _, err := fx.svc.GetDraft(ctx, alice, "../etc"); !errors.Is(err, favorites.ErrNotFound) {
			t.Fatalf("bad key: %v", err)
		}
		fx.dir.Block("n1", "alice", "bob")
		if _, err := fx.svc.GetDraft(ctx, alice, "mine"); !errors.Is(err, favorites.ErrUnavailable) {
			t.Fatalf("blocked after draft: %v", err)
		}
	})
}

func TestDraftFromFavorite(t *testing.T) {
	eachStore(t, func(t *testing.T, f storeFactory) {
		fx := newFixture(t, f)
		if _, err := fx.svc.DraftFromFavorite(ctx, alice, "bob", "k"); !errors.Is(err, favorites.ErrNotFound) {
			t.Fatalf("not a favorite: %v", err)
		}
		mustAdd(t, fx.svc, alice, "bob", "")
		d, err := fx.svc.DraftFromFavorite(ctx, alice, "bob", "k")
		if err != nil || d.Recipient != "bob" || d.Description != "" || d.Amount != nil || d.TradeID != "" {
			t.Fatalf("%+v %v", d, err)
		}
		d2, _ := fx.svc.DraftFromFavorite(ctx, alice, "bob", "k")
		if d2 != d {
			t.Fatal("retry differs")
		}
		d3, _ := fx.svc.DraftFromFavorite(ctx, alice, "bob", "k2")
		if d3.Intent == d.Intent {
			t.Fatal("independent intents")
		}
		fx.dir.Block("n1", "alice", "bob")
		if _, err := fx.svc.DraftFromFavorite(ctx, alice, "bob", "k9"); !errors.Is(err, favorites.ErrUnavailable) {
			t.Fatalf("blocked: %v", err)
		}
	})
}

func TestConcurrentSameKeyYieldsOneIntent(t *testing.T) {
	eachStore(t, func(t *testing.T, f storeFactory) {
		fx := newFixture(t, f)
		addTrades(fx)
		var wg sync.WaitGroup
		out := make([]favorites.Draft, 12)
		for i := range out {
			wg.Add(1)
			go func(i int) {
				defer wg.Done()
				d, err := fx.svc.RepeatTrade(ctx, alice, "t1", "same")
				if err != nil {
					t.Errorf("%v", err)
				}
				out[i] = d
			}(i)
		}
		wg.Wait()
		for _, d := range out {
			if d != out[0] || d.Intent == "" {
				t.Fatalf("diverged: %+v vs %+v", d, out[0])
			}
		}
	})
}
