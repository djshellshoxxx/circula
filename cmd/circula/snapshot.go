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
	// Encrypted marks a backup written as an encrypted container (OPS-019).
	// File, Bytes and SHA256 then describe the ciphertext on disk, and the
	// plaintext database is described by PlaintextBytes and PlaintextSHA256.
	Encrypted       bool   `json:"encrypted"`
	Cipher          string `json:"cipher,omitempty"`
	PlaintextBytes  int64  `json:"plaintext_bytes,omitempty"`
	PlaintextSHA256 string `json:"plaintext_sha256,omitempty"`
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
	if m.Encrypted {
		// Covered only for encrypted backups so older manifests still verify.
		// Flipping "encrypted" off therefore breaks the signature.
		fmt.Fprintf(h, "encrypted=true\ncipher=%s\nplain_bytes=%d\nplain_sha256=%s\n", m.Cipher, m.PlaintextBytes, m.PlaintextSHA256)
	}
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
	return verifyNetworks(ctx, svc, db, requestID, filter, false)
}

// verifyNetworks is verifyAllNetworks that can also record each run in
// consistency_runs (scheduled job, ARC-015); the audit event is written only
// on failure or a state change.
func verifyNetworks(ctx context.Context, svc *service.Service, db *sql.DB, requestID, filter string, record bool) ([]networkCheck, error) {
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
		if record {
			if err := svc.RecordScheduledConsistencyRun(ctx, domain.SystemActor(requestID), rep); err != nil {
				return out, fmt.Errorf("record run for network %s: %w", n.Slug, err)
			}
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
	return snapshotWith(ctx, db, dest, reason, now, ledger, snapshotKeys{Sign: key})
}

// snapshotKeys are the secrets a snapshot may use: Sign signs the manifest
// (SA-07) and Encrypt, when set, writes the backup as an encrypted container
// (OPS-019). With Encrypt the plaintext copy lives only in a private temporary
// directory next to dest and is overwritten and removed before returning,
// whether the backup succeeded or not.
type snapshotKeys struct{ Sign, Encrypt []byte }

func snapshotWith(ctx context.Context, db *sql.DB, dest, reason string, now time.Time, ledger ledgerFunc, keys snapshotKeys) (*Manifest, error) {
	key := keys.Sign
	if _, err := os.Stat(dest); err == nil {
		return nil, fmt.Errorf("backup destination %s already exists", dest)
	}
	encrypting := len(keys.Encrypt) > 0
	partial := dest + ".partial"
	if encrypting {
		sweepStaleTemp(filepath.Dir(dest))
		work, err := os.MkdirTemp(filepath.Dir(dest), tempDirPrefix)
		if err != nil {
			return nil, fmt.Errorf("private work directory: %w", err)
		}
		defer secureRemoveAll(work)
		partial = filepath.Join(work, "snapshot.db")
	}
	rm := removeDBFiles
	if encrypting {
		rm = secureRemoveDB // plaintext copies are overwritten, not just unlinked
	}
	removeDBFiles(partial)
	if err := store.VacuumInto(ctx, db, partial); err != nil {
		removeDBFiles(partial)
		return nil, fmt.Errorf("online snapshot: %w", err)
	}
	fail := func(err error) (*Manifest, error) {
		removeDBFiles(partial)
		if encrypting {
			_ = os.Remove(dest + ".partial")
		}
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
			rm(scratch)
			return fail(fmt.Errorf("verify snapshot: %w", err))
		}
		ledgerSum, err = ledger(ctx, scratch)
		rm(scratch)
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
	onDisk, onDiskSize := sum, size
	if encrypting {
		// Encrypt, then prove the ciphertext decrypts back to exactly the
		// verified plaintext before it replaces anything (OPS-019).
		encPartial := dest + ".partial"
		_ = os.Remove(encPartial)
		onDisk, onDiskSize, err = encryptFile(partial, encPartial, keys.Encrypt)
		if err != nil {
			return fail(fmt.Errorf("encrypt backup: %w", err))
		}
		back, backSize, err := decryptTo(encPartial, io.Discard, keys.Encrypt)
		if err != nil || back != sum || backSize != size {
			return fail(fmt.Errorf("encrypted backup did not decrypt to the verified snapshot: %v", err))
		}
		if err := os.Rename(encPartial, dest); err != nil {
			return fail(err)
		}
	} else if err := os.Rename(partial, dest); err != nil {
		return fail(err)
	}
	m := &Manifest{
		Format:        manifestFormat,
		CreatedAt:     now.UTC().Truncate(time.Second),
		Reason:        reason,
		AppVersion:    version,
		SchemaVersion: schema.Applied,
		File:          filepath.Base(dest),
		Bytes:         onDiskSize,
		SHA256:        onDisk,
		Integrity:     integrity,
		Ledger:        ledgerSum,
	}
	if encrypting {
		m.Encrypted, m.Cipher, m.PlaintextBytes, m.PlaintextSHA256 = true, cipherName, size, sum
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

// secureRemoveDB is removeDBFiles that overwrites the files first.
func secureRemoveDB(path string) {
	secureRemove(path + "-wal")
	secureRemove(path + "-shm")
	secureRemove(path)
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
const encryptedSuffix = ".db.enc"
const cipherName = "aes-256-gcm-chunked-v1"
const tempDirPrefix = ".circula-bk-"
const backupTimeLayout = "20060102T150405Z"

func backupFileName(t time.Time) string {
	return backupPrefix + t.UTC().Format(backupTimeLayout) + backupSuffix
}

// encryptedBackupFileName is backupFileName for an encrypted backup.
func encryptedBackupFileName(t time.Time) string {
	return backupPrefix + t.UTC().Format(backupTimeLayout) + encryptedSuffix
}

// backupFileNameFor names a new backup: *.db.enc when cfg encrypts, else *.db.
func backupFileNameFor(cfg *Config, t time.Time) string {
	if len(cfg.BackupEncryptionKey) > 0 {
		return encryptedBackupFileName(t)
	}
	return backupFileName(t)
}

// sweepStaleTemp removes plaintext work directories that a crashed backup
// left behind. Only directories older than an hour are touched, so a backup
// running in another process is never disturbed.
func sweepStaleTemp(dir string) {
	ents, err := os.ReadDir(dir)
	if err != nil {
		return
	}
	for _, e := range ents {
		if !e.IsDir() || !strings.HasPrefix(e.Name(), tempDirPrefix) {
			continue
		}
		if fi, err := e.Info(); err == nil && time.Since(fi.ModTime()) > time.Hour {
			secureRemoveAll(filepath.Join(dir, e.Name()))
		}
	}
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
		if e.IsDir() || !strings.HasPrefix(name, backupPrefix) {
			continue
		}
		stem := strings.TrimPrefix(name, backupPrefix)
		switch {
		case strings.HasSuffix(stem, encryptedSuffix):
			stem = strings.TrimSuffix(stem, encryptedSuffix)
		case strings.HasSuffix(stem, backupSuffix):
			stem = strings.TrimSuffix(stem, backupSuffix)
		default:
			continue
		}
		t, err := time.Parse(backupTimeLayout, stem)
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
		_ = os.Remove(e.path + ".anchor.json")
		removed = append(removed, filepath.Base(e.path))
	}
	return removed, nil
}
