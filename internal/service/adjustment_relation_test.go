package service_test

import (
	"bytes"
	"context"
	"encoding/json"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/djshellshoxxx/circula/internal/domain"
	"github.com/djshellshoxxx/circula/internal/service"
	"github.com/djshellshoxxx/circula/internal/store"
	tu "github.com/djshellshoxxx/circula/internal/testutil"
)

func TestAdjustmentRelationPostingPrivacyAndIdempotency(t *testing.T) {
	w := newWorld(t, nil)
	original := w.Trade(w.net.ID, w.a, w.b, w.aM, w.bM, 500)
	beforeOriginal := tu.Must(w.Svc.GetTransaction(w.Ctx, w.a, w.net.ID, original.ID))
	in := service.AdjustInput{MemberID: w.aM.ID, Amount: 25, Direction: service.AdjustCredit, Reason: "rounding correction", RelatedTxID: original.ID, IdempotencyKey: w.Key()}
	posted := tu.Must(w.Svc.CreateAdjustment(w.Ctx, w.admin, w.net.ID, in))
	if posted.RelatedTxID != original.ID || posted.OriginalTxID != "" || posted.ReversedTotal != 0 {
		t.Fatalf("relation changed reversal semantics: %+v", posted)
	}
	if w.balance(w.a).Balance != -475 || w.balance(w.b).Balance != 500 {
		t.Fatal("relation changed posting accounts")
	}
	if after := tu.Must(w.Svc.GetTransaction(w.Ctx, w.a, w.net.ID, original.ID)); !reflect.DeepEqual(beforeOriginal, after) {
		t.Fatal("adjustment rewrote original/history")
	}
	if len(tu.Must(w.Svc.ReversalsOf(w.Ctx, w.a, w.net.ID, original.ID))) != 0 {
		t.Fatal("adjustment counted as reversal")
	}
	if tu.Must(w.Svc.GetTransaction(w.Ctx, w.a, w.net.ID, posted.ID)).Tx.RelatedTxID != original.ID {
		t.Fatal("member lost relation")
	}
	if _, err := w.Svc.GetTransaction(w.Ctx, w.b, w.net.ID, posted.ID); domain.CodeOf(err) != domain.CodeNotFound {
		t.Fatalf("counterparty saw unrelated adjustment: %v", err)
	}
	before := dbChecksum(t, w.Env)
	if replay := tu.Must(w.Svc.CreateAdjustment(w.Ctx, w.admin, w.net.ID, in)); replay.ID != posted.ID {
		t.Fatal("replay duplicated adjustment")
	}
	in.RelatedTxID = ""
	if _, err := w.Svc.CreateAdjustment(w.Ctx, w.admin, w.net.ID, in); domain.CodeOf(err) != domain.CodeKeyReused {
		t.Fatalf("relation missing from fingerprint: %v", err)
	}
	if dbChecksum(t, w.Env) != before {
		t.Fatal("replay or conflicting relation changed state")
	}
	events := tu.Must(w.Svc.ListAudit(w.Ctx, w.admin, w.net.ID, service.AuditQuery{Action: domain.ActAdjustment, TargetID: posted.ID}))
	if len(events) != 1 || !strings.Contains(events[0].SummaryAfter, original.ID) {
		t.Fatalf("missing audit relation: %+v", events)
	}

	unrelated := w.Trade(w.net.ID, w.admin, w.b, w.adminM, w.bM, 100)
	open := tu.Must(w.propose(w.a, w.bM, service.ProposerPays, 100))
	// Reject a malformed stored proposal even if it has a positive sequence.
	tu.Must(0, w.Store.DB().QueryRow(`UPDATE transactions SET ledger_seq=999 WHERE id=? RETURNING ledger_seq`, open.ID).Scan(new(int64)))
	otherNet, otherAdmin := w.Network(w.admin, "other", nil)
	other, otherM := w.Member(w.admin, otherNet, "other@example.org")
	cross := w.Trade(otherNet.ID, w.admin, other, otherAdmin, otherM, 100)
	for _, id := range []string{"missing", unrelated.ID, open.ID, cross.ID} {
		before = dbChecksum(t, w.Env)
		in.RelatedTxID, in.IdempotencyKey = id, w.Key()
		if _, err := w.Svc.CreateAdjustmentRequest(w.Ctx, w.admin, w.net.ID, in); domain.CodeOf(err) != domain.CodeValidation {
			t.Fatalf("invalid relation %s: %v", id, err)
		}
		if dbChecksum(t, w.Env) != before {
			t.Fatal("invalid relation changed state")
		}
	}
	// Existing empty-link input marshals identically to pre-feature payloads.
	in.RelatedTxID = ""
	legacy := struct {
		MemberID       string
		Amount         int64
		Direction      service.AdjustDirection
		Reason         string
		IdempotencyKey string `json:"-"`
	}{in.MemberID, in.Amount, in.Direction, in.Reason, in.IdempotencyKey}
	if !bytes.Equal(tu.Must(json.Marshal(in)), tu.Must(json.Marshal(legacy))) {
		t.Fatal("standalone idempotency fingerprint changed")
	}
}

