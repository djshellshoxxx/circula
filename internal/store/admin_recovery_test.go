package store_test

import (
	"context"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/djshellshoxxx/circula/internal/domain"
	"github.com/djshellshoxxx/circula/internal/store"
)

// GOV-021: the schema keeps an administrator recovery request append-only,
// lets each step happen once, and refuses an applied state that lacks two
// distinct operator labels or the requester's own admin role assignment, even
// for a caller that bypasses the service.
func TestGOV021RecoverySchemaGuards(t *testing.T) {
	s := openStore(t)
	netID, _, _, _ := seed(t, s)
	ctx := context.Background()
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	db := s.DB()
	mustFail := func(what, q string, args ...any) {
		t.Helper()
		if _, err := db.Exec(q, args...); err == nil {
			t.Errorf("%s: accepted", what)
		}
	}
	req := &domain.AdminRecoveryRequest{ID: "r1", NetworkID: netID, RequesterID: "mp1", Status: domain.AdminRecoveryRequested,
		RequestedAt: now, ExpiresAt: now.Add(domain.AdminRecoveryTTL)}
	if err := s.Update(ctx, func(tx store.Tx) error { return tx.InsertAdminRecovery(req) }); err != nil {
		t.Fatal(err)
	}

	mustFail("insert in a later state", `INSERT INTO admin_recovery_requests(id,network_id,requester_member_id,status,requested_at,expires_at,
		first_operator,first_reason,first_approved_at) VALUES ('r2',?,'mp2','approved',1,2,'a','b',1)`, netID)
	mustFail("open longer than seven days", `INSERT INTO admin_recovery_requests(id,network_id,requester_member_id,requested_at,expires_at)
		VALUES ('r3',?,'mp2',0,604800001)`, netID)
	mustFail("unknown requester", `INSERT INTO admin_recovery_requests(id,network_id,requester_member_id,requested_at,expires_at)
		VALUES ('r4',?,'nobody',0,1)`, netID)
	mustFail("second open request of the member", `INSERT INTO admin_recovery_requests(id,network_id,requester_member_id,requested_at,expires_at)
		VALUES ('r5',?,'mp1',0,1)`, netID)
	mustFail("replace", `INSERT OR REPLACE INTO admin_recovery_requests(id,network_id,requester_member_id,requested_at,expires_at)
		VALUES ('r1',?,'mp2',0,1)`, netID)
	mustFail("delete", `DELETE FROM admin_recovery_requests WHERE id = 'r1'`)
	mustFail("change requester", `UPDATE admin_recovery_requests SET requester_member_id = 'mp2' WHERE id = 'r1'`)
	mustFail("extend expiry", `UPDATE admin_recovery_requests SET expires_at = expires_at + 1 WHERE id = 'r1'`)
	mustFail("skip the first approval", `UPDATE admin_recovery_requests SET status='applied', first_operator='a', first_reason='r',
		first_approved_at=1, closed_by='b', closed_reason='r', closed_at=1 WHERE id = 'r1'`)

	var ok bool
	err := s.Update(ctx, func(tx store.Tx) (err error) {
		ok, err = tx.ApproveAdminRecoveryFirst("r1", "Olga", "ticket 1", now)
		return err
	})
	if err != nil || !ok {
		t.Fatalf("first approval: %v %v", ok, err)
	}
	err = s.Update(ctx, func(tx store.Tx) (err error) {
		ok, err = tx.ApproveAdminRecoveryFirst("r1", "Piet", "again", now)
		return err
	})
	if err != nil || ok {
		t.Fatalf("first approval recorded twice: %v %v", ok, err)
	}
	mustFail("rewrite the first approval", `UPDATE admin_recovery_requests SET first_operator = 'Piet' WHERE id = 'r1'`)

	// Role rows for the guards on the applied state.
	if _, err := db.Exec(`INSERT INTO role_assignments(id,network_id,member_id,role,granted_at) VALUES ('ra-own',?,'mp1','admin',1),
		('ra-other',?,'mp2','admin',1), ('ra-rev',?,'mp1','reviewer',1)`, netID, netID, netID); err != nil {
		t.Fatal(err)
	}
	apply := `UPDATE admin_recovery_requests SET status='applied', closed_by=?, closed_reason='ticket 2', closed_at=2, role_assignment_id=? WHERE id = 'r1'`
	mustFail("same operator, other case", apply, " olga", "ra-own")
	mustFail("no appointment", apply, "Piet", nil)
	mustFail("someone else's role", apply, "Piet", "ra-other")
	mustFail("not the admin role", apply, "Piet", "ra-rev")
	err = s.Update(ctx, func(tx store.Tx) (err error) {
		ok, err = tx.CloseAdminRecovery("r1", domain.AdminRecoveryApproved, domain.AdminRecoveryApplied, "Piet", "ticket 2", now, "ra-own")
		return err
	})
	if err != nil || !ok {
		t.Fatalf("apply: %v %v", ok, err)
	}
	for _, q := range []string{
		`UPDATE admin_recovery_requests SET status='refused' WHERE id = 'r1'`,
		`UPDATE admin_recovery_requests SET closed_reason='edited' WHERE id = 'r1'`,
		`UPDATE admin_recovery_requests SET role_assignment_id='ra-other' WHERE id = 'r1'`,
	} {
		mustFail("change an applied request: "+q, q)
	}
	// The same appointment cannot close a second request.
	if _, err := db.Exec(`INSERT INTO admin_recovery_requests(id,network_id,requester_member_id,requested_at,expires_at) VALUES ('r6',?,'mp1',0,1)`, netID); err != nil {
		t.Fatalf("a new request after the applied one: %v", err)
	}
	if _, err := db.Exec(`UPDATE admin_recovery_requests SET status='approved', first_operator='Olga', first_reason='x', first_approved_at=1 WHERE id='r6'`); err != nil {
		t.Fatal(err)
	}
	mustFail("reuse an appointment", apply[:len(apply)-len("'r1'")]+"'r6'", "Piet", "ra-own")

	got, err := func() (out *domain.AdminRecoveryRequest, err error) {
		err = s.View(ctx, func(tx store.Tx) (err error) { out, err = tx.GetAdminRecovery("r1"); return })
		return
	}()
	if err != nil || got.Status != domain.AdminRecoveryApplied || got.FirstOperator != "Olga" || got.ClosedBy != "Piet" ||
		got.RoleAssignment != "ra-own" || got.FirstApprovedAt == nil || got.ClosedAt == nil {
		t.Fatalf("stored request = %+v, %v", got, err)
	}
}

