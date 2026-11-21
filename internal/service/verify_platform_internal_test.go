package service

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/djshellshoxxx/circula/internal/domain"
	store "github.com/djshellshoxxx/circula/internal/repository"
)

// These inert repository rows exercise the shared chain verifier without
// editing audit records or disabling protections in a database.
type platformAuditProbe struct {
	store.Store
	store.Tx
	events     []*domain.AuditEvent
	headSeq    int64
	headHash   string
	headFound  bool
	listError  error
	headError  error
	views      int
	wrongScope bool
}

func (p *platformAuditProbe) View(_ context.Context, fn func(store.Tx) error) error {
	p.views++
	return fn(p)
}

func (p *platformAuditProbe) ListAudit(scope string, filter store.AuditFilter) ([]*domain.AuditEvent, error) {
	p.wrongScope = p.wrongScope || scope != ""
	if filter.AfterSeq != 0 {
		return nil, nil
	}
	return p.events, p.listError
}

func (p *platformAuditProbe) AuditHead(scope string) (int64, string, bool, error) {
	p.wrongScope = p.wrongScope || scope != ""
	return p.headSeq, p.headHash, p.headFound, p.headError
}

type platformAuditClock struct{ at time.Time }

func (c platformAuditClock) Now() time.Time { return c.at }

func TestVerifyPlatformAuditScopeAndFindings(t *testing.T) {
	at := time.Date(2026, 11, 20, 12, 0, 0, 0, time.UTC)
	first := domain.AuditEvent{ID: "first", Seq: 1, OccurredAt: at, Action: "fixture.created"}
	first.Hash = AuditHash(&first)
	second := domain.AuditEvent{ID: "second", Seq: 2, OccurredAt: at.Add(time.Second), Action: "fixture.updated", PrevHash: first.Hash}
	second.Hash = AuditHash(&second)
	contentMismatch := first
	contentMismatch.Hash = strings.Repeat("0", 64)
	gap := second
	gap.Seq = 3
	gap.Hash = AuditHash(&gap)
	brokenLink := second
	brokenLink.PrevHash = "other"
	brokenLink.Hash = AuditHash(&brokenLink)
	lookupFailure := errors.New("audit storage unavailable")
	for _, tc := range []struct {
		name    string
		probe   platformAuditProbe
		finding string
		wantErr error
	}{
		{name: "empty history"},
		{name: "zero head is not empty history", probe: platformAuditProbe{headFound: true}, finding: "invalid sequence 0"},
		{name: "complete history", probe: platformAuditProbe{events: []*domain.AuditEvent{&first}, headSeq: 1, headHash: first.Hash, headFound: true}},
		{name: "legitimate growth", probe: platformAuditProbe{events: []*domain.AuditEvent{&first, &second}, headSeq: 2, headHash: second.Hash, headFound: true}},
		{name: "missing head", probe: platformAuditProbe{events: []*domain.AuditEvent{&first}}, finding: "head record missing"},
		{name: "incomplete history", probe: platformAuditProbe{events: []*domain.AuditEvent{&first}, headSeq: 2, headHash: second.Hash, headFound: true}, finding: "recorded head is 2"},
		{name: "head without events", probe: platformAuditProbe{headSeq: 1, headHash: first.Hash, headFound: true}, finding: "recorded head is 1"},
		{name: "content mismatch", probe: platformAuditProbe{events: []*domain.AuditEvent{&contentMismatch}, headSeq: 1, headHash: contentMismatch.Hash, headFound: true}, finding: "content does not match"},
		{name: "sequence gap", probe: platformAuditProbe{events: []*domain.AuditEvent{&first, &gap}, headSeq: 3, headHash: gap.Hash, headFound: true}, finding: "sequence gap"},
		{name: "broken link", probe: platformAuditProbe{events: []*domain.AuditEvent{&first, &brokenLink}, headSeq: 2, headHash: brokenLink.Hash, headFound: true}, finding: "breaks the hash chain"},
		{name: "event read error", probe: platformAuditProbe{listError: lookupFailure}, wantErr: lookupFailure},
		{name: "head read error", probe: platformAuditProbe{headError: lookupFailure}, wantErr: lookupFailure},
	} {
		t.Run(tc.name, func(t *testing.T) {
			p := &tc.probe
			s := &Service{st: p, cfg: Config{Clock: platformAuditClock{at}}}
			report, err := s.VerifyPlatformAudit(context.Background(), domain.SystemActor("verification"))
			if !errors.Is(err, tc.wantErr) {
				t.Fatalf("verification error: got %v, want %v", err, tc.wantErr)
			}
			if tc.wantErr != nil {
				if report != nil {
					t.Fatal("failed lookup returned a completed verification report")
				}
			} else if report == nil || report.OK != (tc.finding == "") || report.AuditEvents != int64(len(p.events)) ||
				!strings.Contains(strings.Join(report.Findings, "\n"), tc.finding) || !report.Started.Equal(at) || !report.Finished.Equal(at) {
				t.Fatalf("unexpected verification report: %+v", report)
			}
			if p.views != 1 || p.wrongScope {
				t.Fatalf("platform verification must use one correctly scoped read snapshot: views=%d wrongScope=%v", p.views, p.wrongScope)
			}
		})
	}
}

func TestVerifyPlatformAuditRequiresSystem(t *testing.T) {
	p := &platformAuditProbe{}
	s := &Service{st: p}
	for _, actor := range []domain.Actor{{}, {PersonID: "member", Scopes: []domain.Scope{domain.ScopeRead, domain.ScopeWrite}}} {
		if report, err := s.VerifyPlatformAudit(context.Background(), actor); !errors.Is(err, domain.ErrForbidden) || report != nil {
			t.Fatalf("ordinary actor must not inspect the platform audit: report=%+v error=%v", report, err)
		}
	}
	if p.views != 0 {
		t.Fatal("unauthorized actor reached storage")
	}
}
