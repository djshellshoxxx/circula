package service

import (
	"archive/zip"
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
// single ledger_seq the whole export was read at (DAT-037, DAT-039). Scopes
// that export entries also carry a reconciliation summary (GOV-022).
type Manifest struct {
	SchemaVersion  int             `json:"schema_version"`
	NetworkID      string          `json:"network_id"`
	Scope          ExportScope     `json:"scope"`
	Format         ExportFormat    `json:"format"`
	Decimals       int             `json:"decimals"`
	LedgerSeq      int64           `json:"ledger_seq"`
	GeneratedAt    time.Time       `json:"generated_at"`
	Files          []ManifestFile  `json:"files"`
	Reconciliation *Reconciliation `json:"reconciliation,omitempty"`
}

// Reconciliation is the summary a recipient recomputes from the exported
// entries (GOV-022, QA-030), read at the manifest's ledger_seq. Entries and
// EntrySum cover every exported entry row. BalanceTotal sums the balances of
// the covered accounts: every account of the network for the ledger and full
// scopes (zero in a sound ledger, as is EntrySum), the caller's own account
// for mine (equal to EntrySum, since mine holds only the caller's entries).
// Accounts fingerprints the per-account balances; it is present for the
// ledger and full scopes only.
type Reconciliation struct {
	Entries      int64          `json:"entries"`
	EntrySum     int64          `json:"entry_sum"`
	BalanceTotal int64          `json:"balance_total"`
	Accounts     *AccountDigest `json:"accounts,omitempty"`
}

// AccountDigest fingerprints per-account balances so that they can be
// checked against the entries alone. SHA256 is the hex SHA-256 of one line
// "<account_id>,<balance>\n" per account whose balance is not zero, in byte
// order of account_id; NonZero is the number of those lines.
type AccountDigest struct {
	NonZero int64  `json:"nonzero"`
	SHA256  string `json:"sha256"`
}

var scopeTables = map[ExportScope][]store.ExportTable{
	ExportMine:    {store.TblTransactions, store.TblEntries},
	ExportLedger:  {store.TblAccounts, store.TblTransactions, store.TblStatusEvents, store.TblEntries},
	ExportMembers: {store.TblMembers, store.TblRoles},
	ExportAudit:   {store.TblAudit},
	ExportFull: {store.TblMembers, store.TblRoles, store.TblAccounts, store.TblListings, store.TblTransactions,
		store.TblStatusEvents, store.TblEntries, store.TblDisputes, store.TblPolicies, store.TblAudit},
}

// AuthorizeExportDownload re-checks, at download time, that actor may still
// obtain an export of scope from netID (SA2-17): a member who was demoted,
// suspended, departed or erased since the export was made must not fetch a
// stored artefact. Members may fetch ExportMine; every other scope needs an
// active administrator. It writes nothing.
func (s *Service) AuthorizeExportDownload(ctx context.Context, actor domain.Actor, netID string, scope ExportScope) error {
	if _, ok := scopeTables[scope]; !ok {
		return domain.Invalid("scope", "unknown export scope")
	}
	return s.view(ctx, func(tx store.Tx) error {
		p, err := s.open(tx, actor, netID, domain.ScopeRead)
		if err != nil {
			return err
		}
		if scope == ExportMine {
			if p.member == nil || p.member.Status == domain.MemberPending {
				return domain.Forbidden("no data to export")
			}
			if done, err := tx.GetErasure(netID, p.member.ID); err != nil {
				return err
			} else if done != nil {
				return domain.Forbidden("no data to export")
			}
			return nil
		}
		return p.requireAdmin()
	})
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
		// SA-11: per-actor quota, checked before anything is written, so a flood
		// of requests neither fills the audit log nor starves other members.
		if lim := s.cfg.ExportsPerHour; lim > 0 && !p.actor.System {
			n, err := tx.CountAuditSince(netID, domain.ActExport, p.memberID(), s.now().Add(-time.Hour))
			if err != nil {
				return err
			}
			if n >= lim {
				return domain.Errf(domain.CodeRateLimited, "export limit reached (%d per hour); try again later", lim)
			}
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
		rc := newReconciler(tables, memberID == "")
		if format == FormatCSV {
			err = writeCSV(tx, sink, man, tables, memberID, rc)
		} else {
			err = writeJSON(tx, sink, man, tables, memberID, rc)
		}
		if err != nil || rc == nil {
			return err
		}
		if memberID != "" { // mine: the caller's own balance, from the same snapshot
			if err := tx.StreamTable(store.TblAccounts, netID, memberID, func(cols []string, vals []any) error {
				rc.observe(store.TblAccounts, cols, vals)
				return nil
			}); err != nil {
				return err
			}
		}
		man.Reconciliation = rc.summary()
		return nil
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

// reconciler builds the reconciliation summary from the rows as they stream
// past, so it needs no second read and holds nothing but running totals.
type reconciler struct {
	rec     Reconciliation
	digest  hash.Hash // nil unless the scope covers every account of the network
	nonzero int64
	amount  int // column of entries.amount
	id, bal int // columns of accounts.id and accounts.balance
}

// newReconciler returns nil when tables hold no entries. networkWide selects
// the per-account digest (ledger and full scopes).
func newReconciler(tables []store.ExportTable, networkWide bool) *reconciler {
	for _, t := range tables {
		if t == store.TblEntries {
			r := &reconciler{amount: -1, id: -1, bal: -1}
			if networkWide {
				r.digest = sha256.New()
			}
			return r
		}
	}
	return nil
}

// observe takes one StreamTable callback: the header (nil vals) or a row.
// Accounts arrive ordered by id, which is the digest's order.
func (r *reconciler) observe(t store.ExportTable, cols []string, vals []any) {
	if r == nil {
		return
	}
	switch t {
	case store.TblEntries:
		if vals == nil {
			r.amount = colIndex(cols, "amount")
			return
		}
		r.rec.Entries++
		if v, ok := cell[int64](vals, r.amount); ok {
			r.rec.EntrySum += v
		}
	case store.TblAccounts:
		if vals == nil {
			r.id, r.bal = colIndex(cols, "id"), colIndex(cols, "balance")
			return
		}
		b, _ := cell[int64](vals, r.bal)
		r.rec.BalanceTotal += b
		if id, ok := cell[string](vals, r.id); ok && r.digest != nil && b != 0 {
			r.nonzero++
			fmt.Fprintf(r.digest, "%s,%d\n", id, b)
		}
	}
}

func (r *reconciler) summary() *Reconciliation {
	out := r.rec
	if r.digest != nil {
		out.Accounts = &AccountDigest{NonZero: r.nonzero, SHA256: hex.EncodeToString(r.digest.Sum(nil))}
	}
	return &out
}

func colIndex(cols []string, name string) int {
	for i, c := range cols {
		if c == name {
			return i
		}
	}
	return -1
}

func cell[T any](vals []any, i int) (T, bool) {
	var zero T
	if i < 0 || i >= len(vals) {
		return zero, false
	}
	v, ok := vals[i].(T)
	return v, ok
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

// checkFileName refuses names that could leave the package directory.
func checkFileName(name string) error {
	if name == "" || name != filepath.Base(name) || strings.HasPrefix(name, ".") {
		return fmt.Errorf("export: bad file name %q", name)
	}
	return nil
}

// DirSink writes export files into a directory (for CLI use).
type DirSink struct{ Dir string }

// Create implements ExportSink.
func (d DirSink) Create(name string) (io.WriteCloser, error) {
	if err := checkFileName(name); err != nil {
		return nil, err
	}
	return os.OpenFile(filepath.Join(d.Dir, name), os.O_CREATE|os.O_WRONLY|os.O_EXCL, 0o600)
}

// ZipSink writes the package as one zip archive to an io.Writer, entry by
// entry in the order the service creates the files (manifest.json last).
// Nothing is buffered beyond the compressor's window, so the archive can go
// straight to a file or a response (FR-095). The API and the CLI both use
// it, so they produce the same archive (ARC-020). Close it after Export.
type ZipSink struct{ zw *zip.Writer }

// NewZipSink starts an archive on w.
func NewZipSink(w io.Writer) *ZipSink { return &ZipSink{zw: zip.NewWriter(w)} }

// Create implements ExportSink. Creating a file ends the previous one.
func (z *ZipSink) Create(name string) (io.WriteCloser, error) {
	if err := checkFileName(name); err != nil {
		return nil, err
	}
	w, err := z.zw.Create(name)
	if err != nil {
		return nil, err
	}
	return zipEntry{w}, nil
}

// Close writes the archive's central directory; it does not close the
// underlying writer.
func (z *ZipSink) Close() error { return z.zw.Close() }

type zipEntry struct{ io.Writer }

func (zipEntry) Close() error { return nil }

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
