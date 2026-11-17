package notifications

import (
	"errors"
	"fmt"
	"sort"
)

// Channels. Both are optional and default off for every member.
const (
	ChannelEmail = "email"
	ChannelPush  = "push"
)

// Notification kinds a member can opt in to (spec requirement 1). Every kind is
// delivered with the same generic body; the kind only selects the preference.
const (
	KindProposalReceived = "proposal_received"
	KindProposalResolved = "proposal_resolved" // accepted or declined
	KindDispute          = "dispute"
	KindProposalExpiring = "proposal_expiring"
	KindAccountSecurity  = "account_security"

	// KindEmailVerification is internal: it is created by AddEmailTarget so a
	// member can prove control of an address (double opt-in). It is never a
	// preference and carries no network activity.
	KindEmailVerification = "email_verification"
)

// Kinds lists the member-selectable kinds in display order.
var Kinds = []string{KindProposalReceived, KindProposalResolved, KindDispute, KindProposalExpiring, KindAccountSecurity}

// Channels lists the delivery channels in display order.
var Channels = []string{ChannelEmail, ChannelPush}

// ErrInvalid wraps every validation failure of untrusted input. The text of a
// wrapped error never echoes the rejected value.
var ErrInvalid = errors.New("notifications: invalid input")

// ErrNotFound is returned for a missing row and also for a row that belongs to
// another member or network, so callers cannot probe for foreign identifiers.
var ErrNotFound = errors.New("notifications: not found")

// ErrForbidden is returned when the trusted directory says the member is not
// active in the network.
var ErrForbidden = errors.New("notifications: member not active")

func invalid(format string, a ...any) error {
	return fmt.Errorf("%w: %s", ErrInvalid, fmt.Sprintf(format, a...))
}

// ValidKind reports whether k is a member-selectable preference kind.
func ValidKind(k string) bool {
	for _, x := range Kinds {
		if x == k {
			return true
		}
	}
	return false
}

// ValidChannel reports whether c is a supported channel.
func ValidChannel(c string) bool { return c == ChannelEmail || c == ChannelPush }

func sortedCopy(in []string) []string {
	out := append([]string(nil), in...)
	sort.Strings(out)
	return out
}
