package service_test

import (
	"context"
	"sync/atomic"
	"testing"

	"github.com/djshellshoxxx/circula/internal/domain"
	"github.com/djshellshoxxx/circula/internal/service"
	"github.com/djshellshoxxx/circula/internal/store"
	tu "github.com/djshellshoxxx/circula/internal/testutil"
)

// exportBoundaryStore commits a real service operation after export admission
// and auditing, but before the read snapshot opens. It does not fabricate
// authorization failures or replace any production repository query.
type exportBoundaryStore struct {
	store.Store
	beforeView atomic.Pointer[func()]
}

func (s *exportBoundaryStore) View(ctx context.Context, fn func(store.Tx) error) error {
	if f := s.beforeView.Swap(nil); f != nil {
		(*f)()
	}
	return s.Store.View(ctx, fn)
}

func exportBoundaryWorld(t *testing.T) (*world, *exportBoundaryStore) {
	t.Helper()
	var boundary *exportBoundaryStore
	e := tu.NewEnvWith(t, func(st store.Store) store.Store {
		boundary = &exportBoundaryStore{Store: st}
		return boundary
	})
	w := &world{Env: e}
	w.admin = e.Person("admin@example.org")
	w.net, w.adminM = e.Network(w.admin, "demo", nil)
	w.a, w.aM = e.Member(w.admin, w.net, "ana@example.org")
	w.b, w.bM = e.Member(w.admin, w.net, "ben@example.org")
	return w, boundary
}

// SEC-009 / FR-091..094: admission does not preserve network-wide export
// authority if the role or membership changes before the snapshot opens.
// A reviewer left after demotion still cannot export administrator scopes.
func TestExportRechecksNetworkAuthorityAtSnapshot(t *testing.T) {
	for _, scope := range []service.ExportScope{
		service.ExportLedger, service.ExportMembers, service.ExportAudit, service.ExportFull,
	} {
		for _, format := range []service.ExportFormat{service.FormatCSV, service.FormatJSON} {
			for _, change := range []string{"demoted to reviewer", "suspended", "removed"} {
				t.Run(string(scope)+"/"+string(format)+"/"+change, func(t *testing.T) {
					w, boundary := exportBoundaryWorld(t)
					if err := w.Svc.AssignRole(w.Ctx, w.admin, w.net.ID, w.aM.ID, domain.RoleAdmin); err != nil {
						t.Fatal(err)
					}
					if err := w.Svc.AssignRole(w.Ctx, w.admin, w.net.ID, w.aM.ID, domain.RoleReviewer); err != nil {
						t.Fatal(err)
					}
					mutate := func() {
						var err error
						switch change {
						case "demoted to reviewer":
							err = w.Svc.RevokeRole(w.Ctx, w.admin, w.net.ID, w.aM.ID, domain.RoleAdmin)
						case "suspended":
							err = w.Svc.SuspendMember(w.Ctx, w.admin, w.net.ID, w.aM.ID, "export boundary fixture")
						case "removed":
							_, err = w.Svc.DepartMember(w.Ctx, w.admin, w.net.ID, service.DepartInput{
								MemberID: w.aM.ID, ReasonCode: domain.RemovalRuleBreach, Reason: "export boundary fixture",
							})
						}
						if err != nil {
							t.Fatalf("change export authority: %v", err)
						}
					}
					boundary.beforeView.Store(&mutate)
					sink := &memSink{}
					manifest, err := w.Svc.Export(w.Ctx, w.a, w.net.ID, scope, format, sink)
					if domain.CodeOf(err) != domain.CodeForbidden {
						t.Errorf("export after %s = %v, want forbidden", change, err)
					}
					if manifest != nil || len(sink.files) != 0 {
						t.Errorf("refused export created manifest %v and %d files", manifest != nil, len(sink.files))
					}
					if boundary.beforeView.Load() != nil {
						t.Fatal("export never reached the admission/snapshot boundary")
					}
				})
			}
		}
	}
}

// SEC-018 / SA2-17: the same erased-membership rule applies when creating a
// fresh export and when downloading an existing artifact. A live membership
// elsewhere keeps the credentials usable, making this a real member boundary.
func TestExportRefusesErasedMembership(t *testing.T) {
	for _, format := range []service.ExportFormat{service.FormatCSV, service.FormatJSON} {
		for _, eraseAtSnapshot := range []bool{false, true} {
			name := "before admission"
			if eraseAtSnapshot {
				name = "before snapshot"
			}
			t.Run(string(format)+"/"+name, func(t *testing.T) {
				w, boundary := exportBoundaryWorld(t)
				w.Network(w.a, "other", nil)
				w.depart(w.a, w.aM)
				erase := func() {
					rep := tu.Must(w.Svc.EraseMyData(w.Ctx, w.a, w.net.ID, ""))
					if rep.PersonErased {
						t.Fatal("other membership must preserve usable credentials")
					}
				}
				if eraseAtSnapshot {
					boundary.beforeView.Store(&erase)
				} else {
					erase()
				}
				sink := &memSink{}
				manifest, err := w.Svc.Export(w.Ctx, w.a, w.net.ID, service.ExportMine, format, sink)
				if domain.CodeOf(err) != domain.CodeForbidden {
					t.Errorf("export of erased membership = %v, want forbidden", err)
				}
				if manifest != nil || len(sink.files) != 0 {
					t.Errorf("refused export created manifest %v and %d files", manifest != nil, len(sink.files))
				}
				if boundary.beforeView.Load() != nil {
					t.Fatal("export never reached the admission/snapshot boundary")
				}
				wantEvents := 0 // already erased requests must not consume quota or add an audit row
				if eraseAtSnapshot {
					wantEvents = 1 // admission was valid; the recorded attempt remains truthful
				}
				events := tu.Must(w.Svc.ListAudit(w.Ctx, w.admin, w.net.ID, service.AuditQuery{Action: domain.ActExport}))
				if len(events) != wantEvents {
					t.Errorf("export audit events = %d, want %d", len(events), wantEvents)
				}
				if _, _, err := w.Svc.Login(w.Ctx, "ana@example.org", "correct horse battery"); err != nil {
					t.Fatalf("credentials for the other network stopped working: %v", err)
				}
			})
		}
	}
}

// DOM-039 / FR-090: losing active membership does not revoke the right to
// export one's own retained history when the membership has not been erased.
func TestExportOwnHistorySurvivesMembershipChangeAtSnapshot(t *testing.T) {
	for _, change := range []string{"suspended", "departed"} {
		t.Run(change, func(t *testing.T) {
			w, boundary := exportBoundaryWorld(t)
			mutate := func() {
				if change == "departed" {
					w.depart(w.a, w.aM)
				} else if err := w.Svc.SuspendMember(w.Ctx, w.admin, w.net.ID, w.aM.ID, "export boundary fixture"); err != nil {
					t.Fatal(err)
				}
			}
			boundary.beforeView.Store(&mutate)
			sink := &memSink{}
			manifest, err := w.Svc.Export(w.Ctx, w.a, w.net.ID, service.ExportMine, service.FormatJSON, sink)
			if err != nil || manifest == nil || len(sink.files) == 0 {
				t.Fatalf("own retained history after %s: manifest=%v, files=%d, err=%v", change, manifest != nil, len(sink.files), err)
			}
		})
	}
}

