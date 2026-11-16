package service

import "github.com/djshellshoxxx/circula/internal/domain"

// validatePageIDs applies only to UUID keyset cursors after the caller's
// existing authorization checks (SEC-024). Empty means the first page;
// numeric audit/ledger sequences and opaque signed cursors do not use it.
func validatePageIDs(ids ...string) error {
	for _, id := range ids {
		if id != "" && !domain.ValidRecordID(id) {
			return domain.Invalid("cursor", "must be a record identifier")
		}
	}
	return nil
}
