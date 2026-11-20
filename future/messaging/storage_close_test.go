package messaging

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"io"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

type storageCloseConnector struct{ conn *storageCloseConn }

func (c storageCloseConnector) Connect(context.Context) (driver.Conn, error) { return c.conn, nil }
func (storageCloseConnector) Driver() driver.Driver                          { return storageCloseDriver{} }

type storageCloseDriver struct{}

func (storageCloseDriver) Open(string) (driver.Conn, error) {
	return nil, errors.New("fixture uses its connector")
}

type storageCloseConn struct {
	entered  chan struct{}
	release  chan struct{}
	finished chan struct{}
	once     sync.Once
}

func (*storageCloseConn) Prepare(string) (driver.Stmt, error) {
	return nil, errors.New("unused fixture statement")
}
func (*storageCloseConn) Begin() (driver.Tx, error) {
	return nil, errors.New("unused fixture transaction")
}
func (c *storageCloseConn) Close() error {
	c.once.Do(func() {
		close(c.entered)
		<-c.release
		close(c.finished)
	})
	return nil
}

type storageCloseGuard struct {
	driverFinished <-chan struct{}
	closed         chan struct{}
	once           sync.Once
	premature      atomic.Bool
}

func (g *storageCloseGuard) Close() error {
	g.once.Do(func() {
		select {
		case <-g.driverFinished:
		default:
			g.premature.Store(true)
		}
		close(g.closed)
	})
	return nil
}

