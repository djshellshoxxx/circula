package service_test

import (
	"testing"

	"github.com/djshellshoxxx/circula/internal/domain"
	"github.com/djshellshoxxx/circula/internal/service"
	tu "github.com/djshellshoxxx/circula/internal/testutil"
)

const (
	ok = ""
	nf = domain.CodeNotFound
	fb = domain.CodeForbidden
)

type authzOp struct {
	name string
	run  func(w *world, who domain.Actor, tx *domain.Transaction) error
	// want per actor class
	want map[string]domain.Code
}

func all(outsider, otherAdmin, scoped, readTok, member, suspended, pending, departed, reviewer, admin domain.Code) map[string]domain.Code {
	return map[string]domain.Code{"outsider": outsider, "otherAdmin": otherAdmin, "scopedToken": scoped, "readToken": readTok,
		"member": member, "suspended": suspended, "pending": pending, "departed": departed, "reviewer": reviewer, "admin": admin}
}

// SEC-009 / API-017 / PER matrix: authorization is enforced server-side for every
// role x membership state x token type, and non-members cannot learn the network exists.
func TestAuthorizationMatrix(t *testing.T) {
	ops := []authzOp{
		{"balance", func(w *world, who domain.Actor, _ *domain.Transaction) error {
			_, err := w.Svc.GetBalance(w.Ctx, who, w.net.ID)
			return err
		},
			all(nf, nf, nf, ok, ok, ok, fb, ok, ok, ok)},
		{"history", func(w *world, who domain.Actor, _ *domain.Transaction) error {
			_, err := w.Svc.ListTransactions(w.Ctx, who, w.net.ID, nil, service.Page{})
			return err
		}, all(nf, nf, nf, ok, ok, ok, fb, ok, ok, ok)},
		{"directory", func(w *world, who domain.Actor, _ *domain.Transaction) error {
			_, err := w.Svc.ListDirectory(w.Ctx, who, w.net.ID, service.Page{})
			return err
		}, all(nf, nf, nf, ok, ok, fb, fb, fb, ok, ok)},
		{"propose", func(w *world, who domain.Actor, _ *domain.Transaction) error {
			_, err := w.Svc.ProposeTransaction(w.Ctx, who, w.net.ID, service.ProposeInput{CounterpartyID: w.bM.ID, Direction: service.ProposerPays, Amount: 1, IdempotencyKey: w.Key()})
			return err
		}, all(nf, nf, nf, fb, ok, fb, fb, fb, ok, ok)},
		{"create-listing", func(w *world, who domain.Actor, _ *domain.Transaction) error {
			_, err := w.Svc.CreateListing(w.Ctx, who, w.net.ID, service.ListingInput{Type: domain.ListingOffer, Title: "t", IdempotencyKey: w.Key()})
			return err
		}, all(nf, nf, nf, fb, ok, fb, fb, fb, ok, ok)},
		{"list-listings", func(w *world, who domain.Actor, _ *domain.Transaction) error {
			_, err := w.Svc.ListListings(w.Ctx, who, w.net.ID, service.ListingQuery{})
			return err
		}, all(nf, nf, nf, ok, ok, fb, fb, fb, ok, ok)},
		{"get-foreign-tx", func(w *world, who domain.Actor, tx *domain.Transaction) error {
			_, err := w.Svc.GetTransaction(w.Ctx, who, w.net.ID, tx.ID)
			return err
		}, all(nf, nf, nf, nf, nf, nf, nf, nf, nf, ok)},
		{"audit", func(w *world, who domain.Actor, _ *domain.Transaction) error {
			_, err := w.Svc.ListAudit(w.Ctx, who, w.net.ID, service.AuditQuery{})
			return err
		}, all(nf, nf, nf, ok, ok, ok, fb, ok, ok, ok)},
		{"export-mine", func(w *world, who domain.Actor, _ *domain.Transaction) error {
			_, err := w.Svc.Export(w.Ctx, who, w.net.ID, service.ExportMine, service.FormatCSV, &service.MemSink{})
			return err
		}, all(nf, nf, nf, ok, ok, ok, fb, ok, ok, ok)},
		{"export-ledger", func(w *world, who domain.Actor, _ *domain.Transaction) error {
			_, err := w.Svc.Export(w.Ctx, who, w.net.ID, service.ExportLedger, service.FormatCSV, &service.MemSink{})
			return err
		}, all(nf, nf, nf, fb, fb, fb, fb, fb, fb, ok)},
		{"admin-list-members", func(w *world, who domain.Actor, _ *domain.Transaction) error {
			_, err := w.Svc.AdminListMembers(w.Ctx, who, w.net.ID, nil, service.Page{})
			return err
		}, all(nf, nf, nf, fb, fb, fb, fb, fb, fb, ok)},
		{"verify", func(w *world, who domain.Actor, _ *domain.Transaction) error {
			_, err := w.Svc.VerifyLedger(w.Ctx, who, w.net.ID)
			return err
		}, all(nf, nf, nf, fb, fb, fb, fb, fb, fb, ok)},
		{"invite", func(w *world, who domain.Actor, _ *domain.Transaction) error {
			_, _, err := w.Svc.CreateInvitation(w.Ctx, who, w.net.ID, service.InviteInput{})
			return err
		}, all(nf, nf, nf, fb, fb, fb, fb, fb, fb, ok)},
		{"adjust", func(w *world, who domain.Actor, _ *domain.Transaction) error {
			_, err := w.Svc.CreateAdjustment(w.Ctx, who, w.net.ID, service.AdjustInput{MemberID: w.bM.ID, Amount: 1, Direction: service.AdjustCredit, Reason: "r", IdempotencyKey: w.Key()})
			return err
		}, all(nf, nf, nf, fb, fb, fb, fb, fb, fb, ok)},
		{"set-limit", func(w *world, who domain.Actor, _ *domain.Transaction) error {
			l := int64(1)
			return w.Svc.SetCreditLimit(w.Ctx, who, w.net.ID, service.SetLimitInput{MemberID: w.bM.ID, Limit: &l, Reason: "r"})
		}, all(nf, nf, nf, fb, fb, fb, fb, fb, fb, ok)},
		{"assign-role", func(w *world, who domain.Actor, _ *domain.Transaction) error {
			return w.Svc.AssignRole(w.Ctx, who, w.net.ID, w.bM.ID, domain.RoleReviewer)
		}, all(nf, nf, nf, fb, fb, fb, fb, fb, fb, ok)},
		{"update-policy", func(w *world, who domain.Actor, _ *domain.Transaction) error {
			p := w.net.Policy
			p.ProposalExpiryDays = 9
			_, err := w.Svc.UpdatePolicy(w.Ctx, who, w.net.ID, service.PolicyChange{Policy: p, Reason: "r", IdempotencyKey: w.Key()})
			return err
		}, all(nf, nf, nf, fb, fb, fb, fb, fb, fb, ok)},
		{"reverse", func(w *world, who domain.Actor, tx *domain.Transaction) error {
			_, err := w.Svc.ReverseTransaction(w.Ctx, who, w.net.ID, service.ReverseInput{TxID: tx.ID, Amount: 1, Reason: "r", IdempotencyKey: w.Key()})
			return err
		}, all(nf, nf, nf, fb, fb, fb, fb, fb, fb, ok)},
		{"suspend-other", func(w *world, who domain.Actor, _ *domain.Transaction) error {
			return w.Svc.SuspendMember(w.Ctx, who, w.net.ID, w.bM.ID, "r")
		}, all(nf, nf, nf, fb, fb, fb, fb, fb, fb, ok)},
		{"list-disputes", func(w *world, who domain.Actor, _ *domain.Transaction) error {
			_, err := w.Svc.ListDisputes(w.Ctx, who, w.net.ID, nil, service.Page{})
			return err
		}, all(nf, nf, nf, ok, ok, ok, fb, ok, ok, ok)},
		// the admin's earlier propose left a pending proposal, so closure is authorised but blocked
		{"close-network", func(w *world, who domain.Actor, _ *domain.Transaction) error {
			return w.Svc.CloseNetwork(w.Ctx, who, w.net.ID, "r")
		}, all(nf, nf, nf, fb, fb, fb, fb, fb, fb, domain.CodeBlocked)},
	}
	// "suspend-other" and "close-network" mutate shared state, so they run last.
	last := map[string]bool{"suspend-other": true, "close-network": true}
	ordered := make([]authzOp, 0, len(ops))
	for _, pass := range []bool{false, true} {
		for _, op := range ops {
			if last[op.name] == pass {
				ordered = append(ordered, op)
			}
		}
	}
	classes := []string{"outsider", "otherAdmin", "scopedToken", "readToken", "member", "suspended", "pending", "departed", "reviewer", "admin"}
	for _, class := range classes {
		t.Run(class, func(t *testing.T) {
			w := newWorld(t, func(p *domain.Policy) { p.JoinMode = domain.JoinApproval })
			tx := w.Trade(w.net.ID, w.a, w.b, w.aM, w.bM, 100)
			who := actorFor(t, w, class)
			for _, op := range ordered {
				err := op.run(w, who, tx)
				if got := domain.CodeOf(err); got != op.want[class] {
					t.Errorf("%s: code = %q (err=%v), want %q", op.name, got, err, op.want[class])
				}
			}
		})
	}
}

