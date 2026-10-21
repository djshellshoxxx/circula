package service

import (
	"context"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/djshellshoxxx/circula/internal/domain"
	"github.com/djshellshoxxx/circula/internal/store"
)

// Notification kinds for governance notices (GOV-003, GOV-009, GOV-010, GOV-011).
const (
	NotifPolicyNotice   = "policy.notice"
	NotifTermsPublished = "terms.published"
	NotifMemberLeft     = "member.left"
	NotifMemberRemoved  = "member.removed"
)

const maxTermsBody = 20000

func govRepo(tx store.Tx) (store.GovRepo, error) {
	r, ok := tx.(store.GovRepo)
	if !ok {
		return nil, domain.Errf(domain.CodeInternal, "store does not support terms")
	}
	return r, nil
}

// notifyActive writes one inbox entry for every active member except skip.
func (s *Service) notifyActive(tx store.Tx, netID, skip, kind, title, body, link string) error {
	after := ""
	for {
		ms, err := tx.ListMembers(netID, store.MemberFilter{Statuses: []domain.MemberStatus{domain.MemberActive}, AfterID: after, Limit: 500})
		if err != nil {
			return err
		}
		ids := make([]string, 0, len(ms))
		for _, m := range ms {
			ids = append(ids, m.ID)
			after = m.ID
		}
		if err := s.notifyOthers(tx, netID, ids, skip, kind, title, body, link); err != nil {
			return err
		}
		if len(ms) < 500 {
			return nil
		}
	}
}

// PublishTerms publishes a new version of the network's onboarding terms
// (GOV-009). Every active member is told; each must accept the new version
// before their next trade.
func (s *Service) PublishTerms(ctx context.Context, actor domain.Actor, netID, body string) (*domain.Terms, error) {
	var out *domain.Terms
	err := s.update(ctx, func(tx store.Tx) error {
		p, err := s.open(tx, actor, netID, domain.ScopeWrite)
		if err != nil {
			return err
		}
		if err := p.requireAdmin(); err != nil {
			return err
		}
		if err := p.requireLasting(); err != nil {
			return err
		}
		body = strings.TrimSpace(body)
		if n := utf8.RuneCountInString(body); n < 1 || n > maxTermsBody {
			return domain.Invalid("body", "must be 1 to 20000 characters")
		}
		g, err := govRepo(tx)
		if err != nil {
			return err
		}
		cur, err := g.LatestTerms(netID)
		if err != nil {
			return err
		}
		v := 1
		if cur != nil {
			if cur.Body == body {
				return domain.Invalid("body", "is the same as the current terms")
			}
			v = cur.Version + 1
		}
		t := domain.Terms{Version: v, Body: body, PublishedBy: p.memberID(), PublishedAt: s.now()}
		if err := g.InsertTerms(netID, t); err != nil {
			return err
		}
		if err := s.notifyActive(tx, netID, "", NotifTermsPublished, "Updated network terms",
			"The network published version "+strconv.Itoa(v)+" of its rules and terms. Read and accept them before your next trade.", "/terms"); err != nil {
			return err
		}
		out = &t
		return s.audit(tx, p, auditEntry{Type: domain.ActorAdmin, Action: domain.ActTermsPublished, TargetType: "terms",
			TargetID: netID, After: map[string]any{"version": v}})
	})
	return out, err
}

// TermsView is the current terms and where the caller stands against them.
type TermsView struct {
	Current  *domain.Terms // nil when the network has published none
	Accepted int           // highest version the caller accepted, 0 for none
}

// NeedsAcceptance reports whether the caller must accept before trading.
func (v TermsView) NeedsAcceptance() bool { return v.Current != nil && v.Accepted < v.Current.Version }

// CurrentTerms returns the network's current terms and the caller's accepted
// version. Any member may read them, whatever their status.
func (s *Service) CurrentTerms(ctx context.Context, actor domain.Actor, netID string) (TermsView, error) {
	var out TermsView
	err := s.view(ctx, func(tx store.Tx) error {
		p, err := s.open(tx, actor, netID, domain.ScopeRead)
		if err != nil {
			return err
		}
		g, err := govRepo(tx)
		if err != nil {
			return err
		}
		if out.Current, err = g.LatestTerms(netID); err != nil {
			return err
		}
		if p.member != nil {
			out.Accepted, err = g.AcceptedTermsVersion(netID, p.member.ID)
		}
		return err
	})
	return out, err
}

// AcceptTerms records the caller's acceptance of terms version v, which must
// be the current one (GOV-009). Accepting again is a no-op.
func (s *Service) AcceptTerms(ctx context.Context, actor domain.Actor, netID string, v int) error {
	return s.update(ctx, func(tx store.Tx) error {
		p, err := s.open(tx, actor, netID, domain.ScopeWrite)
		if err != nil {
			return err
		}
		if p.member == nil || (p.member.Status != domain.MemberActive && p.member.Status != domain.MemberPending) {
			return domain.Forbidden("only an active member can accept the terms")
		}
		g, err := govRepo(tx)
		if err != nil {
			return err
		}
		cur, err := g.LatestTerms(netID)
		if err != nil {
			return err
		}
		if cur == nil {
			return domain.InvalidState("the network has no terms to accept")
		}
		if v != cur.Version {
			return domain.Invalid("version", "is not the current terms version ("+strconv.Itoa(cur.Version)+")")
		}
		had, err := g.AcceptedTermsVersion(netID, p.member.ID)
		if err != nil {
			return err
		}
		if had >= v {
			return nil
		}
		if err := g.RecordTermsAcceptance(netID, p.member.ID, v, s.now()); err != nil {
			return err
		}
		return s.audit(tx, p, auditEntry{Type: domain.ActorMember, Action: domain.ActTermsAccepted, TargetType: "terms",
			TargetID: netID, Subjects: []string{p.member.ID}, After: map[string]any{"version": v}})
	})
}

