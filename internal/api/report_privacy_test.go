package api_test

import (
	"strings"
	"testing"

	"github.com/djshellshoxxx/circula/internal/api"
	"github.com/djshellshoxxx/circula/internal/domain"
	"github.com/djshellshoxxx/circula/internal/service"
	"github.com/djshellshoxxx/circula/internal/testutil"
)

// FR-024 / SEC-025: an ordinary member who becomes an administrator remains
// excluded from report identities and correlated audit records about them.
func TestReportSubjectPrivacyAcrossAPIViews(t *testing.T) {
	for _, target := range []string{service.ReportMember, service.ReportListing} {
		t.Run(target, func(t *testing.T) {
			w := newWorld(t)
			t.Cleanup(func() { api.CloseHandler(w.h) })
			targetID := w.bobM.ID
			if target == service.ReportListing {
				listing, _ := w.createListing(w.bobT, "Subject's listing")
				targetID = listing.ID
			}
			filed := w.post(w.aliceT, "/reports", m{"target_type": target, "target_id": targetID,
				"category": "other", "description": "Private report fixture"}, w.key())
			wantStatus(t, filed, 201)
			id := decode[struct{ ID string }](t, filed).ID
			testutil.Must(0, w.env.Svc.AssignRole(w.env.Ctx, w.admin, w.net.ID, w.bobM.ID, domain.RoleAdmin))
			for _, path := range []string{"/admin/reports?status=open&limit=1", "/admin/audit-events?action=report.created&limit=1"} {
				result := w.get(w.bobT, path)
				wantStatus(t, result, 200)
				if got := decode[page](t, result); len(got.Data) != 0 || got.NextCursor != nil {
					t.Fatalf("subject received a report row or cursor: %+v", got)
				}
				for _, hidden := range []string{id, w.aliceM.ID, "Private report fixture"} {
					if strings.Contains(result.Body.String(), hidden) {
						t.Fatal("subject response contained private report content")
					}
				}
				independent := w.get(w.adminT, path)
				wantStatus(t, independent, 200)
				if got := decode[page](t, independent); len(got.Data) != 1 || !strings.Contains(independent.Body.String(), w.aliceM.ID) {
					t.Fatal("independent administrator lost report visibility")
				}
			}
			before := testutil.Snapshot(t, w.env.Store.DB())
			result := w.post(w.bobT, "/admin/reports/"+id+"/handle", m{"note": "Subject's request"}, w.key())
			wantProblem(t, result, 404, "not-found")
			if changed := before.Changed(testutil.Snapshot(t, w.env.Store.DB())); len(changed) != 0 {
				t.Fatalf("subject handling changed durable state: %v", changed)
			}
			wantStatus(t, w.post(w.adminT, "/admin/reports/"+id+"/handle", m{"note": "Independent review"}, w.key()), 200)
			if got := decode[page](t, w.get(w.bobT, "/admin/reports?status=handled")); len(got.Data) != 0 {
				t.Fatal("handled queue disclosed a report to its subject")
			}
		})
	}
}

func TestReportHandlingReplayRechecksRoleAndPreservesSuccess(t *testing.T) {
	w := newWorld(t)
	t.Cleanup(func() { api.CloseHandler(w.h) })
	testutil.Must(0, w.env.Svc.AssignRole(w.env.Ctx, w.admin, w.net.ID, w.bobM.ID, domain.RoleAdmin))
	r := testutil.Must(w.env.Svc.CreateReport(w.env.Ctx, w.alice, w.net.ID, service.ReportInput{
		TargetType: service.ReportMember, TargetID: w.bobM.ID, Category: "other", IdempotencyKey: w.env.Key(),
	}))
	path, body, key := "/admin/reports/"+r.ID+"/handle", m{"note": "Independent review"}, w.key()
	first := w.post(w.adminT, path, body, key)
	wantStatus(t, first, 200)
	operator := domain.SystemActor("role fixture")
	testutil.Must(0, w.env.Svc.RevokeRole(w.env.Ctx, operator, w.net.ID, w.adminM.ID, domain.RoleAdmin))
	wantProblem(t, w.post(w.adminT, path, body, key), 403, "forbidden")
	testutil.Must(0, w.env.Svc.AssignRole(w.env.Ctx, operator, w.net.ID, w.adminM.ID, domain.RoleAdmin))
	before := testutil.Snapshot(t, w.env.Store.DB())
	replay := w.post(w.adminT, path, body, key)
	wantStatus(t, replay, 200)
	if replay.Body.String() != first.Body.String() || replay.Header().Get("Idempotent-Replayed") != "true" {
		t.Fatal("restored independent authority lost the original cached result")
	}
	if changed := before.Changed(testutil.Snapshot(t, w.env.Store.DB())); len(changed) != 0 {
		t.Fatalf("cached report replay repeated durable changes: %v", changed)
	}
}

func TestReportConflictRechecksRetainedAuditExports(t *testing.T) {
	for _, scope := range []string{"audit", "full"} {
		t.Run(scope, func(t *testing.T) {
			w := newWorld(t)
			t.Cleanup(func() { api.CloseHandler(w.h) })
			testutil.Must(0, w.env.Svc.AssignRole(w.env.Ctx, w.admin, w.net.ID, w.bobM.ID, domain.RoleAdmin))
			body, key := m{"scope": scope, "format": "json"}, w.key()
			first := w.post(w.bobT, "/exports", body, key)
			wantStatus(t, first, 201)
			export := decode[struct {
				ArtefactURL string `json:"artefact_url"`
			}](t, first)
			if export.ArtefactURL == "" {
				t.Fatal("export omitted its download URL")
			}
			testutil.Must(w.env.Svc.CreateReport(w.env.Ctx, w.alice, w.net.ID, service.ReportInput{
				TargetType: service.ReportMember, TargetID: w.bobM.ID, Category: "other", IdempotencyKey: w.env.Key(),
			}))
			wantProblem(t, w.do(req{method: "GET", path: export.ArtefactURL, token: w.bobT}), 403, "forbidden")
			wantProblem(t, w.post(w.bobT, "/exports", body, key), 403, "forbidden")
			wantProblem(t, w.post(w.bobT, "/exports", body, w.key()), 403, "forbidden")
			wantStatus(t, w.post(w.adminT, "/exports", body, w.key()), 201)
		})
	}
}
