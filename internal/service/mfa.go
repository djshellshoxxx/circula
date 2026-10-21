package service

import (
	"context"
	"errors"
	"time"

	"github.com/djshellshoxxx/circula/internal/domain"
	"github.com/djshellshoxxx/circula/internal/store"
)

const (
	mfaPendingTTL   = 15 * time.Minute
	mfaChallengeTTL = 5 * time.Minute
)

// MFAStatus is what a person may learn about their own second factor. It never
// includes the secret.
type MFAStatus struct {
	Enabled           bool
	Pending           bool // enrolment begun, not confirmed
	RecoveryRemaining int
}

// MFAEnrolment is returned by BeginMFAEnrolment. Secret is shown to the person
// only while enrolment is pending; after confirmation no method returns it.
type MFAEnrolment struct {
	Secret string // base32
	URI    string // otpauth://
}

// mfaKey is the login_throttle key for second-factor attempts. It starts with a
// control character, which validLogin rejects, so no login string can collide
// with (and lock or reset) it.
func mfaKey(personID string) string { return "\x1fmfa:" + personID }

func errBadCode() error { return domain.Invalid("code", "is not correct") }

func (s *Service) mfaActor(actor domain.Actor) error {
	if actor.System || actor.PersonID == "" || !actor.Can(domain.ScopeWrite) {
		return domain.Forbidden("write scope required")
	}
	return nil
}

// mfaReserve counts one second-factor attempt up front (same scheme as Login):
// refused with rate-limited while locked. It must run in its own committed unit
// so a failed attempt stays counted.
func (s *Service) mfaReserve(tx store.Tx, personID string) error {
	f, until, _, err := tx.GetThrottle(mfaKey(personID))
	if err != nil {
		return err
	}
	now := s.now()
	if now.Before(until) {
		return domain.Errf(domain.CodeRateLimited, "too many attempts; try again later")
	}
	n := f + 1
	return tx.PutThrottle(mfaKey(personID), n, now.Add(lockFor(n)))
}

func (s *Service) reserveMFA(ctx context.Context, personID string) error {
	return s.update(ctx, func(tx store.Tx) error { return s.mfaReserve(tx, personID) })
}

// VerifyAndConsumeRecoveryCode checks code against the person's unused recovery
// codes and, if it matches, marks it used, all inside the caller's unit of work.
// It writes no audit event, notice or throttle: callers (login, disable,
// account recovery) own those. It reports false for malformed, unknown and
// already-used codes alike. The caller must rate-limit attempts.
func (s *Service) VerifyAndConsumeRecoveryCode(tx store.Tx, personID, code string) (bool, error) {
	c := normalizeCode(code)
	if !validRecoveryFormat(c) {
		return false, nil
	}
	return tx.ConsumeRecoveryCode(personID, hashRecoveryCode(personID, c), s.now())
}

// checkFactor verifies a TOTP code (6 digits) or, when allowRecovery, a
// recovery code, inside tx. TOTP steps are advanced atomically so a code is
// accepted once. kind is "totp" or "recovery".
func (s *Service) checkFactor(tx store.Tx, rec *domain.MFARecord, code string, allowRecovery bool) (kind string, ok bool, err error) {
	c := normalizeCode(code)
	if isDigits(c, totpDigits) {
		secret, err := s.openSecret(rec.PersonID, rec.Secret)
		if err != nil {
			return "", false, err
		}
		step, match := matchTOTP(secret, c, s.now(), rec.LastStep)
		if !match {
			return "", false, nil
		}
		adv, err := tx.AdvanceMFAStep(rec.PersonID, step)
		if err != nil || !adv {
			return "", false, err
		}
		rec.LastStep = step
		return "totp", true, nil
	}
	if !allowRecovery {
		return "", false, nil
	}
	ok, err = s.VerifyAndConsumeRecoveryCode(tx, rec.PersonID, c)
	return "recovery", ok, err
}

func (s *Service) hasMFA(tx store.Tx, personID string) (bool, error) {
	rec, err := tx.GetMFA(personID)
	if errors.Is(err, domain.ErrNotFound) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return rec.Enabled, nil
}

