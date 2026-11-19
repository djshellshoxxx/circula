package service

import (
	"context"
	"net/netip"
	"time"

	"github.com/djshellshoxxx/circula/internal/domain"
	store "github.com/djshellshoxxx/circula/internal/repository"
)

// Login and recovery throttling (SEC-006, SA2-10).
// With a client address, attempts count against both (login, address) and a
// larger login-wide budget. Without an address, the local budget applies.
// Operator settings may tighten the defaults; locks are always finite.
// Inactive counters decay after a day, independently of the configured lock.
const throttleDecay = 24 * time.Hour

type clientIPKey struct{}

// WithClientIP returns ctx carrying the caller's network address for
// LoginStep, Login and RecoverPassword. Adapters must pass the address they
// trust (after proxy handling). IPv6 addresses are reduced to their /64.
// Unparseable input is bucketed together.
func WithClientIP(ctx context.Context, ip string) context.Context {
	return context.WithValue(ctx, clientIPKey{}, canonicalIP(ip))
}

func canonicalIP(s string) string {
	if pf, err := netip.ParsePrefix(s); err == nil { // an adapter rate key such as "2001:db8::/64"
		s = pf.Addr().String()
	}
	a, err := netip.ParseAddr(s)
	if err != nil {
		return "?"
	}
	a = a.Unmap().WithZone("")
	if a.Is6() {
		p, _ := a.Prefix(64)
		return p.Addr().String() + "/64"
	}
	return a.String()
}

func clientIP(ctx context.Context) string {
	ip, _ := ctx.Value(clientIPKey{}).(string)
	return ip
}

func (s *Service) lockAfter(failures, after int) time.Duration {
	if failures < after {
		return 0
	}
	return s.cfg.AuthThrottleLock
}

func ipKey(key, ip string) string { return key + " ip:" + ip }

// readThrottle returns the live failure count and lock end of key.
func (s *Service) readThrottle(tx store.Tx, key string, after int) (int, time.Time, error) {
	f, until, _, err := tx.GetThrottle(key)
	if err != nil {
		return 0, time.Time{}, err
	}
	if f > 0 && s.now().After(until.Add(throttleDecay)) {
		f = 0 // idle for a day: the history is forgotten
	}
	if f >= after && !until.IsZero() && !s.now().Before(until) {
		// A lock that has run out grants half the allowance again, so a single
		// attempt timed at expiry cannot re-arm it (A3-02).
		f = after - after/2
	}
	return f, until, nil
}

// reserveAttemptCtx refuses while key is locked and otherwise counts the
// attempt up front (so parallel guesses cannot beat the limit). key is a login
// or recoverKey(login); the address comes from ctx.
func (s *Service) reserveAttemptCtx(ctx context.Context, tx store.Tx, key string) error {
	ip := clientIP(ctx)
	if ip == "" {
		return s.reserveAttempt(tx, key)
	}
	pk := ipKey(key, ip)
	pf, pu, err := s.readThrottle(tx, pk, s.cfg.AuthThrottleAttempts)
	if err != nil {
		return err
	}
	gf, gu, err := s.readThrottle(tx, key, s.cfg.AuthThrottleGlobalAttempts)
	if err != nil {
		return err
	}
	now := s.now()
	if now.Before(pu) || now.Before(gu) {
		return domain.Errf(domain.CodeRateLimited, "too many attempts; try again later")
	}
	if err := tx.PutThrottle(pk, pf+1, now.Add(s.lockAfter(pf+1, s.cfg.AuthThrottleAttempts))); err != nil {
		return err
	}
	return tx.PutThrottle(key, gf+1, now.Add(s.lockAfter(gf+1, s.cfg.AuthThrottleGlobalAttempts)))
}

// clearThrottle forgets the counters of key after a success: the caller's own
// pair row and the login-wide row.
func (s *Service) clearThrottle(ctx context.Context, tx store.Tx, key string) error {
	if ip := clientIP(ctx); ip != "" {
		if err := tx.DeleteThrottle(ipKey(key, ip)); err != nil {
			return err
		}
	}
	return tx.DeleteThrottle(key)
}

// OperatorUnlockLogin clears every login and recovery lock of login, for all
// addresses (`circula unlock`). actor must be the system actor. The login must
// exist. The audit event carries no secret; the result is the number of
// cleared rows.
func (s *Service) OperatorUnlockLogin(ctx context.Context, actor domain.Actor, login string) (int64, error) {
	if !actor.System {
		return 0, domain.ErrForbidden
	}
	login = normalizeLogin(login)
	var n int64
	err := s.update(ctx, func(tx store.Tx) error {
		person, _, err := tx.GetPersonByLogin(login)
		if err != nil {
			return domain.NotFound("person")
		}
		for _, k := range []string{login, recoverKey(login), mfaKey(person.ID), recgenKey(person.ID)} {
			c, err := tx.DeleteThrottleFamily(k)
			if err != nil {
				return err
			}
			n += c
		}
		return s.auditPlatform(tx, actor, auditEntry{Type: domain.ActorOperator, Action: domain.ActLoginUnlocked,
			TargetType: "person", TargetID: person.ID, After: map[string]any{"cleared": n}})
	})
	return n, err
}
