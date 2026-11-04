package service

import (
	"context"
	"strings"

	"github.com/djshellshoxxx/circula/internal/domain"
	"github.com/djshellshoxxx/circula/internal/store"
)

// Challenges (GOV-016). A member can challenge a membership removal, a
// credit-limit reduction applied to them and a removal of their listing. A
// dispute outcome is contested through the one re-review (RequestReReview,
// FR-078), which has its own period, a different reviewer and an audit event.
//
// A challenge is decided by an administrator or reviewer who is neither the
// challenger nor the person who took the original decision. The decision
// carries reasons, an audit event and a notice to the member. "upheld" says
// the decision should be reversed; the decider then uses the ordinary
// administrator action (reinstate, set the limit, restore the listing), which
// is audited as usual. A rejected challenge changes nothing.

// Challenge kinds.
const (
	ChallengeRemoval        = "removal"
	ChallengeCreditLimit    = "credit_limit"
	ChallengeListingRemoval = "listing_removal"
)

// Challenge outcomes.
const (
	ChallengeUpheld   = "upheld"
	ChallengeRejected = "rejected"
)

// Challenge notification kinds and audit actions.
const (
	NotifChallengeAsked   = "challenge.requested"
	NotifChallengeDecided = "challenge.decided"

	ActChallengeOpened  = "challenge.opened"
	ActChallengeDecided = "challenge.decided"
)

// ChallengeInput opens a challenge.
type ChallengeInput struct {
	Kind      string
	SubjectID string // listing id for listing_removal
	Reason    string
}

// decisionActor finds who took the latest administrator decision of the given
// kinds on a target, other than the member themself. "" when none is found.
func decisionActor(tx store.Tx, netID, targetID, self string, actions ...string) (string, error) {
	want := map[string]bool{}
	for _, a := range actions {
		want[a] = true
	}
	var who string
	var after int64
	for {
		evs, err := tx.ListAudit(netID, store.AuditFilter{TargetID: targetID, AfterSeq: after, Limit: 500})
		if err != nil {
			return "", err
		}
		for _, e := range evs {
			after = e.Seq
			if want[e.Action] && e.ActorID != "" && e.ActorID != self {
				who = e.ActorID
			}
		}
		if len(evs) < 500 {
			return who, nil
		}
	}
}

// OpenChallenge records a member's challenge and tells the deciders who may
// take it (everyone with the reviewer or administrator role except the
// challenger and the person who made the original decision).
func (s *Service) OpenChallenge(ctx context.Context, actor domain.Actor, netID string, in ChallengeInput) (*store.Challenge, error) {
	var out *store.Challenge
	err := s.update(ctx, func(tx store.Tx) error {
		p, err := s.open(tx, actor, netID, domain.ScopeWrite)
		if err != nil {
			return err
		}
		if p.member == nil || p.member.Status == domain.MemberPending {
			return domain.Forbidden("membership cannot perform this action")
		}
		reason := strings.TrimSpace(in.Reason)
		if reason == "" {
			return domain.Invalid("reason", "required")
		}
		if err := boundedText("reason", reason, maxTextLen); err != nil {
			return err
		}
		me := p.member.ID
		var involved string
		var subject string
		switch in.Kind {
		case ChallengeRemoval:
			if p.member.Status != domain.MemberDeparted && p.member.Status != domain.MemberSuspended {
				return domain.InvalidState("only a removed or suspended membership can be challenged")
			}
			if involved, err = decisionActor(tx, netID, me, me, domain.ActMemberDeparted, domain.ActMemberSuspended); err != nil {
				return err
			}
		case ChallengeCreditLimit:
			acc, err := tx.GetAccountByMember(netID, me)
			if err != nil {
				return domain.NotFound("account")
			}
			if involved, err = decisionActor(tx, netID, acc.ID, me, domain.ActLimitChanged); err != nil {
				return err
			}
		case ChallengeListingRemoval:
			l, err := tx.GetListing(netID, in.SubjectID)
			if err != nil || l.MemberID != me {
				return domain.NotFound("listing")
			}
			subject = l.ID
			if involved, err = decisionActor(tx, netID, l.ID, me, domain.ActListingUpdated); err != nil {
				return err
			}
		default:
			return domain.Invalid("kind", "must be removal, credit_limit or listing_removal")
		}
		if involved == "" {
			return domain.InvalidState("no administrator decision of this kind was found to challenge")
		}
		c := store.Challenge{ID: s.id(), NetworkID: netID, MemberID: me, Kind: in.Kind, SubjectID: subject, Reason: reason,
			InvolvedID: involved, Status: "requested", CreatedAt: s.now()}
		if err := tx.InsertChallenge(c); err != nil {
			return err
		}
		ids, err := deciderIDs(tx, netID)
		if err != nil {
			return err
		}
		if err := s.notifyOthers(tx, netID, ids, me, NotifChallengeAsked, "A member challenged a decision",
			"A challenge awaits a reviewer who was not involved in the decision.", "/admin/challenges"); err != nil {
			return err
		}
		out = &c
		return s.audit(tx, p, auditEntry{Type: domain.ActorMember, Action: ActChallengeOpened, TargetType: "challenge", TargetID: c.ID,
			Subjects: []string{me}, After: map[string]any{"kind": c.Kind}})
	})
	return out, err
}

