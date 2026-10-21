package service

import (
	"context"
	"time"

	"github.com/djshellshoxxx/circula/internal/domain"
	"github.com/djshellshoxxx/circula/internal/store"
)

// Direction says who pays in a proposal.
type Direction string

// Proposal directions.
const (
	ProposerPays   Direction = "pay"     // proposer is the payer; counterparty must accept
	ProposerIsPaid Direction = "request" // proposer is the payee; counterparty (payer) must accept
)

// ProposeInput proposes a trade (FR-030).
type ProposeInput struct {
	CounterpartyID string // member id
	Direction      Direction
	Amount         int64 // minor units, > 0
	ListingID      string
	Note           string
	IdempotencyKey string `json:"-"`
}

// ProposeTransaction creates a `proposed` trade and reserves the payer's
// capacity (DOM-011, DOM-012). The payer's available credit
// (balance + limit - reserved) must cover the amount; nothing posts yet.
func (s *Service) ProposeTransaction(ctx context.Context, actor domain.Actor, netID string, in ProposeInput) (*domain.Transaction, error) {
	var out *domain.Transaction
	err := s.update(ctx, func(tx store.Tx) error {
		p, err := s.open(tx, actor, netID, domain.ScopeWrite)
		if err != nil {
			return err
		}
		if err := p.requireActive(); err != nil {
			return err
		}
		out, err = idempotent(s, tx, netID, actor, "tx.propose", in.IdempotencyKey, in, func() (string, error) {
			return s.propose(tx, p, in)
		}, func(id string) (*domain.Transaction, error) { return tx.GetTransaction(netID, id) })
		return err
	})
	return out, err
}

func (s *Service) propose(tx store.Tx, p *principal, in ProposeInput) (string, error) {
	if err := p.requireNetworkActive(); err != nil {
		return "", err
	}
	pol := p.net.Policy
	if in.Direction != ProposerPays && in.Direction != ProposerIsPaid {
		return "", domain.Invalid("direction", "must be pay or request")
	}
	if in.Amount <= 0 {
		return "", domain.Invalid("amount", "must be greater than zero")
	}
	if in.Amount > pol.MaxTxAmount {
		return "", domain.Errf(domain.CodeAmountExceeds, "maximum transaction amount is %s", fmtAmt(p.net, pol.MaxTxAmount))
	}
	if err := boundedText("note", in.Note, maxNoteLen); err != nil {
		return "", err
	}
	if in.CounterpartyID == p.member.ID {
		return "", domain.Invalid("counterparty_id", "cannot trade with yourself")
	}
	cp, err := tx.GetMember(p.net.ID, in.CounterpartyID)
	if err != nil {
		return "", domain.NotFound("member") // also covers other networks (DOM-001)
	}
	if cp.Status != domain.MemberActive {
		return "", domain.Errf(domain.CodeCounterpartyInact, "counterparty is not an active member")
	}
	if in.ListingID != "" {
		if _, err := tx.GetListing(p.net.ID, in.ListingID); err != nil {
			return "", domain.Invalid("listing_id", "unknown listing")
		}
	}
	payerM, payeeM := p.member, cp
	if in.Direction == ProposerIsPaid {
		payerM, payeeM = cp, p.member
	}
	if pol.DefaultFlagBlocksPayer && payerM.DefaultFlag {
		return "", domain.Forbidden("payer is flagged and cannot commit new payments")
	}
	payer, err := tx.GetAccountByMember(p.net.ID, payerM.ID)
	if err != nil {
		return "", err
	}
	payee, err := tx.GetAccountByMember(p.net.ID, payeeM.ID)
	if err != nil {
		return "", err
	}
	now := s.now()
	reserved, err := tx.ReservedOutgoing(p.net.ID, payer.ID, now)
	if err != nil {
		return "", err
	}
	if avail := payer.Balance + payer.EffectiveLimit(pol.DefaultCreditLimit, now) - reserved; avail < in.Amount {
		if avail < 0 {
			avail = 0
		}
		return "", domain.Errf(domain.CodeInsufficientCredit, "payer has %s available", fmtAmt(p.net, avail))
	}
	exp := now.Add(time.Duration(pol.ProposalExpiryDays) * 24 * time.Hour)
	t := &domain.Transaction{ID: s.id(), NetworkID: p.net.ID, Kind: domain.KindTrade, Status: domain.TxProposed,
		PayerAccountID: payer.ID, PayeeAccountID: payee.ID, Amount: in.Amount, ListingID: in.ListingID,
		Note: in.Note, CreatedBy: p.member.ID, ProposedAt: now, ExpiresAt: &exp}
	if err := tx.InsertTransaction(t); err != nil {
		return "", err
	}
	if err := tx.InsertStatusEvent(&domain.StatusEvent{ID: s.id(), NetworkID: p.net.ID, TxID: t.ID,
		To: domain.TxProposed, ActorID: p.member.ID, RequestID: p.actor.RequestID, OccurredAt: now}); err != nil {
		return "", err
	}
	return t.ID, s.audit(tx, p, auditEntry{Action: domain.ActTxProposed, TargetType: "transaction", TargetID: t.ID,
		IdemRef: keyRef(in.IdempotencyKey), Subjects: []string{payerM.ID, payeeM.ID},
		After: map[string]any{"status": "proposed", "amount": in.Amount, "payer": payerM.ID, "payee": payeeM.ID}})
}

