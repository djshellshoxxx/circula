package passwordwork_test

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/djshellshoxxx/circula/internal/domain"
	"github.com/djshellshoxxx/circula/internal/passwordwork"
)

func newLimit(t *testing.T, capacity int) *passwordwork.Limiter {
	t.Helper()
	l, err := passwordwork.New(capacity)
	if err != nil {
		t.Fatal(err)
	}
	return l
}

func acquire(t *testing.T, l *passwordwork.Limiter) *passwordwork.Ticket {
	t.Helper()
	w, err := l.Acquire(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(w.Release)
	return w
}

func TestAdmissionRefusesWithoutWaiting(t *testing.T) {
	l := newLimit(t, 2)
	first, second := acquire(t, l), acquire(t, l)
	// Only inert tickets occupy capacity: no costly work or request flood.
	ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
	defer cancel()
	if ticket, err := l.Acquire(ctx); ticket != nil || !errors.Is(err, domain.ErrUnavailable) {
		t.Fatalf("full admission = %v, %v; want unavailable before context timeout", ticket, err)
	}
	first.Release()
	first.Release()
	replacement := acquire(t, l)
	if ticket, err := l.Acquire(ctx); ticket != nil || !errors.Is(err, domain.ErrUnavailable) {
		t.Fatalf("repeated release freed another operation's capacity: %v, %v", ticket, err)
	}
	replacement.Release()
	second.Release()
	acquire(t, l)
	acquire(t, l)
}

func TestAdmissionRejectsCanceledContextWithoutCapacityLeak(t *testing.T) {
	l := newLimit(t, 1)
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	if ticket, err := l.Acquire(ctx); ticket != nil || !errors.Is(err, context.Canceled) {
		t.Fatalf("canceled admission = %v, %v", ticket, err)
	}
	acquire(t, l)
}

func TestTicketRetainsCapacityUntilCanceledWorkReturns(t *testing.T) {
	l := newLimit(t, 1)
	work := acquire(t, l)
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	started, resume := make(chan struct{}), make(chan struct{})
	unblock := sync.OnceFunc(func() { close(resume) })
	defer unblock()
	done := make(chan error, 1)
	go func() {
		_, err := work.Run(ctx, func() []byte {
			close(started)
			<-resume
			return []byte("inert result")
		})
		done <- err
	}()
	select {
	case <-started:
	case <-time.After(5 * time.Second):
		t.Fatal("inert work did not start")
	}
	cancel()
	work.Release() // even an early release must not free running work
	if ticket, err := l.Acquire(t.Context()); ticket != nil || !errors.Is(err, domain.ErrUnavailable) {
		t.Fatalf("canceled synchronous work lost its capacity: %v, %v", ticket, err)
	}
	select {
	case err := <-done:
		t.Fatalf("Run returned before synchronous work completed: %v", err)
	default:
	}
	unblock()
	select {
	case err := <-done:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("completed canceled work: %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("Run did not return after inert work completed")
	}
	acquire(t, l)
}

func TestTicketCoversSequentialWorkAndStopsAfterCancellation(t *testing.T) {
	l := newLimit(t, 1)
	work := acquire(t, l)
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	calls := 0
	for range 2 {
		if _, err := work.Run(ctx, func() []byte { calls++; return nil }); err != nil {
			t.Fatal(err)
		}
	}
	if ticket, err := l.Acquire(ctx); ticket != nil || !errors.Is(err, domain.ErrUnavailable) {
		t.Fatalf("sequential hashes released the operation ticket: %v, %v", ticket, err)
	}
	cancel()
	if _, err := work.Run(ctx, func() []byte { calls++; return nil }); !errors.Is(err, context.Canceled) || calls != 2 {
		t.Fatalf("canceled work ran: calls=%d err=%v", calls, err)
	}
	work.Release()
	acquire(t, l)
}

func TestTicketDoesNotHoldBookkeepingLockAcrossWork(t *testing.T) {
	l := newLimit(t, 2)
	outer := acquire(t, l)
	_, err := outer.Run(t.Context(), func() []byte {
		// Reentry needs its own ordinary ticket; there is no special bypass.
		inner := acquire(t, l)
		defer inner.Release()
		if _, err := inner.Run(t.Context(), func() []byte { return nil }); err != nil {
			t.Error(err)
		}
		if _, err := outer.Run(t.Context(), func() []byte { t.Error("same ticket ran concurrently"); return nil }); domain.CodeOf(err) != domain.CodeInternal {
			t.Errorf("running ticket reused: %v", err)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestTicketReleasesAfterPanic(t *testing.T) {
	l := newLimit(t, 1)
	func() {
		defer func() {
			if r := recover(); r != "inert panic" {
				t.Errorf("unexpected panic: %v", r)
			}
		}()
		work := acquire(t, l)
		defer work.Release()
		_, _ = work.Run(t.Context(), func() []byte { panic("inert panic") })
	}()
	acquire(t, l)
}

func TestAdmissionRequiresConfiguredLiveTicket(t *testing.T) {
	for _, n := range []int{-1, 0} {
		if _, err := passwordwork.New(n); err == nil {
			t.Errorf("capacity %d accepted", n)
		}
	}
	for _, l := range []*passwordwork.Limiter{nil, {}} {
		if ticket, err := l.Acquire(t.Context()); ticket != nil || domain.CodeOf(err) != domain.CodeInternal {
			t.Errorf("unconfigured limiter: %v, %v", ticket, err)
		}
	}
	released := acquire(t, newLimit(t, 1))
	released.Release()
	for _, work := range []*passwordwork.Ticket{nil, {}, released} {
		if _, err := work.Run(t.Context(), func() []byte { t.Error("invalid ticket started work"); return nil }); domain.CodeOf(err) != domain.CodeInternal {
			t.Errorf("invalid ticket error: %v", err)
		}
		work.Release()
	}
}

func TestProcessDefaultIsSharedAndFixed(t *testing.T) {
	first, second := passwordwork.Process(), passwordwork.Process()
	if first != second || passwordwork.DefaultLimit != 2 {
		t.Fatal("process admission must retain its documented shared two-operation default")
	}
	for range passwordwork.DefaultLimit {
		acquire(t, passwordwork.Process())
	}
	if _, err := passwordwork.Process().Acquire(t.Context()); !errors.Is(err, domain.ErrUnavailable) {
		t.Fatalf("process limit exceeded: %v", err)
	}
}
