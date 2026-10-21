package service

import (
	"context"

	"github.com/djshellshoxxx/circula/internal/domain"
	"github.com/djshellshoxxx/circula/internal/store"
)

// BalanceView is a member's position in one network (FR-050, DOM-006/008).
// Balance may be negative (credit used). Available is what the member can newly
// commit as payer: Balance + CreditLimit - ReservedOutgoing; it may be negative
// after a limit reduction and is never a balance. AwaitingFromOthers is shown
// separately and does not increase Available.
type BalanceView struct {
	MemberID           string
	AccountState       domain.AccountState
	Balance            int64
	CreditLimit        int64
	ReservedOutgoing   int64
	Available          int64
	AwaitingFromOthers int64
	Decimals           int
	LedgerSeq          int64
}

func (s *Service) balanceView(tx store.Tx, net *domain.Network, m *domain.Member) (*BalanceView, error) {
	acc, err := tx.GetAccountByMember(net.ID, m.ID)
	if err != nil {
		return nil, err
	}
	now := s.now()
	res, err := tx.ReservedOutgoing(net.ID, acc.ID, now)
	if err != nil {
		return nil, err
	}
	in, err := tx.PendingIncoming(net.ID, acc.ID, now)
	if err != nil {
		return nil, err
	}
	seq, err := tx.LedgerSeq(net.ID)
	if err != nil {
		return nil, err
	}
	lim := acc.EffectiveLimit(net.Policy.DefaultCreditLimit, now)
	return &BalanceView{MemberID: m.ID, AccountState: domain.AccountStateFor(m.Status), Balance: acc.Balance,
		CreditLimit: lim, ReservedOutgoing: res, Available: acc.Balance + lim - res, AwaitingFromOthers: in,
		Decimals: net.Policy.Decimals, LedgerSeq: seq}, nil
}

// GetBalance returns the caller's own balance view. Suspended and departed
// members can still read their own (DOM-039, FR-014).
func (s *Service) GetBalance(ctx context.Context, actor domain.Actor, netID string) (*BalanceView, error) {
	var out *BalanceView
	err := s.view(ctx, func(tx store.Tx) error {
		p, err := s.open(tx, actor, netID, domain.ScopeRead)
		if err != nil {
			return err
		}
		if p.member == nil || p.member.Status == domain.MemberPending {
			return domain.Forbidden("no account")
		}
		out, err = s.balanceView(tx, p.net, p.member)
		return err
	})
	return out, err
}

// AdminGetMemberBalance lets an administrator see another member's balance and
// limit. The read is itself audited (matrix in 02, DAT-034).
func (s *Service) AdminGetMemberBalance(ctx context.Context, actor domain.Actor, netID, memberID string) (*BalanceView, error) {
	var out *BalanceView
	err := s.update(ctx, func(tx store.Tx) error {
		p, err := s.open(tx, actor, netID, domain.ScopeRead)
		if err != nil {
			return err
		}
		if err := p.requireAdmin(); err != nil {
			return err
		}
		m, err := tx.GetMember(netID, memberID)
		if err != nil {
			return domain.NotFound("member")
		}
		if out, err = s.balanceView(tx, p.net, m); err != nil {
			return err
		}
		return s.audit(tx, p, auditEntry{Type: domain.ActorAdmin, Action: domain.ActAccountViewed, TargetType: "member", TargetID: m.ID,
			Subjects: []string{m.ID}})
	})
	return out, err
}

// BalanceAt reconstructs the caller's balance at a ledger_seq by summing
// entries (DAT-021, FR-054). seq <= 0 means the latest.
func (s *Service) BalanceAt(ctx context.Context, actor domain.Actor, netID string, seq int64) (int64, error) {
	var out int64
	err := s.view(ctx, func(tx store.Tx) error {
		p, err := s.open(tx, actor, netID, domain.ScopeRead)
		if err != nil {
			return err
		}
		if p.member == nil || p.member.Status == domain.MemberPending {
			return domain.Forbidden("no account")
		}
		acc, err := tx.GetAccountByMember(netID, p.member.ID)
		if err != nil {
			return err
		}
		out, err = tx.SumEntries(acc.ID, seq)
		return err
	})
	return out, err
}
