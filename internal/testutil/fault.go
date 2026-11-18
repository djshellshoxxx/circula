package testutil

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/djshellshoxxx/circula/internal/domain"
	"github.com/djshellshoxxx/circula/internal/store"
)

// ErrInjected is returned by a faulty operation.
var ErrInjected = errors.New("testutil: injected fault")

// FaultStore wraps a Store so that the Nth call (1-based) of a repository
// method, counted per unit of work, fails with ErrInjected. Counting resets at
// the start of each Update.
type FaultStore struct {
	store.Store
	mu     sync.Mutex
	failOn map[string]int
}

// NewFaultStore wraps s with no faults armed.
func NewFaultStore(s store.Store) *FaultStore { return &FaultStore{Store: s, failOn: map[string]int{}} }

// Arm makes the nth call of op fail; Disarm clears all faults.
func (f *FaultStore) Arm(op string, nth int) { f.mu.Lock(); f.failOn[op] = nth; f.mu.Unlock() }

// Disarm clears all faults.
func (f *FaultStore) Disarm() { f.mu.Lock(); f.failOn = map[string]int{}; f.mu.Unlock() }

func (f *FaultStore) armed() map[string]int {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := make(map[string]int, len(f.failOn))
	for k, v := range f.failOn {
		out[k] = v
	}
	return out
}

// Update implements store.Store with fault injection.
func (f *FaultStore) Update(ctx context.Context, fn func(store.Tx) error) error {
	return f.Store.Update(ctx, func(tx store.Tx) error {
		return fn(&faultTx{Tx: tx, fail: f.armed(), n: map[string]int{}})
	})
}

type faultTx struct {
	store.Tx
	fail map[string]int
	n    map[string]int
}

func (t *faultTx) hit(op string) error {
	t.n[op]++
	if want, ok := t.fail[op]; ok && t.n[op] == want {
		return ErrInjected
	}
	return nil
}

// The wrapped methods below are the write steps of a posting unit; any of them
// failing must leave no trace (LED-4, LED-6).

func (t *faultTx) InsertEntry(e *domain.Entry) error {
	if err := t.hit("InsertEntry"); err != nil {
		return err
	}
	return t.Tx.InsertEntry(e)
}

func (t *faultTx) AddBalance(netID, id string, delta int64) error {
	if err := t.hit("AddBalance"); err != nil {
		return err
	}
	return t.Tx.AddBalance(netID, id, delta)
}

func (t *faultTx) UpdateTransaction(x *domain.Transaction) error {
	if err := t.hit("UpdateTransaction"); err != nil {
		return err
	}
	return t.Tx.UpdateTransaction(x)
}

func (t *faultTx) InsertStatusEvent(e *domain.StatusEvent) error {
	if err := t.hit("InsertStatusEvent"); err != nil {
		return err
	}
	return t.Tx.InsertStatusEvent(e)
}

func (t *faultTx) InsertAudit(e *domain.AuditEvent) error {
	if err := t.hit("InsertAudit"); err != nil {
		return err
	}
	return t.Tx.InsertAudit(e)
}

func (t *faultTx) PutIdem(r store.IdemRecord) error {
	if err := t.hit("PutIdem"); err != nil {
		return err
	}
	return t.Tx.PutIdem(r)
}

// Activity signal steps (SEC-026): a failure here must not affect the trade
// whose evaluation it belongs to. InsertNotification is counted per unit, so
// arming its Nth call selects one notice among those a unit writes.

func (t *faultTx) CountProposalsSince(netID, memberID string, since time.Time) (int, error) {
	if err := t.hit("CountProposalsSince"); err != nil {
		return 0, err
	}
	return t.Tx.CountProposalsSince(netID, memberID, since)
}

func (t *faultTx) CountCompletedTradesBetween(netID, a, b string, since time.Time) (int, int, error) {
	if err := t.hit("CountCompletedTradesBetween"); err != nil {
		return 0, 0, err
	}
	return t.Tx.CountCompletedTradesBetween(netID, a, b, since)
}

func (t *faultTx) InsertActivitySignal(s store.ActivitySignal) (bool, error) {
	if err := t.hit("InsertActivitySignal"); err != nil {
		return false, err
	}
	return t.Tx.InsertActivitySignal(s)
}

func (t *faultTx) InsertNotification(n store.Notification) (bool, error) {
	if err := t.hit("InsertNotification"); err != nil {
		return false, err
	}
	return t.Tx.InsertNotification(n)
}