// MFAStatus reports the caller's own second-factor state.
func (s *Service) MFAStatus(ctx context.Context, actor domain.Actor) (*MFAStatus, error) {
	if actor.System || actor.PersonID == "" || !actor.Can(domain.ScopeRead) {
		return nil, domain.Forbidden("read scope required")
	}
	st := &MFAStatus{}
	err := s.view(ctx, func(tx store.Tx) error {
		rec, err := tx.GetMFA(actor.PersonID)
		if errors.Is(err, domain.ErrNotFound) {
			return nil
		}
		if err != nil {
			return err
		}
		st.Enabled, st.Pending = rec.Enabled, !rec.Enabled && s.now().Before(rec.CreatedAt.Add(mfaPendingTTL))
		if rec.Enabled {
			st.RecoveryRemaining, err = tx.CountRecoveryCodes(actor.PersonID)
		}
		return err
	})
	if err != nil {
		return nil, err
	}
	return st, nil
}

// checkCurrentPassword proves the caller knows the account password (A3-03): a
// stolen session alone must not be able to bind an attacker's authenticator.
// Guesses share the recgen throttle with GenerateRecoveryCodes and are counted
// before the hash is compared. A wrong password is a validation error on the
// "password" field.
func (s *Service) checkCurrentPassword(ctx context.Context, actor domain.Actor, password string) error {
	var hash string
	err := s.update(ctx, func(tx store.Tx) error {
		if err := s.reserveAttempt(tx, recgenKey(actor.PersonID)); err != nil {
			return err
		}
		_, h, err := tx.GetPerson(actor.PersonID)
		hash = h
		return authnErr(err)
	})
	if err != nil {
		return err
	}
	if !verifyPassword(password, hash) {
		return domain.Invalid("password", "is not correct") // the attempt was counted above
	}
	return nil
}

