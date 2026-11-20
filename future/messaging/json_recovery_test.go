package messaging

import (
	"encoding/json"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
)

func recoveryDecode(raw string, out any) error {
	r := httptest.NewRequest("POST", "/api/v1/conversations/t/read", strings.NewReader(raw))
	r.Header.Set("Content-Type", "application/json")
	defer r.Body.Close()
	return decode(r, out)
}

func TestRecoveryRejectsUnicodeAliasedJSONFields(t *testing.T) {
	type fields struct {
		Sequence int
		Reason   string
		Key      int
		Σ        int
		Nested   *struct{ Sequence int }
	}
	for _, raw := range []string{
		`{"Sequence":1,"ſequence":2}`,
		`{"ſequence":1,"Sequence":2}`,
		`{"Sequence":1,"\u017fequence":2}`,
		`{"Reason":"one","Reaſon":"two"}`,
		`{"sequence":1,"SEQUENCE":2}`,
		`{"Key":1,"Key":2}`,
		`{"Σ":1,"ς":2}`,
		`{"Nested":{"Sequence":1,"ſequence":2}}`,
	} {
		t.Run(raw, func(t *testing.T) {
			var permissive fields
			if err := json.Unmarshal([]byte(raw), &permissive); err != nil {
				t.Fatalf("invalid decoder-equivalence fixture: %v", err)
			}
			if permissive.Sequence != 2 && permissive.Reason != "two" && permissive.Key != 2 && permissive.Σ != 2 && (permissive.Nested == nil || permissive.Nested.Sequence != 2) {
				t.Fatalf("fixture did not assign the same Go field twice: %+v", permissive)
			}
			var got fields
			if err := recoveryDecode(raw, &got); err == nil {
				t.Fatalf("ambiguous JSON field accepted: %+v", got)
			}
		})
	}
}

func TestRecoveryJSONSingleAliasesAndDistinctUnicode(t *testing.T) {
	for _, raw := range []string{`{"ſequence":7}`, `{"\u017fequence":7}`, `{"sequence":7}`} {
		var got struct{ Sequence int }
		if err := recoveryDecode(raw, &got); err != nil || got.Sequence != 7 {
			t.Fatalf("single existing alias changed: %+v, %v", got, err)
		}
	}
	var got map[string]string
	raw := `{"Straße":"one","STRASSE":"two","ﬀ":"three","ff":"four","text":"literal � café 北 🧭"}`
	want := map[string]string{"Straße": "one", "STRASSE": "two", "ﬀ": "three", "ff": "four", "text": "literal � café 北 🧭"}
	if err := recoveryDecode(raw, &got); err != nil || !reflect.DeepEqual(got, want) {
		t.Fatalf("distinct Unicode names or text changed: %#v, %v", got, err)
	}
	var known struct{ Sequence int }
	if err := recoveryDecode(`{"unknown":1}`, &known); err == nil {
		t.Fatal("unknown field accepted")
	}
}
