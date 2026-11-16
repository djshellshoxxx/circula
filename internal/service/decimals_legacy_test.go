package service_test

import (
	"encoding/json"
	"slices"
	"testing"
	"time"

	"github.com/djshellshoxxx/circula/internal/domain"
	"github.com/djshellshoxxx/circula/internal/service"
	"github.com/djshellshoxxx/circula/internal/store"
	tu "github.com/djshellshoxxx/circula/internal/testutil"
)

// scaleNotes returns the stored-scale-change audit notes of a network.
func scaleNotes(t *testing.T, e *tu.Env, netID string) []*domain.AuditEvent {
	t.Helper()
	var out []*domain.AuditEvent
	if err := e.Store.View(e.Ctx, func(tx store.Tx) (err error) {
		out, err = tx.ListAudit(netID, store.AuditFilter{Action: domain.ActPolicyScaleChangeDetected, Limit: 500})
		return err
	}); err != nil {
		t.Fatal(err)
	}
	return out
}

// logChange returns the new value of key in policy version v of the rules log.
func logChange(t *testing.T, e *tu.Env, admin domain.Actor, netID string, v int, key string) (any, bool) {
	t.Helper()
	for _, le := range tu.Must(e.Svc.PolicyLog(e.Ctx, admin, netID)) {
		if le.Version == v {
			got, ok := le.Changes[key]
			return got, ok
		}
	}
	t.Fatalf("rules log has no version %d", v)
	return nil, false
}

// DOM-031, FR-005: a build before c34d947 allowed a scale change that took
// effect at once while nothing was posted, and the network then ran under the
// new scale. Such a row, seeded here directly through the store (both ways an
// earlier build wrote one), keeps its scale on every read: the limits stored
// in it, the amounts posted after it and the rules log keep the meaning they
// were written with. The job notes nothing for it (it is not a scheduled
// change).
func TestLegacyImmediateScaleChangeKeepsItsScale(t *testing.T) {
	seeds := map[string]func(g store.GovRepo, tx store.Tx, netID, by string, p domain.Policy, now time.Time) error{
		"AppendPolicy": func(_ store.GovRepo, tx store.Tx, netID, by string, p domain.Policy, now time.Time) error {
			return tx.AppendPolicy(netID, 2, p, by, "finer units", now)
		},
		"AppendPolicyNotice at once": func(g store.GovRepo, _ store.Tx, netID, by string, p domain.Policy, now time.Time) error {
			return g.AppendPolicyNotice(netID, 2, p, by, "finer units", now, now, 0, "")
		},
	}
	for name, seed := range seeds {
		t.Run(name, func(t *testing.T) {
			w := newWorld(t, nil) // decimals 2, limit 10000, nothing posted
			v2 := w.net.Policy
			v2.Decimals, v2.DefaultCreditLimit = 3, 20000 // 20.000 credits
			if err := w.Store.Update(w.Ctx, func(tx store.Tx) error {
				return seed(tx.(store.GovRepo), tx, w.net.ID, w.adminM.ID, v2, w.Clock.Now())
			}); err != nil {
				t.Fatal(err)
			}
			w.Trade(w.net.ID, w.a, w.b, w.aM, w.bM, 2500) // 2.500 credits at decimals 3
			n := w.netNow()
			if n.PolicyVersion != 2 || n.Policy != v2 {
				t.Fatalf("network reads v%d %+v, want v2 %+v", n.PolicyVersion, n.Policy, v2)
			}
			if got := domain.FormatAmount(w.balance(w.b).Balance, n.Policy.Decimals); got != "2.500" {
				t.Errorf("payee balance reads %s, want 2.500", got)
			}
			if got := domain.FormatAmount(n.Policy.DefaultCreditLimit, n.Policy.Decimals); got != "20.000" {
				t.Errorf("default limit reads %s, want 20.000", got)
			}
			if err := w.Store.View(w.Ctx, func(tx store.Tx) error {
				g := tx.(store.GovRepo)
				g.SetAsOf(w.Clock.Now())
				if s, err := tx.GetNetworkBySlug(w.net.Slug); err != nil || s.Policy.Decimals != 3 {
					t.Errorf("by slug: %+v %v", s, err)
				}
				rows, err := g.ListPolicyVersions(w.net.ID)
				if err != nil {
					return err
				}
				if len(rows) != 2 || rows[0].Policy.Decimals != 2 || rows[1].Policy.Decimals != 3 {
					t.Errorf("versions = %+v", rows)
				}
				return nil
			}); err != nil {
				t.Fatal(err)
			}
			if got, ok := logChange(t, w.Env, w.admin, w.net.ID, 2, "decimals"); !ok || got != 3 {
				t.Errorf("rules log v2 decimals = %v (%v), want 3", got, ok)
			}
			if k := tu.Must(w.Svc.RecordStoredScaleChanges(w.Ctx, sysJob)); k != 0 || len(scaleNotes(t, w.Env, w.net.ID)) != 0 {
				t.Errorf("noted %d for a change that took effect at once", k)
			}
			if s := storedScales(t, w.Env, w.net.ID); !slices.Equal(s, []int{2, 3}) {
				t.Errorf("stored scales %v", s)
			}
			if rep := tu.Must(w.Svc.VerifyLedger(w.Ctx, w.admin, w.net.ID)); !rep.OK {
				t.Error(rep.Findings)
			}
		})
	}
}

