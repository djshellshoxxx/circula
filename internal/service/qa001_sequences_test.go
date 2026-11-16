package service_test

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/djshellshoxxx/circula/internal/domain"
	"github.com/djshellshoxxx/circula/internal/service"
	tu "github.com/djshellshoxxx/circula/internal/testutil"
)

// QA-001: property test over random sequences of domain operations across at
// least two networks, checking LED-1, LED-2, LED-3, LED-5, LED-7 and GOV-007
// with raw SQL after every operation.
//
// Configuration (recorded in the test log):
//   - CIRCULA_QA001_SEQUENCES: number of sequences. Default 250, 50 under
//     the race detector (QA-001 recommends 1000 for the short run; DEC-023
//     pending). The long run on merge and release sets it to 100000 or more.
//   - CIRCULA_QA001_SEED: replay one sequence by its seed.
//   - CIRCULA_QA001_FAILDIR: where a failing sequence's fixture is written
//     (default: a new directory under the system temp directory).
//
// Each sequence has at most qa001MaxOps operations, the first six of which
// create two networks with two members each. Operations: create network, add
// member, change policy, propose, accept (which completes the trade),
// decline, cancel, expire, reverse, adjust, suspend (and restore), depart.
// A failure saves the seed and the operation log as a JSON fixture and
// reports both.

const qa001MaxOps = 50

var qa001Ops = []string{"create-network", "add-member", "change-policy", "propose", "accept", "complete", "decline", "cancel",
	"expire", "reverse", "adjust", "suspend", "depart"}

type qa001Party struct {
	actor domain.Actor
	m     *domain.Member
}

type qa001Net struct {
	id      string
	admin   domain.Actor
	members []qa001Party
}

type qa001Step struct {
	Op     string `json:"op"`
	Detail string `json:"detail,omitempty"`
	Result string `json:"result"`
}

type qa001Fixture struct {
	Seed    int64       `json:"seed"`
	MaxOps  int         `json:"max_ops"`
	Failure string      `json:"failure"`
	Steps   []qa001Step `json:"steps"`
}

type qa001Run struct {
	t      *testing.T
	e      *tu.Env
	rng    *rand.Rand
	seed   int64
	nets   []*qa001Net
	steps  []qa001Step
	ok     map[string]int // operation -> successful runs, across sequences
	tried  map[string]int
	ledger map[string]string // entry or posted-transaction id -> row image (LED-2)
	bal    map[string]int64  // account id -> balance after the previous step
	actor  map[string]domain.Actor
	serial int
}

func qa001Env(name string, def int) int {
	if v := os.Getenv(name); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			return n
		}
	}
	return def
}

func TestQA001InvariantSequences(t *testing.T) {
	// 1000 sequences take about two minutes on a 4-CPU runner, too long for
	// every change; the default is 250 (50 under the race detector) and CI
	// or a release run sets CIRCULA_QA001_SEQUENCES for longer runs.
	sequences := 250
	if raceOn {
		sequences = 50
	}
	sequences = qa001Env("CIRCULA_QA001_SEQUENCES", sequences)
	seeds := make([]int64, 0, sequences)
	if v := os.Getenv("CIRCULA_QA001_SEED"); v != "" {
		s, err := strconv.ParseInt(v, 10, 64)
		if err != nil {
			t.Fatalf("CIRCULA_QA001_SEED=%q", v)
		}
		seeds = append(seeds, s)
	} else {
		for i := 0; i < sequences; i++ {
			seeds = append(seeds, int64(100_000+i))
		}
	}
	// Sequences run in parallel shards, one per CPU (at most 8), each shard
	// on its own database. A sequence touches only the networks it creates,
	// so its seed replays it (CIRCULA_QA001_SEED) on a fresh database; a
	// fresh database per sequence would cost more than the sequence itself.
	shards := runtime.GOMAXPROCS(0)
	if shards > 8 {
		shards = 8
	}
	if shards > len(seeds) {
		shards = len(seeds)
	}
	var mu sync.Mutex
	ok, tried := map[string]int{}, map[string]int{}
	ops := 0
	t.Run("shards", func(t *testing.T) {
		for s := 0; s < shards; s++ {
			t.Run(strconv.Itoa(s), func(t *testing.T) {
				t.Parallel()
				r := &qa001Run{t: t, ok: map[string]int{}, tried: map[string]int{},
					e: tu.NewEnvConfig(t, nil, func(c *service.Config) { c.MaxOpenRequestsPerPair = 1_000_000 })}
				n := 0
				for i := s; i < len(seeds); i += shards {
					n += r.sequence(seeds[i])
				}
				mu.Lock()
				defer mu.Unlock()
				ops += n
				for k, v := range r.ok {
					ok[k] += v
				}
				for k, v := range r.tried {
					tried[k] += v
				}
			})
		}
	})
	if t.Failed() {
		return
	}
	t.Logf("QA-001: %d sequences, %d operations (at most %d per sequence), checks after every operation", len(seeds), ops, qa001MaxOps)
	names := make([]string, 0, len(tried))
	for k := range tried {
		names = append(names, k)
	}
	sort.Strings(names)
	for _, k := range names {
		t.Logf("  %-15s tried %6d  succeeded %6d", k, tried[k], ok[k])
	}
	if len(seeds) < 50 {
		return // a single-seed replay cannot reach every operation
	}
	for _, op := range qa001Ops {
		if ok[op] == 0 {
			t.Errorf("operation %q never succeeded; the generator does not cover QA-001's operation list", op)
		}
	}
}

