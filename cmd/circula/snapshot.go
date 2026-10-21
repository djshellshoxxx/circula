package main

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"time"

	"github.com/djshellshoxxx/circula/internal/domain"
	"github.com/djshellshoxxx/circula/internal/service"
	"github.com/djshellshoxxx/circula/internal/store"
)

// manifestFormat is the version of the backup manifest layout.
const manifestFormat = 1

// LedgerSummary is the ledger verification result recorded with a backup or a
// command run. Status is ok, failed or not-run. Only counts are recorded, not
// network identifiers, so manifests carry no member or network data.
type LedgerSummary struct {
	Status   string `json:"status"`
	Networks int    `json:"networks"`
	Failing  int    `json:"failing_networks"`
	Reason   string `json:"reason,omitempty"`
}

// Manifest accompanies every backup file (OPS-017).
type Manifest struct {
	Format        int           `json:"format"`
	CreatedAt     time.Time     `json:"created_at"`
	Reason        string        `json:"reason"`
	AppVersion    string        `json:"app_version"`
	SchemaVersion int           `json:"schema_version"`
	File          string        `json:"file"`
	Bytes         int64         `json:"bytes"`
	SHA256        string        `json:"sha256"`
	Integrity     string        `json:"integrity_check"`
	Ledger        LedgerSummary `json:"ledger_verification"`
	// HMAC is hex HMAC-SHA256 over the identifying fields, present when
	// backup-key was configured (SA-07). Without the key it proves nothing.
	HMAC string `json:"hmac_sha256,omitempty"`
}

// manifestMAC computes the signature of m under key. It covers every field
// that identifies the content (format, time, schema, size, digest), not the
// file name, so a renamed backup still verifies.
func manifestMAC(m *Manifest, key []byte) string {
	h := hmac.New(sha256.New, key)
	fmt.Fprintf(h, "circula-backup-v1\nformat=%d\ncreated=%s\nschema=%d\nbytes=%d\nsha256=%s\n",
		m.Format, m.CreatedAt.UTC().Format(time.RFC3339), m.SchemaVersion, m.Bytes, m.SHA256)
	return hex.EncodeToString(h.Sum(nil))
}

// checkManifestAuth enforces the backup key policy for restore. With a key
// the manifest must carry a valid signature (an unsigned manifest may have
// been stripped by an attacker). Without a key the backup is only
// integrity-checked, and the caller says so (authenticated is false).
func checkManifestAuth(m *Manifest, key []byte) (authenticated bool, err error) {
	if len(key) == 0 {
		return false, nil
	}
	if m.HMAC == "" {
		return false, errors.New("backup-key is configured but the manifest is not signed; refusing an unauthenticated backup")
	}
	want, err := hex.DecodeString(m.HMAC)
	got, _ := hex.DecodeString(manifestMAC(m, key))
	if err != nil || !hmac.Equal(want, got) {
		return false, errors.New("manifest signature does not verify under the configured backup-key (wrong key, or the backup was altered)")
	}
	return true, nil
}

// networkCheck is the outcome of verifying one network.
type networkCheck struct {
	ID       string
	Slug     string
	OK       bool
	Findings []string
}

// verifyAllNetworks runs the ledger verification for every network (or only
// the network whose id or slug equals filter, when filter is not empty) as the
// system actor. It returns an error only when the check could not run.
func verifyAllNetworks(ctx context.Context, svc *service.Service, db *sql.DB, requestID, filter string) ([]networkCheck, error) {
	nets, err := store.ListNetworks(ctx, db, filter)
	if err != nil {
		return nil, err
	}
	out := make([]networkCheck, 0, len(nets))
	for _, n := range nets {
		rep, err := svc.VerifyLedger(ctx, domain.SystemActor(requestID), n.ID)
		if err != nil {
			return out, fmt.Errorf("verify network %s: %w", n.Slug, err)
		}
		out = append(out, networkCheck{ID: n.ID, Slug: n.Slug, OK: rep.OK, Findings: rep.Findings})
	}
	return out, nil
}

// summarise turns per-network results into a LedgerSummary.
func summarise(checks []networkCheck) LedgerSummary {
	s := LedgerSummary{Status: "ok", Networks: len(checks)}
	for _, c := range checks {
		if !c.OK {
			s.Failing++
		}
	}
	if s.Failing > 0 {
		s.Status = "failed"
	}
	return s
}

