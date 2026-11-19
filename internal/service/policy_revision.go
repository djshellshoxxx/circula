package service

import (
	"errors"

	"github.com/djshellshoxxx/circula/internal/domain"
	store "github.com/djshellshoxxx/circula/internal/repository"
)

// PolicyRevision identifies the effective policy and its newest pending
// change. These are the same versions bound by the administrative policy ETag.
type PolicyRevision struct {
	Version        int
	PendingVersion int
}

// ErrPolicyChanged means a conditional policy update lost its precondition
// inside the write transaction. No policy, audit or notification was written.
var ErrPolicyChanged = errors.New("policy changed since it was read")

func checkPolicyRevision(tx store.Tx, net *domain.Network, expected *PolicyRevision) error {
	if expected == nil {
		return nil
	}
	current := PolicyRevision{Version: net.PolicyVersion}
	if g, ok := tx.(store.GovRepo); ok {
		latest, err := g.LatestPolicyVersion(net.ID)
		if err != nil {
			return err
		}
		// The newest stored version cannot be withdrawn by an older one. If
		// it is above the effective version, it is the newest pending change.
		if latest > current.Version {
			current.PendingVersion = latest
		}
	}
	if current != *expected {
		return ErrPolicyChanged
	}
	return nil
}