// loadParty loads a transaction the acting member is a party to. Everything
// else, including other networks' ids, is an indistinguishable not-found (SEC-009).
func loadParty(tx store.Tx, p *principal, txID string) (*domain.Transaction, error) {
	t, err := tx.GetTransaction(p.net.ID, txID)
	if err != nil || !t.IsParty(p.memberID()) {
		return nil, domain.NotFound("transaction")
	}
	return t, nil
}

// TxRef identifies a transaction in a retryable command.
type TxRef struct {
	TxID           string
	IdempotencyKey string `json:"-"`
}

// AcceptTransaction is consent by the party who did not propose. In one
// atomic unit it re-checks that both parties are active, re-checks LED-5 on the
// payer, inserts both entries, completes the transaction and writes the audit
// event (DOM-014). If any check fails nothing is written and the transaction
// stays `proposed`.
func (s *Service) AcceptTransaction(ctx context.Context, actor domain.Actor, netID string, in TxRef) (*domain.Transaction, error) {
	var out *domain.Transaction
	err := s.update(ctx, func(tx store.Tx) error {
		p, err := s.open(tx, actor, netID, domain.ScopeWrite)
		if err != nil {
			return err
		}
		if err := p.requireActive(); err != nil {
			return err
		}
		out, err = idempotent(s, tx, netID, actor, "tx.accept", in.IdempotencyKey, in, func() (string, error) {
			t, err := loadParty(tx, p, in.TxID)
			if err != nil {
				return "", err
			}
			if t.Kind != domain.KindTrade {
				return "", domain.InvalidState("only trades are accepted")
			}
			if t.CreatedBy == p.member.ID {
				return "", domain.Forbidden("the proposer cannot accept their own proposal")
			}
			if _, err := domain.NextTxStatus(t.Status, domain.CmdAccept); err != nil {
				return "", err
			}
			if err := p.requireNetworkActive(); err != nil {
				return "", err
			}
			now := s.now()
			if t.ExpiresAt != nil && !now.Before(*t.ExpiresAt) {
				return "", domain.InvalidState("proposal expired")
			}
			for _, id := range []string{t.PayerMemberID, t.PayeeMemberID} {
				m, err := tx.GetMember(netID, id)
				if err != nil || m.Status != domain.MemberActive {
					return "", domain.Errf(domain.CodeCounterpartyInact, "a party is no longer active")
				}
			}
			t.AcceptedBy = p.member.ID
			if err := s.postTx(tx, p.net, t, p.member.ID, p.actor.RequestID); err != nil {
				return "", err
			}
			return t.ID, s.audit(tx, p, auditEntry{Action: domain.ActTxCompleted, TargetType: "transaction", TargetID: t.ID,
				IdemRef: keyRef(in.IdempotencyKey), Subjects: []string{t.PayerMemberID, t.PayeeMemberID},
				Before: map[string]any{"status": "proposed"}, After: map[string]any{"status": "completed", "amount": t.Amount, "ledger_seq": t.LedgerSeq}})
		}, func(id string) (*domain.Transaction, error) { return tx.GetTransaction(netID, id) })
		return err
	})
	return out, err
}

// DeclineTransaction is the counterparty's exit. Suspended members may decline
// (DOM-039); it releases the reservation and posts nothing.
func (s *Service) DeclineTransaction(ctx context.Context, actor domain.Actor, netID string, in TxRef) (*domain.Transaction, error) {
	return s.exit(ctx, actor, netID, in, "tx.decline", domain.CmdDecline, domain.ActTxDeclined, "")
}

// CancelInput withdraws a proposal (proposer) or cancels it administratively.
type CancelInput struct {
	TxID           string
	Reason         string // mandatory for administrator cancellation
	IdempotencyKey string `json:"-"`
}

// CancelTransaction lets the proposer withdraw (also when suspended), or an
// administrator who is not a party cancel with a reason (DOM-010, DOM-037).
func (s *Service) CancelTransaction(ctx context.Context, actor domain.Actor, netID string, in CancelInput) (*domain.Transaction, error) {
	return s.exit(ctx, actor, netID, TxRef{TxID: in.TxID, IdempotencyKey: in.IdempotencyKey}, "tx.cancel", domain.CmdCancel, domain.ActTxCancelled, in.Reason)
}

