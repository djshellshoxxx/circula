package service_test

import (
	"testing"

	"github.com/djshellshoxxx/circula/internal/domain"
	"github.com/djshellshoxxx/circula/internal/service"
	tu "github.com/djshellshoxxx/circula/internal/testutil"
)

// DOM-003 / FR-013 and FR-011 / FR-012 / FR-010.
func TestJoinModes(t *testing.T) {
	t.Run("invite-only refuses open join", func(t *testing.T) {
		w := newWorld(t, nil)
		_, err := w.Svc.JoinNetwork(w.Ctx, w.Person("p@example.org"), w.net.ID, "P")
		wantCode(t, err, domain.ErrForbidden)
	})
	t.Run("approval", func(t *testing.T) {
		w := newWorld(t, func(p *domain.Policy) { p.JoinMode = domain.JoinApproval })
		p, pM := w.joinPending("p@example.org")
		// FR-017: second join returns the same membership
		again := tu.Must(w.Svc.JoinNetwork(w.Ctx, p, w.net.ID, "Other Name"))
		if again.ID != pM.ID {
			t.Error("duplicate join created a second member")
		}
		// a pending member can do nothing
		_, err := w.propose(p, w.aM, service.ProposerPays, 1)
		wantCode(t, err, domain.ErrForbidden)
		_, err = w.Svc.CreateListing(w.Ctx, p, w.net.ID, service.ListingInput{Type: domain.ListingOffer, Title: "x", IdempotencyKey: w.Key()})
		wantCode(t, err, domain.ErrForbidden)
		_, err = w.propose(w.a, pM, service.ProposerPays, 1)
		wantCode(t, err, domain.ErrCounterpartyInact)
		got := tu.Must(w.Svc.DecideJoinRequest(w.Ctx, w.admin, w.net.ID, pM.ID, true, ""))
		if got.Status != domain.MemberActive {
			t.Error("not approved")
		}
		_, err = w.Svc.DecideJoinRequest(w.Ctx, w.admin, w.net.ID, pM.ID, true, "") // not pending any more
		wantCode(t, err, domain.ErrInvalidState)
	})
	t.Run("denied", func(t *testing.T) {
		w := newWorld(t, func(p *domain.Policy) { p.JoinMode = domain.JoinApproval })
		_, pM := w.joinPending("p@example.org")
		got := tu.Must(w.Svc.DecideJoinRequest(w.Ctx, w.admin, w.net.ID, pM.ID, false, "not in this town"))
		if got.Status != domain.MemberDeparted {
			t.Errorf("status = %s", got.Status)
		}
	})
	t.Run("open", func(t *testing.T) {
		w := newWorld(t, func(p *domain.Policy) { p.JoinMode = domain.JoinOpen })
		m := tu.Must(w.Svc.JoinNetwork(w.Ctx, w.Person("p@example.org"), w.net.ID, "P"))
		if m.Status != domain.MemberActive {
			t.Errorf("status = %s", m.Status)
		}
	})
}

// FR-010: single-use, expiring, hashed; indistinguishable failures.
func TestInvitations(t *testing.T) {
	w := newWorld(t, nil)
	tok, inv := tu.Must2(w.Svc.CreateInvitation(w.Ctx, w.admin, w.net.ID, service.InviteInput{}))
	var stored string
	_ = w.Store.DB().QueryRow(`SELECT token_hash FROM invitations WHERE id = ?`, inv.ID).Scan(&stored)
	if stored == "" || stored == tok {
		t.Errorf("token must be stored hashed, stored=%q", stored)
	}
	p := w.Person("p@example.org")
	m := tu.Must(w.Svc.RedeemInvitation(w.Ctx, p, tok, "Pat"))
	if m.Status != domain.MemberActive || m.JoinMethod != "invite" {
		t.Fatalf("member = %+v", m)
	}
	// re-redeeming by the same person returns the same membership (idempotent)
	if again := tu.Must(w.Svc.RedeemInvitation(w.Ctx, p, tok, "Pat")); again.ID != m.ID {
		t.Error("redeem is not idempotent for the same person")
	}
	// used token for someone else == unknown token
	q := w.Person("q@example.org")
	_, errUsed := w.Svc.RedeemInvitation(w.Ctx, q, tok, "Q")
	_, errUnknown := w.Svc.RedeemInvitation(w.Ctx, q, "inv_nope", "Q")
	wantCode(t, errUsed, domain.ErrNotFound)
	if errUsed.Error() != errUnknown.Error() {
		t.Errorf("used and unknown tokens distinguishable: %q vs %q", errUsed, errUnknown)
	}
	// expired
	tok2, _ := tu.Must2(w.Svc.CreateInvitation(w.Ctx, w.admin, w.net.ID, service.InviteInput{TTL: 3600e9}))
	w.Clock.Advance(2 * 3600e9)
	_, err := w.Svc.RedeemInvitation(w.Ctx, q, tok2, "Q")
	wantCode(t, err, domain.ErrNotFound)
	// revoked
	tok3, inv3 := tu.Must2(w.Svc.CreateInvitation(w.Ctx, w.admin, w.net.ID, service.InviteInput{}))
	tu.Must(0, w.Svc.RevokeInvitation(w.Ctx, w.admin, w.net.ID, inv3.ID))
	_, err = w.Svc.RedeemInvitation(w.Ctx, q, tok3, "Q")
	wantCode(t, err, domain.ErrNotFound)
	if n := len(tu.Must(w.Svc.ListInvitations(w.Ctx, w.admin, w.net.ID))); n != 5 {
		t.Errorf("invitations listed = %d", n)
	}
	// members cannot invite
	_, _, err = w.Svc.CreateInvitation(w.Ctx, w.a, w.net.ID, service.InviteInput{})
	wantCode(t, err, domain.ErrForbidden)
}

