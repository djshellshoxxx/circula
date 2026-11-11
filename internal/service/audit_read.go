package service

import (
	"context"
	"encoding/json"
	"time"

	"github.com/djshellshoxxx/circula/internal/domain"
	"github.com/djshellshoxxx/circula/internal/store"
)

// AuditQuery filters audit reads (FR-084). AfterSeq is the keyset cursor.
type AuditQuery struct {
	Action   string
	ActorID  string
	TargetID string
	From, To time.Time
	AfterSeq int64
	Limit    int
}

// memberSafeKeys are the only summary keys members may see: no other
// member's identifiers or free text (DAT-033, SEC-019, FR-082).
var memberSafeKeys = map[string]bool{"status": true, "amount": true, "outcome": true,
	"transaction_status": true, "original_status": true, "reversed_total": true, "effective_limit": true}

func memberSafe(summary string) string {
	if summary == "" {
		return ""
	}
	var m map[string]any
	if json.Unmarshal([]byte(summary), &m) != nil {
		return ""
	}
	for k := range m {
		if !memberSafeKeys[k] {
			delete(m, k)
		}
	}
	return summarize(m)
}

// ListAudit reads the audit log; what the caller sees depends on role (DAT-033..035, FR-082).
// In detail:
//   - administrators: the whole network log; the read is itself audited;
//   - reviewers: events about disputes;
//   - members (including suspended and departed): only events concerning their
//     own account, with actor identity reduced to the role label, safe summary
//     keys only, and no hash material.
func (s *Service) ListAudit(ctx context.Context, actor domain.Actor, netID string, q AuditQuery) ([]*domain.AuditEvent, error) {
	var out []*domain.AuditEvent
	f := store.AuditFilter{Action: q.Action, ActorID: q.ActorID, TargetID: q.TargetID, From: q.From, To: q.To,
		AfterSeq: q.AfterSeq, Limit: q.Limit}
	err := s.update(ctx, func(tx store.Tx) error {
		p, err := s.open(tx, actor, netID, domain.ScopeRead)
		if err != nil {
			return err
		}
		switch {
		case p.requireAdmin() == nil:
			if out, err = tx.ListAudit(netID, f); err != nil {
				return err
			}
			return s.audit(tx, p, auditEntry{Type: domain.ActorAdmin, Action: domain.ActAuditRead, TargetType: "audit", TargetID: netID})
		case p.requireDecider() == nil:
			f.TargetTypes = []string{"dispute"}
			out, err = tx.ListAudit(netID, f)
			return err
		case p.member != nil && p.member.Status != domain.MemberPending:
			f.SubjectMember = p.member.ID
			evs, err := tx.ListAudit(netID, f)
			for _, e := range evs {
				own := e.ActorID == p.member.ID
				if !own {
					e.ActorID = ""
				}
				e.SubjectIDs, e.PrevHash, e.Hash, e.RequestID, e.IdemRef = nil, "", "", "", ""
				e.SummaryBefore, e.SummaryAfter = memberSafe(e.SummaryBefore), memberSafe(e.SummaryAfter)
				if !own {
					e.Reason = "" // other people's reasons (e.g. an administrator's) are for administrators only (FR-015)
				}
			}
			out = evs
			return err
		}
		return domain.Forbidden("no audit access")
	})
	return out, err
}
