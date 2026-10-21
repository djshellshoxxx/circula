package service

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base32"
	"encoding/hex"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/djshellshoxxx/circula/internal/domain"
	"github.com/djshellshoxxx/circula/internal/store"
)

// Account recovery without email (SEC-008). See docs/dev/recovery.md.

const (
	// RecoveryCodeCount is the size of a batch.
	RecoveryCodeCount = 10
	recoveryCodeBytes = 10 // 80 bits, 16 base32 characters
	recoveryCodeChars = 16
	// adminResetsPerHour caps successful AdminResetPassword calls per admin.
	adminResetsPerHour = 10
	maxResetReasonLen  = 500
)

var recoveryEnc = base32.StdEncoding.WithPadding(base32.NoPadding)

// errRecovery is the single refusal of RecoverPassword for an unknown login,
// a malformed code, a wrong code and a used code (SEC-005 style uniformity).
func errRecovery() error {
	return domain.Errf(domain.CodeUnauthenticated, "recovery details are not valid")
}

// recoverKey names the login_throttle row of self-service recovery. The space
// cannot occur in a login (validLogin), so it never collides with a login's
// own throttle row.
func recoverKey(login string) string { return "recover: " + login }

// recgenKey throttles password guesses made while generating codes.
func recgenKey(personID string) string { return "recgen: " + personID }

// reserveAttempt is the Login pattern: refuse while locked, otherwise count
// the attempt up front so parallel guesses cannot beat the limit.
func (s *Service) reserveAttempt(tx store.Tx, key string) error {
	f, until, _, err := tx.GetThrottle(key)
	if err != nil {
		return err
	}
	now := s.now()
	if now.Before(until) {
		return domain.Errf(domain.CodeRateLimited, "too many attempts; try again later")
	}
	n := f + 1
	return tx.PutThrottle(key, n, now.Add(lockFor(n)))
}

// normalizeRecoveryCode upper-cases and strips separators. ok is false when
// the result cannot be a code.
func normalizeRecoveryCode(c string) (string, bool) {
	var b strings.Builder
	for _, r := range strings.ToUpper(c) {
		switch {
		case r == '-' || r == ' ' || r == '\t':
		case (r >= 'A' && r <= 'Z') || (r >= '2' && r <= '7'):
			b.WriteRune(r)
		default:
			return "", false
		}
	}
	return b.String(), b.Len() == recoveryCodeChars
}

func formatRecoveryCode(norm string) string {
	return norm[0:4] + "-" + norm[4:8] + "-" + norm[8:12] + "-" + norm[12:16]
}

// recoveryHash salts with the person id so equal codes of two people differ.
func recoveryHash(personID, norm string) string {
	sum := sha256.Sum256([]byte("circula/recovery/v1\x00" + personID + "\x00" + norm))
	return hex.EncodeToString(sum[:])
}

// GenerateRecoveryCodes replaces the person's recovery codes with a fresh
// batch of RecoveryCodeCount and returns them once; only salted SHA-256
// hashes are stored. It needs the current password and a full-scope session
// or token (not a network-restricted one). Every previous code stops working.
// Password guesses are throttled like login guesses.
func (s *Service) GenerateRecoveryCodes(ctx context.Context, actor domain.Actor, password string) ([]string, error) {
	if actor.MustChangePassword {
		return nil, domain.PasswordChangeRequired()
	}
	if actor.PersonID == "" || !actor.Can(domain.ScopeWrite) || actor.NetworkScope != "" {
		return nil, domain.ErrForbidden
	}
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
		return nil, err
	}
	if !verifyPassword(password, hash) {
		return nil, domain.ErrUnauthenticated // the attempt was counted above
	}
	codes := make([]string, RecoveryCodeCount)
	hashes := make([]string, RecoveryCodeCount)
	ids := make([]string, RecoveryCodeCount)
	for i := range codes {
		raw := make([]byte, recoveryCodeBytes)
		if _, err := rand.Read(raw); err != nil {
			return nil, err
		}
		norm := recoveryEnc.EncodeToString(raw)
		codes[i], hashes[i], ids[i] = formatRecoveryCode(norm), recoveryHash(actor.PersonID, norm), s.id()
	}
	err = s.update(ctx, func(tx store.Tx) error {
		if err := tx.ReplaceAccountRecoveryCodes(actor.PersonID, ids, hashes, s.now()); err != nil {
			return err
		}
		if err := tx.DeleteThrottle(recgenKey(actor.PersonID)); err != nil {
			return err
		}
		return s.auditPlatform(tx, actor, auditEntry{Action: domain.ActRecoveryCodesGenerated,
			TargetType: "person", TargetID: actor.PersonID, After: map[string]any{"count": RecoveryCodeCount}})
	})
	if err != nil {
		return nil, err
	}
	return codes, nil
}

