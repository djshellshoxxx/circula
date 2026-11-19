package repository

// RestoredCredentialCounts records authority invalidated after restoring a snapshot.
// Break-glass grants remain append-only history; closing their sessions ends them.
type RestoredCredentialCounts struct {
	Sessions           int64
	Tokens             int64
	MFAChallenges      int64
	BreakGlassSessions int64
}
