package main

import (
	"bufio"
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/djshellshoxxx/circula/internal/service"
	"github.com/djshellshoxxx/circula/internal/store"
)

// operatorPasswordEnv carries the operator password for admin create-operator.
// The password is never accepted as a command-line argument.
const operatorPasswordEnv = envPrefix + "OPERATOR_PASSWORD"

// openDatabase opens the configured database after checking that it exists
// and has no pending migrations. Commands that must not migrate use it.
func openDatabase(ctx context.Context, cfg *Config) (*store.SQLite, *service.Service, error) {
	state, err := inspectSchema(ctx, cfg.Database)
	if err != nil {
		return nil, nil, err
	}
	if !state.Exists {
		return nil, nil, fmt.Errorf("database %s does not exist", cfg.Database)
	}
	if err := state.CheckNotNewer(); err != nil {
		return nil, nil, err
	}
	if state.Pending() > 0 {
		return nil, nil, fmt.Errorf("database has %d pending migration(s); run `circula migrate` first", state.Pending())
	}
	st, err := store.Open(ctx, cfg.Database)
	if err != nil {
		return nil, nil, err
	}
	svc, err := newService(st, cfg)
	if err != nil {
		_ = st.Close()
		return nil, nil, err
	}
	return st, svc, nil
}

// resolveFlags parses a command's flags with the shared settings and returns
// the configuration. done reports that the caller must return code.
func resolveFlags(name string, args []string, stderr io.Writer, extra func(*flag.FlagSet)) (cfg *Config, rest []string, code int, done bool) {
	fs := newFlagSet(name, stderr)
	sf := bindSettingFlags(fs)
	if extra != nil {
		extra(fs)
	}
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil, nil, exitOK, true
		}
		return nil, nil, exitUsage, true
	}
	c, _, err := sf.Resolve(os.LookupEnv, false)
	if err != nil {
		fmt.Fprintf(stderr, "circula: invalid configuration:\n%v\n", err)
		return nil, nil, exitUsage, true
	}
	return c, fs.Args(), exitOK, false
}

func printChecks(w io.Writer, checks []networkCheck) bool {
	ok := true
	for _, c := range checks {
		if c.OK {
			fmt.Fprintf(w, "network %s: ok\n", c.Slug)
			continue
		}
		ok = false
		fmt.Fprintf(w, "network %s: FAILED\n", c.Slug)
		for _, f := range c.Findings {
			fmt.Fprintf(w, "  - %s\n", f)
		}
	}
	if len(checks) == 0 {
		fmt.Fprintln(w, "no networks to verify")
	}
	return ok
}

func cmdMigrate(ctx context.Context, args []string, stdout, stderr io.Writer) int {
	var dryRun bool
	cfg, _, code, done := resolveFlags("migrate", args, stderr, func(fs *flag.FlagSet) {
		fs.BoolVar(&dryRun, "dry-run", false, "report pending migrations without changing anything")
	})
	if done {
		return code
	}
	ctx, stop := withInterrupt(ctx)
	defer stop()
	state, err := inspectSchema(ctx, cfg.Database)
	if err != nil {
		fmt.Fprintf(stderr, "circula: migrate: %v\n", err)
		return exitFailure
	}
	if err := state.CheckNotNewer(); err != nil {
		fmt.Fprintf(stderr, "circula: migrate: %v\n", err)
		return exitFailure
	}
	if state.Pending() == 0 {
		fmt.Fprintf(stdout, "schema is up to date at v%d; nothing to do\n", state.Applied)
		return exitOK
	}
	fmt.Fprintf(stdout, "pending migrations: v%d to v%d\n", state.Applied, state.Known)
	if dryRun {
		return exitOK
	}
	lock, err := acquireLock(lockPath(cfg.Database))
	if err != nil {
		return reportLockError(stderr, err, cfg.Database)
	}
	defer func() { _ = lock.Release() }()
	if state.Exists && state.Applied > 0 {
		mf, path, err := preMigrationSafeguard(ctx, cfg, state)
		if err != nil {
			fmt.Fprintf(stderr, "circula: %v\n", err)
			return exitFailure
		}
		fmt.Fprintf(stdout, "pre-migration backup: %s (verified: checksum, integrity ok, schema v%d; ledger: %s)\n",
			path, mf.SchemaVersion, mf.Ledger.Status)
	}
	if _, err := secureDatabase(cfg.Database); err != nil {
		fmt.Fprintf(stderr, "circula: migrate: %v\n", err)
		return exitFailure
	}
	st, err := store.Open(ctx, cfg.Database)
	if err != nil {
		fmt.Fprintf(stderr, "circula: migrate failed (database left at its previous version): %v\n", err)
		return exitFailure
	}
	defer func() { _ = st.Close() }()
	if err := store.CheckPragmas(ctx, st.DB()); err != nil {
		fmt.Fprintf(stderr, "circula: %v\n", err)
		return exitFailure
	}
	fmt.Fprintf(stdout, "migrated to v%d\n", store.SchemaVersion())
	svc, err := newService(st, cfg)
	if err != nil {
		fmt.Fprintf(stderr, "circula: %v\n", err)
		return exitFailure
	}
	checks, err := verifyAllNetworks(ctx, svc, st.DB(), "migrate-verify", "")
	if err != nil {
		fmt.Fprintf(stderr, "circula: verify after migration: %v\n", err)
		return exitFailure
	}
	if !printChecks(stdout, checks) {
		fmt.Fprintln(stderr, "circula: ledger verification failed after migration; restore the pre-migration backup (OPS-023)")
		return exitLedgerFailed
	}
	return exitOK
}

