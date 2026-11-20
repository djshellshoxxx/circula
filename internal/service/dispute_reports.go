package service

import (
	"context"

	"github.com/djshellshoxxx/circula/internal/domain"
	store "github.com/djshellshoxxx/circula/internal/repository"
)

// DisputeReport is the minimum report context available to an independent
// dispute reviewer (SEC-025 / PER-003). Reporter identity and all free text
// remain in the administrator-only report record (FR-024). Do not embed Report:
// adding a report field must never silently enlarge this projection.
type DisputeReport struct {
	ID       string
	Category string
	Status   string
}

// requireIndependentDisputeReportReader preserves current dispute authority
// and the existing party/proposer/acceptor/reversal-participant conflicts.
// It is used in the same read snapshot as the dispute and report query.
func requireIndependentDisputeReportReader(tx store.Tx, p *principal, t *domain.Transaction) error {
	if p.actor.System {
		return nil
	}
	if err := p.requireDecider(); err != nil {
		return err
	}
	return independent(tx, p.net.ID, t, p.member.ID)
}

// ListDisputeReports reads only transaction reports on this dispute's exact
// transaction. It grants no report handling or full-queue access. Authorization
// is repeated for every page; the caller cannot supply a report target filter.
func (s *Service) ListDisputeReports(ctx context.Context, actor domain.Actor, netID, disputeID string, pg Page) ([]DisputeReport, error) {
	var out []DisputeReport
	err := s.view(ctx, func(tx store.Tx) error {
		p, err := s.open(tx, actor, netID, domain.ScopeRead)
		if err != nil {
			return err
		}
		if !p.actor.System {
			if err := p.requireDecider(); err != nil {
				return err
			}
		}
		if !domain.ValidRecordID(disputeID) {
			return domain.NotFound("dispute")
		}
		d, err := tx.GetDispute(netID, disputeID)
		if err != nil {
			return domain.NotFound("dispute")
		}
		t, err := tx.GetTransaction(netID, d.TxID)
		if err != nil {
			return err
		}
		if err := requireIndependentDisputeReportReader(tx, p, t); err != nil {
			return err
		}
		if err := validatePageIDs(pg.AfterID); err != nil {
			return err
		}
		reports, err := tx.ListReports(netID, store.ReportFilter{
			TargetType: ReportTransaction, TargetID: t.ID,
			AfterID: pg.AfterID, Limit: clampPage(pg), ExcludeSubjectMemberID: p.memberID(),
		})
		if err != nil {
			return err
		}
		for _, r := range reports {
			// Persisted categories are unconstrained in older schemas. Never
			// pass arbitrary stored text through this redacted projection.
			if !domain.ValidRecordID(r.ID) || !knownReportCategory(r.Category) || (r.Status != "open" && r.Status != "handled") {
				return domain.Errf(domain.CodeInternal, "report summary is unavailable")
			}
			out = append(out, DisputeReport{ID: r.ID, Category: r.Category, Status: r.Status})
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return out, nil
}
