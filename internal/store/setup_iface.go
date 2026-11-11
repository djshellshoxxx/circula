package store

// SetupStateRepo reads the platform-scope audit events that record the
// one-time setup secret state (SEC-028).
type SetupStateRepo interface {
	// LastPlatformAuditSeq returns the sequence number of the newest
	// platform-scope audit event with the given action; found is false when
	// there is none.
	LastPlatformAuditSeq(action string) (seq int64, found bool, err error)
}
