package store

import "github.com/djshellshoxxx/circula/internal/repository"

// Compatibility aliases preserve the existing storage API and exact type
// identity. Driver-neutral contracts live in internal/repository (ARC-004).
type (
	PolicyVersionRow = repository.PolicyVersionRow
	PolicyLogRow     = repository.PolicyLogRow
	GovRepo          = repository.GovRepo
)

var _ GovRepo = (*txImpl)(nil)
