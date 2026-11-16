package qr

import (
	"testing"
	"time"
)

func TestLinks(t *testing.T) {
	a, e := NewToken()
	if e != nil {
		t.Fatal(e)
	}
	b, _ := NewToken()
	if a == b {
		t.Fatal("duplicate entropy")
	}
	u, e := Link("https://c.test", a)
	if e != nil {
		t.Fatal(e)
	}
	got, e := Parse("https://c.test", u)
	if e != nil || got != a {
		t.Fatal(got, e)
	}
	for _, bad := range []string{u + "?amount=1", u + "#x", u + "#", "https://evil.test/qr/v1/" + a, "https://c.test/qr/v1/short"} {
		if _, e := Parse("https://c.test", bad); e == nil {
			t.Fatal(bad)
		}
	}
}
func TestRequestExpiryAndScope(t *testing.T) {
	now := time.Unix(100, 0)
	r := Request{Network: "n", Recipient: "b", Amount: 1, Expires: now.Add(time.Minute)}
	if r.Validate("n", "a", now) != nil {
		t.Fatal("valid")
	}
	if r.Validate("x", "a", now) == nil || r.Validate("n", "b", now) == nil || r.Validate("n", "a", r.Expires) == nil {
		t.Fatal("bad boundary")
	}
	r.ClaimedTransaction = "tx"
	if r.Validate("n", "a", now) == nil {
		t.Fatal("claimed")
	}
}

func FuzzParse(f *testing.F) {
	f.Add("https://c.test/qr/v1/AAAAAAAAAAAAAAAAAAAAAA")
	f.Add("javascript:alert(1)")
	f.Fuzz(func(t *testing.T, raw string) {
		token, err := Parse("https://c.test", raw)
		if err != nil {
			return
		}
		roundtrip, err := Link("https://c.test", token)
		if err != nil || roundtrip != raw {
			t.Fatalf("accepted noncanonical URL: %q", raw)
		}
	})
}
