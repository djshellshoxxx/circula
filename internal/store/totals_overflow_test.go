package store_test

import (
	"context"
	"fmt"
	"math"
	"testing"
	"time"

	"github.com/djshellshoxxx/circula/internal/domain"
	"github.com/djshellshoxxx/circula/internal/service"
	"github.com/djshellshoxxx/circula/internal/store"
)

func aggregateAccounts(t *testing.T, st *store.SQLite, netID string, count int) []string {
	t.Helper()
	ids := make([]string, count)
	now := time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC)
	err := st.Update(context.Background(), func(tx store.Tx) error {
		for i := range ids {
			id := fmt.Sprintf("aggregate-%02d", i)
			ids[i] = id
			if err := tx.InsertPerson(&domain.Person{ID: id, Login: id, CreatedAt: now}, "h"); err != nil {
				return err
			}
			if err := tx.InsertMember(&domain.Member{ID: id, NetworkID: netID, PersonID: id, DisplayName: id,
				Status: domain.MemberActive, JoinMethod: "invite", JoinedAt: now}); err != nil {
				return err
			}
			if err := tx.InsertAccount(&domain.Account{ID: id, NetworkID: netID, MemberID: id}); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return ids
}

// Individual balance bounds do not bound the running SUM of a whole
// network. Intermediate overflow must not discard later cancelling values.
// Genuine out-of-range final totals still saturate and remain non-zero.
func TestTotalsOverflowCancelsBeforeSaturation(t *testing.T) {
	const b = domain.MaxBalance
	cases := []struct {
		name   string
		values []int64
		want   int64
	}{
		{"positive prefix", []int64{b, b, b, -b, -b, -b}, 0},
		{"negative prefix", []int64{-b, -b, -b, b, b, b}, 0},
		{"alternating", []int64{b, -b, b, -b, b, -b}, 0},
		{"mixed order", []int64{b, b, -b, -b, b, -b}, 0},
		{"positive residual", []int64{b, b, -b, -b, 1}, 1},
		{"negative residual", []int64{-b, -b, -b, b, b, b, -1}, -1},
		{"sign changes after overflow", []int64{b, b, -b, -b, -b}, -b},
		{"exact positive boundary", []int64{b, b, -1}, math.MaxInt64},
		{"exact negative boundary", []int64{-b, -b}, math.MinInt64},
		{"positive final overflow", []int64{b, b}, math.MaxInt64},
		{"negative final overflow", []int64{-b, -b, -1}, math.MinInt64},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			st := openStore(t)
			netID, _, _, _ := seed(t, st)
			ids := aggregateAccounts(t, st, netID, len(tc.values))
			if err := st.Update(context.Background(), func(tx store.Tx) error {
				for i, value := range tc.values {
					if err := tx.AddBalance(netID, ids[i], value); err != nil {
						return err
					}
				}
				return nil
			}); err != nil {
				t.Fatal(err)
			}
			if err := st.View(context.Background(), func(tx store.Tx) error {
				balances, entries, err := tx.Totals(netID)
				if err != nil {
					return err
				}
				if balances != tc.want || entries != 0 {
					t.Errorf("Totals = (%d, %d), want (%d, 0)", balances, entries, tc.want)
				}
				return nil
			}); err != nil {
				t.Fatal(err)
			}
		})
	}
}

// Exercise the actual consistency checker on a persisted, balanced ledger
// whose account order first visits two +2^62 balances. The repository fixture
// uses large postings to keep the test small; it does not relax any schema
// guard or edit historical entries. Every posted transaction remains balanced,
// and each cached balance equals that account's entries within the bound.
func TestVerifyLedgerAcceptsLargeBalancedTotals(t *testing.T) {
	st := openStore(t)
	netID, _, _, _ := seed(t, st)
	ids := aggregateAccounts(t, st, netID, 4)
	now := time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC)
	if err := st.Update(context.Background(), func(tx store.Tx) error {
		for i := 0; i < 2; i++ {
			seq, err := tx.NextLedgerSeq(netID)
			if err != nil {
				return err
			}
			tr := &domain.Transaction{ID: fmt.Sprintf("aggregate-tx-%d", i), NetworkID: netID,
				Kind: domain.KindAdjustment, Status: domain.TxProposed, PayerAccountID: ids[i+2],
				PayeeAccountID: ids[i], Amount: domain.MaxBalance, ProposedAt: now, Reason: "aggregate verification fixture"}
			if err := tx.InsertTransaction(tr); err != nil {
				return err
			}
			for j, value := range []int64{-domain.MaxBalance, domain.MaxBalance} {
				account := []string{tr.PayerAccountID, tr.PayeeAccountID}[j]
				if err := tx.InsertEntry(&domain.Entry{ID: fmt.Sprintf("aggregate-entry-%d-%d", i, j),
					NetworkID: netID, TxID: tr.ID, AccountID: account, Amount: value, LedgerSeq: seq, PostedAt: now}); err != nil {
					return err
				}
				if err := tx.AddBalance(netID, account, value); err != nil {
					return err
				}
			}
			tr.Status, tr.CompletedAt, tr.LedgerSeq = domain.TxCompleted, &now, seq
			if err := tx.UpdateTransaction(tr); err != nil {
				return err
			}
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	svc, err := service.New(st, service.Config{Argon: service.Argon2Params{MemoryKiB: 64, Time: 1, Threads: 1, KeyLen: 16, SaltLen: 8}})
	if err != nil {
		t.Fatal(err)
	}
	rep, err := svc.VerifyLedger(context.Background(), domain.Actor{System: true}, netID)
	if err != nil {
		t.Fatal(err)
	}
	if !rep.OK {
		t.Fatalf("balanced ledger was rejected: %v", rep.Findings)
	}
}
