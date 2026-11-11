package api

import (
	"time"

	"github.com/google/uuid"

	"github.com/djshellshoxxx/circula/internal/service"
)

func (s *server) createInvitation(c *reqCtx) error { // API-061
	var in struct {
		TTLSeconds int `json:"ttl_seconds"`
	}
	if err := c.decodeOptional(&in); err != nil {
		return err
	}
	if in.TTLSeconds < 0 || in.TTLSeconds > 90*24*3600 {
		return validation("ttl_seconds", "out-of-range", "must be 0 (default 7 days) to 90 days")
	}
	tok, inv, err := s.svc.CreateInvitation(c.r.Context(), c.actor, c.netID,
		service.InviteInput{TTL: time.Duration(in.TTLSeconds) * time.Second})
	if err != nil {
		return err
	}
	return ok(c, 201, map[string]any{"id": inv.ID, "token": tok, "expires_at": ts(inv.ExpiresAt), "created_at": ts(inv.CreatedAt)})
}

func (s *server) acceptInvitation(c *reqCtx) error { // API-061
	var in struct {
		DisplayName  string `json:"display_name"`
		TermsVersion int    `json:"terms_version"` // GOV-009: required when the network has terms
	}
	if err := c.decode(&in); err != nil {
		return err
	}
	m, err := s.svc.RedeemInvitationTerms(c.r.Context(), c.actor, c.r.PathValue("token"), in.DisplayName, in.TermsVersion)
	if err != nil {
		return err
	}
	return ok(c, 201, toMembership(m))
}

func (s *server) listAudit(c *reqCtx) error { // API-062
	if err := c.only("limit", "cursor", "action", "target_id", "actor_id", "from", "to"); err != nil {
		return err
	}
	if err := c.adminOnly(); err != nil {
		return err
	}
	return s.auditList(c)
}

// auditList is the shared read; the service scopes rows by role (FR-082).
func (s *server) auditList(c *reqCtx) error {
	q := service.AuditQuery{Action: c.query.Get("action"), TargetID: c.query.Get("target_id"), ActorID: c.query.Get("actor_id")}
	for name, dst := range map[string]*time.Time{"from": &q.From, "to": &q.To} {
		if v := c.query.Get(name); v != "" {
			t, err := time.Parse(time.RFC3339, v)
			if err != nil {
				return invalidQuery(name + " must be an RFC 3339 timestamp")
			}
			*dst = t
		}
	}
	p, err := c.page(q.Action + "|" + q.TargetID + "|" + q.ActorID + "|" + c.query.Get("from") + "|" + c.query.Get("to"))
	if err != nil {
		return err
	}
	if p.after != "" {
		n, ok := atoiNonNeg(p.after)
		if !ok {
			return &apiError{status: 400, code: "invalid-cursor", detail: "cursor is not valid for this request"}
		}
		q.AfterSeq = int64(n)
	}
	q.Limit = p.limit + 1
	rows, err := s.svc.ListAudit(c.r.Context(), c.actor, c.netID, q)
	if err != nil {
		return err
	}
	var next *string
	if len(rows) > p.limit {
		rows = rows[:p.limit]
		n := s.sealCursor(itoa(int(rows[len(rows)-1].Seq)), p.filter)
		next = &n
	}
	out := make([]auditDTO, 0, len(rows))
	for _, e := range rows {
		out = append(out, toAudit(e))
	}
	return listOK(c, out, next)
}

type checkDTO struct {
	ID          string   `json:"id"`
	Status      string   `json:"status"`
	OK          bool     `json:"ok"`
	Findings    []string `json:"findings"`
	LedgerSeq   int64    `json:"ledger_seq"`
	AuditEvents int64    `json:"audit_events"`
	CreatedAt   string   `json:"created_at"`
}

func (s *server) runCheck(c *reqCtx) error { // API-063
	rep, err := s.svc.VerifyAndRecordLedger(c.r.Context(), c.actor, c.netID) // admin action: records the run (read-only VerifyLedger writes nothing)
	if err != nil {
		return err
	}
	id := rep.RunID // the consistency_runs id
	if id == "" {
		id = uuid.NewString()
	}
	d := checkDTO{ID: id, Status: "complete", OK: rep.OK, Findings: append([]string{}, rep.Findings...),
		LedgerSeq: rep.LedgerSeq, AuditEvents: rep.AuditEvents, CreatedAt: ts(s.opt.Now())}
	s.jobs.putCheck(c.netID, c.actor.PersonID, d)
	c.w.Header().Set("Location", v1+"/admin/consistency-checks/"+id)
	return ok(c, 201, d)
}

func (s *server) getCheck(c *reqCtx) error { // API-063
	if err := c.adminOnly(); err != nil {
		return err
	}
	d, found := s.jobs.getCheck(c.netID, c.r.PathValue("id"))
	if !found {
		return notFoundErr()
	}
	return ok(c, 200, d)
}
