package store_test

import (
	"bytes"
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/djshellshoxxx/circula/internal/domain"
	"github.com/djshellshoxxx/circula/internal/store"
)

// schemaSnapshot is every schema object's type, name and SQL, in a stable order.
func schemaSnapshot(t *testing.T, db *sql.DB) string {
	t.Helper()
	rows, err := db.Query(`SELECT type || ' ' || name || ' ' || IFNULL(tbl_name,'') || ' ' || IFNULL(sql,'')
		FROM sqlite_master ORDER BY type, name`)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = rows.Close() }()
	var b strings.Builder
	for rows.Next() {
		var line string
		if err := rows.Scan(&line); err != nil {
			t.Fatal(err)
		}
		b.WriteString(line)
		b.WriteByte('\n')
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}
	return b.String()
}

// rowCounts returns the row count of each named table.
func rowCounts(t *testing.T, db *sql.DB, tables ...string) map[string]int {
	t.Helper()
	out := map[string]int{}
	for _, tbl := range tables {
		var n int
		if err := db.QueryRow("SELECT COUNT(*) FROM " + tbl).Scan(&n); err != nil {
			t.Fatalf("count %s: %v", tbl, err)
		}
		out[tbl] = n
	}
	return out
}

// postBalanced is post with the cached balances updated as the posting unit
// does (DAT-014): the existing helper leaves them stale, so ledger checks
// would fail on the fixture rather than on the store.
func postBalanced(t *testing.T, s *store.SQLite, netID, txID, accA, accB string) {
	t.Helper()
	err := s.Update(context.Background(), func(tx store.Tx) error {
		now := time.Now()
		if err := tx.InsertEntry(&domain.Entry{ID: "e1", NetworkID: netID, TxID: txID, AccountID: accA, Amount: -100, LedgerSeq: 1, PostedAt: now}); err != nil {
			return err
		}
		if err := tx.AddBalance(netID, accA, -100); err != nil {
			return err
		}
		if err := tx.InsertEntry(&domain.Entry{ID: "e2", NetworkID: netID, TxID: txID, AccountID: accB, Amount: 100, LedgerSeq: 1, PostedAt: now}); err != nil {
			return err
		}
		if err := tx.AddBalance(netID, accB, 100); err != nil {
			return err
		}
		x, err := tx.GetTransaction(netID, txID)
		if err != nil {
			return err
		}
		x.Status = domain.TxCompleted
		done, deadline := now, now.Add(30*24*time.Hour) // the service fixes the deadline at completion (DOM-017)
		x.CompletedAt, x.DisputeDeadlineAt, x.DisputePolicyVersion = &done, &deadline, 1
		return tx.UpdateTransaction(x)
	})
	if err != nil {
		t.Fatal(err)
	}
}

// ledgerFindings runs the read-only ledger checks of the store API in one
// snapshot and returns every finding (empty means the ledger verifies).
func ledgerFindings(t *testing.T, s *store.SQLite, netID string) (findings []string, bal, ent int64) {
	t.Helper()
	err := s.View(context.Background(), func(tx store.Tx) error {
		mm, err := tx.BalanceMismatches(netID)
		if err != nil {
			return err
		}
		for _, m := range mm {
			findings = append(findings, fmt.Sprintf("account %s cached %d summed %d", m.AccountID, m.Cached, m.Summed))
		}
		probs, err := tx.TxProblems(netID)
		if err != nil {
			return err
		}
		findings = append(findings, probs...)
		if bal, ent, err = tx.Totals(netID); err != nil {
			return err
		}
		if bal != 0 || ent != 0 {
			findings = append(findings, fmt.Sprintf("totals balances %d entries %d", bal, ent))
		}
		rev, err := tx.ReversalMismatches(netID)
		if err != nil {
			return err
		}
		findings = append(findings, rev...)
		return nil
	})
	if err != nil {
		t.Fatalf("ledger read: %v", err)
	}
	return findings, bal, ent
}

// ---- migrations ----------------------------------------------------------

