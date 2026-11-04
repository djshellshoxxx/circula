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

// scaleNotes returns the neutralisation audit events of a network.
func scaleNotes(t *testing.T, e *tu.Env, netID string) []*domain.AuditEvent {
	t.Helper()
	var out []*domain.AuditEvent
	if err := e.Store.View(e.Ctx, func(tx store.Tx) (err error) {
		out, err = tx.ListAudit(netID, store.AuditFilter{Action: domain.ActPolicyScaleNeutralised, Limit: 500})
		return err
	}); err != nil {
		t.Fatal(err)
	}
	return out
}

// DOM-031 residual (FR-005 implementation note of 2026-11-04): an earlier
// build could store a decimals change as a scheduled policy version. Such a
// row is seeded here directly through the store. When it comes into force the
// network keeps its creation-time scale, every other key of that version
// applies as scheduled, the rules log announces no scale change, the stored
// row is left as it is (append-only) and one audit note records the
// neutralisation. A later scale change stays refused.
func TestLegacyScheduledScaleChangeNeverTakesEffect(t *testing.T) {
	e := tu.NewEnv(t)
	admin := e.Person("admin@example.org")
	n, adminM := e.Network(admin, "legacy", nil)
	created := n.Policy
	legacy := created
	legacy.Decimals = created.Decimals + 1
	legacy.ProposalExpiryDays = created.ProposalExpiryDays + 3 // a second key in the same version
	now := e.Clock.Now()
	if err := e.Store.Update(e.Ctx, func(tx store.Tx) error {
		return tx.(store.GovRepo).AppendPolicyNotice(n.ID, 2, legacy, adminM.ID, "finer units", now, now.Add(48*time.Hour), 2, "")
	}); err != nil {
		t.Fatal(err)
	}

	// Not in force yet: nothing to neutralise or note.
	if got := tu.Must(e.Svc.GetNetwork(e.Ctx, admin, n.ID)); got.PolicyVersion != 1 || got.Policy != created {
		t.Fatalf("before the effective time: v%d %+v", got.PolicyVersion, got.Policy)
	}
	if k := tu.Must(e.Svc.RecordNeutralisedScaleChanges(e.Ctx, sysJob)); k != 0 || len(scaleNotes(t, e, n.ID)) != 0 {
		t.Fatalf("noted %d before the version came into force", k)
	}

	e.Clock.Advance(49 * time.Hour)
	want := legacy
	want.Decimals = created.Decimals
	got := tu.Must(e.Svc.GetNetwork(e.Ctx, admin, n.ID))
	if got.PolicyVersion != 2 || got.Policy != want {
		t.Fatalf("in force: v%d %+v, want %+v", got.PolicyVersion, got.Policy, want)
	}
	// Every read path of the store keeps the scale, not only the service's.
	if err := e.Store.View(e.Ctx, func(tx store.Tx) error {
		tx.(store.GovRepo).SetAsOf(e.Clock.Now())
		for _, get := range []func() (*domain.Network, error){
			func() (*domain.Network, error) { return tx.GetNetwork(n.ID) },
			func() (*domain.Network, error) { return tx.GetNetworkBySlug(n.Slug) },
		} {
			if s, err := get(); err != nil || s.Policy.Decimals != created.Decimals {
				t.Errorf("store read: %+v %v", s, err)
			}
		}
		rows, err := tx.(store.GovRepo).ListPolicyVersions(n.ID)
		if err != nil {
			return err
		}
		if len(rows) != 2 || rows[0].ScaleNeutralised || !rows[1].ScaleNeutralised ||
			rows[1].Policy.Decimals != created.Decimals || rows[1].StoredDecimals != legacy.Decimals {
			t.Errorf("versions = %+v", rows)
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	// The rules log announces the other change and no scale change.
	for _, le := range tu.Must(e.Svc.PolicyLog(e.Ctx, admin, n.ID)) {
		if _, ok := le.Changes["decimals"]; ok && le.Version > 1 {
			t.Errorf("log v%d announces a scale change: %+v", le.Version, le.Changes)
		}
		if le.Version == 2 && le.Changes["proposal_expiry_days"] != legacy.ProposalExpiryDays {
			t.Errorf("log v2 changes = %+v", le.Changes)
		}
	}
	// The stored history is not rewritten.
	if s := storedScales(t, e, n.ID); !slices.Equal(s, []int{created.Decimals, legacy.Decimals}) {
		t.Errorf("stored scales %v", s)
	}

	// One audit note, recorded once.
	if k := tu.Must(e.Svc.RecordNeutralisedScaleChanges(e.Ctx, sysJob)); k != 1 {
		t.Fatalf("noted %d", k)
	}
	if k := tu.Must(e.Svc.RecordNeutralisedScaleChanges(e.Ctx, sysJob)); k != 0 {
		t.Fatalf("noted again: %d", k)
	}
	notes := scaleNotes(t, e, n.ID)
	if len(notes) != 1 || notes[0].ActorType != domain.ActorSystem || notes[0].TargetType != "policy" || notes[0].TargetID != n.ID {
		t.Fatalf("notes = %+v", notes)
	}
	var before, after map[string]any
	if json.Unmarshal([]byte(notes[0].SummaryBefore), &before) != nil || json.Unmarshal([]byte(notes[0].SummaryAfter), &after) != nil ||
		before["decimals"] != float64(legacy.Decimals) || after["decimals"] != float64(created.Decimals) || after["version"] != float64(2) {
		t.Fatalf("note summaries: before %s after %s", notes[0].SummaryBefore, notes[0].SummaryAfter)
	}
	if _, err := e.Svc.RecordNeutralisedScaleChanges(e.Ctx, admin); err == nil {
		t.Error("a non-system actor ran the job")
	}

	// Later changes: the stored scale is still refused; a same-scale change applies.
	bad := got.Policy
	bad.Decimals = legacy.Decimals
	if _, err := e.Svc.UpdatePolicy(e.Ctx, admin, n.ID, service.PolicyChange{Policy: bad, Reason: "again", IdempotencyKey: e.Key()}); !decimalsRefused(err) {
		t.Fatalf("scale change after neutralisation: %v", err)
	}
	ok := got.Policy
	ok.MaxTxAmount++
	if v := tu.Must(e.Svc.UpdatePolicy(e.Ctx, admin, n.ID, service.PolicyChange{Policy: ok, Reason: "cap", IdempotencyKey: e.Key()})); v.PolicyVersion != 3 || v.Policy != ok {
		t.Fatalf("same-scale change: v%d %+v", v.PolicyVersion, v.Policy)
	}
	if rep := tu.Must(e.Svc.VerifyLedger(e.Ctx, admin, n.ID)); !rep.OK {
		t.Error(rep.Findings)
	}
}

// DOM-031: a scheduled scale change that was withdrawn before it took effect
// never comes into force, so nothing is noted for it.
func TestWithdrawnLegacyScaleChangeIsNotNoted(t *testing.T) {
	e := tu.NewEnv(t)
	admin := e.Person("admin@example.org")
	n, adminM := e.Network(admin, "legacy", nil)
	legacy := n.Policy
	legacy.Decimals++
	now := e.Clock.Now()
	if err := e.Store.Update(e.Ctx, func(tx store.Tx) error {
		g := tx.(store.GovRepo)
		if err := g.AppendPolicyNotice(n.ID, 2, legacy, adminM.ID, "finer units", now, now.Add(48*time.Hour), 2, ""); err != nil {
			return err
		}
		// cancelled at once by restating the effective policy (what UpdatePolicy does)
		return g.AppendPolicyNotice(n.ID, 3, n.Policy, adminM.ID, "cancel", now, now, 0, "")
	}); err != nil {
		t.Fatal(err)
	}
	e.Clock.Advance(49 * time.Hour)
	if got := tu.Must(e.Svc.GetNetwork(e.Ctx, admin, n.ID)); got.PolicyVersion != 3 || got.Policy != n.Policy {
		t.Fatalf("v%d %+v", got.PolicyVersion, got.Policy)
	}
	if k := tu.Must(e.Svc.RecordNeutralisedScaleChanges(e.Ctx, sysJob)); k != 0 || len(scaleNotes(t, e, n.ID)) != 0 {
		t.Fatalf("noted a withdrawn version: %d", k)
	}
}
