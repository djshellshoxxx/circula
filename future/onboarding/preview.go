package onboarding

import (
	"errors"
	"math"
)

type Preview struct {
	PayerBefore, PayerAfter, PayeeBefore, PayeeAfter int64
	Practice                                         bool
}

// PreviewTrade is arithmetic only. The result is explicitly synthetic and cannot
// be converted to a transaction. Limit and reservation inputs are minor units.
func PreviewTrade(payer, payee, limit, reserved, amount int64) (Preview, error) {
	if limit < 0 || reserved < 0 || amount <= 0 || payer < math.MinInt64+amount || payee > math.MaxInt64-amount {
		return Preview{}, errors.New("invalid practice amount")
	}
	after := payer - amount
	// Avoid overflow in the capacity comparison after + limit >= reserved.
	if after < 0 {
		if after < -limit || reserved > limit+after {
			return Preview{}, errors.New("practice limit exceeded")
		}
	} else if reserved > after && reserved-after > limit {
		return Preview{}, errors.New("practice limit exceeded")
	}
	return Preview{payer, after, payee, payee + amount, true}, nil
}
