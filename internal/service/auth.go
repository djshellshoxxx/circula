package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
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
	// RevealDuplicate makes a taken login fail with CodeConflict. Only trusted
	// operator tooling (the CLI) may set it; network-facing adapters must leave
	// it false so sign-up cannot be used to probe which logins exist (SA-10).
	RevealDuplicate bool
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
//
// Uniform response (SA-10, SEC-005): when the login is already taken the call
// does the same hashing work, leaves the existing account untouched, records a
// platform audit event and returns success with a decoy Person whose ID matches
// no stored row (nobody can sign in with it). The caller therefore cannot tell
// a new login from an existing one; adapters must not auto-login or otherwise
// branch on the outcome, and should tell the user to sign in with the details
// they chose. RegisterInput.RevealDuplicate restores the conflict for the CLI.
func (s *Service) RegisterPerson(ctx context.Context, in RegisterInput) (*domain.Person, error) {
	login := normalizeLogin(in.Login)
	if err := validLogin(login); err != nil {
		return nil, err
	}
	if err := s.validatePassword(in.Password, login); err != nil {
		return nil, err
	}
	hash, err := hashPassword(in.Password, s.cfg.Argon)
	if err != nil {
		return nil, err
	}
	p := &domain.Person{ID: s.id(), Login: login, CreatedAt: s.now()}
	err = s.update(ctx, func(tx store.Tx) error {
		if err := tx.InsertPerson(p, hash); err != nil {
			if domain.CodeOf(err) != domain.CodeConflict {
				return err
			}
			if in.RevealDuplicate {
				return domain.Errf(domain.CodeConflict, "login unavailable")
			}
			existing, _, gerr := tx.GetPersonByLogin(login)
			if gerr != nil {
				return gerr
			}
			return s.auditPlatform(tx, domain.Actor{PersonID: existing.ID}, auditEntry{
				Action: domain.ActRegisterDuplicate, TargetType: "person", TargetID: existing.ID})
		}
		return s.auditPlatform(tx, domain.Actor{PersonID: p.ID}, auditEntry{
			Action: domain.ActPersonRegistered, TargetType: "person", TargetID: p.ID})
	})
	if err != nil {
		return nil, err
	}
	return p, nil
}

