package service

import (
	"context"
	"errors"
	"fmt"

	"github.com/djshellshoxxx/circula/internal/domain"
	store "github.com/djshellshoxxx/circula/internal/repository"
)

// NotifDisputeOverdue tells reviewers a dispute needs assessment (GOV-014).
const NotifDisputeOverdue = "dispute.overdue"

// disputePolicy returns the network policy whose dispute timelines (DEC-012)
// govern d: the version bound atomically when d was opened. Dispute timelines change
// prospectively (GOV-001; 10 section 2), so a later change to the response
// period or the re-review window, shorter or longer, applies only to disputes
// opened once it is in force (AUD4-F03); the dispute window is fixed earlier,
// at completion (DOM-017). A later version cannot change the binding, even
// when timestamps are equal or the host clock steps backward (AUD4-F03).
func disputePolicy(tx store.Tx, d *domain.Dispute) (domain.Policy, error) {
	p, err := tx.GetPolicyVersion(d.NetworkID, d.PolicyVersion)
	if domain.CodeOf(err) == domain.CodeNotFound {
		// Never substitute the current or a timestamp-reconstructed policy.
		return domain.Policy{}, domain.Errf(domain.CodeInternal, "dispute %s has no bound policy version %d", d.ID, d.PolicyVersion)
	}
	return p, err
}

// disputeDeadlines are d's deadlines under the policy in force when it was
// opened. Every reader (decision gate, re-review request, dispute view and so
// the API and web page, reminders) uses them: what is shown is what is enforced.
func (s *Service) disputeDeadlines(tx store.Tx, d *domain.Dispute, responded bool) (domain.DisputeDeadlines, error) {
	p, err := disputePolicy(tx, d)
	if err != nil {
		return domain.DisputeDeadlines{}, err
	}
	return domain.DeadlinesOf(d, p, responded, s.now()), nil
}

// RemindOverdueDisputes is the body of a scheduled job (GOV-014). For every
// unresolved dispute it sends, once each, a reminder to the reviewers and
// administrators when (a) the response deadline passed without a response and
// (b) the decision target (14 days after the response deadline) passed. It
// decides nothing: a missed deadline never settles a case in anyone's favour,
// the case stays open for reviewer assessment. System actors only; returns the
// number of reminders sent.
func (s *Service) RemindOverdueDisputes(ctx context.Context, actor domain.Actor) (int, error) {
	if !actor.System {
		return 0, domain.Forbidden("system actor required")
	}
	var ids []string
	if err := s.view(ctx, func(tx store.Tx) (err error) {
		ids, err = tx.ListNetworkIDs()
		return err
	}); err != nil {
		return 0, err
	}
	total := 0
	for _, netID := range ids {
		n, err := s.remindNetwork(ctx, netID)
		if err != nil {
			return total, err
		}
		total += n
	}
	return total, nil
}

func (s *Service) remindNetwork(ctx context.Context, netID string) (int, error) {
	sent := 0
	err := s.update(ctx, func(tx store.Tx) error {
		if _, err := tx.GetNetwork(netID); errors.Is(err, domain.ErrNotFound) {
			return nil
		} else if err != nil {
			return err
		}
		after := ""
		for {
			ds, err := tx.ListDisputes(netID, store.DisputeFilter{AfterID: after, Limit: 200,
				Statuses: []domain.DisputeStatus{domain.DisputeOpen, domain.DisputeUnderReview}})
			if err != nil {
				return err
			}
			for _, d := range ds {
				after = d.ID
				resp, err := tx.GetDisputeResponse(netID, d.ID)
				if err != nil {
					return err
				}
				dl, err := s.disputeDeadlines(tx, d, resp != nil)
				if err != nil {
					return err
				}
				for _, k := range []struct {
					kind, title, body string
					due               bool
				}{
					{"response", "A dispute has no response", fmt.Sprintf("The other party did not respond by %s. The case stays open for your assessment.", dl.ResponseDeadline.Format("2 Jan 2006")), dl.ResponseOverdue},
					{"decision", "A dispute is past its decision target", fmt.Sprintf("The decision target of %s has passed.", dl.DecisionTarget.Format("2 Jan 2006")), dl.DecisionOverdue},
				} {
					if !k.due {
						continue
					}
					if ok, err := tx.RecordDeadlineNotice(netID, d.ID, k.kind, s.now()); err != nil {
						return err
					} else if !ok {
						continue
					}
					ids, err := deciderIDs(tx, netID)
					if err != nil {
						return err
					}
					t, err := tx.GetTransaction(netID, d.TxID)
					if err != nil {
						return err
					}
					var eligible []string
					for _, id := range ids {
						if !t.IsParty(id) {
							eligible = append(eligible, id)
						}
					}
					if err := s.notifyOthers(tx, netID, eligible, "", NotifDisputeOverdue, k.title, k.body, "/disputes/"+d.ID); err != nil {
						return err
					}
					sent++
				}
			}
			if len(ds) < 200 {
				return nil
			}
		}
	})
	return sent, err
}
