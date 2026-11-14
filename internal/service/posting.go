package service

import (
	"time"

	"github.com/djshellshoxxx/circula/internal/domain"
	"github.com/djshellshoxxx/circula/internal/store"
)

// checkCapacity applies LED-5 to a prospective debit of amount on payer: a
// non-system account must not end below -(effective limit). It is evaluated
// against the cached balance read inside the caller's write transaction, which
// is serialised (BEGIN IMMEDIATE + write gate), so no concurrent posting can
// interleave between this check and the entries (DOM-014, ARC-006).
func checkCapacity(net *domain.Network, payer *domain.Account, amount int64, now time.Time) error {
	if payer.IsSystem {
		return nil // DOM-026: the system account has no limit
	}
	limit := payer.EffectiveLimit(net.Policy.DefaultCreditLimit, now)
	next, ok := domain.AddChecked(payer.Balance, -amount)
	if !ok {
		return domain.Errf(domain.CodeInsufficientCredit, "amount overflows balance")
	}
	if next < -limit {
		avail := payer.Balance + limit
		if avail < 0 {
			avail = 0
		}
		return domain.Errf(domain.CodeInsufficientCredit, "payer has %s available",
			domain.FormatAmount(avail, net.Policy.Decimals))
	}
	return nil
}

// postTx posts transaction t (already inserted as proposed): LED-5 check on
// the freshly read payer account, two equal and opposite entries, cached
// balance updates, status proposed -> completed with its status event. It
// does NOT write the audit event; the caller does, in the same unit (LED-6).
// On any error the caller must return it so the unit rolls back (LED-4).
func (s *Service) postTx(tx store.Tx, net *domain.Network, t *domain.Transaction, actorMember string, requestID string) error {
	now := s.now()
	payer, err := tx.GetAccount(net.ID, t.PayerAccountID)
	if err != nil {
		return err
	}
	payee, err := tx.GetAccount(net.ID, t.PayeeAccountID)
	if err != nil {
		return err
	}
	// A3-03: both sides, system account included, stay within +-MaxBalance.
	if _, ok := domain.AddBounded(payer.Balance, -t.Amount); !ok {
		return domain.Errf(domain.CodeValidation, "amount overflows payer balance")
	}
	if _, ok := domain.AddBounded(payee.Balance, t.Amount); !ok {
		return domain.Errf(domain.CodeValidation, "amount overflows payee balance")
	}
	if err := checkCapacity(net, payer, t.Amount, now); err != nil {
		return err
	}
	next, err := domain.NextTxStatus(t.Status, domain.CmdAccept)
	if err != nil {
		return err
	}
	seq, err := tx.NextLedgerSeq(net.ID)
	if err != nil {
		return err
	}
	for _, e := range []*domain.Entry{
		{ID: s.id(), NetworkID: net.ID, TxID: t.ID, AccountID: payer.ID, Amount: -t.Amount, LedgerSeq: seq, PostedAt: now},
		{ID: s.id(), NetworkID: net.ID, TxID: t.ID, AccountID: payee.ID, Amount: t.Amount, LedgerSeq: seq, PostedAt: now},
	} {
		if err := tx.InsertEntry(e); err != nil {
			return err
		}
		if err := tx.AddBalance(net.ID, e.AccountID, e.Amount); err != nil {
			return err
		}
	}
	s.unit.posts++
	prev := t.Status
	t.Status, t.CompletedAt, t.LedgerSeq = next, &now, seq
	if err := tx.UpdateTransaction(t); err != nil {
		return err
	}
	return tx.InsertStatusEvent(&domain.StatusEvent{ID: s.id(), NetworkID: net.ID, TxID: t.ID,
		From: prev, To: next, ActorID: actorMember, RequestID: requestID, OccurredAt: now})
}

// newInternalTx inserts a proposed transaction for kinds that complete on
// creation (reversal, adjustment, grant) and returns it ready for postTx.
func (s *Service) newInternalTx(tx store.Tx, net *domain.Network, kind domain.TxKind, payerAcc, payeeAcc string,
	amount int64, reason, createdBy, originalID string) (*domain.Transaction, error) {

	if amount <= 0 {
		return nil, domain.Invalid("amount", "must be greater than zero")
	}
	t := &domain.Transaction{ID: s.id(), NetworkID: net.ID, Kind: kind, Status: domain.TxProposed,
		PayerAccountID: payerAcc, PayeeAccountID: payeeAcc, Amount: amount, Reason: reason,
		CreatedBy: createdBy, OriginalTxID: originalID, ProposedAt: s.now()}
	if err := tx.InsertTransaction(t); err != nil {
		return nil, err
	}
	if err := tx.InsertStatusEvent(&domain.StatusEvent{ID: s.id(), NetworkID: net.ID, TxID: t.ID,
		From: "", To: domain.TxProposed, ActorID: createdBy, OccurredAt: t.ProposedAt}); err != nil {
		return nil, err
	}
	return t, nil
}

func fmtAmt(net *domain.Network, v int64) string { return domain.FormatAmount(v, net.Policy.Decimals) }
