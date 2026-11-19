package repository

import "context"

// StorageChecker is an optional operational capability. Each implementation
// checks the storage settings its correctness depends on; callers do not need
// a database handle, SQL statements or driver-specific types (ARC-004).
type StorageChecker interface {
	CheckStorage(context.Context) error
}
