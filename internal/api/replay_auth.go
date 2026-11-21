package api

import (
	"encoding/json"

	"github.com/djshellshoxxx/circula/internal/service"
)

// replayPermission is explicit and fails closed: registering another cached
// operation also requires choosing its replay authorization. HTTP methods
// alone cannot distinguish read-scope exports from write-scope commands.
func replayPermission(pattern string) service.ReplayPermission {
	switch pattern {
	case "POST " + v1 + "/listings", "POST " + v1 + "/disputes",
		"POST " + v1 + "/disputes/{id}/re-review", "POST " + v1 + "/reports":
		return service.ReplayActiveMember
	case "POST " + v1 + "/transactions", "POST " + v1 + "/transactions/{id}/accept":
		return service.ReplayTrade
	case "POST " + v1 + "/me/terms/accept":
		return service.ReplayAcceptTerms
	case "POST " + v1 + "/me/membership/departure":
		return service.ReplayDeparture
	case "POST " + v1 + "/transactions/{id}/decline":
		return service.ReplayDecline
	case "POST " + v1 + "/transactions/{id}/cancel":
		return service.ReplayCancel
	case "POST " + v1 + "/disputes/{id}/response":
		return service.ReplayDisputeResponse
	case "POST " + v1 + "/challenges":
		return service.ReplayChallenge
	case "POST " + v1 + "/admin/reports/{id}/handle":
		return service.ReplayReportHandle
	case "POST " + v1 + "/admin/transactions/{id}/reversal", "POST " + v1 + "/admin/adjustments",
		"POST " + v1 + "/admin/members/{id}/approve", "POST " + v1 + "/admin/members/{id}/deny",
		"POST " + v1 + "/admin/members/{id}/reinstate",
		"POST " + v1 + "/admin/announcements":
		return service.ReplayAdmin
	case "POST " + v1 + "/admin/terms", "POST " + v1 + "/admin/members/{id}/suspend":
		return service.ReplayLastingAdmin
	case "POST " + v1 + "/admin/members/{id}/remove":
		return service.ReplayMemberRemoval
	case "POST " + v1 + "/disputes/{id}/resolve", "POST " + v1 + "/disputes/{id}/re-review/decision",
		"POST " + v1 + "/challenges/{id}/decision":
		return service.ReplayDecider
	case "POST " + v1 + "/admin/consistency-checks":
		return service.ReplayConsistencyCheck
	case "POST " + v1 + "/admin/invitations":
		return service.ReplayInvitationCreate
	case "POST " + v1 + "/invitations/{token}/accept":
		return service.ReplayInvitationAccept
	case "POST " + v1 + "/exports":
		return service.ReplayExport
	default:
		return service.ReplayDenied
	}
}

// authorizeReplay runs only for an existing successful cache entry. A refusal
// must leave that entry intact: deleting it could execute the command again.
func (s *server) authorizeReplay(c *reqCtx, rt route, rec *idemRec) error {
	if rec == nil {
		return errInternal
	}
	in := service.ReplayAuthorization{Permission: replayPermission(rt.pattern),
		TransactionID: c.r.PathValue("id"), MemberID: c.r.PathValue("id"), ReportID: c.r.PathValue("id")}
	netID := c.netID
	switch in.Permission {
	case service.ReplayInvitationAccept:
		// This noNet route consumes the invitation on its first execution.
		// The successful response binds the network and resulting membership;
		// neither the request token nor a new network header supplies them.
		var m membershipDTO
		if err := json.Unmarshal(rec.body, &m); err != nil || m.NetworkID == "" || m.MemberID == "" {
			return errInternal
		}
		netID, in.MemberID = m.NetworkID, m.MemberID
	case service.ReplayDeparture:
		var m struct {
			MemberID string `json:"member_id"`
		}
		if err := json.Unmarshal(rec.body, &m); err != nil || m.MemberID == "" {
			return errInternal
		}
		in.MemberID = m.MemberID
	case service.ReplayExport:
		var e exportDTO
		if err := json.Unmarshal(rec.body, &e); err != nil {
			return errInternal
		}
		scope, ok := exportScopes[e.Scope]
		if !ok {
			return errInternal
		}
		in.ExportScope = scope
	}
	return s.svc.AuthorizeReplay(c.r.Context(), c.actor, netID, in)
}
