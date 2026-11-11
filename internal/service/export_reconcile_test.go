package service_test

import (
	"archive/zip"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/csv"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"sort"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/djshellshoxxx/circula/internal/domain"
	"github.com/djshellshoxxx/circula/internal/service"
	"github.com/djshellshoxxx/circula/internal/store"
	tu "github.com/djshellshoxxx/circula/internal/testutil"
)

// exportedTables reads every table of a package without the service's code:
// encoding/csv for CSV files, encoding/json with exact numbers for the JSON
// document. Cells are strings; null is "".
func exportedTables(t *testing.T, sink *service.MemSink, man *service.Manifest) map[string][]map[string]string {
	t.Helper()
	out := map[string][]map[string]string{}
	for _, f := range man.Files {
		b := sink.Bytes(f.Name)
		if name, ok := strings.CutSuffix(f.Name, ".csv"); ok {
			recs, err := csv.NewReader(bytes.NewReader(b)).ReadAll()
			if err != nil || len(recs) == 0 {
				t.Fatalf("%s: %v", f.Name, err)
			}
			rows := []map[string]string{}
			for _, r := range recs[1:] {
				m := map[string]string{}
				for i, h := range recs[0] {
					m[h] = r[i]
				}
				rows = append(rows, m)
			}
			out[name] = rows
			continue
		}
		dec := json.NewDecoder(bytes.NewReader(b))
		dec.UseNumber()
		var doc struct {
			Tables map[string][]map[string]any `json:"tables"`
		}
		if err := dec.Decode(&doc); err != nil {
			t.Fatalf("%s: %v", f.Name, err)
		}
		for name, rows := range doc.Tables {
			out[name] = []map[string]string{}
			for _, r := range rows {
				m := map[string]string{}
				for k, v := range r {
					if v != nil {
						m[k] = fmt.Sprint(v)
					}
				}
				out[name] = append(out[name], m)
			}
		}
	}
	return out
}

func atoi(t *testing.T, s string) int64 {
	t.Helper()
	v, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		t.Fatalf("not an integer: %q", s)
	}
	return v
}

// snapshotBalances recomputes every account's balance as of ledger_seq seq
// with raw SQL over the append-only entries, independently of the export.
func snapshotBalances(t *testing.T, w *world, seq int64) map[string]int64 {
	t.Helper()
	rows, err := w.Store.DB().Query(`SELECT account_id, SUM(amount) FROM entries WHERE network_id = ? AND ledger_seq <= ?
		GROUP BY account_id`, w.net.ID, seq)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = rows.Close() }()
	out := map[string]int64{}
	for rows.Next() {
		var id string
		var sum int64
		if err := rows.Scan(&id, &sum); err != nil {
			t.Fatal(err)
		}
		out[id] = sum
	}
	return out
}

