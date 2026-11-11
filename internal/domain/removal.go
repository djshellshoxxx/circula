package domain

// Removal reasons (GOV-011). An administrator may remove a member only for a
// reason from this documented list. It is exactly the list named in
// docs/specs/10-governance-operations.md GOV-011; adding or changing a reason
// is an owner decision, not a code change made in passing.
const (
	RemovalRuleBreach         = "rule_breach"          // breach of network rules
	RemovalConfirmedFraud     = "confirmed_fraud"      // confirmed fraud
	RemovalDisputeNonResponse = "dispute_non_response" // prolonged non-response to a dispute
)

// RemovalReasons lists the documented removal reasons in display order.
var RemovalReasons = []string{RemovalRuleBreach, RemovalConfirmedFraud, RemovalDisputeNonResponse}

var removalLabels = map[string]string{
	RemovalRuleBreach:         "Breach of network rules",
	RemovalConfirmedFraud:     "Confirmed fraud",
	RemovalDisputeNonResponse: "Prolonged non-response to a dispute",
}

// RemovalReasonLabel is the plain-language label of a removal reason; "" for
// a code that is not on the documented list.
func RemovalReasonLabel(code string) string { return removalLabels[code] }

// RemovalNeedsIndependence reports whether a removal for this reason must not
// be decided by an administrator who is the counterparty in the matter
// (GOV-011, SEC-010): removals for fraud or for a breach of the rules.
func RemovalNeedsIndependence(code string) bool {
	return code == RemovalConfirmedFraud || code == RemovalRuleBreach
}
