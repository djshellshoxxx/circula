package service_test

import (
	"sync"
	"testing"
	"time"

	"github.com/djshellshoxxx/circula/internal/domain"
	"github.com/djshellshoxxx/circula/internal/histbal"
	"github.com/djshellshoxxx/circula/internal/service"
	tu "github.com/djshellshoxxx/circula/internal/testutil"
)

// histWorld posts, for member b (all trades a pays b unless noted):
//
//	seq 1  T1        1000   (later reversed)
//	seq 2  T1         200   (equal-timestamp tie with seq 1)
//	seq 3  T1+1ms      50
//	       T1+1ms     999   proposed only, never accepted
//	seq 4  T2       -1000   reversal of seq 1
//	seq 5  T0+30m       7   clock rolled back: earlier than seq 1-4, highest seq
type histWorld struct {
	*world
	t0, t1, t2, t3 time.Time
}

func newHistWorld(t *testing.T) *histWorld {
	w := &histWorld{world: newWorld(t, nil)}
	w.t0 = w.Clock.Now()
	w.Clock.Advance(time.Hour)
	w.t1 = w.Clock.Now()
	first := w.Trade(w.net.ID, w.a, w.b, w.aM, w.bM, 1000)
	w.Trade(w.net.ID, w.a, w.b, w.aM, w.bM, 200)
	w.Clock.Advance(time.Millisecond)
	w.Trade(w.net.ID, w.a, w.b, w.aM, w.bM, 50)
	tu.Must(w.propose(w.a, w.bM, service.ProposerPays, 999))
	w.Clock.Advance(time.Hour)
	w.t2 = w.Clock.Now()
	tu.Must(w.Svc.ReverseTransaction(w.Ctx, w.admin, w.net.ID, service.ReverseInput{TxID: first.ID, Reason: "entered twice", IdempotencyKey: w.Key()}))
	w.Clock.Advance(-(w.t2.Sub(w.t0) - 30*time.Minute)) // rollback to T0+30m
	w.t3 = w.Clock.Now()
	w.Trade(w.net.ID, w.a, w.b, w.aM, w.bM, 7)
	w.Clock.Advance(24 * time.Hour) // service clock well past every posting
	return w
}

func (w *histWorld) at(a domain.Actor, asOf time.Time) histbal.Result {
	w.T.Helper()
	return tu.Must(w.Svc.HistoricalBalance(w.Ctx, a, w.net.ID, asOf))
}

// QA-046, FR-054, DAT-050: inclusive timestamp cutoff over one snapshot.
func TestHistoricalBalanceTimestampCutoff(t *testing.T) {
	w := newHistWorld(t)
	ms := time.Millisecond
	cases := []struct {
		name string
		at   time.Time
		want int64
		has  bool
	}{
		{"before any posting", w.t3.Add(-ms), 0, false},
		{"rolled-back posting with the highest seq is included", w.t3, 7, true},
		{"just before the tie", w.t1.Add(-ms), 7, true},
		{"tie set at the boundary is complete", w.t1, 1207, true},
		{"posting immediately after the boundary", w.t1.Add(ms), 1257, true},
		{"pending proposal and later reversal excluded", w.t2.Add(-ms), 1257, true},
		{"reversal counts from its own instant", w.t2, 257, true},
		{"now", w.Clock.Now(), 257, true},
	}
	for _, c := range cases {
		r := w.at(w.b, c.at)
		if r.BalanceMinor != c.want || r.HasPostingsByAsOf != c.has {
			t.Errorf("%s: got %d has=%v, want %d has=%v", c.name, r.BalanceMinor, r.HasPostingsByAsOf, c.want, c.has)
		}
		// The high-water sequence is the snapshot boundary, even when later
		// entries (by timestamp) at lower or higher sequences are excluded.
		if r.LedgerSeq != 5 || !r.AsOf.Equal(c.at) || r.AsOf.Location() != time.UTC || r.Decimals != w.net.Policy.Decimals {
			t.Errorf("%s: meta %+v", c.name, r)
		}
	}
	if cur := w.balance(w.b); cur.Balance != 257 {
		t.Errorf("cached balance %d", cur.Balance)
	}
	// A greatest-sequence prefix would give 1200 at T1 (seq<=2) and miss the
	// seq-5 entry timestamped earlier: the timestamp read must differ.
	if prefix := tu.Must(w.Svc.BalanceAt(w.Ctx, w.b, w.net.ID, 2)); prefix != 1200 {
		t.Fatalf("prefix sum %d", prefix)
	}
	// A posting timestamped before network creation (clock rollback): entries
	// are append-only, so move the network's recorded creation after the
	// seq-5 posting instead. The read uses posting timestamps, never created_at.
	if _, err := w.Store.DB().Exec(`UPDATE networks SET created_at = ? WHERE id = ?`, w.t3.Add(time.Hour).UnixMilli(), w.net.ID); err != nil {
		t.Fatal(err)
	}
	if r := w.at(w.b, w.t3); r.BalanceMinor != 7 || !r.HasPostingsByAsOf {
		t.Errorf("pre-creation posting: %+v", r)
	}
	if r := w.at(w.b, w.t3.Add(-ms)); r.BalanceMinor != 0 || r.HasPostingsByAsOf {
		t.Errorf("before pre-creation posting: %+v", r)
	}
	// The payer sees only their own account.
	if r := w.at(w.a, w.t1); r.BalanceMinor != -1207 {
		t.Errorf("payer at T1: %d", r.BalanceMinor)
	}
}