// ChallengeDecision decides a challenge.
type ChallengeDecision struct {
	ID        string
	Outcome   string // upheld | rejected
	Rationale string
}

// DecideChallenge records the decision. The decider must be an administrator
// or reviewer who is not the challenger and not the person who took the
// original decision; reasons are mandatory.
func (s *Service) DecideChallenge(ctx context.Context, actor domain.Actor, netID string, in ChallengeDecision) (*store.Challenge, error) {
	var out *store.Challenge
	err := s.update(ctx, func(tx store.Tx) error {
		p, err := s.open(tx, actor, netID, domain.ScopeWrite)
		if err != nil {
			return err
		}
		if err := p.requireDecider(); err != nil {
			return err
		}
		c, err := tx.GetChallenge(netID, in.ID)
		if err != nil {
			return domain.NotFound("challenge")
		}
		if c.Status != "requested" {
			return domain.InvalidState("this challenge has already been decided")
		}
		if p.member.ID == c.MemberID || p.member.ID == c.InvolvedID {
			return domain.Forbidden("a challenge is decided by someone who was not involved in the original decision")
		}
		if in.Outcome != ChallengeUpheld && in.Outcome != ChallengeRejected {
			return domain.Invalid("outcome", "must be upheld or rejected")
		}
		why := strings.TrimSpace(in.Rationale)
		if why == "" {
			return domain.Invalid("rationale", "required")
		}
		if err := boundedText("rationale", why, maxTextLen); err != nil {
			return err
		}
		c.Outcome, c.DecidedBy, c.Rationale, c.DecidedAt = in.Outcome, p.member.ID, why, s.now()
		if err := tx.DecideChallenge(*c); err != nil {
			return err
		}
		c.Status = "decided"
		if err := s.notify(tx, netID, c.MemberID, NotifChallengeDecided, "Your challenge was decided",
			"Your challenge was "+in.Outcome+". The reasons are on your challenge page.", "/challenges/"+c.ID, ""); err != nil {
			return err
		}
		out = c
		return s.audit(tx, p, auditEntry{Type: p.actorType(domain.ActorAdmin), Action: ActChallengeDecided, TargetType: "challenge", TargetID: c.ID,
			Reason: why, Subjects: []string{c.MemberID}, After: map[string]any{"outcome": in.Outcome, "kind": c.Kind}})
	})
	return out, err
}

// ListChallenges lists the caller's own challenges; deciders and administrators
// see all of the network's. status "" lists every status.
func (s *Service) ListChallenges(ctx context.Context, actor domain.Actor, netID, status string) ([]store.Challenge, error) {
	var out []store.Challenge
	err := s.view(ctx, func(tx store.Tx) error {
		p, err := s.open(tx, actor, netID, domain.ScopeRead)
		if err != nil {
			return err
		}
		who := p.memberID()
		if p.actor.System || (p.member != nil && p.requireDecider() == nil) {
			who = ""
		} else if p.member == nil {
			return domain.Forbidden("membership required")
		}
		out, err = tx.ListChallenges(netID, who, status, 200)
		return err
	})
	return out, err
}
