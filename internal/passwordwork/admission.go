// Package passwordwork bounds simultaneous memory-hard credential work in one
// process. It does not change password parameters or queue callers.
package passwordwork

import (
	"context"
	"fmt"
	"sync"

	"github.com/djshellshoxxx/circula/internal/domain"
)

// DefaultLimit is an operation-count bound, not a byte budget. Historical
// password hashes may request more memory than the current default parameters.
const DefaultLimit = 2

// Limiter has a fixed capacity. Every service in one process must share the
// same limiter; creating one per request or adapter would multiply the bound.
type Limiter struct {
	slots chan struct{}
}

var process = &Limiter{slots: make(chan struct{}, DefaultLimit)}

// Process returns the process-wide default used by authentication and backups.
// It cannot be reconfigured while requests are running.
func Process() *Limiter { return process }

// New constructs an isolated limiter for deterministic accounting tests. The
// application uses Process for every caller instead of creating local budgets.
func New(limit int) (*Limiter, error) {
	if limit < 1 {
		return nil, fmt.Errorf("password work capacity must be positive")
	}
	return &Limiter{slots: make(chan struct{}, limit)}, nil
}

// Acquire refuses immediately when full: there are no retained waiters. Call
// before identity-dependent reads, durable guess reservations or mutations.
func (l *Limiter) Acquire(ctx context.Context) (*Ticket, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if l == nil || cap(l.slots) == 0 {
		return nil, domain.Errf(domain.CodeInternal, "password work capacity is not configured")
	}
	select {
	case l.slots <- struct{}{}:
		ticket := &Ticket{limiter: l}
		if err := ctx.Err(); err != nil {
			ticket.Release()
			return nil, err
		}
		return ticket, nil
	default:
		return nil, domain.Errf(domain.CodeUnavailable, "password work is busy; try again later")
	}
}

// Ticket admits one operation, including sequential proof and replacement
// hashes. It must not be shared by parallel computations. Release is safe to
// defer, including when a synchronous computation panics.
type Ticket struct {
	mu       sync.Mutex
	limiter  *Limiter
	running  bool
	released bool
}

// Run performs one synchronous hash while retaining the ticket. Cancellation
// cannot interrupt Argon2: capacity remains occupied until work returns. The
// callback runs without the bookkeeping mutex, so test observers may reenter
// a separately admitted operation without deadlocking the limiter.
func (t *Ticket) Run(ctx context.Context, work func() []byte) ([]byte, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if t == nil {
		return nil, domain.Errf(domain.CodeInternal, "password work requires admission")
	}
	t.mu.Lock()
	if t.limiter == nil || t.released || t.running {
		t.mu.Unlock()
		return nil, domain.Errf(domain.CodeInternal, "password work ticket is not available")
	}
	t.running = true
	t.mu.Unlock()
	defer t.finished()
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	result := work()
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return result, nil
}

func (t *Ticket) finished() {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.running = false
	if t.released {
		t.releaseSlot()
	}
}

// Release relinquishes the operation slot exactly once. If another goroutine
// requests release during a hash, the slot stays occupied until that hash ends.
func (t *Ticket) Release() {
	if t == nil {
		return
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	t.released = true
	if !t.running {
		t.releaseSlot()
	}
}

func (t *Ticket) releaseSlot() {
	if t.limiter != nil {
		<-t.limiter.slots
		t.limiter = nil
	}
}
