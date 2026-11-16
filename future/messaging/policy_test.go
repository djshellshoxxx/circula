package messaging

import (
	"strings"
	"testing"
	"time"
)

func TestPrivacy(t *testing.T) {
	th := Thread{Network: "n", A: "a", B: "b"}
	for _, a := range []Access{{"x", "a", true}, {"n", "c", true}, {"n", "a", false}} {
		if _, e := Prepare(th, a, "secret"); e == nil {
			t.Fatal(a)
		}
	}
	a := Access{"n", "a", true}
	body := "<script>not HTML</script>"
	if got, e := Prepare(th, a, body); e != nil || got != body {
		t.Fatal(got, e)
	}
	th.Blocked = true
	if _, e := Prepare(th, a, "hi"); e == nil {
		t.Fatal("blocked")
	}
	th.Blocked = false
	if _, e := Prepare(th, a, strings.Repeat("x", 4097)); e == nil {
		t.Fatal("size")
	}
}
func TestRetention(t *testing.T) {
	now := time.Unix(1, 0)
	end, e := PurgeAfter(now, 30)
	if e != nil || end.Sub(now) != 720*time.Hour {
		t.Fatal(end, e)
	}
	if _, e = PurgeAfter(now, 31); e == nil {
		t.Fatal("retention")
	}
}
