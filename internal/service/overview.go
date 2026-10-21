package service

import (
	"context"
	"time"

	"github.com/djshellshoxxx/circula/internal/domain"
	"github.com/djshellshoxxx/circula/internal/store"
)

// Thresholds behind the overview's "older than" and "ending soon" counts (FR-067).
const (
	OverviewStaleAfter   = 48 * time.Hour
	OverviewInviteWithin = 72 * time.Hour
)

// AdminOverview is what needs an administrator's attention (FR-067). Every count is computed
// from the ledger, member, dispute and invitation tables when it is read, never cached;
// AsOf says when.
type AdminOverview struct {
	AsOf time.Time

	PendingMembers int
	// StaleProposals counts live (unexpired) proposed trades older than StaleAfter.
	StaleProposals int
	StaleAfter     time.Duration
	// OpenDisputes counts disputes waiting for or under review.
	OpenDisputes int
	// PostingPending counts disputes decided but whose reversal is not yet posted.
	PostingPending int
	// NearLimit counts active members whose credit in use is at or beyond their effective limit.
	NearLimit int
	// ExpiringInvites counts unused, unrevoked invitations that end within InviteWithin.
	ExpiringInvites int
	InviteWithin    time.Duration

	// LastCheck is the newest recorded ledger consistency check; nil when none has run.
	LastCheck *CheckSummary

	// Frozen is set while the network is read-only after a failed ledger check.
	Frozen *FreezeInfo
}

// CheckSummary is the outcome of a recorded consistency check.
type CheckSummary struct {
	At time.Time
	OK bool
}

// AdminOverview returns the administrator overview counts (FR-067).
func (s *Service) AdminOverview(ctx context.Context, actor domain.Actor, netID string) (*AdminOverview, error) {
	var out *AdminOverview
	err := s.view(ctx, func(tx store.Tx) error {
		p, err := s.open(tx, actor, netID, domain.ScopeRead)
		if err != nil {
			return err
		}
		if err := p.requireAdmin(); err != nil {
			return err
		}
		repo, ok := tx.(store.OverviewRepo)
		mp, ok2 := tx.(store.MemberPager)
		if !ok || !ok2 {
			return domain.Errf(domain.CodeInternal, "store does not support the overview")
		}
		now := s.now()
		c, err := repo.OverviewCounts(netID, store.OverviewParams{Now: now, DefaultLimit: p.net.Policy.DefaultCreditLimit,
			StaleAfter: OverviewStaleAfter, InviteWithin: OverviewInviteWithin})
		if err != nil {
			return err
		}
		pend, err := mp.CountMembers(netID, []domain.MemberStatus{domain.MemberPending}, "")
		if err != nil {
			return err
		}
		out = &AdminOverview{AsOf: now, PendingMembers: pend, StaleProposals: c.StaleProposals, StaleAfter: OverviewStaleAfter,
			OpenDisputes: c.OpenDisputes, PostingPending: c.PostingPending, NearLimit: c.NearLimit,
			ExpiringInvites: c.ExpiringInvites, InviteWithin: OverviewInviteWithin}
		if c.LastCheck != nil {
			out.LastCheck = &CheckSummary{At: c.LastCheck.Finished, OK: c.LastCheck.OK}
		}
		if fi, ok := s.FrozenInfo(netID); ok {
			out.Frozen = &fi
		}
		return nil
	})
	return out, err
}
