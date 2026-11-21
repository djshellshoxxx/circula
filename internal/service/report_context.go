package service

import (
	"context"
	"time"

	"github.com/djshellshoxxx/circula/internal/domain"
	store "github.com/djshellshoxxx/circula/internal/repository"
)

// ReportDisputeContext describes the report targets a party can choose from
// an existing dispute. It contains no reports, report counts or reporter data.
// ListingID is empty when the transaction has no listing owned by the other
// party. The labels are already within the caller's authorized case context.
type ReportDisputeContext struct {
	DisputeID, MemberID, MemberName, ListingID, ListingTitle string
	OpenedAt                                                 time.Time
}

// reportDisputeContext is also used inside the create transaction, so the
// submitted case, party and target relation are checked in the write snapshot.
// Case status and age impose no additional restriction on reporting.
func reportDisputeContext(tx store.Tx, p *principal, disputeID string) (*ReportDisputeContext, error) {
	if err := p.requireActive(); err != nil {
		return nil, err
	}
	if !domain.ValidRecordID(disputeID) {
		return nil, domain.NotFound("dispute")
	}
	d, err := tx.GetDispute(p.net.ID, disputeID)
	if err != nil {
		if domain.CodeOf(err) == domain.CodeNotFound {
			return nil, domain.NotFound("dispute")
		}
		return nil, err
	}
	t, err := tx.GetTransaction(p.net.ID, d.TxID)
	if err != nil {
		return nil, err
	}
	if !t.IsParty(p.member.ID) {
		return nil, domain.NotFound("dispute")
	}
	subjectID, ok := transactionReportSubject(t, p.member.ID)
	if !ok {
		return nil, domain.Invalid("dispute_id", "case context requires two member parties")
	}
	m, err := tx.GetMember(p.net.ID, subjectID)
	if err != nil {
		return nil, err
	}
	if m.Status == domain.MemberPending {
		return nil, domain.NotFound("member")
	}
	out := &ReportDisputeContext{DisputeID: d.ID, MemberID: m.ID, MemberName: m.DisplayName, OpenedAt: d.OpenedAt}
	if t.ListingID != "" {
		l, err := tx.GetListing(p.net.ID, t.ListingID)
		if err != nil {
			return nil, err
		}
		if l.MemberID == subjectID {
			out.ListingID = l.ID
			// The relation remains usable for a historical case without
			// exposing a title hidden by ordinary listing visibility.
			if l.Status == domain.ListingActive && m.Status == domain.MemberActive {
				out.ListingTitle = l.Title
			}
		}
	}
	return out, nil
}

// GetReportDisputeContext supplies an ordinary party's case-linked report
// form. It grants no access to an administrator queue or reviewer projection.
func (s *Service) GetReportDisputeContext(ctx context.Context, actor domain.Actor, netID, disputeID string) (*ReportDisputeContext, error) {
	var out *ReportDisputeContext
	err := s.view(ctx, func(tx store.Tx) error {
		p, err := s.open(tx, actor, netID, domain.ScopeRead)
		if err != nil {
			return err
		}
		out, err = reportDisputeContext(tx, p, disputeID)
		return err
	})
	return out, err
}
