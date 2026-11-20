package recurring

import "testing"

func TestRecoveredJSONSimpleFoldDuplicates(t *testing.T) {
	for _, raw := range []string{
		`{"Version":1,"Verſion":2}`,
		`{"Verſion":1,"Version":2}`,
		`{"Version":1,"Ver\u017fion":2}`,
		`{"Key":1,"Key":2}`,
		`{"Record":{"Version":1,"Verſion":2}}`,
	} {
		t.Run(raw, func(t *testing.T) {
			var out struct {
				Version int
				Key     int
				Record  struct{ Version int }
			}
			if err := decodeJSON([]byte(raw), &out); err == nil {
				t.Fatalf("decoder-equivalent duplicate names accepted: %s -> %+v", raw, out)
			}
		})
	}
}

func TestRecoveredJSONSingleAliasesAndUnknownFields(t *testing.T) {
	for _, raw := range []string{`{"Version":7}`, `{"version":7}`, `{"Verſion":7}`, `{"Ver\u017fion":7}`} {
		var out struct{ Version int }
		if err := decodeJSON([]byte(raw), &out); err != nil || out.Version != 7 {
			t.Fatalf("valid single alias changed: %s -> %+v, %v", raw, out, err)
		}
	}
	var out struct{ Version int }
	if err := decodeJSON([]byte(`{"Versions":7}`), &out); err == nil {
		t.Fatal("unknown field accepted")
	}
	// Unicode simple-fold equivalence does not expand one rune into several.
	var names map[string]int
	if err := decodeJSON([]byte(`{"Straße":1,"STRASSE":2}`), &names); err != nil || len(names) != 2 {
		t.Fatalf("multi-rune case expansion collapsed distinct names: %+v, %v", names, err)
	}
	var separate []struct{ Version int }
	if err := decodeJSON([]byte(`[{"Version":1},{"Verſion":2}]`), &separate); err != nil || len(separate) != 2 || separate[0].Version != 1 || separate[1].Version != 2 {
		t.Fatalf("duplicate scope leaked between objects: %+v, %v", separate, err)
	}
}
