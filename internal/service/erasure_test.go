package service_test

import (
	"bytes"
	"database/sql"
	"io"
	"strings"
	"testing"

	"github.com/djshellshoxxx/circula/internal/domain"
	"github.com/djshellshoxxx/circula/internal/service"
	tu "github.com/djshellshoxxx/circula/internal/testutil"
)

const (
	anaLogin = "ana@example.org"
	memoText = "thanks Ana for the zucchini"
	listText = "Ana's zucchini patch"
	stmtText = "ana says the zucchini was small"
)

type memSink struct{ files map[string]*bytes.Buffer }

type nopWC struct{ io.Writer }

func (nopWC) Close() error { return nil }

func (m *memSink) Create(name string) (io.WriteCloser, error) {
	if m.files == nil {
		m.files = map[string]*bytes.Buffer{}
	}
	b := &bytes.Buffer{}
	m.files[name] = b
	return nopWC{b}, nil
}

func (m *memSink) all() string {
	var sb strings.Builder
	for _, b := range m.files {
		sb.Write(b.Bytes())
	}
	return sb.String()
}

// erasureWorld: ana trades with ben (with a memo), has a listing and a withdrawn dispute, then departs.
func erasureWorld(t *testing.T) (*world, *domain.Transaction) {
	w := newWorld(t, nil)
	tx := tu.Must(w.Svc.ProposeTransaction(w.Ctx, w.a, w.net.ID, service.ProposeInput{CounterpartyID: w.bM.ID,
		Direction: service.ProposerPays, Amount: 2500, Note: memoText, IdempotencyKey: w.Key()}))
	tx = tu.Must(w.accept(w.b, tx.ID))
	tu.Must(w.Svc.CreateListing(w.Ctx, w.a, w.net.ID, service.ListingInput{Type: domain.ListingOffer, Title: listText,
		Description: "fresh, " + listText, IdempotencyKey: w.Key()}))
	d := tu.Must(w.Svc.OpenDispute(w.Ctx, w.a, w.net.ID, service.OpenDisputeInput{TxID: tx.ID, Category: "other",
		Statement: stmtText, IdempotencyKey: w.Key()}))
	tu.Must(w.Svc.WithdrawDispute(w.Ctx, w.a, w.net.ID, d.ID))
	return w, tx
}

func (w *world) depart(actor domain.Actor, m *domain.Member) {
	w.T.Helper()
	tu.Must(w.Svc.DepartMember(w.Ctx, actor, w.net.ID, service.DepartInput{MemberID: m.ID, Reason: "left town"}))
}

// dbCount counts rows of all text columns of all tables that contain needle.
func dbMatches(t *testing.T, db *sql.DB, needle string) []string {
	t.Helper()
	rows, err := db.Query(`SELECT name FROM sqlite_master WHERE type='table' AND name NOT LIKE 'sqlite_%'`)
	if err != nil {
		t.Fatal(err)
	}
	var tables []string
	for rows.Next() {
		var n string
		_ = rows.Scan(&n)
		tables = append(tables, n)
	}
	_ = rows.Close()
	var hits []string
	for _, tb := range tables {
		cr, err := db.Query(`SELECT name FROM pragma_table_info('` + tb + `')`)
		if err != nil {
			t.Fatal(err)
		}
		var cols []string
		for cr.Next() {
			var c string
			_ = cr.Scan(&c)
			cols = append(cols, c)
		}
		_ = cr.Close()
		for _, c := range cols {
			var n int
			if err := db.QueryRow(`SELECT COUNT(*) FROM "`+tb+`" WHERE CAST("`+c+`" AS TEXT) LIKE ?`, "%"+needle+"%").Scan(&n); err != nil {
				t.Fatal(err)
			}
			if n > 0 {
				hits = append(hits, tb+"."+c)
			}
		}
	}
	return hits
}

