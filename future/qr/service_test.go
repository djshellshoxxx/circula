package qr_test

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	qr "github.com/djshellshoxxx/circula/future/qr"
	"github.com/djshellshoxxx/circula/future/qr/qrfixture"
)

var bg = context.Background()

func TestStaticAliasConfirmedTradeIsTwoBalancedEntries(t *testing.T) {
	e := newEnv(t)
	alias, err := e.svc.MyAlias(bg, p("n1", "a"))
	must(t, err)
	if alias.Generation != 1 || !strings.HasPrefix(alias.URL, origin+"/qr/v1/a/") || strings.ContainsAny(alias.URL, "?#&=") {
		t.Fatalf("alias %+v", alias)
	}
	again, _ := e.svc.MyAlias(bg, p("n1", "a"))
	if again.Code != alias.Code {
		t.Fatal("alias not stable")
	}
	pv, err := e.svc.ResolveAlias(bg, p("n1", "b"), alias.Code)
	must(t, err)
	if pv.PayeeName != "Alice" || pv.PayerName != "Bob" || pv.NetworkName != "Alpha" || pv.AmountMinor != 0 || pv.AvailableCredit != 1000 || pv.Kind != qr.KindAlias {
		t.Fatalf("preview %+v", pv)
	}
	res, err := e.svc.ProposeToAlias(bg, p("n1", "b"), alias.Code, 250, "coffee", "key-00000001")
	must(t, err)
	if res.State != qr.TxProposed {
		t.Fatalf("state %v", res.State)
	}
	if c := e.counts(); c.Proposed != 1 || c.Completed != 0 || c.Entries != 0 {
		t.Fatalf("scan alone must not post: %+v", c)
	}
	must(t, e.accept("n1", res.TransactionID, "a"))
	c := e.counts()
	if c.Completed != 1 || c.Entries != 2 || c.EntrySum != 0 || c.Proposed != 0 {
		t.Fatalf("%+v", c)
	}
	ba, _ := qrfixture.Balance(bg, e.db, "n1", "a")
	bb, _ := qrfixture.Balance(bg, e.db, "n1", "b")
	if ba != 250 || bb != -250 {
		t.Fatal(ba, bb)
	}
}

func TestAliasRetryAndConflict(t *testing.T) {
	e := newEnv(t)
	al, _ := e.svc.MyAlias(bg, p("n1", "a"))
	r1, err := e.svc.ProposeToAlias(bg, p("n1", "b"), al.Code, 10, "x", "k-alias-0001")
	must(t, err)
	r2, err := e.svc.ProposeToAlias(bg, p("n1", "b"), al.Code, 10, "x", "k-alias-0001")
	must(t, err)
	if !r2.Replayed || r2.TransactionID != r1.TransactionID || e.counts().Proposed != 1 {
		t.Fatalf("%+v %+v %+v", r1, r2, e.counts())
	}
	for _, c := range []struct {
		amt  int64
		note string
	}{{11, "x"}, {10, "y"}} {
		if _, err := e.svc.ProposeToAlias(bg, p("n1", "b"), al.Code, c.amt, c.note, "k-alias-0001"); !errors.Is(err, qr.ErrIdempotencyConflict) {
			t.Fatalf("%v", err)
		}
	}
	if e.counts().Proposed != 1 {
		t.Fatal("duplicate")
	}
	// A different payer may use the same key text independently.
	if _, err := e.svc.ProposeToAlias(bg, p("n1", "c"), al.Code, 10, "x", "k-alias-0001"); err != nil {
		t.Fatal(err)
	}
}

