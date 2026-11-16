package service_test

import (
	"errors"
	"testing"

	"github.com/djshellshoxxx/circula/internal/domain"
	"github.com/djshellshoxxx/circula/internal/service"
	tu "github.com/djshellshoxxx/circula/internal/testutil"
)

// ARC-008 / A3-F23: a host-created identity is the target, not the actor.
func TestOperatorRegistrationAudit(t *testing.T) {
	e := tu.NewEnv(t)
	self := e.Person("self-registration")
	input := service.RegisterInput{Login: "operator-created", Password: pw}
	if _, err := e.Svc.RegisterPersonAsOperator(e.Ctx, self, input); !errors.Is(err, domain.ErrForbidden) {
		t.Fatalf("member provisioning: %v", err)
	}
	operator := domain.SystemActor("operator-create-request")
	created := tu.Must(e.Svc.RegisterPersonAsOperator(e.Ctx, operator, input))
	if _, err := e.Svc.RegisterPersonAsOperator(e.Ctx, operator, input); !errors.Is(err, domain.ErrConflict) {
		t.Fatalf("duplicate operator registration: %v", err)
	}
	for _, tc := range []struct {
		id, actorID, requestID string
		actorType              domain.ActorType
	}{
		{self.PersonID, self.PersonID, "", domain.ActorPerson},
		{created.ID, "", operator.RequestID, domain.ActorOperator},
	} {
		var typ domain.ActorType
		var actorID, requestID string
		var count int
		err := e.Store.DB().QueryRow(`SELECT actor_type,actor_id,request_id,count(*) FROM audit_events WHERE network_id='' AND action=? AND target_id=?`, domain.ActPersonRegistered, tc.id).Scan(&typ, &actorID, &requestID, &count)
		if err != nil || count != 1 || typ != tc.actorType || actorID != tc.actorID || requestID != tc.requestID {
			t.Fatalf("registration audit %s: %q/%q/%q count=%d err=%v", tc.id, typ, actorID, requestID, count, err)
		}
	}
	var persons, memberships, grants int
	if err := e.Store.DB().QueryRow(`SELECT (SELECT count(*) FROM persons),(SELECT count(*) FROM members),(SELECT count(*) FROM role_assignments)`).Scan(&persons, &memberships, &grants); err != nil {
		t.Fatal(err)
	}
	if persons != 2 || memberships != 0 || grants != 0 {
		t.Fatalf("operator registration changed authority: persons=%d memberships=%d grants=%d", persons, memberships, grants)
	}
	if _, _, err := e.Svc.Login(e.Ctx, input.Login, input.Password); err != nil {
		t.Fatal(err)
	}
}