// reconcileExport recomputes the reconciliation of one export from its own
// files (QA-030) and returns every difference: per-transaction sums, per-
// account sums against the exported balances and against the database at the
// export's ledger_seq, and the manifest's summary. memberID is set for mine.
func reconcileExport(t *testing.T, w *world, sink *service.MemSink, man *service.Manifest, memberID string) []string {
	t.Helper()
	var diffs []string
	bad := func(f string, a ...any) { diffs = append(diffs, fmt.Sprintf(string(man.Scope)+"/"+string(man.Format)+": "+f, a...)) }
	tables := exportedTables(t, sink, man)
	entries := tables["entries"]
	var sum int64
	perTx, perAcct := map[string]int64{}, map[string]int64{}
	for _, e := range entries {
		amt := atoi(t, e["amount"])
		if seq := atoi(t, e["ledger_seq"]); seq > man.LedgerSeq {
			bad("entry %s posted at %d, after the snapshot %d", e["id"], seq, man.LedgerSeq)
		}
		sum += amt
		perTx[e["tx_id"]] += amt
		perAcct[e["account_id"]] += amt
	}
	at := snapshotBalances(t, w, man.LedgerSeq)
	var rawCount int64
	q := `SELECT COUNT(*) FROM entries WHERE network_id = ? AND ledger_seq <= ?`
	args := []any{w.net.ID, man.LedgerSeq}
	if memberID != "" {
		q += ` AND account_id = (SELECT id FROM accounts WHERE network_id = ? AND member_id = ?)`
		args = append(args, w.net.ID, memberID)
	}
	if err := w.Store.DB().QueryRow(q, args...).Scan(&rawCount); err != nil {
		t.Fatal(err)
	}
	if int64(len(entries)) != rawCount {
		bad("%d entries exported, %d posted by the snapshot", len(entries), rawCount)
	}
	got := man.Reconciliation
	if got == nil {
		bad("no reconciliation summary")
		return diffs
	}
	if got.Entries != int64(len(entries)) || got.EntrySum != sum {
		bad("summary says %d entries summing to %d; recomputed %d summing to %d", got.Entries, got.EntrySum, len(entries), sum)
	}
	if memberID != "" {
		var own string
		if err := w.Store.DB().QueryRow(`SELECT id FROM accounts WHERE network_id = ? AND member_id = ?`, w.net.ID, memberID).Scan(&own); err != nil {
			t.Fatal(err)
		}
		if sum != at[own] || got.BalanceTotal != at[own] {
			bad("own entries sum to %d, summary balance %d, balance at the snapshot %d", sum, got.BalanceTotal, at[own])
		}
		if got.Accounts != nil {
			bad("member export carries the network-wide balance digest")
		}
		return diffs
	}
	for tx, s := range perTx {
		if s != 0 {
			bad("transaction %s entries sum to %d", tx, s)
		}
	}
	if sum != 0 {
		bad("entries sum to %d", sum)
	}
	for id, b := range at {
		if perAcct[id] != b {
			bad("account %s: exported entries sum to %d, database at the snapshot %d", id, perAcct[id], b)
		}
	}
	var total int64
	for _, a := range tables["accounts"] {
		b := atoi(t, a["balance"])
		total += b
		if perAcct[a["id"]] != b {
			bad("account %s: exported balance %d, its exported entries sum to %d", a["id"], b, perAcct[a["id"]])
		}
	}
	if got.BalanceTotal != total || total != 0 {
		bad("summary balance total %d, exported balances total %d", got.BalanceTotal, total)
	}
	var ids []string
	for id, s := range perAcct {
		if s != 0 {
			ids = append(ids, id)
		}
	}
	sort.Strings(ids)
	h := sha256.New()
	for _, id := range ids {
		fmt.Fprintf(h, "%s,%d\n", id, perAcct[id])
	}
	if d := got.Accounts; d == nil || d.NonZero != int64(len(ids)) || d.SHA256 != hex.EncodeToString(h.Sum(nil)) {
		bad("balance digest %+v does not match the %d non-zero accounts recomputed from entries", d, len(ids))
	}
	return diffs
}

