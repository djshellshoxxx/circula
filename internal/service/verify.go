package service

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/djshellshoxxx/circula/internal/domain"
	"github.com/djshellshoxxx/circula/internal/store"
)

// LedgerReport is the result of VerifyLedger (DAT-020, ARC-015).
type LedgerReport struct {
	NetworkID   string
	OK          bool
	Findings    []string
	LedgerSeq   int64
	AuditEvents int64
	Started     time.Time
	Finished    time.Time
	// RunID is the consistency_runs id; set only by RecordConsistencyRun /
	// VerifyAndRecordLedger.
	RunID string
}

// VerifyLedger runs the consistency check for one network. It is READ-ONLY
// (OPS-016, FR-052, SVC-01): it runs in one Store.View snapshot, takes no write
// lock, writes no consistency_runs row and no audit event, and leaves the
// database byte-for-byte as it found it. Use VerifyAndRecordLedger or
// RecordConsistencyRun when an auditable run record is wanted (an explicit
// administrator action, not startup, backup or scheduled checks).
//
// It verifies, per network: cached balance == sum of entries for every account
// (LED-3); every posted transaction has exactly two entries summing to zero and
// unposted ones have none (LED-1, DAT-007); the sum of balances and of entries
// are both zero including the system account (LED-1, DAT-009); reversed totals
// equal the sum of their reversals (DOM-020); and the audit hash chain is
// contiguous, unmodified and ends at the recorded head so tail truncation is
// detected (DAT-030, SA-09). Administrators or System only.
func (s *Service) VerifyLedger(ctx context.Context, actor domain.Actor, netID string) (*LedgerReport, error) {
	rep := &LedgerReport{NetworkID: netID}
	err := s.view(ctx, func(tx store.Tx) error {
		p, err := s.open(tx, actor, netID, domain.ScopeRead)
		if err != nil {
			return err
		}
		if err := p.requireAdmin(); err != nil {
			return err
		}
		rep.Started = s.now()
		add := func(f string, a ...any) { rep.Findings = append(rep.Findings, fmt.Sprintf(f, a...)) }
		mm, err := tx.BalanceMismatches(netID)
		if err != nil {
			return err
		}
		for _, m := range mm {
			add("account %s cached balance %d != sum of entries %d", m.AccountID, m.Cached, m.Summed)
		}
		probs, err := tx.TxProblems(netID)
		if err != nil {
			return err
		}
		rep.Findings = append(rep.Findings, probs...)
		bal, ent, err := tx.Totals(netID)
		if err != nil {
			return err
		}
		if bal != 0 || ent != 0 {
			add("network totals not zero: balances %d, entries %d", bal, ent)
		}
		rev, err := tx.ReversalMismatches(netID)
		if err != nil {
			return err
		}
		rep.Findings = append(rep.Findings, rev...)
		if err := s.verifyAuditChain(tx, netID, rep); err != nil {
			return err
		}
		if rep.LedgerSeq, err = tx.LedgerSeq(netID); err != nil {
			return err
		}
		rep.OK = len(rep.Findings) == 0
		rep.Finished = s.now()
		return nil
	})
	if err != nil {
		return nil, err
	}
	return rep, nil
}

// RecordConsistencyRun appends rep (which must come from VerifyLedger) to the
// append-only consistency_runs table and writes the ledger.verified audit
// event, in one short write unit that does no scanning (SVC-01). Administrators
// or System only; write scope. It sets rep.RunID.
func (s *Service) RecordConsistencyRun(ctx context.Context, actor domain.Actor, rep *LedgerReport) error {
	return s.recordConsistencyRun(ctx, actor, rep, false)
}

// RecordScheduledConsistencyRun is RecordConsistencyRun for the hourly job: the
// run is always recorded, but the ledger.verified audit event is written only
// when the run fails or the state changes from the previous run.
func (s *Service) RecordScheduledConsistencyRun(ctx context.Context, actor domain.Actor, rep *LedgerReport) error {
	return s.recordConsistencyRun(ctx, actor, rep, true)
}

func (s *Service) recordConsistencyRun(ctx context.Context, actor domain.Actor, rep *LedgerReport, quiet bool) error {
	if rep == nil || rep.NetworkID == "" {
		return domain.Invalid("report", "a ledger report is required")
	}
	return s.update(ctx, func(tx store.Tx) error {
		p, err := s.openAny(tx, actor, rep.NetworkID, domain.ScopeWrite)
		if err != nil {
			return err
		}
		if err := p.requireAdmin(); err != nil {
			return err
		}
		prev, err := tx.LastConsistencyRun(rep.NetworkID)
		if err != nil {
			return err
		}
		id := s.id()
		if err := tx.InsertConsistencyRun(store.ConsistencyRun{ID: id, NetworkID: rep.NetworkID, Started: rep.Started,
			Finished: rep.Finished, OK: rep.OK, Findings: strings.Join(rep.Findings, "\n")}); err != nil {
			return err
		}
		rep.RunID = id
		// Scheduled runs are recorded in consistency_runs but audited only on a
		// failure or a change of state, so an hourly job cannot flood the log (ARC-015).
		if quiet && rep.OK && (prev == nil || prev.OK) {
			return nil
		}
		return s.audit(tx, p, auditEntry{Type: p.actorType(domain.ActorAdmin), Action: domain.ActVerify, TargetType: "network", TargetID: rep.NetworkID,
			After: map[string]any{"status": map[bool]string{true: "ok", false: "findings"}[rep.OK]}})
	})
}

// VerifyAndRecordLedger verifies read-only, then records the run in a separate
// short write unit. Only explicit administrator-initiated checks should use it.
func (s *Service) VerifyAndRecordLedger(ctx context.Context, actor domain.Actor, netID string) (*LedgerReport, error) {
	rep, err := s.VerifyLedger(ctx, actor, netID)
	if err != nil {
		return nil, err
	}
	if err := s.RecordConsistencyRun(ctx, actor, rep); err != nil {
		return nil, err
	}
	return rep, nil
}

func (s *Service) verifyAuditChain(tx store.Tx, netID string, rep *LedgerReport) error {
	var after int64
	prev := ""
	for {
		evs, err := tx.ListAudit(netID, store.AuditFilter{AfterSeq: after, Limit: 500})
		if err != nil {
			return err
		}
		if len(evs) == 0 {
			break
		}
		for _, e := range evs {
			rep.AuditEvents++
			switch {
			case e.Seq != after+1:
				rep.Findings = append(rep.Findings, fmt.Sprintf("audit sequence gap before %d", e.Seq))
			case e.PrevHash != prev:
				rep.Findings = append(rep.Findings, fmt.Sprintf("audit event %d breaks the hash chain", e.Seq))
			case AuditHash(e) != e.Hash:
				rep.Findings = append(rep.Findings, fmt.Sprintf("audit event %d content does not match its hash", e.Seq))
			}
			after, prev = e.Seq, e.Hash
		}
	}
	// The chain must end exactly at the head recorded in the same unit as the
	// last append; this catches deletion of the newest events (SA-09).
	hs, hh, found, err := tx.AuditHead(netID)
	if err != nil {
		return err
	}
	switch {
	case !found && after > 0:
		rep.Findings = append(rep.Findings, fmt.Sprintf("audit head record missing (chain ends at %d)", after))
	case found && (hs != after || hh != prev):
		rep.Findings = append(rep.Findings, fmt.Sprintf(
			"audit chain ends at event %d but the recorded head is %d: events were removed or the head was altered", after, hs))
	}
	return nil
}
