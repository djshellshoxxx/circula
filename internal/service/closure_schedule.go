package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"

	"github.com/djshellshoxxx/circula/internal/domain"
	"github.com/djshellshoxxx/circula/internal/store"
)

// Notification kinds and audit actions for scheduled closure (GOV-025).
const (
	NotifClosureScheduled = "network.closure_scheduled"
	NotifClosureCancelled = "network.closure_cancelled"
	NotifClosureBlocked   = "network.closure_blocked"

	ActClosureScheduled = "network.closure_scheduled"
	ActClosureCancelled = "network.closure_cancelled"
	ActClosureExported  = "network.closure_exported"
	ActClosureErased    = "network.closure_erased"
)

// Scheduled closure (GOV-025, FR-068, DEC-025).
//
// ScheduleClosure announces a closing date at least 30 days away. Trading
// continues until then; members are told. At the date the network is frozen
// for proposals, acceptances, postings and joins (derived from the clock in
// lifecycle, so no job has to run for the freeze). FinaliseClosures then
// cancels open proposals with notices and, when no dispute is unresolved,
// completes the closure and records the final export checksum. Read and
// export access continue for ReadOnlyDays (90) after completion, after which
// EraseClosedNetworks anonymises the members through AnonymiseMember.

// lifecycle derives the in-memory status the clock and the roster imply:
// closing (closure date reached, closure not finished) and the administrator
// required state (GOV-020). Nothing is stored.
func (s *Service) lifecycle(tx store.Tx, n *domain.Network) error {
	if n.Status != domain.NetworkActive {
		return nil
	}
	if c := n.Closure; c != nil && !c.Done() && c.Due(s.now()) {
		n.Status = domain.NetworkClosing
		return nil
	}
	k, err := tx.CountActiveAdmins(n.ID)
	if err != nil {
		return err
	}
	n.AdminRequired = k == 0
	return nil
}

// ClosureInput schedules a closure.
type ClosureInput struct {
	Reason string
	// NoticeDays is the notice before the closure date: 0 selects the default
	// of 30 days; fewer than 30 is refused (DEC-025).
	NoticeDays int
}

// ScheduleClosure announces a closure (GOV-025 a). It is refused while a
// dispute is unresolved; the error lists them. Open proposals are not a
// blocker: they are cancelled at the closure date.
func (s *Service) ScheduleClosure(ctx context.Context, actor domain.Actor, netID string, in ClosureInput) (*domain.Closure, error) {
	var out *domain.Closure
	err := s.update(ctx, func(tx store.Tx) error {
		p, err := s.open(tx, actor, netID, domain.ScopeWrite)
		if err != nil {
			return err
		}
		if err := p.requireAdmin(); err != nil {
			return err
		}
		switch {
		case p.net.Status == domain.NetworkClosed:
			return domain.InvalidState("network is already closed")
		case p.net.Closure != nil:
			return domain.InvalidState("a closure is already scheduled for %s; cancel it first", p.net.Closure.ClosingAt.Format("2006-01-02"))
		case p.net.Status != domain.NetworkActive:
			return domain.InvalidState("network is %s; only an active network can be closed", p.net.Status)
		}
		if err := requireReason(in.Reason); err != nil {
			return err
		}
		days := in.NoticeDays
		if days == 0 {
			days = domain.MinClosureNoticeDays
		}
		if days < domain.MinClosureNoticeDays || days > domain.MaxClosureNoticeDays {
			return domain.Invalid("notice_days", fmt.Sprintf("must be %d to %d", domain.MinClosureNoticeDays, domain.MaxClosureNoticeDays))
		}
		blockers, err := closureBlockers(tx, netID)
		if err != nil {
			return err
		}
		if len(blockers) > 0 {
			return &domain.Error{Code: domain.CodeBlocked, Msg: "network has unresolved items", Blockers: blockers}
		}
		now := s.now()
		c := domain.Closure{NetworkID: netID, Reason: strings.TrimSpace(in.Reason), ScheduledBy: p.memberID(), ScheduledAt: now,
			ClosingAt: now.AddDate(0, 0, days), NoticeDays: days, ReadOnlyDays: domain.ClosureReadOnlyDays}
		if ok, err := tx.PutClosure(c); err != nil {
			return err
		} else if !ok {
			return domain.InvalidState("a closure is already scheduled")
		}
		if err := s.audit(tx, p, auditEntry{Type: domain.ActorAdmin, Action: ActClosureScheduled, TargetType: "network", TargetID: netID,
			Reason: c.Reason, After: map[string]any{"closing_at": c.ClosingAt.UnixMilli(), "notice_days": days}}); err != nil {
			return err
		}
		if err := s.notifyMembers(tx, netID, false, NotifClosureScheduled, "This network will close",
			"The network closes on "+c.ClosingAt.Format("2 Jan 2006")+": "+c.Reason+". You can keep trading until then; open proposals are cancelled on that date. "+
				fmt.Sprintf("History stays readable and you can download your data for %d days afterwards.", c.ReadOnlyDays)); err != nil {
			return err
		}
		out = &c
		return nil
	})
	return out, err
}

