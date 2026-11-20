package feed

import (
	"testing"
	"time"
)

// A duplicate source reference can arise from a join. Identical snapshots are
// harmless; conflicting facts have no safe winner and must not depend on row
// order, including a stale visible row paired with a current hidden one.
func TestPageRejectsConflictingSourceSnapshots(t *testing.T) {
	codec, _ := testCodec(t)
	first := Source{"n", "listing", "duplicate", "member", time.Unix(100, 0), true, true}
	for name, change := range map[string]func(*Source){
		"creation time": func(s *Source) { s.Created = s.Created.Add(time.Second) },
		"owner":         func(s *Source) { s.Member = "another-member" },
		"visibility":    func(s *Source) { s.Visible = false },
		"active state":  func(s *Source) { s.Active = false },
	} {
		t.Run(name, func(t *testing.T) {
			second := first
			change(&second)
			for _, rows := range [][]Source{{first, second}, {second, first}} {
				result, err := Page(codec, testQuery(1), rows, allowAll)
				if err != ErrInvalidSource || result.Items != nil || result.Next != "" {
					t.Errorf("conflicting snapshots returned %+v, %v; want no result and ErrInvalidSource", result, err)
				}
			}
		})
	}
}

func TestPageAcceptsEquivalentSourceSnapshots(t *testing.T) {
	codec, _ := testCodec(t)
	first := Source{"n", "listing", "duplicate", "member", time.Unix(100, 0).UTC(), true, true}
	second := first
	second.Created = second.Created.In(time.FixedZone("fixture", 3600))
	otherNetwork := second
	otherNetwork.Network = "other-network"
	otherNetwork.Visible = false
	result, err := Page(codec, testQuery(1), []Source{first, second, otherNetwork}, allowAll)
	if err != nil || len(result.Items) != 1 || result.Items[0].ID != first.ID || result.Next != "" {
		t.Fatalf("equivalent snapshots: %+v, %v", result, err)
	}
}

func TestZeroValueCursorCodecFailsClosed(t *testing.T) {
	valid, _ := testCodec(t)
	binding := Binding{Network: "n", Viewer: "member-A"}
	position := Cursor{Created: time.Unix(100, 0), Kind: "listing", ID: "one"}
	token, err := valid.Seal(binding, position)
	if err != nil {
		t.Fatal(err)
	}
	var zero CursorCodec
	t.Run("seal", func(t *testing.T) {
		defer func() {
			if recovered := recover(); recovered != nil {
				t.Errorf("zero-value codec panicked: %v", recovered)
			}
		}()
		if token, err := zero.Seal(binding, position); err != errSeal || token != "" {
			t.Fatalf("zero-value seal = %q, %v", token, err)
		}
	})
	t.Run("open", func(t *testing.T) {
		defer func() {
			if recovered := recover(); recovered != nil {
				t.Errorf("zero-value codec panicked: %v", recovered)
			}
		}()
		if position, err := zero.Open(binding, token); err != ErrInvalidCursor || position != (Cursor{}) {
			t.Fatalf("zero-value open = %+v, %v", position, err)
		}
	})
	t.Run("page", func(t *testing.T) {
		result, err := Page(&zero, testQuery(100), feedRows(1), allowAll)
		if err != ErrInvalidQuery || result.Items != nil || result.Next != "" {
			t.Fatalf("zero-value page = %+v, %v", result, err)
		}
	})
}
