package service

import (
	"context"
	"errors"

	"github.com/djshellshoxxx/circula/internal/domain"
	"github.com/djshellshoxxx/circula/internal/store"
)

// ReverseInput reverses (part of) a completed transaction administratively.
type ReverseInput struct {
	TxID           string
	Amount         int64 // minor units; 0 = the full remaining reversible amount
	Reason         string
	IdempotencyKey string `json:"-"`
}

// ReverseTransaction posts a `reversal` transaction against a completed one
// (FR-066, DOM-036, DOM-019/020): the original payee is debited and the
// original payer credited. The original is never edited except for its status
// and reversed total. Reason is mandatory; administrators who are parties are
// refused (DOM-037); a reversal that would breach LED-5 is refused.
func (s *Service) ReverseTransaction(ctx context.Context, actor domain.Actor, netID string, in ReverseInput) (*domain.Transaction, error) {
	var out *domain.Transaction
	err := s.update(ctx, func(tx store.Tx) error {
		p, err := s.open(tx, actor, netID, domain.ScopeWrite)
		if err != nil {
			return err
		}
		if err := p.requireAdmin(); err != nil {
			return err
		}
		out, err = idempotent(s, tx, netID, actor, "tx.reverse", in.IdempotencyKey, in, func() (string, error) {
			if err := p.requireNetworkActive(); err != nil {
				return "", err
			}
			if err := requireReason(in.Reason); err != nil {
				return "", err
			}
			orig, err := tx.GetTransaction(netID, in.TxID)
			if err != nil {
				return "", domain.NotFound("transaction")
			}
			if orig.IsParty(p.memberID()) {
				return "", domain.Forbidden("administrators cannot reverse their own transactions")
			}
			if orig.Status != domain.TxCompleted { // disputed ones go through the dispute process
				return "", domain.InvalidState("transaction is %s; only completed transactions can be reversed here", orig.Status)
			}
			amt := in.Amount
			if amt == 0 {
				amt = orig.Amount - orig.ReversedTotal
			}
			rev, err := s.reverse(tx, p.net, orig, amt, in.Reason, p.memberID(), p.actor.RequestID)
			if err != nil {
				return "", err
			}
			if err := s.notifyOthers(tx, netID, []string{orig.PayerMemberID, orig.PayeeMemberID}, p.memberID(), NotifAdminReversal,
				"A trade was reversed by an administrator", "An administrator reversed all or part of a trade you were in. See your history.", "/trades/"+orig.ID); err != nil {
				return "", err
			}
			return rev.ID, s.audit(tx, p, auditEntry{Type: domain.ActorAdmin, Action: domain.ActTxReversed, TargetType: "transaction",
				TargetID: rev.ID, Reason: in.Reason, IdemRef: keyRef(in.IdempotencyKey),
				Subjects: []string{orig.PayerMemberID, orig.PayeeMemberID},
				Before:   map[string]any{"original": orig.ID}, After: map[string]any{"amount": amt, "original_status": orig.Status, "reversed_total": orig.ReversedTotal}})
		}, func(id string) (*domain.Transaction, error) { return tx.GetTransaction(netID, id) })
		return err
	})
	return out, err
}

// errBlockedByLimit marks a reversal refused by LED-5; disputes turn it into
// `posting_pending` instead of failing (DOM-021).
func blockedByLimit(err error) bool { return errors.Is(err, domain.ErrInsufficientCredit) }

// reverse validates and posts a reversal of amount against orig, then updates
// orig (reversed total and, via the state machine, status). The LED-5 check
// happens before any row is written, so a refusal leaves the unit clean and
// the caller may continue (disputes). orig is mutated in place.
func (s *Service) reverse(tx store.Tx, net *domain.Network, orig *domain.Transaction, amount int64,
	reason, createdBy, requestID string) (*domain.Transaction, error) {

	if orig.Kind == domain.KindReversal {
		return nil, domain.InvalidState("a reversal cannot be reversed")
	}
	if orig.Status != domain.TxCompleted && orig.Status != domain.TxDisputed {
		return nil, domain.InvalidState("transaction is %s", orig.Status)
	}
	remaining := orig.Amount - orig.ReversedTotal
	if amount <= 0 || amount > remaining {
		return nil, domain.Invalid("amount", "must be between 1 and the remaining reversible amount "+fmtAmt(net, remaining))
	}
	debited, err := tx.GetAccount(net.ID, orig.PayeeAccountID)
	if err != nil {
		return nil, err
	}
	if err := checkCapacity(net, debited, amount, s.now()); err != nil {
		return nil, err
	}
	rev, err := s.newInternalTx(tx, net, domain.KindReversal, orig.PayeeAccountID, orig.PayerAccountID, amount, reason, createdBy, orig.ID)
	if err != nil {
		return nil, err
	}
	if err := s.postTx(tx, net, rev, createdBy, requestID); err != nil {
		return nil, err
	}
	orig.ReversedTotal += amount
	cmd := domain.CmdReversePartial
	if orig.ReversedTotal == orig.Amount {
		cmd = domain.CmdReverseFull
	}
	if err := s.transition(tx, net.ID, orig, cmd, createdBy, requestID); err != nil {
		return nil, err
	}
	rev.PayerMemberID, rev.PayeeMemberID = orig.PayeeMemberID, orig.PayerMemberID
	return rev, nil
}
