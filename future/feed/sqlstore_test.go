package feed

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"io"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestSQLStoreRejectsMissingDatabase(t *testing.T) {
	if store, err := NewSQLStore(nil); err != ErrInvalidQuery || store != nil {
		t.Fatalf("nil constructor = %v, %v", store, err)
	}
	for _, store := range []*SQLStore{nil, {}} {
		if err := store.InitSchema(context.Background()); err != ErrInvalidQuery {
			t.Errorf("schema = %v", err)
		}
		if err := store.Upsert(context.Background(), Source{}); err != ErrInvalidQuery {
			t.Errorf("upsert = %v", err)
		}
		if err := store.Delete(context.Background(), "n", "listing", "id"); err != ErrInvalidQuery {
			t.Errorf("delete = %v", err)
		}
		if _, err := store.HighWater(context.Background(), "n"); err != ErrInvalidQuery {
			t.Errorf("high water = %v", err)
		}
		if _, err := store.QueryBatch(context.Background(), StoreQuery{Network: "n", Limit: 1}); err != ErrInvalidQuery {
			t.Errorf("query = %v", err)
		}
	}
}

// The capture driver proves rejected requests never reach a database and that
// executed queries use parameters, a seek, and a database-level LIMIT.
type storeCapture struct {
	mu       sync.Mutex
	query    string
	args     []driver.NamedValue
	rows     [][]driver.Value
	affected int64
	err      error
}
type storeConnector struct{ capture *storeCapture }

func (c storeConnector) Connect(context.Context) (driver.Conn, error) {
	return &storeConn{c.capture}, nil
}
func (c storeConnector) Driver() driver.Driver { return storeDriver{c.capture} }

type storeDriver struct{ capture *storeCapture }

func (d storeDriver) Open(string) (driver.Conn, error) { return &storeConn{d.capture}, nil }

type storeConn struct{ c *storeCapture }

func (c *storeConn) Prepare(string) (driver.Stmt, error) {
	return nil, errors.New("unexpected prepare")
}
func (c *storeConn) Close() error              { return nil }
func (c *storeConn) Begin() (driver.Tx, error) { return storeTx{}, nil }

type storeTx struct{}

func (storeTx) Commit() error   { return nil }
func (storeTx) Rollback() error { return nil }
func (c *storeConn) QueryContext(_ context.Context, q string, args []driver.NamedValue) (driver.Rows, error) {
	c.c.mu.Lock()
	defer c.c.mu.Unlock()
	c.c.query = q
	c.c.args = append([]driver.NamedValue(nil), args...)
	if c.c.err != nil {
		return nil, c.c.err
	}
	return &storeRows{rows: append([][]driver.Value(nil), c.c.rows...)}, nil
}
func (c *storeConn) ExecContext(_ context.Context, q string, args []driver.NamedValue) (driver.Result, error) {
	c.c.mu.Lock()
	defer c.c.mu.Unlock()
	c.c.query = q
	c.c.args = append([]driver.NamedValue(nil), args...)
	if c.c.err != nil {
		return nil, c.c.err
	}
	return driver.RowsAffected(c.c.affected), nil
}

type storeRows struct {
	rows [][]driver.Value
	i    int
}

func (r *storeRows) Columns() []string {
	if len(r.rows) > 0 && len(r.rows[0]) == 1 {
		return []string{"sequence"}
	}
	return []string{"sequence", "network", "kind", "source_id", "member", "order_time", "visible", "active"}
}
func (r *storeRows) Close() error { return nil }
func (r *storeRows) Next(out []driver.Value) error {
	if r.i == len(r.rows) {
		return io.EOF
	}
	copy(out, r.rows[r.i])
	r.i++
	return nil
}
func captureStore(t *testing.T) (*SQLStore, *storeCapture) {
	t.Helper()
	c := &storeCapture{affected: 1}
	db := sql.OpenDB(storeConnector{c})
	t.Cleanup(func() { db.Close() })
	s, err := NewSQLStore(db)
	if err != nil {
		t.Fatal(err)
	}
	return s, c
}

func TestSQLStoreValidatesInputsBeforeQuery(t *testing.T) {
	s, c := captureStore(t)
	for _, q := range []StoreQuery{
		{Network: "n", Limit: 0}, {Network: "n", Limit: 102}, {Network: "", Limit: 1},
		{Network: "n", Limit: 1, Kinds: []string{"trade"}},
		{Network: "n", Limit: 1, After: &Cursor{}},
		{Network: "n", Limit: 1, Kinds: []string{"listing"}, After: &Cursor{Created: time.Unix(1, 0), Kind: "announcement", ID: "id"}},
	} {
		if rows, err := s.QueryBatch(context.Background(), q); err != ErrInvalidQuery || rows != nil {
			t.Fatalf("query %+v = %v, %v", q, rows, err)
		}
	}
	if c.query != "" {
		t.Fatalf("invalid inputs reached DB: %s", c.query)
	}
	if _, err := s.QueryBatch(nil, StoreQuery{Network: "n", Limit: 1}); err != ErrInvalidQuery {
		t.Fatal(err)
	}
	for _, source := range []Source{{}, {Network: "n", Kind: "trade", ID: "id", Member: "m", Created: time.Unix(1, 0)}, {Network: "n", Kind: "listing", ID: "id", Member: strings.Repeat("m", 257), Created: time.Unix(1, 0)}} {
		if err := s.Upsert(context.Background(), source); err != ErrInvalidSource {
			t.Fatalf("source %+v: %v", source, err)
		}
	}
	if c.query != "" {
		t.Fatalf("invalid source reached DB: %s", c.query)
	}
}