// BeginMFAEnrolment starts (or, within 15 minutes, resumes) enrolment: it
// returns the secret and otpauth URI. MFA stays off until ConfirmMFAEnrolment.
// Resuming returns the same pending secret so a mistyped confirmation does not
// strand the authenticator app. Conflict when MFA is already enabled. It needs
// the current password (A3-03).
func (s *Service) BeginMFAEnrolment(ctx context.Context, actor domain.Actor, password string) (*MFAEnrolment, error) {
	if err := s.mfaActor(actor); err != nil {
		return nil, err
	}
	if err := s.checkCurrentPassword(ctx, actor, password); err != nil {
		return nil, err
	}
	var out *MFAEnrolment
	err := s.update(ctx, func(tx store.Tx) error {
		person, _, err := tx.GetPerson(actor.PersonID)
		if err != nil {
			return authnErr(err)
		}
		now := s.now()
		rec, err := tx.GetMFA(actor.PersonID)
		switch {
		case err == nil && rec.Enabled:
			return domain.Errf(domain.CodeConflict, "second factor already enabled")
		case err == nil && now.Before(rec.CreatedAt.Add(mfaPendingTTL)):
			secret, oerr := s.openSecret(actor.PersonID, rec.Secret)
			if oerr != nil {
				return oerr
			}
			out = &MFAEnrolment{Secret: b32.EncodeToString(secret), URI: otpauthURI(person.Login, secret)}
			return nil
		case err != nil && !errors.Is(err, domain.ErrNotFound):
			return err
		}
		secret, err := newTOTPSecret()
		if err != nil {
			return err
		}
		sealed, err := s.sealSecret(actor.PersonID, secret)
		if err != nil {
			return err
		}
		if err := tx.PutMFA(&domain.MFARecord{PersonID: actor.PersonID, Secret: sealed, CreatedAt: now}); err != nil {
			return err
		}
		out = &MFAEnrolment{Secret: b32.EncodeToString(secret), URI: otpauthURI(person.Login, secret)}
		return s.auditPlatform(tx, actor, auditEntry{Action: domain.ActMFAEnrolBegun, TargetType: "person", TargetID: actor.PersonID})
	})
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (s *Service) issueRecoveryCodes(tx store.Tx, personID string) ([]string, error) {
	codes, err := newRecoveryCodes()
	if err != nil {
		return nil, err
	}
	hashes := make([]string, len(codes))
	for i, c := range codes {
		hashes[i] = hashRecoveryCode(personID, normalizeCode(c))
	}
	return codes, tx.ReplaceRecoveryCodes(personID, hashes)
}

// ConfirmMFAEnrolment enables the second factor when code is valid for the
// pending secret and returns ten single-use recovery codes. The codes are
// shown once; only salted hashes are kept. Attempts are throttled. It needs the
// current password as well as the code (A3-03), and on success revokes every
// other session of the person, so a session stolen earlier cannot ride along
// with the new second factor.
func (s *Service) ConfirmMFAEnrolment(ctx context.Context, actor domain.Actor, password, code string) ([]string, error) {
	if err := s.mfaActor(actor); err != nil {
		return nil, err
	}
	if err := s.checkCurrentPassword(ctx, actor, password); err != nil {
		return nil, err
	}
	if err := s.reserveMFA(ctx, actor.PersonID); err != nil {
		return nil, err
	}
	var codes []string
	err := s.update(ctx, func(tx store.Tx) error {
		rec, err := tx.GetMFA(actor.PersonID)
		if errors.Is(err, domain.ErrNotFound) {
			return domain.InvalidState("no enrolment in progress")
		}
		if err != nil {
			return err
		}
		now := s.now()
		if rec.Enabled {
			return domain.Errf(domain.CodeConflict, "second factor already enabled")
		}
		if !now.Before(rec.CreatedAt.Add(mfaPendingTTL)) {
			return domain.InvalidState("enrolment expired; start again")
		}
		_, ok, err := s.checkFactor(tx, rec, code, false)
		if err != nil {
			return err
		}
		if !ok {
			return errBadCode()
		}
		rec.Enabled, rec.EnabledAt = true, &now
		if err := tx.PutMFA(rec); err != nil { // rec.LastStep holds the step checkFactor accepted
			return err
		}
		if codes, err = s.issueRecoveryCodes(tx, actor.PersonID); err != nil {
			return err
		}
		if err := tx.DeleteThrottle(mfaKey(actor.PersonID)); err != nil {
			return err
		}
		if err := tx.RevokeOtherSessionsOf(actor.PersonID, actor.SessionID, now); err != nil {
			return err
		}
		if err := s.auditPlatform(tx, actor, auditEntry{Action: domain.ActMFAEnabled, TargetType: "person", TargetID: actor.PersonID}); err != nil {
			return err
		}
		return s.notify(tx, actor.PersonID, domain.NoticeMFAEnabled, "")
	})
	if err != nil {
		return nil, err
	}
	return codes, nil
}

// DisableMFA removes the second factor. It needs the current password and a
// valid TOTP or recovery code. Existing sessions stay valid. Attempts are
// throttled together with other second-factor attempts.
func (s *Service) DisableMFA(ctx context.Context, actor domain.Actor, password, code string) error {
	if err := s.mfaActor(actor); err != nil {
		return err
	}
	if err := s.reserveMFA(ctx, actor.PersonID); err != nil {
		return err
	}
	var hash string
	if err := s.view(ctx, func(tx store.Tx) (err error) { _, hash, err = tx.GetPerson(actor.PersonID); return }); err != nil {
		return authnErr(err)
	}
	if !verifyPassword(password, hash) {
		return domain.Invalid("password", "is not correct")
	}
	return s.update(ctx, func(tx store.Tx) error {
		rec, err := tx.GetMFA(actor.PersonID)
		if errors.Is(err, domain.ErrNotFound) || (err == nil && !rec.Enabled) {
			return domain.InvalidState("second factor is not enabled")
		}
		if err != nil {
			return err
		}
		_, ok, err := s.checkFactor(tx, rec, code, true)
		if err != nil {
			return err
		}
		if !ok {
			return errBadCode()
		}
		if err := tx.DeleteMFA(actor.PersonID); err != nil {
			return err
		}
		if err := tx.ReplaceRecoveryCodes(actor.PersonID, nil); err != nil {
			return err
		}
		if err := tx.DeleteThrottle(mfaKey(actor.PersonID)); err != nil {
			return err
		}
		if err := s.auditPlatform(tx, actor, auditEntry{Action: domain.ActMFADisabled, TargetType: "person", TargetID: actor.PersonID}); err != nil {
			return err
		}
		return s.notify(tx, actor.PersonID, domain.NoticeMFADisabled, "")
	})
}

// RegenerateRecoveryCodes replaces all recovery codes with ten new ones. It
// needs a current TOTP code (a recovery code is not accepted here).
func (s *Service) RegenerateRecoveryCodes(ctx context.Context, actor domain.Actor, code string) ([]string, error) {
	if err := s.mfaActor(actor); err != nil {
		return nil, err
	}
	if err := s.reserveMFA(ctx, actor.PersonID); err != nil {
		return nil, err
	}
	var codes []string
	err := s.update(ctx, func(tx store.Tx) error {
		rec, err := tx.GetMFA(actor.PersonID)
		if errors.Is(err, domain.ErrNotFound) || (err == nil && !rec.Enabled) {
			return domain.InvalidState("second factor is not enabled")
		}
		if err != nil {
			return err
		}
		_, ok, err := s.checkFactor(tx, rec, code, false)
		if err != nil {
			return err
		}
		if !ok {
			return errBadCode()
		}
		if codes, err = s.issueRecoveryCodes(tx, actor.PersonID); err != nil {
			return err
		}
		if err := tx.DeleteThrottle(mfaKey(actor.PersonID)); err != nil {
			return err
		}
		if err := s.auditPlatform(tx, actor, auditEntry{Action: domain.ActMFARecoveryRegen, TargetType: "person", TargetID: actor.PersonID}); err != nil {
			return err
		}
		return s.notify(tx, actor.PersonID, domain.NoticeRecoveryRegen, "")
	})
	if err != nil {
		return nil, err
	}
	return codes, nil
}

// OperatorResetMFA removes a person's second factor from the CLI
// (`circula reset-mfa`) for someone who lost both authenticator and recovery
// codes. actor must be domain.SystemActor. It deletes the MFA record, the
// recovery codes and every pending second-factor challenge, clears the
// second-factor throttles and revokes every session (the password is not
// touched; reset that separately if it is in doubt). The person gets a
// security notice and the audit event carries the mandatory reason. An unknown
// login is NotFound and a person without a second factor InvalidState (the CLI
// is trusted).
func (s *Service) OperatorResetMFA(ctx context.Context, actor domain.Actor, login, reason string) error {
	if !actor.System {
		return domain.ErrForbidden
	}
	reason, err := checkResetReason(reason)
	if err != nil {
		return err
	}
	login = normalizeLogin(login)
	return s.update(ctx, func(tx store.Tx) error {
		person, _, err := tx.GetPersonByLogin(login)
		if err != nil {
			return domain.NotFound("person")
		}
		if _, err := tx.GetMFA(person.ID); err != nil {
			if errors.Is(err, domain.ErrNotFound) {
				return domain.InvalidState("second factor is not set up")
			}
			return err
		}
		if err := tx.DeleteMFA(person.ID); err != nil {
			return err
		}
		if err := tx.ReplaceRecoveryCodes(person.ID, nil); err != nil {
			return err
		}
		if err := tx.DeleteMFAChallengesOf(person.ID); err != nil {
			return err
		}
		if _, err := tx.DeleteThrottleFamily(mfaKey(person.ID)); err != nil {
			return err
		}
		if err := tx.RevokeSessionsOf(person.ID, s.now()); err != nil {
			return err
		}
		if err := s.notify(tx, person.ID, domain.NoticeMFADisabled, "removed by the platform operator"); err != nil {
			return err
		}
		return s.auditPlatform(tx, actor, auditEntry{Type: domain.ActorOperator, Action: domain.ActMFAReset,
			TargetType: "person", TargetID: person.ID, Reason: reason})
	})
}
