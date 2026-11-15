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
// recompute the whole chain. An anchor records each network's chain head
// (sequence plus hash) in a small file meant to be copied off-host. Verifying
// the live database against an older anchor detects tail truncation and any
// rewrite of history at or before the anchored sequence.

func init() {
	extraCommands["audit-anchor"] = cmdAuditAnchor
	extraUsage = append(extraUsage, "  audit-anchor write|verify   write or check an off-host audit chain checkpoint (RB-010)")
}

const anchorVersion = 1

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
	MAC       string          `json:"mac,omitempty"`
}

func anchorMAC(a auditAnchor, key []byte) string {
	a.MAC = ""
	b, _ := json.Marshal(a)
	m := hmac.New(sha256.New, key)
	m.Write(b)
	return hex.EncodeToString(m.Sum(nil))
}

// anchorDBID identifies the database by its first audit event, which a
// rewrite of history necessarily changes. Empty when no event exists.
func anchorDBID(ctx context.Context, db *sql.DB) (string, error) {
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

// buildAnchor reads every network's current chain head.
func buildAnchor(ctx context.Context, db *sql.DB, key []byte, now time.Time) (*auditAnchor, error) {
	a := &auditAnchor{Version: anchorVersion, CreatedAt: now.UTC().Format(time.RFC3339), Networks: []anchorNetwork{}}
	id, err := anchorDBID(ctx, db)
	if err != nil {
		return nil, err
	}
	a.DBID = id
	rows, err := db.QueryContext(ctx, `SELECT n.id, n.slug, e.seq, e.hash FROM networks n
		JOIN audit_events e ON e.network_id = n.id
		WHERE e.seq = (SELECT MAX(seq) FROM audit_events WHERE network_id = n.id) ORDER BY n.id`)
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
	if len(key) > 0 {
		a.MAC = anchorMAC(*a, key)
	}
	return a, nil
}

// writeAnchorFile writes the anchor atomically with owner-only permissions.
func writeAnchorFile(path string, a *auditAnchor) error {
	b, err := json.MarshalIndent(a, "", "  ")
	if err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, append(b, '\n'), 0o600); err != nil { //nolint:gosec // G703: operator-supplied path
		return err
	}
	if err := os.Rename(tmp, path); err != nil { //nolint:gosec // G703: operator-supplied path
		_ = os.Remove(tmp) //nolint:gosec // G703: operator-supplied path
		return err
	}
	return nil
}

// verifyAnchor compares the live chain with an anchor and returns one finding
// per problem; an empty result means the anchored history is intact.
func verifyAnchor(ctx context.Context, db *sql.DB, a *auditAnchor, key []byte) ([]string, error) {
	var out []string
	if a.Version != anchorVersion {
		return []string{fmt.Sprintf("unsupported anchor version %d", a.Version)}, nil
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
	id, err := anchorDBID(ctx, db)
	if err != nil {
		return nil, err
	}
	if a.DBID != id {
		out = append(out, fmt.Sprintf("database id %q differs from the anchor's %q: different database, or the first audit event was rewritten", id, a.DBID))
	}
	for _, n := range a.Networks {
		var h string
		err := db.QueryRowContext(ctx, `SELECT hash FROM audit_events WHERE network_id = ? AND seq = ?`, n.NetworkID, n.Seq).Scan(&h)
		switch {
		case errors.Is(err, sql.ErrNoRows):
			var maxSeq sql.NullInt64
			_ = db.QueryRowContext(ctx, `SELECT MAX(seq) FROM audit_events WHERE network_id = ?`, n.NetworkID).Scan(&maxSeq)
			out = append(out, fmt.Sprintf("network %s: event %d is missing (live chain ends at %d): tail truncated or network removed", n.Slug, n.Seq, maxSeq.Int64))
		case err != nil:
			return nil, err
		case h != n.Hash:
			out = append(out, fmt.Sprintf("network %s: event %d hash differs from the anchor: history was rewritten", n.Slug, n.Seq))
		}
	}
	return out, nil
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
		fmt.Fprintf(stdout, "anchor written: %s (%d network(s)); copy it off this host\n", path, len(a.Networks))
		return exitOK
	}
	b, err := os.ReadFile(path) //nolint:gosec // operator-supplied path
	if err != nil {
		fmt.Fprintf(stderr, "circula: audit-anchor verify: %v\n", err)
		return exitFailure
	}
	var a auditAnchor
	dec := json.NewDecoder(bytes.NewReader(b))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&a); err != nil {
		fmt.Fprintf(stderr, "circula: audit-anchor verify: %s is not an anchor file: %v\n", filepath.Base(path), err)
		return exitFailure
	}
	findings, err := verifyAnchor(ctx, db, &a, cfg.AuditAnchorKey)
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
	fmt.Fprintf(stdout, "audit anchor: ok (%d network(s) at %s)\n", len(a.Networks), a.CreatedAt)
	return exitOK
}
