package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestTerminalEventsRejectMalformedAndRepeatedResults(t *testing.T) {
	const pass = `{"Time":"2026-11-16T12:30:00Z","Action":"pass","Package":"example/domain","Test":"TestBalance"}`
	for name, src := range map[string]string{
		"malformed": `{`, "missing time": `{"Action":"pass","Package":"example/domain","Test":"TestBalance"}`,
		"duplicate": pass + "\n" + pass, "no terminal event": `{"Action":"run","Package":"example/domain","Test":"TestBalance"}`,
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := terminalEvents(strings.NewReader(src)); err == nil {
				t.Fatal("accepted invalid evidence")
			}
		})
	}
	events, err := terminalEvents(strings.NewReader(pass))
	if err != nil || len(events) != 1 || events[0].Action != "pass" {
		t.Fatalf("events=%+v err=%v", events, err)
	}
	e := &executionEvidence{Results: events}
	cites := []Cite{{Path: "domain/balance_test.go", Symbol: "TestBalance", Test: true}}
	if got := executionCell(cites, e, "example", 6); !strings.Contains(got, "pass 2026-11-16T12:30:00Z") {
		t.Fatal(got)
	}
	for _, cs := range [][]Cite{
		{{Path: "other/balance_test.go", Symbol: "TestBalance", Test: true}},
		{{Path: "domain/balance_test.go", Symbol: "helper", Test: true}},
		{{Path: "domain/balance_test.go", Test: true}},
		{{Path: "domain/balance.go", Symbol: "TestBalance"}},
	} {
		if got := executionCell(cs, e, "example", 6); got != "not recorded" {
			t.Fatalf("unexecuted citation qualified: %s", got)
		}
	}
	if got := executionCell(cites, nil, "example", 6); got != "not recorded" {
		t.Fatal(got)
	}
}

func evidenceFixture(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	writeFiles(t, root, map[string]string{
		"go.mod":              "module example\n\ngo 1.25\n",
		"docs/specs/04-fr.md": "# Functional\n| ID | Priority | Status | Note |\n|---|---|---|---|\n| FR-001 | P0 | built | TestPass |\n| FR-002 | P0 | partial | TestSkip |\n| FR-003 | P0 | built | TestFail |\n",
		"sample_test.go":      "package example\nimport \"testing\"\nfunc TestPass(t *testing.T) {}\nfunc TestSkip(t *testing.T) { t.Skip(\"manual gate remains\") }\nfunc TestFail(t *testing.T) {t.Fatal(\"intentional negative control\")}\n",
	})
	for _, args := range [][]string{{"init", "-q"}, {"config", "user.name", "Traceability Test"}, {"config", "user.email", "traceability@example.invalid"}, {"add", "."}, {"commit", "-qm", "fixture"}} {
		if b, err := gitOutput(root, args...); err != nil {
			t.Fatalf("git %v: %v %s", args, err, b)
		}
	}
	return root
}