// FR-069 / SEC-010.
func TestAdministratorSuccession(t *testing.T) {
	w := newWorld(t, nil)
	err := w.Svc.RevokeRole(w.Ctx, w.admin, w.net.ID, w.adminM.ID, domain.RoleAdmin)
	wantCode(t, err, domain.ErrBlocked)
	err = w.Svc.SuspendMember(w.Ctx, w.admin, w.net.ID, w.adminM.ID, "x") // also self
	wantCode(t, err, domain.ErrForbidden)
	err = w.Svc.AssignRole(w.Ctx, w.admin, w.net.ID, w.adminM.ID, domain.RoleReviewer) // self-grant
	wantCode(t, err, domain.ErrForbidden)
	_, m2 := w.Member2(w.admin, "adm2@example.org")
	tu.Must(0, w.Svc.AssignRole(w.Ctx, w.admin, w.net.ID, m2.ID, domain.RoleAdmin))
	err = w.Svc.AssignRole(w.Ctx, w.admin, w.net.ID, m2.ID, domain.RoleAdmin) // already holds
	wantCode(t, err, domain.ErrInvalidState)
	tu.Must(0, w.Svc.RevokeRole(w.Ctx, w.admin, w.net.ID, w.adminM.ID, domain.RoleAdmin)) // now fine
	// the second administrator is the last: protected, including suspension by... nobody else
	err = w.Svc.RevokeRole(w.Ctx, w.admin, w.net.ID, m2.ID, domain.RoleAdmin)
	wantCode(t, err, domain.ErrForbidden) // original admin no longer admin
	err = w.Svc.AssignRole(w.Ctx, w.admin, w.net.ID, w.aM.ID, "emperor")
	wantCode(t, err, domain.ErrForbidden)
}

// DOM-003/039/004: suspension.
func TestSuspension(t *testing.T) {
	w := newWorld(t, nil)
	tx := w.Trade(w.net.ID, w.a, w.b, w.aM, w.bM, 1000)
	pendingToB := tu.Must(w.propose(w.a, w.bM, service.ProposerPays, 100))
	pendingFromB := tu.Must(w.propose(w.b, w.aM, service.ProposerPays, 100))
	before := w.balance(w.b)
	tu.Must(0, w.Svc.SuspendMember(w.Ctx, w.admin, w.net.ID, w.bM.ID, "investigating"))
	if after := w.balance(w.b); after.Balance != before.Balance || after.AccountState != domain.AccountFrozen {
		t.Errorf("suspension changed balance or state: %+v", after)
	}
	_, err := w.propose(w.b, w.aM, service.ProposerPays, 1)
	wantCode(t, err, domain.ErrForbidden)
	_, err = w.accept(w.b, pendingToB.ID)
	wantCode(t, err, domain.ErrForbidden)
	_, err = w.accept(w.a, pendingFromB.ID) // counterparty suspended between proposal and accept
	wantCode(t, err, domain.ErrCounterpartyInact)
	_, err = w.open(w.b, tx.ID)
	wantCode(t, err, domain.ErrForbidden)
	_, err = w.Svc.ListDirectory(w.Ctx, w.b, w.net.ID, service.Page{})
	wantCode(t, err, domain.ErrForbidden)
	// exits and reads remain: decline, withdraw, balance, history, export
	tu.Must(w.Svc.DeclineTransaction(w.Ctx, w.b, w.net.ID, service.TxRef{TxID: pendingToB.ID, IdempotencyKey: w.Key()}))
	tu.Must(w.Svc.CancelTransaction(w.Ctx, w.b, w.net.ID, service.CancelInput{TxID: pendingFromB.ID, IdempotencyKey: w.Key()}))
	tu.Must(w.Svc.ListTransactions(w.Ctx, w.b, w.net.ID, nil, service.Page{}))
	tu.Must(w.Svc.Export(w.Ctx, w.b, w.net.ID, service.ExportMine, service.FormatJSON, &service.MemSink{}))
	// restore
	tu.Must(0, w.Svc.RestoreMember(w.Ctx, w.admin, w.net.ID, w.bM.ID, "cleared"))
	tu.Must(w.propose(w.b, w.aM, service.ProposerPays, 1))
	err = w.Svc.RestoreMember(w.Ctx, w.admin, w.net.ID, w.bM.ID, "again") // active -> active
	wantCode(t, err, domain.ErrInvalidState)
}
