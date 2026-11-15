package service_test

import (
	"testing"

	"github.com/djshellshoxxx/circula/internal/domain"
	"github.com/djshellshoxxx/circula/internal/service"
	tu "github.com/djshellshoxxx/circula/internal/testutil"
)

// SA2-17: the role is checked again when an artefact is downloaded.
func TestAuthorizeExportDownload(t *testing.T) {
	w := newWorld(t, nil)
	if err := w.Svc.AuthorizeExportDownload(w.Ctx, w.admin, w.net.ID, service.ExportFull); err != nil {
		t.Fatalf("admin: %v", err)
	}
	if err := w.Svc.AuthorizeExportDownload(w.Ctx, w.a, w.net.ID, service.ExportMine); err != nil {
		t.Fatalf("member own export: %v", err)
	}
	wantCode(t, w.Svc.AuthorizeExportDownload(w.Ctx, w.a, w.net.ID, service.ExportFull), domain.ErrForbidden)
	wantCode(t, w.Svc.AuthorizeExportDownload(w.Ctx, w.a, w.net.ID, "bogus"), domain.ErrValidation)
	// an administrator demoted after making the export is refused
	if err := w.Svc.AssignRole(w.Ctx, w.admin, w.net.ID, w.aM.ID, domain.RoleAdmin); err != nil {
		t.Fatal(err)
	}
	if err := w.Svc.AuthorizeExportDownload(w.Ctx, w.a, w.net.ID, service.ExportAudit); err != nil {
		t.Fatalf("new admin: %v", err)
	}
	if err := w.Svc.RevokeRole(w.Ctx, w.admin, w.net.ID, w.aM.ID, domain.RoleAdmin); err != nil {
		t.Fatal(err)
	}
	wantCode(t, w.Svc.AuthorizeExportDownload(w.Ctx, w.a, w.net.ID, service.ExportAudit), domain.ErrForbidden)
	// outsiders see nothing
	out := w.Person("out@example.org")
	wantCode(t, w.Svc.AuthorizeExportDownload(w.Ctx, out, w.net.ID, service.ExportMine), domain.ErrNotFound)
}

// SA2-17: erasure drops the person's stored replay records, runs the adapter
// hook, and the erased member can no longer authorise a download.
func TestErasurePurgesReplayRecordsAndExports(t *testing.T) {
	var hookNet, hookPerson string
	w := newWorldCfg(t, nil, func(c *service.Config) {
		c.OnMemberErased = func(n, p string) { hookNet, hookPerson = n, p }
	})
	prop := tu.Must(w.Svc.ProposeTransaction(w.Ctx, w.a, w.net.ID, service.ProposeInput{CounterpartyID: w.bM.ID,
		Direction: service.ProposerPays, Amount: 100, IdempotencyKey: w.Key()}))
	cnt := func(actor string) int {
		return count(t, w.Env, `SELECT COUNT(*) FROM idempotency_keys WHERE actor_id = ?`, actor)
	}
	if cnt(w.a.PersonID) == 0 {
		t.Fatal("no replay record to purge")
	}
	// the open proposal blocks erasure; withdraw it first
	tu.Must(w.Svc.CancelTransaction(w.Ctx, w.a, w.net.ID, service.CancelInput{TxID: prop.ID, IdempotencyKey: w.Key()}))
	w.depart(w.a, w.aM)
	tu.Must(w.Svc.EraseMyData(w.Ctx, w.a, w.net.ID, ""))
	if n := cnt(w.a.PersonID); n != 0 {
		t.Errorf("replay records of the erased person: %d", n)
	}
	if hookNet != w.net.ID || hookPerson != w.a.PersonID {
		t.Errorf("hook got %q %q", hookNet, hookPerson)
	}
	if err := w.Svc.AuthorizeExportDownload(w.Ctx, w.a, w.net.ID, service.ExportMine); err == nil {
		t.Error("erased member can still authorise a download")
	}
}