func cmdBackup(ctx context.Context, args []string, stdout, stderr io.Writer) int {
	cfg, rest, code, done := resolveFlags("backup", args, stderr, nil)
	if done {
		return code
	}
	if len(rest) != 1 {
		fmt.Fprintln(stderr, "usage: circula backup [flags] <dest-file-or-directory>")
		return exitUsage
	}
	ctx, stop := withInterrupt(ctx)
	defer stop()
	state, err := inspectSchema(ctx, cfg.Database)
	if err != nil {
		fmt.Fprintf(stderr, "circula: backup: %v\n", err)
		return exitFailure
	}
	if !state.Exists {
		fmt.Fprintf(stderr, "circula: backup: database %s does not exist\n", cfg.Database)
		return exitFailure
	}
	now := time.Now().UTC()
	dest := rest[0]
	if fi, err := os.Stat(dest); err == nil && fi.IsDir() {
		dest = filepath.Join(dest, backupFileName(now))
	} else if strings.HasSuffix(dest, string(filepath.Separator)) {
		fmt.Fprintf(stderr, "circula: backup: destination directory %s does not exist\n", dest)
		return exitFailure
	}
	raw, err := openRaw(cfg.Database)
	if err != nil {
		fmt.Fprintf(stderr, "circula: backup: %v\n", err)
		return exitFailure
	}
	defer func() { _ = raw.Close() }()
	ledger := func(ctx context.Context, path string) (LedgerSummary, error) { return verifyFile(ctx, path, cfg) }
	mf, err := snapshot(ctx, raw, dest, "manual", now, ledger, cfg.BackupKey)
	if err != nil {
		fmt.Fprintf(stderr, "circula: backup failed: %v\n", err)
		return exitFailure
	}
	fmt.Fprintf(stdout, "backup: %s\nmanifest: %s.manifest.json\nsha256: %s\nschema: v%d\nledger: %s\n",
		dest, dest, mf.SHA256, mf.SchemaVersion, mf.Ledger.Status)
	if mf.Ledger.Status == "failed" {
		fmt.Fprintln(stderr, "circula: WARNING: the backup contains a ledger verification failure; see verify-ledger")
		return exitLedgerFailed
	}
	return exitOK
}

