package service

import (
	"context"
	"fmt"
	"strings"

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
}

// VerifyLedger runs the consistency check for one network and records the run
// (append-only). It verifies, per network: cached balance == sum of entries for
// every account (LED-3); every posted transaction has exactly two entries summing
// to zero and unposted ones have none (LED-1, DAT-007); the sum of balances and
// of entries are both zero including the system account (LED-1, DAT-009);
// reversed totals equal the sum of their reversals (DOM-020); and the audit hash
// chain is contiguous and unmodified (DAT-030). Administrators or System only.
func (s *Service) VerifyLedger(ctx context.Context, actor domain.Actor, netID string) (*LedgerReport, error) {
	rep := &LedgerReport{NetworkID: netID}
	err := s.update(ctx, func(tx store.Tx) error {
		p, err := s.open(tx, actor, netID, domain.ScopeRead)
		if err != nil {
			return err
		}
		if err := p.requireAdmin(); err != nil {
			return err
		}
		started := s.now()
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
		if err := tx.InsertConsistencyRun(store.ConsistencyRun{ID: s.id(), NetworkID: netID, Started: started,
			Finished: s.now(), OK: rep.OK, Findings: strings.Join(rep.Findings, "\n")}); err != nil {
			return err
		}
		return s.audit(tx, p, auditEntry{Type: p.actorType(domain.ActorAdmin), Action: domain.ActVerify, TargetType: "network", TargetID: netID,
			After: map[string]any{"status": map[bool]string{true: "ok", false: "findings"}[rep.OK]}})
	})
	if err != nil {
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
			return nil
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
}