func TestAliasRotationRevokesOldAndIsIdempotent(t *testing.T) {
	e := newEnv(t)
	old, _ := e.svc.MyAlias(bg, p("n1", "a"))
	r1, err := e.svc.RotateAlias(bg, p("n1", "a"), "rot-key-0001")
	must(t, err)
	r2, err := e.svc.RotateAlias(bg, p("n1", "a"), "rot-key-0001")
	must(t, err)
	if r1.Generation != 2 || r2.Generation != 2 || r1.Code != r2.Code || r1.Code == old.Code {
		t.Fatalf("%+v %+v", r1, r2)
	}
	cur, _ := e.svc.MyAlias(bg, p("n1", "a"))
	if cur.Code != r1.Code {
		t.Fatal("live alias is not newest")
	}
	if _, err := e.svc.ResolveAlias(bg, p("n1", "b"), old.Code); !errors.Is(err, qr.ErrRevoked) {
		t.Fatalf("old alias must be revoked, got %v", err)
	}
	if _, err := e.svc.ProposeToAlias(bg, p("n1", "b"), old.Code, 5, "", "k-old-000001"); !errors.Is(err, qr.ErrRevoked) {
		t.Fatalf("%v", err)
	}
	if _, err := e.svc.ResolveAlias(bg, p("n1", "b"), r1.Code); err != nil {
		t.Fatal(err)
	}
	// Retrying the first key after a later rotation still reports generation 2 (now revoked).
	r3, _ := e.svc.RotateAlias(bg, p("n1", "a"), "rot-key-0002")
	r4, _ := e.svc.RotateAlias(bg, p("n1", "a"), "rot-key-0001")
	if r3.Generation != 3 || r4.Generation != 2 || !r4.Revoked {
		t.Fatalf("%+v %+v", r3, r4)
	}
	if n := e.scalar(`SELECT COUNT(*) FROM qr_member_alias WHERE network_id='n1' AND member_id='a' AND revoked_at IS NULL`); n != 1 {
		t.Fatal(n)
	}
}

func TestAliasRotationLimit(t *testing.T) {
	e := newEnv(t, func(c *qr.Config) { c.MaxRotations = 3 })
	for i, k := range []string{"rot-aaaa0001", "rot-aaaa0002", "rot-aaaa0003"} {
		if _, err := e.svc.RotateAlias(bg, p("n1", "a"), k); err != nil {
			t.Fatal(i, err)
		}
	}
	if _, err := e.svc.RotateAlias(bg, p("n1", "a"), "rot-aaaa0004"); !errors.Is(err, qr.ErrRateLimited) {
		t.Fatal(err)
	}
}

func TestAliasTenantIsolation(t *testing.T) {
	e := newEnv(t)
	al, _ := e.svc.MyAlias(bg, p("n1", "a"))
	if _, err := e.svc.ResolveAlias(bg, p("n2", "x"), al.Code); !errors.Is(err, qr.ErrNotFound) {
		t.Fatalf("cross-network alias resolve: %v", err)
	}
	if _, err := e.svc.ProposeToAlias(bg, p("n2", "x"), al.Code, 5, "", "k-cross-0001"); !errors.Is(err, qr.ErrNotFound) {
		t.Fatalf("%v", err)
	}
	// An unrelated, well-formed code is the same not-found outcome.
	tok, _ := qr.NewToken()
	if _, err := e.svc.ResolveAlias(bg, p("n1", "b"), tok); !errors.Is(err, qr.ErrNotFound) {
		t.Fatal(err)
	}
	if _, err := e.svc.ResolveAlias(bg, p("n1", "b"), "garbage"); !errors.Is(err, qr.ErrNotFound) {
		t.Fatal(err)
	}
	if e.counts().Proposed != 0 {
		t.Fatal("proposal created")
	}
}

func TestAliasSelfAndInactive(t *testing.T) {
	e := newEnv(t)
	al, _ := e.svc.MyAlias(bg, p("n1", "a"))
	if _, err := e.svc.ProposeToAlias(bg, p("n1", "a"), al.Code, 5, "", "k-self-00001"); !errors.Is(err, qr.ErrInvalid) {
		t.Fatal(err)
	}
	must(t, qrfixture.SetMember(bg, e.db, "n1", "a", false, true, 1000))
	if _, err := e.svc.ResolveAlias(bg, p("n1", "b"), al.Code); !errors.Is(err, qr.ErrCounterpartyInactive) {
		t.Fatal(err)
	}
	if _, err := e.svc.ProposeToAlias(bg, p("n1", "b"), al.Code, 5, "", "k-inact-0001"); !errors.Is(err, qr.ErrCounterpartyInactive) {
		t.Fatal(err)
	}
	if _, err := e.svc.MyAlias(bg, p("n1", "a")); !errors.Is(err, qr.ErrForbidden) {
		t.Fatal("suspended member must not use their alias", err)
	}
	must(t, qrfixture.SetMember(bg, e.db, "n1", "b", false, true, 1000))
	must(t, qrfixture.SetMember(bg, e.db, "n1", "a", true, true, 1000))
	if _, err := e.svc.ProposeToAlias(bg, p("n1", "b"), al.Code, 5, "", "k-susp-00001"); !errors.Is(err, qr.ErrForbidden) {
		t.Fatalf("suspended payer: %v", err)
	}
}

