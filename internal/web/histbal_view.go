package web

import (
	"bytes"
	"context"
	"errors"
	"html/template"
	"time"

	"github.com/djshellshoxxx/circula/internal/domain"
	"github.com/djshellshoxxx/circula/internal/histbal"
)

// histBalanceExample is the format example shown in the hint (UX-061).
const histBalanceExample = "2026-03-01T12:00:00Z"

// histBalanceVM is the view model for the historical-balance lookup on the
// History screen (UX-061, FR-054). It is presentation only and unwired: no
// route renders it until the service has a timestamp read.
type histBalanceVM struct {
	Raw        string // submitted value, preserved verbatim (escaped by the template)
	FieldError string // reason text for field "at"; empty when valid
	Hint       string
	Submitted  bool // a lookup succeeded; the result fields are meaningful

	Balance     balanceVM
	AsOf        string // normalized UTC instant with Z
	NoPostings  bool
	Explanation string
}

// buildHistBalanceVM validates raw, queries rd only for a valid instant, and
// builds the view model. An empty raw yields the bare form. The zone is never
// inferred: only histbal.ParseInstant decides what is acceptable.
func buildHistBalanceVM(ctx context.Context, raw string, now time.Time, rd histbal.Reader, actor domain.Actor, netID, unit string) (histBalanceVM, error) {
	vm := histBalanceVM{Raw: raw, Hint: "Use a full date and time with Z or a numeric offset, for example " + histBalanceExample + "."}
	if raw == "" {
		return vm, nil
	}
	at, err := histbal.ParseInstant(raw, now)
	if err != nil {
		var ie *histbal.InputError
		if errors.As(err, &ie) {
			vm.FieldError = ie.Reason
			return vm, nil
		}
		return vm, err
	}
	res, err := rd.HistoricalBalance(ctx, actor, netID, at)
	if err != nil {
		return vm, err
	}
	bal := res.BalanceMinor
	if !res.HasPostingsByAsOf {
		bal = 0
		vm.NoPostings = true
	}
	vm.Submitted = true
	vm.Balance = makeBalance(bal, res.Decimals, unit)
	vm.AsOf = histbal.FormatInstant(res.AsOf)
	vm.Explanation = "This is the posted balance of your account at this instant. It counts only postings made up to then."
	return vm, nil
}

var histBalanceTmpl = template.Must(template.New("histbal").Parse(`<section aria-labelledby="histbal-h">
<h2 id="histbal-h">Balance at an exact time</h2>
<form method="get" action="">
<div class="field">
<label for="histbal-at">Date and time (with Z or offset)</label>
<p id="histbal-hint" class="hint">{{.Hint}}</p>
{{if .FieldError}}<p id="histbal-err" class="error" role="alert">Date and time: {{.FieldError}}</p>{{end}}
<input id="histbal-at" name="at" type="text" autocomplete="off" spellcheck="false" value="{{.Raw}}" aria-describedby="histbal-hint{{if .FieldError}} histbal-err{{end}}"{{if .FieldError}} aria-invalid="true"{{end}}>
</div>
<button type="submit">Show balance</button>
</form>
{{if .Submitted}}<div id="histbal-result" aria-live="polite">
<p><span aria-label="{{.Balance.Spoken}}">{{.Balance.Signed}} &middot; {{.Balance.State}}</span></p>
<p>At <time datetime="{{.AsOf}}">{{.AsOf}}</time> (UTC)</p>
{{if .NoPostings}}<p>No postings as of this instant</p>{{end}}
<p>{{.Explanation}}</p>
</div>{{end}}
</section>`))

// renderHistBalance renders the fragment. html/template escapes every value.
func renderHistBalance(vm histBalanceVM) (template.HTML, error) {
	var b bytes.Buffer
	if err := histBalanceTmpl.Execute(&b, vm); err != nil {
		return "", err
	}
	return template.HTML(b.String()), nil
}