func TestStoreCloseKeepsStorageGuardUntilDriverCloseFinishes(t *testing.T) {
	conn := &storageCloseConn{entered: make(chan struct{}), release: make(chan struct{}), finished: make(chan struct{})}
	db := sql.OpenDB(storageCloseConnector{conn})
	defer db.Close()
	var releaseOnce sync.Once
	release := func() { releaseOnce.Do(func() { close(conn.release) }) }
	defer release()
	active, err := db.Conn(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	// Return the physical connection to the idle pool so DB.Close owns its
	// blocking driver close, rather than an unrelated active-query lifecycle.
	if err = active.Close(); err != nil {
		t.Fatal(err)
	}
	guard := &storageCloseGuard{driverFinished: conn.finished, closed: make(chan struct{})}
	s := &Store{db: db, storage: guard}
	results := make(chan error, 2)
	go func() { results <- s.Close() }()
	select {
	case <-conn.entered:
	case <-time.After(5 * time.Second):
		t.Fatal("first close did not reach the driver")
	}
	secondStarted := make(chan struct{})
	go func() {
		close(secondStarted)
		results <- s.Close()
	}()
	<-secondStarted
	select {
	case <-guard.closed:
	case <-time.After(100 * time.Millisecond):
	}
	release()
	for i := 0; i < 2; i++ {
		select {
		case err := <-results:
			if err != nil {
				t.Fatal(err)
			}
		case <-time.After(5 * time.Second):
			t.Fatal("ordered close did not finish")
		}
	}
	if guard.premature.Load() {
		t.Fatal("storage guard closed before the SQLite driver close finished")
	}
	select {
	case <-guard.closed:
	default:
		t.Fatal("storage guard was not released after the driver")
	}
}

type transactionStorageGuard struct {
	db       *sql.DB
	delegate io.Closer
	closed   chan struct{}
	once     sync.Once
	active   atomic.Bool
	err      error
}

func (g *transactionStorageGuard) Close() error {
	g.once.Do(func() {
		g.active.Store(g.db.Stats().InUse != 0)
		if g.delegate != nil {
			g.err = g.delegate.Close()
		}
		close(g.closed)
	})
	return g.err
}

func TestStoreCloseKeepsStorageGuardUntilAuthorizedTransactionFinishes(t *testing.T) {
	f := setup(t)
	guard := &transactionStorageGuard{db: f.s.db, delegate: f.s.storage, closed: make(chan struct{})}
	f.s.storage = guard
	entered, release := make(chan struct{}), make(chan struct{})
	var releaseOnce, callbackOnce sync.Once
	releaseTransaction := func() { releaseOnce.Do(func() { close(release) }) }
	defer releaseTransaction()
	authorize := f.s.authorize
	f.s.authorize = func(ctx context.Context, tx *sql.Tx, actor Actor, action, target string) error {
		callbackOnce.Do(func() {
			close(entered)
			<-release
		})
		return authorize(ctx, tx, actor, action, target)
	}
	created := make(chan error, 1)
	go func() {
		_, err := f.s.Create(ctx, alice, "bob", "listing-1", "close-active-tx", 30)
		created <- err
	}()
	select {
	case <-entered:
	case <-time.After(5 * time.Second):
		t.Fatal("Create did not reach its transaction-bound authorizer")
	}
	if f.s.db.Stats().InUse != 1 {
		t.Fatal("fixture did not hold the real SQLite connection")
	}
	closing := make(chan error, 1)
	go func() { closing <- f.s.Close() }()
	premature := false
	select {
	case <-guard.closed:
		premature = true
	case <-time.After(100 * time.Millisecond):
	}
	releaseTransaction()
	for _, result := range []<-chan error{created, closing} {
		select {
		case err := <-result:
			if err != nil {
				t.Fatal(err)
			}
		case <-time.After(5 * time.Second):
			t.Fatal("transaction/close did not finish")
		}
	}
	if premature || guard.active.Load() {
		t.Fatal("storage guard closed while an authorized SQLite transaction was active")
	}
	select {
	case <-guard.closed:
	default:
		t.Fatal("storage guard not released after transaction completion")
	}
}

type storageRollbackConnector struct{ conn *storageRollbackConn }

func (c storageRollbackConnector) Connect(context.Context) (driver.Conn, error) { return c.conn, nil }
func (storageRollbackConnector) Driver() driver.Driver                          { return storageCloseDriver{} }

type storageRollbackConn struct {
	rollbackEntered  chan struct{}
	rollbackRelease  chan struct{}
	connectionClosed chan struct{}
	closeOnce        sync.Once
	closeErr         error
}

func (*storageRollbackConn) Prepare(string) (driver.Stmt, error) {
	return nil, errors.New("unused fixture statement")
}
func (c *storageRollbackConn) Begin() (driver.Tx, error) { return storageRollbackTx{c}, nil }
func (c *storageRollbackConn) BeginTx(context.Context, driver.TxOptions) (driver.Tx, error) {
	return c.Begin()
}
func (c *storageRollbackConn) PrepareContext(_ context.Context, query string) (driver.Stmt, error) {
	return c.Prepare(query)
}
func (*storageRollbackConn) Exec(string, []driver.Value) (driver.Result, error) {
	return driver.RowsAffected(0), nil
}
func (*storageRollbackConn) ExecContext(context.Context, string, []driver.NamedValue) (driver.Result, error) {
	return driver.RowsAffected(0), nil
}
func (*storageRollbackConn) Query(string, []driver.Value) (driver.Rows, error) {
	return nil, errors.New("unused fixture query")
}
func (*storageRollbackConn) QueryContext(context.Context, string, []driver.NamedValue) (driver.Rows, error) {
	return nil, errors.New("unused fixture query")
}
func (*storageRollbackConn) Ping(context.Context) error { return nil }

// Match the pinned SQLite driver's session reset/validation support: canceled
// transactions return their connection only after the asynchronous rollback.
func (*storageRollbackConn) ResetSession(context.Context) error { return nil }
func (*storageRollbackConn) IsValid() bool                      { return true }
func (c *storageRollbackConn) Close() error {
	c.closeOnce.Do(func() { close(c.connectionClosed) })
	return c.closeErr
}

type storageRollbackTx struct{ conn *storageRollbackConn }

func (storageRollbackTx) Commit() error { return errors.New("fixture transaction must roll back") }
func (tx storageRollbackTx) Rollback() error {
	close(tx.conn.rollbackEntered)
	<-tx.conn.rollbackRelease
	return nil
}

func TestStoreCloseKeepsStorageGuardUntilCanceledRollbackFinishes(t *testing.T) {
	t.Run("successful close", func(t *testing.T) { testCanceledStorageRollback(t, nil) })
	t.Run("asynchronous close failure", func(t *testing.T) {
		testCanceledStorageRollback(t, errors.New("synthetic driver close failure"))
	})
}

func testCanceledStorageRollback(t *testing.T, closeErr error) {
	t.Helper()
	conn := &storageRollbackConn{
		rollbackEntered: make(chan struct{}), rollbackRelease: make(chan struct{}), connectionClosed: make(chan struct{}),
		closeErr: closeErr,
	}
	var releaseOnce sync.Once
	release := func() { releaseOnce.Do(func() { close(conn.rollbackRelease) }) }
	defer release()
	guard := &storageCloseGuard{driverFinished: conn.connectionClosed, closed: make(chan struct{})}
	db, storage := newOwnedStorageDatabase(storageRollbackConnector{conn}, guard)
	db.SetMaxOpenConns(1)
	defer func() {
		release()
		closeStorageDatabase(db, storage)
	}()
	request, cancel := context.WithCancel(context.Background())
	defer cancel()
	s := &Store{db: db, storage: storage, now: time.Now}
	s.authorize = func(context.Context, *sql.Tx, Actor, string, string) error {
		cancel()
		select {
		case <-conn.rollbackEntered:
			// awaitDone owns the rollback before unit's deferred Rollback can
			// observe ErrTxDone; no scheduling delay is needed for this race.
			return context.Canceled
		case <-time.After(5 * time.Second):
			return errors.New("asynchronous rollback did not start")
		}
	}
	created := make(chan error, 1)
	go func() {
		_, err := s.Create(request, alice, "bob", "listing-1", "close-canceled-tx", 30)
		created <- err
	}()
	select {
	case <-conn.rollbackEntered:
	case <-time.After(5 * time.Second):
		t.Fatal("Create did not start its context-canceled driver rollback")
	}
	closing := make(chan error, 1)
	go func() { closing <- s.Close() }()
	select {
	case <-guard.closed:
	case <-time.After(100 * time.Millisecond):
	}
	release()
	select {
	case err := <-created:
		if !errors.Is(err, ErrDenied) {
			t.Fatalf("canceled authorization = %v, want ErrDenied", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("canceled Create did not finish")
	}
	select {
	case err := <-closing:
		if !errors.Is(err, closeErr) {
			t.Fatalf("Close = %v, want %v", err, closeErr)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("close did not finish after rollback release")
	}
	if guard.premature.Load() {
		t.Fatal("storage guard closed before the canceled transaction's driver rollback and connection close finished")
	}
	if closeErr != nil {
		select {
		case <-guard.closed:
			t.Fatal("storage guard released after asynchronous driver close failure")
		default:
		}
		if err := s.Close(); !errors.Is(err, closeErr) {
			t.Fatalf("repeated Close lost driver failure: %v", err)
		}
		return
	}
	select {
	case <-guard.closed:
	default:
		t.Fatal("storage guard not released after canceled rollback and driver close")
	}
}
