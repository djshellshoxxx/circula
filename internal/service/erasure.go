package service

import (
	"context"
	"strings"

	"github.com/djshellshoxxx/circula/internal/domain"
	"github.com/djshellshoxxx/circula/internal/store"
)

// unusablePasswordHash is not a PHC string, so verifyPassword always refuses it.
const unusablePasswordHash = "!erased"

// ErasureReport says what AnonymiseMember changed. It holds counts, no personal data.
type ErasureReport struct {
	MemberID          string
	DisplayName       string // the replacement name
	Listings          int64
	Memos             int64
	DisputeStatements int64
	PersonErased      bool
}

// formerName is the replacement display name: stable, unique enough, no personal data.
func formerName(memberID string) string {
	short := memberID
	if len(short) > 8 {
		short = short[len(short)-8:]
	}
	return "Former member " + short
}

// AnonymiseMember irreversibly removes the personal data of a DEPARTED member
// while keeping the ledger intact (SEC-018). An administrator of the network
// (or a System actor, the operator CLI) may do it for anyone; the member may do
// it for themselves through EraseMyData.
//
// Refused unless the member is departed and has no blocker: a proposed or
// disputed transaction, or an unresolved dispute they are a party to. A reason
// is mandatory and goes to the audit event member.anonymised. It must not name
// the person: the audit chain is hashed over it and is never rewritten.
//
// Changed: the display name becomes "Former member <short id>"; the member's
// listings are archived and their text replaced; memos (transaction notes) the
// member wrote and the statements of disputes they opened are cleared; and,
// when the person holds no other non-departed membership anywhere, the person
// row is anonymised (login "erased-<id>", unusable password hash), all sessions
// are revoked and API tokens revoked and renamed.
//
// Kept untouched: ids, accounts, balances, entries, transaction rows (amounts,
// status, reasons written by administrators), status events, audit events and
// their hashes (docs/dev/erasure.md). VerifyLedger passes before and after.
func (s *Service) AnonymiseMember(ctx context.Context, actor domain.Actor, netID, memberID, reason string) (*ErasureReport, error) {
	return s.anonymise(ctx, actor, netID, memberID, reason, false)
}

// EraseMyData is the departed member erasing their own personal data (SEC-018).
// The reason is optional.
func (s *Service) EraseMyData(ctx context.Context, actor domain.Actor, netID, reason string) (*ErasureReport, error) {
	if strings.TrimSpace(reason) == "" {
		reason = "requested by the member"
	}
	return s.anonymise(ctx, actor, netID, "", reason, true)
}

func (s *Service) anonymise(ctx context.Context, actor domain.Actor, netID, memberID, reason string, self bool) (*ErasureReport, error) {
	if err := requireReason(reason); err != nil {
		return nil, err
	}
	reason = strings.TrimSpace(reason)
	var rep *ErasureReport
	err := s.update(ctx, func(tx store.Tx) error {
		p, err := s.open(tx, actor, netID, domain.ScopeWrite)
		if err != nil {
			return err
		}
		if self {
			if p.member == nil {
				return domain.Forbidden("a member is required")
			}
			memberID = p.member.ID
		} else if err := p.requireAdmin(); err != nil {
			return err
		}
		target, err := tx.GetMember(netID, memberID)
		if err != nil {
			return domain.NotFound("member")
		}
		if done, err := tx.GetErasure(netID, target.ID); err != nil {
			return err
		} else if done != nil {
			return domain.InvalidState("member is already anonymised")
		}
		if target.Status != domain.MemberDeparted {
			return domain.InvalidState("member is %s; only departed members can be anonymised", target.Status)
		}
		if blockers, err := erasureBlockers(tx, netID, target.ID); err != nil {
			return err
		} else if len(blockers) > 0 {
			return &domain.Error{Code: domain.CodeBlocked, Msg: "anonymisation is blocked", Blockers: blockers}
		}
		now := s.now()
		rep = &ErasureReport{MemberID: target.ID, DisplayName: formerName(target.ID)}
		if err := tx.SetMemberDisplayName(netID, target.ID, rep.DisplayName); err != nil {
			return err
		}
		if rep.Listings, err = tx.AnonymiseListings(netID, target.ID, "Erased listing", now); err != nil {
			return err
		}
		if rep.Memos, err = tx.RedactTransactionNotes(netID, target.ID); err != nil {
			return err
		}
		if rep.DisputeStatements, err = tx.RedactDisputeStatements(netID, target.ID); err != nil {
			return err
		}
		if rep.PersonErased, err = s.erasePersonIfUnused(tx, target); err != nil {
			return err
		}
		if err := tx.InsertErasure(&domain.ErasureRecord{MemberID: target.ID, NetworkID: netID, PersonID: target.PersonID,
			ErasedAt: now, PersonErased: rep.PersonErased}); err != nil {
			return err
		}
		at := domain.ActorAdmin
		if self {
			at = domain.ActorMember
		}
		return s.audit(tx, p, auditEntry{Type: p.actorType(at), Action: domain.ActMemberAnonymised, TargetType: "member",
			TargetID: target.ID, Reason: reason, Subjects: []string{target.ID},
			Before: map[string]any{"status": target.Status},
			After: map[string]any{"person_erased": rep.PersonErased, "listings": rep.Listings, "memos": rep.Memos,
				"dispute_statements": rep.DisputeStatements, "self": self}})
	})
	if err != nil {
		return nil, err
	}
	return rep, nil
}

// erasureBlockers lists what prevents anonymisation: anything still in flight.
func erasureBlockers(tx store.Tx, netID, memberID string) ([]string, error) {
	var out []string
	open, err := tx.ListTransactions(netID, store.TxFilter{MemberID: memberID,
		Statuses: []domain.TxStatus{domain.TxProposed, domain.TxDisputed}, Limit: 500})
	if err != nil {
		return nil, err
	}
	for _, t := range open {
		out = append(out, string(t.Status)+" transaction "+t.ID)
	}
	ds, err := tx.ListDisputes(netID, store.DisputeFilter{PartyID: memberID, Limit: 500,
		Statuses: []domain.DisputeStatus{domain.DisputeOpen, domain.DisputeUnderReview, domain.DisputePostingPending}})
	if err != nil {
		return nil, err
	}
	for _, d := range ds {
		out = append(out, "unresolved dispute "+d.ID)
	}
	return out, nil
}

// erasePersonIfUnused anonymises the person row when no membership of theirs
// (other than the one being erased) is still alive. Departed memberships in
// other networks do not keep the login alive: they cannot sign in to anything.
func (s *Service) erasePersonIfUnused(tx store.Tx, target *domain.Member) (bool, error) {
	ms, err := tx.ListMembershipsOfPerson(target.PersonID)
	if err != nil {
		return false, err
	}
	for _, m := range ms {
		if m.ID != target.ID && m.Status != domain.MemberDeparted {
			return false, nil
		}
	}
	person, _, err := tx.GetPerson(target.PersonID)
	if err != nil {
		return false, err
	}
	now := s.now()
	if err := tx.AnonymisePerson(person.ID, "erased-"+person.ID, unusablePasswordHash); err != nil {
		return false, err
	}
	if err := tx.DeleteThrottle(person.Login); err != nil {
		return false, err
	}
	if err := tx.RevokeSessionsOf(person.ID, now); err != nil {
		return false, err
	}
	return true, tx.RedactTokensOf(person.ID, now)
}
