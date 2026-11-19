package receipts

import (
	"context"
	"encoding/json"
	"strconv"
	"strings"
	"testing"
)

func numericDocument(t *testing.T, field, number string) string {
	t.Helper()
	_, service := fixture()
	document, err := service.Receipt(context.Background(), Actor{"n", "a"}, DownloadRequest{"tx", "json", true})
	if err != nil {
		t.Fatal(err)
	}
	raw, err := document.JSON()
	if err != nil {
		t.Fatal(err)
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(raw, &fields); err != nil {
		t.Fatal(err)
	}
	fields[field] = json.RawMessage(number)
	raw, err = json.Marshal(fields)
	if err != nil {
		t.Fatal(err)
	}
	return string(raw)
}

func TestDecodeDocumentNumericRoundingRejected(t *testing.T) {
	for _, tc := range []struct {
		name, field, number string
	}{
		{"amount_above_bound", "amount_minor", "1125899906842624.01"},
		{"amount_fraction_above_one", "amount_minor", "1.00000000000000001"},
		{"amount_fraction_below_one", "amount_minor", "0.99999999999999999"},
		{"amount_fraction_exponent", "amount_minor", "100000000000000001e-17"},
		{"revision_above_bound", "revision", "1125899906842624.01"},
		{"revision_fraction_above_one", "revision", "1.00000000000000001"},
		{"revision_fraction_exponent", "revision", "100000000000000001e-17"},
		{"precision_above_bound", "precision", "8.0000000000000001"},
		{"precision_fraction_above_one", "precision", "1.00000000000000001"},
		{"precision_underflow", "precision", "1e-400"},
		{"precision_negative_underflow", "precision", "-1e-400"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			document, err := DecodeDocument(strings.NewReader(numericDocument(t, tc.field, tc.number)))
			if err == nil || document != (Document{}) {
				t.Fatalf("inexact integer token was accepted or exposed a partial document: amount=%d precision=%d revision=%d error=%v", document.Amount, document.Precision, document.Revision, err)
			}
		})
	}
}

func TestDecodeDocumentNumericExactRepresentationsPreserved(t *testing.T) {
	for _, tc := range []struct {
		name, field, number, integer string
	}{
		{"amount_bound", "amount_minor", "1125899906842624", "1125899906842624"},
		{"amount_bound_decimal", "amount_minor", "1125899906842624.0", "1125899906842624"},
		{"amount_bound_exponent", "amount_minor", "112589990684262400e-2", "1125899906842624"},
		{"amount_one_decimal", "amount_minor", "1.0", "1"},
		{"amount_one_exponent", "amount_minor", "10e-1", "1"},
		{"amount_uppercase_exponent", "amount_minor", "1E+0", "1"},
		{"revision_bound", "revision", "1125899906842624", "1125899906842624"},
		{"revision_bound_decimal", "revision", "1125899906842624.000", "1125899906842624"},
		{"revision_bound_exponent", "revision", "1125899906842624000e-3", "1125899906842624"},
		{"revision_one_exponent", "revision", "1e0", "1"},
		{"precision_zero", "precision", "0", "0"},
		{"precision_negative_zero", "precision", "-0", "0"},
		{"precision_zero_decimal", "precision", "0.000", "0"},
		{"precision_negative_zero_decimal", "precision", "-0.0", "0"},
		{"precision_zero_large_exponent", "precision", "0e99999999999999999999", "0"},
		{"precision_bound", "precision", "8", "8"},
		{"precision_bound_decimal", "precision", "8.000", "8"},
		{"precision_bound_exponent", "precision", "800e-2", "8"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			document, err := DecodeDocument(strings.NewReader(numericDocument(t, tc.field, tc.number)))
			if err != nil {
				t.Fatal(err)
			}
			var got string
			switch tc.field {
			case "amount_minor":
				got = strconv.FormatInt(document.Amount, 10)
			case "precision":
				got = strconv.Itoa(document.Precision)
			case "revision":
				got = strconv.FormatUint(document.Revision, 10)
			}
			if got != tc.integer {
				t.Fatalf("exact integer changed: got %s, want %s", got, tc.integer)
			}
			if _, err := document.Canonical(); err != nil {
				t.Fatal(err)
			}
		})
	}
}
