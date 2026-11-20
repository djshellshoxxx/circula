package service

import (
	"context"
	"strings"
	"testing"

	"github.com/djshellshoxxx/circula/internal/passwordwork"
)

func passwordVectorTicket(t testing.TB) *passwordwork.Ticket {
	t.Helper()
	limit, err := passwordwork.New(1)
	if err != nil {
		t.Fatal(err)
	}
	work, err := limit.Acquire(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(work.Release)
	return work
}

// A stored hash with t=0 must be rejected, not panic inside argon2.IDKey.
func TestVerifyPasswordRejectsDegenerateParams(t *testing.T) {
	ctx, work := context.Background(), passwordVectorTicket(t)
	p := Argon2Params{MemoryKiB: 64, Time: 1, Threads: 1, KeyLen: 16, SaltLen: 8}
	h, err := hashPassword(ctx, work, "correct horse", p)
	if err != nil {
		t.Fatal(err)
	}
	if ok, err := verifyPassword(ctx, work, "correct horse", h); err != nil || !ok {
		t.Fatalf("valid hash rejected: %v", err)
	}
	for name, bad := range map[string]string{
		"t=0":    strings.Replace(h, "t=1", "t=0", 1),
		"t huge": strings.Replace(h, "t=1", "t=4000000", 1),
		"p=0":    strings.Replace(h, "p=1", "p=0", 1),
	} {
		if bad == h {
			t.Fatalf("%s: substitution did not apply to %s", name, h)
		}
		if ok, err := verifyPassword(ctx, work, "correct horse", bad); err != nil || ok {
			t.Errorf("%s: accepted=%v err=%v", name, ok, err)
		}
	}
}

// The PHC string layout: $argon2id$v=19$m=..,t=..,p=..$salt$hash, unpadded standard base64.
func TestHashPasswordPHCFormat(t *testing.T) {
	ctx, work := context.Background(), passwordVectorTicket(t)
	d := DefaultArgon2Params()
	if d.MemoryKiB < 19*1024 || d.Time < 2 {
		t.Errorf("defaults below the OWASP minimum (19 MiB, t=2): %+v", d)
	}
	p := Argon2Params{MemoryKiB: 64, Time: 1, Threads: 1, KeyLen: 16, SaltLen: 8}
	h, err := hashPassword(ctx, work, "x", p)
	if err != nil {
		t.Fatal(err)
	}
	parts := strings.Split(h, "$")
	if len(parts) != 6 || parts[0] != "" || parts[1] != "argon2id" || parts[2] != "v=19" || parts[3] != "m=64,t=1,p=1" {
		t.Fatalf("PHC layout: %s", h)
	}
	if strings.Contains(parts[4]+parts[5], "=") {
		t.Errorf("salt/hash must be unpadded base64 (PHC string format): %s", h)
	}
}