// A fresh database reaches the head schema and records every version with
// its file name; the pragmas the design depends on are set.
func TestFreshDatabaseRecordsEveryMigration(t *testing.T) {
	ctx := context.Background()
	s, err := store.Open(ctx, filepath.Join(t.TempDir(), "fresh.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = s.Close() }()

	files, err := filepath.Glob("migrations/*.sql")
	if err != nil {
		t.Fatal(err)
	}
	head := store.SchemaVersion()
	if head != len(files) || head == 0 {
		t.Fatalf("SchemaVersion %d, %d migration files", head, len(files))
	}
	rows, err := s.DB().Query(`SELECT version, name FROM schema_migrations ORDER BY version`)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = rows.Close() }()
	v := 0
	for rows.Next() {
		var ver int
		var name string
		if err := rows.Scan(&ver, &name); err != nil {
			t.Fatal(err)
		}
		v++
		if ver != v {
			t.Fatalf("recorded versions not contiguous: got %d at position %d", ver, v)
		}
		base := strings.TrimSuffix(filepath.Base(files[v-1]), ".sql")
		if want := strings.SplitN(base, "_", 2)[1]; name != want {
			t.Errorf("version %d recorded as %q, file says %q", ver, name, want)
		}
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}
	if v != head {
		t.Fatalf("%d versions recorded, want %d", v, head)
	}
	if err := store.CheckPragmas(ctx, s.DB()); err != nil {
		t.Fatal(err)
	}
}

// Reopening an already-migrated database changes nothing: no version is
// re-applied (applied_at unchanged) and the schema objects are identical.
func TestMigratingAgainIsANoOp(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "again.db")
	s, err := store.Open(ctx, path)
	if err != nil {
		t.Fatal(err)
	}
	applied := func(db *sql.DB) string {
		var b strings.Builder
		rows, err := db.Query(`SELECT version || ':' || applied_at FROM schema_migrations ORDER BY version`)
		if err != nil {
			t.Fatal(err)
		}
		defer func() { _ = rows.Close() }()
		for rows.Next() {
			var x string
			if err := rows.Scan(&x); err != nil {
				t.Fatal(err)
			}
			b.WriteString(x + ";")
		}
		return b.String()
	}
	wantApplied, wantSchema := applied(s.DB()), schemaSnapshot(t, s.DB())
	if err := s.Close(); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 2; i++ {
		s, err := store.Open(ctx, path)
		if err != nil {
			t.Fatalf("reopen #%d: %v", i, err)
		}
		if got := applied(s.DB()); got != wantApplied {
			t.Fatalf("reopen #%d re-recorded migrations: %s != %s", i, got, wantApplied)
		}
		if got := schemaSnapshot(t, s.DB()); got != wantSchema {
			t.Fatalf("reopen #%d changed the schema", i)
		}
		if err := s.Close(); err != nil {
			t.Fatal(err)
		}
	}
}