// LoginStep verifies credentials and returns an opaque bearer session token (shown
// once; only its SHA-256 is stored). Unknown logins and wrong passwords return
// the identical ErrUnauthenticated and do equal hashing work (SEC-005). After
// five consecutive failures further attempts for that login (from that client
// address, see WithClientIP and throttle.go, SA2-10) are refused with
// ErrRateLimited for 15 minutes (SEC-006); success resets the counters.
//
// Atomicity (AUTH-01, SA-04): the lock check and the attempt reservation happen
// in ONE write unit before the password is verified. Every evaluated attempt
// is counted up front, so N parallel guesses are counted as N and, once the
// threshold is reached, later ones are refused without being evaluated; at most
// throttleAfter guesses can be evaluated per lock window regardless of
// concurrency. Success deletes the counter. Malformed logins (validLogin) are
// refused uniformly without touching the throttle table, so its keys are
// bounded by valid login strings (and purged by Purge).
//
// A person with a second factor gets LoginResult.MFAChallenge instead of a
// token (SEC-007); CompleteMFA then issues the session. Login (mfa_login.go)
// is the password-only wrapper that refuses MFA persons.
func (s *Service) LoginStep(ctx context.Context, login, password string) (*LoginResult, error) {
	login = normalizeLogin(login)
	if validLogin(login) != nil {
		verifyPassword(password, s.dummyHash) // equalise effort; nothing is stored
		return nil, domain.ErrUnauthenticated
	}
	var person *domain.Person
	var hash string
	err := s.update(ctx, func(tx store.Tx) error {
		if err := s.reserveAttemptCtx(ctx, tx, login); err != nil {
			return err
		}
		p, h, err := tx.GetPersonByLogin(login)
		switch {
		case err == nil:
			person, hash = p, h
		case !errors.Is(err, domain.ErrNotFound):
			return err
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	ok := false
	if person != nil {
		ok = verifyPassword(password, hash)
	} else {
		verifyPassword(password, s.dummyHash) // equalise effort
	}
	if !ok {
		return nil, domain.ErrUnauthenticated // the attempt was counted above
	}
	token, err := randToken(32)
	if err != nil {
		return nil, err
	}
	now := s.now()
	var challenge string
	err = s.update(ctx, func(tx store.Tx) error {
		if err := s.clearThrottle(ctx, tx, login); err != nil {
			return err
		}
		if c, ok, err := s.mfaChallengeIfEnabled(tx, person.ID); err != nil || ok {
			challenge = c
			return err
		}
		return tx.InsertSession(&domain.Session{IDHash: hashToken(token), PersonID: person.ID,
			CreatedAt: now, LastSeenAt: now, ExpiresAt: now.Add(s.cfg.SessionMax)})
	})
	if err != nil {
		return nil, err
	}
	if challenge != "" {
		return &LoginResult{Person: person, MFAChallenge: challenge}, nil
	}
	return &LoginResult{Token: token, Person: person}, nil
}

func hashToken(t string) string {
	sum := sha256.Sum256([]byte(t))
	return hex.EncodeToString(sum[:])
}

// authnErr maps a lookup failure for a credential: only "no such credential"
// is unauthenticated. Store failures (busy database, I/O) are returned as they
// are, so an outage never looks like a logout (AUTH-02).
func authnErr(err error) error {
	if errors.Is(err, domain.ErrNotFound) {
		return domain.ErrUnauthenticated
	}
	return err
}

// Authenticate resolves a session token to an Actor. Revoked, idle-expired or
// absolutely expired sessions are rejected (SEC-003). last_seen is refreshed
// at most once a minute.
func (s *Service) Authenticate(ctx context.Context, token string) (domain.Actor, error) {
	id := hashToken(token)
	var sess *domain.Session
	var must bool
	if err := s.view(ctx, func(tx store.Tx) (err error) {
		if sess, err = tx.GetSession(id); err != nil {
			return err
		}
		must, err = tx.MustChangePassword(sess.PersonID)
		return err
	}); err != nil {
		return domain.Actor{}, authnErr(err)
	}
	now := s.now()
	if sess.RevokedAt != nil || !now.Before(sess.ExpiresAt) || !now.Before(sess.LastSeenAt.Add(s.cfg.SessionIdle)) {
		return domain.Actor{}, domain.ErrUnauthenticated
	}
	if now.Sub(sess.LastSeenAt) > time.Minute {
		_ = s.update(ctx, func(tx store.Tx) error { return tx.TouchSession(id, now) })
	}
	return domain.Actor{PersonID: sess.PersonID, Scopes: []domain.Scope{domain.ScopeRead, domain.ScopeWrite}, MustChangePassword: must, SessionID: id}, nil
}

// Logout revokes the session behind token. Unknown tokens are ignored.
func (s *Service) Logout(ctx context.Context, token string) error {
	return s.update(ctx, func(tx store.Tx) error { return tx.RevokeSession(hashToken(token), s.now()) })
}

// ChangePassword verifies the old password, sets the new one and revokes every
// session of the person, including the caller's (SEC-003). API tokens stay valid.
func (s *Service) ChangePassword(ctx context.Context, actor domain.Actor, oldPW, newPW string) error {
	// A pending forced change must not block the change itself (SEC-008), but
	// the scope rule still applies, so judge the scope without the flag.
	scoped := actor
	scoped.MustChangePassword = false
	if !scoped.Can(domain.ScopeWrite) || actor.PersonID == "" {
		return domain.ErrForbidden
	}
	var hash, login string
	if err := s.view(ctx, func(tx store.Tx) error {
		p, h, err := tx.GetPerson(actor.PersonID)
		if err == nil {
			hash, login = h, p.Login
		}
		return err
	}); err != nil {
		return authnErr(err)
	}
	if err := s.validatePassword(newPW, login); err != nil {
		return err
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
		if err := s.notify(tx, actor.PersonID, domain.NoticePasswordChanged, ""); err != nil {
			return err
		}
		if err := tx.SetMustChangePassword(actor.PersonID, false); err != nil {
			return err
		}
		return s.auditPlatform(tx, actor, auditEntry{Action: domain.ActPasswordChanged,
			TargetType: "person", TargetID: actor.PersonID})
	})
}
