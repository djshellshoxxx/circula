package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"hash"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/djshellshoxxx/circula/internal/domain"
	"github.com/djshellshoxxx/circula/internal/store"
)

// ExportSchemaVersion is the version of the export package layout (DAT-041).
const ExportSchemaVersion = 1

// ExportScope selects what is exported.
type ExportScope string

// Export scopes.
const (
	ExportMine    ExportScope = "mine"    // FR-090: caller's own transactions and entries
	ExportLedger  ExportScope = "ledger"  // FR-091: accounts, transactions, status history, entries
	ExportMembers ExportScope = "members" // FR-092: member list and role assignments
	ExportAudit   ExportScope = "audit"   // FR-093: audit history
	ExportFull    ExportScope = "full"    // FR-094: everything, also for closed networks
)

// ExportFormat is csv (one file per table) or json (one structured file).
type ExportFormat string

// Export formats.
const (
	FormatCSV  ExportFormat = "csv"
	FormatJSON ExportFormat = "json"
)

// ExportSink receives the export files. Each Create'd file is closed by the service.
type ExportSink interface {
	Create(name string) (io.WriteCloser, error)
}

// ManifestFile describes one exported file.
type ManifestFile struct {
	Name   string `json:"name"`
	Rows   int64  `json:"rows"`
	Bytes  int64  `json:"bytes"`
	SHA256 string `json:"sha256"`
}

// Manifest lists the files with row counts and checksums, and states the
// single ledger_seq the whole export was read at (DAT-037, DAT-039).
type Manifest struct {
	SchemaVersion int            `json:"schema_version"`
	NetworkID     string         `json:"network_id"`
	Scope         ExportScope    `json:"scope"`
	Format        ExportFormat   `json:"format"`
	Decimals      int            `json:"decimals"`
	LedgerSeq     int64          `json:"ledger_seq"`
	GeneratedAt   time.Time      `json:"generated_at"`
	Files         []ManifestFile `json:"files"`
}

var scopeTables = map[ExportScope][]store.ExportTable{
	ExportMine:    {store.TblTransactions, store.TblEntries},
	ExportLedger:  {store.TblAccounts, store.TblTransactions, store.TblStatusEvents, store.TblEntries},
	ExportMembers: {store.TblMembers, store.TblRoles},
	ExportAudit:   {store.TblAudit},
	ExportFull: {store.TblMembers, store.TblRoles, store.TblAccounts, store.TblListings, store.TblTransactions,
		store.TblStatusEvents, store.TblEntries, store.TblDisputes, store.TblPolicies, store.TblAudit},
}

// Export streams an export package to sink and returns its manifest. Members
// may export only their own data (also when suspended or departed, SEC-017);
// every other scope needs an administrator. The export is read from one
// consistent snapshot and is itself audited. Credentials, tokens and recovery
// material are unreachable by construction; text cells that could trigger
// spreadsheet formulas are neutralised in CSV.
func (s *Service) Export(ctx context.Context, actor domain.Actor, netID string, scope ExportScope, format ExportFormat, sink ExportSink) (*Manifest, error) {
	tables, ok := scopeTables[scope]
	if !ok {
		return nil, domain.Invalid("scope", "unknown export scope")
	}
	if format != FormatCSV && format != FormatJSON {
		return nil, domain.Invalid("format", "must be csv or json")
	}
	var memberID string
	err := s.update(ctx, func(tx store.Tx) error {
		p, err := s.open(tx, actor, netID, domain.ScopeRead)
		if err != nil {
			return err
		}
		if scope == ExportMine {
			if p.member == nil || p.member.Status == domain.MemberPending {
				return domain.Forbidden("no data to export")
			}
			memberID = p.member.ID
		} else if err := p.requireAdmin(); err != nil {
			return err
		}
		return s.audit(tx, p, auditEntry{Action: domain.ActExport, TargetType: "export", TargetID: netID,
			After: map[string]any{"status": string(scope)}})
	})
	if err != nil {
		return nil, err
	}
	var man *Manifest
	err = s.view(ctx, func(tx store.Tx) error {
		net, err := tx.GetNetwork(netID)
		if err != nil {
			return domain.NotFound("network")
		}
		seq, err := tx.LedgerSeq(netID)
		if err != nil {
			return err
		}
		man = &Manifest{SchemaVersion: ExportSchemaVersion, NetworkID: netID, Scope: scope, Format: format,
			Decimals: net.Policy.Decimals, LedgerSeq: seq, GeneratedAt: s.now()}
		if format == FormatCSV {
			return writeCSV(tx, sink, man, tables, memberID)
		}
		return writeJSON(tx, sink, man, tables, memberID)
	})
	if err != nil {
		return nil, err
	}
	if err := writeManifest(sink, man); err != nil {
		return nil, err
	}
	return man, nil
}

