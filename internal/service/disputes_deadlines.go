package service

import (
	"context"
	"errors"
	"fmt"

	"github.com/djshellshoxxx/circula/internal/domain"
	"github.com/djshellshoxxx/circula/internal/store"
)

// NotifDisputeOverdue tells reviewers a dispute needs assessment (GOV-014).
const NotifDisputeOverdue = "dispute.overdue"

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
		net, err := tx.GetNetwork(netID)
		if errors.Is(err, domain.ErrNotFound) {
			return nil
		}
		if err != nil {
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
				dl := domain.DeadlinesOf(d, net.Policy.ResponsePeriodDays, resp != nil, s.now())
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
