package service

import (
	"context"
	"strings"
	"time"

	"github.com/djshellshoxxx/circula/internal/domain"
	"github.com/djshellshoxxx/circula/internal/store"
)

const tokenPrefix = "cir_"

// TokenInput requests a new API token.
type TokenInput struct {
	Name      string
	Scopes    []domain.Scope // subset of the caller's own scopes
	NetworkID string         // optional: restrict the token to one network
	ExpiresAt *time.Time
}

// CreateAPIToken returns the bearer value once; only its hash is stored. A
// token can never hold a scope its creator lacks.
func (s *Service) CreateAPIToken(ctx context.Context, actor domain.Actor, in TokenInput) (string, *domain.APIToken, error) {
	if actor.MustChangePassword {
		return "", nil, domain.PasswordChangeRequired() // SEC-008
	}
	if actor.PersonID == "" || !actor.Can(domain.ScopeWrite) {
		return "", nil, domain.Forbidden("write scope required")
	}
	name := strings.TrimSpace(in.Name)
	if name == "" || len(name) > 80 {
		return "", nil, domain.Invalid("name", "must be 1 to 80 characters")
	}
	if len(in.Scopes) == 0 {
		return "", nil, domain.Invalid("scopes", "at least one scope required")
	}
	for _, sc := range in.Scopes {
		if (sc != domain.ScopeRead && sc != domain.ScopeWrite) || !actor.Can(sc) {
			return "", nil, domain.Invalid("scopes", "unknown scope or scope exceeds creator's")
		}
	}
	if actor.NetworkScope != "" && in.NetworkID != actor.NetworkScope {
		return "", nil, domain.Forbidden("token cannot widen its network scope")
	}
	if in.ExpiresAt != nil && !in.ExpiresAt.After(s.now()) {
		return "", nil, domain.Invalid("expires_at", "must be in the future")
	}
	raw, err := randToken(32)
	if err != nil {
		return "", nil, err
	}
	plain := tokenPrefix + raw
	t := &domain.APIToken{ID: s.id(), PersonID: actor.PersonID, NetworkID: in.NetworkID, Name: name,
		TokenHash: hashToken(plain), Scopes: in.Scopes, CreatedAt: s.now(), ExpiresAt: in.ExpiresAt}
	err = s.update(ctx, func(tx store.Tx) error {
		if in.NetworkID != "" {
			if _, err := tx.GetMemberByPerson(in.NetworkID, actor.PersonID); err != nil {
				return domain.NotFound("network")
			}
		}
		if err := tx.InsertToken(t); err != nil {
			return err
		}
		if err := s.secNotice(tx, actor.PersonID, domain.NoticeTokenCreated, name); err != nil {
			return err
		}
		return s.auditPlatform(tx, actor, auditEntry{Action: domain.ActTokenCreated, TargetType: "api_token", TargetID: t.ID})
	})
	if err != nil {
		return "", nil, err
	}
	return plain, t, nil
}

// AuthenticateToken resolves an API token to an Actor carrying the token's
// scopes and optional network restriction.
func (s *Service) AuthenticateToken(ctx context.Context, token string) (domain.Actor, error) {
	if !strings.HasPrefix(token, tokenPrefix) {
		return domain.Actor{}, domain.ErrUnauthenticated
	}
	var t *domain.APIToken
	var must bool
	if err := s.view(ctx, func(tx store.Tx) (err error) {
		if t, err = tx.GetTokenByHash(hashToken(token)); err != nil {
			return err
		}
		must, err = tx.MustChangePassword(t.PersonID)
		return err
	}); err != nil {
		return domain.Actor{}, authnErr(err)
	}
	now := s.now()
	if t.RevokedAt != nil || (t.ExpiresAt != nil && !now.Before(*t.ExpiresAt)) {
		return domain.Actor{}, domain.ErrUnauthenticated
	}
	if t.LastUsedAt == nil || now.Sub(*t.LastUsedAt) > time.Minute {
		_ = s.update(ctx, func(tx store.Tx) error { return tx.TouchToken(t.ID, now) })
	}
	return domain.Actor{PersonID: t.PersonID, Scopes: t.Scopes, NetworkScope: t.NetworkID, MustChangePassword: must}, nil
}

// RevokeAPIToken revokes one of the caller's own tokens.
func (s *Service) RevokeAPIToken(ctx context.Context, actor domain.Actor, tokenID string) error {
	if actor.MustChangePassword {
		return domain.PasswordChangeRequired() // SEC-008
	}
	if actor.PersonID == "" || !actor.Can(domain.ScopeWrite) {
		return domain.Forbidden("write scope required")
	}
	return s.update(ctx, func(tx store.Tx) error {
		ok, err := tx.RevokeToken(actor.PersonID, tokenID, s.now())
		if err != nil {
			return err
		}
		if !ok {
			return domain.NotFound("token")
		}
		if err := s.secNotice(tx, actor.PersonID, domain.NoticeTokenRevoked, ""); err != nil {
			return err
		}
		return s.auditPlatform(tx, actor, auditEntry{Action: domain.ActTokenRevoked, TargetType: "api_token", TargetID: tokenID})
	})
}

// ListAPITokens lists the caller's tokens (never the secret values).
func (s *Service) ListAPITokens(ctx context.Context, actor domain.Actor) ([]*domain.APIToken, error) {
	if actor.PersonID == "" || !actor.Can(domain.ScopeRead) {
		return nil, domain.Forbidden("read scope required")
	}
	var out []*domain.APIToken
	err := s.view(ctx, func(tx store.Tx) (err error) { out, err = tx.ListTokens(actor.PersonID); return })
	return out, err
}
