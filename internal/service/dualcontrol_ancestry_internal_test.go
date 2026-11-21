package service

import (
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/djshellshoxxx/circula/internal/domain"
	store "github.com/djshellshoxxx/circula/internal/repository"
)

type grantProofClock struct{ at time.Time }

func (c grantProofClock) Now() time.Time { return c.at }

// An inert repository fixture exercises only the authorization predicate. Its
// provenance labels are not people, memberships, or executable grant actions.
type grantProofTx struct {
	store.Tx
	grants   map[string]string
	at       time.Time
	failure  error
	lookups  int
	wrongNet bool
}

func (tx *grantProofTx) LatestAdminGrant(netID, memberID string) (string, time.Time, bool, error) {
	tx.lookups++
	tx.wrongNet = tx.wrongNet || netID != "network"
	by, ok := tx.grants[memberID]
	return by, tx.at, ok, tx.failure
}

func grantProofPath(n int, operator bool) map[string]string {
	grants := make(map[string]string, n)
	for i := range n {
		id := fmt.Sprintf("provenance-%d", i)
		switch {
		case i < n-1:
			grants[id] = fmt.Sprintf("provenance-%d", i+1)
		case operator:
			grants[id] = ""
		default:
			grants[id] = id
		}
	}
	return grants
}

func TestIndependentApproverRequiresCompleteGrantProof(t *testing.T) {
	now := time.Date(2026, 11, 20, 12, 0, 0, 0, time.UTC)
	lookupFailure := errors.New("grant history unavailable")
	for _, tc := range []struct {
		name       string
		grants     map[string]string
		fresh      bool
		breakGlass bool
		noMember   bool
		failure    error
		want       error
		lookups    int
	}{
		{name: "founder root has no wait", grants: grantProofPath(1, false), fresh: true, lookups: 1},
		{name: "operator root at waiting boundary", grants: grantProofPath(1, true), lookups: 1},
		{name: "operator root still waits", grants: grantProofPath(1, true), fresh: true, want: domain.ErrForbidden, lookups: 1},
		{name: "founder root at work limit", grants: grantProofPath(maxGrantChain, false), lookups: maxGrantChain},
		{name: "operator root at work limit", grants: grantProofPath(maxGrantChain, true), lookups: maxGrantChain},
		{name: "root beyond work limit", grants: grantProofPath(maxGrantChain+1, true), want: domain.ErrForbidden, lookups: maxGrantChain},
		{name: "missing current grant", want: domain.ErrForbidden, lookups: 1},
		{name: "missing ancestor", grants: map[string]string{"provenance-0": "missing"}, want: domain.ErrForbidden, lookups: 2},
		{name: "repeated ancestor", grants: map[string]string{"provenance-0": "provenance-1", "provenance-1": "provenance-0"}, want: domain.ErrForbidden, lookups: 2},
		{name: "lookup failure", failure: lookupFailure, want: lookupFailure, lookups: 1},
		{name: "live break-glass authority", breakGlass: true, failure: lookupFailure},
		{name: "absent member has no root", noMember: true, want: domain.ErrForbidden},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s := &Service{cfg: Config{Clock: grantProofClock{now}, NewAdminApprovalDelay: DefaultNewAdminApprovalDelay}}
			tx := &grantProofTx{grants: tc.grants, at: now.Add(-DefaultNewAdminApprovalDelay), failure: tc.failure}
			if tc.fresh {
				tx.at = now
			}
			p := &principal{member: &domain.Member{ID: "provenance-0"}, bgAdmin: tc.breakGlass}
			if tc.noMember {
				p.member = nil
			}
			pa := &domain.PendingAdjustment{NetworkID: "network", Proposer: "request-member"}
			err := s.requireIndependentApprover(tx, p, pa)
			if !errors.Is(err, tc.want) {
				t.Fatalf("authorization result: got %v, want %v", err, tc.want)
			}
			if tx.lookups != tc.lookups || tx.wrongNet {
				t.Fatalf("grant lookups: got %d, want %d; wrong network=%v", tx.lookups, tc.lookups, tx.wrongNet)
			}
		})
	}
}
