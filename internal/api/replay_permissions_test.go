package api_test

import (
	"testing"

	"github.com/djshellshoxxx/circula/internal/domain"
	"github.com/djshellshoxxx/circula/internal/testutil"
)

// A2-03: a cached success must not bypass the caller's current authority.
// Denied attempts retain the record: an authorized retry still returns the
// original response without re-executing a non-idempotent command.
func TestReplayRechecksWriteScopeAndMembership(t *testing.T) {
	w := newWorld(t)
	body := m{"counterparty_member_id": w.bobM.ID, "direction": "pay", "amount_minor": "100"}
	key := w.key()
	first := w.post(w.aliceT, "/transactions", body, key)
	wantStatus(t, first, 201)
	readOnly := w.token(w.alice, domain.ScopeRead)
	wantProblem(t, w.post(readOnly, "/transactions", body, key), 403, "forbidden")
	for i := 0; i < 2; i++ {
		if i == 1 {
			testutil.Must(0, w.env.Svc.RestoreMember(w.env.Ctx, w.admin, w.net.ID, w.aliceM.ID, "restore"))
		}
		replay := w.post(w.aliceT, "/transactions", body, key)
		wantStatus(t, replay, 201)
		if replay.Body.String() != first.Body.String() || replay.Header().Get("Idempotent-Replayed") != "true" {
			t.Fatal("authorized retry did not preserve the original response")
		}
		if i == 0 {
			testutil.Must(0, w.env.Svc.SuspendMember(w.env.Ctx, w.admin, w.net.ID, w.aliceM.ID, "test"))
			wantProblem(t, w.post(w.aliceT, "/transactions", body, key), 403, "forbidden")
		}
	}
	var count int
	if err := w.env.Store.DB().QueryRow(`SELECT count(*) FROM transactions WHERE network_id=?`, w.net.ID).Scan(&count); err != nil || count != 1 {
		t.Fatalf("replay repeated the command: %d, %v", count, err)
	}
}

func TestReplayRechecksAdministratorRole(t *testing.T) {
	w := newWorld(t)
	testutil.Must(0, w.env.Svc.AssignRole(w.env.Ctx, domain.SystemActor("fixture"), w.net.ID, w.aliceM.ID, domain.RoleAdmin))
	body := m{"title": "Notice", "body": "Members only"}
	key := w.key()
	first := w.post(w.aliceT, "/admin/announcements", body, key)
	wantStatus(t, first, 201)
	wantStatus(t, w.do(req{method: "DELETE", path: "/api/v1/admin/members/" + w.aliceM.ID + "/roles/admin", token: w.adminT}), 200)
	wantProblem(t, w.post(w.aliceT, "/admin/announcements", body, key), 403, "forbidden")
}

func TestReplayPreservesDepartureAndReadOnlyExport(t *testing.T) {
	w := newWorld(t)
	readOnly := w.token(w.alice, domain.ScopeRead)
	body := m{"scope": "mine", "format": "json"}
	key := w.key()
	first := w.post(readOnly, "/exports", body, key)
	wantStatus(t, first, 201)
	replay := w.post(readOnly, "/exports", body, key)
	wantStatus(t, replay, 201)
	if replay.Body.String() != first.Body.String() {
		t.Fatal("read-only export replay changed")
	}
	key = w.key()
	first = w.post(w.aliceT, "/me/membership/departure", m{}, key)
	wantStatus(t, first, 200)
	replay = w.post(w.aliceT, "/me/membership/departure", m{}, key)
	wantStatus(t, replay, 200)
	if replay.Body.String() != first.Body.String() || replay.Header().Get("Idempotent-Replayed") != "true" {
		t.Fatal("self-departure retry was not preserved")
	}
}
