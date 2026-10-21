package api

import (
	"strings"

	"github.com/djshellshoxxx/circula/internal/domain"
	"github.com/djshellshoxxx/circula/internal/service"
)

type proposeReq struct {
	CounterpartyMemberID string `json:"counterparty_member_id"`
	Direction            string `json:"direction"`
	AmountMinor          string `json:"amount_minor"`
	ListingID            string `json:"listing_id"`
	Description          string `json:"description"`
}

func (s *server) propose(c *reqCtx) error { // API-048
	var in proposeReq
	if err := c.decode(&in); err != nil {
		return err
	}
	amt, err := parseMinor("amount_minor", in.AmountMinor, false)
	if err != nil {
		return err
	}
	var dir service.Direction
	switch in.Direction {
	case "pay":
		dir = service.ProposerPays
	case "receive":
		dir = service.ProposerIsPaid
	default:
		return validation("direction", "invalid-enum", "must be pay or receive")
	}
	t, err := s.svc.ProposeTransaction(c.r.Context(), c.actor, c.netID, service.ProposeInput{
		CounterpartyID: in.CounterpartyMemberID, Direction: dir, Amount: amt, ListingID: in.ListingID,
		Note: in.Description, IdempotencyKey: c.idemKey})
	if err != nil {
		return err
	}
	c.w.Header().Set("Location", v1+"/transactions/"+t.ID)
	return ok(c, 201, toTx(t, c.member.ID))
}

func parseStatuses(raw string) ([]domain.TxStatus, error) {
	if raw == "" {
		return nil, nil
	}
	var out []domain.TxStatus
	for _, p := range strings.Split(raw, ",") {
		st := domain.TxStatus(p)
		known := false
		for _, k := range domain.AllTxStatuses {
			known = known || k == st
		}
		if !known || st == domain.TxAccepted {
			return nil, invalidQuery("unknown status " + p)
		}
		out = append(out, st)
	}
	return out, nil
}

func (s *server) listTx(c *reqCtx) error { // API-049
	if err := c.only("limit", "cursor", "status", "role"); err != nil {
		return err
	}
	sts, err := parseStatuses(c.query.Get("status"))
	if err != nil {
		return err
	}
	role := c.query.Get("role")
	if role != "" && role != "payer" && role != "payee" {
		return invalidQuery("role must be payer or payee")
	}
	p, err := c.page(c.query.Get("status") + "|" + role)
	if err != nil {
		return err
	}
	me := c.member.ID
	rows, next, err := collect(c, p, func(pg service.Page) ([]*domain.Transaction, error) {
		return s.svc.ListTransactions(c.r.Context(), c.actor, c.netID, sts, pg)
	}, func(t *domain.Transaction) string { return t.ID }, func(t *domain.Transaction) bool {
		return role == "" || (role == "payer" && t.PayerMemberID == me) || (role == "payee" && t.PayeeMemberID == me)
	})
	if err != nil {
		return err
	}
	out := make([]txDTO, 0, len(rows))
	for _, t := range rows {
		out = append(out, toTx(t, me))
	}
	return listOK(c, out, next)
}

type txDetail struct {
	txDTO
	EventsURL string `json:"events_url"`
}

func (s *server) getTx(c *reqCtx) error { // API-049
	v, err := s.svc.GetTransaction(c.r.Context(), c.actor, c.netID, c.r.PathValue("id"))
	if err != nil {
		return err
	}
	return ok(c, 200, txDetail{txDTO: toTx(v.Tx, c.member.ID), EventsURL: v1 + "/transactions/" + v.Tx.ID + "/events"})
}

type eventDTO struct {
	From       string `json:"from"`
	To         string `json:"to"`
	ActorType  string `json:"actor_type"`
	ActorID    string `json:"actor_id"`
	OccurredAt string `json:"occurred_at"`
}

func (s *server) txEvents(c *reqCtx) error { // API-050
	if err := c.only("limit", "cursor"); err != nil {
		return err
	}
	p, err := c.page(c.r.PathValue("id"))
	if err != nil {
		return err
	}
	v, err := s.svc.GetTransaction(c.r.Context(), c.actor, c.netID, c.r.PathValue("id"))
	if err != nil {
		return err
	}
	start := 0
	if p.after != "" {
		n, ok := atoiNonNeg(p.after)
		if !ok {
			return &apiError{status: 400, code: "invalid-cursor", detail: "cursor is not valid for this request"}
		}
		start = n
	}
	out := []eventDTO{}
	for i := start; i < len(v.History) && len(out) < p.limit; i++ {
		e := v.History[i]
		at := "member"
		if e.ActorID == "" {
			at = "system"
		}
		out = append(out, eventDTO{From: string(e.From), To: string(e.To), ActorType: at, ActorID: e.ActorID, OccurredAt: ts(e.OccurredAt)})
	}
	var next *string
	if end := start + len(out); end < len(v.History) {
		n := s.sealCursor(itoa(end), p.filter)
		next = &n
	}
	return listOK(c, out, next)
}

// txActionError enriches invalid-state refusals with the current status (API-028).
func (s *server) txActionError(c *reqCtx, id string, err error) error {
	if domain.CodeOf(err) != domain.CodeInvalidState {
		return err
	}
	ae, _ := fromErr(err)
	if v, gerr := s.svc.GetTransaction(c.r.Context(), c.actor, c.netID, id); gerr == nil {
		cp := *ae
		cp.currentStatus = string(v.Tx.Status)
		return &cp
	}
	return err
}

type txActionResp struct {
	txDTO
	Account *accountDTO `json:"account,omitempty"`
}

func (s *server) acceptTx(c *reqCtx) error { // API-051
	id := c.r.PathValue("id")
	t, err := s.svc.AcceptTransaction(c.r.Context(), c.actor, c.netID, service.TxRef{TxID: id, IdempotencyKey: c.idemKey})
	if err != nil {
		return s.txActionError(c, id, err)
	}
	out := txActionResp{txDTO: toTx(t, c.member.ID)}
	if b, err := s.svc.GetBalance(c.r.Context(), c.actor, c.netID); err == nil {
		a := toAccount(b)
		out.Account = &a
	}
	return ok(c, 200, out)
}

func (s *server) declineTx(c *reqCtx) error { // API-052
	id := c.r.PathValue("id")
	t, err := s.svc.DeclineTransaction(c.r.Context(), c.actor, c.netID, service.TxRef{TxID: id, IdempotencyKey: c.idemKey})
	if err != nil {
		return s.txActionError(c, id, err)
	}
	return ok(c, 200, toTx(t, c.member.ID))
}

func (s *server) cancelTx(c *reqCtx) error { // API-053
	var in struct {
		Reason string `json:"reason"`
	}
	if err := c.decodeOptional(&in); err != nil {
		return err
	}
	id := c.r.PathValue("id")
	t, err := s.svc.CancelTransaction(c.r.Context(), c.actor, c.netID, service.CancelInput{TxID: id, Reason: in.Reason, IdempotencyKey: c.idemKey})
	if err != nil {
		return s.txActionError(c, id, err)
	}
	return ok(c, 200, toTx(t, c.member.ID))
}
