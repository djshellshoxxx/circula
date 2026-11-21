package web

import (
	"strconv"
	"testing"

	"github.com/djshellshoxxx/circula/internal/domain"
	"github.com/djshellshoxxx/circula/internal/web/messages"
)

// The dispute amount detail passes through the shared refusal adapter. Its
// singular wording must still select the precision-specific correction.
func TestRefusePrecisionCatalogue(t *testing.T) {
	for _, tc := range []struct {
		decimals int
		words    string
	}{{0, "0 decimal places"}, {1, "1 decimal place"}, {4, "4 decimal places"}} {
		t.Run(strconv.Itoa(tc.decimals), func(t *testing.T) {
			c := &rc{Net: &domain.Network{Policy: domain.Policy{Decimals: tc.decimals}}}
			err := domain.Invalid("amount", messages.Text("dispute.amount_invalid", tc.decimals))
			summary, fields, ok := c.refuse(err)
			if !ok || summary != "Not saved. Fix the problems below. Nothing changed." || len(fields) != 1 || fields[0].Field != "amount" {
				t.Fatalf("refusal contract changed: %q, %#v, %v", summary, fields, ok)
			}
			want := "Amount: enter the amount to reverse, with up to " + tc.words + ". Use at most " + tc.words + " and try again."
			if fields[0].Text != want {
				t.Fatalf("got %q, want %q", fields[0].Text, want)
			}
		})
	}
	for name, c := range map[string]*rc{"missing network": {}, "missing context": nil} {
		t.Run(name, func(t *testing.T) {
			err := domain.Invalid("amount", messages.Text("dispute.amount_invalid", 1))
			summary, fields, ok := c.refuse(err)
			if !ok || summary != "Not saved. Fix the problems below. Nothing changed." || len(fields) != 1 || fields[0].Field != "amount" {
				t.Fatalf("fallback refusal changed: %q, %#v, %v", summary, fields, ok)
			}
			if want := "Amount: enter the amount to reverse, with up to 1 decimal place. Check the value and try again."; fields[0].Text != want {
				t.Fatalf("got %q, want %q", fields[0].Text, want)
			}
		})
	}
}
