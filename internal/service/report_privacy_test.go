package service_test

import (
	"strings"
	"testing"

	"github.com/djshellshoxxx/circula/internal/domain"
	"github.com/djshellshoxxx/circula/internal/service"
	tu "github.com/djshellshoxxx/circula/internal/testutil"
)

func reportPrivacyWorld(t *testing.T, target string) (*world, string) {
	t.Helper()
	w := newWorld(t, nil)
	tu.Must(0, w.Svc.AssignRole(w.Ctx, w.admin, w.net.ID, w.bM.ID, domain.RoleAdmin))
	id := w.bM.ID
	switch target {
	case service.ReportListing:
		id = w.listing(w.b, "Subject's listing")
	case service.ReportTransaction:
		// The reporter is the payee here; pagination below covers a payer
		// reporter as well, so both other-party resolutions are exercised.
		id = tu.Must(w.propose(w.a, w.bM, service.ProposerIsPaid, 100)).ID
	}
	return w, tu.Must(w.report(w.a, target, id, "other", "Private report fixture"))
}

// FR-024 / SEC-025 / WAPI-01: gaining administrator authority does not make
// reports about oneself visible. Exclude those rows before both keyset limits.
func TestReportsExcludeSubjectsBeforePagination(t *testing.T) {
	w := newWorld(t, nil)
	other, _ := w.Member2(w.admin, "other-subject@example.org")
	old := tu.Must(w.report(w.a, service.ReportListing, w.listing(other, "Older visible listing"), "other", "Older visible report"))
	hiddenMember := tu.Must(w.report(w.a, service.ReportMember, w.bM.ID, "other", "Private member report"))
	newer := tu.Must(w.report(w.a, service.ReportListing, w.listing(other, "Newer visible listing"), "other", "Newer visible report"))
	hiddenListing := tu.Must(w.report(w.a, service.ReportListing, w.listing(w.b, "Subject's listing"), "other", "Private listing report"))
	hiddenTransaction := tu.Must(w.report(w.a, service.ReportTransaction,
		tu.Must(w.propose(w.a, w.bM, service.ProposerPays, 100)).ID, "other", "Private transaction report"))
	tu.Must(0, w.Svc.AssignRole(w.Ctx, w.admin, w.net.ID, w.bM.ID, domain.RoleAdmin))

	afterID := ""
	for _, want := range []string{newer, old} {
		page := tu.Must(w.Svc.ListReports(w.Ctx, w.b, w.net.ID, "open", service.Page{AfterID: afterID, Limit: 1}))
		if len(page) != 1 || page[0].ID != want {
			t.Fatalf("visible report page after %q: %+v", afterID, page)
		}
		afterID = page[0].ID
	}
	if page := tu.Must(w.Svc.ListReports(w.Ctx, w.b, w.net.ID, "", service.Page{AfterID: afterID, Limit: 1})); len(page) != 0 {
		t.Fatalf("report pagination did not finish: %+v", page)
	}

	afterSeq := int64(0)
	for _, want := range []string{old, newer} {
		events := tu.Must(w.Svc.ListAudit(w.Ctx, w.b, w.net.ID, service.AuditQuery{
			Action: domain.ActReportCreated, AfterSeq: afterSeq, Limit: 1,
		}))
		if len(events) != 1 || events[0].TargetID != want {
			t.Fatalf("visible audit page after %d: %+v", afterSeq, events)
		}
		afterSeq = events[0].Seq
	}
	if events := tu.Must(w.Svc.ListAudit(w.Ctx, w.b, w.net.ID, service.AuditQuery{
		Action: domain.ActReportCreated, AfterSeq: afterSeq, Limit: 1,
	})); len(events) != 0 {
		t.Fatalf("audit pagination did not finish: %+v", events)
	}
	for _, id := range []string{hiddenMember, hiddenListing, hiddenTransaction} {
		events := tu.Must(w.Svc.ListAudit(w.Ctx, w.b, w.net.ID, service.AuditQuery{TargetID: id, ActorID: w.aM.ID}))
		if len(events) != 0 {
			t.Fatal("subject audit filters returned a report event")
		}
	}
	for _, actor := range []domain.Actor{w.admin, domain.SystemActor("report fixture")} {
		queue := tu.Must(w.Svc.ListReports(w.Ctx, actor, w.net.ID, "", service.Page{Limit: 10}))
		events := tu.Must(w.Svc.ListAudit(w.Ctx, actor, w.net.ID, service.AuditQuery{Action: domain.ActReportCreated, Limit: 10}))
		if len(queue) != 5 || len(events) != 5 {
			t.Fatalf("independent authority lost report records: queue=%d audit=%d", len(queue), len(events))
		}
		for _, report := range queue {
			if report.ReporterID != w.aM.ID || report.ReporterName != w.aM.DisplayName {
				t.Fatal("independent administrator lost reporter information")
			}
		}
	}
}

