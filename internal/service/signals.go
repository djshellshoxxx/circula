package service

import (
	"context"
	"time"

	"github.com/djshellshoxxx/circula/internal/domain"
	store "github.com/djshellshoxxx/circula/internal/repository"
)

// Activity signals (SEC-026). After a trade proposal the proposer's velocity
// is evaluated, and after a trade completes the pair's reciprocal pattern;
// both use the network policy in force at that moment (domain/signals.go).
// A signal above the threshold is recorded once per period, audited without
// member subjects and announced to the network's administrators and dispute
// reviewers, except the members it concerns.
//
// Signals never block a trade. The evaluation runs at the end of the
// command's own unit of work, inside a savepoint: any failure rolls back the
// savepoint only, the trade commits as if signals were off, and the failure
// is reported to the SetSignalFailureHandler callback after the commit.

// NotifActivitySignal is the inbox kind telling a reviewer that a signal needs review.
const NotifActivitySignal = "review.activity_signal"

// activitySignalsLink is the reviewer page for signals, inside the network.
const activitySignalsLink = "/review/activity-signals"

// signalFailure is a signal evaluation that failed inside a committed unit.
type signalFailure struct {
	netID, kind string
	err         error
}

// SetSignalFailureHandler installs f (nil removes it). It is called after a
// unit of work commits, once for every activity signal evaluation in it that
// failed and was rolled back, so the operator can log it; the trade itself was
// not affected. f runs on the calling goroutine and must not call back into
// the Service. Call before serving starts.
func (s *Service) SetSignalFailureHandler(f func(netID, kind string, err error)) {
	s.onSignalFailure = f
}

// velocitySignal evaluates the velocity rule for the member who just
// proposed a trade.
func (s *Service) velocitySignal(tx store.Tx, p *principal, memberID string) {
	pol := p.net.Policy
	if !pol.ActivitySignalsEnabled {
		return
	}
	s.signal(tx, p, domain.SignalVelocity, func(now time.Time) (*store.ActivitySignal, error) {
		window := pol.VelocitySignalWindow()
		n, err := tx.CountProposalsSince(p.net.ID, memberID, now.Add(-window), now)
		if err != nil || n <= pol.VelocitySignalProposals {
			return nil, err
		}
		return &store.ActivitySignal{Kind: domain.SignalVelocity, MemberID: memberID, Window: window,
			Observed: n, Threshold: pol.VelocitySignalProposals}, nil
	})
}

// reciprocalSignal evaluates the reciprocal-pattern rule for the parties of
// a trade that just completed.
func (s *Service) reciprocalSignal(tx store.Tx, p *principal, t *domain.Transaction) {
	pol := p.net.Policy
	if !pol.ActivitySignalsEnabled || t.Kind != domain.KindTrade {
		return
	}
	s.signal(tx, p, domain.SignalReciprocal, func(now time.Time) (*store.ActivitySignal, error) {
		window := pol.ReciprocalSignalWindow()
		ab, ba, err := tx.CountCompletedTradesBetween(p.net.ID, t.PayerAccountID, t.PayeeAccountID, now.Add(-window), now)
		if err != nil {
			return nil, err
		}
		n, ok := domain.ReciprocalCount(ab, ba)
		if !ok || n <= pol.ReciprocalSignalTrades {
			return nil, nil
		}
		lo, hi := t.PayerMemberID, t.PayeeMemberID
		if hi < lo {
			lo, hi = hi, lo
		}
		return &store.ActivitySignal{Kind: domain.SignalReciprocal, MemberID: lo, OtherMemberID: hi, Window: window,
			Observed: n, Threshold: pol.ReciprocalSignalTrades}, nil
	})
}

// signal runs one evaluation in a savepoint and records a signal when check
// returns one. It never returns an error: a failure is rolled back and kept
// for the post-commit failure report.
func (s *Service) signal(tx store.Tx, p *principal, kind string, check func(now time.Time) (*store.ActivitySignal, error)) {
	err := tx.Savepoint(func() error {
		now := s.now()
		sig, err := check(now)
		if err != nil || sig == nil {
			return err
		}
		sig.ID, sig.NetworkID, sig.RaisedAt = s.id(), p.net.ID, now
		sig.PeriodStart = domain.SignalPeriodStart(now, sig.Window)
		inserted, err := tx.InsertActivitySignal(*sig)
		if err != nil || !inserted { // already raised in this period
			return err
		}
		// No subjects: the members concerned must not see it in their audit view.
		if err := s.emit(tx, p.net.ID, "", p.actor.RequestID, auditEntry{Type: domain.ActorSystem,
			Action: domain.ActActivitySignalRaised, TargetType: "activity_signal", TargetID: sig.ID,
			After: map[string]any{"kind": sig.Kind, "observed": sig.Observed, "threshold": sig.Threshold,
				"window_minutes": int(sig.Window / time.Minute), "period_start": sig.PeriodStart.UnixMilli()}}); err != nil {
			return err
		}
		return s.notifySignalReviewers(tx, p.net.ID, sig)
	})
	if err != nil {
		s.unit.signalFailures = append(s.unit.signalFailures, signalFailure{netID: p.net.ID, kind: kind, err: err})
	}
}

