package service

import (
	"context"
	"errors"

	"github.com/djshellshoxxx/circula/internal/domain"
	"github.com/djshellshoxxx/circula/internal/store"
)

// LoginResult is the outcome of LoginStep. Exactly one of Token and
// MFAChallenge is set. MFAChallenge is a single-use bearer value (valid 5
// minutes) to pass to CompleteMFA together with a second-factor code.
type LoginResult struct {
	Token        string
	Person       *domain.Person
	MFAChallenge string
}

// Login is the password-only sign-in. For a person with a second factor it
// issues NO session and fails with domain.CodeMFARequired; such callers must
// use LoginStep and CompleteMFA. Non-MFA persons are unaffected.
func (s *Service) Login(ctx context.Context, login, password string) (string, *domain.Person, error) {
	r, err := s.LoginStep(ctx, login, password)
	if err != nil {
		return "", nil, err
	}
	if r.MFAChallenge != "" {
		return "", nil, domain.Errf(domain.CodeMFARequired, "second factor required")
	}
	return r.Token, r.Person, nil
}

// mfaChallengeIfEnabled issues a challenge when the person has MFA enabled.
func (s *Service) mfaChallengeIfEnabled(tx store.Tx, personID string) (string, bool, error) {
	on, err := s.hasMFA(tx, personID)
	if err != nil || !on {
		return "", false, err
	}
	raw, err := randToken(32)
	if err != nil {
		return "", false, err
	}
	now := s.now()
	if err := tx.DeleteDeadMFAChallenges(personID, now); err != nil {
		return "", false, err
	}
	err = tx.InsertMFAChallenge(&domain.MFAChallenge{IDHash: hashToken(raw), PersonID: personID,
		CreatedAt: now, ExpiresAt: now.Add(mfaChallengeTTL)})
	return raw, err == nil, err
}

// CompleteMFA finishes a login started by LoginStep: it verifies code (a 6
// digit TOTP code or a recovery code) against the challenge's person and, on
// success, issues the session token. A challenge works once; a wrong code
// leaves it usable (until expiry) but counts toward the per-person throttle
// (5 failures, then 15 minutes and doubling), shared with other second-factor
// operations. A recovery code is consumed, audited and notified. All failures
// except throttling return ErrUnauthenticated.
func (s *Service) CompleteMFA(ctx context.Context, challenge, code string) (string, *domain.Person, error) {
	h := hashToken(challenge)
	var pid string
	err := s.update(ctx, func(tx store.Tx) error {
		c, err := tx.GetMFAChallenge(h)
		if err != nil {
			return authnErr(err)
		}
		if c.UsedAt != nil || !s.now().Before(c.ExpiresAt) {
			return domain.ErrUnauthenticated
		}
		pid = c.PersonID
		return s.mfaReserve(tx, pid)
	})
	if err != nil {
		return "", nil, err
	}
	token, err := randToken(32)
	if err != nil {
		return "", nil, err
	}
	var person *domain.Person
	err = s.update(ctx, func(tx store.Tx) error {
		now := s.now()
		used, err := tx.UseMFAChallenge(h, now)
		if err != nil {
			return err
		}
		if !used {
			return domain.ErrUnauthenticated
		}
		c, err := tx.GetMFAChallenge(h)
		if err != nil {
			return authnErr(err)
		}
		if !now.Before(c.ExpiresAt) {
			return domain.ErrUnauthenticated
		}
		rec, err := tx.GetMFA(pid)
		if errors.Is(err, domain.ErrNotFound) || (err == nil && !rec.Enabled) {
			return domain.ErrUnauthenticated
		}
		if err != nil {
			return err
		}
		kind, ok, err := s.checkFactor(tx, rec, code, true)
		if err != nil {
			return err
		}
		if !ok {
			return domain.ErrUnauthenticated // rolls back the challenge use; the attempt stays counted
		}
		if person, _, err = tx.GetPerson(pid); err != nil {
			return authnErr(err)
		}
		if err := tx.DeleteThrottle(mfaKey(pid)); err != nil {
			return err
		}
		if kind == "recovery" {
			left, err := tx.CountRecoveryCodes(pid)
			if err != nil {
				return err
			}
			if err := s.auditPlatform(tx, domain.Actor{PersonID: pid}, auditEntry{Action: domain.ActMFARecoveryCodeUsed,
				TargetType: "person", TargetID: pid, After: map[string]any{"remaining": left}}); err != nil {
				return err
			}
			if err := s.notify(tx, pid, domain.NoticeRecoveryCodeUsed, ""); err != nil {
				return err
			}
		}
		return tx.InsertSession(&domain.Session{IDHash: hashToken(token), PersonID: pid,
			CreatedAt: now, LastSeenAt: now, ExpiresAt: now.Add(s.cfg.SessionMax)})
	})
	if err != nil {
		return "", nil, err
	}
	return token, person, nil
}
