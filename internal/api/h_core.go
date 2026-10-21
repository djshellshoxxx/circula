package api

import (
	"net/http"

	"github.com/djshellshoxxx/circula/internal/domain"
	"github.com/djshellshoxxx/circula/internal/service"
)

// notImpl answers 501 for features the service layer does not provide.
func (s *server) notImpl(what string) handlerFunc {
	return func(c *reqCtx) error { return notImplemented(what) }
}

func (s *server) getNetwork(c *reqCtx) error { // API-040
	n, err := s.svc.GetNetwork(c.r.Context(), c.actor, c.netID)
	if err != nil {
		return err
	}
	return ok(c, 200, toNetwork(n))
}

func (s *server) getMe(c *reqCtx) error { // API-041
	ms, err := s.svc.MyMemberships(c.r.Context(), c.actor)
	if err != nil {
		return err
	}
	out := []membershipDTO{}
	for _, m := range ms {
		if c.actor.NetworkScope == "" || c.actor.NetworkScope == m.NetworkID {
			out = append(out, toMembership(m))
		}
	}
	return ok(c, 200, map[string]any{"person_id": c.actor.PersonID, "memberships": out})
}

func (s *server) listMembers(c *reqCtx) error { // API-042
	if err := c.only("limit", "cursor"); err != nil {
		return err
	}
	p, err := c.page("")
	if err != nil {
		return err
	}
	rows, next, err := collect(c, p, func(pg service.Page) ([]service.DirectoryEntry, error) {
		return s.svc.ListDirectory(c.r.Context(), c.actor, c.netID, pg)
	}, func(e service.DirectoryEntry) string { return e.MemberID }, nil)
	if err != nil {
		return err
	}
	out := make([]memberDTO, 0, len(rows))
	for _, e := range rows {
		out = append(out, memberDTO{ID: e.MemberID, DisplayName: e.DisplayName, Status: "active"})
	}
	return listOK(c, out, next)
}

// findDirectory scans the directory (bounded) for one member: the service has
// no single-member read yet.
func (s *server) findDirectory(c *reqCtx, id string) (*service.DirectoryEntry, error) {
	after := ""
	for i := 0; i < 20; i++ {
		rows, err := s.svc.ListDirectory(c.r.Context(), c.actor, c.netID, service.Page{AfterID: after, Limit: 100})
		if err != nil {
			return nil, err
		}
		for _, e := range rows {
			if e.MemberID == id {
				e := e
				return &e, nil
			}
		}
		if len(rows) < 100 {
			break
		}
		after = rows[len(rows)-1].MemberID
	}
	return nil, nil
}

func (s *server) getMember(c *reqCtx) error { // API-043
	e, err := s.findDirectory(c, c.r.PathValue("id"))
	if err != nil {
		return err
	}
	if e == nil {
		return notFoundErr()
	}
	return ok(c, 200, memberDTO{ID: e.MemberID, DisplayName: e.DisplayName, Status: "active"})
}

// names resolves display names best-effort from the directory (active members only).
func (s *server) names(c *reqCtx, ids map[string]bool) map[string]string {
	out := map[string]string{}
	after := ""
	for i := 0; i < 10 && len(out) < len(ids); i++ {
		rows, err := s.svc.ListDirectory(c.r.Context(), c.actor, c.netID, service.Page{AfterID: after, Limit: 100})
		if err != nil {
			break
		}
		for _, e := range rows {
			if ids[e.MemberID] {
				out[e.MemberID] = e.DisplayName
			}
		}
		if len(rows) < 100 {
			break
		}
		after = rows[len(rows)-1].MemberID
	}
	return out
}

func (s *server) getAccount(c *reqCtx) error { // API-044
	b, err := s.svc.GetBalance(c.r.Context(), c.actor, c.netID)
	if err != nil {
		return err
	}
	return ok(c, 200, toAccount(b))
}

type entryDTO struct {
	TransactionID           string  `json:"transaction_id"`
	Kind                    string  `json:"kind"`
	Status                  string  `json:"status"`
	LedgerSeq               int64   `json:"ledger_seq"`
	AmountMinor             string  `json:"amount_minor"`
	CounterpartyMemberID    *string `json:"counterparty_member_id"`
	CounterpartyDisplayName *string `json:"counterparty_display_name"`
	PostedAt                *string `json:"posted_at"`
}

// listEntries derives the caller's ledger lines from posted transactions
// (the service has no entry listing). Order: transaction id descending.
func (s *server) listEntries(c *reqCtx) error { // API-045
	if err := c.only("limit", "cursor"); err != nil {
		return err
	}
	p, err := c.page("")
	if err != nil {
		return err
	}
	posted := []domain.TxStatus{domain.TxCompleted, domain.TxDisputed, domain.TxReversed}
	rows, next, err := collect(c, p, func(pg service.Page) ([]*domain.Transaction, error) {
		return s.svc.ListTransactions(c.r.Context(), c.actor, c.netID, posted, pg)
	}, func(t *domain.Transaction) string { return t.ID }, func(t *domain.Transaction) bool { return t.LedgerSeq > 0 })
	if err != nil {
		return err
	}
	me := c.member.ID
	ids := map[string]bool{}
	for _, t := range rows {
		ids[otherParty(t, me)] = true
	}
	names := s.names(c, ids)
	out := make([]entryDTO, 0, len(rows))
	for _, t := range rows {
		amt := t.Amount
		if t.PayerMemberID == me {
			amt = -amt
		}
		cp := otherParty(t, me)
		e := entryDTO{TransactionID: t.ID, Kind: string(t.Kind), Status: string(t.Status), LedgerSeq: t.LedgerSeq,
			AmountMinor: minor(amt), CounterpartyMemberID: strPtr(cp), PostedAt: tsPtr(t.CompletedAt)}
		if n, ok := names[cp]; ok {
			e.CounterpartyDisplayName = &n
		}
		out = append(out, e)
	}
	return listOK(c, out, next)
}

func otherParty(t *domain.Transaction, me string) string {
	if t.PayerMemberID == me {
		return t.PayeeMemberID
	}
	return t.PayerMemberID
}

func (s *server) depart(c *reqCtx) error { // API-067
	var in struct {
		Reason string `json:"reason"`
	}
	if err := c.decodeOptional(&in); err != nil {
		return err
	}
	m, err := s.svc.DepartMember(c.r.Context(), c.actor, c.netID, service.DepartInput{Reason: in.Reason})
	if err != nil {
		return err
	}
	out := map[string]any{"member_id": m.ID, "status": string(m.Status), "settlement": "retained"}
	if b, err := s.svc.GetBalance(c.r.Context(), c.actor, c.netID); err == nil {
		acc := toAccount(b)
		out["account"] = acc
		if b.Balance == 0 {
			out["settlement"] = "settled"
		}
	}
	c.w.Header().Set("Content-Type", "application/json; charset=utf-8")
	return ok(c, http.StatusOK, out)
}