func TestOneTimeRequestFullLifecycle(t *testing.T) {
	e := newEnv(t)
	cr, err := e.svc.CreateRequest(bg, p("n1", "a"), "create-0001", qr.CreateRequestInput{AmountMinor: 300, Note: "invoice 7"})
	must(t, err)
	if len(cr.Token) != 22 || cr.URL != origin+"/qr/v1/"+cr.Token || cr.View.Status != qr.StatusOpen || cr.View.AmountMinor != 300 {
		t.Fatalf("%+v", cr)
	}
	pv, err := e.svc.PreviewRequest(bg, p("n1", "b"), cr.Token)
	must(t, err)
	if pv.AmountMinor != 300 || pv.PayeeName != "Alice" || pv.PayerName != "Bob" || pv.NetworkName != "Alpha" || pv.AvailableCredit != 1000 || !pv.Sufficient || pv.Note != "invoice 7" {
		t.Fatalf("%+v", pv)
	}
	if e.counts().Proposed != 0 {
		t.Fatal("preview must not claim")
	}
	res, err := e.svc.ClaimRequest(bg, p("n1", "b"), cr.Token, "claim-key-01")
	must(t, err)
	v, _ := e.svc.GetRequest(bg, p("n1", "a"), cr.View.ID)
	if v.Status != qr.StatusPending || !v.Claimed || v.TransactionID != res.TransactionID {
		t.Fatalf("recipient must see pending, got %+v", v)
	}
	if c := e.counts(); c.Completed != 0 || c.Entries != 0 {
		t.Fatalf("%+v", c)
	}
	must(t, e.accept("n1", res.TransactionID, "a"))
	v, _ = e.svc.GetRequest(bg, p("n1", "a"), cr.View.ID)
	if v.Status != qr.StatusCompleted {
		t.Fatalf("%+v", v)
	}
	if c := e.counts(); c.Completed != 1 || c.Entries != 2 || c.EntrySum != 0 || c.Proposed != 0 || c.Audit != 2 {
		t.Fatalf("%+v", c)
	}
	// Payer also sees it; completion keeps it claimed (a token cannot be reused).
	if v2, err := e.svc.GetRequest(bg, p("n1", "b"), cr.View.ID); err != nil || v2.Status != qr.StatusCompleted {
		t.Fatal(v2, err)
	}
	if _, err := e.svc.ClaimRequest(bg, p("n1", "c"), cr.Token, "claim-key-02"); !errors.Is(err, qr.ErrClaimed) {
		t.Fatal(err)
	}
	// Third parties cannot read the request record.
	if _, err := e.svc.GetRequest(bg, p("n1", "c"), cr.View.ID); !errors.Is(err, qr.ErrNotFound) {
		t.Fatal(err)
	}
	if _, err := e.svc.GetRequest(bg, p("n2", "x"), cr.View.ID); !errors.Is(err, qr.ErrNotFound) {
		t.Fatal(err)
	}
}

