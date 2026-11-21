package api_test

import (
	"reflect"
	"testing"

	"github.com/djshellshoxxx/circula/internal/service"
)

// API-063 / OPS-016: the created and retained check expose the same coverage;
// the existing administrator restriction also applies to the retained result.
func TestHistoricalLimitCoverageAPI(t *testing.T) {
	w := newWorld(t)
	w.env.Trade(w.net.ID, w.alice, w.bob, w.aliceM, w.bobM, 100)
	created := w.post(w.adminT, "/admin/consistency-checks", nil, w.key())
	wantStatus(t, created, 201)
	type response struct {
		ID               string
		OK               bool
		HistoricalLimits *service.HistoricalLimitCoverage `json:"historical_limit_coverage"`
	}
	check := decode[response](t, created)
	want := &service.HistoricalLimitCoverage{Status: "complete", CheckedDebits: 1}
	if !check.OK || !reflect.DeepEqual(check.HistoricalLimits, want) {
		t.Fatalf("created check lost historical coverage: %s", created.Body.String())
	}
	path := "/admin/consistency-checks/" + check.ID
	retained := w.get(w.adminT, path)
	wantStatus(t, retained, 200)
	if got := decode[response](t, retained); !reflect.DeepEqual(got, check) {
		t.Fatalf("retained check changed: %+v, want %+v", got, check)
	}
	wantProblem(t, w.get(w.aliceT, path), 403, "forbidden")
}
