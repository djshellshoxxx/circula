package discovery

import (
	"testing"
	"time"
)

func TestScopeFiltersAndPagination(t *testing.T) {
	now := time.Unix(1, 0)
	rows := []Listing{{"b", "n", "Bike", "tools", "north", true, true, now}, {"a", "n", "BIKE", "tools", "north", true, true, now}, {"secret", "x", "bike", "tools", "north", true, true, now}, {"hidden", "n", "bike", "tools", "north", true, false, now}}
	q := Query{Network: "n", Text: "bike", Category: "tools", Area: "north", Limit: 1}
	r, e := Search(rows, q)
	if e != nil || len(r) != 1 || r[0].ID != "a" {
		t.Fatal(r, e)
	}
	q.AfterID = r[0].ID
	q.AfterTime = r[0].Updated
	r, e = Search(rows, q)
	if e != nil || len(r) != 1 || r[0].ID != "b" {
		t.Fatal(r, e)
	}
	q.Limit = 101
	if _, e = Search(rows, q); e == nil {
		t.Fatal("unbounded")
	}
}
