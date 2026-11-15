package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"reflect"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/djshellshoxxx/circula/internal/domain"
	"github.com/djshellshoxxx/circula/internal/histbal"
)

type fakeHist struct {
	calls int
	actor domain.Actor
	netID string
	asOf  time.Time
	res   histbal.Result
	err   error
}

func (f *fakeHist) HistoricalBalance(_ context.Context, a domain.Actor, netID string, asOf time.Time) (histbal.Result, error) {
	f.calls++
	f.actor, f.netID, f.asOf = a, netID, asOf
	return f.res, f.err
}

var histNow = time.Date(2026, 10, 27, 12, 0, 0, 0, time.UTC)

func callHist(t *testing.T, f *fakeHist, rawQuery string) *httptest.ResponseRecorder {
	t.Helper()
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/api/v1/accounts/me?"+rawQuery, nil)
	q, _ := url.ParseQuery(rawQuery)
	c := &reqCtx{w: w, r: r, query: q, actor: domain.Actor{PersonID: "p1"}, netID: "net1"}
	if err := histBalanceHandler(f, func() time.Time { return histNow })(c); err != nil {
		ae, _ := fromErr(err)
		writeProblem(w, r, "rid", ae)
	}
	return w
}

func TestHistBalanceOffsetsNormalize(t *testing.T) {
	want := time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC)
	for _, at := range []string{"2026-03-01T12:00:00Z", "2026-03-01T14:00:00%2B02:00", "2026-03-01T07:00:00-05:00", "2026-03-01T12:00:00.000Z"} {
		f := &fakeHist{res: histbal.Result{AsOf: want, Decimals: 2}}
		w := callHist(t, f, "at="+at)
		if w.Code != 200 || f.calls != 1 {
			t.Fatalf("%s: code %d calls %d body %s", at, w.Code, f.calls, w.Body)
		}
		if !f.asOf.Equal(want) || f.asOf.Location() != time.UTC {
			t.Fatalf("%s: reader got %v", at, f.asOf)
		}
	}
}

func TestHistBalanceRejectsBadInput(t *testing.T) {
	cases := map[string]string{
		"date-only": "at=2026-03-01", "no-offset": "at=2026-03-01T12:00:00", "precision": "at=2026-03-01T12:00:00.1234Z",
		"future": "at=2026-10-27T12:00:01Z", "missing": "", "empty": "at=", "repeated": "at=2026-03-01T12:00:00Z&at=2026-03-02T12:00:00Z",
		"unknown-param": "at=2026-03-01T12:00:00Z&x=1",
	}
	for name, q := range cases {
		f := &fakeHist{}
		w := callHist(t, f, q)
		if w.Code != 400 || f.calls != 0 {
			t.Fatalf("%s: code %d calls %d", name, w.Code, f.calls)
		}
		if !strings.Contains(w.Body.String(), "invalid-query") {
			t.Fatalf("%s: not invalid-query: %s", name, w.Body)
		}
	}
}

func TestHistBalanceResponseShape(t *testing.T) {
	f := &fakeHist{res: histbal.Result{AsOf: time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC), BalanceMinor: -1250, Decimals: 2, LedgerSeq: 42, HasPostingsByAsOf: true}}
	w := callHist(t, f, "at=2026-03-01T12:00:00Z")
	var m map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &m); err != nil {
		t.Fatal(err)
	}
	var keys []string
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	want := []string{"as_of", "balance_minor", "decimals", "direction", "has_postings_by_as_of", "ledger_seq"}
	if !reflect.DeepEqual(keys, want) {
		t.Fatalf("keys %v", keys)
	}
	if m["as_of"] != "2026-03-01T12:00:00.000Z" || m["balance_minor"] != "-1250" || m["direction"] != "in_debt" ||
		m["decimals"] != float64(2) || m["ledger_seq"] != float64(42) || m["has_postings_by_as_of"] != true {
		t.Fatalf("body %v", m)
	}
}

func TestHistBalanceZeroNoPostings(t *testing.T) {
	f := &fakeHist{res: histbal.Result{AsOf: time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC), Decimals: 2, LedgerSeq: 7}}
	w := callHist(t, f, "at=2020-01-01T00:00:00Z")
	var m map[string]any
	_ = json.Unmarshal(w.Body.Bytes(), &m)
	if w.Code != 200 || m["balance_minor"] != "0" || m["direction"] != "zero" || m["has_postings_by_as_of"] != false {
		t.Fatalf("%d %v", w.Code, m)
	}
}

func TestHistBalanceReaderErrorAndPassThrough(t *testing.T) {
	f := &fakeHist{err: domain.Forbidden("not allowed")}
	w := callHist(t, f, "at=2026-03-01T12:00:00Z")
	if w.Code != http.StatusForbidden || f.calls != 1 {
		t.Fatalf("code %d calls %d", w.Code, f.calls)
	}
	if f.actor.PersonID != "p1" || f.netID != "net1" {
		t.Fatalf("actor %+v net %q", f.actor, f.netID)
	}
}
