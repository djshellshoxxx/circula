package api_test

import (
	"maps"
	"slices"
	"strings"
	"testing"

	"github.com/djshellshoxxx/circula/internal/domain"
	"github.com/djshellshoxxx/circula/internal/service"
	"github.com/djshellshoxxx/circula/internal/testutil"
)

// A4 / API-042 / API-043 / FR-020: through the API a member reads another
// member of the same network as directory data only (id, display name,
// status), and only while that member is active. Suspended and departed
// members answer like a missing id; balances, history and the admin member
// list of others are not reachable.
func TestSameNetworkMemberReadsThroughAPI(t *testing.T) {
	w := newWorld(t)
	ctx, svc := w.env.Ctx, w.env.Svc
	w.env.Trade(w.net.ID, w.bob, w.admin, w.bobM, w.adminM, 4321) // bob's balance and a trade alice is no party to
	if err := svc.SetDefaultFlag(ctx, w.admin, w.net.ID, w.bobM.ID, true, "test"); err != nil {
		t.Fatal(err)
	}
	_, sueM := w.env.Member(w.admin, w.net, "sue")
	if err := svc.SuspendMember(ctx, w.admin, w.net.ID, sueM.ID, "test"); err != nil {
		t.Fatal(err)
	}
	dee, deeM := w.env.Member(w.admin, w.net, "dee")
	testutil.Must(svc.DepartMember(ctx, dee, w.net.ID, service.DepartInput{}))

	directoryKeys := []string{"display_name", "id", "status"}
	keys := func(o map[string]any) []string { return slices.Sorted(maps.Keys(o)) }

	rec := w.get(w.aliceT, "/members/"+w.bobM.ID)
	wantStatus(t, rec, 200)
	got := decode[map[string]any](t, rec)
	if !slices.Equal(keys(got), directoryKeys) || got["id"] != w.bobM.ID || got["status"] != "active" {
		t.Errorf("member read of another member = %v, want only %v", got, directoryKeys)
	}
	for _, leak := range []string{"4321", "43.21", "default", "person", "role", "joined"} {
		if strings.Contains(strings.ToLower(rec.Body.String()), leak) {
			t.Errorf("member detail leaks %q: %s", leak, rec.Body.String())
		}
	}

	list := decode[struct {
		Data []map[string]any `json:"data"`
	}](t, w.get(w.aliceT, "/members"))
	var ids []string
	for _, e := range list.Data {
		if !slices.Equal(keys(e), directoryKeys) || e["status"] != "active" {
			t.Errorf("directory item %v, want only %v with status active", e, directoryKeys)
		}
		ids = append(ids, e["id"].(string))
	}
	want := []string{w.adminM.ID, w.aliceM.ID, w.bobM.ID}
	slices.Sort(ids)
	slices.Sort(want)
	if !slices.Equal(ids, want) {
		t.Errorf("directory ids %v, want active members %v", ids, want)
	}

	missing := wantProblem(t, w.get(w.aliceT, "/members/no-such-member"), 404, "not-found")
	for name, id := range map[string]string{"suspended": sueM.ID, "departed": deeM.ID} {
		p := wantProblem(t, w.get(w.aliceT, "/members/"+id), 404, "not-found")
		if p.Title != missing.Title || p.Detail != missing.Detail {
			t.Errorf("%s member distinguishable from a missing one: %+v vs %+v", name, p, missing)
		}
	}

	// Own account only: alice has no postings; others' balances need admin.
	acc := decode[map[string]any](t, w.get(w.aliceT, "/accounts/me"))
	if acc["balance_minor"] != "0" {
		t.Errorf("alice's account = %v", acc)
	}
	entries := decode[struct {
		Data []map[string]any `json:"data"`
	}](t, w.get(w.aliceT, "/accounts/me/entries"))
	if len(entries.Data) != 0 {
		t.Errorf("alice's entries include others' postings: %v", entries.Data)
	}
	wantProblem(t, w.get(w.aliceT, "/admin/members"), 403, string(domain.CodeForbidden))
}
