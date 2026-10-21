package api

import (
	"github.com/djshellshoxxx/circula/internal/domain"
	"github.com/djshellshoxxx/circula/internal/service"
	"github.com/djshellshoxxx/circula/internal/store"
)

// finalExportDTO is the final export record of a completed closure (GOV-025 e).
type finalExportDTO struct {
	SHA256            string `json:"sha256"`
	LedgerSeq         int64  `json:"ledger_seq"`
	EntriesSumMinor   string `json:"entries_sum_minor"`
	BalanceTotalMinor string `json:"balance_total_minor"`
}

type closureDTO struct {
	Reason        string          `json:"reason"`
	ScheduledAt   string          `json:"scheduled_at"`
	ClosingAt     string          `json:"closing_at"`
	NoticeDays    int             `json:"notice_days"`
	ClosedAt      *string         `json:"closed_at"`
	ReadOnlyUntil *string         `json:"read_only_until"`
	FinalExport   *finalExportDTO `json:"final_export"`
	ErasedAt      *string         `json:"erased_at"`
}

func toClosure(c *domain.Closure) *closureDTO {
	if c == nil {
		return nil
	}
	d := &closureDTO{Reason: c.Reason, ScheduledAt: ts(c.ScheduledAt), ClosingAt: ts(c.ClosingAt), NoticeDays: c.NoticeDays}
	if c.Done() {
		t, u := c.ClosedAt, c.ReadOnlyUntil()
		d.ClosedAt, d.ReadOnlyUntil = tsPtr(&t), tsPtr(&u)
		if c.FinalSHA256 != "" {
			d.FinalExport = &finalExportDTO{SHA256: c.FinalSHA256, LedgerSeq: c.FinalLedgerSeq,
				EntriesSumMinor: minor(c.FinalEntriesSum), BalanceTotalMinor: minor(c.FinalBalanceTotal)}
		}
	}
	if !c.ErasedAt.IsZero() {
		t := c.ErasedAt
		d.ErasedAt = tsPtr(&t)
	}
	return d
}

func (s *server) closeNetwork(c *reqCtx) error { // FR-068, GOV-025: announces the closure date
	var in struct {
		Reason     string `json:"reason"`
		NoticeDays int    `json:"notice_days"`
	}
	if err := c.decode(&in); err != nil {
		return err
	}
	cl, err := s.svc.ScheduleClosure(c.r.Context(), c.actor, c.netID, service.ClosureInput{Reason: in.Reason, NoticeDays: in.NoticeDays})
	if err != nil {
		return err
	}
	return ok(c, 200, toClosure(cl))
}

func (s *server) cancelClosure(c *reqCtx) error { // GOV-025
	reason, err := s.reasonOnly(c)
	if err != nil {
		return err
	}
	if err := s.svc.CancelClosure(c.r.Context(), c.actor, c.netID, reason); err != nil {
		return err
	}
	c.w.WriteHeader(204)
	return nil
}

func (s *server) verifyClosure(c *reqCtx) error { // GOV-025 e
	chk, err := s.svc.VerifyClosureExport(c.r.Context(), c.actor, c.netID)
	if err != nil {
		return err
	}
	return ok(c, 200, map[string]any{"stored_sha256": chk.Stored, "recomputed_sha256": chk.Recomputed, "match": chk.Match,
		"ledger_seq": chk.LedgerSeq, "entries_sum_minor": minor(chk.EntriesSum), "balance_total_minor": minor(chk.BalanceTotal), "erased": chk.Erased})
}

// Challenges (GOV-016).

type challengeDTO struct {
	ID        string  `json:"id"`
	MemberID  string  `json:"member_id"`
	Kind      string  `json:"kind"`
	SubjectID *string `json:"subject_id"`
	Reason    string  `json:"reason"`
	Status    string  `json:"status"`
	Outcome   *string `json:"outcome"`
	DecidedBy *string `json:"decided_by"`
	Rationale string  `json:"rationale"`
	CreatedAt string  `json:"created_at"`
	DecidedAt *string `json:"decided_at"`
}

func toChallenge(ch store.Challenge) challengeDTO {
	d := challengeDTO{ID: ch.ID, MemberID: ch.MemberID, Kind: ch.Kind, SubjectID: strPtr(ch.SubjectID), Reason: ch.Reason,
		Status: ch.Status, Outcome: strPtr(ch.Outcome), DecidedBy: strPtr(ch.DecidedBy), Rationale: ch.Rationale, CreatedAt: ts(ch.CreatedAt)}
	if !ch.DecidedAt.IsZero() {
		t := ch.DecidedAt
		d.DecidedAt = tsPtr(&t)
	}
	return d
}

func (s *server) openChallenge(c *reqCtx) error {
	var in struct {
		Kind      string `json:"kind"`
		SubjectID string `json:"subject_id"`
		Reason    string `json:"reason"`
	}
	if err := c.decode(&in); err != nil {
		return err
	}
	ch, err := s.svc.OpenChallenge(c.r.Context(), c.actor, c.netID, service.ChallengeInput{Kind: in.Kind, SubjectID: in.SubjectID, Reason: in.Reason})
	if err != nil {
		return err
	}
	c.w.Header().Set("Location", v1+"/challenges")
	return ok(c, 201, toChallenge(*ch))
}

func (s *server) listChallenges(c *reqCtx) error {
	rows, err := s.svc.ListChallenges(c.r.Context(), c.actor, c.netID, c.query.Get("status"))
	if err != nil {
		return err
	}
	out := make([]challengeDTO, 0, len(rows))
	for _, r := range rows {
		out = append(out, toChallenge(r))
	}
	return listOK(c, out, nil)
}

func (s *server) decideChallenge(c *reqCtx) error {
	var in struct {
		Outcome string `json:"outcome"`
		Reason  string `json:"reason"`
	}
	if err := c.decode(&in); err != nil {
		return err
	}
	ch, err := s.svc.DecideChallenge(c.r.Context(), c.actor, c.netID, service.ChallengeDecision{ID: c.r.PathValue("id"), Outcome: in.Outcome, Rationale: in.Reason})
	if err != nil {
		return err
	}
	return ok(c, 200, toChallenge(*ch))
}
