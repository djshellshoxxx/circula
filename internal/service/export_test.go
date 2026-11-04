package service_test

import (
	"crypto/sha256"
	"encoding/csv"
	"encoding/hex"
	"encoding/json"
	"strconv"
	"strings"
	"testing"

	"github.com/djshellshoxxx/circula/internal/domain"
	"github.com/djshellshoxxx/circula/internal/service"
	tu "github.com/djshellshoxxx/circula/internal/testutil"
)

func readCSV(t *testing.T, s *service.MemSink, name string) (header []string, rows [][]string) {
	t.Helper()
	recs, err := csv.NewReader(strings.NewReader(string(s.Bytes(name)))).ReadAll()
	if err != nil || len(recs) == 0 {
		t.Fatalf("%s: %v (%d records)", name, err, len(recs))
	}
	return recs[0], recs[1:]
}

func col(t *testing.T, header []string, name string) int {
	t.Helper()
	for i, h := range header {
		if h == name {
			return i
		}
	}
	t.Fatalf("no column %q in %v", name, header)
	return -1
}

// FR-091: entries sum to zero, per-account sums equal exported balances, checksums verify.
func TestLedgerExportCSV(t *testing.T) {
	w := newWorld(t, nil)
	w.Trade(w.net.ID, w.a, w.b, w.aM, w.bM, 2500)
	w.Trade(w.net.ID, w.b, w.admin, w.bM, w.adminM, 700)
	tu.Must(w.adjust(w.admin, w.aM, 100, service.AdjustCredit, "bonus"))
	sink := &service.MemSink{}
	man := tu.Must(w.Svc.Export(w.Ctx, w.admin, w.net.ID, service.ExportLedger, service.FormatCSV, sink))

	if man.SchemaVersion != service.ExportSchemaVersion || man.LedgerSeq != 3 || man.Decimals != 2 || len(man.Files) != 4 {
		t.Fatalf("manifest = %+v", man)
	}
	for _, f := range man.Files { // DAT-040: checksum and row counts verify
		sum := sha256.Sum256(sink.Bytes(f.Name))
		if hex.EncodeToString(sum[:]) != f.SHA256 || int64(len(sink.Bytes(f.Name))) != f.Bytes {
			t.Errorf("%s checksum/bytes mismatch", f.Name)
		}
		if _, rows := readCSV(t, sink, f.Name); int64(len(rows)) != f.Rows {
			t.Errorf("%s rows %d != manifest %d", f.Name, len(rows), f.Rows)
		}
	}
	var onDisk Manifest2
	if err := json.Unmarshal(sink.Bytes("manifest.json"), &onDisk); err != nil || onDisk.LedgerSeq != 3 || len(onDisk.Files) != 4 {
		t.Errorf("manifest.json: %v %+v", err, onDisk)
	}
	h, entries := readCSV(t, sink, "entries.csv")
	ai, ci, fi := col(t, h, "account_id"), col(t, h, "amount"), col(t, h, "amount_formatted")
	perAccount := map[string]int64{}
	var total int64
	for _, r := range entries {
		v, _ := strconv.ParseInt(r[ci], 10, 64)
		perAccount[r[ai]] += v
		total += v
		if r[fi] != domain.FormatAmount(v, 2) {
			t.Errorf("formatted %q for %d", r[fi], v)
		}
	}
	if total != 0 {
		t.Errorf("exported entries sum to %d", total)
	}
	ah, accounts := readCSV(t, sink, "accounts.csv")
	idI, bI := col(t, ah, "id"), col(t, ah, "balance")
	for _, r := range accounts {
		b, _ := strconv.ParseInt(r[bI], 10, 64)
		if perAccount[r[idI]] != b {
			t.Errorf("account %s: entries %d != exported balance %d", r[idI], perAccount[r[idI]], b)
		}
	}
	// credentials never appear anywhere in an export
	for _, name := range sink.Names() {
		s := string(sink.Bytes(name))
		if strings.Contains(s, "argon2") || strings.Contains(s, "password") || strings.Contains(s, "token_hash") {
			t.Errorf("%s contains credential material", name)
		}
	}
}

type Manifest2 struct {
	LedgerSeq int64 `json:"ledger_seq"`
	Files     []struct {
		Name string `json:"name"`
	} `json:"files"`
}

// FR-090 / SEC-017: a member's export holds only their own data, counterparties
// by display name only, and works after departure.
func TestMemberExportIsScoped(t *testing.T) {
	w := newWorld(t, nil)
	c, cM := w.Member2(w.admin, "carol@example.org")
	w.Trade(w.net.ID, w.a, w.b, w.aM, w.bM, 2500)
	w.Trade(w.net.ID, w.b, c, w.bM, cM, 300)
	tu.Must(w.Svc.DepartMember(w.Ctx, w.a, w.net.ID, service.DepartInput{}))

	sink := &service.MemSink{}
	tu.Must(w.Svc.Export(w.Ctx, w.a, w.net.ID, service.ExportMine, service.FormatCSV, sink))
	h, txs := readCSV(t, sink, "transactions.csv")
	if len(txs) != 1 {
		t.Fatalf("A's export has %d transactions, want 1 (rows must equal history)", len(txs))
	}
	hist := tu.Must(w.Svc.ListTransactions(w.Ctx, w.a, w.net.ID, nil, service.Page{}))
	if len(hist) != len(txs) {
		t.Errorf("export rows %d != history %d", len(txs), len(hist))
	}
	if r := txs[0]; r[col(t, h, "payer_name")] != "ana@example.org" || r[col(t, h, "payee_name")] != "ben@example.org" || r[col(t, h, "my_role")] != "payer" {
		t.Errorf("row = %v", r)
	}
	_, es := readCSV(t, sink, "entries.csv")
	if len(es) != 1 {
		t.Errorf("A's entries = %d", len(es))
	}
	all := ""
	for _, name := range sink.Names() {
		all += string(sink.Bytes(name))
	}
	for _, other := range []string{w.bM.ID, cM.ID, w.adminM.ID, w.b.PersonID, "carol"} {
		if strings.Contains(all, other) {
			t.Errorf("member export leaks %q", other)
		}
	}
	// a member cannot reach network-wide tables
	_, err := w.Svc.Export(w.Ctx, w.a, w.net.ID, service.ExportAudit, service.FormatCSV, &service.MemSink{})
	wantCode(t, err, domain.ErrForbidden)
}

