// Command traceability writes docs/TRACEABILITY.md, the requirement traceability matrix (QA-042), from the
// specs and the source tree. Standard library only; the output depends only on the files read and on -head
// (no wall clock, no map order), so the same tree and -head give the same bytes.
//
//	go run ./scripts/traceability -head "$(git rev-parse --short HEAD)" -o docs/TRACEABILITY.md
//	go run ./scripts/traceability -check -o docs/TRACEABILITY.md
//
// Requirements (ID, priority, short title, status, note) come from docs/specs/[0-9]*.md: build status tables
// (| ID | Priority | Status | Note |), requirement headings and bold block lines (### FR-001 · P0 · Title,
// **SEC-001** · P0 · Title), rule rows (| DAT-003 · P0 | text |) and other tables whose first column is ID
// (07 operations). Citations are the files outside docs/ that name an ID (*.go with the enclosing declaration,
// *.sql, templates, scripts, JavaScript with the enclosing test title), Go test functions named after an ID
// (TestUX002...), test functions named in a status note, and the frozen hand-made citations of 2026-10-21 in
// legacy-citations.tsv while their targets still exist.
//
// -check regenerates with the head recorded in the existing file and exits 1 if the bytes differ, so a
// committed matrix stays checkable after the commit that adds it.
package main

