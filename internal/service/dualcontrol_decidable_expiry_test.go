package service_test

import (
	"testing"
	"time"

	"github.com/djshellshoxxx/circula/internal/domain"
	tu "github.com/djshellshoxxx/circula/internal/testutil"
)

// A3-01/SEC-011: the read-only decision hint must not offer expired requests.
// The approval endpoint remains the source of authority and rejects them.
func TestDecidableAdjustmentsExcludesExpiredOpenRequest(t *testing.T) {
	d := newDC(t)
	pa := d.pending(5000)
	ids := []string{pa.ID}

	before := tu.Must(d.Svc.DecidableAdjustments(d.Ctx, d.admin2, d.net.ID, ids))
	if !before[pa.ID] {
		t.Fatal("eligible pending request not offered before expiry")
	}

	d.Clock.Advance(7 * 24 * time.Hour) // exactly at expiry is already too late
	after := tu.Must(d.Svc.DecidableAdjustments(d.Ctx, d.admin2, d.net.ID, ids))
	if after[pa.ID] {
		t.Fatal("expired pending request incorrectly offered as decidable")
	}
	// This is a read-only hint; expiry is committed by the decision/job path.
	stored := tu.Must(d.Svc.GetPendingAdjustment(d.Ctx, d.admin, d.net.ID, pa.ID))
	if stored.Status != domain.PendingOpen {
		t.Fatalf("read-only hint changed stored request state to %q", stored.Status)
	}
	if got := d.balance(d.a).Balance; got != 0 {
		t.Fatalf("read-only hint posted an adjustment: balance=%d", got)
	}

	_, err := d.Svc.ApproveAdjustment(d.Ctx, d.admin2, d.net.ID, pa.ID)
	wantCode(t, err, domain.ErrInvalidState)
	expired := tu.Must(d.Svc.GetPendingAdjustment(d.Ctx, d.admin, d.net.ID, pa.ID))
	if expired.Status != domain.PendingExpired {
		t.Fatalf("authoritative approval did not expire request: %q", expired.Status)
	}
	if got := d.balance(d.a).Balance; got != 0 {
		t.Fatalf("expired request posted adjustment: balance=%d", got)
	}
}
