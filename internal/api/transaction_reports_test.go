package api_test

import (
	"net/url"
	"strings"
	"testing"

	"github.com/djshellshoxxx/circula/internal/api"
	"github.com/djshellshoxxx/circula/internal/domain"
	"github.com/djshellshoxxx/circula/internal/service"
	tu "github.com/djshellshoxxx/circula/internal/testutil"
)

// SEC-025 / API-054: a transaction report links to the authorized, paginated
// set of disputes on that exact transaction, including later cases. Ordinary
// dispute responses do not reveal that a report exists or who filed it.
func TestTransactionReportRelatedDisputesAPI(t *testing.T) {
	w := newWorld(t)
	t.Cleanup(func() { api.CloseHandler(w.h) })
	e := w.env
	tx := e.Trade(w.net.ID, w.alice, w.bob, w.aliceM, w.bobM, 100)
	open := func(txID string, actor domain.Actor) *domain.Dispute {
		t.Helper()
		return tu.Must(e.Svc.OpenDispute(e.Ctx, actor, w.net.ID, service.OpenDisputeInput{
			TxID: txID, Category: "other", Statement: "Ordinary dispute fixture", IdempotencyKey: e.Key(),
		}))
	}
	older := open(tx.ID, w.alice)
	tu.Must(e.Svc.WithdrawDispute(e.Ctx, w.alice, w.net.ID, older.ID))
	const privateNote = "Private transaction report fixture"
	body, key := m{"target_type": "transaction", "target_id": tx.ID, "category": "other", "description": privateNote}, w.key()
	filed := w.post(w.aliceT, "/reports", body, key)
	wantStatus(t, filed, 201)
	report := decode[struct {
		ID                 string
		RelatedDisputesURL string `json:"related_disputes_url"`
	}](t, filed)
	if report.ID == "" || report.RelatedDisputesURL != "/api/v1/disputes?transaction_id="+tx.ID {
		t.Fatalf("report omitted its exact-transaction relation: %+v", report)
	}
	newer := open(tx.ID, w.bob)
	otherTx := e.Trade(w.net.ID, w.alice, w.bob, w.aliceM, w.bobM, 200)
	otherCase := open(otherTx.ID, w.alice)
	queue := w.get(w.adminT, "/admin/reports")
	wantStatus(t, queue, 200)
	queued := decode[page](t, queue)
	if len(queued.Data) != 1 || queued.Data[0]["related_disputes_url"] != report.RelatedDisputesURL {
		t.Fatalf("administrator queue omitted report relation: %+v", queued)
	}
	first := w.do(req{method: "GET", path: report.RelatedDisputesURL + "&limit=1", token: w.adminT})
	wantStatus(t, first, 200)
	pg := decode[page](t, first)
	if len(pg.Data) != 1 || pg.Data[0]["id"] != newer.ID || pg.NextCursor == nil {
		t.Fatalf("wrong first related-dispute page: %+v", pg)
	}
	cursor := url.QueryEscape(*pg.NextCursor)
	second := w.do(req{method: "GET", path: report.RelatedDisputesURL + "&limit=1&cursor=" + cursor, token: w.adminT})
	wantStatus(t, second, 200)
	last := decode[page](t, second)
	if len(last.Data) != 1 || last.Data[0]["id"] != older.ID || last.NextCursor != nil {
		t.Fatalf("older related dispute disappeared: %+v", last)
	}
	wantProblem(t, w.get(w.adminT, "/disputes?transaction_id="+otherTx.ID+"&limit=1&cursor="+cursor), 400, "invalid-cursor")
	wantProblem(t, w.get(w.adminT, "/disputes?transaction_id=ordinary-invalid-id"), 400, "invalid-query")
	for _, token := range []string{w.aliceT, w.bobT, w.adminT} {
		result := w.do(req{method: "GET", path: report.RelatedDisputesURL, token: token})
		wantStatus(t, result, 200)
		if got := decode[page](t, result); len(got.Data) != 2 {
			t.Fatalf("authorized reader lost a related dispute: %+v", got)
		}
		for _, absent := range []string{otherCase.ID, report.ID, privateNote, "reporter_member_id"} {
			if strings.Contains(result.Body.String(), absent) {
				t.Fatalf("ordinary dispute list contained unrelated or private report content: %q", absent)
			}
		}
	}
	unrelated, _ := e.Member(w.admin, w.net, "unrelated")
	result := w.do(req{method: "GET", path: report.RelatedDisputesURL, token: w.token(unrelated, domain.ScopeRead)})
	wantStatus(t, result, 200)
	if got := decode[page](t, result); len(got.Data) != 0 {
		t.Fatalf("transaction filter widened dispute authority: %+v", got)
	}
	wantProblem(t, w.post(w.adminT, "/reports", body, w.key()), 404, "not-found")
	before := tu.Snapshot(t, e.Store.DB())
	replay := w.post(w.aliceT, "/reports", body, key)
	wantStatus(t, replay, 201)
	if replay.Body.String() != filed.Body.String() || replay.Header().Get("Idempotent-Replayed") != "true" {
		t.Fatal("cached report replay changed its original result")
	}
	if changed := before.Changed(tu.Snapshot(t, e.Store.DB())); len(changed) != 0 {
		t.Fatalf("cached report replay changed durable state: %v", changed)
	}
}
