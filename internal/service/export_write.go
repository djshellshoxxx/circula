package service

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"strconv"

	"github.com/djshellshoxxx/circula/internal/store"
)

// csvCell renders one value. Text starting with a formula trigger gets a
// leading apostrophe (SEC-017); numbers are never altered, so negative
// amounts stay numeric.
func csvCell(v any) string {
	switch x := v.(type) {
	case nil:
		return ""
	case int64:
		return strconv.FormatInt(x, 10)
	case string:
		if x != "" && !plainNumber(x) && (x[0] == '=' || x[0] == '+' || x[0] == '-' || x[0] == '@' || x[0] == '\t' || x[0] == '\r') {
			return "'" + x
		}
		return x
	default:
		return fmt.Sprint(x)
	}
}

// plainNumber reports whether s is just an optionally negative decimal, which
// no spreadsheet can interpret as a formula (formatted amounts such as "-25.00").
func plainNumber(s string) bool {
	if s != "" && s[0] == '-' {
		s = s[1:]
	}
	dot := false
	if s == "" {
		return false
	}
	for i := 0; i < len(s); i++ {
		switch {
		case s[i] >= '0' && s[i] <= '9':
		case s[i] == '.' && !dot && i > 0 && i < len(s)-1:
			dot = true
		default:
			return false
		}
	}
	return true
}

// writeCSV writes one RFC 4180 UTF-8 file per table with a header row.
func writeCSV(tx store.Tx, sink ExportSink, man *Manifest, tables []store.ExportTable, memberID string) error {
	for _, t := range tables {
		name := string(t) + ".csv"
		raw, err := sink.Create(name)
		if err != nil {
			return err
		}
		cw := newCounting(raw)
		w := csv.NewWriter(cw)
		var rows int64
		err = tx.StreamTable(t, man.NetworkID, memberID, func(cols []string, vals []any) error {
			cols, vals = decorate(cols, vals, man.Decimals)
			if vals == nil {
				return w.Write(cols)
			}
			rec := make([]string, len(vals))
			for i, v := range vals {
				rec[i] = csvCell(v)
			}
			rows++
			return w.Write(rec)
		})
		w.Flush()
		if err == nil {
			err = w.Error()
		}
		if cerr := raw.Close(); err == nil {
			err = cerr
		}
		if err != nil {
			return fmt.Errorf("export %s: %w", name, err)
		}
		man.Files = append(man.Files, cw.file(name, rows))
	}
	return nil
}

// writeJSON streams a single structured document; rows are written one at a
// time so the ledger is never held in memory (FR-095).
func writeJSON(tx store.Tx, sink ExportSink, man *Manifest, tables []store.ExportTable, memberID string) error {
	name := string(man.Scope) + ".json"
	raw, err := sink.Create(name)
	if err != nil {
		return err
	}
	cw := newCounting(raw)
	var rows int64
	err = func() error {
		head, _ := json.Marshal(map[string]any{"schema_version": man.SchemaVersion, "network_id": man.NetworkID,
			"scope": man.Scope, "decimals": man.Decimals, "ledger_seq": man.LedgerSeq, "generated_at": man.GeneratedAt})
		if _, err := cw.Write(append(head[:len(head)-1], []byte(`,"tables":{`)...)); err != nil {
			return err
		}
		for i, t := range tables {
			sep := ""
			if i > 0 {
				sep = ","
			}
			if _, err := fmt.Fprintf(cw, `%s"%s":[`, sep, t); err != nil {
				return err
			}
			first := true
			if err := tx.StreamTable(t, man.NetworkID, memberID, func(cols []string, vals []any) error {
				if vals == nil {
					return nil
				}
				cols, vals = decorate(cols, vals, man.Decimals)
				if !first {
					if _, err := io.WriteString(cw, ","); err != nil {
						return err
					}
				}
				first = false
				rows++
				return writeJSONRow(cw, cols, vals)
			}); err != nil {
				return err
			}
			if _, err := io.WriteString(cw, "]"); err != nil {
				return err
			}
		}
		_, err := io.WriteString(cw, "}}\n")
		return err
	}()
	if cerr := raw.Close(); err == nil {
		err = cerr
	}
	if err != nil {
		return fmt.Errorf("export %s: %w", name, err)
	}
	man.Files = append(man.Files, cw.file(name, rows))
	return nil
}

// writeJSONRow writes one object with keys in column order.
func writeJSONRow(w io.Writer, cols []string, vals []any) error {
	if _, err := io.WriteString(w, "{"); err != nil {
		return err
	}
	for i, c := range cols {
		k, _ := json.Marshal(c)
		v, err := json.Marshal(vals[i])
		if err != nil {
			return err
		}
		sep := ","
		if i == 0 {
			sep = ""
		}
		if _, err := fmt.Fprintf(w, "%s%s:%s", sep, k, v); err != nil {
			return err
		}
	}
	_, err := io.WriteString(w, "}")
	return err
}
