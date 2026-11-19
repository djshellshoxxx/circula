package recurring

import (
	"bytes"
	"context"
	"database/sql"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"testing"
)

func TestAuditDraftRetryAuthorizesCachedTargetAfterEdit(t *testing.T) {
	s, a := fixture(t, filepath.Join(t.TempDir(), "state.db"))
	ctx := context.Background()
	r := create(t, s, a, "create")
	if _, err := s.CreateDraft(ctx, a, r.ID, "draft"); err != nil {
		t.Fatal(err)
	}
	if _, err := s.DB.Exec(`INSERT INTO fixture_members VALUES('n','c',1)`); err != nil {
		t.Fatal(err)
	}
	edit := record()
	edit.ID = r.ID
	edit.Target = "c"
	if _, err := s.Save(ctx, a, edit, r.Version, "edit"); err != nil {
		t.Fatal(err)
	}
	if _, err := s.DB.Exec(`UPDATE fixture_members SET active=0 WHERE network='n' AND member='b'`); err != nil {
		t.Fatal(err)
	}
	if _, err := s.CreateDraft(ctx, a, r.ID, "draft"); !errors.Is(err, ErrDenied) {
		t.Fatalf("cached old-target draft bypassed revocation: %v", err)
	}
}
func TestAuditDepartureRetiresCachedPrivateResponses(t *testing.T) {
	for _, edited := range []bool{false, true} {
		t.Run(map[bool]string{false: "current-target", true: "historic-target"}[edited], func(t *testing.T) {
			s, a := fixture(t, filepath.Join(t.TempDir(), "state.db"))
			ctx := context.Background()
			r := create(t, s, a, "create")
			if _, err := s.CreateDraft(ctx, a, r.ID, "draft"); err != nil {
				t.Fatal(err)
			}
			if edited {
				if _, err := s.DB.Exec(`INSERT INTO fixture_members VALUES('n','c',1)`); err != nil {
					t.Fatal(err)
				}
				input := record()
				input.ID = r.ID
				input.Target = "c"
				if _, err := s.Save(ctx, a, input, r.Version, "edit"); err != nil {
					t.Fatal(err)
				}
			}
			if err := s.Depart(ctx, "n", "b"); err != nil {
				t.Fatal(err)
			}
			var response string
			for _, key := range []string{"create", "draft"} {
				if err := s.DB.QueryRow(`SELECT response FROM future_template_operations WHERE network='n' AND owner='a' AND key=?`, key).Scan(&response); err != nil || response != "" {
					t.Fatalf("private response survived departure: key=%s response=%s err=%v", key, response, err)
				}
			}
			if _, err := s.Save(ctx, a, record(), 0, "create"); !errors.Is(err, ErrConflict) {
				t.Fatalf("retired creation replay: %v", err)
			}
		})
	}
}
func TestAuditSilentWriteFailuresRollBack(t *testing.T) {
	for _, table := range []string{"future_trade_templates", "future_template_operations", "future_template_audit", "future_template_reminders", "future_template_firings"} {
		t.Run(table, func(t *testing.T) {
			s, a := fixture(t, filepath.Join(t.TempDir(), "state.db"))
			ctx := context.Background()
			var r Record
			if table == "future_template_reminders" || table == "future_template_firings" {
				r = create(t, s, a, "create")
			}
			if _, err := s.DB.Exec(`CREATE TRIGGER audit_ignore BEFORE INSERT ON ` + table + ` BEGIN SELECT RAISE(IGNORE); END`); err != nil {
				t.Fatal(err)
			}
			if r.ID != "" {
				if _, err := s.RunDue(ctx, a.Network, "", 10); !errors.Is(err, ErrStorage) {
					t.Fatalf("silent firing accepted: %v", err)
				}
				var n int
				if err := s.DB.QueryRow(`SELECT COUNT(*) FROM future_template_firings`).Scan(&n); err != nil || n != 0 {
					t.Fatal(n, err)
				}
			} else {
				if _, err := s.Save(ctx, a, record(), 0, "create"); !errors.Is(err, ErrStorage) {
					t.Fatalf("silent mutation accepted: %v", err)
				}
				var n int
				if err := s.DB.QueryRow(`SELECT COUNT(*) FROM future_trade_templates`).Scan(&n); err != nil || n != 0 {
					t.Fatal(n, err)
				}
			}
		})
	}
}
func TestAuditMissingLockAndCorruptRecordFailClosed(t *testing.T) {
	s, a := fixture(t, filepath.Join(t.TempDir(), "state.db"))
	ctx := context.Background()
	r := create(t, s, a, "create")
	if _, err := s.DB.Exec(`UPDATE future_trade_templates SET payload=json_set(payload,'$.Network','other') WHERE network='n' AND id=?`, r.ID); err != nil {
		t.Fatal(err)
	}
	if _, _, err := s.List(ctx, a, "", 10); !errors.Is(err, ErrStorage) {
		t.Fatal("cross-scope payload", err)
	}
	if _, err := s.DB.Exec(`DELETE FROM future_recurring_lock`); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Save(ctx, a, record(), 0, "new"); !errors.Is(err, ErrStorage) {
		t.Fatal("missing lock accepted", err)
	}
}
func TestAuditIncompatibleSchemaRollback(t *testing.T) {
	db, err := sql.Open("sqlite", filepath.Join(t.TempDir(), "state.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if _, err = db.Exec(`CREATE TABLE future_trade_templates(network TEXT NOT NULL,id TEXT NOT NULL,owner TEXT NOT NULL,target TEXT NOT NULL,active INTEGER NOT NULL CHECK(active IN(0,1)),cancelled INTEGER NOT NULL CHECK(cancelled IN(0,1)),version INTEGER NOT NULL CHECK(version>0),payload TEXT NOT NULL)`); err != nil {
		t.Fatal(err)
	}
	if err = (Service{DB: db}).Migrate(context.Background()); !errors.Is(err, ErrStorage) {
		t.Fatal("schema without primary key accepted", err)
	}
	var n int
	if err = db.QueryRow(`SELECT COUNT(*) FROM sqlite_master WHERE name='future_recurring_lock'`).Scan(&n); err != nil || n != 0 {
		t.Fatal("partial migration survived", n, err)
	}
}
func TestAuditStrictJSONCaseAliasesAndSurrogates(t *testing.T) {
	for _, raw := range []string{`{"Action":"save","action":"draft"}`, `{"Record":{"Target":"b","target":"c"}}`, `{"Record":{"Note":"\ud800"}}`, `{"Record":{"Note":"\udc00"}}`, `{"Record":{"Note":"\ud800\u0061"}}`} {
		var c command
		if err := decodeJSON([]byte(raw), &c); err == nil {
			t.Fatal("ambiguous JSON accepted", raw)
		}
	}
	for _, raw := range []string{`{"Record":{"Note":"\ud83d\ude00"}}`, `{"Record":{"Note":"literal \\ud800"}}`} {
		var c command
		if err := decodeJSON([]byte(raw), &c); err != nil {
			t.Fatal(raw, err)
		}
	}
}

func TestAuditIgnoredDepartureRetirementRollsBack(t *testing.T) {
	s, a := fixture(t, filepath.Join(t.TempDir(), "state.db"))
	ctx := context.Background()
	r := create(t, s, a, "create")
	if _, err := s.DB.Exec(`CREATE TRIGGER ignore_retirement BEFORE UPDATE ON future_template_operations BEGIN SELECT RAISE(IGNORE); END`); err != nil {
		t.Fatal(err)
	}
	if err := s.Depart(ctx, "n", "b"); !errors.Is(err, ErrStorage) {
		t.Fatal("ignored privacy cleanup acknowledged", err)
	}
	var cancelled bool
	if err := s.DB.QueryRow(`SELECT cancelled FROM future_trade_templates WHERE network='n' AND id=?`, r.ID).Scan(&cancelled); err != nil || cancelled {
		t.Fatal("partial departure survived", cancelled, err)
	}
}
func TestAuditCachedScopeCorruptionDenied(t *testing.T) {
	s, a := fixture(t, filepath.Join(t.TempDir(), "state.db"))
	ctx := context.Background()
	create(t, s, a, "create")
	if _, err := s.DB.Exec(`UPDATE future_template_operations SET response=json_set(response,'$.Network','other') WHERE network='n' AND owner='a' AND key='create'`); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Save(ctx, a, record(), 0, "create"); !errors.Is(err, ErrStorage) {
		t.Fatal("cached cross-network record returned", err)
	}
}

func TestAuditDirectPreviewNotHiddenBehindCancelledHistory(t *testing.T) {
	s, a := fixture(t, filepath.Join(t.TempDir(), "state.db"))
	r := create(t, s, a, "create")
	ctx := context.Background()
	if err := s.write(ctx, func(tx *sql.Tx) error {
		for i := 0; i < 120; i++ {
			old := r
			old.ID = fmt.Sprintf("!archived-%03d", i)
			old.Active = false
			old.Cancelled = true
			old.Note = ""
			old.Amount = nil
			if err := put(ctx, tx, old); err != nil {
				return err
			}
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	if err := RunCLI(ctx, s, a, []string{"preview", r.ID}, nil, &out); err != nil || !strings.Contains(out.String(), "no automatic trade") {
		t.Fatal(out.String(), err)
	}
	if _, err := s.Get(ctx, Actor{"n", "b"}, r.ID); !errors.Is(err, ErrDenied) {
		t.Fatal("cross-owner get", err)
	}
}
