package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/djshellshoxxx/circula/internal/domain"
	store "github.com/djshellshoxxx/circula/internal/repository"
)

// This driver-neutral probe proves the actual production methods authorize
// inside their mutation callback, before either token write is reached.
// SQLite integration tests separately exercise successful issuance and reset.
type tokenAuthorityProbe struct {
	store.Store
	store.Tx
	session                  *domain.Session
	mustChange               bool
	lookupErr, writeBoundary error
	inserts, revokes         int
	inWrite                  bool
}

func (p *tokenAuthorityProbe) Update(_ context.Context, fn func(store.Tx) error) error {
	p.inWrite = true
	defer func() { p.inWrite = false }()
	return fn(p)
}

func (p *tokenAuthorityProbe) GetSession(id string) (*domain.Session, error) {
	if !p.inWrite {
		return nil, errors.New("session authority checked outside mutation")
	}
	if p.lookupErr != nil {
		return nil, p.lookupErr
	}
	if p.session == nil || p.session.IDHash != id {
		return nil, domain.NotFound("session")
	}
	return p.session, nil
}

func (p *tokenAuthorityProbe) MustChangePassword(string) (bool, error) {
	if !p.inWrite {
		return false, errors.New("password state checked outside mutation")
	}
	return p.mustChange, nil
}

func (p *tokenAuthorityProbe) InsertToken(*domain.APIToken) error {
	p.inserts++
	return p.writeBoundary
}

func (p *tokenAuthorityProbe) RevokeToken(string, string, time.Time) (bool, error) {
	p.revokes++
	return false, p.writeBoundary
}

type tokenAuthorityClock struct{ now time.Time }

func (c tokenAuthorityClock) Now() time.Time { return c.now }

type tokenAuthorityIDs struct{}

func (tokenAuthorityIDs) New() string { return "018f0000-0000-7000-8000-000000000001" }

func TestTokenMutationsRecheckAuthorityInWriteUnit(t *testing.T) {
	now := time.Date(2026, 11, 19, 12, 0, 0, 0, time.UTC)
	for _, command := range []string{"create", "revoke"} {
		for _, state := range []string{"active", "trusted-direct", "revoked", "missing", "expired", "foreign-owner", "forced-change", "direct-forced-change", "store-error"} {
			t.Run(command+"/"+state, func(t *testing.T) {
				p := &tokenAuthorityProbe{
					session:       &domain.Session{IDHash: "session-hash", PersonID: "owner", CreatedAt: now, LastSeenAt: now, ExpiresAt: now.Add(time.Hour)},
					writeBoundary: errors.New("token write boundary reached"),
				}
				a := domain.Actor{PersonID: "owner", SessionID: "session-hash", Scopes: []domain.Scope{domain.ScopeRead, domain.ScopeWrite}}
				want := domain.CodeUnauthenticated
				switch state {
				case "trusted-direct":
					a.SessionID = ""
				case "revoked":
					p.session.RevokedAt = &now
				case "missing":
					p.session = nil
				case "expired":
					p.session.ExpiresAt = now
				case "foreign-owner":
					p.session.PersonID = "other"
				case "direct-forced-change":
					a.SessionID = ""
					fallthrough
				case "forced-change":
					p.mustChange = true
					want = domain.CodePasswordChangeRequired
				case "store-error":
					p.lookupErr = errors.New("session storage unavailable")
				}
				s := &Service{st: p, cfg: Config{Clock: tokenAuthorityClock{now}, IDs: tokenAuthorityIDs{},
					SessionIdle: time.Hour, SessionMax: 24 * time.Hour,
					AdminSessionIdle: time.Hour, AdminSessionMax: 24 * time.Hour,
					ReviewerSessionIdle: time.Hour, ReviewerSessionMax: 24 * time.Hour}}
				var err error
				if command == "create" {
					_, _, err = s.CreateAPIToken(context.Background(), a, TokenInput{Name: "integration", Scopes: []domain.Scope{domain.ScopeRead}})
				} else {
					err = s.RevokeAPIToken(context.Background(), a, tokenAuthorityIDs{}.New())
				}
				if state == "active" || state == "trusted-direct" {
					if !errors.Is(err, p.writeBoundary) || p.inserts+p.revokes != 1 {
						t.Fatalf("authorized caller did not reach one token write: err=%v insert=%d revoke=%d", err, p.inserts, p.revokes)
					}
					return
				}
				if state == "store-error" {
					if !errors.Is(err, p.lookupErr) {
						t.Errorf("storage error hidden: %v", err)
					}
				} else if domain.CodeOf(err) != want {
					t.Errorf("stale authority result=%v, want %s", err, want)
				}
				if p.inserts+p.revokes != 0 {
					t.Fatalf("stale authority reached token write: insert=%d revoke=%d", p.inserts, p.revokes)
				}
			})
		}
	}
}
