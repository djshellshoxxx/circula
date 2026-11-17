package favorites

import (
	"reflect"
	"testing"
)

func TestVisibleFavoritesIsolationAndOrder(t *testing.T) {
	owner := Member{ID: "alice", Network: "net1", Active: true, Visible: true}
	saved := []Entry{
		{Favorite: Favorite{Network: "net1", Owner: "alice", Target: "z"}, Order: 2},
		{Favorite: Favorite{Network: "net1", Owner: "bob", Target: "x"}, Pinned: true},
		{Favorite: Favorite{Network: "net1", Owner: "alice", Target: "b"}, Order: 4, Pinned: true},
		{Favorite: Favorite{Network: "net1", Owner: "alice", Target: "a"}, Order: 1, Pinned: true},
		{Favorite: Favorite{Network: "net2", Owner: "alice", Target: "t"}, Pinned: true},
		{Favorite: Favorite{Network: "net1", Owner: "alice", Target: "b"}, Order: 0, Pinned: true},
		{Favorite: Favorite{Network: "net1", Owner: "alice", Target: "blocked"}, Pinned: true},
	}
	targets := map[string]Member{}
	for _, id := range []string{"a", "b", "z", "blocked"} {
		targets[id] = Member{ID: id, Network: "net1", Active: true, Visible: true}
	}
	x := targets["blocked"]
	x.Blocked = true
	targets["blocked"] = x
	got := VisibleFavorites(owner, saved, targets)
	ids := []string{}
	for _, e := range got {
		ids = append(ids, e.Target)
	}
	if !reflect.DeepEqual(ids, []string{"a", "b", "z"}) {
		t.Fatalf("got %v", ids)
	}
	saved[3].Nickname = "changed"
	if got[0].Nickname == "changed" {
		t.Fatal("result aliases caller")
	}
}
func TestRemoveFavoriteOnlyWithinOwnerNetwork(t *testing.T) {
	entries := []Entry{{Favorite: Favorite{Network: "n", Owner: "a", Target: "b"}}, {Favorite: Favorite{Network: "n", Owner: "other", Target: "b"}}, {Favorite: Favorite{Network: "other", Owner: "a", Target: "b"}}}
	got, removed := RemoveFavorite(Member{ID: "a", Network: "n", Active: true}, entries, "b")
	if !removed || len(got) != 2 || len(entries) != 3 {
		t.Fatal("wrong owner scope or input mutation")
	}
	if got[0].Owner != "other" || got[1].Network != "other" {
		t.Fatalf("unexpected %v", got)
	}
	if _, ok := RemoveFavorite(Member{ID: "a", Network: "n", Active: false}, entries, "b"); ok {
		t.Fatal("inactive owner removed favorite")
	}
}