func TestClaimRetriesAndIdempotency(t *testing.T) {
	e := newEnv(t)
	cr, _ := e.svc.CreateRequest(bg, p("n1", "a"), "create-0001", qr.CreateRequestInput{AmountMinor: 50})
	r1, err := e.svc.ClaimRequest(bg, p("n1", "b"), cr.Token, "claim-key-01")
	must(t, err)
	r2, err := e.svc.ClaimRequest(bg, p("n1", "b"), cr.Token, "claim-key-01")
	must(t, err)
	r3, err := e.svc.ClaimRequest(bg, p("n1", "b"), cr.Token, "another-key-1") // same intent, new key
	must(t, err)
	r4, err := e.svc.ClaimRequest(bg, p("n1", "b"), cr.Token, "") // derived key
	must(t, err)
	for _, r := range []qr.ClaimResult{r2, r3, r4} {
		if r.TransactionID != r1.TransactionID || !r.Replayed {
			t.Fatalf("%+v vs %+v", r, r1)
		}
	}
	if r1.Replayed || e.counts().Proposed != 1 {
		t.Fatal("duplicate posting")
	}
	// Same key against a different request is a conflict.
	cr2, _ := e.svc.CreateRequest(bg, p("n1", "a"), "create-0002", qr.CreateRequestInput{AmountMinor: 51})
	if _, err := e.svc.ClaimRequest(bg, p("n1", "b"), cr2.Token, "claim-key-01"); !errors.Is(err, qr.ErrIdempotencyConflict) {
		t.Fatal(err)
	}
	// Another payer with the first request: claimed.
	if _, err := e.svc.ClaimRequest(bg, p("n1", "c"), cr.Token, "claim-key-01"); !errors.Is(err, qr.ErrClaimed) {
		t.Fatal(err)
	}
	// Derived key is stable per intent.
	d1, err := e.svc.ClaimRequest(bg, p("n1", "c"), cr2.Token, "")
	must(t, err)
	d2, err := e.svc.ClaimRequest(bg, p("n1", "c"), cr2.Token, "")
	must(t, err)
	if d1.TransactionID != d2.TransactionID || !d2.Replayed || e.counts().Proposed != 2 {
		t.Fatal("derived key unstable")
	}
}

func TestReplayAfterCompletionAndAfterExpiry(t *testing.T) {
	e := newEnv(t)
	cr, _ := e.svc.CreateRequest(bg, p("n1", "a"), "create-0001", qr.CreateRequestInput{AmountMinor: 50, TTL: time.Minute})
	r1, _ := e.svc.ClaimRequest(bg, p("n1", "b"), cr.Token, "k-replay-001")
	must(t, e.accept("n1", r1.TransactionID, "a"))
	e.clk.Add(time.Hour) // far past expiry
	r2, err := e.svc.ClaimRequest(bg, p("n1", "b"), cr.Token, "k-replay-001")
	must(t, err)
	if r2.TransactionID != r1.TransactionID || r2.State != qr.TxCompleted || !r2.Replayed {
		t.Fatalf("%+v", r2)
	}
	if e.counts().Completed != 1 || e.counts().Entries != 2 {
		t.Fatal("double posting")
	}
}

func TestCreateRequestValidationAndLimits(t *testing.T) {
	e := newEnv(t, func(c *qr.Config) { c.MaxActiveRequests = 2; c.MaxNoteRunes = 10 })
	bad := []qr.CreateRequestInput{
		{AmountMinor: 0}, {AmountMinor: -5}, {AmountMinor: 1_000_000_000_001},
		{AmountMinor: 5, Note: "01234567890"},
		{AmountMinor: 5, Note: "bad\x00"}, {AmountMinor: 5, Note: "line\nbreak"}, {AmountMinor: 5, Note: "rtl‮over"}, {AmountMinor: 5, Note: "\xff\xfe"},
		{AmountMinor: 5, TTL: time.Second}, {AmountMinor: 5, TTL: 8 * 24 * time.Hour}, {AmountMinor: 5, TTL: -time.Hour},
	}
	for i, in := range bad {
		if _, err := e.svc.CreateRequest(bg, p("n1", "a"), "bad-key-000"+string(rune('a'+i)), in); !errors.Is(err, qr.ErrInvalid) {
			t.Fatalf("%d %+v: %v", i, in, err)
		}
	}
	if _, err := e.svc.CreateRequest(bg, p("n1", "a"), "has space", qr.CreateRequestInput{AmountMinor: 5}); !errors.Is(err, qr.ErrInvalid) {
		t.Fatal(err)
	}
	if _, err := e.svc.CreateRequest(bg, p("n1", "a"), "", qr.CreateRequestInput{AmountMinor: 5}); !errors.Is(err, qr.ErrInvalid) {
		t.Fatal(err)
	}
	if e.scalar(`SELECT COUNT(*) FROM qr_trade_request`) != 0 {
		t.Fatal("rejected input persisted")
	}
	c1, err := e.svc.CreateRequest(bg, p("n1", "a"), "ok-key-0001", qr.CreateRequestInput{AmountMinor: 5, Note: "  padded  "})
	must(t, err)
	if c1.View.Note != "padded" {
		t.Fatal(c1.View.Note)
	}
	_, err = e.svc.CreateRequest(bg, p("n1", "a"), "ok-key-0002", qr.CreateRequestInput{AmountMinor: 5})
	must(t, err)
	if _, err := e.svc.CreateRequest(bg, p("n1", "a"), "ok-key-0003", qr.CreateRequestInput{AmountMinor: 5}); !errors.Is(err, qr.ErrRateLimited) {
		t.Fatal(err)
	}
	// Quantity limit is per member; revoking frees a slot.
	if _, err := e.svc.CreateRequest(bg, p("n1", "b"), "ok-key-0001", qr.CreateRequestInput{AmountMinor: 5}); err != nil {
		t.Fatal(err)
	}
	_, err = e.svc.RevokeRequest(bg, p("n1", "a"), c1.View.ID)
	must(t, err)
	if _, err := e.svc.CreateRequest(bg, p("n1", "a"), "ok-key-0003", qr.CreateRequestInput{AmountMinor: 5}); err != nil {
		t.Fatal(err)
	}
}

