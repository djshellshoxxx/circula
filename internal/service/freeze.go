package service

import (
	"sort"
	"sync"

	"github.com/djshellshoxxx/circula/internal/domain"
)

// frozenSet is the in-memory set of networks whose ledger failed verification
// (CMD-07, OPS-033). A frozen network refuses every write that goes through
// Service.open with write scope, plus member creation and proposal expiry;
// reads and ledger verification still work so an administrator can see the
// state and the operator can investigate. Other networks are unaffected.
//
// The set is process memory by design: the start-up verification repopulates
// it on every start, and the periodic verification both adds and clears
// entries, so a restored or repaired network thaws on the next passing run.
type frozenSet struct {
	mu sync.RWMutex
	m  map[string]string // network id -> reason
}

// FreezeNetwork refuses writes to the network until UnfreezeNetwork is called.
// reason is kept for operators and never shown to members.
func (s *Service) FreezeNetwork(netID, reason string) {
	s.frozen.mu.Lock()
	defer s.frozen.mu.Unlock()
	if s.frozen.m == nil {
		s.frozen.m = map[string]string{}
	}
	s.frozen.m[netID] = reason
}

// UnfreezeNetwork lifts a freeze.
func (s *Service) UnfreezeNetwork(netID string) {
	s.frozen.mu.Lock()
	defer s.frozen.mu.Unlock()
	delete(s.frozen.m, netID)
}

// IsFrozen reports whether the network refuses writes.
func (s *Service) IsFrozen(netID string) bool {
	s.frozen.mu.RLock()
	defer s.frozen.mu.RUnlock()
	_, ok := s.frozen.m[netID]
	return ok
}

// FrozenNetworks returns the ids of frozen networks, sorted.
func (s *Service) FrozenNetworks() []string {
	s.frozen.mu.RLock()
	defer s.frozen.mu.RUnlock()
	out := make([]string, 0, len(s.frozen.m))
	for id := range s.frozen.m {
		out = append(out, id)
	}
	sort.Strings(out)
	return out
}

// checkNotFrozen returns the network-frozen error for a frozen network.
func (s *Service) checkNotFrozen(netID string) error {
	if s.IsFrozen(netID) {
		return domain.Errf(domain.CodeNetworkFrozen,
			"this network is read-only while its ledger is being checked; nothing was changed")
	}
	return nil
}
