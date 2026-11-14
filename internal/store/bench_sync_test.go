package store_test

import (
	"context"
	"fmt"
	"path/filepath"
	"testing"
	"time"

	"github.com/djshellshoxxx/circula/internal/domain"
	"github.com/djshellshoxxx/circula/internal/store"
)

// BenchmarkPostingSynchronous measures a ledger-write workload (propose a
// trade, post both entries and complete it in one transaction) under PRAGMA
// synchronous NORMAL and FULL in WAL mode (SA2-20). Run:
//
//	go test ./internal/store -run '^$' -bench PostingSynchronous -benchtime 2000x
//
// Results are recorded in docs/ops/configuration.md.
func BenchmarkPostingSynchronous(b *testing.B) {
	for _, level := range []string{store.SyncNormal, store.SyncFull} {
		b.Run(level, func(b *testing.B) {
			ctx := context.Background()
			s, err := store.OpenWith(ctx, filepath.Join(b.TempDir(), "c.db"), store.Options{Synchronous: level})
			if err != nil {
				b.Fatal(err)
			}
			defer func() { _ = s.Close() }()
			_, _, accA, accB := seed(b, s)
			now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				err := s.Update(ctx, func(tx store.Tx) error {
					id := fmt.Sprintf("bt%d", i)
					exp := now.Add(time.Hour)
					if err := tx.InsertTransaction(&domain.Transaction{ID: id, NetworkID: "n1", Kind: domain.KindTrade,
						PayerAccountID: accA, PayeeAccountID: accB, Amount: 100, CreatedBy: "mp1", ProposedAt: now, ExpiresAt: &exp}); err != nil {
						return err
					}
					for j, e := range []struct {
						acc string
						amt int64
					}{{accA, -100}, {accB, 100}} {
						if err := tx.InsertEntry(&domain.Entry{ID: fmt.Sprintf("be%d-%d", i, j), NetworkID: "n1", TxID: id, AccountID: e.acc, Amount: e.amt, LedgerSeq: int64(i + 10), PostedAt: now}); err != nil {
							return err
						}
					}
					x, err := tx.GetTransaction("n1", id)
					if err != nil {
						return err
					}
					x.Status = domain.TxCompleted
					return tx.UpdateTransaction(x)
				})
				if err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

// SA2-20: the synchronous level is applied to every pooled connection and an
// unknown level is refused.
func TestSynchronousOption(t *testing.T) {
	ctx := context.Background()
	for level, want := range map[string]int{"": 1, "normal": 1, "FULL": 2, "full": 2} {
		s, err := store.OpenWith(ctx, filepath.Join(t.TempDir(), "c.db"), store.Options{Synchronous: level})
		if err != nil {
			t.Fatal(err)
		}
		var got int
		if err := s.DB().QueryRow(`PRAGMA synchronous`).Scan(&got); err != nil || got != want {
			t.Errorf("level %q: PRAGMA synchronous = %d (err %v), want %d", level, got, err, want)
		}
		_ = s.Close()
	}
	if _, err := store.OpenWith(ctx, filepath.Join(t.TempDir(), "c.db"), store.Options{Synchronous: "OFF"}); err == nil {
		t.Error("synchronous=OFF accepted")
	}
}
