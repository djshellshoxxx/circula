package qr_test

import (
	"bytes"
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	qr "github.com/djshellshoxxx/circula/future/qr"
	"github.com/djshellshoxxx/circula/future/qr/qrfixture"
)

// worker is an independent connection pool + store + service on the same file,
// modelling separate writers contending for one SQLite database.
func (e *env) worker() *qr.Service {
	e.t.Helper()
	db := openDB(e.t, e.path)
	return e.newService(qr.NewStore(db))
}

func (e *env) addMembers(prefix string, n int) []string {
	var ids []string
	for i := 0; i < n; i++ {
		id := fmt.Sprintf("%s%02d", prefix, i)
		must(e.t, qrfixture.AddMember(bg, e.db, "n1", id, "M"+id, 1000))
		ids = append(ids, id)
	}
	return ids
}

func TestConcurrentScansClaimAtMostOnce(t *testing.T) {
	e := newEnv(t)
	payers := e.addMembers("p", 12)
	cr, err := e.svc.CreateRequest(bg, p("n1", "a"), "create-0001", qr.CreateRequestInput{AmountMinor: 100})
	must(t, err)
	svcs := []*qr.Service{e.svc, e.worker(), e.worker()}
	var wg sync.WaitGroup
	var wins, claimed, other atomic.Int64
	var mu sync.Mutex
	winners := map[string]string{}
	start := make(chan struct{})
	for i, payer := range payers {
		for rep := 0; rep < 2; rep++ { // each payer also retries concurrently
			wg.Add(1)
			go func() {
				defer wg.Done()
				<-start
				r, err := svcs[(i+rep)%3].ClaimRequest(bg, p("n1", payer), cr.Token, "")
				switch {
				case err == nil:
					mu.Lock()
					winners[payer] = r.TransactionID
					mu.Unlock()
					if !r.Replayed {
						wins.Add(1)
					}
				case errors.Is(err, qr.ErrClaimed):
					claimed.Add(1)
				default:
					other.Add(1)
					t.Errorf("unexpected: %v", err)
				}
			}()
		}
	}
	close(start)
	wg.Wait()
	if wins.Load() != 1 || len(winners) != 1 || other.Load() != 0 {
		t.Fatalf("wins=%d winners=%v claimed=%d", wins.Load(), winners, claimed.Load())
	}
	if c := e.counts(); c.Proposed != 1 || e.fx.Proposed.Load() != 1 {
		t.Fatalf("exactly one proposal expected: %+v proposed-calls=%d", c, e.fx.Proposed.Load())
	}
	if e.scalar(`SELECT COUNT(*) FROM qr_trade_request WHERE claimed_tx_id IS NOT NULL`) != 1 {
		t.Fatal("claim rows")
	}
	if e.scalar(`SELECT COUNT(*) FROM qr_idempotency WHERE scope='claim.request'`) != 1 {
		t.Fatal("idempotency rows")
	}
}

func TestConcurrentClaimVersusRevokeIsConsistent(t *testing.T) {
	e := newEnv(t)
	w2 := e.worker()
	for i := 0; i < 25; i++ {
		cr, err := e.svc.CreateRequest(bg, p("n1", "a"), fmt.Sprintf("create-rv-%03d", i), qr.CreateRequestInput{AmountMinor: 1})
		must(t, err)
		var claimErr, revErr error
		var wg sync.WaitGroup
		wg.Add(2)
		go func() { defer wg.Done(); _, claimErr = e.svc.ClaimRequest(bg, p("n1", "b"), cr.Token, "") }()
		go func() { defer wg.Done(); _, revErr = w2.RevokeRequest(bg, p("n1", "a"), cr.View.ID) }()
		wg.Wait()
		switch {
		case claimErr == nil && errors.Is(revErr, qr.ErrClaimed):
		case errors.Is(claimErr, qr.ErrRevoked) && revErr == nil:
		default:
			t.Fatalf("iter %d inconsistent: claim=%v revoke=%v", i, claimErr, revErr)
		}
		var claimed, revoked bool
		must(t, e.db.QueryRow(`SELECT claimed_tx_id IS NOT NULL, revoked_at IS NOT NULL FROM qr_trade_request WHERE id=?`, cr.View.ID).Scan(&claimed, &revoked))
		if claimed == revoked {
			t.Fatalf("iter %d: claimed=%v revoked=%v", i, claimed, revoked)
		}
	}
	if got, want := int64(e.counts().Proposed), e.scalar(`SELECT COUNT(*) FROM qr_trade_request WHERE claimed_tx_id IS NOT NULL`); got != want {
		t.Fatalf("proposals %d != claims %d", got, want)
	}
}