// CancelClosure withdraws a scheduled closure before its date (GOV-025).
func (s *Service) CancelClosure(ctx context.Context, actor domain.Actor, netID, reason string) error {
	return s.update(ctx, func(tx store.Tx) error {
		p, err := s.open(tx, actor, netID, domain.ScopeWrite)
		if err != nil {
			return err
		}
		if err := p.requireAdmin(); err != nil {
			return err
		}
		if err := requireReason(reason); err != nil {
			return err
		}
		if p.net.Closure == nil {
			return domain.InvalidState("no closure is scheduled")
		}
		if p.net.Status != domain.NetworkActive {
			return domain.InvalidState("the closure date has passed; the closure can no longer be cancelled")
		}
		if ok, err := tx.DeleteOpenClosure(netID); err != nil {
			return err
		} else if !ok {
			return domain.InvalidState("no closure is scheduled")
		}
		if err := s.audit(tx, p, auditEntry{Type: domain.ActorAdmin, Action: ActClosureCancelled, TargetType: "network", TargetID: netID,
			Reason: strings.TrimSpace(reason), Before: map[string]any{"closing_at": p.net.Closure.ClosingAt.UnixMilli()}}); err != nil {
			return err
		}
		return s.notifyMembers(tx, netID, false, NotifClosureCancelled, "The closure is cancelled",
			"The planned closure of this network was cancelled: "+strings.TrimSpace(reason)+".")
	})
}

// FinaliseClosures completes every closure whose date has passed (GOV-025
// b, c, e). It is the body of a scheduled job and is idempotent. System actors
// only. A network with an unresolved dispute stays "closing" (frozen) and its
// administrators are told once; the next run retries. Returns the number of
// closures completed.
func (s *Service) FinaliseClosures(ctx context.Context, actor domain.Actor) (int, error) {
	if !actor.System {
		return 0, domain.Forbidden("system actor required")
	}
	var due []domain.Closure
	if err := s.view(ctx, func(tx store.Tx) (err error) {
		due, err = tx.ListClosures(true)
		return err
	}); err != nil {
		return 0, err
	}
	done := 0
	for _, c := range due {
		if !c.Due(s.now()) || s.IsFrozen(c.NetworkID) {
			continue
		}
		closed, err := s.finaliseOne(ctx, actor, c.NetworkID)
		if err != nil {
			return done, err
		}
		if !closed {
			continue
		}
		if err := s.recordFinalExport(ctx, actor, c.NetworkID); err != nil {
			return done, err
		}
		done++
	}
	return done, nil
}