// SEC-018: the whole workflow; ledger totals and the audit chain are untouched.
func TestAnonymiseMemberKeepsLedger(t *testing.T) {
	w, trade := erasureWorld(t)
	tok, _ := tu.Must2(w.Svc.Login(w.Ctx, anaLogin, "correct horse battery"))
	w.depart(w.admin, w.aM)
	balBefore := tu.Must(w.Svc.AdminGetMemberBalance(w.Ctx, w.admin, w.net.ID, w.aM.ID)).Balance
	repBefore := tu.Must(w.Svc.VerifyLedger(w.Ctx, w.admin, w.net.ID))
	if !repBefore.OK {
		t.Fatal(repBefore.Findings)
	}
	if len(dbMatches(t, w.Store.DB(), "ana@example.org")) == 0 {
		t.Fatal("test premise: the login should be in the database before erasure")
	}

	if _, err := w.Svc.AnonymiseMember(w.Ctx, w.admin, w.net.ID, w.aM.ID, ""); err == nil {
		t.Fatal("erasure without a reason accepted")
	}
	// A3-05: only the operator CLI (or the person) erases the login; a tenant admin cannot.
	rep := tu.Must(w.Svc.AnonymiseMember(w.Ctx, domain.SystemActor("cli"), w.net.ID, w.aM.ID, "erasure request 7"))
	if !strings.HasPrefix(rep.DisplayName, "Former member ") || !rep.PersonErased || rep.Listings != 1 || rep.Memos != 1 || rep.DisputeStatements != 1 {
		t.Fatalf("report: %+v", rep)
	}

	// Member row, listing, memo, statement.
	m := tu.Must(w.Svc.GetMember(w.Ctx, w.admin, w.net.ID, w.aM.ID))
	if m.DisplayName != rep.DisplayName || m.Status != domain.MemberDeparted || m.ID != w.aM.ID || m.AccountID != w.aM.AccountID {
		t.Fatalf("member: %+v", m)
	}
	// Nothing of the person is left anywhere: names, login, memo, listing text, dispute statement.
	for _, needle := range []string{"ana@example.org", "zucchini", "Ana"} {
		if hits := dbMatches(t, w.Store.DB(), needle); len(hits) > 0 {
			t.Errorf("%q still found in %v", needle, hits)
		}
	}

	// Ledger intact: ids, amounts, status, balances, totals, audit chain.
	got := tu.Must(w.Svc.GetTransaction(w.Ctx, w.admin, w.net.ID, trade.ID))
	if got.Tx.Amount != 2500 || got.Tx.Status != domain.TxCompleted || len(got.Entries) != 2 {
		t.Errorf("transaction changed: %+v", got)
	}
	if got.Tx.Note != "" {
		t.Errorf("memo kept: %q", got.Tx.Note)
	}
	if b := tu.Must(w.Svc.AdminGetMemberBalance(w.Ctx, w.admin, w.net.ID, w.aM.ID)).Balance; b != balBefore || b != -2500 {
		t.Errorf("balance %d -> %d", balBefore, b)
	}
	repAfter := tu.Must(w.Svc.VerifyLedger(w.Ctx, w.admin, w.net.ID))
	if !repAfter.OK {
		t.Fatalf("VerifyLedger after erasure: %v", repAfter.Findings)
	}
	if repAfter.AuditEvents != repBefore.AuditEvents+2 { // account.viewed read + member.anonymised
		t.Errorf("audit events %d -> %d", repBefore.AuditEvents, repAfter.AuditEvents)
	}

	// The audit event exists, is visible to administrators and holds no personal data.
	evs := tu.Must(w.Svc.ListAudit(w.Ctx, w.admin, w.net.ID, service.AuditQuery{Action: domain.ActMemberAnonymised}))
	if len(evs) != 1 || evs[0].TargetID != w.aM.ID || evs[0].ActorType != domain.ActorSystem || evs[0].Reason != "erasure request 7" {
		t.Fatalf("audit: %+v", evs)
	}

	// Credentials are dead: login, old session, throttle.
	if _, _, err := w.Svc.Login(w.Ctx, anaLogin, "correct horse battery"); err == nil {
		t.Error("erased person can still log in")
	}
	if _, err := w.Svc.Authenticate(w.Ctx, tok); err == nil {
		t.Error("session survived erasure")
	}
	if _, _, err := w.Svc.Login(w.Ctx, "erased-"+w.aM.PersonID, "!erased"); err == nil {
		t.Error("login to the erased person with the marker hash worked")
	}

	// Exports show the anonymised name only.
	sink := &memSink{}
	tu.Must(w.Svc.Export(w.Ctx, w.admin, w.net.ID, service.ExportFull, service.FormatJSON, sink))
	out := sink.all()
	if !strings.Contains(out, rep.DisplayName) || strings.Contains(out, "ana@example.org") || strings.Contains(out, "zucchini") {
		t.Errorf("export after erasure: %s", out)
	}

	// Irreversible and not repeatable.
	_, err := w.Svc.AnonymiseMember(w.Ctx, w.admin, w.net.ID, w.aM.ID, "again")
	wantCode(t, err, domain.ErrInvalidState)
	// Other members are untouched.
	if bm := tu.Must(w.Svc.GetMember(w.Ctx, w.admin, w.net.ID, w.bM.ID)); bm.DisplayName != "ben@example.org" {
		t.Errorf("another member changed: %q", bm.DisplayName)
	}
}

