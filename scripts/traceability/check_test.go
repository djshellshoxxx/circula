package main

import (
	"bytes"
	"flag"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// The command is exercised in a subprocess because its failure path exits.
func TestTraceabilityCLIProcess(t *testing.T) {
	if os.Getenv("CIRCULA_TRACEABILITY_CHECK_TEST") != "1" {
		return
	}
	for i, arg := range os.Args {
		if arg == "--" {
			os.Args = append([]string{"traceability"}, os.Args[i+1:]...)
			flag.CommandLine = flag.NewFlagSet("traceability", flag.ExitOnError)
			main()
			return
		}
	}
	t.Fatal("missing command arguments")
}

func traceabilityCommand(args ...string) ([]byte, error) {
	cmd := exec.Command(os.Args[0], append([]string{"-test.run=^TestTraceabilityCLIProcess$", "--"}, args...)...)
	cmd.Env = append(os.Environ(), "CIRCULA_TRACEABILITY_CHECK_TEST=1")
	return cmd.CombinedOutput()
}

func TestCheckRejectsRegeneratedDuplicateStatusIDs(t *testing.T) {
	const header = "# Requirements\n| ID | Priority | Status | Note |\n|---|---|---|---|\n"
	const row = "| FR-001 | P0 | built | Original definition |\n"
	for name, files := range map[string]map[string]string{
		"same status table": {
			"docs/specs/04-fr.md": header + row + "| FR-001 | P0 | partial | Conflicting definition |\n",
		},
		"separate specification files": {
			"docs/specs/04-fr.md":           header + row,
			"docs/specs/05-architecture.md": header + "| FR-001 | P0 | partial | Conflicting definition |\n",
		},
	} {
		t.Run(name, func(t *testing.T) {
			root := t.TempDir()
			writeFiles(t, root, files)
			// Generation still produces a diagnostic report. Checking must reject
			// duplicates even after that exact report has been committed.
			if out, err := traceabilityCommand("-root", root, "-head", "fixture"); err != nil {
				t.Fatalf("generate: %v\n%s", err, out)
			}
			reportPath := filepath.Join(root, "docs", "TRACEABILITY.md")
			before, err := os.ReadFile(reportPath)
			if err != nil {
				t.Fatal(err)
			}
			out, err := traceabilityCommand("-root", root, "-check")
			if err == nil {
				t.Fatalf("check accepted a regenerated duplicate definition:\n%s", out)
			}
			if exit, ok := err.(*exec.ExitError); !ok || exit.ExitCode() == 0 {
				t.Fatalf("check did not return a nonzero exit status: %v\n%s", err, out)
			}
			for _, want := range []string{"duplicate requirement definitions", "FR-001"} {
				if !strings.Contains(string(out), want) {
					t.Errorf("check diagnostic lacks %q:\n%s", want, out)
				}
			}
			after, err := os.ReadFile(reportPath)
			if err != nil {
				t.Fatal(err)
			}
			if !bytes.Equal(before, after) {
				t.Fatal("check modified the diagnostic report")
			}
		})
	}
}

func TestCheckAllowsSharedIDsAndConsecutiveRuleRows(t *testing.T) {
	root := t.TempDir()
	writeFiles(t, root, map[string]string{
		"docs/specs/04-fr.md": "# Functional requirements\n" +
			"| ID | Priority | Status | Note |\n|---|---|---|---|\n" +
			"| FR-001 | P0 | built | One status definition |\n\n" +
			"### FR-001 · P0 · Create a network\n",
		"docs/specs/07-api.md": "# API\n" +
			"| ID | Operation | Role | Pri. | Acceptance |\n|---|---|---|---|---|\n" +
			"| FR-001 | Network creation | Member | P0 | Refers to the functional requirement |\n",
		"docs/specs/08-ux.md": "# Interface\n" +
			"| Rule | Requirement |\n|---|---|\n" +
			"| UX-004 · P0 | In credit |\n" +
			"| UX-004 · P0 | In debt |\n" +
			"| UX-004 · P0 | Zero balance |\n",
	})
	if out, err := traceabilityCommand("-root", root, "-head", "fixture"); err != nil {
		t.Fatalf("generate: %v\n%s", err, out)
	}
	if out, err := traceabilityCommand("-root", root, "-check"); err != nil {
		t.Fatalf("valid cross-reference and consecutive rule rows refused: %v\n%s", err, out)
	} else if !strings.Contains(string(out), "is up to date") {
		t.Fatalf("missing successful check diagnostic:\n%s", out)
	}
}