// GOV-022: every scope that exports entries carries a reconciliation summary
// whose entry sum and balance total are zero; scopes without entries carry none.
func TestExportReconciliationSummary(t *testing.T) {
	w := newWorld(t, nil)
	w.Trade(w.net.ID, w.a, w.b, w.aM, w.bM, 2500)
	w.Trade(w.net.ID, w.b, w.admin, w.bM, w.adminM, 700)
	tu.Must(w.adjust(w.admin, w.aM, 100, service.AdjustCredit, "bonus"))
	for _, format := range []service.ExportFormat{service.FormatCSV, service.FormatJSON} {
		for _, scope := range []service.ExportScope{service.ExportLedger, service.ExportFull} {
			sink := &service.MemSink{}
			man := tu.Must(w.Svc.Export(w.Ctx, w.admin, w.net.ID, scope, format, sink))
			r := man.Reconciliation
			if r == nil || r.Entries != 6 || r.EntrySum != 0 || r.BalanceTotal != 0 || r.Accounts == nil || r.Accounts.NonZero < 3 {
				t.Fatalf("%s/%s: reconciliation %+v", scope, format, r)
			}
			for _, d := range reconcileExport(t, w, sink, man, "") {
				t.Error(d)
			}
			// The summary is in manifest.json as written.
			var onDisk struct {
				Reconciliation *service.Reconciliation `json:"reconciliation"`
			}
			if err := json.Unmarshal(sink.Bytes("manifest.json"), &onDisk); err != nil || onDisk.Reconciliation == nil ||
				*onDisk.Reconciliation.Accounts != *r.Accounts || onDisk.Reconciliation.Entries != r.Entries {
				t.Errorf("%s/%s manifest.json: %v %s", scope, format, err, sink.Bytes("manifest.json"))
			}
		}
		// A's own export: two entries (-2500 trade, +100 adjustment) equal A's balance.
		sink := &service.MemSink{}
		man := tu.Must(w.Svc.Export(w.Ctx, w.a, w.net.ID, service.ExportMine, format, sink))
		if r := man.Reconciliation; r == nil || r.Entries != 2 || r.EntrySum != -2400 || r.BalanceTotal != -2400 || r.Accounts != nil {
			t.Errorf("mine/%s: reconciliation %+v", format, r)
		}
		for _, d := range reconcileExport(t, w, sink, man, w.aM.ID) {
			t.Error(d)
		}
		for _, scope := range []service.ExportScope{service.ExportMembers, service.ExportAudit} {
			sink := &service.MemSink{}
			man := tu.Must(w.Svc.Export(w.Ctx, w.admin, w.net.ID, scope, format, sink))
			if man.Reconciliation != nil || bytes.Contains(sink.Bytes("manifest.json"), []byte("reconciliation")) {
				t.Errorf("%s/%s exports no entries but has a reconciliation summary", scope, format)
			}
		}
	}
}

// exportCorpus builds a network with seeded random trades, partial and full
// reversals, adjustments and decided disputes (QA-030 property corpus).
// Refused commands are part of the corpus; internal errors fail the test.
func exportCorpus(t *testing.T, seed int64, steps int) (*world, []domain.Actor, []*domain.Member) {
	t.Helper()
	rng := rand.New(rand.NewSource(seed))
	w := newWorldCfg(t, func(p *domain.Policy) { p.DefaultCreditLimit = 5000 }, func(c *service.Config) { c.ExportsPerHour = -1 })
	rev, _ := w.reviewer()
	actors, members := []domain.Actor{w.a, w.b}, []*domain.Member{w.aM, w.bM}
	for i := 0; i < 3; i++ {
		a, m := w.Member2(w.admin, fmt.Sprintf("c%d@example.org", i))
		actors, members = append(actors, a), append(members, m)
	}
	check := func(err error, what string) {
		if err != nil && domain.CodeOf(err) == domain.CodeInternal {
			t.Fatalf("seed %d %s: %v", seed, what, err)
		}
	}
	completed := func() *domain.Transaction {
		l := tu.Must(w.Svc.AdminListTransactions(w.Ctx, w.admin, w.net.ID, "", []domain.TxStatus{domain.TxCompleted}, service.Page{Limit: 200}))
		if len(l) == 0 {
			return nil
		}
		return l[rng.Intn(len(l))]
	}
	for step := 0; step < steps; step++ {
		switch rng.Intn(7) {
		case 0, 1, 2: // trade
			i, j := rng.Intn(len(members)), rng.Intn(len(members))
			tx, err := w.Svc.ProposeTransaction(w.Ctx, actors[i], w.net.ID, service.ProposeInput{CounterpartyID: members[j].ID,
				Direction: service.ProposerPays, Amount: 1 + rng.Int63n(2000), IdempotencyKey: w.Key()})
			check(err, "propose")
			if err == nil {
				_, err = w.Svc.AcceptTransaction(w.Ctx, actors[j], w.net.ID, service.TxRef{TxID: tx.ID, IdempotencyKey: w.Key()})
				check(err, "accept")
			}
		case 3: // full or partial reversal
			if tx := completed(); tx != nil && tx.Kind != domain.KindReversal {
				amt := int64(0)
				if rng.Intn(2) == 0 {
					amt = 1 + rng.Int63n(tx.Amount)
				}
				_, err := w.Svc.ReverseTransaction(w.Ctx, w.admin, w.net.ID, service.ReverseInput{TxID: tx.ID, Amount: amt, Reason: "r", IdempotencyKey: w.Key()})
				check(err, "reverse")
			}
		case 4: // adjustment
			dir := service.AdjustCredit
			if rng.Intn(2) == 0 {
				dir = service.AdjustDebit
			}
			_, err := w.Svc.CreateAdjustment(w.Ctx, w.admin, w.net.ID, service.AdjustInput{MemberID: members[rng.Intn(len(members))].ID,
				Amount: 1 + rng.Int63n(3000), Direction: dir, Reason: "r", IdempotencyKey: w.Key()})
			check(err, "adjust")
		case 5, 6: // dispute, decided at once
			tx := completed()
			if tx == nil || tx.Kind != domain.KindTrade {
				break
			}
			payer := actors[0]
			for i, m := range members {
				if m.ID == tx.PayerMemberID {
					payer = actors[i]
				}
			}
			d, err := w.Svc.OpenDispute(w.Ctx, payer, w.net.ID, service.OpenDisputeInput{TxID: tx.ID, Category: "other", IdempotencyKey: w.Key()})
			check(err, "dispute")
			if err != nil {
				break
			}
			outcome := []domain.DisputeOutcome{domain.OutcomeDismissed, domain.OutcomeUpheld, domain.OutcomePartiallyUpheld}[rng.Intn(3)]
			amt := int64(0)
			if outcome == domain.OutcomePartiallyUpheld {
				amt = 1 + rng.Int63n(tx.Amount)
			}
			_, err = w.Svc.ResolveDispute(w.Ctx, rev, w.net.ID, service.ResolveInput{DisputeID: d.ID, Outcome: outcome, Amount: amt, Rationale: "r", IdempotencyKey: w.Key()})
			check(err, "resolve")
		}
	}
	return w, actors, members
}