import (
	"bytes"
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

type options struct {
	root, specs, decisions, legacy, head string
	evidence, report                     string
	maxCites                             int
}

// result is the generated matrix plus the flag lists used for the report and the tests.
type result struct {
	out          []byte
	reqs         []*Req
	builtNoTest  []string
	builtNoNamed int
	unknown      map[string][]string
	dups         []Duplicate
	legacyKept   int
	legacyTotal  int
	evidence     *executionEvidence
	module       string
}

var headLineRe = regexp.MustCompile("(?m)^Input: git HEAD `([^`]+)`")

func main() {
	var o options
	var out string
	var check bool
	var record string
	flag.StringVar(&o.root, "root", ".", "repository root")
	flag.StringVar(&o.specs, "specs", "docs/specs", "spec directory, relative to -root")
	flag.StringVar(&o.decisions, "decisions", "docs/DECISIONS.md", "decision log whose DEC headings count as known IDs, relative to -root")
	flag.StringVar(&o.legacy, "legacy", "scripts/traceability/legacy-citations.tsv", "frozen hand-made citations, relative to -root")
	flag.StringVar(&o.head, "head", "", "git HEAD short SHA the inputs were read at (recorded in the header)")
	flag.IntVar(&o.maxCites, "max-cites", 6, "citations shown per cell before \"+N more\"")
	flag.StringVar(&out, "o", "docs/TRACEABILITY.md", "output file, relative to -root")
	flag.BoolVar(&check, "check", false, "regenerate with the head recorded in -o and fail if the file differs")
	flag.StringVar(&o.evidence, "evidence", "", "explicit execution evidence JSON; omit for deterministic citation-only inventory")
	flag.StringVar(&record, "record-results", "", "record actual go test results and provenance here; pass test arguments after --")
	flag.Parse()
	o.report = out
	if record != "" {
		if err := recordEvidence(o.root, record, flag.Args()); err != nil {
			fatal(err)
		}
		return
	}
	if check && o.evidence != "" {
		fatal(errors.New("-check is for the committed inventory; omit -evidence"))
	}
	outPath := out
	if !filepath.IsAbs(out) {
		outPath = filepath.Join(o.root, filepath.FromSlash(out))
	}
	if check {
		old, err := os.ReadFile(outPath)
		if err != nil {
			fatal(err)
		}
		m := headLineRe.FindSubmatch(old)
		if m == nil {
			fatal(fmt.Errorf("%s has no \"Input: git HEAD\" line; run make traceability", out))
		}
		o.head = string(m[1])
		res, err := generate(o)
		if err != nil {
			fatal(err)
		}
		if !bytes.Equal(old, res.out) {
			fmt.Fprintf(os.Stderr, "traceability: %s is out of date (%s); run make traceability\n", out, firstDiff(old, res.out))
			os.Exit(1)
		}
		fmt.Fprintf(os.Stderr, "traceability: %s is up to date\n", out)
		return
	}
	if o.head == "" {
		fatal(errors.New("-head is required (git rev-parse --short HEAD)"))
	}
	res, err := generate(o)
	if err != nil {
		fatal(err)
	}
	if err := os.WriteFile(outPath, res.out, 0o644); err != nil { //nolint:gosec // developer tool: -o is the path the person running it chose
		fatal(err)
	}
	fmt.Fprintf(os.Stderr, "traceability: wrote %s: %d IDs, %d built without test citation, %d unknown IDs, %d duplicates, %d of %d legacy citations still resolve\n",
		out, len(res.reqs), len(res.builtNoTest), len(res.unknown), len(res.dups), res.legacyKept, res.legacyTotal)
}

func fatal(err error) {
	fmt.Fprintln(os.Stderr, "traceability:", err)
	os.Exit(2)
}

func firstDiff(a, b []byte) string {
	al, bl := strings.Split(string(a), "\n"), strings.Split(string(b), "\n")
	for i := 0; i < len(al) || i < len(bl); i++ {
		if i >= len(al) || i >= len(bl) || al[i] != bl[i] {
			want := ""
			if i < len(bl) {
				want = shorten(bl[i], 120)
			}
			return fmt.Sprintf("first difference at line %d, expected %q", i+1, want)
		}
	}
	return "no line difference"
}

type specFile struct {
	name, title string
}

func generate(o options) (*result, error) {
	evidence, err := readEvidence(o.root, o.evidence, o.report)
	if err != nil {
		return nil, err
	}
	module := ""
	if evidence != nil {
		mod, err := os.ReadFile(filepath.Join(o.root, "go.mod"))
		if err != nil {
			return nil, err
		}
		for _, line := range strings.Split(string(mod), "\n") {
			fields := strings.Fields(line)
			if len(fields) == 2 && fields[0] == "module" {
				module = fields[1]
				break
			}
		}
		if module == "" {
			return nil, errors.New("go.mod has no module path")
		}
	}
	specDir := filepath.Join(o.root, filepath.FromSlash(o.specs))
	paths, err := filepath.Glob(filepath.Join(specDir, "[0-9]*.md"))
	if err != nil {
		return nil, err
	}
	if len(paths) == 0 {
		return nil, fmt.Errorf("no spec files in %s", specDir)
	}
	sort.Strings(paths)
	var specs []specFile
	var facts []fact
	for _, p := range paths {
		b, err := os.ReadFile(p)
		if err != nil {
			return nil, err
		}
		name := filepath.Base(p)
		title, fs := parseSpec(name, string(b))
		specs = append(specs, specFile{name, title})
		facts = append(facts, fs...)
	}
	reqMap, dups := buildReqs(facts)

	known := map[string]bool{}
	if b, err := os.ReadFile(filepath.Join(o.root, filepath.FromSlash(o.decisions))); err == nil {
		known = decisionIDs(string(b))
	} else if !errors.Is(err, os.ErrNotExist) {
		return nil, err
	}

	cites, testFuncs, err := scanTree(o.root)
	if err != nil {
		return nil, err
	}
	for _, r := range sortedReqs(reqMap) {
		for _, name := range uniqueStrings(testNameRe.FindAllString(r.Note, -1)) {
			for _, p := range testFuncs[name] {
				cites = append(cites, Cite{ID: r.ID, Path: p, Symbol: name, Test: true, Source: srcSpec})
			}
		}
	}
	legacyKept, legacyTotal := 0, 0
	if b, err := os.ReadFile(filepath.Join(o.root, filepath.FromSlash(o.legacy))); err == nil {
		rows := parseLegacy(string(b))
		kept := resolveLegacy(o.root, rows)
		legacyKept, legacyTotal = len(kept), len(rows)
		cites = append(cites, kept...)
	} else if !errors.Is(err, os.ErrNotExist) {
		return nil, err
	}

	unknownSet := map[string]map[string]bool{}
	type key struct {
		id, path, sym string
		test          bool
	}
	merged := map[key]map[string]bool{}
	for _, c := range cites {
		if reqMap[c.ID] == nil {
			if c.Source == srcMention && !known[c.ID] {
				if unknownSet[c.ID] == nil {
					unknownSet[c.ID] = map[string]bool{}
				}
				unknownSet[c.ID][c.Path] = true
			}
			continue
		}
		k := key{c.ID, c.Path, c.Symbol, c.Test}
		if merged[k] == nil {
			merged[k] = map[string]bool{}
		}
		merged[k][c.Source] = true
	}
	// A bare file citation adds nothing when the same file is cited with a symbol for the same ID.
	withSym := map[key]bool{}
	for k := range merged {
		if k.sym != "" {
			withSym[key{k.id, k.path, "", k.test}] = true
		}
	}
	for k, srcs := range merged {
		if k.sym == "" && withSym[k] {
			continue
		}
		c := Cite{ID: k.id, Path: k.path, Symbol: k.sym, Test: k.test, Source: srcMention}
		if len(srcs) == 1 && srcs[srcLegacy] {
			c.Source = srcLegacy
		} else if !srcs[srcMention] && !srcs[srcName] && srcs[srcSpec] {
			c.Source = srcSpec
		}
		r := reqMap[k.id]
		if k.test {
			r.Tests = append(r.Tests, c)
		} else {
			r.Code = append(r.Code, c)
		}
	}
	reqs := sortedReqs(reqMap)
	res := &result{reqs: reqs, dups: dups, unknown: map[string][]string{}, legacyKept: legacyKept, legacyTotal: legacyTotal, evidence: evidence, module: module}
	for _, r := range reqs {
		sortCites(r.Code)
		sortCites(r.Tests)
		if isDecision(r.ID) || statusClass(r.Status) != "built" {
			continue
		}
		if len(r.Tests) == 0 {
			res.builtNoTest = append(res.builtNoTest, r.ID)
		}
		named := false
		for _, c := range r.Tests {
			if c.Source != srcLegacy && c.Source != srcSpec {
				named = true
			}
		}
		if !named {
			res.builtNoNamed++
		}
	}
	for id, ps := range unknownSet {
		res.unknown[id] = sortedKeys(ps)
	}
	res.out = render(o, specs, reqs, res)
	return res, nil
}

func sortedReqs(m map[string]*Req) []*Req {
	out := make([]*Req, 0, len(m))
	for _, r := range m {
		out = append(out, r)
	}
	sort.Slice(out, func(i, j int) bool {
		a, b := out[i], out[j]
		if a.Spec != b.Spec {
			return a.Spec < b.Spec
		}
		if a.line != b.line {
			return a.line < b.line
		}
		return idLess(a.ID, b.ID)
	})
	return out
}

func sortCites(cs []Cite) {
	sort.Slice(cs, func(i, j int) bool {
		a, b := cs[i], cs[j]
		if (a.Source == srcLegacy) != (b.Source == srcLegacy) {
			return b.Source == srcLegacy
		}
		if a.Path != b.Path {
			return a.Path < b.Path
		}
		return a.Symbol < b.Symbol
	})
}

func sortedKeys(m map[string]bool) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

func uniqueStrings(in []string) []string {
	seen := map[string]bool{}
	var out []string
	for _, s := range in {
		if !seen[s] {
			seen[s] = true
			out = append(out, s)
		}
	}
	return out
}

func esc(s string) string {
	s = strings.ReplaceAll(s, "|", `\|`)
	if strings.TrimSpace(s) == "" {
		return "-"
	}
	return s
}

func fmtCite(c Cite) string {
	var s string
	switch {
	case c.Symbol == "":
		s = "`" + c.Path + "`"
	case c.Test && strings.HasSuffix(c.Path, ".go"):
		s = "`" + c.Path + ":" + c.Symbol + "`"
	default:
		s = "`" + c.Path + "` (" + c.Symbol + ")"
	}
	if c.Source == srcLegacy {
		s += " †"
	}
	return s
}

func citeCell(cs []Cite, max int) string {
	if len(cs) == 0 {
		return "-"
	}
	var parts []string
	for i, c := range cs {
		if i == max {
			parts = append(parts, fmt.Sprintf("+%d more", len(cs)-max))
			break
		}
		parts = append(parts, fmtCite(c))
	}
	return strings.ReplaceAll(strings.Join(parts, "<br>"), "|", `\|`)
}

// countLine renders status counts in a fixed order: the usual requirement states, then the rest by name.
func countLine(counts map[string]int) string {
	order := []string{"built", "partial", "not built", "deferred"}
	seen := map[string]bool{}
	var parts []string
	for _, k := range order {
		if counts[k] > 0 {
			parts = append(parts, fmt.Sprintf("%s %d", k, counts[k]))
		}
		seen[k] = true
	}
	var rest []string
	for k := range counts {
		if !seen[k] {
			rest = append(rest, k)
		}
	}
	sort.Strings(rest)
	for _, k := range rest {
		parts = append(parts, fmt.Sprintf("%s %d", k, counts[k]))
	}
	if len(parts) == 0 {
		return "none"
	}
	return strings.Join(parts, ", ")
}

func statusLabel(s string) string {
	if s == "" {
		return "no status row"
	}
	return statusClass(s)
}

func render(o options, specs []specFile, reqs []*Req, res *result) []byte {
	var b bytes.Buffer
	w := func(format string, a ...any) { fmt.Fprintf(&b, format, a...) }
	reqCounts, decCounts := map[string]int{}, map[string]int{}
	withCode, withTest := 0, 0
	for _, r := range reqs {
		if isDecision(r.ID) {
			decCounts[statusLabel(r.Status)]++
		} else {
			reqCounts[statusLabel(r.Status)]++
		}
		if len(r.Code) > 0 {
			withCode++
		}
		if len(r.Tests) > 0 {
			withTest++
		}
	}

	w("# Traceability matrix\n\n")
	w("Generated file (QA-042); do not edit by hand. Regenerate with `make traceability`, which runs `go run ./scripts/traceability -head \"$(git rev-parse --short HEAD)\" -o docs/TRACEABILITY.md`. `make traceability-check` fails when this file does not match the specs and code.\n\n")
	w("Input: git HEAD `%s` (base commit supplied by the caller; this inventory can include working-tree edits and is not a tested-build claim). Without -evidence the generator reads no clock: the same tree and head give the same file.\n\n", o.head)
	w("Depends on: the spec files in [specs/](specs/), [DECISIONS.md](DECISIONS.md) (DEC-028 onward count as known IDs), the source tree outside docs/, and the frozen citations in `scripts/traceability/legacy-citations.tsv`.\n\n")

	if res.evidence != nil {
		e := res.evidence
		w("Execution evidence: build `%s`, dirty tree `%t`, source SHA-256 `%s`, exit code `%d`. Command: `%s`. Dates below are actual terminal test-event timestamps; results describe cited tests, not whole-requirement acceptance.\n\n", e.Head, e.Dirty, e.SourceDigest, e.ExitCode, esc(strings.Join(e.Command, " ")))
	} else {
		w("Execution evidence: not supplied. This committed inventory contains no last-run or build qualification claims. Generate a separate runtime report with `-evidence`; do not replace this inventory with runtime results.\n\n")
	}
	w("## How to read this file\n\n")
	w("- **ID, Pri, Requirement (short)**: from the spec's requirement heading, bold block line or rule row, else from the ID table row (07 operations). Titles are cut at 90 characters.\n")
	w("- **Status, Note**: the spec's build status table, verbatim. `-` where the spec has no status row (the 07 API IDs: their status lives in 07 section 12 prose, not in a table).\n")
	w("- **Code** and **Tests**: `path` (Symbol) for code, `path:TestName` for Go tests, `path` (test title) for browser tests. A file is cited when it names the ID in a comment, string, route comment or test title (Symbol is the enclosing Go declaration or the nearest preceding JavaScript test title), when a Go test function carries the ID in its name (`TestUX002...` cites UX-002), or when the spec's status note names the test function.\n")
	w("- **Layer**: test location (domain, service, store, API/web, contract, CLI, browser or tooling); this is not a claim about every layer covered. **Last recorded test result**: terminal events only from explicit matching execution evidence; `not recorded` means unqualified. A passing cited test does not establish whole-requirement acceptance; manual and external gates remain in the notes.\n")
	w("- **†**: citation from the hand-made matrix of 2026-10-21 (HEAD `925f637`), kept only while its file and symbol still exist. It was matched by meaning, not by the ID string.\n")
	w("- A citation shows that the code or test exists and refers to the requirement. It does not show that the test passes, when it last ran, or that it covers the whole requirement. Cells list at most %d citations, then \"+N more\".\n", o.maxCites)
	w("- Scanned: `*.go`, `*.sql`, `*.html`, `*.tmpl`, `*.gohtml`, `*.sh`, `*.py`, `*.js` outside `docs/`, `prompts/`, hidden directories, `node_modules/`, `bin/`, `testdata/`, `vendor/` and `scripts/traceability/`.\n\n")

	w("## Status values\n\n")
	w("- **built**: behaviour exists in the code (the spec's own claim; see the spec status note).\n")
	w("- **partial**: exists with the gaps stated in the note.\n")
	w("- **not built**: no implementation.\n")
	w("- **deferred**: out of MVP by the spec or by a recorded decision.\n")
	w("- Decision-register rows (DEC-001 to DEC-004 in 00, DEC-005 to DEC-027 in 13) use decision states: decided, open or deferred, with the build state after the comma. The owner decisions are in [DECISIONS.md](DECISIONS.md).\n\n")

	w("## Summary\n\n")
	w("Requirement IDs (excluding decision-register rows): %d; %s.\n\n", len(reqs)-sumCounts(decCounts), countLine(reqCounts))
	w("Decision-register IDs: %d; %s.\n\n", sumCounts(decCounts), countLine(decCounts))
	w("| Measure | Count |\n|---|---|\n")
	w("| IDs in the matrix | %d |\n", len(reqs))
	w("| IDs with at least one code citation | %d |\n", withCode)
	w("| IDs with at least one test citation | %d |\n", withTest)
	w("| Built requirements with no test citation of any kind | %d |\n", len(res.builtNoTest))
	w("| Built requirements with no test that names the ID (legacy and spec-note citations not counted) | %d |\n", res.builtNoNamed)
	w("| IDs cited in code but not defined in the specs or DECISIONS.md | %d |\n", len(res.unknown))
	w("| IDs defined more than once | %d |\n\n", len(res.dups))

	bySpec := map[string][]*Req{}
	for _, r := range reqs {
		bySpec[r.Spec] = append(bySpec[r.Spec], r)
	}
	for _, s := range specs {
		rs := bySpec[s.name]
		if len(rs) == 0 {
			continue
		}
		counts := map[string]int{}
		for _, r := range rs {
			counts[statusLabel(r.Status)]++
		}
		title := s.title
		if title == "" {
			title = s.name
		}
		w("## %s\n\n", title)
		w("Source: [%s](specs/%s) · IDs: %d · %s\n\n", s.name, s.name, len(rs), countLine(counts))
		w("| ID | Pri | Requirement (short) | Status | Code | Tests | Layer | Last recorded test result | Note |\n|---|---|---|---|---|---|---|---|---|\n")
		for _, r := range rs {
			w("| %s | %s | %s | %s | %s | %s | %s | %s | %s |\n", r.ID, esc(r.Pri), esc(shorten(r.Title, 90)), esc(r.Status),
				citeCell(r.Code, o.maxCites), citeCell(r.Tests, o.maxCites), testLayers(r.Tests), executionCell(r.Tests, res.evidence, res.module, o.maxCites), esc(r.Note))
		}
		w("\n")
	}

	w("## Flags\n\n")
	w("### Built requirements with no test citation (%d)\n\n", len(res.builtNoTest))
	if len(res.builtNoTest) == 0 {
		w("None.\n\n")
	} else {
		w("Status built in the spec, and no test file names the ID, carries it in a test name, is named in the status note, or was cited by the 2026-10-21 matrix.\n\n")
		w("%s\n\n", strings.Join(res.builtNoTest, ", "))
	}
	unknownIDs := make([]string, 0, len(res.unknown))
	for id := range res.unknown {
		unknownIDs = append(unknownIDs, id)
	}
	sort.Slice(unknownIDs, func(i, j int) bool { return idLess(unknownIDs[i], unknownIDs[j]) })
	w("### IDs cited in code but not defined in the specs (%d)\n\n", len(unknownIDs))
	if len(unknownIDs) == 0 {
		w("None.\n\n")
	} else {
		w("The ID string appears in a scanned file, but no spec status table, requirement block or ID table defines it and docs/DECISIONS.md has no heading for it. Either the citation is wrong or the spec lacks the requirement.\n\n")
		w("| ID | Cited in |\n|---|---|\n")
		for _, id := range unknownIDs {
			ps := res.unknown[id]
			var cells []string
			for i, p := range ps {
				if i == o.maxCites {
					cells = append(cells, fmt.Sprintf("+%d more", len(ps)-o.maxCites))
					break
				}
				cells = append(cells, "`"+p+"`")
			}
			w("| %s | %s |\n", id, strings.Join(cells, "<br>"))
		}
		w("\n")
	}
	w("### IDs defined more than once (%d)\n\n", len(res.dups))
	if len(res.dups) == 0 {
		w("None.\n")
	} else {
		w("More than one build status row, or more than one requirement block, for the same ID. The first one in file order is used above.\n\n")
		w("| ID | Definitions |\n|---|---|\n")
		for _, d := range res.dups {
			var cells []string
			for _, l := range d.Locs {
				cells = append(cells, fmt.Sprintf("`%s:%d` (%s)", l.File, l.Line, l.Kind))
			}
			w("| %s | %s |\n", d.ID, strings.Join(cells, "<br>"))
		}
	}
	return b.Bytes()
}

func sumCounts(m map[string]int) int {
	n := 0
	for _, v := range m {
		n += v
	}
	return n
}
