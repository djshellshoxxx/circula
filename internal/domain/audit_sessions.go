package domain

// Audit actions of session and sign-up controls (SEC-027, OPS-010, SEC-032).
const (
	// ActSessionsRevoked is the platform audit event of `circula
	// revoke-sessions`: the operator ended every session, or every session of
	// one person, for example after a suspected secret exposure or when
	// rotating the web signing key. It carries the count and the reason only.
	ActSessionsRevoked = "sessions.revoked"
	// ActOpenGrantRiskConfirmed records that a network administrator confirmed
	// the Sybil risk of combining open join mode with a sign-up grant
	// (SEC-032). It is written in the same unit of work as the policy change
	// (or network creation) it confirms.
	ActOpenGrantRiskConfirmed = "policy.open_grant_risk_confirmed"
)
