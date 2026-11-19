package service_test

import (
	"context"
	"errors"
	"testing"

	"github.com/djshellshoxxx/circula/internal/store"
	tu "github.com/djshellshoxxx/circula/internal/testutil"
)

type operationalStore struct {
	store.Store
	check func(context.Context) error
}

func (s operationalStore) CheckStorage(ctx context.Context) error { return s.check(ctx) }

// ARC-004: operational verification uses only the optional repository
// capability. This wrapper deliberately exposes no DB handle or SQL type.
func TestCheckStorageUsesDriverNeutralCapability(t *testing.T) {
	calls := 0
	var checkErr error
	e := tu.NewEnvWith(t, func(st store.Store) store.Store {
		return operationalStore{st, func(ctx context.Context) error {
			calls++
			if err := ctx.Err(); err != nil {
				return err
			}
			return checkErr
		}}
	})
	if err := e.Svc.CheckStorage(e.Ctx); err != nil || calls != 1 {
		t.Fatalf("neutral checker calls=%d err=%v", calls, err)
	}
	checkErr = errors.New("driver storage settings unsafe")
	if err := e.Svc.CheckStorage(e.Ctx); !errors.Is(err, checkErr) || calls != 2 {
		t.Fatalf("driver refusal lost: calls=%d err=%v", calls, err)
	}
	ctx, cancel := context.WithCancel(e.Ctx)
	cancel()
	if err := e.Svc.CheckStorage(ctx); !errors.Is(err, context.Canceled) {
		t.Fatalf("checker context lost: %v", err)
	}
}
