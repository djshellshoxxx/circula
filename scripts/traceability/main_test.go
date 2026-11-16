package main

import (
	"bytes"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestIDPattern(t *testing.T) {
	src := "LED-1 FR-082 API-062a OQ-FR-002 API-01 FR-0821 DOM-1 xFR-001 (UX-034): SA-DOM-010 QA-042."
	var got []string
	for _, loc := range findIDs([]byte(src)) {
		got = append(got, src[loc[0]:loc[1]])
	}
	want := []string{"LED-1", "FR-082", "API-062a", "UX-034", "QA-042"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("findIDs = %v, want %v", got, want)
	}
}

func TestParseSpecStatusTableAndBlocks(t *testing.T) {
	spec := strings.Join([]string{
		"# 04 Functional Requirements",
		"",
		"| ID | Priority | Status | Note |",
		"|---|---|---|---|",
		"| FR-001 | P0 | built | Create network; `TestCreate` covers it |",
		"| FR-002 | P1 | partial | Pipe \\| inside note |",
		"",
		"```",
		"### FR-099 · P0 · Inside a code fence",
		"```",
		"",
		"### FR-001 · P0 · Create a network",
		"### ARC-001 · Single binary · P1",
		"**SEC-001** · P0 · Password storage",
		"**QA-025 to QA-027** · P2 · Scenario range",
		"",
		"| Rule | Requirement |",
		"|---|---|",
		"| DAT-003 · P0 | Identifiers MUST NOT encode personal data. |",
		"| UX-034 · P0 (1.4.4, 1.4.12) | Text MUST remain readable. |",
		"",
		"- **UX-005 · P0 (no colour alone).** Direction MUST be conveyed by the sign.",
		"- **UX-006 · P1.** Accessible name spells out the sign.",
		"",
		"| ID | Operation | Role | Pri. | Acceptance |",
		"|---|---|---|---|---|",
		"| API-040 | `GET /network` | Member | P0 | Returns name |",
		"| D-01 | not a requirement | x | P0 | y |",
		"",
		"- DEC-001: Single binary.",
		"3. LED-3: Balance is derivable.",
	}, "\n")
	title, facts := parseSpec("04-fr.md", spec)
	if title != "04 Functional Requirements" {
		t.Errorf("title = %q", title)
	}
	type row struct{ ID, Kind, Pri, Status, Note, Title string }
	var got []row
	for _, f := range facts {
		got = append(got, row{f.ID, f.Kind, f.Pri, f.Status, f.Note, f.Title})
	}
	want := []row{
		{"FR-001", kindStatus, "P0", "built", "Create network; `TestCreate` covers it", ""},
		{"FR-002", kindStatus, "P1", "partial", "Pipe | inside note", ""},
		{"FR-001", kindBlock, "P0", "", "", "Create a network"},
		{"ARC-001", kindBlock, "P1", "", "", "Single binary"},
		{"SEC-001", kindBlock, "P0", "", "", "Password storage"},
		{"QA-025", kindBlock, "P2", "", "", "Scenario range"},
		{"QA-026", kindBlock, "P2", "", "", "Scenario range"},
		{"QA-027", kindBlock, "P2", "", "", "Scenario range"},
		{"DAT-003", kindBlock, "P0", "", "", "Identifiers MUST NOT encode personal data."},
		{"UX-034", kindBlock, "P0", "", "", "Text MUST remain readable."},
		{"UX-005", kindBlock, "P0", "", "", "no colour alone"},
		{"UX-006", kindBlock, "P1", "", "", "Accessible name spells out the sign."},
		{"API-040", kindTable, "P0", "", "", "`GET /network`"},
		{"DEC-001", kindList, "", "", "", "Single binary."},
		{"LED-3", kindList, "", "", "", "Balance is derivable."},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("facts:\n got %+v\nwant %+v", got, want)
	}
}

func TestBuildReqsMergesAndFindsDuplicates(t *testing.T) {
	facts := []fact{
		{ID: "FR-001", Kind: kindStatus, Pri: "P0", Status: "built", Note: "n", File: "04.md", Line: 3},
		{ID: "FR-001", Kind: kindBlock, Pri: "P1", Title: "Create", File: "04.md", Line: 40},
		{ID: "FR-002", Kind: kindStatus, Pri: "P0", Status: "partial", File: "04.md", Line: 4},
		{ID: "FR-002", Kind: kindStatus, Pri: "P0", Status: "built", File: "05.md", Line: 9},
		{ID: "UX-004", Kind: kindBlock, Pri: "P0", Title: "In credit", File: "08.md", Line: 146},
		{ID: "UX-004", Kind: kindBlock, Pri: "P0", Title: "In debt", File: "08.md", Line: 147},
		{ID: "API-040", Kind: kindTable, Pri: "P0", Title: "GET /network", File: "07.md", Line: 160},
		{ID: "DEC-001", Kind: kindStatus, Pri: "-", Status: "decided, built", File: "00.md", Line: 24},
		{ID: "DEC-001", Kind: kindTable, Title: "Single binary", File: "13.md", Line: 55},
		{ID: "DEC-001", Kind: kindList, Title: "ignored, table title wins", File: "00.md", Line: 81},
		{ID: "LED-9", Kind: kindList, Title: "list items alone define nothing", File: "00.md", Line: 90},
	}
	reqs, dups := buildReqs(facts)
	if r := reqs["FR-001"]; r.Pri != "P0" || r.Title != "Create" || r.Status != "built" || r.Spec != "04.md" {
		t.Errorf("FR-001 = %+v", r)
	}
	if r := reqs["FR-002"]; r.Status != "partial" || r.Spec != "04.md" {
		t.Errorf("FR-002 first status row should win: %+v", r)
	}
	if r := reqs["API-040"]; r.HasStatus || r.Title != "GET /network" || r.Spec != "07.md" {
		t.Errorf("API-040 = %+v", r)
	}
	if r := reqs["DEC-001"]; r.Title != "Single binary" || r.Spec != "00.md" {
		t.Errorf("DEC-001 = %+v", r)
	}
	if reqs["LED-9"] != nil {
		t.Error("a list item alone must not create a requirement")
	}
	if len(dups) != 1 || dups[0].ID != "FR-002" || len(dups[0].Locs) != 2 {
		t.Errorf("dups = %+v, want FR-002 only (UX-004 rows are consecutive)", dups)
	}
}

func TestIDLessAndStatusClass(t *testing.T) {
	ids := []string{"API-063", "API-062a", "API-062", "LED-10", "LED-2"}
	want := []string{"API-062", "API-062a", "API-063", "LED-2", "LED-10"}
	for i := range ids {
		for j := i + 1; j < len(ids); j++ {
			if idLess(ids[j], ids[i]) {
				ids[i], ids[j] = ids[j], ids[i]
			}
		}
	}
	if !reflect.DeepEqual(ids, want) {
		t.Errorf("order = %v", ids)
	}
	for in, want := range map[string]string{"built (2026-10-28, unreviewed)": "built", " Partial ": "partial", "": "-", "open, partly built": "open, partly built"} {
		if got := statusClass(in); got != want {
			t.Errorf("statusClass(%q) = %q, want %q", in, got, want)
		}
	}
}

func writeFiles(t *testing.T, root string, files map[string]string) {
	t.Helper()
	for p, c := range files {
		full := filepath.Join(root, filepath.FromSlash(p))
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(full, []byte(c), 0o644); err != nil {
			t.Fatal(err)
		}
	}
}

func TestScanResolvesSymbols(t *testing.T) {
	root := t.TempDir()
	writeFiles(t, root, map[string]string{
		"internal/x/x.go": "// Package x (FR-001 at file level).\npackage x\n\n// Post posts (FR-002).\nfunc Post() {\n\t_ = \"QA-003\"\n}\n\n" +
			"type T struct {\n\tA int // DAT-004\n}\n\nvar (\n\ta = 1\n\t// b is DAT-005.\n\tb = 2\n)\n",
		"internal/x/x_test.go":           "package x\n\nimport \"testing\"\n\nfunc TestUX002Links(t *testing.T) {\n\tt.Run(\"SEC-009 matrix\", func(t *testing.T) {})\n}\n",
		"e2e/tests/a.spec.js":            "// file comment GOV-001\ntest('UX-034: review', async () => {\n  // OPS-001\n});\n",
		"internal/store/m.sql":           "-- LED-2 append only\n",
		"docs/specs/ignored.go":          "package specs // FR-009\n",
		"scripts/traceability/sample.go": "package main // FR-010\n",
		".hidden/h.go":                   "package h // FR-011\n",
	})
	cites, funcs, err := scanTree(root)
	if err != nil {
		t.Fatal(err)
	}
	type c struct {
		ID, Path, Symbol string
		Test             bool
		Source           string
	}
	var got []c
	for _, x := range cites {
		got = append(got, c(x))
	}
	want := []c{
		{"GOV-001", "e2e/tests/a.spec.js", "", true, srcMention},
		{"UX-034", "e2e/tests/a.spec.js", "UX-034: review", true, srcMention},
		{"OPS-001", "e2e/tests/a.spec.js", "UX-034: review", true, srcMention},
		{"LED-2", "internal/store/m.sql", "", false, srcMention},
		{"FR-001", "internal/x/x.go", "", false, srcMention},
		{"FR-002", "internal/x/x.go", "Post", false, srcMention},
		{"QA-003", "internal/x/x.go", "Post", false, srcMention},
		{"DAT-004", "internal/x/x.go", "T", false, srcMention},
		{"DAT-005", "internal/x/x.go", "b", false, srcMention},
		{"SEC-009", "internal/x/x_test.go", "TestUX002Links", true, srcMention},
		{"UX-002", "internal/x/x_test.go", "TestUX002Links", true, srcName},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("cites:\n got %+v\nwant %+v", got, want)
	}
	if !reflect.DeepEqual(funcs["TestUX002Links"], []string{"internal/x/x_test.go"}) {
		t.Errorf("test func index = %v", funcs)
	}
}

func TestResolveLegacyDropsMissingTargets(t *testing.T) {
	root := t.TempDir()
	writeFiles(t, root, map[string]string{
		"a/a.go":      "package a\n\nfunc keep() {}\n",
		"a/a_test.go": "package a\n\nfunc TestKeep(t *testing.T) {}\n// TestGone is only a comment\n",
		"e2e/t.js":    "test('balance lookup', () => {})\n",
	})
	rows := parseLegacy("# header\nFR-001\tcode\ta/a.go\tkeep\nFR-001\tcode\ta/a.go\tgone\nFR-001\ttest\ta/a_test.go\tTestKeep\n" +
		"FR-001\ttest\ta/a_test.go\tTestGone\nFR-001\ttest\te2e/t.js\tbalance lookup\nFR-001\tcode\tmissing.go\t\nFR-002\tcode\ta/a.go\n")
	var got []string
	for _, c := range resolveLegacy(root, rows) {
		got = append(got, c.ID+" "+c.Path+" "+c.Symbol)
	}
	want := []string{"FR-001 a/a.go keep", "FR-001 a/a_test.go TestKeep", "FR-001 e2e/t.js balance lookup", "FR-002 a/a.go "}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("resolved = %q, want %q", got, want)
	}
}