func TestAnonymiseMemberRefusals(t *testing.T) {
	w, trade := erasureWorld(t)
	// Active member: not departed.
	_, err := w.Svc.AnonymiseMember(w.Ctx, w.admin, w.net.ID, w.aM.ID, "x")
	wantCode(t, err, domain.ErrInvalidState)
	w.depart(w.admin, w.aM)
	// A plain member (even the counterparty) cannot erase someone else.
	_, err = w.Svc.AnonymiseMember(w.Ctx, w.b, w.net.ID, w.aM.ID, "x")
	wantCode(t, err, domain.ErrForbidden)
	// An admin of another network cannot see the member at all.
	boss := w.Person("boss@other.example")
	other, _ := w.Network(boss, "other", nil)
	_, err = w.Svc.AnonymiseMember(w.Ctx, boss, other.ID, w.aM.ID, "x")
	wantCode(t, err, domain.ErrNotFound)
	_, err = w.Svc.AnonymiseMember(w.Ctx, boss, w.net.ID, w.aM.ID, "x")
	wantCode(t, err, domain.ErrNotFound)
	// A member of another network and an unknown id.
	_, err = w.Svc.AnonymiseMember(w.Ctx, w.admin, w.net.ID, "no-such-member", "x")
	wantCode(t, err, domain.ErrNotFound)
	// Departed members cannot use the admin path on themselves either.
	_, err = w.Svc.AnonymiseMember(w.Ctx, w.a, w.net.ID, w.aM.ID, "x")
	wantCode(t, err, domain.ErrForbidden)

	// Blocker: an unresolved item. A proposed trade of a departed member cannot arise through the
	// service, so insert one directly and check nothing is changed when erasure is refused.
	if _, err := w.Store.DB().Exec(`INSERT INTO transactions(id,network_id,kind,status,payer_account_id,payee_account_id,amount,proposed_at,created_by)
		VALUES ('open-tx',?, 'trade','proposed',?,?,100,1,?)`, w.net.ID, w.aM.AccountID, w.bM.AccountID, w.aM.ID); err != nil {
		t.Fatal(err)
	}
	_, err = w.Svc.AnonymiseMember(w.Ctx, w.admin, w.net.ID, w.aM.ID, "x")
	wantCode(t, err, domain.ErrBlocked)
	var de *domain.Error
	if !asDomain(err, &de) || len(de.Blockers) != 1 || !strings.Contains(de.Blockers[0], "open-tx") {
		t.Errorf("blockers: %v", err)
	}
	if m := tu.Must(w.Svc.GetMember(w.Ctx, w.admin, w.net.ID, w.aM.ID)); m.DisplayName != "ana@example.org" {
		t.Error("a refused erasure changed the member")
	}
	if got := tu.Must(w.Svc.GetTransaction(w.Ctx, w.admin, w.net.ID, trade.ID)); got.Tx.Note != memoText {
		t.Error("a refused erasure cleared the memo")
	}
}

func asDomain(err error, target **domain.Error) bool {
	for err != nil {
		if d, ok := err.(*domain.Error); ok { //nolint:errorlint // test helper, errors are not wrapped here
			*target = d
			return true
		}
		u, ok := err.(interface{ Unwrap() error })
		if !ok {
			return false
		}
		err = u.Unwrap()
	}
	return false
}