// sequence runs one seeded sequence and returns the number of operations.
func (r *qa001Run) sequence(seed int64) int {
	r.seed, r.rng, r.nets, r.steps, r.serial = seed, rand.New(rand.NewSource(seed)), nil, nil, 0
	r.ledger, r.bal, r.actor = map[string]string{}, map[string]int64{}, map[string]domain.Actor{}
	n := 0
	do := func(op string) {
		n++
		r.tried[op]++
		detail, err := r.apply(op)
		res := "ok"
		if err != nil {
			res = err.Error()
			if err != errSkip && domain.CodeOf(err) == domain.CodeInternal {
				r.fail(fmt.Sprintf("%s: internal error: %v", op, err))
			}
		} else {
			r.ok[op]++
			if op == "accept" {
				r.ok["complete"]++ // an accepted trade completes in the same unit (DOM-012)
				r.tried["complete"]++
			}
		}
		r.steps = append(r.steps, qa001Step{Op: op, Detail: detail, Result: res})
		if msg := r.check(); msg != "" {
			r.fail(fmt.Sprintf("after %s: %s", op, msg))
		}
	}
	for i := 0; i < 2; i++ {
		do("create-network")
		do("add-member")
		do("add-member")
	}
	total := 6 + r.rng.Intn(qa001MaxOps-5)
	weights := []struct {
		op string
		w  int
	}{{"create-network", 1}, {"add-member", 2}, {"change-policy", 2}, {"propose", 10}, {"accept", 8}, {"decline", 2},
		{"cancel", 2}, {"expire", 2}, {"reverse", 3}, {"adjust", 3}, {"suspend", 2}, {"depart", 1}}
	sum := 0
	for _, w := range weights {
		sum += w.w
	}
	for n < total {
		x := r.rng.Intn(sum)
		for _, w := range weights {
			if x < w.w {
				do(w.op)
				break
			}
			x -= w.w
		}
	}
	return n
}

func (r *qa001Run) fail(msg string) {
	r.t.Helper()
	fx := qa001Fixture{Seed: r.seed, MaxOps: qa001MaxOps, Failure: msg, Steps: r.steps}
	dir := os.Getenv("CIRCULA_QA001_FAILDIR")
	var err error
	if dir == "" {
		dir, err = os.MkdirTemp("", "circula-qa001-")
	} else {
		err = os.MkdirAll(dir, 0o755)
	}
	path := "(not written)"
	if err == nil {
		path = filepath.Join(dir, fmt.Sprintf("qa001-seed-%d.json", r.seed))
		b, _ := json.MarshalIndent(fx, "", "  ")
		if werr := os.WriteFile(path, b, 0o644); werr != nil {
			path = "(not written: " + werr.Error() + ")"
		}
	}
	r.t.Fatalf("QA-001 invariant violation, seed %d (replay: CIRCULA_QA001_SEED=%d), fixture %s: %s", r.seed, r.seed, path, msg)
}

func (r *qa001Run) key() string { r.serial++; return fmt.Sprintf("qa001-%d-%d", r.seed, r.serial) }

func (r *qa001Run) net() *qa001Net { return r.nets[r.rng.Intn(len(r.nets))] }

