package store

import (
	"database/sql"
	"fmt"
	"strconv"
	"testing"

	"github.com/djshellshoxxx/circula/internal/domain"
)

// Constant SELECT rows exercise evidence decoding without changing a ledger,
// a policy history, or any of their immutability guards.
func TestHistoricalCreditEvidenceDecoderRejectsUnusableInputs(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = db.Close() }()
	cases := []struct {
		name    string
		columns map[int]any
		problem bool
		err     bool
	}{
		{name: "forward"},
		{name: "account-system-missing", columns: map[int]any{7: nil}, problem: true},
		{name: "account-system-invalid", columns: map[int]any{7: 2}, problem: true},
		{name: "witness-system-invalid", columns: map[int]any{14: -1}, problem: true},
		{name: "expired-negative-override", columns: map[int]any{15: -10, 16: 0}, problem: true},
		{name: "override-out-of-range", columns: map[int]any{15: domain.MaxPolicyAmount + 1}, problem: true},
		{name: "negative-effective-limit", columns: map[int]any{17: -1}, problem: true},
		{name: "effective-limit-out-of-range", columns: map[int]any{17: domain.MaxPolicyAmount + 1}, problem: true},
		{name: "zero-policy-version", columns: map[int]any{12: 0}, problem: true},
		{name: "negative-policy-version", columns: map[int]any{12: -1}, problem: true},
		{name: "missing-prior-balance", columns: map[int]any{13: nil}, problem: true},
		{name: "expiry-without-override", columns: map[int]any{16: 1000}, problem: true},
		{name: "invalid-legacy-marker", columns: map[int]any{10: 2}, problem: true},
		{name: "wrong-network", columns: map[int]any{9: "different"}, problem: true},
		{name: "invented-legacy-inputs", columns: map[int]any{10: 1}, problem: true},
		{name: "null-policy", columns: map[int]any{18: "null"}, problem: true},
		{name: "array-policy", columns: map[int]any{18: "[]"}, err: true},
		{name: "string-policy", columns: map[int]any{18: `"policy"`}, err: true},
		{name: "malformed-policy", columns: map[int]any{18: "{"}, err: true},
		{name: "null-default-limit", columns: map[int]any{18: `{"default_credit_limit":null}`}, problem: true},
		{name: "negative-default-with-positive-override", columns: map[int]any{15: 100, 18: `{"default_credit_limit":-1}`}, problem: true},
		{name: "out-of-range-default-with-positive-override", columns: map[int]any{15: 100, 18: fmt.Sprintf(`{"default_credit_limit":%d}`, domain.MaxPolicyAmount+1)}, problem: true},
	}
	if strconv.IntSize == 32 {
		cases = append(cases, struct {
			name    string
			columns map[int]any
			problem bool
			err     bool
		}{name: "policy-version-does-not-fit-int", columns: map[int]any{12: int64(1 << 32)}, problem: true})
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			values := []any{"entry", "n", "tx", "a", -1, 1, 100, 0, "entry", "n", 0, 100, 1, 0, 0, nil, nil, 100, `{"default_credit_limit":100}`}
			for column, value := range tc.columns {
				values[column] = value
			}
			r, err := scanCreditLimitHistory(db.QueryRow(`SELECT ?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?`, values...))
			if (err != nil) != tc.err {
				t.Fatalf("decode error = %v, want error=%t", err, tc.err)
			}
			if (r.Problem != "") != tc.problem {
				t.Fatalf("decode problem = %q, want problem=%t", r.Problem, tc.problem)
			}
			if !tc.err && !tc.problem && (r.Check == nil || r.Policy == nil || r.Policy.DefaultCreditLimit != 100) {
				t.Fatalf("valid decision inputs lost: %+v", r)
			}
		})
	}
}

func TestHistoricalCreditEvidenceDecoderPreservesUnknownAndOmittedDefaults(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = db.Close() }()
	legacy, err := scanCreditLimitHistory(db.QueryRow(`SELECT 'entry','n','tx','a',-1,1,100,0,
 'entry','n',1,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL`))
	if err != nil || legacy.Problem != "" || !legacy.LegacyUnverified || legacy.Check != nil || legacy.Policy != nil {
		t.Fatalf("legacy inputs were inferred: %+v err=%v", legacy, err)
	}
	missing, err := scanCreditLimitHistory(db.QueryRow(`SELECT 'entry','n','tx','a',-1,1,100,0,
 'entry','n',0,100,1,0,0,NULL,NULL,0,NULL`))
	if err != nil || missing.Check == nil || missing.Policy != nil {
		t.Fatalf("missing policy was invented: %+v err=%v", missing, err)
	}
	omitted, err := scanCreditLimitHistory(db.QueryRow(`SELECT 'entry','n','tx','a',-1,1,100,0,
 'entry','n',0,100,1,0,0,NULL,NULL,0,'{"dispute_window_days":30}'`))
	if err != nil || omitted.Problem != "" || omitted.Check == nil || omitted.Policy == nil || omitted.Policy.DefaultCreditLimit != domain.DefaultPolicy().DefaultCreditLimit {
		t.Fatalf("legitimate omitted defaults changed: %+v err=%v", omitted, err)
	}
}