func TestCreateRequestIdempotency(t *testing.T) {
	e := newEnv(t)
	in := qr.CreateRequestInput{AmountMinor: 77, Note: "n", TTL: time.Hour}
	c1, err := e.svc.CreateRequest(bg, p("n1", "a"), "create-same1", in)
	must(t, err)
	c2, err := e.svc.CreateRequest(bg, p("n1", "a"), "create-same1", in)
	must(t, err)
	if !c2.Replayed || c2.View.ID != c1.View.ID || c2.Token != "" || c2.URL != "" || c1.Token == "" {
		t.Fatalf("%+v %+v", c1, c2)
	}
	if e.scalar(`SELECT COUNT(*) FROM qr_trade_request`) != 1 {
		t.Fatal("duplicate request")
	}
	in.AmountMinor = 78
	if _, err := e.svc.CreateRequest(bg, p("n1", "a"), "create-same1", in); !errors.Is(err, qr.ErrIdempotencyConflict) {
		t.Fatal(err)
	}
}

func TestExpiryBoundaryIsExact(t *testing.T) {
	e := newEnv(t)
	cr, _ := e.svc.CreateRequest(bg, p("n1", "a"), "create-0001", qr.CreateRequestInput{AmountMinor: 10, TTL: time.Minute})
	e.clk.Add(time.Minute - time.Millisecond)
	if _, err := e.svc.PreviewRequest(bg, p("n1", "b"), cr.Token); err != nil {
		t.Fatalf("1ms before expiry must be valid: %v", err)
	}
	e.clk.Add(time.Millisecond) // now == expires_at
	if _, err := e.svc.PreviewRequest(bg, p("n1", "b"), cr.Token); !errors.Is(err, qr.ErrExpired) {
		t.Fatalf("at expiry: %v", err)
	}
	if _, err := e.svc.ClaimRequest(bg, p("n1", "b"), cr.Token, "k-exp-000001"); !errors.Is(err, qr.ErrExpired) {
		t.Fatal(err)
	}
	if e.counts().Proposed != 0 || e.scalar(`SELECT COUNT(*) FROM qr_trade_request WHERE claimed_tx_id IS NOT NULL`) != 0 {
		t.Fatal("expired request produced a claim")
	}
	v, _ := e.svc.GetRequest(bg, p("n1", "a"), cr.View.ID)
	if v.Status != qr.StatusExpired {
		t.Fatal(v.Status)
	}
}