func TestGenerateIsDeterministicAndFlags(t *testing.T) {
	root := t.TempDir()
	writeFiles(t, root, map[string]string{
		"docs/specs/04-fr.md": "# 04 Functional\n\n| ID | Priority | Status | Note |\n|---|---|---|---|\n" +
			"| FR-001 | P0 | built | covered by `TestNamed` |\n| FR-002 | P0 | built | nobody tests this |\n| FR-003 | P1 | partial | gap |\n" +
			"| FR-003 | P1 | partial | second row |\n\n### FR-001 · P0 · Create a network\n",
		"docs/specs/13-dec.md":                      "# 13 Decisions\n\n| ID | Priority | Status | Note |\n|---|---|---|---|\n| DEC-005 | - | open, not built | none |\n",
		"docs/DECISIONS.md":                         "### DEC-040 · Logged decision\n",
		"internal/a/a.go":                           "package a\n\n// Make implements FR-001 and cites the unknown FR-777 and the logged DEC-040.\nfunc Make() {}\n",
		"internal/a/a_test.go":                      "package a\n\nimport \"testing\"\n\nfunc TestNamed(t *testing.T) {}\n",
		"scripts/traceability/legacy-citations.tsv": "FR-003\ttest\tinternal/a/a_test.go\tTestNamed\n",
	})
	o := options{root: root, specs: "docs/specs", decisions: "docs/DECISIONS.md", legacy: "scripts/traceability/legacy-citations.tsv", head: "abc1234", maxCites: 6}
	a, err := generate(o)
	if err != nil {
		t.Fatal(err)
	}
	b, err := generate(o)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(a.out, b.out) {
		t.Fatal("two runs on the same tree differ")
	}
	if !reflect.DeepEqual(a.builtNoTest, []string{"FR-002"}) {
		t.Errorf("builtNoTest = %v", a.builtNoTest)
	}
	if a.builtNoNamed != 2 { // FR-001 is tested only through the spec note
		t.Errorf("builtNoNamed = %d, want 2", a.builtNoNamed)
	}
	if !reflect.DeepEqual(a.unknown, map[string][]string{"FR-777": {"internal/a/a.go"}}) {
		t.Errorf("unknown = %v", a.unknown)
	}
	if len(a.dups) != 1 || a.dups[0].ID != "FR-003" {
		t.Errorf("dups = %+v", a.dups)
	}
	out := string(a.out)
	for _, want := range []string{
		"Input: git HEAD `abc1234`",
		"| FR-001 | P0 | Create a network | built | `internal/a/a.go` (Make) | `internal/a/a_test.go:TestNamed` | a | not recorded | covered by `TestNamed` |",
		"| FR-003 | P1 | - | partial | - | `internal/a/a_test.go:TestNamed` † | a | not recorded | gap |",
		"## 13 Decisions",
		"| FR-777 | `internal/a/a.go` |",
		"Requirement IDs (excluding decision-register rows): 3; built 2, partial 1.",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("output lacks %q\n%s", want, out)
		}
	}
	if m := headLineRe.FindStringSubmatch(out); m == nil || m[1] != "abc1234" {
		t.Errorf("-check cannot read the head back: %v", m)
	}
	o.head = "def5678"
	c, _ := generate(o)
	if bytes.Equal(a.out, c.out) {
		t.Error("a different head must change the header")
	}
}

