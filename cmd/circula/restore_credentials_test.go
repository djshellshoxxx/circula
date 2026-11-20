package main

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/djshellshoxxx/circula/internal/domain"
	"github.com/djshellshoxxx/circula/internal/service"
	"github.com/djshellshoxxx/circula/internal/store"
	tu "github.com/djshellshoxxx/circula/internal/testutil"
)

// A3-F09 / OPS-020: authentic old backups must not revive sessions, API tokens,
// pending MFA bearer challenges or closed temporary administrator grants.
func TestRestoreInvalidatesOldBearerCredentials(t *testing.T) {
	ctx := context.Background()
	in := qa28Seed(t, t.TempDir())
	t.Cleanup(func() { _ = in.st.Close() })
	actor := in.parties[0][0].actor
	cookie, _ := tu.Must2(in.svc.Login(ctx, "n0m0@example.org", "correct horse battery"))
	token, tokenRow := tu.Must2(in.svc.CreateAPIToken(ctx, actor, service.TokenInput{Name: "restore fixture", Scopes: []domain.Scope{domain.ScopeRead, domain.ScopeWrite}}))
	bg := tu.Must(in.svc.OpenBreakGlass(ctx, domain.SystemActor("test"), service.OpenBreakGlassInput{Network: in.nets[0], Operator: "test operator", Reason: "restore fixture"}))
	tu.Must(in.svc.GrantBreakGlassAdmin(ctx, domain.SystemActor("test"), service.GrantBreakGlassInput{Network: in.nets[0], Login: "n0m0@example.org", Reason: "restore fixture"}))
	now := time.Now().UTC()
	if err := in.st.Update(ctx, func(tx store.Tx) error {
		return tx.InsertMFAChallenge(&domain.MFAChallenge{IDHash: "synthetic-restore-challenge", PersonID: actor.PersonID, CreatedAt: now, ExpiresAt: now.Add(time.Hour)})
	}); err != nil {
		t.Fatal(err)
	}
	tu.Must(in.svc.CreateAdjustment(ctx, in.admins[0], in.nets[0], service.AdjustInput{MemberID: in.parties[0][1].m.ID, Amount: 123, Direction: service.AdjustCredit, Reason: "seed ledger", IdempotencyKey: "restore-fixture"}))
	ledger := restoreRows(t, in.st.DB(), []string{"entries", "transactions", "accounts", "networks"})
	cfg := testConfigFor(in.path)
	manifest, backup := tu.Must2(backupTo(ctx, cfg, cfg.BackupDir, "restore-test", nil))
	if err := in.svc.Logout(ctx, cookie); err != nil {
		t.Fatal(err)
	}
	if err := in.svc.RevokeAPIToken(ctx, actor, tokenRow.ID); err != nil {
		t.Fatal(err)
	}
	tu.Must(in.svc.CloseBreakGlass(ctx, domain.SystemActor("test"), bg.ID, "closed after backup"))
	if err := in.st.Close(); err != nil {
		t.Fatal(err)
	}
	var out, stderr bytes.Buffer
	if code := run([]string{"restore", "-allow-unsigned", "-overwrite", "-database", in.path, "-backup-dir", cfg.BackupDir, backup}, &out, &stderr); code != exitOK {
		t.Fatalf("restore=%d: %s", code, stderr.String())
	}
	st := tu.Must(store.Open(ctx, in.path))
	defer func() { _ = st.Close() }()
	svc := tu.Must(service.New(st, service.Config{Argon: qa28Argon}))
	if _, err := svc.Authenticate(ctx, cookie); !errors.Is(err, domain.ErrUnauthenticated) {
		t.Fatalf("restored session usable: %v", err)
	}
	if _, err := svc.AuthenticateToken(ctx, token); !errors.Is(err, domain.ErrUnauthenticated) {
		t.Fatalf("restored token usable: %v", err)
	}
	if err := svc.RequireBreakGlass(ctx, in.nets[0]); !errors.Is(err, domain.ErrForbidden) {
		t.Fatalf("restored operator authority usable: %v", err)
	}
	if err := st.View(ctx, func(tx store.Tx) error {
		roles, err := tx.LiveBreakGlassRoles(in.nets[0], in.parties[0][0].m.ID, time.Now())
		if err == nil && len(roles) != 0 {
			t.Errorf("restored temporary grants usable: %v", roles)
		}
		return err
	}); err != nil {
		t.Fatal(err)
	}
	if got := restoreRows(t, st.DB(), []string{"entries", "transactions", "accounts", "networks"}); got != ledger {
		t.Fatal("restore credential invalidation changed ledger or network state")
	}
	var challenges, grants, events int
	if err := st.DB().QueryRow(`SELECT (SELECT count(*) FROM mfa_challenges), (SELECT count(*) FROM break_glass_grants), (SELECT count(*) FROM audit_events WHERE action=?)`, service.ActRestoreCredentialsInvalidated).Scan(&challenges, &grants, &events); err != nil {
		t.Fatal(err)
	}
	if challenges != 0 || grants != 1 || events != 1 {
		t.Fatalf("challenges=%d grant history=%d audit=%d", challenges, grants, events)
	}
	if !strings.Contains(out.String(), "restored credentials invalidated: sessions=1 api_tokens=1 mfa_challenges=1 break_glass_sessions=1") {
		t.Fatal(out.String())
	}
	if err := verifyBackupFile(backup, manifest); err != nil {
		t.Fatalf("source backup modified: %v", err)
	}
	checks := tu.Must(verifyCopy(ctx, in.path, cfg))
	if !printChecks(&bytes.Buffer{}, checks) {
		t.Fatal("restored ledger/audit verification failed")
	}
}