// DOM-031 residual (FR-005 implementation note of 2026-11-04): an earlier
// build could store a decimals change as a scheduled policy version. Whether
// to neutralise such a row is an owner or operator call (audit remediation
// 2026-11-04, open item 2), so the job only reports it: one system audit note
// per version, written as soon as the row is seen (before or after its date),
// with the scale in force before it and the stored one. No read changes: the
// version applies on its date exactly as before the job existed, the rules log
// shows its scale change and the stored rows are left as they are.
func TestLegacyScheduledScaleChangeIsReportedNotRewritten(t *testing.T) {
	e := tu.NewEnv(t)
	admin := e.Person("admin@example.org")
	n, adminM := e.Network(admin, "legacy", nil)
	created := n.Policy
	legacy := created
	legacy.Decimals = created.Decimals + 1
	legacy.ProposalExpiryDays = created.ProposalExpiryDays + 3 // a second key in the same version
	now := e.Clock.Now()
	eff := now.Add(48 * time.Hour)
	if err := e.Store.Update(e.Ctx, func(tx store.Tx) error {
		return tx.(store.GovRepo).AppendPolicyNotice(n.ID, 2, legacy, adminM.ID, "finer units", now, eff, 2, "")
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := e.Svc.RecordStoredScaleChanges(e.Ctx, admin); err == nil {
		t.Error("a non-system actor ran the job")
	}

	// Seen while pending: noted once, nothing else changes.
	if k := tu.Must(e.Svc.RecordStoredScaleChanges(e.Ctx, sysJob)); k != 1 {
		t.Fatalf("noted %d, want 1", k)
	}
	if k := tu.Must(e.Svc.RecordStoredScaleChanges(e.Ctx, sysJob)); k != 0 {
		t.Fatalf("noted again: %d", k)
	}
	if got := tu.Must(e.Svc.GetNetwork(e.Ctx, admin, n.ID)); got.PolicyVersion != 1 || got.Policy != created {
		t.Fatalf("before the effective time: v%d %+v", got.PolicyVersion, got.Policy)
	}
	notes := scaleNotes(t, e, n.ID)
	if len(notes) != 1 || notes[0].ActorType != domain.ActorSystem || notes[0].TargetType != "policy" || notes[0].TargetID != n.ID {
		t.Fatalf("notes = %+v", notes)
	}
	var before, after map[string]any
	if json.Unmarshal([]byte(notes[0].SummaryBefore), &before) != nil || json.Unmarshal([]byte(notes[0].SummaryAfter), &after) != nil ||
		before["decimals"] != float64(created.Decimals) || after["decimals"] != float64(legacy.Decimals) ||
		after["version"] != float64(2) || after["effective_at"] != float64(eff.UnixMilli()) {
		t.Fatalf("note summaries: before %s after %s", notes[0].SummaryBefore, notes[0].SummaryAfter)
	}

	// On its date the version applies as stored, as it did before this job.
	e.Clock.Advance(49 * time.Hour)
	if got := tu.Must(e.Svc.GetNetwork(e.Ctx, admin, n.ID)); got.PolicyVersion != 2 || got.Policy != legacy {
		t.Fatalf("in force: v%d %+v, want %+v", got.PolicyVersion, got.Policy, legacy)
	}
	if k := tu.Must(e.Svc.RecordStoredScaleChanges(e.Ctx, sysJob)); k != 0 || len(scaleNotes(t, e, n.ID)) != 1 {
		t.Fatalf("noted again once in force: %d", k)
	}
	if got, ok := logChange(t, e, admin, n.ID, 2, "decimals"); !ok || got != legacy.Decimals {
		t.Errorf("rules log v2 decimals = %v (%v), want %d", got, ok, legacy.Decimals)
	}
	if s := storedScales(t, e, n.ID); !slices.Equal(s, []int{created.Decimals, legacy.Decimals}) {
		t.Errorf("stored scales %v", s)
	}
	// A further scale change is still refused (FR-005).
	bad := legacy
	bad.Decimals = created.Decimals
	if _, err := e.Svc.UpdatePolicy(e.Ctx, admin, n.ID, service.PolicyChange{Policy: bad, Reason: "back", IdempotencyKey: e.Key()}); !decimalsRefused(err) {
		t.Fatalf("scale change after the legacy version: %v", err)
	}
	if rep := tu.Must(e.Svc.VerifyLedger(e.Ctx, admin, n.ID)); !rep.OK {
		t.Error(rep.Findings)
	}
}

// DOM-031: an administrator can still cancel a pending legacy scale change
// (the Cancel the scheduled change action restates the rules in force). A
// cancelled version never comes into force, so nothing is noted for it.
func TestCancelledLegacyScaleChangeIsNotNoted(t *testing.T) {
	e := tu.NewEnv(t)
	admin := e.Person("admin@example.org")
	n, adminM := e.Network(admin, "legacy", nil)
	legacy := n.Policy
	legacy.Decimals++
	now := e.Clock.Now()
	if err := e.Store.Update(e.Ctx, func(tx store.Tx) error {
		return tx.(store.GovRepo).AppendPolicyNotice(n.ID, 2, legacy, adminM.ID, "finer units", now, now.Add(48*time.Hour), 2, "")
	}); err != nil {
		t.Fatal(err)
	}
	if v := tu.Must(e.Svc.UpdatePolicy(e.Ctx, admin, n.ID, service.PolicyChange{Policy: n.Policy, Reason: "cancel", IdempotencyKey: e.Key()})); v.PolicyVersion != 3 {
		t.Fatalf("cancel: v%d", v.PolicyVersion)
	}
	e.Clock.Advance(49 * time.Hour)
	if got := tu.Must(e.Svc.GetNetwork(e.Ctx, admin, n.ID)); got.PolicyVersion != 3 || got.Policy != n.Policy {
		t.Fatalf("v%d %+v", got.PolicyVersion, got.Policy)
	}
	if k := tu.Must(e.Svc.RecordStoredScaleChanges(e.Ctx, sysJob)); k != 0 || len(scaleNotes(t, e, n.ID)) != 0 {
		t.Fatalf("noted a cancelled version: %d", k)
	}
}
