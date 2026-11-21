package api_test

import (
	"strings"
	"testing"
	"time"

	"github.com/djshellshoxxx/circula/internal/domain"
	tu "github.com/djshellshoxxx/circula/internal/testutil"
)

// GOV-025 e: the API maps checkpoint version deliberately and distinguishes
// an unsupported/legacy representation from a successfully computed mismatch.
func TestClosureCheckpointVersionContract(t *testing.T) {
	w := newWorld(t)
	wantStatus(t, w.post(w.adminT, "/admin/network/close", m{"reason": "finished", "notice_days": 30}, ""), 200)
	w.env.Clock.Advance(31 * 24 * time.Hour)
	tu.Must(w.env.Svc.FinaliseClosures(w.env.Ctx, domain.SystemActor("closure-test")))
	w.env.Clock.Advance(time.Hour)
	status := w.get(w.adminT, "/admin/network/closure")
	wantStatus(t, status, 200)
	data := decode[struct {
		Closure struct {
			FinalExport struct {
				Version int
				SHA256  string
			} `json:"final_export"`
		}
	}](t, status)
	if data.Closure.FinalExport.Version != 1 || len(data.Closure.FinalExport.SHA256) != 64 {
		t.Fatalf("closure status missing versioned checksum: %s", status.Body.String())
	}
	verify := w.post(w.adminT, "/admin/network/closure/verify", nil, "")
	wantStatus(t, verify, 200)
	check := decode[map[string]any](t, verify)
	if check["checkpoint_version"] != float64(1) || check["match"] != true || check["stored_sha256"] != data.Closure.FinalExport.SHA256 {
		t.Fatalf("versioned verification contract: %s", verify.Body.String())
	}
	wantProblem(t, w.post(w.aliceT, "/admin/network/closure/verify", nil, ""), 403, "forbidden")
	for _, version := range []int{0, 99} {
		if _, err := w.env.Store.DB().Exec(`UPDATE network_closures SET final_export_version=? WHERE network_id=?`, version, w.net.ID); err != nil {
			t.Fatal(err)
		}
		p := wantProblem(t, w.post(w.adminT, "/admin/network/closure/verify", nil, ""), 409, "invalid-transaction-state")
		if !strings.Contains(p.Detail, map[int]string{0: "legacy closure checkpoint", 99: "version 99 is not supported"}[version]) {
			t.Fatalf("unverifiable checkpoint has no compatibility explanation: %+v", p)
		}
		status = w.get(w.adminT, "/admin/network/closure")
		wantStatus(t, status, 200)
		legacy := decode[map[string]any](t, status)["closure"].(map[string]any)["final_export"].(map[string]any)
		if legacy["version"] != float64(version) || legacy["sha256"] != data.Closure.FinalExport.SHA256 {
			t.Fatalf("legacy/unsupported checkpoint changed or lost its version: %s", status.Body.String())
		}
	}
}
