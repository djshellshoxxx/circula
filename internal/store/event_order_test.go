package store_test

import (
	"bytes"
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/djshellshoxxx/circula/internal/domain"
	"github.com/djshellshoxxx/circula/internal/service"
	"github.com/djshellshoxxx/circula/internal/store"
)

// L12 / DAT-014: equal or decreasing clocks and non-sortable event IDs must
// not reorder transaction or dispute histories. The insertion sequence also
// survives VACUUM and cannot be edited through normal SQL writes.
func TestEventOrderIgnoresClockAndIDs(t *testing.T) {
	s := openStore(t)
	net, trade, _, _ := seed(t, s)
	now := time.Now()
	ids := []string{"z-first", "a-second", "m-third"}
	err := s.Update(context.Background(), func(tx store.Tx) error {
		if err := tx.InsertDispute(&domain.Dispute{ID: "d", NetworkID: net, TxID: trade,
			OpenedBy: "mp1", Category: "other", Status: domain.DisputeOpen, OpenedAt: now}); err != nil {
			return err
		}
		for i, id := range ids {
			at := now.Add(-time.Duration(i/2) * time.Hour)
			if err := tx.InsertStatusEvent(&domain.StatusEvent{ID: id, NetworkID: net, TxID: trade,
				From: domain.TxProposed, To: domain.TxCompleted, OccurredAt: at}); err != nil {
				return err
			}
			if err := tx.InsertDisputeEvent(store.DisputeEvent{ID: id, NetworkID: net, DisputeID: "d", Action: "opened", OccurredAt: at}); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	for pass := 0; pass < 2; pass++ {
		err = s.View(context.Background(), func(tx store.Tx) error {
			statuses, err := tx.ListStatusEvents(net, trade)
			if err != nil {
				return err
			}
			var got []string
			for _, e := range statuses {
				got = append(got, e.ID)
			}
			if !reflect.DeepEqual(got, ids) {
				t.Errorf("status order: %v, want %v", got, ids)
			}
			disputes, err := tx.ListDisputeEvents(net, "d")
			if err != nil {
				return err
			}
			got = nil
			for _, e := range disputes {
				got = append(got, e.ID)
			}
			if !reflect.DeepEqual(got, ids) {
				t.Errorf("dispute order: %v, want %v", got, ids)
			}
			return nil
		})
		if err != nil {
			t.Fatal(err)
		}
		if _, err := s.DB().Exec("VACUUM"); err != nil {
			t.Fatal(err)
		}
	}
	for _, table := range []string{"transaction_status_event_order", "dispute_event_order"} {
		for _, q := range []string{"UPDATE " + table + " SET seq = seq + 100", "DELETE FROM " + table,
			"INSERT OR REPLACE INTO " + table + "(seq,event_id) VALUES (99,'z-first')",
			"INSERT OR REPLACE INTO " + table + "(seq,event_id) VALUES (1,'a-second')"} {
			if _, err := s.DB().Exec(q); err == nil {
				t.Errorf("event order mutation allowed: %s", q)
			}
		}
	}
}

type eventOrderClock struct{ at time.Time }

func (c eventOrderClock) Now() time.Time { return c.at }

// GOV-025: closure verification binds export bytes. Changing display history
// ordering must not change the canonical ledger export of a pre-upgrade DB.
func TestEventOrderMigrationPreservesExportChecksum(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "closure-v16.db")
	legacy, err := store.OpenAtVersion(ctx, path, 16)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = legacy.Close() })
	seedV1Ledger(t, legacy.DB())
	for _, id := range []string{"z-first", "a-last"} {
		if _, err := legacy.DB().Exec(`INSERT INTO transaction_status_events(id,network_id,tx_id,from_status,to_status,occurred_at)
			VALUES (?,'n1','t1','proposed','completed',1)`, id); err != nil {
			t.Fatal(err)
		}
	}
	export := func(st *store.SQLite, format service.ExportFormat) (*service.Manifest, *service.MemSink) {
		t.Helper()
		svc, err := service.New(st, service.Config{Clock: eventOrderClock{time.Unix(1700000000, 0)}})
		if err != nil {
			t.Fatal(err)
		}
		sink := &service.MemSink{}
		man, err := svc.Export(ctx, domain.SystemActor("export-compatibility"), "n1", service.ExportLedger, format, sink)
		if err != nil {
			t.Fatal(err)
		}
		return man, sink
	}
	formats := []service.ExportFormat{service.FormatJSON, service.FormatCSV}
	manifests := map[service.ExportFormat]*service.Manifest{}
	sinks := map[service.ExportFormat]*service.MemSink{}
	for _, format := range formats {
		manifests[format], sinks[format] = export(legacy, format)
	}
	if err := legacy.Close(); err != nil {
		t.Fatal(err)
	}
	upgraded, err := store.Open(ctx, path)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = upgraded.Close() }()
	for _, format := range formats {
		man, sink := export(upgraded, format)
		if got, want := service.ManifestChecksum(man), service.ManifestChecksum(manifests[format]); got != want {
			t.Errorf("%s checksum changed after migration: %s != %s", format, got, want)
		}
		for _, file := range manifests[format].Files {
			if !bytes.Equal(sink.Bytes(file.Name), sinks[format].Bytes(file.Name)) {
				t.Errorf("%s bytes changed after migration", file.Name)
			}
		}
	}
}

func TestEventOrderMigrationPreservesLegacyInsertionOrder(t *testing.T) {
	path := filepath.Join(t.TempDir(), "v16.db")
	db, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = db.Close() }()
	files, err := filepath.Glob("migrations/*.sql")
	if err != nil {
		t.Fatal(err)
	}
	sort.Strings(files)
	if _, err := db.Exec(`CREATE TABLE schema_migrations (version INTEGER PRIMARY KEY, name TEXT NOT NULL, applied_at INTEGER NOT NULL)`); err != nil {
		t.Fatal(err)
	}
	for i, f := range files[:16] {
		b, err := os.ReadFile(f)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := db.Exec(string(b)); err != nil {
			t.Fatal(err)
		}
		name := strings.SplitN(strings.TrimSuffix(filepath.Base(f), ".sql"), "_", 2)[1]
		if _, err := db.Exec("INSERT INTO schema_migrations VALUES (?,?,0)", i+1, name); err != nil {
			t.Fatal(err)
		}
	}
	seedV1Ledger(t, db)
	for _, id := range []string{"z-old", "a-old"} {
		if _, err := db.Exec(`INSERT INTO transaction_status_events(id,network_id,tx_id,from_status,to_status,occurred_at)
			VALUES (?,'n1','t1','proposed','completed',1)`, id); err != nil {
			t.Fatal(err)
		}
	}
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}
	s, err := store.Open(context.Background(), path)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = s.Close() }()
	var ordered string
	if err := s.DB().QueryRow(`SELECT group_concat(event_id, ',') FROM
		(SELECT event_id FROM transaction_status_event_order WHERE event_id IN ('z-old','a-old') ORDER BY seq)`).Scan(&ordered); err != nil {
		t.Fatal(err)
	}
	if ordered != "z-old,a-old" {
		t.Fatalf("legacy order changed: %s", ordered)
	}
}