// RecoveryCodesRemaining reports how many unused recovery codes the caller has.
func (s *Service) RecoveryCodesRemaining(ctx context.Context, actor domain.Actor) (int, error) {
	if actor.PersonID == "" || !actor.Can(domain.ScopeRead) {
		return 0, domain.ErrForbidden
	}
	var n int
	err := s.view(ctx, func(tx store.Tx) (err error) { n, err = tx.CountAccountRecoveryCodes(actor.PersonID); return })
	return n, err
}

// RecoverPassword resets a forgotten password with a recovery code. Nothing is
// signed in: the person must log in afterwards.
//
//   - Uniform: unknown login, malformed code, wrong code and used code all
//     return the same ErrUnauthenticated, after the same hashing work.
//   - Throttled: attempts are counted per login ("recover: "+login) before the
//     code is evaluated, with the escalating lock of Login; a locked login is
//     refused with ErrRateLimited whether or not it exists.
//   - Atomic: consuming the code, the new password, revocation of every session
//     and API token, removal of the remaining codes and the audit event are one
//     unit; of N parallel uses of one code exactly one succeeds.
//   - The remaining codes are deleted because a code in the wrong hands means
//     the sheet may be compromised; generate a new batch after signing in.
func (s *Service) RecoverPassword(ctx context.Context, login, code, newPassword string) error {
	if err := s.validatePassword(newPassword); err != nil {
		return err // says nothing about the login or the code
	}
	nh, err := hashPassword(newPassword, s.cfg.Argon) // equal work on every path
	if err != nil {
		return err
	}
	login = normalizeLogin(login)
	if validLogin(login) != nil {
		return errRecovery()
	}
	var person *domain.Person
	err = s.update(ctx, func(tx store.Tx) error {
		if err := s.reserveAttempt(tx, recoverKey(login)); err != nil {
			return err
		}
		p, _, err := tx.GetPersonByLogin(login)
		switch {
		case err == nil:
			person = p
		case domain.CodeOf(err) != domain.CodeNotFound:
			return err
		}
		return nil
	})
	if err != nil {
		return err
	}
	norm, wellFormed := normalizeRecoveryCode(code)
	pid := "-" // matches no row: the unknown-login path runs the same statements
	if person != nil {
		pid = person.ID
	}
	if !wellFormed {
		norm = strings.Repeat("A", recoveryCodeChars)
	}
	return s.update(ctx, func(tx store.Tx) error {
		ok, err := tx.ConsumeAccountRecoveryCode(pid, recoveryHash(pid, norm), s.now())
		if err != nil {
			return err
		}
		if !ok || person == nil || !wellFormed {
			return errRecovery()
		}
		now := s.now()
		if err := tx.SetPasswordHash(pid, nh); err != nil {
			return err
		}
		if err := tx.SetMustChangePassword(pid, false); err != nil {
			return err
		}
		if err := tx.RevokeSessionsOf(pid, now); err != nil {
			return err
		}
		if err := tx.RevokeTokensOf(pid, now); err != nil {
			return err
		}
		if err := tx.DeleteAccountRecoveryCodes(pid); err != nil {
			return err
		}
		if err := tx.DeleteThrottle(recoverKey(login)); err != nil {
			return err
		}
		if err := tx.DeleteThrottle(login); err != nil {
			return err
		}
		if err := s.secNotice(tx, pid, domain.NoticePasswordChanged, "recovery code used"); err != nil { // SA2-03
			return err
		}
		return s.auditPlatform(tx, domain.Actor{PersonID: pid}, auditEntry{Action: domain.ActPersonRecovered,
			TargetType: "person", TargetID: pid})
	})
}

// newTempPassword returns a random password that satisfies the policy.
func (s *Service) newTempPassword() (string, error) {
	for range 5 {
		t, err := randToken(15) // 120 bits
		if err != nil {
			return "", err
		}
		if s.validatePassword(t) == nil {
			return t, nil
		}
	}
	return "", domain.Errf(domain.CodeInternal, "cannot generate a temporary password")
}

func checkResetReason(r string) (string, error) {
	r = strings.TrimSpace(r)
	switch n := utf8.RuneCountInString(r); {
	case n < 3:
		return "", domain.Invalid("reason", "required (at least 3 characters)")
	case n > maxResetReasonLen:
		return "", domain.Invalid("reason", "too long")
	}
	return r, nil
}

// resetPerson applies the common effects of an administrator or operator
// reset inside tx: new password, forced change, all sessions, tokens and
// recovery codes revoked, and the login's throttles cleared so the temporary
// password can be used at once.
func (s *Service) resetPerson(tx store.Tx, personID, login, tempHash string) error {
	now := s.now()
	if err := tx.SetPasswordHash(personID, tempHash); err != nil {
		return err
	}
	if err := tx.SetMustChangePassword(personID, true); err != nil {
		return err
	}
	if err := tx.RevokeSessionsOf(personID, now); err != nil {
		return err
	}
	if err := tx.RevokeTokensOf(personID, now); err != nil {
		return err
	}
	if err := tx.DeleteAccountRecoveryCodes(personID); err != nil {
		return err
	}
	for _, k := range []string{login, recoverKey(login)} {
		if err := tx.DeleteThrottle(k); err != nil {
			return err
		}
	}
	return s.secNotice(tx, personID, domain.NoticePasswordChanged, "password reset by an administrator or operator") // SA2-03
}

