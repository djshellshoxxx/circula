package store

import (
	"context"
	"path/filepath"
	"strings"
	"testing"
)

// Assert the actual production query plans. LIMIT alone is insufficient if
// a later page scans all earlier owner rows or sorts an unbounded result.
func TestSessionQueriesUseIndexedKeysetSeeks(t *testing.T) {
	s, err := Open(context.Background(), filepath.Join(t.TempDir(), "sessions.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = s.Close() }()
	if _, err := s.DB().Exec(`INSERT INTO persons(id,login,password_hash,created_at) VALUES ('owner','owner','hash',0);
		WITH RECURSIVE n(i) AS (VALUES(1) UNION ALL SELECT i+1 FROM n WHERE i<2100)
		INSERT INTO sessions(id,person_id,created_at,last_seen_at,expires_at) SELECT printf('private-%d',i),'owner',i/3,0,0 FROM n`); err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name, query, seek string
		args              []any
	}{
		{"first", sessionListFirstSQL, "USING INDEX sessions_owner_page (person_id=?)", []any{"owner", 100}},
		{"continuation", sessionListAfterSQL, "USING INDEX sessions_owner_page (person_id=? AND (created_at,public_id)<(?,?))", []any{"owner", 350, strings.Repeat("a", 64), 100}},
		{"owned", sessionOwnedSQL, "USING INDEX sessions_public_id (public_id=?)", []any{strings.Repeat("a", 64), "owner"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			rows, err := s.DB().Query("EXPLAIN QUERY PLAN "+tc.query, tc.args...)
			if err != nil {
				t.Fatal(err)
			}
			defer func() { _ = rows.Close() }()
			var details []string
			for rows.Next() {
				var id, parent, unused int
				var detail string
				if err := rows.Scan(&id, &parent, &unused, &detail); err != nil {
					t.Fatal(err)
				}
				details = append(details, detail)
			}
			if err := rows.Err(); err != nil {
				t.Fatal(err)
			}
			plan := strings.Join(details, "\n")
			if !strings.Contains(plan, "SEARCH sessions "+tc.seek) || strings.Contains(plan, "SCAN ") || strings.Contains(plan, "TEMP B-TREE") {
				t.Fatalf("session query does not seek directly: %s", plan)
			}
		})
	}
}