func actorFor(t *testing.T, w *world, class string) domain.Actor {
	switch class {
	case "outsider":
		return w.Person("outsider@example.org")
	case "otherAdmin":
		other := w.Person("otheradmin@example.org")
		w.Network(other, "other", nil)
		return other
	case "scopedToken", "readToken":
		c, _ := w.Member2(w.admin, "carol@example.org")
		if class == "readToken" {
			tok, _ := tu.Must2(w.Svc.CreateAPIToken(w.Ctx, c, service.TokenInput{Name: "ro", Scopes: []domain.Scope{domain.ScopeRead}}))
			return tu.Must(w.Svc.AuthenticateToken(w.Ctx, tok))
		}
		other := w.Person("otheradmin@example.org")
		on, _ := w.Network(other, "other", nil)
		oc, _ := w.Member(other, on, "carol-b@example.org")
		_ = oc
		// carol joins the other network too, then holds a token scoped to it
		tokenInv, _ := tu.Must2(w.Svc.CreateInvitation(w.Ctx, other, on.ID, service.InviteInput{}))
		tu.Must(w.Svc.RedeemInvitation(w.Ctx, c, tokenInv, "carol"))
		tok, _ := tu.Must2(w.Svc.CreateAPIToken(w.Ctx, c, service.TokenInput{Name: "scoped", Scopes: []domain.Scope{domain.ScopeRead, domain.ScopeWrite}, NetworkID: on.ID}))
		return tu.Must(w.Svc.AuthenticateToken(w.Ctx, tok))
	case "member":
		c, _ := w.Member2(w.admin, "carol@example.org")
		return c
	case "suspended":
		c, m := w.Member2(w.admin, "dave@example.org")
		tu.Must(0, w.Svc.SuspendMember(w.Ctx, w.admin, w.net.ID, m.ID, "r"))
		return c
	case "pending":
		p, _ := w.joinPending("pending@example.org")
		return p
	case "departed":
		c, _ := w.Member2(w.admin, "gone@example.org")
		tu.Must(w.Svc.DepartMember(w.Ctx, c, w.net.ID, service.DepartInput{}))
		return c
	case "reviewer":
		r, _ := w.reviewer()
		return r
	case "admin":
		return w.admin
	}
	t.Fatalf("unknown class %s", class)
	return domain.Actor{}
}