func cmdRestore(ctx context.Context, args []string, stdout, stderr io.Writer) int {
	var overwrite bool
	cfg, rest, code, done := resolveFlags("restore", args, stderr, func(fs *flag.FlagSet) {
		fs.BoolVar(&overwrite, "overwrite", false, "replace an existing database (a pre-restore backup is taken first)")
	})
	if done {
		return code
	}
	if len(rest) != 1 {
		fmt.Fprintln(stderr, "usage: circula restore [flags] <backup-file>")
		return exitUsage
	}
	ctx, stop := withInterrupt(ctx)
	defer stop()
	src := rest[0]
	m, err := readManifest(src + ".manifest.json")
	if err != nil {
		fmt.Fprintf(stderr, "circula: restore refused: %v\n", err)
		return exitFailure
	}
	if err := verifyBackupFile(src, m); err != nil {
		fmt.Fprintf(stderr, "circula: restore refused: %v\n", err)
		return exitFailure
	}
	authenticated, err := checkManifestAuth(m, cfg.BackupKey)
	if err != nil {
		fmt.Fprintf(stderr, "circula: restore refused: %v\n", err)
		return exitFailure
	}
	if m.SchemaVersion > store.SchemaVersion() {
		fmt.Fprintf(stderr, "circula: restore refused: backup schema v%d is newer than this binary (v%d)\n", m.SchemaVersion, store.SchemaVersion())
		return exitFailure
	}

	lock, err := acquireLock(lockPath(cfg.Database))
	if err != nil {
		return reportLockError(stderr, err, cfg.Database)
	}
	defer func() { _ = lock.Release() }()

	state, err := inspectSchema(ctx, cfg.Database)
	if err != nil {
		fmt.Fprintf(stderr, "circula: restore: %v\n", err)
		return exitFailure
	}
	if state.Exists {
		if !overwrite {
			fmt.Fprintf(stderr, "circula: restore refused: %s exists and would be replaced; re-run with -overwrite once the current data is not needed\n", cfg.Database)
			return exitFailure
		}
		mf, path, err := backupTo(ctx, cfg, cfg.BackupDir, "pre-restore", nil)
		if err != nil {
			fmt.Fprintf(stderr, "circula: restore refused: pre-restore backup failed, nothing changed: %v\n", err)
			return exitFailure
		}
		fmt.Fprintf(stdout, "pre-restore backup: %s (ledger: %s)\n", path, mf.Ledger.Status)
	}

	partial := cfg.Database + ".restore-partial"
	removeDBFiles(partial)
	if err := copyFileSync(src, partial); err != nil {
		removeDBFiles(partial)
		fmt.Fprintf(stderr, "circula: restore: copy failed: %v\n", err)
		return exitFailure
	}
	fail := func(format string, a ...any) int {
		removeDBFiles(partial)
		fmt.Fprintf(stderr, "circula: restore refused: "+format+"; the current database was not touched\n", a...)
		return exitFailure
	}
	if res, err := integrityCheck(ctx, partial); err != nil || res != "ok" {
		return fail("integrity check of the copy failed: %v %s", err, res)
	}
	pend, err := inspectSchema(ctx, partial)
	if err != nil {
		return fail("%v", err)
	}
	if pend.Pending() > 0 {
		return fail("backup schema v%d is older than this binary (v%d); restore with the matching binary or run migrate on a copy first", pend.Applied, pend.Known)
	}
	// SA-07: the manifest only vouches for the file bytes, and it sits next to
	// them. Check the ledger and the audit hash chains inside the backup itself,
	// on a throw-away copy, before anything is replaced.
	checks, err := verifyCopy(ctx, partial, cfg)
	if err != nil {
		return fail("ledger verification of the backup could not run: %v", err)
	}
	if !printChecks(stdout, checks) {
		removeDBFiles(partial)
		fmt.Fprintln(stderr, "circula: restore refused: the backup FAILS ledger or audit-chain verification; the current database was not touched (OPS-020)")
		return exitLedgerFailed
	}

	// Swap with a way back (CMD-08): the current database is moved aside, not
	// deleted, until the restored one has opened and passed its checks.
	if err := swapDatabase(ctx, cfg.Database, partial, state.Exists, func(ctx context.Context) error {
		st, err := store.Open(ctx, cfg.Database)
		if err != nil {
			return fmt.Errorf("opening the restored database: %w", err)
		}
		if err := store.CheckPragmas(ctx, st.DB()); err != nil {
			_ = st.Close()
			return fmt.Errorf("checking the restored database: %w", err)
		}
		return st.Close()
	}); err != nil {
		removeDBFiles(partial)
		fmt.Fprintf(stderr, "circula: restore failed: %v; the previous database was put back\n", err)
		return exitFailure
	}
	restoredAt := time.Now().UTC().Truncate(time.Second)
	fmt.Fprintf(stdout, "restored %s (backup created %s) into %s at %s\n",
		src, m.CreatedAt.Format(time.RFC3339), cfg.Database, restoredAt.Format(time.RFC3339))
	fmt.Fprintf(stdout, "data written after %s may be missing and must be reconciled from other sources (OPS-020)\n",
		m.CreatedAt.Format(time.RFC3339))
	if authenticated {
		fmt.Fprintln(stdout, "backup signature: valid (backup-key)")
	} else {
		fmt.Fprintln(stdout, "WARNING: the backup is not authenticated (no backup-key configured): ledger and audit chains verified, but "+
			"accounts, password hashes and sessions are not covered by any check. Restore only files you trust.")
	}
	if err := writeRestoreRecord(cfg.BackupDir, m, restoredAt, true, src); err != nil {
		fmt.Fprintf(stderr, "circula: warning: restore record not written: %v\n", err)
	}
	return exitOK
}