func (n *qa001Net) pick(rng *rand.Rand) qa001Party { return n.members[rng.Intn(len(n.members))] }

// txIn returns a random transaction of the network in one of the statuses.
func (r *qa001Run) txIn(n *qa001Net, kinds []domain.TxKind, st ...domain.TxStatus) *domain.Transaction {
	l := tu.Must(r.e.Svc.AdminListTransactions(r.e.Ctx, n.admin, n.id, "", st, service.Page{Limit: 200}))
	var out []*domain.Transaction
	for _, x := range l {
		for _, k := range kinds {
			if x.Kind == k {
				out = append(out, x)
			}
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out[r.rng.Intn(len(out))]
}

func (r *qa001Run) counterparty(tx *domain.Transaction) string {
	if tx.CreatedBy == tx.PayeeMemberID {
		return tx.PayerMemberID
	}
	return tx.PayeeMemberID
}

var errSkip = errors.New("skipped: nothing to act on")

func (r *qa001Run) apply(op string) (string, error) {
	e, rng := r.e, r.rng
	switch op {
	case "create-network":
		if len(r.nets) >= 3 {
			return "", errSkip
		}
		r.serial++
		admin := e.Person(fmt.Sprintf("qa001-admin-%d-%d@example.org", r.seed, r.serial))
		limit := rng.Int63n(20001)
		n, m := e.Network(admin, fmt.Sprintf("q%d-%d", r.seed, r.serial), func(p *domain.Policy) {
			p.DefaultCreditLimit, p.ProposalExpiryDays, p.DisputeWindowDays = limit, 1+rng.Intn(4), 5
		})
		r.nets = append(r.nets, &qa001Net{id: n.ID, admin: admin, members: []qa001Party{{admin, m}}})
		r.actor[m.ID] = admin
		return fmt.Sprintf("net=%s limit=%d", n.ID, limit), nil
	case "add-member":
		n := r.net()
		r.serial++
		a, m := e.Member(n.admin, &domain.Network{ID: n.id}, fmt.Sprintf("qa001-m-%d-%d@example.org", r.seed, r.serial))
		n.members = append(n.members, qa001Party{a, m})
		r.actor[m.ID] = a
		return "net=" + n.id + " member=" + m.ID, nil
	case "change-policy":
		n := r.net()
		cur := tu.Must(e.Svc.GetNetwork(e.Ctx, n.admin, n.id))
		p := cur.Policy
		p.DefaultCreditLimit = rng.Int63n(20001)
		p.ProposalExpiryDays = 1 + rng.Intn(5)
		in := service.PolicyChange{Policy: p, Reason: "qa001", IdempotencyKey: r.key()}
		if rng.Intn(2) == 0 {
			zero := 0
			in.NoticeDays, in.NoticeReason = &zero, "qa001 immediate"
		}
		_, err := e.Svc.UpdatePolicy(e.Ctx, n.admin, n.id, in)
		return fmt.Sprintf("net=%s limit=%d", n.id, p.DefaultCreditLimit), err
	case "propose":
		n := r.net()
		a, b := n.pick(rng), n.pick(rng)
		dir := service.ProposerPays
		if rng.Intn(2) == 0 {
			dir = service.ProposerIsPaid
		}
		amt := 1 + rng.Int63n(6000)
		_, err := e.Svc.ProposeTransaction(e.Ctx, a.actor, n.id, service.ProposeInput{CounterpartyID: b.m.ID, Direction: dir, Amount: amt, IdempotencyKey: r.key()})
		return fmt.Sprintf("net=%s from=%s to=%s %s %d", n.id, a.m.ID, b.m.ID, dir, amt), err
	case "accept", "decline", "cancel":
		n := r.net()
		tx := r.txIn(n, []domain.TxKind{domain.KindTrade}, domain.TxProposed)
		if tx == nil {
			return "", errSkip
		}
		var err error
		switch op {
		case "accept":
			_, err = e.Svc.AcceptTransaction(e.Ctx, r.actor[r.counterparty(tx)], n.id, service.TxRef{TxID: tx.ID, IdempotencyKey: r.key()})
		case "decline":
			_, err = e.Svc.DeclineTransaction(e.Ctx, r.actor[r.counterparty(tx)], n.id, service.TxRef{TxID: tx.ID, IdempotencyKey: r.key()})
		case "cancel":
			_, err = e.Svc.CancelTransaction(e.Ctx, r.actor[tx.CreatedBy], n.id, service.CancelInput{TxID: tx.ID, IdempotencyKey: r.key()})
		}
		return "tx=" + tx.ID, err
	case "expire":
		d := time.Duration(rng.Int63n(int64(4 * 24 * time.Hour)))
		e.Clock.Advance(d)
		k, err := e.Svc.ExpireProposals(e.Ctx, domain.SystemActor("qa001"), 1000)
		if err == nil && k == 0 {
			err = errSkip
		}
		return fmt.Sprintf("advance=%s expired=%d", d, k), err
	case "reverse":
		n := r.net()
		tx := r.txIn(n, []domain.TxKind{domain.KindTrade, domain.KindAdjustment}, domain.TxCompleted)
		if tx == nil {
			return "", errSkip
		}
		amt := int64(0)
		if rng.Intn(2) == 0 {
			amt = 1 + rng.Int63n(tx.Amount)
		}
		_, err := e.Svc.ReverseTransaction(e.Ctx, n.admin, n.id, service.ReverseInput{TxID: tx.ID, Amount: amt, Reason: "qa001", IdempotencyKey: r.key()})
		return fmt.Sprintf("tx=%s amount=%d", tx.ID, amt), err
	case "adjust":
		n := r.net()
		m := n.pick(rng)
		dir := service.AdjustCredit
		if rng.Intn(2) == 0 {
			dir = service.AdjustDebit
		}
		amt := 1 + rng.Int63n(5000)
		_, err := e.Svc.CreateAdjustment(e.Ctx, n.admin, n.id, service.AdjustInput{MemberID: m.m.ID, Amount: amt, Direction: dir, Reason: "qa001", IdempotencyKey: r.key()})
		return fmt.Sprintf("member=%s %s %d", m.m.ID, dir, amt), err
	case "suspend":
		n := r.net()
		m := n.pick(rng)
		if rng.Intn(3) == 0 {
			return "restore member=" + m.m.ID, e.Svc.RestoreMember(e.Ctx, n.admin, n.id, m.m.ID, "qa001")
		}
		return "suspend member=" + m.m.ID, e.Svc.SuspendMember(e.Ctx, n.admin, n.id, m.m.ID, "qa001")
	case "depart":
		n := r.net()
		m := n.pick(rng)
		_, err := e.Svc.DepartMember(e.Ctx, m.actor, n.id, service.DepartInput{})
		return "member=" + m.m.ID, err
	}
	return "", fmt.Errorf("unknown op %s", op)
}

// check verifies the invariants for the sequence's networks with raw SQL and
// returns a description of the first violation, or "".
func (r *qa001Run) check() string {
	db := r.e.Store.DB()
	ids := make([]any, len(r.nets))
	marks := make([]string, len(r.nets))
	for i, n := range r.nets {
		ids[i], marks[i] = n.id, "?"
	}
	if len(ids) == 0 {
		return ""
	}
	in := "(" + strings.Join(marks, ",") + ")"
	q := func(dst any, query string, args ...any) string {
		if err := db.QueryRow(query, args...).Scan(dst); err != nil {
			return fmt.Sprintf("query %q: %v", query, err)
		}
		return ""
	}
	var bad int
	for _, c := range []struct{ name, sql string }{
		{"LED-1 network balance sum", `SELECT COUNT(*) FROM (SELECT network_id FROM accounts WHERE network_id IN ` + in + ` GROUP BY network_id HAVING SUM(balance) <> 0)`},
		{"LED-1 network entry sum", `SELECT COUNT(*) FROM (SELECT network_id FROM entries WHERE network_id IN ` + in + ` GROUP BY network_id HAVING SUM(amount) <> 0)`},
		{"LED-1/GOV-007 two balanced entries per transaction on two accounts", `SELECT COUNT(*) FROM (SELECT tx_id FROM entries WHERE network_id IN ` + in +
			` GROUP BY tx_id HAVING COUNT(*) <> 2 OR SUM(amount) <> 0 OR COUNT(DISTINCT account_id) <> 2)`},
		{"LED-3 cached balance equals sum of entries", `SELECT COUNT(*) FROM accounts a WHERE a.network_id IN ` + in +
			` AND a.balance <> (SELECT COALESCE(SUM(e.amount),0) FROM entries e WHERE e.account_id = a.id)`},
		{"LED-7 integer non-zero entry amounts", `SELECT COUNT(*) FROM entries WHERE network_id IN ` + in + ` AND (typeof(amount) <> 'integer' OR amount = 0)`},
		{"LED-7 integer positive transaction amounts", `SELECT COUNT(*) FROM transactions WHERE network_id IN ` + in + ` AND (typeof(amount) <> 'integer' OR amount <= 0)`},
		{"GOV-007 corrections are reversal or adjustment transactions", `SELECT COUNT(*) FROM transactions WHERE network_id IN ` + in + ` AND kind NOT IN ('trade','reversal','adjustment')`},
		{"posted transactions have entries", `SELECT COUNT(*) FROM transactions t WHERE t.network_id IN ` + in +
			` AND t.status IN ('completed','disputed','reversed') AND NOT EXISTS (SELECT 1 FROM entries e WHERE e.tx_id = t.id)`},
		{"unposted transactions have no entries", `SELECT COUNT(*) FROM transactions t WHERE t.network_id IN ` + in +
			` AND t.status IN ('proposed','declined','cancelled','expired') AND EXISTS (SELECT 1 FROM entries e WHERE e.tx_id = t.id)`},
	} {
		if msg := q(&bad, c.sql, ids...); msg != "" {
			return msg
		}
		if bad != 0 {
			return fmt.Sprintf("%s: %d violation(s)", c.name, bad)
		}
	}
	if msg := statusChains(db, in, ids); msg != "" {
		return msg
	}
	// LED-2: every entry and posted transaction seen before is still there, unchanged.
	seen := map[string]bool{}
	scan := func(query, prefix string) string {
		rows, err := db.Query(query, ids...)
		if err != nil {
			return err.Error()
		}
		defer func() { _ = rows.Close() }()
		for rows.Next() {
			var id, img string
			if err := rows.Scan(&id, &img); err != nil {
				return err.Error()
			}
			id = prefix + id
			seen[id] = true
			if old, ok := r.ledger[id]; ok && old != img {
				return fmt.Sprintf("LED-2: %s changed from %q to %q", id, old, img)
			}
			r.ledger[id] = img
		}
		if err := rows.Err(); err != nil {
			return err.Error()
		}
		return ""
	}
	if msg := scan(`SELECT id, tx_id || '|' || account_id || '|' || amount || '|' || ledger_seq || '|' || posted_at FROM entries WHERE network_id IN `+in, "e:"); msg != "" {
		return msg
	}
	if msg := scan(`SELECT id, kind || '|' || payer_account_id || '|' || payee_account_id || '|' || amount || '|' || ledger_seq || '|' || completed_at
		FROM transactions WHERE network_id IN `+in+` AND completed_at IS NOT NULL`, "t:"); msg != "" {
		return msg
	}
	for id := range r.ledger {
		if !seen[id] {
			return "LED-2: " + id + " disappeared"
		}
	}
	// LED-5: a non-system account whose balance went down is not below its
	// effective limit: its unexpired override, else the default of the
	// policy version in force now.
	now := r.e.Clock.Now().UnixMilli()
	rows, err := db.Query(`SELECT a.id, a.balance, a.limit_override, a.limit_override_until,
		(SELECT json_extract(p.policy_json, '$.default_credit_limit') FROM network_policies p
		 WHERE p.network_id = a.network_id AND p.effective_at <= ? ORDER BY p.version DESC LIMIT 1)
		FROM accounts a WHERE a.is_system = 0 AND a.network_id IN `+in, append([]any{now}, ids...)...)
	if err != nil {
		return err.Error()
	}
	defer func() { _ = rows.Close() }()
	for rows.Next() {
		var id string
		var bal int64
		var over, until, def sql.NullInt64
		if err := rows.Scan(&id, &bal, &over, &until, &def); err != nil {
			return err.Error()
		}
		limit := def.Int64
		if over.Valid && (!until.Valid || now < until.Int64) {
			limit = over.Int64
		}
		if prev, ok := r.bal[id]; ok && bal < prev && bal < -limit {
			return fmt.Sprintf("LED-5: account %s went from %d to %d, limit %d", id, prev, bal, limit)
		}
		r.bal[id] = bal
	}
	if err := rows.Err(); err != nil {
		return err.Error()
	}
	return ""
}