func TestReportHandlingRequiresIndependentAdministrator(t *testing.T) {
	for _, target := range []string{service.ReportMember, service.ReportListing, service.ReportTransaction} {
		for _, system := range []bool{false, true} {
			name := target + "/administrator"
			if system {
				name = target + "/system"
			}
			t.Run(name, func(t *testing.T) {
				w, id := reportPrivacyWorld(t, target)
				in := service.HandleReportInput{ReportID: id, Note: "Reviewed independently", IdempotencyKey: w.Key()}
				before := tu.Snapshot(t, w.Store.DB())
				_, err := w.Svc.HandleReport(w.Ctx, w.b, w.net.ID, in)
				wantCode(t, err, domain.ErrNotFound)
				err = w.Svc.AuthorizeReplay(w.Ctx, w.b, w.net.ID, service.ReplayAuthorization{
					Permission: service.ReplayReportHandle, ReportID: id,
				})
				wantCode(t, err, domain.ErrNotFound)
				if changed := before.Changed(tu.Snapshot(t, w.Store.DB())); len(changed) != 0 {
					t.Fatalf("refused subject handling changed durable state: %v", changed)
				}
				handler := w.admin
				if system {
					handler = domain.SystemActor("report fixture")
				}
				done := tu.Must(w.Svc.HandleReport(w.Ctx, handler, w.net.ID, in))
				if done.Status != "handled" || done.ReporterID != w.aM.ID || done.HandlingNote != in.Note {
					t.Fatalf("independent handling failed: %+v", done)
				}
				before = tu.Snapshot(t, w.Store.DB())
				again := tu.Must(w.Svc.HandleReport(w.Ctx, handler, w.net.ID, in))
				tu.Must(0, w.Svc.AuthorizeReplay(w.Ctx, handler, w.net.ID, service.ReplayAuthorization{
					Permission: service.ReplayReportHandle, ReportID: id,
				}))
				if again.ID != done.ID {
					t.Fatal("durable replay changed the report identity")
				}
				if changed := before.Changed(tu.Snapshot(t, w.Store.DB())); len(changed) != 0 {
					t.Fatalf("durable report replay repeated state changes: %v", changed)
				}
				if hasKind(w.inbox(w.b), service.NotifReportHandled) || !hasKind(w.inbox(w.a), service.NotifReportHandled) {
					t.Fatal("report handling notification reached the wrong membership")
				}
				if queue := tu.Must(w.Svc.ListReports(w.Ctx, w.b, w.net.ID, "handled", service.Page{})); len(queue) != 0 {
					t.Fatal("handled reports disclosed the reporter to their subject")
				}
			})
		}
	}
}

func TestReportConflictRestrictsOnlyAuditBearingExports(t *testing.T) {
	for _, target := range []string{service.ReportMember, service.ReportListing, service.ReportTransaction} {
		t.Run(target, func(t *testing.T) {
			w, id := reportPrivacyWorld(t, target)
			for _, scope := range []service.ExportScope{service.ExportAudit, service.ExportFull} {
				for _, format := range []service.ExportFormat{service.FormatCSV, service.FormatJSON} {
					t.Run(string(scope)+"/"+string(format), func(t *testing.T) {
						before := tu.Snapshot(t, w.Store.DB())
						sink := &service.MemSink{}
						manifest, err := w.Svc.Export(w.Ctx, w.b, w.net.ID, scope, format, sink)
						wantCode(t, err, domain.ErrForbidden)
						if manifest != nil || len(sink.Names()) != 0 || strings.Contains(err.Error(), id) || strings.Contains(err.Error(), w.aM.ID) {
							t.Fatal("refused export created data or disclosed a report identifier")
						}
						wantCode(t, w.Svc.AuthorizeExportDownload(w.Ctx, w.b, w.net.ID, scope), domain.ErrForbidden)
						if changed := before.Changed(tu.Snapshot(t, w.Store.DB())); len(changed) != 0 {
							t.Fatalf("refused export changed durable state: %v", changed)
						}
						for _, actor := range []domain.Actor{w.admin, domain.SystemActor("report fixture")} {
							sink = &service.MemSink{}
							tu.Must(w.Svc.Export(w.Ctx, actor, w.net.ID, scope, format, sink))
							all := ""
							for _, name := range sink.Names() {
								all += string(sink.Bytes(name))
							}
							if !strings.Contains(all, id) || !strings.Contains(all, w.aM.ID) {
								t.Fatal("independent audit-bearing export lost report history")
							}
						}
					})
				}
			}
			for _, scope := range []service.ExportScope{service.ExportMine, service.ExportLedger, service.ExportMembers} {
				tu.Must(w.Svc.Export(w.Ctx, w.b, w.net.ID, scope, service.FormatJSON, &service.MemSink{}))
			}
			tu.Must(w.Svc.HandleReport(w.Ctx, w.admin, w.net.ID, service.HandleReportInput{ReportID: id, IdempotencyKey: w.Key()}))
			wantCode(t, w.Svc.AuthorizeExportDownload(w.Ctx, w.b, w.net.ID, service.ExportAudit), domain.ErrForbidden)
		})
	}
}

func TestReportConflictRecheckedAtExportSnapshot(t *testing.T) {
	for _, scope := range []service.ExportScope{service.ExportAudit, service.ExportFull} {
		t.Run(string(scope), func(t *testing.T) {
			w, boundary := exportBoundaryWorld(t)
			tu.Must(0, w.Svc.AssignRole(w.Ctx, w.admin, w.net.ID, w.bM.ID, domain.RoleAdmin))
			listing := w.listing(w.b, "Subject's listing")
			fileReport := func() { tu.Must(w.report(w.a, service.ReportListing, listing, "other", "Snapshot boundary fixture")) }
			boundary.beforeView.Store(&fileReport)
			sink := &service.MemSink{}
			manifest, err := w.Svc.Export(w.Ctx, w.b, w.net.ID, scope, service.FormatJSON, sink)
			wantCode(t, err, domain.ErrForbidden)
			if manifest != nil || len(sink.Names()) != 0 || boundary.beforeView.Load() != nil {
				t.Fatal("export did not recheck the subject conflict before creating snapshot files")
			}
		})
	}
}
