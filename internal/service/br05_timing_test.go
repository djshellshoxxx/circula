package service_test

import (
	"fmt"
	"runtime"
	"sort"
	"testing"
	"time"

	"github.com/djshellshoxxx/circula/internal/service"
	tu "github.com/djshellshoxxx/circula/internal/testutil"
)

// BR-05 (SEC-005): an unknown login and a known login with a wrong secret
// take comparable time, for sign-in and for account recovery, so response
// time does not reveal which logins exist. Samples alternate between the two
// paths and medians are compared, which keeps the check stable on a loaded
// machine; the bounds still catch a skipped hash (a 10x or larger gap).
func TestBR05EnumerationTimingIsComparable(t *testing.T) {
	if testing.Short() || raceOn {
		t.Skip("timing test; skipped with -short and under the race detector")
	}
	env := tu.NewEnvConfig(t, nil, func(c *service.Config) {
		c.Argon = service.Argon2Params{MemoryKiB: 8 << 10, Time: 2, Threads: 1, KeyLen: 32, SaltLen: 16}
	})
	const n = 21
	known := make([]string, n)
	for i := range known {
		known[i] = fmt.Sprintf("member%02d@example.org", i)
		tu.Must(env.Svc.RegisterPerson(env.Ctx, service.RegisterInput{Login: known[i], Password: "correct horse battery"}))
	}
	measure := func(f func()) time.Duration { s := time.Now(); f(); return time.Since(s) }
	var unknownLogin, wrongPassword, unknownRecover, wrongCode []time.Duration
	for i := 0; i < n; i++ { // distinct logins, so the per-login throttle never engages
		// Each pair is measured back to back, in alternating order, so garbage
		// collection after an Argon2 hash and other drift fall on both sides.
		pairs := []struct {
			a, b   func()
			da, db *[]time.Duration
		}{
			{func() {
				_, _ = env.Svc.LoginStep(env.Ctx, fmt.Sprintf("nobody%02d@example.org", i), "wrong horse battery")
			},
				func() { _, _ = env.Svc.LoginStep(env.Ctx, known[i], "wrong horse battery") }, &unknownLogin, &wrongPassword},
			{func() {
				_ = env.Svc.RecoverPassword(env.Ctx, fmt.Sprintf("nobody%02d@example.org", i), "ABCD-EFGH-JKLM", "a new long passphrase")
			}, func() { _ = env.Svc.RecoverPassword(env.Ctx, known[i], "ABCD-EFGH-JKLM", "a new long passphrase") },
				&unknownRecover, &wrongCode},
		}
		for _, p := range pairs {
			runtime.GC()
			if i%2 == 0 {
				*p.da = append(*p.da, measure(p.a))
				*p.db = append(*p.db, measure(p.b))
			} else {
				*p.db = append(*p.db, measure(p.b))
				*p.da = append(*p.da, measure(p.a))
			}
		}
	}
	check := func(what string, a, b []time.Duration, bound float64) {
		ma, mb := median(a), median(b)
		r := float64(ma) / float64(mb)
		t.Logf("%s: medians %s vs %s (ratio %.2f)", what, ma, mb, r)
		if r < 1/bound || r > bound {
			t.Errorf("%s: unknown-login median %s vs known-login median %s (ratio %.2f, allowed 1/%.0f..%.0f)", what, ma, mb, r, bound, bound)
		}
	}
	check("sign-in", unknownLogin, wrongPassword, 2)
	check("recovery", unknownRecover, wrongCode, 3)
}

func median(d []time.Duration) time.Duration {
	s := append([]time.Duration(nil), d...)
	sort.Slice(s, func(i, j int) bool { return s[i] < s[j] })
	return s[len(s)/2]
}
