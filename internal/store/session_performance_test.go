package store

import (
	"context"
	"fmt"
	"path/filepath"
	"testing"
	"time"

	"github.com/djshellshoxxx/circula/internal/domain"
)

func retainedSessionStore(tb testing.TB, retained int) *SQLite {
	tb.Helper()
	s, err := Open(context.Background(), filepath.Join(tb.TempDir(), "sessions.db"))
	if err != nil {
		tb.Fatal(err)
	}
	tb.Cleanup(func() { _ = s.Close() })
	if _, err := s.DB().Exec(`INSERT INTO persons(id,login,password_hash,created_at)
		VALUES ('owner','owner','hash',0),('foreign','foreign','hash',0);
		WITH RECURSIVE n(i) AS (VALUES(1) UNION ALL SELECT i+1 FROM n WHERE i < ?)
		INSERT INTO sessions(id,person_id,created_at,last_seen_at,expires_at,revoked_at,public_id)
		SELECT printf('private-%d',i),'owner',i/3,0,
		CASE WHEN i%3=0 THEN 0 ELSE 999999 END,
		CASE WHEN i%3=1 THEN 1 ELSE NULL END,printf('%064x',i) FROM n`, retained); err != nil {
		tb.Fatal(err)
	}
	return s
}

// API-036: exercise real retained storage, including equal timestamps,
// expired/revoked history and deletion of a cursor's boundary row.
func TestSessionKeysetTenThousandRetainedRows(t *testing.T) {
	const retained = 10000
	s := retainedSessionStore(t, retained)
	ctx := context.Background()
	var cursor *SessionPosition
	var previous *domain.Session
	seen := make(map[string]bool, retained)
	for page := 0; page < retained/100+2; page++ {
		var rows []*domain.Session
		if err := s.View(ctx, func(tx Tx) error {
			var err error
			rows, err = tx.ListSessions("owner", cursor, 100)
			return err
		}); err != nil {
			t.Fatal(err)
		}
		if len(rows) > 100 {
			t.Fatalf("repository page is not bounded: %d rows", len(rows))
		}
		for _, row := range rows {
			if row.PersonID != "owner" || seen[row.PublicID] {
				t.Fatal("keyset repeated a row or returned another owner")
			}
			if previous != nil && (row.CreatedAt.After(previous.CreatedAt) ||
				row.CreatedAt.Equal(previous.CreatedAt) && row.PublicID >= previous.PublicID) {
				t.Fatal("keyset order is not strictly descending")
			}
			seen[row.PublicID] = true
			previous = row
		}
		if len(rows) == 0 {
			break
		}
		last := rows[len(rows)-1]
		cursor = &SessionPosition{CreatedAt: last.CreatedAt, PublicID: last.PublicID}
		if page == 0 {
			if _, err := s.DB().Exec(`DELETE FROM sessions WHERE public_id=?`, cursor.PublicID); err != nil {
				t.Fatal(err)
			}
		}
	}
	if len(seen) != retained {
		t.Fatalf("lost sessions around a deleted boundary: got %d want %d", len(seen), retained)
	}
	if err := s.View(ctx, func(tx Tx) error {
		_, err := tx.GetOwnedSession("foreign", fmt.Sprintf("%064x", retained))
		if domain.CodeOf(err) != domain.CodeNotFound {
			t.Fatalf("foreign lookup: %v", err)
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	t.Logf("traversed %d retained sessions in pages of at most 100; boundary deletion and owner isolation passed", retained)
}

// Measure allocations as well as time at fixed output size. Run with
// -bench '^BenchmarkSessionKeyset$' -benchmem; fixture construction is untimed.
func BenchmarkSessionKeyset(b *testing.B) {
	for _, retained := range []int{100, 10000, 100000} {
		b.Run(fmt.Sprintf("retained_%d", retained), func(b *testing.B) {
			s := retainedSessionStore(b, retained)
			ctx := context.Background()
			anchor := max(1, retained/2)
			position := &SessionPosition{CreatedAt: time.UnixMilli(int64(anchor / 3)), PublicID: fmt.Sprintf("%064x", anchor)}
			for _, deep := range []bool{false, true} {
				name := "first"
				var before *SessionPosition
				if deep {
					name, before = "deep", position
				}
				b.Run(name, func(b *testing.B) {
					b.ReportAllocs()
					b.ResetTimer()
					for i := 0; i < b.N; i++ {
						if err := s.View(ctx, func(tx Tx) error {
							rows, err := tx.ListSessions("owner", before, 25)
							if err == nil && len(rows) != 25 {
								b.Fatalf("unexpected page size: %d", len(rows))
							}
							return err
						}); err != nil {
							b.Fatal(err)
						}
					}
				})
			}
		})
	}
}
