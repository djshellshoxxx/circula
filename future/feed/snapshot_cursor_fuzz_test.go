package feed

import (
	"strings"
	"sync"
	"testing"
	"time"
	"unicode/utf8"
)

type snapshotFuzzToken struct {
	binding  Binding
	snapshot SnapshotCursor
}

func FuzzOpenSnapshotCursor(f *testing.F) {
	now := func() time.Time { return testNow }
	codec := mustCodec(f, mustRing(f, "k1", map[string][]byte{"k1": testKey1}), 0, now)
	rotated := mustCodec(f, mustRing(f, "k2", map[string][]byte{"k1": testKey1, "k2": testKey2}), 0, now)
	retired := mustCodec(f, mustRing(f, "k2", map[string][]byte{"k2": testKey2}), 0, now)
	wrong := mustCodec(f, mustRing(f, "k1", map[string][]byte{"k1": testKey2}), 0, now)
	expired := mustCodec(f, codec.keys, 0, func() time.Time { return testNow.Add(2 * DefaultCursorTTL) })
	accepted := map[string]snapshotFuzzToken{}
	bindings := []Binding{
		{Network: "net-1", Viewer: "person-1"},
		{Network: "net-1", Viewer: "person-2", Filter: Filter{Kinds: []string{"listing"}}},
		{Network: "net-2", Viewer: "person-1", Filter: Filter{Kinds: []string{"announcement"}, PreferenceRevision: "rev-9"}},
	}
	var seed string
	for _, binding := range bindings {
		for _, signing := range []struct {
			kid string
			key []byte
		}{{"k1", testKey1}, {"k2", testKey2}} {
			kid, key := signing.kid, signing.key
			snapshot := testSnapshot()
			token := authenticatedSnapshotToken(f, key, kid, binding, snapshotFixturePayload(f, snapshot))
			accepted[token] = snapshotFuzzToken{binding, snapshot}
			mask := uint8(0)
			if len(binding.Filter.Kinds) == 1 {
				mask = 1
				if binding.Filter.Kinds[0] == "announcement" {
					mask = 2
				}
			}
			f.Add(token, binding.Network, binding.Viewer, binding.Filter.PreferenceRevision, mask, "item-1", int64(100000), testNow.UnixNano())
			if binding.Network == "net-1" && binding.Viewer == "person-1" && kid == "k1" {
				seed = token
			}
		}
	}
	for _, malformed := range []string{"", ".", "..", "3", "3.k1", "3.k1.", "3.k1.A", seed[:len(seed)-1], seed + "=", seed + ".", seed + "\n", strings.Repeat("A", maxTokenLen+1), "3.\x00.\xff"} {
		f.Add(malformed, "net-1", "person-1", "", uint8(0), "item-1", int64(1), testNow.UnixNano())
	}
	f.Add(seed, "net-1", "person-1", "", uint8(0), "\xff", int64(-1), int64(-1))
	f.Add(seed, "net-1", "person-1", "", uint8(0), strings.Repeat("<", maxSourceIDLen), int64(1), testNow.Add(-DefaultCursorTTL).UnixNano())
	f.Add(seed, "net-1", "person-1", "", uint8(0), "item-1", int64(1), testNow.Add(maxIssueSkew+time.Nanosecond).UnixNano())

	f.Fuzz(func(t *testing.T, token, network, viewer, revision string, mask uint8, id string, highWater, issuedNanos int64) {
		binding := Binding{Network: network, Viewer: viewer, Filter: Filter{Kinds: fuzzKinds(mask), PreferenceRevision: revision}}
		for _, current := range []*CursorCodec{codec, rotated} {
			got, err := current.OpenSnapshot(binding, token)
			if err != nil {
				if err != ErrInvalidCursor || got != (SnapshotCursor{}) {
					t.Fatalf("nonuniform refusal: %+v %v", got, err)
				}
				continue
			}
			fixture, ok := accepted[token]
			if !ok || !sameBinding(binding, fixture.binding) || !sameSnapshot(got, fixture.snapshot) {
				t.Fatalf("accepted an unminted or wrongly bound snapshot: %+v %v", got, err)
			}
		}

		snapshot := SnapshotCursor{Position: Cursor{Created: testPos.Created, Kind: "listing", ID: id}, HighWater: highWater, IssuedAt: time.Unix(0, issuedNanos)}
		minted, err := codec.SealSnapshot(binding, snapshot)
		if err != nil {
			_, _, bindingErr := binding.canonical()
			validID := id != "" && len(id) <= maxSourceIDLen && utf8.ValidString(id)
			withinTime := issuedNanos > 0 && !testNow.Before(snapshot.IssuedAt.Add(-maxIssueSkew)) && testNow.Before(snapshot.IssuedAt.Add(DefaultCursorTTL))
			if bindingErr == nil && validID && highWater > 0 && withinTime {
				t.Fatalf("valid snapshot refused: %+v %v", snapshot, err)
			}
			return
		}
		if len(minted) > maxTokenLen || !isTokenText(minted) || !strings.HasPrefix(minted, "3.k1.") || strings.Count(minted, ".") != 2 {
			t.Fatalf("minted token is not bounded and canonical: %q", minted)
		}
		for _, current := range []*CursorCodec{codec, rotated} {
			if got, err := current.OpenSnapshot(binding, minted); err != nil || !sameSnapshot(got, snapshot) {
				t.Fatalf("snapshot did not round-trip during rotation: %+v %v", got, err)
			}
		}
		for _, current := range []*CursorCodec{retired, wrong, expired} {
			wantInvalidSnapshot(t, current, binding, minted)
		}
		otherFilter := binding.Filter
		kinds, _, _ := binding.Filter.canonical()
		if len(kinds) == 1 {
			otherFilter.Kinds = nil
		} else {
			otherFilter.Kinds = []string{"listing"}
		}
		for _, other := range []Binding{
			{Network: network + "x", Viewer: viewer, Filter: binding.Filter},
			{Network: network, Viewer: viewer + "x", Filter: binding.Filter},
			{Network: network, Viewer: viewer, Filter: otherFilter},
			{Network: network, Viewer: viewer, Filter: Filter{Kinds: binding.Filter.Kinds, PreferenceRevision: revision + "x"}},
		} {
			wantInvalidSnapshot(t, codec, other, minted)
		}
		wire, _ := b64.DecodeString(strings.TrimPrefix(minted, "3.k1."))
		wire[len(wire)/2] ^= 1
		wantInvalidSnapshot(t, codec, binding, "3.k1."+b64.EncodeToString(wire))
	})
}

func TestSnapshotCursorConcurrentUseAndKeyCopies(t *testing.T) {
	c, _ := testCodec(t)
	ring := c.keys.(*KeyRing)
	const workers = 8
	var wg sync.WaitGroup
	errs := make(chan string, workers)
	for worker := 0; worker < workers; worker++ {
		wg.Add(1)
		go func(worker int) {
			defer wg.Done()
			for i := 0; i < 100; i++ {
				_, key := ring.SigningKey()
				clear(key)
				snapshot := testSnapshot()
				snapshot.Position.ID = "worker-" + itoa64(int64(worker)) + "-item-" + itoa64(int64(i))
				token, err := c.SealSnapshot(testBindA, snapshot)
				if err != nil {
					errs <- "concurrent seal failed"
					return
				}
				verification, _ := ring.VerificationKey("k1")
				clear(verification)
				if got, err := c.OpenSnapshot(testBindA, token); err != nil || !sameSnapshot(got, snapshot) {
					errs <- "concurrent round-trip failed"
					return
				}
			}
		}(worker)
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		t.Fatal(err)
	}
}
