package feed

import (
	"strings"
	"testing"
	"time"
)

// The feed cursor is attacker input (API-035). Opening any byte string under
// any binding must not panic, must refuse with exactly ErrInvalidCursor, and
// may accept only a token this codec minted for an equivalent binding. Minted
// tokens are URL-safe, bounded, and refused under any other network, viewer,
// filter or key, and after their TTL; a rotated-out key still verifies while
// it is in the key set.

type fuzzMinted struct {
	b   Binding
	pos Cursor
}

type fuzzFixture struct {
	codec, rotated, retired, otherKey, expired *CursorCodec
	accept                                     map[string]fuzzMinted // tokens that must open at fuzzNow
	seeds                                      []string
}

var (
	fuzzNow     = time.Date(2026, 11, 18, 12, 0, 0, 0, time.UTC)
	fuzzRowTime = time.Date(2026, 11, 1, 9, 30, 0, 123456789, time.UTC)
)

func newFuzzFixture(t testing.TB) *fuzzFixture {
	at := func(tm time.Time) func() time.Time { return func() time.Time { return tm } }
	k1 := map[string][]byte{"k1": testKey1}
	fx := &fuzzFixture{
		codec:    mustCodec(t, mustRing(t, "k1", k1), 0, at(fuzzNow)),
		rotated:  mustCodec(t, mustRing(t, "k2", map[string][]byte{"k1": testKey1, "k2": testKey2}), 0, at(fuzzNow)),
		retired:  mustCodec(t, mustRing(t, "k2", map[string][]byte{"k2": testKey2}), 0, at(fuzzNow)),
		otherKey: mustCodec(t, mustRing(t, "k1", map[string][]byte{"k1": testKey2}), 0, at(fuzzNow)),
		expired:  mustCodec(t, mustRing(t, "k1", k1), 0, at(fuzzNow.Add(DefaultCursorTTL))),
		accept:   map[string]fuzzMinted{},
	}
	bindings := []Binding{
		{Network: "net-1", Viewer: "person-1"},
		{Network: "net-1", Viewer: "person-2", Filter: Filter{Kinds: []string{"listing"}}},
		{Network: "net-2", Viewer: "person-1", Filter: Filter{Kinds: []string{"announcement"}, PreferenceRevision: "rev-9"}},
	}
	positions := []Cursor{
		{Created: fuzzRowTime, Kind: "listing", ID: "018f0000-0000-7000-8000-000000000042"},
		{Created: fuzzRowTime.Add(time.Nanosecond), Kind: "announcement", ID: "a"},
	}
	for _, b := range bindings {
		for _, pos := range positions {
			for _, c := range []*CursorCodec{fx.codec, fx.rotated} {
				tok := mustSeal(t, c, b, pos)
				fx.accept[tok] = fuzzMinted{b, pos}
				fx.seeds = append(fx.seeds, tok)
			}
		}
	}
	// Valid-looking but refused: expired, wrong key, malformed.
	early := mustCodec(t, mustRing(t, "k1", k1), 0, at(fuzzNow.Add(-DefaultCursorTTL)))
	valid := fx.seeds[0]
	kid, rest, _ := strings.Cut(valid, ".")
	payload, sig, _ := strings.Cut(rest, ".")
	fx.seeds = append(fx.seeds,
		mustSeal(t, early, bindings[0], positions[0]),
		mustSeal(t, fx.otherKey, bindings[0], positions[0]),
		"", ".", "..", "...", kid, payload+"."+sig, kid+"."+payload, valid+".", valid+"A", valid[:len(valid)-1],
		strings.ToUpper(valid), kid+"."+payload+"\n."+sig, kid+"."+payload+"=."+sig, kid+".."+sig,
		"k9"+valid[len(kid):], strings.Repeat("A", maxTokenLen+1), "\x00.\x00.\x00", "%2E",
	)
	return fx
}

func fuzzKinds(mask uint8) []string {
	var kinds []string
	for i, k := range []string{"listing", "announcement", "transaction", "listing"} {
		if mask&(1<<i) != 0 {
			kinds = append(kinds, k)
		}
	}
	return kinds
}

func sameBinding(a, b Binding) bool {
	_, da, ea := a.canonical()
	_, db, eb := b.canonical()
	return ea == nil && eb == nil && a.Network == b.Network && a.Viewer == b.Viewer && da == db
}

func samePos(a, b Cursor) bool {
	return a.Created.Equal(b.Created) && a.Kind == b.Kind && a.ID == b.ID
}

