package service

import (
	"github.com/djshellshoxxx/circula/internal/domain"
	store "github.com/djshellshoxxx/circula/internal/repository"
)

// reverseDispute records the posted reversal under its own transaction target
// as well as the caller's separate dispute decision event (A1-09 / FR-081).
// Posting and both events share the caller's write transaction; a failed audit
// rolls the whole decision back. A credit-blocked decision emits no reversal
// event because reverse returns before posting. Existing audit rows stay intact.
func (s *Service) reverseDispute(tx store.Tx, p *principal, original *domain.Transaction,
	amount int64, reason, disputeID, key string) (*domain.Transaction, error) {
	reversal, err := s.reverse(tx, p.net, original, amount, reason, p.memberID(), p.actor.RequestID)
	if err != nil {
		return nil, err
	}
	if err := s.audit(tx, p, auditEntry{Type: p.deciderType(), Action: domain.ActTxReversed,
		TargetType: "transaction", TargetID: reversal.ID, Reason: reason, IdemRef: keyRef(key),
		Subjects: []string{original.PayerMemberID, original.PayeeMemberID},
		Before:   map[string]any{"original": original.ID},
		After: map[string]any{"amount": amount, "original_status": original.Status,
			"reversed_total": original.ReversedTotal, "dispute": disputeID}}); err != nil {
		return nil, err
	}
	return reversal, nil
}