// GOV-021: the compare-and-set transitions let exactly one of many concurrent
// writers win, and the listing and expiry queries see the right rows.
func TestGOV021RecoveryRepository(t *testing.T) {
	s := openStore(t)
	netID, _, _, _ := seed(t, s)
	ctx := context.Background()
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	for i, m := range []string{"mp1", "mp2"} {
		r := &domain.AdminRecoveryRequest{ID: []string{"ra", "rb"}[i], NetworkID: netID, RequesterID: m,
			Status: domain.AdminRecoveryRequested, RequestedAt: now, ExpiresAt: now.Add(time.Duration(i+1) * time.Hour)}
		if err := s.Update(ctx, func(tx store.Tx) error { return tx.InsertAdminRecovery(r) }); err != nil {
			t.Fatal(err)
		}
	}
	var wg sync.WaitGroup
	var mu sync.Mutex
	wins := 0
	for i := 0; i < 6; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			err := s.Update(ctx, func(tx store.Tx) error {
				ok, err := tx.CloseAdminRecovery("ra", domain.AdminRecoveryRequested, domain.AdminRecoveryRefused, "Olga", "no", now, "")
				if ok {
					mu.Lock()
					wins++
					mu.Unlock()
				}
				return err
			})
			if err != nil && !errors.Is(err, domain.ErrUnavailable) {
				t.Error(err)
			}
		}()
	}
	wg.Wait()
	if wins != 1 {
		t.Fatalf("%d concurrent closers won", wins)
	}
	err := s.View(ctx, func(tx store.Tx) error {
		all, err := tx.ListAdminRecovery(netID, false, "", 10)
		if err != nil || len(all) != 2 || all[0].ID != "rb" {
			t.Errorf("list = %v, %v", all, err)
		}
		open, err := tx.ListAdminRecovery("", true, "", 10)
		if err != nil || len(open) != 1 || open[0].ID != "rb" {
			t.Errorf("open list = %v, %v", open, err)
		}
		page, err := tx.ListAdminRecovery(netID, false, "rb", 10)
		if err != nil || len(page) != 1 || page[0].ID != "ra" {
			t.Errorf("page after rb = %v, %v", page, err)
		}
		exp, err := tx.ExpiredOpenAdminRecovery(now.Add(2*time.Hour), "", 10)
		if err != nil || len(exp) != 1 || exp[0].ID != "rb" {
			t.Errorf("expired = %v, %v", exp, err)
		}
		if exp, _ := tx.ExpiredOpenAdminRecovery(now.Add(2*time.Hour), "rb", 10); len(exp) != 0 {
			t.Errorf("expired after the keyset cursor: %v", exp)
		}
		if exp, _ := tx.ExpiredOpenAdminRecovery(now.Add(2*time.Hour-time.Millisecond), "", 10); len(exp) != 0 {
			t.Errorf("expired before the expiry instant: %v", exp)
		}
		if o, err := tx.GetOpenAdminRecovery(netID, "mp2"); err != nil || o == nil || o.ID != "rb" {
			t.Errorf("open request of mp2 = %v, %v", o, err)
		}
		if o, err := tx.GetOpenAdminRecovery(netID, "mp1"); err != nil || o != nil {
			t.Errorf("mp1 has only a refused request, got %v, %v", o, err)
		}
		if _, err := tx.GetAdminRecovery("missing"); !errors.Is(err, domain.ErrNotFound) || !strings.Contains(err.Error(), "recovery request") {
			t.Errorf("missing request: %v", err)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	// The open-request lookup is an index probe, not a scan of a page of rows.
	rows, err := s.DB().Query(`EXPLAIN QUERY PLAN SELECT id FROM admin_recovery_requests
		WHERE network_id = 'n1' AND requester_member_id = 'mp1' AND status IN ('requested','approved')`)
	if err != nil {
		t.Fatal(err)
	}
	plan := ""
	for rows.Next() {
		var id, parent, unused int
		var detail string
		if err := rows.Scan(&id, &parent, &unused, &detail); err != nil {
			t.Fatal(err)
		}
		plan += detail + ";"
	}
	_ = rows.Close()
	if !strings.Contains(plan, "admin_recovery_open_member") {
		t.Errorf("open-request lookup plan = %q, want the partial unique index", plan)
	}
}
