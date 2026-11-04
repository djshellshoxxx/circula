package store

import "github.com/djshellshoxxx/circula/internal/domain"

// SuspensionRepo covers network suspensions (GOV-023, GOV-024).
type SuspensionRepo interface {
	// PutSuspension records a suspension; false when one of that kind already exists.
	PutSuspension(netID string, s domain.Suspension) (bool, error)
	// DeleteSuspension lifts a suspension; false when none of that kind existed.
	DeleteSuspension(netID string, kind domain.SuspensionKind) (bool, error)
	ListSuspensions(netID string) ([]domain.Suspension, error)
}
