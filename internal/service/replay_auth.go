package service

import (
	"context"

	"github.com/djshellshoxxx/circula/internal/domain"
	store "github.com/djshellshoxxx/circula/internal/repository"
)

// ReplayPermission names the current authority required to disclose a cached
// command response. It deliberately excludes command state transitions: a
// successful accept, departure or decision must not execute a second time.
type ReplayPermission uint8

const (
	ReplayDenied ReplayPermission = iota
	ReplayActiveMember
	ReplayTrade
	ReplayAcceptTerms
	ReplayDeparture
	ReplayDecline
	ReplayCancel
	ReplayDisputeResponse
	ReplayChallenge
	ReplayAdmin
	ReplayLastingAdmin
	ReplayMemberRemoval
	ReplayDecider
	ReplayConsistencyCheck
	ReplayInvitationCreate
	ReplayInvitationAccept
	ReplayReportHandle
	ReplayExport
)

// ReplayAuthorization carries only server-selected permission and identifiers
// bound to the original successful request or its cached response.
type ReplayAuthorization struct {
	Permission    ReplayPermission
	MemberID      string
	TransactionID string
	ReportID      string
	ExportScope   ExportScope
}

// AuthorizeReplay rechecks current scope, effective membership, role, MFA and
// freeze restrictions without running a command or modifying durable replay
// records. Business state changed by the successful command is not rechecked.
func (s *Service) AuthorizeReplay(ctx context.Context, actor domain.Actor, netID string, in ReplayAuthorization) error {
	if in.Permission <= ReplayDenied || in.Permission > ReplayExport {
		return domain.Forbidden("cached response is not available for this operation")
	}
	if in.Permission == ReplayExport {
		// Export creation is a read-scope operation, even though HTTP uses POST.
		return s.AuthorizeExportDownload(ctx, actor, netID, in.ExportScope)
	}
	if in.Permission == ReplayInvitationAccept {
		if err := joinableActor(actor); err != nil {
			return err
		}
	}
	return s.view(ctx, func(tx store.Tx) error {
		var p *principal
		var err error
		if in.Permission == ReplayConsistencyCheck {
			// Verification remains available while the ledger is frozen.
			p, err = s.openAny(tx, actor, netID, domain.ScopeWrite)
		} else {
			p, err = s.open(tx, actor, netID, domain.ScopeWrite)
		}
		if err != nil {
			return err
		}
		switch in.Permission {
		case ReplayActiveMember:
			return p.requireActive()
		case ReplayTrade:
			if err := p.requireActive(); err != nil {
				return err
			}
			return s.requireTerms(tx, p)
		case ReplayAcceptTerms, ReplayInvitationAccept:
			if p.member == nil || (p.member.Status != domain.MemberActive && p.member.Status != domain.MemberPending) {
				return domain.Forbidden("membership cannot perform this action")
			}
			if in.Permission == ReplayInvitationAccept && (in.MemberID == "" || p.member.ID != in.MemberID) {
				return domain.NotFound("member")
			}
			return nil
		case ReplayDeparture:
			return authorizeDepartureReplay(p, in.MemberID)
		case ReplayDecline, ReplayCancel:
			if err := requireReplayExitMember(p); err != nil {
				return err
			}
			t, err := tx.GetTransaction(netID, in.TransactionID)
			if err != nil {
				return domain.NotFound("transaction")
			}
			if !t.IsParty(p.member.ID) {
				if in.Permission != ReplayCancel {
					return domain.NotFound("transaction")
				}
				return p.requireAdmin()
			}
			if (in.Permission == ReplayCancel) != (t.CreatedBy == p.member.ID) {
				return domain.Forbidden("membership cannot perform this action")
			}
			return nil
		case ReplayDisputeResponse:
			return requireReplayExitMember(p)
		case ReplayChallenge:
			if p.member == nil || p.member.Status == domain.MemberPending {
				return domain.Forbidden("membership cannot perform this action")
			}
			return nil
		case ReplayMemberRemoval:
			// The removal endpoint also permits a member to remove themselves.
			if p.member != nil && p.member.ID == in.MemberID {
				return authorizeDepartureReplay(p, in.MemberID)
			}
			if err := p.requireAdmin(); err != nil {
				return err
			}
			return p.requireLasting()
		case ReplayLastingAdmin:
			if err := p.requireAdmin(); err != nil {
				return err
			}
			return p.requireLasting()
		case ReplayAdmin, ReplayConsistencyCheck:
			return p.requireAdmin()
		case ReplayReportHandle:
			if err := p.requireAdmin(); err != nil {
				return err
			}
			_, err := loadIndependentReport(tx, p, in.ReportID)
			return err
		case ReplayInvitationCreate:
			if err := p.requireAdmin(); err != nil {
				return err
			}
			if p.net.Status == domain.NetworkClosed {
				return domain.InvalidState("network is closed")
			}
			return nil
		case ReplayDecider:
			return p.requireDecider()
		default:
			return domain.Forbidden("cached response is not available for this operation")
		}
	})
}

func requireReplayExitMember(p *principal) error {
	if p.member == nil || (p.member.Status != domain.MemberActive && p.member.Status != domain.MemberSuspended) {
		return domain.Forbidden("membership cannot perform this action")
	}
	return nil
}

func authorizeDepartureReplay(p *principal, memberID string) error {
	if p.member == nil || memberID == "" || p.member.ID != memberID || p.member.Status != domain.MemberDeparted {
		return domain.Forbidden("cached departure does not belong to this membership")
	}
	return nil
}
