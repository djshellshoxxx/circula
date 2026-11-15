package service_test

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"testing"
)

// QA-001 run sizes. The default run on every change keeps the counts in
// TestPropTwoNetworkLedger. The long run (nightly CI, before a release) sets
//
//	CIRCULA_QA001_SEQUENCES  number of seeded sequences (spec: 1,000 short, 100,000 long; DEC-023)
//	CIRCULA_QA001_STEPS      operations per sequence (spec: at most 50)
//	CIRCULA_QA001_SEED       replay exactly one seed
//	CIRCULA_QA001_FAILDIR    where a failing seed is saved as a fixture file
//
// and runs TestQA001LongRun. Every failure names its seed.
func qa001Env(name string, def int64) int64 {
	if v, err := strconv.ParseInt(os.Getenv(name), 10, 64); err == nil && v > 0 {
		return v
	}
	return def
}

func TestQA001LongRun(t *testing.T) {
	n := qa001Env("CIRCULA_QA001_SEQUENCES", 0)
	one := qa001Env("CIRCULA_QA001_SEED", 0)
	if n == 0 && one == 0 {
		t.Skip("set CIRCULA_QA001_SEQUENCES or CIRCULA_QA001_SEED for the QA-001 long run")
	}
	steps := int(qa001Env("CIRCULA_QA001_STEPS", 50))
	first, last := int64(1), n
	if one != 0 {
		first, last = one, one
	}
	for seed := first; seed <= last; seed++ {
		seed := seed
		ok := t.Run(fmt.Sprintf("seed%d", seed), func(t *testing.T) { runTwoNet(t, seed+1000, steps) })
		if !ok {
			if dir := os.Getenv("CIRCULA_QA001_FAILDIR"); dir != "" {
				_ = os.MkdirAll(dir, 0o755)
				fixture := fmt.Sprintf("CIRCULA_QA001_SEED=%d\nCIRCULA_QA001_STEPS=%d\n", seed, steps)
				_ = os.WriteFile(filepath.Join(dir, fmt.Sprintf("qa001-seed-%d.env", seed)), []byte(fixture), 0o644)
			}
			t.Fatalf("QA-001 sequence failed; replay: CIRCULA_QA001_SEED=%d CIRCULA_QA001_STEPS=%d go test ./internal/service -run '^TestQA001LongRun$'", seed, steps)
		}
	}
}
