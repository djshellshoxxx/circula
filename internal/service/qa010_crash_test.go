package service_test

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/djshellshoxxx/circula/internal/domain"
	"github.com/djshellshoxxx/circula/internal/service"
	"github.com/djshellshoxxx/circula/internal/store"
	tu "github.com/djshellshoxxx/circula/internal/testutil"
)

// QA-010: a process that dies while posting leaves, after restart, either
// the original state or the fully committed one, never partial entries, and
// the verification operation passes. Each case runs the accept in a child
// process that exits (no rollback, no close) at one write step of the
// posting unit, or right after the commit. The parent reopens the file.

const (
	qa010Child = "CIRCULA_QA010_CHILD" // "<op>:<nth>" or "after-commit"
	qa010DB    = "CIRCULA_QA010_DB"
	qa010Tx    = "CIRCULA_QA010_TX"
	qa010Payee = "CIRCULA_QA010_PAYEE"
	qa010Net   = "CIRCULA_QA010_NET"
	qa010Exit  = 77
)

var qa010Start = time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC)

// TestQA010CrashChild is the child half; it does nothing in a normal run.
func TestQA010CrashChild(t *testing.T) {
	point := os.Getenv(qa010Child)
	if point == "" {
		t.Skip("child process of TestQA010CrashDuringPostingThenRestart")
	}
	var fs *tu.FaultStore
	env := tu.OpenEnvAt(t, os.Getenv(qa010DB), func(s store.Store) store.Store { fs = tu.NewFaultStore(s); return fs },
		qa010Start.Add(time.Minute), 1_000_000)
	crash := func(string) { os.Exit(qa010Exit) } // no deferred cleanup, no rollback, no close
	if point != "after-commit" {
		op, nth, _ := strings.Cut(point, ":")
		n, _ := strconv.Atoi(nth)
		fs.Crash = crash
		fs.Arm(op, n)
	}
	payee := domain.Actor{PersonID: os.Getenv(qa010Payee), Scopes: []domain.Scope{domain.ScopeRead, domain.ScopeWrite}}
	_, err := env.Svc.AcceptTransaction(env.Ctx, payee, os.Getenv(qa010Net), service.TxRef{TxID: os.Getenv(qa010Tx), IdempotencyKey: "qa010"})
	if point == "after-commit" && err == nil {
		crash("")
	}
	fmt.Fprintf(os.Stderr, "child reached the end without crashing: %v\n", err)
	os.Exit(1)
}

func TestQA010CrashDuringPostingThenRestart(t *testing.T) {
	if testing.Short() {
		t.Skip("spawns child processes")
	}
	points := []string{"InsertEntry:1", "InsertEntry:2", "AddBalance:1", "AddBalance:2", "UpdateTransaction:1",
		"InsertStatusEvent:1", "InsertAudit:1", "PutIdem:1", "after-commit"}
	for _, point := range points {
		t.Run(point, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "circula.db")
			env := tu.OpenEnvAt(t, path, nil, qa010Start, 0)
			admin := env.Person("admin@example.org")
			net, _ := env.Network(admin, "crash", nil)
			a, aM := env.Member(admin, net, "a@example.org")
			b, bM := env.Member(admin, net, "b@example.org")
			p := tu.Must(env.Svc.ProposeTransaction(env.Ctx, a, net.ID, service.ProposeInput{CounterpartyID: bM.ID,
				Direction: service.ProposerPays, Amount: 1000, IdempotencyKey: env.Key()}))
			if err := env.Store.Close(); err != nil {
				t.Fatal(err)
			}
			_ = aM

			cmd := exec.Command(os.Args[0], "-test.run=^TestQA010CrashChild$", "-test.count=1")
			cmd.Env = append(os.Environ(), qa010Child+"="+point, qa010DB+"="+path, qa010Tx+"="+p.ID, qa010Payee+"="+b.PersonID, qa010Net+"="+net.ID)
			out, err := cmd.CombinedOutput()
			if ee, ok := err.(*exec.ExitError); !ok || ee.ExitCode() != qa010Exit {
				t.Fatalf("child did not crash at %s: %v\n%s", point, err, out)
			}

			re := tu.OpenEnvAt(t, path, nil, qa010Start.Add(2*time.Minute), 2_000_000)
			defer func() { _ = re.Store.Close() }()
			if res, err := store.IntegrityCheck(re.Ctx, path); err != nil || res != "ok" {
				t.Fatalf("integrity after restart: %q %v", res, err)
			}
			v := tu.Must(re.Svc.GetTransaction(re.Ctx, a, net.ID, p.ID))
			bal := tu.Must(re.Svc.GetBalance(re.Ctx, a, net.ID)).Balance
			switch point {
			case "after-commit":
				if v.Tx.Status != domain.TxCompleted || len(v.Entries) != 2 || bal != -1000 {
					t.Fatalf("committed state lost: status=%s entries=%d balance=%d", v.Tx.Status, len(v.Entries), bal)
				}
			default:
				if v.Tx.Status != domain.TxProposed || len(v.Entries) != 0 || bal != 0 || v.Tx.LedgerSeq != 0 {
					t.Fatalf("partial state after crash: status=%s entries=%d balance=%d seq=%d", v.Tx.Status, len(v.Entries), bal, v.Tx.LedgerSeq)
				}
			}
			if rep := tu.Must(re.Svc.VerifyLedger(re.Ctx, admin, net.ID)); !rep.OK {
				t.Fatalf("verify after restart: %v", rep.Findings)
			}
			// The interrupted request can be retried and posts exactly once.
			tu.Must(re.Svc.AcceptTransaction(re.Ctx, b, net.ID, service.TxRef{TxID: p.ID, IdempotencyKey: "qa010"}))
			if got := tu.Must(re.Svc.GetBalance(re.Ctx, a, net.ID)).Balance; got != -1000 {
				t.Fatalf("after retry: balance %d", got)
			}
		})
	}
}
