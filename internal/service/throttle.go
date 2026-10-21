package service

import (
	"context"
	"net/netip"
	"time"

	"github.com/djshellshoxxx/circula/internal/domain"
	"github.com/djshellshoxxx/circula/internal/store"
)

// Login and recovery throttling (SEC-006, SA2-10).
//
// A lock keyed by login alone lets anyone lock a victim out from one address.
// With a client address (WithClientIP) every attempt is therefore counted
// twice: against the pair (login, address), which locks after throttleAfter
// failures, and against the login as a whole, which locks only after the much
// higher throttleGlobalAfter failures, so guessing spread over many addresses
// is still bounded. A lock never lasts longer than throttleMax. Counters of
// an idle login decay after throttleDecay. Without an address (CLI, tests) the
// login row alone applies with the low limit.
const (
	throttleAfter       = 5
	throttleGlobalAfter = 100
	throttleMax         = 15 * time.Minute
	throttleDecay       = 24 * time.Hour
)

type clientIPKey struct{}

// WithClientIP returns ctx carrying the caller's network address for
// LoginStep, Login and RecoverPassword. Adapters must pass the address they
// trust (after proxy handling). IPv6 addresses are reduced to their /64.
// Unparseable input is bucketed together.
func WithClientIP(ctx context.Context, ip string) context.Context {
	return context.WithValue(ctx, clientIPKey{}, canonicalIP(ip))
}

func canonicalIP(s string) string {
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

// lockFor is the lock after the failures-th consecutive failure at the
// per-address threshold.
func lockFor(failures int) time.Duration { return lockAfter(failures, throttleAfter) }

func lockAfter(failures, after int) time.Duration {
	if failures < after {
		return 0
	}
	return throttleMax
}

func ipKey(key, ip string) string { return key + " ip:" + ip }

// readThrottle returns the live failure count and lock end of key.
func (s *Service) readThrottle(tx store.Tx, key string) (int, time.Time, error) {
	f, until, _, err := tx.GetThrottle(key)
	if err != nil {
		return 0, time.Time{}, err
	}
	if f > 0 && s.now().After(until.Add(throttleDecay)) {
		f = 0 // idle for a day: the history is forgotten
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
	pf, pu, err := s.readThrottle(tx, pk)
	if err != nil {
		return err
	}
	gf, gu, err := s.readThrottle(tx, key)
	if err != nil {
		return err
	}
	now := s.now()
	if now.Before(pu) || now.Before(gu) {
		return domain.Errf(domain.CodeRateLimited, "too many attempts; try again later")
	}
	if err := tx.PutThrottle(pk, pf+1, now.Add(lockAfter(pf+1, throttleAfter))); err != nil {
		return err
	}
	return tx.PutThrottle(key, gf+1, now.Add(lockAfter(gf+1, throttleGlobalAfter)))
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
		for _, k := range []string{login, recoverKey(login), mfaKey(person.ID)} {
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