// swapDatabase replaces the database at path with the verified file new. When
// a database exists it is moved aside (with its WAL files) first; check runs
// on the new database, and if the move, the rename or check fails the previous
// database is put back unchanged. On success the old one is deleted.
func swapDatabase(ctx context.Context, path, newFile string, hadDB bool, check func(context.Context) error) error {
	aside := path + ".pre-restore-swap"
	suffixes := []string{"", "-wal", "-shm"}
	putBack := func() {
		removeDBFiles(path)
		_ = os.Remove(path + "-journal")
		if hadDB {
			for _, sfx := range suffixes {
				_ = os.Rename(aside+sfx, path+sfx)
			}
		}
	}
	if hadDB {
		removeDBFiles(aside)
		for _, sfx := range suffixes {
			if err := os.Rename(path+sfx, aside+sfx); err != nil && !errors.Is(err, fs.ErrNotExist) {
				for _, back := range suffixes {
					_ = os.Rename(aside+back, path+back)
				}
				return fmt.Errorf("moving the current database aside: %w", err)
			}
		}
	}
	if err := os.Rename(newFile, path); err != nil {
		putBack()
		return fmt.Errorf("replacing the database: %w", err)
	}
	if err := check(ctx); err != nil {
		putBack()
		return err
	}
	if hadDB {
		removeDBFiles(aside)
	}
	return nil
}

// restoreRecord is the local record of a restore. It is not a ledger audit
// event; the service layer has no restore event yet (see docs/ops/backup-restore.md).
type restoreRecord struct {
	BackupCreatedAt   time.Time `json:"backup_created_at"`
	BackupSHA256      string    `json:"backup_sha256"`
	BackupFile        string    `json:"backup_file"`
	RestoredAt        time.Time `json:"restored_at"`
	LedgerOK          bool      `json:"ledger_ok"`
	MayBeMissingAfter time.Time `json:"data_may_be_missing_after"`
}

func writeRestoreRecord(dir string, m *Manifest, at time.Time, ok bool, src string) error {
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	rec := restoreRecord{
		BackupCreatedAt:   m.CreatedAt,
		BackupSHA256:      m.SHA256,
		BackupFile:        filepath.Base(src),
		RestoredAt:        at,
		LedgerOK:          ok,
		MayBeMissingAfter: m.CreatedAt,
	}
	name := filepath.Join(dir, "restore-"+at.Format("20060102T150405Z")+".json")
	return writeJSONFile(name, rec)
}

