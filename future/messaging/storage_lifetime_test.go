package messaging

import (
	"context"
	"database/sql/driver"
	"errors"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"modernc.org/sqlite"
)

type storageConnectorFunc func(context.Context) (driver.Conn, error)

func (f storageConnectorFunc) Connect(ctx context.Context) (driver.Conn, error) { return f(ctx) }
func (storageConnectorFunc) Driver() driver.Driver                              { return storageCloseDriver{} }

type storageReleaseRecorder struct {
	closed chan struct{}
	once   sync.Once
}

func (g *storageReleaseRecorder) Close() error {
	g.once.Do(func() { close(g.closed) })
	return nil
}

type storageLateCloseConn struct {
	*storageRollbackConn
	entered chan struct{}
	release chan struct{}
}

func (c *storageLateCloseConn) Close() error {
	close(c.entered)
	<-c.release
	return c.storageRollbackConn.Close()
}

func TestOwnedStorageWaitsForInFlightConnectAndClose(t *testing.T) {
	for _, failClose := range []bool{false, true} {
		name := "successful close"
		var closeErr error
		if failClose {
			name = "late close failure"
			closeErr = errors.New("synthetic late driver close failure")
		}
		t.Run(name, func(t *testing.T) {
			conn := &storageLateCloseConn{
				storageRollbackConn: &storageRollbackConn{connectionClosed: make(chan struct{}), closeErr: closeErr},
				entered:             make(chan struct{}), release: make(chan struct{}),
			}
			connectEntered, connectRelease := make(chan struct{}), make(chan struct{})
			var connectOnce, closeOnce sync.Once
			releaseConnect := func() { connectOnce.Do(func() { close(connectRelease) }) }
			releaseClose := func() { closeOnce.Do(func() { close(conn.release) }) }
			var calls atomic.Int32
			base := storageConnectorFunc(func(context.Context) (driver.Conn, error) {
				calls.Add(1)
				close(connectEntered)
				<-connectRelease
				return conn, nil
			})
			guard := &storageCloseGuard{driverFinished: conn.connectionClosed, closed: make(chan struct{})}
			db, owner := newOwnedStorageDatabase(base, guard)
			db.SetMaxOpenConns(1)
			defer func() {
				releaseConnect()
				releaseClose()
				closeStorageDatabase(db, owner)
			}()
			s := &Store{db: db, storage: owner}
			opening := make(chan error, 1)
			go func() { opening <- db.PingContext(context.Background()) }()
			select {
			case <-connectEntered:
			case <-time.After(5 * time.Second):
				t.Fatal("physical connection attempt did not start")
			}
			closing := make(chan error, 1)
			go func() { closing <- s.Close() }()
			// With the sole connection opening, another pool request can finish
			// only when Close rejects it. This also observes the entry gate.
			request, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			if err := db.PingContext(request); err == nil || errors.Is(err, context.DeadlineExceeded) {
				t.Fatalf("pool did not begin closing: %v", err)
			}
			wrapped := &storageConnector{Connector: base, owner: owner}
			if c, err := wrapped.Connect(context.Background()); c != nil || !errors.Is(err, errStorageClosing) {
				t.Fatalf("new connection was not rejected: %v, %v", c, err)
			}
			if calls.Load() != 1 {
				t.Fatal("closing storage invoked the underlying connector again")
			}
			select {
			case <-guard.closed:
				t.Fatal("guard released while a physical connection was opening")
			default:
			}
			releaseConnect()
			select {
			case <-conn.entered:
			case <-time.After(5 * time.Second):
				t.Fatal("late connection was not closed")
			}
			select {
			case <-guard.closed:
				t.Fatal("guard released before the late driver's Close returned")
			default:
			}
			releaseClose()
			select {
			case err := <-opening:
				if !errors.Is(err, errStorageClosing) {
					t.Fatalf("late connection result = %v", err)
				}
			case <-time.After(5 * time.Second):
				t.Fatal("connection attempt did not finish")
			}
			select {
			case err := <-closing:
				if !errors.Is(err, closeErr) {
					t.Fatalf("Close = %v, want %v", err, closeErr)
				}
			case <-time.After(5 * time.Second):
				t.Fatal("Close did not finish after the physical close")
			}
			select {
			case <-guard.closed:
				if closeErr != nil {
					t.Fatal("guard released despite a late physical-close failure")
				}
			default:
				if closeErr == nil {
					t.Fatal("guard retained after successful physical close")
				}
			}
			if guard.premature.Load() {
				t.Fatal("guard release preceded physical close completion")
			}
		})
	}
}

func TestOwnedStorageFailedConnectRetainsGuard(t *testing.T) {
	openErr := errors.New("synthetic driver open with unreported cleanup")
	base := storageConnectorFunc(func(context.Context) (driver.Conn, error) { return nil, openErr })
	guard := &storageReleaseRecorder{closed: make(chan struct{})}
	db, owner := newOwnedStorageDatabase(base, guard)
	if err := db.PingContext(context.Background()); !errors.Is(err, openErr) {
		t.Fatalf("physical open result = %v", err)
	}
	if err := closeFailedStorageOpen(db, owner, openErr); !errors.Is(err, openErr) {
		t.Fatalf("initialization cleanup lost the failure: %v", err)
	}
	select {
	case <-guard.closed:
		t.Fatal("guard released when failed-open cleanup could not be established")
	default:
	}
}

func TestOwnedStorageUsesPinnedSQLiteConnector(t *testing.T) {
	path := filepath.Join(privateFixtureDirectory(t), "tracked.sqlite")
	guard := &storageReleaseRecorder{closed: make(chan struct{})}
	db, owner, err := openStorageDatabase(privateDatabaseDSN(path), guard)
	if err != nil {
		t.Fatal(err)
	}
	defer closeStorageDatabase(db, owner)
	base, err := sqlite.NewConnector(privateDatabaseDSN(path))
	if err != nil || db.Driver() != base.Driver() {
		t.Fatalf("owned storage changed the registered driver: %v", err)
	}
	db.SetMaxOpenConns(1)
	if err = db.PingContext(context.Background()); err != nil {
		t.Fatal(err)
	}
	if _, err = db.ExecContext(context.Background(), "CREATE TABLE owned_lifetime(value INTEGER)"); err != nil {
		t.Fatal(err)
	}
	tx, err := db.BeginTx(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = tx.ExecContext(context.Background(), "INSERT INTO owned_lifetime VALUES (?)", 42); err != nil {
		tx.Rollback()
		t.Fatal(err)
	}
	if err = tx.Commit(); err != nil {
		t.Fatal(err)
	}
	stmt, err := db.PrepareContext(context.Background(), "SELECT value FROM owned_lifetime")
	if err != nil {
		t.Fatal(err)
	}
	var value int
	err = stmt.QueryRowContext(context.Background()).Scan(&value)
	closeErr := stmt.Close()
	if err != nil || closeErr != nil || value != 42 {
		t.Fatalf("prepared query = %d, %v, %v", value, err, closeErr)
	}
	db.SetMaxIdleConns(0)
	for i := 0; i < 2; i++ {
		if err = db.QueryRowContext(context.Background(), "SELECT value FROM owned_lifetime").Scan(&value); err != nil || value != 42 {
			t.Fatalf("replacement connection = %d, %v", value, err)
		}
	}
	select {
	case <-guard.closed:
		t.Fatal("guard released between replacement physical connections")
	default:
	}
	if err = closeStorageDatabase(db, owner); err != nil {
		t.Fatal(err)
	}
	select {
	case <-guard.closed:
	default:
		t.Fatal("guard not released after successful real-driver cleanup")
	}
}
