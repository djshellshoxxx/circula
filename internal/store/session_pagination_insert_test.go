package store_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/djshellshoxxx/circula/internal/domain"
	"github.com/djshellshoxxx/circula/internal/store"
)

// API-036 pagination is live across requests. Inserts above a consumed keyset
// boundary require a refresh; inserts below it may appear in the continuation.
func TestSessionKeysetContinuationAfterCommittedInserts(t *testing.T) {
	s := openStore(t)
	ctx := context.Background()
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	const pageSize = 3
	if _, err := s.DB().ExecContext(ctx, `INSERT INTO persons(id,login,password_hash,created_at)
		VALUES ('owner','owner','hash',0),('foreign','foreign','hash',0)`); err != nil {
		t.Fatal(err)
	}
	type fixture struct {
		id, owner string
		createdAt time.Time
		selector  int
	}
	insert := func(fixtures []fixture) {
		t.Helper()
		tx, err := s.DB().BeginTx(ctx, nil)
		if err != nil {
			t.Fatal(err)
		}
		defer func() { _ = tx.Rollback() }()
		for _, row := range fixtures {
			// Explicit valid selectors make both sides of a tied timestamp
			// deterministic without changing production selector generation.
			if _, err := tx.ExecContext(ctx, `INSERT INTO sessions
				(id,person_id,created_at,last_seen_at,expires_at,public_id) VALUES (?,?,?,?,?,?)`,
				row.id, row.owner, row.createdAt.UnixMilli(), now.UnixMilli(),
				now.Add(time.Hour).UnixMilli(), fmt.Sprintf("%064x", row.selector)); err != nil {
				t.Fatal(err)
			}
		}
		if err := tx.Commit(); err != nil {
			t.Fatal(err)
		}
	}
	original := []fixture{
		{"original-high", "owner", now, 0x90},
		{"original-middle", "owner", now, 0x70},
		{"original-boundary", "owner", now, 0x50},
		{"original-low", "owner", now, 0x30},
		{"original-lowest-tie", "owner", now, 0x10},
		{"original-older", "owner", now.Add(-time.Millisecond), 0xf0},
		{"original-oldest", "owner", now.Add(-2 * time.Millisecond), 0xe0},
	}
	insert(original)
	readPage := func(before *store.SessionPosition, wantIDs []string) []*domain.Session {
		t.Helper()
		var rows []*domain.Session
		// Each page finishes its own View before the next committed write or
		// read. A traversal-wide snapshot would hide the inserted rows.
		if err := s.View(ctx, func(tx store.Tx) error {
			var err error
			rows, err = tx.ListSessions("owner", before, pageSize)
			return err
		}); err != nil {
			t.Fatal(err)
		}
		if len(rows) > pageSize || len(rows) != len(wantIDs) {
			t.Fatalf("page size = %d, want %d (maximum %d)", len(rows), len(wantIDs), pageSize)
		}
		for i, row := range rows {
			if row.PersonID != "owner" || row.IDHash != wantIDs[i] {
				t.Fatalf("page row %d: owner=%q id=%q, want owner=%q id=%q", i, row.PersonID, row.IDHash, "owner", wantIDs[i])
			}
			if i > 0 && (row.CreatedAt.After(rows[i-1].CreatedAt) ||
				row.CreatedAt.Equal(rows[i-1].CreatedAt) && row.PublicID >= rows[i-1].PublicID) {
				t.Fatalf("page is not strictly descending at %q", row.IDHash)
			}
		}
		return rows
	}
	first := readPage(nil, []string{"original-high", "original-middle", "original-boundary"})
	boundary := first[len(first)-1]
	cursor := &store.SessionPosition{CreatedAt: boundary.CreatedAt, PublicID: boundary.PublicID}
	seen := make(map[string]int)
	for _, row := range first {
		seen[row.PublicID]++
	}
	// This transaction commits after the first View has ended. Foreign rows
	// straddle the boundary so both continuation and refresh must scope owner.
	insert([]fixture{
		{"inserted-newer", "owner", now.Add(time.Millisecond), 0x01},
		{"inserted-above", "owner", now, 0xa0},
		{"inserted-below", "owner", now, 0x40},
		{"foreign-newer", "foreign", now.Add(2 * time.Millisecond), 0xb0},
		{"foreign-below", "foreign", now, 0x20},
	})
	for _, wantIDs := range [][]string{
		{"inserted-below", "original-low", "original-lowest-tie"},
		{"original-older", "original-oldest"},
		{},
	} {
		rows := readPage(cursor, wantIDs)
		for _, row := range rows {
			if row.CreatedAt.After(cursor.CreatedAt) ||
				row.CreatedAt.Equal(cursor.CreatedAt) && row.PublicID >= cursor.PublicID {
				t.Fatalf("continuation did not advance strictly past %q: got %q", cursor.PublicID, row.PublicID)
			}
			seen[row.PublicID]++
			if seen[row.PublicID] != 1 {
				t.Fatalf("insert shifted a previously consumed row into continuation: %q", row.IDHash)
			}
			cursor = &store.SessionPosition{CreatedAt: row.CreatedAt, PublicID: row.PublicID}
		}
	}
	for _, row := range original {
		if n := seen[fmt.Sprintf("%064x", row.selector)]; n != 1 {
			t.Errorf("original session %q appeared %d times, want exactly once", row.id, n)
		}
	}
	if len(seen) != len(original)+1 || seen[fmt.Sprintf("%064x", 0x40)] != 1 {
		t.Fatalf("live continuation must add only the newly committed row below the boundary: seen=%v", seen)
	}
	// Above-boundary inserts do not change an in-progress traversal. A fresh
	// first page sees both, with the newer timestamp ahead of the larger ID.
	readPage(nil, []string{"inserted-newer", "inserted-above", "original-high"})
}
