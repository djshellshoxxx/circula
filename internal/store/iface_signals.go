package store

import "github.com/djshellshoxxx/circula/internal/repository"

// Compatibility aliases for the activity signal contract (SEC-026, ARC-004).
type (
	ActivitySignal       = repository.ActivitySignal
	ActivitySignalFilter = repository.ActivitySignalFilter
	SignalRepo           = repository.SignalRepo
)
