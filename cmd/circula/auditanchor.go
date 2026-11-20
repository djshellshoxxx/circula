package main

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"
)

// External audit-chain anchor (RB-010). The hash chain and its head live in
// the same database file, so someone with write access to that file can
// recompute the whole chain. An anchor records the platform and network heads
// (sequence plus hash) in a small file meant to be copied off-host. Verifying
// the live database against an older anchor detects tail truncation and any
// rewrite of history at or before the anchored sequence.

func init() {
	extraCommands["audit-anchor"] = cmdAuditAnchor
	extraUsage = append(extraUsage, "  audit-anchor write|verify   write or check an off-host audit chain checkpoint (RB-010)")
}

const anchorVersion = 2

// anchorCheckpoint describes the last event in one covered scope. An empty
// scope is explicit: seq is zero and hash is empty. It anchors no past events.
type anchorCheckpoint struct {
	Seq  int64  `json:"seq"`
	Hash string `json:"hash"`
}

// anchorNetwork is one network's chain head at anchor time.
type anchorNetwork struct {
	NetworkID string `json:"network_id"`
	Slug      string `json:"slug"`
	Seq       int64  `json:"seq"`
	Hash      string `json:"hash"`
}

// auditAnchor is the checkpoint file. MAC is hex HMAC-SHA256 over the JSON of
// the same document with MAC empty, present only when a key is configured.
type auditAnchor struct {
	Version   int             `json:"version"`
	CreatedAt string          `json:"created_at"`
	DBID      string          `json:"db_id"`
	Networks  []anchorNetwork `json:"networks"`
	// Omitted in v1, so historical canonical JSON and MAC bytes are unchanged.
	// Required in v2, even when the platform scope has no events yet.
	Platform *anchorCheckpoint `json:"platform,omitempty"`
	MAC      string            `json:"mac,omitempty"`
}

func anchorMAC(a auditAnchor, key []byte) string {
	a.MAC = ""
	b, _ := json.Marshal(a)
	m := hmac.New(sha256.New, key)
	m.Write(b)
	return hex.EncodeToString(m.Sum(nil))
}

type anchorReader interface {
	QueryRowContext(context.Context, string, ...any) *sql.Row
}

// anchorDBID identifies the database by its first audit event. Empty when no
// event exists; changing that event's identity changes the database id.
func anchorDBID(ctx context.Context, db anchorReader) (string, error) {
	var id string
	err := db.QueryRowContext(ctx, `SELECT id FROM audit_events ORDER BY occurred_at, network_id, seq LIMIT 1`).Scan(&id)
	if errors.Is(err, sql.ErrNoRows) {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256([]byte("circula-db-id:" + id))
	return hex.EncodeToString(sum[:8]), nil
}

// buildAnchor reads the platform and every network head in one snapshot.
func buildAnchor(ctx context.Context, db *sql.DB, key []byte, now time.Time) (*auditAnchor, error) {
	a := &auditAnchor{Version: anchorVersion, CreatedAt: now.UTC().Format(time.RFC3339), Networks: []anchorNetwork{}}
	tx, err := db.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback() }()
	id, err := anchorDBID(ctx, tx)
	if err != nil {
		return nil, err
	}
	a.DBID = id
	rows, err := tx.QueryContext(ctx, `SELECT n.id, n.slug, COALESCE(e.seq, 0), COALESCE(e.hash, '') FROM networks n
		LEFT JOIN audit_events e ON e.network_id = n.id
		AND e.seq = (SELECT MAX(seq) FROM audit_events WHERE network_id = n.id) ORDER BY n.id`)
	if err != nil {
		return nil, fmt.Errorf("read chain heads: %w", err)
	}
	defer func() { _ = rows.Close() }()
	for rows.Next() {
		var n anchorNetwork
		if err := rows.Scan(&n.NetworkID, &n.Slug, &n.Seq, &n.Hash); err != nil {
			return nil, err
		}
		a.Networks = append(a.Networks, n)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}
	a.Platform = &anchorCheckpoint{}
	err = tx.QueryRowContext(ctx, `SELECT seq, hash FROM audit_events WHERE network_id = '' ORDER BY seq DESC LIMIT 1`).
		Scan(&a.Platform.Seq, &a.Platform.Hash)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("read platform head: %w", err)
	}
	if len(key) > 0 {
		a.MAC = anchorMAC(*a, key)
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return a, nil
}

