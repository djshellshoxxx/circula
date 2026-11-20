package service_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"runtime"
	"runtime/debug"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/djshellshoxxx/circula/internal/domain"
	"github.com/djshellshoxxx/circula/internal/service"
	tu "github.com/djshellshoxxx/circula/internal/testutil"
)

type qa001Config struct {
	StartSeed int64 `json:"start_seed"`
	Sequences int   `json:"sequences"`
	Shards    int   `json:"shards"`
}

func readQA001Config(getenv func(string) string, budget int) (qa001Config, error) {
	c := qa001Config{StartSeed: 100000, Sequences: budget, Shards: min(runtime.GOMAXPROCS(0), 8)}
	read := func(name string, fallback, low, high int64) (int64, error) {
		raw := getenv(name)
		if raw == "" {
			return fallback, nil
		}
		n, err := strconv.ParseInt(raw, 10, 64)
		if err != nil || n < low || n > high {
			return 0, fmt.Errorf("%s must be an integer in [%d, %d], got %q", name, low, high, raw)
		}
		return n, nil
	}
	var err error
	if getenv("CIRCULA_QA001_SEED") != "" {
		if getenv("CIRCULA_QA001_START_SEED") != "" || getenv("CIRCULA_QA001_SEQUENCES") != "" {
			return c, errors.New("CIRCULA_QA001_SEED cannot be combined with START_SEED or SEQUENCES")
		}
		c.StartSeed, err = read("CIRCULA_QA001_SEED", 0, 0, math.MaxInt64)
		c.Sequences = 1
	} else {
		c.StartSeed, err = read("CIRCULA_QA001_START_SEED", c.StartSeed, 0, math.MaxInt64)
		if err == nil {
			var n int64
			n, err = read("CIRCULA_QA001_SEQUENCES", int64(c.Sequences), 1, 1000000)
			c.Sequences = int(n)
		}
	}
	if err != nil {
		return c, err
	}
	if int64(c.Sequences-1) > math.MaxInt64-c.StartSeed {
		return c, errors.New("QA-001 seed range overflows int64")
	}
	shards, err := read("CIRCULA_QA001_SHARDS", int64(c.Shards), 1, 8)
	c.Shards = min(int(shards), c.Sequences)
	return c, err
}

func newQA001Run(t *testing.T, seed int64, cfg qa001Config) *qa001Run {
	return &qa001Run{t: t, seed: seed, config: cfg, ok: map[string]int{}, tried: map[string]int{},
		steps: []qa001Step{{Op: "setup", Result: "running"}}}
}

func runQA001Sequence(t *testing.T, seed int64, cfg qa001Config) (r *qa001Run) {
	r = newQA001Run(t, seed, cfg)
	defer r.captureFailure()
	r.e = tu.NewEnvConfig(t, nil, func(c *service.Config) { c.MaxOpenRequestsPerPair = 1_000_000 })
	r.sequence(seed)
	return r
}

func (r *qa001Run) step(op string, apply func() (string, error)) {
	r.tried[op]++
	// Record the operation before helpers can panic or call t.Fatal.
	r.steps = append(r.steps, qa001Step{Op: op, Result: "running"})
	i := len(r.steps) - 1
	detail, err := apply()
	r.steps[i].Detail, r.steps[i].Result = detail, "ok"
	if err != nil {
		r.steps[i].Result = err.Error()
		if err != errSkip && domain.CodeOf(err) == domain.CodeInternal {
			r.fail(fmt.Sprintf("%s: internal error: %v", op, err))
		}
	} else {
		r.ok[op]++
		if op == "accept" {
			r.ok["complete"]++
			r.tried["complete"]++
		}
	}
	if msg := r.check(); msg != "" {
		r.fail(fmt.Sprintf("after %s: %s", op, msg))
	}
}

// A defer, rather than only explicit invariant assertions, captures failures
// in fixture helpers too. Every sequence owns its testing.T and environment.
func (r *qa001Run) captureFailure() {
	if value := recover(); value != nil {
		msg := fmt.Sprintf("helper panic: %v", value)
		path := r.saveFailure(msg, string(debug.Stack()))
		r.t.Errorf("QA-001 seed %d: %s; fixture %s", r.seed, msg, path)
	} else if r.t.Failed() && !r.fixtureWritten {
		path := r.saveFailure("test/helper failure; see test log", "")
		r.t.Logf("QA-001 seed %d: helper failure fixture %s", r.seed, path)
	}
}

func (r *qa001Run) saveFailure(msg, stack string) string {
	r.fixtureWritten = true
	if n := len(r.steps); n > 0 && r.steps[n-1].Result == "running" {
		r.steps[n-1].Result = "aborted: " + msg
	}
	revision := os.Getenv("CIRCULA_QA001_REVISION")
	if revision == "" {
		revision = "unknown (run scripts/run_qa001.py to record the checkout)"
	}
	fx := qa001Fixture{Seed: r.seed, MaxOps: qa001MaxOps, Failure: msg, Steps: r.steps,
		Revision: revision, GoVersion: runtime.Version(), Platform: runtime.GOOS + "/" + runtime.GOARCH,
		Config: r.config, Stack: stack}
	if r.e != nil {
		fx.Clock = r.e.Clock.Now().Format(time.RFC3339Nano)
	}
	dir := os.Getenv("CIRCULA_QA001_FAILDIR")
	var err error
	if dir == "" {
		dir, err = os.MkdirTemp("", "circula-qa001-")
	} else {
		err = os.MkdirAll(dir, 0o755)
	}
	if err != nil {
		return "(not written: " + err.Error() + ")"
	}
	path := filepath.Join(dir, fmt.Sprintf("qa001-seed-%d.json", r.seed))
	b, err := json.MarshalIndent(fx, "", "  ")
	if err == nil {
		err = os.WriteFile(path, append(b, '\n'), 0o644)
	}
	if err != nil {
		return "(not written: " + err.Error() + ")"
	}
	return path
}

