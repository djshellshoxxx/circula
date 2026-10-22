package service_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/djshellshoxxx/circula/internal/domain"
	"github.com/djshellshoxxx/circula/internal/httpx"
	"github.com/djshellshoxxx/circula/internal/service"
	tu "github.com/djshellshoxxx/circula/internal/testutil"
)

// A3-01: the web adapter passes httpx.RateKey(ip) ("2001:db8:1::/64") to
// WithClientIP, which cannot parse a prefix and buckets it as "?".
func TestA3_01_IPv6RateKeyKeepsBucketsApart(t *testing.T) {
	e := tu.NewEnv(t)
	e.Person("ana@example.org")
	atk := service.WithClientIP(e.Ctx, httpx.RateKey("2001:db8:aaaa::99"))
	vic := service.WithClientIP(e.Ctx, httpx.RateKey("2001:db8:bbbb::7"))
	for i := 0; i < 5; i++ {
		_, _ = e.Svc.LoginStep(atk, "ana@example.org", bad)
	}
	_, err := e.Svc.LoginStep(vic, "ana@example.org", pw)
	if err != nil {
		t.Fatalf("victim on a different IPv6 /64 locked out by attacker: %v", err)
	}
}

// A3-02: the login-wide lock (100 failures from many addresses) is renewable
// forever by ONE request per 15 minutes, because the counter never resets
// after a lock expires.
func TestA3_02_GlobalLockNotRenewable(t *testing.T) {
	e := tu.NewEnv(t)
	e.Person("ana@example.org")
	for i := 0; i < 20; i++ {
		ctx := service.WithClientIP(e.Ctx, fmt.Sprintf("203.0.113.%d", i+1))
		for j := 0; j < 5; j++ {
			_, _ = e.Svc.LoginStep(ctx, "ana@example.org", bad)
		}
	}
	vic := service.WithClientIP(e.Ctx, "198.51.100.7")
	_, err := e.Svc.LoginStep(vic, "ana@example.org", pw)
	wantCode(t, err, domain.ErrRateLimited) // victim locked from a clean address
	e.Clock.Advance(15*time.Minute + time.Second)
	atk := service.WithClientIP(e.Ctx, "192.0.2.1")
	_, _ = e.Svc.LoginStep(atk, "ana@example.org", bad) // one request at expiry must not re-arm the lock
	if _, err = e.Svc.LoginStep(vic, "ana@example.org", pw); err != nil {
		t.Fatalf("victim still locked after expiry: %v", err)
	}
}
