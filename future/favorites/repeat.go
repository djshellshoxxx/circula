// Package favorites provides owner-private saved trading partners and quick
// repeat drafts as an isolated, optional module.
//
// The pure validators in this file (Save, Repeat) operate on trusted adapter
// snapshots. The Service in service.go adds durable, owner- and
// network-scoped state behind the Store interface, with every authorization
// fact re-read through the injected Policy on every operation.
package favorites

import (
	"crypto/rand"
	"encoding/hex"
)

// Member is a trusted adapter snapshot of a member as seen by the acting
// owner. Blocked covers suspension and any block relation in either direction.
type Member struct {
	ID, Network              string
	Active, Visible, Blocked bool
	// DisplayName is the member's own public name, never the owner's nickname.
	DisplayName string
}

// Favorite is the owner-private relation. Nickname is visible only to Owner.
type Favorite struct{ Network, Owner, Target, Nickname string }

// Save validates that owner may favorite target and returns the relation with
// a cleaned nickname. It refuses self, cross-network, inactive, hidden and
// blocked targets uniformly with ErrUnavailable; a bad nickname is an InputError.
func Save(owner, target Member, nickname string) (Favorite, error) {
	nick, err := CleanNickname(nickname)
	if err != nil {
		return Favorite{}, err
	}
	if !validID(owner.ID) || !validID(target.ID) || !validID(owner.Network) || owner.Network != target.Network ||
		owner.ID == target.ID || !owner.Active || !target.Active || !target.Visible || owner.Blocked || target.Blocked {
		return Favorite{}, ErrUnavailable
	}
	return Favorite{Network: owner.Network, Owner: owner.ID, Target: target.ID, Nickname: nick}, nil
}

// HistoricalTrade is an authoritative snapshot of one past trade, loaded by a
// TradeSource for the requesting member.
type HistoricalTrade struct {
	ID, Network, Payer, Payee, Description string
	Completed                              bool
}

// Draft is an unpriced proposal prefill. Amount is always nil: the member must
// enter it and confirm through the host's normal proposal flow. Intent is a
// fresh random value for use as the new proposal's idempotency key; it is never
// the original trade's key. Draft carries no authority to post anything.
type Draft struct {
	Network, Recipient, Description, Intent string
	Amount                                  *int64
	// RequestKey identifies this deliberate user intent so retries of the same
	// action return the same Intent. It is set by Service, not by Repeat.
	RequestKey string
	// TradeID is the source trade, empty for a draft started from a favorite.
	TradeID string
	// RecipientName is the recipient's current public display name, for
	// presentation only. Service fills it from the live Policy snapshot.
	RecipientName string
}

// Repeat deliberately leaves Amount nil and never copies an old intent key.
// Only the payer of a completed trade may repeat it toward the same payee;
// repeating from the payee side (a request-direction trade) is not supported.
func Repeat(owner, target Member, trade HistoricalTrade) (Draft, error) {
	if _, e := Save(owner, target, ""); e != nil {
		return Draft{}, e
	}
	if !trade.Completed || trade.Network != owner.Network || trade.Payer != owner.ID || trade.Payee != target.ID || trade.Payer == trade.Payee {
		return Draft{}, ErrTradeUnavailable
	}
	desc, err := CleanDescription(trade.Description)
	if err != nil {
		return Draft{}, ErrTradeUnavailable
	}
	intent, err := newIntent()
	if err != nil {
		return Draft{}, err
	}
	return Draft{Network: owner.Network, Recipient: target.ID, Description: desc, Intent: intent, TradeID: trade.ID}, nil
}

func newIntent() (string, error) {
	b := make([]byte, 16)
	if _, e := rand.Read(b); e != nil {
		return "", e
	}
	return hex.EncodeToString(b), nil
}
