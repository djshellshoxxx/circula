package receipts

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"io"
	"strings"
	"time"
)

var ErrUnavailable = errors.New("receipt unavailable")

// Actor is supplied by the host's existing authentication boundary, never from
// client JSON. There is no role field: only Reader may grant scoped admin access.
type Actor struct{ Network, Member string }

// Trade is a minimal authorized snapshot from an existing read transaction. It
// contains no dispute text, contact details, account balance or other history.
type Trade struct {
	Snapshot
	Precision              int
	PayerEntry, PayeeEntry string
	Revision               uint64
	Supplement             *StatusSupplement
}

// StatusSupplement describes only an authorized later event. It does not rewrite
// the completed snapshot. Its event reference and time come from the same read.
type StatusSupplement struct {
	Reference  string `json:"reference"`
	Status     string `json:"status"`
	ObservedAt string `json:"observed_at"`
}

// Reader must reauthorize current active parties or explicitly scoped admins and
// read the immutable completion and authorized later status atomically. General
// transaction-history visibility (such as dispute reviewer access) is insufficient.
// It must return the same error for missing, denied and other-network records.
type Reader interface {
	ReadReceipt(context.Context, Actor, string) (Trade, error)
}
type Audit interface {
	ReceiptOutcome(context.Context, string) error
}
type Service struct {
	Reader         Reader
	Enabled        bool
	NetworkEnabled func(context.Context, string) bool
	Audit          Audit
}
type DownloadRequest struct {
	Reference, Format string
	Explicit          bool
}

// Document v2 freezes historical completion separately from current supplements.
// It is an ephemeral projection; neither this package nor its adapter stores files.
type Document struct {
	Schema        string            `json:"schema"`
	Network       string            `json:"network"`
	Transaction   string            `json:"transaction"`
	Payer         string            `json:"payer"`
	Payee         string            `json:"payee"`
	Amount        int64             `json:"amount_minor"`
	Precision     int               `json:"precision"`
	Unit          string            `json:"unit"`
	Status        string            `json:"status"`
	CompletedAt   string            `json:"completed_at"`
	LedgerEntries [2]string         `json:"ledger_entries"`
	Revision      uint64            `json:"revision"`
	Explanation   string            `json:"explanation"`
	Supplement    *StatusSupplement `json:"status_supplement,omitempty"`
}

func actorValid(a Actor) bool { return safe(a.Network) && safe(a.Member) }
func project(a Actor, ref string, t Trade) (Document, error) {
	s := t.Snapshot
	if s.Network != a.Network || s.Transaction != ref || s.Payer == s.Payee || s.Status != "completed" || s.Amount <= 0 || s.Amount > 1<<50 || s.CompletedAt.IsZero() || s.CompletedAt.Year() < 1 || s.CompletedAt.Year() > 9999 || t.Precision < 0 || t.Precision > 8 || t.Revision == 0 || t.Revision > 1<<50 || t.PayerEntry == t.PayeeEntry {
		return Document{}, ErrUnavailable
	}
	for _, v := range []string{s.Network, s.Transaction, s.Payer, s.Payee, s.Unit, t.PayerEntry, t.PayeeEntry} {
		if !safe(v) {
			return Document{}, ErrUnavailable
		}
	}
	var supplement *StatusSupplement
	if t.Supplement != nil {
		v := *t.Supplement
		observed, err := time.Parse(time.RFC3339Nano, v.ObservedAt)
		if !safe(v.Reference) || v.Reference == ref || (v.Status != "reversed" && v.Status != "adjusted") || err != nil || observed.Before(s.CompletedAt) {
			return Document{}, ErrUnavailable
		}
		v.ObservedAt = observed.UTC().Format(time.RFC3339Nano)
		supplement = &v
	}
	return Document{"circula-receipt/v2", s.Network, s.Transaction, s.Payer, s.Payee, s.Amount, t.Precision, s.Unit, "completed", s.CompletedAt.UTC().Format(time.RFC3339Nano), [2]string{t.PayerEntry, t.PayeeEntry}, t.Revision, Explanation + " Mutual-credit record; not bank/cash settlement.", supplement}, nil
}

// Receipt never signs, stores, sends or posts a trade. Disabled/absent-consent
// paths do not invoke Reader or Audit. Missing/denied/inconsistent share one error.
func (s Service) Receipt(ctx context.Context, a Actor, q DownloadRequest) (Document, error) {
	return s.receipt(ctx, a, q, true)
}

