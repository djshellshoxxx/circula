package api_test

import (
	"context"
	"net/http"
	"net/url"
	"strings"
	"testing"

	"github.com/djshellshoxxx/circula/internal/api"
	"github.com/djshellshoxxx/circula/internal/domain"
	"github.com/djshellshoxxx/circula/internal/service"
	"github.com/djshellshoxxx/circula/internal/store"
	"github.com/djshellshoxxx/circula/internal/testutil"
)

type pathLookupStore struct {
	store.Store
	ids []string
}

func (s *pathLookupStore) View(ctx context.Context, fn func(store.Tx) error) error {
	return s.Store.View(ctx, func(tx store.Tx) error {
		return fn(&pathLookupTx{Tx: tx, GovRepo: tx.(store.GovRepo), ids: &s.ids})
	})
}

func (s *pathLookupStore) Update(ctx context.Context, fn func(store.Tx) error) error {
	return s.Store.Update(ctx, func(tx store.Tx) error {
		return fn(&pathLookupTx{Tx: tx, GovRepo: tx.(store.GovRepo), ids: &s.ids})
	})
}

type pathLookupTx struct {
	store.Tx
	store.GovRepo
	ids *[]string
}

func (t *pathLookupTx) GetTransaction(netID, id string) (*domain.Transaction, error) {
	*t.ids = append(*t.ids, id)
	return t.Tx.GetTransaction(netID, id)
}

func (t *pathLookupTx) GetMember(netID, id string) (*domain.Member, error) {
	*t.ids = append(*t.ids, id)
	return t.Tx.GetMember(netID, id)
}

func (t *pathLookupTx) GetListing(netID, id string) (*domain.Listing, error) {
	*t.ids = append(*t.ids, id)
	return t.Tx.GetListing(netID, id)
}

func (t *pathLookupTx) GetDispute(netID, id string) (*domain.Dispute, error) {
	*t.ids = append(*t.ids, id)
	return t.Tx.GetDispute(netID, id)
}

// A2-16 / SEC-024: the malformed resource identifier must never reach a
// database lookup. Authentication and network lookup still run first, as
// required by API-013; the boundary must not turn an unauthenticated 401 into 404.
func TestPathIDsRejectedBeforeResourceLookup(t *testing.T) {
	var lookups *pathLookupStore
	env := testutil.NewEnvWith(t, func(s store.Store) store.Store {
		lookups = &pathLookupStore{Store: s}
		return lookups
	})
	w := &world{t: t, env: env, h: api.NewHandler(env.Svc)}
	t.Cleanup(func() { api.CloseHandler(w.h) })
	w.admin = env.Person("admin")
	w.net, w.adminM = env.Network(w.admin, "acme", nil)
	w.adminT = w.token(w.admin, domain.ScopeRead, domain.ScopeWrite)
	for _, id := range []string{
		"x' OR '1'='1", "bad/id", "bad\x00id",
		"018f0000000070008000000000000001", // compact UUID is not the API text format
		"{018f0000-0000-7000-8000-000000000001}",
		"urn:uuid:018f0000-0000-7000-8000-000000000001",
		"018f0000-0000-7000-8000-00000000000z",
	} {
		for _, route := range []struct{ method, path string }{
			{"GET", "/transactions/{id}"}, {"GET", "/transactions/{id}/events"},
			{"POST", "/transactions/{id}/accept"}, {"GET", "/members/{id}"},
			{"GET", "/listings/{id}"}, {"GET", "/disputes/{id}"},
			{"POST", "/disputes/{id}/review"}, {"GET", "/admin/adjustments/{id}"},
			{"PUT", "/admin/members/{id}/roles/reviewer"}, {"DELETE", "/tokens/{id}"},
			{"GET", "/exports/{id}"}, {"GET", "/exports/{id}/artefact"},
			{"GET", "/admin/consistency-checks/{id}"},
		} {
			path := "/api/v1" + strings.ReplaceAll(route.path, "{id}", url.PathEscape(id))
			lookups.ids = nil
			r := w.do(req{method: route.method, path: path, token: w.adminT, hdr: map[string]string{"Idempotency-Key": w.key()}})
			wantProblem(t, r, 404, "not-found")
			for _, seen := range lookups.ids {
				if seen == id {
					t.Errorf("%s %s passed malformed identifier %q to a database lookup", route.method, route.path, id)
				}
			}
		}
	}
	// A valid UUID still reaches the service and its ordinary not-found path.
	const missing = "018f0000-0000-7000-8000-ffffffffffff"
	lookups.ids = nil
	wantProblem(t, w.get(w.adminT, "/transactions/"+missing), 404, "not-found")
	if len(lookups.ids) != 1 || lookups.ids[0] != missing {
		t.Fatalf("valid resource id did not reach normal lookup: %v", lookups.ids)
	}
}

func TestMalformedPathIDsKeepAuthenticationAndPrivacyContract(t *testing.T) {
	w := newWorld(t)
	for _, credential := range []string{"", "cir_not-a-valid-token"} {
		r := w.get(credential, "/members/not-a-uuid")
		wantProblem(t, r, 401, "unauthenticated")
		if r.Header().Get("WWW-Authenticate") == "" {
			t.Fatal("malformed path suppressed the authentication challenge")
		}
	}
	session, _ := testutil.Must2(w.env.Svc.Login(w.env.Ctx, "alice", "correct horse battery"))
	r := w.do(req{method: "POST", path: "/api/v1/transactions/not-a-uuid/accept",
		hdr: map[string]string{"Cookie": (&http.Cookie{Name: "circula_session", Value: session}).String()}})
	wantProblem(t, r, 403, "csrf-invalid")

	testutil.Must(0, w.env.Svc.SuspendMember(w.env.Ctx, w.admin, w.net.ID, w.bobM.ID, "privacy fixture"))
	hidden := wantProblem(t, w.get(w.aliceT, "/members/"+w.bobM.ID), 404, "not-found")
	for _, id := range []string{"not-a-uuid", "018f0000-0000-7000-8000-ffffffffffff"} {
		p := wantProblem(t, w.get(w.aliceT, "/members/"+id), 404, "not-found")
		if p.Type != hidden.Type || p.Title != hidden.Title || p.Detail != hidden.Detail {
			t.Errorf("missing/malformed id reveals a different refusal: %+v vs %+v", p, hidden)
		}
	}
	// Other wildcard segments have their own contracts, including non-UUID
	// role names and invitation credentials. A real UUID member path still acts.
	wantStatus(t, w.do(req{method: "PUT", path: "/api/v1/admin/members/" + w.aliceM.ID + "/roles/reviewer", token: w.adminT}), 200)
	guest := w.env.Person("guest")
	token, _ := testutil.Must2(w.env.Svc.CreateInvitation(w.env.Ctx, w.admin, w.net.ID, service.InviteInput{}))
	wantStatus(t, w.post(w.token(guest, domain.ScopeRead, domain.ScopeWrite), "/invitations/"+token+"/accept", m{"display_name": "Guest"}, w.key()), 201)

	// Existing adapter jobs use UUIDv4; only the textual shape is validated.
	check := w.post(w.adminT, "/admin/consistency-checks", nil, w.key())
	wantStatus(t, check, 201)
	wantStatus(t, w.do(req{method: "GET", path: check.Header().Get("Location"), token: w.adminT}), 200)
}