func TestAdjustmentRelationSurvivesDualControlAndIsImmutable(t *testing.T) {
	d := newDC(t)
	original := d.Trade(d.net.ID, d.a, d.b, d.aM, d.bM, 500)
	in := service.AdjustInput{MemberID: d.aM.ID, Amount: 2000, Direction: service.AdjustCredit, Reason: "corrected charge", RelatedTxID: original.ID, IdempotencyKey: d.Key()}
	pa := tu.Must(d.Svc.CreateAdjustmentRequest(d.Ctx, d.admin, d.net.ID, in)).Pending
	if pa.RelatedTxID != original.ID || d.balance(d.a).Balance != -500 {
		t.Fatalf("pending relation/posting: %+v", pa)
	}
	if _, err := d.Store.DB().Exec(`UPDATE pending_adjustments SET related_tx_id=NULL WHERE id=?`, pa.ID); err == nil {
		t.Fatal("pending relation mutable")
	}
	posted := tu.Must(d.Svc.ApproveAdjustment(d.Ctx, d.admin2, d.net.ID, pa.ID))
	if posted.RelatedTxID != original.ID || posted.OriginalTxID != "" {
		t.Fatalf("approval lost relation: %+v", posted)
	}
	if tu.Must(d.Svc.GetPendingAdjustment(d.Ctx, d.admin, d.net.ID, pa.ID)).RelatedTxID != original.ID {
		t.Fatal("decided request lost relation")
	}
	if _, err := d.Store.DB().Exec(`UPDATE transactions SET related_tx_id=NULL WHERE id=?`, posted.ID); err == nil {
		t.Fatal("posted relation mutable")
	}
	if tu.Must(d.Svc.ApproveAdjustment(d.Ctx, d.admin2, d.net.ID, pa.ID)).ID != posted.ID {
		t.Fatal("approval replay duplicated posting")
	}
	for _, action := range []string{domain.ActAdjustmentPending, domain.ActAdjustmentApproved, domain.ActAdjustment} {
		es := tu.Must(d.Svc.ListAudit(d.Ctx, d.admin, d.net.ID, service.AuditQuery{Action: action}))
		if len(es) != 1 || !strings.Contains(es[0].SummaryAfter, original.ID) {
			t.Fatalf("%s lost relation: %+v", action, es)
		}
	}
	for _, format := range []service.ExportFormat{service.FormatCSV, service.FormatJSON} {
		for _, scope := range []service.ExportScope{service.ExportMine, service.ExportLedger, service.ExportFull} {
			actor := d.admin
			if scope == service.ExportMine {
				actor = d.a
			}
			sink := &service.MemSink{}
			man := tu.Must(d.Svc.Export(d.Ctx, actor, d.net.ID, scope, format, sink))
			name := "transactions.csv"
			if format == service.FormatJSON {
				name = man.Files[0].Name
			}
			b := string(sink.Bytes(name))
			if !strings.Contains(b, "related_tx_id") || !strings.Contains(b, original.ID) {
				t.Fatalf("%s/%s lost relation: %s", scope, format, b)
			}
		}
	}
	// Other party's member export must not reveal the adjustment or even add the optional column.
	sink := &service.MemSink{}
	tu.Must(d.Svc.Export(d.Ctx, d.b, d.net.ID, service.ExportMine, service.FormatCSV, sink))
	if b := string(sink.Bytes("transactions.csv")); strings.Contains(b, "related_tx_id") || strings.Contains(b, posted.ID) {
		t.Fatalf("other-party export leaked adjustment: %s", b)
	}
}

