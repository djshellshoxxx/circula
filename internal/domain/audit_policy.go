package domain

// ActPolicyScaleNeutralised notes, once per policy version, that a stored
// version which changes the decimal scale came into force without its scale
// (DOM-031). Only an earlier build could store such a version; the network
// keeps the scale fixed at creation. It is written by a system job.
const ActPolicyScaleNeutralised = "policy.scale_change_neutralised"
