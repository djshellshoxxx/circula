package store_test

import (
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"testing"
)

// DAT-045: schema changes follow expand-then-contract. A migration may add
// structures; removing or renaming one (the contract step) is only allowed in
// a later release, after the reads moved off it. Every migration that drops or
// renames anything must therefore be listed here, with the earlier migration
// and release after which nothing reads the old structure. Any other DROP or
// RENAME fails the build. Migrations 0001..0015 contain none. Dropping a TEMP
// table that the same migration created is not a contract step: no release
// ever read it (0016 uses one as a backfill check).
var contractSteps = map[string]string{
	// "00NN_name.sql": "reads moved to <new structure> in 00MM (release vX.Y); <what is dropped or renamed>",
}

// contractRE matches the statements that remove or rename schema objects:
// DROP TABLE/INDEX/TRIGGER/VIEW, ALTER TABLE ... DROP COLUMN and
// ALTER TABLE ... RENAME [COLUMN] ... TO.
var contractRE = regexp.MustCompile(`(?i)\b(drop|rename)\b`)

// tempTableRE and dropTableRE find TEMP tables a migration creates and the
// DROP TABLE statements that remove them again.
var (
	tempTableRE = regexp.MustCompile(`(?i)\bcreate\s+temp(?:orary)?\s+table\s+(?:if\s+not\s+exists\s+)?(\w+)`)
	dropTableRE = regexp.MustCompile(`(?i)^\s*drop\s+table\s+(?:if\s+exists\s+)?(\w+)\s*;\s*$`)
)

// sqlCode blanks comments and quoted strings or identifiers, keeping newlines,
// so a keyword in a comment or a RAISE message is not taken for a statement.
func sqlCode(src string) string {
	b := []byte(src)
	blank := func(i int) {
		if b[i] != '\n' {
			b[i] = ' '
		}
	}
	for i := 0; i < len(b); i++ {
		switch {
		case b[i] == '-' && i+1 < len(b) && b[i+1] == '-':
			for ; i < len(b) && b[i] != '\n'; i++ {
				blank(i)
			}
		case b[i] == '/' && i+1 < len(b) && b[i+1] == '*':
			for ; i < len(b); i++ {
				end := b[i] == '*' && i+1 < len(b) && b[i+1] == '/'
				blank(i)
				if end {
					i++
					blank(i)
					break
				}
			}
		case b[i] == '\'' || b[i] == '"' || b[i] == '`':
			q := b[i]
			blank(i)
			for i++; i < len(b); i++ {
				if b[i] == q {
					if i+1 < len(b) && b[i+1] == q { // doubled quote escapes itself
						blank(i)
						i++
						blank(i)
						continue
					}
					blank(i)
					break
				}
				blank(i)
			}
		}
	}
	return string(b)
}

// contractStatements returns "line N: <source line>" for every DROP or RENAME
// outside comments and strings.
func contractStatements(src string) []string {
	code := sqlCode(src)
	temp := map[string]bool{}
	for _, m := range tempTableRE.FindAllStringSubmatch(code, -1) {
		temp[strings.ToLower(m[1])] = true
	}
	lines := strings.Split(src, "\n")
	var out []string
	for i, l := range strings.Split(code, "\n") {
		if m := dropTableRE.FindStringSubmatch(l); m != nil && temp[strings.ToLower(m[1])] {
			continue // the migration's own scratch table
		}
		if contractRE.MatchString(l) {
			out = append(out, "line "+strconv.Itoa(i+1)+": "+strings.TrimSpace(lines[i]))
		}
	}
	return out
}

func TestMigrationsExpandThenContract(t *testing.T) {
	files, err := filepath.Glob(filepath.Join("migrations", "*.sql"))
	if err != nil || len(files) == 0 {
		t.Fatalf("no migrations found: %v", err)
	}
	sort.Strings(files)
	seen := map[string]bool{}
	for _, f := range files {
		name := filepath.Base(f)
		seen[name] = true
		src, err := os.ReadFile(f)
		if err != nil {
			t.Fatal(err)
		}
		found := contractStatements(string(src))
		_, allowed := contractSteps[name]
		switch {
		case len(found) > 0 && !allowed:
			t.Errorf("%s drops or renames schema objects without a contract-step entry (DAT-045: expand, backfill, switch reads, "+
				"then contract in a later release; list it in contractSteps with the migration that moved the reads):\n  %s",
				name, strings.Join(found, "\n  "))
		case len(found) == 0 && allowed:
			t.Errorf("contractSteps lists %s but it neither drops nor renames anything; remove the entry", name)
		}
	}
	for name := range contractSteps {
		if !seen[name] {
			t.Errorf("contractSteps lists %s, which is not a migration", name)
		}
	}
}

// The scanner itself: every contract form is caught, and keywords inside
// comments, strings or longer identifiers are not.
func TestContractScannerForms(t *testing.T) {
	caught := []string{
		"DROP TABLE accounts;",
		"drop index idx_x;",
		"DROP TRIGGER IF EXISTS entries_no_update;",
		"DROP VIEW v;",
		"ALTER TABLE persons DROP COLUMN login;",
		"ALTER TABLE persons RENAME TO people;",
		"ALTER TABLE persons RENAME COLUMN login TO handle;",
		"CREATE TABLE t (a INT); /* c */ Drop Table t;",
	}
	for _, s := range caught {
		if len(contractStatements(s)) == 0 {
			t.Errorf("not caught: %q", s)
		}
	}
	ignored := []string{
		"-- we never DROP or RENAME anything here",
		"/* DROP TABLE x;\n RENAME */ CREATE TABLE y (a INT);",
		"SELECT RAISE(ABORT, 'cannot drop: append-only');",
		`CREATE TABLE "drop" (a INT);`,
		"CREATE TABLE t (dropped_at INTEGER, drop_reason TEXT, renamed INTEGER);",
		"SELECT 'it''s DROP';",
	}
	for _, s := range ignored {
		if got := contractStatements(s); len(got) != 0 {
			t.Errorf("false positive for %q: %v", s, got)
		}
	}
	if got := contractStatements("CREATE TABLE a (x INT);\n-- note\nDROP TABLE a;"); len(got) != 1 || !strings.HasPrefix(got[0], "line 3:") {
		t.Errorf("line reporting = %v", got)
	}
}

// The TEMP-table exemption covers only the migration's own scratch tables.
func TestContractStatementsTempTableExemption(t *testing.T) {
	src := "CREATE TEMP TABLE scratch (x INTEGER);\nDROP TABLE scratch;\nDROP TABLE members;\nALTER TABLE t RENAME TO u;\n"
	got := contractStatements(src)
	if len(got) != 2 || !strings.Contains(got[0], "members") || !strings.Contains(got[1], "RENAME") {
		t.Fatalf("contractStatements = %q", got)
	}
	if got := contractStatements("DROP TABLE scratch;\n"); len(got) != 1 {
		t.Fatalf("a drop of a table this migration did not create as TEMP must be flagged: %q", got)
	}
}
