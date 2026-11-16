package contract

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"sync"
	"testing"
)

// API-071: a valid reviewed proposal can become unaffordable before it is
// accepted. Both adapters refuse without a posting/audit mutation, permit the
// same key after the cause is removed, and preserve an already successful retry.
func TestAdapterEquivalenceCreditAndRetries(t *testing.T) {
	p := newAdapterPair(t)
	proposal := adapterStep{key: "credit-proposal", who: "bob", apiMethod: "POST", apiPath: "/transactions", webMethod: "POST", webPath: "{base}/trades/new", apiStatus: 201, review: true,
		json: m{"counterparty_member_id": "{alice}", "direction": "receive", "amount_minor": "9000", "description": "credit race"},
		form: map[string]string{"counterparty_id": "{alice}", "direction": "request", "amount": "90.00", "note": "credit race"}}
	p.step(proposal)
	p.remember("trade", "transactions", "note=?", "credit race")
	// Replay proposal through the exact same service key. The web review's
	// generated idem token must be retained, so use one prepared submission.
	retry := adapterStep{key: "retry-proposal", who: "alice", apiMethod: "POST", apiPath: "/transactions", webMethod: "POST", webPath: "{base}/trades/new", apiStatus: 201, review: true,
		json: m{"counterparty_member_id": "{bob}", "direction": "pay", "amount_minor": "100", "description": "replay once"},
		form: map[string]string{"counterparty_id": "{bob}", "direction": "pay", "amount": "1.00", "note": "replay once"}}
	av, wv := p.api.prepare(retry), p.web.prepare(retry)
	am, wm := p.api.auditMark(), p.web.auditMark()
	p.api.execute(retry, av)
	p.web.execute(retry, wv)
	if a, w := p.api.normalize(p.api.auditAfter(am)), p.web.normalize(p.web.auditAfter(wm)); a != w {
		t.Fatalf("first proposal audit differs: %s / %s", a, w)
	}
	for _, f := range []*adapterFixture{p.api, p.web} {
		vals := wv
		if f.isAPI {
			vals = av
		}
		mark := f.auditMark()
		before := f.normalize(f.ledger())
		f.execute(retry, vals)
		if before != f.normalize(f.ledger()) || len(f.auditAfter(mark)) != 0 {
			t.Fatal("same-key replay duplicated proposal or audit")
		}
	}
	// A changed payload with that key fails on both surfaces, without hiding
	// a second transaction behind transport-specific conflict responses.
	conflict := retry
	conflict.apiStatus = 422
	conflict.webStatus = 422
	conflict.json = m{"counterparty_member_id": "{bob}", "direction": "pay", "amount_minor": "101", "description": "replay once"}
	changed := map[string]string{}
	for k, v := range wv {
		changed[k] = v
	}
	changed["amount"] = "1.01"
	for _, f := range []*adapterFixture{p.api, p.web} {
		vals := changed
		if f.isAPI {
			vals = av
		}
		mark := f.auditMark()
		before := f.normalize(f.ledger())
		f.execute(conflict, vals)
		if before != f.normalize(f.ledger()) || len(f.auditAfter(mark)) != 0 {
			t.Fatal("different-body key reuse changed state")
		}
	}
	limit := func(key, minor, decimal string) {
		p.step(adapterStep{key: key, who: "admin", apiMethod: "PUT", apiPath: "/admin/accounts/{alice}/credit-limit", webMethod: "POST", webPath: "{base}/admin/members/{alice}/limit",
			json: m{"credit_limit_minor": minor, "reason": "Changed circumstances"}, form: map[string]string{"amount": decimal, "reason": "Changed circumstances"}})
	}
	limit("lower-limit", "5000", "50.00")
	accept := adapterStep{key: "accept-after-limit-change", who: "alice", apiMethod: "POST", apiPath: "/transactions/{trade}/accept", webMethod: "POST", webPath: "{base}/trades/{trade}/accept", apiStatus: 422, webStatus: 422}
	beforeA, beforeW := p.api.normalize(p.api.ledger()), p.web.normalize(p.web.ledger())
	p.step(accept)
	if beforeA != p.api.normalize(p.api.ledger()) || beforeW != p.web.normalize(p.web.ledger()) {
		t.Fatal("unaffordable acceptance mutated the ledger")
	}
	limit("restore-limit", "10000", "100.00")
	accept.apiStatus, accept.webStatus = 200, 303
	p.step(accept)
	for _, f := range []*adapterFixture{p.api, p.web} {
		before := f.normalize(f.ledger())
		mark := f.auditMark()
		f.execute(accept, f.prepare(accept))
		if before != f.normalize(f.ledger()) || len(f.auditAfter(mark)) != 0 {
			t.Fatal("successful acceptance retry posted/audited twice")
		}
	}
}