// notifySignalReviewers tells every active administrator and dispute reviewer
// of the network, except the members the signal concerns, that a signal needs
// review. The notice names no member and no count; the queue holds the detail.
func (s *Service) notifySignalReviewers(tx store.Tx, netID string, sig *store.ActivitySignal) error {
	return s.eachActiveMember(tx, netID, func(m *domain.Member) error {
		if m.ID == sig.MemberID || m.ID == sig.OtherMemberID || (!m.HasRole(domain.RoleAdmin) && !m.HasRole(domain.RoleReviewer)) {
			return nil
		}
		return s.notify(tx, netID, m.ID, NotifActivitySignal, "Activity to review",
			"Unusual activity crossed a review threshold. It is not a finding and did not block any trade.",
			activitySignalsLink, "activity-signal:"+sig.ID)
	})
}

// NotifActivitySignalSettings tells reviewers that the signal settings changed.
const NotifActivitySignalSettings = "review.activity_signal_settings"

func signalSettingsChanged(old, nw domain.Policy) bool {
	return old.ActivitySignalsEnabled != nw.ActivitySignalsEnabled ||
		old.VelocitySignalProposals != nw.VelocitySignalProposals || old.VelocitySignalWindowMinutes != nw.VelocitySignalWindowMinutes ||
		old.ReciprocalSignalTrades != nw.ReciprocalSignalTrades || old.ReciprocalSignalWindowDays != nw.ReciprocalSignalWindowDays
}

// noticeSignalSettings tells the other active administrators and dispute
// reviewers that the activity signal settings were changed, so one
// administrator cannot quietly switch signals off or loosen them (compare
// A3-01). The settings also appear in the members' rules log (GOV-002);
// the change itself is the audited policy.changed event.
func (s *Service) noticeSignalSettings(tx store.Tx, netID, changedBy string) error {
	return s.eachActiveMember(tx, netID, func(m *domain.Member) error {
		if m.ID == changedBy || (!m.HasRole(domain.RoleAdmin) && !m.HasRole(domain.RoleReviewer)) {
			return nil
		}
		return s.notify(tx, netID, m.ID, NotifActivitySignalSettings, "Activity signal settings changed",
			"An administrator changed when activity signals are raised. The activity signals page shows the settings in force.",
			activitySignalsLink, "")
	})
}

// ActivitySignalQuery selects signals for the reviewer queue, newest first.
type ActivitySignalQuery struct {
	Kind string // "", "velocity" or "reciprocal"
	Page Page
}

// ActivitySignalView is a signal with the current display names of the
// members it concerns (the anonymised name after erasure).
type ActivitySignalView struct {
	*store.ActivitySignal
	MemberName, OtherMemberName string
}

// ListActivitySignals lists the network's activity signals (SEC-026) to its
// administrators and dispute reviewers, under their current effective roles
// and the administrator MFA rule. Signals that concern the reader are left
// out. Plain members are refused; outsiders get not-found. The read changes
// nothing.
func (s *Service) ListActivitySignals(ctx context.Context, actor domain.Actor, netID string, q ActivitySignalQuery) ([]ActivitySignalView, error) {
	switch q.Kind {
	case "", domain.SignalVelocity, domain.SignalReciprocal:
	default:
		return nil, domain.Invalid("kind", "must be velocity or reciprocal")
	}
	var out []ActivitySignalView
	err := s.view(ctx, func(tx store.Tx) error {
		p, err := s.open(tx, actor, netID, domain.ScopeRead)
		if err != nil {
			return err
		}
		if err := p.requireDecider(); err != nil {
			return err
		}
		if err := validatePageIDs(q.Page.AfterID); err != nil {
			return err
		}
		rows, err := tx.ListActivitySignals(netID, store.ActivitySignalFilter{Kind: q.Kind, Viewer: p.memberID(),
			AfterID: q.Page.AfterID, Limit: clampPage(q.Page)})
		if err != nil {
			return err
		}
		names := map[string]string{}
		name := func(id string) string {
			if id == "" {
				return ""
			}
			if n, ok := names[id]; ok {
				return n
			}
			if m, err := tx.GetMember(netID, id); err == nil {
				names[id] = m.DisplayName
			}
			return names[id]
		}
		for _, r := range rows {
			out = append(out, ActivitySignalView{ActivitySignal: r, MemberName: name(r.MemberID), OtherMemberName: name(r.OtherMemberID)})
		}
		return nil
	})
	return out, err
}
