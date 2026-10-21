package api

import (
	"net/http"
	"strconv"

	"github.com/djshellshoxxx/circula/internal/domain"
	"github.com/djshellshoxxx/circula/internal/service"
)

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
	zone, err := s.svc.GetTimeZone(c.r.Context(), c.actor)
	if err != nil {
		return err
	}
	return ok(c, 200, map[string]any{"person_id": c.actor.PersonID, "memberships": out, "time_zone": zonePtr(zone)})
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

func (s *server) getMember(c *reqCtx) error { // API-043
	m, err := s.svc.GetMember(c.r.Context(), c.actor, c.netID, c.r.PathValue("id"))
	if err != nil {
		return err
	}
	return ok(c, 200, memberDTO{ID: m.ID, DisplayName: m.DisplayName, Status: string(m.Status)})
}

// names resolves display names best-effort, one GetMember per distinct id
// (at most one page of ids); members the caller may not see are left out.
func (s *server) names(c *reqCtx, ids map[string]bool) map[string]string {
	out := map[string]string{}
	for id := range ids {
		if m, err := s.svc.GetMember(c.r.Context(), c.actor, c.netID, id); err == nil {
			out[id] = m.DisplayName
		}
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

// listEntries lists the caller's ledger lines ordered by ledger_seq descending
// (API-045); the cursor is the ledger_seq of the last line returned.
func (s *server) listEntries(c *reqCtx) error { // API-045
	if err := c.only("limit", "cursor"); err != nil {
		return err
	}
	p, err := c.page("")
	if err != nil {
		return err
	}
	rows, next, err := collect(c, p, func(pg service.Page) ([]service.EntryRow, error) {
		before, _ := strconv.ParseInt(pg.AfterID, 10, 64)
		return s.svc.ListEntries(c.r.Context(), c.actor, c.netID, service.EntryQuery{BeforeSeq: before, Limit: pg.Limit})
	}, func(r service.EntryRow) string { return strconv.FormatInt(r.Entry.LedgerSeq, 10) }, nil)
	if err != nil {
		return err
	}
	me := c.member.ID
	ids := map[string]bool{}
	for _, r := range rows {
		ids[otherParty(r.Tx, me)] = true
	}
	names := s.names(c, ids)
	out := make([]entryDTO, 0, len(rows))
	for _, r := range rows {
		t := r.Tx
		amt := t.Amount
		if t.PayerMemberID == me {
			amt = -amt
		}
		cp := otherParty(t, me)
		e := entryDTO{TransactionID: t.ID, Kind: string(t.Kind), Status: string(t.Status), LedgerSeq: r.Entry.LedgerSeq,
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