// QA-030: zero reconciliation differences across the property corpus, for
// every export type that holds entries (ledger, full, each member's own) in
// both formats; the test recomputes the summary itself.
func TestExportReconcilesAcrossCorpus(t *testing.T) {
	steps := 120
	if raceOn {
		steps = 40
	}
	var diffs, exports, entries int
	var kinds = map[string]bool{}
	for seed := int64(1); seed <= 4; seed++ {
		w, actors, members := exportCorpus(t, seed, steps)
		rows, _ := w.Store.DB().Query(`SELECT DISTINCT kind FROM transactions WHERE network_id = ? AND ledger_seq > 0`, w.net.ID)
		for rows.Next() {
			var k string
			_ = rows.Scan(&k)
			kinds[k] = true
		}
		_ = rows.Close()
		var decided int
		_ = w.Store.DB().QueryRow(`SELECT COUNT(*) FROM disputes WHERE network_id = ? AND outcome IS NOT NULL`, w.net.ID).Scan(&decided)
		if decided > 0 {
			kinds["decided dispute"] = true
		}
		for _, format := range []service.ExportFormat{service.FormatCSV, service.FormatJSON} {
			for _, scope := range []service.ExportScope{service.ExportLedger, service.ExportFull} {
				sink := &service.MemSink{}
				man := tu.Must(w.Svc.Export(w.Ctx, w.admin, w.net.ID, scope, format, sink))
				for _, d := range reconcileExport(t, w, sink, man, "") {
					t.Errorf("seed %d: %s", seed, d)
					diffs++
				}
				exports++
				entries += int(man.Reconciliation.Entries)
			}
			for i, a := range actors {
				sink := &service.MemSink{}
				man := tu.Must(w.Svc.Export(w.Ctx, a, w.net.ID, service.ExportMine, format, sink))
				for _, d := range reconcileExport(t, w, sink, man, members[i].ID) {
					t.Errorf("seed %d member %d: %s", seed, i, d)
					diffs++
				}
				exports++
			}
		}
	}
	t.Logf("%d exports, %d network entries, kinds posted %v, %d differences", exports, entries, kinds, diffs)
	// guard against a corpus that never reaches the interesting postings
	for _, k := range []string{"trade", "reversal", "adjustment", "decided dispute"} {
		if !kinds[k] {
			t.Errorf("corpus posted no %s", k)
		}
	}
}