// Two indistinguishable requests for 80.00 compete for the same 100.00 limit.
// The winner may differ by scheduling; transaction identities are normalized
// by their outcome, while every amount, party, status and audit row is retained.
func TestAdapterEquivalenceConcurrentAcceptances(t *testing.T) {
	p := newAdapterPair(t)
	limit := func(key, minor, decimal string) {
		p.step(adapterStep{key: key, who: "admin", apiMethod: "PUT", apiPath: "/admin/accounts/{alice}/credit-limit", webMethod: "POST", webPath: "{base}/admin/members/{alice}/limit",
			json: m{"credit_limit_minor": minor, "reason": "Concurrent acceptance test"}, form: map[string]string{"amount": decimal, "reason": "Concurrent acceptance test"}})
	}
	limit("concurrent-initial-limit", "20000", "200.00")
	for i := 0; i < 2; i++ {
		p.step(adapterStep{key: fmt.Sprintf("concurrent-proposal-%d", i), who: "bob", apiMethod: "POST", apiPath: "/transactions", webMethod: "POST", webPath: "{base}/trades/new", apiStatus: 201, review: true,
			json: m{"counterparty_member_id": "{alice}", "direction": "receive", "amount_minor": "8000", "description": "concurrent"}, form: map[string]string{"counterparty_id": "{alice}", "direction": "request", "amount": "80.00", "note": "concurrent"}})
	}
	limit("concurrent-current-limit", "10000", "100.00")
	out := []string{}
	for _, f := range []*adapterFixture{p.api, p.web} {
		ids := f.rows("SELECT id FROM transactions ORDER BY id")
		mark := f.auditMark()
		start := make(chan struct{})
		codes := make(chan int, 2)
		var wg sync.WaitGroup
		for i, row := range ids {
			wg.Add(1)
			go func(i int, id string) {
				defer wg.Done()
				<-start
				if f.isAPI {
					codes <- f.request("alice", "POST", "/api/v1/transactions/"+id+"/accept", nil, map[string]string{"Idempotency-Key": fmt.Sprintf("race-accept-%d", i)}).Code
				} else {
					codes <- f.request("alice", "POST", "{base}/trades/"+id+"/accept", map[string]string{"idem": fmt.Sprintf("race-accept-%d", i)}, nil).Code
				}
			}(i, row[0].(string))
		}
		close(start)
		wg.Wait()
		close(codes)
		statuses := []int{}
		for code := range codes {
			statuses = append(statuses, code)
		}
		sort.Ints(statuses)
		want := []int{200, 422}
		if !f.isAPI {
			want[0] = 303
		}
		if fmt.Sprint(statuses) != fmt.Sprint(want) {
			t.Fatalf("api=%t concurrent statuses=%v want=%v", f.isAPI, statuses, want)
		}
		state := f.ledger()
		state["audit"] = f.auditAfter(mark)
		out = append(out, normalizeConcurrentOutcome(f, state))
		rows := f.rows("SELECT balance FROM accounts WHERE member_id=?", f.ids["alice"])
		if rows[0][0].(int64) != -8000 || len(f.rows("SELECT id FROM entries")) != 2 {
			t.Fatal("concurrent requests overspent or failed to post one balanced trade")
		}
	}
	if out[0] != out[1] {
		t.Fatalf("concurrent adapter outcomes differ\nAPI %s\nHTML %s", out[0], out[1])
	}
}

func normalizeConcurrentOutcome(f *adapterFixture, state map[string][][]any) string {
	s := f.normalize(state)
	for i, row := range f.rows("SELECT status FROM transactions ORDER BY id") {
		s = strings.ReplaceAll(s, fmt.Sprintf("transactions#%d", i+1), "transaction:"+row[0].(string))
	}
	var normalized map[string][][]any
	if err := json.Unmarshal([]byte(s), &normalized); err != nil {
		f.t.Fatal(err)
	}
	for _, rows := range normalized {
		sort.Slice(rows, func(i, j int) bool {
			a, _ := json.Marshal(rows[i])
			b, _ := json.Marshal(rows[j])
			return string(a) < string(b)
		})
	}
	b, err := json.Marshal(normalized)
	if err != nil {
		f.t.Fatal(err)
	}
	return string(b)
}