func FuzzOpenCursor(f *testing.F) {
	fx := newFuzzFixture(f)
	for _, tok := range fx.seeds {
		f.Add(tok, "net-1", "person-1", "", uint8(0), "item-1")
	}
	// seeds[4:8] are minted for bindings[1], seeds[8:12] for bindings[2].
	f.Add(fx.seeds[4], "net-1", "person-2", "", uint8(1), "x")
	f.Add(fx.seeds[5], "net-1", "person-2", "", uint8(9), "x")
	f.Add(fx.seeds[8], "net-2", "person-1", "rev-9", uint8(2), "\xff")
	f.Add(fx.seeds[8], "net-2", "person-1", "rev-9", uint8(3), "y")
	f.Add("", "\x00", "\xff", strings.Repeat("r", maxBindingLen+1), uint8(4), strings.Repeat("<", maxSourceIDLen))

	f.Fuzz(func(t *testing.T, tok, network, viewer, rev string, mask uint8, id string) {
		b := Binding{Network: network, Viewer: viewer, Filter: Filter{Kinds: fuzzKinds(mask), PreferenceRevision: rev}}

		// 1. Arbitrary input: the uniform refusal, or a token minted for an
		// equivalent binding. The fuzzer cannot compute a MAC, so anything
		// else accepted would be a forgery.
		for _, c := range []*CursorCodec{fx.codec, fx.rotated} {
			got, err := c.Open(b, tok)
			if err != nil {
				if err != ErrInvalidCursor || got != (Cursor{}) {
					t.Fatalf("refusal = %+v, %v; want zero Cursor and ErrInvalidCursor", got, err)
				}
				continue
			}
			m, ok := fx.accept[tok]
			if !ok {
				t.Fatalf("accepted %q, which was never minted", tok)
			}
			if !sameBinding(m.b, b) || !samePos(m.pos, got) {
				t.Fatalf("accepted %q for %+v as %+v; minted for %+v as %+v", tok, b, got, m.b, m.pos)
			}
			if _, err := fx.expired.Open(b, tok); err != ErrInvalidCursor {
				t.Fatalf("accepted %q after its TTL", tok)
			}
		}

		// 2. Tokens minted for the fuzzed binding and position.
		pos := Cursor{Created: fuzzRowTime, Kind: "listing", ID: id}
		minted, err := fx.codec.Seal(b, pos)
		if err != nil {
			if _, _, berr := b.canonical(); berr == nil && validPosition(pos) {
				t.Fatalf("valid inputs refused: %+v %+v: %v", b, pos, err)
			}
			return
		}
		if len(minted) > maxTokenLen || !isTokenText(minted) || strings.Count(minted, ".") != 2 {
			t.Fatalf("minted token %q is not a bounded 3-segment base64url token", minted)
		}
		// The readable segments (key ID, payload) do not depend on the
		// binding: network, viewer and filter reach only the MAC.
		ref := mustSeal(t, fx.codec, Binding{Network: "ref", Viewer: "ref"}, pos)
		if minted[:strings.LastIndexByte(minted, '.')] != ref[:strings.LastIndexByte(ref, '.')] {
			t.Fatalf("readable token part depends on the binding: %q vs %q", minted, ref)
		}
		if got, err := fx.codec.Open(b, minted); err != nil || !samePos(got, pos) {
			t.Fatalf("round trip: %+v, %v; want %+v", got, err, pos)
		}
		if got, err := fx.rotated.Open(b, minted); err != nil || !samePos(got, pos) {
			t.Fatalf("rotated-out key no longer verifies: %+v, %v", got, err)
		}
		other := Filter{Kinds: []string{"listing"}, PreferenceRevision: rev}
		if kinds, _, _ := b.Filter.canonical(); len(kinds) == 1 {
			other.Kinds = nil
		}
		for name, ob := range map[string]Binding{
			"network":  {Network: network + "x", Viewer: viewer, Filter: b.Filter},
			"viewer":   {Network: network, Viewer: viewer + "x", Filter: b.Filter},
			"swapped":  {Network: viewer, Viewer: network, Filter: b.Filter},
			"kinds":    {Network: network, Viewer: viewer, Filter: other},
			"revision": {Network: network, Viewer: viewer, Filter: Filter{Kinds: b.Filter.Kinds, PreferenceRevision: rev + "x"}},
		} {
			if name == "swapped" && network == viewer {
				continue
			}
			if _, err := fx.codec.Open(ob, minted); err != ErrInvalidCursor {
				t.Fatalf("cursor opened under another %s: %+v", name, ob)
			}
		}
		for name, c := range map[string]*CursorCodec{"retired key": fx.retired, "other key": fx.otherKey, "expired": fx.expired} {
			if _, err := c.Open(b, minted); err != ErrInvalidCursor {
				t.Fatalf("cursor opened under %s", name)
			}
		}
	})
}