// A disputed trade blocks the departure itself, and a live dispute blocks erasure.
func TestAnonymiseMemberLiveDisputeBlocks(t *testing.T) {
	w := newWorld(t, nil)
	tx := w.Trade(w.net.ID, w.a, w.b, w.aM, w.bM, 1000)
	d := tu.Must(w.Svc.OpenDispute(w.Ctx, w.b, w.net.ID, service.OpenDisputeInput{TxID: tx.ID, Category: "other", Statement: "late", IdempotencyKey: w.Key()}))
	_, err := w.Svc.DepartMember(w.Ctx, w.admin, w.net.ID, service.DepartInput{MemberID: w.aM.ID, Reason: "x"})
	wantCode(t, err, domain.ErrBlocked)
	_ = d
	// Resolve it, depart, then erasure works (deferral ends with the blocker).
	tu.Must(w.Svc.StartDisputeReview(w.Ctx, w.admin, w.net.ID, d.ID))
	tu.Must(w.Svc.ResolveDispute(w.Ctx, w.admin, w.net.ID, service.ResolveInput{DisputeID: d.ID, Outcome: domain.OutcomeDismissed, Rationale: "no case", IdempotencyKey: w.Key()}))
	w.depart(w.admin, w.aM)
	tu.Must(w.Svc.AnonymiseMember(w.Ctx, w.admin, w.net.ID, w.aM.ID, "request"))
	if rep := tu.Must(w.Svc.VerifyLedger(w.Ctx, w.admin, w.net.ID)); !rep.OK {
		t.Fatal(rep.Findings)
	}
}

// The departed member erases their own data; only when departed; the login survives other memberships.
func TestEraseMyData(t *testing.T) {
	w, _ := erasureWorld(t)
	// Active members cannot erase themselves.
	_, err := w.Svc.EraseMyData(w.Ctx, w.b, w.net.ID, "")
	wantCode(t, err, domain.ErrInvalidState)
	// Ana also belongs to a second, live network.
	boss := w.Person("boss@other.example")
	other, _ := w.Network(boss, "other", nil)
	tok, _ := tu.Must2(w.Svc.CreateInvitation(w.Ctx, boss, other.ID, service.InviteInput{}))
	tu.Must(w.Svc.RedeemInvitation(w.Ctx, w.a, tok, "Ana in other"))
	w.depart(w.admin, w.aM)
	// A stranger to the network cannot erase anything here.
	_, err = w.Svc.EraseMyData(w.Ctx, boss, w.net.ID, "")
	wantCode(t, err, domain.ErrNotFound)

	rep := tu.Must(w.Svc.EraseMyData(w.Ctx, w.a, w.net.ID, ""))
	if rep.PersonErased {
		t.Error("login erased although the person is still an active member elsewhere")
	}
	if _, _, err := w.Svc.Login(w.Ctx, anaLogin, "correct horse battery"); err != nil {
		t.Errorf("login lost: %v", err)
	}
	// The other membership keeps its own name.
	mine := tu.Must(w.Svc.MyMemberships(w.Ctx, w.a))
	for _, m := range mine {
		if m.NetworkID == other.ID && m.DisplayName != "Ana in other" {
			t.Errorf("other network's name changed: %q", m.DisplayName)
		}
		if m.NetworkID == w.net.ID && !strings.HasPrefix(m.DisplayName, "Former member ") {
			t.Errorf("name: %q", m.DisplayName)
		}
	}
	evs := tu.Must(w.Svc.ListAudit(w.Ctx, w.admin, w.net.ID, service.AuditQuery{Action: domain.ActMemberAnonymised}))
	if len(evs) != 1 || evs[0].ActorType != domain.ActorMember || evs[0].Reason != "requested by the member" {
		t.Fatalf("audit: %+v", evs)
	}
	if rep := tu.Must(w.Svc.VerifyLedger(w.Ctx, w.admin, w.net.ID)); !rep.OK {
		t.Fatal(rep.Findings)
	}
	// Her remaining membership elsewhere is later left and erased too: now the login goes.
	tu.Must(w.Svc.DepartMember(w.Ctx, w.a, other.ID, service.DepartInput{}))
	rep2 := tu.Must(w.Svc.EraseMyData(w.Ctx, w.a, other.ID, "done"))
	if !rep2.PersonErased {
		t.Error("login kept after the last membership was erased")
	}
	if _, _, err := w.Svc.Login(w.Ctx, anaLogin, "correct horse battery"); err == nil {
		t.Error("login works after the last membership was erased")
	}
}

