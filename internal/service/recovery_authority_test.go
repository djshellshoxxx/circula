package service_test

import (
	"errors"
	"reflect"
	"testing"
	"time"

	"github.com/djshellshoxxx/circula/internal/domain"
	"github.com/djshellshoxxx/circula/internal/service"
	"github.com/djshellshoxxx/circula/internal/store"
	tu "github.com/djshellshoxxx/circula/internal/testutil"
)

// AUTH-01 / SA2-01: departure and local anonymisation retain history authority
// for the same person. Network-local recovery must respect that authority.
func TestAdminResetRetainedMembershipAuthority(t *testing.T) {
	for _, anonymised := range []bool{false, true} {
		name := "departed"
		if anonymised {
			name = "anonymised"
		}
		t.Run(name, func(t *testing.T) {
			e := tu.NewEnv(t)
			admin := e.Person("admin@example.org")
			net, _ := e.Network(admin, "current-network", nil)
			member, memberRow := e.Member(admin, net, "member@example.org")
			otherAdmin := e.Person("other-admin@example.org")
			otherNet, _ := e.Network(otherAdmin, "former-network", nil)
			invite, _ := tu.Must2(e.Svc.CreateInvitation(e.Ctx, otherAdmin, otherNet.ID, service.InviteInput{}))
			former := tu.Must(e.Svc.RedeemInvitation(e.Ctx, member, invite, "Member"))
			tu.Must(e.Svc.DepartMember(e.Ctx, otherAdmin, otherNet.ID, service.DepartInput{
				MemberID: former.ID, ReasonCode: domain.RemovalRuleBreach, Reason: "Membership ended"}))
			if anonymised {
				report := tu.Must(e.Svc.AnonymiseMember(e.Ctx, otherAdmin, otherNet.ID, former.ID, "Member requested anonymisation"))
				if report.PersonErased {
					t.Fatal("local anonymisation erased a person active in another network")
				}
			}
			// The existing owner retains ordinary history access. No reset
			// credential is used to read another network's data in this test.
			tu.Must(e.Svc.ListTransactions(e.Ctx, member, otherNet.ID, nil, service.Page{}))
			tu.Must(e.Svc.ListEntries(e.Ctx, member, otherNet.ID, service.EntryQuery{}))

			codesFor(t, e, member)
			secret, _ := enrol(t, e, member)
			challenge := login(e, "member@example.org")
			tu.Must2(e.Svc.CompleteMFA(e.Ctx, challenge.MFAChallenge, totpAt(t, secret, now(e))))
			login(e, "member@example.org") // One outstanding MFA challenge.
			tu.Must2(e.Svc.CreateAPIToken(e.Ctx, member, service.TokenInput{
				Name: "Owner token", Scopes: []domain.Scope{domain.ScopeRead}}))
			// Inert persisted throttle state must also survive a refusal.
			tu.Must(0, e.Store.Update(e.Ctx, func(tx store.Tx) error {
				for _, key := range []string{"member@example.org", "recover: member@example.org", "recgen: " + member.PersonID} {
					if err := tx.PutThrottle(key, 1, now(e).Add(time.Minute)); err != nil {
						return err
					}
				}
				return nil
			}))
			tables := []string{"persons", "sessions", "api_tokens", "account_recovery_codes", "login_throttle",
				"person_mfa", "recovery_codes", "mfa_challenges", "security_notices", "audit_events", "audit_heads"}
			before := authorityTableRows(t, e, tables...)
			temporary, err := e.Svc.AdminResetPassword(e.Ctx, admin, net.ID, memberRow.ID, "Member requested account recovery")
			if !errors.Is(err, domain.ErrForbidden) || temporary != "" {
				t.Fatalf("retained authority must refuse local recovery without credentials: %v", err)
			}
			after := authorityTableRows(t, e, tables...)
			for _, table := range tables {
				if !reflect.DeepEqual(after[table], before[table]) {
					t.Errorf("refused local recovery changed %s", table)
				}
			}

			// The platform operator remains authorized, with an audit event and
			// the enrolled second factor preserved exactly as stored.
			mfaBefore := authorityTableRows(t, e, "person_mfa", "recovery_codes")
			temporary = tu.Must(e.Svc.OperatorResetPassword(e.Ctx, domain.SystemActor("recovery-test"),
				"member@example.org", "Verified operator recovery"))
			if temporary == "" {
				t.Fatal("operator recovery did not issue a temporary password")
			}
			if !reflect.DeepEqual(authorityTableRows(t, e, "person_mfa", "recovery_codes"), mfaBefore) {
				t.Fatal("password recovery changed the enrolled second factor")
			}
			if count(t, e, `SELECT COUNT(*) FROM audit_events WHERE action = ? AND actor_type = ? AND reason = ?`,
				domain.ActPersonPasswordReset, domain.ActorOperator, "Verified operator recovery") != 1 {
				t.Fatal("operator recovery audit missing")
			}
		})
	}
}

func TestSingleNetworkAdminResetPreservesMFA(t *testing.T) {
	e := tu.NewEnv(t)
	admin := e.Person("admin@example.org")
	net, _ := e.Network(admin, "single-network", nil)
	member, memberRow := e.Member(admin, net, "member@example.org")
	enrol(t, e, member)
	before := authorityTableRows(t, e, "person_mfa", "recovery_codes")
	if temporary := tu.Must(e.Svc.AdminResetPassword(e.Ctx, admin, net.ID, memberRow.ID, "Verified member recovery")); temporary == "" {
		t.Fatal("single-network recovery did not issue a temporary password")
	}
	if !reflect.DeepEqual(authorityTableRows(t, e, "person_mfa", "recovery_codes"), before) {
		t.Fatal("network-local password recovery changed the enrolled second factor")
	}
}
