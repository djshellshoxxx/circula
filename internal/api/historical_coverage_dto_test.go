package api

import (
	"encoding/json"
	"reflect"
	"testing"

	"github.com/djshellshoxxx/circula/internal/service"
)

func TestHistoricalLimitCoverageDTOSeparatesCoverageFromResult(t *testing.T) {
	for _, tc := range []struct {
		name     string
		ok       bool
		coverage *service.HistoricalLimitCoverage
	}{
		{"unknown old result", true, nil},
		{"legacy partial passes", true, &service.HistoricalLimitCoverage{Status: "partial", CheckedDebits: 2, SystemDebits: 1, LegacyUnverifiedDebits: 4}},
		{"complete with finding", false, &service.HistoricalLimitCoverage{Status: "complete", CheckedDebits: 3}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			body, err := json.Marshal(checkDTO{OK: tc.ok, HistoricalLimits: tc.coverage})
			if err != nil {
				t.Fatal(err)
			}
			var fields map[string]json.RawMessage
			if err := json.Unmarshal(body, &fields); err != nil {
				t.Fatal(err)
			}
			coverage, present := fields["historical_limit_coverage"]
			if present != (tc.coverage != nil) {
				t.Fatalf("unknown coverage was relabelled: %s", body)
			}
			var ok bool
			if err := json.Unmarshal(fields["ok"], &ok); err != nil || ok != tc.ok {
				t.Fatalf("coverage changed invariant success: %s", body)
			}
			if present {
				var got service.HistoricalLimitCoverage
				if err := json.Unmarshal(coverage, &got); err != nil || !reflect.DeepEqual(&got, tc.coverage) {
					t.Fatalf("coverage fields changed: %s (%v)", body, err)
				}
			}
		})
	}
}
