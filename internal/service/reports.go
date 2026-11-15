package service

import (
	"context"
	"strings"
	"time"

	"github.com/djshellshoxxx/circula/internal/domain"
	"github.com/djshellshoxxx/circula/internal/store"
)

// Report limits (FR-024, SA-11 style abuse control). A member may file at most
// MaxReportsPerDay reports in any rolling 24 hours, and at most one open
// report per target.
const (
	MaxReportsPerDay = 10
	maxReportNote    = 500
)

// ReportCategories are the accepted report categories.
var ReportCategories = []string{"spam", "scam", "inappropriate", "harassment", "other"}

// Report target types.
const (
	ReportListing = "listing"
	ReportMember  = "member"
)

// ReportInput files a report about a listing or a member (FR-024).
type ReportInput struct {
	TargetType     string // listing | member
	TargetID       string
	Category       string
	Note           string // optional, bounded
	IdempotencyKey string `json:"-"`
}

// CreateReport lets an active member report a listing or another member. It
// changes nothing about the target: no hiding, suspension or flag. The reporter
// is visible to administrators only (the queue is administrator-only and the
// audit event is about the reporter, not the reported member). A second open
// report by the same member about the same target is a conflict; more than
// MaxReportsPerDay in 24 hours is rate-limited.
func (s *Service) CreateReport(ctx context.Context, actor domain.Actor, netID string, in ReportInput) (*store.Report, error) {
	var out *store.Report
	err := s.update(ctx, func(tx store.Tx) error {
		p, err := s.open(tx, actor, netID, domain.ScopeWrite)
		if err != nil {
			return err
		}
		if err := p.requireActive(); err != nil {
			return err
		}
		out, err = idempotent(s, tx, netID, actor, "report.create", in.IdempotencyKey, in, func() (string, error) {
			return s.createReport(tx, p, in)
		}, func(id string) (*store.Report, error) { return tx.GetReport(netID, id) })
		return err
	})
	return out, err
}

func (s *Service) createReport(tx store.Tx, p *principal, in ReportInput) (string, error) {
	netID := p.net.ID
	if err := p.requireNetworkActive(); err != nil {
		return "", err
	}
	valid := false
	for _, c := range ReportCategories {
		valid = valid || c == in.Category
	}
	if !valid {
		return "", domain.Invalid("category", "must be one of "+strings.Join(ReportCategories, ", "))
	}
	note := strings.TrimSpace(in.Note)
	if err := boundedText("note", note, maxReportNote); err != nil {
		return "", err
	}
	switch in.TargetType {
	case ReportListing:
		l, err := tx.GetListing(netID, in.TargetID)
		if err != nil {
			return "", domain.NotFound("listing")
		}
		if l.MemberID == p.member.ID {
			return "", domain.Invalid("target_id", "you cannot report your own listing")
		}
	case ReportMember:
		m, err := tx.GetMember(netID, in.TargetID)
		if err != nil || m.Status == domain.MemberPending {
			return "", domain.NotFound("member")
		}
		if m.ID == p.member.ID {
			return "", domain.Invalid("target_id", "you cannot report yourself")
		}
	default:
		return "", domain.Invalid("target_type", "must be listing or member")
	}
	now := s.now()
	n, err := tx.CountReportsSince(netID, p.member.ID, now.Add(-24*time.Hour))
	if err != nil {
		return "", err
	}
	if n >= MaxReportsPerDay {
		return "", domain.Errf(domain.CodeRateLimited, "too many reports today; try again tomorrow")
	}
	r := store.Report{ID: s.id(), NetworkID: netID, ReporterID: p.member.ID, TargetType: in.TargetType, TargetID: in.TargetID,
		Category: in.Category, Note: note, Status: "open", CreatedAt: now}
	if err := tx.InsertReport(r); err != nil {
		if domain.CodeOf(err) == domain.CodeConflict {
			return "", domain.Errf(domain.CodeConflict, "you already have an open report about this")
		}
		return "", err
	}
	return r.ID, s.audit(tx, p, auditEntry{Action: domain.ActReportCreated, TargetType: "report", TargetID: r.ID,
		IdemRef: keyRef(in.IdempotencyKey), Subjects: []string{p.member.ID}, // not the reported member (FR-024)
		After: map[string]any{"status": "open", "category": in.Category, "target_type": in.TargetType}})
}