func TestConcurrentRotationAndCreationLimits(t *testing.T) {
	e := newEnv(t, func(c *qr.Config) { c.MaxActiveRequests = 5 })
	svcs := []*qr.Service{e.svc, e.worker(), e.worker()}
	var wg sync.WaitGroup
	var ok, limited atomic.Int64
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := svcs[i%3].CreateRequest(bg, p("n1", "a"), fmt.Sprintf("create-lim-%03d", i), qr.CreateRequestInput{AmountMinor: 1})
			switch {
			case err == nil:
				ok.Add(1)
			case errors.Is(err, qr.ErrRateLimited):
				limited.Add(1)
			default:
				t.Errorf("%v", err)
			}
		}()
	}
	wg.Wait()
	if ok.Load() != 5 || limited.Load() != 15 {
		t.Fatal(ok.Load(), limited.Load())
	}
	// Same-key rotation storm: one generation. Distinct keys: sequential, one live.
	for i := 0; i < 9; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if _, err := svcs[i%3].RotateAlias(bg, p("n1", "b"), fmt.Sprintf("rot-storm-%03d", i/3)); err != nil {
				t.Errorf("%v", err)
			}
		}()
	}
	wg.Wait()
	if n := e.scalar(`SELECT COUNT(*) FROM qr_member_alias WHERE member_id='b'`); n != 3 {
		t.Fatalf("generations: %d", n)
	}
	if n := e.scalar(`SELECT COUNT(*) FROM qr_member_alias WHERE member_id='b' AND revoked_at IS NULL`); n != 1 {
		t.Fatalf("live aliases: %d", n)
	}
}

func TestRestartRecovery(t *testing.T) {
	e := newEnv(t)
	cr, _ := e.svc.CreateRequest(bg, p("n1", "a"), "create-0001", qr.CreateRequestInput{AmountMinor: 120, Note: "n"})
	open, _ := e.svc.CreateRequest(bg, p("n1", "a"), "create-0002", qr.CreateRequestInput{AmountMinor: 5})
	al, _ := e.svc.MyAlias(bg, p("n1", "a"))
	r1, err := e.svc.ClaimRequest(bg, p("n1", "b"), cr.Token, "claim-key-01")
	must(t, err)
	// "Crash": drop every handle, reopen the same file with a brand-new process-like stack.
	must(t, e.db.Close())
	db2 := openDB(t, e.path)
	st2 := qr.NewStore(db2)
	must(t, st2.Migrate(bg)) // idempotent on restart
	svc2 := e.newService(st2)
	r2, err := svc2.ClaimRequest(bg, p("n1", "b"), cr.Token, "claim-key-01")
	must(t, err)
	if !r2.Replayed || r2.TransactionID != r1.TransactionID {
		t.Fatalf("%+v %+v", r1, r2)
	}
	if _, err := svc2.ClaimRequest(bg, p("n1", "c"), cr.Token, ""); !errors.Is(err, qr.ErrClaimed) {
		t.Fatal(err)
	}
	if pv, err := svc2.PreviewRequest(bg, p("n1", "c"), open.Token); err != nil || pv.AmountMinor != 5 {
		t.Fatal(pv, err)
	}
	if a2, _ := svc2.MyAlias(bg, p("n1", "a")); a2.Code != al.Code {
		t.Fatal("alias changed across restart")
	}
	c, _ := qrfixture.Count(bg, db2)
	if c.Proposed != 1 {
		t.Fatalf("%+v", c)
	}
	// The completed trade is still reported as completed after another restart.
	must(t, st2.Update(bg, func(ctx context.Context, q qr.DBTX) error { return e.fx.Accept(ctx, q, "n1", r1.TransactionID, "a") }))
	v, _ := svc2.GetRequest(bg, p("n1", "a"), cr.View.ID)
	if v.Status != qr.StatusCompleted {
		t.Fatal(v.Status)
	}
}

func TestNoRawCodesAtRest(t *testing.T) {
	e := newEnv(t)
	cr, _ := e.svc.CreateRequest(bg, p("n1", "a"), "create-0001", qr.CreateRequestInput{AmountMinor: 120, Note: "n"})
	al, _ := e.svc.MyAlias(bg, p("n1", "a"))
	rot, _ := e.svc.RotateAlias(bg, p("n1", "a"), "rot-key-0001")
	_, _ = e.svc.ClaimRequest(bg, p("n1", "b"), cr.Token, "claim-key-01")
	_, _ = e.svc.ProposeToAlias(bg, p("n1", "c"), rot.Code, 5, "", "alias-key-01")
	secrets := []string{cr.Token, al.Code, rot.Code, string(secret)}
	// 1. every text value of every module and fixture table
	for _, tbl := range []string{"qr_member_alias", "qr_trade_request", "qr_idempotency", "qr_schema_migrations", "fx_tx", "fx_audit"} {
		rows, err := e.db.Query("SELECT * FROM " + tbl)
		must(t, err)
		cols, _ := rows.Columns()
		for rows.Next() {
			vals := make([]any, len(cols))
			ptrs := make([]any, len(cols))
			for i := range vals {
				ptrs[i] = &vals[i]
			}
			must(t, rows.Scan(ptrs...))
			for _, v := range vals {
				var s string
				switch x := v.(type) {
				case []byte:
					s = string(x)
				case string:
					s = x
				}
				for _, sec := range secrets {
					if strings.Contains(s, sec) {
						t.Fatalf("raw secret in %s", tbl)
					}
				}
			}
		}
		rows.Close()
	}
	// 2. raw database and WAL file bytes
	_, _ = e.db.Exec(`PRAGMA wal_checkpoint(FULL)`)
	for _, suf := range []string{"", "-wal"} {
		b, err := os.ReadFile(e.path + suf)
		if err != nil {
			continue
		}
		for _, sec := range secrets {
			if bytes.Contains(b, []byte(sec)) {
				t.Fatalf("raw secret in file%s", suf)
			}
		}
	}
}

