package web

import (
	"context"
	"html"
	"strings"
	"testing"
	"time"

	"github.com/djshellshoxxx/circula/internal/domain"
	"github.com/djshellshoxxx/circula/internal/histbal"
)

type fakeHB struct {
	calls int
	at    time.Time
	res   histbal.Result
}

func (f *fakeHB) HistoricalBalance(_ context.Context, _ domain.Actor, _ string, at time.Time) (histbal.Result, error) {
	f.calls++
	f.at = at
	r := f.res
	r.AsOf = at
	return r, nil
}

var hbNow = time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)

func hbRun(t *testing.T, raw string, f *fakeHB) (histBalanceVM, string) {
	t.Helper()
	vm, err := buildHistBalanceVM(context.Background(), raw, hbNow, f, domain.Actor{}, "n1", "credit")
	if err != nil {
		t.Fatal(err)
	}
	var h strings.Builder
	if err := renderHistBalance(&h, vm); err != nil {
		t.Fatal(err)
	}
	return vm, h.String()
}

func TestHistBalanceFormOnly(t *testing.T) {
	f := &fakeHB{}
	_, out := hbRun(t, "", f)
	if f.calls != 0 || strings.Contains(out, "histbal-result") || strings.Contains(out, "aria-invalid") {
		t.Fatal(out)
	}
	for _, w := range []string{`<form method="get"`, `<label for="histbal-at"`, `id="histbal-at"`, `aria-describedby="histbal-hint"`, "2026-03-01T12:00:00Z"} {
		if !strings.Contains(out, w) {
			t.Fatalf("missing %q in %s", w, out)
		}
	}
}

func TestHistBalanceInputErrors(t *testing.T) {
	cases := map[string]string{
		"2026-03-01":                histbal.ReasonFormat,
		"2026-03-01T12:00:00":       histbal.ReasonFormat,
		"2026-03-01T12:00:00.1234Z": histbal.ReasonPrecision,
		"2026-07-01T12:00:00Z":      histbal.ReasonFuture,
	}
	for raw, reason := range cases {
		f := &fakeHB{}
		vm, out := hbRun(t, raw, f)
		if f.calls != 0 {
			t.Fatalf("%s: reader called", raw)
		}
		if vm.FieldError != reason || !strings.Contains(out, reason) {
			t.Fatalf("%s: %s", raw, out)
		}
		for _, w := range []string{`aria-invalid="true"`, `aria-describedby="histbal-hint histbal-err"`, `value="` + raw + `"`, `Date and time:`} {
			if !strings.Contains(out, w) {
				t.Fatalf("%s: missing %q in %s", raw, w, out)
			}
		}
		if strings.Contains(out, "histbal-result") {
			t.Fatal("result shown on error")
		}
	}
}

func TestHistBalanceOffsetsNormalize(t *testing.T) {
	var got []time.Time
	for _, raw := range []string{"2026-03-01T12:00:00Z", "2026-03-01T14:00:00+02:00", "2026-03-01T07:00:00-05:00"} {
		f := &fakeHB{res: histbal.Result{BalanceMinor: 1250, Decimals: 2, HasPostingsByAsOf: true}}
		vm, out := hbRun(t, raw, f)
		if f.calls != 1 || vm.AsOf != "2026-03-01T12:00:00.000Z" || !strings.Contains(out, "2026-03-01T12:00:00.000Z") {
			t.Fatalf("%s: %s", raw, out)
		}
		got = append(got, f.at)
	}
	if !got[0].Equal(got[1]) || !got[1].Equal(got[2]) || got[1].Location() != time.UTC {
		t.Fatal(got)
	}
}

func TestHistBalanceSigns(t *testing.T) {
	cases := []struct {
		v    int64
		want []string
	}{
		{1250, []string{"+12.50", "In credit", "In credit by 12.50 credits"}},
		{-1250, []string{"−12.50", "In debt", "In debt by 12.50 credits"}},
		{0, []string{"0.00", "Level", "Level, zero credits"}},
	}
	for _, c := range cases {
		f := &fakeHB{res: histbal.Result{BalanceMinor: c.v, Decimals: 2, HasPostingsByAsOf: true}}
		_, out := hbRun(t, "2026-03-01T12:00:00Z", f)
		out = html.UnescapeString(out) // the plus sign is entity-escaped, browsers show it as +
		for _, w := range append(c.want, "posted balance") {
			if !strings.Contains(out, w) {
				t.Fatalf("%d: missing %q in %s", c.v, w, out)
			}
		}
		if strings.Contains(out, "No postings") {
			t.Fatal(out)
		}
	}
}

func TestHistBalanceNoPostings(t *testing.T) {
	f := &fakeHB{res: histbal.Result{BalanceMinor: 999, Decimals: 2}}
	vm, out := hbRun(t, "2026-03-01T12:00:00Z", f)
	if !vm.NoPostings || vm.Balance.State != "Level" || !strings.Contains(out, "0.00") || !strings.Contains(out, "No postings as of this instant") {
		t.Fatal(out)
	}
}

func TestHistBalanceEscapesAndNoCurrentLabels(t *testing.T) {
	f := &fakeHB{}
	_, out := hbRun(t, `"><script>alert(1)</script>`, f)
	if strings.Contains(out, "<script>") || !strings.Contains(out, "&lt;script&gt;") {
		t.Fatal(out)
	}
	f = &fakeHB{res: histbal.Result{BalanceMinor: -5, Decimals: 2, HasPostingsByAsOf: true}}
	_, ok := hbRun(t, "2026-03-01T12:00:00Z", f)
	_, none := hbRun(t, "", f)
	_, bad := hbRun(t, "x", f)
	for _, o := range []string{ok, none, bad} {
		l := strings.ToLower(o)
		for _, w := range []string{"available", "credit limit", "reserved"} {
			if strings.Contains(l, w) {
				t.Fatalf("forbidden %q in %s", w, o)
			}
		}
	}
}
