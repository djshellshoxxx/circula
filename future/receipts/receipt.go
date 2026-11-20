// Package receipts renders user-requested read-only snapshots. Optional candidate
// certificates use separate enablement and dependencies; integration stays on HOLD.
package receipts

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"
)

const Explanation = "Records a completed Circula credit transfer, not delivery of goods or external payment. Credits are accounting units, not necessarily cash or redeemable currency."

type Snapshot struct {
	Network, Transaction, Payer, Payee, Unit, Status string
	Amount                                           int64
	CompletedAt                                      time.Time
}
type Request struct {
	Network, Member             string
	Active, ExplicitlyRequested bool
}
type Receipt struct {
	Version     int    `json:"version"`
	Network     string `json:"network"`
	Transaction string `json:"transaction"`
	Payer       string `json:"payer"`
	Payee       string `json:"payee"`
	Amount      int64  `json:"amount_minor"`
	Unit        string `json:"unit"`
	CompletedAt string `json:"completed_at"`
	Explanation string `json:"explanation"`
}

func safe(s string) bool {
	if s == "" || len(s) > 256 || !utf8.ValidString(s) {
		return false
	}
	for _, r := range s {
		if unicode.IsControl(r) || unicode.Is(unicode.Cf, r) || unicode.Is(unicode.Zl, r) || unicode.Is(unicode.Zp, r) {
			return false
		}
	}
	return true
}

// Create accepts only an authoritative, authorized database snapshot. No public
// route may deserialize Snapshot from a client. Snapshot status is historical.
func Create(req Request, s Snapshot) (Receipt, error) {
	if !req.ExplicitlyRequested || !req.Active || req.Network != s.Network || (req.Member != s.Payer && req.Member != s.Payee) || s.Payer == s.Payee || s.Status != "completed" || s.Amount <= 0 || s.Amount > 1<<50 || s.CompletedAt.IsZero() || s.CompletedAt.Year() < 1 || s.CompletedAt.Year() > 9999 {
		return Receipt{}, errors.New("receipt unavailable")
	}
	for _, v := range []string{s.Network, s.Transaction, s.Payer, s.Payee, s.Unit} {
		if !safe(v) {
			return Receipt{}, errors.New("invalid snapshot")
		}
	}
	return Receipt{1, s.Network, s.Transaction, s.Payer, s.Payee, s.Amount, s.Unit, s.CompletedAt.UTC().Format(time.RFC3339Nano), Explanation}, nil
}
func (r Receipt) JSON() ([]byte, error) { return json.Marshal(r) }
func (r Receipt) Text() string {
	return strings.Join([]string{"Circula receipt v1", "Network: " + r.Network, "Transaction: " + r.Transaction, "Payer: " + r.Payer, "Payee: " + r.Payee, fmt.Sprintf("Amount (minor units): %d %s", r.Amount, r.Unit), "Completed at: " + r.CompletedAt, r.Explanation}, "\n") + "\n"
}