func TestSpecNoteRequiresExistingTestDeclaration(t *testing.T) {
	root := t.TempDir()
	writeFiles(t, root, map[string]string{
		"docs/specs/04-fr.md":                    "# Functional\n| ID | Priority | Status | Note |\n|---|---|---|---|\n| FR-001 | P0 | built | TestGone |\n| QA-042 | P0 | partial | TestGenerator |\n",
		"internal/a/a_test.go":                   "package a\n// TestGone is a removed test, not a declaration.\n",
		"scripts/traceability/generator_test.go": "package main\n// Sample FR-777 must never become a citation.\nfunc TestGenerator(t *testing.T) {}\n",
	})
	res, err := generate(options{root: root, specs: "docs/specs", decisions: "docs/DECISIONS.md", legacy: "legacy.tsv", head: "abc1234", maxCites: 6})
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(res.builtNoTest, []string{"FR-001"}) {
		t.Fatalf("missing test silently qualified: %v", res.builtNoTest)
	}
	if len(res.unknown) != 0 {
		t.Fatalf("generator sample leaked: %v", res.unknown)
	}
	if !strings.Contains(string(res.out), "`scripts/traceability/generator_test.go:TestGenerator` | tooling | not recorded") {
		t.Fatal("explicit generator citation absent")
	}
}