// pauseStore runs hook (once) inside an export's read transaction, after the
// accounts and before the entries are read, so a test can commit postings
// while the export is in flight.
type pauseStore struct {
	store.Store
	hook atomic.Pointer[func()]
}

func (p *pauseStore) View(ctx context.Context, fn func(store.Tx) error) error {
	return p.Store.View(ctx, func(tx store.Tx) error { return fn(pauseTx{Tx: tx, p: p}) })
}

type pauseTx struct {
	store.Tx
	p *pauseStore
}

func (t pauseTx) StreamTable(tbl store.ExportTable, netID, memberID string, fn func([]string, []any) error) error {
	if tbl == store.TblEntries {
		if h := t.p.hook.Swap(nil); h != nil {
			(*h)()
		}
	}
	return t.Tx.StreamTable(tbl, netID, memberID, fn)
}

// QA-030: an export taken during concurrent postings reconciles to its own
// snapshot time. First deterministically (postings commit inside the export's
// read transaction, between the balances and the entries), then with a
// free-running poster beside repeated exports.
func TestExportDuringConcurrentPostingsReconciles(t *testing.T) {
	var ps *pauseStore
	e := tu.NewEnvConfig(t, func(s store.Store) store.Store { ps = &pauseStore{Store: s}; return ps },
		func(c *service.Config) { c.ExportsPerHour = -1 })
	w := &world{Env: e}
	w.admin = e.Person("admin@example.org")
	w.net, w.adminM = e.Network(w.admin, "demo", func(p *domain.Policy) { p.DefaultCreditLimit = 1_000_000 })
	w.a, w.aM = e.Member(w.admin, w.net, "ana@example.org")
	w.b, w.bM = e.Member(w.admin, w.net, "ben@example.org")
	for i := 0; i < 5; i++ {
		w.Trade(w.net.ID, w.a, w.b, w.aM, w.bM, int64(100+i))
	}
	ledgerSeq := func() int64 {
		var s int64
		if err := w.Store.DB().QueryRow(`SELECT ledger_seq FROM networks WHERE id = ?`, w.net.ID).Scan(&s); err != nil {
			t.Fatal(err)
		}
		return s
	}

	cases := []struct {
		actor  domain.Actor
		member string
		scope  service.ExportScope
		format service.ExportFormat
	}{
		{w.admin, "", service.ExportLedger, service.FormatCSV},
		{w.admin, "", service.ExportFull, service.FormatJSON},
		{w.a, w.aM.ID, service.ExportMine, service.FormatCSV},
		{w.b, w.bM.ID, service.ExportMine, service.FormatJSON},
	}
	for _, c := range cases {
		before := ledgerSeq()
		hook := func() {
			w.Trade(w.net.ID, w.b, w.a, w.bM, w.aM, 777)
			tu.Must(w.adjust(w.admin, w.aM, 50, service.AdjustCredit, "mid-export"))
		}
		ps.hook.Store(&hook)
		sink := &service.MemSink{}
		man := tu.Must(w.Svc.Export(w.Ctx, c.actor, w.net.ID, c.scope, c.format, sink))
		if ps.hook.Load() != nil {
			t.Fatal("test premise: no posting ran during the export")
		}
		if man.LedgerSeq != before || ledgerSeq() != before+2 {
			t.Fatalf("%s/%s: snapshot at %d, ledger before %d and after %d", c.scope, c.format, man.LedgerSeq, before, ledgerSeq())
		}
		for _, d := range reconcileExport(t, w, sink, man, c.member) {
			t.Errorf("mid-export postings: %s", d)
		}
	}

	// Free-running: a poster commits trades while exports are taken.
	stop := make(chan struct{})
	var posted atomic.Int64
	var wg sync.WaitGroup
	errc := make(chan error, 1)
	wg.Add(1)
	go func() {
		defer wg.Done()
		pairs := [][2]int{{0, 1}, {1, 0}}
		ac, ms := []domain.Actor{w.a, w.b}, []*domain.Member{w.aM, w.bM}
		for i := 0; ; i++ {
			select {
			case <-stop:
				return
			default:
			}
			p := pairs[i%2]
			tx, err := w.Svc.ProposeTransaction(context.Background(), ac[p[0]], w.net.ID, service.ProposeInput{CounterpartyID: ms[p[1]].ID,
				Direction: service.ProposerPays, Amount: int64(1 + i%97), IdempotencyKey: fmt.Sprintf("poster-p-%d", i)})
			if err == nil {
				_, err = w.Svc.AcceptTransaction(context.Background(), ac[p[1]], w.net.ID, service.TxRef{TxID: tx.ID, IdempotencyKey: fmt.Sprintf("poster-a-%d", i)})
			}
			if err != nil {
				errc <- err
				return
			}
			posted.Add(1)
		}
	}()
	rounds := 12
	if raceOn {
		rounds = 6
	}
	seqs := map[int64]bool{}
	for i := 0; i < rounds; i++ {
		c := cases[i%len(cases)]
		sink := &service.MemSink{}
		man, err := w.Svc.Export(w.Ctx, c.actor, w.net.ID, c.scope, c.format, sink)
		if err != nil {
			close(stop)
			wg.Wait()
			t.Fatal(err)
		}
		seqs[man.LedgerSeq] = true
		for _, d := range reconcileExport(t, w, sink, man, c.member) {
			t.Errorf("round %d: %s", i, d)
		}
		// let the poster move the ledger on before the next export
		for target := posted.Load() + 1; posted.Load() < target; {
			select {
			case err := <-errc:
				t.Fatalf("poster: %v", err)
			default:
			}
		}
	}
	close(stop)
	wg.Wait()
	select {
	case err := <-errc:
		t.Fatalf("poster: %v", err)
	default:
	}
	if len(seqs) < rounds/2 {
		t.Errorf("exports saw only %d distinct snapshots in %d rounds", len(seqs), rounds)
	}
	t.Logf("%d postings beside %d exports at %d distinct ledger_seq values", posted.Load(), rounds, len(seqs))
}