// Migration 0015 (the head) is blocked by an object it must create. The SQL of
// the migration creates its first trigger before the second one fails, so the
// failure is a real mid-migration error. The open must fail, the schema must be
// byte-for-byte what it was (no partial DDL), the previous version must still be
// recorded, and the database must open cleanly once the blocker is removed.
func TestFailingMigrationRollsBackAndReopensCleanly(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "mig.db")
	head := store.SchemaVersion()
	if head < 2 {
		t.Fatalf("need at least two migrations, have %d", head)
	}

	// Build the database at head-1 with one committed row.
	prev, err := store.OpenAtVersion(ctx, path, head-1)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := prev.DB().Exec(`INSERT INTO persons(id,login,password_hash,created_at) VALUES ('p1','before-upgrade','h',1)`); err != nil {
		t.Fatal(err)
	}
	if err := prev.Close(); err != nil {
		t.Fatal(err)
	}

	// Find the object the last migration creates last and make it already exist.
	files, err := filepath.Glob("migrations/*.sql")
	if err != nil || len(files) != head {
		t.Fatalf("migration files: %v (%d)", err, len(files))
	}
	sort.Strings(files)
	last, err := os.ReadFile(files[len(files)-1])
	if err != nil {
		t.Fatal(err)
	}
	creates := regexp.MustCompile(`(?i)CREATE\s+(TABLE|TRIGGER|INDEX)\s+(?:IF\s+NOT\s+EXISTS\s+)?(\w+)`).FindAllStringSubmatch(string(last), -1)
	if len(creates) < 2 {
		t.Fatalf("head migration creates %d objects; need at least 2 to fail midway", len(creates))
	}
	kind, blocked := strings.ToUpper(creates[len(creates)-1][1]), creates[len(creates)-1][2]

	raw, err := store.OpenRaw(path)
	if err != nil {
		t.Fatal(err)
	}
	stmt := map[string]string{
		"TABLE":   fmt.Sprintf(`CREATE TABLE %s (x INTEGER)`, blocked),
		"INDEX":   fmt.Sprintf(`CREATE TABLE blocker_tbl (x INTEGER); CREATE INDEX %s ON blocker_tbl(x)`, blocked),
		"TRIGGER": fmt.Sprintf(`CREATE TABLE blocker_tbl (x INTEGER); CREATE TRIGGER %s AFTER INSERT ON blocker_tbl BEGIN SELECT 1; END`, blocked),
	}[kind]
	if _, err := raw.ExecContext(ctx, stmt); err != nil {
		t.Fatalf("plant blocker: %v", err)
	}
	before := schemaSnapshot(t, raw)
	if err := raw.Close(); err != nil {
		t.Fatal(err)
	}

	s, err := store.Open(ctx, path)
	if err == nil {
		_ = s.Close()
		t.Fatal("open succeeded although the head migration must fail")
	}
	if !strings.Contains(err.Error(), fmt.Sprintf("migration %04d_", head)) {
		t.Errorf("error does not name the failing migration: %v", err)
	}

	raw, err = store.OpenRaw(path)
	if err != nil {
		t.Fatal(err)
	}
	if got := schemaSnapshot(t, raw); got != before {
		t.Fatalf("failed migration left partial schema changes:\nbefore:\n%s\nafter:\n%s", before, got)
	}
	if v, err := store.AppliedVersion(ctx, raw); err != nil || v != head-1 {
		t.Fatalf("applied version = %d err=%v, want %d", v, err, head-1)
	}
	if err := raw.Close(); err != nil {
		t.Fatal(err)
	}

	// Remove the blocker; the same file now upgrades and keeps its data.
	raw, err = store.OpenRaw(path)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := raw.ExecContext(ctx, `DROP TRIGGER IF EXISTS `+blocked); err != nil {
		t.Fatal(err)
	}
	if kind == "TABLE" {
		if _, err := raw.ExecContext(ctx, `DROP TABLE `+blocked); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := raw.ExecContext(ctx, `DROP TABLE IF EXISTS blocker_tbl`); err != nil {
		t.Fatal(err)
	}
	if err := raw.Close(); err != nil {
		t.Fatal(err)
	}
	s, err = store.Open(ctx, path)
	if err != nil {
		t.Fatalf("reopen after clearing blocker: %v", err)
	}
	defer func() { _ = s.Close() }()
	var n int
	var v int
	if err := s.DB().QueryRow(`SELECT COUNT(*), MAX(version) FROM schema_migrations`).Scan(&n, &v); err != nil || v != head || n != head {
		t.Fatalf("after upgrade: %d rows, max %d (err %v), want head %d", n, v, err, head)
	}
	if err := s.DB().QueryRow(`SELECT COUNT(*) FROM persons WHERE id='p1' AND login='before-upgrade'`).Scan(&n); err != nil || n != 1 {
		t.Fatalf("pre-upgrade row lost: n=%d err=%v", n, err)
	}
}

// ---- reopen and ledger verification ---------------------------------------

