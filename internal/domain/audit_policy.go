package domain

// ActPolicyScaleChangeDetected notes, once per policy version, that a stored
// scheduled version changes the decimal scale (DOM-031). Only a build before
// c34d947 could store one. It is a report written by a system job: nothing is
// changed, and what to do with the version is an owner or operator decision.
const ActPolicyScaleChangeDetected = "policy.stored_scale_change_detected"