func TestRevocation(t *testing.T) {
	e := newEnv(t)
	cr, _ := e.svc.CreateRequest(bg, p("n1", "a"), "create-0001", qr.CreateRequestInput{AmountMinor: 10})
	// Only the recipient may revoke; others see not-found (no existence leak).
	for _, who := range []qr.Principal{p("n1", "b"), p("n2", "x")} {
		if _, err := e.svc.RevokeRequest(bg, who, cr.View.ID); !errors.Is(err, qr.ErrNotFound) {
			t.Fatalf("%v: %v", who, err)
		}
	}
	v, err := e.svc.RevokeRequest(bg, p("n1", "a"), cr.View.ID)
	must(t, err)
	if v.Status != qr.StatusRevoked || v.RevokedAt == nil {
		t.Fatalf("%+v", v)
	}
	v2, err := e.svc.RevokeRequest(bg, p("n1", "a"), cr.View.ID)
	must(t, err)
	if !v2.RevokedAt.Equal(*v.RevokedAt) {
		t.Fatal("revoke not idempotent")
	}
	if _, err := e.svc.PreviewRequest(bg, p("n1", "b"), cr.Token); !errors.Is(err, qr.ErrRevoked) {
		t.Fatal(err)
	}
	if _, err := e.svc.ClaimRequest(bg, p("n1", "b"), cr.Token, "k-revoke-001"); !errors.Is(err, qr.ErrRevoked) {
		t.Fatal(err)
	}
	if _, err := e.svc.RevokeRequest(bg, p("n1", "a"), "nonsense"); !errors.Is(err, qr.ErrNotFound) {
		t.Fatal(err)
	}
	// A claimed request cannot be revoked.
	cr2, _ := e.svc.CreateRequest(bg, p("n1", "a"), "create-0002", qr.CreateRequestInput{AmountMinor: 10})
	_, err = e.svc.ClaimRequest(bg, p("n1", "b"), cr2.Token, "k-revoke-002")
	must(t, err)
	if _, err := e.svc.RevokeRequest(bg, p("n1", "a"), cr2.View.ID); !errors.Is(err, qr.ErrClaimed) {
		t.Fatal(err)
	}
}

func TestFailClosedScenarios(t *testing.T) {
	e := newEnv(t)
	mk := func(key string, amt int64) qr.CreatedRequest {
		c, err := e.svc.CreateRequest(bg, p("n1", "a"), key, qr.CreateRequestInput{AmountMinor: amt})
		must(t, err)
		return c
	}
	cr := mk("create-0001", 100)
	t.Run("cross network", func(t *testing.T) {
		if _, err := e.svc.PreviewRequest(bg, p("n2", "x"), cr.Token); !errors.Is(err, qr.ErrNotFound) {
			t.Fatal(err)
		}
		if _, err := e.svc.ClaimRequest(bg, p("n2", "x"), cr.Token, "k-cross-0001"); !errors.Is(err, qr.ErrNotFound) {
			t.Fatal(err)
		}
	})
	t.Run("unknown and malformed tokens", func(t *testing.T) {
		tok, _ := qr.NewToken()
		for _, bad := range []string{tok, "", "short", cr.Token + "A", cr.Token[:21], strings.ToLower(cr.Token) + "="} {
			if _, err := e.svc.ClaimRequest(bg, p("n1", "b"), bad, "k-unknown-001"); !errors.Is(err, qr.ErrNotFound) {
				t.Fatalf("%q: %v", bad, err)
			}
		}
	})
	t.Run("suspended payer", func(t *testing.T) {
		must(t, qrfixture.SetMember(bg, e.db, "n1", "b", false, true, 1000))
		defer qrfixture.SetMember(bg, e.db, "n1", "b", true, true, 1000)
		if _, err := e.svc.ClaimRequest(bg, p("n1", "b"), cr.Token, "k-susp-00001"); !errors.Is(err, qr.ErrForbidden) {
			t.Fatal(err)
		}
		if _, err := e.svc.PreviewRequest(bg, p("n1", "b"), cr.Token); !errors.Is(err, qr.ErrForbidden) {
			t.Fatal(err)
		}
	})
	t.Run("terms not accepted payer", func(t *testing.T) {
		must(t, qrfixture.SetMember(bg, e.db, "n1", "b", true, false, 1000))
		defer qrfixture.SetMember(bg, e.db, "n1", "b", true, true, 1000)
		if _, err := e.svc.ClaimRequest(bg, p("n1", "b"), cr.Token, "k-terms-0001"); !errors.Is(err, qr.ErrForbidden) {
			t.Fatal(err)
		}
	})
	t.Run("suspended recipient", func(t *testing.T) {
		must(t, qrfixture.SetMember(bg, e.db, "n1", "a", false, true, 1000))
		defer qrfixture.SetMember(bg, e.db, "n1", "a", true, true, 1000)
		if _, err := e.svc.ClaimRequest(bg, p("n1", "b"), cr.Token, "k-recip-0001"); !errors.Is(err, qr.ErrCounterpartyInactive) {
			t.Fatal(err)
		}
		if _, err := e.svc.CreateRequest(bg, p("n1", "a"), "create-susp-1", qr.CreateRequestInput{AmountMinor: 1}); !errors.Is(err, qr.ErrForbidden) {
			t.Fatal(err)
		}
	})
	t.Run("inactive network", func(t *testing.T) {
		must(t, qrfixture.SetNetworkActive(bg, e.db, "n1", false))
		defer qrfixture.SetNetworkActive(bg, e.db, "n1", true)
		if _, err := e.svc.ClaimRequest(bg, p("n1", "b"), cr.Token, "k-net-000001"); !errors.Is(err, qr.ErrForbidden) {
			t.Fatal(err)
		}
	})
	t.Run("self", func(t *testing.T) {
		if _, err := e.svc.ClaimRequest(bg, p("n1", "a"), cr.Token, "k-self-00001"); !errors.Is(err, qr.ErrInvalid) {
			t.Fatal(err)
		}
		pv, err := e.svc.PreviewRequest(bg, p("n1", "a"), cr.Token)
		if err != nil || !pv.Own {
			t.Fatal(pv, err)
		}
	})
	t.Run("insufficient credit does not burn the token", func(t *testing.T) {
		big := mk("create-0002", 5000)
		if _, err := e.svc.ClaimRequest(bg, p("n1", "b"), big.Token, "k-credit-001"); !errors.Is(err, qr.ErrInsufficientCredit) {
			t.Fatal(err)
		}
		if e.scalar(`SELECT COUNT(*) FROM qr_trade_request WHERE id=? AND claimed_tx_id IS NULL`, big.View.ID) != 1 {
			t.Fatal("token burnt by failed proposal")
		}
		pv, _ := e.svc.PreviewRequest(bg, p("n1", "b"), big.Token)
		if pv.Sufficient {
			t.Fatal("estimate must show shortfall")
		}
		must(t, qrfixture.SetMember(bg, e.db, "n1", "b", true, true, 9000))
		if _, err := e.svc.ClaimRequest(bg, p("n1", "b"), big.Token, "k-credit-002"); err != nil {
			t.Fatal(err)
		}
	})
	if e.counts().EntrySum != 0 {
		t.Fatal("ledger imbalance")
	}
}