func TestHistoricalBalanceOffsetsNormalize(t *testing.T) {
	w := newHistWorld(t)
	plus2 := w.t1.In(time.FixedZone("+02:00", 2*3600))
	minus5 := w.t1.In(time.FixedZone("-05:00", -5*3600))
	for _, at := range []time.Time{w.t1, plus2, minus5} {
		r := w.at(w.b, at)
		if r.BalanceMinor != 1207 || r.AsOf.Location() != time.UTC || !r.AsOf.Equal(w.t1) {
			t.Errorf("%v: %+v", at, r)
		}
	}
}

// Zero sum with postings is not the no-postings state.
func TestHistoricalBalanceZeroSumHasPostings(t *testing.T) {
	w := newWorld(t, nil)
	c, cM := w.Member(w.admin, w.net, "cy@example.org")
	w.Trade(w.net.ID, c, w.a, cM, w.aM, 30)
	w.Trade(w.net.ID, w.a, c, w.aM, cM, 30)
	r := tu.Must(w.Svc.HistoricalBalance(w.Ctx, c, w.net.ID, w.Clock.Now()))
	if r.BalanceMinor != 0 || !r.HasPostingsByAsOf {
		t.Fatalf("%+v", r)
	}
}

func TestHistoricalBalanceRefusals(t *testing.T) {
	w := newWorld(t, func(p *domain.Policy) { p.JoinMode = domain.JoinApproval })
	now := w.Clock.Now()
	for name, at := range map[string]time.Time{
		"future":        now.Add(time.Millisecond),
		"sub-ms":        now.Add(-time.Second + time.Microsecond),
		"future sub-ms": now.Add(time.Microsecond),
	} {
		_, err := w.Svc.HistoricalBalance(w.Ctx, w.b, w.net.ID, at)
		if !histbal.IsInputError(err) {
			t.Errorf("%s: err %v", name, err)
		}
	}
	pend, _ := w.joinPending("pat@example.org")
	if _, err := w.Svc.HistoricalBalance(w.Ctx, pend, w.net.ID, now); domain.CodeOf(err) != domain.CodeForbidden {
		t.Errorf("pending member: %v", err)
	}
	stranger := w.Person("zed@example.org")
	if _, err := w.Svc.HistoricalBalance(w.Ctx, stranger, w.net.ID, now); err == nil {
		t.Error("non-member read a balance")
	}
}

// DAT-050: concurrent posting. With the clock fixed, every posting has the
// same instant, so a result is consistent only if its sum and its high-water
// sequence come from one snapshot: b's balance must be 10 x ledger_seq.
func TestHistoricalBalanceConcurrentPosting(t *testing.T) {
	w := newWorld(t, nil)
	const n = 25
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < n; i++ {
			w.Trade(w.net.ID, w.a, w.b, w.aM, w.bM, 10)
		}
	}()
	now := w.Clock.Now()
	for i := 0; i < 4*n; i++ {
		r, err := w.Svc.HistoricalBalance(w.Ctx, w.b, w.net.ID, now)
		if err != nil {
			if domain.CodeOf(err) == domain.CodeUnavailable {
				continue // busy is an explicit, retryable refusal
			}
			t.Fatal(err)
		}
		if r.BalanceMinor != 10*r.LedgerSeq || r.HasPostingsByAsOf != (r.LedgerSeq > 0) {
			t.Fatalf("torn read: %+v", r)
		}
	}
	wg.Wait()
	if r := tu.Must(w.Svc.HistoricalBalance(w.Ctx, w.b, w.net.ID, now)); r.BalanceMinor != 10*n || r.LedgerSeq != n {
		t.Fatalf("final %+v", r)
	}
}
