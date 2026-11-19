package messages

import "testing"

// UX-047: the network picker keeps the complete onboarding guidance and the
// existing lowercase membership labels without translating network names.
func TestNetworkPickerMessages(t *testing.T) {
	for key, want := range map[string]string{
		"onboarding.networks.title":            "Your networks",
		"onboarding.networks.empty":            "You are not a member of any network yet. Open an invitation link from a network administrator to join one, or start your own.",
		"onboarding.networks.create":           "Create a network",
		"onboarding.networks.status.pending":   "pending",
		"onboarding.networks.status.active":    "active",
		"onboarding.networks.status.suspended": "suspended",
		"onboarding.networks.status.departed":  "departed",
	} {
		t.Run(key, func(t *testing.T) {
			if got := Text(key); got != want {
				t.Fatalf("got %q, want %q", got, want)
			}
		})
	}
}
