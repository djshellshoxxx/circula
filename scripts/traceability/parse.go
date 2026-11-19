package main

import (
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

// prefixes are the requirement ID families defined in docs/specs (00 section 2 and each spec's scope line).
var prefixes = []string{"LED", "VIS", "PER", "DOM", "FR", "ARC", "DAT", "API", "UX", "SEC", "GOV", "OPS", "QA", "DEC"}

var (
	// LED invariants are numbered 1 to 7; every other family uses three digits (00 section 2). Two-digit forms
	// such as API-01 or OQ-FR-2 are audit finding and open-question labels, not requirement IDs.
	idPat       = `(?:LED-[0-9]{1,2}|(?:` + strings.Join(prefixes[1:], "|") + `)-[0-9]{3}[a-z]?)`
	idRe        = regexp.MustCompile(`\b` + idPat + `\b`)
	idFullRe    = regexp.MustCompile(`^` + idPat + `$`)
	priRe       = regexp.MustCompile(`\bP[0-3]\b`)
	priOnlyRe   = regexp.MustCompile(`^P[0-3]$`)
	sepRowRe    = regexp.MustCompile(`^\|[\s:|-]+\|?$`)
	ruleCellRe  = regexp.MustCompile(`^\*{0,2}(` + idPat + `)\*{0,2}\s*·\s*\*{0,2}(P[0-3])\*{0,2}(?:\s+\(.*\))?$`)
	bulletRe    = regexp.MustCompile(`^\s*[-*]\s+\*\*(` + idPat + `)\s*·\s*(P[0-3])(?:\s*\(([^)]*)\))?\.?\*\*\s*(.*)$`)
	blockRe     = regexp.MustCompile(`^(?:#{2,6}\s+|\*\*)(` + idPat + `)(?:\s+to\s+(` + idPat + `))?(?:\*\*)?\s*·\s*(.+)$`)
	listRe      = regexp.MustCompile(`^\s*(?:[-*]|[0-9]+\.)\s+(` + idPat + `):\s+(.+)$`)
	decHeadRe   = regexp.MustCompile(`^#{2,6}\s+(DEC-[0-9]{3}[a-z]?)\b`)
	testNameRe  = regexp.MustCompile(`\bTest[A-Z][A-Za-z0-9_]*`)
	h1Re        = regexp.MustCompile(`^#\s+(.+)$`)
	idNumSplitR = regexp.MustCompile(`^([A-Z]+)-([0-9]+)$`)
)

// Fact kinds, in the order they win when several sources describe one ID.
const (
	kindStatus = "status" // row of a "| ID | Priority | Status | Note |" build status table
	kindBlock  = "block"  // requirement heading, bold block line or "| ID · P0 | text |" rule row
	kindTable  = "table"  // row of another table whose first column is ID (07 operations, 12 scenarios)
	kindList   = "list"   // "- ID: text" or "3. ID: text" list item (title only)
)

// fact is one place in a spec file that says something about a requirement ID.
type fact struct {
	ID, Kind          string
	Pri, Status, Note string
	Title             string
	File              string
	Line              int
}

type tableCols struct {
	isID                     bool
	pri, status, note, title int
}

func analyseHeader(h []string) tableCols {
	c := tableCols{pri: -1, status: -1, note: -1, title: -1}
	if len(h) == 0 || !strings.EqualFold(strings.Trim(h[0], "* "), "ID") {
		return c
	}
	c.isID = true
	for i := 1; i < len(h); i++ {
		switch strings.ToLower(strings.TrimSuffix(strings.TrimSpace(h[i]), ".")) {
		case "pri", "priority":
			c.pri = i
		case "status":
			c.status = i
		case "note":
			c.note = i
		}
	}
	for i := 1; i < len(h); i++ {
		if i != c.pri && i != c.status && i != c.note {
			c.title = i
			break
		}
	}
	return c
}

func (c tableCols) statusTable() bool { return c.isID && c.pri >= 0 && c.status >= 0 }

// splitRow splits a Markdown table row into trimmed cells. An escaped pipe (\|) stays in its cell as "|".
func splitRow(line string) []string {
	s := strings.TrimSpace(line)
	s = strings.TrimPrefix(s, "|")
	if strings.HasSuffix(s, "|") && !strings.HasSuffix(s, `\|`) {
		s = strings.TrimSuffix(s, "|")
	}
	var cells []string
	var b strings.Builder
	for i := 0; i < len(s); i++ {
		switch {
		case s[i] == '\\' && i+1 < len(s) && s[i+1] == '|':
			b.WriteByte('|')
			i++
		case s[i] == '|':
			cells = append(cells, strings.TrimSpace(b.String()))
			b.Reset()
		default:
			b.WriteByte(s[i])
		}
	}
	return append(cells, strings.TrimSpace(b.String()))
}

func cell(cells []string, i int) string {
	if i < 0 || i >= len(cells) {
		return ""
	}
	return cells[i]
}

// expandRange turns "QA-025 to QA-026" into both IDs and every ID between them (same prefix, at most 50).
func expandRange(from, to string) []string {
	if to == "" {
		return []string{from}
	}
	a, b := idNumSplitR.FindStringSubmatch(from), idNumSplitR.FindStringSubmatch(to)
	if a == nil || b == nil || a[1] != b[1] {
		return []string{from, to}
	}
	x, _ := strconv.Atoi(a[2])
	y, _ := strconv.Atoi(b[2])
	if y < x || y-x > 50 {
		return []string{from, to}
	}
	out := make([]string, 0, y-x+1)
	for n := x; n <= y; n++ {
		out = append(out, fmt.Sprintf("%s-%0*d", a[1], len(a[2]), n))
	}
	return out
}

// parseBlockRest reads "P0 · Title" or "Title · P0" after the ID of a block line.
func parseBlockRest(rest string) (pri, title string) {
	var parts []string
	for _, p := range strings.Split(rest, "·") {
		p = strings.TrimSpace(strings.Trim(strings.TrimSpace(p), "*"))
		if p == "" {
			continue
		}
		if pri == "" && priOnlyRe.MatchString(p) {
			pri = p
			continue
		}
		parts = append(parts, p)
	}
	return pri, strings.Join(parts, " · ")
}

// parseSpec extracts the H1 title and every requirement fact from one spec file.
func parseSpec(file, src string) (title string, facts []fact) {
	lines := strings.Split(strings.ReplaceAll(src, "\r\n", "\n"), "\n")
	inTable, inFence := false, false
	var cols tableCols
	for n, line := range lines {
		ln := n + 1
		trim := strings.TrimSpace(line)
		if strings.HasPrefix(trim, "```") {
			inFence = !inFence
			inTable = false
			continue
		}
		if inFence {
			continue
		}
		if title == "" {
			if m := h1Re.FindStringSubmatch(line); m != nil {
				title = strings.TrimSpace(m[1])
			}
		}
		if strings.HasPrefix(trim, "|") {
			cells := splitRow(trim)
			if !inTable {
				inTable = true
				cols = analyseHeader(cells)
				// A header row can itself be a rule row in a table without a header (not used by the specs today).
				if m := ruleCellRe.FindStringSubmatch(cell(cells, 0)); m != nil {
					facts = append(facts, fact{ID: m[1], Kind: kindBlock, Pri: m[2], Title: cell(cells, 1), File: file, Line: ln})
				}
				continue
			}
			if sepRowRe.MatchString(trim) {
				continue
			}
			c0 := strings.Trim(cell(cells, 0), "* ")
			if m := ruleCellRe.FindStringSubmatch(cell(cells, 0)); m != nil {
				facts = append(facts, fact{ID: m[1], Kind: kindBlock, Pri: m[2], Title: cell(cells, 1), File: file, Line: ln})
				continue
			}
			if !cols.isID || !idFullRe.MatchString(c0) {
				continue
			}
			if cols.statusTable() {
				facts = append(facts, fact{ID: c0, Kind: kindStatus, Pri: cell(cells, cols.pri), Status: cell(cells, cols.status),
					Note: cell(cells, cols.note), File: file, Line: ln})
				continue
			}
			facts = append(facts, fact{ID: c0, Kind: kindTable, Pri: priRe.FindString(cell(cells, cols.pri)),
				Title: cell(cells, cols.title), File: file, Line: ln})
			continue
		}
		inTable = false
		if m := blockRe.FindStringSubmatch(trim); m != nil {
			pri, t := parseBlockRest(m[3])
			for _, id := range expandRange(m[1], m[2]) {
				facts = append(facts, fact{ID: id, Kind: kindBlock, Pri: pri, Title: t, File: file, Line: ln})
			}
			continue
		}
		if m := bulletRe.FindStringSubmatch(line); m != nil {
			t := m[3] // "- **UX-005 · P0 (no colour alone).** text": the parenthetical is the short title
			if t == "" {
				t = m[4]
			}
			facts = append(facts, fact{ID: m[1], Kind: kindBlock, Pri: m[2], Title: strings.TrimSpace(t), File: file, Line: ln})
			continue
		}
		if m := listRe.FindStringSubmatch(line); m != nil {
			facts = append(facts, fact{ID: m[1], Kind: kindList, Title: strings.TrimSpace(m[2]), File: file, Line: ln})
		}
	}
	return title, facts
}

// Loc is a definition site of an ID in a spec file.
type Loc struct {
	File string
	Line int
	Kind string
}

// Req is one requirement row of the matrix.
type Req struct {
	ID, Pri, Title, Status, Note string
	HasStatus                    bool
	Spec                         string // spec file that owns the row (status table first, then block, then table)
	line                         int
	Defs                         []Loc // status rows and blocks; more than one of a kind is a duplicate
	Code, Tests                  []Cite
}

// Duplicate is an ID defined more than once by the same kind of source.
type Duplicate struct {
	ID   string
	Locs []Loc
}

// buildReqs merges facts (from files parsed in sorted order) into requirements.
func buildReqs(facts []fact) (map[string]*Req, []Duplicate) {
	reqs := map[string]*Req{}
	defs := map[string]map[string][]Loc{}
	addDef := func(f fact) {
		if defs[f.ID] == nil {
			defs[f.ID] = map[string][]Loc{}
		}
		defs[f.ID][f.Kind] = append(defs[f.ID][f.Kind], Loc{File: f.File, Line: f.Line, Kind: f.Kind})
	}
	get := func(f fact) *Req {
		r := reqs[f.ID]
		if r == nil {
			r = &Req{ID: f.ID, Spec: f.File, line: f.Line}
			reqs[f.ID] = r
		}
		return r
	}
	for _, kind := range []string{kindStatus, kindBlock, kindTable, kindList} {
		for _, f := range facts {
			if f.Kind != kind {
				continue
			}
			if kind == kindList && reqs[f.ID] == nil {
				continue // list items name a title; they do not define an ID on their own
			}
			r := get(f)
			switch kind {
			case kindStatus:
				addDef(f)
				if !r.HasStatus {
					r.HasStatus, r.Pri, r.Status, r.Note = true, f.Pri, f.Status, f.Note
				}
			case kindBlock:
				addDef(f)
				if len(defs[f.ID][kindBlock]) == 1 {
					if f.Title != "" {
						r.Title = f.Title
					}
					if r.Pri == "" || r.Pri == "-" {
						if f.Pri != "" {
							r.Pri = f.Pri
						}
					}
				}
			default:
				if r.Title == "" {
					r.Title = f.Title
				}
				if r.Pri == "" && f.Pri != "" {
					r.Pri = f.Pri
				}
			}
		}
	}
	var dups []Duplicate
	for id, byKind := range defs {
		var locs []Loc
		if len(byKind[kindStatus]) > 1 {
			locs = append(locs, byKind[kindStatus]...)
		}
		if distinctDefs(byKind[kindBlock]) > 1 {
			locs = append(locs, byKind[kindBlock]...)
		}
		if len(locs) > 0 {
			dups = append(dups, Duplicate{ID: id, Locs: locs})
		}
		for _, k := range []string{kindStatus, kindBlock} {
			reqs[id].Defs = append(reqs[id].Defs, byKind[k]...)
		}
	}
	sort.Slice(dups, func(i, j int) bool { return idLess(dups[i].ID, dups[j].ID) })
	return reqs, dups
}

// distinctDefs counts definition sites, treating rows on consecutive lines of one file as one site (UX-004 is
// three rule rows of one table: one per balance label).
func distinctDefs(locs []Loc) int {
	n := 0
	for i, l := range locs {
		if i > 0 && locs[i-1].File == l.File && locs[i-1].Line+1 == l.Line {
			continue
		}
		n++
	}
	return n
}

// idLess orders IDs by prefix, then number, then suffix (API-062 < API-062a < API-063).
func idLess(a, b string) bool {
	pa, na, sa := splitID(a)
	pb, nb, sb := splitID(b)
	if pa != pb {
		return pa < pb
	}
	if na != nb {
		return na < nb
	}
	return sa < sb
}

func splitID(id string) (string, int, string) {
	i := strings.IndexByte(id, '-')
	if i < 0 {
		return id, 0, ""
	}
	rest := id[i+1:]
	j := 0
	for j < len(rest) && rest[j] >= '0' && rest[j] <= '9' {
		j++
	}
	n, _ := strconv.Atoi(rest[:j])
	return id[:i], n, rest[j:]
}

// statusClass normalises a status cell for counting: lower case, parenthetical dropped.
func statusClass(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	if i := strings.Index(s, "("); i > 0 {
		s = strings.TrimSpace(s[:i])
	}
	if s == "" {
		return "-"
	}
	return s
}

func isDecision(id string) bool { return strings.HasPrefix(id, "DEC-") }

// decisionIDs returns the DEC IDs that docs/DECISIONS.md defines as headings (DEC-028 onward).
func decisionIDs(src string) map[string]bool {
	out := map[string]bool{}
	for _, line := range strings.Split(src, "\n") {
		if m := decHeadRe.FindStringSubmatch(line); m != nil {
			out[m[1]] = true
		}
	}
	return out
}
