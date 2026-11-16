// Package favorites creates fresh, unpriced trade drafts from authorized history.
package favorites

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"strings"
	"unicode/utf8"
)

type Member struct {
	ID, Network              string
	Active, Visible, Blocked bool
}
type Favorite struct{ Network, Owner, Target, Nickname string }

func Save(owner, target Member, nickname string) (Favorite, error) {
	if owner.Network == "" || owner.Network != target.Network || owner.ID == "" || target.ID == "" || owner.ID == target.ID || !owner.Active || !target.Active || !target.Visible || owner.Blocked || target.Blocked || !utf8.ValidString(nickname) || utf8.RuneCountInString(nickname) > 80 || strings.ContainsAny(nickname, "\r\n\x00") {
		return Favorite{}, errors.New("favorite unavailable")
	}
	return Favorite{owner.Network, owner.ID, target.ID, nickname}, nil
}

type HistoricalTrade struct {
	Network, Payer, Payee, Description string
	Completed                          bool
}
type Draft struct {
	Network, Recipient, Description, Intent string
	Amount                                  *int64
}

// Repeat deliberately leaves Amount nil and never copies an old intent key.
func Repeat(owner, target Member, trade HistoricalTrade) (Draft, error) {
	if _, e := Save(owner, target, ""); e != nil {
		return Draft{}, e
	}
	if !trade.Completed || trade.Network != owner.Network || trade.Payer != owner.ID || trade.Payee != target.ID || len(trade.Description) > 4096 {
		return Draft{}, errors.New("trade unavailable")
	}
	b := make([]byte, 16)
	if _, e := rand.Read(b); e != nil {
		return Draft{}, e
	}
	return Draft{Network: owner.Network, Recipient: target.ID, Description: trade.Description, Intent: hex.EncodeToString(b)}, nil
}
