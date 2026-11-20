package service_test

import (
	"testing"

	"github.com/djshellshoxxx/circula/internal/domain"
	"github.com/djshellshoxxx/circula/internal/service"
	tu "github.com/djshellshoxxx/circula/internal/testutil"
)

// A1-06: departure must not leave proposals after the first 500.
func TestDepartureCancelsMoreThanFiveHundredProposals(t *testing.T) {
	// The default per-member open-proposal cap (200) would stop setup
	// before 501 proposals exist. Raise it here; the cap itself is covered
	// by its own tests and the production default is unchanged.
	w := newWorldCfg(t, nil, func(c *service.Config) { c.MaxOpenProposalsPerMember = 1000 })
	const count = 501
	ids := make([]string, 0, count)
	for i := 0; i < count; i++ {
		p := tu.Must(w.propose(w.a, w.bM, service.ProposerPays, 1))
		ids = append(ids, p.ID)
	}
	tu.Must(w.Svc.DepartMember(w.Ctx, w.a, w.net.ID, service.DepartInput{}))
	for _, id := range ids {
		if got := w.txStatus(id).Status; got != domain.TxCancelled {
			t.Fatalf("proposal %s remained %s after departure", id, got)
		}
	}
}