func (s *Service) exit(ctx context.Context, actor domain.Actor, netID string, in TxRef, op string, cmd domain.TxCommand, action, reason string) (*domain.Transaction, error) {
	var out *domain.Transaction
	err := s.update(ctx, func(tx store.Tx) error {
		p, err := s.open(tx, actor, netID, domain.ScopeWrite)
		if err != nil {
			return err
		}
		if p.member == nil || (p.member.Status != domain.MemberActive && p.member.Status != domain.MemberSuspended) {
			return domain.Forbidden("membership cannot perform this action")
		}
		payload := struct {
			TxRef
			Reason string
		}{in, reason}
		out, err = idempotent(s, tx, netID, actor, op, in.IdempotencyKey, payload, func() (string, error) {
			t, err := tx.GetTransaction(netID, in.TxID)
			if err != nil {
				return "", domain.NotFound("transaction")
			}
			asAdmin := false
			switch {
			case !t.IsParty(p.member.ID):
				if p.requireAdmin() != nil || cmd != domain.CmdCancel {
					return "", domain.NotFound("transaction")
				}
				asAdmin = true
				if err := requireReason(reason); err != nil {
					return "", err
				}
			case cmd == domain.CmdDecline && t.CreatedBy == p.member.ID:
				return "", domain.Forbidden("the proposer withdraws instead of declining")
			case cmd == domain.CmdCancel && t.CreatedBy != p.member.ID:
				return "", domain.Forbidden("only the proposer can withdraw; decline instead")
			}
			if t.Kind != domain.KindTrade {
				return "", domain.InvalidState("only trades can be declined or cancelled")
			}
			prev := t.Status
			if err := s.transition(tx, p.net.ID, t, cmd, p.member.ID, p.actor.RequestID); err != nil {
				return "", err
			}
			at := domain.ActorMember
			if asAdmin {
				at = domain.ActorAdmin
			}
			return t.ID, s.audit(tx, p, auditEntry{Type: at, Action: action, TargetType: "transaction", TargetID: t.ID, Reason: reason,
				IdemRef: keyRef(in.IdempotencyKey), Subjects: []string{t.PayerMemberID, t.PayeeMemberID},
				Before: map[string]any{"status": prev}, After: map[string]any{"status": t.Status}})
		}, func(id string) (*domain.Transaction, error) { return tx.GetTransaction(netID, id) })
		return err
	})
	return out, err
}

// transition applies a state-machine command and records the status event.
// It never posts. Illegal transitions return CodeInvalidState and change nothing.
func (s *Service) transition(tx store.Tx, netID string, t *domain.Transaction, cmd domain.TxCommand, actorMember, requestID string) error {
	next, err := domain.NextTxStatus(t.Status, cmd)
	if err != nil {
		return err
	}
	prev := t.Status
	t.Status = next
	if err := tx.UpdateTransaction(t); err != nil {
		return err
	}
	if prev == next { // partial reversal: no status change to record
		return nil
	}
	return tx.InsertStatusEvent(&domain.StatusEvent{ID: s.id(), NetworkID: netID, TxID: t.ID,
		From: prev, To: next, ActorID: actorMember, RequestID: requestID, OccurredAt: s.now()})
}

// ExpireProposals moves proposals whose expiry has elapsed to `expired`,
// releasing their reservations (DOM-013, FR-034). System actors only. It is
// idempotent by construction: re-running changes nothing. Returns the count
// expired in this run (at most limit, default 100).
func (s *Service) ExpireProposals(ctx context.Context, actor domain.Actor, limit int) (int, error) {
	if !actor.System {
		return 0, domain.Forbidden("system actor required")
	}
	n := 0
	err := s.update(ctx, func(tx store.Tx) error {
		ts, err := tx.ListExpiredProposals(s.now(), limit)
		if err != nil {
			return err
		}
		for _, t := range ts {
			if err := s.transition(tx, t.NetworkID, t, domain.CmdExpire, "", actor.RequestID); err != nil {
				return err
			}
			if err := s.emit(tx, t.NetworkID, "", actor.RequestID, auditEntry{Type: domain.ActorSystem, Action: domain.ActTxExpired,
				TargetType: "transaction", TargetID: t.ID, Subjects: []string{t.PayerMemberID, t.PayeeMemberID},
				Before: map[string]any{"status": "proposed"}, After: map[string]any{"status": "expired"}}); err != nil {
				return err
			}
			n++
		}
		return nil
	})
	if err != nil {
		return 0, err
	}
	return n, nil
}