// ledgerFunc verifies a database file and returns its summary.
type ledgerFunc func(ctx context.Context, path string) (LedgerSummary, error)

// verifyFileChecks opens the database file at path (which must already be at
// the binary's schema version), runs ledger verification for every network and
// closes it. Verification records its run in the file, so callers pass a copy
// whenever the file itself must stay unchanged.
func verifyFileChecks(ctx context.Context, path string, cfg *Config) ([]networkCheck, error) {
	st, err := store.Open(ctx, path)
	if err != nil {
		return nil, err
	}
	defer func() { _ = st.Close() }()
	svc, err := newService(st, cfg)
	if err != nil {
		return nil, err
	}
	return verifyAllNetworks(ctx, svc, st.DB(), "backup-verify", "")
}

// verifyFile is verifyFileChecks reduced to a summary.
func verifyFile(ctx context.Context, path string, cfg *Config) (LedgerSummary, error) {
	checks, err := verifyFileChecks(ctx, path, cfg)
	if err != nil {
		return LedgerSummary{}, err
	}
	return summarise(checks), nil
}

// verifyCopy verifies a throw-away copy of path, leaving path untouched.
func verifyCopy(ctx context.Context, path string, cfg *Config) ([]networkCheck, error) {
	scratch := path + ".verify"
	removeDBFiles(scratch)
	defer removeDBFiles(scratch)
	if err := copyFileSync(path, scratch); err != nil {
		return nil, err
	}
	return verifyFileChecks(ctx, scratch, cfg)
}

// snapshot writes a consistent online copy of the database behind db to dest,
// verifies it, and writes dest + ".manifest.json". The copy is built under a
// .partial name and renamed only after every check passed, so an interrupted
// backup never looks complete. VACUUM INTO reads a snapshot and does not hold
// the write lock (OPS-017). The ledger check runs on a throw-away copy, because
// verification records its run in the database it checks: the backup must stay
// byte-identical to what VACUUM INTO produced (CMD-01), which snapshot proves
// by comparing digests before and after the checks. A non-empty key signs the
// manifest (SA-07). Files are created 0600 (SA-06).
func snapshot(ctx context.Context, db *sql.DB, dest, reason string, now time.Time, ledger ledgerFunc, key []byte) (*Manifest, error) {
	if _, err := os.Stat(dest); err == nil {
		return nil, fmt.Errorf("backup destination %s already exists", dest)
	}
	partial := dest + ".partial"
	removeDBFiles(partial)
	if err := store.VacuumInto(ctx, db, partial); err != nil {
		removeDBFiles(partial)
		return nil, fmt.Errorf("online snapshot: %w", err)
	}
	fail := func(err error) (*Manifest, error) {
		removeDBFiles(partial)
		return nil, err
	}
	if err := os.Chmod(partial, 0o600); err != nil && runtime.GOOS != "windows" {
		return fail(fmt.Errorf("restrict backup permissions: %w", err))
	}
	before, _, err := fileSHA256(partial)
	if err != nil {
		return fail(err)
	}

	integrity, err := integrityCheck(ctx, partial)
	if err != nil {
		return fail(err)
	}
	schema, err := inspectSchema(ctx, partial)
	if err != nil {
		return fail(err)
	}
	ledgerSum := LedgerSummary{Status: "not-run", Reason: "snapshot schema is older than this binary (pre-migration copy)"}
	if schema.Pending() == 0 && ledger != nil {
		scratch := partial + ".verify"
		removeDBFiles(scratch)
		if err := copyFileSync(partial, scratch); err != nil {
			removeDBFiles(scratch)
			return fail(fmt.Errorf("verify snapshot: %w", err))
		}
		ledgerSum, err = ledger(ctx, scratch)
		removeDBFiles(scratch)
		if err != nil {
			return fail(fmt.Errorf("verify snapshot: %w", err))
		}
	}
	removeSideFiles(partial)

	sum, size, err := fileSHA256(partial)
	if err != nil {
		return fail(err)
	}
	if sum != before {
		return fail(errors.New("snapshot changed while it was being checked; backup discarded"))
	}
	if err := os.Rename(partial, dest); err != nil {
		return fail(err)
	}
	m := &Manifest{
		Format:        manifestFormat,
		CreatedAt:     now.UTC().Truncate(time.Second),
		Reason:        reason,
		AppVersion:    version,
		SchemaVersion: schema.Applied,
		File:          filepath.Base(dest),
		Bytes:         size,
		SHA256:        sum,
		Integrity:     integrity,
		Ledger:        ledgerSum,
	}
	if len(key) > 0 {
		m.HMAC = manifestMAC(m, key)
	}
	if err := writeManifest(dest+".manifest.json", m); err != nil {
		return nil, fmt.Errorf("backup %s written but manifest failed: %w", dest, err)
	}
	return m, nil
}

