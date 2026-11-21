package store_test

import (
	"context"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/djshellshoxxx/circula/internal/domain"
	"github.com/djshellshoxxx/circula/internal/store"
)

// GOV-025 e: migration 22 marks historical checkpoints as unversioned without
// inventing a generation timestamp or rewriting any checksum or financial row.
func TestClosureCheckpointMigrationPreservesLegacyRecords(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "v21.db")
	legacy, err := store.OpenAtVersion(ctx, path, 21)
	if err != nil {
		t.Fatal(err)
	}
	const scheduled = int64(1_700_000_000_000)
	const closing = scheduled + 30*24*60*60*1000
	const closed = closing + 1000
	hash := strings.Repeat("0123456789abcdef", 4)
	want := map[string]*domain.Closure{}
	for _, state := range []string{"complete", "awaiting-export", "scheduled"} {
		closedAt, finalHash, netStatus := int64(0), "", "active"
		if state != "scheduled" {
			closedAt, netStatus = closed, "closed"
		}
		if state == "complete" {
			finalHash = hash
		}
		if _, err := legacy.DB().Exec(`INSERT INTO networks(id,name,slug,status,ledger_seq,created_at) VALUES (?,?,?, ?,0,?)`,
			state, state, state, netStatus, scheduled); err != nil {
			t.Fatal(err)
		}
		if _, err := legacy.DB().Exec(`INSERT INTO network_closures(network_id,reason,scheduled_by,scheduled_at,closing_at,notice_days,read_only_days,closed_at,final_sha256)
			VALUES (?,?,?, ?,?,30,90,?,?)`, state, "legacy closure", "administrator", scheduled, closing, closedAt, finalHash); err != nil {
			t.Fatal(err)
		}
		c := &domain.Closure{NetworkID: state, Reason: "legacy closure", ScheduledBy: "administrator",
			ScheduledAt: time.UnixMilli(scheduled).UTC(), ClosingAt: time.UnixMilli(closing).UTC(), NoticeDays: 30, ReadOnlyDays: 90, FinalSHA256: finalHash}
		if closedAt != 0 {
			c.ClosedAt = time.UnixMilli(closedAt).UTC()
		}
		want[state] = c
	}
	if err := legacy.Close(); err != nil {
		t.Fatal(err)
	}
	for pass := 0; pass < 2; pass++ {
		upgraded, err := store.Open(ctx, path)
		if err != nil {
			t.Fatal(err)
		}
		if err := upgraded.View(ctx, func(tx store.Tx) error {
			for id, original := range want {
				got, err := tx.GetClosure(id)
				if err != nil {
					return err
				}
				if !reflect.DeepEqual(got, original) {
					t.Fatalf("migration or restart changed %s: got %+v want %+v", id, got, original)
				}
			}
			return nil
		}); err != nil {
			t.Fatal(err)
		}
		if pass == 0 {
			// A closure that never recorded a checksum can complete its missing
			// phase with explicit version metadata in the same repository write.
			if err := upgraded.Update(ctx, func(tx store.Tx) error {
				return tx.SetClosureFinal("awaiting-export", 1, 0, hash, 0, 0)
			}); err != nil {
				t.Fatal(err)
			}
			want["awaiting-export"].FinalExportVersion = 1
			want["awaiting-export"].FinalSHA256 = hash
		}
		if err := upgraded.Close(); err != nil {
			t.Fatal(err)
		}
	}
}
