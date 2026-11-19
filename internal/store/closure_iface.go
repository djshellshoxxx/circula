package store

import "github.com/djshellshoxxx/circula/internal/repository"

// Compatibility aliases preserve the existing storage API and exact type
// identity. Driver-neutral contracts live in internal/repository (ARC-004).
type (
	Challenge   = repository.Challenge
	ClosureRepo = repository.ClosureRepo
)
