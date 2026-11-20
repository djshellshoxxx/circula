package service

import (
	"context"
	"errors"
	"testing"
)

// Digest-only exports must not allocate storage for the serialized ledger.
// This bounded fixture writes 2 MiB in reusable chunks; retained-output sinks
// grow an allocation on repeated writes, while the checksum sink stays fixed.
func TestClosureChecksumSinkDoesNotRetainOutput(t *testing.T) {
	payload := make([]byte, 32<<10)
	ctx := context.Background()
	allocs := func(chunks int) float64 {
		return testing.AllocsPerRun(5, func() {
			w, err := (checksumSink{ctx}).Create("ledger.json")
			if err != nil {
				t.Fatal(err)
			}
			for i := 0; i < chunks; i++ {
				if n, err := w.Write(payload); n != len(payload) || err != nil {
					t.Fatalf("write = %d, %v", n, err)
				}
			}
			if err := w.Close(); err != nil {
				t.Fatal(err)
			}
		})
	}
	if small, large := allocs(1), allocs(64); large > small {
		t.Fatalf("sink allocations grew with serialized output: one chunk %.0f, 64 chunks %.0f", small, large)
	}
}

func TestClosureChecksumSinkReportsCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	sink := checksumSink{ctx}
	w, err := sink.Create("ledger.json")
	if err != nil {
		t.Fatal(err)
	}
	cancel()
	if n, err := w.Write([]byte("ordinary row")); n != 0 || !errors.Is(err, context.Canceled) {
		t.Fatalf("cancelled write = %d, %v", n, err)
	}
	if err := w.Close(); !errors.Is(err, context.Canceled) {
		t.Fatalf("cancelled close = %v", err)
	}
	if _, err := sink.Create("manifest.json"); !errors.Is(err, context.Canceled) {
		t.Fatalf("cancelled create = %v", err)
	}
}
