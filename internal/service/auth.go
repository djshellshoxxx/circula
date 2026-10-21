package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"strings"
	"time"
	"unicode"

	"github.com/djshellshoxxx/circula/internal/domain"
	"github.com/djshellshoxxx/circula/internal/store"
)

// RegisterInput creates a global identity. Adapters decide who may call
// RegisterPerson (setup secret, invitation, open sign-up): the service
// enforces only data rules.
type RegisterInput struct {
	Login    string
	Password string
}

func normalizeLogin(l string) string { return strings.ToLower(strings.TrimSpace(l)) }

func validLogin(l string) error {
	if len(l) < 3 || len(l) > 254 {
		return domain.Invalid("login", "must be 3 to 254 characters")
	}
	for _, r := range l {
		if unicode.IsSpace(r) || unicode.IsControl(r) {
			return domain.Invalid("login", "must not contain spaces or control characters")
		}
	}
	return nil
}

// RegisterPerson creates a Person with an Argon2id password hash.
func (s *Service) RegisterPerson(ctx context.Context, in RegisterInput) (*domain.Person, error) {
	login := normalizeLogin(in.Login)
	if err := validLogin(login); err != nil {
		return nil, err
	}
	if err := s.validatePassword(in.Password); err != nil {
		return nil, err
	}
	hash, err := hashPassword(in.Password, s.cfg.Argon)
	if err != nil {
		return nil, err
	}
	p := &domain.Person{ID: s.id(), Login: login, CreatedAt: s.now()}
	err = s.update(ctx, func(tx store.Tx) error {
		if err := tx.InsertPerson(p, hash); err != nil {
			if domain.CodeOf(err) == domain.CodeConflict {
				return domain.Errf(domain.CodeConflict, "login unavailable")
			}
			return err
		}
		return s.auditPlatform(tx, domain.Actor{PersonID: p.ID}, auditEntry{
			Action: domain.ActPersonRegistered, TargetType: "person", TargetID: p.ID})
	})
	if err != nil {
		return nil, err
	}
	return p, nil
}

const (
	throttleAfter = 5
	throttleBase  = 15 * time.Minute
	throttleMax   = 24 * time.Hour
)

func lockFor(failures int) time.Duration {
	if failures < throttleAfter {
		return 0
	}
	d := throttleBase
	for i := throttleAfter; i < failures && d < throttleMax; i++ {
		d *= 2
	}
	if d > throttleMax {
		d = throttleMax
	}
	return d
}

// Login verifies credentials and returns an opaque bearer session token (shown
// once; only its SHA-256 is stored). Unknown logins and wrong passwords return
// the identical ErrUnauthenticated and do equal hashing work (SEC-005). After
// five consecutive failures further attempts for that login are refused with
// ErrRateLimited for an increasing delay starting at 15 minutes (SEC-006);
// success resets the counter. Per-address limits belong to the adapter.
func (s *Service) Login(ctx context.Context, login, password string) (string, *domain.Person, error) {
	login = normalizeLogin(login)
	var person *domain.Person
	var hash string
	var failures int
	err := s.view(ctx, func(tx store.Tx) error {
		f, until, found, err := tx.GetThrottle(login)
		if err != nil {
			return err
		}
		if found {
			failures = f
			if s.now().Before(until) {
				return domain.Errf(domain.CodeRateLimited, "too many attempts; try again later")
			}
		}
		if p, h, err := tx.GetPersonByLogin(login); err == nil {
			person, hash = p, h
		}
		return nil
	})
	if err != nil {
		return "", nil, err
	}
	ok := false
	if person != nil {
		ok = verifyPassword(password, hash)
	} else {
		verifyPassword(password, s.dummyHash) // equalise effort
	}
	if !ok {
		_ = s.update(ctx, func(tx store.Tx) error {
			n := failures + 1
			return tx.PutThrottle(login, n, s.now().Add(lockFor(n)))
		})
		return "", nil, domain.ErrUnauthenticated
	}
	token, err := randToken(32)
	if err != nil {
		return "", nil, err
	}
	now := s.now()
	err = s.update(ctx, func(tx store.Tx) error {
		if err := tx.DeleteThrottle(login); err != nil {
			return err
		}
		return tx.InsertSession(&domain.Session{IDHash: hashToken(token), PersonID: person.ID,
			CreatedAt: now, LastSeenAt: now, ExpiresAt: now.Add(s.cfg.SessionMax)})
	})
	if err != nil {
		return "", nil, err
	}
	return token, person, nil
}

func hashToken(t string) string {
	sum := sha256.Sum256([]byte(t))
	return hex.EncodeToString(sum[:])
}

// Authenticate resolves a session token to an Actor. Revoked, idle-expired or
// absolutely expired sessions are rejected (SEC-003). last_seen is refreshed
// at most once a minute.
func (s *Service) Authenticate(ctx context.Context, token string) (domain.Actor, error) {
	id := hashToken(token)
	var sess *domain.Session
	if err := s.view(ctx, func(tx store.Tx) (err error) {
		sess, err = tx.GetSession(id)
		return err
	}); err != nil {
		return domain.Actor{}, domain.ErrUnauthenticated
	}
	now := s.now()
	if sess.RevokedAt != nil || !now.Before(sess.ExpiresAt) || !now.Before(sess.LastSeenAt.Add(s.cfg.SessionIdle)) {
		return domain.Actor{}, domain.ErrUnauthenticated
	}
	if now.Sub(sess.LastSeenAt) > time.Minute {
		_ = s.update(ctx, func(tx store.Tx) error { return tx.TouchSession(id, now) })
	}
	return domain.Actor{PersonID: sess.PersonID, Scopes: []domain.Scope{domain.ScopeRead, domain.ScopeWrite}}, nil
}

// Logout revokes the session behind token. Unknown tokens are ignored.
func (s *Service) Logout(ctx context.Context, token string) error {
	return s.update(ctx, func(tx store.Tx) error { return tx.RevokeSession(hashToken(token), s.now()) })
}

// ChangePassword verifies the old password, sets the new one and revokes every
// session of the person, including the caller's (SEC-003). API tokens stay valid.
func (s *Service) ChangePassword(ctx context.Context, actor domain.Actor, oldPW, newPW string) error {
	if !actor.Can(domain.ScopeWrite) || actor.PersonID == "" {
		return domain.ErrForbidden
	}
	if err := s.validatePassword(newPW); err != nil {
		return err
	}
	var hash string
	if err := s.view(ctx, func(tx store.Tx) (err error) { _, hash, err = tx.GetPerson(actor.PersonID); return }); err != nil {
		return domain.ErrUnauthenticated
	}
	if !verifyPassword(oldPW, hash) {
		return domain.ErrUnauthenticated
	}
	nh, err := hashPassword(newPW, s.cfg.Argon)
	if err != nil {
		return err
	}
	return s.update(ctx, func(tx store.Tx) error {
		if err := tx.SetPasswordHash(actor.PersonID, nh); err != nil {
			return err
		}
		if err := tx.RevokeSessionsOf(actor.PersonID, s.now()); err != nil {
			return err
		}
		return s.auditPlatform(tx, actor, auditEntry{Action: domain.ActPasswordChanged,
			TargetType: "person", TargetID: actor.PersonID})
	})
}