// receipt also supports a final authorization-only projection during signing.
// That recheck must not double-count the original fixed-label audit event.
func (s Service) receipt(ctx context.Context, a Actor, q DownloadRequest, audit bool) (Document, error) {
	if ctx.Err() != nil {
		return Document{}, ctx.Err()
	}
	if !q.Explicit || !s.Enabled || s.NetworkEnabled == nil || !actorValid(a) || !safe(q.Reference) || s.Reader == nil || !s.NetworkEnabled(ctx, a.Network) {
		return Document{}, ErrUnavailable
	}
	if ctx.Err() != nil {
		return Document{}, ctx.Err()
	}
	t, err := s.Reader.ReadReceipt(ctx, a, q.Reference)
	if ctx.Err() != nil {
		return Document{}, ctx.Err()
	}
	if !s.NetworkEnabled(ctx, a.Network) {
		return Document{}, ErrUnavailable
	}
	if ctx.Err() != nil {
		return Document{}, ctx.Err()
	}
	if err != nil {
		if audit && s.Audit != nil {
			s.Audit.ReceiptOutcome(ctx, "unavailable")
		}
		return Document{}, ErrUnavailable
	}
	d, err := project(a, q.Reference, t)
	if err != nil {
		return Document{}, ErrUnavailable
	}
	if audit && s.Audit != nil {
		if err = s.Audit.ReceiptOutcome(ctx, "download"); err != nil {
			return Document{}, ErrUnavailable
		}
	}
	if ctx.Err() != nil {
		return Document{}, ctx.Err()
	}
	if !s.NetworkEnabled(ctx, a.Network) {
		return Document{}, ErrUnavailable
	}
	if ctx.Err() != nil {
		return Document{}, ctx.Err()
	}
	return d, nil
}
func (d Document) JSON() ([]byte, error) { return json.Marshal(d) }
func (d Document) Text() string {
	var b strings.Builder
	fmt.Fprintf(&b, "Circula receipt v2\nNetwork: %s\nTransaction: %s\nPayer: %s\nPayee: %s\nAmount (minor units): %d %s\nPrecision: %d\nStatus: %s\nCompleted at: %s\nLedger entries: %s, %s\nRevision: %d\n%s\n", d.Network, d.Transaction, d.Payer, d.Payee, d.Amount, d.Unit, d.Precision, d.Status, d.CompletedAt, d.LedgerEntries[0], d.LedgerEntries[1], d.Revision, d.Explanation)
	if d.Supplement != nil {
		fmt.Fprintf(&b, "Later status supplement: %s; reference %s; observed %s\n", d.Supplement.Status, d.Supplement.Reference, d.Supplement.ObservedAt)
	}
	return b.String()
}

var printTemplate = template.Must(template.New("receipt").Parse(`<!doctype html><html lang="en"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width"><title>Circula trade receipt</title><style>body{font-family:system-ui,sans-serif;max-width:48rem;margin:2rem auto;padding:1rem}dt{font-weight:bold}dd{margin:0 0 1rem}p{line-height:1.5}@media print{body{margin:0}}</style></head><body><main><h1>Circula trade receipt</h1><p>{{.Explanation}}</p><dl><dt>Network</dt><dd>{{.Network}}</dd><dt>Transaction</dt><dd>{{.Transaction}}</dd><dt>Payer</dt><dd>{{.Payer}}</dd><dt>Payee</dt><dd>{{.Payee}}</dd><dt>Amount in minor units</dt><dd>{{.Amount}} {{.Unit}} (precision {{.Precision}})</dd><dt>Status at completion</dt><dd>{{.Status}}</dd><dt>Completed at (UTC)</dt><dd>{{.CompletedAt}}</dd><dt>Ledger entries</dt><dd>{{index .LedgerEntries 0}}, {{index .LedgerEntries 1}}</dd><dt>Receipt revision</dt><dd>{{.Revision}}</dd></dl>{{with .Supplement}}<section aria-label="Later status supplement"><h2>Later status supplement</h2><p>{{.Status}}; reference {{.Reference}}; observed {{.ObservedAt}}.</p></section>{{end}}<p>This is a historical snapshot. Downloaded copies cannot be remotely recalled. This receipt has no cryptographic certificate.</p></main></body></html>`))

func (d Document) HTML(w io.Writer) error { return printTemplate.Execute(w, d) }
func (d Document) Export(format string) ([]byte, string, error) {
	switch format {
	case "txt":
		return []byte(d.Text()), "text/plain; charset=utf-8", nil
	case "json":
		v, e := d.JSON()
		return v, "application/json", e
	case "html":
		var b bytes.Buffer
		e := d.HTML(&b)
		return b.Bytes(), "text/html; charset=utf-8", e
	default:
		return nil, "", errors.New("unsupported receipt format")
	}
}

// Download is shared by CLI/API adapters. It requests no client-supplied facts.
func (s Service) Download(ctx context.Context, a Actor, q DownloadRequest, w io.Writer) error {
	if q.Format != "txt" && q.Format != "json" && q.Format != "html" {
		return errors.New("unsupported receipt format")
	}
	d, e := s.Receipt(ctx, a, q)
	if e != nil {
		return e
	}
	body, _, e := d.Export(q.Format)
	if e != nil {
		return e
	}
	if ctx.Err() != nil {
		return ctx.Err()
	}
	_, e = w.Write(body)
	return e
}
