package service_test

import (
	"errors"
	"testing"
	"time"

	"github.com/djshellshoxxx/circula/internal/domain"
	tu "github.com/djshellshoxxx/circula/internal/testutil"
)

// ARC-008 / OPS-020: CLI restore is an operator action, not an unattended
// system job. Member administrators retain their own identity when recording
// a restore; an ordinary member cannot create that audit assertion.
func TestRestoreAuditRecordsOperatorAndAdministratorIdentity(t *testing.T) {
	e := tu.NewEnv(t)
	admin := e.Person("restore-admin")
	net, adminMember := e.Network(admin, "restore-audit", nil)
	member, _ := e.Member(admin, net, "restore-member")
	backupCreated := e.Clock.Now().Add(-time.Hour)
	if err := e.Svc.RecordRestore(e.Ctx, member, net.ID, backupCreated); !errors.Is(err, domain.ErrForbidden) {
		t.Fatalf("ordinary member restore audit: %v", err)
	}
	var count int
	if err := e.Store.DB().QueryRow(`SELECT count(*) FROM audit_events WHERE action=?`, domain.ActRestore).Scan(&count); err != nil || count != 0 {
		t.Fatalf("refused audit wrote %d events: %v", count, err)
	}
	admin.RequestID = "administrator-restore"
	for _, tc := range []struct {
		actor    domain.Actor
		wantType domain.ActorType
		wantID   string
	}{
		{domain.SystemActor("operator-restore"), domain.ActorOperator, ""},
		{admin, domain.ActorAdmin, adminMember.ID},
	} {
		if err := e.Svc.RecordRestore(e.Ctx, tc.actor, net.ID, backupCreated); err != nil {
			t.Fatal(err)
		}
		var typ domain.ActorType
		var id, request, summary string
		err := e.Store.DB().QueryRow(`SELECT actor_type,actor_id,request_id,summary_after FROM audit_events WHERE network_id=? AND action=? ORDER BY seq DESC LIMIT 1`, net.ID, domain.ActRestore).Scan(&typ, &id, &request, &summary)
		if err != nil {
			t.Fatal(err)
		}
		if typ != tc.wantType || id != tc.wantID || request != tc.actor.RequestID {
			t.Errorf("restore identity = %q/%q/%q, want %q/%q/%q", typ, id, request, tc.wantType, tc.wantID, tc.actor.RequestID)
		}
		if want := `{"backup_created_at":"` + backupCreated.UTC().Format(time.RFC3339) + `","status":"restored"}`; summary != want {
			t.Errorf("restore summary = %q, want %q", summary, want)
		}
		rep := tu.Must(e.Svc.VerifyLedger(e.Ctx, domain.SystemActor("verify-restore-audit"), net.ID))
		if !rep.OK || rep.LedgerSeq != 0 {
			t.Fatalf("restore audit changed ledger or broke chain: %+v", rep)
		}
	}
}
