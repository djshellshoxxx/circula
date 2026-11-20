package web

import (
	"fmt"
	"testing"

	"github.com/djshellshoxxx/circula/internal/domain"
)

// Exact amounts and spoken labels stay consistent, including unit names
// which already end in s. No floating-point formatting is used for amounts.
func TestBalanceCatalogueMessages(t *testing.T) {
	assertBalance := func(value int64, dec int, unit string, want balanceVM) {
		t.Helper()
		if got := makeBalance(value, dec, unit); got != want {
			t.Errorf("balance(%d,%d,%s)=%+v, want %+v", value, dec, unit, got, want)
		}
	}
	assertBalance(0, 2, "credit", balanceVM{"0.00", "Level", "Level, zero credits"})
	assertBalance(0, 2, "Hours", balanceVM{"0.00", "Level", "Level, zero Hours"})
	assertBalance(100, 2, "credit", balanceVM{"+1.00", "In credit", "In credit by 1.00 credit"})
	assertBalance(-100, 2, "credit", balanceVM{"−1.00", "In debt", "In debt by 1.00 credit"})
	assertBalance(123456789012345, 2, "credit", balanceVM{"+1234567890123.45", "In credit", "In credit by 1234567890123.45 credits"})
	assertBalance(-42, 0, "Hours", balanceVM{"−42", "In debt", "In debt by 42 Hours"})
}

// Each kind/direction/state uses a complete sentence. Ledger references remain
// ungrouped identifiers, even though catalogue counts use locale grouping.
func TestTransactionCatalogueRows(t *testing.T) {
	me := &domain.Member{ID: "me"}
	name := "Zoë 100%"
	n := names{known: map[string]*string{"other": &name}}
	for _, kind := range []domain.TxKind{domain.KindTrade, domain.KindReversal, domain.KindAdjustment, domain.KindGrant} {
		prefix := ""
		switch kind {
		case domain.KindReversal:
			prefix = "Reversal: "
		case domain.KindAdjustment:
			prefix = "Adjustment: "
		case domain.KindGrant:
			prefix = "Sign-up grant: "
		}
		for _, outgoing := range []bool{false, true} {
			for _, pending := range []bool{false, true} {
				t.Run(fmt.Sprintf("%s/outgoing=%v/pending=%v", kind, outgoing, pending), func(t *testing.T) {
					tx := &domain.Transaction{Kind: kind, Amount: 1250, PayerMemberID: "other", PayeeMemberID: "me", Status: domain.TxCompleted, LedgerSeq: 12345}
					want, signed := prefix+"You received 12.50 credits from Zoë 100%", "+12.50"
					if pending {
						tx.Status = domain.TxProposed
						want = prefix + "You will receive 12.50 credits from Zoë 100%"
					}
					if outgoing {
						tx.PayerMemberID, tx.PayeeMemberID = "me", "other"
						want, signed = prefix+"You paid Zoë 100% 12.50 credits", "−12.50"
						if pending {
							want = prefix + "You pay Zoë 100% 12.50 credits"
						}
					}
					got := makeRow(tx, me, n, "credit", 2)
					if got.Text != want || got.Signed != signed {
						t.Fatalf("row=%+v; want %q (%s)", got, want, signed)
					}
					if !pending && got.Ledger != "Ledger reference 12345" {
						t.Fatal(got.Ledger)
					}
				})
			}
		}
	}
}