func TestProposalFailureRollsBackClaimAndProposal(t *testing.T) {
	e := newEnv(t)
	cr, _ := e.svc.CreateRequest(bg, p("n1", "a"), "create-0001", qr.CreateRequestInput{AmountMinor: 100})
	boom := errors.New("simulated failure after proposal rows written")
	e.fx.AfterPropose = func() error { return boom }
	if _, err := e.svc.ClaimRequest(bg, p("n1", "b"), cr.Token, "k-boom-00001"); !errors.Is(err, boom) {
		t.Fatal(err)
	}
	if e.fx.Proposed.Load() != 1 {
		t.Fatal("fault was not exercised after the proposal insert")
	}
	if c := e.counts(); c.Proposed != 0 || c.Audit != 0 {
		t.Fatalf("proposal rows survived rollback: %+v", c)
	}
	if e.scalar(`SELECT COUNT(*) FROM qr_trade_request WHERE claimed_tx_id IS NOT NULL`) != 0 || e.scalar(`SELECT COUNT(*) FROM qr_idempotency WHERE scope='claim.request'`) != 0 {
		t.Fatal("claim state survived rollback")
	}
	e.fx.AfterPropose = nil
	r, err := e.svc.ClaimRequest(bg, p("n1", "b"), cr.Token, "k-boom-00001") // same key works after the failure
	must(t, err)
	if r.Replayed || e.counts().Proposed != 1 {
		t.Fatal("retry after failed claim must create exactly one proposal")
	}
}

func TestCancelledContextPersistsNothing(t *testing.T) {
	e := newEnv(t)
	cr, _ := e.svc.CreateRequest(bg, p("n1", "a"), "create-0001", qr.CreateRequestInput{AmountMinor: 100})
	ctx, cancel := context.WithCancel(bg)
	e.fx.AfterPropose = func() error { cancel(); return nil } // cancelled after the proposal rows, before COMMIT
	_, err := e.svc.ClaimRequest(ctx, p("n1", "b"), cr.Token, "k-cancel-001")
	if err == nil {
		t.Fatal("expected failure")
	}
	e.fx.AfterPropose = nil
	if c := e.counts(); c.Proposed != 0 {
		t.Fatalf("partial commit: %+v", c)
	}
	if e.scalar(`SELECT COUNT(*) FROM qr_trade_request WHERE claimed_tx_id IS NOT NULL`) != 0 {
		t.Fatal("claimed without proposal")
	}
	if _, err := e.svc.ClaimRequest(bg, p("n1", "b"), cr.Token, "k-cancel-001"); err != nil {
		t.Fatal(err)
	}
}