func cmdVerifyLedger(ctx context.Context, args []string, stdout, stderr io.Writer) int {
	var network string
	cfg, _, code, done := resolveFlags("verify-ledger", args, stderr, func(fs *flag.FlagSet) {
		fs.StringVar(&network, "network", "", "verify only this network (id or slug); default all")
	})
	if done {
		return code
	}
	ctx, stop := withInterrupt(ctx)
	defer stop()
	// Verification records its run in the database, so it is a writer: it must
	// not run beside `serve` (OPS-011, ARC-011).
	lock, err := acquireLock(lockPath(cfg.Database))
	if err != nil {
		return reportLockError(stderr, err, cfg.Database)
	}
	defer func() { _ = lock.Release() }()
	st, svc, err := openDatabase(ctx, cfg)
	if err != nil {
		fmt.Fprintf(stderr, "circula: verify-ledger: %v\n", err)
		return exitFailure
	}
	defer func() { _ = st.Close() }()
	checks, err := verifyAllNetworks(ctx, svc, st.DB(), "cli-verify", network)
	if err != nil {
		fmt.Fprintf(stderr, "circula: verify-ledger: %v\n", err)
		return exitFailure
	}
	if network != "" && len(checks) == 0 {
		fmt.Fprintf(stderr, "circula: verify-ledger: no network matches %q\n", network)
		return exitFailure
	}
	if !printChecks(stdout, checks) {
		return exitLedgerFailed
	}
	return exitOK
}

func cmdAdmin(ctx context.Context, args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 || args[0] != "create-operator" {
		fmt.Fprintln(stderr, "usage: circula admin create-operator -login <login>  (password from "+operatorPasswordEnv+" or stdin)")
		return exitUsage
	}
	var login string
	cfg, _, code, done := resolveFlags("admin create-operator", args[1:], stderr, func(fs *flag.FlagSet) {
		fs.StringVar(&login, "login", "", "login name for the new operator")
	})
	if done {
		return code
	}
	if login == "" {
		fmt.Fprintln(stderr, "circula: admin create-operator: -login is required")
		return exitUsage
	}
	password, err := readOperatorPassword()
	if err != nil {
		fmt.Fprintf(stderr, "circula: admin create-operator: %v\n", err)
		return exitUsage
	}
	ctx, stop := withInterrupt(ctx)
	defer stop()
	// create-operator writes: refuse while `serve` (or another command) holds the database.
	lock, err := acquireLock(lockPath(cfg.Database))
	if err != nil {
		return reportLockError(stderr, err, cfg.Database)
	}
	defer func() { _ = lock.Release() }()
	st, svc, err := openDatabase(ctx, cfg)
	if err != nil {
		fmt.Fprintf(stderr, "circula: admin create-operator: %v\n", err)
		return exitFailure
	}
	defer func() { _ = st.Close() }()
	p, err := svc.RegisterPerson(ctx, service.RegisterInput{Login: login, Password: password, RevealDuplicate: true})
	if err != nil {
		fmt.Fprintf(stderr, "circula: admin create-operator: %v\n", err)
		return exitFailure
	}
	fmt.Fprintf(stdout, "created person %s (login %s)\n", p.ID, p.Login)
	return exitOK
}

// readOperatorPassword reads the password from the environment or, when that
// is unset, the first line of stdin. It is never taken from the command line.
func readOperatorPassword() (string, error) {
	if v, ok := os.LookupEnv(operatorPasswordEnv); ok && v != "" {
		return v, nil
	}
	line, err := bufio.NewReader(os.Stdin).ReadString('\n')
	if err != nil && !errors.Is(err, io.EOF) {
		return "", fmt.Errorf("read password from stdin: %w", err)
	}
	pw := strings.TrimRight(line, "\r\n")
	if pw == "" {
		return "", fmt.Errorf("no password: set %s or pipe the password on stdin", operatorPasswordEnv)
	}
	return pw, nil
}

// reportLockError explains a lock failure in operator terms.
func reportLockError(stderr io.Writer, err error, database string) int {
	if errors.Is(err, ErrLocked) {
		fmt.Fprintf(stderr, "circula: %s is in use by another circula process (%s); stop it first\n", database, lockPath(database))
		return exitFailure
	}
	fmt.Fprintf(stderr, "circula: %v\n", err)
	return exitFailure
}

// withInterrupt cancels ctx on Ctrl-C or SIGTERM.
func withInterrupt(parent context.Context) (context.Context, func()) {
	return signalContext(parent)
}