func TestBackupRestore(t *testing.T) {
	e := newEnv(t)
	cr, _ := e.svc.CreateRequest(bg, p("n1", "a"), "create-0001", qr.CreateRequestInput{AmountMinor: 9})
	al, _ := e.svc.MyAlias(bg, p("n1", "a"))
	dest := filepath.Join(t.TempDir(), "backup.db")
	must(t, e.store.Backup(bg, dest))
	if err := e.store.Backup(bg, dest); err == nil {
		t.Fatal("backup must not overwrite")
	}
	if err := e.store.Backup(bg, ""); err == nil {
		t.Fatal("empty path")
	}
	// Mutate the source after the backup, then restore from the copy.
	_, err := e.svc.ClaimRequest(bg, p("n1", "b"), cr.Token, "claim-key-01")
	must(t, err)
	db2 := openDB(t, dest)
	svc2 := e.newService(qr.NewStore(db2))
	pv, err := svc2.PreviewRequest(bg, p("n1", "b"), cr.Token)
	if err != nil || pv.Claimed {
		t.Fatalf("restored copy must show the pre-claim state: %+v %v", pv, err)
	}
	a2, _ := svc2.MyAlias(bg, p("n1", "a"))
	if a2.Code != al.Code {
		t.Fatal("alias must re-derive identically from the server secret")
	}
	if _, err := svc2.ClaimRequest(bg, p("n1", "b"), cr.Token, "claim-key-01"); err != nil {
		t.Fatal(err)
	}
}

func TestMigrationsIdempotentAndRollback(t *testing.T) {
	e := newEnv(t)
	must(t, e.store.Migrate(bg))
	must(t, e.store.Migrate(bg))
	if e.scalar(`SELECT COUNT(*) FROM qr_schema_migrations`) != 1 {
		t.Fatal("ledger")
	}
	_, _ = e.svc.MyAlias(bg, p("n1", "a"))
	must(t, e.store.Rollback(bg, 1)) // nothing newer than 1
	if e.scalar(`SELECT COUNT(*) FROM qr_member_alias`) != 1 {
		t.Fatal("rollback to current version must be a no-op")
	}
	must(t, e.store.Rollback(bg, 0))
	var n int
	must(t, e.db.QueryRow(`SELECT COUNT(*) FROM sqlite_master WHERE name LIKE 'qr\_%' ESCAPE '\' AND name <> 'qr_schema_migrations'`).Scan(&n))
	if n != 0 {
		t.Fatalf("%d module objects survive rollback", n)
	}
	if e.scalar(`SELECT COUNT(*) FROM fx_member`) == 0 {
		t.Fatal("rollback touched host tables")
	}
	must(t, e.store.Rollback(bg, 0)) // repeat is safe
	must(t, e.store.Migrate(bg))
	if _, err := e.svc.MyAlias(bg, p("n1", "a")); err != nil {
		t.Fatal(err)
	}
}