// writeAnchorFile writes the anchor atomically with owner-only permissions.
func writeAnchorFile(path string, a *auditAnchor) error {
	b, err := json.MarshalIndent(a, "", "  ")
	if err != nil {
		return err
	}
	return writeAtomicPrivateFile(path, append(b, '\n'))
}

// verifyAnchor compares the live chain with an anchor and returns one finding
// per problem; an empty result means the covered checkpoints still match.
// Routine ledger/platform verification checks the complete live hash chains.
func verifyAnchor(ctx context.Context, db *sql.DB, a *auditAnchor, key []byte) ([]string, error) {
	var out []string
	if a.Version != 1 && a.Version != anchorVersion {
		return []string{fmt.Sprintf("unsupported anchor version %d", a.Version)}, nil
	}
	if a.Version == 1 && a.Platform != nil {
		return []string{"version 1 anchors cover networks only; a platform checkpoint requires version 2"}, nil
	}
	if a.Version == anchorVersion && a.Platform == nil {
		return []string{"version 2 anchor is missing its platform checkpoint"}, nil
	}
	if a.Platform != nil && !validAnchorCheckpoint(*a.Platform) {
		return []string{"platform checkpoint must contain a positive sequence and hash, or an explicit empty scope"}, nil
	}
	seen := make(map[string]bool, len(a.Networks))
	nonempty := a.Platform != nil && a.Platform.Seq > 0
	for _, n := range a.Networks {
		if n.NetworkID == "" || seen[n.NetworkID] || !validAnchorCheckpoint(anchorCheckpoint{n.Seq, n.Hash}) || (a.Version == 1 && n.Seq == 0) {
			return []string{"anchor contains an invalid or repeated network checkpoint"}, nil
		}
		seen[n.NetworkID] = true
		nonempty = nonempty || n.Seq > 0
	}
	if a.Version == anchorVersion && a.DBID == "" && nonempty {
		return []string{"nonempty version 2 checkpoints require a database identity"}, nil
	}
	switch {
	case len(key) > 0 && a.MAC == "":
		out = append(out, "anchor has no MAC but an audit-anchor-key is configured")
	case len(key) > 0:
		want := anchorMAC(*a, key)
		if !hmac.Equal([]byte(want), []byte(a.MAC)) {
			return []string{"anchor MAC does not verify: the file was altered or the key differs"}, nil
		}
	case a.MAC != "":
		out = append(out, "anchor carries a MAC but no audit-anchor-key is configured, so it is unauthenticated")
	}
	tx, err := db.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback() }()
	id, err := anchorDBID(ctx, tx)
	if err != nil {
		return nil, err
	}
	// A v2 anchor with no audit events has no database identity to bind yet.
	// Its zero checkpoints cover no past events; legitimate first writes pass.
	if a.DBID != id && (a.Version == 1 || a.DBID != "") {
		out = append(out, fmt.Sprintf("database id %q differs from the anchor's %q: different database, or the first audit event was rewritten", id, a.DBID))
	}
	for _, n := range a.Networks {
		finding, err := verifyAnchorCheckpoint(ctx, tx, n.NetworkID, "network "+n.Slug, anchorCheckpoint{n.Seq, n.Hash})
		if err != nil {
			return nil, err
		}
		if finding != "" {
			out = append(out, finding)
		}
	}
	if a.Platform != nil {
		finding, err := verifyAnchorCheckpoint(ctx, tx, "", "platform audit", *a.Platform)
		if err != nil {
			return nil, err
		}
		if finding != "" {
			out = append(out, finding)
		}
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return out, nil
}

func validAnchorCheckpoint(c anchorCheckpoint) bool {
	if c.Seq == 0 {
		return c.Hash == ""
	}
	if c.Seq < 0 || len(c.Hash) != sha256.Size*2 {
		return false
	}
	_, err := hex.DecodeString(c.Hash)
	return err == nil
}

