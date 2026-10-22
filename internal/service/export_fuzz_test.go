package service

import (
	"bytes"
	"encoding/csv"
	"strings"
	"testing"
	"unicode/utf8"
)

// QA-032 / SEC-017: no exported text cell can start with a spreadsheet formula
// trigger, the original text is recoverable, and encoding/csv round-trips it.
func FuzzCSVCell(f *testing.F) {
	for _, s := range []string{"", "=1+1", "+1", "-1", "@SUM(A1)", "\tcmd", "\rcmd", "\ncmd", "-25.00", "-", "--1", "-1.", "-.5", "1.5",
		"=cmd|' /C calc'!A0", "\"quoted\"", "a,b", "multi\nline", "multi\r\nline", " =1", "'=1", " =1", "＝1", "-1e5", "-1-1", "+-", "\x00=1"} {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, s string) {
		out := csvCell(s)
		if out != "" {
			switch out[0] {
			case '=', '+', '@', '\t', '\r':
				t.Fatalf("cell %q starts with trigger (from %q)", out, s)
			case '-':
				if !plainNumber(out) {
					t.Fatalf("cell %q starts with '-' and is not a plain number (from %q)", out, s)
				}
			}
		}
		if out != s && out != "'"+s {
			t.Fatalf("cell altered beyond the apostrophe: %q -> %q", s, out)
		}
		if !utf8.ValidString(out) || strings.ContainsAny(out, "\r\x00") {
			return // encoding/csv normalises CR and is not faithful for these
		}
		var buf bytes.Buffer
		w := csv.NewWriter(&buf)
		if err := w.Write([]string{out, "x"}); err != nil {
			t.Fatal(err)
		}
		w.Flush()
		recs, err := csv.NewReader(&buf).ReadAll()
		if err != nil {
			t.Fatalf("reparse %q: %v", out, err)
		}
		if len(recs) != 1 || len(recs[0]) != 2 || recs[0][0] != out {
			t.Fatalf("roundtrip %q -> %q", out, recs)
		}
	})
}

func TestCSVCellNumbersAndNil(t *testing.T) {
	if csvCell(int64(-5)) != "-5" || csvCell(nil) != "" || csvCell("-25.00") != "-25.00" {
		t.Fatal("numbers must stay numeric")
	}
	for _, s := range []string{"=1", "+1", "-1+2", "@a", "\tx", "\rx", "-a"} {
		if got := csvCell(s); got != "'"+s {
			t.Errorf("%q -> %q", s, got)
		}
	}
}
