package api

import (
	"encoding/json"
	"strings"
	"testing"
)

// A2-01/A2-18: field names have one spelling and decoder errors expose no Go types.
func TestCanonicalRequestFields(t *testing.T) {
	type amount struct {
		Amount string `json:"amount_minor"`
	}
	type body struct {
		amount
		Nested []amount          `json:"nested"`
		Map    map[string]amount `json:"map"`
		Raw    json.RawMessage   `json:"raw"`
	}
	for _, input := range []string{
		`{"amount_minor":"100","AMOUNT_MINOR":"7"}`,
		`{"AMOUNT_MINOR":"7"}`, `{"Amount":"7"}`,
		`{"nested":[{"AMOUNT_MINOR":"7"}]}`,
		`{"map":{"UserKey":{"Amount_Minor":"7"}}}`,
		`{"amount_minor":7}`, `{"amount_minor":1e9999}`,
		`{"amount_minor":"100","\u0061mount_minor":"7"}`,
	} {
		t.Run(input, func(t *testing.T) {
			var out body
			err := strictUnmarshal([]byte(input), &out)
			if err == nil {
				t.Fatal("accepted non-canonical or invalid field")
			}
			if strings.Contains(err.Error(), "Go struct") || strings.Contains(err.Error(), "api.") || strings.Contains(err.Error(), "float64") {
				t.Fatalf("internal type leaked: %v", err)
			}
		})
	}
	var out body
	if err := strictUnmarshal([]byte(`{"amount_minor":"100","nested":[{"amount_minor":"7"}],"map":{"UserKey":{"amount_minor":"2"}},"raw":{"UserChosenKey":1}}`), &out); err != nil {
		t.Fatalf("canonical fields/free-form keys rejected: %v", err)
	}
	if out.Amount != "100" || out.Nested[0].Amount != "7" {
		t.Fatal("canonical value changed")
	}
}