// ReportView is a report with display names for the administrator queue.
type ReportView struct {
	*store.Report
	ReporterName string
	TargetLabel  string // member display name or listing title
}

// ListReports lists the report queue, newest first, to administrators only.
// status is "open", "handled" or "" for all.
func (s *Service) ListReports(ctx context.Context, actor domain.Actor, netID, status string, pg Page) ([]ReportView, error) {
	var out []ReportView
	if status != "" && status != "open" && status != "handled" {
		return nil, domain.Invalid("status", "must be open or handled")
	}
	err := s.view(ctx, func(tx store.Tx) error {
		p, err := s.open(tx, actor, netID, domain.ScopeRead)
		if err != nil {
			return err
		}
		if err := p.requireAdmin(); err != nil {
			return err
		}
		rs, err := tx.ListReports(netID, store.ReportFilter{Status: status, AfterID: pg.AfterID, Limit: clampPage(pg)})
		if err != nil {
			return err
		}
		for _, r := range rs {
			v := ReportView{Report: r}
			if m, err := tx.GetMember(netID, r.ReporterID); err == nil {
				v.ReporterName = m.DisplayName
			}
			switch r.TargetType {
			case ReportMember:
				if m, err := tx.GetMember(netID, r.TargetID); err == nil {
					v.TargetLabel = m.DisplayName
				}
			case ReportListing:
				if l, err := tx.GetListing(netID, r.TargetID); err == nil {
					v.TargetLabel = l.Title
				}
			}
			out = append(out, v)
		}
		return nil
	})
	return out, err
}

// HandleReportInput marks a report handled.
type HandleReportInput struct {
	ReportID       string
	Note           string // optional outcome note, kept on the report for administrators
	IdempotencyKey string `json:"-"`
}

// HandleReport marks an open report handled (administrators only), audits it
// and tells the reporter that it was reviewed. It does not itself act on the
// target; the administrator uses the normal, audited member and listing tools.
func (s *Service) HandleReport(ctx context.Context, actor domain.Actor, netID string, in HandleReportInput) (*store.Report, error) {
	var out *store.Report
	err := s.update(ctx, func(tx store.Tx) error {
		p, err := s.open(tx, actor, netID, domain.ScopeWrite)
		if err != nil {
			return err
		}
		if err := p.requireAdmin(); err != nil {
			return err
		}
		out, err = idempotent(s, tx, netID, actor, "report.handle", in.IdempotencyKey, in, func() (string, error) {
			r, err := tx.GetReport(netID, in.ReportID)
			if err != nil {
				return "", domain.NotFound("report")
			}
			if r.Status != "open" {
				return "", domain.InvalidState("report is already handled")
			}
			note := strings.TrimSpace(in.Note)
			if err := boundedText("note", note, maxReportNote); err != nil {
				return "", err
			}
			now := s.now()
			r.Status, r.HandledBy, r.HandledAt, r.HandlingNote = "handled", p.memberID(), &now, note
			if err := tx.UpdateReport(*r); err != nil {
				return "", err
			}
			if err := s.notify(tx, netID, r.ReporterID, NotifReportHandled, "Your report was reviewed",
				"An administrator reviewed your report. Thank you for raising it.", "", ""); err != nil {
				return "", err
			}
			return r.ID, s.audit(tx, p, auditEntry{Type: p.actorType(domain.ActorAdmin), Action: domain.ActReportHandled,
				TargetType: "report", TargetID: r.ID, Reason: note, IdemRef: keyRef(in.IdempotencyKey),
				Subjects: []string{r.ReporterID}, Before: map[string]any{"status": "open"}, After: map[string]any{"status": "handled"}})
		}, func(id string) (*store.Report, error) { return tx.GetReport(netID, id) })
		return err
	})
	return out, err
}