// integrityCheck runs PRAGMA integrity_check on a file and returns "ok" or the
// first reported problem.
func integrityCheck(ctx context.Context, path string) (string, error) {
	return store.IntegrityCheck(ctx, path)
}

// removeDBFiles deletes a database file and its WAL side files, ignoring
// missing ones.
func removeDBFiles(path string) {
	removeSideFiles(path)
	_ = os.Remove(path)
}

// removeSideFiles deletes the WAL and shared-memory files next to path.
func removeSideFiles(path string) {
	_ = os.Remove(path + "-wal")
	_ = os.Remove(path + "-shm")
}

// fileSHA256 returns the hex SHA-256 and size of a file.
func fileSHA256(path string) (string, int64, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", 0, err
	}
	defer func() { _ = f.Close() }()
	h := sha256.New()
	n, err := io.Copy(h, f)
	if err != nil {
		return "", 0, err
	}
	return hex.EncodeToString(h.Sum(nil)), n, nil
}

// writeManifest writes the manifest atomically.
func writeManifest(path string, m *Manifest) error {
	b, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return err
	}
	b = append(b, '\n')
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, b, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

// readManifest loads and validates the manifest for a backup file.
func readManifest(path string) (*Manifest, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read manifest: %w", err)
	}
	var m Manifest
	if err := json.Unmarshal(b, &m); err != nil {
		return nil, fmt.Errorf("manifest %s is not valid JSON: %w", path, err)
	}
	if m.Format != manifestFormat {
		return nil, fmt.Errorf("manifest %s has unsupported format %d", path, m.Format)
	}
	return &m, nil
}

// verifyBackupFile recomputes the checksum of a backup and compares it with
// its manifest.
func verifyBackupFile(path string, m *Manifest) error {
	sum, size, err := fileSHA256(path)
	if err != nil {
		return err
	}
	if sum != m.SHA256 || size != m.Bytes {
		return errors.New("backup file does not match its manifest checksum or size")
	}
	return nil
}

// backupFileName returns the backup file name for a time (UTC, second precision).
const backupPrefix = "circula-"
const backupSuffix = ".db"
const backupTimeLayout = "20060102T150405Z"

func backupFileName(t time.Time) string {
	return backupPrefix + t.UTC().Format(backupTimeLayout) + backupSuffix
}

type backupEntry struct {
	path string
	at   time.Time
}

// listBackups returns the backups in dir, newest first. Files that do not
// match the naming scheme are ignored.
func listBackups(dir string) ([]backupEntry, error) {
	ents, err := os.ReadDir(dir)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var out []backupEntry
	for _, e := range ents {
		name := e.Name()
		if e.IsDir() || !strings.HasPrefix(name, backupPrefix) || !strings.HasSuffix(name, backupSuffix) {
			continue
		}
		t, err := time.Parse(backupTimeLayout, strings.TrimSuffix(strings.TrimPrefix(name, backupPrefix), backupSuffix))
		if err != nil {
			continue
		}
		out = append(out, backupEntry{path: filepath.Join(dir, name), at: t})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].at.After(out[j].at) })
	return out, nil
}

// pruneBackups keeps the newest keepDaily backups and, for each of the newest
// keepMonthly calendar months that has backups, the newest backup of that
// month. It returns the names of removed backup files.
func pruneBackups(dir string, keepDaily, keepMonthly int) ([]string, error) {
	entries, err := listBackups(dir)
	if err != nil {
		return nil, err
	}
	keep := map[string]bool{}
	for i, e := range entries {
		if i < keepDaily {
			keep[e.path] = true
		}
	}
	months := map[string]bool{}
	for _, e := range entries {
		key := e.at.Format("2006-01")
		if months[key] || len(months) >= keepMonthly {
			continue
		}
		months[key] = true
		keep[e.path] = true
	}
	var removed []string
	for _, e := range entries {
		if keep[e.path] {
			continue
		}
		removeDBFiles(e.path)
		_ = os.Remove(e.path + ".manifest.json")
		removed = append(removed, filepath.Base(e.path))
	}
	return removed, nil
}
