package store_test

import (
	"context"
	"database/sql"
	"fmt"
	"path/filepath"
	"strings"
	"testing"

	"github.com/djshellshoxxx/circula/internal/store"
)

// A3-F12: corrupt migration metadata must cause a startup refusal, not an
// index panic, and must not allow any pending migration to change the schema.
func TestMigrationRejectsNonPositiveRecordedVersion(t *testing.T) {
	for _, version := range []int{0, -1, -17} {
		t.Run(fmt.Sprint(version), func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "invalid-version.db")
			db, err := sql.Open("sqlite", path)
			if err != nil {
				t.Fatal(err)
			}
			defer func() { _ = db.Close() }()
			if _, err := db.Exec(`CREATE TABLE schema_migrations (
				version INTEGER PRIMARY KEY, name TEXT NOT NULL, applied_at INTEGER NOT NULL)`); err != nil {
				t.Fatal(err)
			}
			if _, err := db.Exec(`INSERT INTO schema_migrations VALUES (?, 'corrupt', 123)`, version); err != nil {
				t.Fatal(err)
			}
			const schemaQuery = `SELECT group_concat(type || ':' || name || ':' || COALESCE(sql,''), char(10))
				FROM (SELECT * FROM sqlite_schema ORDER BY type, name)`
			var before string
			if err := db.QueryRow(schemaQuery).Scan(&before); err != nil {
				t.Fatal(err)
			}
			var openErr error
			func() {
				defer func() {
					if v := recover(); v != nil {
						t.Errorf("opening migration version %d panicked: %v", version, v)
					}
				}()
				s, err := store.Open(context.Background(), path)
				openErr = err
				if s != nil {
					_ = s.Close()
					t.Error("invalid migration metadata returned a store")
				}
			}()
			want := fmt.Sprintf("invalid recorded migration version %d", version)
			if openErr == nil || !strings.Contains(openErr.Error(), want) {
				t.Errorf("error = %v, want %q", openErr, want)
			}
			var after string
			if err := db.QueryRow(schemaQuery).Scan(&after); err != nil {
				t.Fatal(err)
			}
			if after != before {
				t.Errorf("startup refusal changed the schema:\nbefore: %s\nafter: %s", before, after)
			}
			var count, gotVersion, appliedAt int
			var name string
			if err := db.QueryRow(`SELECT COUNT(*), MIN(version), MIN(name), MIN(applied_at) FROM schema_migrations`).Scan(
				&count, &gotVersion, &name, &appliedAt); err != nil {
				t.Fatal(err)
			}
			if count != 1 || gotVersion != version || name != "corrupt" || appliedAt != 123 {
				t.Errorf("startup refusal changed migration metadata: %d, %d, %q, %d", count, gotVersion, name, appliedAt)
			}
		})
	}
}
