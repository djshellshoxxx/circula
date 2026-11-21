// Package integrity evaluates ledger and audit invariants through read-only
// repository interfaces. Callers supply the snapshot and handle authorization,
// migration transactions, timestamps, metrics and result persistence.
package integrity

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sort"
	"strings"

	"github.com/djshellshoxxx/circula/internal/domain"
	"github.com/djshellshoxxx/circula/internal/repository"
)

// AuditReader is the read-only audit surface needed for verification.
type AuditReader interface {
	ListAudit(string, repository.AuditFilter) ([]*domain.AuditEvent, error)
	AuditHead(string) (int64, string, bool, error)
}

// LedgerReader is the read-only snapshot surface needed for verification.
type LedgerReader interface {
	repository.VerifyRepo
	AuditReader
	LedgerSeq(string) (int64, error)
}

// Options supports schemas before migrations 0002 and 0024 introduced audit
// heads and credit witnesses. Ordinary service verification uses the zero value.
type Options struct {
	LegacyAuditWithoutHead           bool
	LegacyCreditLimitsWithoutWitness bool
}

// Report contains the invariant findings from the supplied snapshot.
// Time, authorization and recording belong to the caller.
type Report struct {
	Findings         []string
	LedgerSeq        int64
	AuditEvents      int64
	HistoricalLimits *repository.HistoricalLimitCoverage
}

// CheckLedger evaluates the same ledger and audit invariants for service
// verification and migration admission. It performs no writes.
func CheckLedger(tx LedgerReader, netID string, options Options) (*Report, error) {
	rep := &Report{}
	add := func(f string, a ...any) { rep.Findings = append(rep.Findings, fmt.Sprintf(f, a...)) }
	mm, err := tx.BalanceMismatches(netID)
	if err != nil {
		return nil, err
	}
	for _, m := range mm {
		add("account %s cached balance %d != sum of entries %d", m.AccountID, m.Cached, m.Summed)
	}
	probs, err := tx.TxProblems(netID)
	if err != nil {
		return nil, err
	}
	rep.Findings = append(rep.Findings, probs...)
	bal, ent, err := tx.Totals(netID)
	if err != nil {
		return nil, err
	}
	if bal != 0 || ent != 0 {
		add("network totals not zero: balances %d, entries %d", bal, ent)
	}
	rev, err := tx.ReversalMismatches(netID)
	if err != nil {
		return nil, err
	}
	rep.Findings = append(rep.Findings, rev...)
	if !options.LegacyCreditLimitsWithoutWitness {
		if err := checkHistoricalLimits(tx, netID, rep); err != nil {
			return nil, err
		}
	}
	if err := checkAudit(tx, netID, rep, options); err != nil {
		return nil, err
	}
	if rep.LedgerSeq, err = tx.LedgerSeq(netID); err != nil {
		return nil, err
	}
	return rep, nil
}

// CheckAudit verifies one network or platform audit scope without writing it.
func CheckAudit(tx AuditReader, netID string, options Options) (*Report, error) {
	rep := &Report{}
	if err := checkAudit(tx, netID, rep, options); err != nil {
		return nil, err
	}
	return rep, nil
}

func checkAudit(tx AuditReader, netID string, rep *Report, options Options) error {
	var after int64
	prev := ""
	for {
		evs, err := tx.ListAudit(netID, repository.AuditFilter{AfterSeq: after, Limit: 500})
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
	if options.LegacyAuditWithoutHead {
		return nil
	}
	hs, hh, found, err := tx.AuditHead(netID)
	if err != nil {
		return err
	}
	switch {
	case !found && after > 0:
		rep.Findings = append(rep.Findings, fmt.Sprintf("audit head record missing (chain ends at %d)", after))
	case found && hs <= 0:
		rep.Findings = append(rep.Findings, fmt.Sprintf("recorded audit head has invalid sequence %d", hs))
	case found && (hs != after || hh != prev):
		rep.Findings = append(rep.Findings, fmt.Sprintf(
			"audit chain ends at event %d but the recorded head is %d: events were removed or the head was altered", after, hs))
	}
	return nil
}

// AuditHash computes the chain hash of an event over its content and the
// previous hash. It detects silent edits; it is not tamper-proofing (DAT-030).
func AuditHash(e *domain.AuditEvent) string {
	subj := append([]string(nil), e.SubjectIDs...)
	sort.Strings(subj)
	h := sha256.New()
	_, _ = fmt.Fprintf(h, "%s\x1f%s\x1f%s\x1f%d\x1f%d\x1f%s\x1f%s\x1f%s\x1f%s\x1f%s\x1f%s\x1f%s\x1f%s\x1f%s\x1f%s\x1f%s",
		e.PrevHash, e.ID, e.NetworkID, e.Seq, e.OccurredAt.UnixMilli(), e.ActorType, e.ActorID, e.Action,
		e.TargetType, e.TargetID, e.RequestID, e.IdemRef, e.SummaryBefore, e.SummaryAfter, e.Reason,
		strings.Join(subj, ","))
	return hex.EncodeToString(h.Sum(nil))
}
