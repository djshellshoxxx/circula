package api

import (
	"time"

	"github.com/djshellshoxxx/circula/internal/service"
)

// activitySignalDTO is one velocity or reciprocal-pattern signal (SEC-026).
// observed_count counts the events in the window_minutes before raised_at;
// period_start/period_end bound the fixed period in which at most one
// signal per member (or pair) and kind is raised.
type activitySignalDTO struct {
	ID              string  `json:"id"`
	Kind            string  `json:"kind"`
	MemberID        string  `json:"member_id"`
	MemberName      string  `json:"member_display_name"`
	OtherMemberID   *string `json:"other_member_id"`
	OtherMemberName *string `json:"other_member_display_name"`
	ObservedCount   int     `json:"observed_count"`
	Threshold       int     `json:"threshold"`
	WindowMinutes   int     `json:"window_minutes"`
	PeriodStart     string  `json:"period_start"`
	PeriodEnd       string  `json:"period_end"`
	RaisedAt        string  `json:"raised_at"`
}

func toActivitySignal(v service.ActivitySignalView) activitySignalDTO {
	d := activitySignalDTO{ID: v.ID, Kind: v.Kind, MemberID: v.MemberID, MemberName: v.MemberName,
		ObservedCount: v.Observed, Threshold: v.Threshold, WindowMinutes: int(v.Window / time.Minute),
		PeriodStart: ts(v.PeriodStart), PeriodEnd: ts(v.PeriodStart.Add(v.Window)), RaisedAt: ts(v.RaisedAt)}
	if v.OtherMemberID != "" {
		d.OtherMemberID, d.OtherMemberName = strPtr(v.OtherMemberID), strPtr(v.OtherMemberName)
	}
	return d
}

// activitySignals lists the network's activity signals to its administrators
// and dispute reviewers (SEC-026); the service decides who may read them.
func (s *server) activitySignals(c *reqCtx) error {
	if err := c.only("limit", "cursor", "kind"); err != nil {
		return err
	}
	kind := c.query.Get("kind")
	if kind != "" && kind != "velocity" && kind != "reciprocal" {
		return invalidQuery("kind must be velocity or reciprocal")
	}
	p, err := c.page(kind)
	if err != nil {
		return err
	}
	rows, next, err := collect(c, p, func(pg service.Page) ([]service.ActivitySignalView, error) {
		return s.svc.ListActivitySignals(c.r.Context(), c.actor, c.netID, service.ActivitySignalQuery{Kind: kind, Page: pg})
	}, func(v service.ActivitySignalView) string { return v.ID }, nil)
	if err != nil {
		return err
	}
	out := make([]activitySignalDTO, 0, len(rows))
	for _, v := range rows {
		out = append(out, toActivitySignal(v))
	}
	return listOK(c, out, next)
}