func TestEvidenceRecorderUsesActualResultsAndRejectsStaleTree(t *testing.T) {
	root := evidenceFixture(t)
	output := filepath.Join(t.TempDir(), "results.json")
	if err := recordEvidence(root, output, []string{"-run", "TestPass|TestSkip", "./..."}); err != nil {
		t.Fatal(err)
	}
	e, err := readEvidence(root, output, "")
	if err != nil {
		t.Fatal(err)
	}
	if e.Dirty || e.ExitCode != 0 || len(e.Head) != 40 || len(e.SourceDigest) != 64 {
		t.Fatalf("bad provenance: %+v", e)
	}
	actions := map[string]string{}
	for _, ev := range e.Results {
		actions[ev.Test] = ev.Action
	}
	if actions["TestPass"] != "pass" || actions["TestSkip"] != "skip" || actions["TestFail"] != "" {
		t.Fatalf("wrong recorded results: %v", actions)
	}

	report, err := generate(options{root: root, specs: "docs/specs", decisions: "docs/DECISIONS.md", legacy: "legacy.tsv", head: e.Head, evidence: output, maxCites: 6})
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"`TestPass`: pass ", "`TestSkip`: skip ", "| FR-003 | P0 | - | built | - | `sample_test.go:TestFail` | other | not recorded", "partial"} {
		if !strings.Contains(string(report.out), want) {
			t.Fatalf("runtime report lacks %q", want)
		}
	}
	cs := []Cite{{Path: "sample_test.go", Symbol: "TestSkip", Test: true}}
	// Root package path is a valid citation too.
	if got := executionCell(cs, e, "example", 6); !strings.Contains(got, "skip ") {
		t.Fatal(got)
	}
	for _, arg := range []string{"-count=2", "-json=false", "-args"} {
		if err := recordEvidence(root, output, []string{arg, "./..."}); err == nil {
			t.Fatalf("accepted %s", arg)
		}
	}
	// A report-only change must not invalidate the source build fingerprint.
	writeFiles(t, root, map[string]string{"docs/TRACEABILITY.md": "generated inventory\n"})
	if _, err := readEvidence(root, output, ""); err != nil {
		t.Fatal(err)
	}
	writeFiles(t, root, map[string]string{"sample.go": "package example\nconst Changed=true\n"})
	if _, err := readEvidence(root, output, ""); err == nil {
		t.Fatal("stale evidence accepted after source change")
	}
	if err := recordEvidence(root, output, []string{"-run", "TestFail", "./..."}); err == nil {
		t.Fatal("failing tests reported success")
	}
	e, err = readEvidence(root, output, "")
	if err != nil {
		t.Fatal(err)
	}
	if !e.Dirty || e.ExitCode == 0 {
		t.Fatalf("failure provenance lost: %+v", e)
	}
	found := false
	for _, ev := range e.Results {
		if ev.Test == "TestFail" && ev.Action == "fail" {
			found = true
		}
	}
	if !found {
		t.Fatal("failure event not saved")
	}
	// Structural tampering is refused even when provenance matches the source.
	e.Results = append(e.Results, e.Results[0])
	b, err := json.Marshal(e)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(output, b, 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := readEvidence(root, output, ""); err == nil {
		t.Fatal("duplicate evidence accepted")
	}
}

func TestEvidenceRecorderRejectsSourceChangeDuringRun(t *testing.T) {
	root := evidenceFixture(t)
	writeFiles(t, root, map[string]string{"mutate_test.go": "package example\nimport (\"os\";\"testing\")\nfunc TestMutate(t *testing.T) { if err:=os.WriteFile(\"source.go\",[]byte(\"package example\\n\"),0600);err!=nil{t.Fatal(err)} }\n"})
	output := filepath.Join(t.TempDir(), "results.json")
	if err := recordEvidence(root, output, []string{"-run", "TestMutate", "./..."}); err == nil || !strings.Contains(err.Error(), "changed during tests") {
		t.Fatalf("err=%v", err)
	}
	if _, err := os.Stat(output); !os.IsNotExist(err) {
		t.Fatal("unstable build produced evidence")
	}
}

func TestEvidenceLayerClassification(t *testing.T) {
	cs := []Cite{{Path: "internal/service/a_test.go"}, {Path: "internal/contract/b_test.go"}, {Path: "cmd/circula/a_test.go"}, {Path: "e2e/tests/a.spec.js"}, {Path: "scripts/traceability/a_test.go"}}
	if got := testLayers(cs); got != "CLI, browser, contract, service, tooling" {
		t.Fatal(got)
	}
}

func TestFailedExecutionKeepsDiagnosticOutput(t *testing.T) {
	raw := `{"Action":"output","Package":"example","Test":"TestPass","Output":"success noise\n"}
{"Action":"output","Package":"example","Test":"TestFail","Output":"mismatched balance\n"}
{"Action":"output","Package":"example","Output":"FAIL example\n"}`
	var log strings.Builder
	writeFailureOutput([]byte(raw), []testEvent{{Package: "example", Test: "TestFail", Action: "fail"}, {Package: "example", Action: "fail"}}, &log)
	if log.String() != "mismatched balance\nFAIL example\n" {
		t.Fatal(log.String())
	}
}