// requireTerms refuses a trade action while the member has not accepted the
// network's current terms (GOV-009). Networks without terms are unaffected,
// and so are system actors.
func (s *Service) requireTerms(tx store.Tx, p *principal) error {
	if p.member == nil {
		return nil
	}
	g, ok := tx.(store.GovRepo)
	if !ok {
		return nil
	}
	cur, err := g.LatestTerms(p.net.ID)
	if err != nil || cur == nil {
		return err
	}
	got, err := g.AcceptedTermsVersion(p.net.ID, p.member.ID)
	if err != nil {
		return err
	}
	if got < cur.Version {
		return domain.Errf(domain.CodeTermsRequired, "accept the network's current terms (version %d) before trading", cur.Version)
	}
	return nil
}

// joinTerms checks, at join, that the person accepted the terms version they
// were shown, and records the acceptance for the new member (GOV-009). A
// network without terms needs no acceptance.
func (s *Service) joinTerms(tx store.Tx, netID string, accepted int) (record func(memberID string) error, err error) {
	g, ok := tx.(store.GovRepo)
	if !ok {
		return func(string) error { return nil }, nil
	}
	cur, err := g.LatestTerms(netID)
	if err != nil {
		return nil, err
	}
	if cur == nil {
		return func(string) error { return nil }, nil
	}
	if accepted != cur.Version {
		return nil, domain.Invalid("terms_version", "accept the network's current terms (version "+strconv.Itoa(cur.Version)+") to join")
	}
	return func(memberID string) error { return g.RecordTermsAcceptance(netID, memberID, cur.Version, s.now()) }, nil
}

// InvitationTerms returns the terms of the network an invitation leads to,
// for display before joining; nil when the network has none. Unknown, used,
// expired and revoked tokens are not-found (SEC-005).
func (s *Service) InvitationTerms(ctx context.Context, token string) (*domain.Terms, error) {
	var out *domain.Terms
	err := s.view(ctx, func(tx store.Tx) error {
		inv, err := tx.GetInvitationByHash(hashToken(token))
		if err != nil || inv.UsedAt != nil || inv.RevokedAt != nil || !s.now().Before(inv.ExpiresAt) {
			return domain.NotFound("invitation")
		}
		g, err := govRepo(tx)
		if err != nil {
			return err
		}
		out, err = g.LatestTerms(inv.NetworkID)
		return err
	})
	return out, err
}

// NetworkTerms returns a network's current terms for the join page; nil when
// none. The text is meant for people about to join, so it is public.
func (s *Service) NetworkTerms(ctx context.Context, netID string) (*domain.Terms, error) {
	var out *domain.Terms
	err := s.view(ctx, func(tx store.Tx) error {
		g, err := govRepo(tx)
		if err != nil {
			return err
		}
		out, err = g.LatestTerms(netID)
		return err
	})
	return out, err
}

// PolicyLogEntry is one row of the member-visible rules change log
// (GOV-002, GOV-003).
type PolicyLogEntry struct {
	Version      int
	Changes      map[string]any // changed keys with their new values
	Previous     map[string]any // the same keys before the change
	Reason       string
	AnnouncedAt  time.Time
	EffectiveAt  time.Time
	NoticeDays   int
	NoticeReason string
	Pending      bool // not in force yet
	Withdrawn    bool // cancelled before it took effect
}

// PolicyLog lists every policy version, newest first, with the change it made,
// when it was announced and when it takes (or took) effect. Active, suspended
// and departed members may read it; pending ones may not.
func (s *Service) PolicyLog(ctx context.Context, actor domain.Actor, netID string) ([]PolicyLogEntry, error) {
	var out []PolicyLogEntry
	err := s.view(ctx, func(tx store.Tx) error {
		p, err := s.open(tx, actor, netID, domain.ScopeRead)
		if err != nil {
			return err
		}
		if p.member != nil && p.member.Status == domain.MemberPending {
			return domain.Forbidden("membership cannot read the rules log")
		}
		g, err := govRepo(tx)
		if err != nil {
			return err
		}
		rows, err := g.ListPolicyVersions(netID)
		if err != nil {
			return err
		}
		now := s.now()
		// A version is withdrawn when a later one is in force by the time it
		// would have taken effect (a cancelled scheduled change).
		withdrawn := make([]bool, len(rows))
		for i := range rows {
			for j := i + 1; j < len(rows); j++ {
				if !rows[j].EffectiveAt.After(rows[i].EffectiveAt) {
					withdrawn[i] = true
				}
			}
		}
		prev := -1 // last version that was not withdrawn
		for i := range rows {
			r := rows[i]
			e := PolicyLogEntry{Version: r.Version, Reason: r.Reason, AnnouncedAt: r.AnnouncedAt, EffectiveAt: r.EffectiveAt,
				NoticeDays: r.NoticeDays, NoticeReason: r.NoticeReason, Withdrawn: withdrawn[i], Pending: !withdrawn[i] && r.EffectiveAt.After(now)}
			if prev >= 0 {
				e.Changes, e.Previous = policyDiff(rows[prev].Policy, r.Policy, false), policyDiff(rows[prev].Policy, r.Policy, true)
			} else {
				e.Changes = policyMap(r.Policy)
			}
			delete(e.Changes, "dual_control_threshold")
			delete(e.Previous, "dual_control_threshold")
			if !withdrawn[i] {
				prev = i
			}
			out = append([]PolicyLogEntry{e}, out...)
		}
		return nil
	})
	return out, err
}
