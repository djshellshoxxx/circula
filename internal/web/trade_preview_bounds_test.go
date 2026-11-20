package web

import (
	"math"
	"net/url"
	"strings"
	"testing"

	"github.com/djshellshoxxx/circula/internal/domain"
	"github.com/djshellshoxxx/circula/internal/service"
	"github.com/djshellshoxxx/circula/internal/testutil"
)

// DOM-029, UX-021, UX-022: do not present a confirmation for an amount that
// already exceeds the displayed network policy. Keep the entered value so
// the member can correct it without submitting an impossible proposal.
func TestTradeProposalReviewChecksAmountLimit(t *testing.T) {
	w := newWorld(t)
	br := w.browser(t)
	br.login("alice@example.org")
	path := w.base() + "/trades/new"
	form := br.get(path)
	for _, direction := range []string{"pay", "request"} {
		for _, amount := range []int64{w.net.Policy.MaxTxAmount + 1, math.MaxInt64} {
			entered := domain.FormatAmount(amount, w.net.Policy.Decimals)
			preview := br.post(path, form, url.Values{"counterparty_id": {w.bobM.ID}, "direction": {direction}, "amount": {entered}, "step": {"review"}})
			if preview.Status != 422 {
				t.Errorf("%s amount %s review status = %d", direction, entered, preview.Status)
			}
			br.mustContain(preview, `href="#f-amount"`, `value="`+entered+`"`, "largest single trade")
			if strings.Contains(preview.Body, `name="step" value="confirm"`) {
				t.Error("an over-limit amount received a confirmation control")
			}
		}
		valid := br.post(path, form, url.Values{"counterparty_id": {w.bobM.ID}, "direction": {direction},
			"amount": {domain.FormatAmount(w.net.Policy.MaxTxAmount, w.net.Policy.Decimals)}, "step": {"review"}})
		if valid.Status != 200 {
			t.Fatalf("amount at policy limit review status = %d", valid.Status)
		}
		br.mustContain(valid, `name="step" value="confirm"`)
	}
	transactions := testutil.Must(w.env.Svc.ListTransactions(w.env.Ctx, w.alice, w.net.ID, nil, service.Page{}))
	if len(transactions) != 0 {
		t.Fatal("amount previews created a transaction")
	}
}

// Boundary fixture only: seed a supported cached balance directly so the
// actual HTTP previews can exercise the account limit without thousands of
// unrelated postings. These reads do not establish ledger reconciliation.
func setPreviewBalance(t *testing.T, w *world, memberID string, balance int64) {
	t.Helper()
	if _, err := w.env.Store.DB().Exec(`UPDATE accounts SET balance = ? WHERE network_id = ? AND member_id = ?`, balance, w.net.ID, memberID); err != nil {
		t.Fatal(err)
	}
}

func TestTradeProposalReviewChecksProjectedBalance(t *testing.T) {
	w := newWorld(t)
	br := w.browser(t)
	br.login("alice@example.org")
	path := w.base() + "/trades/new"
	for _, tc := range []struct {
		direction string
		balance   int64
	}{
		{"request", domain.MaxBalance},
		{"pay", -domain.MaxBalance},
	} {
		setPreviewBalance(t, w, w.aliceM.ID, tc.balance)
		preview := br.post(path, br.get(path), url.Values{"counterparty_id": {w.bobM.ID}, "direction": {tc.direction}, "amount": {"0.01"}, "step": {"review"}})
		if preview.Status != 422 {
			t.Errorf("%s out-of-range projected balance review status = %d", tc.direction, preview.Status)
		}
		br.mustContain(preview, `href="#f-amount"`, "supported range", `value="0.01"`)
		if strings.Contains(preview.Body, `name="step" value="confirm"`) {
			t.Error("an out-of-range projected balance received a confirmation control")
		}
	}
}

func TestTradeAcceptanceProjectedBalanceBoundaryAndReplay(t *testing.T) {
	w := newWorld(t)
	tx := testutil.Must(w.env.Svc.ProposeTransaction(w.env.Ctx, w.bob, w.net.ID, service.ProposeInput{
		CounterpartyID: w.aliceM.ID, Direction: service.ProposerPays, Amount: 500, IdempotencyKey: w.env.Key()}))
	br := w.browser(t)
	br.login("alice@example.org")
	path := w.base() + "/trades/" + tx.ID + "/accept"
	setPreviewBalance(t, w, w.aliceM.ID, domain.MaxBalance)
	refused := br.get(path)
	if refused.Status != 422 {
		t.Fatalf("out-of-range acceptance preview status = %d", refused.Status)
	}
	if strings.Contains(refused.Body, `action="`+path+`"`) {
		t.Fatal("out-of-range acceptance rendered a submit control")
	}
	setPreviewBalance(t, w, w.aliceM.ID, domain.MaxBalance-500)
	preview := br.get(path)
	if preview.Status != 200 {
		t.Fatalf("exact-boundary acceptance preview status = %d", preview.Status)
	}
	br.mustContain(preview, domain.FormatAmount(domain.MaxBalance, w.net.Policy.Decimals))
	values := url.Values{}
	for attempt := 0; attempt < 2; attempt++ {
		posted := br.post(path, preview, values)
		if posted.Status != 200 {
			t.Fatalf("acceptance attempt %d status = %d", attempt+1, posted.Status)
		}
		after := testutil.Must(w.env.Svc.GetBalance(w.env.Ctx, w.alice, w.net.ID))
		if after.Balance != domain.MaxBalance {
			t.Fatalf("acceptance attempt %d balance = %d", attempt+1, after.Balance)
		}
	}
}

// An earlier committed proposal remains an idempotent result after the
// network lowers its maximum. Preview validation must not bypass that result.
func TestTradeProposalConfirmationReplayAfterPolicyChange(t *testing.T) {
	w := newWorld(t)
	br := w.browser(t)
	br.login("alice@example.org")
	path := w.base() + "/trades/new"
	form := br.get(path)
	values := url.Values{"counterparty_id": {w.bobM.ID}, "direction": {"pay"}, "amount": {"5.00"}, "step": {"confirm"}}
	first := br.post(path, form, values)
	if first.Status != 200 {
		t.Fatalf("proposal status = %d", first.Status)
	}
	pol := testutil.Must(w.env.Svc.GetNetwork(w.env.Ctx, w.admin, w.net.ID)).Policy
	pol.MaxTxAmount = 100
	testutil.Must(w.env.Svc.UpdatePolicy(w.env.Ctx, w.admin, w.net.ID, service.PolicyChange{
		Policy: pol, Reason: "Lower maximum for new proposals", IdempotencyKey: w.env.Key()}))
	replayed := br.post(path, form, values)
	if replayed.Status != 200 || replayed.URL != first.URL {
		t.Fatalf("proposal replay status/path = %d %s; want 200 %s", replayed.Status, replayed.URL, first.URL)
	}
	transactions := testutil.Must(w.env.Svc.ListTransactions(w.env.Ctx, w.alice, w.net.ID, nil, service.Page{}))
	if len(transactions) != 1 || transactions[0].Amount != 500 {
		t.Fatal("proposal replay changed the original result")
	}
}
