package store_test

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/djshellshoxxx/circula/internal/store"
)

func TestInspectSchemaAndPragmas(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "c.db")

	st, err := store.InspectSchema(ctx, path)
	if err != nil || st.Exists || st.Applied != 0 || st.Known != store.SchemaVersion() {
		t.Fatalf("missing file: %+v err=%v", st, err)
	}
	if st.Pending() != st.Known {
		t.Errorf("pending = %d", st.Pending())
	}

	s, err := store.Open(ctx, path)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = s.Close() }()
	if err := store.CheckPragmas(ctx, s.DB()); err != nil {
		t.Fatalf("pragmas: %v", err)
	}
	p, err := store.ReadPragmas(ctx, s.DB())
	if err != nil || p.JournalMode != "wal" || !p.ForeignKeys {
		t.Fatalf("pragmas = %+v err=%v", p, err)
	}
	st, err = store.InspectSchema(ctx, path)
	if err != nil || !st.Exists || st.Pending() != 0 || st.CheckNotNewer() != nil {
		t.Fatalf("migrated file: %+v err=%v", st, err)
	}
	newer := store.SchemaState{Exists: true, Applied: st.Known + 1, Known: st.Known}
	if newer.CheckNotNewer() == nil {
		t.Error("newer schema accepted")
	}
	nets, err := store.ListNetworks(ctx, s.DB(), "")
	if err != nil || len(nets) != 0 {
		t.Fatalf("networks = %v err=%v", nets, err)
	}
	if v, err := store.AppliedVersion(ctx, s.DB()); err != nil || v != st.Known {
		t.Fatalf("applied = %d err=%v", v, err)
	}
	if res, err := store.IntegrityCheck(ctx, path); err != nil || res != "ok" {
		t.Fatalf("integrity = %q err=%v", res, err)
	}
}

func TestCheckPragmasRejectsRollbackJournal(t *testing.T) {
	ctx := context.Background()
	db, err := store.OpenRaw(filepath.Join(t.TempDir(), "raw.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = db.Close() }()
	db.SetMaxOpenConns(1)
	if err := store.CheckPragmas(ctx, db); err == nil {
		t.Fatal("raw connection (no WAL, no foreign keys) accepted")
	}
}
