package service_test

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/djshellshoxxx/circula/internal/domain"
	"github.com/djshellshoxxx/circula/internal/service"
	"github.com/djshellshoxxx/circula/internal/store"
	tu "github.com/djshellshoxxx/circula/internal/testutil"
)

// Arm only after ordinary setup. The fixture changes one repository read,
// preserving the real snapshot, authority checks and policy-clock extension.
type disputeReportReadFaultStore struct {
	store.Store
	networkID, disputeID string
	failure              error
	lookups              int
}

func (s *disputeReportReadFaultStore) View(ctx context.Context, fn func(store.Tx) error) error {
	if s.failure == nil {
		return s.Store.View(ctx, fn)
	}
	return s.Store.View(ctx, func(tx store.Tx) error {
		return fn(&disputeReportReadFaultTx{Tx: tx, GovRepo: tx.(store.GovRepo), control: s})
	})
}

type disputeReportReadFaultTx struct {
	store.Tx
	store.GovRepo
	control *disputeReportReadFaultStore
}

func (tx *disputeReportReadFaultTx) GetDispute(netID, id string) (*domain.Dispute, error) {
	if netID == tx.control.networkID && id == tx.control.disputeID {
		tx.control.lookups++
		return nil, tx.control.failure
	}
	return tx.Tx.GetDispute(netID, id)
}

// A retryable or failed repository read is not an absent dispute. The new
// reviewer-report read must preserve its typed error and original cause.
func TestDisputeReportsPreserveDisputeReadErrors(t *testing.T) {
	var faults *disputeReportReadFaultStore
	e := tu.NewEnvWith(t, func(st store.Store) store.Store {
		faults = &disputeReportReadFaultStore{Store: st}
		return faults
	})
	admin := e.Person("admin@example.org")
	net, _ := e.Network(admin, "report-read", nil)
	ana, anaM := e.Member(admin, net, "ana@example.org")
	ben, benM := e.Member(admin, net, "ben@example.org")
	trade := e.Trade(net.ID, ana, ben, anaM, benM, 100)
	dispute := tu.Must(e.Svc.OpenDispute(e.Ctx, ana, net.ID, service.OpenDisputeInput{
		TxID: trade.ID, Category: "other", Statement: "Ordinary case", IdempotencyKey: e.Key(),
	}))
	report := tu.Must(e.Svc.CreateReport(e.Ctx, ana, net.ID, service.ReportInput{
		TargetType: service.ReportTransaction, TargetID: trade.ID, Category: "other", IdempotencyKey: e.Key(),
	}))
	checkSuccess := func() {
		t.Helper()
		rows := tu.Must(e.Svc.ListDisputeReports(e.Ctx, admin, net.ID, dispute.ID, service.Page{}))
		if len(rows) != 1 || rows[0].ID != report.ID {
			t.Fatal("ordinary independent report read did not return its summary")
		}
	}
	checkSuccess()
	faults.networkID, faults.disputeID = net.ID, dispute.ID
	for _, tc := range []struct {
		name    string
		failure error
	}{
		{"unavailable", domain.ErrUnavailable},
		{"wrapped unavailable", fmt.Errorf("repository fixture: %w", domain.ErrUnavailable)},
		{"read failure", errors.New("repository fixture read failed")},
		{"cancelled", context.Canceled},
		{"deadline", context.DeadlineExceeded},
		{"not found", domain.NotFound("stored fixture label")},
		{"wrapped not found", fmt.Errorf("repository fixture: %w", domain.NotFound("stored fixture label"))},
	} {
		t.Run(tc.name, func(t *testing.T) {
			faults.failure, faults.lookups = tc.failure, 0
			rows, err := e.Svc.ListDisputeReports(e.Ctx, admin, net.ID, dispute.ID, service.Page{})
			if rows != nil || faults.lookups != 1 {
				t.Fatal("failed lookup returned summaries or did not reach the selected repository read")
			}
			if domain.CodeOf(tc.failure) == domain.CodeNotFound {
				if !errors.Is(err, domain.ErrNotFound) || err.Error() != domain.NotFound("dispute").Error() {
					t.Fatalf("missing dispute was not normalized: %v", err)
				}
			} else if !errors.Is(err, tc.failure) || domain.CodeOf(err) != domain.CodeOf(tc.failure) {
				t.Fatalf("dispute read error lost its code or cause: got %v, want %v", err, tc.failure)
			}
		})
	}
	faults.failure = nil
	checkSuccess()
}
