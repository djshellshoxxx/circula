package governance

import "testing"

func FuzzAuditStrictJSON(f *testing.F) {
	f.Add([]byte(`{"Action":"vote","Intent":"i"}`))
	f.Add([]byte(`{"Body":"\ud800"}`))
	f.Fuzz(func(t *testing.T, raw []byte) {
		var value struct{ Action, Intent, Body string }
		_ = strictDecode(raw, &value)
	})
}
