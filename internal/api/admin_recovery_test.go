package api_test

import (
	"net/http"
	"net/url"
	"testing"

	"github.com/djshellshoxxx/circula/internal/domain"
	"github.com/djshellshoxxx/circula/internal/service"
	"github.com/djshellshoxxx/circula/internal/testutil"
)

// GOV-021: both member lists of administrator recovery continue with the
// signed next_cursor until every request and every recorded step was read; a
// cursor of one list is refused by the other.
func TestGOV021AdminRecoveryListsContinue(t *testing.T) {
	w := newWorld(t)
	if _, err := w.env.Store.DB().Exec(`UPDATE members SET status = 'suspended' WHERE id = ?`, w.adminM.ID); err != nil {
		t.Fatal(err)
	}
	op := domain.SystemActor("cli-admin-recovery")
	const cycles = 7 // 7 requests, 14 recorded steps
	for i := 0; i < cycles; i++ {
		r := testutil.Must(w.env.Svc.RequestAdminRecovery(w.env.Ctx, w.alice, w.net.ID, service.AdminRecoveryInput{IdempotencyKey: w.key()}))
		testutil.Must(w.env.Svc.RefuseAdminRecovery(w.env.Ctx, op, service.AdminRecoveryDecision{Network: w.net.Slug, RequestID: r.ID,
			Operator: "Olga", Reason: "ticket"}))
	}
	type page struct {
		Data []struct {
			ID  string `json:"id"`
			Seq int64  `json:"seq"`
		} `json:"data"`
		NextCursor *string `json:"next_cursor"`
	}
	walk := func(path string) (int, string) {
		t.Helper()
		seen, cursor, first := map[string]bool{}, "", ""
		for i := 0; i < 20; i++ {
			q := path + "?limit=3"
			if cursor != "" {
				q += "&cursor=" + url.QueryEscape(cursor)
			}
			rec := w.get(w.bobT, q)
			if rec.Code != http.StatusOK {
				t.Fatalf("%s: %d %s", q, rec.Code, rec.Body.String())
			}
			p := decode[page](t, rec)
			for _, d := range p.Data {
				if seen[d.ID] {
					t.Fatalf("%s repeated %s", path, d.ID)
				}
				seen[d.ID] = true
			}
			if p.NextCursor == nil {
				return len(seen), first
			}
			cursor = *p.NextCursor
			if first == "" {
				first = cursor
			}
		}
		t.Fatalf("%s did not end", path)
		return 0, ""
	}
	n, reqCursor := walk("/admin-recovery/requests")
	if n != cycles {
		t.Fatalf("requests read = %d, want %d", n, cycles)
	}
	n, evCursor := walk("/admin-recovery/events")
	if n != 2*cycles {
		t.Fatalf("events read = %d, want %d", n, 2*cycles)
	}
	wantProblem(t, w.get(w.bobT, "/admin-recovery/events?cursor="+url.QueryEscape(reqCursor)), 400, "invalid-cursor")
	wantProblem(t, w.get(w.bobT, "/admin-recovery/requests?cursor="+url.QueryEscape(evCursor)), 400, "invalid-cursor")
}