// SEC-017: spreadsheet formula injection is neutralised; numbers are untouched.
func TestCSVFormulaNeutralisation(t *testing.T) {
	w := newWorld(t, nil)
	// display names are set at join; use the raw name field through a member joining with a formula name
	p := w.Person("formula@example.org")
	tok, _ := tu.Must2(w.Svc.CreateInvitation(w.Ctx, w.admin, w.net.ID, service.InviteInput{}))
	tu.Must(w.Svc.RedeemInvitation(w.Ctx, p, tok, "=cmd|' /C calc'!A0"))
	w.Trade(w.net.ID, w.a, w.b, w.aM, w.bM, 2500)
	sink := &service.MemSink{}
	tu.Must(w.Svc.Export(w.Ctx, w.admin, w.net.ID, service.ExportMembers, service.FormatCSV, sink))
	h, rows := readCSV(t, sink, "members.csv")
	ni := col(t, h, "display_name")
	found := false
	for _, r := range rows {
		if strings.Contains(r[ni], "cmd|") {
			found = true
			if !strings.HasPrefix(r[ni], "'=") {
				t.Errorf("formula not neutralised: %q", r[ni])
			}
		}
	}
	if !found {
		t.Fatal("formula member not exported")
	}
	l := &service.MemSink{}
	tu.Must(w.Svc.Export(w.Ctx, w.admin, w.net.ID, service.ExportLedger, service.FormatCSV, l))
	eh, es := readCSV(t, l, "entries.csv")
	ai := col(t, eh, "amount")
	neg := false
	for _, r := range es {
		if strings.HasPrefix(r[ai], "-") {
			neg = true
		}
	}
	if !neg {
		t.Error("negative amounts must stay numeric (no apostrophe)")
	}
}

// FR-094 / DAT-037: one JSON package incl. everything, valid JSON, on any network status.
func TestFullExportJSON(t *testing.T) {
	w := newWorld(t, nil)
	tx := w.Trade(w.net.ID, w.a, w.b, w.aM, w.bM, 2500)
	tu.Must(w.open(w.a, tx.ID))
	tu.Must(w.Svc.CreateListing(w.Ctx, w.a, w.net.ID, service.ListingInput{Type: domain.ListingOffer, Title: "Bike", IdempotencyKey: w.Key()}))
	sink := &service.MemSink{}
	man := tu.Must(w.Svc.Export(w.Ctx, w.admin, w.net.ID, service.ExportFull, service.FormatJSON, sink))
	var doc struct {
		SchemaVersion int                         `json:"schema_version"`
		LedgerSeq     int64                       `json:"ledger_seq"`
		Tables        map[string][]map[string]any `json:"tables"`
	}
	if err := json.Unmarshal(sink.Bytes("full.json"), &doc); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	for _, tbl := range []string{"members", "accounts", "listings", "transactions", "transaction_status_events", "entries", "disputes", "network_policies", "audit_events", "role_assignments"} {
		if len(doc.Tables[tbl]) == 0 {
			t.Errorf("table %s missing or empty", tbl)
		}
	}
	if doc.LedgerSeq != man.LedgerSeq || doc.SchemaVersion != 1 {
		t.Errorf("header %+v vs manifest %+v", doc, man)
	}
	if got := doc.Tables["entries"][0]["amount_formatted"]; got == nil {
		t.Error("money columns need formatted twins")
	}
	if len(doc.Tables["entries"]) != 2 || int64(len(doc.Tables["entries"])+len(doc.Tables["audit_events"])+
		len(doc.Tables["members"])+len(doc.Tables["accounts"])+len(doc.Tables["listings"])+len(doc.Tables["transactions"])+
		len(doc.Tables["transaction_status_events"])+len(doc.Tables["disputes"])+len(doc.Tables["network_policies"])+
		len(doc.Tables["role_assignments"])) != man.Files[0].Rows {
		t.Errorf("manifest row count %d does not match the document", man.Files[0].Rows)
	}
	sum := sha256.Sum256(sink.Bytes("full.json"))
	if hex.EncodeToString(sum[:]) != man.Files[0].SHA256 {
		t.Error("JSON checksum mismatch")
	}
	// audit export is complete and sorted by sequence (FR-093)
	as := &service.MemSink{}
	tu.Must(w.Svc.Export(w.Ctx, w.admin, w.net.ID, service.ExportAudit, service.FormatCSV, as))
	h, rows := readCSV(t, as, "audit_events.csv")
	si := col(t, h, "seq")
	prev := int64(0)
	for _, r := range rows {
		s, _ := strconv.ParseInt(r[si], 10, 64)
		if s != prev+1 {
			t.Fatalf("seq %d after %d", s, prev)
		}
		prev = s
	}
	// the export itself is audited
	if got := tu.Must(w.Svc.ListAudit(w.Ctx, w.admin, w.net.ID, service.AuditQuery{Action: domain.ActExport})); len(got) != 2 {
		t.Errorf("export audit events = %d", len(got))
	}
}
