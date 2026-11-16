// Package governance validates non-binding ballots, with no policy-write API.
package governance

import (
	"errors"
	"time"
)

type Choice string

const (
	Yes     Choice = "yes"
	No      Choice = "no"
	Abstain Choice = "abstain"
)

type Poll struct {
	Network, ID   string
	Opens, Closes time.Time
	Withdrawn     bool
	Eligible      map[string]bool
}
type Ballot struct {
	Network, Poll, Member string
	Choice                Choice
	CastAt                time.Time
}

func valid(c Choice) bool { return c == Yes || c == No || c == Abstain }

// Cast checks a trusted snapshot; the adapter must enforce the unique ballot
// constraint atomically. Current membership remains required after snapshotting.
func Cast(p Poll, network, member string, active bool, c Choice, now time.Time, existing []Ballot) (Ballot, error) {
	if p.Network == "" || network != p.Network || p.ID == "" || member == "" || !active || !p.Eligible[member] || p.Withdrawn || !p.Opens.Before(p.Closes) || now.Before(p.Opens) || !now.Before(p.Closes) || !valid(c) {
		return Ballot{}, errors.New("ballot unavailable")
	}
	for _, b := range existing {
		if b.Network == network && b.Poll == p.ID && b.Member == member {
			return Ballot{}, errors.New("already voted")
		}
	}
	return Ballot{network, p.ID, member, c, now}, nil
}

type Outcome struct {
	Yes, No, Abstain, Eligible int
	Advisory                   bool
}

func Tally(p Poll, now time.Time, ballots []Ballot) (Outcome, error) {
	if p.Network == "" || p.ID == "" || !p.Opens.Before(p.Closes) || now.Before(p.Closes) || p.Withdrawn {
		return Outcome{}, errors.New("poll not final")
	}
	o := Outcome{Advisory: true}
	for _, eligible := range p.Eligible {
		if eligible {
			o.Eligible++
		}
	}
	seen := map[string]bool{}
	for _, b := range ballots {
		if b.Network != p.Network || b.Poll != p.ID || b.Member == "" || !p.Eligible[b.Member] || seen[b.Member] || b.CastAt.Before(p.Opens) || !b.CastAt.Before(p.Closes) || !valid(b.Choice) {
			return Outcome{}, errors.New("invalid ballot snapshot")
		}
		seen[b.Member] = true
		switch b.Choice {
		case Yes:
			o.Yes++
		case No:
			o.No++
		case Abstain:
			o.Abstain++
		}
	}
	return o, nil
}
