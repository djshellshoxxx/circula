package domain

// Audit actions for the one-time setup secret (SEC-028). Both are
// platform-scope events. The setup secret is "used" when the newest of the two
// events is ActSetupSecretUsed; no migration is needed because the audit log
// itself is the durable, append-only, hash-chained state.
const (
	ActSetupSecretUsed  = "setup.secret_used"
	ActSetupSecretReset = "setup.secret_reset"
)
