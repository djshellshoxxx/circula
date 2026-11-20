package web

import (
	"context"
	"errors"
	"fmt"
	"html"
	"html/template"
	"strings"
	"testing"
	"time"

	"github.com/djshellshoxxx/circula/internal/domain"
	"github.com/djshellshoxxx/circula/internal/histbal"
)

type catalogueHistoricalReader struct {
	err    error
	calls  int
	at     time.Time
	actor  domain.Actor
	netID  string
	result histbal.Result
}

func (f *catalogueHistoricalReader) HistoricalBalance(_ context.Context, actor domain.Actor, netID string, at time.Time) (histbal.Result, error) {
	f.calls++
	f.at, f.actor, f.netID = at, actor, netID
	f.result.AsOf = at
	return f.result, f.err
}

func TestSharedPagesHistoricalReasonMapping(t *testing.T) {
	const unsafe = `unknown 100% %! "><img src=x>`
	for _, reason := range []string{histbal.ReasonMissing, histbal.ReasonFormat, histbal.ReasonPrecision, histbal.ReasonFuture, unsafe, ""} {
		if got := histBalanceReason(reason); got != reason {
			t.Fatalf("English display changed the shared reason/fallback: %q -> %q", reason, got)
		}
	}
	for _, reason := range []string{histbal.ReasonPrecision, histbal.ReasonFuture, unsafe} {
		reader := &catalogueHistoricalReader{err: fmt.Errorf("reader: %w", &histbal.InputError{Reason: reason})}
		vm, err := buildHistBalanceVM(context.Background(), "2026-03-01T12:00:00Z", hbNow, reader, domain.Actor{}, "net-exact", "credit")
		if err != nil || reader.calls != 1 || vm.Submitted || vm.FieldError != reason || vm.Raw != "2026-03-01T12:00:00Z" {
			t.Fatalf("reader refusal changed: %#v, %v, calls %d", vm, err, reader.calls)
		}
		var out strings.Builder
		if err := renderHistBalance(&out, vm); err != nil {
			t.Fatal(err)
		}
		if reason == unsafe && (strings.Contains(out.String(), "<img") || !strings.Contains(out.String(), template.HTMLEscapeString(unsafe))) {
			t.Fatal("unknown reader reason became markup")
		}
	}
	failed := errors.New("reader failure")
	reader := &catalogueHistoricalReader{err: failed}
	if _, err := buildHistBalanceVM(context.Background(), "2026-03-01T12:00:00Z", hbNow, reader, domain.Actor{}, "net-exact", "credit"); !errors.Is(err, failed) {
		t.Fatal("unexpected reader failure was converted to a field message")
	}
}

func TestSharedPagesHistoricalExactValueAndBothLoaders(t *testing.T) {
	const raw = "2026-03-01T14:00:00.123+02:00"
	const instant = "2026-03-01T12:00:00.123Z"
	actor := domain.Actor{PersonID: "exact-person-id"}
	reader := &catalogueHistoricalReader{result: histbal.Result{BalanceMinor: -9007199254740993, Decimals: 2, HasPostingsByAsOf: true}}
	vm, err := buildHistBalanceVM(context.Background(), raw, hbNow, reader, actor, "exact-network-id", "Hours")
	if err != nil || reader.calls != 1 || reader.actor.PersonID != actor.PersonID || reader.netID != "exact-network-id" || reader.at.Location() != time.UTC {
		t.Fatalf("reader call changed: %#v, %v", reader, err)
	}
	if vm.Raw != raw || vm.AsOf != instant || vm.Balance.Signed != "−90071992547409.93" || vm.Balance.Spoken != "In debt by 90071992547409.93 Hours" {
		t.Fatalf("historical presentation changed exact values: %#v", vm)
	}
	var standalone, embedded strings.Builder
	if err := renderHistBalance(&standalone, vm); err != nil {
		t.Fatal(err)
	}
	pages := mustLoadTemplates()
	if err := pages["history"].ExecuteTemplate(&embedded, "content", &page{Slug: "exact-network-id", D: &historyData{HistBal: vm}}); err != nil {
		t.Fatal(err)
	}
	for _, body := range []string{standalone.String(), embedded.String()} {
		body = html.UnescapeString(body)
		for _, text := range []string{`value="` + raw + `"`, `datetime="` + instant + `"`, "−90071992547409.93", "In debt by 90071992547409.93 Hours"} {
			if !strings.Contains(body, text) {
				t.Errorf("historical loading path lost %q", text)
			}
		}
	}
}