// legacyRelationExportTx projects the exact pre-0018 transaction columns. It is
// a regression oracle for byte compatibility, independent of the new projection.
type legacyRelationExportStore struct{ store.Store }

func (s legacyRelationExportStore) View(ctx context.Context, fn func(store.Tx) error) error {
	return s.Store.View(ctx, func(tx store.Tx) error { return fn(legacyRelationExportTx{tx}) })
}

type legacyRelationExportTx struct{ store.Tx }

func (t legacyRelationExportTx) StreamTable(tbl store.ExportTable, net, member string, fn func([]string, []any) error) error {
	if tbl != store.TblTransactions {
		return t.Tx.StreamTable(tbl, net, member, fn)
	}
	columns := "id,kind,status,payer_account_id,payee_account_id,amount,listing_id,note,reason,original_tx_id,created_by,accepted_by,reversed_total,proposed_at,expires_at,completed_at,ledger_seq"
	if member != "" {
		columns = "id,kind,status,my_role,amount,payer_name,payee_name,note,reason,original_tx_id,proposed_at,completed_at,ledger_seq"
	}
	old := strings.Split(columns, ",")
	return t.Tx.StreamTable(tbl, net, member, func(cols []string, vals []any) error {
		if vals == nil {
			return fn(old, nil)
		}
		out := make([]any, len(old))
		for i, key := range old {
			for j, col := range cols {
				if col == key {
					out[i] = vals[j]
					break
				}
			}
		}
		return fn(old, out)
	})
}
func relationExportWorld(t *testing.T, legacy bool) *world {
	e := tu.NewEnvWith(t, func(st store.Store) store.Store {
		if legacy {
			return legacyRelationExportStore{st}
		}
		return st
	})
	w := &world{Env: e}
	w.admin = e.Person("admin@example.org")
	w.net, w.adminM = e.Network(w.admin, "demo", nil)
	w.a, w.aM = e.Member(w.admin, w.net, "ana@example.org")
	w.b, w.bM = e.Member(w.admin, w.net, "ben@example.org")
	w.Trade(w.net.ID, w.a, w.b, w.aM, w.bM, 500)
	tu.Must(w.Svc.CreateAdjustment(w.Ctx, w.admin, w.net.ID, service.AdjustInput{MemberID: w.aM.ID, Amount: 25, Direction: service.AdjustCredit, Reason: "standalone", IdempotencyKey: w.Key()}))
	return w
}
func TestAdjustmentRelationPreservesLegacyExportBytesAndClosureChecksum(t *testing.T) {
	for _, format := range []service.ExportFormat{service.FormatCSV, service.FormatJSON} {
		for _, scope := range []service.ExportScope{service.ExportMine, service.ExportLedger, service.ExportFull} {
			t.Run(string(scope)+"/"+string(format), func(t *testing.T) {
				old, current := relationExportWorld(t, true), relationExportWorld(t, false)
				a, b := &service.MemSink{}, &service.MemSink{}
				oldActor, actor := old.admin, current.admin
				if scope == service.ExportMine {
					oldActor, actor = old.a, current.a
				}
				om := tu.Must(old.Svc.Export(old.Ctx, oldActor, old.net.ID, scope, format, a))
				cm := tu.Must(current.Svc.Export(current.Ctx, actor, current.net.ID, scope, format, b))
				for name := range a.Files {
					if !bytes.Equal(a.Bytes(name), b.Bytes(name)) {
						t.Fatalf("pre-0018 export bytes changed: %s", name)
					}
				}
				if !reflect.DeepEqual(om, cm) {
					t.Fatal("legacy manifest changed")
				}
			})
		}
	}
	old, current := relationExportWorld(t, true), relationExportWorld(t, false)
	for _, w := range []*world{old, current} {
		tu.Must(w.schedule("end", 0))
		w.Clock.Advance(31 * 24 * time.Hour)
		tu.Must(w.Svc.FinaliseClosures(w.Ctx, sysJob))
	}
	if old.netNow().Closure.FinalSHA256 != current.netNow().Closure.FinalSHA256 {
		t.Fatal("pre-0018 closure checksum changed")
	}
	if !tu.Must(current.Svc.VerifyClosureExport(current.Ctx, current.admin, current.net.ID)).Match {
		t.Fatal("closure checksum no longer verifies")
	}
}