// Committed postings survive close and reopen, and the ledger verifies after
// the restart with the store's own read-only checks.
func TestCommittedPostingsSurviveReopenAndVerify(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "reopen.db")
	s, err := store.Open(ctx, path)
	if err != nil {
		t.Fatal(err)
	}
	netID, txID, a, b := seed(t, s)
	postBalanced(t, s, netID, txID, a, b)
	if f, bal, ent := ledgerFindings(t, s, netID); len(f) != 0 || bal != 0 || ent != 0 {
		t.Fatalf("ledger before restart: %v (bal %d ent %d)", f, bal, ent)
	}
	var seq int64
	before := rowCounts(t, s.DB(), "networks", "members", "accounts", "transactions", "entries", "transaction_status_events")
	if err := s.View(ctx, func(tx store.Tx) error {
		var err error
		seq, err = tx.LedgerSeq(netID)
		return err
	}); err != nil {
		t.Fatal(err)
	}
	if err := s.Close(); err != nil {
		t.Fatal(err)
	}

	s, err = store.Open(ctx, path)
	if err != nil {
		t.Fatalf("reopen: %v", err)
	}
	defer func() { _ = s.Close() }()
	if got := rowCounts(t, s.DB(), "networks", "members", "accounts", "transactions", "entries", "transaction_status_events"); fmt.Sprint(got) != fmt.Sprint(before) {
		t.Fatalf("row counts changed across restart: %v -> %v", before, got)
	}
	if f, bal, ent := ledgerFindings(t, s, netID); len(f) != 0 || bal != 0 || ent != 0 {
		t.Fatalf("ledger after restart: %v (bal %d ent %d)", f, bal, ent)
	}
	var got int64
	var status string
	if err := s.View(ctx, func(tx store.Tx) error {
		var err error
		if got, err = tx.LedgerSeq(netID); err != nil {
			return err
		}
		x, err := tx.GetTransaction(netID, txID)
		if err != nil {
			return err
		}
		status = string(x.Status)
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	if got != seq || status != string(domain.TxCompleted) {
		t.Fatalf("after restart ledger_seq %d (want %d), status %q", got, seq, status)
	}
}

// ---- unit-of-work atomicity ------------------------------------------------

// A posting that fails on its second leg leaves no entry, no balance change and
// no status change behind, and the gate is released so the next unit runs.
func TestPostingFailingMidwayLeavesNoPartialRows(t *testing.T) {
	ctx := context.Background()
	s := openStore(t)
	netID, txID, a, b := seed(t, s)
	var before domain.Transaction
	if err := s.View(ctx, func(tx store.Tx) error {
		x, err := tx.GetTransaction(netID, txID)
		if err != nil {
			return err
		}
		before = *x
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	counts := func() map[string]int {
		return rowCounts(t, s.DB(), "entries", "transaction_status_events", "audit_events")
	}
	base := counts()

	now := time.Now()
	err := s.Update(ctx, func(tx store.Tx) error {
		if err := tx.InsertEntry(&domain.Entry{ID: "e1", NetworkID: netID, TxID: txID, AccountID: a, Amount: -100, LedgerSeq: 1, PostedAt: now}); err != nil {
			return err
		}
		if err := tx.AddBalance(netID, a, -100); err != nil {
			return err
		}
		if err := tx.InsertEntry(&domain.Entry{ID: "e2", NetworkID: netID, TxID: txID, AccountID: b, Amount: 100, LedgerSeq: 1, PostedAt: now}); err != nil {
			return err
		}
		// Past the +-2^62 bound of migration 0015: the database refuses it.
		if err := tx.AddBalance(netID, b, 1<<62+1); err != nil {
			return err
		}
		x, err := tx.GetTransaction(netID, txID)
		if err != nil {
			return err
		}
		x.Status = domain.TxCompleted
		done, deadline := now, now.Add(30*24*time.Hour) // the service fixes the deadline at completion (DOM-017)
		x.CompletedAt, x.DisputeDeadlineAt, x.DisputePolicyVersion = &done, &deadline, 1
		return tx.UpdateTransaction(x)
	})
	if err == nil {
		t.Fatal("second leg must fail")
	}
	if !strings.Contains(err.Error(), "account balance out of range") {
		t.Fatalf("second leg must fail on the balance bound, got: %v", err)
	}
	if domain.CodeOf(err) == domain.CodeUnavailable {
		t.Fatalf("a constraint failure must not look retryable: %v", err)
	}
	if got := counts(); fmt.Sprint(got) != fmt.Sprint(base) {
		t.Fatalf("rows left behind: %v -> %v", base, got)
	}
	if f, bal, ent := ledgerFindings(t, s, netID); len(f) != 0 || bal != 0 || ent != 0 {
		t.Fatalf("ledger not clean after rollback: %v", f)
	}
	var after domain.Transaction
	if err := s.View(ctx, func(tx store.Tx) error {
		x, err := tx.GetTransaction(netID, txID)
		if err != nil {
			return err
		}
		after = *x
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	if after.Status != before.Status {
		t.Fatalf("status %s -> %s despite rollback", before.Status, after.Status)
	}
	// The gate was released: a following unit of work runs and commits.
	if err := s.Update(ctx, func(tx store.Tx) error { return tx.PutThrottle("after-failure", 1, now) }); err != nil {
		t.Fatalf("unit after failed unit: %v", err)
	}
}

// A foreign-key failure on a later row of a multi-row write rolls back the
// earlier rows of the same unit, including a fresh person and network.
func TestMultiRowWriteIsAllOrNothing(t *testing.T) {
	ctx := context.Background()
	s := openStore(t)
	now := time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)
	tables := []string{"persons", "networks", "network_policies", "members", "accounts"}
	base := rowCounts(t, s.DB(), tables...)

	err := s.Update(ctx, func(tx store.Tx) error {
		if err := tx.InsertPerson(&domain.Person{ID: "pa", Login: "pa", CreatedAt: now}, "h"); err != nil {
			return err
		}
		if err := tx.InsertNetwork(&domain.Network{ID: "na", Name: "N", Slug: "na", Status: domain.NetworkActive, CreatedAt: now}); err != nil {
			return err
		}
		if err := tx.AppendPolicy("na", 1, domain.DefaultPolicy(), "", "", now); err != nil {
			return err
		}
		if err := tx.InsertMember(&domain.Member{ID: "ma", NetworkID: "na", PersonID: "pa", DisplayName: "pa", Status: domain.MemberActive, JoinMethod: "founder", JoinedAt: now}); err != nil {
			return err
		}
		// The account's member does not exist: the foreign key must refuse it.
		return tx.InsertAccount(&domain.Account{ID: "aa", NetworkID: "na", MemberID: "no-such-member"})
	})
	if err == nil {
		t.Fatal("dangling member reference must fail the unit")
	}
	if got := rowCounts(t, s.DB(), tables...); fmt.Sprint(got) != fmt.Sprint(base) {
		t.Fatalf("partial setup committed: %v -> %v", base, got)
	}
}

// ---- backup and restore ---------------------------------------------------

// A snapshot taken while committed data sits in the WAL opens, passes the
// integrity check, verifies the ledger and matches the source row counts.
func TestVacuumIntoBackupMatchesSource(t *testing.T) {
	ctx := context.Background()
	s := openStore(t)
	netID, txID, a, b := seed(t, s)
	postBalanced(t, s, netID, txID, a, b)
	tables := []string{"persons", "networks", "network_policies", "members", "accounts", "transactions",
		"transaction_status_events", "entries", "audit_events", "schema_migrations"}
	want := rowCounts(t, s.DB(), tables...)

	dest := filepath.Join(t.TempDir(), "snap.db")
	if err := store.VacuumInto(ctx, s.DB(), dest); err != nil {
		t.Fatalf("snapshot: %v", err)
	}
	res, err := store.IntegrityCheck(ctx, dest)
	if err != nil || res != "ok" {
		t.Fatalf("integrity of snapshot: %q err=%v", res, err)
	}
	raw, err := store.OpenRaw(dest)
	if err != nil {
		t.Fatal(err)
	}
	if got := rowCounts(t, raw, tables...); fmt.Sprint(got) != fmt.Sprint(want) {
		t.Fatalf("snapshot rows %v, source %v", got, want)
	}
	if err := raw.Close(); err != nil {
		t.Fatal(err)
	}

	bk, err := store.Open(ctx, dest)
	if err != nil {
		t.Fatalf("snapshot does not open as a store: %v", err)
	}
	defer func() { _ = bk.Close() }()
	if f, _, _ := ledgerFindings(t, bk, netID); len(f) != 0 {
		t.Fatalf("snapshot ledger: %v", f)
	}
}

// A snapshot never overwrites an existing file: VACUUM INTO refuses the
// destination and its content is untouched. Restores go through the same
// no-overwrite rule (a restore into a new path).
func TestSnapshotNeverOverwritesExistingFile(t *testing.T) {
	ctx := context.Background()
	s := openStore(t)
	seed(t, s)
	dest := filepath.Join(t.TempDir(), "existing.db")
	sentinel := []byte("an existing file that must survive")
	if err := os.WriteFile(dest, sentinel, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := store.VacuumInto(ctx, s.DB(), dest); err == nil {
		t.Fatal("snapshot over an existing file must fail")
	}
	got, err := os.ReadFile(dest)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, sentinel) {
		t.Fatal("existing file was modified")
	}
}

// ---- corruption ------------------------------------------------------------

// A file that is not a SQLite database is refused by Open and by the schema
// inspection, and Open leaves its bytes alone.
func TestGarbageFileIsRefusedNotOpened(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "garbage.db")
	junk := bytes.Repeat([]byte("not a sqlite database page "), 300)
	if err := os.WriteFile(path, junk, 0o600); err != nil {
		t.Fatal(err)
	}
	if s, err := store.Open(ctx, path); err == nil {
		_ = s.Close()
		t.Fatal("garbage file opened as a store")
	}
	if _, err := store.InspectSchema(ctx, path); err == nil {
		t.Fatal("schema inspection accepted a garbage file")
	}
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, junk) {
		t.Fatal("Open modified a file it refused")
	}
}

// A database cut short is reported: the store refuses to open it, or, if the
// damaged pages are not touched by the open, the integrity check reports it.
// It is never reported as healthy.
func TestTruncatedDatabaseIsReported(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "trunc.db")
	s, err := store.Open(ctx, path)
	if err != nil {
		t.Fatal(err)
	}
	netID, txID, a, b := seed(t, s)
	postBalanced(t, s, netID, txID, a, b)
	if _, err := s.DB().ExecContext(ctx, `PRAGMA wal_checkpoint(TRUNCATE)`); err != nil {
		t.Fatal(err)
	}
	if err := s.Close(); err != nil {
		t.Fatal(err)
	}
	fi, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Truncate(path, fi.Size()/2); err != nil {
		t.Fatal(err)
	}

	st, err := store.Open(ctx, path)
	if err != nil {
		t.Logf("truncated file refused by Open: %v", err)
		return
	}
	defer func() { _ = st.Close() }()
	res, err := store.IntegrityCheck(ctx, path)
	t.Logf("truncated file opened; integrity check: %q (err %v)", res, err)
	if err == nil && res == "ok" {
		t.Fatal("truncated database opened and passed the integrity check")
	}
}

// A single damaged b-tree page (its type byte overwritten with an invalid
// value) is reported by PRAGMA integrity_check, which is the check backup and
// restore run before they trust a file. Whether Open itself notices is logged:
// Open does not run the integrity check, so it may not.
func TestDamagedPageIsReportedByIntegrityCheck(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "page.db")
	s, err := store.Open(ctx, path)
	if err != nil {
		t.Fatal(err)
	}
	netID, txID, a, b := seed(t, s)
	postBalanced(t, s, netID, txID, a, b)
	if _, err := s.DB().ExecContext(ctx, `PRAGMA wal_checkpoint(TRUNCATE)`); err != nil {
		t.Fatal(err)
	}
	if err := s.Close(); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	pageSize := int(data[16])<<8 | int(data[17])
	if pageSize == 1 {
		pageSize = 65536
	}
	// Damage the last table leaf page (type 0x0d) that is not page 1.
	target := -1
	for off := len(data) - pageSize; off >= pageSize; off -= pageSize {
		if data[off] == 0x0d {
			target = off
			break
		}
	}
	if target < 0 {
		t.Fatal("no table leaf page found to damage")
	}
	data[target] = 0x01 // not a valid b-tree page type
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatal(err)
	}

	res, err := store.IntegrityCheck(ctx, path)
	if err == nil && res == "ok" {
		t.Fatal("integrity check passed on a damaged page")
	}
	st, err := store.Open(ctx, path)
	if err != nil {
		t.Logf("Open refused the damaged file: %v (integrity: %q)", err, res)
		return
	}
	defer func() { _ = st.Close() }()
	t.Logf("Open accepted the damaged file; integrity check reports %q", res)
}