func verifyAnchorCheckpoint(ctx context.Context, db anchorReader, scope, label string, checkpoint anchorCheckpoint) (string, error) {
	if checkpoint.Seq == 0 {
		return "", nil // no past events are covered; later growth is legitimate
	}
	var hash string
	err := db.QueryRowContext(ctx, `SELECT hash FROM audit_events WHERE network_id = ? AND seq = ?`, scope, checkpoint.Seq).Scan(&hash)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		var maxSeq sql.NullInt64
		if err := db.QueryRowContext(ctx, `SELECT MAX(seq) FROM audit_events WHERE network_id = ?`, scope).Scan(&maxSeq); err != nil {
			return "", err
		}
		return fmt.Sprintf("%s: event %d is missing (live chain ends at %d): tail truncated or scope removed", label, checkpoint.Seq, maxSeq.Int64), nil
	case err != nil:
		return "", err
	case hash != checkpoint.Hash:
		return fmt.Sprintf("%s: event %d hash differs from the anchor: history was rewritten", label, checkpoint.Seq), nil
	default:
		return "", nil
	}
}

func cmdAuditAnchor(ctx context.Context, args []string, stdout, stderr io.Writer) int {
	const usage = "usage: circula audit-anchor write <path> | verify <file>   (store the written file off-host)"
	if len(args) == 0 || (args[0] != "write" && args[0] != "verify") {
		fmt.Fprintln(stderr, usage)
		return exitUsage
	}
	mode := args[0]
	cfg, rest, code, done := resolveFlags("audit-anchor "+mode, args[1:], stderr, nil)
	if done {
		return code
	}
	if len(rest) != 1 {
		fmt.Fprintln(stderr, usage)
		return exitUsage
	}
	path := rest[0]
	db, err := openRaw(cfg.Database)
	if err != nil {
		fmt.Fprintf(stderr, "circula: audit-anchor: %v\n", err)
		return exitFailure
	}
	defer func() { _ = db.Close() }()
	if mode == "write" {
		a, err := buildAnchor(ctx, db, cfg.AuditAnchorKey, time.Now())
		if err == nil {
			err = writeAnchorFile(path, a)
		}
		if err != nil {
			fmt.Fprintf(stderr, "circula: audit-anchor write: %v\n", err)
			return exitFailure
		}
		fmt.Fprintf(stdout, "anchor written: %s (platform audit and %d network(s)); copy it off this host\n", path, len(a.Networks))
		return exitOK
	}
	b, err := os.ReadFile(path) //nolint:gosec // operator-supplied path
	if err != nil {
		fmt.Fprintf(stderr, "circula: audit-anchor verify: %v\n", err)
		return exitFailure
	}
	var a *auditAnchor
	dec := json.NewDecoder(bytes.NewReader(b))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&a); err != nil {
		fmt.Fprintf(stderr, "circula: audit-anchor verify: %s is not an anchor file: %v\n", filepath.Base(path), err)
		return exitFailure
	}
	if a == nil {
		fmt.Fprintf(stderr, "circula: audit-anchor verify: %s is not an anchor file: expected a JSON object\n", filepath.Base(path))
		return exitFailure
	}
	if _, err := dec.Token(); err != io.EOF {
		fmt.Fprintf(stderr, "circula: audit-anchor verify: %s is not an anchor file: trailing content\n", filepath.Base(path))
		return exitFailure
	}
	findings, err := verifyAnchor(ctx, db, a, cfg.AuditAnchorKey)
	if err != nil {
		fmt.Fprintf(stderr, "circula: audit-anchor verify: %v\n", err)
		return exitFailure
	}
	if len(findings) > 0 {
		fmt.Fprintln(stdout, "audit anchor: FAILED")
		for _, f := range findings {
			fmt.Fprintf(stdout, "  - %s\n", f)
		}
		return exitLedgerFailed
	}
	if a.Version == 1 {
		fmt.Fprintf(stdout, "audit anchor: ok (%d network(s) at %s; legacy v1 network-only coverage, platform audit not covered)\n", len(a.Networks), a.CreatedAt)
	} else {
		fmt.Fprintf(stdout, "audit anchor: ok (platform audit and %d network(s) at %s)\n", len(a.Networks), a.CreatedAt)
	}
	return exitOK
}
