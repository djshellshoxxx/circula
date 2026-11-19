package store

import "github.com/djshellshoxxx/circula/internal/repository"

// Compatibility aliases preserve the existing storage API and exact type
// identity. Driver-neutral contracts live in internal/repository (ARC-004).
type (
	Store           = repository.Store
	Tx              = repository.Tx
	NetworkRepo     = repository.NetworkRepo
	PersonRepo      = repository.PersonRepo
	SessionPosition = repository.SessionPosition
	MemberFilter    = repository.MemberFilter
	MemberRepo      = repository.MemberRepo
	AccountRepo     = repository.AccountRepo
	ListingFilter   = repository.ListingFilter
	ListingRepo     = repository.ListingRepo
	InvitationRepo  = repository.InvitationRepo
	TxFilter        = repository.TxFilter
	TxnRepo         = repository.TxnRepo
	DisputeFilter   = repository.DisputeFilter
	DisputeEvent    = repository.DisputeEvent
	DisputeRepo     = repository.DisputeRepo
	AuditFilter     = repository.AuditFilter
	AuditRepo       = repository.AuditRepo
	IdemRecord      = repository.IdemRecord
	IdemRepo        = repository.IdemRepo
	ConsistencyRun  = repository.ConsistencyRun
	BalanceMismatch = repository.BalanceMismatch
	VerifyRepo      = repository.VerifyRepo
	ExportTable     = repository.ExportTable
	ExportRepo      = repository.ExportRepo
)

const (
	TblMembers      = repository.TblMembers
	TblAccounts     = repository.TblAccounts
	TblListings     = repository.TblListings
	TblTransactions = repository.TblTransactions
	TblStatusEvents = repository.TblStatusEvents
	TblEntries      = repository.TblEntries
	TblDisputes     = repository.TblDisputes
	TblPolicies     = repository.TblPolicies
	TblAudit        = repository.TblAudit
	TblRoles        = repository.TblRoles
)