// ---- concurrency -----------------------------------------------------------

// Writers on two handles to one file (two processes in production) either
// commit or fail with the retryable unavailable error. A third connection holds
// the write lock until one writer has reported that retryable failure, so the
// contention path runs on every test run (no sleep: the lock is released by the
// writer's signal). Every acknowledged write is present afterwards and no
// failure is silent.
func TestConcurrentWritersNeverSilentlyLoseWrites(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "conc.db")
	var handles []*store.SQLite
	for i := 0; i < 2; i++ {
		s, err := store.OpenWith(ctx, path, store.Options{BusyTimeout: 20 * time.Millisecond})
		if err != nil {
			t.Fatal(err)
		}
		defer func() { _ = s.Close() }()
		handles = append(handles, s)
	}
	holder, err := store.OpenRaw(path)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = holder.Close() }()
	conn, err := holder.Conn(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = conn.Close() }()
	if _, err := conn.ExecContext(ctx, "BEGIN IMMEDIATE"); err != nil {
		t.Fatal(err)
	}

	const writers, perWriter = 6, 20
	var mu sync.Mutex
	acked := map[string]bool{}
	var failed int
	busySeen := make(chan struct{}, 1)
	var wg sync.WaitGroup
	bad := make(chan error, writers*perWriter)
	for w := 0; w < writers; w++ {
		wg.Add(1)
		go func(w int) {
			defer wg.Done()
			s := handles[w%len(handles)]
			for i := 0; i < perWriter; i++ {
				login := fmt.Sprintf("w%02d-%03d", w, i)
				err := s.Update(ctx, func(tx store.Tx) error {
					return tx.PutThrottle(login, 1, time.UnixMilli(1))
				})
				if err == nil {
					mu.Lock()
					acked[login] = true
					mu.Unlock()
					continue
				}
				if domain.CodeOf(err) != domain.CodeUnavailable || !domain.IsRetryable(err) {
					bad <- fmt.Errorf("%s: non-retryable failure: %w", login, err)
				}
				mu.Lock()
				failed++
				mu.Unlock()
				select {
				case busySeen <- struct{}{}:
				default:
				}
			}
		}(w)
	}
	select {
	case <-busySeen:
	case <-time.After(60 * time.Second): // a guard against a hang, not a synchronization sleep
		t.Error("no writer reported contention while the lock was held")
	}
	if _, err := conn.ExecContext(ctx, "ROLLBACK"); err != nil {
		t.Fatal(err)
	}
	wg.Wait()
	close(bad)
	for err := range bad {
		t.Error(err)
	}
	t.Logf("acknowledged %d, failed with retryable error %d", len(acked), failed)
	if failed == 0 || len(acked) == 0 {
		t.Fatalf("expected both contention failures and acknowledged writes, got %d and %d", failed, len(acked))
	}

	seen := map[string]bool{}
	rows, err := handles[0].DB().Query(`SELECT login FROM login_throttle WHERE login LIKE 'w__-%'`)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = rows.Close() }()
	stored := 0
	for rows.Next() {
		var l string
		if err := rows.Scan(&l); err != nil {
			t.Fatal(err)
		}
		seen[l] = true
		stored++
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}
	for login := range acked {
		if !seen[login] {
			t.Errorf("acknowledged write %s is missing", login)
		}
	}
	if stored != len(acked) {
		t.Errorf("stored %d rows, acknowledged %d", stored, len(acked))
	}
}