func (s *Service) finaliseOne(ctx context.Context, actor domain.Actor, netID string) (bool, error) {
	closed := false
	err := s.update(ctx, func(tx store.Tx) error {
		n, err := tx.GetNetwork(netID)
		if err != nil || n.Status != domain.NetworkActive || n.Closure == nil || n.Closure.Done() {
			return nil
		}
		// (c) cancel open proposals, telling both parties.
		for {
			props, err := tx.ListTransactions(netID, store.TxFilter{Statuses: []domain.TxStatus{domain.TxProposed}, Limit: 200})
			if err != nil {
				return err
			}
			for _, t := range props {
				if err := s.transition(tx, netID, t, domain.CmdCancel, "", actor.RequestID); err != nil {
					return err
				}
				for _, id := range []string{t.PayerMemberID, t.PayeeMemberID} {
					if err := s.notify(tx, netID, id, NotifTradeCancelled, "A trade was cancelled",
						"The network is closing, so the open trade proposal was cancelled.", "/trades/"+t.ID, ""); err != nil {
						return err
					}
				}
				if err := s.emit(tx, netID, "", actor.RequestID, auditEntry{Type: domain.ActorSystem, Action: domain.ActTxCancelled,
					TargetType: "transaction", TargetID: t.ID, Reason: "network closed", Subjects: []string{t.PayerMemberID, t.PayeeMemberID},
					Before: map[string]any{"status": "proposed"}, After: map[string]any{"status": "cancelled"}}); err != nil {
					return err
				}
			}
			if len(props) < 200 {
				break
			}
		}
		blockers, err := closureBlockers(tx, netID)
		if err != nil {
			return err
		}
		if len(blockers) > 0 {
			return s.notifyAdmins(tx, netID, NotifClosureBlocked, "Closure is waiting for disputes",
				"The closing date has passed but the network cannot close until these are resolved: "+strings.Join(blockers, ", ")+".", "closure-blocked")
		}
		now := s.now()
		if err := tx.SetNetworkStatus(netID, domain.NetworkClosed); err != nil {
			return err
		}
		if err := tx.SetClosureDone(netID, now); err != nil {
			return err
		}
		if err := s.emit(tx, netID, "", actor.RequestID, auditEntry{Type: domain.ActorSystem, Action: domain.ActNetworkClosed, TargetType: "network",
			TargetID: netID, Reason: n.Closure.Reason, Before: map[string]any{"status": domain.NetworkActive}, After: map[string]any{"status": domain.NetworkClosed}}); err != nil {
			return err
		}
		closed = true
		return s.notifyMembers(tx, netID, false, NotifNetworkClosed, "This network is closed",
			"The network is closed: "+n.Closure.Reason+". History stays readable and you can download your data for "+
				fmt.Sprintf("%d days; new trades are not possible.", n.Closure.ReadOnlyDays))
	})
	return closed, err
}

// notifyAdmins tells the active administrators once (dedupe key).
func (s *Service) notifyAdmins(tx store.Tx, netID, kind, title, body, dedupe string) error {
	ids, err := deciderIDs(tx, netID)
	if err != nil {
		return err
	}
	for _, id := range ids {
		if err := s.notify(tx, netID, id, kind, title, body, "/admin/network", dedupe); err != nil {
			return err
		}
	}
	return nil
}

// ManifestChecksum is the closure checksum: SHA-256 over one line per export
// file ("name TAB rows TAB sha256"), in manifest order. It binds every file's
// content through the per-file digests (GOV-025 e).
func ManifestChecksum(m *Manifest) string {
	h := sha256.New()
	for _, f := range m.Files {
		fmt.Fprintf(h, "%s\t%d\t%s\n", f.Name, f.Rows, f.SHA256)
	}
	return hex.EncodeToString(h.Sum(nil))
}

// recordFinalExport generates the ledger export of the closed network and
// stores its checksum with the reconciliation summary (sum of entries and
// balance totals, both zero in a sound ledger).
func (s *Service) recordFinalExport(ctx context.Context, actor domain.Actor, netID string) error {
	man, err := s.Export(ctx, actor, netID, ExportLedger, FormatJSON, &MemSink{})
	if err != nil {
		return err
	}
	sum := ManifestChecksum(man)
	return s.update(ctx, func(tx store.Tx) error {
		bal, ent, err := tx.Totals(netID)
		if err != nil {
			return err
		}
		if err := tx.SetClosureFinal(netID, man.LedgerSeq, sum, bal, ent); err != nil {
			return err
		}
		return s.emit(tx, netID, "", actor.RequestID, auditEntry{Type: domain.ActorSystem, Action: ActClosureExported, TargetType: "network",
			TargetID: netID, After: map[string]any{"sha256": sum, "ledger_seq": man.LedgerSeq, "entries_sum": ent, "balance_total": bal}})
	})
}

// ClosureCheck compares the stored final checksum with one recomputed now.
type ClosureCheck struct {
	Stored, Recomputed string
	Match              bool
	LedgerSeq          int64
	BalanceTotal       int64
	EntriesSum         int64
	// Erased: personal data was erased after the read-only period, so the
	// recomputed export legitimately differs.
	Erased bool
}

