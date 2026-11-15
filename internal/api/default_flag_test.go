package api_test

import (
	"context"
	"testing"
)

func TestDefaultFlagAPI(t *testing.T) {
	w := newWorld(t)
	path := "/admin/members/" + w.aliceM.ID + "/default-flag"
	before, err := w.env.Svc.GetBalance(context.Background(), w.alice, w.net.ID)
	if err != nil {
		t.Fatal(err)
	}
	for _, flag := range []bool{true, false} {
		rec := w.do(req{method: "PUT", path: "/api/v1" + path, token: w.adminT, body: m{"default_flag": flag, "reason": "governance review"}})
		wantStatus(t, rec, 200)
		got := decode[struct {
			DefaultFlag bool `json:"default_flag"`
		}](t, rec)
		if got.DefaultFlag != flag {
			t.Fatalf("default flag = %v, want %v", got.DefaultFlag, flag)
		}
	}
	for _, body := range []m{{"reason": "missing flag"}, {"default_flag": nil, "reason": "null flag"}, {"default_flag": true}} {
		wantStatus(t, w.do(req{method: "PUT", path: "/api/v1" + path, token: w.adminT, body: body}), 422)
	}
	wantStatus(t, w.do(req{method: "PUT", path: "/api/v1" + path, token: w.bobT, body: m{"default_flag": true, "reason": "unauthorised"}}), 403)
	readOnly := w.token(w.admin, "read")
	wantStatus(t, w.do(req{method: "PUT", path: "/api/v1" + path, token: readOnly, body: m{"default_flag": true, "reason": "read only"}}), 403)
	after, err := w.env.Svc.GetBalance(context.Background(), w.alice, w.net.ID)
	if err != nil {
		t.Fatal(err)
	}
	if before.Balance != after.Balance || before.LedgerSeq != after.LedgerSeq {
		t.Fatal("default flag changed the ledger")
	}
}
