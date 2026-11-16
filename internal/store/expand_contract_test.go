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
// RENAME fails the build. Migrations 0001..0016 contain none: dropping an
// object the same migration created (0016's temporary backfill check table)
// removes nothing a release ever read, so it is not a contract step.
var contractSteps = map[string]string{
	// "00NN_name.sql": "reads moved to <new structure> in 00MM (release vX.Y); <what is dropped or renamed>",
}

// contractRE matches the statements that remove or rename schema objects:
// DROP TABLE/INDEX/TRIGGER/VIEW, ALTER TABLE ... DROP COLUMN and
// ALTER TABLE ... RENAME [COLUMN] ... TO.
var contractRE = regexp.MustCompile(`(?i)\b(drop|rename)\b`)

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

// createRE and dropObjRE name the object a CREATE or DROP statement acts on.
// Quoted names are blanked by sqlCode, so a DROP of a quoted name never matches
// dropObjRE and is always reported.
var (
	createRE  = regexp.MustCompile(`(?i)\bcreate\s+(?:(?:temp|temporary|unique)\s+)?(table|index|trigger|view)\s+(?:if\s+not\s+exists\s+)?(?:(?:temp|main)\s*\.\s*)?(\w+)`)
	dropObjRE = regexp.MustCompile(`(?i)^drop\s+(table|index|trigger|view)\s+(?:if\s+exists\s+)?(?:(?:temp|main)\s*\.\s*)?(\w+)`)
)

// contractStatements returns "line N: <source line>" for every DROP or RENAME
// outside comments and strings, except a DROP of an object created earlier in
// the same source: such an object existed only during the migration.
func contractStatements(src string) []string {
	code := sqlCode(src)
	lines := strings.Split(src, "\n")
	created := map[string]int{} // kind+" "+lower(name) -> offset of its CREATE
	for _, m := range createRE.FindAllStringSubmatchIndex(code, -1) {
		key := strings.ToLower(code[m[2]:m[3]]) + " " + strings.ToLower(code[m[4]:m[5]])
		if _, ok := created[key]; !ok {
			created[key] = m[0]
		}
	}
	reported := map[int]bool{}
	var out []string
	for _, m := range contractRE.FindAllStringIndex(code, -1) {
		if d := dropObjRE.FindStringSubmatch(code[m[0]:]); d != nil {
			key := strings.ToLower(d[1]) + " " + strings.ToLower(d[2])
			if at, ok := created[key]; ok && at < m[0] {
				continue
			}
		}
		line := strings.Count(code[:m[0]], "\n")
		if !reported[line] {
			reported[line] = true
			out = append(out, "line "+strconv.Itoa(line+1)+": "+strings.TrimSpace(lines[line]))
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
		"CREATE TABLE t (a INT); /* c */ Drop Table t;\nDROP TABLE accounts;",
		// Dropping a table before the migration creates one of that name is a contract.
		"DROP TABLE x; CREATE TABLE x (a INT);",
		// A same-named object of another kind is not the one created.
		"CREATE INDEX accounts ON t(a); DROP TABLE accounts;",
		// A quoted name cannot be matched to its CREATE, so it is always reported.
		`CREATE TEMP TABLE "chk" (a INT); DROP TABLE "chk";`,
		// Renaming an object created in the same migration still counts (table rebuild).
		"CREATE TABLE persons_new (a INT); ALTER TABLE persons_new RENAME TO persons;",
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
		// An object created and dropped in the same migration never reached a release.
		"CREATE TEMP TABLE chk (n INT CHECK (n = 0));\nINSERT INTO chk VALUES (0);\nDROP TABLE chk;",
		"CREATE TEMPORARY TABLE IF NOT EXISTS temp.chk2 (n INT); drop table if exists Chk2;",
		"CREATE UNIQUE INDEX ix ON t(a);\nDROP INDEX ix;",
	}
	for _, s := range ignored {
		if got := contractStatements(s); len(got) != 0 {
			t.Errorf("false positive for %q: %v", s, got)
		}
	}
	if got := contractStatements("CREATE TABLE a (x INT);\n-- note\nDROP TABLE b;"); len(got) != 1 || !strings.HasPrefix(got[0], "line 3:") {
		t.Errorf("line reporting = %v", got)
	}
}