func TestSQLStoreUsesBoundedIndexedSeek(t *testing.T) {
	s, c := captureStore(t)
	pos := Cursor{Created: time.Unix(100, 42), Kind: "listing", ID: "'; DROP TABLE x; --"}
	for _, kinds := range [][]string{nil, {"listing"}} {
		rows, err := s.QueryBatch(context.Background(), StoreQuery{Network: "n", Kinds: kinds, After: &pos, Limit: 101})
		if err != nil || rows == nil || len(rows) != 0 {
			t.Fatalf("empty = %v, %v", rows, err)
		}
		if !strings.Contains(c.query, "LIMIT ?") || !strings.Contains(c.query, ") > (") || !strings.Contains(c.query, "INDEXED BY") || strings.Contains(c.query, pos.ID) || strings.Contains(c.query, "sequence <=") {
			t.Fatalf("query must seek/limit candidates before high-water filtering: %s", c.query)
		}
		if c.args[len(c.args)-1].Value != int64(101) {
			t.Fatalf("limit binding = %+v", c.args)
		}
	}
}

func TestSQLStoreReadsSourcesAndRejectsCorruptRows(t *testing.T) {
	s, c := captureStore(t)
	valid := []driver.Value{int64(7), "n", "listing", "id", "member", int64(-100_000_000_042), true, true}
	c.rows = [][]driver.Value{valid}
	rows, err := s.QueryBatch(context.Background(), StoreQuery{Network: "n", Limit: 1})
	if err != nil || len(rows) != 1 || rows[0].Sequence != 7 || !rows[0].Source.Created.Equal(time.Unix(100, 42)) {
		t.Fatalf("read = %+v, %v", rows, err)
	}
	for index, value := range map[int]driver.Value{0: int64(0), 1: "other", 2: "trade", 3: "", 4: "", 5: int64(0), 6: false, 7: false} {
		bad := append([]driver.Value(nil), valid...)
		bad[index] = value
		c.rows = [][]driver.Value{bad}
		if rows, err := s.QueryBatch(context.Background(), StoreQuery{Network: "n", Limit: 1}); err != ErrInvalidSource || rows != nil {
			t.Fatalf("corrupt %d: %+v, %v", index, rows, err)
		}
	}
	c.rows = [][]driver.Value{valid, valid}
	if rows, err := s.QueryBatch(context.Background(), StoreQuery{Network: "n", Limit: 1}); err != ErrInvalidSource || rows != nil {
		t.Fatalf("over-limit DB result: %+v, %v", rows, err)
	}
}

func TestSQLStoreConflictingAndDeletedSourcesFailClosed(t *testing.T) {
	s, c := captureStore(t)
	c.affected = 0
	if err := s.Upsert(context.Background(), Source{Network: "n", Kind: "listing", ID: "id", Member: "m", Created: time.Unix(1, 0), Visible: true, Active: true}); err != ErrInvalidSource {
		t.Fatalf("conflict = %v", err)
	}
	if !strings.Contains(c.query, "deleted = 0") || !strings.Contains(c.query, "order_time = excluded.order_time") {
		t.Fatalf("missing immutable/tombstone guard: %s", c.query)
	}
	c.affected = 1
	if err := s.Delete(context.Background(), "n", "listing", "id"); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(c.query, "ON CONFLICT") || !strings.Contains(c.query, "deleted = 1") {
		t.Fatalf("delete must preserve tombstone: %s", c.query)
	}
}

func TestSQLStoreHighWaterAndErrors(t *testing.T) {
	s, c := captureStore(t)
	if v, err := s.HighWater(context.Background(), "n"); v != 0 || err != nil {
		t.Fatalf("empty water = %d, %v", v, err)
	}
	c.rows = [][]driver.Value{{int64(7)}}
	if v, err := s.HighWater(context.Background(), "n"); v != 7 || err != nil {
		t.Fatalf("water = %d, %v", v, err)
	}
	if !reflect.DeepEqual(c.args, []driver.NamedValue{{Ordinal: 1, Value: "n"}}) || !strings.Contains(c.query, "LIMIT 1") {
		t.Fatal(c.query, c.args)
	}
	c.err = errors.New("database unavailable")
	if _, err := s.QueryBatch(context.Background(), StoreQuery{Network: "n", Limit: 1}); err != ErrStore {
		t.Fatalf("query error = %v", err)
	}
	if _, err := s.HighWater(context.Background(), "n"); err != ErrStore {
		t.Fatalf("water error = %v", err)
	}
}