func TestSchemaConstraintsHoldEvenAgainstDirectSQL(t *testing.T) {
	e := newEnv(t)
	h := bytes.Repeat([]byte{1}, 32)
	h2 := bytes.Repeat([]byte{2}, 32)
	ins := `INSERT INTO qr_trade_request(id,network_id,recipient_id,token_hash,amount_minor,note,created_at,expires_at,revoked_at,claimed_tx_id,claimed_by,claimed_at,claim_fingerprint) VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?)`
	ok := func(id string, hash []byte, amt int64, exp int64, rev any, tx, by any, at any, fp any) error {
		_, err := e.db.Exec(ins, id, "n1", "a", hash, amt, "", 10, exp, rev, tx, by, at, fp)
		return err
	}
	must(t, ok("r1", h, 5, 20, nil, nil, nil, nil, nil))
	cases := map[string]error{
		"duplicate token hash": ok("r2", h, 5, 20, nil, nil, nil, nil, nil),
		"zero amount":          ok("r3", h2, 0, 20, nil, nil, nil, nil, nil),
		"expiry before create": ok("r4", h2, 5, 10, nil, nil, nil, nil, nil),
		"short hash":           ok("r5", []byte{1}, 5, 20, nil, nil, nil, nil, nil),
		"claimed and revoked":  ok("r6", h2, 5, 20, 15, "tx_1", "b", 15, h),
		"claim without by":     ok("r7", h2, 5, 20, nil, "tx_1", nil, 15, h),
		"claimed by recipient": ok("r8", h2, 5, 20, nil, "tx_1", "a", 15, h),
		"by without claim":     ok("r9", h2, 5, 20, nil, nil, "b", nil, nil),
	}
	for name, err := range cases {
		if err == nil {
			t.Errorf("%s accepted", name)
		}
	}
	must(t, ok("r10", h2, 5, 20, nil, "tx_9", "b", 15, h))
	if err := ok("r11", bytes.Repeat([]byte{3}, 32), 5, 20, nil, "tx_9", "c", 15, h); err == nil {
		t.Error("same transaction claimed by two requests")
	}
	// alias table: a second live alias per member, or a duplicate hash in a network
	al := `INSERT INTO qr_member_alias(network_id,member_id,generation,alias_hash,revoked_at,created_at) VALUES(?,?,?,?,?,1)`
	_, err := e.db.Exec(al, "n1", "a", 1, h, nil)
	must(t, err)
	if _, err := e.db.Exec(al, "n1", "a", 2, h2, nil); err == nil {
		t.Error("two live aliases")
	}
	if _, err := e.db.Exec(al, "n1", "b", 1, h, nil); err == nil {
		t.Error("duplicate alias hash in one network")
	}
	if _, err := e.db.Exec(al, "n2", "x", 1, h, nil); err != nil {
		t.Errorf("the same hash in another network is a different tenant namespace: %v", err)
	}
	if _, err := e.db.Exec(`UPDATE qr_trade_request SET revoked_at=30 WHERE id='r10'`); err == nil {
		t.Error("revoking a claimed request must violate the check")
	}
}

func TestPurgeRetention(t *testing.T) {
	e := newEnv(t, func(c *qr.Config) { c.MaxActiveRequests = 50 })
	old, _ := e.svc.CreateRequest(bg, p("n1", "a"), "create-0001", qr.CreateRequestInput{AmountMinor: 1, TTL: time.Minute})
	rev, _ := e.svc.CreateRequest(bg, p("n1", "a"), "create-0002", qr.CreateRequestInput{AmountMinor: 1})
	_, _ = e.svc.RevokeRequest(bg, p("n1", "a"), rev.View.ID)
	cl, _ := e.svc.CreateRequest(bg, p("n1", "a"), "create-0003", qr.CreateRequestInput{AmountMinor: 1})
	res, err := e.svc.ClaimRequest(bg, p("n1", "b"), cl.Token, "claim-key-01")
	must(t, err)
	live, _ := e.svc.CreateRequest(bg, p("n1", "a"), "create-0004", qr.CreateRequestInput{AmountMinor: 1, TTL: 7 * 24 * time.Hour})
	e.clk.Add(31 * 24 * time.Hour)
	live2, _ := e.svc.CreateRequest(bg, p("n1", "a"), "create-0005", qr.CreateRequestInput{AmountMinor: 1})
	if _, _, err := e.store.Purge(bg, e.clk.Now(), qr.Retention{}); err == nil {
		t.Fatal("zero retention must be rejected")
	}
	reqs, idem, err := e.store.Purge(bg, e.clk.Now(), qr.DefaultRetention)
	must(t, err)
	if reqs != 2 || idem != 5 { // old(expired) and rev(revoked); the 7-day request is inside retention; idempotency rows 1-4 + claim
		t.Fatalf("purged requests=%d idempotency=%d", reqs, idem)
	}
	if _, err := e.svc.PreviewRequest(bg, p("n1", "b"), old.Token); !errors.Is(err, qr.ErrNotFound) {
		t.Fatal(err)
	}
	_ = live
	if pv, err := e.svc.PreviewRequest(bg, p("n1", "b"), live2.Token); err != nil || pv.AmountMinor != 1 {
		t.Fatal(pv, err)
	}
	// Claimed requests survive purge, so an intent replay still returns the first proposal.
	r2, err := e.svc.ClaimRequest(bg, p("n1", "b"), cl.Token, "fresh-key-after-purge")
	must(t, err)
	if !r2.Replayed || r2.TransactionID != res.TransactionID {
		t.Fatalf("%+v", r2)
	}
	_ = sql.ErrNoRows
}