// An error or silent ignore while sanitising the staging copy must refuse the
// restore before swap; it must neither mutate the live DB nor report success.
func TestRestoreCredentialInvalidationFailureKeepsCurrentDatabase(t *testing.T) {
	for _, action := range []string{"ABORT, 'injected restore write failure'", "IGNORE"} {
		t.Run(action, func(t *testing.T) {
			ctx := context.Background()
			in := qa28Seed(t, t.TempDir())
			t.Cleanup(func() { _ = in.st.Close() })
			tu.Must2(in.svc.CreateAPIToken(ctx, in.parties[0][0].actor, service.TokenInput{Name: "fixture", Scopes: []domain.Scope{domain.ScopeRead}}))
			if _, err := in.st.DB().Exec(`CREATE TRIGGER refuse_restore_tokens BEFORE UPDATE ON api_tokens BEGIN SELECT RAISE(` + action + `); END`); err != nil {
				t.Fatal(err)
			}
			cfg := testConfigFor(in.path)
			manifest, backup := tu.Must2(backupTo(ctx, cfg, cfg.BackupDir, "fault", nil))
			if _, err := in.st.DB().Exec(`DROP TRIGGER refuse_restore_tokens`); err != nil {
				t.Fatal(err)
			}
			tu.Must(in.svc.RegisterPerson(ctx, service.RegisterInput{Login: "current-only", Password: "correct horse battery"}))
			current := restoreRows(t, in.st.DB(), []string{"persons", "api_tokens", "audit_events", "networks"})
			if err := in.st.Close(); err != nil {
				t.Fatal(err)
			}
			var out, stderr bytes.Buffer
			code := run([]string{"restore", "-allow-unsigned", "-overwrite", "-database", in.path, "-backup-dir", cfg.BackupDir, backup}, &out, &stderr)
			if code == exitOK || !strings.Contains(stderr.String(), "credential invalidation failed") || !strings.Contains(stderr.String(), "current database was not touched") || strings.Contains(out.String(), "restored "+backup) {
				t.Fatalf("restore=%d stdout=%s stderr=%s", code, out.String(), stderr.String())
			}
			st := tu.Must(store.Open(ctx, in.path))
			defer func() { _ = st.Close() }()
			if got := restoreRows(t, st.DB(), []string{"persons", "api_tokens", "audit_events", "networks"}); got != current {
				t.Fatal("failed restore changed current database")
			}
			if err := verifyBackupFile(backup, manifest); err != nil {
				t.Fatalf("source backup modified: %v", err)
			}
			for _, suffix := range []string{"", "-wal", "-shm"} {
				if matches, _ := filepath.Glob(in.path + ".restore-partial" + suffix); len(matches) != 0 {
					t.Fatalf("staging file leaked: %v", matches)
				}
			}
		})
	}
}

func restoreRows(t *testing.T, db *sql.DB, tables []string) string {
	t.Helper()
	out := map[string][][]any{}
	for _, table := range tables {
		rows, err := db.Query("SELECT * FROM " + table + " ORDER BY rowid")
		if err != nil {
			t.Fatal(err)
		}
		cols, err := rows.Columns()
		if err != nil {
			t.Fatal(err)
		}
		for rows.Next() {
			v := make([]any, len(cols))
			p := make([]any, len(cols))
			for i := range v {
				p[i] = &v[i]
			}
			if err := rows.Scan(p...); err != nil {
				t.Fatal(err)
			}
			out[table] = append(out[table], v)
		}
		if err := rows.Err(); err != nil {
			t.Fatal(err)
		}
		if err := rows.Close(); err != nil {
			t.Fatal(err)
		}
	}
	b, err := json.Marshal(out)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}