// AdminResetPassword lets an active administrator of netID reset the password
// of an active member of that network. It returns a one-time temporary
// password; show it to the administrator once and hand it over out of band.
// The person must change it at first sign-in (Actor.MustChangePassword).
//
// Guards: administrator role (or System actor); the target must be an active
// member of THIS network (anyone else is NotFound, so another network's
// administrator learns nothing); never one's own account (including through a
// second membership); the target must not hold the administrator role here nor
// an active membership in any other network (SA2-01: those need the operator
// CLI, since a reset hands over the person's whole identity); a reason is mandatory; at most adminResetsPerHour
// successful resets per administrator per hour. Resetting another
// administrator is flagged high_risk in the audit event.
func (s *Service) AdminResetPassword(ctx context.Context, actor domain.Actor, netID, memberID, reason string) (string, error) {
	reason, err := checkResetReason(reason)
	if err != nil {
		return "", err
	}
	temp, err := s.newTempPassword()
	if err != nil {
		return "", err
	}
	th, err := hashPassword(temp, s.cfg.Argon)
	if err != nil {
		return "", err
	}
	err = s.update(ctx, func(tx store.Tx) error {
		p, m, err := s.adminOnMember(tx, actor, netID, memberID, false)
		if err != nil {
			return err
		}
		if m.Status != domain.MemberActive {
			return domain.NotFound("member")
		}
		if actor.PersonID != "" && m.PersonID == actor.PersonID {
			return domain.Forbidden("administrators cannot perform this action on themselves")
		}
		if !p.actor.System {
			n, err := tx.CountAuditSince(netID, domain.ActMemberPasswordReset, p.memberID(), s.now().Add(-time.Hour))
			if err != nil {
				return err
			}
			if n >= adminResetsPerHour {
				return domain.Errf(domain.CodeRateLimited, "too many password resets this hour")
			}
		}
		person, _, err := tx.GetPerson(m.PersonID)
		if err != nil {
			return domain.NotFound("member")
		}
		if !p.actor.System {
			// SA2-01: a password reset replaces the person's global identity, so a
			// network administrator may only reset an account that exists in this
			// network alone and holds no administrator role here. Anything else
			// needs the platform operator (`circula reset-password`).
			if m.HasRole(domain.RoleAdmin) {
				return domain.Forbidden("this account can only be reset by the platform operator")
			}
			others, err := tx.ListMembershipsOfPerson(person.ID)
			if err != nil {
				return err
			}
			for _, o := range others {
				if o.NetworkID != netID && o.Status != domain.MemberDeparted {
					return domain.Forbidden("this account can only be reset by the platform operator")
				}
			}
		}
		if err := s.resetPerson(tx, person.ID, person.Login, th); err != nil {
			return err
		}
		risk := m.HasRole(domain.RoleAdmin)
		if err := s.audit(tx, p, auditEntry{Type: p.actorType(domain.ActorAdmin), Action: domain.ActMemberPasswordReset,
			TargetType: "member", TargetID: m.ID, Reason: reason, Subjects: []string{m.ID},
			After: map[string]any{"must_change_password": true, "high_risk": risk}}); err != nil {
			return err
		}
		return s.auditPlatform(tx, actor, auditEntry{Action: domain.ActPersonPasswordReset,
			TargetType: "person", TargetID: person.ID, Reason: reason,
			After: map[string]any{"by": "network_administrator", "high_risk": risk}})
	})
	if err != nil {
		return "", err
	}
	return temp, nil
}

// OperatorResetPassword is the platform operator's reset for use from the CLI
// (`circula reset-password`). actor must be domain.SystemActor. It does what
// AdminResetPassword does without a network: temporary password returned once,
// forced change, every session, token and recovery code revoked, audit event
// with the mandatory reason. An unknown login is NotFound (the CLI is trusted).
func (s *Service) OperatorResetPassword(ctx context.Context, actor domain.Actor, login, reason string) (string, error) {
	if !actor.System {
		return "", domain.ErrForbidden
	}
	reason, err := checkResetReason(reason)
	if err != nil {
		return "", err
	}
	login = normalizeLogin(login)
	temp, err := s.newTempPassword()
	if err != nil {
		return "", err
	}
	th, err := hashPassword(temp, s.cfg.Argon)
	if err != nil {
		return "", err
	}
	err = s.update(ctx, func(tx store.Tx) error {
		person, _, err := tx.GetPersonByLogin(login)
		if err != nil {
			return domain.NotFound("person")
		}
		if err := s.resetPerson(tx, person.ID, login, th); err != nil {
			return err
		}
		return s.auditPlatform(tx, actor, auditEntry{Type: domain.ActorOperator, Action: domain.ActPersonPasswordReset,
			TargetType: "person", TargetID: person.ID, Reason: reason,
			After: map[string]any{"by": "operator", "must_change_password": true}})
	})
	if err != nil {
		return "", err
	}
	return temp, nil
}
