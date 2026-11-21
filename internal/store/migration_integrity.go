package store

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/djshellshoxxx/circula/internal/integrity"
)

type migrationFinding struct {
	networkID string
	message   string
}

// migrationFindings evaluates the ordinary read-only verification operation
// on the migration's own connection. Both calls occur inside BEGIN IMMEDIATE,
// so another writer cannot change the baseline before the migration commits.
// Version 0 has no ledger tables; version 1 predates the recorded audit heads.
// Every supported later version must provide those heads: a missing table is
// an error, not permission to skip verification.
func migrationFindings(ctx context.Context, conn *sql.Conn, version int) ([]migrationFinding, error) {
	if version == 0 {
		return nil, nil
	}
	rows, err := conn.QueryContext(ctx, `SELECT id FROM networks ORDER BY id`)
	if err != nil {
		return nil, err
	}
	var networks []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			_ = rows.Close()
			return nil, err
		}
		networks = append(networks, id)
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	tx := &txImpl{ctx: ctx, q: conn}
	options := integrity.Options{LegacyAuditWithoutHead: version == 1, LegacyCreditLimitsWithoutWitness: version < 24}
	var findings []migrationFinding
	for _, id := range networks {
		report, err := integrity.CheckLedger(tx, id, options)
		if err != nil {
			return nil, fmt.Errorf("verify network %q: %w", id, err)
		}
		for _, message := range report.Findings {
			findings = append(findings, migrationFinding{networkID: id, message: message})
		}
	}
	// Account recovery and operator history use the independent platform
	// scope. It must remain valid even in a database with no networks.
	report, err := integrity.CheckAudit(tx, "", options)
	if err != nil {
		return nil, fmt.Errorf("verify platform audit: %w", err)
	}
	for _, message := range report.Findings {
		findings = append(findings, migrationFinding{message: message})
	}
	return findings, nil
}

// refuseNewMigrationFindings implements OPS-023's comparison rule. Existing
// findings remain visible to the ordinary post-upgrade verification and do
// not become new permission to write. A migration can preserve or resolve
// them, but it cannot introduce another finding before committing.
func refuseNewMigrationFindings(before, after []migrationFinding) error {
	known := make(map[migrationFinding]struct{}, len(before))
	for _, finding := range before {
		known[finding] = struct{}{}
	}
	for _, finding := range after {
		if _, found := known[finding]; !found {
			return fmt.Errorf("introduced ledger finding in scope %q: %s", finding.networkID, finding.message)
		}
	}
	return nil
}
