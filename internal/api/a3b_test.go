package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/djshellshoxxx/circula/internal/domain"
	"github.com/djshellshoxxx/circula/internal/httpx"
	"github.com/djshellshoxxx/circula/internal/service"
	"github.com/djshellshoxxx/circula/internal/testutil"
)

// A3-06: with Config.OnMemberErased wired to OnMemberErased, erasing a member
// drops the API replay cache and the export artefacts of that person; before
// the fix production left the hook nil and the cached 2xx bodies survived.
func TestA3_06_ErasureDropsReplayCacheAndExports(t *testing.T) {
	env := testutil.NewEnvConfig(t, nil, func(c *service.Config) { c.OnMemberErased = OnMemberErased })
	ctx := context.Background()
	admin := env.Person("admin")
	net, _ := env.Network(admin, "acme", nil)
	alice, aliceM := env.Member(admin, net, "alice")
	_, bobM := env.Member(admin, net, "bob")
	raw, _, err := env.Svc.CreateAPIToken(ctx, alice, service.TokenInput{Name: "t", Scopes: []domain.Scope{domain.ScopeRead, domain.ScopeWrite}})
	if err != nil {
		t.Fatal(err)
	}
	h := NewHandlerWithOptions(env.Svc, Options{}).(*server)

	body := `{"counterparty_member_id":"` + bobM.ID + `","direction":"pay","amount_minor":"1234","description":"secret memo"}`
	r := httptest.NewRequest("POST", "/api/v1/transactions", strings.NewReader(body))
	r.Header.Set("Authorization", "Bearer "+raw)
	r.Header.Set("Content-Type", "application/json")
	r.Header.Set("Idempotency-Key", "replay-key-0001")
	r.Header.Set("X-Network", "acme")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, r)
	if rec.Code/100 != 2 {
		t.Fatalf("first request: %d %s", rec.Code, rec.Body.String())
	}
	if len(h.idem.m) == 0 {
		t.Fatal("test premise: no replay record cached")
	}
	// An export artefact of the same person.
	h.jobs.putExport(&exportEntry{net: net.ID, person: alice.PersonID, at: time.Now(), dto: exportDTO{ID: "exp1"}, file: []byte("data")})
	// And one of somebody else, which must survive.
	h.jobs.putExport(&exportEntry{net: net.ID, person: admin.PersonID, at: time.Now(), dto: exportDTO{ID: "exp2"}, file: []byte("other")})

	txs, err := env.Svc.ListTransactions(ctx, alice, net.ID, nil, service.Page{})
	if err != nil || len(txs) == 0 {
		t.Fatalf("list: %v %d", err, len(txs))
	}
	if _, err := env.Svc.CancelTransaction(ctx, alice, net.ID, service.CancelInput{TxID: txs[0].ID, IdempotencyKey: "c1"}); err != nil {
		t.Fatal(err)
	}
	if _, err := env.Svc.DepartMember(ctx, alice, net.ID, service.DepartInput{}); err != nil {
		t.Fatal(err)
	}
	if _, err := env.Svc.EraseMyData(ctx, alice, net.ID, ""); err != nil {
		t.Fatal(err)
	}
	_ = aliceM
	if n := len(h.idem.m); n != 0 {
		t.Errorf("%d replay record(s) of the erased person survive", n)
	}
	if h.jobs.getExport(net.ID, alice.PersonID, "exp1") != nil {
		t.Error("export artefact of the erased person survives")
	}
	if h.jobs.getExport(net.ID, admin.PersonID, "exp2") == nil {
		t.Error("another person's export artefact was dropped")
	}
	if h.jobs.bytes != int64(len("other")) {
		t.Errorf("job store byte count = %d", h.jobs.bytes)
	}
}

// OnMemberErased before any handler exists must be a harmless no-op.
func TestA3_06_HookWithoutHandlerIsNoop(t *testing.T) {
	setErasureHook(nil)
	OnMemberErased("n", "p")
}

// A3-09: POST /tokens returns the plaintext token once and must never put it
// in the replay cache, with or without an Idempotency-Key.
func TestA3_09_TokenNotCached(t *testing.T) {
	env := testutil.NewEnv(t)
	admin := env.Person("admin")
	env.Network(admin, "acme", nil)
	csrf := []byte("0123456789abcdef0123456789abcdef")
	h := NewHandlerWithOptions(env.Svc, Options{CSRFKey: csrf}).(*server)
	sess, _, err := env.Svc.Login(context.Background(), "admin", "correct horse battery")
	if err != nil {
		t.Fatal(err)
	}
	var tokens []string
	for i := 0; i < 2; i++ {
		r := httptest.NewRequest("POST", "/api/v1/tokens", strings.NewReader(`{"name":"ci","scopes":["read"]}`))
		r.Header.Set("Content-Type", "application/json")
		r.Header.Set("Cookie", (&http.Cookie{Name: "circula_session", Value: sess}).String())
		r.Header.Set("X-CSRF-Token", httpx.CSRF{Key: csrf}.Token(sess))
		r.Header.Set("Idempotency-Key", "tok-key-0001")
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, r)
		if rec.Code != 201 {
			t.Fatalf("create %d: %d %s", i, rec.Code, rec.Body.String())
		}
		if rec.Header().Get("Idempotent-Replayed") != "" {
			t.Errorf("token response %d replayed from cache", i)
		}
		tokens = append(tokens, rec.Body.String())
	}
	if tokens[0] == tokens[1] {
		t.Error("the same plaintext token was served twice")
	}
	if n := len(h.idem.m); n != 0 {
		t.Errorf("%d replay record(s) hold a plaintext token", n)
	}
}