func writeManifest(sink ExportSink, m *Manifest) error {
	w, err := sink.Create("manifest.json")
	if err != nil {
		return err
	}
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	if err := enc.Encode(m); err != nil {
		_ = w.Close()
		return err
	}
	return w.Close()
}

// countingWriter hashes and counts everything written to a sink file.
type countingWriter struct {
	w io.Writer
	h hash.Hash
	n int64
}

func newCounting(w io.Writer) *countingWriter { return &countingWriter{w: w, h: sha256.New()} }

func (c *countingWriter) Write(p []byte) (int, error) {
	n, err := c.w.Write(p)
	c.h.Write(p[:n])
	c.n += int64(n)
	return n, err
}

func (c *countingWriter) file(name string, rows int64) ManifestFile {
	return ManifestFile{Name: name, Rows: rows, Bytes: c.n, SHA256: hex.EncodeToString(c.h.Sum(nil))}
}

// moneyCols are minor-unit columns that also get a formatted twin (FR-090).
var moneyCols = map[string]bool{"amount": true, "balance": true, "limit_override": true, "price": true,
	"reversed_total": true, "reversal_amount": true}

// decorate appends "<col>_formatted" for money columns.
func decorate(cols []string, vals []any, decimals int) ([]string, []any) {
	outC := make([]string, 0, len(cols)+2)
	var outV []any
	if vals != nil {
		outV = make([]any, 0, len(vals)+2)
	}
	for i, c := range cols {
		outC = append(outC, c)
		if vals != nil {
			outV = append(outV, vals[i])
		}
		if moneyCols[c] {
			outC = append(outC, c+"_formatted")
			if vals != nil {
				if v, ok := vals[i].(int64); ok {
					outV = append(outV, domain.FormatAmount(v, decimals))
				} else {
					outV = append(outV, nil)
				}
			}
		}
	}
	return outC, outV
}

// DirSink writes export files into a directory (for CLI use).
type DirSink struct{ Dir string }

// Create implements ExportSink.
func (d DirSink) Create(name string) (io.WriteCloser, error) {
	if name != filepath.Base(name) || strings.HasPrefix(name, ".") {
		return nil, fmt.Errorf("export: bad file name %q", name)
	}
	return os.OpenFile(filepath.Join(d.Dir, name), os.O_CREATE|os.O_WRONLY|os.O_EXCL, 0o600)
}

// MemSink collects export files in memory (tests, small exports).
type MemSink struct{ Files map[string]*memFile }

type memFile struct{ b []byte }

func (m *memFile) Write(p []byte) (int, error) { m.b = append(m.b, p...); return len(p), nil }
func (m *memFile) Close() error                { return nil }

// Create implements ExportSink.
func (m *MemSink) Create(name string) (io.WriteCloser, error) {
	if m.Files == nil {
		m.Files = map[string]*memFile{}
	}
	f := &memFile{}
	m.Files[name] = f
	return f, nil
}

// Names lists the files created so far, sorted.
func (m *MemSink) Names() []string {
	out := make([]string, 0, len(m.Files))
	for n := range m.Files {
		out = append(out, n)
	}
	sort.Strings(out)
	return out
}

// Bytes returns a file's content (nil if absent).
func (m *MemSink) Bytes(name string) []byte {
	if f := m.Files[name]; f != nil {
		return f.b
	}
	return nil
}
