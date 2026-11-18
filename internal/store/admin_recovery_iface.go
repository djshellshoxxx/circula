package store

import "github.com/djshellshoxxx/circula/internal/repository"

// Compatibility alias preserving the storage API and exact type identity.
// The driver-neutral contract lives in internal/repository (ARC-004).
type (
	AdminRecoveryRepo = repository.AdminRecoveryRepo
)