func TestPanicInsideUpdateRollsBack(t *testing.T) {
	e := newEnv(t)
	func() {
		defer func() { _ = recover() }()
		_ = e.store.Update(bg, func(ctx context.Context, q qr.DBTX) error {
			_, _ = q.ExecContext(ctx, `INSERT INTO qr_idempotency VALUES('n','m','alias.rotate','k',zeroblob(32),'1',1)`)
			panic("boom")
		})
	}()
	if e.scalar(`SELECT COUNT(*) FROM qr_idempotency`) != 0 {
		t.Fatal("panic left data")
	}
	// The gate was released: later writes still work.
	if _, err := e.svc.MyAlias(bg, p("n1", "a")); err != nil {
		t.Fatal(err)
	}
}

func TestListRequestsPaginationAndIsolation(t *testing.T) {
	e := newEnv(t, func(c *qr.Config) { c.MaxActiveRequests = 100 })
	var ids []string
	for i := 0; i < 7; i++ {
		e.clk.Add(time.Second)
		c, err := e.svc.CreateRequest(bg, p("n1", "a"), "list-key-000"+string(rune('0'+i)), qr.CreateRequestInput{AmountMinor: int64(i + 1)})
		must(t, err)
		ids = append(ids, c.View.ID)
	}
	_, err := e.svc.CreateRequest(bg, p("n1", "b"), "list-key-b001", qr.CreateRequestInput{AmountMinor: 9})
	must(t, err)
	var got []string
	cur := ""
	for i := 0; i < 10; i++ {
		page, next, err := e.svc.ListRequests(bg, p("n1", "a"), 3, cur)
		must(t, err)
		for _, v := range page {
			if v.RecipientID != "a" {
				t.Fatal("leak")
			}
			got = append(got, v.ID)
		}
		if next == "" {
			break
		}
		cur = next
	}
	if len(got) != 7 {
		t.Fatal(len(got))
	}
	for i := range got { // newest first
		if got[i] != ids[len(ids)-1-i] {
			t.Fatalf("order %d", i)
		}
	}
	for _, bad := range []string{"!!!", "bm9wZQ", "MS5ub3BlCg"} {
		if _, _, err := e.svc.ListRequests(bg, p("n1", "a"), 3, bad); !errors.Is(err, qr.ErrInvalid) {
			t.Fatalf("%q: %v", bad, err)
		}
	}
	page, _, _ := e.svc.ListRequests(bg, p("n2", "x"), 50, "")
	if len(page) != 0 {
		t.Fatal("cross-network list")
	}
}

func TestInvalidPrincipalAndConfig(t *testing.T) {
	e := newEnv(t)
	if _, err := e.svc.MyAlias(bg, qr.Principal{}); !errors.Is(err, qr.ErrForbidden) {
		t.Fatal(err)
	}
	if _, err := e.svc.ClaimRequest(bg, qr.Principal{NetworkID: "n1"}, "x", "k"); !errors.Is(err, qr.ErrForbidden) {
		t.Fatal(err)
	}
	if _, err := e.svc.CreateRequest(bg, p("n1", "ghost"), "create-0001", qr.CreateRequestInput{AmountMinor: 1}); !errors.Is(err, qr.ErrNotFound) {
		t.Fatal("unknown member", err)
	}
	for name, cfg := range map[string]qr.Config{
		"http origin":  {Origin: "http://c.test", AliasSecret: secret},
		"path origin":  {Origin: "https://c.test/x", AliasSecret: secret},
		"short secret": {Origin: "https://c.test", AliasSecret: []byte("short")},
		"png":          {Origin: "https://c.test", AliasSecret: secret, PNGPixels: 10},
	} {
		if _, err := qr.NewService(cfg, e.store, e.fx); err == nil {
			t.Fatal(name)
		}
	}
	if _, err := qr.NewService(e.cfg, nil, e.fx); err == nil {
		t.Fatal("nil store")
	}
}