// Cross-network identifiers are indistinguishable from missing ones (DOM-001, FR-006).
func TestCrossNetworkIdsAreNotFound(t *testing.T) {
	w := newWorld(t, nil)
	other := w.Person("other@example.org")
	on, om := w.Network(other, "other-net", nil)
	oc, ocM := w.Member(other, on, "oc@example.org")
	otx := w.Trade(on.ID, oc, other, ocM, om, 0+1)

	_, e1 := w.Svc.GetTransaction(w.Ctx, w.a, w.net.ID, otx.ID)       // other network's tx via my network
	_, e2 := w.Svc.GetTransaction(w.Ctx, w.a, w.net.ID, "no-such-id") // nonexistent
	wantCode(t, e1, domain.ErrNotFound)
	if e1.Error() != e2.Error() {
		t.Errorf("distinguishable: %q vs %q", e1, e2)
	}
	_, err := w.propose(w.a, ocM, service.ProposerPays, 1) // counterparty in another network
	wantCode(t, err, domain.ErrNotFound)
	_, err = w.Svc.AcceptTransaction(w.Ctx, w.a, w.net.ID, service.TxRef{TxID: otx.ID, IdempotencyKey: w.Key()})
	wantCode(t, err, domain.ErrNotFound)
	_, err = w.Svc.GetNetwork(w.Ctx, w.a, on.ID)
	wantCode(t, err, domain.ErrNotFound)
	err = w.Svc.SuspendMember(w.Ctx, w.admin, w.net.ID, ocM.ID, "x")
	wantCode(t, err, domain.ErrNotFound)
	// listings and disputes likewise
	ol := tu.Must(w.Svc.CreateListing(w.Ctx, oc, on.ID, service.ListingInput{Type: domain.ListingOffer, Title: "x", IdempotencyKey: w.Key()}))
	tt := domain.ListingPaused
	_, err = w.Svc.UpdateListing(w.Ctx, w.a, w.net.ID, service.ListingUpdate{ListingID: ol.ID, Status: &tt})
	wantCode(t, err, domain.ErrNotFound)
}

// Members cannot see other members' balances; the admin read is audited.
func TestAdminBalanceViewIsAudited(t *testing.T) {
	w := newWorld(t, nil)
	w.Trade(w.net.ID, w.a, w.b, w.aM, w.bM, 700)
	bv := tu.Must(w.Svc.AdminGetMemberBalance(w.Ctx, w.admin, w.net.ID, w.aM.ID))
	if bv.Balance != -700 {
		t.Errorf("balance = %d", bv.Balance)
	}
	if _, err := w.Svc.AdminGetMemberBalance(w.Ctx, w.b, w.net.ID, w.aM.ID); err == nil {
		t.Error("member read another member's balance")
	}
	evs := tu.Must(w.Svc.ListAudit(w.Ctx, w.admin, w.net.ID, service.AuditQuery{Action: domain.ActAccountViewed}))
	if len(evs) != 1 {
		t.Errorf("account.viewed events = %d", len(evs))
	}
}

// Anonymous / empty actors are rejected before any data access.
func TestEmptyActorRejected(t *testing.T) {
	w := newWorld(t, nil)
	_, err := w.Svc.GetBalance(w.Ctx, domain.Actor{Scopes: []domain.Scope{domain.ScopeRead}}, w.net.ID)
	wantCode(t, err, domain.ErrUnauthenticated)
	_, err = w.Svc.GetBalance(w.Ctx, domain.Actor{PersonID: w.a.PersonID}, w.net.ID) // no scopes
	wantCode(t, err, domain.ErrForbidden)
}
