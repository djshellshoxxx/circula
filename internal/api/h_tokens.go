package api

import (
	"time"

	"github.com/djshellshoxxx/circula/internal/domain"
	"github.com/djshellshoxxx/circula/internal/service"
)

type tokenDTO struct {
	ID         string   `json:"id"`
	Name       string   `json:"name"`
	Scopes     []string `json:"scopes"`
	NetworkID  *string  `json:"network_id"`
	CreatedAt  string   `json:"created_at"`
	LastUsedAt *string  `json:"last_used_at"`
	ExpiresAt  *string  `json:"expires_at"`
	RevokedAt  *string  `json:"revoked_at"`
}

func toToken(t *domain.APIToken) tokenDTO {
	sc := make([]string, 0, len(t.Scopes))
	for _, s := range t.Scopes {
		sc = append(sc, string(s))
	}
	return tokenDTO{ID: t.ID, Name: t.Name, Scopes: sc, NetworkID: strPtr(t.NetworkID), CreatedAt: ts(t.CreatedAt),
		LastUsedAt: tsPtr(t.LastUsedAt), ExpiresAt: tsPtr(t.ExpiresAt), RevokedAt: tsPtr(t.RevokedAt)}
}

func (s *server) listTokens(c *reqCtx) error { // API-066
	rows, err := s.svc.ListAPITokens(c.r.Context(), c.actor)
	if err != nil {
		return err
	}
	if len(rows) > 100 {
		rows = rows[:100] // API-036: unpaginated lists return at most 100 rows
	}
	out := make([]tokenDTO, 0, len(rows))
	for _, t := range rows {
		out = append(out, toToken(t))
	}
	return listOK(c, out, nil)
}

// createToken requires session authentication (route flag cookieOnly,
// API-011). The value is returned once.
func (s *server) createToken(c *reqCtx) error { // API-066
	var in struct {
		Name      string   `json:"name"`
		Scopes    []string `json:"scopes"`
		NetworkID string   `json:"network_id"`
		ExpiresAt *string  `json:"expires_at"`
	}
	if err := c.decode(&in); err != nil {
		return err
	}
	var scopes []domain.Scope
	for _, sc := range in.Scopes {
		switch domain.Scope(sc) {
		case domain.ScopeRead, domain.ScopeWrite:
			scopes = append(scopes, domain.Scope(sc))
		case "admin":
			return notImplemented("the admin token scope (admin commands are governed by the caller's role)")
		default:
			return validation("scopes", "invalid-enum", "scopes must be read or write")
		}
	}
	now := s.opt.Now()
	exp := now.Add(365 * 24 * time.Hour)
	if in.ExpiresAt != nil {
		t, err := time.Parse(time.RFC3339, *in.ExpiresAt)
		if err != nil {
			return validation("expires_at", "invalid-timestamp", "must be an RFC 3339 timestamp")
		}
		if t.After(exp) {
			return validation("expires_at", "out-of-range", "must be within 365 days")
		}
		exp = t
	}
	raw, tok, err := s.svc.CreateAPIToken(c.r.Context(), c.actor, service.TokenInput{Name: in.Name, Scopes: scopes,
		NetworkID: in.NetworkID, ExpiresAt: &exp})
	if err != nil {
		return err
	}
	c.w.Header().Set("Location", v1+"/tokens/"+tok.ID)
	return ok(c, 201, struct {
		tokenDTO
		Token string `json:"token"`
	}{toToken(tok), raw})
}

func (s *server) revokeToken(c *reqCtx) error { // API-066
	if err := s.svc.RevokeAPIToken(c.r.Context(), c.actor, c.r.PathValue("id")); err != nil {
		return err
	}
	c.w.WriteHeader(204)
	return nil
}
