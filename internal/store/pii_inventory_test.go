package store_test

import (
	"os"
	"regexp"
	"sort"
	"strings"
	"testing"
)

// inventoryPath is the committed personal-data inventory (SEC-014). It is the
// list this test checks the migrated schema against.
const inventoryPath = "../../docs/security/personal-data-inventory.md"

// personalCategories is the vocabulary of the Category cell in the inventory.
var personalCategories = map[string]bool{
	"identifier": true, "credential": true, "profile": true, "free text": true, "activity": true, "governance marker": true,
}

// forbiddenWords are column-name words for data SEC-014 forbids collecting:
// national identity numbers, government identifiers, bank account numbers,
// card data and date of birth.
var forbiddenWords = map[string]bool{
	"national": true, "ssn": true, "nin": true, "passport": true, "government": true, "taxid": true, "tin": true,
	"iban": true, "bic": true, "swift": true, "bank": true, "sortcode": true, "routing": true,
	"card": true, "pan": true, "cvv": true, "cvc": true,
	"birth": true, "birthday": true, "birthdate": true, "dob": true,
}

type inventory struct {
	personal map[string][]string // "table.column" -> cells after the first
	other    map[string][]string // table -> columns
}

var codeSpan = regexp.MustCompile("`([^`]+)`")

// readInventory parses the two machine-checked tables of the inventory: the
// section "Personal data columns" (one row per column) and "Other columns"
// (one row per table listing the remaining columns).
func readInventory(t *testing.T) inventory {
	t.Helper()
	b, err := os.ReadFile(inventoryPath)
	if err != nil {
		t.Fatalf("personal-data inventory missing (SEC-014): %v", err)
	}
	inv := inventory{personal: map[string][]string{}, other: map[string][]string{}}
	section := ""
	for _, line := range strings.Split(string(b), "\n") {
		if strings.HasPrefix(line, "## ") {
			section = strings.TrimSpace(strings.TrimPrefix(line, "## "))
			continue
		}
		if !strings.HasPrefix(line, "|") || strings.HasPrefix(line, "|---") {
			continue
		}
		cells := strings.Split(strings.Trim(line, "|"), "|")
		for i := range cells {
			cells[i] = strings.TrimSpace(cells[i])
		}
		key := codeSpan.FindStringSubmatch(cells[0])
		if key == nil { // header row
			continue
		}
		switch {
		case strings.HasSuffix(section, "Personal data columns"):
			if _, dup := inv.personal[key[1]]; dup {
				t.Errorf("inventory lists %s twice", key[1])
			}
			inv.personal[key[1]] = cells[1:]
		case strings.HasSuffix(section, "Other columns"):
			if len(cells) < 2 {
				t.Fatalf("inventory row for %s has no column list", key[1])
			}
			if _, dup := inv.other[key[1]]; dup {
				t.Errorf("inventory lists table %s twice under Other columns", key[1])
			}
			var cols []string
			for _, m := range codeSpan.FindAllStringSubmatch(cells[1], -1) {
				cols = append(cols, m[1])
			}
			inv.other[key[1]] = cols
		}
	}
	if len(inv.personal) == 0 || len(inv.other) == 0 {
		t.Fatalf("inventory has no machine-readable tables (personal %d, other %d)", len(inv.personal), len(inv.other))
	}
	return inv
}

// schemaColumns returns "table.column" for every column of every table after
// all migrations.
func schemaColumns(t *testing.T) []string {
	t.Helper()
	s := openStore(t)
	rows, err := s.DB().Query(`SELECT m.name, p.name FROM sqlite_master m JOIN pragma_table_info(m.name) p
		WHERE m.type = 'table' AND m.name NOT LIKE 'sqlite_%' ORDER BY m.name, p.cid`)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = rows.Close() }()
	var out []string
	for rows.Next() {
		var tbl, col string
		if err := rows.Scan(&tbl, &col); err != nil {
			t.Fatal(err)
		}
		out = append(out, tbl+"."+col)
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}
	return out
}

// TestPersonalDataInventoryCoversSchema fails when a migration adds a column
// (or table) that the committed inventory does not classify, or when the
// inventory names a column that no longer exists (SEC-014 traceability check).
func TestPersonalDataInventoryCoversSchema(t *testing.T) {
	inv := readInventory(t)
	classified := map[string]string{}
	for col := range inv.personal {
		classified[col] = "personal"
	}
	for tbl, cols := range inv.other {
		for _, c := range cols {
			k := tbl + "." + c
			if prev, ok := classified[k]; ok {
				t.Errorf("inventory classifies %s twice (%s and other)", k, prev)
			}
			classified[k] = "other"
		}
	}
	inSchema := map[string]bool{}
	var missing []string
	for _, c := range schemaColumns(t) {
		inSchema[c] = true
		if _, ok := classified[c]; !ok {
			missing = append(missing, c)
		}
	}
	if len(missing) > 0 {
		t.Errorf("columns not classified in %s (add each to \"Personal data columns\" with purpose, retention, export "+
			"and erasure, or to \"Other columns\"): %s", inventoryPath, strings.Join(missing, ", "))
	}
	var stale []string
	for c := range classified {
		if !inSchema[c] {
			stale = append(stale, c)
		}
	}
	sort.Strings(stale)
	if len(stale) > 0 {
		t.Errorf("inventory names columns that are not in the schema: %s", strings.Join(stale, ", "))
	}
}

// TestPersonalDataInventoryRowsComplete requires a category, purpose,
// retention, export and erasure entry for every personal-data column.
func TestPersonalDataInventoryRowsComplete(t *testing.T) {
	inv := readInventory(t)
	for col, cells := range inv.personal {
		if len(cells) != 5 {
			t.Errorf("%s: want 5 cells after the column (category, purpose, retention, export, erasure), got %d", col, len(cells))
			continue
		}
		if !personalCategories[cells[0]] {
			t.Errorf("%s: category %q is not one of the inventory categories", col, cells[0])
		}
		for i, name := range []string{"purpose", "retention", "export", "erasure"} {
			if cells[i+1] == "" {
				t.Errorf("%s: empty %s", col, name)
			}
		}
	}
}

// TestSchemaHasNoForbiddenPersonalData is the SEC-014 schema test: no table
// column stores a national or government identifier, bank account, card data
// or date of birth.
func TestSchemaHasNoForbiddenPersonalData(t *testing.T) {
	split := regexp.MustCompile(`[^a-z0-9]+`)
	for _, c := range schemaColumns(t) {
		col := strings.ToLower(c[strings.IndexByte(c, '.')+1:])
		joined := strings.ReplaceAll(col, "_", "")
		for _, bad := range []string{"dateofbirth", "accountnumber", "cardnumber", "nationalid", "taxid"} {
			if strings.Contains(joined, bad) {
				t.Errorf("column %s looks like forbidden personal data (%s)", c, bad)
			}
		}
		for _, w := range split.Split(col, -1) {
			if forbiddenWords[w] {
				t.Errorf("column %s looks like forbidden personal data (word %q)", c, w)
			}
		}
	}
}
