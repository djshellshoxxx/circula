package store_test

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/djshellshoxxx/circula/internal/store"
	"github.com/djshellshoxxx/circula/internal/storetest"
	"github.com/djshellshoxxx/circula/internal/testutil"
)

// ARC-004: the repository contract suite, written against store.Store and
// store.Tx only, holds for the SQLite store.
func TestRepositoryContractSQLite(t *testing.T) {
	storetest.Run(t, func(t *testing.T) store.Store { return openStore(t) })
}

// A pass-through wrapper (the fault injector with nothing armed) keeps the
// contract: the suite is not tied to the concrete *store.SQLite type.
func TestRepositoryContractWrapped(t *testing.T) {
	storetest.Run(t, func(t *testing.T) store.Store {
		s, err := store.Open(context.Background(), filepath.Join(t.TempDir(), "w.db"))
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { _ = s.Close() })
		return testutil.NewFaultStore(s)
	})
}