// The system actor (operator CLI) may erase too; the erasure table is append-only.
func TestAnonymiseBySystemAndImmutableRecord(t *testing.T) {
	w, _ := erasureWorld(t)
	w.depart(w.admin, w.aM)
	rep := tu.Must(w.Svc.AnonymiseMember(w.Ctx, domain.SystemActor("cli"), w.net.ID, w.aM.ID, "operator request"))
	if !rep.PersonErased {
		t.Error("person not erased")
	}
	evs := tu.Must(w.Svc.ListAudit(w.Ctx, w.admin, w.net.ID, service.AuditQuery{Action: domain.ActMemberAnonymised}))
	if len(evs) != 1 || evs[0].ActorType != domain.ActorSystem {
		t.Fatalf("audit: %+v", evs)
	}
	if _, err := w.Store.DB().Exec(`DELETE FROM member_erasures`); err == nil {
		t.Error("erasure record deletable")
	}
}

// SA2-05: second-factor rows and notices must not outlive erasure.
func TestErasureRemovesMFAAndNotices(t *testing.T) {
	w, _ := erasureWorld(t)
	tu.Must2(w.Svc.CreateAPIToken(w.Ctx, w.a, service.TokenInput{Name: "AnaSecretLaptop", Scopes: []domain.Scope{domain.ScopeRead}}))
	tu.Must(w.Svc.BeginMFAEnrolment(w.Ctx, w.a, pw))
	w.depart(w.admin, w.aM)
	tu.Must(w.Svc.AnonymiseMember(w.Ctx, domain.SystemActor("cli"), w.net.ID, w.aM.ID, "gdpr"))
	for _, q := range []string{"person_mfa", "security_notices", "recovery_codes", "account_recovery_codes"} {
		var n int
		if err := w.Store.DB().QueryRow(`SELECT COUNT(*) FROM `+q+` WHERE person_id = ?`, w.a.PersonID).Scan(&n); err != nil || n != 0 {
			t.Errorf("%s rows after erasure = %d (err %v)", q, n, err)
		}
	}
	if hits := dbMatches(t, w.Store.DB(), "AnaSecretLaptop"); len(hits) != 0 {
		t.Errorf("token name survives erasure: %v", hits)
	}
}

// A3-05: an administrator's erasure anonymises the membership in that network
// only; the login, sessions and credentials of the person stay usable.
func TestA3_05_AdminErasureKeepsLogin(t *testing.T) {
	w, _ := erasureWorld(t)
	boss := w.Person("boss@other.example")
	other, _ := w.Network(boss, "other", nil)
	inv, _ := tu.Must2(w.Svc.CreateInvitation(w.Ctx, boss, other.ID, service.InviteInput{}))
	tu.Must(w.Svc.RedeemInvitation(w.Ctx, w.a, inv, "Ana in other"))
	w.depart(w.admin, w.aM)
	rep := tu.Must(w.Svc.AnonymiseMember(w.Ctx, w.admin, w.net.ID, w.aM.ID, "admin request"))
	if rep.PersonErased {
		t.Fatal("admin erased the person row")
	}
	if !strings.HasPrefix(rep.DisplayName, "Former member ") {
		t.Errorf("membership not anonymised: %+v", rep)
	}
	if _, _, err := w.Svc.Login(w.Ctx, anaLogin, "correct horse battery"); err != nil {
		t.Errorf("login destroyed by a tenant admin: %v", err)
	}
	// Even with no other membership left the admin cannot erase the login.
	w2, _ := erasureWorld(t)
	w2.depart(w2.admin, w2.aM)
	if rep := tu.Must(w2.Svc.AnonymiseMember(w2.Ctx, w2.admin, w2.net.ID, w2.aM.ID, "admin request")); rep.PersonErased {
		t.Error("admin erased the login of a person with no other membership")
	}
}
