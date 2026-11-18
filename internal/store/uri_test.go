package store_test

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"testing"

	"github.com/djshellshoxxx/circula/internal/store"
)

// AUD4-D01: a database path is embedded in a SQLite file: URI. Directory
// names containing '?', '#' or '%' (for example an operator's backup
// directory) must open exactly the requested file and keep every pragma the
// store sets, including the E-02 schema boundary.
func TestSQLiteOpensExactPathWithURIMetacharacters(t *testing.T) {
	for _, sub := range []string{"plain", "a?b", "a#b", "a%41b", "a%", "a?&_fk=0&x=", "a?&_pragma=trusted_schema(1)&x="} {
		for _, raw := range []bool{false, true} {
			dir := filepath.Join(t.TempDir(), sub)
			if err := os.MkdirAll(dir, 0o700); err != nil {
				t.Fatal(err)
			}
			path := filepath.Join(dir, "x.db")
			var db *sql.DB
			if raw {
				d, err := store.OpenRaw(path)
				if err != nil {
					t.Fatalf("%q raw: %v", sub, err)
				}
				db = d
			} else {
				st, err := store.Open(context.Background(), path)
				if err != nil {
					t.Fatalf("%q: %v", sub, err)
				}
				db = st.DB()
			}
			var file string
			var trusted, fk, secureDelete int
			row := func(q string, v any) {
				if err := db.QueryRow(q).Scan(v); err != nil {
					t.Fatalf("%q raw=%v %s: %v", sub, raw, q, err)
				}
			}
			row(`SELECT file FROM pragma_database_list WHERE name = 'main'`, &file)
			row(`PRAGMA trusted_schema`, &trusted)
			row(`PRAGMA foreign_keys`, &fk)
			row(`PRAGMA secure_delete`, &secureDelete)
			_ = db.Close()
			want, err := filepath.EvalSymlinks(path)
			if err != nil {
				t.Fatalf("%q raw=%v: requested file was not created: %v", sub, raw, err)
			}
			got, err := filepath.EvalSymlinks(file)
			if err != nil || got != want {
				t.Errorf("%q raw=%v: opened %q, want %q", sub, raw, file, want)
			}
			if trusted != 0 {
				t.Errorf("%q raw=%v: trusted_schema = %d, want 0", sub, raw, trusted)
			}
			if !raw && (fk != 1 || secureDelete != 1) {
				t.Errorf("%q: foreign_keys = %d, secure_delete = %d, want 1 and 1", sub, fk, secureDelete)
			}
		}
	}
}