// FR-095 / ARC-020: ZipSink writes each file as it is created, manifest last,
// and refuses unsafe names like DirSink.
func TestZipSinkStreamsPackage(t *testing.T) {
	w := newWorld(t, nil)
	w.Trade(w.net.ID, w.a, w.b, w.aM, w.bM, 2500)
	var zb bytes.Buffer
	zs := service.NewZipSink(&zb)
	man := tu.Must(w.Svc.Export(w.Ctx, w.admin, w.net.ID, service.ExportLedger, service.FormatCSV, zs))
	if err := zs.Close(); err != nil {
		t.Fatal(err)
	}
	mem := &service.MemSink{}
	tu.Must(w.Svc.Export(w.Ctx, w.admin, w.net.ID, service.ExportLedger, service.FormatCSV, mem))
	zr := tu.Must(zip.NewReader(bytes.NewReader(zb.Bytes()), int64(zb.Len())))
	var names []string
	for _, f := range zr.File {
		names = append(names, f.Name)
		rc := tu.Must(f.Open())
		b := tu.Must(io.ReadAll(rc))
		_ = rc.Close()
		if f.Name != "manifest.json" && !bytes.Equal(b, mem.Bytes(f.Name)) {
			t.Errorf("%s differs between ZipSink and MemSink", f.Name)
		}
	}
	want := []string{"accounts.csv", "transactions.csv", "transaction_status_events.csv", "entries.csv", "manifest.json"}
	if strings.Join(names, ",") != strings.Join(want, ",") || len(man.Files) != 4 {
		t.Errorf("archive entries %v, want %v", names, want)
	}
	for _, bad := range []string{"../x.csv", ".hidden", "a/b.csv", ""} {
		if _, err := zs.Create(bad); err == nil {
			t.Errorf("ZipSink accepted %q", bad)
		}
	}
}