// VerifyClosureExport recomputes the final export checksum (GOV-025 e).
// Administrator only; it performs (and audits) a ledger export.
func (s *Service) VerifyClosureExport(ctx context.Context, actor domain.Actor, netID string) (*ClosureCheck, error) {
	var c *domain.Closure
	if err := s.view(ctx, func(tx store.Tx) error {
		p, err := s.open(tx, actor, netID, domain.ScopeRead)
		if err != nil {
			return err
		}
		if err := p.requireAdmin(); err != nil {
			return err
		}
		c = p.net.Closure
		return nil
	}); err != nil {
		return nil, err
	}
	if c == nil || !c.Done() || c.FinalSHA256 == "" {
		return nil, domain.InvalidState("the network has no completed closure with a final export")
	}
	man, err := s.Export(ctx, actor, netID, ExportLedger, FormatJSON, &MemSink{})
	if err != nil {
		return nil, err
	}
	re := ManifestChecksum(man)
	return &ClosureCheck{Stored: c.FinalSHA256, Recomputed: re, Match: re == c.FinalSHA256, LedgerSeq: c.FinalLedgerSeq,
		BalanceTotal: c.FinalBalanceTotal, EntriesSum: c.FinalEntriesSum, Erased: !c.ErasedAt.IsZero()}, nil
}

// EraseClosedNetworks applies the erasure rules (SEC-018) to networks whose
// read-only period (GOV-025 d, 90 days after completion) has ended. It reuses
// AnonymiseMember through its public API: members still listed as active,
// suspended or pending are first marked departed (a status change only: no
// settlement, no ledger entry; balances stay on the closed accounts) and then
// anonymised. The ledger, audit chain and balances are untouched. System
// actors only; idempotent; returns the number of networks finished. The first
// member that fails stops the network (it is retried on the next run).
func (s *Service) EraseClosedNetworks(ctx context.Context, actor domain.Actor) (int, error) {
	if !actor.System {
		return 0, domain.Forbidden("system actor required")
	}
	var cs []domain.Closure
	if err := s.view(ctx, func(tx store.Tx) (err error) {
		cs, err = tx.ListClosures(false)
		return err
	}); err != nil {
		return 0, err
	}
	finished := 0
	for _, c := range cs {
		if s.now().Before(c.ReadOnlyUntil()) {
			continue
		}
		if err := s.eraseNetwork(ctx, actor, c.NetworkID); err != nil {
			return finished, err
		}
		finished++
	}
	return finished, nil
}

func (s *Service) eraseNetwork(ctx context.Context, actor domain.Actor, netID string) error {
	after := ""
	for {
		var ms []*domain.Member
		if err := s.view(ctx, func(tx store.Tx) (err error) {
			ms, err = tx.ListMembers(netID, store.MemberFilter{AfterID: after, Limit: 200})
			return err
		}); err != nil {
			return err
		}
		for _, m := range ms {
			after = m.ID
			if err := s.eraseMember(ctx, actor, netID, m); err != nil {
				return err
			}
		}
		if len(ms) < 200 {
			break
		}
	}
	return s.update(ctx, func(tx store.Tx) error {
		now := s.now()
		if err := tx.SetClosureErased(netID, now); err != nil {
			return err
		}
		return s.emit(tx, netID, "", actor.RequestID, auditEntry{Type: domain.ActorSystem, Action: ActClosureErased, TargetType: "network",
			TargetID: netID, Reason: "read-only period ended"})
	})
}

func (s *Service) eraseMember(ctx context.Context, actor domain.Actor, netID string, m *domain.Member) error {
	var skip bool
	err := s.update(ctx, func(tx store.Tx) error {
		if e, err := tx.GetErasure(netID, m.ID); err != nil {
			return err
		} else if e != nil {
			skip = true
			return nil
		}
		if m.Status == domain.MemberDeparted {
			return nil
		}
		now := s.now()
		if err := tx.SetMemberStatus(netID, m.ID, domain.MemberDeparted, &now); err != nil {
			return err
		}
		return s.emit(tx, netID, "", actor.RequestID, auditEntry{Type: domain.ActorSystem, Action: domain.ActMemberDeparted, TargetType: "member",
			TargetID: m.ID, Reason: "network closed", Subjects: []string{m.ID}, Before: map[string]any{"status": m.Status},
			After: map[string]any{"status": domain.MemberDeparted}})
	})
	if err != nil || skip {
		return err
	}
	_, err = s.AnonymiseMember(ctx, actor, netID, m.ID, "network closed; read-only period ended")
	return err
}