func (r *qa001Run) fail(msg string) {
	r.t.Helper()
	path := r.saveFailure(msg, "")
	r.t.Fatalf("QA-001 invariant violation, seed %d (replay: CIRCULA_QA001_SEED=%d), fixture %s: %s", r.seed, r.seed, path, msg)
}

func TestQA001StrictRanges(t *testing.T) {
	for _, budget := range []int{250, 50} {
		cfg, err := readQA001Config(func(string) string { return "" }, budget)
		if err != nil || cfg.Sequences != budget || cfg.StartSeed != 100000 {
			t.Fatalf("short budget changed: %+v %v", cfg, err)
		}
	}
	for _, values := range []map[string]string{
		{"SEQUENCES": "bad"}, {"SEQUENCES": "0"}, {"SEQUENCES": "1000001"},
		{"START_SEED": "-1"}, {"START_SEED": "9223372036854775807", "SEQUENCES": "2"},
		{"SEED": "1", "SEQUENCES": "2"}, {"SEED": "1", "START_SEED": "2"},
		{"SHARDS": "0"}, {"SHARDS": "9"}, {"SEED": "bad"},
	} {
		if _, err := readQA001Config(func(k string) string { return values[strings.TrimPrefix(k, "CIRCULA_QA001_")] }, 250); err == nil {
			t.Errorf("invalid configuration accepted: %v", values)
		}
	}
	values := map[string]string{"START_SEED": "200000", "SEQUENCES": "37", "SHARDS": "8"}
	cfg, err := readQA001Config(func(k string) string { return values[strings.TrimPrefix(k, "CIRCULA_QA001_")] }, 250)
	if err != nil {
		t.Fatal(err)
	}
	seen := map[int64]bool{}
	for shard := 0; shard < cfg.Shards; shard++ {
		for i := shard; i < cfg.Sequences; i += cfg.Shards {
			seed := cfg.StartSeed + int64(i)
			if seen[seed] || seed < 200000 || seed > 200036 {
				t.Fatalf("overlapping/out-of-range seed: %d", seed)
			}
			seen[seed] = true
		}
	}
	if len(seen) != 37 {
		t.Fatalf("range covered %d seeds", len(seen))
	}
}

func TestQA001IsolatedReplay(t *testing.T) {
	run := func(name string, seed int64) []qa001Step {
		var steps []qa001Step
		t.Run(name, func(t *testing.T) {
			r := runQA001Sequence(t, seed, qa001Config{StartSeed: seed, Sequences: 1, Shards: 1})
			steps = append([]qa001Step(nil), r.steps...)
		})
		return steps
	}
	before := run("first", 100017)
	run("intervening", 100001)
	if after := run("replayed", 100017); !reflect.DeepEqual(before, after) {
		t.Fatal("single-seed replay changed operation IDs, choices or outcomes")
	}
}

// Run deliberate failures in a child test process so the parent can inspect
// real t.Fatal/panic paths without making the successful suite itself fail.
func TestQA001FailureCapture(t *testing.T) {
	if mode := os.Getenv("CIRCULA_QA001_CAPTURE_TEST"); mode != "" {
		r := newQA001Run(t, 42, qa001Config{StartSeed: 42, Sequences: 1, Shards: 1})
		defer r.captureFailure()
		r.step("injected-"+mode, func() (string, error) {
			switch mode {
			case "panic":
				panic("fixture helper failed")
			case "fatal":
				t.Fatal("fixture helper failed")
			}
			return "failing operation details", errors.New("injected internal failure")
		})
		return
	}
	for _, mode := range []string{"panic", "fatal", "internal"} {
		t.Run(mode, func(t *testing.T) {
			dir := t.TempDir()
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()
			cmd := exec.CommandContext(ctx, os.Args[0], "-test.run=^TestQA001FailureCapture$", "-test.v")
			cmd.Env = append(os.Environ(), "CIRCULA_QA001_CAPTURE_TEST="+mode,
				"CIRCULA_QA001_FAILDIR="+dir, "CIRCULA_QA001_REVISION=fixture-revision")
			out, err := cmd.CombinedOutput()
			if err == nil {
				t.Fatal("injected failure unexpectedly passed")
			}
			b, err := os.ReadFile(filepath.Join(dir, "qa001-seed-42.json"))
			if err != nil {
				t.Fatalf("failure fixture missing: %v\n%s", err, out)
			}
			var fx qa001Fixture
			if err := json.Unmarshal(b, &fx); err != nil {
				t.Fatal(err)
			}
			if fx.Seed != 42 || fx.Revision != "fixture-revision" || fx.GoVersion == "" || fx.Failure == "" || len(fx.Steps) < 1 {
				t.Fatalf("incomplete fixture: %+v", fx)
			}
			step := fx.Steps[len(fx.Steps)-1]
			if step.Op != "injected-"+mode || step.Result == "running" || step.Result == "ok" {
				t.Fatalf("failing step was lost: %+v", step)
			}
			if mode == "internal" && step.Detail != "failing operation details" {
				t.Fatalf("internal error lost operation detail: %+v", step)
			}
		})
	}
}
