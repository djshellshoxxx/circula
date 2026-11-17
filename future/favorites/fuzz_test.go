package favorites_test

import (
	"testing"
	"unicode"
	"unicode/utf8"

	"github.com/djshellshoxxx/circula/future/favorites"
	"github.com/djshellshoxxx/circula/future/favorites/internal/testkit"
	"github.com/djshellshoxxx/circula/future/favorites/memstore"
)

func FuzzCleanNickname(f *testing.F) {
	for _, s := range []string{"", "Bob", "  x ", "a\nb", "a‮b", "\xff", "家", "a​b"} {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, s string) {
		out, err := favorites.CleanNickname(s)
		if err != nil {
			return
		}
		if !utf8.ValidString(out) || utf8.RuneCountInString(out) > favorites.MaxNicknameRunes {
			t.Fatalf("accepted %q -> %q", s, out)
		}
		for _, r := range out {
			if unicode.IsControl(r) || unicode.Is(unicode.Cf, r) || unicode.Is(unicode.Zl, r) || unicode.Is(unicode.Zp, r) || r == utf8.RuneError {
				t.Fatalf("accepted forbidden rune %U in %q", r, s)
			}
		}
		again, err := favorites.CleanNickname(out)
		if err != nil || again != out {
			t.Fatalf("not idempotent: %q -> %q -> %q (%v)", s, out, again, err)
		}
	})
}

func FuzzCleanDescription(f *testing.F) {
	for _, s := range []string{"", "a\nb\tc", "a b", "\x00"} {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, s string) {
		out, err := favorites.CleanDescription(s)
		if err != nil {
			return
		}
		if again, err := favorites.CleanDescription(out); err != nil || again != out {
			t.Fatalf("not idempotent: %q -> %q", out, again)
		}
	})
}

// Arbitrary cursors and identifiers never panic and never widen scope.
func FuzzListCursorAndIDs(f *testing.F) {
	for _, s := range []string{"", "!!!", "eyJwIjp0cnVlLCJvIjoxLCJ0IjoiYm9iIn0", "eyJ0IjoiLi4ifQ"} {
		f.Add(s, "bob")
	}
	dir := testkit.NewDirectory()
	for _, id := range []string{"alice", "bob", "carol"} {
		dir.Add("n1", id)
	}
	svc, _ := favorites.NewService(favorites.Config{Store: memstore.New(), Policy: dir, Enabled: dir.Enabled, Trades: dir})
	_, _, _ = svc.Add(ctx, alice, favorites.AddRequest{Target: "bob"})
	_, _, _ = svc.Add(ctx, bob, favorites.AddRequest{Target: "carol"})
	f.Fuzz(func(t *testing.T, cursor, id string) {
		pg, err := svc.List(ctx, alice, favorites.ListOptions{Cursor: cursor})
		if err == nil {
			for _, it := range pg.Items {
				if it.Target == "carol" {
					t.Fatal("cursor widened scope to another owner's data")
				}
			}
		}
		_, _ = svc.Remove(ctx, alice, id)
		_ = svc.SetPinned(ctx, alice, id, true)
		if _, err := svc.GetDraft(ctx, alice, id); err == nil {
			t.Fatal("draft from nothing")
		}
		p, _ := svc.List(ctx, bob, favorites.ListOptions{})
		if len(p.Items) != 1 || p.Items[0].Target != "carol" {
			t.Fatal("alice's operations changed bob's list")
		}
		_, _, _ = svc.Add(ctx, alice, favorites.AddRequest{Target: "bob"})
	})
}
