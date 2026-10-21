package service

import (
	"context"
	"sort"
	"sync"
	"time"

	"github.com/djshellshoxxx/circula/internal/domain"
	"github.com/djshellshoxxx/circula/internal/store"
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
	m  map[string]FreezeInfo // network id -> reason and start time
}

// FreezeInfo says why and since when a network is read-only. It lives in process memory only.
type FreezeInfo struct {
	Reason string
	Since  time.Time
}

// FreezeStatus is what a caller may see of a network's freeze. Reason is filled for
// administrators only; members see just that the network is read-only and since when.
type FreezeStatus struct {
	ReadOnly bool
	Since    time.Time
	Reason   string
}

// FreezeNetwork refuses writes to the network until UnfreezeNetwork is called.
// reason is kept for operators and administrators and never shown to other members.
// Freezing an already frozen network updates the reason and keeps the original start time.
func (s *Service) FreezeNetwork(netID, reason string) {
	s.frozen.mu.Lock()
	defer s.frozen.mu.Unlock()
	if s.frozen.m == nil {
		s.frozen.m = map[string]FreezeInfo{}
	}
	since := s.now()
	if cur, ok := s.frozen.m[netID]; ok {
		since = cur.Since
	}
	s.frozen.m[netID] = FreezeInfo{Reason: reason, Since: since}
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

// FrozenInfo returns the freeze reason and start time of a frozen network.
func (s *Service) FrozenInfo(netID string) (FreezeInfo, bool) {
	s.frozen.mu.RLock()
	defer s.frozen.mu.RUnlock()
	fi, ok := s.frozen.m[netID]
	return fi, ok
}

// NetworkFreezeStatus tells a member of the network whether it is read-only and since when;
// an administrator also gets the reason.
func (s *Service) NetworkFreezeStatus(ctx context.Context, actor domain.Actor, netID string) (FreezeStatus, error) {
	var out FreezeStatus
	err := s.view(ctx, func(tx store.Tx) error {
		p, err := s.open(tx, actor, netID, domain.ScopeRead)
		if err != nil {
			return err
		}
		if fi, ok := s.FrozenInfo(netID); ok {
			out = FreezeStatus{ReadOnly: true, Since: fi.Since}
			if p.requireAdmin() == nil {
				out.Reason = fi.Reason
			}
		}
		return nil
	})
	return out, err
}
