package api

import (
	"time"

	"github.com/djshellshoxxx/circula/internal/histbal"
)

// histBalanceBody is the API-083 response. It carries no credit limit,
// reserved or available amount: those are current values (FR-054).
type histBalanceBody struct {
	AsOf              string `json:"as_of"`
	BalanceMinor      string `json:"balance_minor"`
	Direction         string `json:"direction"`
	Decimals          int    `json:"decimals"`
	LedgerSeq         int64  `json:"ledger_seq"`
	HasPostingsByAsOf bool   `json:"has_postings_by_as_of"`
}

// histBalanceHandler implements the historical variant of API-044 (API-083).
// It is not registered on any route yet; the service has no timestamp read.
func histBalanceHandler(rd histbal.Reader, now func() time.Time) handlerFunc {
	return func(c *reqCtx) error {
		if err := c.only("at"); err != nil {
			return err
		}
		vals := c.query["at"]
		if len(vals) > 1 {
			return invalidQuery("at must be given once")
		}
		raw := ""
		if len(vals) == 1 {
			raw = vals[0]
		}
		asOf, err := histbal.ParseInstant(raw, now())
		if err != nil {
			if ie, isIn := err.(*histbal.InputError); isIn {
				return invalidQuery("at: " + ie.Reason)
			}
			return err
		}
		res, err := rd.HistoricalBalance(c.r.Context(), c.actor, c.netID, asOf)
		if err != nil {
			return err
		}
		return ok(c, 200, histBalanceBody{
			AsOf: histbal.FormatInstant(res.AsOf), BalanceMinor: minor(res.BalanceMinor),
			Direction: direction(res.BalanceMinor), Decimals: res.Decimals,
			LedgerSeq: res.LedgerSeq, HasPostingsByAsOf: res.HasPostingsByAsOf,
		})
	}
}
