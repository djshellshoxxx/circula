package receipts

import (
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func TestConsentAuthorizationAndFormats(t *testing.T) {
	s := Snapshot{"n", "tx", "a", "b", "credits", "completed", 123, time.Unix(1, 0)}
	q := Request{"n", "a", true, true}
	r, e := Create(q, s)
	if e != nil {
		t.Fatal(e)
	}
	b, e := r.JSON()
	var decoded Receipt
	if e != nil {
		t.Fatal(e)
	}
	if e = json.Unmarshal(b, &decoded); e != nil || decoded != r || !strings.Contains(r.Text(), "123 credits") {
		t.Fatal(string(b), e)
	}
	q.ExplicitlyRequested = false
	if _, e = Create(q, s); e == nil {
		t.Fatal("automatic generation")
	}
	q.ExplicitlyRequested = true
	q.Member = "c"
	if _, e = Create(q, s); e == nil {
		t.Fatal("nonparty")
	}
	q.Member = "a"
	q.Network = "x"
	if _, e = Create(q, s); e == nil {
		t.Fatal("tenant")
	}
	q.Network = "n"
	s.Status = "reversed"
	if _, e = Create(q, s); e == nil {
		t.Fatal("not completed")
	}
	s.Status = "completed"
	s.Unit = "credits\nFAKE"
	if _, e = Create(q, s); e == nil {
		t.Fatal("text injection")
	}
}
