package service

import (
	"context"
	"runtime"
	"testing"

	"github.com/djshellshoxxx/circula/internal/passwordwork"
)

// SEC-001 / DEC-006: cost of the Argon2id parameters the product uses
// (DefaultArgon2Params; `circula serve` does not override them). Run with
//
//	go test ./internal/service -run NONE -bench Argon2id -benchtime 20x
//
// and record results with the CPU model in docs/dev/argon2-benchmark.md. The
// numbers describe the machine they ran on; they do not pick the minimum
// supported hardware, which is the owner's decision (DEC-006).

const benchPassword = "correct horse battery staple 42"

// BenchmarkArgon2idHash measures one hash with the default parameters
// (registration, password change and reset).
func BenchmarkArgon2idHash(b *testing.B) {
	ctx, work := context.Background(), passwordVectorTicket(b)
	p := DefaultArgon2Params()
	for b.Loop() {
		if _, err := hashPassword(ctx, work, benchPassword, p); err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkArgon2idVerify measures one verification of a default-parameter
// hash: the cost of every sign-in attempt, successful or not (the unknown-login
// path verifies against a dummy hash of the same cost, SEC-005).
func BenchmarkArgon2idVerify(b *testing.B) {
	ctx, work := context.Background(), passwordVectorTicket(b)
	h, err := hashPassword(ctx, work, benchPassword, DefaultArgon2Params())
	if err != nil {
		b.Fatal(err)
	}
	for b.Loop() {
		if ok, err := verifyPassword(ctx, work, benchPassword, h); err != nil || !ok {
			b.Fatalf("verification failed: %v", err)
		}
	}
}

// BenchmarkArgon2idVerifyParallel measures raw verification cost on every CPU.
// Its explicitly isolated capacity admits GOMAXPROCS workers; this is not the
// production service's two-operation admission limit or a sign-in load test.
func BenchmarkArgon2idVerifyParallel(b *testing.B) {
	ctx, setup := context.Background(), passwordVectorTicket(b)
	h, err := hashPassword(ctx, setup, benchPassword, DefaultArgon2Params())
	if err != nil {
		b.Fatal(err)
	}
	setup.Release()
	limit, err := passwordwork.New(runtime.GOMAXPROCS(0))
	if err != nil {
		b.Fatal(err)
	}
	b.RunParallel(func(pb *testing.PB) {
		work, err := limit.Acquire(ctx)
		if err != nil {
			b.Error(err)
			return
		}
		defer work.Release()
		for pb.Next() {
			if ok, err := verifyPassword(ctx, work, benchPassword, h); err != nil || !ok {
				b.Errorf("verification failed: %v", err)
				return
			}
		}
	})
}

// TestDefaultArgon2ParamsMatchBenchmarkDoc pins the parameters the benchmark
// document describes; change both together.
func TestDefaultArgon2ParamsMatchBenchmarkDoc(t *testing.T) {
	if p := DefaultArgon2Params(); p != (Argon2Params{MemoryKiB: 64 * 1024, Time: 3, Threads: 2, KeyLen: 32, SaltLen: 16}) {
		t.Errorf("DefaultArgon2Params = %+v: re-run the benchmark and update docs/dev/argon2-benchmark.md", p)
	}
}
