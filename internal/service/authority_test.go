package service_test

import (
	"testing"

	tu "github.com/djshellshoxxx/circula/internal/testutil"
)

// authorityTableRows captures complete stored state around an authorization
// refusal. Callers supply fixed test table names, never external input.
func authorityTableRows(t *testing.T, e *tu.Env, tables ...string) map[string][][]any {
	t.Helper()
	out := make(map[string][][]any, len(tables))
	for _, table := range tables {
		rows, err := e.Store.DB().Query("SELECT * FROM " + table + " ORDER BY rowid")
		if err != nil {
			t.Fatal(err)
		}
		columns, err := rows.Columns()
		if err != nil {
			_ = rows.Close()
			t.Fatal(err)
		}
		out[table] = nil
		for rows.Next() {
			values := make([]any, len(columns))
			pointers := make([]any, len(columns))
			for i := range values {
				pointers[i] = &values[i]
			}
			if err := rows.Scan(pointers...); err != nil {
				_ = rows.Close()
				t.Fatal(err)
			}
			out[table] = append(out[table], values)
		}
		if err := rows.Err(); err != nil {
			_ = rows.Close()
			t.Fatal(err)
		}
		if err := rows.Close(); err != nil {
			t.Fatal(err)
		}
	}
	return out
}
