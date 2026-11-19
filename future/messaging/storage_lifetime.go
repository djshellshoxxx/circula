package messaging

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"io"
	"sync"

	"modernc.org/sqlite"
)

// These are all database/sql/driver connection interfaces implemented by the
// pinned SQLite driver. Embedding them preserves context handling, transaction
// options, session reset, validation and both legacy and context query paths.
// A different connection implementation fails closed instead of silently
// changing the pool's behavior by dropping an optional interface.
type storageDriverConnection interface {
	driver.Conn
	driver.ConnBeginTx
	driver.ConnPrepareContext
	driver.Execer
	driver.ExecerContext
	driver.Queryer
	driver.QueryerContext
	driver.Pinger
	driver.SessionResetter
	driver.Validator
}

var errStorageClosing = errors.New("messaging storage is closing")

// storageOwner tracks physical driver calls rather than sql.DB statistics.
// database/sql may return from Close while a canceled transaction is still
// rolling back, and it can discard an asynchronous connection-close error.
type storageOwner struct {
	guard       io.Closer
	mu          sync.Mutex
	changed     *sync.Cond
	closing     bool
	connecting  int
	connections int
	failure     error
	closeOnce   sync.Once
	closeErr    error
}

type storageConnector struct {
	driver.Connector
	owner *storageOwner
}

type storageConnection struct {
	storageDriverConnection
	owner     *storageOwner
	closeOnce sync.Once
	closeErr  error
}

func newOwnedStorageDatabase(base driver.Connector, guard io.Closer) (*sql.DB, *storageOwner) {
	owner := &storageOwner{guard: guard}
	owner.changed = sync.NewCond(&owner.mu)
	return sql.OpenDB(&storageConnector{Connector: base, owner: owner}), owner
}

func openStorageDatabase(dsn string, guard io.Closer) (*sql.DB, io.Closer, error) {
	if guard == nil {
		// Preserve the existing Unix connection path and its registered driver.
		db, err := sql.Open("sqlite", dsn)
		return db, nil, err
	}
	// NewConnector uses the same registered driver, including host-registered
	// functions, collations and hooks; it opens no physical connection itself.
	base, err := sqlite.NewConnector(dsn)
	if err != nil {
		return nil, nil, errors.Join(err, guard.Close())
	}
	db, owner := newOwnedStorageDatabase(base, guard)
	return db, owner, nil
}

func (c *storageConnector) Connect(ctx context.Context) (driver.Conn, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	o := c.owner
	o.mu.Lock()
	if o.closing {
		o.mu.Unlock()
		return nil, errStorageClosing
	}
	o.connecting++
	o.mu.Unlock()

	conn, err := c.Connector.Connect(ctx)
	if err != nil {
		if conn != nil {
			err = errors.Join(err, conn.Close())
		}
		// A failed driver Open can perform internal cleanup whose result it
		// does not expose. Retain the guard when release cannot be established.
		o.finishConnect(err)
		return nil, err
	}
	compatible, ok := conn.(storageDriverConnection)
	if !ok {
		err = errors.New("messaging SQLite connection interfaces changed")
		if conn != nil {
			err = errors.Join(err, conn.Close())
		}
		o.finishConnect(err)
		return nil, err
	}
	o.mu.Lock()
	if o.closing {
		o.mu.Unlock()
		err = conn.Close()
		o.finishConnect(err)
		return nil, errors.Join(errStorageClosing, err)
	}
	o.connecting--
	o.connections++
	o.changed.Broadcast()
	o.mu.Unlock()
	return &storageConnection{storageDriverConnection: compatible, owner: o}, nil
}

func (o *storageOwner) finishConnect(err error) {
	o.mu.Lock()
	o.connecting--
	o.failure = errors.Join(o.failure, err)
	o.changed.Broadcast()
	o.mu.Unlock()
}

func (c *storageConnection) Close() error {
	c.closeOnce.Do(func() {
		c.closeErr = c.storageDriverConnection.Close()
		c.owner.mu.Lock()
		c.owner.connections--
		c.owner.failure = errors.Join(c.owner.failure, c.closeErr)
		c.owner.changed.Broadcast()
		c.owner.mu.Unlock()
	})
	return c.closeErr
}

func (o *storageOwner) startClosing() {
	o.mu.Lock()
	o.closing = true
	o.mu.Unlock()
}

func (o *storageOwner) Close() error {
	o.closeOnce.Do(func() {
		o.mu.Lock()
		o.closing = true
		for o.connecting != 0 || o.connections != 0 {
			o.changed.Wait()
		}
		o.closeErr = o.failure
		o.mu.Unlock()
		if o.closeErr == nil {
			o.closeErr = o.guard.Close()
		}
	})
	return o.closeErr
}

func closeStorageDatabase(db *sql.DB, storage io.Closer) error {
	owner, owned := storage.(*storageOwner)
	if owned {
		owner.startClosing()
	}
	err := db.Close()
	if owned {
		owner.mu.Lock()
		owner.failure = errors.Join(owner.failure, err)
		owner.mu.Unlock()
		return owner.Close()
	}
	if err != nil || storage == nil {
		// Do not release even an injected/non-driver guard after a known error.
		return err
	}
	return storage.Close()
}

func closeFailedStorageOpen(db *sql.DB, storage io.Closer, openErr error) error {
	closeErr := closeStorageDatabase(db, storage)
	if storage == nil {
		// Unix Open historically returns the initialization error unchanged.
		return openErr
	}
	return errors.Join(openErr, closeErr)
}
