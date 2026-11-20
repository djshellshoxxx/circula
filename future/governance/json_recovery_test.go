package governance

import (
	"encoding/json"
	"reflect"
	"testing"
)

func TestRecoveryRejectsUnicodeAliasedJSONFields(t *testing.T) {
	type fields struct {
		Revision int
		Proposal string
		Key      int
		Σ        int
		Nested   *struct{ Revision int }
	}
	for _, raw := range []string{
		`{"Revision":1,"Reviſion":2}`,
		`{"Reviſion":1,"Revision":2}`,
		`{"Revision":1,"Revi\u017fion":2}`,
		`{"Proposal":"one","Propoſal":"two"}`,
		`{"revision":1,"REVISION":2}`,
		`{"Key":1,"Key":2}`,
		`{"Σ":1,"ς":2}`,
		`{"Nested":{"Revision":1,"Reviſion":2}}`,
	} {
		t.Run(raw, func(t *testing.T) {
			var permissive fields
			if err := json.Unmarshal([]byte(raw), &permissive); err != nil {
				t.Fatalf("invalid decoder-equivalence fixture: %v", err)
			}
			if permissive.Revision != 2 && permissive.Proposal != "two" && permissive.Key != 2 && permissive.Σ != 2 && (permissive.Nested == nil || permissive.Nested.Revision != 2) {
				t.Fatalf("fixture did not assign the same Go field twice: %+v", permissive)
			}
			var got fields
			if err := strictDecode([]byte(raw), &got); err == nil {
				t.Fatalf("ambiguous JSON field accepted: %+v", got)
			}
		})
	}
}

func TestRecoveryJSONSingleAliasesAndDistinctUnicode(t *testing.T) {
	for _, raw := range []string{`{"Reviſion":7}`, `{"Revi\u017fion":7}`, `{"revision":7}`} {
		var got Command
		if err := strictDecode([]byte(raw), &got); err != nil || got.Revision != 7 {
			t.Fatalf("single existing alias changed: %+v, %v", got, err)
		}
	}
	var got map[string]string
	raw := []byte(`{"Straße":"one","STRASSE":"two","ﬀ":"three","ff":"four","text":"literal � café 北 🧭"}`)
	want := map[string]string{"Straße": "one", "STRASSE": "two", "ﬀ": "three", "ff": "four", "text": "literal � café 北 🧭"}
	if err := strictDecode(raw, &got); err != nil || !reflect.DeepEqual(got, want) {
		t.Fatalf("distinct Unicode names or text changed: %#v, %v", got, err)
	}
	var command Command
	if err := strictDecode([]byte(`{"unknown":1}`), &command); err == nil {
		t.Fatal("unknown field accepted")
	}
}
